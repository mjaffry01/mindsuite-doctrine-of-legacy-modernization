/* MindSuite Pathfinder · cloud services catalogue and rules (Rules & people → Technology rules)
   The cloud area above sets rules per provider; this panel sets rules per SERVICE, for Azure, AWS,
   Google Cloud, IBM Cloud, Oracle Cloud and private / on-premises, grouped by category.
   Each service has a role that matches the technical plan (assets/tech-plan.js, TP_SVC keys): vm, paas,
   ctr, k8s, fn, static, postgres, mysql, sqlserver, oracle, nosql, redis, blob, search, tsdb, analytics,
   queue, stream, events, apigw, ipaas, workflow, waf, cdn, lb, dns, firewall, ddos, private, idStaff,
   idCust, vault, kms, posture, mon, cicd, iac, backup, dr, aiModels, aiSafety, ml, iotHub, iotEdge, video,
   notify.
   Rules are stored in M.decide.rules.services[provider][serviceId] = {state, reason}; the plan uses them
   through cloudServiceFor(role, provider, defaultName): Standard first, then Preferred, and an Excluded
   service is replaced by the first allowed service with the same role.
   Chips show "in your plan" when the System blueprint's shared platform names the service. */

const CLOUD_CATS=[['compute','Compute and hosting'],['data','Data and storage'],['messaging','Messaging and integration'],['edge','Edge and network'],
 ['identity','Identity, security and governance'],['monitor','Monitoring and observability'],['ops','Delivery, backup and recovery'],['ai','AI'],['iot','IoT and devices'],['comms','Email and messaging to people']];
