/* MindSuite Pathfinder · Technology rules: overview, business-need search, design areas, capability cards
   with a provider comparison, and explained presets with a per-rule preview.
   Three different things are kept apart on this page and in the HLD:
     - a PRESET is a starting point that proposes rules (nothing changes until "Apply selected changes");
     - a RULE is the organization's policy (Standard, Preferred, Needs approval, Retiring, Excluded);
     - a DECISION is what this project has chosen (platform decision, routes) — recorded on Design options.
   Browsing a provider here never changes the approved provider. Uses capabilities.js. */

const RO_UI={area:'',all:false,q:'',browse:{},presetSel:[]};

/* ---- Presets explained (definitions in rule-catalog.js; text checked against them) ---- */
const PRESET_INFO={
 'web-mvp':{kind:'How users reach the product',name:'First-release web app',
  sentence:'You are launching a product people use in a browser. This starting point favours one modular application, PostgreSQL and REST APIs.',
  suits:'A small team shipping a first customer portal or internal tool that it deploys and runs as one unit.',
  tradeoff:'One deployment is simple to run, but parts that later need to scale on their own must be split out.',
  decide:'Which cloud, how customers sign in, and whether any part needs to scale separately.'},
 'mobile-api':{kind:'How users reach the product',name:'Mobile app with a shared API',
  sentence:'The mobile app calls a backend that handles business rules and data. A future website or partner app can use the same backend.',
  suits:'A product whose main channel is a mobile app, with a website or partners expected later.',
  tradeoff:'Makes REST the standard for every client; real-time or offline needs are separate decisions.',
  decide:'Offline behaviour and synchronization, mobile sign-in, push notifications and the backend architecture itself.'},
 'event-product':{kind:'How parts of the product communicate',name:'Event-driven product',
  sentence:'Parts of the product tell each other what happened through events and queues instead of calling each other directly.',
  suits:'Products with bursts of work, many follow-up actions per event, or devices and integrations sending data.',
  tradeoff:'Absorbs spikes and decouples teams, but tracing a request across events needs good monitoring.',
  decide:'Which events are needed, ordering and replay needs, and the event platform on your cloud.'},
 'opensource':{kind:'Organization preference',name:'Open-source first',
  sentence:'Prefer open-source technologies when they meet the product\'s needs.',
  suits:'Organizations that want portability and to avoid licence lock-in.',
  tradeoff:'You can still use a paid managed service (e.g. managed PostgreSQL); someone must own patching where you self-run.',
  decide:'This preset also prefers private cloud: untick that row to keep an open-source preference on AWS, Azure or Google Cloud. Licences of commercial databases need review.'},
 'cautious':{kind:'Organization preference',name:'Low risk: gradual change',
  sentence:'Prefer changes the team can make step by step, with approval before larger architectural shifts.',
  suits:'Teams modernizing a critical system with little room for disruption.',
  tradeoff:'Safer delivery, slower to reach a modern architecture.',
  decide:'Which parts can move first, and who approves microservices or event-driven designs.'},
 'regulated':{kind:'Organization preference',name:'Regulated: keep data on-premises',
  sentence:'Keep data in your own data centre or private cloud; public clouds are excluded.',
  suits:'Regulated organizations whose rules do not yet allow public cloud for this data.',
  tradeoff:'Strong control, but more to run yourselves and fewer managed services.',
  decide:'Capacity, backup and recovery on your own platform.'},
 azure:{kind:'Hosting provider',name:'Microsoft Azure as the standard cloud',sentence:'Azure is the organization standard; other public clouds need approval. Prefers .NET, SQL Server and PostgreSQL.',
  suits:'Organizations with Microsoft agreements, Entra ID and .NET skills.',tradeoff:'Easier governance on one cloud; using a service only another cloud offers needs approval.',decide:'Region, landing zone and which Azure services need approval.'},
 aws:{kind:'Hosting provider',name:'AWS as the standard cloud',sentence:'AWS is the organization standard; other public clouds need approval. Prefers PostgreSQL and MySQL.',
  suits:'Organizations with AWS accounts, skills and agreements.',tradeoff:'Easier governance on one cloud; using a service only another cloud offers needs approval.',decide:'Region, account structure and which AWS services need approval.'},
 gcp:{kind:'Hosting provider',name:'Google Cloud as the standard cloud',sentence:'Google Cloud is the organization standard; other public clouds need approval. Prefers PostgreSQL.',
  suits:'Organizations with Google Cloud agreements, BigQuery or Kubernetes skills.',tradeoff:'Easier governance on one cloud; email/SMS and managed SFTP need partners on Google Cloud.',decide:'Region, project structure and partner services.'}};
