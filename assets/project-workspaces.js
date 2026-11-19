/* Project selection is fixed for the lifetime of the page. Opening another
   workspace loads a new page so pending callbacks never see a different project. */
const projectPrefix = 'mindsuite-workspace-v1:';
const projectPattern = /^[a-f0-9]{8}(?:-[a-f0-9]{4}){3}-[a-f0-9]{12}$/;
const recentProjectKey = 'mindsuite-recent-workspace-v1';
const migratedProjectKey = 'mindsuite-migrated-workspace-v1';

function readProject(id) {
  try {
    const state = JSON.parse(localStorage.getItem(projectPrefix + id));
    return state?.version === 1 && Array.isArray(state.files) ? state : null;
  } catch {
    return null;
  }
}

function projectAddress(id) {
  const url = new URL(location.href);
  url.searchParams.set('workspace', id);
  url.hash = '';
  return url;
}

function openProjectState(legacyKey, blank) {
  try {
    const requested = new URL(location.href).searchParams.get('workspace');
    const recent = localStorage.getItem(recentProjectKey);
    let id = [requested, recent].find(key => projectPattern.test(key || '') && readProject(key));
    let state = id ? readProject(id) : null;
    if (!state) {
      id = crypto.randomUUID();
      state = blank();
      const migrated = localStorage.getItem(migratedProjectKey);
      if (!migrated) {
        const legacy = JSON.parse(localStorage.getItem(legacyKey) || 'null');
        if (legacy?.version === 1 && Array.isArray(legacy.files)) {
          state = legacy;
          state.legacyRunIds = Object.entries(state.agentBackend || {})
            .filter(([key, value]) => /runid$/i.test(key) && typeof value === 'string')
            .map(([, value]) => value);
          for (const [key, result] of Object.entries(state)) {
            if (/agentRun$/i.test(key) && result?.id) state.legacyRunIds.push(result.id);
          }
        }
      }
      writeProject(id, state);
      if (!migrated) localStorage.setItem(migratedProjectKey, id);
    }
    localStorage.setItem(recentProjectKey, id);
    history.replaceState(null, '', projectAddress(id));
    return {id, state: {...blank(), ...state}};
  } catch {
    return {id: crypto.randomUUID(), state: blank()};
  }
}

function writeProject(id, state) {
  localStorage.setItem(projectPrefix + id, JSON.stringify({...state, version: 1}));
}

function savedProjects() {
  return Object.keys(localStorage)
    .filter(key => key.startsWith(projectPrefix))
    .map(key => key.slice(projectPrefix.length))
    .filter(id => projectPattern.test(id))
    .map(id => ({id, project: readProject(id)}))
    .filter(entry => entry.project)
    .sort((a, b) => (a.project.project || '').localeCompare(b.project.project || ''));
}

function showProjectPicker(create) {
  const dialog = document.getElementById('projectPicker');
  dialog.replaceChildren();
  const title = h('h2', {id: 'projectPickerTitle', text: create ? 'New project' : 'Project workspaces'});
  const close = button('Close', () => dialog.close());
  if (create) {
    const name = h('input', {id: 'newWorkspaceName', maxlength: '180', autocomplete: 'off'});
    const form = h('form', {}, [
      title,
      h('label', {for: 'newWorkspaceName', text: 'Project name (optional)'}),
      name,
      h('small',{class:'muted',text:'Leave this blank to name the project from your first upload.'}),
      h('div', {class: 'project-actions'}, [close, h('button', {type: 'submit', class: 'btn primary', text: 'Create workspace'})])
    ]);
    form.addEventListener('submit', event => {
      event.preventDefault();
      const project = name.value.trim();
      try {
        writeProject(workspaceId, M);
        const id = crypto.randomUUID();
        writeProject(id, {...emptyMock(), project});
        location.assign(projectAddress(id));
      } catch {
        dialog.close();
        notice('Browser storage is unavailable or full. Export your current project before making room.');
      }
    });
    dialog.append(form);
  } else {
    const choices = savedProjects().map(({id, project}) => {
      const current = id === workspaceId;
      const name = project.project?.trim() || 'Untitled project';
      const suffix = current ? ' · Current workspace' : '';
      return h('li', {}, [h('button', {
        type: 'button', class: 'btn', disabled: current ? '' : null,
        text: name + suffix, onclick: () => {
          try {
            writeProject(workspaceId, M);
            location.assign(projectAddress(id));
          } catch {
            dialog.close();
            notice('Browser storage is full. Export your current project before switching.');
          }
        }
      })]);
    });
    dialog.append(title, h('ul', {class: 'project-choices'}, choices),
      h('div', {class: 'project-actions'}, [close, button('New project', () => showProjectPicker(true), 'btn primary')]));
  }
  if (!dialog.open) dialog.showModal();
  if (create) document.getElementById('newWorkspaceName').focus();
}

