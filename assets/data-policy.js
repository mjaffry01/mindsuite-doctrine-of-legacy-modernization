/* MindSuite web app · Data and caching policy (screen 'datapolicy').
   One policy per business object (stock, order, payment, customer account …): who owns it, which single
   system may write it, how consistent it must be, retention, classification and residency, backup, access;
   and whether it may be cached, for how long, how entries are removed, what happens when the cache is down and
   whether personal data may be cached.
   The Data policy agent (Claude on the project backend, POST /api/design/datapolicy) drafts them on request,
   from design metadata only: the platform decision, targets, the database/cache/messaging/recovery layers, the
   business objects with the systems that write them, and the validation journeys. No records, no people's
   names, no secrets. Drafts stay drafts until accepted by the person working (M.actor); accepted policies are
   editable. A local check flags caching that contradicts a journey's acceptance measure.
   Stored in M.dataPolicy = {objects:[{name,source,writers}], removed:{}, ai:{…,at,basis}, accepted:{name:{data,cache,by,at}}}. */

const DP_UI={busy:false,message:''};
const DP_KEYWORDS=[['stock',/\b(stock|inventory|last item)\b/i],['order',/\border/i],['payment',/\b(pay|payment|checkout|billing|refund)/i],
 ['customer account',/\b(customer|account|sign.?in|profile)\b/i],['menu',/\bmenu/i],['booking',/\b(booking|reservation|appointment)/i],
 ['subscription',/\bsubscri/i],['invoice',/\binvoice/i],['payout',/\bpayout/i],['delivery',/\b(delivery|shipment|courier)/i],
 ['catalogue',/\b(catalog|catalogue|product list)/i],['video',/\bvideo/i],['device reading',/\b(sensor|reading|telemetry)\b/i]];
const DP_CONS=[['strong','Strong: must be exact'],['eventual','Eventual: brief staleness is harmless'],['mixed','Mixed']];
const DP_CACHE=[['no','Never cache'],['short','Cache briefly'],['yes','Cache freely']];

function dpStore(){M.dataPolicy=M.dataPolicy||{};const D=M.dataPolicy;D.objects=D.objects||[];D.removed=D.removed||{};D.accepted=D.accepted||{};return D;}
const dpKey=n=>String(n||'').trim().toLowerCase();

/* Business objects: validation rows, the brief (or capabilities) by keyword, and data stores from discovery. */
function dpSuggestedObjects(){
 const out=[],add=(name,source,writers)=>{const k=dpKey(name);if(!k)return;const o=out.find(x=>dpKey(x.name)===k);if(o){if(writers)for(const w of writers)if(!o.writers.includes(w))o.writers.push(w);return;}out.push({name:String(name).trim(),source,writers:[...(writers||[])]});};
 for(const r of ((M.validation&&M.validation.rows)||[]))if(String(r.object||'').trim())add(r.object,'Architecture validation');
 const P=M.product||{},txt=[P.problem,P.users,P.journeys,P.mvp,P.integrations].join(' ')+' '+((typeof decCtxs==='function'?decCtxs():[]).map(c=>c.name+' '+(c.description||'')).join(' '));
 for(const [n,re] of DP_KEYWORDS)if(re.test(txt))add(n,isNewProduct()?'Product brief':'Business areas');
 if(!isNewProduct()&&M.inventory){const ctx=id=>((M.inventory.contexts||[]).find(c=>c.id===id)||{}).name||id;
  for(const s of M.inventory.dataStores||[])if(s.name&&s.kind!=='cache'&&s.kind!=='queue')add(s.name,'Discovery › data store',[].concat(s.writers||[]).map(ctx));}
 return out;
}
function dpObjects(){
 const D=dpStore(),out=[];
 for(const o of dpSuggestedObjects())if(!D.removed[dpKey(o.name)])out.push(o);
 for(const o of D.objects)if(!out.some(x=>dpKey(x.name)===dpKey(o.name)))out.push({name:o.name,source:'Added by you',writers:o.writers||[]});
 return out.slice(0,30);
}

