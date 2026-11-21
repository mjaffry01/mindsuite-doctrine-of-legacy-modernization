/* MindSuite web app · Cookie and session policy (second tab of the 'datapolicy' screen).
   The Session policy agent (Claude on the project backend, POST /api/design/sessionpolicy) drafts, on request:
   cookies by category (strictly necessary, preferences, analytics, marketing) with consent, attributes
   (Secure, HttpOnly, SameSite), lifetime and domain/path; and a session policy per user type (store, idle
   timeout, absolute lifetime, re-authentication, MFA, sign-out everywhere, rotation on sign-in, concurrent
   sessions, remember-me, behaviour during a cache or region outage). It sends design metadata only: the
   platform, targets and regulations, the identity, cache, security and recovery design, the user types (staff
   or not) and the channels. No records, no names of people, no secrets.
   Cookies are accepted by the privacy or data owner, sessions by the security owner; drafts stay drafts.
   Stored in M.dataPolicy.session = {userTypes:[…], removed:{}, ai, cookies:{rules,by,byName,role,at}, sessions:{…}}. */

const SP_UI={busy:false,message:''};
const SP_CATS=[['necessary','Strictly necessary'],['preferences','Preferences'],['analytics','Analytics'],['marketing','Marketing']];
const SP_STORES=[['server','Kept on the server (opaque id)'],['token','Signed tokens (apps and APIs)'],['hybrid','Both']];
const SP_MFA=[['required','Required'],['optional','Optional'],['not-required','Not required']];
const SP_STAFF=/staff|admin|editor|operator|manager|agent\b|support|restaurant|employee|clerk|officer/i;

function spStore(){const D=dpStore();D.session=D.session||{};const S=D.session;S.userTypes=S.userTypes||[];S.removed=S.removed||{};return S;}
function spUserTypes(){
 const S=spStore(),out=[],add=(name,staff,source)=>{const k=dpKey(name);if(!k||S.removed[k]||out.some(u=>dpKey(u.name)===k))return;out.push({name:String(name).trim().slice(0,60),staff:!!staff,source});};
 const P=M.product||{};let roles=[];try{roles=typeof tpRoles==='function'?tpRoles(P.users||''):[];}catch(e){}
 // The brief lists users as "Name: what they do." — take every name before a colon at the start of a sentence.
 for(const m of String(P.users||'').matchAll(/(?:^|[.;\n]\s*)([A-Z][A-Za-z /&-]{2,40}?):/g))if(!roles.some(r=>dpKey(r)===dpKey(m[1])))roles.push(m[1].trim());
 for(const r of roles)add(r,SP_STAFF.test(r),'Product brief');
 if(!out.length){if(isNewProduct())add('Customer',false,'Default');add(isNewProduct()?'Staff':'Business user',!isNewProduct()?false:true,'Default');add('Administrator',true,'Default');}
 for(const u of S.userTypes)add(u.name,u.staff,'Added by you');
 return out.slice(0,12);
}
function spChannels(){const ch=((M.product||{}).deploy||{}).channels||[];const lab={web:'Web browser',mobile:'Mobile app',api:'API (no user interface)',desktop:'Desktop app',kiosk:'Kiosk'};return (ch.length?ch:['web']).map(c=>lab[c]||c).slice(0,8);}

