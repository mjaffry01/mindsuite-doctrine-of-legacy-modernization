/* Project-specific steps. Shared design and planning screens use the active workflow. */
function hldStatus(){if(typeof hldModel!=='function')return '';try{const m=hldModel();return m.meta.draft?'Draft':'Ready for review';}catch(e){return 'Draft';}}
function assumptionStatus(){if(typeof assumptionsRegister!=='function')return '';try{const l=assumptionsRegister(),o=l.filter(a=>a.state==='open'||a.state==='changed').length;return o?o+' of '+l.length+' to confirm':'All '+l.length+' confirmed or accepted';}catch(e){return '';}}
function workflowGroups(){
 const people=(M.people||[]).length+' people';
 const approvals=M.plan.signoff;
 const pas=approvals&&typeof planApprovalState==='function'?planApprovalState().state:(approvals?(approvals.fingerprint===planFingerprint()?'current':'changed'):'none');const approvalStatus=pas==='current'?'Approved':pas==='legacy'?'Approved (dates not covered)':pas==='none'?'Not approved':'Changed since approval';
 const contexts=decCtxs();
 const decisions=contexts.filter(c=>decisionFor(c).status==='decided').length+' of '+contexts.length+' decided';
 if(isNewProduct())return [
  {title:'Define',steps:[
   ['upload','01','Product brief',briefProgress()],
   ['release-scope','02','First-release scope','Scope and capabilities']
  ]},
  {title:'Design',steps:[
   ['rules','03','Rules & people',people],
   ['decide','04','Design options',decisions],
   ['assumptions','✓','Assumptions',assumptionStatus()],
   ['hld','✓','Architecture (HLD)',hldStatus()],
   ['validation','✓','Architecture validation',typeof validationStatus==='function'?validationStatus():''],
   ['datapolicy','✓','Data, cookie and session policy',typeof dataPolicyStatus==='function'?dataPolicyStatus():'']
  ]},
  {title:'Deliver',steps:[
   ['roadmap','05','Release plan','Pilot, first release and later releases'],
   ['signoff','06','Launch approval',approvalStatus]
  ]},
  {title:'About',steps:[['agents','ⓘ','AI agent teams','Who does what, and how they talk']]}
 ];
 const s=stats(),iv=invStats(),verification=verifyStats();
 return [
  {title:'Discover',steps:[
   ['upload','01','Interview & evidence',evidenceProgress()],
   ['analysis','02','Agent reports','Workspace runs and findings'],
   ['validate','03','Validate findings',(iv.open+needsReview().length)+' to check · '+iv.total+' records'],
   ['summary','04','Discovery summary',s.confirmed+' reviewed fields'],
   ['verify','✓','Verify with people',verification.total?verification.confirmed+verification.corrected+' of '+verification.total+' claims verified':'Confirm findings with system experts']
  ]},
  {title:'Decide',steps:[
   ['rules','05','Rules & people',people],
   ['decide','06','Target decisions',decisions],
   ['assumptions','✓','Assumptions',assumptionStatus()],
   ['hld','✓','Architecture (HLD)',hldStatus()],
   ['validation','✓','Architecture validation',typeof validationStatus==='function'?validationStatus():''],
   ['datapolicy','✓','Data, cookie and session policy',typeof dataPolicyStatus==='function'?dataPolicyStatus():'']
  ]},
  {title:'Migrate',steps:[
   ['roadmap','07','Migration roadmap',contexts.filter(c=>planOf(c).wave).length+' of '+contexts.length+' planned'],
   ['signoff','08','Sign-off & export',approvalStatus]
  ]},
  {title:'About',steps:[['agents','ⓘ','AI agent teams','Who does what, and how they talk']]}
 ];
}

function workflowRoute(next){
 if(isNewProduct()){
  if(['upload','evidence','analysis'].includes(next))return 'upload';
  if(['inventory','review','gaps','validate','summary'].includes(next))return 'release-scope';
  return next;
 }
 return next==='release-scope'?'upload':next;
}

function workflowStep(){
 if(screen==='evidence')return 'upload';
 return isNewProduct()&&screen==='verify'?'decide':workflowRoute(screen);
}

