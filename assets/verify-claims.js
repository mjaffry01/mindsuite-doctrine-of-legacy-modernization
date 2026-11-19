/* Verify before deciding (doctrine, README "Verify before deciding").
   Agent findings become claims. Each claim belongs to a verification role; each role is assigned to a person.
   The owner makes one offline form file per person, sends it themself, and imports the answers file that comes back.
   Nothing here talks to Claude: claims, answers, names and comments stay in this browser and the files the owner sends. */

// The roles that verify claims. `suggest` picks a person by their role in Rules & people.
const VERIFY_ROLES = [
  {id:'process', title:'Process owner', covers:'Business rules, processes and workflows', suggest:['Domain expert','Business owner']},
  {id:'data', title:'Data owner', covers:'Data meaning, ownership and quality', suggest:['Domain expert','Solution architect']},
  {id:'access', title:'Business owner, with security', covers:'Users, roles and access', suggest:['Business owner','Security & compliance']},
  {id:'custodian', title:'System custodian', covers:'Interfaces, batch jobs and schedules', suggest:['Operations','Engineering lead']},
  {id:'security', title:'Security or compliance officer', covers:'Security and compliance', suggest:['Security & compliance']},
  {id:'vendor', title:'Vendor or contract owner', covers:'Product versions, licences and support', suggest:['Vendor','Enterprise architect']}
];

// Which finding kinds are claims for people, and which role verifies them. Kinds not listed are technical facts
// that the verifier already checked against the evidence; they are not sent to people.
const CLAIM_KINDS = {
  process: 'business-rule capability journey purpose domain-event operating-flow workflow-trace actor term calendar rule-statement doc-process message-meaning business-calendar batch-process db-operating-flow constraint-rule freeze-period peak-period peak-months peak-hours quiet-window function-usage term-conflict intended-component',
  data: 'schema table shared-table shared-write sensitive-column data-cluster hub-table key-value-table generic-columns empty-tables large-table cache-as-store dataset legacy-file',
  access: 'role-check auth-method role-model admin-group admin-account target-role identity-provider mfa-coverage service-identity broad-groups idp-app',
  custodian: 'endpoint outbound-call file-exchange soap-service messaging integration partner partner-channel batch-job job job-stream critical-job schedule batch-window entry-point external-endpoint db-link remote-write external-call consumer-group unused-endpoint rarely-used never-ran failing-job long-job db-job host-role single-point-of-failure cics-transaction no-reader not-in-code slo-target cut-over connection',
  security: 'secret injection-risk weak-crypto csrf-disabled cors-any plaintext-protocol legacy-auth-open mfa-not-enforced policy-not-enforced password-column channel-no-tls default-user public-grant public-endpoint rdp-between-servers admin-channel',
  vendor: 'version runtime platform-eol image-eol broker-eol cache-eol db-version db-engine vendor-feature language app-server hypervisor os-mix'
};
const CLAIM_ROLE_OF = Object.fromEntries(Object.entries(CLAIM_KINDS).flatMap(([role, kinds]) => kinds.split(' ').map(k => [k, role])));
// Above this many findings of one kind in a run, they become one claim with examples (business claims are never grouped).
const CLAIM_GROUP_OVER = 5;
const CLAIM_STATUS = {proposed:'Not sent yet', open:'Waiting for an answer', confirmed:'Confirmed', corrected:'Corrected', rejected:'Rejected'};
const VERDICTS = {confirmed:'Confirmed', corrected:'Corrected', rejected:'Rejected', notmine:'Not mine'};
const verifyUI = {busy:false, message:'', status:'all', role:'all'};

function ensureVerify() {
  M.verify = M.verify || {};
  const V = M.verify;
  V.claims = V.claims || {};
  V.assign = V.assign || {};
  V.forms = V.forms || {};
  M.history = M.history || [];
  return V;
}

const kindLabel = kind => { const s = String(kind || '').replace(/-/g, ' '); return s.charAt(0).toUpperCase() + s.slice(1); };
const verifyRole = id => VERIFY_ROLES.find(r => r.id === id);
const claimText = f => [f.title, f.detail].filter(Boolean).join(': ');
// A secret's quote is never copied into a form: only where it is.
const citeOf = (f, c) => ({file:String(c.file || ''), line:c.line || 0, endLine:c.endLine || 0, quote: f.kind === 'secret' || f.kind === 'password-column' ? '' : String(c.quote || '').slice(0, 400)});

// Who answers a claim: a person it was forwarded to, else the person assigned to its role.
function claimPerson(c) { const V = ensureVerify(); return c.assignee || V.assign[c.role] || ''; }

function suggestAssignments() {
  const V = ensureVerify();
  for (const r of VERIFY_ROLES) {
    if (V.assign[r.id] && person(V.assign[r.id])) continue;
    const p = r.suggest.map(role => (M.people || []).find(x => x.role === role)).find(Boolean);
    if (p) V.assign[r.id] = p.id;
  }
}

