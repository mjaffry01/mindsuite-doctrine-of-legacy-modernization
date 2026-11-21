/* MindSuite web app · 06 Target decisions: the decision advisor (Phase A)
   Rule-based and explainable, no model calls. For each context it:
     1. scores the seven approaches against the evidence (approachAdvice), with + / − reasons;
     2. rates its confidence and lists the missing inputs, each with a link to where to add it;
     3. builds three complete, consistent target scenarios (Safe, Balanced, Ambitious) from the
        existing option ranking (rankOptions / recommended), with effort, risk and cross-checks;
     4. asks one deciding question when the evidence is thin or the call is close;
     5. drafts the rationale from the chosen target's reasons.
   The person deciding still records the decision; nothing is decided automatically.
   Phase B adds: reasons weighted by What matters most, a robustness check, and portfolioPanel() (all contexts,
   platforms, skills, effort and 3-year cost). Phase C adds reviewPanel(): Claude challenges a target through
   POST /api/decide/review, with consent per request and no rationale, names or unverified claims.
   Loaded by index.html before the main script; decidePage() calls advisedCard(c), which puts this
   panel on top of decisionCard(c) and folds the detailed choices under "Adjust in detail". */

const ADVISOR_OPEN=new Set();   // contexts whose detailed choices are unfolded

/* ---- Inputs ---- */
const REASON_TO_DRIVER={support:'eos',contract:'eos',dependencies:'eos',compatibility:'eos',hosting:'hosting',
 security:'risk',regulation:'risk',privacy:'risk',risk:'risk',recovery:'risk',growth:'scale',performance:'scale',
 cost:'cost',licensing:'cost',energy:'cost','manual-work':'speed',speed:'speed',complexity:'speed',testing:'speed',
 architecture:'speed',skills:'skills',portability:'hosting',experience:'ux',capabilities:'speed',integration:'integration',
 data:'data',merger:'strategy',strategy:'strategy',retirement:'retire',consolidation:'retire',observability:'risk',accessibility:'ux'};
function advisorAnswers(){ensureDecide();return M.decide.advisor||(M.decide.advisor={});}
function advisorInputs(c){
 const S=ctxSignals(c),b=(typeof brief==='function'?brief():{})||{},K=constraints(),A=advisorAnswers();
 const product=isNewProduct(), why=product?[]:[].concat(b.why||[]).filter(Boolean);
 const drivers=new Set(product?[]:[...(M.values.drivers||[]),...why.map(w=>REASON_TO_DRIVER[w]).filter(Boolean)]);
 let months=null,deadlineText='',deadlinePassed='';
 if(b.deadline){const d=new Date(b.deadline);if(!isNaN(d)){
  if(d<new Date(new Date().toDateString()))deadlinePassed=b.deadline;   // a past date is not used for advice
  else{months=Math.max(0,Math.round((d-new Date())/2629800000));deadlineText='the deadline '+b.deadline+(b.deadlineWhat?' ('+b.deadlineWhat+')':'');}}}
 else if(A.horizon){months={lt6:4,'6to12':9,gt12:18,none:null}[A.horizon];deadlineText={lt6:'a hard date within 6 months','6to12':'a hard date in 6–12 months',gt12:'a hard date more than a year away',none:'no hard date'}[A.horizon];}
 const strategic=c.strategic&&c.strategic!=='unsure'?c.strategic:'';
 const P=M.product||{},Dp=P.deploy||{};
 return {S,drivers,P,deploy:Dp,mode:typeof modeOf==='function'?modeOf():'modernize',months,deadlineText,deadlinePassed,hasDeadline:!!((b.deadline&&!deadlinePassed)||A.horizon),strategic,crit:c.criticality||'',
  knowledge:M.values.knowledge||'',risk:K.risk||'',downtime:K.downtime||'',retireSignal:why.includes('retirement')||why.includes('consolidation')||drivers.has('retire')};
}

