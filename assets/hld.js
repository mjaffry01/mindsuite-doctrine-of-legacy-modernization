/* MindSuite web app · High-level design (HLD) for architects — draft
   Generated from what the project already holds: the System blueprint (shared platform layers and the
   capabilities on it, Current picks), the cost per tier, the sizing inputs, the deployment answers, the
   decisions and the assumptions register. Nothing is sent anywhere; no model calls.
   The HLD stays a DRAFT while any assumption is open, any decision is open or nobody is named to approve;
   the page lists exactly what keeps it a draft.
   hldModel() returns the structured HLD (also the base for the other audience diagrams);
   hldDiagram(model) draws the architecture as SVG (users → edge → platform with its modules → data, with
   identity, bought products, partner platforms and other systems on the side); hldPage() is screen 'hld'.
   Downloads: Markdown (hldMarkdown) and the diagram as SVG; Print / save as PDF uses the app's print styles. */

const HLD_SECTIONS=['Context','Quality targets','Architecture diagram','Component diagram','Shared platform','Capabilities','Cost','Decisions','Open assumptions and risks'];

function hldModel(){
 ensureDecide();
 const product=isNewProduct(),C=decCtxs(),P=M.product||{},D=P.deploy||{};
 const B=systemBlueprint(C,'current');
 let X=null;try{X=tierCost(C,'current');}catch(e){}
 const sys={id:'system',name:M.project||'',description:C.map(c=>c.name).join(' ')};
 const I=techPlanInputs(sys);
 const open=typeof assumptionsOpen==='function'?assumptionsOpen():[];
 const decisions=C.map(c=>{const d=decisionFor(c);return {name:c.name,status:d.status==='decided'?'Decided':'Open',approach:d.approach&&d.approach!=='undecided'?apprLabel(d.approach):'',by:d.decidedBy?personLabel(d.decidedBy):'',at:d.at?String(d.at).slice(0,10):''};});
 const why=[];
 if(typeof platformEffective==='function'&&!platformEffective().decided)why.push('platform not decided');
 if(open.length)why.push(open.length+' assumption'+(open.length>1?'s':'')+' not confirmed or accepted');
 const openDec=decisions.filter(d=>d.status==='Open').length;if(openDec)why.push(openDec+' of '+decisions.length+' decisions open');
 if(!(M.people||[]).length)why.push('nobody named in Rules & people');
 const lbl=(list,id)=>((typeof list!=='undefined'?list:[]).find(x=>x[0]===id)||[,id])[1];
 const regs=I.regs||[];
 const nfr=[
  ['Users',I.users?tpN(I.users)+' ('+I.uSrc+')':'not given'],
  ['Peak load',I.peak?tpN(I.peak)+' requests/s ('+I.pSrc+'); sized for '+tpN(I.sized)+'/s with headroom'+(I.growth?' and '+I.growth+'% yearly growth':''):'not given'],
  ['Data after 3 years',I.dataGb?tpN(I.dataGb)+' GB ('+I.dSrc+')':'not given'],
  ['Availability',I.avail+'% ('+I.aSrc+'), about '+tpN((100-I.avail)/100*8760)+' hours down a year; recover within '+(I.avail>=99.95?'1 hour, lose at most 5 minutes':I.avail>=99.5?'4 hours, lose at most 15 minutes':'1 day, lose at most 24 hours')],
  ['Where it runs',D.where?lbl(typeof DEPLOY_WHERE!=='undefined'?DEPLOY_WHERE:[],D.where):'not decided'],
  ['Tenancy',D.tenancy?lbl(typeof DEPLOY_TENANCY!=='undefined'?DEPLOY_TENANCY:[],D.tenancy):'not answered'],
  ['Channels',(D.channels||[]).length?D.channels.map(x=>lbl(typeof DEPLOY_CHANNELS!=='undefined'?DEPLOY_CHANNELS:[],x)).join(', '):'not answered'],
  ['Regulations',regs.length?regs.join(', '):'none recorded']];
 const context=product?[['Problem',P.problem],['Users and their jobs',P.users],['Key journeys',P.journeys],['First release',P.mvp],['Success measures',P.success],['Integrations',P.integrations],['Build, buy or partner',P.buyPref]].filter(([,v])=>String(v||'').trim())
  :[['System',M.project],['Contexts',C.map(c=>c.name).join(', ')]];
 return {meta:{project:M.project||'Untitled',kind:product?'Product':'Modernization',date:new Date().toISOString().slice(0,10),draft:why.length>0,why,basis:'platform decision (or its suggestion) and each route (recorded, else best fit)'},
  context,nfr,blueprint:B,cost:X,decisions,assumptions:open,inputs:I};
}

/* ---- Diagram ---- */
function hldWrap(t,max,lines){const w=String(t||'').split(/\s+/);const out=[];let cur='';for(const x of w){if((cur+' '+x).trim().length>max){if(cur)out.push(cur);cur=x;}else cur=(cur+' '+x).trim();}if(cur)out.push(cur);
 if(out.length>lines){const k=out.slice(0,lines);k[lines-1]=k[lines-1].replace(/.{0,2}$/,'')+'…';return k;}return out;}
const hldEsc=s=>String(s).replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;');
function hldSplit(t){const s=String(t||'');const i=s.indexOf(': ');return i>0&&i<40?[s.slice(0,i),s.slice(i+2)]:[s,''];}

