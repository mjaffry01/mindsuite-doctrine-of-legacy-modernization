/* MindSuite web app · what a plan approval covers, and what changed since.
   A modernization approval now covers the wave dates, change-freeze months, business events and team
   capacity as well as each decision's wave, pattern and size. Moving an approved wave (for example into a
   freeze month) makes the approval "Changed since approval", says what changed, which capabilities and
   journeys it affects, and who must review it again. Approvals saved before this change used the old
   comparison: they still count for the decisions, but are flagged as not covering the dates. */

function planApprovalSnapshot(){
 const product=isNewProduct(),C=product?launchContexts():decCtxs();
 const waves={};for(const [n,w] of Object.entries((M.plan&&M.plan.waves)||{}))waves[n]={start:(w&&w.start)||'',end:(w&&w.end)||''};
 return {waves,freeze:[...((M.plan&&M.plan.freezeMonths)||[])].sort((a,b)=>a-b),events:(M.plan&&M.plan.events)||'',teams:(M.plan&&M.plan.teams)||'',
  items:C.map(c=>{const p=planOf(c);return {id:c.id,name:c.name,wave:p.wave,pattern:p.pattern,size:p.size,approach:p.d.approach,status:p.d.status};})};
}
// The old modernization comparison (decisions only), kept to recognise approvals saved before dates were covered.
function planFingerprintV1(){const C=decCtxs();return JSON.stringify(C.map(c=>{const p=planOf(c);return [c.id,p.d.status,p.d.approach,p.d.choices,p.wave,p.pattern,p.size];}));}

const PA_MONTH=['Jan','Feb','Mar','Apr','May','Jun','Jul','Aug','Sep','Oct','Nov','Dec'];
function paMonths(start,end){ // months (1–12) covered by a YYYY-MM range
 const p=v=>{const m=/^(\d{4})-(\d{2})/.exec(v||'');return m?(+m[1])*12+(+m[2]-1):null;};
 const a=p(start),b=p(end||start);if(a==null&&b==null)return [];const s=a??b,e=b??a,out=[];
 for(let x=s;x<=Math.min(e,s+36);x++)out.push(x%12+1);return [...new Set(out)];
}
function paFmt(v){const m=/^(\d{4})-(\d{2})/.exec(v||'');return m?PA_MONTH[+m[2]-1]+' '+m[1]:'not set';}

/* {state:'none'|'current'|'changed'|'legacy', changes:[...], reviewers:[...]} */
function planApprovalState(){
 const so=M.plan&&M.plan.signoff;if(!so)return {state:'none',changes:[],reviewers:[]};
 if(so.fingerprint===planFingerprint())return {state:'current',changes:[],reviewers:[]};
 const product=isNewProduct(),now=planApprovalSnapshot(),was=so.snapshot,changes=[];
 const reviewers=[so.name].concat(((M.people||[]).filter(p=>p.role==='Business owner').map(p=>p.name))).filter((x,i,a)=>x&&a.indexOf(x)===i);
 if(!was){
  if(!product&&so.fingerprint===planFingerprintV1())return {state:'legacy',changes:['This approval was given before approvals covered wave dates, freeze months and business events. Approve again so the signed plan includes them.'],reviewers};
  return {state:'changed',changes:['The plan changed after it was approved (no record of the approved version to compare with).'],reviewers};
 }
 const stageName=n=>{try{if(typeof planStageLabel==='function')return planStageLabel(+n);}catch(e){}return (product?'Stage ':'Wave ')+n;};
 const inWave=n=>now.items.filter(i=>String(i.wave)===String(n)).map(i=>i.name);
 const freeze=new Set(now.freeze);
 for(const n of new Set([...Object.keys(was.waves||{}),...Object.keys(now.waves||{})])){
  const a=(was.waves||{})[n]||{},b=now.waves[n]||{};
  if((a.start||'')!==(b.start||'')||(a.end||'')!==(b.end||'')){
   const caps=inWave(n);
   changes.push(stageName(n)+' moved: '+paFmt(a.start)+' – '+paFmt(a.end)+' → '+paFmt(b.start)+' – '+paFmt(b.end)+(caps.length?'. Affects: '+caps.join(', '):''));
   const hit=paMonths(b.start,b.end).filter(m=>freeze.has(m)),had=paMonths(a.start,a.end).filter(m=>(was.freeze||[]).includes(m));
   if(hit.length&&hit.join()!==had.join())changes.push(stageName(n)+' now runs in change-freeze month'+(hit.length>1?'s':'')+' ('+hit.map(m=>PA_MONTH[m-1]).join(', ')+')'+(caps.length?': cutover of '+caps.join(', ')+' would fall inside the freeze':''));
  }
 }
 if((was.freeze||[]).join()!==now.freeze.join())changes.push('Change-freeze months changed: '+((was.freeze||[]).map(m=>PA_MONTH[m-1]).join(', ')||'none')+' → '+(now.freeze.map(m=>PA_MONTH[m-1]).join(', ')||'none'));
 if((was.events||'')!==now.events)changes.push((product?'Launch dependencies':'Business events to avoid')+' changed');
 if(String(was.teams||'')!==String(now.teams||''))changes.push('Delivery teams changed: '+(was.teams||'not set')+' → '+(now.teams||'not set'));
 const W=Object.fromEntries((was.items||[]).map(i=>[i.id,i]));
 for(const i of now.items){const o=W[i.id];if(!o){changes.push(i.name+' added to the plan');continue;}
  const d=[];if(String(o.wave)!==String(i.wave))d.push('stage '+(o.wave||'none')+' → '+(i.wave||'none'));if(o.pattern!==i.pattern)d.push('pattern '+(o.pattern||'none')+' → '+(i.pattern||'none'));if(o.size!==i.size)d.push('size '+(o.size||'?')+' → '+(i.size||'?'));if(o.approach!==i.approach)d.push('approach changed');if(o.status!==i.status)d.push('decision '+(o.status||'open')+' → '+(i.status||'open'));
  if(d.length)changes.push(i.name+': '+d.join(', '));}
 for(const o of was.items||[])if(!now.items.some(i=>i.id===o.id))changes.push(o.name+' removed from the plan');
 if(!changes.length)changes.push('Details of a capability or the brief changed since approval');
 return {state:'changed',changes,reviewers};
}
