/* MindSuite web app · organization technology rules (model, scoring effect, presets, import/export)
   Used by 05 Rules & people (assets/rules-page.js) and by the option ranking in 06 Target decisions.

   Stored per workspace in M.decide.rules:
     stance[area][optionId]   'standard' | 'preferred' | 'allowed' | 'approval' | 'retiring' | 'excluded'
                              (older workspaces only have preferred / allowed / excluded; still valid)
     ruleInfo[area][optionId] {reason, owner (person id), reviewBy, exitBy (retiring only), at, by}
     custom[area]             [{id, label, hint}]  options the organization adds itself

   Loaded by index.html before the main script. ensureDecide() calls syncRuleCatalog(R), which
   appends the custom options to TARGETS so ranking, decision cards and exports see them.
   rankOptions() calls ruleStanceEffect() for every option instead of its old preferred/excluded check. */

const RULE_STATES=[
 ['standard','Standard','◆','The organization standard. Use it unless there is a strong reason not to.'],
 ['preferred','Preferred','★','Encouraged. Ranked higher than allowed options.'],
 ['allowed','Allowed','','No rule. Ranked on the evidence only.'],
 ['approval','Needs approval','?','Allowed only with approval, for example from the architecture board.'],
 ['retiring','Retiring','⏳','No new use. Existing use must end by the exit date.'],
 ['excluded','Excluded','✕','Never recommended and cannot be chosen.']];
const RULE_STATE=Object.fromEntries(RULE_STATES.map(s=>[s[0],s]));

/* ---- Catalogue: base options plus the organization's own ---- */
let TARGET_BASE=null;
// Shared technologies retain their IDs so rules saved by earlier versions still apply.
const PRODUCT_OPTIONS={
 architecture:[['clientapi','Client app with a backend API','Web or mobile clients share an application API'],'modular','microservices','events','serverless'],
 language:['java','dotnet','node','python','go'],
 database:['postgres','mysql','sqlserver',['oracle','Oracle Database','Relational database for the application'],'nosql',['sqlite','SQLite (embedded)','Store data locally in an application or device']],
 integration:[['api','REST APIs','Connect clients and partners over HTTP'],['webhook','Webhooks','Notify a partner when something happens'],'events','queue','ipaas','mft',['cdc','Change data capture','Publish database changes to downstream consumers']],
 cloud:['aws','azure','gcp','oci','ibm','private',['onprem','On-premises hosting','Deploy to infrastructure operated by your organization or customer']]
};
let PRODUCT_BASE=null;
function syncRuleCatalog(R){
 R.ruleInfo=R.ruleInfo||{};R.custom=R.custom||{};
 if(!TARGET_BASE)TARGET_BASE=Object.fromEntries(Object.keys(TARGETS).map(k=>[k,TARGETS[k].opts.slice()]));
 if(!PRODUCT_BASE)PRODUCT_BASE=Object.fromEntries(Object.entries(PRODUCT_OPTIONS).map(([dim,options])=>
  [dim,options.map(o=>Array.isArray(o)?o:TARGET_BASE[dim].find(([id])=>id===o))]));
 const base=isNewProduct()?PRODUCT_BASE:TARGET_BASE;
 for(const dim of Object.keys(TARGETS)){
  R.ruleInfo[dim]=R.ruleInfo[dim]||{};
  const own=(R.custom[dim]||[]).map(c=>[c.id,c.label,c.hint||'Added by your organization']);
  TARGETS[dim].opts=base[dim].concat(own);
  if(dim==='cloud')for(const [id,label] of own)if(!CLOUD_SVC[id])CLOUD_SVC[id]=customCloudServices(label);
 }
}
function customCloudServices(label){
 const svc={postgres:'PostgreSQL',mysql:'MySQL',sqlserver:'SQL Server',oracle:'Oracle Database',nosql:'Document database',api:'API gateway',events:'Event streaming',queue:'Message queue',ipaas:'Integration platform',mft:'Managed file transfer',cdc:'Change data capture',containers:'Kubernetes / containers',serverless:'Functions',vm:'Virtual machines',mainframe:'Mainframe hosting'};
 return Object.fromEntries(Object.entries(svc).map(([k,v])=>[k,v+' on '+label+' (confirm with the provider)']));
}
function customRuleId(label){return 'org-'+String(label).toLowerCase().replace(/[^a-z0-9]+/g,'-').replace(/^-|-$/g,'').slice(0,40)+'-'+Math.random().toString(36).slice(2,6);}

