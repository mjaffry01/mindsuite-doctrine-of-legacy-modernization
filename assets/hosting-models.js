/* MindSuite web app · hosting model choices
   The ways an application can be hosted (virtual machines, managed web app platform, managed containers,
   Kubernetes, serverless functions, static web app + API, low-code platform, on-site servers), each explained
   in plain words with what it is good for, what to watch, what you still run, how it scales, how it is billed
   and the matching service on Azure, AWS, Google Cloud, IBM Cloud, Oracle Cloud and on-premises.
   hostingSuggest(P) ranks them from the brief with local rules (where it runs, tenancy, channels, load, wording)
   and gives the reasons; the owner picks one (stored as productBrief().deploy.hosting, an id, so later
   diagrams and the agents can use it). Nothing is chosen automatically.
   Hooks: hostingModelPicker() in the brief's Deployment type block and in deploymentOnlySection();
   hostingFacts(add) from productFacts() (assets/project-mode.js). */

const HOSTING_MODELS=[
 {id:'vm',name:'Virtual machines',kind:'IaaS',
  what:'Rent servers in the cloud (or run them on your own hardware) and install the operating system, runtime and app yourself.',
  good:'Moving an existing app as it is; software that needs full control of the server or a licence tied to a machine.',
  watch:'You patch and secure the operating system; scaling is slower and you pay while servers sit idle.',
  run:'Operating system, runtime, app, scaling',scale:'Add bigger or more machines (scale sets)',bill:'Per server-hour, running or idle',
  azure:'Azure Virtual Machines, VM Scale Sets',aws:'Amazon EC2, Auto Scaling groups',gcp:'Compute Engine, managed instance groups',onprem:'VMware vSphere, Hyper-V, Nutanix',ibm:'IBM Cloud Virtual Servers for VPC',oci:'OCI Compute (VMs, instance pools)'},
 {id:'paas',name:'Managed web app platform',kind:'PaaS',
  what:'Upload the code; the platform runs it with the web server, certificates, scaling and patching handled for you.',
  good:'Standard web apps and APIs, small teams, getting live quickly.',
  watch:'Only the languages and versions the platform supports; less control; costs rise at very large scale.',
  run:'App code and settings',scale:'Automatic or by schedule, more instances of a plan',bill:'Per plan size × instances',
  azure:'Azure App Service',aws:'AWS Elastic Beanstalk, AWS App Runner',gcp:'Google App Engine',onprem:'Not usually available (consider containers on OpenShift)',ibm:'IBM Cloud Code Engine (apps)',oci:'OCI Container Instances (no classic app platform)'},
 {id:'containers',name:'Managed containers',kind:'Serverless containers',
  what:'Package each part of the app as a container; the cloud runs and scales the containers without you managing servers or a cluster.',
  good:'Several services or APIs, steady or variable traffic, teams already using Docker.',
  watch:'Needs container skills and an image registry; a short delay when a service scaled to zero wakes up.',
  run:'Container images and their settings',scale:'Automatic by requests or events, can scale to zero',bill:'Per vCPU-second and GB-second used',
  azure:'Azure Container Apps',aws:'Amazon ECS on AWS Fargate',gcp:'Google Cloud Run',onprem:'Docker hosts, Podman',ibm:'IBM Cloud Code Engine',oci:'OCI Container Instances'},
 {id:'kubernetes',name:'Kubernetes',kind:'Container orchestration',
  what:'A cluster that runs many containers with fine control over networking, scaling and roll-outs.',
  good:'Many services, a platform team, running the same way across clouds or on-premises, products installed by customers.',
  watch:'The most moving parts: needs people who run Kubernetes; overkill for one small app.',
  run:'Cluster add-ons, workloads, upgrades (the cloud runs the control plane)',scale:'Pods and nodes scale automatically',bill:'Per node (server) plus a cluster fee',
  azure:'Azure Kubernetes Service (AKS)',aws:'Amazon EKS',gcp:'Google Kubernetes Engine (GKE)',onprem:'Red Hat OpenShift, Rancher, VMware Tanzu',ibm:'IBM Cloud Kubernetes Service, Red Hat OpenShift on IBM Cloud',oci:'OCI Kubernetes Engine (OKE)'},
 {id:'serverless',name:'Serverless functions',kind:'FaaS',
  what:'Small pieces of code that run only when something happens (a request, a file, a message, a timer).',
  good:'Spiky or low traffic, background jobs, scheduled tasks, reacting to events and webhooks.',
  watch:'Time limits per run, a delay on the first call, and code shaped around one provider.',
  run:'Function code and triggers',scale:'Per event, automatically, to zero',bill:'Per call and run time; almost nothing when idle',
  azure:'Azure Functions',aws:'AWS Lambda',gcp:'Cloud Run functions',onprem:'OpenFaaS, Knative',ibm:'IBM Cloud Code Engine functions',oci:'OCI Functions'},
 {id:'static',name:'Static web app + API',kind:'Front end on a CDN',
  what:'The browser app is served as files from a global content network; the data comes from a separate API.',
  good:'Single-page web apps, content and marketing sites, fast pages worldwide.',
  watch:'Only the front end: the API still needs one of the other models (functions, platform or containers).',
  run:'Front-end build and the chosen API',scale:'The content network scales on its own',bill:'Per GB served, often free to start',
  azure:'Azure Static Web Apps',aws:'AWS Amplify Hosting, S3 + CloudFront',gcp:'Firebase Hosting',onprem:'Nginx or IIS behind a CDN',ibm:'IBM Cloud Object Storage website + IBM Cloud Internet Services',oci:'OCI Object Storage + a partner CDN'},
 {id:'lowcode',name:'Low-code or SaaS platform',kind:'aPaaS',
  what:'Build screens, forms and workflows on a platform the organization already uses, with little code.',
  good:'Internal forms, approvals and workflows; data already in that platform.',
  watch:'Per-user licences, platform limits, and moving off it later is hard.',
  run:'Apps and flows built on the platform',scale:'The vendor scales it within licence limits',bill:'Per user or per app licence',
  azure:'Microsoft Power Apps (Power Platform)',aws:'Amazon Honeycode was retired; use a partner platform',gcp:'AppSheet',onprem:'Mendix, OutSystems (self-managed)',ibm:'IBM watsonx Orchestrate or a partner platform',oci:'Oracle APEX'},
 {id:'onsite',name:'On-site servers or edge devices',kind:'Edge',
  what:'Run the app on servers or devices at the site itself (plant, branch, shop, vehicle), syncing with the centre when possible.',
  good:'Unreliable internet, data that must stay on site, machines and sensors, very low delay.',
  watch:'You look after hardware in many places; updates and backups across sites need planning.',
  run:'Hardware, operating system, app, sync',scale:'More or bigger devices per site',bill:'Hardware bought plus support',
  azure:'Azure Local (Azure Stack HCI), Azure IoT Edge',aws:'AWS Outposts, AWS IoT Greengrass',gcp:'Google Distributed Cloud',onprem:'Industrial PCs, branch servers',ibm:'IBM Cloud Satellite',oci:'Oracle Compute Cloud@Customer, Roving Edge'}];