function openReferenceWorkspace() {
  if (new URL(location.href).searchParams.get('sample') === workspaceId) return;
  const dialog = document.getElementById('projectPicker');
  try {
    writeProject(workspaceId, M);
    const id = crypto.randomUUID();
    writeProject(id, {...emptyMock(), project: 'HarvestLink reference scenario'});
    const url = projectAddress(id);
    url.searchParams.set('sample', id);
    location.assign(url);
  } catch {
    if (dialog.open) dialog.close();
    notice('Browser storage is full. Export your current project before opening the reference scenario.');
  }
}



const projectRuns = {promise: null, entries: null, at: 0};

function loadWorkspaceRuns(force = false) {
  if (projectRuns.promise) return projectRuns.promise;
  if (!force && projectRuns.entries && Date.now() - projectRuns.at < 15000) {
    return Promise.resolve(projectRuns.entries);
  }
  projectRuns.promise = agentFetch('/api/runs').then(runs => {
    if (!Array.isArray(runs)) throw new Error('The backend did not return a run list.');
    projectRuns.entries = runs;
    projectRuns.at = Date.now();
    return runs;
  }).finally(() => { projectRuns.promise = null; });
  return projectRuns.promise;
}

const evidenceReportKeys = {
  code:'agentRun', database:'dbAgentRun', infrastructure:'infraAgentRun',
  messaging:'msgAgentRun', identity:'idAgentRun', telemetry:'telAgentRun',
  batch:'batchAgentRun', network:'netAgentRun', documents:'docAgentRun', incidents:'incAgentRun'
};

function evidenceCardReportSlot(team, title) {
  return h('div',{class:'card-agent-reports','data-report-team':team,'data-report-title':title,hidden:''});
}

function canResumeEvidenceRun(run, team) {
  return (run.status==='interrupted' || (run.status==='failed' && run.batches)) &&
    evidenceUploads().some(upload=>upload.team===team && upload.placed && upload.runId===run.id);
}

function displayEvidenceCardReports(slot, runs) {
  const team = slot.dataset.reportTeam;
  const title = slot.dataset.reportTitle;
  const cached = M[evidenceReportKeys[team]];
  const entries = runs.filter(run=>(run.team || 'code')===team);
  if (cached?.id && !entries.some(run=>run.id===cached.id)) {
    entries.push({id:cached.id, status:cached.status || 'done', source:cached.source, started:cached.at});
  }
  entries.sort((a,b)=>String(b.id).localeCompare(String(a.id)));
  slot.hidden = entries.length === 0;
  if (slot.hidden) return;
  const visible = entries.find(run=>run.status==='done') || entries[0];
  const newerRun = entries[0] !== visible ? entries[0] : null;
  const row = run => {
    const status = {done:'Ready', running:'Analyzing', failed:'Failed', interrupted:'Interrupted'}[run.status] || run.status;
    const when = run.started ? new Date(run.started).toLocaleString() : '';
    const batches = run.batches ? run.batches.completed+' of '+run.batches.total+' batches complete' : '';
    const canResume = canResumeEvidenceRun(run,team);
    return h('li',{class:'project-run'},[
      h('span',{},[
        h('b',{text:run.source?.root || title}),
        h('small',{text:[run.problems && run.status==='done'?'Finished with problems':status,batches,when].filter(Boolean).join(' · ')})
      ]),
      button('Telemetry',()=>openAgentTelemetry(run.id,title),'btn small'),
      canResume ? h('button',{type:'button',class:'btn small',disabled:evidenceUI.busy?'':null,
        text:'Resume analysis',onclick:()=>runEvidenceTeam(team,false,run.id)}) : null,
      run.status==='done' || run.status==='failed' ? h('button',{
        type:'button',class:'btn small primary',text:run.status==='done'?'View report':'View run details',
        'aria-label':(run.status==='done'?'View ':'View run details for ')+title+' '+(run.status==='done'?'report ':'')+run.id,
        onclick:()=>openAgentReport({id:run.id,team:title,cached:cached?.id===run.id?cached:null})
      }) : null
    ]);
  };
  const older = entries.filter(run=>run!==visible && run!==newerRun);
  const expanded = slot.querySelector('details')?.open;
  slot.replaceChildren(
    h('ul',{class:'project-choices'},newerRun?[row(visible),row(newerRun)]:[row(visible)]),
    ...(older.length?[h('details',{class:'more-detail',open:expanded?'':null},[
      h('summary',{text:'Earlier runs ('+older.length+')'}),
      h('ul',{class:'project-choices'},older.map(row))
    ])]:[])
  );
}