/* ---- Effect on ranking (called from rankOptions in index.html) ---- */
function ruleStanceEffect(dim,id,add,exclude){
 const R=M.decide.rules,st=R.stance[dim][id]||'allowed',info=(R.ruleInfo[dim]||{})[id]||{};
 const why=info.reason?': '+info.reason:'';
 if(st==='excluded')exclude('Excluded by organization rules'+why);
 if(st==='retiring')exclude('Being retired by organization rules'+(info.exitBy?' (exit by '+info.exitBy+')':'')+why);
 if(st==='standard')add('rule',3,'Organization standard'+why);
 if(st==='preferred')add('rule',2,'Preferred by organization rules'+why);
 if(st==='approval')add('rule',-1,'Needs approval under organization rules'+why);
 // In single-choice areas, a named standard counts against the others.
 if(st!=='standard'&&!TARGETS[dim].multi){
  const std=TARGETS[dim].opts.filter(([o])=>o!==id&&R.stance[dim][o]==='standard');
  if(std.length&&id!=='keep')add('rule',-1,'Not the organization standard ('+std.map(o=>o[1]).join(', ')+')');
 }
}

// A product forecast describes what will be built. Its future data volume is not data to migrate.
function scoreProductOption(dim,id,add,signals,c,d,R,exclude){
 const deploy=M.product?.deploy||{},channels=deploy.channels||[];
 switch(dim){
 case 'architecture':
  capArch(id,add,c);
  constraintArch(id,add);
  if(id==='modular')add('effort',1,'One application to deploy for the first release');
  if(id==='clientapi'&&channels.some(ch=>['web','mobile','api'].includes(ch)))add('fit',1,'The product brief names a web, mobile or API channel');
  if(id==='microservices')add('effort',-1,'Each service needs its own deployment and monitoring');
  break;
 case 'language':
  if(R.skills.includes(id))add('skills',2,'The team lists this language as a skill');
  break;
 case 'database':
  if(id==='sqlite'&&channels.some(ch=>['mobile','desktop','embedded'].includes(ch)))add('fit',1,'The planned client can store data on the device');
  if(num(M.product?.dataGb)>=10000)add('risk',0,'Check storage limits against the three-year product forecast');
  break;
 case 'integration':
  capInt(id,add,c);
  if(id==='api'&&channels.some(ch=>['mobile','api'].includes(ch)))add('fit',1,'The planned channel needs an application interface');
  if(id==='events'&&effective('architecture',d)==='events')add('fit',1,'The selected architecture publishes and consumes events');
  break;
 case 'cloud': {
  capCloud(id,add,c);
  regulationCloud(id,add);
  const publicCloud=['aws','azure','gcp','oci','ibm'].includes(id);
  if((['public','multicloud'].includes(deploy.where)&&publicCloud)||(['onprem','private'].includes(id)&&deploy.where===id))
   add('fit',2,'Matches the deployment environment in the product brief');
  if(R.residency==='region')add('risk',0,'Confirm availability in '+(R.region||'the required region'));

  if(publicCloud&&R.residency==='private'&&(signals.regulated||signals.sensitive))exclude('Sensitive data must use private infrastructure');
  const names={aws:['aws','amazon'],azure:['azure','microsoft'],gcp:['google','gcp'],oci:['oracle cloud','oci'],ibm:['ibm']};
  if((names[id]||[]).some(name=>R.contracts.toLowerCase().includes(name)))add('cost',1,'Covered by a recorded organization agreement');
  break;
 }
 }
}

/* ---- Changing one rule (logged, so the history shows who set what) ---- */
function setRule(dim,id,state,info){
 const R=M.decide.rules;
 const before={state:R.stance[dim][id]||'allowed',...((R.ruleInfo[dim]||{})[id]||{})};
 if(state==='allowed'&&!(info&&(info.reason||info.owner||info.reviewBy)))delete R.stance[dim][id];else R.stance[dim][id]=state;
 R.ruleInfo[dim]=R.ruleInfo[dim]||{};
 if(info&&Object.values(info).some(v=>String(v||'').trim()))R.ruleInfo[dim][id]={...info,at:new Date().toISOString(),by:actorName()};
 else delete R.ruleInfo[dim][id];
 logInput('rule:'+dim+':'+id,before,{state,...(info||{})});
}
function ruleCounts(){
 const R=M.decide.rules,n=Object.fromEntries(RULE_STATES.map(s=>[s[0],0]));let overdue=0;const today=new Date().toISOString().slice(0,10);
 for(const dim of Object.keys(TARGETS))for(const [id] of TARGETS[dim].opts){
  n[R.stance[dim][id]||'allowed']++;
  const i=(R.ruleInfo[dim]||{})[id];if(i&&i.reviewBy&&i.reviewBy<today)overdue++;
 }
 return {n,overdue};
}