// Turns one run's findings into claims (one per finding, or one per kind when there are many).
function claimsFromRun(team, runId, findings) {
  const out = [];
  const byKind = {};
  for (const f of findings) {
    if (!f || f.status === 'rejected' || !CLAIM_ROLE_OF[f.kind]) continue;
    (byKind[f.kind] = byKind[f.kind] || []).push(f);
  }
  for (const [kind, list] of Object.entries(byKind)) {
    const role = CLAIM_ROLE_OF[kind];
    const base = {team, runId, kind, role};
    if (role !== 'process' && list.length > CLAIM_GROUP_OVER) {
      const examples = [...new Set(list.map(f => f.title))].slice(0, 3).join('; ');
      out.push({...base, id: team + '|' + kind + '|*', findingIds: list.map(f => f.id), grouped: list.length,
        text: list.length + ' findings of this type, for example: ' + examples,
        conflict: list.some(f => f.status === 'conflict'),
        evidence: list.flatMap(f => (f.citations || []).slice(0, 1).map(c => citeOf(f, c))).slice(0, 8)});
      continue;
    }
    for (const f of list) {
      out.push({...base, id: team + '|' + kind + '|' + String(f.module || '') + '|' + String(f.title || f.id).toLowerCase(), findingIds: [f.id], module: String(f.module || ''),
        text: claimText(f), conflict: f.status === 'conflict', confidence: f.confidence || '',
        evidence: (f.citations || []).slice(0, 4).map(c => citeOf(f, c))});
    }
  }
  return out;
}

function logVerify(c, action, actor, reason, before) {
  M.history.push({field:'verify:' + c.id, at:new Date().toISOString(), actor, action, reason:reason || '', before, after:{status:c.status, answer:c.answer ? clone(c.answer) : null}});
}

// Reads the latest finished run of each team in this project and merges its claims, keeping earlier answers.
async function collectClaims() {
  if (verifyUI.busy) return;
  verifyUI.busy = true; verifyUI.message = 'Reading the agent reports…'; render();
  try {
    const V = ensureVerify();
    const runs = await loadWorkspaceRuns(true);
    const latest = {};
    for (const r of runs) if (r.status === 'done' && r.team && (!latest[r.team] || r.started > latest[r.team].started)) latest[r.team] = r;
    if (!Object.keys(latest).length) throw new Error('No finished agent runs in this project yet. Run a team from Record evidence first.');
    const seen = new Set();
    let added = 0, changed = 0;
    for (const [team, run] of Object.entries(latest)) {
      const s = await agentFetch('/api/runs/' + encodeURIComponent(run.id));
      for (const c of claimsFromRun(team, run.id, Array.isArray(s.findings) ? s.findings : [])) {
        seen.add(c.id);
        const old = V.claims[c.id];
        if (!old) { V.claims[c.id] = {...c, status:'proposed', answer:null}; added++; continue; }
        const textChanged = old.text !== c.text;
        Object.assign(old, {module:c.module || '', runId:c.runId, findingIds:c.findingIds, grouped:c.grouped, text:c.text, conflict:c.conflict, evidence:c.evidence, confidence:c.confidence});
        delete old.gone;
        if (textChanged && old.status !== 'proposed') {
          const before = {status:old.status};
          old.status = 'proposed'; old.answer = null;
          logVerify(old, 'reopened', 'MindSuite', 'The findings changed since they were answered', before);
          changed++;
        }
      }
    }
    for (const c of Object.values(V.claims)) if (latest[c.team] && !seen.has(c.id)) c.gone = true;
    suggestAssignments();
    persist();
    verifyUI.message = 'Claims collected: ' + added + ' new' + (changed ? ', ' + changed + ' reopened because their findings changed' : '') + '.';
  } catch (e) {
    verifyUI.message = 'Could not collect claims: ' + e.message;
  } finally {
    verifyUI.busy = false; render();
  }
}

const liveClaims = () => Object.values(ensureVerify().claims).filter(c => !c.gone);

