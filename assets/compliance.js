/* MindSuite web app · compliance catalogue and guide
   Explains regulations and standards in plain words (what it is, when it applies, what it means for the
   design, official source), suggests the ones that are likely for this project from its facts (rule-based,
   the owner confirms), and gives the agents the design implications as facts.
   Stored as before in M.scope.regulation = {list:[ids], other:''}; older ids (gdpr, pci, hipaa, sox, local,
   sector, none, unknown) keep their meaning. Every source link was checked to open the official page on
   2026-11-21. Loaded before the main script: REGULATIONS, regulationBlock() and the strict-regulation check in
   index.html call complianceRegulations(), compliancePicker() and complianceStrict(). */

const COMPLIANCE_GROUPS=[['privacy','Privacy and personal data'],['health','Health and life sciences'],['finance','Finance and payments'],
 ['security','Security standards and certifications'],['public','Government and defence'],['access','Accessibility'],['ai','Artificial intelligence'],['other','Other']];

// [id, name, group, region, what it is, applies when, what it means for the design (list), official source, strict]
// strict = regulated data that normally needs compliant regions, contracts and controls from any cloud provider.
const COMPLIANCE=[
 ['gdpr','GDPR','privacy','European Union / EEA','The EU General Data Protection Regulation: rules for handling personal data.','You process personal data of people in the EU, or you operate in the EU.',
  ['Record a lawful basis for each use of personal data','Data protection impact assessment for high-risk processing','Transfers outside the EEA need approved safeguards','Report personal data breaches to the authority within 72 hours','Support access, correction and erasure requests','Data processing agreements with every vendor that handles the data'],'https://eur-lex.europa.eu/eli/reg/2016/679/oj',false],
 ['uk-gdpr','UK GDPR and Data Protection Act 2018','privacy','United Kingdom','The UK version of the GDPR, overseen by the ICO.','You process personal data of people in the UK.',
  ['Same core duties as the GDPR','Transfers out of the UK need approved safeguards','Report breaches to the ICO within 72 hours'],'https://ico.org.uk/for-organisations/uk-gdpr-guidance-and-resources/',false],
 ['dpdp','Digital Personal Data Protection Act 2023','privacy','India','India\'s law for processing digital personal data, under the Data Protection Board.','You process digital personal data in India, or of people in India for offering goods or services.',
  ['Notice and consent before processing, with easy withdrawal','Reasonable security safeguards and breach reporting to the Board and affected people','Erase data when its purpose ends','Check government restrictions on transfers to specific countries'],'https://www.meity.gov.in/data-protection-framework',false],
 ['ccpa','CCPA / CPRA','privacy','California, USA','California\'s consumer privacy law, as amended by the CPRA.','You do business in California and meet its size or data thresholds.',
  ['Notice at collection and a privacy policy','Right to know, delete, correct and opt out of sale or sharing','Limits on using sensitive personal information','Contracts with service providers'],'https://oag.ca.gov/privacy/ccpa',false],
 ['lgpd','LGPD','privacy','Brazil','Brazil\'s General Personal Data Protection Law, overseen by the ANPD.','You process personal data in Brazil or of people in Brazil.',
  ['Legal basis for each processing purpose','Data subject rights and a named data protection officer','Rules for international transfers','Incident reporting to the ANPD'],'https://www.gov.br/anpd/pt-br',false],
 ['pipeda','PIPEDA','privacy','Canada','Canada\'s federal private-sector privacy law.','Private-sector organizations in Canada handling personal information in commercial activity.',
  ['Consent and limited collection','Safeguards proportionate to sensitivity','Report breaches that create a real risk of significant harm'],'https://www.priv.gc.ca/en/privacy-topics/privacy-laws-in-canada/the-personal-information-protection-and-electronic-documents-act-pipeda/',false],
 ['pdpa-sg','PDPA (Singapore)','privacy','Singapore','Singapore\'s Personal Data Protection Act.','Organizations collecting, using or disclosing personal data in Singapore.',
  ['Consent, purpose limitation and notification','Protection and retention limits','Mandatory breach notification for significant breaches','Transfer limitation for data leaving Singapore'],'https://www.pdpc.gov.sg/overview-of-pdpa/the-legislation/personal-data-protection-act',false],
 ['uae-pdpl','UAE data protection laws','privacy','United Arab Emirates','The UAE federal personal data protection law and free-zone laws (for example DIFC, ADGM).','You process personal data in the UAE; free zones have their own regimes.',
  ['Check which regime applies: federal or a free zone','Consent and data subject rights','Rules for cross-border transfers'],'https://u.ae/en/about-the-uae/digital-uae/data/data-protection-laws',false],
 ['ksa-pdpl','PDPL (Saudi Arabia)','privacy','Saudi Arabia','Saudi Arabia\'s Personal Data Protection Law, overseen by SDAIA.','You process personal data of people in Saudi Arabia.',
  ['Purpose limitation and consent','Restrictions on transferring data outside the Kingdom','Breach notification to the regulator'],'https://sdaia.gov.sa/en/SDAIA/about/Pages/RegulationsAndPolicies.aspx',false],
 ['au-privacy','Privacy Act 1988 (Australia)','privacy','Australia','Australia\'s Privacy Act and the Australian Privacy Principles, overseen by the OAIC.','Most Australian government agencies and businesses above the turnover threshold.',
  ['Australian Privacy Principles for collection, use and disclosure','Notifiable data breaches scheme','Accountability for overseas disclosure'],'https://www.oaic.gov.au/privacy/privacy-legislation/the-privacy-act',false],
 ['coppa','COPPA','privacy','USA','The Children\'s Online Privacy Protection Rule.','Online services directed to children under 13, or that knowingly collect their data.',
  ['Verifiable parental consent before collecting children\'s data','Clear privacy notice to parents','Data retention limits and security'],'https://www.ftc.gov/business-guidance/privacy-security/childrens-privacy',false],
 ['ferpa','FERPA','privacy','USA','The Family Educational Rights and Privacy Act for student education records.','Schools receiving US Department of Education funds, and vendors handling their records.',
  ['Consent before disclosing education records, with set exceptions','Access controls on student records','Vendor agreements for school officials'],'https://studentprivacy.ed.gov/ferpa',false],
 ['hipaa','HIPAA','health','USA','US rules for protected health information (privacy, security and breach notification).','Health plans, providers, clearinghouses and their business associates in the US.',
  ['Business associate agreement (BAA) with every vendor that touches the data, including the cloud','Encryption, access control and audit logging for health data','Breach notification to individuals and HHS','Use only services the cloud provider lists as HIPAA eligible'],'https://www.hhs.gov/hipaa/index.html',true],
 ['hitrust','HITRUST CSF','health','Global (common in US health care)','A certifiable security framework often required by health-care customers.','Your customers or partners ask for HITRUST certification.',
  ['Map controls to the HITRUST framework','Plan time for an assessment and certification'],'https://hitrustalliance.net/',false],
 ['part11','21 CFR Part 11 / GxP','health','USA (FDA-regulated)','FDA rules for electronic records and signatures in regulated life-sciences work.','Records required by FDA rules are kept or signed electronically.',
  ['Validated systems with audit trails','Controls on electronic signatures','Record retention and copies for inspection'],'https://www.fda.gov/regulatory-information/search-fda-guidance-documents/part-11-electronic-records-electronic-signatures-scope-and-application',true],
 ['pci','PCI DSS','finance','Global','The Payment Card Industry Data Security Standard.','You store, process or transmit payment card data, or can affect its security.',
  ['Keep card data out of the system where possible: use a tokenizing payment provider','Never store sensitive authentication data after authorization','Network segmentation, encryption and strict access to the card environment','Quarterly scans and an annual assessment'],'https://www.pcisecuritystandards.org/',true],
 ['sox','SOX','finance','USA (listed companies)','The Sarbanes-Oxley Act: controls over financial reporting.','The system supports financial reporting of a company listed in the US.',
  ['Segregation of duties and change control','Audit trails for financial data and configuration','Evidence of tested internal controls'],'https://www.govinfo.gov/app/details/PLAW-107publ204',false],
 ['glba','GLBA','finance','USA','The Gramm-Leach-Bliley Act privacy and Safeguards rules for financial institutions.','Businesses providing financial products or services to consumers in the US.',
  ['Written information security program','Encryption and multi-factor authentication for customer data','Report qualifying breaches to the FTC within 30 days'],'https://www.ftc.gov/business-guidance/privacy-security/gramm-leach-bliley-act',true],
 ['dora','DORA','finance','European Union','The Digital Operational Resilience Act for financial entities.','Banks, insurers, investment firms and other EU financial entities, and their critical ICT providers.',
  ['ICT risk management and resilience testing','Report major ICT incidents to the regulator','Register and contract terms for ICT third parties, with exit plans'],'https://eur-lex.europa.eu/eli/reg/2022/2554/oj',true],
 ['psd2','PSD2','finance','European Union','The revised Payment Services Directive.','Payment service providers and account-access services in the EU.',
  ['Strong customer authentication','Secure APIs for account access','Incident reporting'],'https://eur-lex.europa.eu/eli/dir/2015/2366/oj',true],
 ['rbi','RBI directions (India)','finance','India','Reserve Bank of India directions for regulated entities, including payment data storage and IT outsourcing.','Banks, NBFCs and payment system operators in India, and their technology providers.',
  ['Payment system data stored only in India','IT and cloud outsourcing rules: due diligence, audit rights, exit plans','Cyber security framework and incident reporting to the RBI'],'https://www.rbi.org.in/',true],
 ['sec17a4','SEC Rule 17a-4','finance','USA (broker-dealers)','SEC recordkeeping rule for broker-dealers.','The system keeps records a US broker-dealer must retain.',
  ['Tamper-proof (non-rewriteable or audit-trailed) record storage','Set retention periods and quick retrieval for regulators'],'https://www.ecfr.gov/current/title-17/section-240.17a-4',true],
 ['apra','APRA CPS 234','finance','Australia','APRA\'s information security standard for banks, insurers and superannuation.','APRA-regulated entities and the third parties that manage their information assets.',
  ['Security capability proportionate to threats','Systematic testing of controls','Notify APRA of material incidents'],'https://www.apra.gov.au/standards/cps-234',true],
 ['iso27001','ISO/IEC 27001','security','Global','The international standard for an information security management system.','Customers or contracts ask for it, or you want a certifiable security baseline.',
  ['Risk assessment and a statement of applicability','Documented controls and internal audits','Plan for certification audits'],'https://www.iso.org/standard/27001',false],
 ['soc2','SOC 2','security','Global (common in the US)','An independent report on controls for security, availability, processing integrity, confidentiality and privacy.','You sell a service, especially SaaS, to customers who require a SOC 2 report.',
  ['Design controls against the trust services criteria','Collect evidence continuously for the audit period','Vendor and change management'],'https://www.aicpa-cima.com/topic/audit-assurance/audit-and-assurance-greater-than-soc-2',false],
 ['nis2','NIS2','security','European Union','The EU directive on cybersecurity for essential and important entities.','Medium and large organizations in sectors it lists (energy, transport, health, digital infrastructure, public administration and more).',
  ['Cyber risk management measures and supply-chain security','Early warning of significant incidents within 24 hours','Management accountability'],'https://eur-lex.europa.eu/eli/dir/2022/2555/oj',false],
 ['cyber-essentials','Cyber Essentials','security','United Kingdom','A UK government-backed baseline security certification.','UK public-sector contracts or customers require it.',
  ['Firewalls, secure configuration, access control, malware protection, patching'],'https://www.ncsc.gov.uk/cyberessentials/overview',false],
 ['fedramp','FedRAMP','public','USA (federal)','The US federal program for authorizing cloud services.','You offer a cloud service to US federal agencies.',
  ['Use FedRAMP-authorized cloud services','Plan for an authorization package and continuous monitoring'],'https://www.fedramp.gov/',true],
 ['cmmc','CMMC','public','USA (defence)','The Cybersecurity Maturity Model Certification for the defence supply chain.','You handle federal contract or controlled unclassified information for the US Department of Defense.',
  ['Controls based on NIST SP 800-171 for the required level','Assessment before contract award'],'https://dodcio.defense.gov/CMMC/',true],
 ['itar','ITAR','public','USA','The International Traffic in Arms Regulations on defence articles and technical data.','The system stores or shares US defence technical data.',
  ['Access only by authorized US persons unless licensed','Data location and cloud services that support ITAR controls'],'https://www.ecfr.gov/current/title-22/chapter-I/subchapter-M',true],
 ['wcag','WCAG 2.2','access','Global','The Web Content Accessibility Guidelines, the reference for accessible web and app interfaces.','You publish web or mobile interfaces, especially to the public or the government.',
  ['Design and test to level AA: keyboard use, contrast, labels, captions','Include accessibility checks in every release'],'https://www.w3.org/TR/WCAG22/',false],
 ['ada','ADA','access','USA','The Americans with Disabilities Act, applied by courts and the DOJ to websites and apps.','You serve the public in the US.',
  ['Accessible web and mobile interfaces, usually measured against WCAG'],'https://www.ada.gov/',false],
 ['eaa','European Accessibility Act','access','European Union','EU accessibility requirements for listed products and services, including e-commerce and banking.','You sell covered products or services to consumers in the EU.',
  ['Accessible interfaces and information','Accessibility statement and documentation'],'https://eur-lex.europa.eu/eli/dir/2019/882/oj',false],
 ['eu-ai-act','EU AI Act','ai','European Union','The EU regulation on artificial intelligence, with duties by risk level.','You provide or use AI systems in the EU.',
  ['Classify each AI feature by risk','Transparency for people interacting with AI','For high-risk uses: risk management, data governance, logging and human oversight'],'https://eur-lex.europa.eu/eli/reg/2024/1689/oj',false],
 ['local','Other local data protection or residency law','other','','A data protection or residency rule not listed here.','A national or state law requires data to stay in a place or be handled in a set way.',
  ['Record the rule under "Other rules" and the regions it allows'],'',true],
 ['sector','Sector regulator','other','','Rules from a sector regulator (bank, energy, telecom, insurance…).','A regulator supervises your sector.',
  ['Record the regulator and its rules under "Other rules"'],'',true]];

