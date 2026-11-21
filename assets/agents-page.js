/* MindSuite web app · AI agent teams: who they are and how they communicate (screen 'agents')
   Lists every agent team and agent the software uses, what each does, which model it uses (or none), and
   how they talk: browser ⇄ project backend ⇄ Claude, and inside a team run through the evidence store with
   typed messages (task, finding, handoff, verification, question, log) streamed to the browser.
   Team members are read live from the project backend (GET /api/agents?team=…; no model call, no cost);
   if the backend is not reachable the built-in list below is shown and the page says so. */

const AGENT_DISCOVERY=[
 ['code','Code','Source code: structure, data access, interfaces, security, quality, business rules'],
 ['database','Database','Schema, size, stored code, jobs, privacy, portability, business rules in the data'],
 ['infrastructure','Infrastructure','Servers, lifecycle and support, capacity, Kubernetes, cloud, infrastructure as code'],
 ['messaging','Messaging and cache','Brokers, queues, flows, backlog, delivery guarantees, caches'],
 ['identity','Identity','Directory, accounts, privileged access, MFA, roles, certificates, protocols'],
 ['network','Network edge','Load balancers, gateways, DNS and CDN, TLS, partner links, firewall openings'],
 ['telemetry','Telemetry','Load and peaks, response times, errors, unused features, monitoring gaps'],
 ['batch','Batch and schedules','Job streams, dependencies, batch window, failures and reruns'],
 ['incidents','Incidents and changes','Hotspots, recurring problems, restore times, failed changes'],
 ['documents','Documents','Freshness, rule statements, glossary, processes, intended architecture']];
const AGENT_DESIGN=[
 ['Formfiller agent','Product brief','POST /api/brief/formfill','Balanced model','Predicts the next unanswered brief field from the answers so far; every suggestion quotes the brief; retries once; never invents numbers or vendors.'],
 ['Capability agent','First-release scope','POST /api/brief/capabilities','Balanced model','Drafts capabilities from the brief, each quoting the line it came from; you add or dismiss each one.'],
 ['Approach suggester','Design options / Target decisions','POST /api/decide/suggest','Balanced model','Suggests a route from claims that people verified; a person decides.'],
 ['Target options team','Design options / Target decisions','POST /api/decide/options','Scouts: balanced; Architect, Challenger: strong','Option scouts (one per area, in parallel) → Verifier (rules: drops uncited options, checks product names) → Rules guard (your technology rules) → Architect (2–4 blueprints) → Challenger (tests each blueprint).'],
 ['Solution designer','Design options / Target decisions','part of the options team','Strong model','Details one blueprint: database and caching strategy, patterns, north–south and east–west traffic, cross-cutting components, starting sizes.'],
 ['Diagram agent','Architecture (HLD), component diagram','POST /api/design/diagram','Strong model','Reads the whole design (platform decision, plan layers, capabilities and routes, owners of shared concerns, quality targets, the rule-based draft) and returns the component diagram, components the design is missing and conflicts. Checked before drawing: valid zones and ids, every flow connects known components, no product named that the design does not mention; one retry.'],
 ['Decision reviewer','Design options / Target decisions','POST /api/decide/review','Strong model','Challenges one decision before it is recorded, from the approach, choices, reasons and verified claims only.']];
const AGENT_LOCAL=[
 ['Decision advisor','Scores routes against the evidence, your rules and priorities; confidence; one deciding question'],
 ['Platform decision suggester','Suggests the shared platform (tier, cloud, hosting, architecture, database); the same recommendation, with its reasons, points and the options it beat, appears on Rules & people → Technology rules'],
 ['Cloud service rules','Your Standard / Preferred / Needs approval / Excluded rules for about 250 individual services on Azure, AWS, Google Cloud, IBM Cloud, Oracle Cloud and private platforms; the plan uses a Standard service first, then Preferred, replaces an Excluded one, and a Standard compute service sets the hosting model'],
 ['Load and data calculators','Peak load and 3-year data volume from users, predicted from the brief'],
 ['Technical plan and System blueprint','Layer-by-layer plan (hosting, traffic, compute and server types, data, messaging, files, search, IoT, AI, payments, sign-in, permissions, security, availability, disaster recovery plan, monitoring and observability, delivery, cloud design patterns), shared platform, owners of shared concerns'],
 ['Cost estimator','Component costs and cost per tier from the sizing'],
 ['AI feature suggester','Spots where AI would help (search, assistant, extraction, anomaly detection, reporting questions, content tagging) and names the model service, vector store and guardrails per cloud'],
 ['Shared-concern owners','Decides who owns payments, customer accounts, media, search and notifications so nothing is built twice'],
 ['Assumptions register and HLD','Collects every assumption; assembles the high-level design, its architecture diagram, a detailed component diagram (private network, subnets, network and access, split monitoring, identity and delivery services, disaster recovery) and the full design JSON (schema mindsuite.design/1)']];
