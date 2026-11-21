/* MindSuite Pathfinder · HLD explanations: why each component exists, who chose it, which need it serves,
   what is still open; the business journeys from the brief; organization rules kept apart from project
   decisions; and what a saved AI diagram review was based on (so it can be marked stale).
   Uses capabilities.js (capStatus) and the component model of hld.js. Never invents values: a missing input
   becomes an open question. */

const HLD_NODE_CAP={dns:'dns-certs',cdn:'cdn-waf',waf:'cdn-waf',lb:'cdn-waf',apigw:'api-gateway',iothub:'iot',app:'app-runtime',workers:'background-jobs',jobs:'background-jobs',ingest:'iot',
 db:'relational',dbreplica:'relational',cache:'cache',mq:'queue',search:'search',object:'object-storage',tsdb:'analytics',backupvault:'backups',privep:'private-network',nat:'private-network',bastion:'private-network',
 vpn:'site-connectivity',idstaff:'workforce-sso',idcust:'customer-signin',vault:'secrets',appconfig:'infra-code',metrics:'logs-metrics',logs:'logs-metrics',apm:'logs-metrics',dash:'logs-metrics',audit:'audit',siem:'audit',
 registry:'cicd',cicd:'cicd',policy:'infra-code',drsite:'disaster-recovery',pay:'payments',notify:'notifications',ai:'ai',other:'webhooks'};
const HLD_STATUS={selected:'Selected',required:'Required by policy',suggested:'Suggested',approval:'Needs approval',unresolved:'Unresolved',conflict:'Needs a decision','needs-partner':'Partner needed',design:'Design choice'};

function hldOwner(capStat){
 const R=(M.decide&&M.decide.platform)||{},people=M.people||[];
 const nameOf=id=>{const p=people.find(x=>x.id===id);return p?p.name+' ('+p.role+')':'';};
 if(capStat&&capStat.status==='selected'&&R.by)return nameOf(R.by)||'recorded decision';
 if(capStat&&capStat.status==='required'){try{const prov=capStat.prov,rule=((M.decide.rules.services||{})[prov]||{})[capStat.id]||{};if(rule.owner)return nameOf(rule.owner);}catch(e){}return 'Organization policy';}
 const arch=people.find(p=>/architect/i.test(p.role));return arch?'Not assigned (suggest '+arch.name+')':'Not assigned';
}
function hldOpenQuestions(capId,m){
 const P=M.product||{},I=m.inputs||{},rec=capRecovery(),q=[];
 if(capId==='app-runtime'){if(!num(P.peak))q.push('Peak load not given');else if(P.peakBasis==='estimated-from-users')q.push('Peak load is estimated from users, not measured');if(!String(P.concurrency||'').trim())q.push('Concurrency target not given');}
 if(capId==='relational'&&!num(P.dataGb))q.push('Data volume not given');
 if(capId==='disaster-recovery'){if(!rec.rto)q.push('Recovery time (RTO) not decided');if(!rec.rpo)q.push('Data loss allowed (RPO) not decided');if(!String((P.deploy||{}).regions||'').trim())q.push('Second region not chosen (no residency answer)');}
 if(capId==='cdn-waf'&&I.f&&I.f.video)q.push('Viewing hours, bitrate and delivery geography unknown: delivery cost not estimated');
 if(capId==='payments')q.push('Payment provider not chosen');
 if(capId==='customer-signin')q.push('Social sign-in providers and MFA policy for customers');
 if(capId==='notifications')q.push('Which messages, channels and countries');
 if(capId==='cost-control'&&!num(((M.decide||{}).businessCase||{}).targetRun))q.push('Budget limit not set');
 return q;
}
/* One record per diagram component: what it does, which need it serves, technology, status, why, owner, open. */
function hldComponentDetails(m,cm){
 const pv=capApprovedProvider();const provName=(CAP_PROVIDERS.find(p=>p[0]===pv.id)||[,''])[1];
 const L=id=>(m.blueprint.platform.find(l=>l.id===id)||{});
 const layerFor={'app-runtime':'hosting',relational:'database',cache:'cache',queue:'messaging',search:'search','object-storage':'files','cdn-waf':'north','api-gateway':'north','customer-signin':'identity','workforce-sso':'identity',secrets:'security','logs-metrics':'observability','disaster-recovery':'dr',backups:'dr',payments:'payments',iot:'iot',ai:'ai','private-network':'north','background-jobs':'messaging',cicd:'operations','infra-code':'operations',notifications:'missed',audit:'observability'};
 return cm.nodes.filter(n=>!['actors'].includes(n.zone)).map(n=>{
  const capId=HLD_NODE_CAP[n.id]||'';const cs=capId?capStatus(capId):null;
  let status=cs?cs.status:(n.zone==='ext'?'suggested':'suggested');
  if(/Open decision/.test(n.sub))status='unresolved';else if(/needs approval/.test(n.sub))status='approval';else if(/needs a decision/.test(n.sub))status='conflict';
  if((n.id==='vendor'||n.id==='partner'))status='selected';
  const layer=L(layerFor[capId]||'');
  const conn=cm.edges.filter(e=>e.from===n.id||e.to===n.id).map(e=>{const o=cm.nodes.find(x=>x.id===(e.from===n.id?e.to:e.from));return o?o.title+(e.label?' ('+e.label+')':''):'';}).filter(Boolean);
  return {id:n.id,capability:capId,title:n.title,zone:n.zone,purpose:cs?cs.cap[3]:(n.zone==='ext'?'Outside service the product depends on':''),
   serves:cs?(cs.evidence||'Baseline for every product'):(n.zone==='ext'?n.sub:''),technology:n.sub,provider:n.zone==='ext'?'external':(provName||'not approved yet'),
   status,statusLabel:HLD_STATUS[status]||status,why:layer.why||(cs&&cs.note)||'',owner:hldOwner(cs),connects:conn,open:capId?hldOpenQuestions(capId,m):[]};
 });
}