function hldDiagram(m){
 const B=m.blueprint,L=id=>(B.platform.find(l=>l.id===id)||{}),I=m.inputs;
 const W=1120,MAIN=820,GAP=16;
 const parts=[];let H=0;
 const box=(x,y,w,h,title,sub,cls)=>{const tl=hldWrap(title,Math.floor(w/7),2),sl=sub?hldWrap(sub,Math.floor(w/6.2),3):[];
  parts.push('<g class="hld-box '+(cls||'')+'"><rect x="'+x+'" y="'+y+'" width="'+w+'" height="'+h+'" rx="8"/>'+
   tl.map((t,i)=>'<text x="'+(x+w/2)+'" y="'+(y+18+i*15)+'" class="t">'+hldEsc(t)+'</text>').join('')+
   sl.map((t,i)=>'<text x="'+(x+w/2)+'" y="'+(y+20+tl.length*15+i*13)+'" class="s">'+hldEsc(t)+'</text>').join('')+'</g>');
  return {x,y,w,h,cx:x+w/2,cy:y+h/2,b:y+h,r:x+w};};
 const arrow=(x1,y1,x2,y2,dash)=>parts.push('<line x1="'+x1+'" y1="'+y1+'" x2="'+x2+'" y2="'+y2+'" class="hld-edge'+(dash?' dash':'')+'" marker-end="url(#hldArrow)"/>');
 const row=(y,h,items,cls)=>{if(!items.length)return [];const w=Math.min(260,(MAIN-GAP*(items.length-1))/items.length);const total=w*items.length+GAP*(items.length-1);let x=20+(MAIN-total)/2;
  return items.map(([t,s])=>{const b=box(x,y,w,h,t,s,cls);x+=w+GAP;return b;});};
 // 1. Users
 const actors=[];if(I.f.customers)actors.push(['Customers / viewers',((m.nfr.find(n=>n[0]==='Channels')||[])[1]||'').replace('not answered','web and mobile')]);
 if(I.f.staff)actors.push(['Staff','admin, content and support screens']);
 if(I.f.iot)actors.push(['Devices and sensors','MQTT over TLS to the device hub']);if(!actors.length)actors.push(['Users','']);
 const A=row(20,52,actors,'actor');
 // 2. Edge
 const ns=String(L('north').t||'').split(' → ').filter(Boolean).map(hldSplit);
 const E=row(112,60,ns.length?ns:[['Entry point','']],'edge');
 A.forEach(a=>arrow(a.cx,a.b,E[0].cx,E[0].y-2));
 for(let i=1;i<E.length;i++)arrow(E[i-1].r,E[i-1].cy,E[i].x-2,E[i].cy);
 // 3. Platform with modules
 const mods=B.capabilities.filter(c=>/Shared platform/.test(c.runsOn));
 const perRow=4,mw=(MAIN-40-GAP*(perRow-1))/perRow,mh=40;
 const rows=Math.max(1,Math.ceil(mods.length/perRow));
 const py=212,ph=72+rows*(mh+10);
 const [ht,hs]=hldSplit(L('hosting').t||'Application platform');
 parts.push('<g class="hld-platform"><rect x="20" y="'+py+'" width="'+MAIN+'" height="'+ph+'" rx="10"/><text x="36" y="'+(py+22)+'" class="pt">'+hldEsc('Shared platform · '+ht)+'</text><text x="36" y="'+(py+40)+'" class="s left">'+hldEsc(hldWrap(hs,110,1)[0]||'')+'</text></g>');
 mods.forEach((c,i)=>box(40+(i%perRow)*(mw+GAP),py+56+Math.floor(i/perRow)*(mh+10),mw,mh,c.name,'','module'));
 if(!mods.length)box(40,py+56,MAIN-40,mh,'No capability runs on the shared platform in this basis','','module');
 const lastE=E[E.length-1];arrow(lastE.cx,lastE.b,lastE.cx,py-2);
 // 4. Data
 const dataItems=[];
 for(const [id,name] of [['database','Database'],['cache','Cache'],['search','Search'],['messaging','Messaging'],['files','Files and media']]){
  const l=L(id);if(!l.t)continue;if(id==='cache'&&/^In-app/.test(l.t))continue;
  const first=l.t.split(' · ')[0];const [t,s]=hldSplit(first);dataItems.push([name,(s||t).replace(/^Object storage: /,'')]);}
 const dy=py+ph+44;const Dt=row(dy,70,dataItems,'data');
 Dt.forEach(d=>arrow(d.cx,py+ph,d.cx,d.y-2));
 H=Math.max(dy+70,0);
 // 5. Side column: identity, bought, partner, other systems
 let sy=112;const sx=MAIN+50,sw=W-sx-20;const side=[];
 const idl=L('identity').t;if(idl)side.push(['Identity',idl.replace(/ · MFA.*$/,''),'ext id']);
 const payl=L('payments').t;if(payl){const prov=(payl.match(/(?:in front of |Outside the platform: )([^;·]+?)(?: with| and|;|$)/)||[])[1]||'Payment provider';side.push(['Payment gateway',prov.replace(/ \(named in the brief\)/,'')+' · checkout, billing, signed webhooks in','ext pay']);}
 const vend=B.capabilities.filter(c=>/Vendor product/.test(c.runsOn));if(vend.length)side.push(['Bought products (SaaS)',vend.map(c=>c.name).join(' · '),'ext vendor']);
 const part=B.capabilities.filter(c=>/Partner/.test(c.runsOn));if(part.length)side.push(['Partner platform',part.map(c=>c.name).join(' · '),'ext partner']);
 const kinds=String((M.product||{}).integrations||'').split(/[;\n]/).map(x=>x.split(':')[0].trim()).filter(Boolean).slice(0,4);
 if(kinds.length)side.push(['Other systems',kinds.join(' · '),'ext sys']);
 for(const [t,s,cls] of side){const h=s.length>90?96:s.length>45?80:64;const b=box(sx,sy,sw,h,t,s,cls);
  arrow(MAIN+20,Math.min(Math.max(b.cy,py+10),py+ph-10),sx-2,b.cy,cls.includes('id'));sy+=h+14;}
 H=Math.max(H,sy)+20;
 const css='.hld-box rect{fill:#f7f9ff;stroke:#b8c6e3}.hld-box.actor rect{fill:#fff;stroke:#7c8fb8}.hld-box.edge rect{fill:#eef6ff;stroke:#8fb0dd}.hld-box.module rect{fill:#fff;stroke:#c9c1f2}.hld-box.data rect{fill:#f3faf6;stroke:#9fd3b5}.hld-box.ext rect{fill:#fffaf2;stroke:#e5c48a}.hld-box.ext.id rect{fill:#f6f4ff;stroke:#b9aef0}.hld-box.ext.pay rect{fill:#fff4f4;stroke:#e3a1a1}'+
  '.hld-platform rect{fill:#f6f4ff;stroke:#7a6ae0;stroke-width:1.5}.t{font:600 12px system-ui,sans-serif;text-anchor:middle;fill:#1d2433}.s{font:11px system-ui,sans-serif;text-anchor:middle;fill:#5b6478}.s.left{text-anchor:start}.pt{font:700 13px system-ui,sans-serif;fill:#3f33a8}'+
  '.hld-edge{stroke:#8091b3;stroke-width:1.4}.hld-edge.dash{stroke-dasharray:5 4}';
 return '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 '+W+' '+H+'" class="hld-svg" role="img" aria-label="Architecture diagram"><style>'+css+'</style><defs><marker id="hldArrow" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="7" markerHeight="7" orient="auto"><path d="M0,0 L10,5 L0,10 z" fill="#8091b3"/></marker></defs>'+parts.join('')+'</svg>';
}

