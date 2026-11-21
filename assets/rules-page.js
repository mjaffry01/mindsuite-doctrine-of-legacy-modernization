/* MindSuite web app · 05 Rules & people, in three tabs
   1 Who decides · 2 Technology rules · 3 Priorities & limits.
   It reuses the sections built by rulesPage(), peopleSection(), constraintsSection() and
   businessCaseSection() in index.html, so every answer is stored and scored exactly as before.
   Technology rules are chips that open a rule editor; assets/rule-catalog.js holds the model.
   Loaded by index.html before the main script; uses its helpers at call time. The router calls
   rulesTabsPage() for the 'rules' screen. */
let rulesTab='people';

(function rulesPageStyle(){
 if(document.getElementById('rules-tabs-style'))return;
 const css=`
.rules-tabs .tab{display:inline-flex;flex-direction:column;align-items:flex-start;gap:2px;padding:8px 14px;text-align:left}
.rules-tabs .tab small{font-weight:400;color:var(--muted)}
.rules-tabs .tab small.need{color:#b45309}
.rules-why{margin:4px 0 14px}
.stance-chips{display:flex;flex-wrap:wrap;gap:6px;margin:6px 0 14px}
.stance-chip{border:1px solid #d6dde8;border-radius:999px;padding:5px 12px;font-size:12px;background:#fff;color:var(--ink);cursor:pointer}
.stance-chip.standard{border-color:var(--accent);background:#e5ecff;color:var(--accent);font-weight:700}
.stance-chip.preferred{border-color:var(--green);background:#e8f6ee;color:#166534;font-weight:600}
.stance-chip.approval{border-color:#e9c46a;background:#fff8e6;color:#8a5a00}
.stance-chip.retiring{border-color:#f0b37e;background:#fff2e8;color:#9a3412}
.stance-chip.excluded{border-color:#e5b4b4;background:#fdf0f0;color:#9f1239;text-decoration:line-through}
.stance-chip.own{border-style:dashed}
.stance-group-head{display:flex;align-items:center;gap:10px}
.legend-item.standard{color:var(--accent)}.legend-item.preferred{color:#166534}.legend-item.approval{color:#8a5a00}.legend-item.retiring{color:#9a3412}.legend-item.excluded{color:#9f1239}
.rules-toolbar{display:flex;flex-wrap:wrap;align-items:center;gap:8px;margin:0 0 12px}
.rules-toolbar .field.compact{width:auto;min-width:220px}
.rules-toolbar-sep{width:1px;height:22px;background:var(--line);margin:0 4px}
.rules-csv-help{flex-basis:100%;font-size:12px}
.rules-csv-help summary{cursor:pointer;color:var(--muted)}
.rule-states{display:grid;gap:6px;margin-bottom:12px}
.rule-state{display:flex;gap:10px;align-items:flex-start;border:1px solid var(--line);border-radius:8px;padding:8px 10px;cursor:pointer}
.rule-state small{display:block;color:var(--muted)}
.rule-state:has(input:checked){border-color:var(--accent);background:#f3f6ff}
.stance-group-head small{flex:1}
.preset-preview[hidden]{display:none}
.preset-preview{flex-basis:100%;display:flex;flex-wrap:wrap;gap:6px;align-items:center;padding:8px 10px;border:1px dashed var(--line);border-radius:8px}
.preset-preview .stance-chip{cursor:default}
.rules-force{margin:4px 0 14px}.rules-force .stance-chip{cursor:default;padding:2px 8px}.rules-none{margin:4px 0 14px}
.rules-browse{display:flex;flex-wrap:wrap;align-items:center;justify-content:space-between;gap:10px;margin:6px 0 10px}
.rules-filter{display:flex;flex-wrap:wrap;gap:8px;align-items:center}.rules-filter .field.compact{min-width:260px;width:auto}
.warn-text{color:#8a5a00}
.stance-legend{display:flex;flex-wrap:wrap;gap:12px;font-size:12px;color:var(--muted);margin:2px 0 12px}
.rules-money summary{cursor:pointer;font-weight:600;padding:4px 0}
.rules-money[open] summary{margin-bottom:6px}`;
 document.head.append(Object.assign(document.createElement('style'),{id:'rules-tabs-style',textContent:css}));
})();

