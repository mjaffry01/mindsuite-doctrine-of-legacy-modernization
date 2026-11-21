/* MindSuite web app · Architecture validation (screen 'validation').
   Proves, journey by journey, whether the architecture will work for the business. One row per critical
   journey: the design decision it tests, a consequential failure, the operating condition, an acceptance
   measure and test method, the result and its evidence, an explicit assumption, an open question, and three
   owners (approves the design, owns the business data, operates and restores the service). For
   modernization also: current system, transition wave, the authoritative writer while both systems are live,
   the known writers from verified discovery records, reconciliation and the rollback condition.

   Pathfinder does not pick the verdict; it derives it from what is recorded:
     • Enough evidence to choose      – the test passed and its evidence is recorded
     • Proceed under an assumption    – no passing test yet, but the assumption is written and the approver accepted it
     • Needs a test first             – nothing above, or the test failed
     • Reopen                         – the decision it rests on changed after it was tested or accepted
   Separately, "demonstrated ready to launch / cut over" needs the passing result plus confirmed data owner and
   operator (and, for modernization, writer, reconciliation and rollback). Approved to build ≠ ready to launch.
   Stored in M.validation = {rows:[…]}; exported in the HLD (section 4d, Markdown, JSON). */

const VAL_UI={open:{}};
const VAL_METHODS=[['','Choose a method'],['load','Load or performance test'],['failure','Failure injection (break it on purpose)'],['rehearsal','Rehearsal (cutover, rollback, restore)'],['demo','Working demonstration'],['review','Design review or analysis'],['pilot','Pilot with real users']];
const VAL_RESULTS=[['','Not run yet'],['passed','Passed'],['failed','Failed']];
const VAL_VERDICT={
 enough:['Enough evidence to choose','ok'],
 assume:['Proceed under an explicit assumption','warn'],
 test:['Needs a test first','bad'],
 reopen:['Reopen: what it rests on changed','bad'],
 incomplete:['Not ready to judge','muted']};

function valStore(){M.validation=M.validation||{rows:[]};M.validation.rows=M.validation.rows||[];return M.validation;}
function valProduct(){return typeof isNewProduct==='function'&&isNewProduct();}
function valCtxs(){try{return valProduct()&&typeof launchContexts==='function'?launchContexts():decCtxs();}catch(e){return [];}}
function valPerson(id){const p=(M.people||[]).find(p=>p.id===id);return p?p.name+(p.role?' · '+p.role:''):'';}

/* The brief's journeys (new product) or the business areas (modernization), as candidates for rows. */
function valJourneys(){
 if(valProduct()){
  const t=String(((M.product||{}).journeys)||'');
  const segs=t.split(/\s*(?:\d+\)|\n|;)\s*/).map(s=>s.trim()).filter(s=>s.length>8);
  return segs.map(s=>({key:'j:'+s.slice(0,60).toLowerCase(),title:s.slice(0,200),source:'Brief › Key journeys'}));
 }
 return valCtxs().map(c=>({key:'c:'+c.id,title:c.name+(c.description?': '+c.description:''),source:'Discovery › business area',capId:c.id}));
}