// One self-contained HTML form for one person. Claims are embedded as JSON and shown as text only.
function verificationFormHTML(form, p, claims) {
  const data = {formId:form.id, project:M.project || '', person:{id:p.id, name:p.name, role:p.role}, made:form.made,
    roles:Object.fromEntries(VERIFY_ROLES.map(r => [r.id, r.title])),
    claims:claims.map(c => ({id:c.id, role:c.role, kind:kindLabel(c.kind), team:evidenceTypes[c.team] || c.team, text:c.text, conflict:!!c.conflict, grouped:c.grouped || 0, evidence:c.evidence || []}))};
  const json = JSON.stringify(data).replace(/</g, '\\u003c');
  return `<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Verify findings</title>
<style>
:root{--ink:#1d2433;--muted:#5b6476;--line:#dde2ea;--bg:#f6f7fa;--card:#fff;--accent:#4b5bd6;--ok:#2f7a4b;--warn:#a05a12;--bad:#a8323a}
*{box-sizing:border-box}body{margin:0;background:var(--bg);color:var(--ink);font:15px/1.5 system-ui,-apple-system,Segoe UI,Roboto,sans-serif}
main{max-width:860px;margin:0 auto;padding:24px 16px 80px}h1{font-size:24px;margin:0 0 4px}p{margin:6px 0}.muted{color:var(--muted)}
.intro,.claim{background:var(--card);border:1px solid var(--line);border-radius:10px;padding:16px;margin:14px 0}
.claim h2{font-size:16px;margin:0 0 6px}.tag{display:inline-block;font-size:12px;border:1px solid var(--line);border-radius:999px;padding:1px 8px;margin:0 6px 6px 0;color:var(--muted)}
.tag.warn{border-color:#efc8a0;color:var(--warn)}.ev{font:12.5px/1.45 ui-monospace,Consolas,monospace;background:var(--bg);border-radius:6px;padding:8px;margin:6px 0;white-space:pre-wrap;word-break:break-word}
.choices{display:flex;flex-wrap:wrap;gap:8px;margin:10px 0}.choices label{border:1px solid var(--line);border-radius:8px;padding:6px 10px;cursor:pointer}
.choices input{margin-right:6px}textarea,input[type=text]{width:100%;font:inherit;border:1px solid var(--line);border-radius:8px;padding:8px}
.need{color:var(--bad);font-size:13px}.bar{position:fixed;left:0;right:0;bottom:0;background:var(--card);border-top:1px solid var(--line);padding:10px 16px}
.bar div{max-width:860px;margin:0 auto;display:flex;gap:12px;align-items:center;justify-content:space-between;flex-wrap:wrap}
button{font:inherit;background:var(--accent);color:#fff;border:0;border-radius:8px;padding:9px 16px;cursor:pointer}
</style></head><body><main id="app"></main>
<div class="bar"><div><span id="count" class="muted"></span><button type="button" id="save">Save my answers</button></div></div>
<script type="application/json" id="data">${json}</script>
<script>
(function(){
var D=JSON.parse(document.getElementById('data').textContent);
var KEY='mindsuite-verify-'+D.formId, A={};
try{A=JSON.parse(localStorage.getItem(KEY)||'{}')||{};}catch(e){A={};}
function el(t,a,k){var e=document.createElement(t);a=a||{};for(var n in a){if(n==='text')e.textContent=a[n];else e.setAttribute(n,a[n]);}(k||[]).forEach(function(c){if(c)e.append(c);});return e;}
function keep(){try{localStorage.setItem(KEY,JSON.stringify(A));}catch(e){}count();}
function ok(a){return a&&a.verdict&&(a.verdict==='confirmed'||(a.verdict==='corrected'&&a.correction)||(a.verdict==='rejected'&&a.reason)||(a.verdict==='notmine'&&a.forwardTo));}
function count(){var n=D.claims.filter(function(c){return ok(A[c.id]);}).length;document.getElementById('count').textContent=n+' of '+D.claims.length+' answered. You can save part of them and finish later.';}
var app=document.getElementById('app');
app.append(el('h1',{text:'Please check these findings'}),
 el('div',{class:'intro'},[el('p',{text:'Hello '+D.person.name+'. While reviewing '+(D.project||'the system')+', the analysis found the claims below. You know this part of the system, so please say whether each one is right.'}),
  el('p',{class:'muted',text:'For each claim choose Confirmed (it is right), Corrected (partly right: give the right version), Rejected (wrong: say why) or Not mine (name who should answer). When you are done, click Save my answers and send the file back.'})]));
D.claims.forEach(function(c,i){
 var a=A[c.id]||{};
 var box=el('section',{class:'claim'});
 var tags=[el('span',{class:'tag',text:D.roles[c.role]||c.role}),el('span',{class:'tag',text:c.kind}),el('span',{class:'tag',text:c.team})];
 if(c.conflict)tags.push(el('span',{class:'tag warn',text:'The evidence disagrees with itself here'}));
 box.append(el('h2',{text:(i+1)+'. '+c.text}),el('div',{},tags));
 if(c.evidence.length){box.append(el('p',{class:'muted',text:'Where this comes from:'}));c.evidence.forEach(function(e){box.append(el('div',{class:'ev',text:e.file+(e.line?' line '+e.line+(e.endLine&&e.endLine!==e.line?'-'+e.endLine:''):'')+(e.quote?'\\n'+e.quote:'')}));});}
 var choices=el('div',{class:'choices',role:'radiogroup','aria-label':'Answer for claim '+(i+1)});
 var extra=el('div');
 var names={confirmed:'Confirmed',corrected:'Corrected',rejected:'Rejected',notmine:'Not mine'};
 function drawExtra(){
  extra.replaceChildren();var v=(A[c.id]||{}).verdict;
  var field={corrected:['correction','What is the right version?'],rejected:['reason','Why is it wrong?'],notmine:['forwardTo','Who should answer this? (name)']}[v];
  if(field){var input=el(field[0]==='forwardTo'?'input':'textarea',{'aria-label':field[1],placeholder:field[1],maxlength:'2000'});if(field[0]==='forwardTo')input.type='text';else input.rows=3;
   input.value=(A[c.id]||{})[field[0]]||'';input.addEventListener('input',function(){A[c.id][field[0]]=input.value.trim();keep();need();});extra.append(input);}
  var note=el('textarea',{'aria-label':'Comment (optional)',placeholder:'Comment (optional)',rows:'2',maxlength:'2000'});note.value=(A[c.id]||{}).comment||'';
  note.addEventListener('input',function(){A[c.id]=A[c.id]||{};A[c.id].comment=note.value.trim();keep();});
  if(v)extra.append(note);
  var warn=el('p',{class:'need'});extra.append(warn);
  function need(){var x=A[c.id];warn.textContent=x&&x.verdict&&!ok(x)?'Please fill in the box above.':'';}
  need();
 }
 Object.keys(names).forEach(function(k){
  var r=el('input',{type:'radio',name:'v'+i,value:k});if(a.verdict===k)r.checked=true;
  r.addEventListener('change',function(){A[c.id]={verdict:k,comment:(A[c.id]||{}).comment||''};keep();drawExtra();});
  choices.append(el('label',{},[r,document.createTextNode(names[k])]));
 });
 box.append(choices,extra);drawExtra();app.append(box);
});
count();
document.getElementById('save').addEventListener('click',function(){
 var answers=D.claims.filter(function(c){return ok(A[c.id]);}).map(function(c){var x=A[c.id];return {claimId:c.id,verdict:x.verdict,correction:x.correction||'',reason:x.reason||'',forwardTo:x.forwardTo||'',comment:x.comment||''};});
 if(!answers.length){alert('Answer at least one claim first.');return;}
 var out={mindsuiteVerification:1,formId:D.formId,project:D.project,person:D.person,savedAt:new Date().toISOString(),answers:answers};
 var url=URL.createObjectURL(new Blob([JSON.stringify(out,null,2)],{type:'application/json'}));
 var link=el('a',{href:url,download:'answers-'+D.person.name.replace(/[^A-Za-z0-9]+/g,'-')+'-'+D.formId.slice(0,8)+'.json'});
 document.body.append(link);link.click();link.remove();setTimeout(function(){URL.revokeObjectURL(url);},1000);
 alert('Your answers were saved as a file in your Downloads folder. Send that file back to the person who sent you this form.');
});
})();
</script></body></html>`;
}

