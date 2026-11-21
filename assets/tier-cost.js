/* MindSuite web app · cost per tier and component costs (System blueprint)
   Prices the shared platform of the System blueprint for each basis (Current picks, All Safe, All Balanced,
   All Ambitious) from the quantities techPlan() puts on its layers (layer.q): average and peak vCPU, database
   instances, vCPU and storage, cache GB and nodes, load balancer, WAF, API gateway, CDN, queue or event
   streaming, search, object storage. Monitoring is a share of the infrastructure. Vendor and partner licences
   use a price per user per month that the owner enters (not estimated without it). One-off build effort comes
   from the existing effort sizes (PERSON_MONTHS by autoSize) × the cost of one person-month.
   Rates are rough planning figures in the business-case currency, not quotes; every rate is editable
   (M.decide.businessCase.rates) and the register lists them as an assumption. Private or on-premises
   capacity is priced at 75% of the public rate as the equivalent cost of your own capacity.
   Not included: video transcoding and delivery traffic, network egress, staff, support contracts.
   tierCost(C,basis) returns {lines,monthly,yearly,run3,pm,oneOff,total3} for the diagrams;
   tierCostSection(C) draws the summary across tiers and the component costs for the chosen basis. */

const COST_RATES=[
 ['vm','Virtual machine, per vCPU (4 GB RAM)',30],['paas','App platform, per vCPU',45],['ctr','Managed containers, per vCPU',40],
 ['k8sNode','Kubernetes node, per vCPU',32],['k8sCluster','Kubernetes cluster fee',75],['fn','Serverless, per vCPU-equivalent',35],
 ['db','Managed database, per vCPU per server',110],['dbSelf','Self-run database, per vCPU per server',40],
 ['dbStorage','Database storage, per GB per server',0.13],['backup','Backup storage, per GB',0.05],
 ['redis','Cache, per GB per node',20],['lb','Load balancer',25],['waf','Web application firewall / front door',350],
 ['apigw','API gateway',150],['cdn','CDN base fee',50],['queue','Message queue',20],['stream','Event streaming (small Kafka)',450],
 ['search1','Search service (small)',250],['search2','Search service (larger, with replicas)',500],['object','Object storage, per GB',0.023],
 ['monitorPct','Monitoring and logs, % of infrastructure',10],['privatePct','Own capacity, % of public price',75],
 ['saasUser','Bought product, per user per month (each capability)',''],['partnerUser','Partner platform, per user per month','']];
function costRates(){
 ensureDecide();const B=M.decide.businessCase||(M.decide.businessCase={currency:'USD'});
 const R=B.rates||{};const out={};for(const [k,,d] of COST_RATES){const v=num(R[k]);out[k]=v!=null?v:d===''?null:d;}
 out._edited=Object.keys(R).filter(k=>String(R[k]).trim()!=='').length;return out;
}

