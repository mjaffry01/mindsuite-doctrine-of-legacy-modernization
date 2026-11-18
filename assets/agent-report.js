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
    const status = {done: 'Complete', failed: 'Failed', running: 'Running'}[run.status] || run.status;
    const ruleStep = steps.find(s => s.n === 5);
    const summary = [];
    if (story.some(f => f.basis === 'inferred')) {
      summary.push(h('p', {class: 'muted', text: 'The code team inferred likely business use from source evidence. Confirm it with a domain owner.'}));
    }
    if (run.error) summary.push(h('p', {class: 'conflict-note', text: run.error}));
    if (team === 'Source code' && !story.length && !/^## Executive summary$/m.test(report)) {
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
      ['Report', () => report ? reportSections(report) : h('p', {text: 'No written report is available for this run.'})],
      ['All findings (' + visibleFindings.length + ')', () => reportFindings(visibleFindings, label)],
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