/* Suggested rows for consequential business cases found in the brief or findings. */
function valTemplates(){
 const P=M.product||{},txt=[P.problem,P.users,P.journeys,P.mvp,P.integrations,M.project,...valCtxs().map(c=>c.name+' '+(c.description||''))].join(' ').toLowerCase();
 const T=[],rec=P.recovery||{};const prod=valProduct();
 if(/stock|inventory|basket|cart|last item|order/.test(txt))T.push({key:'t:last-item',journey:'A customer pays for the last item in stock',
  failure:'Two customers pay for the last item at the same moment, and the payment provider sends the same payment confirmation twice',
  condition:'Peak checkout load (from the brief) with retries switched on',
  measure:'Exactly one paid order and one stock reservation (or kitchen ticket); the second customer is refused before paying or refunded automatically; both attempts are in the audit record',
  method:'failure',object:'stock',question:prod?'Build stock reservation ourselves, or buy it from the commerce or payment provider?':'Which system owns stock writes while both systems are live?'});
 if(/pay|checkout|billing|subscription|card/.test(txt))T.push({key:'t:dup-payment',journey:'The payment provider confirms a payment',
  failure:'The provider sends the same confirmation twice, late, or out of order',condition:'Normal load; provider retries for 24 hours',
  measure:'One payment recorded per charge; no duplicate fulfilment; late or out-of-order confirmations are reconciled daily with zero unexplained differences',method:'failure',object:'payment'});
 if(/sign.?in|login|account|identity/.test(txt))T.push({key:'t:idp-down',journey:'Users sign in',failure:'The sign-in provider is unavailable for 30 minutes',
  condition:'Peak hour',measure:'Signed-in users keep working; new sign-ins fail with a clear message; no data is exposed; service recovers without manual steps',method:'failure',object:'account'});
 T.push({key:'t:region-loss',journey:'Restore service after losing the main region or database',failure:'The primary database or region is lost',
  condition:'Any time, including peak',measure:'Service restored within '+(rec.rto||'the agreed RTO (not decided yet)')+'; data loss no more than '+(rec.rpo||'the agreed RPO (not decided yet)'),method:'rehearsal',object:''});
 if(!prod)T.push({key:'t:cutover',journey:'Cut over a business area to the new system',failure:'A cutover step fails after the new system has accepted writes',
  condition:'Inside the planned cutover window',measure:'Reconciliation shows no unexplained differences; rollback is possible until the stated last safe point, and writes accepted after cutover reach the old system',method:'rehearsal',object:''});
 return T;
}

/* What a row rests on: the platform decision, its capability's route, and (modernization) its wave dates. */
function valBasis(r){
 let pf={};try{pf=platformEffective();}catch(e){}
 const c=r.capId?valCtxs().find(x=>x.id===r.capId):null;let d={};try{d=c?decisionFor(c):{};}catch(e){}
 const w=r.wave&&M.plan&&M.plan.waves?(M.plan.waves[r.wave]||{}):{};
 return {cloud:pf.cloud||'',hosting:pf.hosting||'',database:pf.database||'',architecture:pf.architecture||'',tier:pf.tier||'',
  route:d.approach||'',routeStatus:d.status||'',waveStart:w.start||'',waveEnd:w.end||'',assumption:r.assumption||'',decision:r.decision||''};
}
function valBasisDiff(a,b){
 if(!a)return [];const lab={cloud:'cloud',hosting:'hosting model',database:'database',architecture:'architecture',tier:'tier',route:'route for the capability',routeStatus:'decision status',waveStart:'wave start',waveEnd:'wave end',assumption:'the assumption',decision:'the design decision'};
 return Object.keys(lab).filter(k=>String(a[k]||'')!==String(b[k]||'')).map(k=>lab[k]+(['assumption','decision'].includes(k)?' was edited':' changed from '+(a[k]||'none')+' to '+(b[k]||'none')));
}

/* Known writers and consumers of the row's business object, from discovery records (modernization). */
function valWriters(r){
 const obj=String(r.object||'').trim().toLowerCase();if(!obj||valProduct()||!M.inventory)return [];
 const has=x=>String(x||'').toLowerCase().includes(obj),verified=s=>['confirmed','overridden','manual'].includes(s);
 const ctxName=id=>(valCtxs().find(c=>c.id===id)||{}).name||id;const out=[];
 for(const s of M.inventory.dataStores||[])if(has(s.name)||has(s.description)){
  for(const w of [].concat(s.writers||[]))out.push({kind:'writer',name:ctxName(w)+' writes '+s.name,verified:verified(s.status)});
  if(![].concat(s.writers||[]).length)out.push({kind:'writer',name:s.name+': writers not recorded',verified:false});
  for(const rd of [].concat(s.readers||[]))out.push({kind:'consumer',name:ctxName(rd)+' reads '+s.name,verified:verified(s.status)});}
 for(const c of M.inventory.components||[])if((c.kind==='batch'||c.kind==='dbcode')&&(has(c.name)||has(c.description)))out.push({kind:'writer',name:(c.kind==='batch'?'Batch job ':'Database code ')+c.name+(c.schedule?' ('+c.schedule+')':''),verified:verified(c.status)});
 for(const i of M.inventory.interfaces||[])if(has(i.name)||has(i.frequency)||has(i.partner))out.push({kind:i.direction==='in'?'writer':'consumer',name:'Interface '+i.name+(i.partner?' with '+i.partner:''),verified:verified(i.status)});
 return out;
}