/* ---- 1. Approach scoring ---- */
function approachAdvice(c){
 const I=advisorInputs(c),S=I.S,D=I.drivers;
 const out=APPROACH.filter(a=>a[0]!=='undecided'&&(typeof approachAllowed!=='function'||approachAllowed(a[0]))).map(([id,label])=>{
  const reasons=[];const add=(v,t,w)=>reasons.push({v,t,w:w||'fit'});
  const soon=I.months!=null&&I.months<6,within=I.months!=null&&I.months<12;
  if(id==='rehost'){
   if(D.has('hosting'))add(2,'The hosting platform or data centre has to change','fit');
   if(D.has('eos'))add(1,'Moves off end-of-support infrastructure quickly','risk');
   if(soon)add(2,'Fits '+I.deadlineText,'effort');
   if(I.risk==='low')add(1,'Low risk appetite: no code change','risk');
   if(['lost','code','vendor'].includes(I.knowledge))add(1,'Business rules are poorly documented; rehosting keeps behaviour unchanged','risk');
   if(D.has('speed')||D.has('ux')||D.has('integration'))add(-2,'Does not fix '+[D.has('speed')&&'slow change',D.has('ux')&&'user experience',D.has('integration')&&'integration'].filter(Boolean).join(', '),'fit');
   if(I.strategic==='core')add(-1,'Core domain: moving it unchanged keeps its limits','fit');
  }
  if(id==='replatform'){
   if(D.has('eos'))add(2,'Replaces end-of-support platforms with supported, managed ones','risk');
   if(D.has('cost'))add(1,'Managed services usually cut running and licence cost','cost');
   if(I.strategic==='supporting')add(1,'Supporting subdomain: modernize the platform, keep the logic','fit');
   if(I.months!=null&&I.months>=4&&I.months<12)add(1,'Fits '+I.deadlineText,'effort');
   if(S.db==='oracle'&&D.has('cost'))add(1,'Oracle licence cost can be reduced by a managed database','cost');
   if(D.has('speed')&&I.strategic==='core')add(-1,'Core domain that must change faster: a platform move alone may not be enough','fit');
  }
  if(id==='refactor'){
   if(I.strategic==='core')add(2,'Core domain: worth restructuring for faster change','fit');
   if(D.has('speed'))add(2,'Main problem is slow change','fit');
   if(D.has('scale'))add(1,'Restructuring addresses performance and scale limits','fit');
   if(S.noTests)add(-2,'No automated tests to protect a restructure','risk');
   if(S.shared.length)add(-1,'Shared data ('+S.shared.length+' store'+(S.shared.length===1?'':'s')+') must be untangled first','effort');
   if(I.risk==='low')add(-1,'Low risk appetite','risk');
   if(soon)add(-2,'Too large for '+I.deadlineText,'effort');
  }
  if(id==='rebuild'){
   if(I.strategic==='core')add(2,'Core domain: custom work pays off where the business is different','fit');
   if(D.has('skills')&&S.lang==='cobol')add(2,'Skills shortage on COBOL / mainframe','skills');else if(D.has('skills'))add(1,'Skills shortage on the current technology','skills');
   if(['experts','docs'].includes(I.knowledge))add(1,'Business rules are known by experts or documented','risk');
   if(['lost','code','vendor'].includes(I.knowledge))add(-2,'Rules are '+({lost:'largely lost',code:'only in the code',vendor:'held by a vendor'})[I.knowledge]+': a rewrite risks losing them','risk');
   if(within)add(-2,'Too large for '+I.deadlineText,'effort');
   if(I.risk==='low')add(-1,'Low risk appetite','risk');
   if(I.crit==='mission')add(-1,'Mission-critical: a full rewrite is the riskiest switch','risk');
   if(I.strategic==='generic')add(-2,'Generic subdomain: buying is usually cheaper than building','cost');
  }
  if(id==='replace'){
   if(I.strategic==='generic')add(3,'Generic subdomain: a product or SaaS already does this','fit');
   if(I.strategic==='core')add(-3,'Core domain: a standard product removes what makes the business different','fit');
   if(D.has('cost'))add(1,'Can lower total cost of ownership','cost');
   if(S.rules.length>30)add(-1,S.rules.length+' recorded business rules to fit into a product','effort');
   else if(S.rules.length&&S.rules.length<=10)add(1,'Few custom rules ('+S.rules.length+')','effort');
  }
  if(id==='encapsulate'){
   if(D.has('integration'))add(2,'Main problem is integration: APIs open the system to new channels','fit');
   if(I.risk==='low')add(1,'Low risk appetite: the core stays as it is','risk');
   if(I.crit==='mission'&&S.noTests)add(1,'Mission-critical without tests: wrap before touching it','risk');
   if(D.has('eos'))add(-2,'Does not remove the end-of-support technology','risk');
   if(D.has('skills'))add(-1,'The skills problem stays','skills');
   if(soon)add(1,'Quick to deliver for '+I.deadlineText,'effort');
  }
  if(id==='build'){
   if(I.strategic==='core')add(3,'Core capability: custom work is where the business is different','fit');
   if(I.strategic==='generic')add(-3,'Generic capability: building what every business has wastes effort','cost');
   if(R_skills_has_lang())add(1,'The team already has the skills for custom development','skills');else add(-1,'No team skills recorded for custom development','skills');
   if(soon)add(-2,'A custom build is unlikely to reach a first release by '+I.deadlineText,'effort');
   if(I.risk==='low')add(-1,'Low risk appetite: custom builds carry more delivery risk','risk');
   if(String(I.P.buyPref||'').match(/prefer not to build|avoid custom|buy/i))add(-1,'The brief prefers not to build','fit');
  }
  if(id==='buy'){
   if(I.strategic==='generic')add(3,'Generic capability: a product or SaaS already does this','fit');
   if(I.strategic==='core')add(-3,'Core capability: a standard product removes what makes the business different','fit');
   if(soon)add(2,'Fastest route to a first release by '+I.deadlineText,'effort');
   if(I.deploy.where==='onprem'||I.deploy.where==='edge')add(-1,'Deployment is '+(I.deploy.where==='edge'?'on-site':'on-premises')+': fewer products fit','fit');
   if(I.deploy.tenancy==='saas-multi'||I.deploy.tenancy==='saas-single'||I.deploy.tenancy==='customer-hosted')add(-1,'You sell this capability to customers: a bought product limits how you differentiate','fit');
   if(String(I.P.buyPref||'').trim())add(1,'The brief names existing products or contracts','cost');
  }
  if(id==='partner'){
   if(I.strategic==='supporting')add(2,'Supporting capability: extending a platform or partner product balances fit and speed','fit');
   if(I.months!=null&&I.months<12)add(1,'Faster than a custom build for '+I.deadlineText,'effort');
   if(I.strategic==='core')add(-1,'Core capability: a platform limits how far it can differ','fit');
   add(-1,'Ties the capability to the platform\'s roadmap and licence','cost');
  }
  if(id==='retire'){
   if(I.retireSignal)add(3,'The interview says the process is retiring or systems are being consolidated','fit');
   else add(-2,'No sign that the system is unused or duplicated','fit');
  }
  // Each reason counts by the weight of its criterion (05 Rules & people, What matters most; 3 = normal).
  const W=M.decide.rules.weights||{};
  return {id,label,reasons,score:Math.round(reasons.reduce((n,r)=>n+r.v*(+(W[r.w]??3))/3,0)*10)/10};
 }).sort((a,b)=>b.score-a.score);
 return {ranked:out,inputs:I,confidence:advisorConfidence(c,I),close:out[1]&&out[0].score-out[1].score<=1};
}

function R_skills_has_lang(){const R=M.decide.rules||{};return (R.skills||[]).some(x=>['java','dotnet','node','python','go'].includes(x));}

/* ---- 2. Confidence and missing inputs ---- */
function advisorConfidence(c,I){
 const S=I.S;
 const items=[
  ['Strategic value',!!I.strategic,()=>{if(c.virtual)go('review');else{invTab='contexts';go('inventory');}}],
  ['Criticality',!!I.crit,()=>{if(c.virtual)go('review');else{invTab='contexts';go('inventory');}}],
  ['Modernization reasons',I.drivers.size>0,()=>showEvidenceStep('project')],
  ['Who knows the business rules',!!I.knowledge,()=>go('review')],
  [I.deadlinePassed?'Deadline ('+I.deadlinePassed+' has passed)':'Deadline',I.hasDeadline,()=>showEvidenceStep('project')],
  ['Risk appetite',!!I.risk,()=>{rulesTab='priorities';go('rules');}],
  ['Code scanned',!!S.lang,()=>showEvidenceStep('sources')],
  ['Bounded contexts',S.contexts>0,()=>{invTab='contexts';go('inventory');}]];
 if(I.mode==='new'){
  const brief=()=>go('upload');
  const values=()=>document.getElementById('decision-'+c.id)?.querySelector('.ctx-values select')?.focus();
  items.splice(0,items.length,
   ['Strategic value',!!I.strategic,values],['Criticality',!!I.crit,values],
   ['Problem and users',!!(String(I.P.problem||'').trim()&&String(I.P.users||'').trim()),brief],['First release (MVP)',!!String(I.P.mvp||'').trim(),brief],
   ['Scale (users or peak load)',!!(I.P.users_count||I.P.peak),brief],['Deployment type',!!I.deploy.where,brief],
   [I.deadlinePassed?'Deadline ('+I.deadlinePassed+' has passed)':'Deadline',I.hasDeadline,brief],['Risk appetite',!!I.risk,()=>{rulesTab='priorities';go('rules');}]);
 }
 const have=items.filter(x=>x[1]).length;
 return {level:have>=7?'high':have>=4?'medium':'low',have,total:items.length,missing:items.filter(x=>!x[1]).map(([label,,go])=>({label,go}))};
}