/* ---- Full design as JSON: everything the project decided, assumed and derived, for tools and the other diagrams ---- */
function designJson(){
 const m=hldModel(),I=m.inputs,C=decCtxs(),P=M.product||{},product=isNewProduct();
 const pf=typeof platformEffective==='function'?platformEffective():null,R=typeof platformRec==='function'?platformRec():{};
 const layer=l=>({id:l.id,label:l.label,design:l.t,basis:l.why,source:l.src,quantities:l.q||null});
 const tiers={};for(const b of ['current','Safe','Balanced','Ambitious']){try{const x=tierCost(C,b);tiers[b==='current'?'platform':b]={tier:x.tier,monthly:Math.round(x.monthly),yearly:Math.round(x.yearly),running3Years:Math.round(x.run3),buildPersonMonths:x.pm,buildCost:x.oneOff,total3Years:x.total3,notEstimated:x.lines.filter(l=>l.monthly==null).map(l=>l.component)};}catch(e){}}
 const caps=C.map(c=>{
  const d=decisionFor(c),adv=approachAdvice(c),scen=advisorScenarios(c,adv);
  const bp=m.blueprint.capabilities.find(x=>x.id===c.id)||{};
  const route=d.approach&&d.approach!=='undecided'?{name:pf?pf.tier:'Balanced',tier:pf?pf.tier:'Balanced',a:{id:d.approach,label:apprLabel(d.approach)},d}:scen[0];
  let plan=[];try{plan=route?techPlan(c,route).map(layer):[];}catch(e){}
  return {id:c.id,name:c.name,description:c.description||'',release:c.releaseScope||'',
   route:{approach:route?route.a.id:'',label:route?route.a.label:'',basis:d.status==='decided'?'decided':'advisor best fit'},
   runsOn:bp.runsOn||'',usesFromPlatform:(bp.adds||[]).map(a=>a.label),
   decision:{status:d.status||'open',by:d.decidedBy?personLabel(d.decidedBy):'',at:d.at||'',rationale:d.rationale||''},
   advisor:{confidence:adv.confidence.level,options:adv.ranked.slice(0,4).map(a=>({approach:a.id,label:a.label,score:a.score,reasons:a.reasons.map(r=>(r.v>0?'+':r.v<0?'-':'')+' '+r.t)}))},
   plan};
 });
 return {
  schema:'mindsuite.design/1',generatedAt:new Date().toISOString(),
  project:{name:M.project||'',kind:product?'new product':'modernization',mode:typeof modeOf==='function'?modeOf():''},
  status:{draft:m.meta.draft,reasons:m.meta.why},
  brief:product?{problem:P.problem||'',users:P.users||'',journeys:P.journeys||'',firstRelease:P.mvp||'',success:P.success||'',integrations:P.integrations||'',buildBuyPartner:P.buyPref||'',deployment:P.deploy||{}}:{system:M.project||''},
  qualityTargets:{
   users:{value:I.users,source:I.uSrc},
   peakLoad:{requestsPerSecond:I.peak,source:I.pSrc,sizedFor3Years:I.sized,requestsPerVcpu:I.perVcpu,yearlyGrowthPct:I.growth,calculation:P.peakCalc||null},
   dataAfter3Years:{gb:I.dataGb,source:I.dSrc,calculation:P.dataCalc||null},
   availability:{percent:I.avail,source:I.aSrc,downtimeHoursPerYear:Math.round((100-I.avail)/100*8760*10)/10,recoverWithin:I.avail>=99.95?'1 hour':I.avail>=99.5?'4 hours':'1 day',maxDataLoss:I.avail>=99.95?'5 minutes':I.avail>=99.5?'15 minutes':'24 hours'},
   regulations:I.regs||[],features:I.f},
  platformDecision:pf?{decided:pf.decided,by:R.by?personLabel(R.by):'',at:R.at||'',rationale:R.rationale||'',
   values:{tier:pf.tier,cloud:pf.cloud,hostingModel:pf.hosting,architecture:pf.architecture,language:pf.language,database:pf.database,integration:pf.integration},
   suggestion:{values:pf.suggested.v,why:pf.suggested.why}}:null,
  sharedPlatform:{tier:m.blueprint.tier,layers:m.blueprint.platform.map(layer)},
  sharedConcerns:(m.blueprint.owners||[]).map(o=>({concern:o.label,owner:o.owner?{by:o.owner.by,capability:o.owner.name}:'platform',builtBy:o.built,clash:o.clash,split:o.split})),
  capabilities:caps,
  cost:{currency:m.cost?m.cost.currency:'',rates:typeof costRates==='function'?costRates():{},components:m.cost?m.cost.lines:[],perTier:tiers,excluded:['video delivery traffic','network egress','staff','support contracts']},
  assumptions:(typeof assumptionsRegister==='function'?assumptionsRegister():[]).map(a=>({id:a.id,area:a.area,text:a.text,value:a.value,source:a.source,drives:a.drives,state:a.state,confirmedBy:a.record?personLabel(a.record.by)||a.record.by:'',at:a.record?a.record.at:''})),
  people:(M.people||[]).map(p=>({name:p.name,role:p.role})),
  components:(()=>{const cm=hldComponents(m);return {nodes:cm.nodes,flows:cm.edges};})(),
  aiDiagram:M.hldAi?{model:M.hldAi.model,at:M.hldAi.at,summary:M.hldAi.summary,nodes:M.hldAi.nodes,flows:M.hldAi.edges,missing:M.hldAi.missing,conflicts:M.hldAi.conflicts,dropped:M.hldAi.dropped}:null,
  diagrams:{architectSvg:hldDiagram(m),componentSvg:hldComponentDiagram(m)}
 };
}