function valMissing(r){
 const m=[];if(!String(r.journey||'').trim())m.push('the journey');if(!String(r.decision||'').trim())m.push('the design decision it tests');
 if(!String(r.failure||'').trim())m.push('a failure scenario');if(!String(r.measure||'').trim())m.push('an acceptance measure');
 if(!(r.owners&&r.owners.approver&&r.owners.approver.id))m.push('who approves the design');return m;
}
/* The verdict, derived from what is recorded. */
function valVerdict(r){
 const missing=valMissing(r),why=[];
 if(missing.length)return {key:'incomplete',why:['Add '+missing.join(', ')]};
 const now=valBasis(r),drift=valBasisDiff(r.basis,now);
 const passed=r.result==='passed',failed=r.result==='failed',ev=String(r.evidence||'').trim();
 const accepted=!!(r.assumption&&String(r.assumption).trim()&&r.owners&&r.owners.approver&&r.owners.approver.confirmed&&r.accepted);
 if((passed||accepted)&&drift.length)return {key:'reopen',why:drift.map(d=>d.charAt(0).toUpperCase()+d.slice(1)).concat(['Reopen with '+(valPerson(r.owners.approver.id)||'the approver')])};
 if(failed)return {key:'test',why:['The test failed'+(ev?': '+ev:'')+'. Rework the design or the measure, then test again']};
 if(passed&&ev)return {key:'enough',why:['Passed'+(r.resultAt?' on '+r.resultAt:'')+' · evidence: '+ev]};
 if(passed&&!ev)why.push('Marked passed, but no evidence is recorded');
 if(accepted)return {key:'assume',why:['Assumption accepted by '+valPerson(r.owners.approver.id)+': '+r.assumption].concat(why)};
 if(String(r.assumption||'').trim())why.push('An assumption is written but the approver has not accepted it');
 return {key:'test',why:why.length?why:['No result yet'+(r.method?': run the '+((VAL_METHODS.find(m=>m[0]===r.method)||[,''])[1]).toLowerCase():'')]};
}
/* Approved to build is not ready to launch or cut over: that needs a demonstrated result and accepted operations. */
function valReadiness(r){
 const v=valVerdict(r),need=[];const o=r.owners||{};
 if(v.key!=='enough')need.push('a passing test with evidence');
 if(!(o.data&&o.data.id&&o.data.confirmed))need.push('a confirmed owner of the business data');
 if(!(o.operator&&o.operator.id&&o.operator.confirmed))need.push('a confirmed operator who restores the service');
 // The business object at stake needs an accepted data and caching policy (Data and caching policy page).
 if(String(r.object||'').trim()&&typeof dpStore==='function'&&!dpStore().accepted[String(r.object).trim().toLowerCase()])need.push('an accepted data policy for '+String(r.object).trim());
 if(!valProduct()){
  if(!String(r.writer||'').trim())need.push('the authoritative writer during the wave');
  if(!String(r.reconciliation||'').trim())need.push('a reconciliation test and acceptable variance');
  if(!String(r.rollback||'').trim())need.push('the rollback condition and last safe rollback point');
  const unv=valWriters(r).filter(w=>!w.verified);if(unv.length)need.push(unv.length+' unverified writer'+(unv.length>1?'s or consumers':' or consumer'));
 }
 return {ready:!need.length,need};
}

