/* MindSuite web app · system blueprint (Design options / Target decisions)
   One shared platform for the whole product or system, plus what each capability adds to it, instead of
   repeating the full stack per capability. Rule-based, no model calls; reuses techPlan() (assets/tech-plan.js).
   Basis (chips): "Current picks" uses each capability's recorded decision, or its best-fit scenario while it
   is open; "All Safe" / "All Balanced" / "All Ambitious" use that scenario everywhere.
   The shared platform follows the capabilities that run on it (build, rebuild, refactor, replatform,
   rehost, encapsulate): the most common architecture, language, database, integration and cloud among them,
   at the most common tier. Bought and partner capabilities are listed with where they run and how they connect.
   systemBlueprint(C,basis) returns {basis,tier,platform:[layers],capabilities:[...],counts} for the diagrams;
   systemBlueprintPanel(C) draws it. Hook: decidePage() in index.html, under portfolioPanel(). */

const SB_UI={basis:'current'};
const SB_ON_PLATFORM=['build','rebuild','refactor','replatform','rehost','encapsulate'];
const SB_ADDS=[
 ['search',/\b(search|catalogue|catalog|browse|find|look up)\b/,'Search index'],
 ['video',/\b(video|stream(?:ing)?|play(?:back)?|watch|upload a video|transcod\w*)\b/,'Media pipeline (storage, transcoding, streaming)'],
 ['files',/\b(uploads?|photos?|photograph\w*|images?|documents?|attachments?|files?)\b/,'Object storage'],
 ['payments',/\b(payments?|billing|subscriptions?|checkout|refunds?|invoices?)\b/,'Payment provider + webhooks'],
 ['notify',/\b(alerts?|notif\w*|emails?|sms|remind\w*)\b/,'Email / SMS sending'],
 ['reports',/\b(reports?|dashboards?|analytics|insights?|value)\b/,'Reporting copy or read replica'],
 ['realtime',/\b(real-?time|live|chat|resume|progress)\b/,'Fast state in the cache'],
 ['accounts',/\b(accounts?|sign[- ]?in|log[- ]?in|access|roles?|permissions?)\b/,'Identity and roles'],
 ['workflow',/\b(review|approve|approval|publish|escalat\w*|resolve)\b/,'Workflow states and audit log']];

/* Route for one capability: the recorded decision, else the advisor's best fit. The tier is the platform's
   (or the what-if tier chosen with the chips). */
function sbScenarioFor(c,basis){
 const tier=basis==='current'?(typeof platformEffective==='function'?platformEffective().tier:'Balanced'):basis;
 const d=decisionFor(c);
 if(d.approach&&d.approach!=='undecided')return {name:tier,tier,a:{id:d.approach,label:apprLabel(d.approach)},d,picked:'decided'};
 const adv=approachAdvice(c),scen=advisorScenarios(c,adv),best=scen[0];
 return best?{...best,name:tier,tier,picked:'best fit'}:null;
}
/* Shared concerns: who owns each one (a bought product, a partner platform, or the platform itself). */
const SB_CONCERNS=[['payments',/\b(subscriptions?|payments?|billing|checkout|invoic\w*|refunds?)\b/i,'Payments and billing'],
 ['identity',/\b(accounts?|sign[- ]?in|log[- ]?in|access|identity|registration)\b/i,'Customer accounts and sign-in'],
 ['media',/\b(video|play(?:back)?|stream\w*|media|audio|podcasts?)\b/i,'Media playback and delivery'],
 ['search',/\b(search|catalogue|catalog|browse)\b/i,'Search'],
 ['notify',/\b(notif\w*|alerts?|emails?|sms)\b/i,'Notifications']];
function sbOwners(rows){
 const out=[];
 for(const [k,re,label] of SB_CONCERNS){
  const hit=rows.filter(r=>re.test(r.c.name));if(!hit.length)continue;
  const ext=hit.filter(r=>['buy','replace','partner'].includes(r.s.a.id));
  const built=hit.filter(r=>SB_ON_PLATFORM.includes(r.s.a.id));
  const owner=ext[0]?{name:ext[0].c.name,by:ext[0].s.a.id==='partner'?'partner':'vendor'}:null;
  out.push({id:k,label,owner,built:built.map(r=>r.c.name),external:ext.map(r=>r.c.name),
   clash:!!(owner&&built.length),split:ext.length>1&&new Set(ext.map(r=>r.s.a.id)).size>1});
 }
 return out;
}
const sbMost=list=>{const n={};for(const x of list)if(x)n[x]=(n[x]||0)+1;return Object.entries(n).sort((a,b)=>b[1]-a[1])[0]?.[0]||'';};