/* ---- 3. Scenarios ---- */
function buildScenario(c,approach){
 const d={approach,choices:{integration:[]}};
 for(const dim of ['architecture','language','database','integration','cloud']){
  if(!(APPROACH_DIMS[approach]||[]).includes(dim))continue;
  if((APPROACH_FIXED[approach]||{})[dim])continue;
  if(TARGETS[dim].multi){d.choices[dim]=recommended(dim,c,d);continue;}
  const r=rankOptions(dim,c,d).filter(o=>!o.excluded);d.choices[dim]=r[0]?r[0].id:'';
  if(r[1]&&r[0]&&r[1].score===r[0].score)d.tie=(d.tie||[]).concat(dim);
 }
 return d;
}
function scenarioRisk(c,approach,I){
 let r={rehost:0,encapsulate:0,retire:0,replatform:1,replace:1,refactor:2,rebuild:2}[approach]??1;
 if(I.S.noTests&&['refactor','rebuild'].includes(approach))r++;
 if(I.crit==='mission'&&r>=1)r++;
 return ['Low','Medium','High'][Math.min(2,r)];
}
function advisorScenarios(c,adv){
 const rk=adv.ranked,pick=ids=>rk.filter(a=>ids.includes(a.id))[0];
 const safe=pick(['rehost','encapsulate','replatform','buy']),bold=pick(['rebuild','refactor','replace','build']);
 // Balanced: the overall best when it is a middle option, otherwise the best remaining middle option.
 const middle=['replatform','refactor','encapsulate','replace','partner','buy'];
 const bal=(middle.includes(rk[0].id)&&rk[0]!==safe&&rk[0]!==bold?rk[0]:null)||rk.find(a=>middle.includes(a.id)&&a!==safe&&a!==bold)||rk.find(a=>a!==safe&&a!==bold&&a.id!=='retire');
 const list=[['Safe','Least change and risk',safe],['Balanced','Best overall fit',bal],['Ambitious','Most long-term benefit',bold]];
 if(rk[0].id==='retire'&&rk[0].score>0)list.unshift(['Retire','The evidence points to switching it off',rk[0]]);
 const top=rk[0].id;
 return list.filter(x=>x[2]).map(([name,tag,a])=>{
  const d=buildScenario(c,a.id);
  return {name,tag,a,d,best:a.id===top&&a.score>0,size:autoSize(c,d),risk:scenarioRisk(c,a.id,adv.inputs),checks:crossChecks(c,d,adv.inputs)};
 });
}
function scenarioSummary(d){
 const parts=[];const fixed=APPROACH_FIXED[d.approach]||{};
 for(const dim of ['architecture','language','database','integration','cloud']){
  const v=fixed[dim]||d.choices[dim];if(!v||(Array.isArray(v)&&!v.length))continue;
  parts.push([].concat(v).map(id=>(optOf(dim,id)||[,id])[1]).join(' + '));
 }
 if(parts.length)return parts.join(' · ');
 return (APPROACH_DIMS[d.approach]||[]).length?'Technology not chosen yet':'No technology choices for this approach';
}

/* ---- 4. Cross-checks ---- */
function crossChecks(c,d,I){
 I=I||advisorInputs(c);const out=[];const eff=dim=>effective(dim,d);
 for(const v of violations(c,d))out.push(['block',v]);
 if(I.deadlinePassed)out.push(['info','The deadline in the project interview ('+I.deadlinePassed+') has passed, so it is not used. Update it in the interview.']);
 const big={rebuild:12,refactor:6}[d.approach];
 if(big&&I.months!=null&&I.months<big)out.push(['warn',apprLabel(d.approach)+' is '+autoSize(c,d)+' effort, but there is '+I.deadlineText+'.']);
 if(eff('architecture')==='serverless'&&['onprem','private'].includes(eff('cloud')))out.push(['warn','Serverless functions need a cloud function platform; '+optOf('cloud',eff('cloud'))[1].toLowerCase()+' rarely offers one.']);
 if(['microservices','events'].includes(eff('architecture'))&&I.risk==='low')out.push(['warn',optOf('architecture',eff('architecture'))[1]+' is a large change for a low risk appetite.']);
 if(I.downtime==='none'&&['rehost','replatform','rebuild','refactor'].includes(d.approach)&&!(eff('integration')||[]).includes('cdc'))out.push(['warn','Zero cut-over downtime: plan change data capture or a parallel run.']);
 if(d.approach==='rebuild'&&['lost','code','vendor'].includes(I.knowledge))out.push(['warn','Business rules are not documented: extract them from the code (Source code team) before rewriting.']);
 if(['refactor','rebuild'].includes(d.approach)&&eff('language')==='keep'&&!I.S.lang)out.push(['warn','The current language is not known yet, so "keep current language" is a guess. Scan the code before choosing the target language.']);
 if(['refactor','rebuild'].includes(d.approach)&&I.S.noTests)out.push(['warn','No automated tests: add characterisation tests before changing the code.']);
 if(d.approach==='replace'&&!String(d.product||'').trim())out.push(['info','Name the replacement product.']);
 const dw=(I.deploy||{}).where;const cl=eff('cloud');
 if(dw&&cl){const pub=['aws','azure','gcp','oci','ibm'].includes(cl)||String(cl).startsWith('org-');
  if((dw==='onprem'||dw==='edge')&&pub&&!String(cl).startsWith('org-'))out.push(['warn','The deployment type is '+(dw==='edge'?'on-site / edge':'on-premises')+', but the target is '+optOf('cloud',cl)[1]+'.']);
  if((dw==='public'||dw==='multicloud')&&(cl==='onprem'||cl==='private'))out.push(['warn','The deployment type is public cloud, but the target is '+optOf('cloud',cl)[1]+'.']);
  if(dw==='private'&&pub&&!String(cl).startsWith('org-'))out.push(['warn','The deployment type is private cloud, but the target is '+optOf('cloud',cl)[1]+'.']);}
 const cloud=eff('cloud');
 if(cloud){const others=decCtxs().filter(x=>x.id!==c.id).map(x=>effective('cloud',decisionFor(x))).filter(Boolean);const set=new Set(others);
  if(set.size&&!set.has(cloud))out.push(['info','Other contexts use '+[...set].map(id=>(optOf('cloud',id)||[,id])[1]).join(', ')+'; '+optOf('cloud',cloud)[1]+' adds another platform to run.']);}
 for(const dim of d.tie||[])out.push(['info','No clear winner for '+TARGETS[dim].title.toLowerCase()+'; the first of the tied options was taken. Answer the question below or decide on cost, contracts or skills.']);
 return out;
}