/* ---- Technology rules: chips open a rule editor (state, reason, owner, dates) ---- */
(function(){if(document.getElementById('rule-suggest-style'))return;document.head.append(Object.assign(document.createElement('style'),{id:'rule-suggest-style',textContent:'.rule-rec{margin:6px 0 10px;padding:10px 12px;border:1px solid #d9d2ff;border-radius:10px;background:#faf9ff}.rr-head{display:flex;flex-wrap:wrap;gap:6px 10px;align-items:center}.rr-label{font-size:12px;font-weight:700;letter-spacing:.06em;text-transform:uppercase;color:#5a46c8}.rr-pick{display:inline-flex;gap:6px;align-items:center;font-size:15.5px}.rr-badge{font-size:12px;padding:2px 8px;border-radius:10px}.rr-badge.yes{background:#e6f6ec;color:#1d6b3a}.rr-badge.no{background:#fff4e0;color:#7a4b00}.rr-sure{font-size:12.5px;padding:2px 8px;border-radius:10px;background:#eef0f5;color:#3c4257}.rr-sure.slight,.rr-sure.tie{background:#fff4e0;color:#7a4b00}.rr-head .btn{margin-left:auto}.rr-body{display:grid;grid-template-columns:minmax(0,1fr) minmax(0,1.2fr);gap:8px 18px;margin-top:8px;font-size:13.5px}@media(max-width:900px){.rr-body{grid-template-columns:1fr}}.rr-reasons{list-style:none;margin:4px 0;padding:0}.rr-reasons li{margin:3px 0}.rr-reasons li.pos{color:#1d6b3a}.rr-reasons li.neg{color:#9b1c1c}.rr-crit{display:inline-block;min-width:78px;font-size:11.5px;font-weight:600;color:#5d6377;text-transform:uppercase;letter-spacing:.04em}.rr-note{margin:4px 0 0;color:#3c4257}.rr-cmp{width:100%;border-collapse:collapse;margin:4px 0}.rr-cmp th,.rr-cmp td{border-top:1px solid #ece9fb;padding:4px 6px;text-align:left;vertical-align:top}.rr-cmp th{font-weight:600;white-space:nowrap}.rr-score{white-space:nowrap;color:#5d6377;width:70px}'}));})();
function ruleChipText(dim,id,label){
 const R=M.decide.rules,st=R.stance[dim][id]||'allowed',info=(R.ruleInfo[dim]||{})[id]||{};
 const today=new Date().toISOString().slice(0,10);const overdue=info.reviewBy&&info.reviewBy<today;
 return (RULE_STATE[st][2]?RULE_STATE[st][2]+' ':'')+label+(st==='retiring'&&info.exitBy?' · exit '+info.exitBy:'')+(overdue?' · review due':'');
}
const AREA_HELP={architecture:'How the new system is structured and deployed.',language:'What new code is written in and runs on.',database:'Where the data lives after the move.',integration:'How it connects to other systems. A context can use several.',cloud:'Where it runs.'};
const PRODUCT_AREA_HELP={architecture:'How the first release is put together.',language:'What the team will build and run the product with.',database:'Where the product will store its data.',integration:'How clients, partners and services will connect.',cloud:'Where the product will run at launch.'};
function stanceChips(dim){
 const R=M.decide.rules;const set=TARGETS[dim].opts.filter(([id])=>(R.stance[dim][id]||'allowed')!=='allowed').length;
 return h('div',{class:'stance-group','data-area':dim},[
  h('div',{class:'stance-group-head'},[h('b',{text:TARGETS[dim].title}),h('small',{class:'muted',text:(isNewProduct()?PRODUCT_AREA_HELP:AREA_HELP)[dim]+(set?' · '+set+' rule'+(set===1?'':'s'):'')}),button('+ Add your own',()=>customOptionDialog(dim),'linkbtn')]),
  ruleSuggestionLine(dim),
  h('div',{class:'stance-chips'},TARGETS[dim].opts.map(([id,label,hint])=>{
   const st=R.stance[dim][id]||'allowed',info=(R.ruleInfo[dim]||{})[id]||{};
   return h('button',{type:'button',class:'stance-chip '+st+(String(id).startsWith('org-')?' own':''),'data-find':(label+' '+(hint||'')+' '+(info.reason||'')).toLowerCase(),'data-rule':st,
    'aria-label':label+': '+RULE_STATE[st][1]+(info.reason?'. '+info.reason:'')+'. Click to edit the rule.',
    title:(hint?hint+'\n':'')+RULE_STATE[st][1]+(info.reason?' · '+info.reason:'')+' · click to edit',
    onclick:()=>ruleDialog(dim,id)},[ruleChipText(dim,id,label)]);
  }))
 ]);
}
/* Recommendation card per area: what the brief and your rules point to, how sure, why, and how it compares
   with the other options. Same suggestion as Design options step 1 (platformSuggest); scores from rankOptions
   on the whole platform (reasons x weights from What matters most). Rules are the organization's policy;
   this card is advice for this project, and "Make it preferred" turns it into a rule. */