const AGENT_UI={loaded:false,loading:false,teams:{},error:''};

async function agentsLoad(){
 if(AGENT_UI.loading)return;AGENT_UI.loading=true;AGENT_UI.error='';
 try{
  for(const [id] of AGENT_DISCOVERY){AGENT_UI.teams[id]=await agentFetch('/api/agents?team='+encodeURIComponent(id));}
  AGENT_UI.loaded=true;
 }catch(e){AGENT_UI.error=e.message||String(e);}
 finally{AGENT_UI.loading=false;if(typeof screen!=='undefined'&&screen==='agents')render();}
}

function agentsFlowSvg(){
 const box=(x,y,w,h,t,s,c)=>'<g class="ag-b '+(c||'')+'"><rect x="'+x+'" y="'+y+'" width="'+w+'" height="'+h+'" rx="8"/><text x="'+(x+w/2)+'" y="'+(y+(s?20:h/2+4))+'" class="t">'+t+'</text>'+(s?'<text x="'+(x+w/2)+'" y="'+(y+37)+'" class="s">'+s+'</text>':'')+'</g>';
 const ar=(x1,y1,x2,y2,l,dash)=>'<line x1="'+x1+'" y1="'+y1+'" x2="'+x2+'" y2="'+y2+'" class="ag-e'+(dash?' d':'')+'" marker-end="url(#agA)"/>'+(l?'<text x="'+((x1+x2)/2)+'" y="'+((y1+y2)/2-6)+'" class="l">'+l+'</text>':'');
 const p=[];
 // Row 1: browser, backend, Claude
 p.push(box(20,20,250,50,'Web app (your browser)','project data stays here','web'));
 p.push(box(420,20,250,50,'Project backend','your machine or server','be'));
 p.push(box(820,20,250,50,'Claude (Anthropic API)','only with a key and your consent','ai'));
 p.push(ar(270,38,418,38,'request (consent per request)'));p.push(ar(418,56,272,56,'results, live event stream',true));
 p.push(ar(670,38,818,38,'metadata, excerpts'));p.push(ar(818,56,672,56,'structured answers',true));
 // Row 2: one discovery team run
 p.push('<text x="20" y="118" class="h">Inside one discovery team run</text>');
 p.push(box(20,135,170,50,'Orchestrator','splits the work','co'));
 p.push(box(250,135,170,50,'Tool agents','read exports, no model','tool'));
 p.push(box(480,135,190,50,'Evidence store','every message is saved','store'));
 p.push(box(730,135,170,50,'Model agents','business meaning','model'));
 p.push(box(950,135,140,50,'Verifier','different model','ver'));
 p.push(box(480,230,190,50,'Report writer','cited report','model'));
 p.push(box(250,230,170,50,'Questions','for people to answer','q'));
 p.push(ar(190,152,248,152,'task'));p.push(ar(420,152,478,152,'finding'));p.push(ar(670,152,728,152,'handoff'));
 p.push(ar(728,170,672,170,'finding',true));p.push(ar(900,152,948,152,'check'));
 p.push('<path d="M1020 185 V205 H640 V188" class="ag-e d" fill="none" marker-end="url(#agA)"/><text x="830" y="219" class="l">verification (kept or dropped)</text>');
 p.push(ar(575,185,575,228,''));p.push(ar(478,255,422,255,'question'));
 const css='.ag-b rect{fill:#f7f9ff;stroke:#b8c6e3}.ag-b.web rect{fill:#fff;stroke:#7c8fb8}.ag-b.be rect{fill:#eef6ff;stroke:#8fb0dd}.ag-b.ai rect{fill:#f6f4ff;stroke:#7a6ae0}.ag-b.store rect{fill:#f3faf6;stroke:#9fd3b5}.ag-b.ver rect{fill:#fff4f4;stroke:#e3a1a1}.ag-b.q rect{fill:#fffaf2;stroke:#e5c48a}.ag-b.co rect{fill:#eef0ff;stroke:#9aa4e6}'+
  '.t{font:600 12px system-ui,sans-serif;text-anchor:middle;fill:#1d2433}.s{font:11px system-ui,sans-serif;text-anchor:middle;fill:#5b6478}.h{font:700 13px system-ui,sans-serif;fill:#3f33a8}.l{font:10.5px system-ui,sans-serif;text-anchor:middle;fill:#5b6478}.ag-e{stroke:#8091b3;stroke-width:1.3}.ag-e.d{stroke-dasharray:5 4}';
 return '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1110 300" class="hld-svg" role="img" aria-label="How the agents communicate"><style>'+css+'</style><defs><marker id="agA" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="7" markerHeight="7" orient="auto"><path d="M0,0 L10,5 L0,10 z" fill="#8091b3"/></marker></defs>'+p.join('')+'</svg>';
}

