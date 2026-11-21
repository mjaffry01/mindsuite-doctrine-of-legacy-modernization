/* MindSuite web app · technical plan by layer (Design options / Target decisions)
   For each capability, turns the three scenarios (Safe, Balanced, Ambitious) into a layer-by-layer plan:
   route, hosting model, traffic in (north–south), compute sizing, traffic between parts (east–west),
   database, cache, messaging, files and media, search, integrations, identity, security and compliance,
   availability and recovery, operations, design patterns, often-missed components.
   Rule-based, no model calls. Every cell says what it is based on: the brief, a calculation, your choice,
   or a rule. Inputs: the product brief numbers (or the capacity profile in a modernization project), the
   predicted assumptions of the load and data calculators, the deployment answers and hosting model, the
   scenario's approach and technology choices, the brief's wording and the chosen regulations.
   techPlan(c,s) returns structured layers ({id,label,t,why,src}) so the audience diagrams can reuse them;
   techPlanTable(c,scen) draws the side-by-side table. Hook: advisorPanel in assets/decision-advisor.js. */

const TP_CLOUD={azure:'azure',aws:'aws',gcp:'gcp',private:'onprem',onprem:'onprem',oci:'oci',ibm:'ibm',public:'',multicloud:'',hybrid:'onprem',edge:'onprem'};
const TP_CLOUD_NAME={azure:'Azure',aws:'AWS',gcp:'Google Cloud',onprem:'your data centre',oci:'Oracle Cloud',ibm:'IBM Cloud','':'the chosen cloud'};
/* Service names per provider: [azure, aws, gcp, on-premises / private, ibm, oracle (oci)] */
const TP_SVC={
 waf:['Azure Front Door with WAF (or Application Gateway WAF)','Amazon CloudFront + AWS WAF + Application Load Balancer','Cloud Load Balancing + Cloud Armor','F5, NGINX or HAProxy with a WAF (ModSecurity)','IBM Cloud Internet Services (WAF, DDoS) + VPC load balancer','OCI Web Application Firewall + OCI Load Balancer'],
 lb:['Azure Application Gateway','Application Load Balancer','Cloud Load Balancing','HAProxy or NGINX','IBM Cloud Application Load Balancer for VPC','OCI Load Balancer'],
 apigw:['Azure API Management','Amazon API Gateway','Apigee or API Gateway','Kong, Gravitee or NGINX','IBM API Connect','OCI API Gateway'],
 cdn:['Azure Front Door','Amazon CloudFront','Cloud CDN / Media CDN','Akamai, Cloudflare or Varnish','IBM Cloud Internet Services (caching)','a partner CDN (OCI has no native CDN)'],
 redis:['Azure Managed Redis','Amazon ElastiCache (Valkey or Redis)','Memorystore','Redis or Valkey','IBM Cloud Databases for Redis','OCI Cache (Redis)'],
 queue:['Azure Service Bus','Amazon SQS','Pub/Sub','RabbitMQ','IBM MQ as a Service','OCI Queue'],
 stream:['Azure Event Hubs (Kafka API)','Amazon MSK (Kafka)','Pub/Sub or Managed Kafka','Apache Kafka (Strimzi on OpenShift)','IBM Event Streams (Kafka)','OCI Streaming (Kafka-compatible)'],
 blob:['Azure Blob Storage','Amazon S3','Cloud Storage','MinIO or Ceph object storage','IBM Cloud Object Storage','OCI Object Storage'],
 search:['Azure AI Search','Amazon OpenSearch Service','Vertex AI Search or Elastic Cloud','OpenSearch or Elasticsearch','IBM Cloud Databases for OpenSearch','OCI Search with OpenSearch'],
 video:['FFmpeg workers in containers or a video platform from Azure Marketplace (Azure Media Services was retired in 2024)','AWS Elemental MediaConvert','Transcoder API','FFmpeg workers','FFmpeg workers on IBM Cloud Code Engine','OCI Media Flow'],
 idStaff:['Microsoft Entra ID','AWS IAM Identity Center (with your staff directory)','Cloud Identity','Keycloak with Active Directory / LDAP','IBM Verify (single sign-on with your directory)','OCI IAM identity domains'],
 idCust:['Microsoft Entra External ID','Amazon Cognito','Identity Platform','Keycloak','IBM Verify (customer identity)','OCI IAM identity domains (external users)'],
 vault:['Azure Key Vault','AWS Secrets Manager + KMS','Secret Manager + Cloud KMS','HashiCorp Vault','IBM Cloud Secrets Manager + Key Protect','OCI Vault'],
 mon:['Azure Monitor + Application Insights','Amazon CloudWatch + AWS X-Ray','Cloud Monitoring + Cloud Logging + Cloud Trace','Prometheus + Grafana + Loki (OpenTelemetry)','IBM Cloud Monitoring + IBM Cloud Logs','OCI Monitoring + Logging + Application Performance Monitoring'],
 cicd:['GitHub Actions or Azure DevOps, Bicep or Terraform','GitHub Actions or CodePipeline, CloudFormation or Terraform','GitHub Actions or Cloud Build, Terraform','GitLab CI or Jenkins, Terraform or Ansible','IBM Cloud Continuous Delivery (Tekton), Terraform with IBM Cloud Schematics','OCI DevOps, Terraform with OCI Resource Manager'],
 notify:['Azure Communication Services','Amazon SES + SNS','a mail and SMS provider (e.g. SendGrid, Twilio)','your mail relay and an SMS gateway','IBM Cloud Event Notifications','OCI Email Delivery + Notifications'],
 paas:['Azure App Service','AWS Elastic Beanstalk or App Runner','App Engine','OpenShift (containers)','IBM Cloud Code Engine','OCI Container Instances'],
 ctr:['Azure Container Apps','Amazon ECS on Fargate','Cloud Run','OpenShift or Rancher','IBM Cloud Code Engine','OCI Container Instances'],
 k8s:['AKS','Amazon EKS','GKE','Red Hat OpenShift','Red Hat OpenShift on IBM Cloud or IBM Cloud Kubernetes Service','OCI Kubernetes Engine (OKE)'],
 fn:['Azure Functions','AWS Lambda','Cloud Run functions','Knative or OpenFaaS','IBM Cloud Code Engine functions','OCI Functions'],
 iotHub:['Azure IoT Hub (+ Device Provisioning Service)','AWS IoT Core','an MQTT broker such as EMQX or HiveMQ on GKE, feeding Pub/Sub (Google Cloud IoT Core was retired in 2023)','an MQTT broker (EMQX, HiveMQ or Mosquitto)','an MQTT broker (EMQX or HiveMQ) on OpenShift, feeding IBM Event Streams','an MQTT broker (EMQX or HiveMQ) on OKE, feeding OCI Streaming'],
 iotEdge:['Azure IoT Edge','AWS IoT Greengrass','containers on the device (k3s or Docker)','containers on the device (k3s or Docker)','containers on the device (k3s or Docker), IBM Cloud Satellite on site','containers on the device (k3s or Docker)'],
 tsdb:['Azure Data Explorer','Amazon Timestream','Bigtable, or BigQuery for analysis','TimescaleDB or InfluxDB','PostgreSQL with TimescaleDB, or InfluxDB, on OpenShift','PostgreSQL with TimescaleDB, or InfluxDB, on OKE'],
 aiModels:['Azure AI Foundry (Azure OpenAI and other model providers)','Amazon Bedrock','Vertex AI (Gemini and partner models)','open-weight models served on your own GPUs (vLLM)','IBM watsonx.ai','OCI Generative AI'],
 vector:['Azure AI Search (vector search)','Amazon OpenSearch Service (vector search) or Aurora PostgreSQL with pgvector','Vertex AI Vector Search','PostgreSQL with pgvector, or OpenSearch','IBM Cloud Databases for OpenSearch, or watsonx.data','Oracle Database 23ai vector search, or OCI Search with OpenSearch'],
 aiSafety:['Azure AI Content Safety','Amazon Bedrock Guardrails','Vertex AI safety filters and Model Armor','an open-source guard model and PII filter','watsonx.governance','OCI Generative AI guardrails'],
 vm:['Azure Virtual Machines','Amazon EC2','Compute Engine','VMware or Hyper-V virtual machines','IBM Cloud Virtual Servers for VPC','OCI Compute']};