function downloadFile(name, text, type) {
  const url = URL.createObjectURL(new Blob([text], {type}));
  const a = h('a', {href:url, download:name});
  document.body.append(a); a.click(); a.remove();
  setTimeout(() => URL.revokeObjectURL(url), 1000);
}

// Makes a form with every claim this person still has to answer, and marks those claims as waiting.
function makeVerificationForm(personId) {
  const p = person(personId);
  if (!p) return;
  const claims = liveClaims().filter(c => claimPerson(c) === personId && (c.status === 'proposed' || c.status === 'open'));
  if (!claims.length) { notice(p.name + ' has no claims to verify.'); return; }
  const V = ensureVerify();
  const form = {id:crypto.randomUUID(), person:personId, claimIds:claims.map(c => c.id), made:new Date().toISOString()};
  V.forms[form.id] = form;
  for (const c of claims) {
    if (c.status === 'proposed') { const before = {status:c.status}; c.status = 'open'; logVerify(c, 'sent', actorName(), 'Form made for ' + p.name, before); }
    c.formId = form.id;
  }
  persist();
  downloadFile('verify-' + (M.project || 'project').replace(/[^A-Za-z0-9]+/g, '-') + '-' + p.name.replace(/[^A-Za-z0-9]+/g, '-') + '.html', verificationFormHTML(form, p, claims), 'text/html');
  notice('Form for ' + p.name + ' saved to your downloads: ' + claims.length + ' claims. Send it to them yourself; MindSuite sends nothing.');
  render();
}

// Checks an answers file and applies it. Returns a short summary, or throws with what is wrong.
function applyVerificationAnswers(data) {
  const V = ensureVerify();
  if (!data || data.mindsuiteVerification !== 1 || !Array.isArray(data.answers)) throw new Error('This is not a MindSuite answers file.');
  const form = V.forms[data.formId];
  if (!form) throw new Error('This answers file belongs to a form made in another project or browser.');
  const p = person(form.person);
  const who = p ? p.name + ' (' + p.role + ')' : String(data.person?.name || 'Unknown').slice(0, 80);
  const text = v => String(v || '').trim().slice(0, 2000);
  const counts = {confirmed:0, corrected:0, rejected:0, notmine:0, skipped:0};
  for (const a of data.answers) {
    const c = V.claims[a?.claimId];
    if (!c || !form.claimIds.includes(a.claimId) || !VERDICTS[a.verdict]) { counts.skipped++; continue; }
    const answer = {verdict:a.verdict, by:form.person, byName:who, at:String(data.savedAt || new Date().toISOString()).slice(0, 40),
      correction:text(a.correction), reason:text(a.reason), forwardTo:text(a.forwardTo).slice(0, 80), comment:text(a.comment), formId:form.id};
    if ((a.verdict === 'corrected' && !answer.correction) || (a.verdict === 'rejected' && !answer.reason) || (a.verdict === 'notmine' && !answer.forwardTo)) { counts.skipped++; continue; }
    const before = {status:c.status, answer:c.answer ? clone(c.answer) : null};
    if (a.verdict === 'notmine') {
      const next = (M.people || []).find(x => x.name.trim().toLowerCase() === answer.forwardTo.toLowerCase());
      c.assignee = next ? next.id : '';
      c.status = 'proposed';
      c.answer = null;
      c.forwarded = {from:who, to:answer.forwardTo, found:!!next, at:answer.at};
      logVerify(c, 'forwarded', who, 'Not mine: ' + answer.forwardTo + (next ? '' : ' (not in Rules & people yet)'), before);
    } else {
      c.status = a.verdict;
      c.answer = answer;
      delete c.forwarded;
      logVerify(c, a.verdict, who, answer.correction || answer.reason || answer.comment, before);
    }
    counts[a.verdict]++;
  }
  persist();
  return who + ': ' + counts.confirmed + ' confirmed, ' + counts.corrected + ' corrected, ' + counts.rejected + ' rejected, ' + counts.notmine + ' passed on' + (counts.skipped ? ', ' + counts.skipped + ' skipped (not in this form or incomplete)' : '') + '.';
}