function valAddRow(seed){
 const S=valStore();const r={id:'val-'+Date.now().toString(36)+Math.random().toString(36).slice(2,6),journey:'',source:'',decision:'',failure:'',condition:'',measure:'',method:'',result:'',resultAt:'',evidence:'',assumption:'',accepted:false,question:'',owners:{approver:{},data:{},operator:{}},capId:'',object:'',current:'',wave:'',writer:'',reconciliation:'',rollback:'',afterRollback:'',...seed};
 if(!r.decision)r.decision=valSuggestDecision(r);
 S.rows.push(r);VAL_UI.open[r.id]=true;logInput('validation:add',null,r.journey);persist();render();
}
function valSuggestDecision(r){
 let pf={};try{pf=platformEffective();}catch(e){}
 const lab=(dim,id)=>id?((typeof optOf==='function'&&(optOf(dim,id)||[])[1])||id):'';
 const parts=[pf.tier?pf.tier+' tier':'',lab('cloud',pf.cloud),lab('database',pf.database),lab('architecture',pf.architecture)].filter(Boolean);
 const c=r.capId?valCtxs().find(x=>x.id===r.capId):valCtxs().find(c=>String(r.journey||'').toLowerCase().includes(String(c.name||'').toLowerCase().split(' ')[0]));
 let route='';try{if(c){const d=decisionFor(c);route=c.name+': '+(d.approach?apprLabel(d.approach):'route not chosen')+(d.status==='decided'?' (decided)':' (open)');r.capId=r.capId||c.id;}}catch(e){}
 return [parts.length?'Platform: '+parts.join(', ')+((pf.decided)?' (recorded)':' (suggestion, not recorded)'):'',route?'Route for '+route:''].filter(Boolean).join(' · ');
}

function valArea(label,value,on,ph,rows){const id='va-'+Math.random().toString(36).slice(2);return h('label',{class:'context-control val-field',for:id},[h('span',{text:label}),h('textarea',{id,class:'field',rows:String(rows||2),placeholder:ph||'',text:value||'',oninput:e=>{on(e.target.value);persist();}})]);}
function valSelect(label,value,opts,on){const id='vs-'+Math.random().toString(36).slice(2);const s=h('select',{id,class:'field',onchange:e=>{on(e.target.value);persist();render();}},opts.map(([v,t])=>h('option',{value:v,text:t})));s.value=value||'';return h('label',{class:'context-control val-field',for:id},[h('span',{text:label}),s]);}
function valOwner(r,k,label,hint){
 r.owners=r.owners||{};const o=r.owners[k]=r.owners[k]||{};
 const ppl=[['','Not assigned'],...(M.people||[]).map(p=>[p.id,p.name+' · '+p.role])];
 return h('div',{class:'val-owner'},[valSelect(label,o.id,ppl,v=>{o.id=v;o.confirmed=false;logInput('validation:owner:'+k,null,v);}),h('small',{class:'muted',text:hint}),
  o.id?h('label',{class:'val-check'},[h('input',{type:'checkbox',checked:o.confirmed?'':null,onchange:e=>{o.confirmed=e.target.checked;logInput('validation:owner-confirmed:'+k,!e.target.checked,e.target.checked);persist();render();}}),' '+(o.confirmed?'Accepted responsibility':'Suggested: not accepted yet')]):null]);
}