function hostingSuggest(P){
 const D=P.deploy||{},ch=D.channels||[],where=D.where||'',ten=D.tenancy||'';
 const text=['problem','users','journeys','mvp','success','integrations','buyPref'].map(k=>String(P[k]||'')).join(' ').toLowerCase();
 const has=re=>re.test(text);
 const users=num(P.users_count)||0,peak=num(P.peak)||0;
 const s={},why={};for(const m of HOSTING_MODELS){s[m.id]=0;why[m.id]=[];}
 const add=(id,n,r)=>{s[id]+=n;if(n>0&&r)why[id].push(r);};
 if(where==='onprem'||where==='private'){add('vm',2,'runs in your own data centre');add('kubernetes',2,'OpenShift or Rancher run well on-premises');add('paas',-3);add('serverless',-3);add('static',-1);add('lowcode',-2);}
 if(where==='edge'){add('onsite',4,'it runs on site');add('containers',1,'containers are an easy way to ship to devices');add('paas',-3);add('serverless',-2);}
 if(where==='public'){add('paas',1,'public cloud offers managed platforms');add('containers',1,'public cloud offers managed containers');}
 if(where==='multicloud'){add('kubernetes',3,'Kubernetes runs the same way on every cloud');add('containers',1,'containers move between clouds');add('paas',-1);add('serverless',-1);}
 if(where==='hybrid'){add('kubernetes',2,'one way to run in the cloud and on-premises');add('vm',1,'virtual machines exist on both sides');}
 if(ten==='customer-hosted'){add('kubernetes',2,'customers can install it with a standard package');add('vm',2,'customers can install it on their own servers');add('paas',-2);add('serverless',-2);add('lowcode',-3);}
 if(ten==='saas-multi'){add('containers',2,'one shared deployment that scales with customers');add('kubernetes',1,'room to grow into many services');add('paas',1,'a managed platform keeps the operations team small');}
 if(ten==='saas-single'){add('kubernetes',2,'one deployment per customer is easier to repeat on a cluster');add('containers',1,'repeatable per-customer deployments');}
 if(ten==='internal'){add('paas',2,'internal apps rarely need more than a managed platform');}
 if(ten==='internal'&&has(/\b(forms?|approvals?|workflows?|requests?|sign-?off)\b/)){add('lowcode',2,'the brief is mostly forms and approvals');}
 if(has(/\b(power apps|power platform|dynamics|salesforce|servicenow|sharepoint|appsheet|mendix|outsystems)\b/)){add('lowcode',3,'the brief names a platform you already use');}
 if(peak>=500||users>=100000){add('kubernetes',1,'high load ('+(peak?peak+' requests/s':users.toLocaleString()+' users')+')');add('containers',2,'scales out quickly for high load');}
 else if(peak&&peak<5){add('serverless',2,'low load ('+peak+' requests/s): pay almost nothing when idle');add('paas',1,'a small plan covers this load');}
 if(has(/\b(spiky|campaigns?|seasonal|nightly|batch|scheduled|events?|webhooks?|uploads?|notifications?|reminders?)\b/)){add('serverless',2,'the brief has bursts, events or scheduled jobs');}
 if(ch.length===1&&ch[0]==='api'){add('containers',1,'an API-only product fits containers');add('serverless',1,'APIs can run as functions');add('static',-3);}
 if(ch.includes('web')&&ten!=='internal'&&where!=='onprem'&&where!=='edge'){add('static',2,'a public web front end loads fastest from a content network');}
 if(ch.includes('embedded')||has(/\b(sensors?|devices?|machines?|offline|no internet|plant floor)\b/)){add('onsite',2,'devices or poor connectivity');}
 if(has(/\b(legacy|existing server|windows server|licen[cs]ed|lift and shift|as it is)\b/)){add('vm',2,'existing software that expects its own server');}
 if(has(/\b(microservices?|many services|platform team)\b/)){add('kubernetes',2,'many services and a platform team');}
 const ranked=HOSTING_MODELS.map(m=>({id:m.id,score:s[m.id],why:why[m.id]})).filter(x=>x.score>0).sort((a,b)=>b.score-a.score);
 return ranked.slice(0,3);
}

