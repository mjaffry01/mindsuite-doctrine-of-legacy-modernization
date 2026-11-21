/* MindSuite web app · suggested data volume after 3 years
   Turns expected users into a predicted 3-year data volume, showing every step:
     daily active users      = users × daily active share
     records per day         = daily active users × records per user per day
     records in 3 years      = records per day × working days per year × (1 + (1+g) + (1+g)²)
     record data (GB)        = records × KB per record ÷ 1,048,576
     with indexes & history  = record data × (1 + overhead)
     files (GB)              = files per day × MB per file × days in 3 years (with growth) ÷ 1,024
     total                   = with indexes & history + files      (backups and copies are extra)
   The assumptions are predicted from the brief by local rules (dataPredict); the owner can switch to a
   typical profile or edit any value. Nothing is used until "Use … GB"; the value is then marked
   "estimated from users" for sizing and the agents. Needs load-calculator.js (loadRound, loadFmt, loadGrowth).
   Hook: dataCalcPanel() under the brief's data volume field (assets/project-mode.js). */

const DATA_PROFILES=[
 ['internal','Internal business tool',{daily:70,records:20,kb:2,days:250,overhead:50,files:0,fileMb:0.5},'Staff record their work on working days.'],
 ['consumer','Consumer web or mobile app',{daily:20,records:5,kb:2,days:365,overhead:50,files:0,fileMb:0.5},'Many users, a few records each, every day of the year.'],
 ['checkout','Shop counter / checkout',{daily:90,records:60,kb:1,days:360,overhead:50,files:0,fileMb:0.5},'Each sale writes several small lines; shops open most days.']];
const DATA_FIELDS=[['daily','Active on a typical day','% of users',1,100],['records','Records each creates','per day',0.1,100000],
 ['kb','Size of one record','KB',0.1,10000],['days','Days in use','per year',1,366],
 ['overhead','Indexes and history','% extra',0,500],['files','Files uploaded','per day',0,1000000],['fileMb','Size of one file','MB',0.01,2000]];
const DATA_YEARS=3;

function dataEstimate(users,a,growthPct){
 const g=Math.max(0,+growthPct||0)/100;
 let yearsFactor=0;for(let y=0;y<DATA_YEARS;y++)yearsFactor+=Math.pow(1+g,y);
 const dau=users*a.daily/100,perDay=dau*a.records,records=perDay*a.days*yearsFactor;
 const rowGb=records*a.kb/1048576,dbGb=rowGb*(1+a.overhead/100);
 const fileGb=a.files*a.fileMb*a.days*yearsFactor/1024;
 return {dau,perDay,records,rowGb,dbGb,fileGb,total:dbGb+fileGb,growth:g*100,yearsFactor};
}