function valRowCard(r){
 const v=valVerdict(r),rd=valReadiness(r),prod=valProduct(),open=!!VAL_UI.open[r.id];
 const badge=(k)=>h('span',{class:'val-badge '+VAL_VERDICT[k][1],text:VAL_VERDICT[k][0]});
 const head=h('div',{class:'val-head'},[
  h('button',{type:'button',class:'val-toggle','aria-expanded':open?'true':'false',onclick:()=>{VAL_UI.open[r.id]=!open;render();}},[h('b',{text:(open?'▾ ':'▸ ')+(r.journey||'New journey')})]),
  h('div',{class:'val-badges'},[badge(v.key),h('span',{class:'val-badge '+(rd.ready?'ok':'muted'),text:rd.ready?(prod?'Demonstrated ready to launch':'Demonstrated ready to cut over'):(prod?'Not yet demonstrated for launch':'Not yet demonstrated for cutover')})])]);
 const why=h('ul',{class:'plain-list val-why'},v.why.map(t=>h('li',{text:t})));
 if(!open)return h('article',{class:'val-row v-'+v.key},[head,why]);
 const setBasis=()=>{r.basis=valBasis(r);r.basisAt=new Date().toISOString();};
 const writers=valWriters(r);
 const capOpts=[['','Not linked'],...valCtxs().map(c=>[c.id,c.name])];
 const waveOpts=[['','Not planned'],...((typeof planStages==='function'?planStages():[]).map(([n,t])=>[String(n),t]))];
 return h('article',{class:'val-row v-'+v.key},[head,why,
  h('div',{class:'val-grid'},[
   valArea('Business journey',r.journey,x=>r.journey=x,'e.g. A customer pays for the last item in stock',1),
   valSelect('Capability it belongs to',r.capId,capOpts,x=>{r.capId=x;}),
   valArea('Design decision it tests',r.decision,x=>r.decision=x,'Which choice this journey depends on'),
   valArea('Consequential failure',r.failure,x=>r.failure=x,'What goes wrong that matters to the business'),
   valArea('Operating condition',r.condition,x=>r.condition=x,'Load, time, data volume or dependency state when it is tested',1),
   valArea('Acceptance measure (threshold)',r.measure,x=>r.measure=x,'The result that counts as working, with numbers'),
   valSelect('Test method',r.method,VAL_METHODS,x=>{r.method=x;}),
   valSelect('Result',r.result,VAL_RESULTS,x=>{r.result=x;r.resultAt=x?new Date().toISOString().slice(0,10):'';if(x==='passed')setBasis();}),
   valArea('Evidence',r.evidence,x=>r.evidence=x,'Link or reference to the test report, rehearsal log or measurements',1),
   valArea('Explicit assumption (if proceeding without a test)',r.assumption,x=>{r.assumption=x;r.accepted=false;},'e.g. The provider guarantees one confirmation per charge (contract clause 7)'),
   r.assumption&&r.owners&&r.owners.approver&&r.owners.approver.confirmed?h('label',{class:'val-check'},[h('input',{type:'checkbox',checked:r.accepted?'':null,onchange:e=>{r.accepted=e.target.checked;if(r.accepted)setBasis();logInput('validation:accept-assumption',!r.accepted,r.accepted);persist();render();}}),' The approver accepts proceeding under this assumption']):null,
   valArea('Open question',r.question,x=>r.question=x,'What is still unknown and could change the decision',1),
   valArea('Business object at stake',r.object,x=>r.object=x,'e.g. stock, payment, order',1)]),
  h('div',{class:'val-owners'},[
   valOwner(r,'approver','Approves the design','Signs the decision this journey tests'),
   valOwner(r,'data','Owns the business data','Decides what is correct when records disagree'),
   valOwner(r,'operator','Operates and restores the service','Runs it, is paged, restores it')]),
  prod?null:h('div',{class:'val-grid val-mod'},[
   h('h4',{class:'val-sub',text:'During the migration'}),
   valArea('Current system',r.current,x=>r.current=x,'Where this journey runs today',1),
   valSelect('Transition wave',r.wave,waveOpts,x=>{r.wave=x;}),
   valArea('Authoritative writer while both systems are live',r.writer,x=>r.writer=x,'Which system may write the business object in this wave',1),
   valArea('Reconciliation test and acceptable variance',r.reconciliation,x=>r.reconciliation=x,'e.g. nightly stock count per item; zero unexplained differences'),
   valArea('Rollback condition and last safe rollback point',r.rollback,x=>r.rollback=x,'Who can stop, when, and until when rollback is safe'),
   valArea('Writes accepted after cutover, if rolled back',r.afterRollback,x=>r.afterRollback=x,'How they reach the old system'),
   h('div',{class:'val-writers'},[h('b',{text:'Known writers and consumers'+(r.object?' of “'+r.object+'”':'')}),
    !r.object?h('p',{class:'muted',text:'Name the business object to list its writers from the discovery records.'}):
    writers.length?h('ul',{class:'plain-list'},writers.map(w=>h('li',{class:w.verified?'':'warn-text',text:(w.kind==='writer'?'Writes: ':'Reads: ')+w.name+(w.verified?' (verified)':' (not verified)')}))):h('p',{class:'warn-text',text:'No discovery record mentions it: writers are unknown.'})])]),
  rd.ready?null:h('p',{class:'muted val-need',text:(prod?'Before launch: ':'Before cutover: ')+'needs '+rd.need.join(', ')+'.'}),
  h('div',{class:'cost-actions'},[button('Remove this journey',()=>{if(!confirm('Remove “'+(r.journey||'this journey')+'” from validation?'))return;const S=valStore();S.rows=S.rows.filter(x=>x!==r);logInput('validation:remove',r.journey,null);persist();render();},'btn small')])]);
}

