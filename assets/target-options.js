/* MindSuite web app · 06 Target decisions: the Target options team (AI agents propose technical options)
   1 Evidence brief (here, no model): numbered discovery facts F1…Fn for one context. Metadata only: no business
     data, people's names, partner names, comments, rule texts or unverified claims; file paths are masked.
   2–6 run on the backend (POST /api/decide/options): option scouts per area, verifier with a product catalogue,
     rules guard, architect (blueprints) and challenger. See backend/internal/agents/options.go.
   Results stay with the decision (d.optionsRun). People add options to their catalogue, use a blueprint as the
   target, and record the decision; nothing is decided by the agents.
   Loaded by index.html before the main script; advisedCard() adds optionsPanel(c,d) under the advisor. */

const OPTION_AREAS=[['hosting','Hosting & platform'],['architecture','Application & architecture'],['data','Data'],['integration','Integration'],
 ['ui','User interface & channels'],['products','Products & SaaS'],['legacy','Legacy runtime specialist']];
const OPTION_AREA_TO_DIM={hosting:'cloud',architecture:'architecture',data:'database',integration:'integration'};
const optionsUI={consent:{},busy:{},error:{},areas:{},started:{},exploring:{},exploreError:{}};

/* ---- 1. Evidence brief ---- */
const maskPaths=s=>String(s||'').replace(/(?:[A-Za-z]:)?[\w.\-]*[\\/][\w.\-\\/]+(?::\d+)?/g,'[file]').replace(/\b[\w\-]+\.(?:cs|vb|java|cbl|cob|sql|py|js|ts|xml|config|json|ya?ml)(?::\d+)?\b/gi,'[file]');
function optionsBrief(c){
 const S=ctxSignals(c),I=advisorInputs(c),R=M.decide.rules,inv=M.inventory,all=c.virtual;const facts=[];
 const add=(area,text)=>{text=String(text||'').trim();if(text&&facts.length<150)facts.push({id:'F'+(facts.length+1),area,text:text.slice(0,300)});};
 const lbl=(list,id)=>((list||[]).find(x=>x[0]===id)||[,id])[1];
 if(typeof productFacts==='function')productFacts(add);
 if(M.values.legacyType)add('stack','System type: '+lbl(TYPES,M.values.legacyType));
 for(const id of (M.values.technology||[]).slice(0,25)){const t=(TECH_CATALOG||[]).find(x=>x[0]===id);add('stack','Technology: '+(t?t[1]:id)+(M.details&&M.details[id]?' ('+String(M.details[id]).slice(0,80)+')':''));}
 if(!S.lang)add('stack','Current programming language not identified yet');
 const comps=inv.components.filter(x=>all||x.context===c.id);
 if(comps.length){const kinds={};for(const x of comps)kinds[x.kind||'module']=(kinds[x.kind||'module']||0)+1;add('component',comps.length+' components: '+Object.entries(kinds).map(([k,n])=>n+' '+k).join(', '));
  for(const x of comps.slice(0,12))add('component','Component '+maskPaths(x.name)+(x.kind?' ('+x.kind+')':'')+(x.tech?', '+maskPaths(x.tech):'')+(x.size?', size '+x.size:''));}
 const stores=inv.dataStores.filter(d=>all||d.owner===c.id||(d.writers||[]).includes(c.id));
 for(const d of stores.slice(0,15))add('data','Data store '+maskPaths(d.name)+(d.kind?' ('+d.kind+')':'')+(d.sensitivity?', sensitivity '+d.sensitivity:'')+((d.writers||[]).length>1?', written by '+d.writers.length+' contexts':''));
 if(S.shared.length)add('data',S.shared.length+' shared data store(s) written by more than one context');
 const ifs=S.ifs;if(ifs.length){const by={};for(const i of ifs){const k=(i.protocol||'unknown')+' '+(i.direction||'');by[k]=(by[k]||0)+1;}add('integration',ifs.length+' interfaces: '+Object.entries(by).map(([k,n])=>n+' × '+k.trim()).join(', '));
  for(const i of ifs.slice(0,10))add('integration','Interface: '+(i.protocol||'protocol unknown')+(i.direction?', '+i.direction:'')+(i.frequency?', '+i.frequency:''));}
 if(S.eventsWithConsumers.length)add('integration',S.eventsWithConsumers.length+' domain events with consumers');
 if(S.batch)add('component','Has batch jobs');
 if(S.rules.length)add('business',S.rules.length+' business rules recorded (texts not sent)');
 for(const r of S.risks.slice(0,12))add('risk','Risk: '+maskPaths(r.name)+(r.category?' ['+r.category+']':'')+(r.severity?', '+r.severity:''));
 if(S.noTests)add('risk','No automated tests recorded');
 const cp=typeof capOf==='function'?capOf(c):null;
 if(cp){const f=capFacts(cp);add('capacity',[f.peak!=null&&'peak '+f.peak+' requests/s',f.avg!=null&&'average '+f.avg+'/s',f.data!=null&&f.data+' GB data',f.growth&&f.growth+'% growth a year',f.avail!=null&&f.avail+'% availability',cp.users&&cp.users+' users'].filter(Boolean).join(', ')||'Capacity profile recorded without numbers');}
 else add('capacity','No capacity profile yet');
 const reg=((M.scope.regulation||{}).list||[]).filter(x=>!['none','unknown'].includes(x));
 if(reg.length)add('regulation','Regulation: '+reg.map(x=>(REGULATIONS.find(y=>y[0]===x)||[,x])[1]).join(', '));
 if(S.regulated||S.sensitive)add('regulation','Holds '+(S.regulated?'regulated':'personal or financial')+' data');
 if(R.residency==='region')add('regulation','Data must stay in '+(R.region||'a named region'));
 if(R.residency==='private')add('regulation','Sensitive or regulated data must stay on private infrastructure');
 add('skills','Team skills: '+(R.skills.length?R.skills.map(x=>lbl(SKILLS,x)).join(', '):'not recorded'));
 const K=constraints();
 for(const [k,label,type,opts] of CONSTRAINT_FIELDS){if(['budget','standards','procurement'].includes(k))continue;const v=K[k];if(!v)continue;add('constraint',label+': '+(type==='select'?lbl(opts,v):v));}
 if(I.deadlineText)add('constraint','Deadline: '+I.deadlineText);else if(I.deadlinePassed)add('constraint','The recorded deadline has passed; no current deadline');
 if(I.drivers.size)add('business','Modernization drivers: '+[...I.drivers].map(x=>lbl(DRIVERS,x)).join(', '));
 if(I.knowledge)add('business','Business rules known by: '+lbl(KNOWLEDGE,I.knowledge));
 const rules=[];
 for(const dim of Object.keys(TARGETS))for(const [id,label] of TARGETS[dim].opts){const st=R.stance[dim][id];if(!st||st==='allowed')continue;const i=(R.ruleInfo[dim]||{})[id]||{};
  rules.push({area:dim,option:label,state:st,reason:String(i.reason||'').slice(0,200)});}
 return {facts,rules};
}
function defaultOptionAreas(c,d){
 const I=advisorInputs(c);const tech=(M.values.technology||[]).map(id=>((TECH_CATALOG||[]).find(x=>x[0]===id)||[,id])[1]).join(' ');
 const a=['hosting','architecture','data','integration'];
 if(typeof modeOf==='function'&&modeOf()!=='modernize'){const ch=((M.product||{}).deploy||{}).channels||[];if(!ch.length||ch.some(x=>['web','mobile','desktop'].includes(x)))a.push('ui');a.push('products');}
 if((['desktop','clientserver','web'].includes(M.values.legacyType)||I.drivers.has('ux'))&&!a.includes('ui'))a.push('ui');
 if((I.strategic==='generic'||d.approach==='replace'||d.approach==='buy')&&!a.includes('products'))a.push('products');
 if(I.S.lang==='cobol'||M.values.legacyType==='mainframe'||/VB6|Visual Basic 6|PowerBuilder|Delphi|COBOL|RPG|CICS|IBM i|Natural|Adabas/i.test(tech))a.push('legacy');
 return a;
}
function optionsPayload(c,d){
 const sel=optionsUI.areas[c.id]||defaultOptionAreas(c,d);const {facts,rules}=optionsBrief(c);
 return {mode:typeof modeOf==='function'?modeOf():'modernize',context:String(c.name||'').slice(0,80),strategic:String(c.strategic||''),criticality:String(c.criticality||''),approach:d.approach&&d.approach!=='undecided'?d.approach:'',areas:sel,facts,rules};
}