/* ---- Component diagram: every running component, its service, its size, and what flows between them ---- */
function hldComponents(m){
 const B=m.blueprint,I=m.inputs,L=id=>(B.platform.find(l=>l.id===id)||{});
 const pf=typeof platformEffective==='function'?platformEffective():{};
 const ck=(TP_CLOUD[pf.cloud||'']??TP_CLOUD[((M.product||{}).deploy||{}).where||'']??'');
 const sv=k=>tpSvc(k,ck);
 const N=[],E=[];const node=(id,zone,title,sub)=>N.push({id,zone,title,sub:sub||''});const edge=(from,to,label,dash)=>E.push({from,to,label:label||'',dash:!!dash});
 // Users and devices
 if(I.f.customers)node('customers','actors','Customers','web and mobile');
 if(I.f.staff)node('staff','actors','Staff','admin and support screens');
 if(!I.f.customers&&!I.f.staff)node('users','actors','Users','');
 if(I.f.iot)node('devices','actors','Devices and sensors','');
 // Entry
 const nq=L('north').q||{};const entry=[];
 if(nq.cdn){node('cdn','edge','CDN',sv('cdn'));entry.push('cdn');}
 if(nq.waf){node('waf','edge','WAF and front door',sv('waf'));entry.push('waf');}
 else if(nq.lb){node('lb','edge','Load balancer',sv('lb'));entry.push('lb');}
 if(nq.apigw){node('apigw','edge','API gateway',sv('apigw'));entry.push('apigw');}
 if(I.f.iot&&L('iot').t)node('iothub','edge','Device hub',sv('iotHub'));
 // Application
 const hq=L('hosting').q||{},cq=L('compute').q;
 const [kindName,service]=hldSplit(L('hosting').t||'Application');
 const size=cq?(cq.vcpuMax>cq.vcpuAvg?'≈ '+tpN(cq.vcpuAvg)+'–'+cq.vcpuMax+' vCPU, autoscaling':cq.vcpuMax+' vCPU, fixed')+', '+cq.ramPerVcpu+' GB RAM per vCPU':'';
 const srv=(L('servers').t||'').split(' · ');const srvOf=k=>(srv.find(x=>x.startsWith(k))||'').replace(/^[^:]*: /,'');
 node('app','app',kindName+(service?': '+service.split(',')[0]:''),[size,srvOf('app servers').startsWith('no servers')?'':srvOf('app servers'),B.capabilities.filter(c=>/Shared platform/.test(c.runsOn)).length+' capabilities as modules'].filter(Boolean).join(' · '));
 const mq=L('messaging').q;
 if(mq&&mq.kind!=='worker'||(mq&&mq.kind==='worker'))node('workers','app','Background workers',mq&&mq.kind==='worker'?'jobs table in the database':'consume from '+(mq.kind==='stream'?'event streaming':'the queue'));
 if(I.f.iot&&L('iot').t)node('ingest','app','Readings ingestion','validates and stores device readings');
 // Data
 const dq=L('database').q;
 if(dq){const [en]=String(L('database').t).split(' · ');node('db','data','Database',en.replace(/^[^:]*: /,'')+' · '+dq.instances+' × '+dq.vcpu+' vCPU (memory optimised) · '+dq.storageGb+' GB SSD');}
 else if(L('database').t)node('db','data','Database','inside the bought or partner product');
 const kq=L('cache').q;if(kq&&kq.nodes)node('cache','data','Cache',sv('redis')+' · '+kq.nodes+' × '+kq.gb+' GB');
 if(mq&&mq.kind==='queue')node('mq','data','Message queue',sv('queue'));
 if(mq&&mq.kind==='stream')node('mq','data','Event streaming',sv('stream'));
 if(L('search').q)node('search','data','Search',sv('search'));
 const fq=L('files').q;if(fq)node('object','data','Object storage',sv('blob')+' · '+tpN(fq.objectGb)+' GB');
 if(I.f.iot&&L('iot').t)node('tsdb','data','Time-series store',sv('tsdb'));
 // Cross-cutting
 node('idp','cross','Identity',[I.f.staff?'staff: '+sv('idStaff'):'',I.f.customers&&!(B.owners||[]).some(o=>o.id==='identity'&&o.owner)?'customers: '+sv('idCust'):''].filter(Boolean).join(' · ')||sv('idStaff'));
 node('vault','cross','Secrets and keys',sv('vault'));
 node('mon','cross','Monitoring and logs',sv('mon'));
 node('cicd','cross','Build and release',sv('cicd'));
 // External
 const payl=L('payments').t;if(payl)node('pay','ext','Payment gateway',/Handled by the/.test(payl)?'the bought product\'s checkout and billing':(payl.match(/(?:in front of |Outside the platform: )([^;·]+?)(?: with| and|;|$)/)||[])[1]||'payment provider');
 const vend=B.capabilities.filter(c=>/Vendor product/.test(c.runsOn)),part=B.capabilities.filter(c=>/Partner/.test(c.runsOn));
 if(vend.length)node('vendor','ext','Bought products (SaaS)',vend.map(c=>c.name).join(' · '));
 if(part.length)node('partner','ext','Partner platform',part.map(c=>c.name).join(' · '));
 if(L('ai').t)node('ai','ext','AI model service',sv('aiModels'));
 const kinds=String((M.product||{}).integrations||'').split(/[;\n]/).map(x=>x.split(':')[0].trim()).filter(Boolean).filter(k=>!/payment/i.test(k)).slice(0,3);
 if(kinds.length)node('other','ext','Other systems',kinds.join(' · '));
 // Flows
 const first=entry[0]||'app',last=entry[entry.length-1];
 for(const a of ['customers','staff','users'])if(N.find(n=>n.id===a))edge(a,first,'HTTPS');
 for(let i=1;i<entry.length;i++)edge(entry[i-1],entry[i],'HTTPS');
 if(last)edge(last,'app','HTTPS');
 if(N.find(n=>n.id==='iothub')){edge('devices','iothub','MQTT over TLS');edge('iothub',N.find(n=>n.id==='mq')?'mq':'ingest','readings');}
 if(N.find(n=>n.id==='ingest')){if(N.find(n=>n.id==='mq'))edge('mq','ingest','consume');edge('ingest','tsdb','write');}
 if(N.find(n=>n.id==='db')&&dq)edge('app','db','SQL over TLS');
 if(N.find(n=>n.id==='cache'))edge('app','cache','Redis over TLS');
 if(N.find(n=>n.id==='mq')){edge('app','mq',mq.kind==='stream'?'publish events':'send jobs');edge('mq','workers',mq.kind==='stream'?'subscribe':'consume');}
 else if(N.find(n=>n.id==='workers'))edge('app','workers','jobs');
 if(N.find(n=>n.id==='search'))edge('app','search','index and query');
 if(N.find(n=>n.id==='object'))edge('app','object','HTTPS (signed URLs)');
 edge('app','idp','OpenID Connect',true);edge('app','vault','secrets',true);edge('app','mon','OpenTelemetry',true);edge('cicd','app','deploy',true);
 if(N.find(n=>n.id==='pay')){edge('app','pay','API (tokens only)');edge('pay',nq.apigw?'apigw':last||'app','signed webhooks',true);}
 if(N.find(n=>n.id==='vendor'))edge('app','vendor','API');
 if(N.find(n=>n.id==='partner'))edge('app','partner','API, OIDC federation');
 if(N.find(n=>n.id==='ai'))edge('app','ai','HTTPS (prompts, no personal data)');
 if(N.find(n=>n.id==='other'))edge('app','other','APIs and files');
 return {nodes:N,edges:E,cloud:pf.cloud||''};
}