function dataPredict(P){
 const D=P.deploy||{},ch=D.channels||[];
 const text=['problem','users','journeys','mvp','success','integrations'].map(k=>String(P[k]||'')).join(' ').toLowerCase();
 const has=re=>re.test(text);
 const lp=typeof loadPredict==='function'?loadPredict(P):{profile:'internal',profileWhy:''};
 const profile=lp.profile,why={};
 const v={...DATA_PROFILES.find(p=>p[0]===profile)[2]};
 if(has(/\b(sensors?|iot|telemetry|devices? send|readings?|gps|tracking every|logs every)\b/)){v.records=Math.max(v.records,1000);v.kb=0.5;why.records='devices or sensors send readings all day';why.kb='readings are small';}
 else if(has(/\b(scan(?:ning|s)?|barcodes?|stock counts?|count(?:ing)? (?:a )?shelf|transactions?|orders?|bookings?|tickets?)\b/)&&profile!=='consumer'){v.records=Math.max(v.records,40);why.records='scans, orders or transactions each write a record';}
 if(has(/\b(photos?|photograph\w*|images?|pictures?|scanned|attachments?|documents?|pdfs?|invoices?|receipts?|signatures?|videos?)\b/)){
  v.files=Math.max(1,Math.round((num(P.users_count)||0)*v.daily/100*0.2));why.files='the brief mentions photos, documents or attachments (about 1 in 5 daily users uploads one)';
  if(has(/\bvideos?\b/)){v.fileMb=50;why.fileMb='videos are large';}else if(has(/\b(photos?|photograph\w*|images?|pictures?)\b/)){v.fileMb=2;why.fileMb='phone photos are about 2 MB';}
 }
 if(has(/\b(audit|history|compliance|regulat\w*|retain|retention|trace(?:ability)?|versions?)\b/)){v.overhead=100;why.overhead='audit trails or kept history roughly double the stored rows';}
 if(has(/\b(notes?|comments?|descriptions?|free text|messages?|chat)\b/)){v.kb=Math.max(v.kb,4);why.kb=why.kb||'notes and messages make records bigger';}
 return {profile,profileWhy:lp.profileWhy||'',values:v,why};
}
function dataAssumptions(store){
 store.dataCalc=store.dataCalc||{mode:'predicted'};
 const L=store.dataCalc,mode=L.mode||'predicted';
 const pred=mode==='predicted'?dataPredict(store):null;
 const profile=pred?pred.profile:L.profile||'internal';
 const base=pred?pred.values:(DATA_PROFILES.find(p=>p[0]===profile)||DATA_PROFILES[0])[2];
 const a={},src={};
 for(const [k] of DATA_FIELDS){const v=num(L[k]);if(v!=null&&v>=0&&L[k]!==''&&L[k]!=null){a[k]=v;src[k]='edited';}else{a[k]=base[k];src[k]=pred&&pred.why[k]?'predicted':'typical';}}
 return Object.assign(a,{_src:src,_mode:mode,_profile:profile,_pred:pred});
}
const gbFmt=n=>n>=1?loadFmt(n)+' GB':loadFmt(n*1024)+' MB';
function dataSteps(users,a,e){
 const g=e.growth?' × '+loadFmt(e.yearsFactor)+' (3 years with '+e.growth+'% growth)':' × '+DATA_YEARS+' years';
 const rows=[
  ['Active on a typical day',loadFmt(users)+' users × '+a.daily+'%',loadFmt(e.dau)+' users'],
  ['Records per day',loadFmt(e.dau)+' × '+a.records+' records',loadFmt(e.perDay)+' records'],
  ['Records in 3 years',loadFmt(e.perDay)+' × '+a.days+' days'+g,loadFmt(e.records)+' records'],
  ['Record data',loadFmt(e.records)+' × '+a.kb+' KB',gbFmt(e.rowGb)],
  ['With indexes and history',gbFmt(e.rowGb)+' × '+(1+a.overhead/100),gbFmt(e.dbGb)]];
 if(a.files>0)rows.push(['Files',a.files+' a day × '+a.fileMb+' MB × '+a.days+' days'+g,gbFmt(e.fileGb)]);
 rows.push(['Total after 3 years',a.files>0?'database + files':'database',gbFmt(e.total)]);
 return rows;
}

