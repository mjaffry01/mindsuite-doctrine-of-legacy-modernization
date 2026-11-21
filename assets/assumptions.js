/* MindSuite web app · assumptions register
   Collects every number and choice the plan relies on that nobody has stated as a fact: estimated peak and
   data volume and the assumptions behind them, the requests-per-vCPU figure, availability and growth when not
   given, the fixed sizing rules (headroom, storage room, hot cache share, backup retention), where it runs and
   the hosting model when not chosen, regulations not recorded, and routes still taken from the advisor's best
   fit because the decision is open.
   A named person either Confirms an item (it is a fact) or Accepts it as a risk (a known guess). Each record
   keeps the value it was confirmed at; when the value later changes the item reopens. Launch approval /
   Sign-off is blocked while any item is open (check in signoffPage, index.html).
   Stored in M.assumptions {id: {status:'confirmed'|'accepted', by, at, value, note}}; assumptionsRegister()
   returns the list for the diagrams. Page: assumptionsPage() (screen 'assumptions'). */

const ASM_AREAS=['Load','Data','Availability','Growth','Sizing rules','Cost','Deployment','Compliance','Decisions'];
function asmStore(){return M.assumptions||(M.assumptions={});}
const asmFmt=n=>typeof loadFmt==='function'?loadFmt(n):String(n);