/* Two business journeys from the brief. Each step names the component, what it reads or writes, and how it fails safely. */
function hldJourneys(m,cm){
 const P=M.product||{},has=id=>cm.nodes.some(n=>n.id===id),t=id=>(cm.nodes.find(n=>n.id===id)||{}).title||'';
 const segs=String(P.journeys||'').split(/\s*(?:\d\)|\n|;)\s*/).map(x=>x.trim()).filter(x=>x.length>8).slice(0,4);
 const pick=[];
 const I=m.inputs||{f:{}};
 if(I.f.video){pick.push({actor:'Content editor',goal:'Uploads a video and makes it ready to publish',kind:'publish'});pick.push({actor:'Subscriber',goal:'Signs in, proves entitlement and starts playback',kind:'playback'});}
 for(const sgm of segs){if(pick.length>=2)break;const actor=(sgm.match(/^([A-Za-z][A-Za-z /-]{2,30}):/)||[])[1]||'User';pick.push({actor:actor.trim(),goal:sgm.replace(/^[^:]{2,32}:\s*/,'').slice(0,160),kind:'brief'});}
 if(!pick.length)pick.push({actor:'User',goal:'Completes the main task in the first release',kind:'brief'});
 const entry=[has('dns')&&'DNS',has('cdn')&&t('cdn'),has('waf')&&t('waf'),has('apigw')&&t('apigw')].filter(Boolean).join(' → ');
 const roles=(typeof tpRoles==='function'?tpRoles(P.users):[]);
 return pick.slice(0,2).map(j=>{
  const g=(j.goal+' '+j.actor).toLowerCase(),steps=[];const step=(c,action)=>{if(c&&has(c))steps.push({component:t(c),id:c,action});};
  const staffy=/staff|editor|admin|restaurant|agent|manager|operator/.test(j.actor.toLowerCase());
  step(staffy?'idstaff':'idcust','sign in ('+(staffy?'staff SSO with MFA':'customer account')+')');
  const myRole=roles.find(r=>j.actor.toLowerCase().includes(String(r).toLowerCase())||String(r).toLowerCase().includes(j.actor.toLowerCase()))||j.actor;
  step('app','checks the role ('+myRole+')'+(j.kind==='playback'?' and the subscription entitlement':'')+' and runs the request');
  if(j.kind==='publish'){step('object','stores the original upload through a short-lived signed URL');step('mq','queues a processing job');step('workers','processes the video into renditions (transcoding)');step('object','stores the renditions');step('db','records the title and its publish status');}
  else if(j.kind==='playback'){step('cache','reads the catalogue and resume position');step('db','checks the subscription status');step('cdn','delivers the stream (and DRM licence from the partner)');}
  else{
   if(/search|browse|find|menus?|catalog/.test(g)){step('search','finds items');step('cache','serves repeated reads');}
   if(/basket|order|book|place|accept|reject|ready|update/.test(g))step('db','writes the order or record in one transaction');
   if(/pay|checkout|billing/.test(g)){step('pay','takes the payment (tokens only, no card data)');step('mq','processes the payment webhook once');}
   if(/track|status|notify|alert/.test(g))step('notify','sends status updates');
   if(/upload|photo|image|file/.test(g))step('object','stores the file');
   if(!steps.some(x=>x.id==='db'))step('db','reads and writes the records');
  }
  const failure=[j.kind==='publish'?'failed processing goes to the dead-letter queue and the editor sees “failed, retry”':'',
   steps.some(x=>x.id==='pay')?'payment webhooks are verified and processed once; a reconciliation job catches gaps':'',
   'calls retry with backoff; the circuit breaker stops cascading failures','if the region fails, the disaster-recovery plan applies'].filter(Boolean);
  const signals=[j.kind==='playback'?'playback start time, rebuffering, CDN errors':j.kind==='publish'?'processing time and failures per upload':'response time (p95) and error rate for this journey',
   steps.some(x=>x.id==='pay')?'failed payments and webhook backlog':'','sign-in failures'].filter(Boolean);
  return {actor:j.actor,goal:j.goal,entry:entry||'the app',steps,authorization:(staffy?'staff role checked in the app':'customer can act only on their own data')+(j.kind==='playback'?'; entitlement from the subscription status':''),failure,signals,source:j.kind==='brief'?'Brief › Key journeys':'Brief describes video (publish and playback journeys)'};
 });
}

