/* Read-only viewer for saved analysis runs. All report content is rendered as text. */
function openAgentReport({id, cached, snapshot, team = 'Source code'}) {
  const dialog = dialogShell('Agent analysis report', team);
  const content = h('div', {class: 'report-reader', 'aria-live': 'polite', text: 'Loading report…'});
  dialog.append(content);
  if (!dialog.open) dialog.showModal();

  function show(result, isCached) {
    const report = result.report || '';
    const run = result.run || result;
    const source = result.source || {};
    const findings = Array.isArray(result.findings) ? result.findings : [];
    const coverage = result.coverage || {};
    const steps = run.steps || [];
    const requests = result.sentToModel || result.sent || [];
    const story = findings.filter(f => ['purpose','capability','journey','business-rule','domain-event'].includes(f.kind) && f.status !== 'rejected');
    const from = source.root || 'Source';
    const problems = Array.isArray(run.problems) ? run.problems : [];
    const status = run.status === 'done' && problems.length ? 'Finished with problems' : ({done: 'Complete', failed: 'Failed', running: 'Running'}[run.status] || run.status);
    const ruleStep = steps.find(s => s.n === 5);
    const businessOperations=reportSection(report,'How the business operates');
    const summary = [];
    if (story.some(f => f.basis === 'inferred')) {
      summary.push(h('p', {class: 'muted', text: 'The code team inferred likely business use from source evidence. Confirm it with a domain owner.'}));
    }
    if (run.error) summary.push(h('p', {class: 'conflict-note', text: run.error}));
    // Agents that failed or did not finish: the run carried on without them.
    if (problems.length) summary.push(h('div', {class: 'agent-problems', role: 'alert'}, [
      h('b', {text: problems.length === 1 ? 'One agent did not finish, so its findings are missing' : problems.length + ' agents did not finish, so their findings are missing'}),
      h('ul', {class: 'plain-list'}, problems.map(p => h('li', {text: (p.name || p.agent) + ' (step ' + p.step + '): ' + p.error})))
    ]));
    if (team === 'Source code' && !story.length && !/^1\. /m.test(businessOperations) && !/^## Executive summary$/m.test(report)) {
      let message = 'This run did not establish a business story or accepted business rules.';
      if (ruleStep?.status === 'skipped') message += ' ' + ruleStep.note;
      else if (requests.length) message += ' Model requests were recorded. The report contains no verified business explanation from them.';
      summary.push(h('p', {class: 'conflict-note', text: message}));
    }
    if (isCached) summary.push(h('p', {class: 'muted', text: 'Browser copy. Connect to the backend for the complete set of findings and file coverage.'}));
    const visibleFindings = findings.filter(item => item && typeof item === 'object');
    const roles = new Map((result.agents || []).map(agent => [agent.id, agent.name]));
    const label = id => roles.get(id) || AGENT_LABEL[id] || id;
    const roleList = roles.size ? [...roles] : Object.keys(result.counts || {}).map(id => [id, label(id)]);
    const body = h('div', {class: 'report-reader-body'});
    const views = [
      ...(businessOperations ? [['Business operations',()=>businessOperationsView(businessOperations)]] : []),
      ['Report', () => report ? reportSections(report) : h('p', {text: 'No written report is available for this run.'})],
      ['All findings (' + visibleFindings.length + ')', () => reportFindings(visibleFindings, label)],
      ['Telemetry', () => agentTelemetryView(result.telemetry || [])],
      ['Agents & coverage', () => h('div', {}, [
        reportDisclosure('Agents', roleList.map(([id, name]) => name + ' · ' + (result.counts?.[id] || 0) + ' findings'), true),
        reportDisclosure('Run steps', steps.map(step => step.title + ' · ' + step.status + (step.note ? ' · ' + step.note : '')), true),
        reportDisclosure('Files selected for closer reading', (result.reading || []).map(item => item.file + ' · ' + item.reason)),
        ...[['Files not read', coverage.notRead], ['Files skipped', coverage.skipped]].map(([title, items]) =>
          reportDisclosure(title, (Array.isArray(items) ? items : []).map(item => item.path + ' · ' + item.reason))),
        reportDisclosure('Questions for your experts', (result.questions || []).map(item => item.text)),
        reportDisclosure('Model requests', requests.map(item => label(item.agent) + ' · ' + item.model + ' · ' + (item.files || []).join(', ')))
      ])]
    ];
    const controls = h('div', {class: 'report-reader-actions', role: 'group', 'aria-label': 'Report views'});
    for (const [name, build] of views) {
      const tab = button(name, () => {
        for (const control of controls.children) control.setAttribute('aria-pressed', String(control === tab));
        body.replaceChildren(build());
      }, 'btn small');
      tab.setAttribute('aria-pressed', 'false');
      controls.append(tab);
    }
    content.replaceChildren(
      h('h3', {text: from}),
      h('p', {class: 'muted', text: [status, run.id, source.commit ? 'Commit ' + source.commit : '', Number.isInteger(coverage.files) ? coverage.analysed + ' of ' + coverage.files + ' files analysed' : ''].filter(Boolean).join(' · ')}),
      ...summary, controls, body,
      ...(report ? [button('Download report (.md)', () => downloadText('analysis-' + (id || run.id || 'report') + '.md', report), 'btn small')] : [])
    );
    controls.firstElementChild.click();
  }

  if (snapshot) { show(snapshot, false); return; }
  if (cached) show(cached, true);
  if (id) {
    agentFetch('/api/runs/' + encodeURIComponent(id))
      .then(result => { if (content.isConnected) show(result, false); })
      .catch(error => {
        if (content.isConnected && !cached) content.textContent = 'Could not open the report. ' + error.message;
      });
  }
}

