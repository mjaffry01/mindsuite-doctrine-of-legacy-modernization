/* MindSuite web app · project mode (README: "Three kinds of project")
   Every project is one of: modernize an existing system, build a new product, or modernize and extend.
   The mode is stored per project in M.mode. Projects saved before modes existed count as "modernize".
   It changes only what has to differ:
     - the opening screen offers the two starting paths (modePage);
     - "Record evidence" becomes "Product brief" for a new product, and the brief (with deployment type)
       is asked on the project step for new and extend projects (productBriefSection);
     - the approaches offered: the 7 Rs, build / buy / partner, or both (approachAllowed, modeSuggest);
     - product facts for the Target options team (productFacts).
   Loaded by index.html before the main script; uses its helpers at call time. */

const PROJECT_MODES=[
 ['modernize','Modernize an existing system','We study the code, data and operations you have today, then decide per business area: rehost, replatform, refactor, rebuild, replace, encapsulate or retire.'],
 ['new','Build a new product','We start from your product brief: problem, users, first release, scale, deployment and constraints. Then build, buy or partner.'],
 ['extend','Modernize and extend','Both: understand the existing system with evidence, and design the new capabilities it must gain, on one plan.']];
const MODE_APPROACHES={modernize:['rehost','replatform','refactor','rebuild','replace','encapsulate','retire','undecided'],
 new:['build','buy','partner','undecided'],extend:['rehost','replatform','refactor','rebuild','replace','encapsulate','retire','build','buy','partner','undecided']};
const modeOf=()=>M.mode||'modernize';
const isNewProduct=()=>modeOf()==='new';
const hasProductBrief=()=>modeOf()!=='modernize';
const approachAllowed=id=>(MODE_APPROACHES[modeOf()]||MODE_APPROACHES.modernize).includes(id);
function projectIsEmpty(){
 const inv=M.inventory||{};
 return !String(M.project||'').trim()&&!(M.files||[]).length&&!Object.keys(M.values||{}).length&&!Object.values(inv).some(l=>Array.isArray(l)&&l.length);
}
const needsModeChoice=()=>openedWorkspace.entry||(!M.mode&&projectIsEmpty());
function setMode(mode){
 const before=M.mode;
 ensureDiscovery();
 M.mode=mode;
 logInput('project:mode',before,mode);
 if(openedWorkspace.entry){
  try{
   writeProject(workspaceId,M);
   location.assign(projectAddress(workspaceId));
  }catch{
   M.mode=before;
   M.history.pop();
   notice('The project could not be saved. Check that browser storage is available and try again.');
  }
  return;
 }
 persist();
 go('evidence');
 showEvidenceStep('project');
}

/* ---- Opening screen ---- */
const ENTRY_PATHS=[
 ['new','Project from scratch','Define the problem, who it serves, and what the first release should deliver.','✦'],
 ['modernize','Legacy modernization','Assess the code, data, and operations behind a system you already run.','↻']
];
function modePage(){
 return h('section',{class:'mode-page','aria-labelledby':'mode-title'},[
  h('h1',{id:'mode-title',text:'How are you starting?'}),
  h('div',{class:'mode-grid'},ENTRY_PATHS.map(([id,title,description,icon])=>h('button',{
   type:'button',class:'mode-card','data-mode':id,'aria-labelledby':'mode-'+id,'aria-describedby':'mode-description-'+id,
   onclick:()=>setMode(id)
  },[
   h('span',{class:'mode-icon','aria-hidden':'true',text:icon}),
   h('span',{class:'mode-card-title',id:'mode-'+id,text:title}),
   h('span',{class:'mode-card-description',id:'mode-description-'+id,text:description}),
   h('span',{class:'mode-arrow','aria-hidden':'true',text:'→'})
  ])))
 ]);
}

/* ---- Product brief (new and extend projects) ---- */
const DEPLOY_WHERE=[['public','Public cloud'],['private','Private cloud'],['onprem','On-premises (own data centre)'],['hybrid','Hybrid: cloud and on-premises'],
 ['multicloud','More than one public cloud'],['edge','Edge or on-site (plant, branch, device)'],['undecided','Not decided yet']];