function connectEvidenceCardReports(section) {
  const slots = [...section.querySelectorAll('[data-report-team]')];
  const draw = runs => slots.forEach(slot=>displayEvidenceCardReports(slot,runs));
  async function refresh(force = false) {
    if (!section.isConnected) return;
    let runs;
    try { runs = await loadWorkspaceRuns(force); }
    catch { runs = projectRuns.entries || []; }
    if (!section.isConnected) return;
    draw(runs);
    if (runs.some(run=>run.status==='running')) setTimeout(()=>refresh(true),1600);
  }
  draw(projectRuns.entries || []);
  queueMicrotask(()=>refresh());
  return section;
}

function projectRunHistory() {
  const list = h('div', {'aria-live': 'polite'});
  const teams = {
    code: 'Source code', database: 'Database', infrastructure: 'Infrastructure',
    messaging: 'Messaging & caching', identity: 'Identity & access', telemetry: 'Telemetry',
    batch: 'Batch & scheduling', network: 'Network edge & partners',
    documents: 'Documents & diagrams', incidents: 'Incidents & changes'
  };

  let poll;
  function display(runs) {
    clearTimeout(poll);
    if (runs.some(run=>run.status==='running')) poll=setTimeout(()=>{if(list.isConnected)refreshHistory(true);},1600);
    const cached = M.agentRun;
    if (cached && !runs.some(run => run.id === cached.id)) {
      runs = [...runs, {id: cached.id, team: 'code', source: cached.source, started: cached.at, status: cached.status}];
    }
    const resume = {
      code: ['runId', 'agentRun', agentFollow], database: ['dbRunId', 'dbAgentRun', dbAgentFollow],
      infrastructure: ['infraRunId', 'infraAgentRun', infraAgentFollow], messaging: ['msgRunId', 'msgAgentRun', msgAgentFollow],
      identity: ['idRunId', 'idAgentRun', idAgentFollow], telemetry: ['telRunId', 'telAgentRun', telAgentFollow],
      batch: ['batchRunId', 'batchAgentRun', BATCH_TEAM.follow], network: ['netRunId', 'netAgentRun', NETWORK_TEAM.follow],
      documents: ['docRunId', 'docAgentRun', DOCUMENTS_TEAM.follow], incidents: ['incRunId', 'incAgentRun', INCIDENTS_TEAM.follow]
    };

    const statuses = {done: 'Complete', failed: 'Failed', running: 'Running', interrupted: 'Interrupted'};
    const statusText = run => run.status === 'done' && run.problems ? 'Finished with problems (' + run.problems + ' agent' + (run.problems === 1 ? '' : 's') + ' did not finish)' : statuses[run.status] || run.status;
    if (!runs.length) {
      list.replaceChildren(h('div',{class:'empty-state report-empty'},[
        h('b',{text:'No reports yet'}),
        h('p',{text:'Add evidence and start analysis to see reports from the agents.'}),
        button('Go to evidence',()=>showEvidenceStep('sources'),'btn small primary')
      ]));
      return;
    }
    list.replaceChildren(h('ul', {class: 'project-choices'}, runs.map(run => {
      const team = teams[run.team] || 'Agent analysis';
      const follow = resume[run.team];
      const canResume = follow && agentCfg()[follow[0]] === run.id && M[follow[1]]?.id !== run.id;
      return h('li', {class: 'project-run'}, [
        h('span', {}, [
          h('b', {text: team}),
          h('small', {text: [run.source?.root, run.started ? new Date(run.started).toLocaleString() : '', statusText(run), run.batches ? run.batches.completed+' of '+run.batches.total+' batches complete' : '', run.id].filter(Boolean).join(' · ')})
        ]),
        button('Telemetry',()=>openAgentTelemetry(run.id,team),'btn small'),
        canResumeEvidenceRun(run,run.team||'code')
          ? button('Resume analysis', () => runEvidenceTeam(run.team||'code',false,run.id), 'btn small')
          : canResume && run.status !== 'interrupted' ? button('Resume results', () => follow[2](run.id), 'btn small') : null,
        run.status !== 'interrupted' ? button('View report', () => openAgentReport({
          id: run.id, team, cached: cached?.id === run.id ? cached : null
        }), 'btn primary small') : null,
        run.status === 'done' ? h('a', {class: 'btn small', href: agentURL('/api/runs/' + run.id + '/report.md'), text: 'Download report'}) : null
      ]);
    })));
  }

  async function refreshHistory(force = false) {
    if (!force && projectRuns.entries && Date.now() - projectRuns.at < 15000) {
      display(projectRuns.entries);
      return;
    }
    if (!projectRuns.entries) list.textContent = 'Loading analysis history…';
    try {
      display(await loadWorkspaceRuns(force));
    } catch (error) {
      display(projectRuns.entries || []);
      list.prepend(h('p', {class: 'conflict-note', text: 'Analysis history unavailable. ' + error.message}));
    }
  }

  const savedFile = h('input', {type: 'file', accept: '.json,application/json', hidden: ''});
  savedFile.addEventListener('change', async () => {
    const file = savedFile.files[0];
    if (!file) return;
    savedFile.value = '';
    try {
      const parsed = JSON.parse(await file.text());
      if (!parsed?.run || !Array.isArray(parsed.findings) || typeof parsed.report !== 'string') {
        throw new Error('Choose a MindSuite run result JSON file.');
      }
      openAgentReport({snapshot: parsed, team: teams[parsed.run.options?.team] || 'Agent analysis'});
    } catch (error) {
      notice('Could not open saved report. ' + error.message);
    }
  });

  queueMicrotask(() => { if (list.isConnected) refreshHistory(); });
  return h('section', {class: 'summary-section', id: 'project-analysis-history'}, [
    h('h2', {text: 'Workspace runs'}),
    h('p', {class: 'muted', text: 'Open a run to see findings, citations, and agent coverage.'}),
    h('div', {class: 'report-reader-actions'}, [
      button('Refresh report list', () => refreshHistory(true), 'btn small'),
      button('Open saved report JSON', () => savedFile.click(), 'btn small'), savedFile
    ]),
    list
  ]);
}


