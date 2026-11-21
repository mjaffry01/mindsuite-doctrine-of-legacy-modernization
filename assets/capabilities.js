/* MindSuite Pathfinder · capability catalogue (capability first, then provider)
   Every technology choice starts from a capability a person can recognise ("customers sign in",
   "store uploaded videos", "recover after an outage"). Each capability has a stable id, a plain job, everyday
   search words, the brief evidence that makes it needed, the plan role it maps to (TP_SVC / cloud-service
   roles), and per provider: the service or combination, who operates it, and one important limitation.
   When a provider has no close equivalent the entry says so and names a workable design (partner or
   self-run) — it is never filled with another provider's service.
   capStatus(capId) resolves the service for the approved provider against the organization's service rules
   and the platform decision, and returns one status: selected | required | suggested | approval | unresolved.
   Used by: the rules page (rules-overview.js), the plan (tech-plan.js tpSvc), the diagram and the HLD (hld.js),
   the cloud-services panel ("in your plan" by id) and the design export. Provider facts reflect public
   provider documentation as known on 2026-11-21; items marked check:true need confirming before a contract. */

const CAP_AREAS=[
 ['channels','User channels and application structure','Who uses a browser, mobile app, device or partner API, which clients share a backend, and what works offline.'],
 ['compute','Runtime and compute','What runs as an application, container, function, worker or scheduled job, and in which language.'],
 ['data','Data and content','Which data needs transactions, documents, files, cache, search, analytics or long-term archive.'],
 ['integration','APIs and integration','Which interactions need an immediate answer and which can go through a queue, event, webhook, batch or file.'],
 ['identity','Identity, authorization and secrets','How customers, staff, partners and services sign in, and where roles, tenants and keys are enforced.'],
 ['network','Networking and delivery','What is public or private: DNS, certificates, gateways, load balancing, CDN and private endpoints.'],
 ['ops','Operations and recovery','What is logged, measured and alerted on, who responds, and how much downtime and data loss is acceptable.'],
 ['delivery','Delivery and governance','How releases are tested, deployed and rolled back, and which regions, licences, costs and standards apply.'],
 ['specialist','Product-specific needs','Media, payments, IoT, AI, notifications and real-time updates when the brief calls for them.']];

const CAP_PROVIDERS=[['azure','Azure'],['aws','AWS'],['gcp','Google Cloud'],['oci','Oracle Cloud'],['ibm','IBM Cloud'],['private','Private / on-premises']];
const CAP_OPS={managed:'Fully managed: the provider runs it; you configure and pay per use',sized:'Managed: the provider runs it; you choose the size and the maintenance window',runtime:'Managed platform: the provider runs the servers; you ship and operate your code',self:'You run it: install, patch, scale and back it up',partner:'A partner product: contract, support and data handling with a third party',design:'A design choice in your own code; no cloud service decides it'};