function importVerificationAnswers() {
  const input = h('input', {type:'file', accept:'.json,application/json', multiple:''});
  input.onchange = async () => {
    const lines = [];
    for (const file of input.files) {
      try {
        if (file.size > 5 * 1024 * 1024) throw new Error('the file is larger than 5 MB');
        lines.push(applyVerificationAnswers(JSON.parse(await file.text())));
      } catch (e) { lines.push(file.name + ': ' + e.message); }
    }
    verifyUI.message = lines.join(' ');
    render();
  };
  input.click();
}

function verifyStats() {
  const L = liveClaims();
  const by = s => L.filter(c => c.status === s).length;
  return {total:L.length, proposed:by('proposed'), open:by('open'), confirmed:by('confirmed'), corrected:by('corrected'), rejected:by('rejected')};
}

function verifyPage() {
  const V = ensureVerify();
  const S = verifyStats();
  const people = M.people || [];
  const L = liveClaims();
  const who = id => { const p = person(id); return p ? p.name : ''; };

  const assignRows = VERIFY_ROLES.map(r => {
    const sel = h('select', {class:'field compact', 'aria-label':'Person for ' + r.title, onchange:e => { V.assign[r.id] = e.target.value; persist(); render(); }},
      [h('option', {value:'', text:people.length ? 'Choose a person…' : 'Add people in Rules & people'}), ...people.map(p => h('option', {value:p.id, text:p.name + ' · ' + p.role}))]);
    sel.value = V.assign[r.id] || '';
    const n = L.filter(c => c.role === r.id).length;
    return h('tr', {}, [h('td', {}, [h('b', {text:r.title}), h('br'), h('small', {class:'muted', text:r.covers})]), h('td', {}, [sel]), h('td', {text:String(n)})]);
  });

  const toSend = {};
  for (const c of L) if (c.status === 'proposed' || c.status === 'open') { const id = claimPerson(c) || '-'; toSend[id] = (toSend[id] || 0) + 1; }
  const formRows = Object.entries(toSend).filter(([id]) => id !== '-' && person(id)).map(([id, n]) => {
    const waiting = L.filter(c => claimPerson(c) === id && c.status === 'open').length;
    return h('li', {}, [h('b', {text:who(id)}), ' · ' + n + ' claims' + (waiting ? ' (' + waiting + ' already sent, waiting)' : '') + ' ', button('Make form', () => makeVerificationForm(id), 'btn small')]);
  });
  const unassigned = toSend['-'] || 0;

  const statusSel = h('select', {class:'field compact', 'aria-label':'Show claims by status', onchange:e => { verifyUI.status = e.target.value; render(); }},
    [['all', 'All statuses'], ...Object.entries(CLAIM_STATUS)].map(([v, t]) => h('option', {value:v, text:t})));
  statusSel.value = verifyUI.status;
  const roleSel = h('select', {class:'field compact', 'aria-label':'Show claims by role', onchange:e => { verifyUI.role = e.target.value; render(); }},
    [['all', 'All roles'], ...VERIFY_ROLES.map(r => [r.id, r.title])].map(([v, t]) => h('option', {value:v, text:t})));
  roleSel.value = verifyUI.role;
  const shown = L.filter(c => (verifyUI.status === 'all' || c.status === verifyUI.status) && (verifyUI.role === 'all' || c.role === verifyUI.role));

  const answerText = c => {
    if (c.answer) {
      const a = c.answer;
      return VERDICTS[a.verdict] + ' by ' + a.byName + ', ' + String(a.at).slice(0, 10) + (a.correction ? ' · Right version: ' + a.correction : '') + (a.reason ? ' · Why: ' + a.reason : '') + (a.comment ? ' · ' + a.comment : '');
    }
    if (c.forwarded) return 'Passed on by ' + c.forwarded.from + ' to ' + c.forwarded.to + (c.forwarded.found ? '' : ' — add them in Rules & people, then make their form');
    return '';
  };

  return h('section', {class:'page verify-page'}, [
    h('div', {class:'discovery-section-head'}, [h('div', {}, [
      h('h1', {text:'Verify with people'}),
      h('p', {text:'A finding is a claim until the right person confirms it. Collect the claims from the agent reports, make a form for each person, send it yourself, and import the answers that come back. Decisions use only confirmed or corrected claims.'})])]),
    h('div', {class:'metrics'}, [['Claims', S.total], ['Not sent yet', S.proposed], ['Waiting', S.open], ['Confirmed', S.confirmed], ['Corrected', S.corrected], ['Rejected', S.rejected]]
      .map(([l, v]) => h('div', {class:'metric'}, [h('b', {text:String(v)}), h('span', {text:l})]))),
    verifyUI.message ? h('p', {class:'input-status', role:'status', text:verifyUI.message}) : null,

    h('h2', {text:'1. Collect claims'}),
    h('p', {class:'muted', text:'Reads the latest finished run of each agent team in this project. Business rules and processes are listed one by one; when there are more than ' + CLAIM_GROUP_OVER + ' technical findings of one type, they become one claim with examples. Answers you already have are kept.'}),
    h('div', {class:'report-reader-actions'}, [button(verifyUI.busy ? 'Collecting…' : (S.total ? 'Collect again' : 'Collect claims'), collectClaims, 'btn primary')]),

    h('h2', {text:'2. Who verifies what'}),
    people.length ? null : h('div', {class:'summary-callout'}, [h('b', {text:'Add people first'}), h('p', {text:'Add the people who know the system in Rules & people, then choose one for each role.'}), button('Open Rules & people', () => go('rules'), 'btn small')]),
    h('table', {class:'scan-table'}, [h('thead', {}, [h('tr', {}, [h('th', {text:'Role'}), h('th', {text:'Person'}), h('th', {text:'Claims'})])]), h('tbody', {}, assignRows)]),

    h('h2', {text:'3. Send forms'}),
    h('p', {class:'muted', text:'Each form is one file with only that person\'s claims. Send it by email or chat yourself; MindSuite sends nothing and Claude never sees the answers. The person opens it in a browser, answers, and sends back the answers file it saves.'}),
    formRows.length ? h('ul', {class:'plain-list'}, formRows) : h('p', {class:'muted', text:S.total ? 'No claims waiting for a person.' : 'Collect claims first.'}),
    unassigned ? h('p', {class:'input-status open', text:unassigned + ' claims have no person yet: choose one for their role above.'}) : null,

    h('h2', {text:'4. Import answers'}),
    h('div', {class:'report-reader-actions'}, [button('Import answers file', importVerificationAnswers, 'btn')]),

    h('h2', {text:'Claims'}),
    h('div', {class:'report-reader-actions'}, [statusSel, roleSel]),
    shown.length ? h('table', {class:'scan-table verify-table'}, [
      h('thead', {}, [h('tr', {}, [h('th', {text:'Claim'}), h('th', {text:'Role'}), h('th', {text:'Person'}), h('th', {text:'Status'})])]),
      h('tbody', {}, shown.slice(0, 300).map(c => h('tr', {class:'claim-' + c.status}, [
        h('td', {}, [h('b', {text:kindLabel(c.kind)}), ' · ' + (evidenceTypes[c.team] || c.team), h('br'), c.text,
          c.conflict ? h('small', {class:'conflict-note', text:' The evidence disagrees with itself.'}) : null,
          answerText(c) ? h('div', {class:'muted', text:answerText(c)}) : null]),
        h('td', {text:verifyRole(c.role)?.title || c.role}),
        h('td', {text:who(claimPerson(c)) || '—'}),
        h('td', {text:CLAIM_STATUS[c.status] || c.status})])))
    ]) : h('p', {class:'muted', text:S.total ? 'No claims match.' : 'No claims yet.'}),
    shown.length > 300 ? h('p', {class:'muted', text:'Showing the first 300 of ' + shown.length + '. Use the filters to narrow the list.'}) : null
  ]);
}

