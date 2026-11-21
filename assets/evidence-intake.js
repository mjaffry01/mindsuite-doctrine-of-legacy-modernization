const evidenceTypes = {
  code: 'Source evidence', database: 'Database evidence', infrastructure: 'Environment evidence',
  messaging: 'Messaging evidence', identity: 'Identity evidence', telemetry: 'Telemetry evidence',
  batch: 'Batch evidence', network: 'Network evidence', documents: 'Document and diagram evidence',
  incidents: 'Incident and change evidence'
};
// allowModel holds the per-card consent: {team: true} lets that team's next run send selected evidence to Claude.
const evidenceUI = {busy: false, allowModel: {}, message: '', checkedStaging: false, step: 'project'};

function evidenceTeam(source) {
  return source === 'middleware' ? 'messaging' : source;
}

function evidenceUploads() {
  return M.evidenceUploads || (M.evidenceUploads = []);
}


function markIntakeEdit(field) {
  (M.intakeEdits ||= {})[field] = true;
}

function refreshIntakeSuggestions() {
  const uploads = evidenceUploads();
  const code = uploads.find(item=>item.team==='code' && item.profile?.system) || uploads.find(item=>item.team==='code');
  const profile = code?.profile;
  const previous = M.intakeSuggested || {};
  const next = {};
  const fill = (field, current, suggestion, set) => {
    if (M.intakeEdits?.[field] || (current && current!==previous[field])) return;
    if (current || suggestion) set(suggestion || '');
    if (suggestion) next[field] = suggestion;
  };
  fill('project', M.project, profile?.system || code?.name || uploads[0]?.name, value=>{M.project=value;});
  fill('system', M.scope.component, profile?.system || code?.name, value=>{M.scope.component=value;});
  const types = {};
  for (const upload of uploads) {
    for (const item of upload.profile?.types || []) {
      if (TYPES.some(([id])=>id===item.type)) types[item.type] ||= item.file;
    }
  }
  if (!M.intakeEdits?.types && (!selectedTypes().length || JSON.stringify(selectedTypes())===JSON.stringify(previous.types))) {
    M.scope.types = Object.keys(types);
    if (M.scope.types.length) next.types = [...M.scope.types];
  }
  M.intakeSuggested = next;
  M.intakeBasis = {
    name: next.project || next.system ? profile?.nameFile || code?.name || uploads[0]?.name : '',
    types: next.types ? types : {}
  };
}

function evidencePageTitle() {
  const name = M.project.trim();
  return name ? 'Record evidence: '+name : 'Record evidence';
}

function showEvidenceStep(step, focus = true) {
  if (busy) return;
  if (isNewProduct()) { go('upload'); return; }
  const needsName = step === 'sources' && !M.project.trim();
  if (needsName) step = 'project';
  evidenceUI.step = step;
  if (screen !== 'upload' && screen !== 'evidence') {
    go('upload');
  } else {
    for (const panel of document.querySelectorAll('[data-evidence-step]')) {
      panel.hidden = panel.dataset.evidenceStep !== step;
    }
    for (const item of document.querySelectorAll('[data-flow-step]')) {
      if (item.dataset.flowStep === step) item.setAttribute('aria-current', 'step');
      else item.removeAttribute('aria-current');
    }
    window.scrollTo({top:0});
  }
  if (needsName) document.getElementById('interview-project-name')?.reportValidity();
  else if (focus) document.querySelector('.discovery-header button[aria-current="step"]')?.focus({preventScroll:true});
}

function evidenceFlowchart(current) {
  const steps = [
    ['project', 'Project', 'Name and interview', () => showEvidenceStep('project')],
    ['sources', 'Evidence', 'Add files and run teams', () => showEvidenceStep('sources')],
    ['reports', 'Reports', 'Open team results', () => go('analysis')]
  ];
  return h('nav', {class:'evidence-flow', 'aria-label':'Discovery workflow'}, [
    h('ol', {class:'evidence-flow-track'}, steps.map(([id, title, description, action], index) =>
      h('li', {class:'evidence-flow-step'}, [
        h('button', {
          type:'button', 'data-flow-step':id, 'aria-current':current===id?'step':null,
          'aria-label':title+' · '+description, disabled:busy?'':null, onclick:action
        }, [h('b',{text:title}), h('small',{text:description})]),
        index < steps.length-1 ? h('span',{class:'evidence-flow-arrow','aria-hidden':'true'}) : null
      ])
    ))
  ]);
}

function discoveryHeader(title, current, actions = []) {
  return h('header', {class:'discovery-header'}, [
    h('div',{class:'discovery-title'},[h('h1',{text:title}),...actions]),
    evidenceFlowchart(current)
  ]);
}