/* ---- 5. One deciding question ---- */
function advisorQuestion(c,adv,scen){
 const I=adv.inputs,A=advisorAnswers();
 const setCtx=(k,v)=>{if(c.virtual){const before=M.values[k];M.values[k]=v;M.status[k]='confirmed';logInput('advisor:'+k,before,v);}else setDecisionQuiet(c,k==='subdomain'?'strategic':k,v);persist();render();};
 if(!I.strategic)return {q:'Is '+c.name+' what makes your business different from competitors?',why:'Strategic value moves the advice most: build for core, buy for generic.',
  a:[['Yes, it is core',()=>setCtx(c.virtual?'subdomain':'strategic','core')],['Needed, but not special',()=>setCtx(c.virtual?'subdomain':'strategic','supporting')],['Every business has it',()=>setCtx(c.virtual?'subdomain':'strategic','generic')]]};
 if(!I.hasDeadline)return {q:I.deadlinePassed?'The deadline in the interview ('+I.deadlinePassed+') has passed. When must this be done now?':'Is there a hard date this must be done by?',why:'A near deadline rules out large rewrites.',
  a:[['Within 6 months','lt6'],['6–12 months','6to12'],['Over a year','gt12'],['No hard date','none']].map(([t,v])=>[t,()=>{const b=A.horizon;A.horizon=v;logInput('advisor:horizon',b,v);persist();render();}])};
 if(!I.risk)return {q:'How much change can the business absorb at once?',why:'Risk appetite separates gradual moves from big switches.',
  a:[['Gradual only','low'],['Some','medium'],['A big switch is fine','high']].map(([t,v])=>[t,()=>{const K=constraints();const b=K.risk;K.risk=v;logInput('constraint:risk',b,v);persist();render();}])};
 if(!I.knowledge)return {q:'Who knows the business rules today?',why:'Rewriting is only safe when the rules are known.',
  a:[['Active experts','experts'],['Documentation','docs'],['Only the code','code'],['Largely lost','lost']].map(([t,v])=>[t,()=>{const b=M.values.knowledge;M.values.knowledge=v;M.status.knowledge='confirmed';logInput('advisor:knowledge',b,v);persist();render();}])};
 const cloudTie=scen.some(s=>(s.d.tie||[]).includes('cloud'));
 if(cloudTie){const pub=['azure','aws','gcp','oci'];return {q:'Does your organization already have a cloud agreement or standard?',why:'Several clouds score the same; an existing agreement usually decides it.',
  a:[...pub.map(id=>[optOf('cloud',id)[1],()=>{setRule('cloud',id,'preferred',{reason:'Existing agreement (answered in Target decisions)',owner:M.actor||''});persist();render();}]),['None yet',()=>{const b=A.noCloud;A.noCloud=true;logInput('advisor:noCloud',b,true);persist();render();}]]};}
 return null;
}

/* ---- 6. Draft rationale ---- */
function draftRationale(c,d,adv){
 const a=adv.ranked.find(x=>x.id===d.approach);const pos=(a?a.reasons:[]).filter(r=>r.v>0).map(r=>r.t.toLowerCase());
 const lines=[apprLabel(d.approach)+(pos.length?' because '+pos.slice(0,3).join('; '):'')+'.'];
 const fixed=APPROACH_FIXED[d.approach]||{};const tgt=[];
 for(const dim of APPROACH_DIMS[d.approach]||[]){
  const ids=[].concat(fixed[dim]||d.choices[dim]||[]).filter(Boolean);if(!ids.length)continue;
  const ranked=rankOptions(dim,c,d);
  for(const id of ids){const o=ranked.find(x=>x.id===id);const why=o&&o.reasons.find(r=>r.v>0&&r.t);tgt.push(o?o.label+(why?' ('+why.t.charAt(0).toLowerCase()+why.t.slice(1)+')':''):id);}
 }
 if(tgt.length)lines.push('Target: '+tgt.join('; ')+'.');
 const warn=crossChecks(c,d,adv.inputs).filter(x=>x[0]==='warn').map(x=>x[1]);
 if(warn.length)lines.push('Accepted risks: '+warn.join(' '));
 lines.push('Advisor confidence: '+adv.confidence.level+(adv.confidence.missing.length?' (open: '+adv.confidence.missing.map(m=>m.label.toLowerCase()).join(', ')+')':'')+'.');
 return lines.join(' ');
}

