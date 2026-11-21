/* MindSuite web app · "On this page" index in the left rail.
   Under the active page in the left navigation, lists that page's tabs and its sections (h2), with the
   sub-sections (h3) of the section you are reading. Built from the page itself after every render, so every
   page gets one without its own code. Click or Enter jumps to the section; the entry for the section in
   view is highlighted while you scroll. Pages with fewer than two entries show no index. */

const PI={timer:0,sig:'',items:[],spyTimer:0};

function piVisible(e){
 if(!e||!e.offsetParent)return false;
 if(e.closest('dialog,.page-index'))return false;
 const d=e.closest('details');if(d&&!d.open&&!e.closest('summary'))return false;
 return true;
}
function piLabel(e){return String(e.textContent||'').replace(/\s+/g,' ').trim().slice(0,70);}

function piCollect(){
 const main=document.getElementById('main');if(!main)return [];
 const out=[];
 const tabs=[...main.querySelectorAll('[role=tab]')].filter(piVisible);
 for(const t of tabs){const lab=(t.querySelector('b,strong')||t).textContent.replace(/\s+/g,' ').trim()||piLabel(t);out.push({kind:'tab',el:t,label:lab.slice(0,50),selected:t.getAttribute('aria-selected')==='true'||t.classList.contains('active')});}
 let n=0;
 const heads=[...main.querySelectorAll('h2,h3')].filter(piVisible);
 const h2s=heads.filter(e=>e.tagName==='H2');
 // Pages with a single section use their sub-headings as the index.
 const top=h2s.length>=2?'H2':'H3';
 let cur=null;
 for(const e of heads){
  if(!e.id)e.id='pi-'+(++n)+'-'+piLabel(e).toLowerCase().replace(/[^a-z0-9]+/g,'-').slice(0,30);
  if(e.tagName===top){cur={kind:'sec',el:e,label:piLabel(e),subs:[]};out.push(cur);}
  else if(top==='H2'&&e.tagName==='H3'&&cur)cur.subs.push({kind:'sub',el:e,label:piLabel(e)});
 }
 return out;
}

function piJump(el){
 try{if(typeof workspaceMenu==='function'&&document.body.classList.contains('nav-open'))workspaceMenu(false);}catch(e){}
 el.scrollIntoView({behavior:'smooth',block:'start'});
 if(!el.hasAttribute('tabindex'))el.setAttribute('tabindex','-1');
 setTimeout(()=>{try{el.focus({preventScroll:true});}catch(e){}},350);
}

function piBuild(){
 const nav=document.getElementById('phaseNav');if(!nav)return;
 const active=nav.querySelector('.phase.active');
 const items=piCollect();
 const sig=(active?active.dataset.screen:'')+'|'+items.map(i=>i.kind+':'+i.label+(i.selected?'*':'')+(i.subs?'['+i.subs.map(s=>s.label).join(',')+']':'')).join('|');
 const existing=nav.querySelector('.page-index');
 if(sig===PI.sig&&existing&&active&&existing.previousElementSibling===active){PI.items=items;piSpy();return;}
 PI.sig=sig;PI.items=items;
 if(existing)existing.remove();
 if(!active||items.length<2)return;
 const box=document.createElement('nav');box.className='page-index';box.setAttribute('aria-label','On this page');
 const head=document.createElement('div');head.className='pi-head';head.textContent='On this page';box.append(head);
 const ul=document.createElement('ul');
 const link=(it,cls)=>{const a=document.createElement('a');a.href='#'+(it.el.id||'');a.className=cls;a.textContent=it.label;a.title=it.label;
  a.onclick=ev=>{ev.preventDefault();if(it.kind==='tab'){it.el.click();setTimeout(piBuild,60);}else piJump(it.el);};return a;};
 for(const it of items){
  const li=document.createElement('li');
  if(it.kind==='tab'){const a=link(it,'pi-tab'+(it.selected?' on':''));if(it.selected)a.setAttribute('aria-current','true');li.append(a);}
  else{li.append(link(it,'pi-sec'));if(it.subs.length){const sub=document.createElement('ul');sub.className='pi-subs';for(const s of it.subs){const l2=document.createElement('li');l2.append(link(s,'pi-sub'));sub.append(l2);}li.append(sub);}}
  it.li=li;ul.append(li);
 }
 box.append(ul);
 active.after(box);
 piSpy();
}