const DEPLOY_TENANCY=[['internal','Internal use only (one organization)'],['saas-multi','SaaS, multi-tenant (customers share one deployment)'],
 ['saas-single','SaaS, single-tenant (one deployment per customer)'],['customer-hosted','Installed and run by each customer'],['undecided','Not decided yet']];
const DEPLOY_CHANNELS=[['web','Web browser'],['mobile','Mobile apps (app stores)'],['desktop','Desktop installer'],['api','API only (no user interface)'],['embedded','Device or embedded']];
const BRIEF_TEXT=[
 ['problem','Problem and opportunity *','What is wrong or missing today, and for whom','Example: Store staff track stock on paper; counts are wrong and orders arrive late'],
 ['users','Users and their jobs *','User groups and the jobs they need done','Example: Store staff: count and move stock; buyers: reorder; managers: see stock value'],
 ['journeys','Key journeys','The few journeys that must work end to end','Example: Receive a delivery; count a shelf; raise a purchase order'],
 ['mvp','First release (MVP) *','The must-have capabilities, and what is deliberately left out','Example: Stock in/out, counts, low-stock alerts. Not yet: forecasting, supplier portal'],
 ['success','Success measures','How success is measured and by when','Example: Stock accuracy above 98% within 3 months of launch'],
 ['integrations','Integrations','Systems it must connect to','Example: Accounting (Xero), supplier EDI, single sign-on with Microsoft Entra ID'],
 ['buyPref','Build, buy or partner: known preferences','Existing products, contracts or preferences','Example: We already pay for Dynamics 365; prefer not to build accounting']];
const BRIEF_NUM=[['users_count','Expected users','number','Example: 400'],['peak','Peak requests per second (estimate)','number','Example: 30'],
 ['dataGb','Data volume after 3 years (GB)','number','Example: 200'],['availability','Availability target (%)','number','Example: 99.9']];