/* ---- Run ---- */
async function runOptionsTeam(c,d){
 if(!optionsUI.consent[c.id]||optionsUI.busy[c.id])return;
 const body=optionsPayload(c,d);
 delete optionsUI.consent[c.id];optionsUI.busy[c.id]=true;optionsUI.started[c.id]=Date.now();delete optionsUI.error[c.id];render();
 const tick=setInterval(()=>{const el=document.getElementById('opt-elapsed-'+c.id);if(el)el.textContent=Math.round((Date.now()-optionsUI.started[c.id])/1000)+' s';},1000);
 try{
  const out=await agentFetch('/api/decide/options',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(body)});
  d.optionsRun={at:new Date().toISOString(),areas:body.areas,facts:body.facts,result:out,ms:Date.now()-optionsUI.started[c.id]};
  M.history.push({field:'decision:'+c.id,at:d.optionsRun.at,actor:actorName(),action:'ai-options',reason:body.facts.length+' facts and '+body.rules.length+' rules sent to the Target options team ('+(out.requests||0)+' model requests)',before:null,after:{options:(out.options||[]).length,blueprints:(out.blueprints||[]).length}});
  persist();
 }catch(e){optionsUI.error[c.id]=/404|not found/i.test(e.message)?'This backend has no options team yet. Restart it from the latest code.':e.message;}
 finally{clearInterval(tick);delete optionsUI.busy[c.id];render();}
}