const COMPLIANCE_BY_ID=Object.fromEntries(COMPLIANCE.map(c=>[c[0],c]));
const complianceStrict=id=>!!(COMPLIANCE_BY_ID[id]||[])[8];
// Used by index.html as REGULATIONS: [id, label] pairs, plus none and unknown.
function complianceRegulations(){
 return [...COMPLIANCE.map(c=>[c[0],c[1]+(c[3]?' ('+c[3]+')':'')]),['none','None apply'],['unknown','Not known yet']];
}

/* ---- Likely regulations from the project's own facts (rule-based; the owner confirms) ---- */
function complianceText(){
 const P=M.product||{},D=P.deploy||{},sc=M.scope||{},b=sc.brief||{};
 return [M.project,sc.component,sc.journey,b.whyDetail,b.success,P.problem,P.users,P.journeys,P.mvp,P.integrations,D.regions,(M.decide&&M.decide.rules&&M.decide.rules.region)||'',
  ...(M.inventory?M.inventory.dataStores.map(d=>d.name+' '+(d.sensitivity||'')):[])].filter(Boolean).join(' \n ');
}
function complianceSuggestions(){
 const t=complianceText(),has=re=>re.test(t),out=[],add=(id,why)=>{if(!out.some(x=>x[0]===id))out.push([id,why]);};
 const stores=(M.inventory&&M.inventory.dataStores)||[];
 const personal=stores.some(d=>['personal','financial','regulated'].includes(d.sensitivity))||has(/\b(customer|patient|student|employee|user|member)s?\b|personal data|email|phone/i);
 const purpose=(M.values||{}).purpose||'';
 if(has(/\b(EU|European Union|Europe|EEA|Germany|France|Spain|Italy|Netherlands|Ireland|Poland|Sweden)\b/))add('gdpr','The project mentions the EU or an EU country');
 if(has(/\b(UK|United Kingdom|England|Scotland|Wales|Britain)\b/))add('uk-gdpr','The project mentions the United Kingdom');
 if(has(/\b(India|Indian|Mumbai|Delhi|Bengaluru|Bangalore|Chennai|Hyderabad|Pune)\b/)){add('dpdp','The project mentions India');if(has(/payment|UPI|card|bank|NBFC|lending|loan|wallet/i)||purpose==='finance')add('rbi','India plus payments or finance');}
 if(has(/\b(California|CA residents)\b/))add('ccpa','The project mentions California');
 if(has(/\b(Brazil|Brasil)\b/))add('lgpd','The project mentions Brazil');
 if(has(/\bCanad(a|ian)\b/))add('pipeda','The project mentions Canada');
 if(has(/\bSingapore\b/))add('pdpa-sg','The project mentions Singapore');
 if(has(/\b(UAE|Dubai|Abu Dhabi|Emirates)\b/))add('uae-pdpl','The project mentions the UAE');
 if(has(/\b(Saudi|KSA|Riyadh)\b/))add('ksa-pdpl','The project mentions Saudi Arabia');
 if(has(/\bAustralia(n)?\b/)){add('au-privacy','The project mentions Australia');if(purpose==='finance'||has(/bank|insur|superannuation/i))add('apra','Australia plus banking or insurance');}
 if(has(/\b(patient|clinic|hospital|health ?care|medical|PHI|EHR|EMR)\b/i)||purpose==='health')add('hipaa','The project handles health information (check whether US rules apply)');
 if(has(/\b(card|cards|payment|checkout|PCI|POS|point of sale)\b/i))add('pci','The project mentions cards or payments');
 if(has(/\b(child|children|kids|under 13|minors)\b/i))add('coppa','The project mentions children');
 if(has(/\b(student|school|university|education)\b/i))add('ferpa','The project mentions students or schools (US education records)');
 if(has(/\b(bank|insurer|insurance|investment|broker|trading)\b/i)&&has(/\b(EU|Europe)\b/))add('dora','EU financial services');
 if(has(/\b(listed company|financial reporting|general ledger|SOX)\b/i)||purpose==='finance')add('sox','The system touches financial reporting');
 if(has(/\b(federal|government agency|public sector)\b/i)&&has(/\b(US|USA|United States)\b/))add('fedramp','US federal customers');
 if(has(/\b(defen[cs]e|DoD|military)\b/i))add('cmmc','The project mentions defence work');
 if(has(/\b(AI|machine learning|LLM|chatbot|recommendation engine)\b/))add('eu-ai-act','The project uses AI features (check if EU users are in scope)');
 if(has(/\b(SaaS|customers? (ask|require)|enterprise customers)\b/i)||((M.product||{}).deploy||{}).tenancy==='saas-multi')add('soc2','You sell a service to other businesses: customers often ask for SOC 2 or ISO 27001');
 const ch=((M.product||{}).deploy||{}).channels||[];
 if(ch.includes('web')||ch.includes('mobile')||has(/\b(public|citizens|consumers|shoppers)\b/i))add('wcag','Public web or mobile interfaces');
 if(personal&&!out.some(x=>COMPLIANCE_BY_ID[x[0]]&&COMPLIANCE_BY_ID[x[0]][2]==='privacy'))add('unknown-privacy','Personal data is involved but no country is named yet');
 return out;
}