function systemBlueprint(C,basis){
 basis=basis||SB_UI.basis;
 try{window.TP_USED=new Map();}catch(e){}
 const rows=C.map(c=>({c,s:sbScenarioFor(c,basis)})).filter(r=>r.s);
 const on=rows.filter(r=>SB_ON_PLATFORM.includes(r.s.a.id));
 const P=typeof platformEffective==='function'?platformEffective():{tier:'Balanced'};
 const tier=basis==='current'?P.tier:basis;
 const choices={integration:[].concat(P.integration||[])};
 for(const dim of ['architecture','language','database','cloud'])choices[dim]=P[dim]||'';
 const approach=on.length?sbMost(on.map(r=>r.s.a.id)):sbMost(rows.map(r=>r.s.a.id))||'build';
 const owners=sbOwners(rows),ext={};for(const o of owners)if(o.owner)ext[o.id]=o.owner;
 const sysCtx={id:'system',name:M.project||'',description:C.map(c=>c.name+' '+(c.description||'')).join(' '),ext};
 const platformScenario={name:tier,tier,a:{id:approach,label:apprLabel(approach)},d:{approach,choices}};
 const platform=techPlan(sysCtx,platformScenario).filter(l=>l.id!=='route');
 const hostLine=(platform.find(l=>l.id==='hosting')||{}).t||'';
 const shortHost=hostLine.split(':')[0];
 const capabilities=rows.map(({c,s})=>{
  const text=(c.name+' '+(c.description||'')).toLowerCase();
  const adds=SB_ADDS.filter(([,re])=>re.test(text)).map(([id,,label])=>({id,label}));
  const a=s.a.id;
  const runsOn=['buy','replace'].includes(a)?'Vendor product (SaaS), connected through the API gateway'
   :a==='partner'?'Partner or low-code platform, connected through APIs'
   :a==='retire'?'Switched off; data archived'
   :on.length?'Shared platform'+(shortHost?' ('+shortHost+')':''):'Shared platform';
  const tech=typeof scenarioSummary==='function'?scenarioSummary(s.d):'';
  return {id:c.id,name:c.name,release:c.releaseScope==='later'?'Later':c.releaseScope==='first'?'First':'',
   route:s.a.label,scenario:s.name,picked:s.picked,runsOn,tech,adds};
 });
 const counts={platform:on.length,vendor:rows.filter(r=>['buy','replace'].includes(r.s.a.id)).length,partner:rows.filter(r=>r.s.a.id==='partner').length,retire:rows.filter(r=>r.s.a.id==='retire').length};
 return {basis,tier,approach,choices,platform,capabilities,counts,owners,platformDecided:!!P.decided};
}

