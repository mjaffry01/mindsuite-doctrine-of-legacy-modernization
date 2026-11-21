/* MindSuite web app · suggested peak-load calculation
   Turns expected users into a suggested peak load, showing every step:
     active in the busiest hour = users × active share
     concurrent users           = active × session minutes ÷ 60
     peak requests per second   = concurrent × actions per minute × requests per action ÷ 60
     sized for 3 years          = peak × (1 + headroom) × (1 + yearly growth)^3
   It also allows for the busiest minute (burst) and sizes vCPU at an editable requests-per-vCPU figure.
   The assumptions are predicted from the brief by local rules (loadPredict: who the users are, tenancy,
   channels, rushes, heavy screens); the owner can switch to a typical profile or edit any value. Nothing is used until the owner presses
   "Use … as the peak"; the value is then marked "estimated from users" for sizing and the agents, and the
   mark disappears when the owner types a different peak. A mismatch warning appears when an entered peak is
   far from what the users suggest.
   Product brief: loadCalcPanel() under the brief's numbers (assets/project-mode.js).
   Modernization: capLoadSuggestion(cp) in the hosting estimate when a capacity profile has users but no peak.
   Loaded before the main script; uses its helpers at call time. */

const LOAD_PROFILES=[
 ['internal','Internal business tool',{active:30,session:30,actions:3,requests:3,burst:1.5,perVcpu:25},'Staff use it through the working day; busiest hour is a normal morning.'],
 ['consumer','Consumer web or mobile app',{active:10,session:10,actions:6,requests:4,burst:2,perVcpu:25},'Many users, short visits; busiest hour is an evening or a campaign.'],
 ['checkout','Shop counter / checkout',{active:50,session:45,actions:2,requests:4,burst:2,perVcpu:25},'Most staff are on shift together; each sale is a few quick steps.']];
const LOAD_FIELDS=[['active','Active in the busiest hour','% of users',1,100],['session','Session length in that hour','minutes',1,60],
 ['actions','Actions per user','per minute',0.1,60],['requests','Requests per action','server requests',1,50],
 ['burst','Busiest minute vs hour average','× (1 = even)',1,10],['perVcpu','Requests one vCPU handles','per second',1,1000]];
const LOAD_HEADROOM=30,LOAD_YEARS=3;

function loadEstimate(users,a,growthPct,concurrentGiven){
 const active=concurrentGiven?null:users*a.active/100;
 const concurrent=concurrentGiven||active*Math.min(a.session,60)/60;
 const hourly=concurrent*a.actions*a.requests/60;
 const burst=Math.max(1,+a.burst||1),peak=hourly*burst;
 const g=Math.max(0,+growthPct||0);
 const sized=peak*(1+LOAD_HEADROOM/100)*Math.pow(1+g/100,LOAD_YEARS);
 const per=Math.max(1,+a.perVcpu||25);
 return {active,concurrent,hourly,peak,sized,growth:g,vcpu:Math.max(2,Math.ceil(sized/per))};
}
const loadRound=n=>n>=100?Math.round(n):n>=10?Math.round(n*10)/10:Math.round(n*100)/100;
const loadFmt=n=>loadRound(n).toLocaleString();
function loadGrowth(){try{return num(constraints().growth)||0;}catch(e){return 0;}}

/* Predict the assumptions from what the brief already says (local rules; nothing is sent anywhere).
   Returns the profile, the six values and one reason for each value the brief changed. */
