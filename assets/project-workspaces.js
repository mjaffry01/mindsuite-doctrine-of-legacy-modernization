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
    const name = h('input', {id: 'newWorkspaceName', required: '', maxlength: '180', autocomplete: 'off'});
    const form = h('form', {}, [
      title,
      h('label', {for: 'newWorkspaceName', text: 'Project name'}),
      name,
      h('div', {class: 'project-actions'}, [close, h('button', {type: 'submit', class: 'btn primary', text: 'Create workspace'})])
    ]);
    form.addEventListener('submit', event => {
      event.preventDefault();
      const project = name.value.trim();
      if (!project) { name.focus(); return; }
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

function projectRunHistory() {
  const list = h('div', {'aria-live': 'polite'});
  const teams = {
    code: 'Source code', database: 'Database', infrastructure: 'Infrastructure',
    messaging: 'Messaging & caching', identity: 'Identity & access', telemetry: 'Telemetry',
    batch: 'Batch & scheduling', network: 'Network edge & partners',
    documents: 'Documents & diagrams', incidents: 'Incidents & changes'
  };

  function display(runs) {
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
    if (!runs.length) {
      list.textContent = 'No agent runs in this workspace yet. Switch projects to see reports from another workspace, or run the code agent team below.';
      return;
    }
    list.replaceChildren(h('ul', {class: 'project-choices'}, runs.map(run => {
      const team = teams[run.team] || 'Agent analysis';
      const follow = resume[run.team];
      const canResume = follow && agentCfg()[follow[0]] === run.id && M[follow[1]]?.id !== run.id;
      return h('li', {class: 'project-run'}, [
        h('span', {}, [
          h('b', {text: team}),
          h('small', {text: [run.source?.root, run.started ? new Date(run.started).toLocaleString() : '', statuses[run.status] || run.status, run.id].filter(Boolean).join(' · ')})
        ]),
        canResume && run.status !== 'interrupted' ? button('Resume results', () => follow[2](run.id), 'btn small') : null,
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
    if (!projectRuns.promise) {
      projectRuns.promise = agentFetch('/api/runs').then(runs => {
        if (!Array.isArray(runs)) throw new Error('The backend did not return a run list.');
        projectRuns.entries = runs;
        projectRuns.at = Date.now();
        return runs;
      }).finally(() => { projectRuns.promise = null; });
    }
    try {
      display(await projectRuns.promise);
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
    h('h2', {text: 'Agent reports'}),
    h('p', {class: 'muted', text: 'Read the reports saved for this workspace, with findings, citations and agent coverage.'}),
    h('div', {class: 'report-reader-actions'}, [
      button('Refresh report list', () => refreshHistory(true), 'btn small'),
      button('Open saved report JSON', () => savedFile.click(), 'btn small'), savedFile
    ]),
    list
  ]);
}