const TP_DB={postgres:['Azure Database for PostgreSQL (Flexible Server)','Amazon RDS or Aurora PostgreSQL','Cloud SQL or AlloyDB for PostgreSQL','PostgreSQL with Patroni','IBM Cloud Databases for PostgreSQL','OCI Database with PostgreSQL'],
 mysql:['Azure Database for MySQL (Flexible Server)','Amazon RDS or Aurora MySQL','Cloud SQL for MySQL','MySQL with InnoDB Cluster','IBM Cloud Databases for MySQL','HeatWave MySQL'],
 sqlserver:['Azure SQL Database or SQL Managed Instance','Amazon RDS for SQL Server','Cloud SQL for SQL Server','SQL Server Always On','SQL Server on IBM Cloud virtual servers','SQL Server on OCI Compute'],
 oracle:['Oracle Database@Azure','Amazon RDS for Oracle','Oracle Database@Google Cloud','Oracle with Data Guard','Oracle on IBM Power Virtual Server','Autonomous Database or Base Database Service'],
 nosql:['Azure Cosmos DB','Amazon DynamoDB or DocumentDB','Firestore','MongoDB','IBM Cloudant or IBM Cloud Databases for MongoDB','OCI NoSQL Database']};
const TP_DB_NAME={postgres:'PostgreSQL',mysql:'MySQL',sqlserver:'SQL Server',oracle:'Oracle',nosql:'Document / NoSQL database',keep:'the current database'};

const TP_SERVER={
 general:['Dv5 general purpose (Standard_D{n}s_v5)','m7i general purpose (m7i, {n} vCPU), or m7g on Graviton','n2-standard-{n}','general-purpose x86 servers, {n} vCPU per VM','bx2-{n}x{m} (balanced)','VM.Standard.E5.Flex ({o} OCPU)'],
 memory:['Ev5 memory optimised (Standard_E{n}s_v5)','r7i memory optimised ({n} vCPU)','n2-highmem-{n}','memory-heavy servers, {n} vCPU and 8 GB per vCPU','mx2-{n}x{m2} (memory)','VM.Standard.E5.Flex ({o} OCPU, extra memory)'],
 compute:['Fsv2 compute optimised (Standard_F{n}s_v2)','c7i compute optimised ({n} vCPU)','c3-highcpu-{n}','compute-heavy servers, {n} vCPU','cx2-{n}x{m1} (compute)','VM.Standard.E5.Flex ({o} OCPU, less memory)'],
 gpu:['NCasT4_v3 or NVadsA10 v5 (NVIDIA T4 / A10)','g5 or g6 (NVIDIA A10G / L4)','g2-standard (NVIDIA L4)','servers with NVIDIA L4 or A10 GPUs','gx3 (NVIDIA L4)','VM.GPU.A10']};