function spRequest(){
 const pf=typeof platformEffective==='function'?platformEffective():{};const lab=(d,id)=>id?((optOf(d,id)||[,id])[1]):'';
 const cut=(t,n)=>String(t||'').replace(/\s+/g,' ').trim().slice(0,n);
 let B=null;try{B=systemBlueprint(decCtxs(),'current');}catch(e){}
 const layers=B?B.platform.filter(l=>['identity','authz','cache','security','availability','dr'].includes(l.id)&&String(l.t||'').trim()).map(l=>({id:cut(l.id,40),label:cut(l.label,80),design:cut(l.t,1500)})):[];
 const I=(()=>{try{return techPlanInputs({id:'system',name:'',description:''});}catch(e){return {regs:[]};}})();
 const P=M.product||{},rec=typeof capRecovery==='function'?capRecovery():{rto:'',rpo:''};
 return {project:cut(M.project,180),kind:isNewProduct()?'new':(typeof modeOf==='function'&&modeOf()==='extend'?'extend':'modernize'),
  platform:['Tier: '+(pf.tier||''),'Cloud: '+lab('cloud',pf.cloud),'Hosting: '+(pf.hosting||''),'Architecture: '+lab('architecture',pf.architecture)].filter(s=>!/: $/.test(s)).map(s=>cut(s,300)),
  targets:[P.availability?'Availability: '+P.availability+'%':'',rec.rto?'RTO: '+rec.rto:'',(I.regs||[]).length?'Regulations: '+I.regs.join(', '):'Regulations: none recorded',P.deploy&&P.deploy.region?'Region: '+P.deploy.region:''].filter(Boolean).map(t=>cut(t,300)),
  layers:layers.slice(0,12),userTypes:spUserTypes().map(u=>({name:u.name,staff:u.staff})),channels:spChannels()};
}
function spBasis(req){const r=req||spRequest();return JSON.stringify({p:r.platform,t:r.targets,l:r.layers.map(l=>l.id+':'+l.design),u:r.userTypes.map(u=>u.name+(u.staff?'*':'')),c:r.channels});}
function spStaleReasons(){
 const ai=spStore().ai;if(!ai||!ai.basis)return [];let was;try{was=JSON.parse(ai.basis);}catch(e){return ['the design changed'];}
 const now=JSON.parse(spBasis()),out=[],ch=(k,t)=>{if(JSON.stringify(was[k])!==JSON.stringify(now[k]))out.push(t);};
 ch('p','the platform decision changed');ch('t','the targets or regulations changed');ch('l','the identity, cache or recovery design changed');ch('u','the user types changed');ch('c','the channels changed');return out;
}

async function runSessionPolicyAgent(){
 if(SP_UI.busy)return;
 const req=spRequest();SP_UI.busy=true;SP_UI.message='The Session policy agent is reading the design…';render();
 const t0=Date.now();
 try{
  const out=await agentFetch('/api/design/sessionpolicy',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(req)});
  spStore().ai={...out,at:new Date().toISOString(),ms:Date.now()-t0,basis:spBasis(req)};
  logInput('sessionpolicy:agent',null,(out.cookies||[]).length+' cookie rules, '+(out.sessions||[]).length+' session rules, '+(out.conflicts||[]).length+' conflicts');
  SP_UI.message='';persist();
 }catch(e){SP_UI.message=/404|not found/i.test(e.message)?'This backend needs the Session policy agent update. Restart the project backend with the latest build.':'The Session policy agent could not finish. '+e.message;}
 finally{SP_UI.busy=false;render();}
}

