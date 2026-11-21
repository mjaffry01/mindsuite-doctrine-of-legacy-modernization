/* Presentation enhancements only. No business state or localStorage writes.
   Called by the existing final render(), after renderNav(). */
const workspaceUI={query:'',filter:'all'};
function workspaceIcon(paths){
 const ns='http://www.w3.org/2000/svg',svg=document.createElementNS(ns,'svg');
 for(const [k,v] of Object.entries({viewBox:'0 0 24 24',fill:'none',stroke:'currentColor','stroke-width':'1.7','stroke-linecap':'round','stroke-linejoin':'round','aria-hidden':'true'}))svg.setAttribute(k,v);
 for(const d of paths){const p=document.createElementNS(ns,'path');p.setAttribute('d',d);svg.append(p);}return svg;
}
function workspaceMenu(open,restoreFocus=false){
 document.body.classList.toggle('nav-open',open);
 const toggle=document.getElementById('workspaceMenu');toggle?.setAttribute('aria-expanded',String(open));
 if(open)document.querySelector('#phaseNav .active')?.focus();else if(restoreFocus)toggle?.focus();
}
function enhanceWorkspace(groups){
 document.getElementById('main').setAttribute('tabindex','-1');
 const aside=document.querySelector('.layout>aside');aside.id='workspace-navigation';
 if(!document.getElementById('workspaceMenu')){
  const toggle=document.createElement('button');toggle.id='workspaceMenu';toggle.type='button';toggle.className='mobile-menu';toggle.setAttribute('aria-label','Open workspace navigation');toggle.setAttribute('aria-expanded','false');toggle.setAttribute('aria-controls','workspace-navigation');toggle.append(workspaceIcon(['M4 6h16M4 12h16M4 18h16']));toggle.onclick=()=>workspaceMenu(!document.body.classList.contains('nav-open'));
  document.querySelector('.app-top').prepend(toggle);
  const backdrop=document.createElement('button');backdrop.type='button';backdrop.className='nav-backdrop';backdrop.setAttribute('aria-label','Close workspace navigation');backdrop.onclick=()=>workspaceMenu(false,true);document.body.append(backdrop);
  document.addEventListener('keydown',e=>{if(e.key==='Escape'&&document.body.classList.contains('nav-open'))workspaceMenu(false,true);});
  document.getElementById('phaseNav').addEventListener('click',e=>{if(e.target.closest('.phase')){workspaceMenu(false);if(matchMedia('(max-width:760px)').matches)document.getElementById('main').focus({preventScroll:true});}});
 }
 for(const item of document.querySelectorAll('#phaseNav .phase')){if(item.classList.contains('active'))item.setAttribute('aria-current','page');else item.removeAttribute('aria-current');}
 const icons={
  'Source code':['M8 5 2 12l6 7M16 5l6 7-6 7M14 3l-4 18'],
  'Database':['M4 6c0-4 16-4 16 0s-16 4-16 0v12c0 4 16 4 16 0V6M4 12c0 4 16 4 16 0'],
  'Environment & infrastructure':['M4 3h16v7H4zM4 14h16v7H4zM7 6h.01M7 17h.01M11 6h6M11 17h6'],
  'Messaging & caching':['M3 4h18v13H9l-5 4v-4H3zM7 8h10M7 12h6'],
  'Identity & access':['M12 3 3 7v5c0 5 9 10 9 10s9-5 9-10V7zM8 12l3 3 5-6'],
  'Telemetry':['M3 3v18h18M5 15l4-5 4 3 6-8'],
  'Batch & scheduling':['M12 3a9 9 0 1 0 9 9 9 9 0 0 0-9-9M12 7v5l3 2'],
  'Network edge & partners':['M9 3h6v6H9zM2 16h6v6H2zM16 16h6v6h-6zM12 9v4M5 16v-3h14v3'],
  'Documents & diagrams':['M6 3h9l4 4v14H6zM14 3v5h5M9 12h7M9 16h5'],
  'Incidents & changes':['M12 3 2 21h20zM12 9v5M12 17h.01']
 };
 for(const card of document.querySelectorAll('.source-card')){
  const name=card.querySelector('h3')?.textContent,icon=card.querySelector('.category-icon');if(icon&&icons[name])icon.replaceChildren(workspaceIcon(icons[name]));
 }
 const section=document.getElementById('rec-sources'),grid=section?.querySelector(':scope>.source-grid');
 if(grid){
  const tools=document.createElement('div');tools.className='source-tools';
  const search=document.createElement('label');search.className='source-search';search.append(workspaceIcon(['M10 3a7 7 0 1 0 0 14 7 7 0 0 0 0-14M15 15l6 6']));
  const input=document.createElement('input');input.type='search';input.className='field';input.placeholder='Search sources';input.setAttribute('aria-label','Find an evidence source');input.value=workspaceUI.query;search.append(input);
  const filters=document.createElement('div');filters.className='source-filter';filters.setAttribute('role','group');filters.setAttribute('aria-label','Filter evidence sources');
  const count=document.createElement('span');count.className='source-results';count.setAttribute('role','status');
  const empty=document.createElement('div');empty.className='source-empty';empty.hidden=true;empty.textContent='No sources match. Try another search or show all sources.';
  const cards=[...grid.children];
  const apply=()=>{const q=workspaceUI.query.trim().toLowerCase();let shown=0;for(const card of cards){const matches=(card.querySelector('h3').textContent+' '+card.querySelector('.source-detects').textContent).toLowerCase().includes(q);const needs=card.classList.contains('open')||card.classList.contains('partial');card.hidden=!(matches&&(workspaceUI.filter==='all'||needs));if(!card.hidden)shown++;}count.textContent=shown+' of '+cards.length;empty.hidden=shown!==0;for(const b of filters.children)b.setAttribute('aria-pressed',String(b.dataset.filter===workspaceUI.filter));};
  for(const [id,label] of [['all','All sources'],['open','Needs input']]){const b=document.createElement('button');b.type='button';b.textContent=label;b.dataset.filter=id;b.onclick=()=>{workspaceUI.filter=id;apply();};filters.append(b);}
  input.oninput=()=>{workspaceUI.query=input.value;apply();};tools.append(search,filters,count);grid.before(tools);grid.after(empty);apply();
 }
 enhanceContemporaryUI(groups);
}