function dpRequest(){
 const pf=typeof platformEffective==='function'?platformEffective():{};const lab=(d,id)=>id?((optOf(d,id)||[,id])[1]):'';
 const cut=(t,n)=>String(t||'').replace(/\s+/g,' ').trim().slice(0,n);
 let B=null;try{B=systemBlueprint(decCtxs(),'current');}catch(e){}
 const layers=B?B.platform.filter(l=>['database','cache','messaging','dr','availability','security','files'].includes(l.id)&&String(l.t||'').trim()).map(l=>({id:cut(l.id,40),label:cut(l.label,80),design:cut(l.t,1500)})):[];
 const I=(()=>{try{return techPlanInputs({id:'system',name:'',description:''});}catch(e){return {regs:[]};}})();
 const rec=typeof capRecovery==='function'?capRecovery():{rto:'',rpo:''};const P=M.product||{};
 const targets=[P.availability?'Availability: '+P.availability+'%':'',rec.rto?'RTO: '+rec.rto:'RTO: not decided',rec.rpo?'RPO: '+rec.rpo:'RPO: not decided',
  (I.regs||[]).length?'Regulations: '+I.regs.join(', '):'',P.deploy&&P.deploy.region?'Region: '+P.deploy.region:''].filter(Boolean).map(t=>cut(t,300));
 const cons=o=>{const r=((M.validation&&M.validation.rows)||[]).find(r=>dpKey(r.object)===dpKey(o.name));return r&&/exactly one|never negative|one paid|only one|no double/i.test(r.measure||'')?'must be exact':'';};
 return {project:cut(M.project,180),kind:isNewProduct()?'new':(typeof modeOf==='function'&&modeOf()==='extend'?'extend':'modernize'),
  platform:['Tier: '+(pf.tier||''),'Cloud: '+lab('cloud',pf.cloud),'Database: '+lab('database',pf.database),'Architecture: '+lab('architecture',pf.architecture)].filter(s=>!/: $/.test(s)).map(s=>cut(s,300)),
  targets,layers:layers.slice(0,12),
  objects:dpObjects().map(o=>({name:cut(o.name,60),source:cut(o.source,120),writers:(o.writers||[]).slice(0,12).map(w=>cut(w,120)).filter(Boolean),sensitivity:'',consistency:cut(cons(o),40)})),
  journeys:((M.validation&&M.validation.rows)||[]).filter(r=>String(r.journey||'').trim()).slice(0,30).map(r=>({journey:cut(r.journey,200),failure:cut(r.failure,400),measure:cut(r.measure,400)}))};
}
function dpBasis(req){const r=req||dpRequest();return JSON.stringify({p:r.platform,l:r.layers.map(l=>l.id+':'+l.design),o:r.objects.map(o=>o.name+':'+o.writers.join('|')),t:r.targets});}
function dpStaleReasons(){
 const ai=dpStore().ai;if(!ai||!ai.basis)return [];let was;try{was=JSON.parse(ai.basis);}catch(e){return ['the design changed'];}
 const now=JSON.parse(dpBasis()),out=[];
 if(JSON.stringify(was.p)!==JSON.stringify(now.p))out.push('the platform decision changed');
 if(JSON.stringify(was.l)!==JSON.stringify(now.l))out.push('the database, cache or recovery design changed');
 if(JSON.stringify(was.o)!==JSON.stringify(now.o))out.push('the business objects or their writers changed');
 if(JSON.stringify(was.t)!==JSON.stringify(now.t))out.push('the targets changed');
 return out;
}

async function runDataPolicyAgent(){
 if(DP_UI.busy)return;
 const req=dpRequest();if(!req.objects.length){DP_UI.message='Add at least one business object first.';render();return;}
 DP_UI.busy=true;DP_UI.message='The Data policy agent is reading the design…';render();
 const t0=Date.now();
 try{
  const out=await agentFetch('/api/design/datapolicy',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(req)});
  dpStore().ai={...out,at:new Date().toISOString(),ms:Date.now()-t0,basis:dpBasis(req)};
  logInput('datapolicy:agent',null,(out.data||[]).length+' data rules, '+(out.cache||[]).length+' cache rules, '+(out.conflicts||[]).length+' conflicts');
  DP_UI.message='';persist();
 }catch(e){DP_UI.message=/404|not found/i.test(e.message)?'This backend needs the Data policy agent update. Restart the project backend with the latest build.':'The Data policy agent could not finish. '+e.message;}
 finally{DP_UI.busy=false;render();}
}