(function(){try{const st=document.createElement('style');st.textContent='[placeholder^="Example:"]::placeholder{font-style:italic;color:#9a9cb0;opacity:1}';document.head.append(st);}catch(e){}})();
function productBrief(){M.product=M.product||{};const P=M.product;P.deploy=P.deploy||{channels:[]};return P;}
function productBriefSection(){
 const numFields=keys=>BRIEF_NUM.filter(([k])=>keys.includes(k)).map(([k,label,type,ph])=>editable(k,inputControl(label,visibleBriefValue(k),v=>{writeBriefAnswer(k,v);if(typeof refreshLoadCalc==='function'&&(k==='users_count'||k==='peak'))refreshLoadCalc();if(typeof refreshDataCalc==='function'&&(k==='users_count'||k==='dataGb'))refreshDataCalc();},type,ph)));
 const saveChoice=(key,value)=>{
  const before=briefAnswer(key);writeBriefAnswer(key,value);logInput('product:'+key,before,value);persist();render();
  if(formfillUI.afterEdit&&before!==briefAnswer(key))runFormfiller();
 };
 const editable=(key,control)=>{
  let before=briefAnswer(key);
  const field=control.querySelector('textarea,input');
  field.dataset.briefKey=key;
  let edited=false;
  field.addEventListener('input',()=>{edited=true;});
  const finished=()=>{
   if(!edited||!field.value.trim()||!formfillUI.afterEdit||!field.isConnected||document.hidden)return;
   edited=false;
   runFormfiller();
  };
  field.addEventListener('dictationend',finished);
  field.addEventListener('change',()=>{
   const after=briefAnswer(key);logInput('product:'+key,before,after);before=after;persist();
   if(!field.closest('.dictation-control')?.querySelector('.dictation-mic[aria-pressed="true"]'))finished();
  });
  return briefField(key,control);
 };
 const pick=(key,label,choices)=>briefField(key,selectControl(label,visibleBriefValue(key),[['','Not answered'],...choices],v=>saveChoice(key,v)));
 const channels=visibleBriefValue('channels').split(',').map(v=>v.trim()).filter(Boolean);
 const where=visibleBriefValue('where');
 const done=BRIEF_TEXT.filter(([k])=>briefAnswer(k)).length+(briefAnswer('where')?1:0);
 return h('section',{class:'scope-panel rec-step product-brief',id:'product-brief'},[
  h('div',{class:'discovery-section-head'},[h('div',{},[h('h2',{text:isNewProduct()?'Product brief':'What the system must become'}),
   h('p',{text:isNewProduct()?'These answers guide design decisions, sizing and technical options.':'Describe the new capabilities this modernization must add.'})]),
   h('span',{class:'input-status '+(done>=5?'done':'open'),text:done+' of '+(BRIEF_TEXT.length+1)+' answered'})]),
  formfillerPanel(),
  h('div',{class:'brief-grid'},BRIEF_TEXT.map(([k,label,hint,ph])=>editable(k,h('label',{class:'context-control'},[h('span',{text:label}),
   h('textarea',{class:'field',rows:'2',maxlength:'600',placeholder:ph,title:hint,oninput:e=>writeBriefAnswer(k,e.target.value)},[visibleBriefValue(k)])])))),
  /* Load first: expected users and peak, with the suggested peak right under them; then data and availability. */
  h('h3',{class:'brief-sub',text:'Load'}),
  h('div',{class:'scope-grid'},numFields(['users_count','peak'])),
  typeof loadCalcPanel==='function'?loadCalcPanel():null,
  h('h3',{class:'brief-sub',text:'Data and availability'}),
  h('div',{class:'scope-grid'},numFields(['dataGb'])),
  typeof dataCalcPanel==='function'?dataCalcPanel():null,
  h('div',{class:'scope-grid'},numFields(['availability'])),
  (()=>{const PB=productBrief();const R=PB.recovery=PB.recovery||{};const f=(k,label,ph)=>inputControl(label,R[k]||'',v=>{const b=R[k];R[k]=String(v||'').trim();logInput('product:recovery:'+k,b,R[k]);persist();},'text',ph);
   return h('div',{class:'scope-grid recovery-grid'},[f('rto','Recovery time (RTO): how long may the service be down after an outage?','Example: 4 hours'),f('rpo','Data loss allowed (RPO): how much recent data may be lost?','Example: 15 minutes')]);})(),
  h('div',{class:'deploy-block'},[h('h3',{text:'Deployment type'}),h('p',{class:'muted',text:'Where it runs, who it serves, how it reaches users and how it is hosted.'}),
   h('div',{class:'scope-grid'},[
    pick('where','Where it runs *',DEPLOY_WHERE),pick('tenancy','Tenancy',DEPLOY_TENANCY),
    ['public','multicloud','hybrid'].includes(where)||visibleBriefValue('regions')?editable('regions',inputControl('Regions or countries the data must stay in',visibleBriefValue('regions'),v=>writeBriefAnswer('regions',v),'text','Example: EU only; India (data residency)')):null]),
   briefField('channels',h('div',{class:'type-chips'},[h('span',{class:'type-label',text:'How it reaches users'}),...DEPLOY_CHANNELS.map(([id,label])=>h('label',{class:'type-chip'+(channels.includes(id)?' on':'')},[
    h('input',{type:'checkbox',checked:channels.includes(id)?'':null,onchange:e=>{const next=new Set(channels);if(e.target.checked)next.add(id);else next.delete(id);saveChoice('channels',[...next]);}}),label]))])),
   typeof hostingModelPicker==='function'?hostingModelPicker():null])
 ]);
}

/* Deployment type for a modernization project: where the target runs (asked in the project interview step). */
function deploymentOnlySection(){
 const D=productBrief().deploy;const setD=k=>v=>{const b=D[k];D[k]=v;logInput('product:deploy:'+k,b,v);persist();render();};
 return h('section',{class:'scope-panel rec-step deploy-only'},[h('div',{class:'discovery-section-head'},[h('div',{},[h('h2',{text:'Target deployment type'}),
  h('p',{text:'Where the modernized system should run and who it serves. Leave it open if this is what you want the assessment to decide.'})])]),
  h('div',{class:'scope-grid'},[selectControl('Where it should run',D.where||'',[['','Let the assessment decide'],...DEPLOY_WHERE],setD('where')),
   selectControl('Tenancy',D.tenancy||'',[['','Not answered'],...DEPLOY_TENANCY],setD('tenancy'))]),
  typeof hostingModelPicker==='function'?hostingModelPicker():null]);
}

