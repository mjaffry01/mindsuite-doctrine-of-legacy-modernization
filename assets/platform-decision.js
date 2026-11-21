/* MindSuite web app · Step 1 on Design options / Target decisions: the platform decision
   Doctrine: architects decide the shared platform once; each capability (or bounded context) then chooses
   only its route (build on the platform, buy, partner, or for an existing system rehost / replatform /
   refactor / rebuild / replace / encapsulate / retire) and inherits the platform's technology.
   The platform decision holds: technology tier (Safe / Balanced / Ambitious = how much platform you run and
   how far it scales, NOT build vs buy), where it runs, cloud, hosting model, architecture, language,
   database and integration style. Suggestions come from the brief and the organization rules (rankOptions
   on a virtual whole-system context); the brief's hosting answer is only a preference that pre-fills it.
   Stored in M.decide.platform {values:{...}, status:'decided'|'open', by, at, rationale}.
   platformEffective() is what the routes, blueprint, plan, cost, assumptions and HLD use: the recorded
   values, else the suggestion (and says which).
   Hook: decidePage() in index.html shows platformPanel() first and a "Step 2" heading before the cards. */

const PLATFORM_DIMS=['cloud','architecture','language','database','integration'];
const PLATFORM_TIERS=[
 ['Safe','Least to run','A managed app platform or a few fixed servers, one database with a standby, background jobs in the app. Fewest moving parts; scales by adding bigger servers.'],
 ['Balanced','Scales without fuss','Containers that scale automatically, a queue for slow work, a cache and a read replica. A small platform team can run it.'],
 ['Ambitious','Built to grow','Kubernetes or many services, event streaming, several zones, a policy service. Needs a platform team; most room to grow.']];
const PLATFORM_VCTX={id:'platform',name:'Whole platform',virtual:true};

function platformRec(){ensureDecide();const D=M.decide;D.platform=D.platform||{values:{},status:'open'};D.platform.values=D.platform.values||{};return D.platform;}