const RULE_CRIT={fit:'Fit',effort:'Effort',risk:'Risk',cost:'Cost',skills:'Skills',portability:'Portability',rule:'Your rules'};
const RULE_DEVICE_SIDE={database:{sqlite:'stores data on the device, not on the shared platform'},architecture:{clientapi:'describes the client apps; the platform itself still needs an architecture'}};
function ruleSuggestionLine(dim){
 if(typeof platformSuggest!=='function'||!['architecture','language','database','integration','cloud'].includes(dim))return null;
 let S,ranked;
 try{S=platformSuggest();ranked=rankOptions(dim,typeof PLATFORM_VCTX!=='undefined'?PLATFORM_VCTX:{id:'platform',virtual:true},{approach:'build',choices:{}});}catch(e){return null;}
 const R=M.decide.rules,ids=[].concat(S.v[dim]||[]).filter(Boolean);if(!ids.length)return null;
 const lab=id=>((TARGETS[dim].opts.find(o=>o[0]===id)||[,id])[1]);
 const stateOf=id=>R.stance[dim][id]||'allowed';
 const open=ids.filter(id=>!['preferred','standard'].includes(stateOf(id)));
 const skip=RULE_DEVICE_SIDE[dim]||{};
 const valid=ranked.filter(o=>!o.excluded&&!skip[o.id]);
 const top=valid.find(o=>o.id===ids[0])||valid[0];
 const others=valid.filter(o=>!ids.includes(o.id));
 const gap=top&&others[0]?Math.round((top.score-others[0].score)*10)/10:null;
 const tied=others.filter(o=>top&&o.score===top.score);
 const sure=!TARGETS[dim].multi&&tied.length?['tie','Tie with '+tied.map(o=>o.label).join(', ')+'. Decided by: '+(S.why[dim]||'the first in the list')]
  :gap==null?['clear','Only option that fits']:gap>=6?['clear','Clear lead (+'+gap+' points)']:gap>0?['slight','Slight lead (+'+gap+' points): worth a second look']:['clear','Recommended'];
 const make=()=>{for(const id of open){const b=stateOf(id);R.stance[dim][id]='preferred';R.ruleInfo[dim]=R.ruleInfo[dim]||{};R.ruleInfo[dim][id]={...(R.ruleInfo[dim][id]||{}),reason:'Recommended for this '+(isNewProduct()?'product':'system')+': '+(S.why[dim]||'best score against the brief and your rules')};logInput('rule:'+dim+':'+id,b,'preferred');}persist();render();};
 const reason=r=>h('li',{class:r.v>0?'pos':r.v<0?'neg':''},[h('span',{class:'rr-crit',text:RULE_CRIT[r.w]||r.w}),(r.v>0?'+ ':r.v<0?'− ':'')+r.t]);
 const chosen=ids.map(id=>valid.find(o=>o.id===id)||ranked.find(o=>o.id===id)).filter(Boolean);
 const ruleBadge=id=>{const st=stateOf(id);return st==='allowed'?h('span',{class:'rr-badge no',text:'Not yet a rule'}):h('span',{class:'rr-badge yes',text:'In your rules: '+(RULE_STATE[st]?RULE_STATE[st][1]:st)});};
 const cmpRows=[...others.slice(0,3),...ranked.filter(o=>o.excluded||skip[o.id]).slice(0,2)];
 const pts=n=>(n>0?'+':'')+Math.round(n*10)/10;
 return h('div',{class:'rule-rec'},[
  h('div',{class:'rr-head'},[h('span',{class:'rr-label',text:'Recommendation'}),
   ...ids.map(id=>h('span',{class:'rr-pick'},[h('b',{text:lab(id)}),ruleBadge(id)])),
   h('span',{class:'rr-sure '+sure[0],text:sure[1]}),
   open.length?button('Make it preferred',make,'btn small primary'):null]),
  h('div',{class:'rr-body'},[
   h('div',{},[h('b',{text:'Why'}),
    chosen.length&&chosen.some(o=>o.reasons.length)?h('ul',{class:'rr-reasons'},chosen.flatMap(o=>o.reasons.map(reason))):h('p',{class:'muted',text:'No rule or fact in the brief favours one option; this is the default for this kind of product.'}),
    S.why[dim]?h('p',{class:'rr-note',text:'In short: '+S.why[dim]+'.'}):null]),
   h('div',{},[h('b',{text:'Compared with'}),
    cmpRows.length?h('table',{class:'rr-cmp'},h('tbody',{},cmpRows.map(o=>h('tr',{},[h('th',{scope:'row',text:o.label}),
     h('td',{class:'rr-score',text:o.excluded?'excluded':skip[o.id]?'n/a':pts(o.score)}),
     h('td',{text:o.excluded?o.excluded:skip[o.id]?'Not used for the shared platform: '+skip[o.id]:(o.reasons.length?o.reasons.slice(0,2).map(r=>(r.v>0?'+ ':r.v<0?'− ':'')+r.t).join('; '):'nothing in the brief or your rules favours it')})])))):h('p',{class:'muted',text:'No other option fits.'}),
    h('small',{class:'muted',text:'Points = each reason × how much its criterion matters (What matters most, under Priorities & limits).'+(top?' Recommendation: '+pts(top.score)+' points.':'')})])])]);
}
function ruleDialog(dim,id){
 const R=M.decide.rules,o=TARGETS[dim].opts.find(x=>x[0]===id);if(!o)return;
 const info={...((R.ruleInfo[dim]||{})[id]||{})};let state=R.stance[dim][id]||'allowed';
 const dialog=dialogShell(TARGETS[dim].title+' · '+o[1],o[2]||"Set your organization's rule for this option.");
 const exitRow=h('div',{});
 const drawExit=()=>exitRow.replaceChildren(...(state==='retiring'?[inputControl('Exit by (existing use must end)',info.exitBy,v=>{info.exitBy=v;},'date')]:[]));
 const states=h('div',{class:'rule-states',role:'radiogroup','aria-label':'Rule'},RULE_STATES.map(([v,label,icon,help])=>h('label',{class:'rule-state '+v},[
  h('input',{type:'radio',name:'rule-state',value:v,checked:v===state?'':null,onchange:()=>{state=v;drawExit();}}),
  h('span',{},[h('b',{text:(icon?icon+' ':'')+label}),h('small',{text:help})])])));
 const people=M.people||[];
 const owner=h('select',{class:'field compact','aria-label':'Rule owner'},[h('option',{value:'',text:people.length?'Not named':'Add people in tab 1 to name an owner'}),...people.map(p=>h('option',{value:p.id,text:p.name+' · '+p.role}))]);owner.value=info.owner||'';
 drawExit();
 dialog.append(h('div',{class:'drawer-content'},[states,exitRow,
  inputControl('Reason or source',info.reason,v=>{info.reason=v;},'text','e.g. Group cloud policy 2026; licence ends 2028; PCI scope'),
  h('label',{class:'context-control'},[h('span',{text:'Owner'}),owner]),
  inputControl('Review by',info.reviewBy,v=>{info.reviewBy=v;},'date'),
  h('p',{class:'muted',text:'Your project history records this rule. Its reason appears in '+(isNewProduct()?'Design options.':'Target decisions.')})]));
 const own=String(id).startsWith('org-');
 dialog.append(h('div',{class:'drawer-footer'},[
  own?button('Remove option',()=>{if(!confirm('Remove "'+o[1]+'"? Decisions that chose it keep their record.'))return;R.custom[dim]=(R.custom[dim]||[]).filter(c=>c.id!==id);delete R.stance[dim][id];delete R.ruleInfo[dim][id];logInput('rule-option:'+dim+':'+id,o[1],null);syncRuleCatalog(R);persist();dialog.close();render();},'btn small'):null,
  button('Cancel',()=>dialog.close()),
  button('Save rule',()=>{info.owner=owner.value;if(state!=='retiring')delete info.exitBy;delete info.at;delete info.by;setRule(dim,id,state,info);persist();dialog.close();render();},'btn primary')]));
 dialog.showModal();
}
function customOptionDialog(dim){
 const dialog=dialogShell('Add an option · '+TARGETS[dim].title,'For technology your organization uses that is not in the list, such as an internal platform or a product you already own.');
 const name=h('input',{class:'field',maxlength:'60',placeholder:'e.g. Atlas internal platform, MongoDB Atlas, Yotta Cloud','aria-label':'Option name'});
 const hint=h('input',{class:'field',maxlength:'120',placeholder:'Optional: one line on what it is','aria-label':'Description'});
 const err=h('p',{class:'conflict-note',hidden:''});
 dialog.append(h('div',{class:'drawer-content'},[h('label',{class:'context-control'},[h('span',{text:'Name *'}),name]),h('label',{class:'context-control'},[h('span',{text:'Description'}),hint]),
  h('p',{class:'muted',text:'Your own options are ranked on your rules and team skills only: the tool has no built-in evidence for them. Set a rule on it next.'}),err]));
 dialog.append(h('div',{class:'drawer-footer'},[button('Cancel',()=>dialog.close()),button('Add option',()=>{
  const label=name.value.trim();if(!label){name.focus();return;}
  if(TARGETS[dim].opts.some(o=>normRule(o[1])===normRule(label))){err.hidden=false;err.textContent='"'+label+'" is already in this list.';return;}
  const R=M.decide.rules;R.custom[dim]=R.custom[dim]||[];const id=customRuleId(label);R.custom[dim].push({id,label,hint:hint.value.trim()});
  logInput('rule-option:'+dim+':'+id,null,label);syncRuleCatalog(R);persist();dialog.close();render();ruleDialog(dim,id);},'btn primary')]));
 dialog.showModal();name.focus();
}
function presetPreview(key){
 const p=rulePresets().find(x=>x[0]===key);if(!p)return [];
 return Object.entries(p[2]).flatMap(([dim,rules])=>Object.entries(rules).filter(([id])=>optOf(dim,id)).map(([id,st])=>h('span',{class:'stance-chip '+st,text:(RULE_STATE[st][2]?RULE_STATE[st][2]+' ':'')+optOf(dim,id)[1]})));
}
function rulesToolbar(){
 // Presets moved to presetPanel() (rules-overview.js): explained, previewed per rule, with a real Cancel.
 const file=h('input',{type:'file',accept:'.csv,.txt,text/csv',hidden:'',onchange:async e=>{const f=e.target.files[0];if(!f)return;const res=importRuleCsv(await f.text());e.target.value='';
  if(res.error){alert(res.error);return;}persist();render();
  const msg=[res.applied+' rules imported'+(res.added?' ('+res.added+' new options added)':'')+'.'];
  if(res.outsideMode.length)msg.push('Skipped options for a different project type: '+res.outsideMode.join(', '));
  if(res.unmatched.length)msg.push(res.unmatched.length+' not in the list (add an "area" column to add them as your own options): '+res.unmatched.slice(0,8).join(', ')+(res.unmatched.length>8?'…':''));
  if(res.unknownRing.length)msg.push(res.unknownRing.length+' with an unknown ring or status: '+res.unknownRing.slice(0,5).join(', '));
  alert(msg.join('\n\n'));}});
 return h('div',{class:'rules-toolbar'},[
  button('Import tech radar (CSV)',()=>file.click(),'btn small'),button('Export rules (CSV)',()=>downloadText('technology-rules.csv',exportRuleCsv()),'btn small'),file,
  h('details',{class:'rules-csv-help'},[h('summary',{text:'CSV format'}),h('p',{class:'muted',text:'Columns: name, ring (or status), and optionally area, reason, owner, review_by, exit_by. Ring Adopt / Trial / Assess / Hold becomes Preferred / Allowed / Needs approval / Retiring; Standard and Excluded are accepted too. Area is architecture, language, database, integration or cloud; with it, names that are not in the list are added as your own options. Export gives the same format, so you can edit it in a spreadsheet and import it back.'})])]);
}