/* ---- Panel ---- */
function useScenario(c,d,s){
 changeDecision(c,d,()=>{d.approach=s.a.id;if(!c.virtual)setDecisionQuiet(c,'approach',s.a.id);
  for(const [k,v] of Object.entries(s.d.choices))d.choices[k]=Array.isArray(v)?[...v]:v;});
 logInput('advisor:scenario:'+c.id,null,s.name+': '+s.a.id);
}
function advisorPanel(c,d){
 const adv=approachAdvice(c),scen=advisorScenarios(c,adv),conf=adv.confidence,top=adv.ranked[0];
 const q=advisorQuestion(c,adv,scen);
 const chosen=d.approach&&d.approach!=='undecided'?d:null;
 const checks=chosen?crossChecks(c,d,adv.inputs):[];
 const reasonChip=r=>h('span',{class:'reason '+(r.v>0?'pos':r.v<0?'neg':'neu'),text:(r.v>0?'+ ':r.v<0?'− ':'• ')+r.t});
 return h('section',{class:'advisor'},[
  h('div',{class:'advisor-head'},[
   h('div',{},[h('small',{class:'eyebrow',text:'Advisor'}),
    h('h4',{text:top.score>0?'Recommended approach: '+top.label:'No approach stands out yet'}),
    h('div',{class:'reason-chips'},top.reasons.slice(0,4).map(reasonChip)),
    adv.close&&adv.ranked[1]?h('p',{class:'muted',text:'Close call with '+adv.ranked[1].label+' ('+adv.ranked[1].score+' against '+top.score+').'}):null]),
   h('div',{class:'advisor-conf '+conf.level,title:conf.have+' of '+conf.total+' key inputs known'},[h('b',{text:conf.level[0].toUpperCase()+conf.level.slice(1)+' confidence'}),h('small',{text:conf.have+' of '+conf.total+' key inputs'})])]),
  conf.missing.length?h('div',{class:'advisor-missing'},[h('span',{class:'muted',text:'Would make this surer: '}),...conf.missing.map(m=>button(m.label,m.go,'linkbtn'))]):null,
  q?h('div',{class:'advisor-question'},[h('b',{text:q.q}),h('small',{class:'muted',text:q.why}),h('div',{class:'advisor-answers'},q.a.map(([t,fn])=>button(t,fn,'btn small')))]):null,
  h('div',{class:'scenario-grid'},scen.map(s=>{const on=chosen&&d.approach===s.a.id&&JSON.stringify(s.d.choices.cloud||'')===JSON.stringify(d.choices.cloud||'');
   const blocks=s.checks.filter(x=>x[0]==='block').length,warns=s.checks.filter(x=>x[0]==='warn').length;
   return h('article',{class:'scenario'+(s.best?' best':'')+(on?' chosen':'')},[
    h('div',{class:'scenario-top'},[h('b',{text:s.name}),s.best?h('span',{class:'fit-tag',text:'best fit'}):null]),
    h('small',{class:'muted',text:s.tag}),
    h('h5',{text:s.a.label}),
    h('p',{class:'scenario-tech',text:scenarioSummary(s.d)}),
    h('dl',{class:'scenario-facts'},[h('dt',{text:'Fit'}),h('dd',{text:(s.a.score>0?'+':'')+(Number.isInteger(s.a.score)?s.a.score:s.a.score.toFixed(1))}),h('dt',{text:'Effort'}),h('dd',{text:s.size}),h('dt',{text:'Risk'}),h('dd',{text:s.risk}),
     h('dt',{text:'Checks'}),h('dd',{class:blocks?'bad':warns?'warn':'ok',text:blocks?blocks+' breaks rules':warns?warns+' to watch':'none'})]),
    h('div',{class:'reason-chips'},s.a.reasons.slice(0,3).map(reasonChip)),
    on?h('span',{class:'badge confirmed',text:'Current target'}):button('Use this scenario',()=>useScenario(c,d,s),'btn small'+(s.best?' primary':''))]);})),
  chosen?h('div',{class:'advisor-current'},[
   h('b',{text:'Current target: '+apprLabel(d.approach)}),h('span',{class:'muted',text:' · '+scenarioSummary(d)}),
   checks.length?h('ul',{class:'advisor-checks'},checks.map(([lvl,t])=>h('li',{class:lvl,text:(lvl==='block'?'✕ ':lvl==='warn'?'! ':'i ')+t}))):h('p',{class:'muted',text:'No conflicts found with your rules, deadline or other contexts.'}),
   (()=>{const rb=robustness(c,d);return h('div',{class:'advisor-robust '+(rb.robust?'ok':'warn')},[h('b',{text:rb.robust?'Robust':'Sensitive to your priorities'}),
    h('small',{class:'muted',text:rb.robust?' · the choice holds if any criterion in What matters most counts a little more or less.':' · it changes if one criterion counts a little more or less:'}),
    rb.notes.length?h('ul',{class:'plain-list'},rb.notes.slice(0,4).map(t=>h('li',{text:t}))):null]);})(),
   h('div',{class:'advisor-actions'},[button('Draft rationale',()=>{const t=draftRationale(c,d,adv);if(String(d.rationale||'').trim()&&!confirm('Replace the current rationale with a draft?'))return;d.rationale=t;logInput('advisor:rationale:'+c.id,null,'drafted');persist();render();},'btn small')]),
   reviewPanel(c,d,adv)]):null
 ]);
}
function advisedCard(c){
 const d=decisionFor(c);const card=decisionCard(c);
 let panel;try{panel=advisorPanel(c,d);}catch(e){console.error('Advisor failed for '+c.name,e);return card;}
 const kids=[...card.children];const start=kids.findIndex(x=>x.classList&&x.classList.contains('approach-row'));
 const footer=kids.findIndex(x=>x.classList&&x.classList.contains('decide-footer'));
 if(start<0||footer<0){card.insertBefore(panel,card.children[1]||null);return card;}
 const detail=h('details',{class:'advisor-detail',open:ADVISOR_OPEN.has(c.id)||d.status==='decided'&&!d.approach?'':null,
  ontoggle:e=>{e.target.open?ADVISOR_OPEN.add(c.id):ADVISOR_OPEN.delete(c.id);}},[h('summary',{text:'Adjust in detail: approach, each technology choice, services, hosting and consistency'})]);
 // Verified claims and Claude suggestions stay visible; everything else between the approach and the footer folds.
 const keep=el=>el.classList&&(el.classList.contains('gate-row')||el.classList.contains('ai-row'));
 const section=kids.slice(start,footer);const anchor=section.find(keep)||kids[footer];
 card.insertBefore(panel,kids[start]);
 if(typeof optionsPanel==='function'){try{card.insertBefore(optionsPanel(c,d),kids[start]);}catch(e){console.error('Options panel failed for '+c.name,e);}}
 card.insertBefore(detail,anchor);
 for(const el of section)if(!keep(el))detail.append(el);
 return card;
}


/* ==== Phase B ==== */