/* ---- Facts for the Target options team ---- */
function productFacts(add){
 const P=M.product||{},D=P.deploy||{};const lbl=(l,id)=>(l.find(x=>x[0]===id)||[,id])[1];
 if(hasProductBrief())add('product','Project type: '+lbl(PROJECT_MODES,modeOf()));
 for(const [k,label] of BRIEF_TEXT){const v=String(P[k]||'').trim();if(v)add('product',label.replace(' *','')+': '+v.slice(0,260));}
 for(const [k,label] of BRIEF_NUM)if(String(P[k]||'').trim())add('capacity',label+': '+P[k]+(k==='peak'&&typeof peakFactNote==='function'?peakFactNote(P):'')+(k==='dataGb'&&typeof dataFactNote==='function'?dataFactNote(P):''));
 if(D.where)add('deployment','Deployment: runs on '+lbl(DEPLOY_WHERE,D.where)+(D.regions?'; data must stay in '+D.regions:''));
 if(D.tenancy)add('deployment','Tenancy: '+lbl(DEPLOY_TENANCY,D.tenancy));
 if((D.channels||[]).length)add('deployment','Reaches users through: '+D.channels.map(x=>lbl(DEPLOY_CHANNELS,x)).join(', '));
 if(typeof hostingFacts==='function')hostingFacts(add);
}

/* ---- Approach suggestion for new capabilities ---- */
function modeSuggest(c){
 if(modeOf()==='modernize')return null;
 if(modeOf()==='new'||c.virtual&&modeOf()==='extend'&&!(M.files||[]).length)
  return c.strategic==='core'?['build','Core: custom work pays off where the business is different']:c.strategic==='generic'?['buy','Generic: a product or SaaS already does this']:c.strategic==='supporting'?['partner','Supporting: extend a platform or partner product for speed']:null;
 return null;
}