/* Evidence: find where the brief supports a capability and quote it. */
function capEvidence(re){
 const P=M.product||{};const fields=[['problem','Problem'],['users','Users and their jobs'],['journeys','Key journeys'],['mvp','First release'],['success','Success measures'],['integrations','Integrations'],['buyPref','Build, buy or partner']];
 for(const [k,l] of fields){const t=String(P[k]||'');const m=t.match(re);if(m){const i=Math.max(0,m.index-30);return 'Brief › '+l+': “…'+t.slice(i,m.index+m[0].length+30).replace(/\s+/g,' ').trim()+'…”';}}
 for(const c of ((M.inventory&&M.inventory.contexts)||[])){const t=c.name+' '+(c.description||'');if(re.test(t))return 'Capability: “'+c.name+'”';}
 return '';
}
function capBriefFacts(){
 const P=M.product||{},D=P.deploy||{},ch=D.channels||[];
 let I={f:{},regs:[],avail:99.5,aSrc:'assumed'};try{I=techPlanInputs({id:'system',name:M.project||'',description:''});}catch(e){}
 return {P,D,ch,I};
}
// [id, area, label, job, search words, roles (plan keys), needWhen(F)->evidence|'' , provider details]
// provider detail: {s: service(s), o: CAP_OPS key, l: limitation, check?: true when it needs confirming}
const CAP_LIST=[
 ['web-client','channels','Web front end','Serves the browser app quickly to people anywhere.','website browser web portal pages customers browse','static',
  F=>F.ch.includes('web')?'Brief › How it reaches users: web browser':capEvidence(/\b(website|web app|portal|browser)\b/i),
  {azure:{s:'Azure Static Web Apps, or the app platform serving pages',o:'managed',l:'Static Web Apps suits single-page apps; server-rendered pages run on the app platform instead'},
   aws:{s:'AWS Amplify Hosting, or S3 + CloudFront',o:'managed',l:'Server-side rendering needs Amplify compute or the app platform'},
   gcp:{s:'Firebase Hosting, or Cloud Storage + Cloud CDN',o:'managed',l:'Dynamic pages run on Cloud Run behind it'}}],
 ['mobile-backend','channels','Shared backend for web and mobile','One API that the mobile app, the website and partners all call, so business rules live in one place.','mobile app api backend shared partners','apigw',
  F=>F.ch.includes('mobile')||F.ch.includes('api')?'Brief › How it reaches users: '+F.ch.filter(x=>['mobile','api','web'].includes(x)).join(', '):capEvidence(/\b(mobile app|app store|android|ios)\b/i),
  {azure:{s:'Azure API Management in front of the app',o:'managed',l:'Developer and Premium tiers differ a lot in price and network features'},
   aws:{s:'Amazon API Gateway in front of the app',o:'managed',l:'Per-request pricing adds up at very high volume'},
   gcp:{s:'Apigee, or API Gateway for simpler cases',o:'managed',l:'Apigee is powerful but priced for larger programmes; API Gateway is simpler'}}],
 ['offline-sync','channels','Offline use and sync','Lets the mobile app keep working without a connection and sync changes later.','offline no connection sync poor network field','',
  F=>capEvidence(/\b(offline|no (?:internet|connection|signal)|poor (?:network|connectivity)|sync)\b/i),
  {azure:{s:'',o:'design',l:'No managed offline-sync service: keep a local database in the app and sync through your API'},
   aws:{s:'AWS AppSync with Amplify DataStore',o:'managed',l:'Ties the app to AppSync and Amplify libraries'},
   gcp:{s:'Firestore with offline persistence',o:'managed',l:'Works when Firestore is the app\'s database; otherwise sync through your API'}}],
 ['app-runtime','compute','Application runtime','Runs the product\'s own code: the API and business rules.','run the app application server containers hosting compute','ctr',
  F=>'Every product that builds its own code needs a runtime',
  {azure:{s:'Azure Container Apps · Azure App Service · AKS',o:'runtime',l:'AKS gives most control but needs a platform team'},
   aws:{s:'Amazon ECS on Fargate · AWS App Runner · Amazon EKS',o:'runtime',l:'EKS gives most control but needs a platform team'},
   gcp:{s:'Cloud Run · App Engine · GKE',o:'runtime',l:'GKE gives most control but needs a platform team'}}],
 ['functions','compute','Event-driven functions','Short pieces of code that run when something happens, without servers to manage.','functions serverless triggers webhook handler','fn',
  F=>F.I.f.notify||F.I.f.payments?capEvidence(/\b(alerts?|notif\w*|webhooks?|payments?)\b/i):'',
  {azure:{s:'Azure Functions',o:'managed',l:'Cold starts on the Consumption plan; long jobs need another plan'},
   aws:{s:'AWS Lambda',o:'managed',l:'15-minute maximum run time'},
   gcp:{s:'Cloud Run functions',o:'managed',l:'Long jobs belong in Cloud Run jobs instead'}}],
 ['background-jobs','compute','Background and scheduled jobs','Runs slow or timed work outside the request: reminders, exports, clean-up, reconciliation.','background jobs scheduled cron nightly reminders reconciliation batch','scheduler',
  F=>F.I.f.payments||F.I.f.reports||F.I.f.notify?capEvidence(/\b(payments?|reports?|alerts?|notif\w*|reminders?)\b/i):'',
  {azure:{s:'Azure Container Apps jobs (or Azure Functions timers)',o:'managed',l:'Very large batch work suits Azure Batch'},
   aws:{s:'Amazon EventBridge Scheduler with ECS tasks or Lambda',o:'managed',l:'Large batch work suits AWS Batch'},
   gcp:{s:'Cloud Scheduler with Cloud Run jobs',o:'managed',l:'Large batch work suits Google Cloud Batch'}}],
 ['relational','data','Managed relational database','Keeps data that needs transactions and consistency: orders, payments, accounts.','database orders transactions sql postgres records','postgres',
  F=>'Most products keep their core records in a relational database'+(F.I.f.payments?' — '+capEvidence(/\b(orders?|payments?|bookings?)\b/i):''),
  {azure:{s:'Azure Database for PostgreSQL (Flexible Server) · Azure SQL Database',o:'sized',l:'Cross-region replicas and failover must be planned separately'},
   aws:{s:'Amazon RDS for PostgreSQL · Amazon Aurora',o:'sized',l:'Aurora costs more but fails over faster'},
   gcp:{s:'Cloud SQL for PostgreSQL · AlloyDB for PostgreSQL',o:'sized',l:'AlloyDB is faster for analytics but newer and pricier'}}],
 ['document','data','Document database','Stores flexible records such as menus, profiles or settings that change shape often.','documents json flexible catalogue profiles nosql','nosql',
  F=>capEvidence(/\b(catalogue|catalog|menus?|profiles?|flexible)\b/i)?'Optional: '+capEvidence(/\b(catalogue|catalog|menus?|profiles?)\b/i):'',
  {azure:{s:'Azure Cosmos DB',o:'managed',l:'Pricing in request units needs care'},
   aws:{s:'Amazon DynamoDB · Amazon DocumentDB',o:'managed',l:'DynamoDB needs access patterns designed up front'},
   gcp:{s:'Firestore',o:'managed',l:'Query patterns are more limited than SQL'}}],
 ['object-storage','data','Object and file storage','Keeps files: uploaded photos, videos, documents and exports.','files uploads photos images videos documents storage upload','blob',
  F=>F.I.f.files||F.I.f.video?capEvidence(/\b(uploads?|photos?|photograph\w*|images?|videos?|documents?|files?)\b/i):'',
  {azure:{s:'Azure Blob Storage',o:'managed',l:'Choose hot, cool or archive tiers per file type'},
   aws:{s:'Amazon S3',o:'managed',l:'Request and egress charges can exceed storage charges'},
   gcp:{s:'Cloud Storage',o:'managed',l:'Choose regional, dual-region or multi-region up front'}}],
 ['cache','data','Cache','Answers repeated reads fast and holds sessions, so the database is not overloaded.','cache fast reads sessions speed','redis',
  F=>(F.I.peak||0)>=50||F.I.f.search?'Peak '+Math.round(F.I.peak||0)+' requests/s'+(F.I.f.search?' and browsing/search in the brief':''):'',
  {azure:{s:'Azure Managed Redis',o:'sized',l:'Data in the cache can be lost; never keep the only copy there'},
   aws:{s:'Amazon ElastiCache (Valkey or Redis)',o:'sized',l:'Same: a cache, not a database'},
   gcp:{s:'Memorystore',o:'sized',l:'Same: a cache, not a database'}}],
 ['search','data','Search','Lets people find items by words, filters and spelling-tolerant matching.','search find browse filter catalogue restaurants products','search',
  F=>F.I.f.search?capEvidence(/\b(search|browse|catalogue|catalog|find)\b/i):'',
  {azure:{s:'Azure AI Search',o:'sized',l:'Index updates are near-real-time, not instant'},
   aws:{s:'Amazon OpenSearch Service',o:'sized',l:'You size and tune the cluster'},
   gcp:{s:'Vertex AI Search, or Elastic Cloud from Marketplace',o:'managed',l:'Vertex AI Search is AI-ranked; classic keyword search may suit Elastic better',check:true}}],
 ['analytics','data','Analytics and reporting','Runs reports and dashboards without slowing the live system.','reports dashboards analytics insights payouts kpi','analytics',
  F=>F.I.f.reports?capEvidence(/\b(reports?|dashboards?|analytics|insights?|payouts?)\b/i):'',
  {azure:{s:'Microsoft Fabric, or a read replica for small reports',o:'managed',l:'Start with a replica; add Fabric when reports grow'},
   aws:{s:'Amazon Redshift, or a read replica for small reports',o:'sized',l:'Start with a replica; add Redshift when reports grow'},
   gcp:{s:'BigQuery, or a read replica for small reports',o:'managed',l:'Pay per query: set limits'}}],
 ['archive','data','Retention and archival','Keeps old records and files cheaply for as long as rules require, then deletes them.','retention archive keep years delete records compliance','',
  F=>F.I.regs.length?'Regulations recorded: '+F.I.regs.slice(0,3).join(', '):capEvidence(/\b(retain|retention|archive|keep for)\b/i),
  {azure:{s:'Blob Storage cool/archive tiers with lifecycle rules',o:'managed',l:'Archive reads take hours'},
   aws:{s:'S3 Glacier storage classes with lifecycle rules',o:'managed',l:'Archive reads take minutes to hours'},
   gcp:{s:'Cloud Storage Coldline/Archive classes with lifecycle rules',o:'managed',l:'Minimum storage durations apply'}}],
 ['api-gateway','integration','API entry point','One front door for APIs: authentication, quotas, versioning and routing.','api gateway entry partners quotas rest','apigw',
  F=>F.ch.includes('api')||F.ch.includes('mobile')?'Brief › How it reaches users: '+F.ch.join(', '):'',
  {azure:{s:'Azure API Management',o:'managed',l:'Private networking needs the higher tiers'},
   aws:{s:'Amazon API Gateway',o:'managed',l:'REST and HTTP API types differ in features and price'},
   gcp:{s:'API Gateway (simple) · Apigee (full API management)',o:'managed',l:'Apigee for partner programmes; API Gateway for internal APIs'}}],
 ['queue','integration','Message queue','Holds work that does not need an immediate answer, so spikes do not overload the system.','queue background async later jobs spikes retry','queue',
  F=>F.I.f.payments||F.I.f.notify||F.I.f.video?capEvidence(/\b(payments?|webhooks?|alerts?|notif\w*|uploads?|orders?)\b/i):'',
  {azure:{s:'Azure Service Bus',o:'managed',l:'Premium tier needed for private networking'},
   aws:{s:'Amazon SQS',o:'managed',l:'FIFO queues have lower throughput'},
   gcp:{s:'Pub/Sub',o:'managed',l:'One service for queues and events; ordering needs ordering keys'}}],
 ['pubsub','integration','Publish/subscribe and event routing','Tells several parts that something happened (an order was paid) without them calling each other.','events notify subscribers order paid publish subscribe','events',
  F=>F.I.f.payments?capEvidence(/\b(orders?|payments?)\b/i):'',
  {azure:{s:'Azure Event Grid (or Service Bus topics)',o:'managed',l:'Event Grid is for routing, not long retention'},
   aws:{s:'Amazon EventBridge · Amazon SNS',o:'managed',l:'Keep event schemas versioned'},
   gcp:{s:'Eventarc · Pub/Sub',o:'managed',l:'Keep event schemas versioned'}}],
 ['streaming','integration','Event streaming','Keeps an ordered, replayable stream of events for analytics and many consumers.','streaming kafka events replay telemetry clickstream','stream',
  F=>F.I.f.iot?'Brief mentions devices or readings':'',
  {azure:{s:'Azure Event Hubs (Kafka API)',o:'managed',l:'Retention and throughput units drive cost'},
   aws:{s:'Amazon MSK (Kafka) · Amazon Kinesis Data Streams',o:'sized',l:'MSK needs Kafka skills'},
   gcp:{s:'Managed Service for Apache Kafka · Pub/Sub',o:'sized',l:'Pub/Sub is simpler when Kafka compatibility is not needed'}}],
 ['webhooks','integration','Webhooks to and from partners','Lets partners and providers tell you about events (payment captured) and you tell them.','webhooks callbacks partners notify payment provider','events',
  F=>F.I.f.payments||String(F.P.integrations||'').replace(/\b(company |corporate )?(single sign[- ]?on|sso|active directory|entra( id)?|okta|identity provider)\b/ig,'').replace(/[\s,.;:]+/g,'')?capEvidence(/\b(payment provider|webhooks?|partners?|integrations?)\b/i)||'Brief › Integrations':'',
  {azure:{s:'API Management to receive; Event Grid to deliver',o:'managed',l:'Verify signatures and process each call once'},
   aws:{s:'API Gateway to receive; EventBridge API destinations to deliver',o:'managed',l:'Verify signatures and process each call once'},
   gcp:{s:'API Gateway or Cloud Run to receive; Eventarc to deliver',o:'managed',l:'Verify signatures and process each call once'}}],
 ['file-exchange','integration','File exchange with partners','Swaps files with partners over SFTP when they cannot call an API.','sftp files exchange partner batch csv edi','mft',
  F=>capEvidence(/\b(sftp|edi|csv files?|file exchange|batch files?|ftp)\b/i),
  {azure:{s:'Blob Storage SFTP support',o:'managed',l:'Local users per storage account; plan the key management'},
   aws:{s:'AWS Transfer Family',o:'managed',l:'Charged per protocol per hour'},
   gcp:{s:'',o:'design',l:'No first-party managed SFTP: run an SFTP server on Cloud Run/GKE in front of Cloud Storage, or use a partner'}}],
 ['customer-signin','identity','Customers sign in','Sign-up, sign-in, password reset and social login for customers.','customers sign in login accounts sign up social','idCust',
  F=>F.I.f.customers?capEvidence(/\b(customers?|subscribers?|viewers?|members?|shoppers?)\b/i):'',
  {azure:{s:'Microsoft Entra External ID',o:'managed',l:'Custom branding and flows take configuration'},
   aws:{s:'Amazon Cognito',o:'managed',l:'Migrating users out later is hard'},
   gcp:{s:'Identity Platform',o:'managed',l:'Ties customer accounts to Google Cloud'}}],
 ['workforce-sso','identity','Staff sign in (single sign-on)','Staff and admins sign in with the company directory and MFA.','staff employees admin single sign-on sso mfa','idStaff',
  F=>F.I.f.staff?capEvidence(/\b(staff|admins?|administrators?|managers?|agents?|editors?)\b/i)||'Brief names staff roles':'',
  {azure:{s:'Microsoft Entra ID',o:'managed',l:'Conditional access needs Entra ID P1 or higher'},
   aws:{s:'AWS IAM Identity Center with your directory',o:'managed',l:'For apps, connect your own directory over SAML/OIDC'},
   gcp:{s:'Cloud Identity, or your directory over SAML/OIDC',o:'managed',l:'Many organizations keep Entra ID or Okta as the source'}}],
 ['service-identity','identity','Service identities','Lets parts of the system call each other and the database without passwords in config.','service accounts managed identity workload passwordless','',
  F=>'Every system with more than one part needs this',
  {azure:{s:'Managed identities',o:'managed',l:'Not every third-party tool supports them'},
   aws:{s:'IAM roles for ECS tasks, Lambda and EKS (IRSA / Pod Identity)',o:'managed',l:'Least-privilege policies take discipline'},
   gcp:{s:'Service accounts with Workload Identity',o:'managed',l:'Avoid downloadable service-account keys'}}],
 ['authorization','identity','Roles, entitlements and permissions','Decides what each person may see and do, such as a customer seeing only their own orders.','permissions roles entitlements access rights authorization','',
  F=>'Roles from the brief: '+((typeof tpRoles==='function'?tpRoles(F.P.users):[]).join(', ')||'to be named'),
  {azure:{s:'Roles in your app (Entra app roles); a policy service only if needed',o:'design',l:'No managed fine-grained authorization service'},
   aws:{s:'Roles in your app, or Amazon Verified Permissions (Cedar)',o:'managed',l:'Verified Permissions adds a call per check'},
   gcp:{s:'Roles in your app; IAM Conditions for Google Cloud resources',o:'design',l:'No managed app-level authorization service'}}],
 ['tenant-isolation','identity','Tenant isolation','Keeps each customer organization\'s data separate in a shared product.','tenants isolation multi-tenant saas separate customers','',
  F=>F.D.tenancy==='saas-multi'?'Brief › Tenancy: SaaS, multi-tenant':'',
  {azure:{s:'Tenant id on every row with row-level security (or a database per tenant)',o:'design',l:'Database-per-tenant isolates best but costs more'},
   aws:{s:'Tenant id on every row with row-level security (or a database per tenant)',o:'design',l:'Same trade-off'},
   gcp:{s:'Tenant id on every row with row-level security (or a database per tenant)',o:'design',l:'Same trade-off'}}],
 ['secrets','identity','Secrets and keys','Keeps passwords, API keys, certificates and encryption keys out of code.','secrets keys passwords certificates vault encryption','vault',
  F=>'Every product has secrets',
  {azure:{s:'Azure Key Vault',o:'managed',l:'Request limits per vault: cache secrets'},
   aws:{s:'AWS Secrets Manager + AWS KMS',o:'managed',l:'Charged per secret per month'},
   gcp:{s:'Secret Manager + Cloud KMS',o:'managed',l:'Customer-managed keys need separate setup'}}],
 ['audit','identity','Audit trail','Records who changed what and when, for investigations and regulators.','audit trail who changed logs compliance','audit',
  F=>F.I.regs.length?'Regulations recorded: '+F.I.regs.slice(0,3).join(', '):'Recommended for every product',
  {azure:{s:'Azure Activity Log + app audit logs in Log Analytics',o:'managed',l:'Default retention is short: export for long-term'},
   aws:{s:'AWS CloudTrail + app audit logs',o:'managed',l:'Data events cost extra'},
   gcp:{s:'Cloud Audit Logs + app audit logs',o:'managed',l:'Data access logs are off by default'}}],
 ['dns-certs','network','DNS and certificates','Gives the product its address and keeps HTTPS certificates valid.','domain dns certificates https tls address','dns',
  F=>'Every internet-facing product',
  {azure:{s:'Azure DNS; certificates managed by Front Door or Key Vault',o:'managed',l:'Plan who owns the domain'},
   aws:{s:'Amazon Route 53; AWS Certificate Manager',o:'managed',l:'Free public certificates only on AWS services'},
   gcp:{s:'Cloud DNS; Certificate Manager',o:'managed',l:'Plan who owns the domain'}}],
 ['cdn-waf','network','Edge protection and CDN','Puts content close to users and filters attacks before they reach the app.','cdn edge waf firewall ddos fast global protection','waf',
  F=>F.I.f.customers?'Customers reach the product over the internet':'',
  {azure:{s:'Azure Front Door with WAF',o:'managed',l:'Premium tier needed for private origins'},
   aws:{s:'Amazon CloudFront + AWS WAF (+ AWS Shield)',o:'managed',l:'WAF rules need tuning'},
   gcp:{s:'Cloud CDN / Media CDN + Cloud Armor behind Cloud Load Balancing',o:'managed',l:'WAF rules need tuning'}}],
 ['private-network','network','Private network and endpoints','Keeps the database, cache and secrets off the public internet.','private network endpoints vnet vpc no public','private',
  F=>'Recommended whenever data is stored',
  {azure:{s:'Azure Virtual Network + Private Link',o:'managed',l:'Private DNS zones must be planned'},
   aws:{s:'Amazon VPC + PrivateLink / VPC endpoints',o:'managed',l:'Endpoints are charged per hour per zone'},
   gcp:{s:'VPC network + Private Service Connect',o:'managed',l:'Plan IP ranges early'}}],
 ['site-connectivity','network','Private connection to your sites','Links the cloud to offices, data centres or partners without the public internet.','vpn private line expressroute direct connect office erp on-premises','vpn',
  F=>F.D.where==='hybrid'||F.D.where==='onprem'?'Brief › Where it runs: '+F.D.where:capEvidence(/\b(erp|sap|on-?prem\w*|mainframe|data cent(?:er|re))\b/i),
  {azure:{s:'Azure VPN Gateway or ExpressRoute',o:'managed',l:'ExpressRoute takes weeks to order'},
   aws:{s:'AWS Site-to-Site VPN or Direct Connect',o:'managed',l:'Direct Connect takes weeks to order'},
   gcp:{s:'Cloud VPN or Cloud Interconnect',o:'managed',l:'Interconnect takes weeks to order'}}],
 ['logs-metrics','ops','Logs, metrics and alerts','Shows what the system is doing and wakes someone up when it breaks.','monitoring logs metrics alerts on-call dashboards','metrics',
  F=>'Every product in production',
  {azure:{s:'Azure Monitor + Log Analytics + Application Insights',o:'managed',l:'Log ingestion is the main cost: filter early'},
   aws:{s:'Amazon CloudWatch (+ Logs, X-Ray / Application Signals)',o:'managed',l:'Log ingestion is the main cost: filter early'},
   gcp:{s:'Cloud Monitoring + Cloud Logging + Cloud Trace',o:'managed',l:'Log ingestion is the main cost: filter early'}}],
 ['backups','ops','Backups and restore tests','Keeps copies of data in another place and proves they restore.','backup restore copies recover data loss','backup',
  F=>'Every product that stores data',
  {azure:{s:'Azure Backup with a geo-redundant vault',o:'managed',l:'Test restores; a backup never restored is a hope'},
   aws:{s:'AWS Backup with cross-region copies',o:'managed',l:'Test restores'},
   gcp:{s:'Backup and DR Service',o:'managed',l:'Test restores'}}],
 ['disaster-recovery','ops','Recover after an outage','Brings the service back in another region within the agreed time.','outage disaster recovery region failover rto rpo downtime','dr',
  F=>'Recovery targets: '+(capRecovery().rto?'RTO '+capRecovery().rto:'RTO not set')+', '+(capRecovery().rpo?'RPO '+capRecovery().rpo:'RPO not set'),
  {azure:{s:'Azure Site Recovery; database geo-replicas; Front Door failover',o:'managed',l:'Region pairs decide where copies can go'},
   aws:{s:'AWS Elastic Disaster Recovery; cross-region replicas; Route 53 failover',o:'managed',l:'Second-region capacity must be reserved or tested'},
   gcp:{s:'Backup and DR Service; cross-region replicas; global load balancing',o:'managed',l:'Second-region capacity must be tested'}}],
 ['cicd','delivery','Build, test and release','Builds, tests and deploys every change the same way, and rolls back.','release deploy pipeline ci cd testing rollback','cicd',
  F=>'Every product that ships code',
  {azure:{s:'GitHub Actions or Azure DevOps; Azure Container Registry',o:'managed',l:'Choose one pipeline tool per organization'},
   aws:{s:'GitHub Actions or AWS CodePipeline; Amazon ECR',o:'managed',l:'Choose one pipeline tool per organization'},
   gcp:{s:'GitHub Actions or Cloud Build + Cloud Deploy; Artifact Registry',o:'managed',l:'Choose one pipeline tool per organization'}}],
 ['infra-code','delivery','Infrastructure as code and environments','Creates dev, test and production the same way from code, kept separate.','terraform bicep infrastructure environments dev test prod','iac',
  F=>'Every product with more than one environment',
  {azure:{s:'Bicep or Terraform; separate subscriptions per environment',o:'design',l:'Bicep is Azure-only'},
   aws:{s:'CloudFormation/CDK or Terraform; separate AWS accounts per environment',o:'design',l:'CloudFormation is AWS-only'},
   gcp:{s:'Terraform (Infrastructure Manager); separate projects per environment',o:'design',l:'Plan folders and projects early'}}],
 ['data-residency','delivery','Data residency and regions','Keeps production data in approved countries or regions.','region country residency data location india europe','',
  F=>String(F.D.regions||'').trim()?'Brief › Data must stay in: '+F.D.regions:(F.I.regs.length?'Regulations recorded: '+F.I.regs.slice(0,3).join(', '):''),
  {azure:{s:'Azure regions in the approved area (India: Central India, South India, West India)',o:'design',l:'Not every service is in every region',check:true},
   aws:{s:'AWS regions in the approved area (India: Mumbai, Hyderabad)',o:'design',l:'Not every service is in every region',check:true},
   gcp:{s:'Google Cloud regions in the approved area (India: Mumbai, Delhi)',o:'design',l:'Not every service is in every region',check:true}}],
 ['cost-control','delivery','Budgets and cost monitoring','Warns before spend goes over budget and shows which part costs what.','budget cost spend money alerts limits','costmon',
  F=>'Every product in the cloud',
  {azure:{s:'Microsoft Cost Management budgets',o:'managed',l:'Tag resources so costs can be split'},
   aws:{s:'AWS Budgets + Cost Explorer',o:'managed',l:'Tag resources so costs can be split'},
   gcp:{s:'Cloud Billing budgets + reports',o:'managed',l:'Label resources so costs can be split'}}],
 ['notifications','specialist','Email, SMS and push messages','Sends order updates, reminders and alerts to people.','notifications send email sms push messages alerts reminders','notify',
  F=>F.I.f.notify||F.I.f.customers?capEvidence(/\b(alerts?|notif\w*|emails?|sms|reminders?|track\w*)\b/i)||'Customers need updates':'',
  {azure:{s:'Azure Communication Services (email, SMS); Azure Notification Hubs (push)',o:'managed',l:'SMS availability varies by country',check:true},
   aws:{s:'Amazon SES (email); Amazon SNS / AWS End User Messaging (SMS, push)',o:'managed',l:'SMS needs sender registration in some countries',check:true},
   gcp:{s:'',o:'partner',l:'No first-party email or SMS: use Firebase Cloud Messaging for push and a partner (e.g. SendGrid, Twilio) for email and SMS'}}],
 ['realtime','specialist','Live updates to apps','Pushes changes to screens as they happen, such as order status or a live dashboard.','real-time live updates track status websocket push','',
  F=>capEvidence(/\b(track\w*|live|real-?time|status updates?)\b/i),
  {azure:{s:'Azure Web PubSub or Azure SignalR Service',o:'managed',l:'Connection limits per unit'},
   aws:{s:'API Gateway WebSocket APIs or AWS AppSync subscriptions',o:'managed',l:'Charged per connection minute and message'},
   gcp:{s:'',o:'design',l:'No dedicated managed WebSocket service: hold WebSockets on Cloud Run, or use Firestore listeners'}}],
 ['payments','specialist','Take payments','Charges cards and other methods without card data touching your servers.','payments pay checkout cards billing subscriptions refunds','',
  F=>F.I.f.payments?capEvidence(/\b(pay\w*|checkout|billing|subscriptions?|card)\b/i):'',
  {azure:{s:'',o:'partner',l:'Not a cloud service: an external payment provider (e.g. Stripe, Adyen, Razorpay) on any cloud'},
   aws:{s:'',o:'partner',l:'Not a cloud service: an external payment provider on any cloud'},
   gcp:{s:'',o:'partner',l:'Not a cloud service: an external payment provider on any cloud'}}],
 ['video-processing','specialist','Process uploaded videos','Turns an uploaded video into versions for every screen and connection speed.','video upload transcode encode process media','video',
  F=>F.I.f.video?capEvidence(/\b(videos?|upload\w*|stream\w*|transcod\w*)\b/i):'',
  {azure:{s:'',o:'partner',l:'No first-party transcoding since Azure Media Services retired (2024): use a partner video platform or FFmpeg workers on Container Apps'},
   aws:{s:'AWS Elemental MediaConvert',o:'managed',l:'Priced per output minute'},
   gcp:{s:'Transcoder API',o:'managed',l:'Priced per output minute'}}],
 ['video-delivery','specialist','Deliver video playback','Streams video to viewers smoothly and protects paid content.','playback stream watch drm protect delivery viewers','cdn',
  F=>F.I.f.video?capEvidence(/\b(play\w*|watch|stream\w*|viewers?)\b/i):'',
  {azure:{s:'Azure Front Door for delivery; packaging and DRM from a partner',o:'partner',l:'No first-party packaging or DRM: needs a partner'},
   aws:{s:'AWS Elemental MediaPackage + CloudFront; DRM keys from a partner (SPEKE)',o:'managed',l:'DRM licences still come from a partner'},
   gcp:{s:'Media CDN; packaging and DRM from a partner',o:'partner',l:'DRM licences come from a partner'}}],
 ['iot','specialist','Connect devices','Registers devices, receives their readings and sends commands securely.','devices sensors iot readings telemetry machines','iotHub',
  F=>F.I.f.iot?capEvidence(/\b(sensors?|devices?|readings?|iot|machines?)\b/i):'',
  {azure:{s:'Azure IoT Hub (+ Device Provisioning Service)',o:'managed',l:'Per-unit message quotas'},
   aws:{s:'AWS IoT Core',o:'managed',l:'Rules engine and messaging are charged per message'},
   gcp:{s:'',o:'self',l:'Google Cloud IoT Core was retired in 2023: run an MQTT broker (EMQX, HiveMQ) on GKE feeding Pub/Sub'}}],
 ['ai','specialist','AI features','Adds search, assistants, extraction or predictions using AI models.','ai assistant chatbot recommendations extraction predict','aiModels',
  F=>{let t='';try{const B=systemBlueprint(decCtxs(),'current');t=(B.platform.find(l=>l.id==='ai')||{}).t||'';}catch(e){}if(!t)return '';const m=/Where AI can help: ([^.]*)/.exec(t);return 'Optional, suggested by the plan: '+(m?m[1].split(' (')[0]:'AI features');},
  {azure:{s:'Azure AI Foundry (Azure OpenAI and other models)',o:'managed',l:'Model availability differs by region',check:true},
   aws:{s:'Amazon Bedrock',o:'managed',l:'Model availability differs by region',check:true},
   gcp:{s:'Vertex AI',o:'managed',l:'Model availability differs by region',check:true}}]];