/* The gate in Target decisions (README "Verify before deciding", principles 4 and 6).
   A decision may rest only on confirmed or corrected claims. A claim nobody answered blocks the decision until the owner
   accepts it as an open assumption with a reason; it is then listed as a risk in the decision record.
   When a claim the decision rests on changes, the decision reopens. */
const claimIsBasis = c => c.status === 'confirmed' || c.status === 'corrected';
const claimSig = c => [c.status, c.answer ? c.answer.verdict + '@' + c.answer.at : '', c.text].join('|');

// Words that tie a claim to a bounded context: its name, its components' names and the paths they were found in.
function ctxTerms(ctx) {
  const comps = (M.inventory?.components || []).filter(x => x.context === ctx.id);
  const words = [ctx.name, ...comps.map(x => x.name), ...comps.flatMap(x => String(x.cite || '').split(/[;,\s]+/).filter(t => !t.endsWith(':')))];
  return [...new Set(words.map(t => String(t || '').toLowerCase().trim()).filter(t => t.length >= 4))];
}
function claimHits(claim, terms) {
  const hay = [claim.module, claim.text, ...(claim.evidence || []).map(e => e.file)].join(' ').toLowerCase();
  return terms.some(t => hay.includes(t));
}
// The claims a context's decision rests on: those that name it, plus those that name no context (they apply to the whole system).
function claimsForContext(ctx) {
  const L = liveClaims();
  const ctxs = M.inventory?.contexts || [];
  if (ctx.virtual || !ctxs.length) return L;
  const terms = Object.fromEntries(ctxs.map(x => [x.id, ctxTerms(x)]));
  return L.filter(c => claimHits(c, terms[ctx.id]) || !ctxs.some(x => claimHits(c, terms[x.id])));
}
function claimGate(ctx, d) {
  const list = claimsForContext(ctx);
  const acc = d.accepted || {};
  const open = list.filter(c => !claimIsBasis(c) && c.status !== 'rejected');
  return {list, basis:list.filter(claimIsBasis), rejected:list.filter(c => c.status === 'rejected'),
    risks:open.filter(c => acc[c.id]), blocking:open.filter(c => !acc[c.id])};
}
// What changed since the decision: claims that were answered, reopened or dropped, and new claims.
function claimsChanged(ctx, d) {
  if (!d.basis) return [];
  const V = ensureVerify();
  const out = [];
  for (const [id, sig] of Object.entries(d.basis)) {
    const c = V.claims[id];
    if (!c || c.gone) out.push('no longer found: ' + (c ? c.text : id).slice(0, 80));
    else if (claimSig(c) !== sig) out.push((CLAIM_STATUS[c.status] || c.status).toLowerCase() + ': ' + c.text.slice(0, 80));
  }
  for (const c of claimsForContext(ctx)) if (!(c.id in d.basis)) out.push('new: ' + c.text.slice(0, 80));
  return out;
}
// Called when the decision is recorded: what it rests on, and the open assumptions it accepts.
function gateRecord(ctx, d) {
  const g = claimGate(ctx, d);
  const acc = d.accepted || {};
  d.basis = Object.fromEntries(g.list.map(c => [c.id, claimSig(c)]));
  d.risks = g.risks.map(c => ({claimId:c.id, text:c.text, reason:acc[c.id].reason, by:acc[c.id].by, at:acc[c.id].at}));
  d.verified = g.basis.length;
  return g;
}
function acceptClaimsAsRisks(ctx, d, claims, reason) {
  reason = String(reason || '').trim();
  if (!reason) return false;
  d.accepted = d.accepted || {};
  const by = actorName(), at = new Date().toISOString();
  for (const c of claims) d.accepted[c.id] = {reason, by, at};
  M.history.push({field:'decision:' + ctx.id, at, actor:by, action:'accept-risk', reason, before:null, after:{claims:claims.map(c => c.id)}});
  persist(); render();
  notice(claims.length + ' open assumption' + (claims.length === 1 ? '' : 's') + ' accepted as risks for ' + ctx.name + '.');
  return true;
}
// Decisions that rest on accepted open assumptions (for the sign-off readiness list).
function gateRiskSummary(C) {
  return C.map(c => [c, decisionFor(c)]).filter(([, d]) => d.status === 'decided' && (d.risks || []).length).map(([c, d]) => c.name + ' (' + d.risks.length + ')');
}