const inventoryReportStatus={busy:false,message:''};

async function populateInventoryFromReports(){
  if(inventoryReportStatus.busy)return;
  inventoryReportStatus.busy=true;
  inventoryReportStatus.message='Reading completed team reports…';
  render();
  try {
    const importers={
      code:agentImport,database:dbAgentImport,infrastructure:infraAgentImport,messaging:msgAgentImport,
      identity:idAgentImport,telemetry:telAgentImport,batch:BATCH_TEAM.importResults,network:NETWORK_TEAM.importResults,
      documents:DOCUMENTS_TEAM.importResults,incidents:INCIDENTS_TEAM.importResults
    };
    const runs=await loadWorkspaceRuns(true);
    const latest=new Map();
    const ordered=[...runs].sort((a,b)=>String(b.started||'').localeCompare(String(a.started||''))||String(b.id).localeCompare(String(a.id)));
    for(const run of ordered){
      const team=run.team||'code';
      if(run.status==='done'&&Object.hasOwn(importers,team)&&!latest.has(team))latest.set(team,run);
    }
    if(!latest.size){
      inventoryReportStatus.message='No completed team reports yet. Run a team from Evidence and try again when its report is ready.';
      return;
    }
    const reports=await Promise.allSettled([...latest].map(async ([team,run])=>{
      const s=await agentFetch('/api/runs/'+encodeURIComponent(run.id));
      if(s.run?.id!==run.id||s.run.status!=='done'||!Array.isArray(s.findings))throw new Error('Report unavailable');
      const coverage=s.coverage||{};
      return [team,{
        id:run.id,at:new Date(run.started||s.run.started||Date.now()).toISOString(),
        source:s.source||{},coverage:{files:coverage.files||0,analysed:coverage.analysed||0},
        findings:s.findings.filter(f=>f.status!=='rejected').map(f=>{
          const cite=f.citations?.[0];
          return {...f,detail:f.detail||'',cite:cite?.file?cite.file+(cite.line?':'+cite.line:''):''};
        })
      }];
    }));
    let loaded=0;
    for(const report of reports){
      if(report.status!=='fulfilled')continue;
      const [team,result]=report.value;
      const inventory=clone(M.inventory), files=[...M.files];
      try{
        importers[team](null,result);
        loaded++;
      }catch{
        M.inventory=inventory;
        M.files=files;
      }
    }
    if(loaded)persist();
    inventoryReportStatus.message=loaded
      ? 'Inventory updated from '+loaded+' team '+(loaded===1?'report.':'reports.')+(loaded<latest.size?' '+(latest.size-loaded)+' could not be read.':'')
      : 'Could not read the completed team reports. Try again from this workspace.';
  } catch(error) {
    inventoryReportStatus.message='Could not load team reports. '+error.message;
  } finally {
    inventoryReportStatus.busy=false;
    render();
  }
}