const CAP_BY_ID=Object.fromEntries(CAP_LIST.map(c=>[c[0],c]));
function capRecovery(){const r=((M.product||{}).recovery)||{};return {rto:String(r.rto||'').trim(),rpo:String(r.rpo||'').trim()};}

/* Provider the project has approved: recorded platform decision, else a single Standard cloud rule. */
function capApprovedProvider(){
 let pf={};try{pf=platformEffective();}catch(e){}
 if(pf.decided&&pf.cloud)return {id:csProviderOf(pf.cloud),why:'recorded in the platform decision',firm:true};
 const R=(M.decide&&M.decide.rules)||{},std=Object.entries((R.stance||{}).cloud||{}).filter(([,s])=>s==='standard').map(([k])=>k);
 // A cloud the project has chosen on Design options (not yet recorded) is what the plan and diagram use,
 // so the capabilities follow it too; the rule state for that cloud is carried so it shows as needing approval.
 const pv0=((typeof platformRec==='function'&&platformRec().values)||{}).cloud;
 if(pv0){const rs=((R.stance||{}).cloud||{})[pv0]||'';const nm=(typeof optOf==='function'&&(optOf('cloud',pv0)||[])[1])||pv0;
  return {id:csProviderOf(pv0),why:'chosen on Design options (not recorded yet)'+(rs==='approval'?'; '+nm+' needs approval in your rules':'')+(std.length&&!std.includes(pv0)?'; your standard is '+std.map(k=>(typeof optOf==='function'&&(optOf('cloud',k)||[])[1])||k).join(', '):''),firm:false,ruleState:rs,pending:true};}
 if(std.length===1)return {id:csProviderOf(std[0]),why:'your organization standard',firm:true};
 if(std.length>1)return {id:'',why:'two cloud standards compete ('+std.join(', ')+')',firm:false,conflict:true};
 const pv=((platformRec&&platformRec().values)||{}).cloud;
 if(pv)return {id:csProviderOf(pv),why:'chosen in the platform decision (not recorded yet)',firm:false};
 return {id:'',why:'no provider approved yet',firm:false};
}
function csProviderOf(cloud){return ['azure','aws','gcp','ibm','oci'].includes(cloud)?cloud:cloud?'private':'';}