/* ---- Explore: the concrete choices inside one option (services, tiers, configuration) ---- */
const exploreKey=o=>o.area+':'+normRule(o.name);
async function exploreOption(c,d,o){
 const key=exploreKey(o);if(optionsUI.exploring[key])return;
 const body={...optionsPayload(c,d),explore:{area:o.area,name:o.name,vendor:o.vendor||'',summary:(o.summary||'').slice(0,300)}};
 if(!confirm('Explore "'+o.name+'" in detail?\n\nOne request to Claude with the same '+body.facts.length+' discovery facts and '+body.rules.length+' technology rules as the team run, plus this option. No business data, names or comments.'))return;
 optionsUI.exploring[key]=true;delete optionsUI.exploreError[key];render();
 const t0=Date.now();
 try{
  const out=await agentFetch('/api/decide/options',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(body)});
  d.optionsRun.explore=d.optionsRun.explore||{};
  d.optionsRun.explore[key]={at:new Date().toISOString(),result:out,ms:Date.now()-t0};
  M.history.push({field:'decision:'+c.id,at:new Date().toISOString(),actor:actorName(),action:'ai-explore',reason:'Explored '+o.name+': '+body.facts.length+' facts and '+body.rules.length+' rules sent ('+(out.requests||1)+' model request)',before:null,after:{choices:(out.options||[]).length}});
  persist();
 }catch(e){optionsUI.exploreError[key]=/404|not found/i.test(e.message)?'This backend has no Explore yet. Restart it from the latest code.':e.message;}
 finally{delete optionsUI.exploring[key];render();}
}
function exploredChoices(c,d,o,run){
 const key=exploreKey(o);const ex=(run.explore||{})[key];const busy=optionsUI.exploring[key];const err=optionsUI.exploreError[key];
 const btn=button(busy?'Exploring…':ex?'Explore again':'Explore inside '+o.name,()=>exploreOption(c,d,o),'btn small');if(busy)btn.disabled=true;
 const head=h('div',{class:'explore-actions'},[btn,err?h('span',{class:'conflict-note',text:' '+err}):null]);
 if(!ex)return head;
 const list=(ex.result.options||[]);
 return h('div',{class:'explore'},[head,
  h('small',{class:'muted',text:list.length+' choices inside '+o.name+' · '+new Date(ex.at).toLocaleString()+' · '+Math.round(ex.ms/1000)+' s'}),
  list.length?exploreMatrix(run,o,list):null,
  (ex.result.north||[]).length||(ex.result.south||[]).length?h('div',{class:'traffic'},[
   trafficList(run,'Northbound: traffic coming in','From users and partners into the system',ex.result.north||[]),
   trafficList(run,'Southbound: traffic going out','From the application to databases, files, legacy and partner systems',ex.result.south||[])]):null,
  !list.length?h('p',{class:'input-status open',text:'No usable choices came back, even after one retry. Explore again, or check the discovery facts.'}):null]);
}
/* Architects' comparison matrix: one column per choice, one row per aspect. */
const SPEC_ROWS=[['sku','SKU / plan'],['os-runtime','OS & runtime'],['scaling','Scaling'],['network','Network'],['availability','Availability & SLA'],
 ['disaster-recovery','Disaster recovery'],['deployment','Deployment'],['limits','Limits'],['cost-model','Cost model'],['compliance','Compliance'],
 ['operations','Operations'],['lock-in','Lock-in']];