/* 3. Every rule in force in one table, with its reason, owner and review date. */
function rulesInForce(){
 const R=M.decide.rules,rows=[];const today=new Date().toISOString().slice(0,10);
 for(const dim of Object.keys(TARGETS))for(const [id,label] of TARGETS[dim].opts){const st=R.stance[dim][id]||'allowed';if(st==='allowed')continue;const i=(R.ruleInfo[dim]||{})[id]||{};rows.push({dim,id,label,st,i});}
 const order=RULE_STATES.map(x=>x[0]);rows.sort((a,b)=>order.indexOf(a.st)-order.indexOf(b.st));
 if(!rows.length)return h('p',{class:'muted rules-none',text:isNewProduct()?'All options are open. Set a rule if your team has a preference or restriction.':'No rules yet: every option is allowed and ranked on the evidence alone.'});
 const noReason=rows.filter(r=>!r.i.reason).length,noOwner=rows.filter(r=>!r.i.owner).length;
 return h('details',{class:'compare rules-force',open:''},[h('summary',{text:'Rules in force · '+rows.length}),
  noReason||noOwner?h('p',{class:'input-status open',text:[noReason&&noReason+' without a reason',noOwner&&noOwner+' without an owner'].filter(Boolean).join(' · ')+'. A rule with a reason and an owner is easier to defend at sign-off.'}):null,
  h('table',{class:'scan-table'},[h('thead',{},[h('tr',{},['Rule','Area','Option','Reason','Owner','Review by',''].map(t=>h('th',{text:t})))]),
   h('tbody',{},rows.map(r=>h('tr',{},[h('td',{},[h('span',{class:'stance-chip '+r.st,text:(RULE_STATE[r.st][2]?RULE_STATE[r.st][2]+' ':'')+RULE_STATE[r.st][1]+(r.st==='retiring'&&r.i.exitBy?' · exit '+r.i.exitBy:'')})]),
    h('td',{text:TARGETS[r.dim].title}),h('td',{text:r.label}),h('td',{class:r.i.reason?'':'muted',text:r.i.reason||'—'}),h('td',{class:r.i.owner?'':'muted',text:person(r.i.owner)?.name||'—'}),
    h('td',{class:r.i.reviewBy&&r.i.reviewBy<today?'warn-text':'',text:r.i.reviewBy?(r.i.reviewBy<today?r.i.reviewBy+' (due)':r.i.reviewBy):'—'}),h('td',{},[button('Edit',()=>ruleDialog(r.dim,r.id),'linkbtn')])])))])]);
}