/* ---- 7. Robustness: does the choice hold if one criterion counts a little more or less? ---- */
function withWeights(W2,fn){const R=M.decide.rules,old=R.weights;R.weights=W2;try{return fn();}finally{R.weights=old;}}
function robustness(c,d){
 const W=M.decide.rules.weights||{},notes=[],seen=new Set();
 const approachNow=d.approach&&d.approach!=='undecided'?d.approach:approachAdvice(c).ranked[0].id;
 let dims=(APPROACH_DIMS[d.approach]||[]).filter(dim=>!TARGETS[dim].multi&&!(APPROACH_FIXED[d.approach]||{})[dim]&&d.choices[dim]);
 // A choice that already loses under the current priorities is reported once, not as a "what if".
 const label0=(dim,id)=>(optOf(dim,id)||[,id])[1];
 const top0=approachAdvice(c).ranked[0];
 if(d.approach&&d.approach!=='undecided'&&top0.id!==d.approach&&top0.score>0)notes.push('With your current priorities, '+top0.label+' already scores higher than '+apprLabel(d.approach)+'.');
 dims=dims.filter(dim=>{const r=rankOptions(dim,c,d).filter(o=>!o.excluded);
  if(r[0]&&r[0].id!==d.choices[dim]&&(!r[1]||r[0].score>r[1].score)){notes.push('With your current priorities, '+r[0].label+' already ranks above '+label0(dim,d.choices[dim])+' for '+TARGETS[dim].title.toLowerCase()+'.');return false;}
  return true;});
 for(const [k,label] of WEIGHT_KEYS)for(const delta of [2,-2]){
  const v=Math.max(0,Math.min(5,(+(W[k]??3))+delta));if(v===+(W[k]??3))continue;
  const W2={...W,[k]:v},more=delta>0?'more':'less';
  withWeights(W2,()=>{
   const top=approachAdvice(c).ranked[0];
   if(top.id!==approachNow&&top.score>0){const key='a:'+top.id+k+more;if(!seen.has(key)){seen.add(key);notes.push('If '+label.toLowerCase()+' counted '+more+', '+top.label+' would be recommended instead of '+apprLabel(approachNow)+'.');}}
   for(const dim of dims){const r=rankOptions(dim,c,d).filter(o=>!o.excluded);
    if(r[0]&&r[0].id!==d.choices[dim]&&(!r[1]||r[0].score>r[1].score)){const key=dim+r[0].id+k+more;if(!seen.has(key)){seen.add(key);notes.push('If '+label.toLowerCase()+' counted '+more+', '+r[0].label+' would rank above '+(optOf(dim,d.choices[dim])||[,d.choices[dim]])[1]+' for '+TARGETS[dim].title.toLowerCase()+'.');}}}
  });
 }
 return {robust:!notes.length,notes};
}

/* ---- 8. Portfolio: all contexts together, effort and 3-year cost ---- */
const PERSON_MONTHS={S:[2,6],M:[6,18],L:[18,48],XL:[48,120]};
function portfolioPanel(C){
 ensureDecide();const R=M.decide.rules,B=M.decide.businessCase||(M.decide.businessCase={currency:'USD'});
 const rows=C.map(c=>{const d=decisionFor(c);const adv=approachAdvice(c);const has=d.approach&&d.approach!=='undecided';
  const appr=has?d.approach:adv.ranked[0].score>0?adv.ranked[0].id:'';
  return {c,d,adv,has,appr,size:appr?autoSize(c,has?d:{approach:appr,choices:{}}):'',cloud:has?effective('cloud',d):'',db:has?effective('database',d):'',lang:has?effective('language',d):''};});
 const label=(dim,id)=>!id?'':id==='keep'?'Keep current':(optOf(dim,id)||[,id])[1];
 const distinct=dim=>[...new Set(rows.map(r=>r[dim==='cloud'?'cloud':dim==='database'?'db':'lang']).filter(x=>x&&x!=='keep'))];
 const clouds=distinct('cloud'),dbs=distinct('database'),langs=distinct('language');
 const skillGap=langs.filter(l=>!R.skills.includes(l));
 const pm=rows.filter(r=>r.size).reduce((a,r)=>[a[0]+PERSON_MONTHS[r.size][0],a[1]+PERSON_MONTHS[r.size][1]],[0,0]);
 const cur=B.currency||'',fmt=n=>Math.round(n).toLocaleString()+' '+cur;
 const product=isNewProduct(),today=product?0:runCost(),target=num(B.targetRun),rate=num(B.personMonth);
 const oneOff=rate?[pm[0]*rate,pm[1]*rate]:null;
 const set=k=>v=>{const b=B[k];B[k]=v;logInput('cost:'+k,b,v);persist();render();};
 const costRows=[];
 if(today)costRows.push(['Today, 3 years',fmt(today*3),'from the business case in 05 Rules & people']);
 if(oneOff)costRows.push([product?'Implementation, one-off':'Migration, one-off',fmt(oneOff[0])+' – '+fmt(oneOff[1]),pm[0]+'–'+pm[1]+' person-months (rule of thumb by effort size)']);
 if(target)costRows.push(['Target running, 3 years',fmt(target*3),'your estimate per year × 3']);
 if(product&&target&&oneOff)costRows.push(['Product total, 3 years',fmt(oneOff[0]+target*3)+' – '+fmt(oneOff[1]+target*3),'Implementation and running costs']);
 if(today&&target&&oneOff){const lo=oneOff[0]+target*3,hi=oneOff[1]+target*3;costRows.push(['Target total, 3 years',fmt(lo)+' – '+fmt(hi),hi<today*3?'cheaper than today within 3 years, even at the high estimate':lo<today*3?'cheaper than today only at the low estimate':'more than today over 3 years']);
  const save=today-target;costRows.push(['Payback',save>0?(oneOff[0]/save).toFixed(1)+' – '+(oneOff[1]/save).toFixed(1)+' years':'no yearly saving',save>0?'one-off cost ÷ yearly saving of '+fmt(save):'the target costs as much or more to run']);}
 return h('section',{class:'summary-section portfolio'},[
  h('div',{class:'portfolio-head'},[h('h2',{text:'Portfolio view'}),h('small',{class:'muted',text:C.length+(product?(C.length===1?' capability':' capabilities'):(C.length===1?' context':' contexts'))+' · '+rows.filter(r=>r.d.status==='decided').length+' decided'})]),
  h('table',{class:'scan-table portfolio-table'},[h('thead',{},[h('tr',{},[product?'Capability':'Context','Approach','Cloud','Database','Language','Effort','Confidence','Status'].map(t=>h('th',{text:t})))]),
   h('tbody',{},rows.map(r=>h('tr',{},[h('td',{text:r.c.name}),h('td',{},[r.appr?apprLabel(r.appr):'—',!r.has&&r.appr?h('small',{class:'muted',text:' (suggested)'}):null]),
    h('td',{text:label('cloud',r.cloud)||'—'}),h('td',{text:label('database',r.db)||'—'}),h('td',{text:label('language',r.lang)||'—'}),h('td',{text:r.size||'—'}),
    h('td',{class:'conf-'+r.adv.confidence.level,text:r.adv.confidence.level}),h('td',{text:r.d.status==='decided'?(violations(r.c,r.d).length?'Breaks rules':'Decided'):r.d.reopened?'Changed':'Open'})])))]),
  h('div',{class:'portfolio-facts'},[
   h('div',{},[h('b',{text:'Platforms to run'}),h('p',{text:[clouds.length&&clouds.length+' cloud'+(clouds.length===1?'':'s')+' ('+clouds.map(x=>label('cloud',x)).join(', ')+')',dbs.length&&dbs.length+' database engine'+(dbs.length===1?'':'s'),langs.length&&langs.length+' new language'+(langs.length===1?'':'s')].filter(Boolean).join(' · ')||'None chosen yet'}),
    clouds.length>1?h('small',{class:'warn-text',text:'More than one cloud: more skills, tooling and contracts to run.'}):null]),
   h('div',{},[h('b',{text:'Skills'}),h('p',{text:skillGap.length?'Not yet a team skill: '+skillGap.map(x=>label('language',x)).join(', '):langs.length?'The team already knows the chosen languages':'No new language chosen'})]),
   h('div',{},[h('b',{text:'Effort'}),h('p',{text:pm[1]?pm[0]+'–'+pm[1]+' person-months in total':'Choose or accept approaches first'}),h('small',{class:'muted',text:'Rule of thumb: S 2–6, M 6–18, L 18–48, XL 48–120 person-months.'})])]),
  h('details',{class:'compare portfolio-cost',open:costRows.length>1?'':null},[h('summary',{text:product?'3-year product cost':'3-year cost against today'}),
   h('div',{class:'rules-grid'},[inputControl('Cost of one person-month',B.personMonth,set('personMonth'),'number','e.g. 12000'),inputControl(product?'Expected yearly product running cost':'Expected yearly running cost after the move',B.targetRun,set('targetRun'),'number','e.g. 90000'),inputControl('Currency',B.currency,set('currency'),'text','USD')]),
   !product&&!today?h('p',{class:'muted',text:'Add today\'s yearly run cost in 05 Rules & people (Priorities & limits → Money) to compare.'}):null,
   costRows.length?h('dl',{class:'inv-meta cost-table'},costRows.flatMap(([k,v,n])=>[h('dt',{text:k}),h('dd',{},[h('b',{text:v}),h('small',{class:'muted',text:' · '+n})])])):null,
   h('small',{class:'muted',text:'Estimates help compare options. Refine effort and cost when planning delivery.'})])
 ]);
}