/* Why the backend checks dropped parts of the draft (reasons only, never the dropped text). */
function dpDropNotes(ai){
 const n=ai.dropped||0,notes=ai.dropNotes||[];
 const head=n+(n===1?' suggestion was':' suggestions were')+' dropped by the checks'+(notes.length?'':' (this backend does not report the reasons yet)');
 if(!notes.length)return h('small',{class:'muted',text:head+'.'});
 return h('details',{class:'dp-drops'},[h('summary',{text:head+': see why'}),h('ul',{class:'plain-list'},notes.map(t=>h('li',{text:t})))]);
}
function dpDraft(name){const ai=dpStore().ai||{};const k=dpKey(name);return {data:(ai.data||[]).find(d=>dpKey(d.object)===k)||null,cache:(ai.cache||[]).find(c=>dpKey(c.object)===k)||null};}
function dpPolicy(name){const a=dpStore().accepted[dpKey(name)];return a?{...a,accepted:true}:{...dpDraft(name),accepted:false};}

/* Local check: caching that contradicts a validation journey's acceptance measure. */
function dpConflicts(){
 const out=[];
 for(const o of dpObjects()){const p=dpPolicy(o.name);if(!p.cache||p.cache.cacheable==='no')continue;
  for(const r of ((M.validation&&M.validation.rows)||[]))if(dpKey(r.object)===dpKey(o.name)&&/exactly one|never negative|one paid|only one|no double/i.test(r.measure||''))
   out.push(o.name+' may be cached ('+(p.cache.cacheable==='short'?'briefly'+(p.cache.maxStaleness?', up to '+p.cache.maxStaleness:''):'freely')+'), but the journey “'+r.journey+'” requires: '+r.measure);}
 return out;
}

function dpAccept(name){
 const d=dpDraft(name),D=dpStore();
 D.accepted[dpKey(name)]={object:name,data:d.data?{...d.data}:{object:name,owner:'',authoritativeWriter:'',consistency:'',retention:'',classification:'',backup:'',access:'',why:''},
  cache:d.cache?{...d.cache}:{object:name,cacheable:'',maxStaleness:'',invalidation:'',onCacheFailure:'',personalData:'',why:''},by:M.actor||'',byName:M.actor&&typeof personLabel==='function'?personLabel(M.actor):'Local user',at:new Date().toISOString()};
 logInput('datapolicy:accept:'+dpKey(name),null,'accepted');persist();render();
}

function dpField(label,value,on,ph){const id='dp-'+Math.random().toString(36).slice(2);return h('label',{class:'context-control val-field',for:id},[h('span',{text:label}),h('textarea',{id,class:'field',rows:'2',placeholder:ph||'',text:value||'',oninput:e=>{on(e.target.value);persist();}})]);}
function dpSel(label,value,opts,on){const id='dps-'+Math.random().toString(36).slice(2);const s=h('select',{id,class:'field',onchange:e=>{on(e.target.value);persist();render();}},[h('option',{value:'',text:'Not decided'}),...opts.map(([v,t])=>h('option',{value:v,text:t}))]);s.value=value||'';return h('label',{class:'context-control val-field',for:id},[h('span',{text:label}),s]);}
const dpLabel=(list,v)=>(list.find(x=>x[0]===v)||[,v||'—'])[1];