/* ---- Presets ---- */
const RULE_PRESETS=[
 ['azure','Azure shop',{cloud:{azure:'standard',aws:'approval',gcp:'approval',oci:'approval',ibm:'approval'},language:{dotnet:'preferred'},database:{sqlserver:'preferred',postgres:'preferred'}}],
 ['aws','AWS shop',{cloud:{aws:'standard',azure:'approval',gcp:'approval',oci:'approval',ibm:'approval'},database:{postgres:'preferred',mysql:'preferred'}}],
 ['gcp','Google Cloud shop',{cloud:{gcp:'standard',aws:'approval',azure:'approval',oci:'approval',ibm:'approval'},database:{postgres:'preferred'}}],
 ['regulated','Regulated: keep data on-premises',{cloud:{onprem:'standard',private:'preferred',aws:'excluded',azure:'excluded',gcp:'excluded',oci:'excluded',ibm:'excluded'},architecture:{serverless:'excluded'}}],
 ['opensource','Open-source first',{database:{postgres:'standard',mysql:'preferred',oracle:'approval',sqlserver:'approval'},language:{java:'preferred',python:'preferred',go:'preferred'},cloud:{private:'preferred'}}],
 ['cautious','Low risk: gradual change',{architecture:{modular:'preferred',wrap:'preferred',microservices:'approval',events:'approval',serverless:'approval'},integration:{api:'preferred',cdc:'preferred'}}]];
const PRODUCT_PRESETS=[
 ['web-mvp','First-release web app',{architecture:{modular:'preferred'},database:{postgres:'preferred'},integration:{api:'preferred'}}],
 ['mobile-api','Mobile app with a shared API',{architecture:{clientapi:'preferred'},integration:{api:'standard'}}],
 ['event-product','Event-driven product',{architecture:{events:'preferred'},integration:{events:'preferred',queue:'preferred'}}]
];
function rulePresets(){
 return isNewProduct()?[...PRODUCT_PRESETS,...RULE_PRESETS.filter(([key])=>['azure','aws','gcp','opensource'].includes(key))]:RULE_PRESETS;
}
function applyRulePreset(key,replace){
 const p=rulePresets().find(x=>x[0]===key);if(!p)return 0;const R=M.decide.rules;let n=0;
 if(replace)for(const dim of Object.keys(TARGETS))for(const [id] of TARGETS[dim].opts){delete R.stance[dim][id];delete R.ruleInfo[dim][id];}
 for(const [dim,rules] of Object.entries(p[2]))for(const [id,state] of Object.entries(rules)){
  if(!TARGETS[dim].opts.some(o=>o[0]===id))continue;
  setRule(dim,id,state,{reason:'Preset: '+p[1],owner:M.actor||''});n++;
 }
 return n;
}

/* ---- Import / export: technology radar or our own rules CSV ----
   Columns (header names are matched loosely): name, ring or status, area (optional),
   reason, owner, review_by, exit_by, description. Ring Adopt / Trial / Assess / Hold maps to
   Preferred / Allowed / Needs approval / Retiring; our own state names are accepted too. */
const RING_MAP={adopt:'preferred',trial:'allowed',assess:'approval',hold:'retiring',
 standard:'standard',mandatory:'standard',preferred:'preferred',allowed:'allowed',approval:'approval','needs approval':'approval',
 retiring:'retiring',retire:'retiring',sunset:'retiring',deprecated:'retiring',excluded:'excluded',forbidden:'excluded',banned:'excluded'};
const RULE_ALIASES={
 cloud:{aws:['aws','amazon','amazon web services'],azure:['azure','microsoft azure'],gcp:['gcp','google cloud','google cloud platform'],oci:['oci','oracle cloud','oracle cloud infrastructure'],ibm:['ibm cloud'],private:['private cloud','openshift','vmware'],onprem:['on-premises','on premises','on-prem','data centre','data center']},
 language:{java:['java','spring','spring boot','jakarta ee'],dotnet:['.net','dotnet','c#','csharp','asp.net core'],node:['node','node.js','nodejs','typescript','javascript'],python:['python'],go:['go','golang']},
 database:{postgres:['postgres','postgresql'],mysql:['mysql'],sqlserver:['sql server','mssql','microsoft sql server'],oracle:['oracle','oracle database'],nosql:['nosql','mongodb','document database','cosmos db','dynamodb']},
 integration:{api:['api gateway','rest','rest api','apis'],events:['kafka','event streaming','event hubs','pub/sub'],queue:['message queue','rabbitmq','ibm mq','mq','service bus','sqs'],ipaas:['ipaas','mulesoft','boomi','logic apps'],mft:['mft','sftp','managed file transfer'],cdc:['cdc','change data capture','debezium','goldengate']},
 architecture:{modular:['modular monolith'],microservices:['microservices'],events:['event-driven','event driven','event-driven architecture'],serverless:['serverless','lambda','functions'],wrap:['api facade','api façade','strangler fig']}};