function valSummary(){
 const rows=valStore().rows,c={enough:0,assume:0,test:0,reopen:0,incomplete:0};let ready=0;
 for(const r of rows){c[valVerdict(r).key]++;if(valReadiness(r).ready)ready++;}
 const J=valJourneys(),covered=J.filter(j=>rows.some(r=>r.sourceKey===j.key||(j.capId&&r.capId===j.capId))),uncovered=J.filter(j=>!covered.includes(j));
 return {rows,c,ready,journeys:J,uncovered};
}

function validationPage(){
 const prod=valProduct(),S=valSummary(),rows=S.rows;
 const T=valTemplates().filter(t=>!rows.some(r=>r.sourceKey===t.key));
 const metric=(n,l,cls)=>h('div',{class:'val-metric '+(cls||'')},[h('b',{text:String(n)}),h('span',{text:l})]);
 return h('div',{class:'val-page'},[
  pageHeader(prod?'✓ / ARCHITECTURE VALIDATION':'✓ / ARCHITECTURE VALIDATION','Will this architecture work for the business?',
   prod?'One row per critical journey: the decision it tests, what could go wrong, how you will prove it and who owns it. Pathfinder says whether there is enough evidence to choose, whether you can proceed under an explicit assumption, or whether a test is needed first.'
       :'One row per critical journey: the decision it tests, what could go wrong during and after migration, who may write the business data in each wave, and how you would roll back. Pathfinder says whether there is enough evidence to choose, proceed under an assumption, or test first.',
   [button('Architecture (HLD)',()=>go('hld'),'btn'),button(prod?'Release plan →':'Migration roadmap →',()=>go('roadmap'),'btn primary')]),
  h('section',{class:'summary-section'},[h('h2',{text:'Where the evidence stands'}),
   h('div',{class:'val-metrics'},[metric(S.c.enough,'Enough evidence to choose','ok'),metric(S.c.assume,'Proceed under an assumption','warn'),metric(S.c.test,'Need a test first','bad'),metric(S.c.reopen,'Reopen','bad'),metric(S.c.incomplete,'Not ready to judge'),metric(S.ready+' of '+rows.length,prod?'Demonstrated ready to launch':'Demonstrated ready to cut over')]),
   h('p',{class:'muted',text:'Approved to build is not the same as ready to '+(prod?'launch':'cut over')+': that needs a passing test with evidence and a confirmed data owner and operator'+(prod?'.':', plus the writer, reconciliation and rollback for the wave.')})]),
  h('section',{class:'summary-section'},[h('h2',{text:'Critical journeys'}),
   rows.length?h('div',{class:'val-list'},rows.map(valRowCard)):h('p',{class:'muted',text:'No journeys yet. Add one from the suggestions below, or from the journeys that are not covered.'})]),
  h('section',{class:'summary-section'},[h('h2',{text:'Suggested from your '+(prod?'brief':'findings')}),
   T.length?h('div',{class:'val-suggest'},T.map(t=>h('div',{class:'val-sugg'},[h('b',{text:t.journey}),h('small',{text:'If it fails: '+t.failure}),t.question?h('small',{class:'muted',text:'Decision it informs: '+t.question}):null,
    button('Add to validation',()=>valAddRow({journey:t.journey,sourceKey:t.key,source:'Suggested',failure:t.failure,condition:t.condition,measure:t.measure,method:t.method,object:t.object,question:t.question||''}),'btn small')]))):h('p',{class:'muted',text:'All suggestions are added.'})]),
  h('section',{class:'summary-section'},[h('h2',{text:'Journeys not covered yet'}),
   S.uncovered.length?h('ul',{class:'scan-list'},S.uncovered.map(j=>h('li',{},[h('b',{text:j.title}),h('small',{text:j.source}),button('Validate this journey',()=>valAddRow({journey:j.title.slice(0,160),sourceKey:j.key,source:j.source,capId:j.capId||''}),'btn small')]))):
   h('p',{class:'muted',text:S.journeys.length?'Every journey from the '+(prod?'brief':'findings')+' has a row.':'No journeys recorded yet: add them to the '+(prod?'product brief':'discovery findings')+'.'}),
   button('Add a journey of my own',()=>valAddRow({journey:'',source:'Added by hand'}),'btn small')])]);
}