/* 4. Find an option, or show only options with a rule (filters on the page, no re-render). */
let rulesFilter={q:'',onlyRules:false};
function applyRulesFilter(root){
 const q=rulesFilter.q.trim().toLowerCase();
 for(const g of root.querySelectorAll('.stance-group[data-area]')){let shown=0;
  for(const c of g.querySelectorAll('.stance-chip')){const ok=(!q||c.dataset.find.includes(q))&&(!rulesFilter.onlyRules||c.dataset.rule!=='allowed');c.hidden=!ok;if(ok)shown++;}
  g.hidden=!shown&&(q||rulesFilter.onlyRules);}
}
function rulesFilterBar(){
 const box=h('input',{class:'field compact',type:'search',placeholder:'Find an option, e.g. Kafka, Oracle, Azure','aria-label':'Find an option',value:rulesFilter.q,
  oninput:e=>{rulesFilter.q=e.target.value;applyRulesFilter(document.getElementById('main'));}});
 return h('div',{class:'rules-filter'},[box,h('label',{class:'type-chip'+(rulesFilter.onlyRules?' on':'')},[h('input',{type:'checkbox',checked:rulesFilter.onlyRules?'':null,
  onchange:e=>{rulesFilter.onlyRules=e.target.checked;e.target.parentElement.classList.toggle('on',e.target.checked);applyRulesFilter(document.getElementById('main'));}}),'Only options with a rule'])]);
}
function stanceSummary(){
 const {n,overdue}=ruleCounts();const parts=RULE_STATES.filter(s=>s[0]!=='allowed'&&n[s[0]]).map(s=>n[s[0]]+' '+s[1].toLowerCase());
 return {text:parts.length?parts.join(' · '):'Everything allowed (optional)',overdue};
}