function dataCalcPanel(){
 const P=productBrief();
 const users=num(typeof briefAnswer==='function'?briefAnswer('users_count'):P.users_count);
 if(!users)return h('div',{class:'load-calc data-calc empty'},[h('b',{text:'Suggested data volume'}),h('p',{class:'muted',text:'Enter expected users above to get a predicted 3-year data volume, worked out step by step from your brief.'})]);
 const a=dataAssumptions(P),e=dataEstimate(users,a,loadGrowth());
 const entered=num(typeof briefAnswer==='function'?briefAnswer('dataGb'):P.dataGb);
 const suggested=Math.max(1,Math.round(e.total));
 const usedOurs=P.dataBasis==='estimated-from-users'&&entered!=null&&P.dataCalc&&+P.dataCalc.gb===entered;
 const ratio=entered&&e.total?entered/e.total:null;
 const setA=(k,v)=>{const b=P.dataCalc[k];P.dataCalc[k]=v;logInput('product:data:'+k,b,v);persist();render();};
 const setMode=(mode,profile)=>{const b=P.dataCalc;P.dataCalc=mode==='predicted'?{mode}:{mode,profile};logInput('product:data:mode',(b.mode||'')+':'+(b.profile||''),mode+':'+(profile||''));persist();render();};
 const label=id=>(DATA_PROFILES.find(p=>p[0]===id)||DATA_PROFILES[0])[1];
 const pred=a._pred,reasons=pred?DATA_FIELDS.filter(([k])=>pred.why[k]&&a._src[k]==='predicted'):[];
 const steps=dataSteps(users,a,e);
 return h('div',{class:'load-calc data-calc'},[
  h('div',{class:'load-head'},[h('b',{text:'Suggested data volume'}),h('small',{class:'muted',text:'How much data 3 years of daily use creates, predicted from your brief. Change any assumption; nothing is used until you choose to.'})]),
  h('div',{class:'load-profiles'},[h('span',{class:'muted',text:'Assumptions:'}),
   h('button',{type:'button',class:'type-chip'+(a._mode==='predicted'?' on':''),onclick:()=>setMode('predicted')},['Predicted from the brief']),
   ...DATA_PROFILES.map(([id,lbl,,hint])=>h('button',{type:'button',class:'type-chip'+(a._mode==='profile'&&a._profile===id?' on':''),title:hint,onclick:()=>setMode('profile',id)},['Typical: '+lbl]))]),
  pred?h('div',{class:'load-why'},[h('p',{},[h('b',{text:'Prediction: '}),label(pred.profile)+(pred.profileWhy?', because '+pred.profileWhy:'')+'.']),
   reasons.length?h('ul',{},reasons.map(([k,l])=>h('li',{text:l+' '+a[k]+': '+pred.why[k]+'.'}))):h('p',{class:'muted',text:'The brief did not change any typical values. Mention what users record (orders, scans, photos, documents) and any audit or retention needs to sharpen it.'})]):null,
  h('div',{class:'load-inputs'},DATA_FIELDS.map(([k,lbl,unit,min,max])=>h('label',{class:'context-control'},[h('span',{text:lbl}),
   h('div',{class:'load-input'},[h('input',{class:'field compact',type:'number',min:String(min),max:String(max),step:'any',value:String(a[k]),'aria-label':lbl+' ('+unit+')',
    onchange:ev=>{const v=num(ev.target.value);if(v==null||v<min||v>max){ev.target.value=String(a[k]);return;}setA(k,v);}}),h('small',{class:'muted',text:unit})]),
   h('small',{class:'load-src '+a._src[k],text:a._src[k]==='edited'?'you set this':a._src[k]})]))),
  h('table',{class:'scan-table load-steps'},[h('tbody',{},steps.map(([l,f,r],i)=>h('tr',{class:i===steps.length-1?'key':''},[h('th',{scope:'row',text:l}),h('td',{class:'muted',text:f}),h('td',{},[h('b',{text:r})])])))]),
  h('small',{class:'muted',text:(e.growth?'':'No yearly growth set; add it under Rules & people → Priorities & limits. ')+'Backups, test copies and reporting copies are extra; plan about 2–3× this for total storage.'}),
  ratio&&(ratio>3||ratio<1/3)&&!usedOurs?h('p',{class:'input-status open',text:'Your data volume of '+loadFmt(entered)+' GB is '+(ratio>1?loadFmt(ratio)+'× more':loadFmt(1/ratio)+'× less')+' than these users suggest ('+gbFmt(e.total)+'). Check the users, the volume or the assumptions.'}):null,
  h('div',{class:'load-actions'},[
   usedOurs?h('span',{class:'input-status done',text:'Data volume in the brief is this estimate ('+loadFmt(entered)+' GB), marked "estimated from users".'})
    :button('Use '+loadFmt(suggested)+' GB as the data volume',()=>{
     const before=entered;
     if(typeof writeBriefAnswer==='function')writeBriefAnswer('dataGb',String(suggested));else P.dataGb=String(suggested);
     const keep={};for(const [k] of DATA_FIELDS)keep[k]=a[k];
     P.dataBasis='estimated-from-users';P.dataCalc={...P.dataCalc,gb:suggested,users,assumptions:keep,basis:a._mode==='predicted'?'predicted from the brief':'typical '+label(a._profile).toLowerCase(),growth:e.growth,at:new Date().toISOString()};
     logInput('product:data-estimated',before,suggested);persist();render();notice('Data volume set to '+loadFmt(suggested)+' GB, estimated from '+loadFmt(users)+' users. Edit it any time.');
    },'btn small primary')])]);
}
/* Fact text for the agents. */
function dataFactNote(P){
 if(P.dataBasis!=='estimated-from-users'||!P.dataCalc||+P.dataCalc.gb!==num(P.dataGb))return '';
 const a=P.dataCalc.assumptions||{};
 return ' (estimated from '+P.dataCalc.users+' users, '+P.dataCalc.basis+': '+a.daily+'% active daily, '+a.records+' records each a day at '+a.kb+' KB, '+a.days+' days a year, +'+a.overhead+'% indexes and history'+(a.files?', '+a.files+' files a day at '+a.fileMb+' MB':'')+'; backups extra)';
}
function refreshDataCalc(){
 const old=document.querySelector('.data-calc');if(!old)return;
 try{old.replaceWith(dataCalcPanel());}catch(e){console.error('Data calculator',e);}
}