// [id, name, category, role(s comma-separated), what it is]
const CLOUD_SERVICES={
 azure:[
  ['acr','Azure Container Registry','ops','registry','Private container images'],['appconfig','Azure App Configuration','ops','appconfig','Settings and feature flags'],
  ['azmon','Azure Monitor (metrics and alerts)','monitor','mon,metrics,alerting','Platform and custom metrics, alert rules and action groups'],
  ['loganalytics','Log Analytics (Azure Monitor Logs)','monitor','logs','Central log store and queries (KQL)'],
  ['appinsights','Application Insights','monitor','apm,synthetic','Application performance, traces, availability tests'],
  ['amp','Azure Monitor managed service for Prometheus','monitor','metrics','Managed Prometheus metrics for containers'],
  ['grafana','Azure Managed Grafana','monitor','dashboards','Managed Grafana dashboards'],['workbooks','Azure Workbooks','monitor','dashboards','Interactive reports over metrics and logs'],
  ['activity','Azure Activity Log','monitor','audit','Who changed what in the subscription'],['sentinel','Microsoft Sentinel','monitor','siem','Security monitoring (SIEM)'],
  ['netwatch','Azure Network Watcher','monitor','netmon','Network monitoring and flow logs'],['servicehealth','Azure Service Health','monitor','health','Azure incidents affecting you'],
  ['cost','Microsoft Cost Management','monitor','cost','Cost reports, budgets and alerts'],['advisor','Azure Advisor','monitor','advisor','Recommendations on cost, security and reliability'],
  ['chaos','Azure Chaos Studio','monitor','chaos','Fault injection tests'],['loadtest','Azure Load Testing','monitor','loadtest','Load tests'],
  ['vnet','Azure Virtual Network','edge','network','Private network'],['nat','Azure NAT Gateway','edge','nat','Outbound internet for private resources'],['vpngw','Azure VPN Gateway','edge','vpn','Site-to-site VPN'],['er','Azure ExpressRoute','edge','directconnect','Private line to your sites'],['bastion','Azure Bastion','edge','bastion','Secure admin access to servers'],['policy','Azure Policy','identity','governance','Guardrails on what may be created'],
  ['vm','Azure Virtual Machines','compute','vm','Virtual servers'],['vmss','Virtual Machine Scale Sets','compute','vm','Groups of identical VMs that scale'],
  ['appsvc','Azure App Service','compute','paas','Managed web app platform'],['aca','Azure Container Apps','compute','ctr','Serverless containers'],
  ['aks','Azure Kubernetes Service (AKS)','compute','k8s','Managed Kubernetes'],['func','Azure Functions','compute','fn','Serverless functions'],
  ['swa','Azure Static Web Apps','compute','static','Static front ends with an API'],
  ['sqldb','Azure SQL Database','data','sqlserver','Managed SQL Server database'],['sqlmi','Azure SQL Managed Instance','data','sqlserver','Near-full SQL Server, managed'],
  ['pg','Azure Database for PostgreSQL (Flexible Server)','data','postgres','Managed PostgreSQL'],['mysql','Azure Database for MySQL (Flexible Server)','data','mysql','Managed MySQL'],
  ['cosmos','Azure Cosmos DB','data','nosql','Document and multi-model database'],['redis','Azure Managed Redis','data','redis','Managed cache'],
  ['blob','Azure Blob Storage','data','blob','Object storage'],['search','Azure AI Search','data','search,vector','Search and vector search'],
  ['adx','Azure Data Explorer','data','tsdb','Time-series and log analytics'],['fabric','Microsoft Fabric','data','analytics','Analytics and reporting platform'],
  ['sb','Azure Service Bus','messaging','queue','Message queues and topics'],['eh','Azure Event Hubs','messaging','stream','Event streaming (Kafka API)'],
  ['eg','Azure Event Grid','messaging','events','Event routing'],['apim','Azure API Management','messaging','apigw','API gateway and portal'],
  ['logic','Azure Logic Apps','messaging','ipaas,workflow','Integration workflows'],
  ['afd','Azure Front Door','edge','cdn,waf','Global entry, CDN and WAF'],['appgw','Azure Application Gateway','edge','lb,waf','Regional load balancer with WAF'],
  ['lb','Azure Load Balancer','edge','lb','Network load balancer'],['dns','Azure DNS','edge','dns','DNS hosting'],['fw','Azure Firewall','edge','firewall','Managed network firewall'],
  ['ddos','Azure DDoS Protection','edge','ddos','DDoS protection'],['pl','Azure Private Link','edge','private','Private endpoints to services'],
  ['entra','Microsoft Entra ID','identity','idStaff','Staff identity and single sign-on'],['entraext','Microsoft Entra External ID','identity','idCust','Customer identity'],
  ['kv','Azure Key Vault','identity','vault,kms','Secrets, keys and certificates'],['defender','Microsoft Defender for Cloud','identity','posture','Security posture and threat protection'],
['devops','Azure DevOps','ops','cicd','Build and release pipelines'],
  ['gha','GitHub Actions','ops','cicd','Build and release pipelines'],['bicep','Bicep','ops','iac','Infrastructure as code'],
  ['backup','Azure Backup','ops','backup','Backups'],['asr','Azure Site Recovery','ops','dr','Disaster recovery to another region'],
  ['foundry','Azure AI Foundry (Azure OpenAI and other models)','ai','aiModels','Generative AI models'],['safety','Azure AI Content Safety','ai','aiSafety','AI guardrails'],
  ['aml','Azure Machine Learning','ai','ml','Train and run your own models'],
  ['iothub','Azure IoT Hub','iot','iotHub','Device connectivity and management'],['iotedge','Azure IoT Edge','iot','iotEdge','Run workloads on devices'],
  ['acs','Azure Communication Services','comms','notify','Email, SMS and chat']],
 aws:[
  ['ecr','Amazon ECR','ops','registry','Private container images'],['appconfig','AWS AppConfig','ops','appconfig','Settings and feature flags'],['scheduler','Amazon EventBridge Scheduler','compute','scheduler','Scheduled jobs'],
  ['cloudwatch','Amazon CloudWatch (metrics, alarms, dashboards)','monitor','mon,metrics,alerting,dashboards','Metrics, alarms and dashboards'],
  ['cwlogs','Amazon CloudWatch Logs','monitor','logs','Central logs and Logs Insights queries'],['xray','AWS X-Ray','monitor','apm','Distributed tracing'],
  ['appsignals','CloudWatch Application Signals','monitor','apm','Application performance and service objectives'],
  ['synthetics','CloudWatch Synthetics','monitor','synthetic','Scripted uptime and journey checks'],['rum','CloudWatch RUM','monitor','synthetic','Real-user monitoring in the browser'],
  ['amp','Amazon Managed Service for Prometheus','monitor','metrics','Managed Prometheus metrics'],['amg','Amazon Managed Grafana','monitor','dashboards','Managed Grafana dashboards'],
  ['adot','AWS Distro for OpenTelemetry','monitor','apm','OpenTelemetry collector and agents'],
  ['cloudtrail','AWS CloudTrail','monitor','audit','Who called which API and when'],['config','AWS Config','monitor','audit','Configuration history and compliance rules'],
  ['guardduty','Amazon GuardDuty','monitor','siem','Threat detection'],['flowlogs','VPC Flow Logs','monitor','netmon','Network traffic logs'],
  ['health','AWS Health Dashboard','monitor','health','AWS incidents affecting you'],['costexplorer','AWS Cost Explorer + AWS Budgets','monitor','cost','Cost reports, budgets and alerts'],
  ['fis','AWS Fault Injection Service','monitor','chaos','Fault injection tests'],['ssm','AWS Systems Manager','monitor','opsmgmt','Patching, inventory and run commands'],
  ['vpc','Amazon VPC','edge','network','Private network'],['natgw','NAT gateway','edge','nat','Outbound internet for private resources'],['vpn','AWS Site-to-Site VPN','edge','vpn','Site-to-site VPN'],['dx','AWS Direct Connect','edge','directconnect','Private line to your sites'],['tgw','AWS Transit Gateway','edge','network','Connects networks and accounts'],['orgs','AWS Organizations + AWS Control Tower','identity','governance','Accounts, guardrails and policies'],
  ['ec2','Amazon EC2','compute','vm','Virtual servers'],['asg','EC2 Auto Scaling','compute','vm','Groups of servers that scale'],
  ['eb','AWS Elastic Beanstalk','compute','paas','Managed web app platform'],['apprunner','AWS App Runner','compute','paas','Run web apps from code or containers'],
  ['ecs','Amazon ECS on AWS Fargate','compute','ctr','Serverless containers'],['eks','Amazon EKS','compute','k8s','Managed Kubernetes'],
  ['lambda','AWS Lambda','compute','fn','Serverless functions'],['amplify','AWS Amplify Hosting','compute','static','Static front ends'],
  ['rdspg','Amazon RDS for PostgreSQL','data','postgres','Managed PostgreSQL'],['aurora','Amazon Aurora (PostgreSQL or MySQL)','data','postgres,mysql','High-availability relational database'],
  ['rdsmysql','Amazon RDS for MySQL','data','mysql','Managed MySQL'],['rdssql','Amazon RDS for SQL Server','data','sqlserver','Managed SQL Server'],
  ['rdsoracle','Amazon RDS for Oracle','data','oracle','Managed Oracle Database'],
  ['dynamo','Amazon DynamoDB','data','nosql','Key-value and document database'],['docdb','Amazon DocumentDB','data','nosql','MongoDB-compatible documents'],
  ['elasticache','Amazon ElastiCache (Valkey or Redis)','data','redis','Managed cache'],['s3','Amazon S3','data','blob','Object storage'],
  ['opensearch','Amazon OpenSearch Service','data','search,vector','Search and vector search'],['timestream','Amazon Timestream','data','tsdb','Time-series database'],
  ['redshift','Amazon Redshift','data','analytics','Data warehouse'],
  ['sqs','Amazon SQS','messaging','queue','Message queues'],['sns','Amazon SNS','messaging','events,notify','Publish/subscribe and notifications'],
  ['msk','Amazon MSK','messaging','stream','Managed Kafka'],['kinesis','Amazon Kinesis Data Streams','messaging','stream','Event streaming'],
  ['eventbridge','Amazon EventBridge','messaging','events','Event routing'],['apigw','Amazon API Gateway','messaging','apigw','API gateway'],
  ['stepfn','AWS Step Functions','messaging','workflow','Workflows'],
  ['cloudfront','Amazon CloudFront','edge','cdn','CDN'],['waf','AWS WAF','edge','waf','Web application firewall'],['alb','Application Load Balancer','edge','lb','Load balancer'],
  ['route53','Amazon Route 53','edge','dns','DNS'],['netfw','AWS Network Firewall','edge','firewall','Network firewall'],['shield','AWS Shield','edge','ddos','DDoS protection'],
  ['privatelink','AWS PrivateLink','edge','private','Private endpoints to services'],
  ['idc','AWS IAM Identity Center','identity','idStaff','Staff single sign-on'],['cognito','Amazon Cognito','identity','idCust','Customer identity'],
  ['secrets','AWS Secrets Manager','identity','vault','Secrets'],['kms','AWS KMS','identity','kms','Encryption keys'],['securityhub','AWS Security Hub','identity','posture','Security posture'],
['codepipeline','AWS CodePipeline','ops','cicd','Build and release pipelines'],
  ['cfn','AWS CloudFormation','ops','iac','Infrastructure as code'],['cdk','AWS CDK','ops','iac','Infrastructure as code in programming languages'],
  ['backup','AWS Backup','ops','backup','Backups'],['drs','AWS Elastic Disaster Recovery','ops','dr','Disaster recovery'],
  ['bedrock','Amazon Bedrock','ai','aiModels','Generative AI models'],['guardrails','Amazon Bedrock Guardrails','ai','aiSafety','AI guardrails'],['sagemaker','Amazon SageMaker','ai','ml','Train and run your own models'],
  ['mediaconvert','AWS Elemental MediaConvert','ai','video','Video transcoding'],
  ['iotcore','AWS IoT Core','iot','iotHub','Device connectivity'],['greengrass','AWS IoT Greengrass','iot','iotEdge','Run workloads on devices'],
  ['ses','Amazon SES','comms','notify','Email sending']],
 gcp:[
  ['artifact','Artifact Registry','ops','registry','Private container images and packages'],['scheduler','Cloud Scheduler','compute','scheduler','Scheduled jobs'],['parameters','Parameter Manager','ops','appconfig','Settings'],
  ['cloudmonitoring','Cloud Monitoring (metrics, alerting, uptime checks)','monitor','mon,metrics,alerting,synthetic,dashboards','Metrics, alerting policies, uptime checks, dashboards'],
  ['cloudlogging','Cloud Logging','monitor','logs','Central logs and Log Analytics'],['trace','Cloud Trace','monitor','apm','Distributed tracing'],
  ['profiler','Cloud Profiler','monitor','apm','Continuous CPU and memory profiling'],['errorreporting','Error Reporting','monitor','apm','Groups application errors'],
  ['gmp','Google Cloud Managed Service for Prometheus','monitor','metrics','Managed Prometheus metrics'],
  ['auditlogs','Cloud Audit Logs','monitor','audit','Who did what and when'],['secops','Google Security Operations','monitor','siem','Security monitoring (SIEM)'],
  ['nic','Network Intelligence Center','monitor','netmon','Network monitoring and troubleshooting'],['psh','Personalized Service Health','monitor','health','Google Cloud incidents affecting you'],
  ['billing','Cloud Billing reports and budgets','monitor','cost','Cost reports, budgets and alerts'],
  ['vpc','Virtual Private Cloud (VPC)','edge','network','Private network'],['cloudnat','Cloud NAT','edge','nat','Outbound internet for private resources'],['cloudvpn','Cloud VPN','edge','vpn','Site-to-site VPN'],['interconnect','Cloud Interconnect','edge','directconnect','Private line to your sites'],['orgpolicy','Organization Policy Service','identity','governance','Guardrails on what may be created'],
  ['gce','Compute Engine','compute','vm','Virtual servers'],['mig','Managed instance groups','compute','vm','Groups of VMs that scale'],
  ['appengine','App Engine','compute','paas','Managed web app platform'],['run','Cloud Run','compute','ctr','Serverless containers'],
  ['gke','Google Kubernetes Engine (GKE)','compute','k8s','Managed Kubernetes'],['functions','Cloud Run functions','compute','fn','Serverless functions'],
  ['firebase','Firebase Hosting','compute','static','Static front ends'],
  ['cloudsqlpg','Cloud SQL for PostgreSQL','data','postgres','Managed PostgreSQL'],['alloydb','AlloyDB for PostgreSQL','data','postgres','High-performance PostgreSQL'],
  ['cloudsqlmysql','Cloud SQL for MySQL','data','mysql','Managed MySQL'],['cloudsqlss','Cloud SQL for SQL Server','data','sqlserver','Managed SQL Server'],
  ['firestore','Firestore','data','nosql','Document database'],['spanner','Spanner','data','postgres','Globally distributed relational database'],
  ['memorystore','Memorystore','data','redis','Managed cache'],['gcs','Cloud Storage','data','blob','Object storage'],
  ['vertexsearch','Vertex AI Search','data','search','Search'],['vectorsearch','Vertex AI Vector Search','data','vector','Vector search'],
  ['bigtable','Bigtable','data','tsdb','Wide-column and time-series data'],['bigquery','BigQuery','data','analytics,tsdb','Data warehouse'],
  ['pubsub','Pub/Sub','messaging','queue,stream','Messaging and event streaming'],['kafka','Managed Service for Apache Kafka','messaging','stream','Managed Kafka'],
  ['eventarc','Eventarc','messaging','events','Event routing'],['apigee','Apigee','messaging','apigw','API management'],['apigateway','API Gateway','messaging','apigw','Simple API gateway'],
  ['workflows','Workflows','messaging','workflow','Workflows'],
  ['clb','Cloud Load Balancing','edge','lb','Load balancing'],['armor','Cloud Armor','edge','waf,ddos','WAF and DDoS protection'],['cdn','Cloud CDN','edge','cdn','CDN'],
  ['mediacdn','Media CDN','edge','cdn','CDN for video'],['dns','Cloud DNS','edge','dns','DNS'],['ngfw','Cloud Next Generation Firewall','edge','firewall','Network firewall'],
  ['psc','Private Service Connect','edge','private','Private endpoints to services'],
  ['cloudidentity','Cloud Identity','identity','idStaff','Staff identity'],['identityplatform','Identity Platform','identity','idCust','Customer identity'],
  ['secret','Secret Manager','identity','vault','Secrets'],['kms','Cloud KMS','identity','kms','Encryption keys'],['scc','Security Command Center','identity','posture','Security posture'],
['build','Cloud Build','ops','cicd','Build pipelines'],
  ['deploy','Cloud Deploy','ops','cicd','Release pipelines'],['infra','Infrastructure Manager (Terraform)','ops','iac','Infrastructure as code'],
  ['backupdr','Backup and DR Service','ops','backup,dr','Backup and disaster recovery'],
  ['vertex','Vertex AI (Gemini and partner models)','ai','aiModels,ml','Generative AI and machine learning'],['modelarmor','Model Armor','ai','aiSafety','AI guardrails'],
  ['transcoder','Transcoder API','ai','video','Video transcoding'],
  ['mqttgke','MQTT broker on GKE (Google Cloud IoT Core was retired in 2023)','iot','iotHub','Device connectivity you run yourself']],
 ibm:[
  ['icr','IBM Cloud Container Registry','ops','registry','Private container images'],['appconfiguration','IBM Cloud App Configuration','ops','appconfig','Settings and feature flags'],
  ['monitoring','IBM Cloud Monitoring','monitor','mon,metrics,alerting,dashboards','Metrics, alerts and dashboards'],['logs','IBM Cloud Logs','monitor','logs','Central logs'],
  ['instana','IBM Instana Observability','monitor','apm,synthetic','Application performance, tracing and synthetic checks'],
  ['atracker','Activity Tracker Event Routing','monitor','audit','Audit events of account activity'],
  ['workload','Security and Compliance Center Workload Protection','monitor','siem','Threat detection for workloads'],
  ['flowlogs','Flow Logs for VPC','monitor','netmon','Network traffic logs'],['turbonomic','IBM Turbonomic','monitor','cost','Cost and performance optimisation'],
  ['status','IBM Cloud Status','monitor','health','IBM Cloud incidents'],
  ['vpc','IBM Cloud VPC','edge','network','Private network'],['pgw','Public Gateway','edge','nat','Outbound internet for private resources'],['vpnvpc','VPN for VPC','edge','vpn','Site-to-site VPN'],['directlink','Direct Link','edge','directconnect','Private line to your sites'],
  ['vsi','Virtual Servers for VPC','compute','vm','Virtual servers'],['powervs','Power Virtual Server','compute','vm','IBM Power servers (AIX, IBM i, Linux)'],
  ['codeengine','Code Engine','compute','ctr,paas,fn','Serverless apps, containers, jobs and functions'],['iks','IBM Cloud Kubernetes Service','compute','k8s','Managed Kubernetes'],
  ['roks','Red Hat OpenShift on IBM Cloud','compute','k8s','Managed OpenShift'],
  ['dbpg','Databases for PostgreSQL','data','postgres','Managed PostgreSQL'],['dbmysql','Databases for MySQL','data','mysql','Managed MySQL'],
  ['db2','Db2 on Cloud','data','sqlserver','Managed Db2 relational database'],['cloudant','Cloudant','data','nosql','Document database'],
  ['dbmongo','Databases for MongoDB','data','nosql','Managed MongoDB'],['dbredis','Databases for Redis','data','redis','Managed cache'],
  ['cos','Cloud Object Storage','data','blob','Object storage'],['dbopensearch','Databases for OpenSearch','data','search,vector','Search'],
  ['watsonxdata','watsonx.data','data','analytics','Data lakehouse'],
  ['mq','IBM MQ as a Service','messaging','queue','Message queues'],['eventstreams','Event Streams','messaging','stream','Managed Kafka'],
  ['apiconnect','API Connect','messaging','apigw','API management'],['appconnect','App Connect','messaging','ipaas','Integration flows'],
  ['cis','Cloud Internet Services','edge','cdn,waf,ddos,dns','Global entry: DNS, CDN caching, WAF and DDoS protection'],
  ['vpclb','Load Balancer for VPC','edge','lb','Load balancer'],['vpe','Virtual Private Endpoints','edge','private','Private endpoints to services'],
  ['transit','Transit Gateway','edge','firewall','Connects networks and regions'],
  ['verify','IBM Verify','identity','idStaff,idCust','Staff and customer identity'],['secrets','Secrets Manager','identity','vault','Secrets and certificates'],
  ['keyprotect','Key Protect','identity','kms','Encryption keys'],['scc','Security and Compliance Center','identity','posture','Security posture and compliance'],

  ['cd','Continuous Delivery (Tekton)','ops','cicd','Build and release pipelines'],['schematics','Schematics (Terraform)','ops','iac','Infrastructure as code'],
  ['backup','Backup for VPC','ops','backup','Backups'],
  ['watsonxai','watsonx.ai','ai','aiModels,ml','Generative AI and machine learning'],['governance','watsonx.governance','ai','aiSafety','AI governance and guardrails'],
  ['satellite','IBM Cloud Satellite','iot','iotEdge','Run IBM Cloud services on site'],
  ['eventnotif','Event Notifications','comms','notify','Email, SMS and push notifications']],
 oci:[
  ['ocir','OCI Container Registry','ops','registry','Private container images'],
  ['monitoring','OCI Monitoring (metrics and alarms)','monitor','mon,metrics,alerting,dashboards','Metrics, alarms and dashboards'],
  ['logging','OCI Logging','monitor','logs','Central logs'],['loganalytics','Logging Analytics','monitor','logs','Log search and analytics'],
  ['apm','Application Performance Monitoring','monitor','apm,synthetic','Tracing, real-user and synthetic monitoring'],
  ['audit','OCI Audit','monitor','audit','Who called which API and when'],['opsinsights','Ops Insights','monitor','capacity','Capacity planning for databases and hosts'],
  ['stackmon','Stack Monitoring','monitor','metrics','Monitoring of applications and middleware'],['dbmgmt','Database Management','monitor','dbmon','Database performance monitoring'],
  ['flowlogs','VCN Flow Logs','monitor','netmon','Network traffic logs'],['costanalysis','Cost Analysis + Budgets','monitor','cost','Cost reports and budgets'],
  ['vcn','Virtual Cloud Network (VCN)','edge','network','Private network'],['natgw','NAT Gateway','edge','nat','Outbound internet for private resources'],['ipsec','Site-to-Site VPN','edge','vpn','Site-to-site VPN'],['fastconnect','FastConnect','edge','directconnect','Private line to your sites'],['bastion','OCI Bastion','edge','bastion','Secure admin access to servers'],['securityzones','Security Zones','identity','governance','Guardrails on what may be created'],
  ['compute','OCI Compute','compute','vm','Virtual servers'],['pools','Instance pools and autoscaling','compute','vm','Groups of servers that scale'],
  ['ci','Container Instances','compute','ctr,paas','Serverless containers'],['oke','OCI Kubernetes Engine (OKE)','compute','k8s','Managed Kubernetes'],
  ['functions','OCI Functions','compute','fn','Serverless functions'],
  ['adb','Autonomous Database','data','oracle','Self-managing Oracle Database'],['basedb','Base Database Service','data','oracle','Oracle Database on VMs'],
  ['pg','OCI Database with PostgreSQL','data','postgres','Managed PostgreSQL'],['heatwave','HeatWave MySQL','data','mysql','Managed MySQL with analytics'],
  ['nosql','OCI NoSQL Database','data','nosql','Key-value and document database'],['cache','OCI Cache','data','redis','Managed cache'],
  ['objectstorage','Object Storage','data','blob','Object storage'],['opensearch','Search with OpenSearch','data','search,vector','Search'],
  ['queue','OCI Queue','messaging','queue','Message queues'],['streaming','OCI Streaming','messaging','stream','Kafka-compatible streaming'],
  ['events','OCI Events','messaging','events','Event routing'],['apigw','OCI API Gateway','messaging','apigw','API gateway'],
  ['integration','Oracle Integration','messaging','ipaas,workflow','Integration flows'],
  ['lb','OCI Load Balancer','edge','lb','Load balancer'],['waf','OCI Web Application Firewall','edge','waf','Web application firewall'],
  ['dns','OCI DNS','edge','dns','DNS'],['netfw','OCI Network Firewall','edge','firewall','Network firewall'],['pe','Private endpoints','edge','private','Private access to services'],
  ['iam','OCI IAM identity domains','identity','idStaff,idCust','Staff and customer identity'],['vault','OCI Vault','identity','vault,kms','Secrets and keys'],
  ['cloudguard','Cloud Guard','identity','posture','Security posture'],

  ['devops','OCI DevOps','ops','cicd','Build and release pipelines'],['rm','Resource Manager (Terraform)','ops','iac','Infrastructure as code'],
  ['fsdr','Full Stack Disaster Recovery','ops','dr','Disaster recovery'],
  ['genai','OCI Generative AI','ai','aiModels','Generative AI models'],['datascience','OCI Data Science','ai','ml','Train and run your own models'],
  ['mediaflow','OCI Media Flow','ai','video','Video transcoding'],
  ['notifications','OCI Notifications','comms','notify','Notifications'],['email','OCI Email Delivery','comms','notify','Email sending']],
 private:[
  ['harbor','Harbor','ops','registry','Private container images'],
  ['prometheus','Prometheus + Alertmanager','monitor','mon,metrics,alerting','Metrics and alert routing'],['grafana','Grafana','monitor','dashboards','Dashboards'],
  ['loki','Grafana Loki or Elasticsearch (ELK)','monitor','logs','Central logs'],['tempo','Jaeger or Grafana Tempo','monitor','apm','Distributed tracing'],
  ['otel','OpenTelemetry Collector','monitor','apm','Collects traces, metrics and logs'],['zabbix','Zabbix or Nagios','monitor','metrics','Infrastructure monitoring'],
  ['splunk','Splunk','monitor','logs,siem','Log analytics and security monitoring'],['apmsaas','Datadog, Dynatrace or New Relic (SaaS)','monitor','apm,metrics,logs','Hosted observability platform'],
  ['pagerduty','PagerDuty or Opsgenie','monitor','alerting','On-call and alert routing'],['wazuh','Wazuh','monitor','siem','Open-source security monitoring'],
  ['uptime','Uptime Kuma or Grafana synthetic monitoring','monitor','synthetic','Uptime and journey checks'],['opencost','OpenCost or Kubecost','monitor','cost','Kubernetes cost monitoring'],
  ['vmware','VMware vSphere','compute','vm','Virtual servers'],['hyperv','Microsoft Hyper-V','compute','vm','Virtual servers'],
  ['openshift','Red Hat OpenShift','compute','k8s,ctr,paas','Kubernetes platform'],['rancher','Rancher (Kubernetes)','compute','k8s','Kubernetes management'],
  ['knative','Knative or OpenFaaS','compute','fn','Functions on Kubernetes'],
  ['postgres','PostgreSQL (with Patroni)','data','postgres','Relational database'],['mysql','MySQL (InnoDB Cluster)','data','mysql','Relational database'],
  ['sqlserver','SQL Server (Always On)','data','sqlserver','Relational database'],['oracledb','Oracle Database (Data Guard)','data','oracle','Relational database'],
  ['mongodb','MongoDB','data','nosql','Document database'],['redis','Redis or Valkey','data','redis','Cache'],['minio','MinIO or Ceph','data','blob','Object storage'],
  ['opensearch','OpenSearch or Elasticsearch','data','search,vector','Search'],['timescale','TimescaleDB or InfluxDB','data','tsdb','Time-series database'],
  ['rabbitmq','RabbitMQ','messaging','queue','Message queues'],['kafka','Apache Kafka','messaging','stream','Event streaming'],
  ['kong','Kong or Gravitee','messaging','apigw','API gateway'],
  ['f5','F5, NGINX or HAProxy','edge','lb,waf','Load balancing and WAF'],['akamai','Akamai or Cloudflare','edge','cdn,ddos','CDN and DDoS protection'],
  ['paloalto','Palo Alto or Fortinet firewalls','edge','firewall','Network firewall'],
  ['keycloak','Keycloak','identity','idCust,idStaff','Identity and single sign-on'],['ad','Active Directory / LDAP','identity','idStaff','Staff directory'],
  ['vault','HashiCorp Vault','identity','vault,kms','Secrets and keys'],
['gitlab','GitLab CI or Jenkins','ops','cicd','Build and release pipelines'],
  ['terraform','Terraform or Ansible','ops','iac','Infrastructure as code'],['veeam','Veeam or Commvault','ops','backup,dr','Backup and recovery'],
  ['vllm','Open-weight models on your GPUs (vLLM)','ai','aiModels','Generative AI models'],
  ['mosquitto','EMQX, HiveMQ or Mosquitto','iot','iotHub','MQTT broker'],['ffmpeg','FFmpeg workers','ai','video','Video transcoding']]};