const PRESET_KIND_HELP={'How users reach the product':'web or mobile','How parts of the product communicate':'events and queues','Organization preference':'what your organization prefers','Hosting provider':'which cloud'};

function presetChanges(keys){
 const R=M.decide.rules,rows={};
 for(const key of keys){const p=rulePresets().find(x=>x[0]===key);if(!p)continue;const info=PRESET_INFO[key]||{name:p[1]};
  for(const [dim,rules] of Object.entries(p[2]))for(const [id,state] of Object.entries(rules)){if(!optOf(dim,id))continue;
   const k=dim+':'+id;const r=rows[k]=rows[k]||{dim,id,label:optOf(dim,id)[1],current:R.stance[dim][id]||'allowed',proposals:[]};r.proposals.push({key,name:info.name,state});}}
 const list=Object.values(rows);
 for(const r of list){
  const states=[...new Set(r.proposals.map(p=>p.state))];r.overlap=states.length>1;r.state=r.proposals[r.proposals.length-1].state;
  const issues=[];
  if(r.overlap)issues.push('presets disagree: '+r.proposals.map(p=>p.name+' → '+RULE_STATE[p.state][1]).join(', '));
  if(r.current!=='allowed'&&r.current!==r.state)issues.push('changes your rule from '+RULE_STATE[r.current][1]);
  if(r.state==='standard'){const other=TARGETS[r.dim].opts.filter(([o])=>o!==r.id&&(R.stance[r.dim][o]==='standard'||list.some(x=>x!==r&&x.dim===r.dim&&x.id===o&&x.state==='standard')));if(other.length)issues.push('another Standard in '+TARGETS[r.dim].title+': '+other.map(o=>o[1]).join(', '));}
  if(r.current==='excluded'&&r.state!=='excluded')issues.push('currently Excluded');
  r.issues=issues;r.noop=r.current===r.state&&!r.overlap;
 }
 return list.sort((a,b)=>Object.keys(TARGETS).indexOf(a.dim)-Object.keys(TARGETS).indexOf(b.dim));
}

function presetPanel(){
 const all=rulePresets();const kinds=[...new Set(all.map(p=>(PRESET_INFO[p[0]]||{}).kind||'Other'))];
 const sel=RO_UI.presetSel;
 const toggle=k=>{const i=sel.indexOf(k);if(i>=0)sel.splice(i,1);else sel.push(k);render();};
 return h('details',{class:'compare preset-panel'},[h('summary',{text:'Start from presets (optional) — nothing changes until you review and apply'}),
  h('p',{class:'muted',text:'Presets are starting points. They can be combined: how users reach the product (web or mobile), how parts communicate (events), what your organization prefers (e.g. open source) and which cloud hosts it. Pick any you like, then preview exactly what would change.'}),
  ...kinds.map(kind=>h('div',{class:'preset-kind'},[h('b',{text:kind}),h('div',{class:'preset-cards'},all.filter(p=>((PRESET_INFO[p[0]]||{}).kind||'Other')===kind).map(p=>{const i=PRESET_INFO[p[0]]||{name:p[1],sentence:''};const on=sel.includes(p[0]);
   const changes=Object.entries(p[2]).flatMap(([dim,rs])=>Object.entries(rs).filter(([id])=>optOf(dim,id)).map(([id,st])=>(RULE_STATE[st][2]?RULE_STATE[st][2]+' ':'')+RULE_STATE[st][1]+': '+optOf(dim,id)[1]));
   return h('label',{class:'preset-card'+(on?' on':'')},[h('div',{class:'preset-top'},[h('input',{type:'checkbox',checked:on?'':null,onchange:()=>toggle(p[0])}),h('b',{text:i.name})]),
    h('p',{text:i.sentence}),
    h('details',{},[h('summary',{text:'Who it suits, what it changes, trade-off'}),h('dl',{class:'inv-meta'},[h('dt',{text:'Suits'}),h('dd',{text:i.suits||'—'}),h('dt',{text:'Changes'}),h('dd',{text:changes.join(' · ')}),h('dt',{text:'Trade-off'}),h('dd',{text:i.tradeoff||'—'}),h('dt',{text:'Still to decide'}),h('dd',{text:i.decide||'—'})])])]);}))])),
  h('div',{class:'cost-actions'},[button(sel.length?'Preview changes from '+sel.length+' preset'+(sel.length>1?'s':''):'Choose presets to preview',()=>{if(sel.length)presetDialog([...sel]);},'btn small primary'),sel.length?button('Clear',()=>{RO_UI.presetSel=[];render();},'linkbtn'):null])]);
}