/* ==== Phase C: Claude challenges a decision ==== */
const reviewUI={consent:{},busy:{},error:{}};
function reviewHash(s){let h1=0;for(let i=0;i<s.length;i++)h1=(h1*31+s.charCodeAt(i))|0;return String(h1);}
function reviewPayload(c,d,adv){
 const fixed=APPROACH_FIXED[d.approach]||{};const target=[],reasons=[];
 const a=adv.ranked.find(x=>x.id===d.approach);
 for(const r of (a?a.reasons:[]))reasons.push((r.v>0?'+ ':'− ')+'Approach: '+r.t);
 for(const dim of APPROACH_DIMS[d.approach]||[]){
  const ids=[].concat(fixed[dim]||d.choices[dim]||[]).filter(Boolean);const ranked=rankOptions(dim,c,d);
  for(const id of ids){const o=ranked.find(x=>x.id===id);target.push({area:TARGETS[dim].title,option:o?o.label:id});
   for(const r of (o?o.reasons:[]).filter(r=>r.t).slice(0,3))reasons.push((r.v>0?'+ ':r.v<0?'− ':'• ')+(o.label)+': '+r.t);}
 }
 let claims=[],sent=[];
 if(typeof suggestPayload==='function'){const p=suggestPayload(c,d);sent=p.sent;claims=p.body.claims;}
 const body={strategic:String(c.strategic||''),criticality:String(c.criticality||''),approach:d.approach,target,
  reasons:reasons.slice(0,40),checks:crossChecks(c,d,adv.inputs).map(x=>x[1]).slice(0,40),missing:adv.confidence.missing.map(m=>m.label),claims};
 return {body,sent};
}
async function requestReview(c,d,adv){
 if(!reviewUI.consent[c.id]||reviewUI.busy[c.id])return;
 const {body,sent}=reviewPayload(c,d,adv);
 delete reviewUI.consent[c.id];reviewUI.busy[c.id]=true;delete reviewUI.error[c.id];render();
 try{
  const out=await agentFetch('/api/decide/review',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(body)});
  d.review={at:new Date().toISOString(),model:out.model||'',verdict:out.verdict,summary:out.summary||'',challenges:out.challenges||[],missing:out.missing||[],questions:out.questions||[],alternatives:out.alternatives||[],
   claimTexts:sent.map(x=>x.text),sig:reviewHash(JSON.stringify(body))};
  M.history.push({field:'decision:'+c.id,at:d.review.at,actor:actorName(),action:'ai-review',reason:'Target, '+body.reasons.length+' reasons, '+body.checks.length+' checks and '+body.claims.length+' verified claims sent to Claude ('+(out.model||'model')+')',before:null,after:{verdict:out.verdict}});
  persist();
 }catch(e){reviewUI.error[c.id]=/404|not found/i.test(e.message)?'This backend has no review service yet. Restart it from the latest code.':e.message;}
 finally{delete reviewUI.busy[c.id];render();}
}
function reviewPanel(c,d,adv){
 const {body}=reviewPayload(c,d,adv);const rv=d.review;const busy=!!reviewUI.busy[c.id];
 const stale=rv&&rv.sig!==reviewHash(JSON.stringify(body));
 const VERDICT={sound:['Sound','ok'],questionable:['Questionable','warn'],risky:['Risky','bad']};
 const claimRef=n=>(rv.claimTexts||[])[n-1]||'claim '+n;
 return h('div',{class:'advisor-review'},[
  h('div',{class:'dim-head'},[h('b',{text:'Second opinion from Claude'}),h('small',{text:'Claude challenges the target; the person deciding records the decision'})]),
  h('div',{class:'type-chips'},[agentChip('Allow this target, its reasons and checks'+(body.claims.length?' and '+body.claims.length+' verified claim'+(body.claims.length===1?'':'s'):'')+' to be sent to Claude for one review',!!reviewUI.consent[c.id],v=>{reviewUI.consent[c.id]=v;render();})]),
  h('details',{class:'compare'},[h('summary',{text:'See exactly what is sent'}),h('pre',{class:'ai-payload',text:JSON.stringify(body,null,2)}),
   h('small',{class:'muted',text:'Not sent: the rationale, names, comments, who answered, rejected or unverified claims, evidence quotes, file paths and business data.'})]),
  h('div',{class:'report-reader-actions'},[h('button',{class:'btn small primary',type:'button',disabled:!reviewUI.consent[c.id]||busy?'':null,onclick:()=>requestReview(c,d,adv),text:busy?'Claude is reviewing…':rv?'Review again':'Challenge this target'})]),
  reviewUI.error[c.id]?h('p',{class:'conflict-note',text:'No review: '+reviewUI.error[c.id]}):null,
  rv?h('div',{class:'ai-result'},[
   h('p',{},[h('span',{class:'verdict '+(VERDICT[rv.verdict]||VERDICT.questionable)[1],text:(VERDICT[rv.verdict]||VERDICT.questionable)[0]}),' ',rv.summary]),
   h('small',{class:'muted',text:'Reviewed '+new Date(rv.at).toLocaleString()+' by '+(rv.model||'Claude')+'.'}),
   stale?h('p',{class:'input-status open',text:'The target or its evidence changed since this review. Review again to check the current one.'}):null,
   rv.challenges.length?h('div',{},[h('b',{text:'Challenges'}),h('ul',{class:'plain-list review-list'},rv.challenges.map(x=>h('li',{class:'sev-'+x.severity},[h('span',{class:'sev',text:x.severity}),' '+x.text,
    (x.claims||[]).length?h('small',{class:'muted',text:' · rests on: '+x.claims.map(n=>'“'+claimRef(n).slice(0,80)+'”').join('; ')}):null])))]):null,
   rv.missing.length?h('div',{},[h('b',{text:'Missing evidence'}),h('ul',{class:'plain-list'},rv.missing.map(t=>h('li',{text:t})))]):null,
   rv.questions.length?h('div',{},[h('b',{text:'Ask the people who know'}),h('ul',{class:'plain-list'},rv.questions.map(t=>h('li',{text:t})))]):null,
   rv.alternatives.length?h('div',{},[h('b',{text:'Worth a second look'}),h('ul',{class:'plain-list'},rv.alternatives.map(a=>h('li',{},[h('b',{text:apprLabel(a.approach)}),': '+a.why])))]):null
  ]):null
 ]);
}

