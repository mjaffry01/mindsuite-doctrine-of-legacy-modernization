/* MindSuite web app · project mode (README: "Three kinds of project")
   Every project is one of: modernize an existing system, build a new product, or modernize and extend.
   The mode is stored per project in M.mode. Projects saved before modes existed count as "modernize".
   It changes only what has to differ:
     - the start screen asks for it on a new, empty project (modePage);
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
const needsModeChoice=()=>!M.mode&&projectIsEmpty();
function setMode(mode){
 const before=M.mode||'';M.mode=mode;
 if(typeof logInput==='function')logInput('project:mode',before,mode);
 persist();
 go('evidence');
 if(typeof showEvidenceStep==='function'){try{showEvidenceStep('project');}catch(e){}}
}

/* ---- Start screen ---- */
function modePage(){
 return h('div',{class:'mode-page'},[
  h('div',{class:'page-heading'},[h('div',{},[h('div',{class:'eyebrow',text:'NEW PROJECT'}),h('h1',{text:'What are you working on?'}),
   h('p',{text:'This sets how discovery starts and which decisions you make. Everything else is shared, and you can change it later in the project interview.'})])]),
  h('div',{class:'mode-grid'},PROJECT_MODES.map(([id,title,text])=>h('button',{type:'button',class:'mode-card','data-mode':id,onclick:()=>setMode(id)},[
   h('span',{class:'mode-icon',text:{modernize:'↻',new:'✦',extend:'↗'}[id]}),h('b',{text:title}),h('span',{class:'muted',text:text})]))),
  h('p',{class:'muted mode-note',text:'Most modernizations also add new capabilities. If yours does, choose Modernize and extend.'})]);
}

/* ---- Product brief (new and extend projects) ---- */
const DEPLOY_WHERE=[['public','Public cloud'],['private','Private cloud'],['onprem','On-premises (own data centre)'],['hybrid','Hybrid: cloud and on-premises'],
 ['multicloud','More than one public cloud'],['edge','Edge or on-site (plant, branch, device)'],['undecided','Not decided yet']];
const DEPLOY_TENANCY=[['internal','Internal use only (one organization)'],['saas-multi','SaaS, multi-tenant (customers share one deployment)'],
 ['saas-single','SaaS, single-tenant (one deployment per customer)'],['customer-hosted','Installed and run by each customer'],['undecided','Not decided yet']];
const DEPLOY_CHANNELS=[['web','Web browser'],['mobile','Mobile apps (app stores)'],['desktop','Desktop installer'],['api','API only (no user interface)'],['embedded','Device or embedded']];
const BRIEF_TEXT=[
 ['problem','Problem and opportunity *','What is wrong or missing today, and for whom','e.g. Store staff track stock on paper; counts are wrong and orders arrive late'],
 ['users','Users and their jobs *','User groups and the jobs they need done','e.g. Store staff: count and move stock; buyers: reorder; managers: see stock value'],
 ['journeys','Key journeys','The few journeys that must work end to end','e.g. Receive a delivery; count a shelf; raise a purchase order'],
 ['mvp','First release (MVP) *','The must-have capabilities, and what is deliberately left out','e.g. Stock in/out, counts, low-stock alerts. Not yet: forecasting, supplier portal'],
 ['success','Success measures','How success is measured and by when','e.g. Stock accuracy above 98% within 3 months of launch'],
 ['integrations','Integrations','Systems it must connect to','e.g. Accounting (Xero), supplier EDI, single sign-on with Microsoft Entra ID'],
 ['buyPref','Build, buy or partner: known preferences','Existing products, contracts or preferences','e.g. We already pay for Dynamics 365; prefer not to build accounting']];
const BRIEF_NUM=[['users_count','Expected users','number','e.g. 400'],['peak','Peak requests per second (estimate)','number','e.g. 30'],
 ['dataGb','Data volume after 3 years (GB)','number','e.g. 200'],['availability','Availability target (%)','number','e.g. 99.9']];