/* Resolve the service for a role on a provider against the service rules. */
function capResolveService(role,prov,fallback){
 const out={role,prov,id:'',name:'',state:'',status:'suggested',note:''};
 if(!role||!prov){out.status='unresolved';out.note=prov?'design choice, no service to pick':'no provider approved yet';return out;}
 const list=(typeof CLOUD_SERVICES!=='undefined'?CLOUD_SERVICES[prov]||[]:[]).filter(s=>s[3].split(',').includes(role));
 let rules={};try{rules=(csRules()[prov])||{};}catch(e){}
 const st=s=>(rules[s[0]]||{}).state||'allowed';
 if(!list.length){out.name=fallback||'';out.status=fallback?'suggested':'unresolved';out.note=fallback?'':'no verified mapping on this provider';return out;}
 const stds=list.filter(s=>st(s)==='standard');
 const allowed=list.filter(s=>!['excluded','retiring'].includes(st(s)));
 if(!allowed.length){out.status='unresolved';out.note='every candidate is excluded by your rules ('+list.map(s=>s[1]).join(', ')+')';return out;}
 let pick=stds[0]||list.find(s=>st(s)==='preferred');
 if(!pick){const fb=String(fallback||'').toLowerCase();const pos=s=>{const k=s[1].toLowerCase().split(' (')[0].split(' + ')[0];const i=fb?fb.indexOf(k):-1;return i<0?1e9:i;};
  // the service the plan names first wins (e.g. 'Amazon SES (email); Amazon SNS …' → SES)
  const named=allowed.filter(s=>pos(s)<1e9).sort((a,b)=>pos(a)-pos(b));pick=named[0]||allowed.find(s=>st(s)!=='approval')||allowed[0];}
 out.id=pick[0];out.name=pick[1];out.state=st(pick);
 if(stds.length>1){out.status='conflict';out.note='two standards compete: '+stds.map(s=>s[1]).join(' and ')+' — needs a decision';}
 else if(out.state==='standard')out.status='required';
 else if(out.state==='approval')out.status='approval';
 return out;
}