function dpObjectCard(o){
 const p=dpPolicy(o.name),A=dpStore().accepted[dpKey(o.name)];
 const head=h('div',{class:'val-head'},[h('b',{text:o.name}),h('div',{class:'val-badges'},[
  h('span',{class:'val-badge '+(A?'ok':p.data||p.cache?'warn':'muted'),text:A?'Policy accepted'+(A.byName?' by '+A.byName:''):p.data||p.cache?'Draft from the agent: not accepted':'No policy yet'}),
  p.cache&&p.cache.cacheable?h('span',{class:'val-badge '+(p.cache.cacheable==='no'?'ok':'muted'),text:dpLabel(DP_CACHE,p.cache.cacheable)}):null])]);
 const meta=h('p',{class:'muted',text:o.source+(o.writers&&o.writers.length?' · written by '+o.writers.join(', '):'')});
 const removeBtn=button('Remove object',()=>{const D=dpStore();D.removed[dpKey(o.name)]=true;D.objects=D.objects.filter(x=>dpKey(x.name)!==dpKey(o.name));delete D.accepted[dpKey(o.name)];logInput('datapolicy:remove',o.name,null);persist();render();},'btn small');
 if(A){const d=A.data,c=A.cache;
  return h('article',{class:'val-row v-enough'},[head,meta,
   h('h4',{class:'val-sub',text:'Data policy'}),
   h('div',{class:'val-grid'},[dpField('Owner (a role)',d.owner,v=>d.owner=v),dpField('Only system allowed to write it',d.authoritativeWriter,v=>d.authoritativeWriter=v),
    dpSel('Consistency',d.consistency,DP_CONS,v=>d.consistency=v),dpField('Retention',d.retention,v=>d.retention=v),dpField('Classification and residency',d.classification,v=>d.classification=v),
    dpField('Backup and recovery',d.backup,v=>d.backup=v),dpField('Who may read and change it',d.access,v=>d.access=v)]),
   h('h4',{class:'val-sub',text:'Caching policy'}),
   h('div',{class:'val-grid'},[dpSel('May it be cached?',c.cacheable,DP_CACHE,v=>c.cacheable=v),dpField('Maximum staleness',c.maxStaleness,v=>c.maxStaleness=v,'e.g. 30 seconds'),
    dpField('How entries are removed',c.invalidation,v=>c.invalidation=v),dpField('When the cache is down',c.onCacheFailure,v=>c.onCacheFailure=v),dpField('Personal data in the cache',c.personalData,v=>c.personalData=v)]),
   h('div',{class:'cost-actions'},[button('Back to draft',()=>{delete dpStore().accepted[dpKey(o.name)];logInput('datapolicy:unaccept:'+dpKey(o.name),'accepted',null);persist();render();},'btn small'),removeBtn])]);}
 const d=p.data,c=p.cache,row=(k,v)=>v?h('tr',{},[h('th',{scope:'row',text:k}),h('td',{text:v})]):null;
 return h('article',{class:'val-row '+(d||c?'v-assume':'')},[head,meta,
  d||c?h('div',{class:'tp-scroll'},h('table',{class:'tp-table sb-table'},h('tbody',{},[
   d?row('Owner',d.owner):null,d?row('Only writer',d.authoritativeWriter):null,d?row('Consistency',dpLabel(DP_CONS,d.consistency)):null,d?row('Retention',d.retention):null,
   d?row('Classification',d.classification):null,d?row('Backup',d.backup):null,d?row('Access',d.access):null,d?row('Why',d.why):null,
   c?row('Caching',dpLabel(DP_CACHE,c.cacheable)+(c.maxStaleness?' · up to '+c.maxStaleness:'')):null,c?row('Removing entries',c.invalidation):null,c?row('Cache down',c.onCacheFailure):null,c?row('Personal data',c.personalData):null,c?row('Why (cache)',c.why):null]))):
  h('p',{class:'muted',text:'Ask the agent for a draft, or write the policy yourself.'}),
  h('div',{class:'cost-actions'},[button(d||c?'Accept as policy':'Write the policy myself',()=>dpAccept(o.name),'btn small primary'),removeBtn])]);
}