function presetDialog(keys){
 const rows=presetChanges(keys);
 const dialog=dialogShell('Preview preset changes',keys.map(k=>(PRESET_INFO[k]||{}).name||k).join(' + ')+'. Untick any row you do not want. Cancel changes nothing.');
 const checks=new Map(),picks=new Map();
 const replace=h('input',{type:'checkbox'});
 const table=h('table',{class:'scan-table preset-diff'},[h('thead',{},h('tr',{},['Apply','Area','Option','Your rule now','Proposed','Check'].map(t=>h('th',{text:t})))),
  h('tbody',{},rows.map(r=>{const cb=h('input',{type:'checkbox','aria-label':'Apply '+r.label,checked:(!r.issues.length&&!r.noop)?'':null,disabled:r.noop?'':null});checks.set(r,cb);
   let prop;
   if(r.overlap){const s=h('select',{class:'field compact','aria-label':'Proposed rule for '+r.label},[...new Set(r.proposals.map(p=>p.state))].map(st=>h('option',{value:st,text:RULE_STATE[st][1]+' ('+r.proposals.filter(p=>p.state===st).map(p=>p.name).join(', ')+')'})));picks.set(r,s);prop=s;}
   else prop=h('span',{class:'stance-chip '+r.state,text:(RULE_STATE[r.state][2]?RULE_STATE[r.state][2]+' ':'')+RULE_STATE[r.state][1]});
   return h('tr',{class:r.issues.length?'warn':''},[h('td',{},[cb]),h('td',{text:TARGETS[r.dim].title}),h('td',{text:r.label}),h('td',{text:RULE_STATE[r.current][1]}),h('td',{},[prop]),
    h('td',{class:r.issues.length?'warn-text':'muted',text:r.noop?'already set':r.issues.join('; ')||'ok'})]);}))]);
 dialog.append(h('div',{class:'drawer-content'},[
  h('p',{class:'muted',text:'Rows with a warning are unticked: tick them only if you want that change. Recorded design decisions are not touched by presets.'}),table,
  h('label',{class:'formfiller-auto'},[replace,' Clear my other technology rules first (keeps cloud-service rules and decisions)'])]));
 dialog.append(h('div',{class:'drawer-footer'},[button('Cancel',()=>dialog.close(),'btn'),button('Apply selected changes',()=>{
  const chosen=rows.filter(r=>checks.get(r).checked);
  if(replace.checked)for(const dim of Object.keys(TARGETS))for(const [id] of TARGETS[dim].opts){if(chosen.some(r=>r.dim===dim&&r.id===id))continue;if(M.decide.rules.stance[dim][id])setRule(dim,id,'allowed',{});}
  for(const r of chosen){const st=picks.get(r)?picks.get(r).value:r.state;setRule(r.dim,r.id,st,{reason:'Preset: '+r.proposals.filter(p=>p.state===st).map(p=>p.name).join(' + '),owner:M.actor||''});}
  persist();dialog.close();RO_UI.presetSel=[];render();notice(chosen.length+' rule'+(chosen.length===1?'':'s')+' changed from the preset'+(keys.length>1?'s':'')+'.');},'btn primary')]));
 dialog.showModal();
}