function loadPredict(P){
 const D=P.deploy||{},ch=D.channels||[];
 const text=['problem','users','journeys','mvp','success','integrations'].map(k=>String(P[k]||'')).join(' ').toLowerCase();
 const has=re=>re.test(text);
 const users=num(P.users_count)||0;
 const why={};let profile='internal',pwhy='';
 if(has(/\b(till|tills|checkout|point of sale|pos|cashier|counter|shop floor)\b/)){profile='checkout';pwhy='the brief mentions tills or checkout';}
 else if(D.tenancy==='internal'){profile='internal';pwhy='tenancy is internal use only';}
 else if(has(/\b(customers?|shoppers?|consumers?|public|patients|citizens|students|members|subscribers|guests|visitors)\b/)||ch.includes('mobile')){profile='consumer';pwhy=has(/\b(customers?|shoppers?|consumers?|public|patients|citizens|students|members|subscribers|guests|visitors)\b/)?'the users are customers or the public':'it reaches users through mobile apps';}
 else if(has(/\b(staff|employees?|team|back office|managers?|agents?|clerks?|buyers?)\b/)){profile='internal';pwhy='the users are staff';}
 else pwhy='the brief does not say who the users are yet, so a typical internal tool is assumed';
 const v={...LOAD_PROFILES.find(p=>p[0]===profile)[2]};
 if(profile==='internal'&&users&&users<=200){v.active=50;why.active='a small team ('+users+' users), so about half are on in the busiest hour';}
 if(profile==='consumer'&&users>=100000){v.active=5;why.active='a large audience ('+users.toLocaleString()+' users), so a smaller share is on at once';}
 if(has(/\b(campaigns?|flash sales?|launch|black friday|tickets?|exams?|results day|enrol(?:ment)?|registration)\b/)){v.burst=3;why.burst='the brief mentions campaigns, launches or deadlines that bring sudden rushes';}
 else if(has(/\b(shifts?|start of (?:the )?day|9 ?am|morning|clock(?:ing)? in)\b/)){v.burst=2;why.burst='work starts at the same time (shifts or morning log-ins)';}
 if(has(/\b(reports?|dashboards?|analytics|exports?|forecast(?:ing)?)\b/)){v.requests=v.requests+1;v.perVcpu=10;why.requests='reports and dashboards load several things per screen';why.perVcpu='report and dashboard requests are heavy, so fewer per vCPU';}
 else if(has(/\b(catalogue|catalog|browse|search|content|news|read-only|lookup)\b/)||(ch.includes('api')&&D.tenancy!=='internal')){v.perVcpu=50;why.perVcpu='mostly reads that can be cached, so more per vCPU';}
 if(profile!=='consumer'&&has(/\b(scan(?:ning|s)?|barcodes?|count(?:ing)? (?:a )?shelf|stock counts?)\b/)){v.actions=Math.max(v.actions,5);why.actions='scanning and counting are quick, repeated actions';}
 return {profile,profileWhy:pwhy,values:v,why};
}
/* Effective assumptions and where each came from: 'edited' (the owner), 'predicted' (from the brief) or 'typical' (profile). */
function loadAssumptions(store){
 store.loadCalc=store.loadCalc||{mode:'predicted'};
 const L=store.loadCalc,mode=L.mode||(L.profile?'profile':'predicted');
 const pred=mode==='predicted'?loadPredict(store):null;
 const profile=pred?pred.profile:L.profile||'internal';
 const typical=(LOAD_PROFILES.find(p=>p[0]===profile)||LOAD_PROFILES[0])[2];
 const base=pred?pred.values:typical;
 const a={},src={};
 for(const [k] of LOAD_FIELDS){const v=num(L[k]);if(v!=null&&v>0){a[k]=v;src[k]='edited';}else{a[k]=base[k];src[k]=pred&&pred.why[k]?'predicted':'typical';}}
 return Object.assign(a,{_src:src,_mode:mode,_profile:profile,_pred:pred});
}
function loadSteps(users,a,e){
 return [
  ['Active in the busiest hour',loadFmt(users)+' users × '+a.active+'%',loadFmt(e.active)+' users'],
  ['Concurrent users',loadFmt(e.active)+' × '+a.session+' min ÷ 60',loadFmt(e.concurrent)+' at once'],
  ['Average in that hour',loadFmt(e.concurrent)+' × '+a.actions+' actions/min × '+a.requests+' requests ÷ 60',loadFmt(e.hourly)+' requests/s'],
  ['Peak (busiest minute)',loadFmt(e.hourly)+' × '+a.burst+' burst',loadFmt(e.peak)+' requests/s'],
  ['Sized for '+LOAD_YEARS+' years',loadFmt(e.peak)+' × '+(1+LOAD_HEADROOM/100)+' headroom'+(e.growth?' × (1 + '+e.growth+'%)^'+LOAD_YEARS+' growth':'')+' ÷ '+a.perVcpu+' per vCPU',loadFmt(e.sized)+' requests/s ≈ '+e.vcpu+' vCPU']];
}