function productBrief(){M.product=M.product||{};const P=M.product;P.deploy=P.deploy||{channels:[]};return P;}
function productBriefSection(){
 const P=productBrief(),D=P.deploy;
 const set=k=>v=>{const b=P[k];P[k]=v;logInput('product:'+k,b,v);persist();};
 const setD=k=>v=>{const b=D[k];D[k]=v;logInput('product:deploy:'+k,b,v);persist();render();};
 const done=BRIEF_TEXT.filter(([k])=>String(P[k]||'').trim()).length+(D.where?1:0);
 return h('section',{class:'scope-panel rec-step product-brief',id:'product-brief'},[
  h('div',{class:'discovery-section-head'},[h('div',{},[h('h2',{text:isNewProduct()?'Product brief':'What the system must become'}),
   h('p',{text:isNewProduct()?'A new product has no code to scan: these answers are the facts the advisor and the agent team work from.':'The new capabilities this modernization must add. They are decided as build, buy or partner, next to the existing parts.'})]),
   h('span',{class:'input-status '+(done>=5?'done':'open'),text:done+' of '+(BRIEF_TEXT.length+1)+' answered'})]),
  h('div',{class:'brief-grid'},BRIEF_TEXT.map(([k,label,hint,ph])=>h('label',{class:'context-control'},[h('span',{text:label}),
   h('textarea',{class:'field',rows:'2',maxlength:'600',placeholder:ph,title:hint,oninput:e=>{P[k]=e.target.value;persist();},onchange:e=>logInput('product:'+k,null,e.target.value)},[P[k]||''])]))),
  h('div',{class:'scope-grid'},BRIEF_NUM.map(([k,label,type,ph])=>inputControl(label,P[k],set(k),type,ph))),
  h('div',{class:'deploy-block'},[h('h3',{text:'Deployment type'}),h('p',{class:'muted',text:'Where it runs, who it serves and how it reaches users. The agent team only proposes hosting that fits these answers.'}),
   h('div',{class:'scope-grid'},[
    selectControl('Where it runs *',D.where||'',[['','Not answered'],...DEPLOY_WHERE],setD('where')),
    selectControl('Tenancy',D.tenancy||'',[['','Not answered'],...DEPLOY_TENANCY],setD('tenancy')),
    D.where==='public'||D.where==='multicloud'||D.where==='hybrid'?inputControl('Regions or countries the data must stay in',D.regions,v=>{const b=D.regions;D.regions=v;logInput('product:deploy:regions',b,v);persist();},'text','e.g. EU only; India (data residency)'):null]),
   h('div',{class:'type-chips'},[h('span',{class:'type-label',text:'How it reaches users'}),...DEPLOY_CHANNELS.map(([id,label])=>h('label',{class:'type-chip'+((D.channels||[]).includes(id)?' on':'')},[
    h('input',{type:'checkbox',checked:(D.channels||[]).includes(id)?'':null,onchange:e=>{const s=new Set(D.channels||[]);e.target.checked?s.add(id):s.delete(id);const b=D.channels;D.channels=[...s];logInput('product:deploy:channels',b,D.channels);persist();render();}}),label]))])])]);
}
/* Deployment type for a modernization project: where the target runs (asked in the project interview step). */
function deploymentOnlySection(){
 const D=productBrief().deploy;const setD=k=>v=>{const b=D[k];D[k]=v;logInput('product:deploy:'+k,b,v);persist();render();};
 return h('section',{class:'scope-panel rec-step deploy-only'},[h('div',{class:'discovery-section-head'},[h('div',{},[h('h2',{text:'Target deployment type'}),
  h('p',{text:'Where the modernized system should run and who it serves. Leave it open if this is what you want the assessment to decide.'})])]),
  h('div',{class:'scope-grid'},[selectControl('Where it should run',D.where||'',[['','Let the assessment decide'],...DEPLOY_WHERE],setD('where')),
   selectControl('Tenancy',D.tenancy||'',[['','Not answered'],...DEPLOY_TENANCY],setD('tenancy'))])]);
}

/* ---- Facts for the Target options team ---- */
function productFacts(add){
 const P=M.product||{},D=P.deploy||{};const lbl=(l,id)=>(l.find(x=>x[0]===id)||[,id])[1];
 if(hasProductBrief())add('product','Project type: '+lbl(PROJECT_MODES,modeOf()));
 for(const [k,label] of BRIEF_TEXT){const v=String(P[k]||'').trim();if(v)add('product',label.replace(' *','')+': '+v.slice(0,260));}
 for(const [k,label] of BRIEF_NUM)if(String(P[k]||'').trim())add('capacity',label+': '+P[k]);
 if(D.where)add('deployment','Deployment: runs on '+lbl(DEPLOY_WHERE,D.where)+(D.regions?'; data must stay in '+D.regions:''));
 if(D.tenancy)add('deployment','Tenancy: '+lbl(DEPLOY_TENANCY,D.tenancy));
 if((D.channels||[]).length)add('deployment','Reaches users through: '+D.channels.map(x=>lbl(DEPLOY_CHANNELS,x)).join(', '));
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
.mode-page{max-width:1080px}
.mode-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(260px,1fr));gap:14px;margin:18px 0}
.mode-card{display:flex;flex-direction:column;gap:8px;text-align:left;border:1px solid var(--line);border-radius:12px;background:#fff;padding:18px;cursor:pointer;font:inherit}
.mode-card:hover,.mode-card:focus-visible{border-color:var(--accent);box-shadow:0 0 0 3px #e5ecff;outline:none}
.mode-card b{font-size:17px}.mode-card .muted{font-size:13px;line-height:1.45}
.mode-icon{font-size:22px;color:var(--accent)}
.mode-note{margin-top:4px}
.brief-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(300px,1fr));gap:10px 14px;margin:10px 0}
.brief-grid textarea{resize:vertical}
.deploy-block{margin-top:14px;padding-top:12px;border-top:1px solid var(--line)}.deploy-block h3{margin:0 0 4px;font-size:15px}
.deploy-block .type-chips{display:flex;flex-wrap:wrap;gap:6px;align-items:center;margin-top:8px}
.mode-badge{display:inline-block;font-size:11px;font-weight:700;border-radius:999px;padding:2px 8px;background:#e5ecff;color:var(--accent);margin-left:6px}`;
 document.head.append(Object.assign(document.createElement('style'),{id:'project-mode-style',textContent:css}));
})();

/* Project type, changeable later (shown on the project interview step). */
function modeSwitcher(){
 return h('div',{class:'mode-switch'},[selectControl('Project type',modeOf(),PROJECT_MODES.map(([id,t])=>[id,t]),v=>{
  if(v===modeOf())return;const b=M.mode||'modernize';M.mode=v;logInput('project:mode',b,v);persist();render();notice('Project type: '+PROJECT_MODES.find(x=>x[0]===v)[1]+'.');})]);
}

/* Sidebar hint for "01 Product brief". */
function briefProgress(){
 const P=M.product||{},D=P.deploy||{};const n=BRIEF_TEXT.filter(([k])=>String(P[k]||'').trim()).length+(D.where?1:0);
 return n?n+' of '+(BRIEF_TEXT.length+1)+' answered':'Describe the product';
}