function hldComponentDiagram(m,cm){
 cm=cm||hldComponents(m);
 const MAIN=860,SX=900,SW=300,W=SX+SW+20;
 const pos={};const rowY={actors:20,edge:120,app:240,data:380};const rowH={actors:50,edge:64,app:70,data:76};
 for(const z of ['actors','edge','app','data']){const items=cm.nodes.filter(n=>n.zone===z);if(!items.length)continue;
  const w=Math.min(250,(MAIN-14*(items.length-1))/items.length),tot=w*items.length+14*(items.length-1);let x=20+(MAIN-tot)/2;
  for(const n of items){pos[n.id]={x,y:rowY[z],w,h:rowH[z]};x+=w+14;}}
 let sy=20;
 const side=z=>{const items=cm.nodes.filter(n=>n.zone===z);if(!items.length)return;sy+=z==='ext'?14:0;
  for(const n of items){const h=n.sub.length>70?78:n.sub.length>34?64:50;pos[n.id]={x:SX,y:sy+18,w:SW,h};sy+=h+10;}sy+=18;};
 const sideTop=sy;side('cross');const extTop=sy;side('ext');
 const H=Math.max(rowY.data+rowH.data+30,sy+20);
 const P=[];
 P.push('<rect x="12" y="'+(rowY.app-14)+'" width="'+(MAIN+16)+'" height="'+(rowH.app+28)+'" rx="10" class="zone app"/><text x="22" y="'+(rowY.app-2)+'" class="zt">Application zone</text>');
 P.push('<rect x="12" y="'+(rowY.data-14)+'" width="'+(MAIN+16)+'" height="'+(rowH.data+28)+'" rx="10" class="zone data"/><text x="22" y="'+(rowY.data-2)+'" class="zt">Data zone (private network)</text>');
 P.push('<text x="'+SX+'" y="'+(sideTop+12)+'" class="zt">Cross-cutting</text>');
 if(cm.nodes.some(n=>n.zone==='ext'))P.push('<text x="'+SX+'" y="'+(extTop+26)+'" class="zt">External services</text>');
 // Orthogonal routes: rows connect through channels between rows; the side column through one trunk;
 // cross-cutting services through one bracket (every application part uses them).
 const lines=[];const trunk=SX-26;
 const chan={actors:rowY.edge-18,edge:rowY.app-24,app:rowY.data-24,data:rowY.data+rowH.data+16};
 const ext=new Set(cm.nodes.filter(n=>n.zone==='ext').map(n=>n.id)),cross=new Set(cm.nodes.filter(n=>n.zone==='cross').map(n=>n.id));
 const lab=(x,y,t,cls)=>t?lines.push('<text x="'+x+'" y="'+y+'" class="cl'+(cls?' '+cls:'')+'">'+hldEsc(t)+'</text>'):0;
 const path=(pts,dash)=>lines.push('<path d="M'+pts.map(p=>p.join(' ')).join(' L')+'" class="ce'+(dash?' d':'')+'" fill="none" marker-end="url(#cmA)"/>');
 const zoneOf=id=>(cm.nodes.find(n=>n.id===id)||{}).zone;
 for(const e of cm.edges){
  if(cross.has(e.to)||cross.has(e.from))continue;
  const A=pos[e.from],Bp=pos[e.to];if(!A||!Bp)continue;
  const ax=A.x+A.w/2,bx=Bp.x+Bp.w/2;
  if(ext.has(e.to)){const ch=chan[zoneOf(e.from)]||chan.app,y2=Bp.y+Bp.h/2;path([[ax,A.y+A.h],[ax,ch],[trunk,ch],[trunk,y2],[Bp.x-2,y2]],e.dash);lab(Bp.x-6,y2-4,e.label,'r');continue;}
  if(ext.has(e.from)){const y1=A.y+A.h/2,ty=Bp.y+Bp.h/2;path([[A.x,y1],[trunk-8,y1],[trunk-8,ty],[Bp.x+Bp.w+2,ty]],e.dash);lab(trunk-12,ty-5,e.label,'r');continue;}
  if(Bp.y>A.y+A.h){const ch=Bp.y-12;if(Math.abs(ax-bx)<4)path([[ax,A.y+A.h],[bx,Bp.y-2]],e.dash);else path([[ax,A.y+A.h],[ax,ch],[bx,ch],[bx,Bp.y-2]],e.dash);lab(bx+4,Bp.y-15,e.label,'l');continue;}
  if(Bp.y+Bp.h<A.y){const ch=A.y-10;path([[ax,A.y],[ax,ch],[bx,ch],[bx,Bp.y+Bp.h+2]],e.dash);lab(bx+4,Bp.y+Bp.h+12,e.label,'l');continue;}
  const y=A.y+A.h/2;path(Bp.x>A.x?[[A.x+A.w,y],[Bp.x-2,y]]:[[A.x,y],[Bp.x+Bp.w+2,y]],e.dash);lab((A.x+A.w+Bp.x)/2,y-5,e.label);
 }
 // Bracket to the cross-cutting group.
 const cb=cm.nodes.filter(n=>n.zone==='cross').map(n=>pos[n.id]).filter(Boolean);
 if(cb.length){const top=cb[0].y,bot=cb[cb.length-1].y+cb[cb.length-1].h,mid=(top+bot)/2,ax=20+MAIN+8;
  lines.push('<path d="M'+(SX-12)+' '+top+' H'+(SX-6)+' V'+bot+' H'+(SX-12)+'" class="ce" fill="none"/>');
  lines.push('<path d="M'+ax+' '+(rowY.app+rowH.app/2)+' H'+(SX-30)+' V'+mid+' H'+(SX-8)+'" class="ce d" fill="none" marker-end="url(#cmA)"/>');
  lab(SX-34,mid-6,'every part: sign-in (OIDC), secrets, telemetry; released by CI/CD','r');}
 const boxes=cm.nodes.filter(n=>pos[n.id]).map(n=>{const p=pos[n.id],tl=hldWrap(n.title,Math.floor(p.w/7),2),sl=hldWrap(n.sub,Math.floor(p.w/6),3);
  return '<g class="cb '+n.zone+'"><rect x="'+p.x+'" y="'+p.y+'" width="'+p.w+'" height="'+p.h+'" rx="7"/>'+tl.map((t,i)=>'<text x="'+(p.x+p.w/2)+'" y="'+(p.y+16+i*14)+'" class="t">'+hldEsc(t)+'</text>').join('')+sl.map((t,i)=>'<text x="'+(p.x+p.w/2)+'" y="'+(p.y+18+tl.length*14+i*12)+'" class="s">'+hldEsc(t)+'</text>').join('')+'</g>';});
 const css='.zone{fill:none;stroke:#d5dcec;stroke-dasharray:4 4}.zone.app{fill:#faf9ff}.zone.data{fill:#f7fcf9}.zt{font:700 11px system-ui,sans-serif;fill:#6b7389;text-transform:uppercase}'+
  '.cb rect{fill:#f7f9ff;stroke:#b8c6e3}.cb.actors rect{fill:#fff;stroke:#7c8fb8}.cb.edge rect{fill:#eef6ff;stroke:#8fb0dd}.cb.app rect{fill:#f1eeff;stroke:#7a6ae0}.cb.data rect{fill:#eef9f3;stroke:#8fcbab}.cb.cross rect{fill:#f6f4ff;stroke:#b9aef0}.cb.ext rect{fill:#fffaf2;stroke:#e5c48a}'+
  '.t{font:600 11.5px system-ui,sans-serif;text-anchor:middle;fill:#1d2433}.s{font:10.5px system-ui,sans-serif;text-anchor:middle;fill:#5b6478}.ce{stroke:#8091b3;stroke-width:1.2}.ce.d{stroke-dasharray:4 4}.cl{font:9.5px system-ui,sans-serif;text-anchor:middle;fill:#4b5570;paint-order:stroke;stroke:#fff;stroke-width:3px}.cl.r{text-anchor:end}.cl.l{text-anchor:start}';
 return '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 '+W+' '+H+'" class="hld-svg" role="img" aria-label="Component diagram"><style>'+css+'</style><defs><marker id="cmA" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="6" markerHeight="6" orient="auto"><path d="M0,0 L10,5 L0,10 z" fill="#8091b3"/></marker></defs>'+lines.join('')+boxes.join('')+'</svg>';
}