/* Contemporary navigation and compact agent actions. Presentation only. */
function enhanceContemporaryUI(groups){
 const navPaths=[['M3 5h18v15H3zM7 3v4M17 3v4M3 10h18M7 14h4'],['M8 6 2 12l6 6M16 6l6 6-6 6M14 4l-4 16'],['M3 3h7v7H3zM14 3h7v7h-7zM3 14h7v7H3zM14 14h7v7h-7z'],['M5 3h14v18H5zM8 8h8M8 12h5M8 16h6'],['M12 3 2 21h20zM12 9v5M12 17h.01'],['M5 3h10l4 4v14H5zM14 3v5h5M8 12h8M8 16h6'],['M9 11a4 4 0 1 0 0-8 4 4 0 0 0 0 8M2 21v-3a7 7 0 0 1 14 0v3M17 4a3 3 0 0 1 0 6M22 21v-3a6 6 0 0 0-4-5'],['M4 4h6v6H4zM14 14h6v6h-6zM10 7h7v7M4 18h6M7 15v6'],['M4 4v16M4 7h13M4 13h9M4 19h16M17 5v4M13 11v4M20 17v4'],['M4 3h11l4 4v5M4 3v18h8M14 3v5h5M14 18l3 3 5-7']];
 const nav=[...document.querySelectorAll('#phaseNav .phase')];
 const iconIndex={upload:0,analysis:1,validate:2,summary:3,verify:4,rules:6,decide:7,roadmap:8,signoff:9,'release-scope':5,assumptions:4,hld:3,agents:1};
 for(const button of nav){
  if(button.querySelector('.nav-icon'))continue;
  const icon=document.createElement('span');icon.className='nav-icon';
  icon.append(workspaceIcon(navPaths[iconIndex[button.dataset.screen]]||navPaths[0]));
  button.prepend(icon);
 }
 let stages=document.querySelector('.workspace-stages');
 if(!stages){
  stages=document.createElement('nav');stages.className='workspace-stages';stages.setAttribute('aria-label','Project stages');
  document.querySelector('.top-right').before(stages);
 }
 const current=workflowStep();
 const focused=document.activeElement?.dataset.stage;
 stages.replaceChildren(...groups.map(({title,steps},index)=>{
  const [first]=steps[0];
  return h('button',{
   type:'button','data-stage':String(index),disabled:busy?'':null,
   'aria-current':steps.some(([id])=>id===current)?'step':null,
   onclick:()=>go(first)
  },[h('span',{text:String(index+1).padStart(2,'0')}),title]);
 }));
 if(focused!=null)stages.querySelector('[data-stage="'+focused+'"]')?.focus({preventScroll:true});
}