/* ---- Picker with explanations ---- */
let complianceUI={q:'',open:''};
function complianceExplainer(id){
 const c=COMPLIANCE_BY_ID[id];if(!c)return;
 const dialog=dialogShell(c[1],[c[3],COMPLIANCE_GROUPS.find(g=>g[0]===c[2])?.[1]].filter(Boolean).join(' · '));
 dialog.append(h('div',{class:'drawer-content compliance-explainer'},[
  h('h3',{text:'What it is'}),h('p',{text:c[4]}),
  h('h3',{text:'When it applies'}),h('p',{text:c[5]}),
  h('h3',{text:'What it means for the design'}),h('ul',{class:'plain-list'},c[6].map(t=>h('li',{text:t}))),
  c[8]?h('p',{class:'input-status open',text:'Regulated data: the cloud, region and vendor choices must support it. MindSuite flags public cloud options for a compliance check.'}):null,
  c[7]?h('p',{},[h('a',{href:c[7],target:'_blank',rel:'noopener noreferrer',text:'Official source ↗'})]):null,
  h('p',{class:'muted',text:'A plain summary to help you ask the right questions. Confirm what applies with your legal or compliance owner.'})]));
 const on=((M.scope.regulation||{}).list||[]).includes(id);
 dialog.append(h('div',{class:'drawer-footer'},[button('Close',()=>dialog.close()),button(on?'Remove from this project':'Applies to this project',()=>{complianceToggle(id,!on);dialog.close();},'btn primary')]));
 dialog.showModal();
}
function complianceToggle(id,on){
 const r=M.scope.regulation||(M.scope.regulation={list:[],other:''});const before=[...r.list];
 r.list=on?[...new Set([...r.list,id])].filter(x=>id==='none'||id==='unknown'?x===id:x!=='none'&&x!=='unknown'):r.list.filter(x=>x!==id);
 logInput('regulation',before,r.list);persist();render();
}
function compliancePicker(){
 const r=M.scope.regulation||(M.scope.regulation={list:[],other:''});const sel=new Set(r.list);
 const sugg=complianceSuggestions().filter(([id])=>!sel.has(id));
 const q=complianceUI.q.trim().toLowerCase();
 const match=c=>!q||(c[1]+' '+c[3]+' '+c[4]+' '+c[5]).toLowerCase().includes(q);
 const chosen=[...sel].filter(id=>COMPLIANCE_BY_ID[id]);
 const chip=c=>h('span',{class:'compliance-chip'+(sel.has(c[0])?' on':'')},[
  h('label',{},[h('input',{type:'checkbox',checked:sel.has(c[0])?'':null,onchange:e=>complianceToggle(c[0],e.target.checked)}),c[1]]),
  c[3]?h('small',{class:'muted',text:c[3]}):null,
  h('button',{type:'button',class:'linkbtn compliance-info','aria-label':'What is '+c[1]+'?',title:'What it is and what it means',onclick:()=>complianceExplainer(c[0]),text:'ⓘ'})]);
 return h('div',{class:'validate-card regulation compliance'+(r.list.length?'':' open')},[
  h('b',{text:'Regulation and compliance *'}),
  h('p',{class:'muted',text:'Which laws and standards apply to this system\'s data and users? They can rule out clouds or regions and add security, audit and retention work. Select ⓘ on any item to learn what it is and what it means for the design.'}),
  sugg.length?h('div',{class:'compliance-suggest'},[h('small',{class:'eyebrow',text:'Likely for this project'}),
   ...sugg.map(([id,why])=>id==='unknown-privacy'?h('div',{class:'compliance-sugg'},[h('span',{text:'Personal data is involved, but no country is named. Add where your users and data are, or pick the laws below.'})])
    :h('div',{class:'compliance-sugg'},[h('b',{text:COMPLIANCE_BY_ID[id][1]}),h('span',{class:'muted',text:' · '+why}),button('Applies',()=>complianceToggle(id,true),'btn small'),button('What is it?',()=>complianceExplainer(id),'linkbtn')])),
   h('small',{class:'muted',text:'Suggestions come from the countries, data and users your project mentions. Nothing is added until you confirm.'})]):null,
  h('div',{class:'compliance-tools'},[h('input',{class:'field compact',type:'search',placeholder:'Find a law or standard, e.g. GDPR, PCI, India','aria-label':'Find a law or standard',value:complianceUI.q,
   oninput:e=>{complianceUI.q=e.target.value;const box=e.target.closest('.compliance');const pos=e.target.selectionStart;render();const again=document.querySelector('.compliance-tools input');if(again){again.focus();try{again.setSelectionRange(pos,pos);}catch(_){}}}}),
   h('label',{class:'type-chip'+(sel.has('none')?' on':'')},[h('input',{type:'checkbox',checked:sel.has('none')?'':null,onchange:e=>complianceToggle('none',e.target.checked)}),'None apply']),
   h('label',{class:'type-chip'+(sel.has('unknown')?' on':'')},[h('input',{type:'checkbox',checked:sel.has('unknown')?'':null,onchange:e=>complianceToggle('unknown',e.target.checked)}),'Not known yet'])]),
  ...COMPLIANCE_GROUPS.map(([g,title])=>{const items=COMPLIANCE.filter(c=>c[2]===g&&match(c));if(!items.length)return null;
   return h('details',{class:'compliance-group',open:q||items.some(c=>sel.has(c[0]))||g==='privacy'?'':null},[h('summary',{text:title+' · '+items.length+(items.some(c=>sel.has(c[0]))?' · '+items.filter(c=>sel.has(c[0])).length+' selected':'')}),
    h('div',{class:'compliance-chips'},items.map(chip))]);}),
  inputControl('Other rules or audit findings',r.other,v=>{const b=r.other;r.other=v;logInput('regulation:other',b,v);persist();},'text','e.g. central bank outsourcing rules; internal audit finding 2025-07'),
  chosen.length?complianceImplications(chosen):null]);
}
function complianceImplications(ids){
 const items=ids.map(id=>COMPLIANCE_BY_ID[id]).filter(Boolean);
 const strict=items.filter(c=>c[8]);
 return h('div',{class:'compliance-impl'},[h('b',{text:'What your selections mean for the design'}),
  strict.length?h('p',{class:'input-status open',text:'Regulated data ('+strict.map(c=>c[1]).join(', ')+'): choose cloud services, regions and vendors that support it, with the right contracts.'}):null,
  h('ul',{class:'plain-list'},items.flatMap(c=>c[6].slice(0,3).map(t=>h('li',{},[h('b',{text:c[1]+': '}),t]))))]);
}