/* ---- Diagram agent (Claude): reads the whole design and returns a reviewed component diagram ---- */
const HLD_AI_UI={busy:false,message:'',view:'ai'};
function diagramRequest(m,cm){
 const pf=typeof platformEffective==='function'?platformEffective():{};const lab=(d,id)=>id?((optOf(d,id)||[,id])[1]):'';
 const cut=(t,n)=>String(t||'').slice(0,n);
 return {project:cut(M.project,180),kind:isNewProduct()?'new':(typeof modeOf==='function'&&modeOf()==='extend'?'extend':'modernize'),
  platform:['Tier: '+pf.tier,'Cloud: '+lab('cloud',pf.cloud),'Hosting: '+(((typeof HOSTING_MODELS!=='undefined'?HOSTING_MODELS:[]).find(x=>x.id===pf.hosting)||{}).name||pf.hosting||''),'Architecture: '+lab('architecture',pf.architecture),'Database: '+lab('database',pf.database),'Integration: '+[].concat(pf.integration||[]).map(x=>lab('integration',x)).join(', '),'Decided: '+(pf.decided?'yes':'no, suggested')].filter(x=>!/: $/.test(x)),
  targets:m.nfr.map(([k,v])=>cut(k+': '+v,300)),
  layers:m.blueprint.platform.slice(0,40).map(l=>({id:cut(l.id,40),label:cut(l.label,80),design:cut(l.t,1500)})),
  capabilities:m.blueprint.capabilities.slice(0,60).map(c=>({name:cut(c.name,120),route:cut(c.route,80),runsOn:cut(c.runsOn,160)})),
  owners:(m.blueprint.owners||[]).slice(0,12).map(o=>cut(o.label+': '+(o.owner?(o.owner.by==='partner'?'partner platform':'bought product')+' for '+o.owner.name:'the shared platform'),300)),
  draftNodes:cm.nodes.slice(0,40).map(n=>({id:cut(n.id,40),zone:n.zone,title:cut(n.title,120),sub:cut(n.sub,300)})),
  draftEdges:cm.edges.slice(0,80).map(e=>({from:e.from,to:e.to,label:cut(e.label,80)}))};
}
function hldFingerprint(req){const t=JSON.stringify({...req,draftNodes:undefined,draftEdges:undefined});let h=0;for(let i=0;i<t.length;i++)h=(h*31+t.charCodeAt(i))|0;return String(h);}
async function runDiagramAgent(){
 if(HLD_AI_UI.busy)return;
 const m=hldModel(),cm=hldComponents(m),req=diagramRequest(m,cm);
 HLD_AI_UI.busy=true;HLD_AI_UI.message='The Diagram agent is reading the whole design…';render();
 const t0=Date.now();
 try{
  const out=await agentFetch('/api/design/diagram',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(req)});
  M.hldAi={...out,at:new Date().toISOString(),fp:hldFingerprint(req),ms:Date.now()-t0};
  logInput('hld:diagram-agent',null,out.nodes.length+' components, '+out.edges.length+' flows, '+(out.missing||[]).length+' missing, '+(out.conflicts||[]).length+' conflicts');
  HLD_AI_UI.view='ai';HLD_AI_UI.message='';persist();
 }catch(e){HLD_AI_UI.message=/404|not found/i.test(e.message)?'This backend needs the Diagram agent update. Restart the project backend with the latest build.':'The Diagram agent could not finish. '+e.message;}
 finally{HLD_AI_UI.busy=false;render();}
}
function hldAiBlock(m,cm){
 const ai=M.hldAi,stale=ai&&ai.fp!==hldFingerprint(diagramRequest(m,cm));
 const view=ai&&HLD_AI_UI.view==='ai'?'ai':'rule';
 const set=v=>{HLD_AI_UI.view=v;render();};
 const chip=(v,l)=>h('button',{type:'button',class:'type-chip'+(view===v?' on':''),onclick:()=>set(v)},[l]);
 const head=h('div',{class:'hld-ai-head'},[
  h('div',{},[h('b',{text:'Diagram agent'}),h('p',{class:'muted',text:'Asks Claude to read the whole design and draw the component diagram an architect would: it reviews the rule-based draft, adds what is missing and flags what does not fit. Sends, once per press: the platform decision, the plan layers, the capabilities with their routes, who owns each shared concern, the quality targets and the draft diagram. No business data, names of people or secrets.'})]),
  h('div',{class:'cost-actions'},[h('button',{type:'button',class:'btn small primary',disabled:HLD_AI_UI.busy?'':null,onclick:runDiagramAgent,text:HLD_AI_UI.busy?'Drawing…':ai?'Ask again':'Ask the Diagram agent'}),
   ai?h('span',{class:'load-profiles'},[chip('ai','AI-reviewed'),chip('rule','Rule-based')]):null])]);
 const status=HLD_AI_UI.message?h('p',{class:'input-status open',text:HLD_AI_UI.message}):null;
 if(!ai)return {head,status,svg:null,extra:null,view:'rule'};
 const am={nodes:ai.nodes,edges:ai.edges.map(e=>({...e,dash:e.kind==='control'||e.kind==='callback'})),cloud:cm.cloud};
 const extra=h('div',{class:'hld-ai-notes'},[
  h('small',{class:'muted',text:'Drawn by '+(ai.model||'Claude')+' on '+String(ai.at).slice(0,16).replace('T',' ')+' · '+ai.nodes.length+' components, '+ai.edges.length+' flows'+(ai.dropped?' · '+ai.dropped+' items dropped by the checks':'')+(ai.attempts>1?' · second attempt':'')}),
  stale?h('p',{class:'input-status open',text:'The design has changed since the agent drew this. Ask again to update it.'}):null,
  ai.summary?h('p',{text:ai.summary}):null,
  (ai.missing||[]).length?h('div',{},[h('b',{text:'Missing from the design'}),h('ul',{class:'plain-list'},ai.missing.map(x=>h('li',{},[h('b',{text:x.component+': '}),x.why])))]):null,
  (ai.conflicts||[]).length?h('div',{},[h('b',{text:'Conflicts'}),h('ul',{class:'plain-list warn-text'},ai.conflicts.map(x=>h('li',{text:x})))]):null]);
 return {head,status,svg:view==='ai'?hldComponentDiagram(m,am):null,extra,view,am};
}