function tierCost(C,basis){
 const B=systemBlueprint(C,basis),R=costRates(),bc=M.decide.businessCase||{};
 const q=id=>((B.platform.find(l=>l.id===id)||{}).q)||null;
 const host=q('hosting')||{kind:'ctr',pub:true},priv=host.pub?1:(R.privatePct||75)/100;
 const lines=[];const add=(component,sizing,rate,monthly,note)=>lines.push({component,sizing,rate,monthly:monthly==null?null:Math.round(monthly*100)/100,note:note||''});
 const money=n=>(Math.round(n*100)/100).toLocaleString();
 const c=q('compute');
 if(c){const kindRate={vm:R.vm,paas:R.paas,ctr:R.ctr,k8s:R.k8sNode,fn:R.fn,saas:R.ctr,platform:R.ctr}[host.kind]||R.ctr;
  const label={vm:'Virtual machines',paas:'App platform',ctr:'Managed containers',k8s:'Kubernetes nodes',fn:'Serverless functions',saas:'Connectors (containers)',platform:'Custom parts (containers)'}[host.kind]||'App compute';
  add(label,'≈ '+tpN(c.vcpuAvg)+' vCPU on average'+(c.vcpuMax>c.vcpuAvg?' (up to '+c.vcpuMax+')':'')+', '+c.ramPerVcpu+' GB RAM per vCPU',money(kindRate*priv)+' per vCPU',c.vcpuAvg*kindRate*priv);
  if(host.kind==='k8s')add('Kubernetes cluster','1 cluster (control plane)',money(R.k8sCluster),R.k8sCluster*(host.pub?1:0),host.pub?'':'included in your own platform');}
 const d=q('database');
 if(d){const rate=d.managed?R.db:R.dbSelf;
  add('Database servers',d.instances+' server'+(d.instances>1?'s':'')+' × '+d.vcpu+' vCPU',money(rate*priv)+' per vCPU',d.instances*d.vcpu*rate*priv);
  add('Database storage',d.instances+' × '+d.storageGb+' GB',money(R.dbStorage)+' per GB',d.instances*d.storageGb*R.dbStorage*priv);
  add('Backups (35 days)','≈ '+Math.round(d.backupGb*1.5)+' GB',money(R.backup)+' per GB',d.backupGb*1.5*R.backup*priv);}
 const k=q('cache');
 if(k&&k.nodes)add('Cache',k.nodes+' node'+(k.nodes>1?'s':'')+' × '+k.gb+' GB',money(R.redis)+' per GB',k.nodes*k.gb*R.redis*priv);
 const n=q('north')||{};
 if(n.lb)add('Load balancer','1',money(R.lb),R.lb*priv);
 if(n.waf)add('Web application firewall','1',money(R.waf),R.waf*priv);
 if(n.apigw)add('API gateway','1',money(R.apigw),R.apigw*priv);
 if(n.cdn)add('CDN','base fee',money(R.cdn),R.cdn,'traffic billed per GB delivered, not included');
 const m=q('messaging');
 if(m&&m.kind==='queue')add('Message queue','1',money(R.queue),R.queue*priv);
 if(m&&m.kind==='stream')add('Event streaming','small cluster',money(R.stream),R.stream*priv);
 const s=q('search');
 if(s&&s.tier>=1)add('Search service',s.tier===1?'small':'with replicas',money(s.tier===1?R.search1:R.search2),(s.tier===1?R.search1:R.search2)*priv);
 const f=q('files');
 if(f)add('Object storage',tpN(f.objectGb)+' GB',money(R.object)+' per GB',f.objectGb*R.object*priv,f.video?'video transcoding and delivery are billed per minute and per GB viewed, not included':'');
 const ai=q('ai');
 if(ai)add('AI model usage',ai.count+' suggested AI feature'+(ai.count>1?'s':''),'per use (tokens)',null,'set a monthly budget; billed per request by the model service');
 const infra=lines.reduce((a,l)=>a+(l.monthly||0),0);
 add('Monitoring and logs',(R.monitorPct||0)+'% of infrastructure','',infra*(R.monitorPct||0)/100);
 const users=num((M.product||{}).users_count)||0;
 const vendorCaps=B.capabilities.filter(x=>/Vendor product/.test(x.runsOn)).length,partnerCaps=B.capabilities.filter(x=>/Partner/.test(x.runsOn)).length;
 if(vendorCaps)add('Bought products (licences)',vendorCaps+' capabilit'+(vendorCaps>1?'ies':'y')+(users?' × '+tpN(users)+' users':''),R.saasUser!=null?money(R.saasUser)+' per user':'enter a price',R.saasUser!=null&&users?vendorCaps*users*R.saasUser:null,R.saasUser==null?'ask the vendors for a price per user':'');
 if(partnerCaps)add('Partner platform (licences)',(users?tpN(users)+' users, ':'')+'one platform for '+partnerCaps+' capabilit'+(partnerCaps>1?'ies':'y'),R.partnerUser!=null?money(R.partnerUser)+' per user':'enter a price',R.partnerUser!=null&&users?users*R.partnerUser:null,R.partnerUser==null?'ask the partner for a price per user':'');
 const monthly=lines.reduce((a,l)=>a+(l.monthly||0),0),missing=lines.filter(l=>l.monthly==null).length;
 // One-off: effort size per capability for this basis.
 const pm=[0,0];
 for(const cap of C){const sc=sbScenarioFor(cap,basis);if(!sc)continue;const size=autoSize(cap,sc.d);const r=PERSON_MONTHS[size];if(r){pm[0]+=r[0];pm[1]+=r[1];}}
 const rate=num(bc.personMonth);
 const oneOff=rate?[pm[0]*rate,pm[1]*rate]:null;
 return {basis,tier:B.tier,lines,monthly,yearly:monthly*12,run3:monthly*36,missing,pm,oneOff,total3:oneOff?[oneOff[0]+monthly*36,oneOff[1]+monthly*36]:null,currency:bc.currency||'USD',priv:!host.pub};
}