/* ---- Overview ---- */
function roCounts(){
 const R=M.decide.rules;let rules=0;
 for(const dim of Object.keys(TARGETS))for(const [id] of TARGETS[dim].opts)if((R.stance[dim][id]||'allowed')!=='allowed')rules++;
 for(const prov of Object.keys(R.services||{}))rules+=Object.values(R.services[prov]||{}).filter(x=>x&&x.state&&x.state!=='allowed').length;
 const st=CAP_LIST.map(c=>capStatus(c[0])).filter(Boolean);
 const needed=st.filter(s=>s.needed||((M.decide.capAdded||{})[s.cap[0]]));
 const attention=needed.filter(s=>['unresolved','conflict','approval','needs-partner'].includes(s.status));
 let pf={};try{pf=platformEffective();}catch(e){}
 const extra=[];
 if(!pf.decided)extra.push({text:'Platform decision not recorded (Design options, step 1)',go:()=>go('decide')});
 const ap=capApprovedProvider();if(!ap.id)extra.push({text:ap.conflict?'Two cloud standards compete — choose one':'No cloud approved yet: provider-specific services stay suggestions',go:null});
 for(const dim of Object.keys(TARGETS)){const std=TARGETS[dim].opts.filter(([id])=>R.stance[dim][id]==='standard');if(std.length>1)extra.push({text:'Two standards in '+TARGETS[dim].title+': '+std.map(o=>o[1]).join(', '),go:null});}
 const rec=capRecovery();if(!rec.rto||!rec.rpo)extra.push({text:'Recovery time (RTO) and data loss (RPO) not set as separate answers',go:()=>go('upload')});
 return {rules,needed,attention,extra,st};
}
function rulesOverview(){
 const c=roCounts();const R=M.decide.rules;
 const ap=capApprovedProvider();
 const q=RO_UI.q;const found=capSearch(q);
 const tiles=CAP_AREAS.map(([id,title,help])=>{const caps=c.st.filter(s=>s.cap[1]===id);const need=caps.filter(s=>s.needed);const att=need.filter(s=>['unresolved','conflict','approval','needs-partner'].includes(s.status));
  const dims={compute:['architecture','language'],data:['database'],integration:['integration']}[id]||[];
  const rn=dims.reduce((n,d)=>n+TARGETS[d].opts.filter(([o])=>(R.stance[d][o]||'allowed')!=='allowed').length,0);
  return {id,title,help,need,att,rn,first:need[0]&&need[0].evidence};});
 tiles.push({id:'provider',title:'Hosting provider and cloud services',help:'Which cloud hosts the product, and rules for individual services.',need:[],att:ap.id?[]:[1],rn:Object.values(R.stance.cloud||{}).filter(x=>x&&x!=='allowed').length,first:ap.id?'Approved: '+(CAP_PROVIDERS.find(p=>p[0]===ap.id)||[,ap.id])[1]+' — '+ap.why:ap.why});
 return h('section',{class:'ro-overview'},[
  h('div',{class:'ro-concepts'},[
   h('div',{},[h('b',{text:'Preset'}),h('small',{text:'A starting point that proposes rules. You review every change first.'})]),
   h('div',{},[h('b',{text:'Rule'}),h('small',{text:'What your organization prefers, requires, restricts or excludes.'})]),
   h('div',{},[h('b',{text:'Decision'}),h('small',{text:'What this project has chosen. Recorded on Design options; the HLD shows it as Selected.'})])]),
  h('div',{class:'metrics ro-metrics'},[
   h('button',{type:'button',class:'metric',onclick:()=>{RO_UI.area='';RO_UI.all=false;render();}},[h('b',{text:String(c.needed.filter(s=>!s.baseline).length)}),h('span',{text:'Decisions for this project (from your brief) + '+c.needed.filter(s=>s.baseline).length+' baseline'})]),
   h('div',{class:'metric'},[h('b',{text:String(c.rules)}),h('span',{text:'Rules already set'})]),
   h('div',{class:'metric'+(c.attention.length+c.extra.length?' warn':'')},[h('b',{text:String(c.attention.length+c.extra.length)}),h('span',{text:'Decisions needing attention'})])]),
  (c.attention.length||c.extra.length)?h('details',{class:'ro-attention'},[h('summary',{text:'What needs attention'}),h('ul',{class:'plain-list'},[
   ...c.extra.map(x=>h('li',{},[x.text,x.go?button('Open',x.go,'linkbtn'):null])),
   ...c.attention.map(s=>h('li',{},[h('b',{text:s.cap[2]+': '}),CAP_STATUS_LABEL[s.status]+(s.note?' — '+s.note:''),button('Show',()=>{RO_UI.area=s.cap[1];render();},'linkbtn')]))])]):null,
  h('div',{class:'ro-search'},[h('input',{class:'field',type:'search',value:q,placeholder:'Search by need, e.g. customers sign in, send notifications, store uploaded videos, recover after an outage','aria-label':'Search by business need',
   oninput:e=>{RO_UI.q=e.target.value;clearTimeout(RO_UI.t);RO_UI.t=setTimeout(()=>{render();const el=document.querySelector('.ro-search input');if(el){el.focus();el.setSelectionRange(el.value.length,el.value.length);}},250);}})]),
  q?h('div',{class:'ro-results'},found.length?found.slice(0,8).map(cap=>capCard(cap[0])):[h('p',{class:'muted',text:'No capability matches “'+q+'”. Try everyday words such as “payments”, “files”, “sign in” or “outage”.'})]):null,
  !q?h('div',{class:'ro-tiles'},tiles.map(t=>h('button',{type:'button',class:'ro-tile'+(RO_UI.area===t.id?' on':''),'aria-expanded':String(RO_UI.area===t.id),onclick:()=>{RO_UI.area=RO_UI.area===t.id?'':t.id;render();}},[
   h('b',{text:t.title}),h('small',{class:'muted',text:t.first?'Why: '+t.first:t.help}),
   h('span',{class:'ro-tile-meta',text:(t.id==='provider'?'':t.need.filter(s=>!s.baseline).length+' from your brief'+(t.need.some(s=>s.baseline)?' + '+t.need.filter(s=>s.baseline).length+' baseline':'')+' · ')+t.rn+' rule'+(t.rn===1?'':'s')+(t.att.length?' · '+t.att.length+' need attention':'')})]))):null,
  !q?h('label',{class:'formfiller-auto'},[h('input',{type:'checkbox',checked:RO_UI.all?'':null,onchange:e=>{RO_UI.all=e.target.checked;render();}}),' Show capabilities the brief does not call for (browse all)']):null,
  !q&&RO_UI.area?roArea(RO_UI.area):null]);
}
function roArea(id){
 if(id==='provider')return h('div',{class:'ro-area'},[h('h3',{text:'Hosting provider and cloud services'}),stanceChips('cloud'),typeof cloudServicesPanel==='function'?cloudServicesPanel():null]);
 const area=CAP_AREAS.find(a=>a[0]===id);const st=CAP_LIST.filter(c=>c[1]===id).map(c=>capStatus(c[0]));
 const show=st.filter(s=>s.needed||RO_UI.all||(M.decide.capAdded||{})[s.cap[0]]);
 const dims={compute:['architecture','language'],data:['database'],integration:['integration']}[id]||[];
 const P=M.product||{},D=P.deploy||{},rec=capRecovery();
 const links={channels:[['How it reaches users',(D.channels||[]).join(', ')||'not answered','upload']],
  ops:[['Availability target',P.availability?P.availability+'%':'not answered','upload'],['Recovery time (RTO)',rec.rto||'not set','upload'],['Data loss allowed (RPO)',rec.rpo||'not set','upload']],
  delivery:[['Data must stay in',D.regions||'not answered','upload'],['Regulations',((M.scope&&M.scope.regulation&&M.scope.regulation.list)||[]).join(', ')||'not recorded','upload']]}[id]||[];
 return h('div',{class:'ro-area'},[h('h3',{text:area[1]}),h('p',{class:'muted',text:area[2]}),
  links.length?h('dl',{class:'inv-meta ro-links'},links.flatMap(([k,v,screen])=>[h('dt',{text:k}),h('dd',{},[v+' ',button('Edit in the brief',()=>go(screen),'linkbtn')])])):null,
  ...dims.map(d=>stanceChips(d)),
  show.length?h('div',{class:'ro-caps'},show.map(s=>capCard(s.cap[0]))):h('p',{class:'muted',text:'Nothing in the brief calls for these yet. Tick “browse all” to see them.'}),
  st.length>show.length?h('small',{class:'muted',text:(st.length-show.length)+' more not called for by the brief.'}):null]);
}
function capCard(capId){
 const s=capStatus(capId);if(!s)return null;const c=s.cap;
 const browse=RO_UI.browse[capId]||'';
 const D=M.decide;D.capDismissed=D.capDismissed||{};D.capAdded=D.capAdded||{};
 const setBrowse=p=>{RO_UI.browse[capId]=p;render();};
 const provRow=([pid,pl])=>{const d=c[7][pid];const r=pid==='private'||pid==='oci'||pid==='ibm'?(c[5]?capResolveService(c[5],pid,''):{name:'',status:'unresolved'}):null;
  const name=d?(d.s||''):(r&&r.name)||'';
  const st=d?capStatus(capId,pid):null;
  return h('tr',{class:(s.prov===pid?'approved':'')+(browse===pid?' browsing':'')},[h('th',{scope:'row',text:pl+(s.prov===pid?' (approved)':'')}),
   h('td',{},[name?name:h('span',{class:'warn-text',text:'No verified equivalent'}),d&&d.check?h('small',{class:'muted',text:' · to confirm'}):null]),
   h('td',{class:'muted',text:d?CAP_OPS[d.o]||'':(name?'see the service catalogue':'—')}),
   h('td',{text:d?d.l:(name?'':'Needs a design decision: self-run or partner')}),
   h('td',{},[st?h('span',{class:'cap-st '+st.status,text:CAP_STATUS_LABEL[st.status]}):h('span',{class:'muted',text:'—'})])]);};
 const main=CAP_PROVIDERS.slice(0,3),more=CAP_PROVIDERS.slice(3);
 return h('article',{class:'cap-card'},[
  h('div',{class:'cap-head'},[h('b',{text:c[2]}),h('span',{class:'cap-st '+s.status,text:CAP_STATUS_LABEL[s.status]+(s.prov?' · '+(CAP_PROVIDERS.find(p=>p[0]===s.prov)||[,s.prov])[1]:'')}),
   s.needed?button('Not needed',()=>{D.capDismissed[capId]=true;delete D.capAdded[capId];logInput('capability:'+capId,'needed','dismissed');persist();render();},'linkbtn')
    :button('Add to this project',()=>{delete D.capDismissed[capId];D.capAdded[capId]=true;logInput('capability:'+capId,'not needed','added');persist();render();},'linkbtn')]),
  h('p',{class:'cap-job',text:c[3]}),
  h('p',{class:'cap-why'},[h('b',{text:s.baseline?'Baseline: ':'Why it appears: '}),s.dismissed?'you marked it not needed':s.evidence||'not called for by the brief (you can add it)']),
  s.prov?h('p',{class:'cap-svc'},[h('b',{text:'For this project ('+(CAP_PROVIDERS.find(p=>p[0]===s.prov)||[,s.prov])[1]+', '+s.approved.why+'): '}),s.name||h('span',{class:'warn-text',text:'no service'}),s.note?h('small',{class:'muted',text:' — '+s.note}):null])
   :h('p',{class:'cap-svc muted',text:'No provider approved yet: compare the options below. Browsing a provider does not change your project.'}),
  h('details',{class:'cap-compare',open:!s.prov?'':null},[h('summary',{text:'Compare providers'}),
   h('div',{class:'tp-scroll'},h('table',{class:'tp-table cap-table'},[h('thead',{},h('tr',{},['Provider','Service','Who operates it','Important limitation','Status'].map(t=>h('th',{text:t})))),
    h('tbody',{},[...main.map(provRow),...(browse==='more'?more.map(provRow):[])])])),
   browse!=='more'?button('Show Oracle Cloud, IBM Cloud and private',()=>setBrowse('more'),'linkbtn'):null])]);
}

