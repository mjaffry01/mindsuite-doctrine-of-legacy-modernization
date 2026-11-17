/* MindSuite web app · documents & diagrams agent team (built with teams/shared/team-panel.js).
   Scope = the "Agent detects" line of the Documents & diagrams card: business rules, processes,
   glossary and intended architecture, read by the agent with citations. */
const DOCUMENTS_TEAM=makeTeamPanel({
 team:'documents',key:'doc',name:'Documents & diagrams',people:'your business analysts and architects',category:'business',source:'documents',
 labels:{orch:'Documents orchestrator',drd:'Document reader',dfr:'Freshness & coverage',drs:'Rule statements',dgl:'Glossary',dpr:'Processes',darc:'Intended architecture',drul:'Business rules',dmea:'Business story & conflicts',dass:'Evidence value',ver:'Verifier',rep:'Report writer',people:'Your experts'},
 accept:'.docx,.pptx,.vsdx,.pdf,.html,.htm,.md,.txt,.drawio,.xml,.doc,.ppt,.vsd',exportRe:/\.(docx?|pptx?|vsdx?|pdf|html?|md|txt|drawio|xml)$/i,maxBytes:20*1024*1024,
 consent:'Allow the AI agents to send the picked document passages and extracted facts to Claude (Anthropic)',
 privacyOn:'With this on, the business-rules agent sends the document sections the orchestrator picked (lines that may hold a password removed), the story agent sends extracted facts only, and the verifier and report writer send findings. Everything sent is listed in the result.',
 privacyOff:'Nothing leaves the backend machine: 10 rule-based agents run; rule statements are found by pattern, and the business rules, capabilities and contradictions are listed as gaps.',
 give:'Give the documents themselves: Word (.docx), PowerPoint (.pptx), PDF with a text layer, Confluence pages exported as HTML, Markdown or text, and diagrams from draw.io (.drawio) or Visio (.vsdx). Old .doc/.vsd files and scanned PDFs cannot be read: save them as .docx or a PDF with text. Up to 20 MB per file.',
 intro:'12 agents read your documents and diagrams, in the same 9 steps as the other teams, and find what the Documents & diagrams card promises: business rules, processes, glossary and intended architecture, each with the exact line it comes from. They also say how current each document is and where documents contradict each other. If you allow it, Claude rewrites the rule passages as plain business rules and describes the capabilities.',
 skippedText:'Business rules, capabilities and contradictions were not written by the AI agents',
 keepData:['path','kind','words','date','version','term','meaning','steps','document','section','from','to','label','tech','level','severity','missing'],
 metrics:(A,src,F)=>[['Documents',src.documents??0],['Diagrams',src.diagrams??0],['Not readable',src.notRead??0],['Rule statements',F('rule-statement').length+F('business-rule').length],['Terms',F('doc-term').length]],
 sections:(A,F,list)=>{const vp=F('validation-plan')[0],cov=F('topic-coverage')[0],pur=F('purpose')[0];return [
  pur?h('div',{class:'validate-card'},[h('b',{text:'What the documented system is for'}),h('p',{text:pur.detail})]):null,
  cov?h('div',{class:'validate-card'},[h('b',{text:cov.title}),h('p',{text:cov.detail})]):null,
  vp?h('div',{class:'validate-card'},[h('b',{text:vp.title}),h('pre',{class:'agent-pre',text:vp.detail})]):null,
  list('Business rules',F('business-rule').concat(F('rule-statement')),true),
  list('Processes',F('doc-process').concat(F('diagram-flow')),true),
  list('Glossary and conflicts',F('term-conflict').concat(F('doc-conflict'),F('doc-term')),true),
  list('Intended architecture',F('intended-link').concat(F('intended-component'),F('documented-tech')),false),
  list('Documents and how far to trust them',F('evidence-value').concat(F('old-document'),F('draft-document'),F('secret')),false),
  list('Capabilities',F('doc-capability'),false)];},
 importTo:(A,{R,put,F,by})=>{
  put('rules',[...F('rule-statement').map(f=>R(f.id,by(f)+' · stated in a document',{name:f.title,context:'',source:'document'})),...F('business-rule').map(f=>R(f.id,by(f)+' · '+(f.status==='accepted'?'verified':'needs an expert'),{name:f.title+(f.detail?': '+f.detail:''),context:'',source:'document'},'inferred-code'))]);
  put('terms',F('doc-term').map(f=>{const conflict=F('term-conflict').some(c=>c.title.startsWith('"'+f.title+'"'));return R(f.id,by(f),{name:f.title,meaning:(f.data&&f.data.meaning)||f.detail,variants:'',context:'',conflict:conflict?'yes':'no'});}));
  put('events',F('doc-process').map(f=>R(f.id,by(f),{name:f.title,context:'',consumers:[],trigger:'Documented steps: '+((f.data&&f.data.steps)||[]).join(' → ')})));
  put('components',F('intended-component').slice(0,60).map(f=>R(f.id,by(f),{name:f.title,kind:'module',context:'',tech:'documented (intended)',size:'',schedule:''})));
  if(!M.inventory.contexts.some(c=>c.origin!=='scan'&&c.origin!=='agent'))put('contexts',F('doc-capability').slice(0,12).map(f=>R('ctx-'+slug(f.title),'documents · '+f.id,{name:f.title,description:f.detail,owner:'',knowledge:'',volumes:''},'inferred-code')));},
 risks:{'term-conflict':['data','Ask the business which definition is right'],'doc-conflict':['data','Resolve the contradiction with the document owners'],'old-document':['knowledge','Confirm what is still current with the authors'],'draft-document':['knowledge','Confirm the unfinished parts'],secret:['security','Remove from shared documents and rotate']}});