function gatePanel(ctx, d) {
  const g = claimGate(ctx, d);
  const acc = d.accepted || {};
  const shortList = (list, extra) => h('ul', {class:'plain-list gate-list'}, list.slice(0, 50).map(c => h('li', {}, [h('b', {text:kindLabel(c.kind) + ': '}), c.text, extra ? h('small', {class:'muted', text:' · ' + extra(c)}) : null])));
  const answerOf = c => c.answer ? VERDICTS[c.answer.verdict] + ' by ' + c.answer.byName + (c.answer.correction ? ': ' + c.answer.correction : '') + (c.answer.reason ? ': ' + c.answer.reason : '') : '';
  const reason = h('input', {class:'field compact', placeholder:'Why decide without an answer? (required)', 'aria-label':'Reason to accept open assumptions for ' + ctx.name, maxlength:'240'});
  const err = h('p', {class:'conflict-note', hidden:''});
  const head = g.list.length
    ? g.basis.length + ' confirmed or corrected · ' + g.rejected.length + ' rejected · ' + g.risks.length + ' accepted as risks · ' + g.blocking.length + ' not verified'
    : 'No claims collected for this context yet';
  return h('div', {class:'dim-row gate-row'}, [
    h('div', {class:'dim-head'}, [h('b', {text:'Verified claims'}), h('small', {text:head})]),
    d.reopenWhy ? h('p', {class:'conflict-note', text:'Reopened: ' + d.reopenWhy}) : null,
    !g.list.length ? h('p', {class:'muted', text:'Collect claims in Verify with people so this decision rests on findings people have confirmed. Until then it rests on unverified evidence.'}) : null,
    g.blocking.length ? h('div', {}, [
      h('p', {class:'input-status open', text:g.blocking.length + ' claim' + (g.blocking.length === 1 ? ' is' : 's are') + ' not verified yet. Verify them with people, or accept them as open assumptions with a reason: they are then listed as risks in the decision record.'}),
      h('div', {class:'people-add'}, [button('Verify with people', () => go('verify'), 'btn small'), reason,
        button('Accept ' + g.blocking.length + ' as risks', () => { if (!acceptClaimsAsRisks(ctx, d, g.blocking, reason.value)) { err.hidden = false; err.textContent = 'Write why the decision can go ahead without these answers.'; reason.focus(); } }, 'btn small')]),
      err]) : null,
    g.list.length ? h('details', {class:'compare'}, [h('summary', {text:'Show the claims this decision rests on'}),
      g.basis.length ? h('div', {}, [h('b', {text:'Confirmed or corrected'}), shortList(g.basis, answerOf)]) : null,
      g.risks.length ? h('div', {}, [h('b', {text:'Open assumptions accepted as risks'}), shortList(g.risks, c => 'Accepted by ' + acc[c.id].by + ': ' + acc[c.id].reason)]) : null,
      g.rejected.length ? h('div', {}, [h('b', {text:'Rejected (kept with the reason, not used)'}), shortList(g.rejected, answerOf)]) : null,
      g.blocking.length ? h('div', {}, [h('b', {text:'Not verified yet'}), shortList(g.blocking, c => CLAIM_STATUS[c.status] || c.status)]) : null,
      g.list.length > 50 ? h('small', {class:'muted', text:'Each list shows up to 50 claims; see all of them in Verify with people.'}) : null]) : null
  ]);
}

/* Approach suggestions from verified claims (README "Verify before deciding", principle 5; Policy: what the AI may see).
   Only claims people confirmed or corrected are sent, as claim text, kind, role, verdict and the right version of a
   corrected claim, with the context's strategic value and criticality. Names, comments, rejected and unverified
   claims never leave this browser. Each request needs the owner's tick on that card; the tick covers one request.
   Claude suggests; the person deciding records the decision. */
const suggestUI = {consent:{}, busy:{}, error:{}};