const spCookies=()=>{const S=spStore();return S.cookies?{rules:S.cookies.rules,accepted:true}:{rules:(S.ai&&S.ai.cookies)||[],accepted:false};};
const spSessions=()=>{const S=spStore();return S.sessions?{rules:S.sessions.rules,accepted:true}:{rules:(S.ai&&S.ai.sessions)||[],accepted:false};};
function spAccept(kind){
 const S=spStore(),src=kind==='cookies'?spCookies().rules:spSessions().rules,p=(M.people||[]).find(x=>x.id===M.actor);
 const blank=kind==='cookies'?SP_CATS.map(([c])=>({category:c,purpose:'',consent:c==='necessary'?'not-required':'required',attributes:c==='necessary'?'Secure; HttpOnly; SameSite=Lax':'Secure; SameSite=Lax',lifetime:'',domainPath:'',why:''}))
  :spUserTypes().map(u=>({userType:u.name,channels:'',store:'',idleTimeout:'',absoluteLifetime:'',reauth:'',mfa:u.staff?'required':'',signOutEverywhere:'',rotation:'New session id on sign-in and on any change of rights',concurrent:'',rememberMe:u.staff?'Never':'',outage:'',why:''}));
 S[kind]={rules:JSON.parse(JSON.stringify(src.length?src:blank)),by:M.actor||'',byName:p?p.name:'Local user',role:p?p.role:'',at:new Date().toISOString()};
 logInput('sessionpolicy:accept:'+kind,null,'accepted');persist();render();
}
function spOwnerNote(kind){const a=spStore()[kind];if(!a)return null;const want=kind==='sessions'?/security/i:/privacy|data|compliance|legal|business owner/i;
 return want.test(a.role||'')?null:h('p',{class:'warn-text',text:'Accepted by '+a.byName+(a.role?' ('+a.role+')':'')+': '+(kind==='sessions'?'the security owner':'the privacy or data owner')+' should accept this.'});}

/* Local checks against the design and the brief. */
function spConflicts(){
 const out=[],P=M.product||{},avail=parseFloat(P.availability)||0;
 let cacheT='',regs=[];try{const B=systemBlueprint(decCtxs(),'current');cacheT=String((B.platform.find(l=>l.id==='cache')||{}).t||'');}catch(e){}
 try{regs=techPlanInputs({id:'system',name:'',description:''}).regs||[];}catch(e){}
 const strict=regs.some(r=>/gdpr|personal data|dpdp|privacy|eprivacy|lgpd|ccpa/i.test(r));
 const ut=Object.fromEntries(spUserTypes().map(u=>[dpKey(u.name),u]));
 for(const s of spSessions().rules){
  if((s.store==='server'||s.store==='hybrid')&&avail>=99.9&&cacheT&&!/replica|cluster|multi|zone|failover|standby|replicat/i.test(cacheT))out.push(s.userType+': sessions are kept on the server, but the cache design ('+cacheT.slice(0,60)+'…) shows no failover while the availability target is '+avail+'%.');
  const u=ut[dpKey(s.userType)];
  if(u&&u.staff&&s.mfa&&s.mfa!=='required')out.push(s.userType+' are staff, but multi-factor sign-in is '+dpLabel(SP_MFA,s.mfa).toLowerCase()+'.');
  if(u&&u.staff&&s.rememberMe&&!/^\s*(no|never|not)/i.test(s.rememberMe))out.push(s.userType+' are staff, but “remember me” is allowed: '+s.rememberMe);}
 for(const c of spCookies().rules){
  if(c.category!=='necessary'&&c.consent==='not-required')out.push(dpLabel(SP_CATS,c.category)+' cookies are set without consent'+(strict?' while '+regs.filter(r=>/gdpr|personal data|dpdp|privacy|eprivacy|lgpd|ccpa/i.test(r)).join(', ')+' applies':'')+'.');
  if(c.category==='necessary'&&c.attributes&&!/secure/i.test(c.attributes))out.push('Strictly necessary cookies are missing the Secure attribute.');}
 return out;
}

