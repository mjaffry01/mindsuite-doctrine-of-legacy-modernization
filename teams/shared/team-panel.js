/* MindSuite web app · shared panel builder for agent teams
   A team folder (teams/<team>/panel.js) describes only what is its own: labels, accepted files, the
   result sections and how results go into the inventory. This file does the rest: connection, consent,
   file picking, sending to the backend, following the run live, keeping the result, report download.
   Loaded by index.html before the main script; uses its helpers (h, button, M, persist, render, go, notice…) at call time. */
function makeTeamPanel(cfg){
 const live={log:[],snap:null,es:null,poll:null,timer:null};
 const runKey=cfg.key+'AgentRun',idKey=cfg.key+'RunId',consentKey=cfg.key+'AllowModel',anchor=cfg.key+'-agent-run';
 const label=id=>cfg.labels[id]||id;
 const open=id=>{const w=window.open('teams/'+cfg.team+'/agents.html'+(typeof id==='string'?'#'+id:''),'mindsuite-'+cfg.team+'-agents','width=1240,height=900');if(w)w.focus();};
 const goPanel=()=>{showAgentTeam(anchor);};
 const options=()=>{const c=agentCfg();return {team:cfg.team,allowModel:!!c[consentKey],gitHistory:false,maxModelFiles:Math.max(1,Math.min(60,+c.maxFiles||12))};};
 const renderSoon=()=>{clearTimeout(live.timer);live.timer=setTimeout(()=>{const a=document.activeElement;if(screen==='analysis'&&!(a&&/^(INPUT|TEXTAREA|SELECT)$/.test(a.tagName)))render();},350);};
 const keep=s=>{const pick=d=>{if(!d)return undefined;const o={};for(const k of cfg.keepData||[])if(d[k]!=null)o[k]=Array.isArray(d[k])?d[k].slice(0,40):d[k];return o;};
  const fs=s.findings.filter(f=>f.status!=='rejected').slice(0,3000).map(f=>({id:f.id,agent:f.agent,kind:f.kind,module:f.module||'',title:f.title,detail:(f.detail||'').slice(0,900),status:f.status,verdict:f.verdict||'',cite:f.citations&&f.citations[0]?f.citations[0].file+(f.citations[0].line?':'+f.citations[0].line:''):'',data:pick(f.data)}));
  M[runKey]={id:s.run.id,url:agentBase(),at:new Date().toISOString(),status:s.run.status,error:s.run.error||'',problems:s.run.problems||[],steps:s.run.steps,source:s.source,counts:s.counts,rejected:s.findings.filter(f=>f.status==='rejected').length,findings:fs,questions:(s.questions||[]).map(q=>({from:q.from,ref:q.ref||'',text:q.text})),sent:s.sentToModel||[],report:(s.report||'').length<400000?s.report:'',imported:false};
  persist();const np=M[runKey].problems.length;notice(cfg.name+' agent run finished'+(np?' with problems: '+np+' agent'+(np===1?'':'s')+' did not finish':'')+': '+fs.length+' findings, '+M[runKey].questions.length+' questions for '+cfg.people+'.');};
 const refresh=async()=>{const c=agentCfg();if(!c[idKey])return;try{const s=await agentFetch('/api/runs/'+c[idKey]);live.snap=s;if(s.run.status!=='running'){clearInterval(live.poll);if(live.es)live.es.close();keep(s);}renderSoon();}catch(e){clearInterval(live.poll);notice('Lost contact with the agent backend: '+e.message);}};
 const follow=id=>{const c=agentCfg();c[idKey]=id;persist();if(live.es)live.es.close();clearInterval(live.poll);live.log=[];live.snap=null;
  try{const es=new EventSource(agentURL('/api/runs/'+id+'/events'));live.es=es;es.addEventListener('message',e=>{try{live.log.push(JSON.parse(e.data));if(live.log.length>300)live.log.shift();renderSoon();}catch(err){}});es.addEventListener('done',()=>{es.close();refresh();});es.onerror=()=>{};}catch(e){}
  live.poll=setInterval(refresh,2000);refresh();goPanel();};
 const send=async(files,kind)=>{const fd=new FormData();fd.append('options',JSON.stringify(options()));let sent=0,skipped=0;
  if(kind==='zip'){fd.append('zip',files[0],files[0].name);sent=1;}
  else{for(const f of files){const p=f.webkitRelativePath||f.name;if(/(^|\/)(node_modules|\.git|target|bin|obj|build|dist|vendor)\//.test('/'+p)||!cfg.exportRe.test(p)||f.size>(cfg.maxBytes||2*1024*1024)){skipped++;continue;}fd.append('file',f,p);sent++;}fd.append('root',(files[0].webkitRelativePath||'').split('/')[0]||cfg.team+'-export');}
  if(!sent){notice('None of the chosen files is a '+cfg.name.toLowerCase()+' export.');return;}
  notice('Sending '+(kind==='zip'?files[0].name:sent+' file'+(sent===1?'':'s')+(skipped?' ('+skipped+' other files left out)':''))+' to the '+cfg.name.toLowerCase()+' agents…');
  await stageAgentForm(fd);};
 const pick=kind=>{const i=document.createElement('input');i.type='file';if(kind==='folder'){i.webkitdirectory=true;i.multiple=true;}else if(kind==='zip')i.accept='.zip,application/zip';else{i.multiple=true;i.accept=cfg.accept;}i.onchange=()=>{if(i.files.length)send([...i.files],kind);};i.click();};
 const doImport=(_event,result)=>{const A=result||M[runKey];if(!A)return;ensureDiscovery();const fileId=cfg.key+'agentrun-'+A.id;
  M.files=M.files.filter(f=>f.id!==fileId);M.files.push({id:fileId,name:cfg.name+' agent run '+A.id,size:0,category:cfg.category,source:cfg.source,parsed:true,sample:false,registeredAt:A.at,context:{component:M.scope.component||'',environment:M.scope.environment||'',asOf:A.at.slice(0,10),revision:'',origin:'MindSuite '+cfg.name.toLowerCase()+' agent team',custodian:'',basis:'implemented',locator:A.findings.length+' findings',lineage:''}});
  const R=(id,cite,fields,status)=>({id:cfg.key+'agent-'+id,origin:'agent',team:cfg.team,status:status||'extracted',method:status==='inferred-code'?'inferred':'extracted',evidence:[fileId],cite,...fields});
  for(const t of INV_TYPES)M.inventory[t.key]=M.inventory[t.key].filter(x=>x.origin!=='agent'||x.team!==cfg.team||protectedStatus(x.status));
  const put=(key,rows)=>{const list=M.inventory[key];for(const row of rows){const i=list.findIndex(x=>x.id===row.id);if(i<0)list.push(row);else if(!protectedStatus(list[i].status))list[i]=row;}};
  const F=k=>A.findings.filter(f=>f.kind===k);const by=f=>f.cite||('finding '+f.id);
  cfg.importTo(A,{R,put,F,by});
  if(cfg.risks)put('risks',A.findings.filter(f=>cfg.risks[f.kind]).slice(0,80).map(f=>R(f.id,by(f),{name:f.title,category:cfg.risks[f.kind][0],severity:(f.data&&f.data.severity)||'medium',context:'',mitigation:cfg.risks[f.kind][1]})));
  A.imported=true;if(!result){persist();notice(cfg.name+' results added to the inventory for '+cfg.people+' to confirm.');render();}};
 const download=()=>{const A=M[runKey];const text=(live.snap&&live.snap.run.id===A?.id&&live.snap.report)||A?.report;if(text)downloadText(cfg.team+'-analysis-'+(A?.id||'run')+'.md',text);else window.open(agentURL('/api/runs/'+A.id+'/report.md'));};
 const panel=()=>{const c=agentCfg();const H=AGENT_LIVE.health;const lv=live.snap&&live.snap.run.id===c[idKey]?live.snap:null;const A=M[runKey];const running=lv&&lv.run.status==='running';
  const steps=(lv?lv.run.steps:A&&A.steps)||[];
  const conn=h('div',{class:'agent-conn'},[
   h('div',{class:'agent-conn-row'},[inputControl('Agent backend address',c.url,v=>{c.url=v;persist();},'url','http://127.0.0.1:8090'),button('Check connection',agentCheck,'btn small'),
    H?h('span',{class:'source-status '+(H.error?'open':'done'),text:H.error?'Not reachable':'Connected · v'+H.version+(H.modelAvailable?' · Claude available':' · rule-based only')}):null]),
   h('div',{class:'type-chips'},[agentChip(cfg.consent,!!c[consentKey],v=>{c[consentKey]=v;persist();render();})]),
   h('p',{class:'scan-privacy',text:c[consentKey]?cfg.privacyOn:cfg.privacyOff}),
   h('p',{class:'muted',text:cfg.give}),
   h('div',{class:'scan-actions'},[button('Stage export files',()=>pick('files'),'btn primary'),button('Choose folder',()=>pick('folder'),'btn small'),button('Choose zip',()=>pick('zip'),'btn small')])]);
  const stepList=steps.length?h('ol',{class:'agent-steps'},steps.map(s=>h('li',{class:'agent-step '+s.status},[h('span',{class:'agent-step-dot'}),h('b',{text:s.n+' · '+s.title}),h('small',{text:(AGENT_STEP_TEXT[s.status]||s.status)+(s.note?' — '+s.note:'')})]))):null;
  const log=running||(lv&&live.log.length)?h('div',{class:'agent-log'},live.log.slice(-14).map(m=>h('div',{class:'agent-msg '+m.type},[h('span',{class:'agent-msg-type',text:m.type}),h('b',{text:label(m.from)}),' ',m.type==='finding'?m.finding.title:m.type==='verification'?m.ref+': '+m.status+(m.text?' — '+m.text:''):m.text||'']))):null;
  let result=null;
  if(A&&!running){const F=k=>A.findings.filter(f=>f.kind===k);const src=A.source||{};const counts=Object.entries(A.counts||{}).sort((a,b)=>b[1]-a[1]);
   const list=(title,fs,open)=>fs.length?h('details',{class:'scan-details',open:open?'':null},[h('summary',{text:title+' ('+fs.length+')'}),h('ul',{class:'plain-list'},fs.slice(0,40).map(f=>h('li',{},[h('b',{text:f.title}),f.detail?' — '+f.detail.slice(0,400):'',f.cite?h('small',{class:'muted',text:' '+f.cite}):null])))]):null;
   result=h('div',{class:'agent-result'},[
    h('div',{class:'metrics'},[['Findings',A.findings.length],['Rejected by verifier',A.rejected],...cfg.metrics(A,src,F),['Expert questions',A.questions.length],['Sent to Claude',A.sent.length?A.sent.length+' requests':'nothing']].map(([l,v])=>h('div',{class:'metric'},[h('b',{text:String(v)}),h('span',{text:l})]))),
    h('p',{class:'muted',text:'Run '+A.id+' · findings per agent: '+counts.map(([k,v])=>label(k)+' '+v).join(', ')}),
    A.error?h('p',{class:'input-status open',text:'The run stopped with an error: '+A.error}):null,
    agentProblemsBox(A,follow),
    ...cfg.sections(A,F,list),
    A.steps&&A.steps[4]&&A.steps[4].status==='skipped'?h('p',{class:'muted',text:cfg.skippedText+': '+A.steps[4].note+'.'}):null,
    A.questions.length?h('details',{class:'scan-details'},[h('summary',{text:'Questions for '+cfg.people+' ('+A.questions.length+')'}),h('ol',{class:'plain-list'},A.questions.map(q=>h('li',{text:q.text})))]):null,
    A.sent.length?h('details',{class:'scan-details'},[h('summary',{text:'What was sent to Claude'}),h('ul',{class:'plain-list'},A.sent.map(s=>h('li',{text:label(s.agent)+' · '+s.model+' · '+s.files.join(', ')+' · '+s.chars.toLocaleString()+' characters'})))]):null,
    h('div',{class:'scan-next'},[button('Download report (.md)',download,'btn small'),button(A.imported?'Update inventory from this run':'Add results to the inventory',doImport,'btn primary'),A.imported?button('Confirm in inventory →',()=>go('inventory'),'btn small'):null])]);}
  return h('section',{class:'summary-section agent-panel',id:anchor},[h('h2',{text:'Run the '+cfg.name.toLowerCase()+' agent team'}),
   h('p',{class:'muted',text:cfg.intro}),
   h('p',{},[button('How the '+cfg.name.toLowerCase()+' agents work ↗',open,'linkbtn'),' · ',h('span',{class:'muted',text:'Same backend as the other teams: '}),h('code',{text:'go run ./backend/cmd/mindsuite-agents serve'})]),
   conn,stepList,log,result]);};
 return {panel,go:goPanel,open,send,follow,importResults:doImport,keep};
}