function assumptionsRegister(){
 ensureDecide();
 const out=[];const add=(id,area,text,value,source,drives,go,high)=>out.push({id,area,text,value:String(value),source,drives,go,high:!!high});
 const product=isNewProduct();
 const P=M.product||{};
 const C=decCtxs();
 const sys={id:'system',name:M.project||'',description:''};
 let I=null;try{I=techPlanInputs(product?sys:(C[0]||sys));}catch(e){}
 const brief=product?['upload','Product brief']:['decide','Capacity profile'];

 /* Load */
 if(I&&I.peak){
  const estimated=product?(P.peakBasis==='estimated-from-users'&&+((P.peakCalc||{}).peak)===num(P.peak))||!num(P.peak):I.pSrc!=='capacity profile';
  if(estimated){
   add('load:peak','Load','Peak load is an estimate, not a measurement: '+asmFmt(I.peak)+' requests/s from '+asmFmt(I.users||0)+' users',asmFmt(I.peak),I.pSrc||'calculated','Compute size, database size, cost',brief,true);
   if(product&&typeof loadAssumptions==='function'){
    // The values the saved estimate was made with; the calculator's current settings only when no estimate is saved.
    const saved=P.peakBasis==='estimated-from-users'&&P.peakCalc&&P.peakCalc.assumptions;
    const a=saved?P.peakCalc.assumptions:loadAssumptions(P);
    for(const [k,label,unit] of LOAD_FIELDS){if(k==='perVcpu'||a[k]==null)continue;if(!saved&&a._src[k]==='edited')continue;
     add('load:'+k,'Load',label+': '+a[k]+' '+unit,a[k],saved?(P.peakCalc.basis||'calculator, typical profile'):a._src[k],'Peak load',brief);}
   }
  }
 }else add('load:none','Load','No users or peak load given; compute cannot be sized','not given','missing','Compute size, cost',brief,true);
 const per=I?I.perVcpu:25;
 const perEdited=product&&typeof loadAssumptions==='function'&&loadAssumptions(P)._src.perVcpu==='edited';
 add('load:perVcpu','Load','One vCPU handles '+per+' requests/s'+(perEdited?' (you set this)':''),per,perEdited?'you set this':'planning guess','Number of vCPU and servers',brief,true);

 /* Data */
 if(I&&I.dataGb){
  const estimated=product?(P.dataBasis==='estimated-from-users'&&+((P.dataCalc||{}).gb)===num(P.dataGb))||!num(P.dataGb):I.dSrc!=='capacity profile';
  if(estimated){
   add('data:volume','Data','Data volume after 3 years is an estimate: '+asmFmt(I.dataGb)+' GB',asmFmt(I.dataGb),I.dSrc||'calculated','Database storage, backups, cost',brief,true);
   if(product&&typeof dataAssumptions==='function'){
    const saved=P.dataBasis==='estimated-from-users'&&P.dataCalc&&P.dataCalc.assumptions;
    const a=saved?P.dataCalc.assumptions:dataAssumptions(P);
    for(const [k,label,unit] of DATA_FIELDS){if(a[k]==null)continue;if(!saved&&a._src[k]==='edited')continue;if(k==='fileMb'&&!a.files)continue;
     add('data:'+k,'Data',label+': '+a[k]+' '+unit,a[k],saved?(P.dataCalc.basis||'calculator'):a._src[k],'Data volume',brief);}
   }
  }
  if(I.f&&I.f.video&&I.dataGb<1000)add('data:video','Data','Video is stored in only '+asmFmt(I.dataGb)+' GB (an hour of HD is about 2–3 GB per quality level)',asmFmt(I.dataGb),'check','Object storage, CDN cost',brief,true);
 }else add('data:none','Data','No data volume given; storage cannot be sized','not given','missing','Database and storage size',brief,true);

 /* Availability and growth */
 if(I&&I.aSrc.startsWith('assumed'))add('availability','Availability','Availability target not given; '+I.avail+'% assumed',I.avail,'assumed','Zones, failover, recovery time, cost',brief,true);
 else if(I)add('availability','Availability','Availability target '+I.avail+'% ('+I.aSrc+'): recover within '+(I.avail>=99.95?'1 hour, lose at most 5 minutes':I.avail>=99.5?'4 hours, lose at most 15 minutes':'1 day, lose at most 24 hours'),I.avail,'derived recovery targets','Recovery time and data-loss limits',brief);
 const g=num(constraints().growth);
 if(!g)add('growth','Growth','No yearly growth set; sizing assumes the same load for 3 years','0%','assumed','Sizing for year 3',['rules','Rules & people'],true);

 /* Sizing rules used by the technical plan */
 add('rule:headroom','Sizing rules','30% headroom on the peak, sized for 3 years','30%','rule','Compute size',null);
 add('rule:dbroom','Sizing rules','Database storage gets 50% room on top of the data volume','50%','rule','Storage size',null);
 add('rule:cache','Sizing rules','About 2% of the data is hot and kept in the cache','2%','rule','Cache size',null);
 add('rule:backup','Sizing rules','Point-in-time backups kept 35 days','35 days','rule','Backup storage, recovery',null);

 /* Deployment */
 const D=P.deploy||{};
 if(!D.where||D.where==='undecided')add('deploy:where','Deployment','Where it runs is not decided; the plan uses each scenario\'s cloud choice','not decided','missing','Every hosting and service choice',brief,true);
 if(!D.hosting&&typeof hostingSuggest==='function'){
  const s=hostingSuggest(P)[0];const m=s&&HOSTING_MODELS.find(x=>x.id===s.id);
  add('deploy:hosting','Deployment','Hosting model not chosen'+(m?'; the brief points to '+m.name:''),m?m.name:'none','suggested','Compute, operations, cost',brief);
 }

 /* Compliance */
 const reg=(M.scope&&M.scope.regulation)||{};
 if(!(reg.list||[]).length&&!String(reg.other||'').trim())add('compliance','Compliance','No regulations recorded; the plan assumes none apply','none recorded','missing','Security, data location, audit',brief,true);

 /* Platform decision */
 if(typeof platformEffective==='function'){const p=platformEffective();if(!p.decided){const lab=(d,id)=>id?((optOf(d,id)||[,id])[1]):'';
  add('platform','Decisions','Platform not decided; the plan uses the suggestion: '+[p.tier+' tier',lab('cloud',p.cloud),lab('architecture',p.architecture),lab('database',p.database)].filter(Boolean).join(', '),p.tier+'/'+p.cloud+'/'+p.architecture+'/'+p.database,'suggested','Every layer, the cost and the HLD',['decide','Design options, step 1'],true);}}
 /* Decisions still open */
 if(typeof systemBlueprint==='function'&&C.length){
  let B=null;try{B=systemBlueprint(C,'current');}catch(e){}
  if(B)for(const c of B.capabilities)if(c.picked!=='decided')
   add('route:'+c.id,'Decisions',(product?'Capability':'Context')+' "'+c.name+'": route assumed from the advisor\'s best fit, '+c.route,c.route,'best fit','System blueprint, release plan',['decide',product?'Design options':'Target decisions']);
 }
 if(typeof costAssumptions==='function')costAssumptions(add);
 // Attach the stored status; a confirmed value that has since changed reopens the item.
 const S=asmStore();
 for(const a of out){const r=S[a.id];
  a.state=!r?'open':r.value!==a.value?'changed':r.status;a.record=r||null;}
 return out;
}
function assumptionsOpen(){try{return assumptionsRegister().filter(a=>a.state==='open'||a.state==='changed');}catch(e){return [];}}

function asmSet(a,status,by,note){
 const S=asmStore(),before=S[a.id]?S[a.id].status:'open';
 if(!status)delete S[a.id];
 else S[a.id]={status,by,at:new Date().toISOString(),value:a.value,note:note||''};
 logInput('assumption:'+a.id,before,status||'open');persist();render();
}