/* ---- Product brief ---- */
function loadCalcPanel(){
 const P=productBrief();
 const users=num(typeof briefAnswer==='function'?briefAnswer('users_count'):P.users_count);
 if(!users)return h('div',{class:'load-calc empty'},[h('b',{text:'Suggested peak load'}),h('p',{class:'muted',text:'Enter expected users to get a predicted peak load, worked out step by step from your brief.'})]);
 const a=loadAssumptions(P),e=loadEstimate(users,a,loadGrowth());
 const entered=num(typeof briefAnswer==='function'?briefAnswer('peak'):P.peak);
 const suggested=Math.max(0.01,loadRound(e.peak));
 const usedOurs=P.peakBasis==='estimated-from-users'&&entered!=null&&P.peakCalc&&+P.peakCalc.peak===entered;
 const ratio=entered&&e.peak?entered/e.peak:null;
 const setA=(k,v)=>{const b=P.loadCalc[k];P.loadCalc[k]=v;logInput('product:load:'+k,b,v);persist();render();};
 const setMode=(mode,profile)=>{const b=P.loadCalc;P.loadCalc=mode==='predicted'?{mode}:{mode,profile};logInput('product:load:mode',(b.mode||'')+':'+(b.profile||''),mode+':'+(profile||''));persist();render();};
 const pred=a._pred,srcTag=k=>a._src[k]==='edited'?'you set this':a._src[k]==='predicted'?'predicted':'typical';
 const label=id=>(LOAD_PROFILES.find(p=>p[0]===id)||LOAD_PROFILES[0])[1];
 const reasons=pred?LOAD_FIELDS.filter(([k])=>pred.why[k]&&a._src[k]==='predicted'):[];
 return h('div',{class:'load-calc'},[
  h('div',{class:'load-head'},[h('b',{text:'Suggested peak load'}),h('small',{class:'muted',text:'Predicted from your brief and worked out step by step. Change any assumption; nothing is used until you choose to.'})]),
  h('div',{class:'load-profiles'},[h('span',{class:'muted',text:'Assumptions:'}),
   h('button',{type:'button',class:'type-chip'+(a._mode==='predicted'?' on':''),title:'Predict the assumptions from what the brief says',onclick:()=>setMode('predicted')},['Predicted from the brief']),
   ...LOAD_PROFILES.map(([id,lbl,,hint])=>h('button',{type:'button',class:'type-chip'+(a._mode==='profile'&&a._profile===id?' on':''),title:hint,onclick:()=>setMode('profile',id)},['Typical: '+lbl]))]),
  pred?h('div',{class:'load-why'},[h('p',{},[h('b',{text:'Prediction: '}),label(pred.profile)+', because '+pred.profileWhy+'.']),
   reasons.length?h('ul',{},reasons.map(([k,l])=>h('li',{text:l+' '+a[k]+': '+pred.why[k]+'.'}))):h('p',{class:'muted',text:'The brief did not change any typical values. Say who the users are, any busy periods, and heavy screens such as reports to sharpen it.'})]):null,
  h('div',{class:'load-inputs'},LOAD_FIELDS.map(([k,lbl,unit,min,max])=>h('label',{class:'context-control'},[h('span',{text:lbl}),
   h('div',{class:'load-input'},[h('input',{class:'field compact',type:'number',min:String(min),max:String(max),step:'any',value:String(a[k]),'aria-label':lbl+' ('+unit+')',
    onchange:ev=>{const v=num(ev.target.value);if(v==null||v<min||v>max){ev.target.value=String(a[k]);return;}setA(k,v);}}),h('small',{class:'muted',text:unit})]),
   h('small',{class:'load-src '+a._src[k],text:srcTag(k)})]))),
  h('table',{class:'scan-table load-steps'},[h('tbody',{},loadSteps(users,a,e).map(([l,f,r],i)=>h('tr',{class:i===3?'key':''},[h('th',{scope:'row',text:l}),h('td',{class:'muted',text:f}),h('td',{},[h('b',{text:r})])])))]),
  !e.growth?h('small',{class:'muted',text:'No yearly growth set; add it under Rules & people → Priorities & limits to size for growth. '}):null,
  h('small',{class:'muted',text:'Requests per vCPU is a planning guess; a load test on the real app gives the true figure.'}),
  ratio&&(ratio>2||ratio<0.5)&&!usedOurs?h('p',{class:'input-status open',text:'Your peak of '+loadFmt(entered)+' requests/s is '+(ratio>1?loadFmt(ratio)+'× more':loadFmt(1/ratio)+'× less')+' than these users suggest ('+loadFmt(e.peak)+'). Check the users, the peak or the assumptions.'}):null,
  h('div',{class:'load-actions'},[
   usedOurs?h('span',{class:'input-status done',text:'Peak in the brief is this estimate ('+loadFmt(entered)+' requests/s), marked "estimated from users".'})
    :button('Use '+loadFmt(suggested)+' requests/s as the peak',()=>{
     const before=entered;
     if(typeof writeBriefAnswer==='function')writeBriefAnswer('peak',String(suggested));else P.peak=String(suggested);
     const keep={};for(const [k] of LOAD_FIELDS)keep[k]=a[k];
     P.peakBasis='estimated-from-users';P.peakCalc={peak:suggested,users,assumptions:keep,basis:a._mode==='predicted'?'predicted from the brief':'typical '+label(a._profile).toLowerCase(),growth:e.growth,vcpu:e.vcpu,at:new Date().toISOString()};
     logInput('product:peak-estimated',before,suggested);persist();render();notice('Peak set to '+loadFmt(suggested)+' requests/s, estimated from '+loadFmt(users)+' users. Edit it any time.');
    },'btn small primary')])]);
}
/* Fact text for the agents: says when the peak is an estimate and how it was made. */
function peakFactNote(P){
 if(P.peakBasis!=='estimated-from-users'||!P.peakCalc||+P.peakCalc.peak!==num(P.peak))return '';
 const a=P.peakCalc.assumptions||{};
 return ' (estimated from '+P.peakCalc.users+' users'+(P.peakCalc.basis?', '+P.peakCalc.basis:'')+': '+a.active+'% active in the busiest hour, '+a.session+' min sessions, '+a.actions+' actions/min × '+a.requests+' requests'+(a.burst?', busiest minute '+a.burst+'× the hour average':'')+(a.perVcpu?'; about '+(P.peakCalc.vcpu||'?')+' vCPU at '+a.perVcpu+' requests/s each':'')+')';
}

