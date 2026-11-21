/* MindSuite web app · Capability agent (First-release scope)
   Drafts capabilities from the product brief with Claude (POST /api/brief/capabilities on the project backend).
   Pressing "Suggest capabilities" is the consent for one request; the panel says exactly what is sent:
   the project name, the project type, the brief's narrative answers and the names of capabilities already listed.
   Each draft quotes the brief text it came from; the backend drops drafts whose quote is not in the brief.
   Drafts are kept in M.capabilityDrafts until the owner adds or dismisses them; nothing is added on its own.
   Hook: releaseScopePage() in assets/project-workflows.js calls capabilityAgentPanel(). */

const capAgentUI={busy:false,message:''};
const CAP_FIELDS=['problem','users','journeys','mvp','success','integrations','buyPref'];
const CAP_FIELD_LABEL={problem:'Problem',users:'Users and their jobs',journeys:'Key journeys',mvp:'First release',success:'Success measures',integrations:'Integrations',buyPref:'Build, buy or partner',project:'Project name'};

function capDrafts(){return M.capabilityDrafts||(M.capabilityDrafts=[]);}
function capRequest(){
 const P=productBrief(),answers={};
 for(const k of CAP_FIELDS){const v=String(P[k]||'').trim();if(v)answers[k]=v.slice(0,600);}
 const existing=[...(M.inventory.contexts||[]).map(c=>c.name),...capDrafts().map(d=>d.name)].filter(Boolean).map(n=>String(n).slice(0,120)).slice(0,60);
 return {project:(M.project||'').trim().slice(0,180),mode:typeof modeOf==='function'&&modeOf()==='extend'?'extend':'new',answers,existing};
}
async function runCapabilityAgent(){
 if(capAgentUI.busy)return;
 const input=capRequest();
 if(!['problem','users','journeys','mvp'].some(k=>input.answers[k])){capAgentUI.message='Describe the problem, users, key journeys or first release in the product brief first.';render();return;}
 capAgentUI.busy=true;capAgentUI.message='The Capability agent is reading the brief…';render();
 const started=Date.now();
 try{
  const out=await agentFetch('/api/brief/capabilities',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(input)});
  const list=(out.capabilities||[]).map(c=>({...c,id:'capd-'+crypto.randomUUID(),model:out.model}));
  capDrafts().push(...list);
  logInput('capability-agent:drafts',null,list.map(c=>c.name).join('; ')+' ('+(Date.now()-started)+' ms, '+(out.attempts||1)+' attempt'+((out.attempts||1)>1?'s':'')+')');
  capAgentUI.message=list.length?list.length+' capabilities suggested. Add the ones you agree with.':'The brief does not support any new capabilities yet. Add more about users, journeys or the first release.';
  persist();
 }catch(e){
  capAgentUI.message=/404|not found/i.test(e.message)?'This backend needs the Capability agent update. Restart the project backend with the latest build.':'The Capability agent could not finish. '+e.message;
 }finally{capAgentUI.busy=false;render();}
}
function acceptCapabilityDraft(d){
 ensureDiscovery();
 const rec={id:'rec-'+crypto.randomUUID(),name:d.name,description:d.description,releaseScope:d.releaseScope==='later'?'later':'first',
  cite:'Product brief · '+(CAP_FIELD_LABEL[d.quoteFrom]||d.quoteFrom)+': "'+d.quote+'"',status:'confirmed',method:'capability-agent',origin:'agent',model:d.model,evidence:[]};
 M.inventory.contexts.push(rec);
 M.capabilityDrafts=capDrafts().filter(x=>x!==d);
 if(typeof logRecord==='function')logRecord('contexts',rec,'add',null);
 persist();render();
}
function dismissCapabilityDraft(d){M.capabilityDrafts=capDrafts().filter(x=>x!==d);logInput('capability-agent:dismiss',d.name,null);persist();render();}

function capabilityAgentPanel(){
 const drafts=capDrafts();
 return h('div',{class:'cap-agent'},[
  h('div',{class:'cap-agent-head'},[
   h('div',{},[h('b',{text:'Capability agent'}),
    h('p',{class:'muted',text:'Suggests capabilities from your product brief. Claude receives the project name, project type, the brief’s written answers and the names of capabilities already listed, once per press. Each suggestion quotes the brief line it came from.'})]),
   h('div',{class:'cap-agent-actions'},[
    h('button',{type:'button',class:'btn small primary',disabled:capAgentUI.busy?'':null,onclick:runCapabilityAgent,text:capAgentUI.busy?'Suggesting…':drafts.length||(M.inventory.contexts||[]).length?'Suggest more capabilities':'Suggest capabilities'}),
    drafts.length>1?button('Add all '+drafts.length,()=>{for(const d of [...drafts])acceptCapabilityDraft(d);},'btn small'):null])]),
  capAgentUI.message?h('small',{role:'status',class:'muted',text:capAgentUI.message}):null,
  drafts.length?h('ul',{class:'scan-list capability-list cap-drafts'},drafts.map(d=>h('li',{},[
   h('div',{},[
    h('b',{text:d.name}),h('span',{class:'cap-badge',text:'AI suggestion'}),
    h('small',{text:(d.releaseScope==='later'?'Later release':'First release')+' · '+d.description}),
    h('small',{class:'muted',text:'From '+(CAP_FIELD_LABEL[d.quoteFrom]||d.quoteFrom)+': “'+d.quote+'” — '+d.reason})]),
   h('div',{class:'finding-actions'},[button('Add',()=>acceptCapabilityDraft(d),'btn small primary'),button('Dismiss',()=>dismissCapabilityDraft(d),'linkbtn')])]))):null]);
}

(function capAgentStyle(){
 if(document.getElementById('cap-agent-style'))return;
 const css=`.cap-agent{margin:8px 0 12px;padding:12px 14px;border:1px dashed #c9d3ea;border-radius:10px;background:#f8faff}
.cap-agent-head{display:flex;gap:12px;justify-content:space-between;align-items:flex-start;flex-wrap:wrap}.cap-agent-head p{margin:4px 0 0;max-width:640px;font-size:13px}
.cap-agent-actions{display:flex;gap:8px;align-items:center}
.cap-drafts li{background:#fff}.cap-drafts small{display:block}
.cap-badge{margin-left:8px;padding:1px 6px;border-radius:8px;background:#ece9ff;color:#5a46c8;font-size:11px}`;
 document.head.append(Object.assign(document.createElement('style'),{id:'cap-agent-style',textContent:css}));
})();