function rulesSectionByTitle(page,title){
 return [...page.querySelectorAll('section.summary-section')].find(s=>s.querySelector('h2')?.textContent.trim()===title)||null;
}

function rulesTabsPage(){
 ensureDecide();
 const product=isNewProduct(), R=M.decide.rules,P=M.people||[];
 const nextLabel=product?'Design options':'Target decisions';
 const old=rulesPage();                       // existing sections, already bound to the data
 const K=constraints();
 const fields=constraintFields();
 const answered=fields.filter(([k])=>String(K[k]||'').trim()).length;
 const sum=stanceSummary();
 const tabs=[
  ['people','1 · Who decides',P.length?P.length+(P.length===1?' person':' people'):'Add at least one person',!P.length],
  ['tech','2 · Technology rules',sum.text+(sum.overdue?' · '+sum.overdue+' to review':''),!!sum.overdue],
  ['priorities','3 · Priorities & limits',answered+' of '+fields.length+' limits answered',false]];
 const cur=tabs.find(t=>t[0]===rulesTab)||tabs[0];

 let body;
 if(cur[0]==='people'){
  body=[h('p',{class:'muted rules-why',text:'Why we ask: every review and decision is recorded under a named person, and the decision makers approve the plan at sign-off. Add at least one person who decides.'}),peopleSection()];
 }else if(cur[0]==='tech'){
  body=[h('section',{class:'summary-section'},[
   h('h2',{text:product?'Technology choices for the first release':'Technology rules'}),
   h('p',{class:'muted rules-why',text:product?'What your product needs, why, and what your organization allows. Start from the decisions below; presets and the full option list are further down. Design options uses these rules alongside your product brief.':'Optional. Everything is allowed unless you set a rule. Click an option to set its rule, with a reason, an owner and a review date. Start from a preset or import your technology radar to save clicks.'}),
   typeof rulesOverview==='function'?rulesOverview():null,
   typeof presetPanel==='function'?presetPanel():null,
   rulesInForce(),
   h('details',{class:'compare rules-all'},[h('summary',{text:'Browse all categories and options (rule chips, CSV import and export)'}),
    rulesToolbar(),
    h('div',{class:'rules-browse'},[rulesFilterBar(),h('div',{class:'stance-legend'},RULE_STATES.map(([v,label,icon,help])=>h('span',{class:'legend-item '+v,title:help},[(icon?icon+' ':'')+label])))]),
    ...Object.keys(TARGETS).map(stanceChips),
    typeof cloudServicesPanel==='function'?cloudServicesPanel():null])
  ])];
  queueMicrotask(()=>applyRulesFilter(document.getElementById('main')));
 }else{
  const weights=rulesSectionByTitle(old,'What matters most');
  const limits=rulesSectionByTitle(old,'Budget, risk and standards');
  const skills=rulesSectionByTitle(old,'Constraints');
  const money=product?null:rulesSectionByTitle(old,'Business case: current yearly run cost');
  if(weights){weights.querySelector('p').textContent='How much each criterion counts when options are ranked: 0 = ignore, 5 = critical. Start here; it drives the ranking most.';}
  if(limits){limits.querySelector('h2').textContent='Risk, budget and standards';}
  if(skills){
   skills.querySelector('h2').textContent='Team skills, data and contracts';
   skills.querySelector('h2').after(h('p',{class:'muted',text:'Languages your team already knows, where the data may live, and agreements that limit the choice.'}));
  }
  const moneyBox=money?h('details',{class:'summary-section rules-money',open:(M.decide.businessCase&&runCost())?'':null},[
   h('summary',{text:'Money: current yearly run cost (optional)'}),...[...money.children].filter(x=>x.tagName!=='H2')]):null;
  body=[weights,limits,skills,moneyBox].filter(Boolean);
 }

 return h('div',{class:'rules-page'},[
  pageHeader(product?'03 / RULES & PEOPLE':'05 / RULES & PEOPLE','Who decides, and within which rules',
   'These answers guide how options are filtered and ranked.',
   [button(product?'← First-release scope':'← Discovery summary',()=>go(product?'release-scope':'summary')),button(nextLabel+' →',()=>go('decide'),'btn primary')]),
  h('div',{class:'tabs rules-tabs',role:'tablist'},tabs.map(([id,title,hint,need])=>h('button',{
   type:'button',role:'tab','aria-selected':String(id===cur[0]),class:'tab '+(id===cur[0]?'active':''),
   onclick:()=>{rulesTab=id;render();}},[h('b',{text:title}),h('small',{class:need?'need':'',text:hint})]))),
  h('div',{class:'rules-body',role:'tabpanel'},body),
  h('div',{class:'page-actions rules-next'},[
   cur[0]!=='priorities'?button('Next: '+tabs[tabs.indexOf(cur)+1][1].replace(/^\d · /,'')+' →',()=>{rulesTab=tabs[tabs.indexOf(cur)+1][0];render();window.scrollTo({top:0});},'btn primary')
    :button(nextLabel+' →',()=>go('decide'),'btn primary')])
 ]);
}
