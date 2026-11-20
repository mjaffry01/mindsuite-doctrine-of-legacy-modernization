/* MindSuite web app · 03 Validate
   One page for checking what the agents found. It replaces the separate System inventory,
   Review findings and Resolve gaps pages, which are now its three tabs. The old page ids
   ('inventory', 'review', 'gaps') still work in go(): they open Validate on that tab.
   Loaded by index.html before the main script; uses its helpers (h, button, pageHeader, stats,
   invStats, needsReview, inventoryPage, reviewPage, gapsPage, render) at call time. */
const VALIDATE_TABS=['inventory','review','gaps'];
let validateTab='inventory';

/* Body of one of the old pages without its own title block: keeps its description and its
   action buttons, drops the buttons that only jumped between the three old pages or repeat
   the page's own "Add evidence". */
function validateTabBody(page){
 const head=page.querySelector('.page-heading');
 if(head){
  const desc=head.querySelector('p')?.textContent||'';
  const actions=[...(head.querySelector('.page-actions')?.children||[])]
   .filter(b=>!/^(Review findings|Resolve gaps|Open inventory|Add evidence|Add artifacts|Add supporting artifacts)/.test(b.textContent.trim()));
  head.replaceWith(h('div',{class:'validate-tab-head'},[h('p',{class:'muted',text:desc}),h('div',{class:'page-actions'},actions)]));
 }
 // The page's own counters at the top replace each tab's counter row.
 for(const row of page.querySelectorAll(':scope > .metrics'))row.remove();
 return page;
}

function validatePage(){
 ensureDiscovery();
 const s=stats(),iv=invStats(),open=needsReview();
 const tabs=[
  ['inventory','System inventory',iv.open+' to confirm',inventoryPage],
  ['review','Field findings',s.populated+' of '+FIELDS.length+' populated',reviewPage],
  ['gaps','Gaps',open.length+' open',gapsPage]];
 const cur=tabs.find(t=>t[0]===validateTab)||tabs[0];
 const todo=iv.open+open.length;
 return h('div',{class:'validate-page'},[
  pageHeader('03 / VALIDATE','Check what the agents found',
   'Confirm, correct or reject each record and finding against its evidence. Gaps and conflicts stay visible until someone resolves them.',
   [button('Add evidence',()=>showEvidenceStep('sources')),button('Discovery summary →',()=>go('summary'),'btn primary')]),
  h('div',{class:'metrics'},[
   ['Records to confirm',iv.open],['Findings to review',open.length],['Conflicts',s.conflicts],['Owner decisions open',iv.decisions]
  ].map(([l,v])=>h('div',{class:'metric'},[h('b',{text:String(v)}),h('span',{text:l})]))),
  h('p',{class:'muted validate-status',role:'status',text:todo?todo+' items still need a person to check them.':'Everything has been checked by a person.'}),
  h('div',{class:'tabs validate-tabs',role:'tablist'},tabs.map(([id,title,hint])=>h('button',{
   type:'button',role:'tab','aria-selected':String(id===cur[0]),class:'tab '+(id===cur[0]?'active':''),
   onclick:()=>{validateTab=id;render();}},[h('b',{text:title}),' · ',h('span',{text:hint})]))),
  h('div',{class:'validate-body',role:'tabpanel'},[validateTabBody(cur[3]())])
 ]);
}