const specOf=(x,a)=>((x.spec||[]).find(s=>s.aspect===a)||{}).value||'';
const unitVal=(v,u)=>{v=String(v||'').trim();if(!v||/^(n\/?a|none|-|—|0)(|\s|$)/i.test(v))return '';return /^[\d.,\s\-–~]+$/.test(v)?v+' '+u:u.split(' ').pop()+': '+v;};
const computeText=c=>c?[unitVal(c.vcpu,'vCPU'),unitVal(c.ramGb,'GB RAM'),unitVal(c.storageGb,'GB disk'),c.instances&&!/^(n\/?a|0|managed)/i.test(c.instances)?'× '+c.instances:''].filter(Boolean).join(' · ')||'Managed: no compute to size':'';
function exploreMatrix(run,o,list){
 const rows=[
  ['Compute (start)',x=>x.compute?h('div',{},[h('b',{text:computeText(x.compute)}),h('small',{class:x.compute.basis==='facts'?'muted':'warn-text',text:x.compute.basis==='facts'?' from your capacity facts':' assumption'+(x.compute.note?': '+x.compute.note:'')})]):'—',x=>computeText(x.compute)+(x.compute&&x.compute.basis!=='facts'?' (assumption'+(x.compute.note?': '+x.compute.note:'')+')':'')],
  ...SPEC_ROWS.filter(([a])=>list.some(x=>specOf(x,a))).map(([a,l])=>[l,x=>specOf(x,a)||'—',x=>specOf(x,a)]),
  ...(list.some(x=>!specOf(x,'scaling')&&x.sizing)?[['Sizing notes',x=>x.sizing||'—',x=>x.sizing||'']]:[]),
  ...(list.some(x=>!specOf(x,'availability')&&x.resilience)?[['Resilience',x=>x.resilience||'—',x=>x.resilience||'']]:[]),
  ['Choose when',x=>x.chooseWhen||'—',x=>x.chooseWhen||''],
  ['Avoid when',x=>x.avoidWhen||'—',x=>x.avoidWhen||''],
  ['Fit with the facts',x=>(x.fit||[]).length?h('div',{},(x.fit||[]).map(r=>h('div',{},['+ '+r.text,factRef(run,r.facts)]))):'—',x=>(x.fit||[]).map(r=>r.text+' ['+r.facts.join(',')+']').join('; ')],
  ['Risks',x=>(x.risks||[]).length?h('div',{},x.risks.map(t=>h('div',{text:'– '+t}))):'—',x=>(x.risks||[]).join('; ')],
  ['Your rules',x=>x.guard&&x.guard!=='allowed'?(x.guardReason||x.guard):'No rule',x=>x.guard&&x.guard!=='allowed'?(x.guardReason||x.guard):'']];
 const csv=()=>{const esc=v=>{v=String(v??'');return /[",\n]/.test(v)?'"'+v.replace(/"/g,'""')+'"':v;};
  const head=['Aspect',...list.map(x=>x.name+(x.tier?' ('+x.tier+')':''))];
  return [head,...rows.map(([l,,t])=>[l,...list.map(t)])].map(r=>r.map(esc).join(',')).join('\n')+'\n';};
 return h('div',{class:'matrix'},[
  h('div',{class:'matrix-tools'},[h('small',{class:'muted',text:'Compare across, decide down. Values come from Claude: confirm SKUs, limits and SLAs with the vendor before sign-off.'}),
   button('Download comparison (CSV)',()=>downloadText('explore-'+normRule(o.name).replace(/\s+/g,'-')+'.csv',csv()),'btn small')]),
  h('div',{class:'matrix-wrap'},[h('table',{class:'matrix-table'},[
   h('thead',{},[h('tr',{},[h('th',{text:'Aspect'}),...list.map(x=>{const m=catalogueMatch(x);const label=x.name+(x.tier?' · '+x.tier:'');
    return h('th',{class:x.guard==='excluded'?'excluded':''},[h('b',{text:x.name}),x.tier?h('div',{class:'explore-tier',text:x.tier}):null,h('div',{class:'matrix-badges'},[catBadge(x),
     x.guard&&x.guard!=='allowed'?h('span',{class:'stance-chip '+x.guard,text:({standard:'◆ Standard',preferred:'★ Preferred',approval:'? Approval',excluded:'✕ Excluded'})[x.guard]||x.guard}):null]),
     OPTION_AREA_TO_DIM[x.area]&&x.guard!=='excluded'?(m&&normRule(optOf(m[0],m[1])[1])===normRule(label)?h('small',{class:'muted',text:'In your options'}):button('Add to my options',()=>addOptionToCatalogue({...x,name:label}),'btn small')):null]);})])]),
   h('tbody',{},rows.map(([l,cell])=>h('tr',{},[h('th',{scope:'row',text:l}),...list.map(x=>{const v=cell(x);return h('td',{},[typeof v==='string'?v:v]);})])))])])]);
}
function trafficList(run,title,sub,items,knownOnly){
 return h('div',{class:'traffic-col'},[h('b',{text:title}),h('small',{class:'muted',text:sub}),
  items.length?h('ul',{class:'plain-list traffic-list'},items.map(t=>h('li',{},[h('div',{class:'opt-title'},[h('b',{text:t.name}),!knownOnly||t.catalogue==='known'?catBadge(t):null]),
   t.role?h('small',{text:t.role}):null,...(t.why||[]).map(r=>h('div',{class:'muted'},['+ '+r.text,factRef(run,r.facts)]))]))):h('p',{class:'muted',text:'None suggested.'})]);
}
function catBadge(o){
 const C={known:['Known product','ok','Matches the curated product catalogue'],pattern:['Pattern','pat','A way of working, not a product'],
  'needs confirmation':['Needs confirmation','warn','Not in the curated catalogue: confirm it exists and is supported before relying on it']}[o.catalogue]||['Needs confirmation','warn',''];
 return h('span',{class:'opt-cat '+C[1],text:C[0],title:C[2]});
}

/* ---- Solution designer: database, caching, patterns, traffic, cross-cutting components, sizing ---- */
async function designBlueprint(c,d,b,opts){
 const key=b.tier||b.name;if(optionsUI.exploring['design:'+key])return;
 const byId=Object.fromEntries(opts.map(o=>[o.id,o]));
 const body={...optionsPayload(c,d),design:{tier:b.tier||'',name:b.name,approach:b.approach,options:b.options.map(id=>byId[id]?.name).filter(Boolean),migration:b.migration||''}};
 if(!confirm('Design the '+(b.tier||'')+' blueprint "'+b.name+'"?\n\nOne request to Claude with the same '+body.facts.length+' discovery facts and '+body.rules.length+' technology rules, plus this blueprint. No business data, names or comments.'))return;
 optionsUI.exploring['design:'+key]=true;delete optionsUI.exploreError['design:'+key];render();
 const t0=Date.now();
 try{
  const out=await agentFetch('/api/decide/options',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(body)});
  d.optionsRun.designs=d.optionsRun.designs||{};
  d.optionsRun.designs[key]={at:new Date().toISOString(),design:out.design,steps:out.steps,ms:Date.now()-t0,blueprint:b.name};
  M.history.push({field:'decision:'+c.id,at:new Date().toISOString(),actor:actorName(),action:'ai-design',reason:'Designed the '+key+' blueprint: '+body.facts.length+' facts and '+body.rules.length+' rules sent (1 model request)',before:null,after:{blueprint:b.name}});
  persist();
 }catch(e){optionsUI.exploreError['design:'+key]=/404|not found/i.test(e.message)?'This backend has no solution designer yet. Restart it from the latest code.':e.message;}
 finally{delete optionsUI.exploring['design:'+key];render();}
}
const DESIGN_CATS={identity:'Identity and sign-in',secrets:'Secrets and keys',monitoring:'Monitoring and alerts','backup-recovery':'Backup and recovery',
 delivery:'Build and release',security:'Security',configuration:'Configuration',messaging:'Messaging',other:'Other'};
function designSection(run,key,entry){
 const D=entry.design||{};
 const items=(title,list,empty)=>h('div',{class:'design-block'},[h('b',{text:title}),list&&list.length?h('ul',{class:'plain-list design-list'},list.map(x=>h('li',{},[
  h('div',{class:'opt-title'},[h('b',{text:x.name}),x.catalogue==='known'?catBadge(x):null]),x.detail?h('div',{text:x.detail}):null,
  ...(x.why||[]).map(r=>h('div',{class:'muted'},['+ '+r.text,factRef(run,r.facts)]))]))):h('p',{class:'muted',text:empty})]);
 const cats={};for(const c of D.components||[])(cats[c.category]=cats[c.category]||[]).push(c);
 return h('div',{class:'design'},[
  h('div',{class:'design-head'},[h('h5',{text:'Solution design · '+key+': '+(entry.blueprint||'')}),h('small',{class:'muted',text:new Date(entry.at).toLocaleString()+' · '+Math.round((entry.ms||0)/1000)+' s'})]),
  D.summary?h('p',{text:D.summary}):null,
  h('div',{class:'design-grid'},[
   items('Database strategy',D.database,'No database decisions.'),
   items('Caching strategy and components',D.caching,'No caching suggested.'),
   items('Design patterns',D.patterns,'No patterns suggested.')]),
  (D.north||[]).length||(D.south||[]).length?h('div',{class:'traffic'},[
   trafficList(run,'Northbound: traffic coming in','From users and partners into the system',D.north||[],true),
   trafficList(run,'Southbound: traffic going out','From the application to databases, files, legacy and partner systems',D.south||[],true)]):null,
  Object.keys(cats).length?h('div',{class:'design-block'},[h('b',{text:'Cross-cutting components (often missed)'}),h('div',{class:'design-cats'},Object.keys(DESIGN_CATS).filter(k=>cats[k]).map(k=>h('div',{class:'design-cat'},[
   h('small',{class:'eyebrow',text:DESIGN_CATS[k]}),...cats[k].map(cmp=>h('div',{},[h('div',{class:'opt-title'},[h('b',{text:cmp.name}),cmp.catalogue==='known'?catBadge(cmp):null]),cmp.role?h('small',{text:cmp.role}):null,
    ...(cmp.why||[]).map(r=>h('div',{class:'muted'},['+ '+r.text,factRef(run,r.facts)]))]))])))]):null,
  (D.sizing||[]).length?h('div',{class:'design-block'},[h('b',{text:'Starting sizes'}),h('table',{class:'scan-table'},[h('thead',{},[h('tr',{},['Component','vCPU','RAM (GB)','Disk (GB)','Instances','Basis'].map(t=>h('th',{text:t})))]),
   h('tbody',{},D.sizing.map(z=>h('tr',{},[h('td',{text:z.component}),h('td',{text:z.vcpu||'—'}),h('td',{text:z.ramGb||'—'}),h('td',{text:z.storageGb||'—'}),h('td',{text:z.instances||'—'}),
    h('td',{class:z.basis==='facts'?'':'warn-text',text:z.basis==='facts'?'From facts':'Assumption'+(z.note?': '+z.note:'')})])))])]):null,
  (D.gaps||[]).length?h('div',{class:'design-block'},[h('b',{text:'Missing facts that would change this design'}),h('ul',{class:'plain-list'},D.gaps.map(g=>h('li',{text:g})))]):null]);
}

/* ---- Results ---- */
function factRef(run,ids){
 const by=Object.fromEntries((run.facts||[]).map(f=>[f.id,f.text]));
 return h('small',{class:'muted fact-refs'},[' · ',...ids.flatMap((id,i)=>[i?', ':'',h('abbr',{title:by[id]||'',text:id})])]);
}
function catalogueMatch(o){
 const dim=OPTION_AREA_TO_DIM[o.area];if(!dim)return null;
 const hit=typeof matchRuleOption==='function'?matchRuleOption(o.name,dim):null;
 if(hit)return hit;
 const t=TARGETS[dim].opts.find(x=>normRule(x[1])===normRule(o.name));return t?[dim,t[0]]:null;
}
function addOptionToCatalogue(o){
 const dim=OPTION_AREA_TO_DIM[o.area];if(!dim)return;const R=M.decide.rules;R.custom[dim]=R.custom[dim]||[];
 const id=customRuleId(o.name);R.custom[dim].push({id,label:o.name,hint:(o.summary||'').slice(0,120)+' (AI-proposed'+(o.catalogue==='known'?'':', needs confirmation')+')'});
 logInput('rule-option:'+dim+':'+id,null,o.name+' (from the Target options team)');syncRuleCatalog(R);persist();render();
 notice('"'+o.name+'" added to '+TARGETS[dim].title+'. Set a rule on it in 05 Rules & people if needed.');
}
function useBlueprint(c,d,b,opts){
 const byId=Object.fromEntries(opts.map(o=>[o.id,o]));const missing=[];
 changeDecision(c,d,()=>{
  d.approach=b.approach;if(!c.virtual)setDecisionQuiet(c,'approach',b.approach);
  for(const id of b.options){const o=byId[id];if(!o)continue;const m=catalogueMatch(o);
   if(!m){if(OPTION_AREA_TO_DIM[o.area])missing.push(o.name);continue;}
   const [dim,oid]=m;if(TARGETS[dim].multi){const s=new Set(d.choices[dim]||[]);s.add(oid);d.choices[dim]=[...s];}else d.choices[dim]=oid;}
  d.blueprint={name:b.name,approach:b.approach,options:b.options.map(id=>byId[id]?.name).filter(Boolean),migration:b.migration,at:new Date().toISOString()};
 });
 logInput('advisor:blueprint:'+c.id,null,b.name);
 if(missing.length)notice('Approach set. Not in your option lists yet: '+missing.join(', ')+'. Use "Add to my options" first, then use the blueprint again.');
}
function optionsResults(c,d){
 const run=d.optionsRun;if(!run)return null;const out=run.result||{};const opts=out.options||[];const bps=out.blueprints||[];
 const nm=Object.fromEntries(opts.map(o=>[o.id,o]));
 const VERDICT={sound:['Sound','ok'],questionable:['Questionable','warn'],risky:['Risky','bad']};
 const GUARD={standard:['◆ Standard','standard'],preferred:['★ Preferred','preferred'],allowed:['Allowed','allowed'],approval:['? Needs approval','approval'],excluded:['✕ Excluded by your rules','excluded']};
 const stale=JSON.stringify((optionsBrief(c).facts||[]).map(f=>f.text))!==JSON.stringify((run.facts||[]).map(f=>f.text));
 return h('div',{class:'options-results'},[
  h('p',{class:'muted',text:'Generated '+new Date(run.at).toLocaleString()+' from '+(run.facts||[]).length+' facts · '+(out.requests||0)+' model requests · '+Math.round((run.ms||0)/1000)+' s.'}),
  stale?h('p',{class:'input-status open',text:'Discovery facts changed since this run. Generate again to use the current ones.'}):null,
  h('details',{class:'compare'},[h('summary',{text:'Agent steps'}),h('table',{class:'scan-table'},[h('thead',{},[h('tr',{},['Agent','Model','Requests','Time','Result'].map(t=>h('th',{text:t})))]),
   h('tbody',{},(out.steps||[]).map(s=>h('tr',{},[h('td',{text:s.agent}),h('td',{text:s.model||'rules'}),h('td',{text:String(s.requests||0)}),h('td',{text:s.ms?(s.ms/1000).toFixed(1)+' s':'—'}),h('td',{text:s.note||''})])))])]),
  bps.length?h('div',{},[h('h5',{class:'opt-head',text:'Blueprints: Safe, Balanced, Ambitious'}),h('div',{class:'scenario-grid'},bps.map(b=>{const on=d.blueprint&&d.blueprint.name===b.name&&d.approach===b.approach;
   return h('article',{class:'scenario'+(on?' chosen':'')},[
    h('small',{class:'eyebrow tier-'+String(b.tier||'').toLowerCase(),text:b.tier||''}),
    h('div',{class:'scenario-top'},[h('b',{text:b.name}),b.verdict?h('span',{class:'verdict '+(VERDICT[b.verdict]||VERDICT.questionable)[1],text:(VERDICT[b.verdict]||VERDICT.questionable)[0]}):null]),
    h('h5',{text:apprLabel(b.approach)}),
    h('div',{class:'opt-chips'},b.options.map(id=>nm[id]?h('span',{class:'stance-chip '+(nm[id].guard||'allowed'),title:nm[id].summary||'',text:nm[id].name}):null)),
    h('dl',{class:'scenario-facts'},[h('dt',{text:'Effort'}),h('dd',{text:b.effort||'—'}),h('dt',{text:'Cost vs today'}),h('dd',{text:b.costTrend||'unknown'}),h('dt',{text:'Deadline'}),h('dd',{text:b.deadline||'—'})]),
    (b.why||[]).length?h('ul',{class:'plain-list opt-why'},b.why.map(r=>h('li',{},['+ '+r.text,factRef(run,r.facts)]))):null,
    b.migration?h('p',{class:'muted',text:'Migration: '+b.migration}):null,
    (b.challenges||[]).length?h('ul',{class:'plain-list opt-challenges'},b.challenges.map(t=>h('li',{text:'! '+t}))):null,
    h('div',{class:'advisor-actions'},[on?h('span',{class:'badge confirmed',text:'Current target'}):button('Use this blueprint',()=>useBlueprint(c,d,b,opts),'btn small'),
     (()=>{const k=b.tier||b.name;const busy=optionsUI.exploring['design:'+k];const done=(run.designs||{})[k];const bt=button(busy?'Designing…':done?'Design again':'Design this blueprint',()=>designBlueprint(c,d,b,opts),'btn small'+(done?'':' primary'));if(busy)bt.disabled=true;return bt;})()]),
    optionsUI.exploreError['design:'+(b.tier||b.name)]?h('p',{class:'conflict-note',text:optionsUI.exploreError['design:'+(b.tier||b.name)]}):null]);}))]):null,
  ...Object.entries(run.designs||{}).map(([k,v])=>designSection(run,k,v)),
  h('h5',{class:'opt-head',text:'Options by area'}),
  ...OPTION_AREAS.filter(([a])=>opts.some(o=>o.area===a)).map(([a,title])=>h('div',{class:'opt-area'},[h('b',{text:title}),
   ...opts.filter(o=>o.area===a).map(o=>{const m=catalogueMatch(o);const g=GUARD[o.guard]||GUARD.allowed;
    return h('div',{class:'opt-row'+(o.guard==='excluded'?' excluded':'')},[
     h('div',{class:'opt-title'},[h('b',{text:o.name}),o.vendor?h('small',{class:'muted',text:' · '+o.vendor}):null,
      h('span',{class:'stance-chip '+g[1],text:g[0],title:o.guardReason||''}),
      catBadge(o)]),
     o.summary?h('p',{class:'opt-summary',text:o.summary}):null,
     h('ul',{class:'plain-list opt-why'},(o.fit||[]).map(r=>h('li',{},['+ '+r.text,factRef(run,r.facts)]))),
     h('p',{class:'muted opt-meta',text:[o.lifecycle&&'Lifecycle: '+o.lifecycle,o.licence&&'Licence: '+o.licence,(o.skills||[]).length&&'Skills: '+o.skills.join(', ')].filter(Boolean).join(' · ')}),
     (o.risks||[]).length||(o.prerequisites||[]).length?h('details',{class:'compare'},[h('summary',{text:'Risks and prerequisites'}),
      (o.risks||[]).length?h('ul',{class:'plain-list'},o.risks.map(t=>h('li',{text:'Risk: '+t}))):null,(o.prerequisites||[]).length?h('ul',{class:'plain-list'},o.prerequisites.map(t=>h('li',{text:'Needs: '+t}))):null]):null,
     OPTION_AREA_TO_DIM[o.area]?(m?h('small',{class:'muted',text:'In your options as '+optOf(m[0],m[1])[1]+' ('+TARGETS[m[0]].title+')'})
      :o.guard==='excluded'?null:button('Add to my options ('+TARGETS[OPTION_AREA_TO_DIM[o.area]].title+')',()=>addOptionToCatalogue(o),'btn small'))
      :h('small',{class:'muted',text:'For the roadmap and the decision rationale; this area has no option list yet.'}),
     o.guard!=='excluded'&&o.catalogue!=='pattern'?exploredChoices(c,d,o,run):null]);})])),
  (out.dropped||[]).length?h('details',{class:'compare'},[h('summary',{text:'Dropped by the verifier · '+out.dropped.length}),h('ul',{class:'plain-list'},out.dropped.map(t=>h('li',{text:t})))]):null
 ]);
}

/* ---- Panel ---- */
function optionsPanel(c,d){
 const body=optionsPayload(c,d);const busy=!!optionsUI.busy[c.id];const sel=new Set(body.areas);
 return h('section',{class:'options-team'},[
  h('div',{class:'dim-head'},[h('b',{text:'Technical options from the AI agent team'}),h('small',{text:'Scouts propose, a verifier and your rules check, an architect builds blueprints, a challenger tests them. People decide.'})]),
  h('div',{class:'type-chips opt-areas'},OPTION_AREAS.map(([id,label])=>h('label',{class:'type-chip'+(sel.has(id)?' on':'')},[h('input',{type:'checkbox',checked:sel.has(id)?'':null,disabled:busy?'':null,
   onchange:e=>{const s=new Set(optionsUI.areas[c.id]||defaultOptionAreas(c,d));e.target.checked?s.add(id):s.delete(id);optionsUI.areas[c.id]=OPTION_AREAS.map(x=>x[0]).filter(x=>s.has(x));render();}}),label]))),
  h('div',{class:'type-chips'},[agentChip('Allow '+body.facts.length+' discovery facts and '+body.rules.length+' technology rules to be sent to Claude for one run of '+(body.areas.length+2)+' requests',!!optionsUI.consent[c.id],v=>{optionsUI.consent[c.id]=v;render();})]),
  h('details',{class:'compare'},[h('summary',{text:'See exactly what is sent'}),h('pre',{class:'ai-payload',text:JSON.stringify(body,null,2)}),
   h('small',{class:'muted',text:'Not sent: business data, people\'s names, partner names, comments, business rule texts, unverified claims, evidence quotes; file paths are replaced by [file].'})]),
  h('div',{class:'report-reader-actions'},[h('button',{class:'btn small primary',type:'button',disabled:!optionsUI.consent[c.id]||busy||!body.areas.length?'':null,onclick:()=>runOptionsTeam(c,d),
   text:busy?'Agents working…':d.optionsRun?'Generate again':'Generate options'}),
   busy?h('span',{class:'muted'},[' Scouts → verifier → rules guard → architect → challenger · ',h('span',{id:'opt-elapsed-'+c.id,text:'0 s'})]):null]),
  optionsUI.error[c.id]?h('p',{class:'conflict-note',text:'No options: '+optionsUI.error[c.id]}):null,
  optionsResults(c,d)
 ]);
}

(function optionsStyle(){
 if(document.getElementById('options-team-style'))return;
 const css=`
.options-team{border:1px solid #e3e0f5;background:#fbfaff;border-radius:10px;padding:14px 16px;margin:0 0 14px}
.opt-areas{margin:8px 0}
.opt-head{margin:14px 0 6px;font-size:14px}
.opt-chips{display:flex;flex-wrap:wrap;gap:4px}.opt-chips .stance-chip{cursor:default;padding:2px 8px;font-size:11px}
.opt-area{margin:10px 0}
.opt-row{background:#fff;border:1px solid var(--line);border-radius:8px;padding:10px 12px;margin:6px 0}
.opt-row.excluded{opacity:.6}
.opt-title{display:flex;flex-wrap:wrap;gap:8px;align-items:center}
.opt-title .stance-chip{cursor:default;padding:2px 8px;font-size:11px}
.opt-cat{font-size:11px;font-weight:700;border-radius:999px;padding:2px 8px}.opt-cat.ok{background:#e8f6ee;color:#166534}.opt-cat.warn{background:#fff8e6;color:#8a5a00}
.opt-summary{margin:4px 0}.opt-meta{margin:4px 0;font-size:12px}
.opt-why{font-size:13px;margin:4px 0}.opt-challenges{font-size:12px;color:#8a5a00}
.fact-refs abbr{text-decoration:underline dotted;cursor:help}
.opt-cat.pat{background:#eef2ff;color:var(--accent)}
.explore-actions{margin-top:6px}
.explore{margin-top:8px;padding:8px 10px;border-left:3px solid #c7d2fe;background:#f8f9ff;border-radius:0 8px 8px 0}
.explore-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(240px,1fr));gap:8px;margin-top:6px}
.explore-card{background:#fff;border:1px solid var(--line);border-radius:8px;padding:10px}.explore-card.excluded{opacity:.6}
.explore-tier{font-size:11px;font-weight:700;background:#e5ecff;color:var(--accent);border-radius:999px;padding:2px 8px}
.explore-facts{grid-template-columns:90px 1fr}.explore-facts dd{font-weight:400}
.explore-compute{display:flex;flex-wrap:wrap;gap:4px 12px;align-items:baseline;background:#f3f6ff;border-radius:6px;padding:6px 8px;margin:6px 0;font-size:13px}
.explore-compute.assumed{background:#fff8e6}.explore-compute small{flex-basis:100%}
.traffic{display:grid;grid-template-columns:repeat(auto-fit,minmax(260px,1fr));gap:12px;margin-top:10px}
.traffic-col{background:#fff;border:1px solid var(--line);border-radius:8px;padding:10px}.traffic-col>small{display:block;margin-bottom:6px}
.traffic-list li{padding:6px 0;border-top:1px solid var(--line)}.traffic-list li small{display:block}
.design{margin:12px 0;padding:12px 14px;border:1px solid #c7d2fe;border-radius:10px;background:#fff}\n.design-head{display:flex;justify-content:space-between;align-items:baseline;gap:10px}.design-head h5{margin:0;font-size:14px}\n.design-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(260px,1fr));gap:12px;margin:8px 0}\n.design-block{margin:10px 0}.design-list li{padding:6px 0;border-top:1px solid var(--line)}\n.design-cats{display:grid;grid-template-columns:repeat(auto-fit,minmax(220px,1fr));gap:10px;margin-top:6px}\n.design-cat{background:#f8f9ff;border-radius:8px;padding:8px 10px}.design-cat>div{margin:6px 0}.design-cat small{display:block}\n.matrix{margin-top:8px}.matrix-tools{display:flex;justify-content:space-between;align-items:center;gap:10px;margin-bottom:6px}\n.matrix-wrap{overflow-x:auto;border:1px solid var(--line);border-radius:8px;background:#fff}\n.matrix-table{border-collapse:collapse;font-size:12px;min-width:100%}\n.matrix-table th,.matrix-table td{border-bottom:1px solid var(--line);border-right:1px solid var(--line);padding:6px 8px;vertical-align:top;text-align:left;min-width:190px;max-width:280px}\n.matrix-table thead th{position:sticky;top:0;background:#f8f9ff;font-size:12px}.matrix-table thead th.excluded{opacity:.6}\n.matrix-table tbody th{position:sticky;left:0;background:#fbfbfd;min-width:130px;max-width:150px;font-weight:600;color:var(--muted)}\n.matrix-table tbody tr:nth-child(even) td{background:#fcfcff}\n.matrix-badges{display:flex;flex-wrap:wrap;gap:4px;margin:4px 0}.matrix-badges .stance-chip{cursor:default;padding:1px 6px;font-size:10px}\n.matrix-table .explore-tier{display:inline-block;margin-top:2px}\n.tier-safe{color:#166534}.tier-balanced{color:var(--accent)}.tier-ambitious{color:#9a3412}
.options-results .scenario .eyebrow{font-weight:700;letter-spacing:.06em;text-transform:uppercase;font-size:11px}`;
 document.head.append(Object.assign(document.createElement('style'),{id:'options-team-style',textContent:css}));
})();