function platformSuggest(){
 const P=M.product||{},Dp=P.deploy||{},why={};
 let d={approach:'build',choices:{integration:[]}};
 try{d=buildScenario(PLATFORM_VCTX,isNewProduct()?'build':'refactor');}catch(e){}
 const v={};for(const dim of PLATFORM_DIMS)v[dim]=d.choices[dim]||(TARGETS[dim].multi?[]:'');
 // The platform is the server side: device-side options (embedded SQLite, the client-app pattern) do not apply here.
 const rank=(dim,skip)=>{try{return rankOptions(dim,PLATFORM_VCTX,{approach:'build',choices:{}}).filter(o=>!o.excluded&&!skip.includes(o.id));}catch(e){return [];}};
 const dbr=rank('database',['sqlite','keep']);
 if(dbr.length){const top=dbr[0],tied=dbr.filter(o=>o.score===top.score);
  v.database=tied.some(o=>o.id==='postgres')&&tied.length>1?'postgres':top.id;
  why.database=tied.length>1?(tied.every(o=>(o.reasons||[]).some(r=>r.w==='rule'))?'your rules prefer '+tied.map(o=>o.label).join(' and ')+' equally; '+(v.database==='postgres'?'PostgreSQL is taken because it is managed on every cloud and has no licence cost':(optOf('database',v.database)||[,v.database])[1]+' scores first'):'no rule prefers one; '+(v.database==='postgres'?'PostgreSQL is managed on every cloud and has no licence cost':(optOf('database',v.database)||[,v.database])[1]+' scores first')):((top.reasons||[])[0]||{}).t||'best score against your rules';}
 const ar=rank('architecture',['clientapi','keep']);
 if(ar.length){v.architecture=ar[0].id;why.architecture=(((ar[0].reasons||[])[0]||{}).t||'best score against your rules')+'; web and mobile clients call its API';}
 const cr=rank('cloud',['keep']);
 if(cr.length){const tied=cr.filter(o=>o.score===cr[0].score);v.cloud=cr[0].id;
  why.cloud=tied.length>1?tied.length+' clouds fit equally ('+tied.map(o=>(optOf('cloud',o.id)||[,o.id])[1]).join(', ')+'): choose by your contracts, skills and regions':(((cr[0].reasons||[])[0]||{}).t||'best score');}
 if(Dp.where&&['private','onprem'].includes(Dp.where)&&!v.cloud){v.cloud=Dp.where;}
 if(Dp.where&&!why.cloud){why.cloud='where it runs: '+Dp.where;}
 // Tier from scale, availability, risk appetite and team.
 let I=null;try{I=techPlanInputs(PLATFORM_VCTX);}catch(e){}
 const K=constraints(),peak=I&&I.peak||0,avail=I?I.avail:99.5,skills=(M.decide.rules.skills||[]).length;
 let tier='Balanced',tw=[];
 if(peak>=200||avail>=99.95){tier='Ambitious';tw.push(peak>=200?'peak '+Math.round(peak)+' requests/s':'availability '+avail+'%');}
 else if((peak&&peak<20)&&avail<99.9){tier='Safe';tw.push('light load ('+Math.round(peak)+' requests/s)');}
 if(K.risk==='low'&&tier!=='Safe'){tier=tier==='Ambitious'?'Balanced':'Safe';tw.push('low risk appetite');}
 if(tier==='Ambitious'&&skills<2){tier='Balanced';tw.push('few team skills recorded for running a large platform');}
 if(!tw.length)tw.push('moderate load'+(peak?' ('+Math.round(peak)+' requests/s)':'')+' and '+avail+'% availability');
 v.tier=tier;why.tier=tw.join(', ');
 // Hosting model: the brief's preference, else the hosting suggestion, matched to the tier.
 if(Dp.hosting){v.hosting=Dp.hosting;why.hosting='your preference in the brief';}
 else{let s=[];try{s=hostingSuggest(P);}catch(e){}
  const byTier={Safe:['paas','vm','static','lowcode'],Balanced:['containers','paas'],Ambitious:['kubernetes','containers']}[tier];
  const pick=s.find(x=>byTier.includes(x.id))||s[0];v.hosting=pick?pick.id:{Safe:'paas',Balanced:'containers',Ambitious:'kubernetes'}[tier];
  why.hosting=pick?'fits the brief ('+(pick.why[0]||'best fit')+') and the '+tier+' tier':'typical for the '+tier+' tier';
  // A compute service your rules make Standard (or Preferred) decides the hosting model.
  if(typeof csRules==='function'&&typeof CLOUD_SERVICES!=='undefined'){
   const prov=csProvider(v.cloud),rules=csRules()[prov]||{},st=s=>(rules[s[0]]||{}).state;
   const comp=(CLOUD_SERVICES[prov]||[]).filter(s=>s[2]==='compute');
   const hit=comp.find(s=>st(s)==='standard')||comp.find(s=>st(s)==='preferred');
   const roleHost={k8s:'kubernetes',ctr:'containers',paas:'paas',fn:'serverless',vm:'vm',static:'static'};
   const hm=hit&&roleHost[hit[3].split(',')[0]];
   if(hm){v.hosting=hm;why.hosting='your rules make '+hit[1]+' the '+(st(hit)==='standard'?'standard':'preferred')+' compute service';}
  }}
 return {v,why};
}
function platformEffective(){
 const R=platformRec(),S=platformSuggest();
 const v={...S.v};for(const [k,x] of Object.entries(R.values))if(x!=null&&x!==''&&!(Array.isArray(x)&&!x.length))v[k]=x;
 return {...v,decided:R.status==='decided',by:R.by,at:R.at,suggested:S};
}
/* Technology choices a route inherits from the platform (only the dimensions that route uses). */
function platformChoicesFor(approach){
 const p=platformEffective(),fixed=APPROACH_FIXED[approach]||{},out={integration:[]};
 for(const dim of PLATFORM_DIMS){if(!(APPROACH_DIMS[approach]||[]).includes(dim)||fixed[dim])continue;out[dim]=Array.isArray(p[dim])?[...p[dim]]:p[dim];}
 return out;
}