function assumptionsPage(){
 const list=assumptionsRegister(),product=isNewProduct();
 const people=M.people||[];
 const counts={open:0,changed:0,confirmed:0,accepted:0};for(const a of list)counts[a.state]++;
 const label={open:'Open',changed:'Changed since confirmed',confirmed:'Confirmed',accepted:'Accepted as risk'};
 const who=id=>{const p=people.find(x=>x.id===id);return p?p.name:id||'someone';};
 const row=a=>{
  const sel=h('select',{class:'field compact','aria-label':'Who confirms'},[h('option',{value:'',text:'Who…'}),...people.map(p=>h('option',{value:p.id,text:p.name+' · '+p.role}))]);
  sel.value=(a.record&&a.record.by)||M.actor||'';
  const act=status=>()=>{if(!sel.value){sel.focus();notice('Choose who '+(status==='confirmed'?'confirms':'accepts')+' this.');return;}asmSet(a,status,sel.value);};
  return h('tr',{class:'asm-'+a.state},[
   h('td',{},[h('div',{text:a.text}),h('small',{class:'tp-why',text:'Drives: '+a.drives+' · source: '+a.source})]),
   h('td',{},[h('span',{class:'asm-state '+a.state,text:label[a.state]}),
    a.record&&a.state!=='open'?h('small',{class:'tp-why',text:who(a.record.by)+', '+String(a.record.at).slice(0,10)+(a.state==='changed'?' (was '+a.record.value+')':'')}):null]),
   h('td',{class:'asm-act-cell'},h('div',{class:'asm-actions'},a.state==='confirmed'||a.state==='accepted'?[button('Reopen',()=>asmSet(a,null),'linkbtn')]
    :[people.length?sel:button('Add people',()=>go('rules'),'btn small'),people.length?button('Confirm',act('confirmed'),'btn small primary'):null,people.length?button('Accept as risk',act('accepted'),'btn small'):null,
      a.go?button('Change it',()=>go(a.go[0]),'linkbtn'):null]))]);
 };
 return h('div',{},[
  pageHeader('✓ / ASSUMPTIONS','Confirm what the plan assumes','Every estimate, default and open choice the design relies on. Confirm it as a fact, accept it as a known risk, or change it at the source. '+(product?'Launch approval':'Sign-off')+' waits until none are open.',[
   button('← '+(product?'Design options':'Target decisions'),()=>go('decide')),button((product?'Release plan':'Migration roadmap')+' →',()=>go('roadmap'),'btn primary')]),
  h('div',{class:'metrics'},[['Open',counts.open+counts.changed],['Confirmed',counts.confirmed],['Accepted as risk',counts.accepted],['Total',list.length]].map(([l,v])=>h('div',{class:'metric'},[h('b',{text:String(v)}),h('span',{text:l})]))),
  !people.length?h('div',{class:'summary-callout'},[h('b',{text:'Add the people who confirm'}),h('p',{text:'Each assumption is confirmed or accepted by a named person.'}),button('Add people',()=>go('rules'))]):null,
  ...ASM_AREAS.map(area=>{const items=list.filter(a=>a.area===area);if(!items.length)return null;
   return h('section',{class:'summary-section asm-area'},[h('h2',{text:area}),
    h('table',{class:'tp-table asm-table'},[h('thead',{},h('tr',{},[h('th',{text:'Assumption'}),h('th',{text:'Status'}),h('th',{text:''})])),
     h('tbody',{},items.sort((x,y)=>(y.high-x.high)).map(row))])]);})]);
}

(function asmStyle(){
 if(document.getElementById('asm-style'))return;
 const css=`.asm-table{min-width:0;table-layout:auto}.asm-table td{vertical-align:middle}.asm-table td:nth-child(2){width:190px}.asm-table td.asm-act-cell{width:430px}
.asm-actions{display:flex;flex-wrap:wrap;gap:6px;align-items:center}.asm-actions .field.compact{width:160px}
.asm-state{font-size:12px;padding:2px 8px;border-radius:10px;background:#fff4e0;color:#7a4b00;white-space:nowrap}
.asm-state.changed{background:#fde8e8;color:#9b1c1c}.asm-state.confirmed{background:#e6f6ec;color:#1d6b3a}.asm-state.accepted{background:#eef0ff;color:#3f3a9c}`;
 document.head.append(Object.assign(document.createElement('style'),{id:'asm-style',textContent:css}));
})();