function tpServer(role,cloud,vcpu){
 const i={azure:0,aws:1,gcp:2,ibm:4,oci:5}[cloud];const row=TP_SERVER[role];const t=(i==null?row[3]:row[i])||row[3];
 const n=[2,4,8,16,32].find(x=>x>=(vcpu||2))||32;
 return t.replace(/\{n\}/g,n).replace(/\{m\}/g,n*4).replace(/\{m2\}/g,n*8).replace(/\{m1\}/g,n*2).replace(/\{o\}/g,Math.max(1,n/2));
}
function tpSvc(key,cloud){const i={azure:0,aws:1,gcp:2,ibm:4,oci:5}[cloud];const row=TP_SVC[key]||TP_DB[key];if(!row)return '';return i==null||!row[i]?row[3]:row[i];}
/* Role names from the brief's "Users and their jobs" (the words before each colon). */
function tpRoles(t){
 const out=[];for(const part of String(t||'').split(/[;\n]|\(\d+\)/)){const m=part.match(/^\s*([A-Za-z][A-Za-z /-]{2,32}?)s?\s*:/);if(m){const r=m[1].trim().replace(/^./,x=>x.toUpperCase());if(!out.includes(r))out.push(r);}}
 return out.slice(0,6);
}
const I_D_channels=D=>(D&&D.channels)||[];
function tpText(c){
 const P=M.product||{};
 return [...['problem','users','journeys','mvp','success','integrations','buyPref'].map(k=>String(P[k]||'')),c&&c.name||'',c&&c.description||''].join(' ').toLowerCase();
}
/* Numbers for sizing, with where each came from. */
function techPlanInputs(c){
 const P=M.product||{},D=P.deploy||{};
 const caps=(M.inventory&&M.inventory.capacity)||[];
 const cp=caps.find(r=>r.context===c.id)||caps.find(r=>!r.context)||{};
 const pick=(...xs)=>{for(const [v,src] of xs){const n=num(v);if(n!=null&&n>0)return [n,src];}return [null,''];};
 const [users,uSrc]=pick([P.users_count,'brief'],[cp.users,'capacity profile']);
 let [peak,pSrc]=pick([P.peak,P.peakBasis==='estimated-from-users'?'estimated from users':'brief'],[cp.peakTps,'capacity profile']);
 let perVcpu=25;
 if(typeof loadAssumptions==='function'&&M.product){try{perVcpu=loadAssumptions(P).perVcpu||25;}catch(e){}}
 if(!peak&&users&&typeof loadEstimate==='function'){try{const a=M.product?loadAssumptions(P):LOAD_PROFILES[0][2];peak=loadRound(loadEstimate(users,a,0).peak);pSrc='calculated from users';}catch(e){}}
 let [dataGb,dSrc]=pick([P.dataGb,P.dataBasis==='estimated-from-users'?'estimated from users':'brief'],[cp.dataGb,'capacity profile']);
 if(!dataGb&&users&&typeof dataEstimate==='function'&&M.product){try{dataGb=Math.max(1,Math.round(dataEstimate(users,dataAssumptions(P),0).total));dSrc='calculated from users';}catch(e){}}
 let [avail,aSrc]=pick([P.availability,'brief'],[cp.availability,'capacity profile']);
 if(!avail){avail=99.5;aSrc='assumed (not answered)';}
 const growth=typeof loadGrowth==='function'?loadGrowth():0;
 const sized=peak?peak*1.3*Math.pow(1+growth/100,3):null;
 const t=tpText(c),has=re=>re.test(t);
 const f={
  video:has(/\b(video|stream(?:ing)?|ott|playback|movies?|episodes?|live tv)\b/),
  payments:has(/\b(payments?|billing|subscriptions?|checkout|card|invoices?)\b/),
  search:has(/\b(search|catalogue|catalog|browse|find)\b/),
  files:has(/\b(uploads?|photos?|photograph\w*|images?|documents?|attachments?|files?|pdfs?)\b/),
  notify:has(/\b(alerts?|notifications?|notify|emails?|sms|reminders?)\b/),
  realtime:has(/\b(real-?time|live|chat|instant|tracking)\b/),
  reports:has(/\b(reports?|dashboards?|analytics|insights?)\b/),
  staff:has(/(staff|employees?|agents?|managers?|admins?|administrators?|operators?|technicians?|engineers?|clerks?|dispatchers?|supervisors?|content managers?|support)/)||D.tenancy==='internal'||tpRoles(P.users).some(r=>!/(customers?|viewers?|subscribers?|shoppers?|consumers?|members?|patients|citizens|students|public|guests?|visitors?|users?)/i.test(r)),
  customers:has(/\b(customers?|viewers?|subscribers?|shoppers?|consumers?|members?|patients|citizens|students|public)\b/)||['saas-multi','saas-single'].includes(D.tenancy),
  iot:has(/\b(sensors?|iot|internet of things|telemetry|meters?|gateways? on site|machines? send|devices? send|readings?|plc|scada|trackers?|wearables?)\b/)||(I_D_channels(D).includes('embedded'))};
 let regs=[];
 try{const r=(M.scope&&M.scope.regulation)||{};regs=(r.list||[]).filter(id=>typeof COMPLIANCE_BY_ID!=='undefined'&&COMPLIANCE_BY_ID[id]).map(id=>COMPLIANCE_BY_ID[id][1]);}catch(e){}
 return {P,D,users,uSrc,peak,pSrc,perVcpu,dataGb,dSrc,avail,aSrc,growth,sized,f,regs,caps:(M.inventory.contexts||[]).length};
}
const tpN=n=>typeof loadFmt==='function'?loadFmt(n):String(Math.round(n*10)/10);