function intakeSuggestionNotes() {
  const basis = M.intakeBasis || {};
  const types = Object.entries(basis.types || {});
  const names = [M.intakeSuggested?.project && !M.intakeEdits?.project ? 'Project name' : '',
    M.intakeSuggested?.system && !M.intakeEdits?.system ? 'System name' : ''].filter(Boolean);
  if (!names.length && (!types.length || M.intakeEdits?.types)) return null;
  return h('div',{class:'validate-card'},[
    h('b',{text:'Suggested from uploaded evidence'}),
    names.length && basis.name ? h('p',{class:'muted',text:names.join(' and ')+' · '+basis.name}) : null,
    !M.intakeEdits?.types ? h('ul',{class:'plain-list'},types.map(([type,file])=>h('li',{text:typeName(type)+' · '+file}))) : null,
    h('small',{class:'muted',text:'Review these suggestions. You can change any field or selection.'})
  ]);
}

function chooseWorkspaceEvidence(team, kind = 'files') {
  document.getElementById('workspace-evidence-picker')?.remove();
  const input = h('input', {id:'workspace-evidence-picker', hidden:'', type: 'file', multiple: kind === 'zip' ? null : '', accept: kind === 'zip' ? '.zip,application/zip' : null});
  if (kind === 'folder') input.webkitdirectory = true;
  input.addEventListener('change', () => {
    const files = Array.from(input.files);
    input.remove();
    if (!files.length) return;
    const data = new FormData();
    const folder = files[0].webkitRelativePath?.split('/')[0];
    if (folder) data.append('name', folder);
    const zip = files.length === 1 && /\.zip$/i.test(files[0].name);
    for (const file of files) {
      const relative = file.webkitRelativePath || file.name;
      if (kind === 'folder' && /(^|\/)(\.git|node_modules|bin|obj|\.vs)\//.test(relative)) continue;
      data.append(zip ? 'zip' : 'file', file, relative);
    }
    submitWorkspaceEvidence(team, data);
  });
  input.addEventListener('cancel', () => input.remove());
  document.body.append(input);
  input.click();
}

async function submitWorkspaceEvidence(team, data) {
  if (evidenceUI.busy) return;
  evidenceUI.busy = true;
  evidenceUI.message = 'Uploading ' + evidenceTypes[team].toLowerCase() + ' to temporary storage…';
  render();
  try {
    const health = await agentFetch('/api/health');
    if (!health.evidenceIntake) throw new Error('The agent backend must support staged evidence uploads.');
    const entry = await agentFetch('/api/evidence/stage?team=' + encodeURIComponent(team), {method: 'POST', body: data});
    evidenceUploads().push(entry);
    refreshIntakeSuggestions();
    evidenceUI.message = entry.name + ' is staged. Run its agent team from the evidence card.';
    persist();
    if (screen === 'analysis') go('upload');
  } catch (error) {
    evidenceUI.message = 'Upload could not be staged. ' + error.message;
    notice(evidenceUI.message);
  } finally {
    evidenceUI.busy = false;
    render();
    if (screen === 'upload') document.getElementById('intake-status')?.scrollIntoView({block:'nearest'});
  }
}

function stageAgentForm(data) {
  const options = JSON.parse(data.get('options') || '{}');
  const team = options.team || 'code';
  const file = data.get('file');
  const root = typeof file?.name === 'string' ? file.name.split('/') : [];
  if (root.length > 1) data.append('name', root[0]);
  return submitWorkspaceEvidence(team, data);
}

async function dropStagedEvidence(entry) {
  if (evidenceUI.busy) return;
  evidenceUI.busy = true;
  try {
    await agentFetch('/api/evidence/' + encodeURIComponent(entry.id) + '/discard', {method:'POST'});
    M.evidenceUploads = evidenceUploads().filter(item => item.id !== entry.id);
    refreshIntakeSuggestions();
    evidenceUI.message = 'Removed from temporary storage.';
    persist();
  } catch (error) {
    evidenceUI.message = 'Could not remove the temporary upload. ' + error.message;
  } finally {
    evidenceUI.busy = false;
    render();
  }
}

function pendingEvidence(team) {
  return evidenceUploads().filter(item => !item.runId && item.team === team);
}

function businessReviewInterview() {
  const answers=brief();
  const text=value=>String(value||'').trim().slice(0,1000);
  const interview={
   reasons:modernizationReasons(answers),problem:text(answers.whyDetail),success:text(answers.success),
   system:text(M.scope.component),objective:text(M.scope.journey),outOfScope:text(M.scope.outOfScope),deadline:answers.deadline||''
  };
  return Object.values(interview).some(value=>value.length>0)?interview:null;
}

async function runEvidenceTeam(team, reanalyze = false, resumeId = "") {
  if (evidenceUI.busy) return;
  const pending = resumeId
    ? evidenceUploads().filter(item=>item.team===team && item.placed && item.runId===resumeId)
    : reanalyze
      ? evidenceUploads().filter(item=>item.team===team && item.placed && item.runId)
      : pendingEvidence(team);
  if (!pending.length) return;
  if (!M.project.trim()) {
    showEvidenceStep('project');
    document.getElementById('interview-project-name')?.reportValidity();
    return;
  }
  evidenceUI.busy = true;
  evidenceUI.message = 'Starting '+evidenceTypes[team].toLowerCase()+' analysis…';
  render();
  try {
    const businessReview=team==='code'||team==='database';
    const interview=businessReview?businessReviewInterview():null;
    if (businessReview || resumeId) {
      const health=await agentFetch('/api/health');
      if ((team==='code' && !health.interviewCodeReview) || (team==='database' && !health.databaseBusinessReview) || (resumeId && !health.batchReview)) throw new Error('Restart the local agent service to use this review feature.');
    }
    const result = await agentFetch('/api/evidence/start', {
      method:'POST', headers:{'Content-Type':'application/json'},
      body:JSON.stringify({name:M.project.trim(), uploads:pending.map(item=>item.id), allowModel:!!evidenceUI.allowModel[team], interview, reanalyze, resume:!!resumeId})
    });
    M.evidenceWorkspace = result.folder;
    // Consent covers one run only.
    delete evidenceUI.allowModel[team];
    for (const placed of result.uploads) {
      const item = evidenceUploads().find(entry => entry.id === placed.id);
      if (!item) continue;
      Object.assign(item, {runId:placed.runId||item.runId, error:placed.error, placed:placed.placed||item.placed});
      if (placed.runId) {
        const keys = {code:'runId', database:'dbRunId', infrastructure:'infraRunId', messaging:'msgRunId',
          identity:'idRunId', telemetry:'telRunId', batch:'batchRunId', network:'netRunId',
          documents:'docRunId', incidents:'incRunId'};
        const key = keys[placed.team];
        if (key) agentCfg()[key] = placed.runId;
      }
      if (!placed.placed) continue;
      const source = placed.team === 'messaging' ? 'middleware' : placed.team;
      const category = SOURCES.find(entry => entry.id === source)?.cat || 'code';
      const id = 'evidence-'+placed.id;
      if (!M.files.some(entry => entry.id === id)) M.files.push({
        id, name:item.name, size:0, category, source, sample:false, registeredAt:new Date().toISOString(),
        context:{component:M.scope.component||'',environment:'',asOf:'',revision:'',origin:'Workspace evidence',custodian:'',basis:'',locator:item.files.join(', '),lineage:''}
      });
    }
    persist();
    if (typeof projectRuns !== 'undefined') projectRuns.at = 0;
    const runs = result.uploads.filter(item => item.runId).length;
    const errors = result.uploads.filter(item => item.error).length;
    const followUp = errors ? ' Check the upload messages below.' : ' Follow progress on the evidence card.';
    evidenceUI.message = runs
      ? 'Started '+runs+' analysis '+(runs===1?'run.':'runs.')+followUp
      : 'Analysis could not start. Check the upload messages below.';
    notice(evidenceUI.message);
  } catch (error) {
    evidenceUI.message = 'Could not start. ' + error.message;
    notice(evidenceUI.message);
  } finally {
    evidenceUI.busy = false;
    render();
  }
}

async function restoreStagedUploads() {
  if (evidenceUI.checkedStaging || evidenceUI.busy) return;
  evidenceUI.checkedStaging = true;
  try {
    const list = await agentFetch('/api/evidence/staged');
    if (evidenceUI.busy) {
      evidenceUI.checkedStaging = false;
      return;
    }
    if (!Array.isArray(list)) return;
    const existing = new Set(evidenceUploads().map(item=>item.id));
    const restored = list.filter(item=>!existing.has(item.id));
    if (!restored.length) return;
    evidenceUploads().push(...restored);
    refreshIntakeSuggestions();
    persist();
    if (document.getElementById('intake-status')) render();
  } catch {
    // Staging is optional until the user chooses to upload evidence.
  }
}

function evidenceIntakeStatus() {
  queueMicrotask(restoreStagedUploads);
  const entries = evidenceUploads();
  const pending = entries.filter(item => !item.placed);
  const attention = entries.some(item => item.error);
  return h('div', {class:'evidence-intake-status', id:'intake-status', hidden:!entries.length && !evidenceUI.message?'':null}, [
    evidenceUI.message ? h('p', {role:'status', text:evidenceUI.message}) : null,
    entries.length ? h('details', {class:'discovery-disclosure', open:attention?'':null}, [
      h('summary', {text:pending.length ? pending.length+' staged for analysis' : entries.length+' saved '+(entries.length===1?'upload':'uploads')}),
      h('ul', {class:'project-choices'}, entries.map(item => h('li', {class:'project-run'}, [
        h('span', {}, [
          h('b',{text:evidenceTypes[item.team]||item.team}),
          h('small',{text:item.name+' · '+item.files.length+' '+(item.files.length===1?'file':'files')+' · '+(item.runId?'Submitted for analysis':item.placed?'Saved in project':'Staged')})
        ]),
        item.error ? h('p',{class:'conflict-note',text:item.error}) : null,
        !item.placed ? h('button',{type:'button',class:'btn small',disabled:evidenceUI.busy?'':null,onclick:()=>dropStagedEvidence(item),text:'Remove staged upload'}) : null
      ])))
    ]) : null
  ]);
}