/* ---- Markdown export ---- */
function hldMarkdown(m){
 const x=m.cost,cur=x?x.currency:'',fmt=n=>Math.round(n).toLocaleString()+' '+cur;
 const t=[];const p=s=>t.push(s);
 p('# High-level design: '+m.meta.project);p('');
 p((m.meta.draft?'**DRAFT** — ':'')+m.meta.kind+' · '+m.meta.date+' · basis: '+m.meta.basis);
 if(m.meta.draft)p('\nStill a draft because: '+m.meta.why.join('; ')+'.');
 p('\n## Context');for(const [k,v] of m.context)p('- **'+k+':** '+String(v).replace(/\n+/g,' '));
 p('\n## Quality targets');for(const [k,v] of m.nfr)p('- **'+k+':** '+v);
 p('\n## Shared platform ('+m.blueprint.tier+')');p('| Layer | Design | Based on |');p('|---|---|---|');
 for(const l of m.blueprint.platform)p('| '+l.label+' | '+l.t.replace(/\|/g,'/')+' | '+l.src+': '+l.why.replace(/\|/g,'/')+' |');
 {const cm=hldComponents(m);p('\n## Components');p('| Component | Zone | Service and size |');p('|---|---|---|');for(const n of cm.nodes)p('| '+n.title+' | '+n.zone+' | '+n.sub.replace(/\|/g,'/')+' |');
  p('\n### Flows');for(const e of cm.edges){const a=cm.nodes.find(x=>x.id===e.from),b=cm.nodes.find(x=>x.id===e.to);if(a&&b)p('- '+a.title+' → '+b.title+(e.label?': '+e.label:''));}}
 if(M.hldAi){p('\n## Diagram agent review ('+(M.hldAi.model||'Claude')+', '+String(M.hldAi.at).slice(0,10)+')');if(M.hldAi.summary)p(M.hldAi.summary);
  for(const x of M.hldAi.missing||[])p('- Missing: **'+x.component+'** — '+x.why);for(const x of M.hldAi.conflicts||[])p('- Conflict: '+x);}
 p('\n## Capabilities');p('| Capability | Route | Runs on | Uses from the platform |');p('|---|---|---|---|');
 for(const c of m.blueprint.capabilities)p('| '+c.name+' | '+c.route+(c.picked!=='decided'?' (best fit)':'')+' | '+c.runsOn+' | '+(c.adds.map(a=>a.label).join(', ')||'core only')+' |');
 if(x){p('\n## Cost (rough planning figures, not quotes)');p('| Component | Sizing | Per month |');p('|---|---|---|');
  for(const l of x.lines)p('| '+l.component+' | '+l.sizing+' | '+(l.monthly==null?'not estimated':fmt(l.monthly))+' |');
  p('| **Total per month** | | **'+fmt(x.monthly)+'** |');
  p('\nRunning for 3 years: '+fmt(x.run3)+(x.oneOff?' · build: '+fmt(x.oneOff[0])+' – '+fmt(x.oneOff[1]):' · build: '+x.pm[0]+'–'+x.pm[1]+' person-months')+'. Not included: video delivery traffic, network egress, staff, support contracts.');}
 p('\n## Decisions');for(const d of m.decisions)p('- '+d.name+': '+d.status+(d.approach?' — '+d.approach:'')+(d.by?' ('+d.by+(d.at?', '+d.at:'')+')':''));
 p('\n## Open assumptions and risks');if(!m.assumptions.length)p('None open.');for(const a of m.assumptions)p('- ['+a.area+'] '+a.text+' — drives: '+a.drives);
 return t.join('\n')+'\n';
}