function techPlan(c,s,I){
 I=I||techPlanInputs(c);
 const d=s.d,ap=d.approach,tier=s.tier||s.name,fixed=APPROACH_FIXED[ap]||{};
 // Shared concerns owned by a bought product or partner platform (set by the System blueprint on its context).
 const EXT=(c&&c.ext)||{},extBy=k=>EXT[k]?(EXT[k].by==='partner'?'partner platform':'bought product')+' chosen for "'+EXT[k].name+'"':'';
 const ch=k=>fixed[k]||d.choices[k];
 const arch=ch('architecture')||'',db=ch('database')||'',integ=[].concat(ch('integration')||[]);
 const where=I.D.where||'';
 const cloud=TP_CLOUD[ch('cloud')||'']??TP_CLOUD[where]??'';
 const cn=TP_CLOUD_NAME[cloud]||'the chosen cloud';
 const svc=k=>tpSvc(k,cloud);
 const saas=ap==='buy'||ap==='replace',platform=ap==='partner';
 const T=tier==='Safe'?0:tier==='Balanced'?1:2;
 // Separate services only when the architecture says so (or Ambitious with no architecture chosen).
 const split=['microservices','events','serverless'].includes(arch)||(T===2&&!arch);
 const L=[];const add=(id,label,t,why,src)=>{if(t)L.push({id,label,t,why:why||'',src:src||'rule'});};
 // Quantities for the cost estimate (assets/tier-cost.js): set on the layer added last with that id.
 const qty=(id,q)=>{const l=[...L].reverse().find(x=>x.id===id);if(l)l.q=q;};
 const pub=['azure','aws','gcp','oci','ibm'].includes(cloud)||(!cloud&&!['onprem','private','edge','hybrid'].includes(where));

 /* Route */
 add('route','Route',s.a.label+(saas?': a product or SaaS does the work; you configure it and build the connections':platform?': built on a platform or low-code product with a partner':''),'Scenario approach','your scenario');

 /* Hosting model */
 const owner=I.D.hosting&&typeof HOSTING_MODELS!=='undefined'?HOSTING_MODELS.find(m=>m.id===I.D.hosting):null;
 let host,hostWhy,hostSrc='rule';
 if(saas){host='The vendor hosts the product (SaaS); your connectors and any add-ons run on '+(T===0?svc('fn'):svc('ctr'));hostWhy='Buying means the vendor runs the main system';}
 else if(platform){host='The partner platform hosts it; custom parts on '+svc('ctr');hostWhy='A low-code or partner platform runs the app';}
 else if(owner){host=owner.name+': '+(owner[cloud]||owner[{onprem:'onprem'}[cloud]||'onprem']||'');hostWhy='You chose this hosting model in the brief';hostSrc='your choice';}
 else if(ap==='rehost'||(arch==='keep'&&T===0&&ap!=='replatform')){host='Virtual machines: '+svc('vm');hostWhy='Least change: the app moves as it is';}
 else if(arch==='serverless'){host='Serverless functions: '+svc('fn');hostWhy='The scenario uses a serverless architecture';}
 else if(T===0){host=pub?'Managed web app platform: '+svc('paas'):'Virtual machines: '+svc('vm');hostWhy=pub?'Least to operate on a public cloud':'Private cloud: simplest to run on your own platform';}
 else if(T===1){host=pub?'Managed containers: '+svc('ctr'):'Containers on '+svc('k8s');hostWhy='Containers scale automatically and package each part the same way';}
 else{host=(arch==='microservices'||arch==='events'||!pub?'Kubernetes: '+svc('k8s'):'Managed containers: '+svc('ctr'))+(I.f.notify||I.f.video||integ.includes('events')?', plus '+svc('fn')+' for background events':'');hostWhy='Most room to grow: many services, automated roll-outs';}
 add('hosting','Hosting model',host+(cloud?'':' (pick a cloud to name the services)'),hostWhy+'. Runs on '+cn+'.',hostSrc);
 const kind=saas?'saas':platform?'platform':/^Virtual machines/.test(host)?'vm':/^Managed web app/.test(host)?'paas':/^Kubernetes|^Containers on/.test(host)?'k8s':/^Serverless/.test(host)?'fn':/^Managed containers/.test(host)?'ctr':owner?({vm:'vm',paas:'paas',containers:'ctr',kubernetes:'k8s',serverless:'fn',static:'paas',lowcode:'platform',onsite:'vm'}[owner.id]||'ctr'):'ctr';
 qty('hosting',{kind,cloud,pub});

 /* Traffic in: north–south */
 const ns=[];
 if(pub||I.f.customers)ns.push(T===0?svc('lb')+' with TLS':svc('waf'));else ns.push(svc('lb')+' with TLS');
 if(T>=1&&(integ.includes('api')||arch==='microservices'||arch==='events'||(I.D.channels||[]).some(x=>x==='api'||x==='mobile')))ns.push('API gateway: '+svc('apigw'));
 if(I.f.video||(T===2&&I.f.customers))ns.push('CDN: '+svc('cdn'));
 if(T===2)ns.push('rate limits per client');
 if(saas&&T===0)ns.length=0,ns.push('Users reach the vendor directly; your connectors call its API');
 add('north','Traffic in (north–south)',ns.join(' → '),(pub?'Internet-facing':'Inside your network')+(I.f.customers?', customers sign in from outside':'')+'; TLS everywhere','rule');
 qty('north',{lb:saas&&T===0?0:1,waf:ns.some(x=>/WAF|Armor|Front Door/.test(x))&&!(saas&&T===0),apigw:ns.some(x=>/^API gateway/.test(x)),cdn:ns.some(x=>/^CDN/.test(x))});

 /* Compute */
 if(saas||platform){
  add('compute','Compute and sizing','Sized by the vendor. Ask them to confirm '+(I.peak?tpN(I.peak)+' requests/s at peak':'your peak load')+(I.users?' for '+tpN(I.users)+' users':'')+(I.sized?' (≈ '+tpN(I.sized)+'/s in 3 years)':'')+'. Connectors: 2 small instances (1 vCPU, 2 GB).',I.peak?'Peak '+I.pSrc:'No peak or users yet','calculation');
  qty('compute',{vcpuAvg:2,vcpuMax:2,ramPerVcpu:2});
 }else if(I.sized){
  const vcpu=Math.max(2,Math.ceil(I.sized/I.perVcpu));
  let t,cq;
  // Autoscaled capacity runs near the minimum most of the month: average = min + 30% of the range.
  const avg=(mn,mx)=>mn+(mx-mn)*0.3;
  if(T===0){const size=vcpu<=4?2:vcpu<=16?4:8,n=Math.max(2,Math.ceil(vcpu/size)+1);t=n+' × '+size+' vCPU / '+(size*4)+' GB RAM, fixed (one spare)';cq={vcpuAvg:n*size,vcpuMax:n*size,ramPerVcpu:4};}
  else if(T===1){const mn=Math.max(2,Math.ceil(I.peak/I.perVcpu/2)),mx=Math.max(mn+1,Math.ceil(vcpu/2)+1);t='Autoscale '+mn+'–'+mx+' instances × 2 vCPU / 4 GB';cq={vcpuAvg:avg(mn,mx)*2,vcpuMax:mx*2,ramPerVcpu:2};}
  else if(T===2&&!split){const mn=Math.max(3,Math.ceil(I.peak/I.perVcpu/2)+1),mx=Math.max(mn+2,Math.ceil(vcpu/2)+2);t='Autoscale '+mn+'–'+mx+' instances × 2 vCPU / 4 GB across zones';cq={vcpuAvg:avg(mn,mx)*2,vcpuMax:mx*2,ramPerVcpu:2};}
  else{const svcs=Math.min(8,Math.max(3,Math.round(I.caps/3)||3));const per=Math.max(1,Math.ceil(vcpu/svcs));t='About '+svcs+' services, each autoscaling 2–'+(per+2)+' pods × 1 vCPU / 2 GB';cq={vcpuAvg:svcs*avg(2,per+2),vcpuMax:svcs*(per+2),ramPerVcpu:2};}
  add('compute','Compute and sizing',t+' · '+vcpu+' vCPU at '+I.perVcpu+' requests/s each, OS disk 64 GB per server',
   'Peak '+tpN(I.peak)+' requests/s ('+I.pSrc+') × 1.3 headroom'+(I.growth?' × '+I.growth+'% growth for 3 years':'')+' = '+tpN(I.sized)+'/s','calculation');
  qty('compute',cq);
 }else add('compute','Compute and sizing','Add expected users or a peak in the brief to size this','No users or peak yet','rule');

 /* East–west */
 add('east','Traffic between parts (east–west)',saas?'Connectors call the vendor API; retries and a dead-letter queue for failed calls'
  :T===0?'In-process calls inside one app; background jobs for slow work'
  :T===1?'Modules call each other in-process; '+(integ.includes('queue')||I.f.video||I.f.notify?'a queue for slow work':'internal REST for the few separate parts')
  :split?'Services talk over REST/gRPC with mutual TLS (service mesh) and publish events for everything slow'
  :'Modules call each other in-process; they publish events (outbox) for slow work and for the other parts',
  T===2&&split?'Many services need encrypted, observable internal traffic':T===2?'One deployable app with clear module boundaries':'Keep internal calls simple','rule');

 /* Database */
 if(saas||platform)add('database','Database','Inside the '+(platform?'partner platform':'vendor product')+'. Plan an export or reporting copy'+(I.dataGb?' of about '+tpN(I.dataGb)+' GB over 3 years':''),'You do not run the vendor database','rule');
 else{
  const eng=db&&db!=='keep'?db:(T===2&&I.f.realtime?'nosql':'postgres');
  const dbSvc=TP_DB[eng]?svc(eng):'',dbN=TP_DB_NAME[eng]||eng;
  const engName=db==='keep'?'Keep the current database':dbSvc.startsWith(dbN)?dbSvc:dbN+(dbSvc?': '+dbSvc:'');
  const gb=I.dataGb?Math.ceil(I.dataGb*1.5/50)*50:null;
  const dbv=I.sized?Math.max(2,Math.min(32,Math.ceil(Math.ceil(I.sized/I.perVcpu)/2))):2;
  const ha=I.avail>=99.99?'3 zones + standby region':I.avail>=99.95?'primary + standby in another zone + read replica':I.avail>=99.5?'primary + standby in another zone':'single server + daily backups';
  const more=T===2&&(arch==='microservices'||arch==='events')?'; one database per service, read models for busy screens (CQRS)':T>=1&&(I.f.reports||(I.peak||0)>=100)?'; a read replica for reports and busy reads':'';
  add('database','Database',engName+' · '+dbv+' vCPU / '+(dbv*4)+' GB RAM'+(gb?' · '+gb+' GB storage':'')+' · '+ha+more+' · point-in-time backups kept 35 days',
   (I.dataGb?'Data '+tpN(I.dataGb)+' GB ('+I.dSrc+') + 50% room':'No data volume yet')+'; availability '+I.avail+'% ('+I.aSrc+')','calculation');
  const inst=(I.avail>=99.99?3:I.avail>=99.5?2:1)+(I.avail>=99.95&&I.avail<99.99?1:0)+(more.includes('read replica')?1:0)+(more.includes('per service')?2:0);
  qty('database',{vcpu:dbv,instances:inst,storageGb:gb||50,backupGb:gb||50,managed:db!=='keep'});
 }

 /* Cache */
 const cacheGb=I.dataGb?Math.max(1,Math.min(64,Math.round(I.dataGb*0.02))):1;
 add('cache','Cache',saas||platform?(T===0?'':'Cache '+(platform?'platform':'vendor')+' API answers in your connectors ('+svc('redis')+', 1 GB)')
  :T===0?((I.peak||0)>=50||I.f.search?svc('redis')+', '+cacheGb+' GB, single node: cache-aside for '+(I.f.search?'catalogue and ':'')+'frequent reads':'In-app memory cache for reference data')
  :svc('redis')+', '+cacheGb+' GB'+(T===2?' clustered across zones':', with a replica')+': cache-aside for '+[I.f.search&&'catalogue',I.f.video&&'resume positions',I.f.customers&&'sessions'].filter(Boolean).concat(['frequent reads']).join(', ')+(T===2&&I.f.customers?'; CDN caching for public pages':''),
  'About 2% of the data kept hot; '+((I.peak||0)>=50?'peak above 50/s':'light load'),'rule');
 qty('cache',{gb:saas||platform?1:cacheGb,nodes:saas||platform?(T===0?0:1):T===0?((I.peak||0)>=50||I.f.search?1:0):T===1?2:3});

 /* Messaging */
 const slow=[I.f.video&&'upload → transcode → publish',I.f.notify&&'alerts and emails',I.f.payments&&'payment webhooks',I.f.reports&&'report exports'].filter(Boolean);
 add('messaging','Messaging and background work',T===0?(slow.length?'Background worker with a job table for '+slow.join(', '):'Scheduled jobs only')
  :T===1?'Queue: '+svc('queue')+(slow.length?' for '+slow.join(', '):' for slow work')
  :'Event streaming: '+svc('stream')+' with the outbox pattern'+(slow.length?'; events for '+slow.join(', '):''),slow.length?'The brief has slow or event-driven work':'Nothing slow in the brief yet','rule');
 qty('messaging',{kind:T===0?'worker':T===1?'queue':'stream'});

 /* Files and media */
 if(I.f.video&&EXT.media){
  add('files','Files and media','Playback and delivery by the '+extBy('media')+'. The platform keeps uploads and their details in object storage ('+svc('blob')+') and hands them over through its API.','Shared concern owned outside the platform','your scenario');
  qty('files',{objectGb:I.dataGb||50,video:false,videoPlatform:false});
 }else if(I.f.video||I.f.files){
  const parts=['Object storage: '+svc('blob')];
  if(I.f.video){parts.push(T===0?'a video platform bought end to end (upload, transcode, player, CDN)':'transcoding: '+svc('video'));parts.push('adaptive streaming (HLS/DASH) through '+svc('cdn'));if(I.f.payments)parts.push('DRM for paid content');}
  add('files','Files and media',parts.join(' · ')+(I.f.video&&I.dataGb&&I.dataGb<1000?' · Check the data volume: '+tpN(I.dataGb)+' GB is small for video (an hour of HD is about 2–3 GB per quality level)':''),I.f.video?'The brief describes video':'The brief mentions uploads or files','rule');
  qty('files',{objectGb:I.dataGb||50,video:I.f.video,videoPlatform:I.f.video&&T===0});
 }

 /* Search */
 if(I.f.search&&EXT.search)add('search','Search','Provided by the '+extBy('search'),'Shared concern owned outside the platform','your scenario');
 else if(I.f.search&&(saas||platform))add('search','Search','Provided by the '+(platform?'partner platform':'vendor product'),'The brief has search or browsing','rule');
 else if(I.f.search)add('search','Search',T===0?'Database full-text search':T===1?svc('search')+', updated when records change':svc('search')+', fed by events, with suggestions',
  'The brief has search or browsing','rule');
 if(I.f.search&&!saas&&!platform&&!EXT.search)qty('search',{tier:T});

 /* Integrations */
 const iParts=[];
 if(integ.includes('api'))iParts.push('REST APIs described with OpenAPI');
 if(integ.includes('ipaas'))iParts.push('an integration platform');
 if(integ.includes('queue'))iParts.push('message queue');
 if(integ.includes('events'))iParts.push('event streaming');
 if(integ.includes('mft'))iParts.push('managed file transfer');
 if(integ.includes('cdc'))iParts.push('change data capture');
 const kinds=String(I.P.integrations||'').split(/[;\n]/).map(x=>x.split(':')[0].trim()).filter(Boolean).slice(0,3);
 add('integration','Connections to other systems',(iParts.length?iParts.join(', '):T===0?'Direct API calls':'APIs through the gateway')+(kinds.length?' · to '+kinds.join('; '):'')+(I.f.payments?' · payment webhooks verified and processed once (idempotent)':''),kinds.length?'Integrations in the brief':'Scenario integration choice',kinds.length?'brief':'your scenario');

 /* Devices and IoT */
 if(I.f.iot){
  const iot=T===0?'Devices connect over MQTT with TLS to '+svc('iotHub')+'; one certificate per device; readings go into the database in batches'
   :T===1?'Devices connect over MQTT with TLS to '+svc('iotHub')+'; device provisioning with X.509 certificates, a device registry and twin (desired and reported state); readings flow to a queue and into a time-series store ('+svc('tsdb')+'); over-the-air updates in stages'
   :'Fleet-scale: '+svc('iotHub')+' with automatic provisioning, device twins and per-device permissions; edge runtime on site ('+svc('iotEdge')+') to work offline and filter readings; readings streamed as events ('+svc('stream')+') into '+svc('tsdb')+'; hot path for alerts, cold path in object storage; staged over-the-air updates with rollback';
  add('iot','Devices and IoT',iot+' · devices never connect to the database directly; lost or stolen devices can be revoked','The brief mentions devices, sensors or readings','rule');
 }

 /* AI features (suggested) */
 const txt=tpText(c),hasT=re=>re.test(txt),ai=[];
 if(I.f.search)ai.push('semantic search and recommendations ("more like this") over the catalogue or records');
 if(hasT(/\b(support|help|questions?|faq|problems?|escalat\w*|tickets?|complaints?)\b/))ai.push('a support assistant that answers from your help content and records, and hands over to a person');
 if(hasT(/\b(documents?|invoices?|forms?|contracts?|receipts?|pdfs?|scanned|photos?|photograph\w*)\b/))ai.push('document and photo extraction (fields from invoices, forms or photos, checked by a person)');
 if(I.f.iot||hasT(/\b(readings?|sensors?|anomal\w*|predict\w*|maintenance|failures?)\b/))ai.push('anomaly detection and failure prediction on readings (a trained model on the time-series data)');
 if(I.f.reports)ai.push('questions in plain language over reports, limited to data the user may see');
 if(hasT(/\b(content|titles?|videos?|articles?|publish\w*|upload\w*)\b/))ai.push('automatic tags, summaries and subtitles for uploaded content, reviewed before publishing');
 if(I.f.payments)ai.push('fraud signals from the payment provider first, before building your own');
 if(ai.length){
  const stack=T===0?'Use the cloud model service ('+svc('aiModels')+') through its API with fixed prompt templates; keep embeddings in the database (PostgreSQL with pgvector); log every prompt and answer'
   :T===1?'Model service: '+svc('aiModels')+'; retrieval (RAG) with '+svc('vector')+'; small fast models for routine work and a larger one for hard cases; cached answers; an evaluation set checked before each release'
   :'An AI gateway in front of '+svc('aiModels')+' (routing between models, rate and cost limits, fallbacks); '+svc('vector')+' kept up to date by events; evaluations and quality monitoring in production; fine-tuned or own models only where they clearly beat the managed ones';
  add('ai','AI features (suggested)','Where AI can help: '+ai.join('; ')+'. Stack: '+stack+'. Guardrails: '+svc('aiSafety')+', personal data removed or masked before it reaches a model, a person approves anything that changes money, access or published content, a monthly usage budget'+(I.regs.length?', and the rules of '+I.regs.slice(0,2).join(', '):''),
   'Opportunities read from the brief and capabilities; you choose which to build','rule');
  qty('ai',{count:ai.length});
 }

 /* Server types: what kind of server each part needs */
 {
  const cq=(L.find(l=>l.id==='compute')||{}).q,kindL=(L.find(l=>l.id==='hosting')||{}).q||{};
  const managedRun=['paas','ctr','fn','saas','platform'].includes(kindL.kind);
  const per=cq?(T===0?Math.max(2,Math.min(8,cq.vcpuMax/Math.max(2,Math.ceil(cq.vcpuMax/8)))):2):2;
  const parts=[];
  const winOnly=/dotnet/.test(String(d.choices.language||''))&&/framework|webforms|wcf|vb\.?net/i.test(tpText(c));
  const os=winOnly?'Windows Server 2022 (only for .NET Framework parts)':'Linux (Ubuntu LTS or Red Hat Enterprise Linux)';
  if(saas||platform)parts.push('main system: run by the '+(platform?'partner':'vendor')+'; connectors on '+(managedRun?'the platform\'s shared capacity':tpServer('general',cloud,2)));
  else parts.push('app servers: '+(managedRun?'no servers to pick on the managed platform (it runs on general-purpose capacity); if you choose dedicated capacity, '+tpServer('general',cloud,per):tpServer('general',cloud,per)));
  const dbq=(L.find(l=>l.id==='database')||{}).q;
  if(dbq)parts.push('database: memory optimised, '+(dbq.managed&&cloud?'the managed service\'s memory-optimised tier ('+dbq.vcpu+' vCPU)':tpServer('memory',cloud,dbq.vcpu))+', SSD storage with provisioned IOPS');
  const kq=(L.find(l=>l.id==='cache')||{}).q;if(kq&&kq.nodes)parts.push('cache: memory optimised ('+kq.gb+' GB per node)');
  if(I.f.video&&!EXT.media&&T>=1)parts.push('transcoding workers: '+tpServer('compute',cloud,8)+', or '+tpServer('gpu',cloud)+' for heavy volumes, started only when there is work');
  else if((L.find(l=>l.id==='messaging')||{}).q)parts.push('background workers: '+(managedRun?'same platform as the app':tpServer('general',cloud,2)));
  if(I.f.iot&&T>=1)parts.push('readings ingestion: '+(managedRun?'same platform as the app':tpServer('compute',cloud,4)));
  parts.push('operating system: '+os+'; OS disk 64 GB SSD; images patched monthly');
  add('servers','Server types',parts.join(' · '),'Workload of each part: web and API = general purpose, databases and caches = memory, encoding and ingestion = compute, video or self-hosted AI = GPU','rule');
 }

 /* Payments: where the payment gateway sits */
 if(I.f.payments&&EXT.payments){
  add('payments','Payments (where the gateway sits)','Handled by the '+extBy('payments')+': use its checkout and billing. The platform receives its signed webhooks through the API gateway, keeps the payment or subscription status it needs for access checks, and never handles card data.','Shared concern owned outside the platform','your scenario');
 }else if(I.f.payments){
  const named=(String(I.P.integrations||'')+' '+String(I.P.buyPref||'')).match(/\b(stripe|adyen|braintree|paypal|razorpay|payu|paytm|checkout\.com|worldpay|square|mollie|chargebee|recurly|paddle)\b/i);
  const prov=named?named[1].replace(/^./,c=>c.toUpperCase())+' (named in the brief)':'a payment provider such as Stripe, Adyen or Braintree (Razorpay or PayU in India)';
  const subs=/\b(subscriptions?|plans?|recurring|renew\w*)\b/.test(tpText(c))||/subscription/i.test(String(I.P.integrations||''));
  const pay=T===0?'Outside the platform: '+prov+' with its hosted checkout page'+(subs?' and its subscription billing and customer portal':'')+'; the app redirects to it and receives one signed webhook per event'
   :T===1?'Outside the platform: '+prov+' with embedded payment fields (card data goes straight to the provider as a token)'+(subs?', its subscription billing':'')+'; webhooks come in through the API gateway, are verified, queued and processed once (idempotency keys); a nightly reconciliation job compares payouts with orders'
   :'A payment module on the platform in front of '+prov+(subs?' and its subscription billing':'')+': tokenised payment fields, webhooks verified at the API gateway and published as events, a payment ledger, retries and refunds as a saga, nightly reconciliation; room for a second provider or local payment methods';
  add('payments','Payments (where the gateway sits)',pay+' · card numbers are never stored or handled by your servers, which keeps PCI DSS scope to the smallest level (SAQ A for hosted pages or fields)'+(subs?' · access to paid content follows the subscription status from the provider':''),
   'The brief mentions payments'+(subs?' and subscriptions':''),named?'brief':'rule');
 }

 /* Authentication: who you are */
 const ids=[];
 if(I.f.staff)ids.push('staff: single sign-on with '+svc('idStaff')+', MFA required'+(T>=1?', conditional access by device and location':''));
 if(I.f.customers&&EXT.identity)ids.push('customers: accounts live in the '+extBy('identity')+'; federate it with the staff sign-in over OpenID Connect so each person signs in once');
 else if(I.f.customers)ids.push('customers: '+svc('idCust')+' (sign-up, email or social sign-in, password reset'+(T===2?', passkeys':'')+'; step-up MFA for '+(I.f.payments?'payment and ':'')+'account changes)');
 if(['saas-multi','saas-single'].includes(I.D.tenancy))ids.push('business customers: SAML or OIDC single sign-on with their own directory');
 if(!ids.length)ids.push('single sign-on with '+svc('idStaff')+', MFA for admins');
 const tok=T===0?'OpenID Connect sign-in, server-side sessions with a short idle timeout':'OpenID Connect with authorization code + PKCE for web and mobile; access tokens valid 15 minutes, refresh tokens rotated';
 const svcAuth=saas||platform?'connectors use OAuth client credentials kept in '+svc('vault')
  :T===0?'service accounts with secrets in '+svc('vault'):T===1?'workload / managed identities for app-to-database and app-to-queue access (no passwords in config)':'workload identities plus mutual TLS between services';
 add('identity','Authentication (who you are)',ids.join(' · ')+' · '+tok+' · '+svcAuth+(I.D.channels&&I.D.channels.includes('api')?' · partners call the API with OAuth client credentials':''),'Who the users are and the channels in the brief','brief');

 /* Authorization: what you may do */
 const roles=tpRoles(I.P.users);
 const az=[];
 az.push('roles (RBAC): '+(roles.length?roles.concat(roles.some(r=>/admin/i.test(r))?[]:['Administrator']).join(', '):'one role per user group in the brief, plus Administrator'));
 if(I.f.customers)az.push('ownership checks: a '+(roles.find(r=>/viewer|customer|member|subscriber|user/i.test(r))||'customer').toLowerCase()+' sees and changes only their own account'+(I.f.payments?', payments':'')+' and data');
 if(I.f.payments&&I.f.video)az.push('entitlements: what a viewer may watch follows their subscription plan and region');
 if(I.D.tenancy==='saas-multi')az.push('tenant isolation: tenant id on every record and row-level security in the database');
 az.push('admin and support screens need the staff sign-in and their own role; "view as customer" is read-only and logged');
 az.push(T===2?'permissions decided in one policy service (Open Policy Agent or Cedar'+(cloud==='aws'?' / Amazon Verified Permissions':'')+') so every module asks the same question'
  :T===1?'permission checks in one shared module, enforced at the API gateway and again in the app':'permission checks in the app for every screen and API call');
 az.push('every permission change and admin action goes to the audit log; least privilege for service accounts');
 add('authz','Authorization (what you may do)',az.join(' · '),roles.length?'User roles in the brief':'Rule for every product',roles.length?'brief':'rule');

 /* Security and compliance */
 add('security','Security and compliance','Secrets in '+svc('vault')+' · encryption at rest and in transit'+(T>=1?' · private network for the database and cache':'')+(T===2?' · image scanning and signed releases':'')+(I.regs.length?' · design for '+I.regs.slice(0,3).join(', '):''),I.regs.length?'Regulations chosen in the brief':'Baseline for every product',I.regs.length?'brief':'rule');

 /* Availability and recovery */
 const downH=(100-I.avail)/100*8760;
 const zones=I.avail>=99.95?3:I.avail>=99.5?2:1;
 const plan=T===0?(zones>1?zones+' zones, warm standby':'one zone, restore from backup')
  :T===1?(zones>1?zones+' zones, automatic failover':'one zone with a standby server, quick manual failover')+', restore tested every quarter'
  :(I.avail>=99.95?'active in 2 regions':Math.max(2,zones)+' zones'+(zones<2?' (more than the target asks)':''))+', automatic failover, chaos drills';
 add('availability','Availability and recovery',I.avail+'% allows about '+(downH>=1?tpN(downH)+' hours':tpN(downH*60)+' minutes')+' down a year · '+plan+' · recover within '+(I.avail>=99.95?'1 hour, lose at most 5 min':I.avail>=99.5?'4 hours, lose at most 15 min':'1 day, lose at most 24 hours'),
  'Availability '+I.aSrc,I.aSrc.startsWith('assumed')?'rule':'brief');

 /* Operations */
 add('operations','Operations',svc('mon')+' · '+svc('cicd')+' · '+(T===0?'dev and prod':T===1?'dev, test and prod; blue-green releases':'dev, test, prod and a preview per change; canary releases'),'Every option needs monitoring and repeatable releases','rule');

 /* Patterns */
 const pat=T===0?['layered application','cache-aside','retry with backoff','health checks']
  :T===1?['modular monolith','cache-aside','queue-based load levelling','retry with backoff and circuit breaker','health checks']
  :split?['microservices or event-driven services','outbox','CQRS for busy reads','saga for multi-step work'+(I.f.payments?' such as payments':''),'circuit breaker','strangler fig for the old parts']
  :['modular monolith with enforced module boundaries','outbox','CQRS read models for busy screens','saga for multi-step work'+(I.f.payments?' such as payments':''),'circuit breaker','feature flags'];
 if(saas)pat.splice(0,pat.length,'anti-corruption layer around the vendor','retry and dead-letter queue','reconciliation job');
 add('patterns','Design patterns',pat.join(', '),'Fits the scenario route and architecture','rule');

 /* Often missed */
 const miss=['audit log of who changed what','admin and support screens','backups tested by restoring'];
 if(I.f.notify||I.f.customers)miss.push('email and SMS sending ('+svc('notify')+')');
 if(I.f.customers)miss.push('cookie consent and privacy requests');
 if(I.f.payments)miss.push('refunds and failed-payment retries');
 if(I.f.video)miss.push('thumbnails, subtitles and content takedown');
 if(I.f.reports)miss.push('a reporting copy so reports do not slow the app');
 add('missed','Often missed',miss.join(' · '),'Common gaps for this kind of product','rule');
 return L;
}