/* Organization rules (policy) kept apart from project decisions. */
function hldPolicyAndDecisions(){
 const R=M.decide.rules,rules=[];
 for(const dim of Object.keys(TARGETS))for(const [id,label] of TARGETS[dim].opts){const st=R.stance[dim][id];if(st&&st!=='allowed')rules.push({area:TARGETS[dim].title,option:label,rule:RULE_STATE[st][1]});}
 for(const [prov,svcs] of Object.entries(R.services||{}))for(const [id,r] of Object.entries(svcs||{})){if(!r||!r.state||r.state==='allowed')continue;const sv=((typeof CLOUD_SERVICES!=='undefined'?CLOUD_SERVICES[prov]:[])||[]).find(x=>x[0]===id);rules.push({area:'Cloud service ('+prov+')',option:sv?sv[1]:id,rule:RULE_STATE[r.state][1]});}
 const pf=platformEffective(),pr=platformRec();
 const decisions=[];
 if(pf.decided)decisions.push({what:'Platform',value:[pf.tier+' tier',(optOf('cloud',pf.cloud)||[,pf.cloud])[1],(optOf('architecture',pf.architecture)||[,pf.architecture])[1],(optOf('database',pf.database)||[,pf.database])[1]].filter(Boolean).join(', '),by:hldOwner({status:'selected'})});
 else decisions.push({what:'Platform',value:'Not recorded — the HLD uses suggestions',by:'—'});
 for(const c of decCtxs()){const d=decisionFor(c);if(d.status==='decided')decisions.push({what:c.name,value:apprLabel(d.approach),by:d.decidedBy?personLabel(d.decidedBy):'—'});}
 for(const id of Object.keys((M.decide&&M.decide.capDismissed)||{})){const c=typeof CAP_BY_ID!=='undefined'&&CAP_BY_ID[id];if(c)decisions.push({what:c[2],value:'Declined: marked “Not needed” (removed from the diagram and cost)',by:'—'});}
 return {rules,decisions};
}

/* What the AI diagram review was based on, to detect staleness. */
function hldBasis(m,cm){
 const pf=platformEffective(),R=M.decide.rules;
 const svc=JSON.stringify(R.services||{}),dims=JSON.stringify(R.stance||{});
 const hash=t=>{let x=0;for(let i=0;i<t.length;i++)x=(x*31+t.charCodeAt(i))|0;return String(x);};
 return {provider:pf.cloud||'',hosting:pf.hosting||'',tier:pf.tier||'',architecture:pf.architecture||'',database:pf.database||'',
  serviceRules:hash(svc),technologyRules:hash(dims),network:cm.nodes.filter(n=>n.zone==='net').map(n=>n.id).sort().join(','),components:cm.nodes.length,declined:Object.keys((M.decide&&M.decide.capDismissed)||{}).sort().join(',')};
}
function hldStaleReasons(saved,now){
 if(!saved)return [];const out=[];const lab={provider:'provider',hosting:'hosting model',tier:'tier',architecture:'architecture',database:'database',serviceRules:'cloud-service rules',technologyRules:'technology rules',network:'network boundary',declined:'declined capabilities'};
 for(const k of Object.keys(lab))if(String(saved[k]||'')!==String(now[k]||''))out.push(lab[k]+(['provider','hosting','tier','architecture','database'].includes(k)?' changed from '+(saved[k]||'none')+' to '+(now[k]||'none'):' changed'));
 return out;
}