(function modeStyle(){
 if(document.getElementById('project-mode-style'))return;
 const css=`
body.project-entry{--entry-header:72px;background:#111322;color:#fcfbff}
body.project-entry .app-top{
 position:relative;height:var(--entry-header);background:#111322;border-color:#ffffff18;color:#fcfbff;
}
body.project-entry .top-right{margin-left:auto}
body.project-entry .brand-mark{background:#ded9ff;color:#242045}
body.project-entry .brand small,
body.project-entry .layout>aside,
body.project-entry #workspaceMenu,
body.project-entry .workspace-stages,
body.project-entry #projectName,
body.project-entry #newProject,
body.project-entry #projectList[hidden],
body.project-entry .nav-backdrop{display:none!important}
body.project-entry #projectList{background:#ffffff0a;border-color:#ffffff30;color:#edeaff}
body.project-entry #projectList:hover{background:#ffffff17}
body.project-entry .layout{display:block}
body.project-entry main{
 min-height:calc(100vh - var(--entry-header));min-height:calc(100svh - var(--entry-header));max-width:none;
 padding:clamp(48px,11vh,130px) clamp(16px,6vw,72px) 70px;
 background:radial-gradient(ellipse at 20% 0%,#292148 0%,transparent 56%),
            radial-gradient(ellipse at 88% 40%,#30232c 0%,transparent 58%);
}
.mode-page{max-width:1080px;margin:0 auto}
.mode-page h1{
 margin:0 0 clamp(28px,5vw,50px);color:#fcfbff;font-size:clamp(34px,6vw,76px);
 font-weight:750;letter-spacing:-.055em;line-height:1;
}
.mode-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:clamp(14px,2.2vw,24px)}
.mode-card{
 --mode-accent:#d1bdff;--mode-surface:#25203a;--mode-line:#a48ee2;
 position:relative;isolation:isolate;display:flex;flex-direction:column;align-items:flex-start;
 gap:10px;min-height:250px;overflow:hidden;padding:clamp(20px,3vw,34px);
 border:1px solid var(--mode-line);border-color:color-mix(in srgb,var(--mode-line) 48%,transparent);
 border-radius:18px;background:var(--mode-surface);color:#fcfbff;text-align:left;font:inherit;
 cursor:pointer;transition:transform .18s ease,border-color .18s ease,box-shadow .18s ease;
}
.mode-card[data-mode="modernize"]{--mode-accent:#ffd3a0;--mode-surface:#332529;--mode-line:#dba57f}
.mode-card::before{
 content:'';position:absolute;z-index:-1;top:-112px;right:-79px;width:275px;height:275px;
 border:1px solid var(--mode-line);border-radius:50%;
 box-shadow:0 0 0 30px color-mix(in srgb,var(--mode-accent) 25%,transparent),
            0 0 0 60px color-mix(in srgb,var(--mode-accent) 16%,transparent);
 opacity:.24;pointer-events:none;
}
.mode-card:hover{transform:translateY(-4px);border-color:var(--mode-line);box-shadow:0 18px 42px #08071066}
.mode-card:focus-visible,body.project-entry #projectList:focus-visible{outline:2px solid #fcfbff;outline-offset:4px}
.mode-icon{
 display:grid;place-items:center;margin-bottom:auto;width:46px;height:46px;
 border:1px solid color-mix(in srgb,var(--mode-accent) 34%,transparent);
 border-radius:12px;background:color-mix(in srgb,var(--mode-accent) 10%,transparent);
 color:var(--mode-accent);font-size:24px;
}
.mode-card-title{font-size:clamp(23px,3.2vw,34px);font-weight:720;letter-spacing:-.035em;line-height:1.1}
.mode-card-description{max-width:31ch;padding-right:12px;color:#e6dfea;font-size:15px;line-height:1.5}
.mode-arrow{
 position:absolute;right:clamp(14px,2.7vw,28px);top:clamp(18px,3vw,34px);
 color:var(--mode-accent);font-size:26px;line-height:1;
}
@media(max-width:640px){
 body.project-entry{--entry-header:64px}
 .mode-grid{grid-template-columns:1fr}
 .mode-card{min-height:176px;gap:8px}
 .mode-icon{width:36px;height:36px;margin-bottom:8px;font-size:20px}
}
@media(prefers-reduced-motion:reduce){.mode-card{transition:none}.mode-card:hover{transform:none}}
.brief-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(min(100%,300px),1fr));gap:10px 14px;margin:10px 0}

.formfiller-panel{display:flex;align-items:center;justify-content:space-between;flex-wrap:wrap;gap:12px;padding:16px;margin:16px 0 20px;border:1px solid #d9d3f3;border-radius:12px;background:#f8f6ff}
.formfiller-panel>div:first-child{flex:1 1 300px}.formfiller-panel h3{margin:0}.formfiller-panel p{margin:5px 0;color:#514c62}.formfiller-panel small:empty{display:none}
.formfiller-auto{display:flex;align-items:center;gap:7px;font-size:12px;width:fit-content;margin:8px 0}.formfiller-auto input{width:auto}.formfiller-actions{display:flex;flex-wrap:wrap;gap:8px;align-items:center}
.brief-proposal{min-width:0;border-left:3px solid #9887db;padding-left:9px}.brief-proposal-note{display:grid;gap:4px;font-size:12px;color:#544776;margin:4px 0 12px}.brief-proposal-note small{overflow-wrap:anywhere}

.brief-grid textarea{resize:vertical}
.deploy-block{margin-top:14px;padding-top:12px;border-top:1px solid var(--line)}.deploy-block h3{margin:0 0 4px;font-size:15px}
.deploy-block .type-chips{display:flex;flex-wrap:wrap;gap:6px;align-items:center;margin-top:8px}
.mode-badge{display:inline-block;font-size:11px;font-weight:700;border-radius:999px;padding:2px 8px;background:#e5ecff;color:var(--accent);margin-left:6px}`;
 document.head.append(Object.assign(document.createElement('style'),{id:'project-mode-style',textContent:css}));
})();

/* Project type, changeable later (shown on the project interview step). */
function modeSwitcher(){
 return h('div',{class:'mode-switch'},[selectControl('Project type',modeOf(),PROJECT_MODES.map(([id,t])=>[id,t]),v=>{
  if(v===modeOf())return;const b=M.mode||'modernize';M.mode=v;logInput('project:mode',b,v);persist();evidenceUI.step='project';go('upload');notice('Project type: '+PROJECT_MODES.find(x=>x[0]===v)[1]+'.');})]);
}

/* Sidebar hint for "01 Product brief". */
function briefProgress(){
 const P=M.product||{},D=P.deploy||{};const n=BRIEF_TEXT.filter(([k])=>String(P[k]||'').trim()).length+(D.where?1:0);
 return n?n+' of '+(BRIEF_TEXT.length+1)+' answered':'Describe the product';
}