function spCookieTable(rules,editable){
 if(!rules.length)return h('p',{class:'muted',text:'No cookie rules yet.'});
 if(!editable)return h('div',{class:'tp-scroll'},h('table',{class:'tp-table sb-table'},[h('thead',{},h('tr',{},['Category','Purpose','Consent','Attributes','Lifetime','Domain and path'].map(t=>h('th',{text:t})))),
  h('tbody',{},rules.map(c=>h('tr',{},[h('th',{scope:'row',text:dpLabel(SP_CATS,c.category)}),h('td',{},[h('div',{text:c.purpose||'—'}),c.why?h('small',{class:'tp-why',text:c.why}):null]),h('td',{text:c.consent==='required'?'Ask first':'Not needed'}),h('td',{text:c.attributes||'—'}),h('td',{text:c.lifetime||'—'}),h('td',{text:c.domainPath||'—'})])))]));
 return h('div',{class:'val-list'},rules.map(c=>h('article',{class:'val-row'},[h('b',{text:dpLabel(SP_CATS,c.category)}),h('div',{class:'val-grid'},[
  dpField('Purpose',c.purpose,v=>c.purpose=v),dpSel('Consent',c.consent,[['required','Ask first'],['not-required','Not needed (strictly necessary only)']],v=>c.consent=v),
  dpField('Attributes',c.attributes,v=>c.attributes=v,'Secure; HttpOnly; SameSite=Lax'),dpField('Lifetime',c.lifetime,v=>c.lifetime=v),dpField('Domain and path',c.domainPath,v=>c.domainPath=v)])])));
}
function spSessionTable(rules,editable){
 if(!rules.length)return h('p',{class:'muted',text:'No session rules yet.'});
 const F=[['Channels','channels'],['Idle timeout','idleTimeout'],['Absolute lifetime','absoluteLifetime'],['Sign in again for','reauth'],['Sign out everywhere','signOutEverywhere'],['New session id (rotation)','rotation'],['Sessions at once','concurrent'],['Remember me','rememberMe'],['Cache or region outage','outage']];
 if(!editable)return h('div',{class:'val-list'},rules.map(s=>h('article',{class:'val-row'},[h('div',{class:'val-head'},[h('b',{text:s.userType}),h('div',{class:'val-badges'},[h('span',{class:'val-badge muted',text:dpLabel(SP_STORES,s.store)}),h('span',{class:'val-badge '+(s.mfa==='required'?'ok':'muted'),text:'MFA: '+dpLabel(SP_MFA,s.mfa)})])]),
  h('div',{class:'tp-scroll'},h('table',{class:'tp-table sb-table'},h('tbody',{},F.filter(([,k])=>s[k]).map(([l,k])=>h('tr',{},[h('th',{scope:'row',text:l}),h('td',{text:s[k]})]))))),s.why?h('small',{class:'tp-why',text:s.why}):null])));
 return h('div',{class:'val-list'},rules.map(s=>h('article',{class:'val-row'},[h('b',{text:s.userType}),h('div',{class:'val-grid'},[
  dpSel('Where sessions are kept',s.store,SP_STORES,v=>s.store=v),dpSel('Multi-factor sign-in',s.mfa,SP_MFA,v=>s.mfa=v),...F.map(([l,k])=>dpField(l,s[k],v=>s[k]=v))])])));
}