const normRule=s=>String(s||'').toLowerCase().replace(/\s+/g,' ').trim();
function matchRuleOption(name,areaHint,catalog=TARGETS){
 const n=normRule(name);const dims=areaHint&&TARGETS[areaHint]?[areaHint]:Object.keys(TARGETS);
 for(const dim of dims){
  const options=catalog===TARGETS?TARGETS[dim].opts:catalog[dim];
  for(const [id,label] of options){
   const l=normRule(label);if(n===l||n===normRule(label.split(' (')[0])||n===id)return [dim,id];
  }
  for(const [id,al] of Object.entries(RULE_ALIASES[dim]||{}))if(al.includes(n)&&options.some(o=>o[0]===id))return [dim,id];
 }
 return null;
}
function parseCsv(text){
 const rows=[];let row=[],cell='',q=false;
 for(let i=0;i<text.length;i++){const ch=text[i];
  if(q){if(ch==='"'){if(text[i+1]==='"'){cell+='"';i++;}else q=false;}else cell+=ch;continue;}
  if(ch==='"')q=true;else if(ch===','||ch===';'||ch==='\t'){row.push(cell);cell='';}
  else if(ch==='\n'||ch==='\r'){if(ch==='\r'&&text[i+1]==='\n')i++;row.push(cell);cell='';if(row.some(c=>c.trim()))rows.push(row);row=[];}
  else cell+=ch;}
 row.push(cell);if(row.some(c=>c.trim()))rows.push(row);
 return rows;
}
function importRuleCsv(text){
 const rows=parseCsv(text);if(rows.length<2)return {error:'The file has no rows under its header.'};
 const head=rows[0].map(normRule);
 const col=(...names)=>head.findIndex(x=>names.some(n=>x===n||x.replace(/[_ -]/g,'')===n.replace(/[_ -]/g,'')));
 const ci={name:col('name','technology','item','option'),ring:col('ring','status','state','stance'),area:col('area','dimension','category'),
  reason:col('reason','rationale','why','description'),owner:col('owner','decided by'),review:col('review_by','reviewby','review date','review'),exit:col('exit_by','exitby','exit date','sunset date')};
 if(ci.name<0||ci.ring<0)return {error:'The file needs a "name" column and a "ring" (or "status") column.'};
 const people=M.people||[];const out={applied:0,added:0,unknownRing:[],unmatched:[],outsideMode:[]};
 for(const r of rows.slice(1)){
  const name=(r[ci.name]||'').trim();if(!name)continue;
  const state=RING_MAP[normRule(r[ci.ring])];if(!state){out.unknownRing.push(name+' ('+(r[ci.ring]||'blank')+')');continue;}
  let area=ci.area>=0?normRule(r[ci.area]).replace(/ \/.*$/,''):'';
  area=area?Object.keys(TARGETS).find(k=>k===area||normRule(TARGETS[k].title).startsWith(area))||'':'';
  let hit=matchRuleOption(name,area);
  if(!hit){
   const alternate=matchRuleOption(name,area,isNewProduct()?TARGET_BASE:PRODUCT_BASE);
   if(alternate){
    if(!optOf(...alternate)){out.outsideMode.push(name);continue;}
    hit=alternate;
   }
  }
  if(!hit&&area){const R=M.decide.rules;R.custom[area]=R.custom[area]||[];const id=customRuleId(name);R.custom[area].push({id,label:name,hint:'Added by import'});syncRuleCatalog(R);hit=[area,id];out.added++;}
  if(!hit){out.unmatched.push(name);continue;}
  const ownerName=ci.owner>=0?(r[ci.owner]||'').trim():'';const owner=people.find(p=>normRule(p.name)===normRule(ownerName))?.id||'';
  const reason=[(ci.reason>=0?(r[ci.reason]||'').trim():''),ownerName&&!owner?'owner: '+ownerName:''].filter(Boolean).join('; ');
  setRule(hit[0],hit[1],state,{reason,owner,reviewBy:ci.review>=0?(r[ci.review]||'').trim():'',exitBy:state==='retiring'&&ci.exit>=0?(r[ci.exit]||'').trim():''});
  out.applied++;
 }
 return out;
}
function exportRuleCsv(){
 const R=M.decide.rules,esc=v=>{const s=String(v??'');return /[",\n;]/.test(s)?'"'+s.replace(/"/g,'""')+'"':s;};
 const lines=[['area','name','status','reason','owner','review_by','exit_by'].join(',')];
 for(const dim of Object.keys(TARGETS))for(const [id,label] of TARGETS[dim].opts){
  const st=R.stance[dim][id]||'allowed',i=(R.ruleInfo[dim]||{})[id]||{};
  lines.push([dim,label,RULE_STATE[st][1],i.reason||'',person(i.owner)?.name||'',i.reviewBy||'',i.exitBy||''].map(esc).join(','));
 }
 return lines.join('\n')+'\n';
}