/* ---- Page ---- */
function hldPage(){
 let m;try{m=hldModel();}catch(e){console.error('HLD',e);return h('div',{},[pageHeader('✓ / HIGH-LEVEL DESIGN','High-level design','The design could not be assembled yet: '+e.message)]);}
 const product=isNewProduct(),x=m.cost,cur=x?x.currency:'',fmt=n=>Math.round(n).toLocaleString()+' '+cur;
 const svg=hldDiagram(m);
 const file=(M.project||'project').replace(/[^\w-]+/g,'-').replace(/^-|-$/g,'').slice(0,60)||'project';
 const holder=h('div',{class:'hld-diagram'});holder.innerHTML=svg;
 const cm=hldComponents(m),aib=hldAiBlock(m,cm),csvg=aib.svg||hldComponentDiagram(m,cm);
 const cholder=h('div',{class:'hld-diagram'});cholder.innerHTML=csvg;
 const shown=aib.view==='ai'&&aib.am?aib.am:cm;
 const zoneName={actors:'Users and devices',edge:'Entry',app:'Application',data:'Data',cross:'Cross-cutting',ext:'External'};
 const sec=(title,body)=>h('section',{class:'summary-section hld-sec'},[h('h2',{text:title}),...[].concat(body)]);
 const dl=rows=>h('dl',{class:'inv-meta hld-dl'},rows.flatMap(([k,v])=>[h('dt',{text:k}),h('dd',{text:String(v||'')})]));
 return h('div',{class:'hld'},[
  pageHeader('✓ / HIGH-LEVEL DESIGN','High-level design: '+m.meta.project,'For architects: the shared platform, what runs on it, how traffic flows, sizing, quality targets and cost. Generated from this project; it changes when the project changes.',[
   button('Download Markdown',()=>downloadBlob('HLD-'+file+'.md',new Blob([hldMarkdown(m)],{type:'text/markdown'})),'btn'),
   button('Download diagram (SVG)',()=>downloadBlob('HLD-'+file+'-diagram.svg',new Blob([svg],{type:'image/svg+xml'})),'btn'),
   button('Download component diagram (SVG)',()=>downloadBlob('HLD-'+file+'-components.svg',new Blob([csvg],{type:'image/svg+xml'})),'btn'),
   button('Download full design (JSON)',()=>downloadBlob('design-'+file+'.json',new Blob([JSON.stringify(designJson(),null,2)],{type:'application/json'})),'btn'),
   button('Print / save as PDF',()=>window.print(),'btn primary')]),
  h('div',{class:'summary-callout '+(m.meta.draft?'stale':'approved')},[h('b',{text:m.meta.draft?'Draft':'Ready for review'}),
   h('p',{text:(m.meta.draft?'Still a draft because: '+m.meta.why.join('; ')+'. ':'All assumptions are confirmed or accepted and all decisions are recorded. ')+m.meta.kind+' · '+m.meta.date+' · basis: '+m.meta.basis+'.'}),
   m.meta.draft?h('div',{class:'cost-actions'},[button('Assumptions',()=>go('assumptions'),'btn small'),button(product?'Design options':'Target decisions',()=>go('decide'),'btn small'),!(M.people||[]).length?button('Rules & people',()=>go('rules'),'btn small'):null]):null]),
  sec('1. Context',dl(m.context)),
  sec('2. Quality targets',dl(m.nfr)),
  sec('3. Architecture diagram',[holder,h('small',{class:'muted',text:'Users → entry (north–south) → shared platform and the capabilities built on it → data. On the right: identity (dashed), bought products, partner platforms and other systems. Basis: '+m.meta.basis+'.'})]),
  sec('4. Component diagram',[aib.head,aib.status,cholder,aib.extra,h('small',{class:'muted',text:'Every running component with its service and size on '+(cm.cloud?((optOf('cloud',cm.cloud)||[,cm.cloud])[1]):'the chosen cloud')+'. Solid arrows carry requests and data; dashed arrows are sign-in, secrets, monitoring, releases and callbacks.'}),
   h('table',{class:'tp-table sb-table'},[h('thead',{},h('tr',{},['Component','Zone','Service and size','Talks to'].map(t=>h('th',{text:t})))),
    h('tbody',{},shown.nodes.map(n=>h('tr',{},[h('th',{scope:'row',text:n.title}),h('td',{text:zoneName[n.zone]||n.zone}),h('td',{text:n.sub+(n.why?' — '+n.why:'')}),
     h('td',{text:shown.edges.filter(e=>e.from===n.id).map(e=>((shown.nodes.find(x=>x.id===e.to)||{}).title||e.to)+(e.label?' ('+e.label+')':'')).join('; ')||'—'})])))])]),
  sec('5. Shared platform ('+m.blueprint.tier+')',h('table',{class:'tp-table sb-table'},h('tbody',{},m.blueprint.platform.map(l=>h('tr',{},[h('th',{scope:'row',text:l.label}),h('td',{},[h('div',{text:l.t}),h('small',{class:'tp-why',text:l.src+' · '+l.why})])]))))),
  sec('6. Capabilities',h('table',{class:'tp-table sb-table'},[h('thead',{},h('tr',{},['Capability','Route','Runs on','Uses from the platform'].map(t=>h('th',{text:t})))),
   h('tbody',{},m.blueprint.capabilities.map(c=>h('tr',{},[h('th',{scope:'row',text:c.name}),h('td',{text:c.route+(c.picked!=='decided'?' (best fit)':'')}),h('td',{text:c.runsOn}),h('td',{text:c.adds.map(a=>a.label).join(' · ')||'core only'})])))])),
  x?sec('7. Cost (rough planning figures, not quotes)',[h('table',{class:'tp-table cost-lines'},[h('thead',{},h('tr',{},['Component','Sizing','Per month'].map(t=>h('th',{text:t})))),
   h('tbody',{},[...x.lines.map(l=>h('tr',{},[h('th',{scope:'row',text:l.component}),h('td',{text:l.sizing}),h('td',{class:'num',text:l.monthly==null?'not estimated':fmt(l.monthly)})])),
    h('tr',{class:'total'},[h('th',{scope:'row',text:'Total per month'}),h('td',{text:''}),h('td',{class:'num'},[h('b',{text:fmt(x.monthly)})])])])]),
   h('p',{class:'muted',text:'Running for 3 years: '+fmt(x.run3)+(x.oneOff?' · build: '+fmt(x.oneOff[0])+' – '+fmt(x.oneOff[1]):' · build: '+x.pm[0]+'–'+x.pm[1]+' person-months')+'. Not included: video delivery traffic, network egress, staff, support contracts.'})]):null,
  sec('8. Decisions',h('ul',{class:'plain-list'},m.decisions.map(d=>h('li',{text:d.name+': '+d.status+(d.approach?' — '+d.approach:'')+(d.by?' ('+d.by+(d.at?', '+d.at:'')+')':'')})))),
  sec('9. Open assumptions and risks',m.assumptions.length?h('ul',{class:'plain-list'},m.assumptions.map(a=>h('li',{text:'['+a.area+'] '+a.text+' — drives: '+a.drives}))):h('p',{class:'muted',text:'None open.'}))]);
}

(function hldStyle(){
 if(document.getElementById('hld-style'))return;
 const css=`.hld-diagram{overflow-x:auto;border:1px solid #e3e8f2;border-radius:10px;background:#fff;padding:8px}.hld-svg{width:100%;min-width:900px;height:auto;display:block}
.hld-dl{font-size:13px}.hld-sec h2{font-size:16px}
.hld-ai-head{display:flex;flex-wrap:wrap;gap:8px 16px;justify-content:space-between;align-items:flex-start;margin-bottom:8px}.hld-ai-head p{margin:4px 0 0;max-width:760px;font-size:12.5px}.hld-ai-notes{margin:8px 0}.hld-ai-notes b{display:block;margin-top:6px}
@media print{.hld-diagram{border:0;padding:0}.hld-svg{min-width:0}.hld .summary-callout .cost-actions{display:none}}`;
 document.head.append(Object.assign(document.createElement('style'),{id:'hld-style',textContent:css}));
})();
