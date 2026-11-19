/* MindSuite web app · shared
   Connection to the MindSuite agent backend (backend/), used by every agent team panel.
   Loaded by index.html before the main script; uses its helpers (h, button, M, persist, render, go, notice…) at call time. */
let AGENT_LIVE={log:[],snap:null,health:null,es:null,poll:null,timer:null};
function agentCfg(){if(!M.agentBackend)M.agentBackend={url:'http://127.0.0.1:8090',allowModel:false,git:true,maxFiles:12,path:''};return M.agentBackend;}
function agentBase(){return (agentCfg().url||'').trim().replace(/\/+$/,'');}
function agentURL(path){
 const url=new URL(agentBase()+path);
 const runId=/^\/api\/runs\/([^/?]+)/.exec(path)?.[1];
 if(path.startsWith('/api/evidence/') || (path.startsWith('/api/runs') && !M.legacyRunIds?.includes(runId))){
  url.searchParams.set('workspace',workspaceId);
  // The backend saves results in a folder named after the project, with one folder per agent team.
  if((M.project||'').trim())url.searchParams.set('project',M.project.trim().slice(0,200));
 }
 return url.href;
}
async function agentFetch(path,opts={}){
 const headers={...(opts.headers||{})};
 if(opts.method==='POST'){
  const health=await agentFetch('/api/health');
  if(!health.projectWorkspaces)throw new Error('Restart the agent backend with the current project files.');
  headers['X-MindSuite']='1';
 }
 const r=await fetch(agentURL(path),{cache:'no-store',...opts,headers});
 const j=await r.json().catch(()=>({}));
 if(!r.ok)throw new Error(j.error||('HTTP '+r.status));
 return j;
}
async function agentCheck(){try{AGENT_LIVE.health=await agentFetch('/api/health');notice('Agent backend connected'+(AGENT_LIVE.health.modelAvailable?' · Claude available':' · Claude not configured: rule-based agents only'));}catch(e){AGENT_LIVE.health={error:e.message};notice('Agent backend not reachable at '+agentBase()+'. Start it with: mindsuite-agents serve');}render();}
// When an agent fails, the run carries on without it. agentProblemsBox shows which agents did not finish, with a button to run each again.
const AGENT_STEP_TEXT={waiting:'Waiting',running:'Running…',done:'Done',partial:'Done with problems',skipped:'Skipped',people:'Your turn'};
function agentProblemsBox(A,follow){const ps=(A&&A.problems)||[];if(!ps.length)return null;
 return h('div',{class:'agent-problems',role:'alert'},[
  h('b',{text:ps.length===1?'One agent did not finish, so its findings are missing':ps.length+' agents did not finish, so their findings are missing'}),
  h('ul',{class:'plain-list'},ps.map(p=>h('li',{},[h('b',{text:p.name||p.agent}),' (step '+p.step+'): '+p.error+' ',
   p.canRerun?button('Run again',()=>agentRerun(A.id,p.part,follow),'btn small'):h('small',{class:'muted',text:'Run the team again to retry.'})]))),
  h('small',{class:'muted',text:'The other agents finished and their results are kept. The report opens with this warning and marks what was not checked.'})]);}
async function agentRerun(runId,part,follow){try{await agentFetch('/api/runs/'+runId+'/rerun?part='+encodeURIComponent(part),{method:'POST'});notice('Running the agent again…');follow(runId);}catch(e){notice('Could not run the agent again: '+e.message);}}
function agentChip(label,on,onchange){return h('label',{class:'type-chip'+(on?' on':'')},[h('input',{type:'checkbox',checked:on?'':null,onchange:e=>onchange(e.target.checked)}),label]);}