function sessionPolicyTab(){
 const S=spStore(),ai=S.ai,stale=spStaleReasons(),conf=spConflicts(),users=spUserTypes(),C=spCookies(),X=spSessions();
 const addIn=h('input',{class:'field',type:'text',placeholder:'e.g. Delivery rider',maxlength:'60','aria-label':'New user type'});
 const staffIn=h('input',{type:'checkbox','aria-label':'Staff'});
 const add=()=>{const v=addIn.value.trim();if(!v)return;delete S.removed[dpKey(v)];S.userTypes.push({name:v,staff:staffIn.checked});logInput('sessionpolicy:add-user-type',null,v);persist();render();};
 const block=(kind,title,state,table,who)=>h('section',{class:'summary-section'},[h('h2',{text:title}),
  h('p',{class:'muted',text:state.accepted?'Accepted by '+S[kind].byName+(S[kind].role?' ('+S[kind].role+')':'')+' on '+new Date(S[kind].at).toLocaleDateString()+'. Edit below; changes are saved as you type.':state.rules.length?'Draft from the agent: not policy until '+who+' accepts it.':'No draft yet: ask the agent, or write it yourself.'}),
  spOwnerNote(kind),table(state.rules,state.accepted),
  h('div',{class:'cost-actions'},[state.accepted?button('Back to draft',()=>{delete S[kind];logInput('sessionpolicy:unaccept:'+kind,'accepted',null);persist();render();},'btn small')
   :button(state.rules.length?'Accept as policy':'Write it myself',()=>spAccept(kind),'btn small primary')])]);
 return h('div',{},[
  h('section',{class:'summary-section'},[h('h2',{text:'Session policy agent'}),
   h('p',{class:'muted',text:'Asks Claude on the project backend to draft the cookie and session policy. Sends, once per press: the platform decision, the availability target and regulations, the identity, cache, security and recovery design, the user types (marked staff or not) and the channels. No records, no names of people, no secrets.'}),
   h('div',{class:'cost-actions'},[h('button',{type:'button',class:'btn small primary',disabled:SP_UI.busy?'':null,onclick:runSessionPolicyAgent,text:SP_UI.busy?'Drafting…':ai?'Ask again':'Ask the Session policy agent'})]),
   SP_UI.message?h('p',{class:'input-status open',text:SP_UI.message}):null,
   ai?h('div',{class:'summary-callout '+(stale.length?'stale':'approved')},[h('b',{text:stale.length?'Draft out of date: '+stale.join('; ')+'. Ask again.':'Draft from '+(ai.model||'Claude')+', '+new Date(ai.at).toLocaleString()}),ai.summary?h('p',{text:ai.summary}):null,
    ai.dropped?(typeof dpDropNotes==='function'?dpDropNotes(ai):h('small',{class:'muted',text:ai.dropped+' dropped by the checks.'})):null]):null]),
  h('section',{class:'summary-section'},[h('h2',{text:'User types and channels'}),
   h('ul',{class:'scan-list'},users.map(u=>h('li',{},[h('b',{text:u.name+(u.staff?' · staff':'')}),h('small',{text:u.source}),button('Remove',()=>{S.removed[dpKey(u.name)]=true;S.userTypes=S.userTypes.filter(x=>dpKey(x.name)!==dpKey(u.name));persist();render();},'linkbtn')]))),
   h('div',{class:'people-add'},[addIn,h('label',{class:'val-check'},[staffIn,' Staff']),button('Add user type',add,'btn small')]),
   h('p',{class:'muted',text:'Channels from the brief: '+spChannels().join(', ')+'.'})]),
  conf.length||(ai&&(ai.conflicts||[]).length)?h('section',{class:'summary-section'},[h('h2',{text:'Conflicts'}),h('ul',{class:'plain-list'},[...conf.map(t=>h('li',{class:'warn-text',text:t+' (found by Pathfinder)'})),...((ai&&ai.conflicts)||[]).map(t=>h('li',{text:t+' (from the agent)'}))])]):null,
  ai&&(ai.questions||[]).length?h('section',{class:'summary-section'},[h('h2',{text:'Questions for the security and privacy owners'}),h('ul',{class:'plain-list'},ai.questions.map(q=>h('li',{text:q})))]):null,
  block('cookies','Cookie policy',C,spCookieTable,'the privacy or data owner'),
  block('sessions','Session policy',X,spSessionTable,'the security owner')]);
}

function sessionPolicyExport(){
 const S=spStore(),C=spCookies(),X=spSessions(),acc=k=>S[k]?{by:S[k].byName,role:S[k].role,at:S[k].at}:null;
 return {cookies:{status:C.accepted?'accepted':C.rules.length?'draft':'none',accepted:acc('cookies'),rules:C.rules},
  sessions:{status:X.accepted?'accepted':X.rules.length?'draft':'none',accepted:acc('sessions'),rules:X.rules},
  userTypes:spUserTypes().map(u=>({name:u.name,staff:u.staff})),channels:spChannels(),
  conflicts:spConflicts(),agentConflicts:(S.ai&&S.ai.conflicts)||[],questions:(S.ai&&S.ai.questions)||[],agent:S.ai?{model:S.ai.model||'',at:S.ai.at,stale:spStaleReasons()}:null};
}