(function roStyle(){
 if(document.getElementById('ro-style'))return;
 const css=`.ro-overview{margin:10px 0 16px}.ro-concepts{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:10px;margin:0 0 10px}.ro-concepts>div{border:1px solid #e3e8f2;border-radius:10px;padding:8px 10px;background:#fff}.ro-concepts small{display:block;color:var(--muted)}
.ro-metrics .metric{text-align:left;font:inherit;cursor:default}.ro-metrics button.metric{cursor:pointer}.ro-metrics .metric.warn b{color:#9b1c1c}
.ro-attention{margin:8px 0}.ro-search{margin:10px 0}.ro-search input{width:100%}
.ro-tiles{display:grid;grid-template-columns:repeat(auto-fill,minmax(260px,1fr));gap:10px;margin:8px 0}.ro-tile{display:flex;flex-direction:column;gap:4px;text-align:left;font:inherit;color:inherit;padding:10px 12px;border:1px solid #d9e1f2;border-radius:10px;background:#fff;cursor:pointer}.ro-tile.on{border:2px solid var(--accent);background:#f6f4ff}.ro-tile-meta{font-size:12.5px;color:#5a46c8}
.ro-area{margin:12px 0;padding:12px 14px;border:1px solid #d9d2ff;border-radius:12px;background:#fcfbff}.ro-area h3{margin:0 0 4px}.ro-links{margin:6px 0 10px}
.ro-caps,.ro-results{display:grid;gap:10px;margin-top:8px}.cap-card{border:1px solid #e3e8f2;border-radius:10px;padding:10px 12px;background:#fff}.cap-head{display:flex;flex-wrap:wrap;gap:6px 10px;align-items:center}.cap-head .linkbtn{margin-left:auto}
.cap-job{margin:4px 0}.cap-why,.cap-svc{margin:4px 0;font-size:14px}.cap-st{font-size:12px;padding:2px 8px;border-radius:10px;background:#eef0f5;color:#3c4257}.cap-st.selected{background:#e6f6ec;color:#1d6b3a}.cap-st.required{background:#e8eefc;color:#2747a0}.cap-st.approval,.cap-st.needs-partner{background:#fff4e0;color:#7a4b00}.cap-st.unresolved,.cap-st.conflict{background:#fde8e8;color:#9b1c1c}.cap-st.design{background:#f1eeff;color:#5a46c8}
.cap-table{min-width:640px}.cap-table tr.approved th{color:#1d6b3a}.preset-panel{margin:10px 0}.preset-kind{margin:8px 0}.preset-cards{display:grid;grid-template-columns:repeat(auto-fill,minmax(260px,1fr));gap:8px;margin-top:4px}
.preset-card{display:block;border:1px solid #e3e8f2;border-radius:10px;padding:8px 10px;background:#fff;cursor:pointer}.preset-card.on{border:2px solid var(--accent);background:#f6f4ff}.preset-top{display:flex;gap:8px;align-items:center}.preset-card p{margin:4px 0;font-size:13.5px}
.preset-diff tr.warn td{background:#fffaf0}
@media(max-width:760px){.ro-concepts{grid-template-columns:1fr}}`;
 document.head.append(Object.assign(document.createElement('style'),{id:'ro-style',textContent:css}));
})();