function reportSection(markdown,title) {
  const lines=markdown.split(/\r?\n/);
  const start=lines.findIndex(line=>line==='## '+title);
  if(start<0)return '';
  const next=lines.findIndex((line,index)=>index>start&&/^#{1,2} /.test(line));
  return lines.slice(start,next<0?lines.length:next).join('\n');
}

function reportDisclosure(title, rows, open = false) {
  return h('details', {class: 'more-detail', open: open ? '' : null}, [
    h('summary', {text: title + ' (' + rows.length + ')'}),
    rows.length ? h('ul', {class: 'plain-list'}, rows.map(text => h('li', {text}))) : h('p', {class: 'muted', text: 'None recorded in this result.'})
  ]);
}

function reportSections(markdown) {
  const article = h('article', {class: 'report-prose'});
  let section = article;
  let lines = [];
  function flush() {
    if (lines.length) section.append(h('p', {class: 'report-lines', text: lines.join('\n')}));
    lines = [];
  }
  for (const line of markdown.split(/\r?\n/)) {
    const heading = /^#{1,3} (.+)$/.exec(line);
    if (heading) {
      flush();
      section = h('section', {}, [h('h4', {text: heading[1]})]);
      article.append(section);
    } else if (!line.trim()) {
      flush();
    } else {
      lines.push(line.replace(/`([^`]+)`|\*\*([^*]+)\*\*/g, (_, code, bold) => code || bold).replace(/^- /, '• '));
    }
  }
  flush();
  return article;
}

function reportFindings(findings, label) {
  const query = h('input', {type: 'search', 'aria-label': 'Search findings', placeholder: 'Search findings, file names or agents'});
  const filter = h('select', {'aria-label': 'Finding status'}, [
    h('option', {value: '', text: 'All statuses'}),
    ...['accepted', 'conflict', 'rejected'].map(status => h('option', {value: status, text: status[0].toUpperCase() + status.slice(1)}))
  ]);
  const rows = h('div', {'aria-live': 'polite'});
  let limit = 40;
  function draw() {
    const term = query.value.toLowerCase().trim();
    const matches = findings.filter(f => {
      if (filter.value && f.status !== filter.value) return false;
      const paths = (f.citations || []).map(c => c.file).join(' ');
      return !term || [f.id, f.title, f.detail, f.module, f.kind, label(f.agent), f.cite, paths].join(' ').toLowerCase().includes(term);
    });
    rows.replaceChildren(
      h('p', {class: 'muted', text: matches.length + ' matching findings'}),
      ...matches.slice(0, limit).map(f => h('details', {class: 'more-detail'}, [
        h('summary', {text: f.title || f.id}),
        h('p', {class: 'muted', text: [f.id, label(f.agent), f.module, f.status, f.basis, f.confidence ? 'Confidence ' + f.confidence : ''].filter(Boolean).join(' · ')}),
        f.detail ? h('p', {class: 'report-lines', text: f.detail}) : null,
        f.verdict ? h('p', {text: 'Verifier: ' + f.verdict}) : null,
        f.cite ? h('p', {text: f.cite}) : null,
        ...(f.citations || []).map(c => h('p', {class: 'report-lines', text: c.file + (c.line ? ':' + c.line : '') + (c.quote ? '\n' + c.quote : '')})),
        f.evidence?.length ? h('p', {text: 'Related findings: ' + f.evidence.join(', ')}) : null
      ])),
      ...(matches.length > limit ? [button('Show 40 more', () => { limit += 40; draw(); })] : [])
    );
  }
  query.addEventListener('input', () => { limit = 40; draw(); });
  filter.addEventListener('change', () => { limit = 40; draw(); });
  draw();
  return h('div', {}, [h('div', {class: 'report-reader-actions'}, [query, filter]), rows]);
}



function agentTelemetryView(attempts, selectedAgent = '') {
  if (!attempts.length) {
    return h('p', {class:'muted',text:'No agent timings were recorded for this run.'});
  }
  const states={done:'Completed',running:'Running',failed:'Failed',partial:'Completed with problems',interrupted:'Interrupted','timed out':'Timed out'};
  const names=[...new Map(attempts.map(a=>[a.agent,a.name||a.agent]))];
  const select=h('select',{'aria-label':'Filter by agent'},[
    h('option',{value:'',text:'All agents'}),
    ...names.map(([id,name])=>h('option',{value:id,text:name}))
  ]);
  if (names.some(([id])=>id===selectedAgent)) select.value=selectedAgent;
  const list=h('div',{class:'run-telemetry-list'});
  let limit=100;
  const formatTime=value=>value?new Date(value).toLocaleString():'—';
  function draw(){
    const filtered=attempts.filter(row=>!select.value||row.agent===select.value);
    list.replaceChildren(h('p',{class:'muted',text:filtered.length+' recorded attempts'}),
      h('table',{class:'scan-table run-telemetry-table'},[
        h('thead',{},[h('tr',{},['Agent','Step / batch','Status','Started','Ended','Elapsed','Model requests']
          .map(text=>h('th',{scope:'col',text})))]),
        h('tbody',{},filtered.slice(0,limit).map(a=>h('tr',{},[
          a.name||a.agent,
          [a.step?'Step '+a.step:'Entire run',a.batch?'Batch '+a.batch:'',a.part&&a.part!==a.agent&&a.part!=='run'?a.part:''].filter(Boolean).join(' · '),
          (states[a.status]||a.status)+(a.reused?' · Reused':''),
          formatTime(a.started),formatTime(a.ended),
          a.ended?(a.durationMs<1?'<1 ms':a.durationMs<1000?a.durationMs+' ms':(a.durationMs/1000).toFixed(2)+' s'):'—',
          String(a.modelRequests||0)
        ].map(text=>h('td',{text})))))
      ]),
      ...(filtered.length>limit?[button('Show more attempts',()=>{limit+=100;draw();},'btn small')]:[])
    );
  }
  select.addEventListener('change',()=>{limit=100;draw();});
  draw();
  return h('div',{},[h('label',{class:'context-control'},[h('span',{text:'Agent'}),select]),list]);
}

function openAgentTelemetry(id, team) {
  const dialog=dialogShell('Agent telemetry',team);
  const status=h('p',{class:'muted',role:'status',text:'Loading agent telemetry…'});
  const view=h('div');
  const controls=h('div',{class:'report-reader-actions'},[button('Refresh telemetry',load,'btn small')]);
  let fetching=false;
  let refreshTimer;
  dialog.addEventListener('close',()=>clearTimeout(refreshTimer),{once:true});
  async function load() {
    if (fetching) return;
    clearTimeout(refreshTimer);
    fetching=true;
    try {
      const data=await agentFetch('/api/runs/'+encodeURIComponent(id)+'/telemetry');
      if (!dialog.open) return;
      status.textContent=data.status==='running'?'Analysis is running. Timings refresh automatically.':
        data.status==='interrupted'?'The run was interrupted. Started attempts with no recorded finish are marked Interrupted.':'Timings are saved with this run.';
      const selectedAgent=view.querySelector('select')?.value||'';
      view.replaceChildren(agentTelemetryView(data.attempts||[],selectedAgent));
      if(data.status==='running')refreshTimer=setTimeout(load,2500);
    } catch(error) {
      if(dialog.open)status.textContent='Telemetry could not be loaded. '+error.message;
    } finally {
      fetching=false;
    }
  }
  dialog.append(h('div',{class:'report-reader'},[status,controls,view]));
  if(!dialog.open)dialog.showModal();
  load();
}


function businessOperationsView(markdown) {
  const article=h('article',{class:'business-operations'});
  let target=article;
  let list=null;
  for(const line of markdown.split(/\r?\n/)){
    const heading=/^(#{2,3}) (.+)$/.exec(line);
    if(heading){
      list=null;
      if(heading[1]==='##') {
        article.append(h('h3',{text:heading[2]}));
        target=article;
      } else {
        target=h('section',{class:'business-flow-card'},[h('h4',{text:heading[2]})]);
        article.append(target);
      }
      continue;
    }
    const row=/^(\d+\.|-) (.+)$/.exec(line);
    if(row){
      const type=row[1]==='-'?'ul':'ol';
      if(!list||list.tagName.toLowerCase()!==type){
        list=h(type,{class:'business-flow-list'});
        target.append(list);
      }
      const parts=/^(.+?) — (.+)$/.exec(row[2]);
      list.append(h('li',{},parts&&type==='ol'?[h('strong',{text:parts[1]}),' — '+parts[2]]:[row[2]]));
      continue;
    }
    list=null;
    if(line.trim()==='What to confirm')target.append(h('h5',{text:'What to confirm'}));
    else if(line.trim())target.append(h('p',{text:line.trim()}));
  }
  return article;
}