function productDesignConnection(){
 const config=agentCfg();
 const open=document.getElementById('product-design-connection')?.open;
 return h('details',{id:'product-design-connection',class:'discovery-disclosure summary-section',open:open?'':null},[
  h('summary',{text:'Design agent connection'}),
  h('div',{class:'agent-conn-row'},[
   inputControl('Agent backend address',config.url,v=>{config.url=v;persist();},'url','http://127.0.0.1:8090'),
   button('Check connection',agentCheck,'btn small')
  ])
 ]);
}

const RELEASE_SCOPE=[['first','First release'],['later','Later release']];
const PRODUCT_CAPABILITY={
 key:'contexts',
 one:'capability',
 hint:'Describe an outcome or task users must be able to complete.',
 fields:[
  {k:'name',label:'Capability name',ph:'e.g. Receive stock'},
  {k:'description',label:'What users need to do',type:'textarea',ph:'e.g. Receive, count and record a delivery'},
  {k:'releaseScope',label:'Release scope',type:'select',opts:()=>RELEASE_SCOPE}
 ]
};

function productStartPage(){
 const timeline=brief();
 const name=h('input',{id:'product-project-name',class:'field',required:'',maxlength:'180',value:M.project,placeholder:'e.g. Stock tracking portal'});
 const saveName=()=>{
  const next=name.value.trim();
  if(next===M.project)return;
  const before=M.project;
  M.project=next;
  markIntakeEdit('project');
  logInput('project:name',before,next);
  persist();
  document.getElementById('projectName').textContent=next||'New assessment';
 };
 name.addEventListener('change',saveName);
 const form=h('form',{class:'project-interview'},[
  h('div',{class:'scope-panel rec-step scope-grid product-basics'},[
   h('label',{class:'context-control',for:name.id},[h('span',{text:'Project name *'}),name]),
   inputControl('Target launch date (optional)',timeline.deadline,v=>{
    const before=timeline.deadline;
    timeline.deadline=v;
    logInput('brief:deadline',before,v);
    persist();
   },'date')
  ]),
  productBriefSection(),
  regulationBlock(),
  h('div',{class:'page-actions'},[h('button',{type:'submit',class:'btn primary',text:'Continue to first-release scope →'})])
 ]);
 form.addEventListener('submit',event=>{
  event.preventDefault();
  name.value=name.value.trim();
  if(!name.reportValidity())return;
  saveName();
  go('release-scope');
 });
 return h('div',{class:'product-workflow'},[
  pageHeader('01 / PRODUCT BRIEF','Define the product','Describe the problem, who will use the product, and what a successful first release must deliver.'),
  modeSwitcher(),
  form
 ]);
}

function releaseScopePage(){
 const capabilities=M.inventory.contexts;
 const p=productBrief();
 const add=()=>recordDialog(PRODUCT_CAPABILITY,null,{releaseScope:'first'});
 return h('div',{class:'product-workflow'},[
  pageHeader('02 / FIRST-RELEASE SCOPE','Decide what ships first','List the capabilities you need. Each feeds its own design decision and release plan.',[
   button('← Product brief',()=>go('upload')),
   button('Rules & people →',()=>go('rules'),'btn primary')
  ]),
  h('section',{class:'summary-section'},[
   h('h2',{text:'Outcomes for the first release'}),
   h('dl',{class:'inv-meta'},[
    h('dt',{text:'Scope from the brief'}),h('dd',{text:p.mvp||'Scope has yet to be described.'}),
    h('dt',{text:'Success measures'}),h('dd',{text:p.success||'Success measures have yet to be recorded.'})
   ])
  ]),
  h('section',{class:'summary-section'},[
   h('div',{class:'discovery-section-head'},[
    h('h2',{text:'Capabilities'}),
    button('Add capability',add,'btn small primary')
   ]),
   typeof capabilityAgentPanel==='function'?capabilityAgentPanel():null,
   capabilities.length?h('ul',{class:'scan-list capability-list'},capabilities.map(c=>h('li',{},[
    h('div',{},[
     h('b',{text:c.name}),
     h('small',{text:(c.releaseScope==='later'?'Later release':'First release')+(c.description?' · '+c.description:'')})
    ]),
    h('div',{class:'finding-actions'},[
     button('Edit',()=>recordDialog(PRODUCT_CAPABILITY,c),'btn small'),
     button('Remove',()=>removeRecord(PRODUCT_CAPABILITY,c),'linkbtn')
    ])
   ]))):h('p',{class:'muted',text:'Add individual capabilities or plan one design for the whole product.'})
  ])
 ]);
}

