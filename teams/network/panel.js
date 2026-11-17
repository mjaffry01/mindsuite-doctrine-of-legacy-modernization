/* MindSuite web app · network edge & partners agent team (built with teams/shared/team-panel.js).
   Scope = the "Agent detects" line of the Network edge & partners card: load balancers, API gateways,
   CDN and DNS entries, TLS certificates and expiry, partner connections (SFTP, EDI, VPN), firewall openings. */
const NETWORK_TEAM=makeTeamPanel({
 team:'network',key:'net',name:'Network edge & partners',people:'your network team',category:'integration',source:'network',
 labels:{orch:'Edge orchestrator',nrd:'Edge reader',ent:'Load balancers & gateways',dns:'DNS & CDN',tls:'TLS certificates',prt:'Partner connections',fw:'Firewall openings',nmea:'Edge map',nass:'Cut-over plan',ver:'Verifier',rep:'Report writer',people:'Your experts'},
 accept:'.txt,.cfg,.conf,.json,.csv,.zone',exportRe:/\.(txt|cfg|conf|json|csv|tsv|zone|ya?ml)$/i,
 consent:'Allow the AI agent to send entry-point and partner names, hosts and protocols (no secrets) to Claude (Anthropic)',
 privacyOn:'With this on, the edge-map agent sends entry-point and partner names, host names and protocols; the verifier and report writer send findings. Never passwords or VPN keys. Everything sent is listed in the result.',
 privacyOff:'Nothing leaves the backend machine: 9 rule-based agents run; who uses which entry is listed as a gap.',
 give:'Give the exports from Record evidence → Network edge & partners → How to extract: nginx -T or HAProxy configuration, kubectl ingress JSON, the DNS zone file or a DNS record CSV (name, type, value, ttl), openssl certificate output, the partner list (partner, host, port, protocol, direction, owner), the firewall rule export (name, source, destination, port, protocol, action, description) and the cloud inventory.',
 intro:'11 agents study how users and partners reach the system, from configuration and inventory exports, in the same 9 steps as the other teams: load balancers and gateways, DNS and CDN, TLS certificates and expiry, partner connections (SFTP, EDI, VPN) and firewall openings. If you allow it, Claude describes who comes in through which entry; the cut-over plan lists what must change and when.',
 skippedText:'Who reaches the system through which entry was not described',
 keepData:['kind','name','hosts','backends','tls','partner','protocol','host','port','direction','severity','items','rules','allowed','notAfter'],
 metrics:(A,src,F)=>[['Entry points',src.entryPoints??0],['DNS records',src.dnsRecords??0],['Certificates',src.certificates??0],['Partners',src.partners??0],['Firewall rules',src.firewallRules??0]],
 sections:(A,F,list)=>{const ap=F('migration-approach')[0],or=F('migration-order')[0];return [
  ap?h('div',{class:'validate-card'},[h('b',{text:'Cut-over plan'}),h('p',{},[h('b',{text:ap.title}),' — '+ap.detail]),or?h('pre',{class:'agent-pre',text:or.detail}):null]):null,
  list('Load balancers, gateways and ingress',F('entry-point').concat(F('plain-http'),F('old-tls'),F('single-backend')),true),
  list('DNS and CDN',F('dns-records').concat(F('cdn'),F('long-ttl'),F('dns-to-entry')),false),
  list('TLS certificates',F('certificate-issue').concat(F('certificate')),true),
  list('Partner connections',F('partner').concat(F('partner-unencrypted'),F('partner-no-owner'),F('edge-flow')),true),
  list('Firewall openings',F('firewall-summary').concat(F('wide-rule'),F('exposed-admin-port'),F('rule-no-description'),F('secret')),true)];},
 importTo:(A,{R,put,F,by})=>{
  const proto=p=>/SFTP|FTP|FTPS|SCP/.test(p)?'file':/AS2|EDI/.test(p)?'file':/HTTP|API|REST/.test(p)?'rest':/MQ/.test(p)?'mq':'other';
  put('interfaces',[...F('partner').map(f=>{const d=f.data||{};return R(f.id,by(f),{name:f.title,partner:d.partner||'',direction:d.direction==='in'?'in':d.direction==='out'?'out':'both',protocol:proto(d.protocol||''),context:'',contract:'unknown',frequency:''});}),
   ...F('entry-point').map(f=>{const d=f.data||{};return R(f.id,by(f),{name:'Entry: '+f.title,partner:(d.hosts||[]).join(', '),direction:'in',protocol:'rest',context:'',contract:'unknown',frequency:''});})]);},
 risks:{'certificate-issue':['security','Renew or replace the certificate'],'plain-http':['security','Redirect to HTTPS'],'old-tls':['security','Allow TLS 1.2 and 1.3 only'],'single-backend':['capacity','Add a second backend'],
  'partner-unencrypted':['security','Move the partner to SFTP, FTPS, AS2 or HTTPS'],'partner-no-owner':['coupling','Name an owner who will contact the partner'],'wide-rule':['security','Narrow the rule'],'exposed-admin-port':['security','Close it or restrict it to a jump host / VPN'],
  'long-ttl':['data','Lower the TTL before the cut-over'],secret:['security','Remove from shared files and rotate']}});