const CLOUD_PROVIDERS=[['azure','Microsoft Azure'],['aws','Amazon Web Services'],['gcp','Google Cloud'],['ibm','IBM Cloud'],['oci','Oracle Cloud (OCI)'],['private','Private cloud / on-premises']];
const CS_UI={provider:''};

function csRules(){ensureDecide();const R=M.decide.rules;R.services=R.services||{};return R.services;}
function csProvider(cloud){return ['azure','aws','gcp','ibm','oci'].includes(cloud)?cloud:'private';}
/* The service the plan should use for one role on one provider, following the service rules. */
function cloudServiceFor(role,cloud,fallback){
 const prov=csProvider(cloud);
 if(typeof capResolveService!=='function')return fallback;
 const r=capResolveService(role,prov,fallback);
 // Excluded default: the plan's default names a service your rules exclude.
 let rules={};try{rules=csRules()[prov]||{};}catch(e){}
 const fb=String(fallback||'').toLowerCase();
 const exHit=(CLOUD_SERVICES[prov]||[]).find(sv=>sv[3].split(',').includes(role)&&((rules[sv[0]]||{}).state==='excluded')&&fb.includes(sv[1].toLowerCase().split(' (')[0].split(' + ')[0]));
 const rec={...r,shown:''};
 let out;
 if(r.status==='unresolved')out='Open decision: '+r.note;
 else if(r.status==='conflict')out=(r.name||fallback)+' (needs a decision: '+r.note+')';
 else if(r.status==='approval')out=r.name+' (needs approval)';
 else if(exHit)out=r.name+' (your rules exclude '+exHit[1]+')';
 else if(r.state==='standard'||r.state==='preferred')out=r.name;
 else out=fallback;
 rec.shown=out;
 try{(window.TP_USED=window.TP_USED||new Map()).set(prov+':'+role,rec);}catch(e){}
 return out;
}

