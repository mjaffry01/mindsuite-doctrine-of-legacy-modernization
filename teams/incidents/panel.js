/* MindSuite web app · incidents & changes agent team (built with teams/shared/team-panel.js).
   Scope = the "Agent detects" line of the Incidents & changes card: where the system breaks most often,
   recurring problems, change frequency and failed changes. */
const INCIDENTS_TEAM=makeTeamPanel({
 team:'incidents',key:'inc',name:'Incidents & changes',people:'your service desk and problem manager',category:'operations',source:'incidents',
 labels:{orch:'Incidents orchestrator',ird:'Ticket reader',ihot:'Hotspots',irec:'Recurring problems',imttr:'Restore times',ichg:'Change frequency',ifail:'Failed changes',imea:'Failure story',iass:'Modernization priorities',ver:'Verifier',rep:'Report writer',people:'Your experts'},
 accept:'.csv',exportRe:/\.(csv|tsv)$/i,maxBytes:20*1024*1024,
 consent:'Allow the AI agent to send counts and cleaned ticket titles (no names, no descriptions) to Claude (Anthropic)',
 privacyOn:'With this on, the failure-story agent sends hotspots, counts and ticket titles grouped into problems, after people\'s names, e-mail addresses, addresses and numbers are removed; the verifier and report writer send findings. Descriptions, comments and the people columns are never read. Everything sent is listed in the result.',
 privacyOff:'Nothing leaves the backend machine: 9 rule-based agents run; which business functions are fragile is listed as a gap.',
 give:'Give 12 months of tickets for this system as CSV, from Record evidence → Incidents & changes → How to extract: the ServiceNow incident, change_request and problem lists (filtered by service or configuration item), or a Jira export. The number and short-description columns are needed; caller, assignee and description columns are ignored. Up to 20 MB per file.',
 intro:'11 agents read your incident, change and problem exports in the same 9 steps as the other teams, and find what the Incidents & changes card promises: where the system breaks most often, recurring problems, change frequency and failed changes. They also measure restore times, find the periods to avoid for the cut-over and rank what to fix first. People\'s names are never kept; tickets are cited by file and line.',
 skippedText:'Which business functions are fragile was not described',
 keepData:['component','incidents','p1','p2','count','components','problem','priority','medianHours','p90Hours','changes','perMonth','standard','normal','emergency','failed','rate','level','failRate','score','period','severity','months','from','to'],
 metrics:(A,src,F)=>[['Incidents',src.incidents??0],['Changes',src.changes??0],['Problems',src.problems??0],['Months',src.months??0],['Recurring problems',F('recurring').length]],
 sections:(A,F,list)=>{const cp=F('change-profile')[0],pa=F('pareto')[0];return [
  cp?h('div',{class:'validate-card'},[h('b',{text:cp.title}),h('p',{text:cp.detail})]):null,
  pa?h('div',{class:'validate-card'},[h('b',{text:pa.title}),h('p',{text:pa.detail})]):null,
  list('What to fix first',F('modernization-priority').concat(F('freeze-period'),F('data-window')),true),
  list('Where the system breaks',F('hotspot').concat(F('peak-period'),F('fragile-area')),true),
  list('Recurring problems',F('recurring').concat(F('no-problem-record'),F('reopened'),F('problem-records')),true),
  list('Restore times',F('restore-time').concat(F('stale-incident')),false),
  list('Changes',F('change-frequency').concat(F('change-failure-rate'),F('change-caused-incident'),F('emergency-changes'),F('weekend-changes'),F('secret')),true)];},
 importTo:()=>{},
 risks:{recurring:['tests','Find the root cause and fix it before the move'],'no-problem-record':['knowledge','Open a problem record and name an owner'],
  'change-failure-rate':['tests','Automate tests and deployment; move in small steps with a way back'],'emergency-changes':['tests','Find why urgent fixes are needed so often'],
  'change-caused-incident':['tests','Test these components before each change'],'peak-period':['capacity','Keep the cut-over out of this period'],
  'stale-incident':['knowledge','Close or hand over before the move'],'fragile-area':['tests','Redesign or stabilise before moving it'],secret:['security','Remove from the ticket and rotate']}});