/* For the HLD and its exports. */
function validationStatus(){try{const S=valSummary();if(!S.rows.length)return S.journeys.length?S.journeys.length+' journeys to validate':'Prove the design works';return S.c.enough+' proven · '+(S.c.test+S.c.reopen)+' to test'+(S.uncovered.length?' · '+S.uncovered.length+' not covered':'');}catch(e){return '';}}
function validationExport(){
 return valStore().rows.map(r=>{const v=valVerdict(r),rd=valReadiness(r);const own=k=>{const o=(r.owners||{})[k]||{};return o.id?{name:valPerson(o.id),accepted:!!o.confirmed}:null;};
  const out={journey:r.journey,decision:r.decision,failure:r.failure,condition:r.condition,measure:r.measure,method:(VAL_METHODS.find(m=>m[0]===r.method)||[,''])[1],
   result:r.result||'not run',resultAt:r.resultAt||'',evidence:r.evidence||'',assumption:r.assumption||'',assumptionAccepted:!!r.accepted,openQuestion:r.question||'',
   owners:{approvesDesign:own('approver'),ownsData:own('data'),operates:own('operator')},verdict:VAL_VERDICT[v.key][0],verdictKey:v.key,reasons:v.why,
   readyFor:{demonstrated:rd.ready,missing:rd.need}};
  if(!valProduct())Object.assign(out,{currentSystem:r.current||'',wave:r.wave||'',authoritativeWriter:r.writer||'',knownWriters:valWriters(r),reconciliation:r.reconciliation||'',rollback:r.rollback||'',writesAfterRollback:r.afterRollback||''});
  return out;});
}

(function(){const css=document.createElement('style');css.textContent=`
.val-metrics{display:grid;grid-template-columns:repeat(auto-fit,minmax(140px,1fr));gap:10px;margin:8px 0}
.val-metric{border:1px solid var(--line,#e5e7ef);border-radius:10px;padding:10px 12px;background:#fff}
.val-metric b{display:block;font-size:22px}.val-metric span{font-size:12.5px;color:#5f6276}
.val-metric.ok b{color:#1f7a4f}.val-metric.warn b{color:#a8650b}.val-metric.bad b{color:#b3363d}
.val-list{display:grid;gap:12px}
.val-row{border:1px solid var(--line,#e5e7ef);border-left:4px solid #c9cbd8;border-radius:10px;padding:12px 14px;background:#fff}
.val-row.v-enough{border-left-color:#2f9a68}.val-row.v-assume{border-left-color:#d08a1a}.val-row.v-test,.val-row.v-reopen{border-left-color:#c2484f}
.val-head{display:flex;flex-wrap:wrap;gap:8px;align-items:center;justify-content:space-between}
.val-toggle{background:none;border:0;padding:0;text-align:left;cursor:pointer;font:inherit;color:inherit}
.val-badges{display:flex;flex-wrap:wrap;gap:6px}
.val-badge{font-size:12px;padding:3px 9px;border-radius:999px;border:1px solid #d9dbe6;background:#f6f7fb;color:#4a4d60}
.val-badge.ok{background:#eaf7f0;border-color:#b6e0c8;color:#1f6b45}.val-badge.warn{background:#fff5e6;border-color:#f0d29d;color:#8a560b}.val-badge.bad{background:#fdeeee;border-color:#efbcbf;color:#9c2e34}
.val-why{margin:6px 0 0;font-size:13px;color:#4f5266}
.val-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(260px,1fr));gap:10px 14px;margin-top:12px}
.val-field textarea{width:100%;resize:vertical;font:inherit}
.val-owners{display:grid;grid-template-columns:repeat(auto-fit,minmax(220px,1fr));gap:10px 14px;margin-top:12px;padding-top:10px;border-top:1px dashed #e3e5ee}
.val-check{display:flex;gap:6px;align-items:center;font-size:12.5px;margin-top:4px}
.val-sub{grid-column:1/-1;margin:8px 0 0;font-size:13px;text-transform:uppercase;letter-spacing:.6px;color:#6b7389}
.val-writers{grid-column:1/-1}
.val-need{margin-top:10px}
.val-suggest{display:grid;grid-template-columns:repeat(auto-fit,minmax(260px,1fr));gap:10px}
.val-sugg{border:1px dashed #cfd3e2;border-radius:10px;padding:10px 12px;display:grid;gap:6px;align-content:start}
@media print{.val-toggle{pointer-events:none}}`;document.head.append(css);})();