function cloudServicesPanel(){
 const R=M.decide.rules,S=csRules();
 const pf=typeof platformEffective==='function'?platformEffective():{};
 const ruled=Object.entries(R.stance.cloud||{}).filter(([,v])=>v==='standard'||v==='preferred').map(([k])=>k).filter(k=>CLOUD_SERVICES[csProvider(k)]);
 const ap=typeof capApprovedProvider==='function'?capApprovedProvider():{id:csProvider(ruled[0]||'')};
 const prov=CS_UI.provider||ap.id||'';
 if(!prov)return h('div',{class:'stance-group cloud-services','data-area':'cloud-services'},[
  h('div',{class:'stance-group-head'},[h('b',{text:'Cloud services'}),h('small',{class:'muted',text:'No provider approved for this project yet ('+(ap.why||'')+'). Pick one to browse its services; browsing does not change your project.'})]),
  h('div',{class:'type-chips cs-providers'},CLOUD_PROVIDERS.map(([id,l])=>h('button',{type:'button',class:'type-chip',onclick:()=>{CS_UI.provider=id;render();}},[l])))]);
 const rules=S[prov]=S[prov]||{};
 // "In your plan": services the current plan resolved, by stable id (no text matching).
 try{window.TP_USED=new Map();systemBlueprint(decCtxs(),'current');}catch(e){}
 const usedIds=new Set([...((window.TP_USED&&window.TP_USED.values())||[])].filter(r=>r.prov===prov&&r.id&&r.status!=='unresolved').map(r=>r.id));
 const usedIn=id=>usedIds.has(id);
 const cnt=Object.values(rules).filter(r=>r&&r.state&&r.state!=='allowed').length;
 const setRule=(id,name)=>{const cur=rules[id]||{};const dialog=dialogShell('Cloud service · '+name,'Set your organization\'s rule for this service.');
  let state=cur.state||'allowed';const reason=h('input',{class:'field',placeholder:'Why (optional)',value:cur.reason||'',maxlength:'240'});
  const opts=h('div',{class:'type-chips'},RULE_STATES.filter(s=>s[0]!=='retiring').map(([k,l,sym,desc])=>h('button',{type:'button',class:'type-chip'+(state===k?' on':''),title:desc,onclick:e=>{state=k;[...opts.children].forEach(c=>c.classList.remove('on'));e.currentTarget.classList.add('on');}},[(sym?sym+' ':'')+l])));
  dialog.append(h('div',{class:'drawer-content'},[opts,h('label',{class:'context-control'},[h('span',{text:'Reason'}),reason]),
   h('div',{class:'cost-actions'},[button('Save rule',()=>{const b=cur.state||'allowed';if(state==='allowed'&&!reason.value.trim())delete rules[id];else rules[id]={state,reason:reason.value.trim()};logInput('cloud-service:'+prov+':'+id,b,state);persist();dialog.close();render();},'btn primary'),button('Cancel',()=>dialog.close(),'btn')])]));
  dialog.showModal();};
 return h('div',{class:'stance-group cloud-services','data-area':'cloud-services'},[
  h('div',{class:'stance-group-head'},[h('b',{text:'Cloud services'}),h('small',{class:'muted',text:'Rules for individual services, not just the provider. The plan uses your Standard service first, then Preferred, and replaces an Excluded one.'+(cnt?' · '+cnt+' rule'+(cnt>1?'s':''):'')})]),
  h('p',{class:'cs-browsing'},[h('b',{text:'Browsing: '}),(CLOUD_PROVIDERS.find(p=>p[0]===prov)||[,prov])[1],h('span',{class:'muted',text:' · approved for the project: '+(ap.id?(CLOUD_PROVIDERS.find(p=>p[0]===ap.id)||[,ap.id])[1]+' ('+ap.why+')':'none yet')+(prov!==ap.id?' · browsing does not change it':'')})]),
  h('div',{class:'type-chips cs-providers'},CLOUD_PROVIDERS.map(([id,l])=>h('button',{type:'button',class:'type-chip'+(prov===id?' on':''),onclick:()=>{CS_UI.provider=id;render();}},[l]))),
  ...CLOUD_CATS.map(([cat,title])=>{const list=(CLOUD_SERVICES[prov]||[]).filter(s=>s[2]===cat);if(!list.length)return null;
   return h('div',{class:'cs-cat'},[h('span',{class:'cs-cat-title',text:title}),h('div',{class:'stance-chips'},list.map(([id,name,,,desc])=>{const st=(rules[id]||{}).state||'allowed',used=usedIn(id);
    return h('button',{type:'button',class:'stance-chip '+st+(used?' cs-used':''),title:desc+(used?' · used in your plan':'')+((rules[id]||{}).reason?' · '+rules[id].reason:''),'aria-label':name+': '+desc+'. Rule: '+(RULE_STATE[st]?RULE_STATE[st][1]:st)+(used?'. Used in your plan':'')+'. Click to set the rule.',onclick:()=>setRule(id,name)},[(RULE_STATE[st]&&RULE_STATE[st][2]?RULE_STATE[st][2]+' ':'')+name,used?h('span',{class:'cs-dot',text:' • in your plan'}):null,h('small',{class:'cs-desc',text:desc})]);}))]);}),
  h('small',{class:'muted',text:'"In your plan" = named by the System blueprint for this project (Design options). Change the platform there; set rules here.'})]);
}

(function csStyle(){
 if(document.getElementById('cs-style'))return;
 const css=`.cloud-services{margin-top:14px;padding-top:12px;border-top:1px dashed #d9e1f2}.cs-providers{margin:6px 0 8px}
.cs-cat{margin:6px 0}.cs-cat-title{display:block;font-size:12.5px;font-weight:600;color:#5d6377;text-transform:uppercase;letter-spacing:.05em;margin:6px 0 4px}
.stance-chip.cs-used{border-color:#7a6ae0;box-shadow:0 0 0 1px #cfc6ff inset}.cloud-services .stance-chip{display:inline-flex;flex-direction:column;align-items:flex-start;text-align:left}.cs-desc{font-size:12px;color:#5d6377;font-weight:400}.cs-browsing{margin:4px 0 8px;font-size:14px}.cs-dot{color:#5a46c8;font-weight:600;font-size:12px}`;
 document.head.append(Object.assign(document.createElement('style'),{id:'cs-style',textContent:css}));
})();