function suggestPayload(ctx, d) {
  const g = claimGate(ctx, d);
  const sent = g.basis.slice(0, 300);
  return {sent, body:{strategic:String(ctx.strategic || ''), criticality:String(ctx.criticality || ''),
    claims:sent.map(c => ({kind:c.kind, role:c.role, text:String(c.text).slice(0, 600), verdict:c.status,
      correction:c.status === 'corrected' ? String(c.answer?.correction || '').slice(0, 600) : ''}))}};
}
const suggestSig = claims => claims.map(c => c.id + '=' + claimSig(c)).join('\n');

async function requestSuggestion(ctx, d) {
  if (!suggestUI.consent[ctx.id] || suggestUI.busy[ctx.id]) return;
  const {sent, body} = suggestPayload(ctx, d);
  if (!sent.length) return;
  // Consent covers one request only.
  delete suggestUI.consent[ctx.id];
  suggestUI.busy[ctx.id] = true; delete suggestUI.error[ctx.id]; render();
  try {
    const out = await agentFetch('/api/decide/suggest', {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify(body)});
    d.ai = {at:new Date().toISOString(), model:out.model || '', summary:out.summary || '', suggestions:out.suggestions || [],
      claimIds:sent.map(c => c.id), claimTexts:sent.map(c => c.text), sig:suggestSig(sent)};
    M.history.push({field:'decision:' + ctx.id, at:d.ai.at, actor:actorName(), action:'ai-suggest',
      reason:sent.length + ' verified claims sent to Claude (' + (out.model || 'model') + ')', before:null, after:{suggestions:d.ai.suggestions.map(s => s.approach)}});
    persist();
  } catch (e) {
    suggestUI.error[ctx.id] = e.message;
  } finally {
    delete suggestUI.busy[ctx.id]; render();
  }
}

function aiPanel(ctx, d) {
  const {sent, body} = suggestPayload(ctx, d);
  const busy = !!suggestUI.busy[ctx.id];
  const ai = d.ai;
  const stale = ai && ai.sig !== suggestSig(sent);
  const FIT = {strong:'Strong fit', possible:'Possible fit', weak:'Weak fit'};
  const claimRef = n => (ai.claimTexts || [])[n - 1] || 'claim ' + n;
  const items = ai ? (ai.suggestions || []).map((s, i) => h('li', {class:'ai-suggestion'}, [
    h('div', {class:'dim-head'}, [h('b', {text:(i + 1) + '. ' + apprLabel(s.approach)}), h('small', {text:FIT[s.fit] || s.fit})]),
    h('ul', {class:'plain-list'}, (s.reasons || []).map(r => h('li', {}, [r.text,
      (r.claims || []).length ? h('small', {class:'muted', text:' · rests on: ' + r.claims.map(n => '“' + claimRef(n).slice(0, 90) + '”').join('; ')}) : h('small', {class:'muted', text:' · from the strategic value or criticality'})]))),
    s.changes ? h('p', {class:'muted', text:'What would change this: ' + s.changes}) : null,
    d.approach === s.approach ? h('small', {class:'muted', text:'This is the chosen approach.'})
      : button('Use this approach', () => changeDecision(ctx, d, () => { d.approach = s.approach; if (!ctx.virtual) setDecisionQuiet(ctx, 'approach', s.approach); }), 'btn small')])) : [];
  return h('div', {class:'dim-row ai-row'}, [
    h('div', {class:'dim-head'}, [h('b', {text:'Approach suggestions from Claude'}), h('small', {text:'Claude suggests; the person deciding records the decision'})]),
    !sent.length ? h('p', {class:'muted', text:'Claude can suggest approaches once claims for this context are confirmed or corrected in Verify with people. Unverified and rejected claims are never sent.'}) : h('div', {}, [
      h('div', {class:'type-chips'}, [agentChip('Allow the ' + sent.length + ' verified claim' + (sent.length === 1 ? '' : 's') + ' (text and verdict only) to be sent to Claude for one suggestion', !!suggestUI.consent[ctx.id], v => { suggestUI.consent[ctx.id] = v; render(); })]),
      h('details', {class:'compare'}, [h('summary', {text:'See exactly what is sent'}), h('pre', {class:'ai-payload', text:JSON.stringify(body, null, 2)}),
        h('small', {class:'muted', text:'Not sent: names, comments, who answered, rejected claims, claims not verified yet, evidence quotes and file paths.'})]),
      h('div', {class:'report-reader-actions'}, [h('button', {class:'btn small primary', type:'button', disabled:!suggestUI.consent[ctx.id] || busy ? '' : null,
        onclick:() => requestSuggestion(ctx, d), text:busy ? 'Asking Claude…' : ai ? 'Ask again' : 'Suggest approaches'})])]),
    suggestUI.error[ctx.id] ? h('p', {class:'conflict-note', text:'No suggestion: ' + suggestUI.error[ctx.id]}) : null,
    ai ? h('div', {class:'ai-result'}, [
      h('p', {class:'muted', text:'Suggested ' + new Date(ai.at).toLocaleString() + ' by ' + (ai.model || 'Claude') + ' from ' + (ai.claimIds || []).length + ' verified claims.'}),
      stale ? h('p', {class:'input-status open', text:'The verified claims changed since this suggestion. Ask again to use the current ones.'}) : null,
      ai.summary ? h('p', {text:ai.summary}) : null,
      h('ol', {class:'plain-list'}, items)]) : null
  ]);
}