function platformPanel(){
 const R=platformRec(),S=platformSuggest(),E=platformEffective(),product=isNewProduct();
 const vals=R.values,decided=R.status==='decided';
 const set=k=>v=>{const b=vals[k];vals[k]=v;if(decided){R.status='open';R.reopened=true;}logInput('platform:'+k,b,v);persist();render();};
 const optLabel=(dim,id)=>id?((optOf(dim,id)||[,id])[1]):'';
 const sugg=(k,lab)=>h('small',{class:'muted pf-sugg',text:'Suggested: '+lab+(S.why[k]?' — '+S.why[k]:'')});
 const sel=(dim)=>{const cur=vals[dim]||'';return h('label',{class:'context-control'},[h('span',{text:TARGETS[dim].title}),
  h('select',{class:'field',onchange:e=>set(dim)(e.target.value)},[h('option',{value:'',text:'Use suggestion ('+(optLabel(dim,S.v[dim])||'none')+')'}),...TARGETS[dim].opts.map(([id,l])=>h('option',{value:id,text:l,selected:cur===id?'':null}))]),
  sugg(dim,optLabel(dim,S.v[dim])||'none')]);};
 const integ=[].concat(vals.integration&&vals.integration.length?vals.integration:S.v.integration||[]);
 const host=vals.hosting||'';
 const people=M.people||[];
 const who=h('select',{class:'field compact','aria-label':'Decided by'},[h('option',{value:'',text:'Decided by…'}),...people.map(p=>h('option',{value:p.id,text:p.name+' · '+p.role}))]);who.value=R.by||M.actor||'';
 const rat=h('input',{class:'field',placeholder:'Why this platform (optional)',maxlength:'300',value:R.rationale||''});
 const record=()=>{if(!who.value){who.focus();notice('Choose who records the platform decision.');return;}
  const b=R.status;for(const dim of PLATFORM_DIMS)if(vals[dim]==null||vals[dim]===''||(Array.isArray(vals[dim])&&!vals[dim].length))vals[dim]=Array.isArray(S.v[dim])?[...S.v[dim]]:S.v[dim];
  for(const k of ['tier','hosting'])if(!vals[k])vals[k]=S.v[k];
  R.status='decided';R.by=who.value;R.at=new Date().toISOString();R.rationale=rat.value.trim();R.reopened=false;logInput('platform:decision',b,'decided');persist();render();notice('Platform decision recorded. Each route below now uses it.');};
 return h('section',{class:'summary-section platform-step'},[
  h('div',{class:'sb-head'},[h('div',{},[h('small',{class:'eyebrow',text:'Step 1 of 2'}),h('h2',{text:'Platform decision'}),
   h('p',{class:'muted',text:'Decide the shared platform once: how much platform to run, where, and with which technology. Every '+(product?'capability':'context')+' in step 2 then only chooses its route and uses this platform.'})]),
   h('span',{class:'input-status '+(decided?'done':'open'),text:decided?'Decided by '+(personLabel(R.by)||R.by)+', '+String(R.at).slice(0,10):R.reopened?'Changed: record it again':'Not decided: the plan uses the suggestion'})]),
  h('div',{class:'pf-tiers'},PLATFORM_TIERS.map(([id,tag,desc])=>{const on=(vals.tier||S.v.tier)===id;
   return h('button',{type:'button',class:'pf-tier'+(on?' on':''),onclick:()=>set('tier')(id)},[h('div',{class:'hosting-title'},[h('b',{text:id}),h('span',{class:'hosting-kind',text:tag}),S.v.tier===id?h('span',{class:'hosting-badge',text:'Suggested'}):null]),h('p',{text:desc}),
    S.v.tier===id?h('p',{class:'hosting-why',text:'Why: '+S.why.tier+'.'}):null]);})),
  h('div',{class:'rules-grid pf-grid'},[
   h('label',{class:'context-control'},[h('span',{text:'Hosting model'}),
    h('select',{class:'field',onchange:e=>set('hosting')(e.target.value)},[h('option',{value:'',text:'Use suggestion ('+(((typeof HOSTING_MODELS!=='undefined'?HOSTING_MODELS:[]).find(m=>m.id===S.v.hosting)||{}).name||'none')+')'}),...(typeof HOSTING_MODELS!=='undefined'?HOSTING_MODELS:[]).map(m=>h('option',{value:m.id,text:m.name,selected:host===m.id?'':null}))]),
    sugg('hosting',(((typeof HOSTING_MODELS!=='undefined'?HOSTING_MODELS:[]).find(m=>m.id===S.v.hosting)||{}).name)||'none')]),
   sel('cloud'),sel('architecture'),sel('language'),sel('database')]),
  h('div',{class:'pf-integ'},[h('span',{class:'type-label',text:TARGETS.integration.title+' (suggested: '+((S.v.integration||[]).map(id=>optLabel('integration',id)).join(', ')||'none')+')'}),
   h('div',{class:'type-chips'},TARGETS.integration.opts.map(([id,l])=>h('label',{class:'type-chip'+(integ.includes(id)?' on':'')},[
    h('input',{type:'checkbox',checked:integ.includes(id)?'':null,onchange:e=>{const n=new Set(integ);if(e.target.checked)n.add(id);else n.delete(id);set('integration')([...n]);}}),l])))]),
  h('div',{class:'cost-actions'},[people.length?who:button('Add people',()=>go('rules'),'btn small'),rat,people.length?button(decided?'Record again':'Record platform decision',record,'btn small primary'):null,
   decided?button('Reopen',()=>{R.status='open';R.reopened=true;logInput('platform:decision','decided','open');persist();render();},'linkbtn'):null])]);
}
function platformStep2Head(){
 const p=platformEffective(),product=isNewProduct();
 return h('div',{class:'platform-step2'},[h('small',{class:'eyebrow',text:'Step 2 of 2'}),h('h2',{text:'Route for each '+(product?'capability':'context')}),
  h('p',{class:'muted',text:(p.decided?'Using the recorded platform':'Using the suggested platform (record it in step 1)')+': '+p.tier+' tier'+(p.cloud?', '+((optOf('cloud',p.cloud)||[,p.cloud])[1]):'')+(p.architecture?', '+((optOf('architecture',p.architecture)||[,p.architecture])[1]):'')+(p.database?', '+((optOf('database',p.database)||[,p.database])[1]):'')+'. Each option below is a route; the technology comes from the platform.'})]);
}

(function pfStyle(){
 if(document.getElementById('pf-style'))return;
 const css=`.platform-step{border:2px solid #cfc6ff}.pf-tiers{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:10px;margin:10px 0}
.pf-tier{display:flex;flex-direction:column;text-align:left;font:inherit;color:inherit;padding:10px 12px;border:1px solid #d9e1f2;border-radius:10px;background:#fff;cursor:pointer}
.pf-tier.on{border:2px solid var(--accent);background:#f6f4ff}.pf-tier p{margin:6px 0 0;font-size:13px}
.pf-grid{margin:8px 0}.pf-sugg{display:block;margin-top:3px;font-size:11px}.pf-integ{margin:8px 0}
.platform-step .cost-actions select.field{width:220px;flex:none}.platform-step .cost-actions input.field{flex:1 1 260px;width:auto;margin:0}
.platform-step2{margin:18px 0 8px}.platform-step2 h2{margin:2px 0}.platform-step2 p{margin:4px 0 0}
@media(max-width:900px){.pf-tiers{grid-template-columns:1fr}}`;
 document.head.append(Object.assign(document.createElement('style'),{id:'pf-style',textContent:css}));
})();