function dpDataPage(){
 const D=dpStore(),objs=dpObjects(),ai=D.ai,stale=dpStaleReasons(),local=dpConflicts();
 const acc=objs.filter(o=>D.accepted[dpKey(o.name)]).length;
 const addIn=h('input',{class:'field',type:'text',placeholder:'e.g. loyalty points',maxlength:'60','aria-label':'New business object'});
 const add=()=>{const v=addIn.value.trim();if(!v)return;delete D.removed[dpKey(v)];if(!D.objects.some(o=>dpKey(o.name)===dpKey(v)))D.objects.push({name:v,writers:[]});logInput('datapolicy:add',null,v);persist();render();};
 const removed=Object.keys(D.removed);
 return h('div',{class:'val-page'},[
  pageHeader('✓ / DATA AND CACHING POLICY','Who owns each kind of data, and may it be cached?',
   'One policy per business object: its owner, the only system allowed to write it, how exact it must be, how long it is kept, how it is protected and backed up, and whether it may be cached. The Data policy agent drafts them; a draft becomes policy only when you accept it.',
   [button('Architecture validation',()=>go('validation'),'btn'),button('Architecture (HLD)',()=>go('hld'),'btn')]),
  h('section',{class:'summary-section'},[h('h2',{text:'Data policy agent'}),
   h('p',{class:'muted',text:'Asks Claude on the project backend to draft a data and caching policy for each business object. Sends, once per press: the platform decision, the availability, recovery and regulation targets, the database, cache, messaging and recovery design, the business objects with the systems that write them, and the validation journeys with their acceptance measures. No records, no names of people, no secrets.'}),
   h('div',{class:'cost-actions'},[h('button',{type:'button',class:'btn small primary',disabled:DP_UI.busy||!objs.length?'':null,onclick:runDataPolicyAgent,text:DP_UI.busy?'Drafting…':ai?'Ask again':'Ask the Data policy agent'}),
    h('span',{class:'muted',text:acc+' of '+objs.length+' policies accepted'})]),
   DP_UI.message?h('p',{class:'input-status open',text:DP_UI.message}):null,
   ai?h('div',{class:'summary-callout '+(stale.length?'stale':'approved')},[h('b',{text:stale.length?'Draft out of date: '+stale.join('; ')+'. Ask again.':'Draft from '+(ai.model||'Claude')+', '+new Date(ai.at).toLocaleString()}),
    ai.summary?h('p',{text:ai.summary}):null,ai.dropped?dpDropNotes(ai):null]):null]),
  local.length||(ai&&(ai.conflicts||[]).length)?h('section',{class:'summary-section'},[h('h2',{text:'Conflicts'}),
   h('ul',{class:'plain-list'},[...local.map(t=>h('li',{class:'warn-text',text:t+' (found by Pathfinder)'})),...((ai&&ai.conflicts)||[]).map(t=>h('li',{text:t+' (from the agent)'}))])]):null,
  ai&&(ai.questions||[]).length?h('section',{class:'summary-section'},[h('h2',{text:'Questions for the data owner'}),h('ul',{class:'plain-list'},ai.questions.map(q=>h('li',{text:q})))]):null,
  h('section',{class:'summary-section'},[h('h2',{text:'Business objects'}),
   objs.length?h('div',{class:'val-list'},objs.map(dpObjectCard)):h('p',{class:'muted',text:'No business objects yet. Add them below, or name the business object on an Architecture validation row.'}),
   h('div',{class:'people-add'},[addIn,button('Add object',add,'btn small')]),
   removed.length?h('p',{class:'muted'},['Removed: '+removed.join(', ')+' ',button('Restore all',()=>{D.removed={};persist();render();},'linkbtn')]):null])]);
}

/* Two tabs: data and caching, cookies and sessions (assets/session-policy.js). */
function dataPolicyPage(){
 const tab=typeof sessionPolicyTab==='function'&&DP_UI.tab==='sessions'?'sessions':'data';
 const page=dpDataPage();
 const t=(id,label)=>h('button',{type:'button',role:'tab','aria-selected':tab===id?'true':'false',class:'type-chip'+(tab===id?' on':''),onclick:()=>{DP_UI.tab=id;render();}},[h('b',{text:label})]);
 const bar=typeof sessionPolicyTab==='function'?h('div',{class:'load-profiles dp-tabs',role:'tablist','aria-label':'Policy'},[t('data','Data and caching'),t('sessions','Cookies and sessions')]):null;
 if(tab==='sessions')return h('div',{class:'val-page'},[pageHeader('✓ / COOKIE AND SESSION POLICY','How long may people stay signed in, and which cookies need consent?',
   'Cookies by category, with consent, attributes and lifetime; and per user type: where sessions are kept, timeouts, signing in again for sensitive actions, multi-factor sign-in for staff, sign-out everywhere and what happens during an outage. The Session policy agent drafts them; the privacy owner accepts the cookie policy and the security owner the session policy.',
   [button('Architecture validation',()=>go('validation'),'btn'),button('Architecture (HLD)',()=>go('hld'),'btn')]),bar,sessionPolicyTab()]);
 if(bar)page.children[0].after(bar);
 return page;
}
function dataPolicyStatus(){try{const o=dpObjects(),a=o.filter(x=>dpStore().accepted[dpKey(x.name)]).length;return o.length?a+' of '+o.length+' accepted'+(dpConflicts().length?' · conflicts':''):'Owner, writer and caching per object';}catch(e){return '';}}
function dataPolicyExport(){
 const D=dpStore();
 return {objects:dpObjects().map(o=>{const p=dpPolicy(o.name);return {object:o.name,source:o.source,writers:o.writers||[],status:p.accepted?'accepted':(p.data||p.cache)?'draft':'none',
   acceptedBy:p.accepted?(D.accepted[dpKey(o.name)].byName||''):'',acceptedAt:p.accepted?D.accepted[dpKey(o.name)].at:'',data:p.data||null,cache:p.cache||null};}),
  conflicts:dpConflicts(),agentConflicts:(D.ai&&D.ai.conflicts)||[],questions:(D.ai&&D.ai.questions)||[],agent:D.ai?{model:D.ai.model||'',at:D.ai.at,stale:dpStaleReasons()}:null};
}
