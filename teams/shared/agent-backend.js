/* MindSuite web app · shared
   Connection to the MindSuite agent backend (backend/), used by every agent team panel.
   Loaded by index.html before the main script; uses its helpers (h, button, M, persist, render, go, notice…) at call time. */
let AGENT_LIVE={log:[],snap:null,health:null,es:null,poll:null,timer:null};
function agentCfg(){if(!M.agentBackend)M.agentBackend={url:'http://127.0.0.1:8090',allowModel:false,git:true,maxFiles:12,path:''};return M.agentBackend;}
function agentBase(){return (agentCfg().url||'').trim().replace(/\/+$/,'');}
async function agentFetch(path,opts={}){const headers={...(opts.headers||{})};if(opts.method==='POST')headers['X-MindSuite']='1';const r=await fetch(agentBase()+path,{...opts,headers});const j=await r.json().catch(()=>({}));if(!r.ok)throw new Error(j.error||('HTTP '+r.status));return j;}
async function agentCheck(){try{AGENT_LIVE.health=await agentFetch('/api/health');notice('Agent backend connected'+(AGENT_LIVE.health.modelAvailable?' · Claude available':' · Claude not configured: rule-based agents only'));}catch(e){AGENT_LIVE.health={error:e.message};notice('Agent backend not reachable at '+agentBase()+'. Start it with: mindsuite-agents serve');}render();}
function agentChip(label,on,onchange){return h('label',{class:'type-chip'+(on?' on':'')},[h('input',{type:'checkbox',checked:on?'':null,onchange:e=>onchange(e.target.checked)}),label]);}