/* Status of a capability for this project. */
function capStatus(capId,provOverride){
 const c=CAP_BY_ID[capId];if(!c)return null;
 const F=capBriefFacts();const ev=(()=>{try{return c[6](F)||'';}catch(e){return '';}})();
 const ap=capApprovedProvider(),prov=provOverride||ap.id;
 const det=prov&&c[7][prov]?c[7][prov]:null;
 const pf=(()=>{try{return platformEffective();}catch(e){return {};}})();
 let role=c[5];
 if(capId==='app-runtime')role={kubernetes:'k8s',containers:'ctr',paas:'paas',serverless:'fn',vm:'vm',static:'static'}[pf.hosting]||'ctr';
 if(capId==='relational'&&['postgres','mysql','sqlserver','oracle','nosql'].includes(pf.database))role=pf.database;
 let res={status:'suggested',name:'',note:''};
 if(det&&!det.s){res={status:det.o==='partner'?'needs-partner':det.o==='design'?'design':'unresolved',name:'',note:det.l};}
 else if(role&&prov){const r=capResolveService(role,prov,det?det.s:'');res={status:r.status,name:r.name||(r.status==='unresolved'?'':(det?det.s:'')),note:r.note,id:r.id,state:r.state,role};}
 else if(det){res={status:'suggested',name:det.s,note:''};}
 else if(!prov){res={status:'suggested',name:'',note:'choose or approve a provider to see the service'};}
 else {res={status:'unresolved',name:'',note:'no verified mapping on this provider'};}
 // Selected only where the recorded platform decision itself names the choice (hosting → runtime, database → relational).
 const rv=(()=>{try{return platformRec().values||{};}catch(e){return {};}})();
 if(!provOverride&&ap.ruleState==='approval'&&res.status==='suggested'&&res.name){res.status='approval';res.note=(res.note?res.note+'; ':'')+'the chosen provider needs approval in your rules';}
 if(res.status==='suggested'&&pf.decided&&ap.firm&&((capId==='app-runtime'&&rv.hosting)||(capId==='relational'&&rv.database)))res.status='selected';
 const dismissed=((M.decide&&M.decide.capDismissed)||{})[capId],added=((M.decide&&M.decide.capAdded)||{})[capId];
 return {cap:c,evidence:ev,baseline:/^(Every |Recommended |Most products)/.test(ev),needed:(!!ev||!!added)&&!dismissed,dismissed:!!dismissed,prov,approved:ap,detail:det,...res};
}
const CAP_STATUS_LABEL={selected:'Selected',required:'Required by policy',suggested:'Suggested',approval:'Needs approval',unresolved:'Unresolved',conflict:'Needs a decision','needs-partner':'Partner needed',design:'Design choice'};

/* Search by business need, before knowing any vendor name. */
function capSearch(q){
 q=String(q||'').trim().toLowerCase();if(!q)return [];
 const words=q.split(/\s+/).filter(w=>w.length>2);
 return CAP_LIST.map(c=>{const hay=(c[2]+' '+c[3]+' '+c[4]+' '+Object.values(c[7]).map(d=>d.s||'').join(' ')).toLowerCase();
  const score=words.reduce((n,w)=>n+(hay.includes(w)?(c[4].includes(w)?2:1):0),0);return {c,score};}).filter(x=>x.score>0).sort((a,b)=>b.score-a.score).map(x=>x.c);
}