function tierCostSection(C){
 if(!C||!C.length)return null;
 const bases=[['current','Platform tier'],['Safe','What if Safe'],['Balanced','What if Balanced'],['Ambitious','What if Ambitious']];
 let T;try{T=bases.map(([b,l])=>({b,l,x:tierCost(C,b)}));}catch(e){console.error('Tier cost',e);return null;}
 const cur=T[0].x.currency,fmt=n=>Math.round(n).toLocaleString()+' '+cur,rng=r=>r?fmt(r[0])+' – '+fmt(r[1]):'—';
 const sel=T.find(t=>t.b===SB_UI.basis)||T[0];
 const B=M.decide.businessCase;
 const setRate=k=>v=>{B.rates=B.rates||{};const b=B.rates[k];B.rates[k]=v;logInput('cost:rate:'+k,b,v);persist();render();};
 const row=(label,fn,strong)=>h('tr',{},[h('th',{scope:'row',text:label}),...T.map(t=>h('td',{class:t.b===sel.b?'on':''},[strong?h('b',{text:fn(t.x)}):fn(t.x)]))]);
 return h('div',{class:'tier-cost'},[
  h('h3',{text:'Cost per tier'}),
  h('p',{class:'muted',text:'Rough planning figures from the sizing above, in '+cur+'. Not quotes: change any rate below. Video delivery traffic, network egress, staff and support contracts are not included.'}),
  h('div',{class:'tp-scroll'},h('table',{class:'tp-table cost-sum'},[
   h('thead',{},h('tr',{},[h('th',{text:''}),...T.map(t=>h('th',{class:t.b===sel.b?'on':''},[h('b',{text:t.l}),h('small',{text:t.x.tier+' platform'})]))])),
   h('tbody',{},[
    row('Running, per month',x=>fmt(x.monthly)+(x.missing?' + items not estimated':''),true),
    row('Running, 3 years',x=>fmt(x.run3)+(x.missing?' + items not estimated':'')),
    row('Build, one-off',x=>x.oneOff?rng(x.oneOff):(x.pm[1]?x.pm[0]+'–'+x.pm[1]+' person-months':'—')),
    row('Total, 3 years',x=>x.total3?rng(x.total3):'add the cost of a person-month',true)])])),
  h('h3',{text:'Component costs · '+sel.l}),
  h('div',{class:'tp-scroll'},h('table',{class:'tp-table cost-lines'},[
   h('thead',{},h('tr',{},['Component','Sizing','Rate','Per month'].map(t=>h('th',{text:t})))),
   h('tbody',{},[...sel.x.lines.map(l=>h('tr',{},[h('th',{scope:'row',text:l.component}),h('td',{},[h('div',{text:l.sizing}),l.note?h('small',{class:'tp-why',text:l.note}):null]),h('td',{text:l.rate}),h('td',{class:'num'},[l.monthly==null?h('small',{class:'muted',text:'not estimated'}):fmt(l.monthly)])])),
    h('tr',{class:'total'},[h('th',{scope:'row',text:'Total per month'}),h('td',{text:sel.x.priv?'own capacity priced at '+costRates().privatePct+'% of public prices':''}),h('td',{text:''}),h('td',{class:'num'},[h('b',{text:fmt(sel.x.monthly)})])])])])),
  h('div',{class:'cost-actions'},[button('Use '+fmt(sel.x.yearly)+' a year as the expected running cost',()=>{const b=B.targetRun;B.targetRun=String(Math.round(sel.x.yearly));logInput('cost:targetRun',b,B.targetRun);persist();render();notice('Expected yearly running cost set from the '+sel.l+' estimate.');},'btn small'),
   num(B.personMonth)?null:h('small',{class:'muted',text:'Add the cost of one person-month in the 3-year cost box above to price the build.'})]),
  h('details',{class:'cost-rates'},[h('summary',{text:'Rates used ('+(costRates()._edited?costRates()._edited+' changed by you':'planning defaults')+')'}),
   h('div',{class:'rules-grid'},COST_RATES.map(([k,label,d])=>inputControl(label,(B.rates||{})[k]??'',setRate(k),'number',d===''?'enter a price':String(d)))),
   h('small',{class:'muted',text:'Leave a box empty to use the default shown in it.'})])]);
}