function hostingModelPicker(){
 const P=productBrief(),D=P.deploy;
 const sug=hostingSuggest(P),rank=id=>sug.findIndex(x=>x.id===id);
 const set=id=>{const b=D.hosting||'';D.hosting=D.hosting===id?'':id;logInput('product:deploy:hosting',b,D.hosting);persist();render();};
 const chosen=HOSTING_MODELS.find(m=>m.id===D.hosting);
 const order=[...HOSTING_MODELS].sort((a,b)=>{const ra=rank(a.id),rb=rank(b.id);return (ra<0?9:ra)-(rb<0?9:rb);});
 return h('div',{class:'hosting-models'},[
  h('div',{class:'hosting-head'},[h('b',{text:'Hosting preference'}),
   h('small',{class:'muted',text:chosen?'Chosen: '+chosen.name+'. Click it again to clear.':sug.length?'Suggested from your brief: '+sug.map(x=>HOSTING_MODELS.find(m=>m.id===x.id).name).join(', ')+'. Pick one, or leave it for the design options to decide.':'Answer where it runs, tenancy and how it reaches users to get a suggestion. Pick one, or leave it for the design options to decide.'})]),
  h('div',{class:'hosting-grid'},order.map(m=>{const r=rank(m.id),on=D.hosting===m.id;
   return h('button',{type:'button',class:'hosting-card'+(on?' on':'')+(r===0?' top':''),'aria-pressed':on?'true':'false',onclick:()=>set(m.id)},[
    h('div',{class:'hosting-title'},[h('b',{text:m.name}),h('span',{class:'hosting-kind',text:m.kind}),r>=0?h('span',{class:'hosting-badge',text:r===0?'Best fit':'Also fits'}):null]),
    h('p',{text:m.what}),
    r>=0&&sug[r].why.length?h('p',{class:'hosting-why',text:'Why: '+sug[r].why.slice(0,3).join('; ')+'.'}):null,
    h('dl',{},[h('dt',{text:'Good for'}),h('dd',{text:m.good}),h('dt',{text:'Watch'}),h('dd',{text:m.watch}),
     h('dt',{text:'You still run'}),h('dd',{text:m.run}),h('dt',{text:'Scales'}),h('dd',{text:m.scale}),h('dt',{text:'Billed'}),h('dd',{text:m.bill}),
     h('dt',{text:'Azure'}),h('dd',{text:m.azure}),h('dt',{text:'AWS'}),h('dd',{text:m.aws}),h('dt',{text:'Google'}),h('dd',{text:m.gcp}),h('dt',{text:'IBM Cloud'}),h('dd',{text:m.ibm||'—'}),h('dt',{text:'Oracle Cloud'}),h('dd',{text:m.oci||'—'}),h('dt',{text:'On-premises'}),h('dd',{text:m.onprem})])]);}))]);
}