/* ---- Modernization: hosting estimate when the capacity profile has users but no peak ---- */
function capLoadSuggestion(cp){
 const users=num(cp.users),conc=num(cp.concurrent);
 if(!users&&!conc)return '';
 const a=LOAD_PROFILES[0][2],growth=num(cp.growth)||loadGrowth();
 let e;
 e=loadEstimate(users,a,growth,conc||0);
 return 'No peak load recorded. Suggested from '+(conc?loadFmt(conc)+' concurrent users':loadFmt(users)+' users ('+a.active+'% active in the busiest hour, '+a.session+' min sessions)')+' at '+a.actions+' actions/min × '+a.requests+' requests, busiest minute '+a.burst+'× the hour: ≈ '+loadFmt(e.peak)+' requests/s at peak, ≈ '+e.vcpu+' vCPU with headroom'+(growth?' and '+growth+'% yearly growth':'')+'. Add the measured or estimated peak to the capacity profile to size it from facts.';
}
/* A check row when users and peak are both recorded but do not fit together. */
function capLoadCheck(cp){
 const users=num(cp.users),peak=num(cp.peakTps);if(!users||!peak)return '';
 const e=loadEstimate(users,LOAD_PROFILES[0][2],0);const r=peak/e.peak;
 if(r<=5&&r>=0.2)return '';
 return 'Recorded peak '+loadFmt(peak)+'/s is '+(r>1?loadFmt(r)+'× more':loadFmt(1/r)+'× less')+' than '+loadFmt(users)+' users of a typical internal tool suggest (≈ '+loadFmt(e.peak)+'/s). Check the numbers or the kind of use.';
}

(function loadStyle(){
 if(document.getElementById('load-calc-style'))return;
 const css=`
.brief-sub{margin:16px 0 6px;font-size:14px}
.load-calc{margin:12px 0;padding:12px 14px;border:1px solid #d9e1f2;border-radius:10px;background:#f8faff}
.load-calc.empty{background:#fbfbfd}
.load-head{display:flex;flex-wrap:wrap;gap:4px 10px;align-items:baseline}
.load-profiles{display:flex;flex-wrap:wrap;gap:6px;align-items:center;margin:8px 0}
.load-inputs{display:grid;grid-template-columns:repeat(auto-fit,minmax(170px,1fr));gap:8px 12px;margin:6px 0}
.load-input{display:flex;align-items:center;gap:6px}.load-input .field.compact{width:90px}
.load-steps{margin:8px 0;font-size:13px}.load-steps th{text-align:left;font-weight:600;color:var(--muted);white-space:nowrap}
.load-steps tr.key td b{color:var(--accent)}
.load-actions{margin-top:8px}
.load-why{margin:6px 0;padding:8px 10px;border-left:3px solid var(--accent);background:#fff;font-size:13px}.load-why p{margin:0 0 4px}.load-why ul{margin:0;padding-left:18px}
.load-src{font-size:11px;color:var(--muted)}.load-src.predicted{color:var(--accent)}.load-src.edited{color:#7a4b00}`;
 document.head.append(Object.assign(document.createElement('style'),{id:'load-calc-style',textContent:css}));
})();

/* Redraw just the calculator when users or peak change in the brief (no full page render while typing). */
function refreshLoadCalc(){
 const old=document.querySelector('.load-calc');if(!old)return;
 try{old.replaceWith(loadCalcPanel());}catch(e){console.error('Load calculator',e);}
}