/* ---- Facts for the agent team ---- */
function complianceFacts(add){
 const r=(M.scope&&M.scope.regulation)||{};const ids=(r.list||[]).filter(id=>COMPLIANCE_BY_ID[id]);
 for(const id of ids.slice(0,10)){const c=COMPLIANCE_BY_ID[id];add('regulation',c[1]+(c[3]?' ('+c[3]+')':'')+' applies. Design needs: '+c[6].slice(0,3).join('; ')+'.');}
 if((r.list||[]).includes('none'))add('regulation','No regulation applies, as confirmed by the owner');
 if(String(r.other||'').trim())add('regulation','Other rules: '+String(r.other).slice(0,200));
 return ids.length;
}

(function complianceStyle(){
 if(document.getElementById('compliance-style'))return;
 const css=`
.compliance-suggest{background:#f8faff;border:1px solid #d9e1f2;border-radius:8px;padding:8px 10px;margin:8px 0;display:grid;gap:6px}
.compliance-sugg{display:flex;flex-wrap:wrap;gap:6px 8px;align-items:center}
.compliance-tools{display:flex;flex-wrap:wrap;gap:8px;align-items:center;margin:8px 0}.compliance-tools .field.compact{min-width:260px;width:auto}
.compliance-group{margin:6px 0}.compliance-group>summary{cursor:pointer;font-weight:600;padding:4px 0}
.compliance-chips{display:flex;flex-wrap:wrap;gap:6px;margin:6px 0}
.compliance-chip{display:inline-flex;align-items:center;gap:6px;border:1px solid #d6dde8;border-radius:999px;padding:4px 6px 4px 10px;background:#fff;font-size:12px}
.compliance-chip.on{border-color:var(--accent);background:var(--soft)}
.compliance-chip label{display:inline-flex;gap:6px;align-items:center;cursor:pointer}.compliance-chip input{margin:0}
.compliance-chip small{font-size:11px}
.compliance-info{font-size:14px;line-height:1;padding:0 4px}
.compliance-impl{margin-top:10px;padding-top:8px;border-top:1px solid var(--line)}.compliance-impl li{padding:2px 0;font-size:13px}
.compliance-explainer h3{font-size:14px;margin:12px 0 4px}`;
 document.head.append(Object.assign(document.createElement('style'),{id:'compliance-style',textContent:css}));
})();