(function advisorStyle(){
 if(document.getElementById('advisor-style'))return;
 const css=`
.advisor{border:1px solid #d9e1f2;background:#f8faff;border-radius:10px;padding:14px 16px;margin:10px 0 14px}
.advisor-head{display:flex;justify-content:space-between;gap:16px;align-items:flex-start}
.advisor-head h4{margin:2px 0 6px;font-size:16px}
.advisor .eyebrow{color:var(--accent);font-weight:700;letter-spacing:.06em;text-transform:uppercase;font-size:11px}
.advisor-conf{min-width:130px;text-align:right;border-radius:8px;padding:6px 10px;background:#fff;border:1px solid var(--line)}
.advisor-conf small{display:block;color:var(--muted)}
.advisor-conf.high b{color:#166534}.advisor-conf.medium b{color:#8a5a00}.advisor-conf.low b{color:#9f1239}
.advisor-missing{margin:8px 0;font-size:12px;display:flex;flex-wrap:wrap;gap:4px 10px;align-items:center}
.advisor-question{background:#fff;border:1px solid #e9c46a;border-radius:8px;padding:10px 12px;margin:10px 0}
.advisor-question small{display:block;margin:2px 0 8px}
.advisor-answers{display:flex;flex-wrap:wrap;gap:6px}
.scenario-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(220px,1fr));gap:10px;margin-top:10px}
.scenario{background:#fff;border:1px solid var(--line);border-radius:10px;padding:12px;display:flex;flex-direction:column;gap:6px}
.scenario.best{border-color:var(--accent)}
.scenario.chosen{box-shadow:0 0 0 2px #16a34a33;border-color:#16a34a}
.scenario-top{display:flex;justify-content:space-between;align-items:center}
.scenario h5{margin:2px 0;font-size:14px}
.scenario-tech{font-size:12px;color:var(--ink);margin:0}
.scenario-facts{display:grid;grid-template-columns:auto 1fr;gap:2px 10px;font-size:12px;margin:4px 0}
.scenario-facts dt{color:var(--muted)}.scenario-facts dd{margin:0;font-weight:600}
.scenario-facts dd.bad{color:#9f1239}.scenario-facts dd.warn{color:#8a5a00}.scenario-facts dd.ok{color:#166534}
.scenario button{margin-top:auto;align-self:flex-start}
.advisor-current{margin-top:12px;padding-top:10px;border-top:1px solid var(--line)}
.advisor-checks{margin:6px 0;padding-left:0;list-style:none;font-size:13px}
.advisor-checks li{padding:2px 0}.advisor-checks li.block{color:#9f1239}.advisor-checks li.warn{color:#8a5a00}.advisor-checks li.info{color:var(--muted)}
.advisor-actions{display:flex;gap:8px;margin-top:6px}
.advisor-detail{margin:6px 0 10px}
.advisor-detail>summary{cursor:pointer;font-weight:600;color:var(--accent);padding:6px 0}
.advisor-robust{margin:8px 0;font-size:13px}.advisor-robust.ok b{color:#166534}.advisor-robust.warn b{color:#8a5a00}.advisor-robust ul{margin:4px 0 0}
.advisor-review{margin-top:12px;padding-top:10px;border-top:1px dashed var(--line)}
.verdict{display:inline-block;border-radius:999px;padding:2px 10px;font-weight:700;font-size:12px}.verdict.ok{background:#e8f6ee;color:#166534}.verdict.warn{background:#fff8e6;color:#8a5a00}.verdict.bad{background:#fdf0f0;color:#9f1239}
.review-list .sev{display:inline-block;min-width:54px;font-size:11px;font-weight:700;text-transform:uppercase}.review-list .sev-high .sev{color:#9f1239}.review-list .sev-medium .sev{color:#8a5a00}.review-list .sev-low .sev{color:var(--muted)}
.portfolio-head{display:flex;justify-content:space-between;align-items:baseline}
.portfolio-table td.conf-high{color:#166534}.portfolio-table td.conf-medium{color:#8a5a00}.portfolio-table td.conf-low{color:#9f1239}
.portfolio-facts{display:grid;grid-template-columns:repeat(auto-fit,minmax(200px,1fr));gap:12px;margin:12px 0}.portfolio-facts p{margin:2px 0}
.warn-text{color:#8a5a00}.cost-table dd small{margin-left:4px}
@media (max-width:640px){.advisor-head{flex-direction:column}.advisor-conf{text-align:left}}`;
 document.head.append(Object.assign(document.createElement('style'),{id:'advisor-style',textContent:css}));
})();