function piSpy(){
 const secs=PI.items.filter(i=>i.kind==='sec'&&i.li);if(!secs.length)return;
 const line=130;let cur=secs[0];
 for(const s of secs){if(s.el.getBoundingClientRect().top<=line)cur=s;else break;}
 // At the very bottom of the page, the last section counts as read.
 const se=document.scrollingElement;if(se&&se.scrollTop+window.innerHeight>=se.scrollHeight-4)cur=secs[secs.length-1];
 for(const s of secs){const on=s===cur;s.li.classList.toggle('on',on);const a=s.li.querySelector('.pi-sec');if(a){if(on)a.setAttribute('aria-current','location');else a.removeAttribute('aria-current');}
  if(on&&s.subs.length){let cs=null;for(const x of s.subs){if(x.el.getBoundingClientRect().top<=line)cs=x;}s.li.querySelectorAll('.pi-sub').forEach((a,i)=>a.classList.toggle('on',s.subs[i]===cs));}}
}

function piSchedule(){clearTimeout(PI.timer);PI.timer=setTimeout(piBuild,120);}
(function piInit(){
 const start=()=>{
  const main=document.getElementById('main');if(!main){setTimeout(start,200);return;}
  new MutationObserver(piSchedule).observe(main,{childList:true,subtree:true,attributes:true,attributeFilter:['open','aria-selected','hidden','class']});
  const nav=document.getElementById('phaseNav');if(nav)new MutationObserver(m=>{if(m.some(r=>[...r.addedNodes].some(n=>n.classList&&n.classList.contains('phase'))))piSchedule();}).observe(nav,{childList:true});
  document.addEventListener('scroll',()=>{if(PI.spyTimer)return;PI.spyTimer=requestAnimationFrame(()=>{PI.spyTimer=0;piSpy();});},{capture:true,passive:true});
  window.addEventListener('resize',piSchedule);
  const css=document.createElement('style');css.textContent=`
.page-index{margin:2px 0 10px 29px;padding:2px 0 4px 10px;border-left:1px solid #e3ddef}
.page-index .pi-head{font-size:10.5px;letter-spacing:.8px;text-transform:uppercase;color:#8a8ca0;font-weight:600;margin:4px 0 4px}
.page-index ul{list-style:none;margin:0;padding:0}
.page-index a{display:block;color:#5f6276;text-decoration:none;font-size:12.5px;line-height:1.35;padding:4px 8px;border-radius:6px;overflow:hidden;display:-webkit-box;-webkit-line-clamp:2;-webkit-box-orient:vertical;word-break:break-word}
.page-index a:hover{background:#f3f0f9;color:#353249}
.page-index a:focus-visible{outline:2px solid #7c6bc4;outline-offset:1px}
.page-index li.on>.pi-sec,.page-index .pi-tab.on,.page-index .pi-sub.on{color:#5b47a8;font-weight:600;background:#f1edfa}
.page-index .pi-tab{font-weight:500}
.page-index .pi-subs{display:none;margin:1px 0 3px 10px;border-left:1px solid #ece8f4}
.page-index li.on>.pi-subs{display:block}
.page-index .pi-sub{font-size:12px;padding:3px 8px}
.layout>aside{overflow-y:auto;scrollbar-width:thin;scrollbar-color:#d9d3e6 transparent}
main h2,main h3{scroll-margin-top:90px}
main h2[tabindex="-1"]:focus:not(:focus-visible),main h3[tabindex="-1"]:focus:not(:focus-visible){outline:none;box-shadow:none}
@media print{.page-index{display:none}}`;
  document.head.append(css);
  piSchedule();
 };
 if(document.readyState==='loading')document.addEventListener('DOMContentLoaded',start);else start();
})();