function systemBlueprintPanel(C){
 if(!C||!C.length)return null;
 let B;try{B=systemBlueprint(C);}catch(e){console.error('System blueprint',e);return null;}
 const product=isNewProduct();
 const setBasis=b=>{SB_UI.basis=b;render();};
 const srcTag={'your choice':'you chose','brief':'from brief','calculation':'calculated','your scenario':'scenario','rule':'rule'};
 const chip=(id,label)=>h('button',{type:'button',class:'type-chip'+(SB_UI.basis===id?' on':''),onclick:()=>setBasis(id)},[label]);
 const k=B.counts;
 const usage=[k.platform&&k.platform+' on the shared platform',k.vendor&&k.vendor+' bought (vendor product)',k.partner&&k.partner+' on a partner platform',k.retire&&k.retire+' switched off'].filter(Boolean).join(' · ');
 const addCount={};for(const c of B.capabilities)for(const a of c.adds)addCount[a.label]=(addCount[a.label]||0)+1;
 return h('section',{class:'sys-blueprint'},[
  h('div',{class:'sb-head'},[
   h('div',{},[h('h2',{text:'System blueprint'}),
    h('p',{class:'muted',text:'The result of steps 1 and 2: one shared platform for the whole '+(product?'product':'system')+', what runs on it, and who owns each shared concern. Use the chips to see the same routes on another platform tier.'})]),
   h('div',{class:'load-profiles'},[h('span',{class:'muted',text:'Platform tier:'}),chip('current',(B.platformDecided?'Decided: ':'Suggested: ')+(typeof platformEffective==='function'?platformEffective().tier:'Balanced')),chip('Safe','What if Safe'),chip('Balanced','What if Balanced'),chip('Ambitious','What if Ambitious')])]),
  h('p',{class:'sb-summary'},[h('b',{text:B.tier+' platform'}),' · '+(usage||'no capabilities yet')+' · open '+(product?'capabilities':'contexts')+' use their best-fit route until decided']),
  B.owners.length?h('div',{class:'sb-owners'},[h('h3',{text:'Who owns each shared concern'}),h('ul',{class:'plain-list'},B.owners.map(o=>h('li',{class:o.clash||o.split?'warn-text':''},[h('b',{text:o.label+': '}),
   o.owner?(o.owner.by==='partner'?'the partner platform':'the bought product')+' chosen for "'+o.owner.name+'"; the platform uses it instead of building its own':'built once on the shared platform'+(o.built.length?' (used by '+o.built.join(', ')+')':''),
   o.clash?' — but '+o.built.join(', ')+' '+(o.built.length>1?'are':'is')+' built on the platform for the same concern: make sure they do not do the same job twice, or change one route':'',
   o.split?' — and the routes differ ('+o.external.join(', ')+'): choose one product for this concern':''])))]):null,
  h('div',{class:'sb-grid'},[
   h('div',{class:'sb-platform'},[h('h3',{text:'Shared platform'}),
    h('table',{class:'tp-table sb-table'},h('tbody',{},B.platform.map(l=>h('tr',{},[h('th',{scope:'row',text:l.label}),h('td',{},[h('div',{text:l.t}),h('small',{class:'tp-why',text:(srcTag[l.src]||l.src)+' · '+l.why})])]))))]),
   h('div',{class:'sb-caps'},[h('h3',{text:(product?'Capabilities':'Contexts')+' on it'}),
    h('div',{class:'tp-scroll'},h('table',{class:'tp-table sb-table'},[
     h('thead',{},h('tr',{},[h('th',{text:product?'Capability':'Context'}),product?h('th',{text:'Release'}):null,h('th',{text:'Route'}),h('th',{text:'Runs on'}),h('th',{text:'Uses from the platform'})])),
     h('tbody',{},B.capabilities.map(c=>h('tr',{},[
      h('th',{scope:'row'},[h('div',{text:c.name}),h('small',{class:'tp-why',text:c.scenario+' · '+c.picked})]),
      product?h('td',{text:c.release||'—'}):null,
      h('td',{},[h('div',{text:c.route}),c.tech?h('small',{class:'tp-why',text:c.tech}):null]),
      h('td',{text:c.runsOn}),
      h('td',{text:c.adds.length?c.adds.map(a=>a.label).join(' · '):'Core app, database and sign-in only'})])))])),
    Object.keys(addCount).length?h('p',{class:'muted sb-shared',text:'Shared by several: '+Object.entries(addCount).filter(([,n])=>n>1).sort((a,b)=>b[1]-a[1]).map(([l,n])=>l+' ('+n+')').join(' · ')+'. Build these once on the platform.'}):null])]),
  typeof tierCostSection==='function'?tierCostSection(C):null]);
}

(function sbStyle(){
 if(document.getElementById('sb-style'))return;
 const css=`.sys-blueprint{margin:14px 0;padding:14px 16px;border:1px solid #d9e1f2;border-radius:12px;background:#fff}
.sb-head{display:flex;flex-wrap:wrap;gap:8px 16px;justify-content:space-between;align-items:flex-start}.sb-head h2{margin:0}.sb-head p{margin:4px 0 0;max-width:640px}
.sb-summary{margin:8px 0 10px}
.sb-grid{display:grid;grid-template-columns:minmax(320px,5fr) minmax(360px,6fr);gap:16px}
@media (max-width:1100px){.sb-grid{grid-template-columns:1fr}}
.sb-grid h3{margin:4px 0 6px;font-size:14px}.sb-table{min-width:0}.sb-caps .sb-table{min-width:560px}
.sb-table td{width:auto}.sb-shared{font-size:12px;margin:6px 0 0}`;
 document.head.append(Object.assign(document.createElement('style'),{id:'sb-style',textContent:css}));
})();
