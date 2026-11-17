/* Shared engine of the agent team explanation pages: diagram, agent panel, steps and cards.
   Each page defines AGENTS, EDGES, COLS, POS, SHORT, ORCH_LINKS and STEPS before loading this file. */
const W=160,Hh=40;
const svg=document.getElementById('diagram');const NS='http://www.w3.org/2000/svg';
const mk=(t,a,txt)=>{const e=document.createElementNS(NS,t);for(const k in a)e.setAttribute(k,a[k]);if(txt!=null)e.textContent=txt;return e;};
for(const [s,x] of Object.entries(COLS))svg.append(mk('text',{x,y:88,'text-anchor':'middle',class:'stage-label'},s==='Facts'?'Facts (in parallel)':s));
const edgeEls=[];const center=id=>POS[id];
function line(a,b,cls){const [x1,y1]=center(a),[x2,y2]=center(b);let sx=x1,sy=y1,ex=x2,ey=y2;if(Math.abs(x2-x1)>5){sx=x1+(x2>x1?W/2:-W/2);ex=x2-(x2>x1?W/2:-W/2);}else{sy=y1+(y2>y1?Hh/2:-Hh/2);ey=y2-(y2>y1?Hh/2:-Hh/2);}const mx=(sx+ex)/2;const p=mk('path',{d:'M'+sx+','+sy+' C'+mx+','+sy+' '+mx+','+ey+' '+ex+','+ey,class:'edge '+(cls||''),'marker-end':'url(#ar)'});svg.append(p);return p;}
const defs=mk('defs',{});const m=mk('marker',{id:'ar',viewBox:'0 0 10 10',refX:'9',refY:'5',markerWidth:'6',markerHeight:'6',orient:'auto-start-reverse'});m.append(mk('path',{d:'M0,0 L10,5 L0,10 z',fill:'#9fb0cc'}));defs.append(m);svg.append(defs);
for(const [a,b] of EDGES){const el=line(a,b);el.dataset.a=a;el.dataset.b=b;edgeEls.push(el);}
// orchestrator and evidence store links (dotted)
for(const id of ORCH_LINKS){const [x1,y1]=POS.orch,[x2,y2]=POS[id];const p=mk('path',{d:'M'+x1+','+(y1+Hh/2)+' Q'+x1+','+(y2-40)+' '+x2+','+(y2-Hh/2),class:'edge store'});p.dataset.a='orch';p.dataset.b=id;svg.append(p);edgeEls.push(p);}
const nodeEls={};
for(const a of AGENTS){const [x,y]=POS[a.id];const w=a.id==='store'?520:W;const g=mk('g',{class:'node '+a.status,tabindex:'0',role:'button','aria-label':a.name});g.append(mk('rect',{x:x-w/2,y:y-Hh/2,width:w,height:Hh}));g.append(mk('text',{x:x,y:y+4,'text-anchor':'middle'},(a.n!==''?a.n+' · ':'')+(SHORT[a.id]||a.name)));g.addEventListener('click',()=>select(a.id));g.addEventListener('keydown',e=>{if(e.key==='Enter'||e.key===' '){e.preventDefault();select(a.id);}});svg.append(g);nodeEls[a.id]=g;}
const byId=Object.fromEntries(AGENTS.map(a=>[a.id,a]));
const statusText={built:'Running today',partly:'Partly running',planned:'Planned',people:'Human',store:'Shared record'};
function select(id){const a=byId[id];const linked=new Set([id]);for(const e of edgeEls){const hit=e.dataset.a===id||e.dataset.b===id;e.classList.toggle('hl',hit);e.classList.toggle('dim',!hit);if(hit){linked.add(e.dataset.a);linked.add(e.dataset.b);}}
 for(const [k,g] of Object.entries(nodeEls)){g.classList.toggle('sel',k===id);g.classList.toggle('dim',!linked.has(k)&&k!=='store');}
 const from=EDGES.filter(e=>e[1]===id).map(e=>e[0]),to=EDGES.filter(e=>e[0]===id).map(e=>e[1]);if(id!=='orch'&&!['store','ppl'].includes(id))from.unshift('orch');
 const chips=list=>list.length?'<span class="chips">'+list.map(x=>'<button type="button" data-go="'+x+'">'+byId[x].name+'</button>').join('')+'</span>':'—';
 const panel=document.getElementById('panel');panel.innerHTML='<h3>'+(a.n!==''?a.n+' · ':'')+a.name+'</h3><div class="tags"><span class="tag '+a.status+'">'+statusText[a.status]+'</span><span class="tag">'+a.type+'</span>'+(a.tier?'<span class="tag">Model: '+a.tier+'</span>':'')+'</div><p>'+a.does+'</p><dl class="kv"><dt>Reads</dt><dd>'+a.reads+'</dd><dt>Produces</dt><dd>'+a.produces+'</dd><dt>Receives from</dt><dd>'+(id==='store'?'Every agent':chips(from))+'</dd><dt>Sends to</dt><dd>'+(id==='store'?'Every agent and your people':chips(to.concat(id==='ppl'?[]:[]))+(['orch','ppl'].includes(id)?'':' · and the evidence store'))+'</dd><dt>Safeguard</dt><dd>'+a.safeguard+'</dd></dl>';
 panel.querySelectorAll('[data-go]').forEach(b=>b.addEventListener('click',()=>select(b.dataset.go)));}
const stepsEl=document.getElementById('steps');
for(const s of STEPS){const d=document.createElement('div');d.className='step';d.innerHTML='<div class="step-num">'+s.n+'</div><div class="step-body"><div class="step-head"><b>'+s.title+'</b><span class="tag '+s.status+'">'+statusText[s.status]+'</span></div><p class="step-goal">Serves the goal: '+s.goal+'</p><p>'+s.does+'</p><p class="step-meta"><span>Agents:</span> '+s.agents.map(a=>'<button type="button" data-go="'+a+'">'+byId[a].name+'</button>').join(' ')+'</p><p class="step-meta"><span>You see:</span> '+s.see+'</p></div>';d.querySelectorAll('[data-go]').forEach(b=>b.addEventListener('click',()=>{select(b.dataset.go);document.getElementById('diagram').scrollIntoView({behavior:'smooth',block:'center'});}));stepsEl.append(d);}
function fromHash(){const id=location.hash.slice(1);if(byId[id]){select(id);setTimeout(()=>document.getElementById('diagram').scrollIntoView({block:'center'}),50);}}
window.addEventListener('hashchange',fromHash);fromHash();
const cards=document.getElementById('cards');for(const a of AGENTS){const c=document.createElement('div');c.className='card';c.tabIndex=0;c.innerHTML='<div class="tags"><span class="tag '+a.status+'">'+statusText[a.status]+'</span><span class="tag">'+a.stage+'</span></div><h3>'+(a.n!==''?a.n+' · ':'')+a.name+'</h3><p>'+a.does+'</p>';c.addEventListener('click',()=>{select(a.id);document.getElementById('diagram').scrollIntoView({behavior:'smooth',block:'center'});});cards.append(c);}
