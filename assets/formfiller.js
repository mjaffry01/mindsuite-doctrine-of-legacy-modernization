/* Formfiller proposals are stored separately from accepted product answers. */
const formfillUI={busy:false,stop:false,afterEdit:false,message:'',revision:0};
const deploymentKeys=new Set(['where','tenancy','channels','regions']);
function briefKeys(){
 return [...BRIEF_TEXT,...BRIEF_NUM].map(([key])=>key).concat([...deploymentKeys]);
}
function briefOwner(key){
 const p=productBrief();
 return deploymentKeys.has(key)?p.deploy:p;
}
function briefAnswer(key){
 const value=briefOwner(key)[key];
 return Array.isArray(value)?value.join(','):String(value??'').trim();
}
function briefDrafts(){return M.formfiller||(M.formfiller={});}
function visibleBriefValue(key){
 return briefAnswer(key)||briefDrafts()[key]?.value||'';
}
function briefFillRequest(){
 const answers={},missing=[];
 const previous=briefProposalKeys().map(field=>({field,value:briefDrafts()[field].value}));
 for(const key of briefKeys()){
  const value=briefAnswer(key);
  if(value)answers[key]=value;
  else if(!briefDrafts()[key])missing.push(key);
 }
 return {mode:modeOf(),project:M.project.trim(),deadline:brief().deadline||'',answers,previous,missing,next:true};
}
function briefProposalKeys(){
 return Object.keys(briefDrafts()).filter(key=>briefKeys().includes(key)&&!briefAnswer(key));
}
function recordFormfill(action,details){
 M.history.push({field:'product:formfiller',at:new Date().toISOString(),actor:actorName(),action,after:details});
 persist();
}
function refreshFormfillPanel(){
 const panel=document.getElementById('formfiller-panel');
 if(panel)panel.replaceWith(formfillerPanel());
}
function briefFieldName(key){
 return [...BRIEF_TEXT,...BRIEF_NUM].find(([id])=>id===key)?.[1].replace(/\s*\*$/,'')||
  ({project:'Project name',where:'Deployment environment',tenancy:'Tenancy',channels:'Delivery channels',regions:'Data residency'})[key]||key;
}
function redrawBrief(){
 if(!document.getElementById('product-brief'))return;
 const active=document.activeElement,key=active?.dataset.briefKey;
 const caret=key?[active.selectionStart,active.selectionEnd]:null;
 render();
 if(key){
  const field=document.querySelector('[data-brief-key="'+key+'"]');
  field?.focus({preventScroll:true});
  if(field&&caret[0]!=null)field.setSelectionRange(caret[0],caret[1]);
 }
}
async function runFormfiller(all=false){
 if(formfillUI.busy)return;
 let input=briefFillRequest();
 if(!input.project&&!Object.keys(input.answers).length){
  formfillUI.message='Describe the product or give it a name first.';
  refreshFormfillPanel();return;
 }
 if(!input.missing.length){
  formfillUI.message='Every field has an answer or a suggestion to review.';refreshFormfillPanel();return;
 }
 formfillUI.afterEdit=false;
 const started=Date.now(),revision=formfillUI.revision;
 formfillUI.busy=true;formfillUI.stop=false;formfillUI.message='Predicting the next answer…';refreshFormfillPanel();
 recordFormfill('agent-start',{agent:'Formfiller',requested:input.missing});
 let count=0;
 try{
  do{
   const stepStarted=Date.now();
   const result=await agentFetch('/api/brief/formfill',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(input)});
   if(revision!==formfillUI.revision||JSON.stringify(input)!==JSON.stringify(briefFillRequest())){
    formfillUI.message='The brief changed. Predict the next answer using your latest edits.';
    recordFormfill('agent-stale',{elapsedMs:Date.now()-started});return;
   }
   const item=result.suggestions.find(s=>input.missing.includes(s.field)&&s.value);
   if(!item)break;
   const inputs=[...Object.keys(input.answers),...input.previous.map(p=>p.field)];
   const draft={...item,model:result.model,inputs};
   briefDrafts()[item.field]=draft;
   count++;
   recordFormfill('agent-prediction',{agent:'Formfiller',field:item.field,draft,elapsedMs:Date.now()-stepStarted});
   input=briefFillRequest();
   formfillUI.message='Suggested '+briefFieldName(item.field)+'.';
   redrawBrief();
  }while(all&&!formfillUI.stop&&input.missing.length);
  formfillUI.message=count?'Review or edit the suggested answers.':'Add more detail to the brief to predict another answer.';
  recordFormfill('agent-complete',{agent:'Formfiller',count,elapsedMs:Date.now()-started});
 }catch(error){
  formfillUI.message=/404|not found/i.test(error.message)?'This backend needs the Formfiller update. Check the project’s agent connection.':'Formfiller could not finish. '+error.message;
  recordFormfill('agent-error',{elapsedMs:Date.now()-started});
 }finally{formfillUI.busy=false;refreshFormfillPanel();}
}
function acceptBriefDrafts(keys){
 formfillUI.revision++;
 for(const key of keys){
  const draft=briefDrafts()[key];
  if(!draft||briefAnswer(key))continue;
  const value=key==='channels'?draft.value.split(',').map(v=>v.trim()):draft.value;
  briefOwner(key)[key]=value;
  delete briefDrafts()[key];
  recordFormfill('accept-suggestion',{key,value,model:draft.model});
 }
 formfillUI.message='';persist();render();
}
function discardDependentDrafts(key){
 const removed=[];
 // Each new draft remembers every field supplied to its prediction request.
 // Earlier saved drafts have no lineage, so regenerate those after a correction.
 for(const field of briefProposalKeys()){
  if(field!==key&&(!briefDrafts()[field].inputs||briefDrafts()[field].inputs.includes(key))){
   delete briefDrafts()[field];removed.push(field);
  }
 }
 return removed;
}
function dismissBriefDraft(key){
 formfillUI.revision++;
 delete briefDrafts()[key];
 const removed=discardDependentDrafts(key);
 recordFormfill('dismiss-suggestion',{key,removed});formfillUI.message=removed.length?'Related suggestions were cleared. Predict again to continue.':'';render();
}
function formfillerPanel(){
 const pending=briefProposalKeys(),remaining=briefFillRequest().missing.length;
 return h('section',{id:'formfiller-panel',class:'formfiller-panel','aria-label':'Formfiller agent'},[
  h('div',{},[h('h3',{text:'Formfiller agent'}),
   h('p',{text:'Each prediction uses your project name, target date, brief answers and earlier suggestions. You can edit any answer. If Claude returns no usable suggestion, Formfiller tries once more.'}),
   h('label',{class:'formfiller-auto'},[
    h('input',{type:'checkbox',checked:formfillUI.afterEdit?'':null,disabled:formfillUI.busy?'':null,onchange:e=>{formfillUI.afterEdit=e.target.checked;}}),
    'Predict one answer after my next edit'
   ]),
   h('small',{role:'status',text:formfillUI.message|| (pending.length?'Review each AI suggestion or use them together.':'')})]),
  h('div',{class:'formfiller-actions'},[
   h('button',{type:'button',class:'btn small primary',disabled:formfillUI.busy?'':null,onclick:()=>runFormfiller(),text:formfillUI.busy?'Formfiller is working…':'Predict next answer'}),
   formfillUI.busy?button('Stop after this answer',()=>{formfillUI.stop=true;formfillUI.message='Finishing the current prediction.';refreshFormfillPanel();},'btn small'):
    remaining>1?h('div',{},[button('Draft remaining answers in sequence',()=>runFormfiller(true),'btn small'),
     h('small',{text:remaining+' fields remaining · up to '+(2*remaining)+' Claude requests including retries'})]):null,
   pending.length?button('Use all '+pending.length+' suggestions',()=>acceptBriefDrafts(pending),'btn small'):null
  ])
 ]);
}
function briefField(key,control){
 control.dataset.briefField=key;
 const draft=briefDrafts()[key];
 if(!draft||briefAnswer(key))return control;
 return h('div',{class:'brief-proposal','data-proposal':key},[control,
  h('div',{class:'brief-proposal-note'},[
   h('strong',{text:'AI suggestion'}),h('span',{text:draft.reason}),
   h('small',{text:(draft.quoteFrom?(briefDrafts()[draft.quoteFrom]&&!briefAnswer(draft.quoteFrom)?'From an earlier AI suggestion · ':'From ')+briefFieldName(draft.quoteFrom):'From your brief')+' · “'+draft.quote+'”'}),
   h('div',{class:'formfiller-actions'},[button('Use answer',()=>acceptBriefDrafts([key]),'btn small'),button('Dismiss',()=>dismissBriefDraft(key),'linkbtn')])
  ])
 ]);
}
function writeBriefAnswer(key,value){
 const before=briefAnswer(key);
 formfillUI.revision++;
 const pending=briefDrafts()[key];
 briefOwner(key)[key]=value;
 const removed=before!==briefAnswer(key)||pending?discardDependentDrafts(key):[];
 if(removed.length){
  const fresh=productBriefSection();
  for(const field of removed){
   const wrapper=document.querySelector('[data-proposal="'+field+'"]');
   const replacement=fresh.querySelector('[data-brief-field="'+field+'"]');
   if(wrapper&&replacement)wrapper.replaceWith(replacement);else wrapper?.remove();
  }
  recordFormfill('invalidate-suggestions',{changed:key,removed});
  formfillUI.message='Your edit cleared related suggestions. Predict the next answer when ready.';
  refreshFormfillPanel();
 }
 if(pending){
  delete briefDrafts()[key];
  document.querySelector('[data-proposal="'+key+'"] .brief-proposal-note')?.remove();
  document.querySelector('[data-proposal="'+key+'"]')?.classList.remove('brief-proposal');
  if(!removed.length)formfillUI.message='';
  recordFormfill('edit-suggestion',{key,proposed:pending.value,model:pending.model});
  refreshFormfillPanel();
 }else if(!removed.length&&Boolean(before)!==Boolean(briefAnswer(key)))refreshFormfillPanel();
 persist();
}