const RELEASES=[
 [1,'Pilot','Release to a small group to test the product and confirm the user journeys.'],
 [2,'First release','Deliver the agreed launch scope and measure the first outcomes.'],
 [3,'Later releases','Sequence follow-up work after launch.']
];
const RELEASE_PATTERNS=[['phased','Gradual rollout'],['pilot','Pilot group'],['launch','Launch to all intended users']];
const planStages=()=>isNewProduct()?RELEASES:WAVES;
const capabilityReleases=c=>RELEASES.filter(([n])=>c.releaseScope==='later'?n===3:n<=2);
const planStageLabel=n=>isNewProduct()?(RELEASES.find(stage=>stage[0]===n)?.[1]||'Not scheduled'):'Wave '+n;
function productReleaseFor(c,saved){
 if(saved===0||capabilityReleases(c).some(([n])=>n===saved))return saved;
 return c.releaseScope==='later'?3:2;
}
const launchContexts=()=>decCtxs().filter(c=>c.releaseScope!=='later');

function productLaunchChecks(contexts){
 const p=M.product||{};
 const hasBrief=[M.project,p.problem,p.users,p.mvp].every(v=>String(v||'').trim());
 const hasSuccess=!!String(p.success||'').trim();
 const scheduled=contexts.length>0&&contexts.every(c=>[1,2].includes(planOf(c).wave));
 const pilot=M.plan.waves[1]||{}, first=M.plan.waves[2]||{};
 const hasLaunchDate=!!first.end;
 const needsPilotDate=contexts.some(c=>planOf(c).wave===1);
 const pilotReady=!needsPilotDate||!!(pilot.end&&first.end&&pilot.end<=first.end);
 const validDates=[pilot,first].every(stage=>!stage.start||!stage.end||stage.start<=stage.end);
 return [
  [hasBrief,hasBrief?'Problem, users and launch scope recorded':'Complete the product brief and first-release scope',true],
  [hasSuccess,hasSuccess?'Success measures recorded':'Record how launch success will be measured',true],
  [scheduled,scheduled?'Every first-release capability is scheduled':'Schedule the first-release capabilities in the pilot or first release',true],
  [hasLaunchDate,hasLaunchDate?'First-release date recorded':'Set the end month for the first release',true],
  [pilotReady,pilotReady?'Any scheduled pilot ends by launch':'Set a pilot end month on or before the first release',true],
  [validDates,validDates?'Release dates are in order':'Set release end months on or after their start months',true]
 ];
}

function productLaunchSummary(){
 const p=M.product||{};
 return h('section',{class:'summary-section'},[
  h('h2',{text:'Launch scope and outcomes'}),
  h('dl',{class:'inv-meta'},[
   h('dt',{text:'First release'}),h('dd',{text:p.mvp||'Not recorded'}),
   h('dt',{text:'Success measures'}),h('dd',{text:p.success||'Not recorded'}),
   h('dt',{text:'Planned launch'}),h('dd',{text:M.plan.waves[2]?.end||'Not scheduled'})
  ])
 ]);
}

function exportProductPlan(){
 return {
  schemaVersion:6,workspaceId,mode:modeOf(),project:M.project,mockup:true,analysisMode:'product-brief',
  product:clone(M.product||{}),targetLaunchDate:brief().deadline||'',regulation:clone(M.scope.regulation||null),
  capabilities:decCtxs().map(c=>{
   const p=planOf(c);
   return {...clone(c),release:planStageLabel(p.wave),rollout:p.pattern,size:p.size,notes:p.it.notes||''};
  }),
  people:clone(M.people),deciding:clone(M.decide),plan:clone(M.plan),
  reviewHistory:clone(M.history),exportedAt:new Date().toISOString()
 };
}
