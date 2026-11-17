/* MindSuite web app · batch & scheduling agent team (built with teams/shared/team-panel.js).
   Scope = the "Agent detects" line of the Batch & scheduling card: job streams and their order,
   schedules and batch windows, dependencies between jobs, failures and reruns. */
const BATCH_TEAM=makeTeamPanel({
 team:'batch',key:'batch',name:'Batch & scheduling',people:'your operations team',category:'operations',source:'batch',
 labels:{orch:'Batch orchestrator',bjr:'Job reader',flo:'Job streams & dependencies',win:'Schedules & batch window',run:'Failures & reruns',bsec:'Accounts & secrets',bmea:'Business meaning',bass:'Migration assessment',ver:'Verifier',rep:'Report writer',people:'Your experts'},
 accept:'.json,.jil,.txt,.csv,.sh,.ps1,.bat,.cmd,.py,.sql',exportRe:/\.(json|jil|txt|csv|tsv|sh|ps1|bat|cmd|py|sql|jcl)$/i,
 consent:'Allow the AI agent to send job names, descriptions and schedules (no passwords) to Claude (Anthropic)',
 privacyOn:'With this on, the business-meaning agent sends job names, descriptions, schedules and the order they run in; command lines with passwords are removed. Everything sent is listed in the result.',
 privacyOff:'Nothing leaves the backend machine: 8 rule-based agents run; the business process behind each stream is listed as a gap.',
 give:'Give the job definitions from Record evidence → Batch & scheduling → How to extract (Control-M JSON, AutoSys JIL, TWS composer, schtasks CSV, crontab per user, named crontab-<host>-<user>.txt) and, for failures and reruns, the run history as CSV (job, start, end, status). Add the job scripts to link jobs to their code.',
 intro:'10 agents study the batch from scheduler exports, in the same 9 steps as the other teams: job streams and their order, schedules and the batch window, dependencies between jobs, failures and reruns, accounts and passwords. If you allow it, Claude names the business process behind each stream; the assessment says which jobs move together, what to retire and in which order.',
 skippedText:'The business process behind each stream was not interpreted',
 keepData:['jobs','depth','window','severity','runs','failed','criticality'],
 metrics:(A,src,F)=>{const rs=F('run-summary')[0];return [['Jobs (disabled)',(src.jobs??0)+' ('+(src.disabled??0)+')'],['Schedulers',(src.schedulers||[]).length],['Job streams',F('job-stream').length],['Batch window',(F('batch-window')[0]?.data?.window)||'—'],['Failed runs',rs&&rs.data?rs.data.failed+' of '+rs.data.runs:'no history']];},
 sections:(A,F,list)=>{const ap=F('migration-approach')[0],or=F('migration-order')[0];return [
  ap?h('div',{class:'validate-card'},[h('b',{text:'Migration approach'}),h('p',{},[h('b',{text:ap.title}),' — '+ap.detail]),or?h('pre',{class:'agent-pre',text:or.detail}):null]):null,
  list('Job streams',F('job-stream').concat(F('critical-job'),F('missing-predecessor'),F('circular-wait')),true),
  list('Schedules and the batch window',F('batch-window').concat(F('daytime-batch'),F('frequent-jobs'),F('month-end-jobs')),true),
  list('Failures and reruns',F('run-summary').concat(F('failing-job'),F('long-job'),F('never-ran')),true),
  list('Business processes',F('batch-process').concat(F('batch-calendar')),true),
  list('Accounts, passwords and scripts',F('admin-job').concat(F('secret'),F('disabled-jobs'),F('job-script')),false),
  list('Target per scheduler',F('scheduler-target'),false)];},
 importTo:(A,{R,put,F,by})=>{
  put('components',F('job').filter(f=>f.data).slice(0,120).map(f=>R(f.id,by(f),{name:f.title,kind:'batch',context:'',tech:(f.data.scheduler||'')+(f.data.after&&f.data.after.length?' · after '+f.data.after.join(', '):''),size:'',schedule:f.data.schedule||''})));
  put('events',F('batch-process').map(f=>R(f.id,'business meaning · '+f.id,{name:f.title,context:'',consumers:[],trigger:f.detail},'inferred-code')));},
 risks:{'failing-job':['capacity','Find out why it fails before the move'],'long-job':['capacity','Check it still fits the window after the move'],'missing-predecessor':['coupling','Find the job or system it waits for'],'circular-wait':['data','Untangle the circular wait'],
  'critical-job':['coupling','Monitor it closely; many jobs wait for it'],'daytime-batch':['capacity','Check it does not slow users down'],'month-end-jobs':['data','Test month-end and year-end runs explicitly'],'admin-job':['security','Run with a least-privilege account'],secret:['security','Move the password to a vault and rotate it'],'never-ran':['data','Confirm, then retire'],'disabled-jobs':['data','Retire or explain']}});