/* Items for the assumptions register (assets/assumptions.js). */
function costAssumptions(add){
 let C=[];try{C=decCtxs();}catch(e){}
 if(!C.length)return;
 const R=costRates(),B=(M.decide&&M.decide.businessCase)||{};
 add('cost:rates','Cost','Cloud and platform prices are rough planning rates, not quotes'+(R._edited?' ('+R._edited+' changed by you)':''),JSON.stringify(B.rates||{}),R._edited?'you set some':'planning defaults','Running cost per tier',['decide','System blueprint → Rates used'],true);
 if(!num(B.personMonth))add('cost:personMonth','Cost','No cost of a person-month; the build cost is shown in person-months only','not given','missing','Build cost, 3-year total',['decide','3-year cost'],true);
 let x=null;try{x=tierCost(C,'current');}catch(e){}
 if(x&&x.lines.some(l=>/licences/.test(l.component)&&l.monthly==null))add('cost:licences','Cost','No licence price for bought or partner products; running cost leaves them out','not given','missing','Running cost',['decide','System blueprint → Rates used'],true);
 if(x&&x.lines.some(l=>l.component==='AI model usage'))add('cost:ai','Cost','AI model usage is not in the running cost; it depends on how much the AI features are used','not estimated','missing','Running cost',['decide','System blueprint'],true);
 if(x&&x.lines.some(l=>/video/.test(l.note)))add('cost:video','Cost','Video transcoding and delivery traffic are not in the running cost','not estimated','missing','Running cost (often the largest part for video)',['decide','System blueprint'],true);
 add('cost:excluded','Cost','Staff, support contracts and network egress are not in the running cost','excluded','rule','3-year total',null);
}

(function tierCostStyle(){
 if(document.getElementById('tier-cost-style'))return;
 const css=`.tier-cost{margin-top:14px;border-top:1px solid #eef1f7;padding-top:10px}.tier-cost h3{margin:10px 0 6px;font-size:14px}
.cost-sum{min-width:640px}.cost-sum thead small{display:block;color:var(--muted);font-weight:400}.cost-sum .on{background:#f6f4ff}
.cost-lines{min-width:640px}.cost-lines td.num{text-align:right;white-space:nowrap;width:140px}.cost-lines tr.total th,.cost-lines tr.total td{border-top:2px solid #d9e1f2}
.cost-actions{display:flex;flex-wrap:wrap;gap:8px;align-items:center;margin:8px 0}.cost-rates{margin-top:6px}.cost-rates summary{cursor:pointer;font-size:13px}`;
 document.head.append(Object.assign(document.createElement('style'),{id:'tier-cost-style',textContent:css}));
})();