function agentsPage(){
 if(!AGENT_UI.loaded&&!AGENT_UI.loading&&!AGENT_UI.error)setTimeout(agentsLoad,0);
 const holder=h('div',{class:'hld-diagram'});holder.innerHTML=agentsFlowSvg();
 const kindLabel={tool:'reads exports, no model',model:'model','tool+model':'tool + model',coordinator:'coordinator'};
 const live=AGENT_UI.loaded;
 return h('div',{},[
  pageHeader('ⓘ / AI AGENT TEAMS','AI agent teams and how they communicate','Every agent the software uses, what it does, which model it uses, and how messages flow. Agents suggest and check; people decide.',[]),
  h('section',{class:'summary-section'},[h('h2',{text:'How they communicate'}),holder,
   h('ul',{class:'plain-list'},[
    h('li',{text:'Browser → project backend: one request per action you take. Requests that use Claude say so and need your consent each time.'}),
    h('li',{text:'Backend → Claude: metadata and short excerpts only (never business data rows, names of people or secrets). Answers come back as structured JSON and are checked before you see them.'}),
    h('li',{text:'Inside a team run, agents never call each other directly: they post typed messages to the evidence store — task, finding, handoff, verification, question, log — and the browser follows them live.'}),
    h('li',{text:'The Verifier uses a different model from the agents it checks; findings without a citation are dropped; product names are checked against a catalogue.'})])]),
  h('section',{class:'summary-section'},[h('h2',{text:'Discovery teams (existing systems)'}),
   h('p',{class:'muted',text:live?'Members read live from the project backend.':AGENT_UI.loading?'Reading the members from the project backend…':'The project backend is not reachable'+(AGENT_UI.error?' ('+AGENT_UI.error+')':'')+'; showing what each team covers.'}),
   h('div',{class:'ag-grid'},AGENT_DISCOVERY.map(([id,name,covers])=>{const a=AGENT_UI.teams[id];
    return h('div',{class:'ag-team'},[h('b',{text:name+' team'}),h('small',{class:'muted',text:covers}),
     a?h('ol',{class:'ag-members'},a.map(x=>h('li',{},[h('span',{text:x.name}),h('small',{class:'ag-kind '+String(x.kind).replace('+','-'),text:x.model?x.model.replace(/^claude-/,''):kindLabel[x.kind]||x.kind}),h('small',{class:'muted ag-does',text:x.does})]))):null]);}))]),
  h('section',{class:'summary-section'},[h('h2',{text:'Design agents (new products and target design)'}),
   h('table',{class:'tp-table sb-table'},[h('thead',{},h('tr',{},['Agent','Where','Endpoint','Model','What it does'].map(t=>h('th',{text:t})))),
    h('tbody',{},AGENT_DESIGN.map(([n,w,e,m,d])=>h('tr',{},[h('th',{scope:'row',text:n}),h('td',{text:w}),h('td',{},[h('code',{text:e})]),h('td',{text:m}),h('td',{text:d})])))])]),
  h('section',{class:'summary-section'},[h('h2',{text:'Rule engines in the browser (no AI)'}),
   h('p',{class:'muted',text:'These run locally from your answers, instantly and without sending anything.'}),
   h('ul',{class:'plain-list'},AGENT_LOCAL.map(([n,d])=>h('li',{},[h('b',{text:n+': '}),d])))])]);
}

(function agStyle(){
 if(document.getElementById('ag-style'))return;
 const css=`.ag-grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(320px,1fr));gap:12px}
.ag-team{border:1px solid #e3e8f2;border-radius:10px;padding:10px 12px;background:#fff}.ag-team>small{display:block;margin:2px 0 6px}
.ag-members{margin:0;padding-left:18px;font-size:12.5px}.ag-members li{margin:3px 0}.ag-members span{font-weight:600;margin-right:6px}
.ag-kind{font-size:10.5px;padding:1px 6px;border-radius:8px;background:#eef0f5;color:#5b6478}.ag-kind.model,.ag-kind.tool-model{background:#ece9ff;color:#5a46c8}
.ag-does{display:block;font-size:11.5px}`;
 document.head.append(Object.assign(document.createElement('style'),{id:'ag-style',textContent:css}));
})();