function hostingFacts(add){
 const P=M.product||{},D=P.deploy||{};
 const m=HOSTING_MODELS.find(x=>x.id===D.hosting);
 if(m)add('deployment','Hosting model chosen by the owner: '+m.name+' ('+m.kind+')');
 else{const s=hostingSuggest(P);if(s.length)add('deployment','Hosting model not chosen; the brief points to '+s.map(x=>HOSTING_MODELS.find(y=>y.id===x.id).name).join(', then ')+' (rule-based suggestion, not a decision)');}
}

(function hostingStyle(){
 if(document.getElementById('hosting-style'))return;
 const css=`.hosting-models{margin:12px 0}
.hosting-head{display:flex;flex-wrap:wrap;gap:4px 10px;align-items:baseline;margin-bottom:8px}
.hosting-grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(260px,1fr));gap:10px}
.hosting-card{display:flex;flex-direction:column;justify-content:flex-start;align-items:stretch;text-align:left;font:inherit;color:inherit;padding:10px 12px;border:1px solid #d9e1f2;border-radius:10px;background:#fff;cursor:pointer}
.hosting-card:hover{border-color:var(--accent)}.hosting-card.on{border:2px solid var(--accent);background:#f6f4ff}.hosting-card.top{box-shadow:0 0 0 1px #cfc6ff inset}
.hosting-title{display:flex;flex-wrap:wrap;gap:6px;align-items:center}.hosting-kind{font-size:11px;color:var(--muted)}
.hosting-badge{font-size:11px;padding:1px 6px;border-radius:8px;background:#ece9ff;color:#5a46c8}
.hosting-card p{margin:6px 0;font-size:13px}.hosting-why{color:#5a46c8}
.hosting-card dl{display:grid;grid-template-columns:auto 1fr;gap:2px 8px;margin:6px 0 0;font-size:12px}.hosting-card dt{color:var(--muted)}.hosting-card dd{margin:0}`;
 document.head.append(Object.assign(document.createElement('style'),{id:'hosting-style',textContent:css}));
})();