const TECH_PLAN_OPEN=new Map();   // capability id → open or closed, remembered while the page is open
function techPlanTable(c,scen){
 if(!scen||!scen.length)return null;
 const first=(typeof decCtxs==='function'?decCtxs()[0]:null);
 // The System blueprint above shows the shared platform, so these stay folded until opened.
 const open=TECH_PLAN_OPEN.has(c.id)?TECH_PLAN_OPEN.get(c.id):typeof systemBlueprintPanel!=='function'&&(!first||first.id===c.id);
 const I=techPlanInputs(c);
 const plans=scen.map(s=>({s,L:techPlan(c,s,I)}));
 const ORDER=['route','hosting','north','compute','servers','east','database','cache','messaging','iot','files','search','integration','payments','ai','identity','authz','security','availability','operations','patterns','missed'];
 const ids=[];for(const p of plans)for(const l of p.L)if(!ids.find(x=>x[0]===l.id))ids.push([l.id,l.label]);
 ids.sort((a,b)=>ORDER.indexOf(a[0])-ORDER.indexOf(b[0]));
 const srcTag={'your choice':'you chose','brief':'from brief','calculation':'calculated','your scenario':'scenario','rule':'rule'};
 const facts=[I.users?tpN(I.users)+' users ('+I.uSrc+')':'users not given',I.peak?tpN(I.peak)+' requests/s peak ('+I.pSrc+')':'peak not given',
  I.dataGb?tpN(I.dataGb)+' GB data ('+I.dSrc+')':'data volume not given',I.avail+'% availability ('+I.aSrc+')',(I.D.where?'runs on: '+I.D.where:'where it runs not given')];
 const box=h('details',{class:'tech-plan',open:open?'':null},[
  h('summary',{},[h('b',{text:'Technical plan for this '+(isNewProduct()?'capability':'context')+' by layer'}),h('small',{class:'muted',text:' · '+ids.length+' layers, '+plans.map(p=>p.s.name).join(' · ')+' side by side'})]),
  h('p',{class:'muted tp-basis',text:'Based on '+facts.join(' · ')+'. A rule-based first cut; "Generate options" below asks the AI agent team for a reviewed blueprint.'}),
  h('div',{class:'tp-scroll'},h('table',{class:'tp-table'},[
   h('thead',{},h('tr',{},[h('th',{text:'Layer'}),...plans.map(p=>h('th',{},[h('b',{text:p.s.name}),h('small',{text:p.s.a.label})]))])),
   h('tbody',{},ids.map(([id,label])=>h('tr',{},[h('th',{scope:'row',text:label}),...plans.map(p=>{const l=p.L.find(x=>x.id===id);
    return h('td',{},l?[h('div',{text:l.t}),h('small',{class:'tp-why',text:(srcTag[l.src]||l.src)+' · '+l.why})]:[h('small',{class:'muted',text:'—'})]);})])))]))]);
 box.addEventListener('toggle',()=>TECH_PLAN_OPEN.set(c.id,box.open));
 return box;
}

(function techPlanStyle(){
 if(document.getElementById('tech-plan-style'))return;
 const css=`.tech-plan{margin:12px 0;border:1px solid #d9e1f2;border-radius:10px;background:#fff}
.tech-plan>summary{padding:10px 12px;cursor:pointer}.tp-basis{margin:0 12px 8px;font-size:12px}
.tp-scroll{overflow-x:auto;padding:0 12px 12px}
.tp-table{width:100%;border-collapse:collapse;font-size:12.5px;min-width:720px}
.tp-table th,.tp-table td{border-top:1px solid #eef1f7;padding:7px 8px;vertical-align:top;text-align:left}
.tp-table thead th{background:#f8faff}.tp-table thead small{display:block;color:var(--muted);font-weight:400}
.tp-table tbody th{width:150px;color:var(--muted);font-weight:600;white-space:nowrap}
.tp-table td{width:calc((100% - 150px)/3)}.tp-why{display:block;margin-top:3px;color:var(--muted);font-size:11px}`;
 document.head.append(Object.assign(document.createElement('style'),{id:'tech-plan-style',textContent:css}));
})();
