# MindSuite Legacy Modernization — Developer Guide

This guide is for any developer or coding agent (Claude Code, Codex, etc.) continuing this project.
Read it before changing `index.html`. For the current state, recent changes and next steps, read `HANDOFF.md` first.

- **Live site:** https://mjaffry01.github.io/mindsuite-doctrine-of-legacy-modernization/ (GitHub Pages, `main` branch, repo root)
- **Doctrine:** `README.md` (the 8 legacy-system categories). Do not overwrite it.
- **App:** `index.html`, a single static file: plain HTML, CSS and JavaScript. No build step, no framework, no backend.

---

## 1. What the app does

A guided tool that takes a legacy system through three phases:

| Phase | Steps (sidebar) | Purpose |
|---|---|---|
| **1 · Discovery plan** (phase header only) | **01 Record evidence** · 02 Code analysis · 03 System inventory · 04 Review findings · 05 Resolve gaps · 06 Discovery summary | Find out what the system is, with evidence |
| **2 · Deciding** | 07 Rules & people · 08 Target decisions | Choose the target per bounded context (approach, architecture, language, database, integration, cloud) |
| **3 · Plan** | 09 Roadmap · 10 Sign-off & export | Sequence the work in waves, get a named approval, export everything |

**Rule:** "Discovery plan" is only the name of the Phase 1 header; there is no separate plan step. All discovery inputs live on **01 Record evidence** (screen id `upload`; `uploadPage()` returns `evidencePage()`, code block `// 01 Record evidence`).

**Design principle of Record evidence: people give the minimum, the agent detects the maximum, people validate.** The page has four steps:
1. **Start**: only *Project name* and *System being modernized* are required; environment, baseline, objective and exclusions are optional and collapsed.
2. **Connect sources** (`SOURCES`): source code (scanned in the browser) plus database, environment & infrastructure, messaging & caching, identity & access, and telemetry, all from a **Discovery Collector bundle** (`importBundle()`, format in `COLLECTOR-SPEC.md`, example from `sampleBundle()`).
3. **What the agent found** (`viewCoverage()`): the 9 evidence views (incl. **Security & identity**), what was detected, from which source, and what a person still needs to do.
4. **Validate**: confirm the agent's detections in the inventory and findings, name the business owner, and answer the only human questions per bounded context: strategic value, criticality, availability target and consistency need (`setTarget()` creates a target profile that copies the measured numbers).

**Evidence is tailored to the type of system** (code block `// Evidence sources, tailored to the type of system`):
- Start asks for the **system type** (one or more of the doctrine's 8 `TYPES`; `M.scope.types`). After a code scan the agent suggests it (`M.scan.systemType`) and a person confirms; the first type is also recorded as the confirmed "System type" finding.
- `SOURCES` lists 10 evidence sources: code, database, environment & infrastructure, messaging & caching, identity & access, telemetry, batch & scheduling, network edge & partners, documents & diagrams, incidents & changes. Each has who provides it, what works now, and 2–3 fallback questions (`ask`).
- `SOURCE_NEED[source][type]` marks each source **r** (required), **o** (recommended) or **n** (not applicable). With several types the strongest wins (`sourceNeed()`). Not-applicable sources are hidden in a collapsed list. `TYPE_NOTES[source][type]` adds type-specific guidance (e.g. JCL libraries for mainframe, ABAP via abapGit for packaged).
- Each card offers: connect (code scan) or **upload an export** (`pickSourceFiles`, files tagged with `source`), **"No access? Answer instead"** (`manualAnswers`, stored in `M.sourceInfo[id].answers` with the person's name; telemetry answers create an estimated capacity profile), and **"Not applicable"** with a required reason (`markNotApplicable`). Status: done, partial (files registered), answered, not applicable, open.
- One **"Import collector bundle"** button at the top fills several sources at once.
- Validate shows **type-specific questions** (`TYPE_QUESTIONS`, stored in `M.scope.typeAnswers`), e.g. mainframe contract end, which SAP customizations are still needed, or whether the plant can stop for the switch-over.
- To add a type rule: edit `SOURCE_NEED`, `TYPE_NOTES` and `TYPE_QUESTIONS`; nothing else needs to change.

**Client inputs: asked when first needed, then reused** (code block `// Client inputs`). Each answer is logged to `M.history` with the person's name.

| Where | What | Stored in | Used by |
|---|---|---|---|
| 01 Start → Modernization goal | Optional multiple reasons, deadline, success criteria and goal details | `M.scope.brief` | `modernizationReasons()` reads legacy strings and current arrays. Sign-off lists every selected reason. Readiness checks deadline and success criteria. |
| Source cards → "Facts only you know" | Code: tests, release process, environments · Database: years of history, archiving, data quality · Telemetry: user groups, unused features · Network/Messaging: connected systems and owners, partner notice · Documents: audit reports, written decisions | `M.sourceInfo[id].facts` (`CARD_FACTS`) | Roadmap foundations (`briefPrereqs()`) |
| 01 Validate | **Regulation and compliance** (GDPR, PCI DSS, HIPAA, SOX, local law, sector regulator); per context: who maintains it, key-person risk | `M.scope.regulation`; `context.maintainer`, `context.keyPerson` | Cloud scoring note (`regulationCloud`), compliance and knowledge-transfer foundations |
| 06 Rules → Budget, risk and standards | Budget, funding model, risk appetite, cut-over downtime, growth %, standards, procurement, operating model | `M.decide.rules.constraints` (`CONSTRAINT_FIELDS`) | Architecture scoring (`constraintArch`: low risk appetite favours gradual change), year-3 hosting size (`growthFactor()`) |
| 06 Rules → Business case | Yearly run cost: infrastructure, licences, support, staff | `M.decide.businessCase` | Sign-off summary (`runCost()`) |
| 08 Roadmap → Calendar and capacity | Change-freeze months, business events, delivery teams | `M.plan.freezeMonths`, `M.plan.events`, `M.plan.teams` | Freeze windows (`freezeMonths()`, `peakClash()`), wave warnings (`planWarnings()`: after the deadline, too many contexts per team) |

**Code analysis report** (code block `// Code analysis report`, screen id `analysis`, step 02): `analyzeFile()` runs per code file during the scan (package/namespace, imports, relative JS imports, decision points, SQL table reads/writes from string literals, `@Table`/`[Table]` mappings, procedure calls, test-file detection with `TEST_RE`). `analyzeCode()` builds `M.scan.analysis` = `{prefix, modules[], edges[], cycles[], shared[], tableMap{}, hotspots[], extractOrder[], summary}`; raw per-file data is discarded. Extractability = 100 − 10×dependents (max 40) − 5×dependencies (max 20) − 25 if in a cycle − 15×shared tables written (max 30) − 15 without tests + 10 if test code > 30%. Effort from lines (S < 5k, M < 20k, L < 60k, XL), one size up without tests or inside a cycle. `applyCodeAnalysis()` proposes contexts (only when no human contexts exist), module components, table writers/readers and risks. `graphSvg()` draws the dependency graph. These are rules of thumb; keep them labelled that way.

**Agent teams in the web app** (one folder per team, no build): `teams/shared/agent-backend.js` (backend connection), `teams/<team>/panel.js` (the team's panel and import), `teams/<team>/agents.html` (explanation page; data in the `AGENTS`, `EDGES`, `POS`, `ORCH_LINKS` and `STEPS` arrays at the bottom; engine and style in `teams/shared/team-page.js` / `.css`). `index.html` loads the panel scripts with `<script src>` before its main script; they use its helpers at call time. When an agent's status changes (planned → partly → built), update `status` there and in `AGENT-DESIGN.md`. It opens via `openAgents()` from the button under the Source code card (01 Record evidence) and from the 02 Code analysis header. `db-agents.html` is the same page for the database team (opened by `openDbAgents()` from the Database card and the database panel); its data is in the same arrays plus `ORCH_LINKS`.

**"How to extract" guides** (code block `// "How to extract" guides`): `EXTRACT_GUIDE[source]` holds, per source, an intro, read-only methods (tool, command, official documentation links), and `fromCode()`, which summarises what the code scan already found. A card with code-scan evidence shows status "From code: …" (`fromCode(id)`). The documentation links were checked against the vendors' official pages in November 2026; re-check them when updating.

**Database export parser** (code block `// Database export parser`): the Database card's "Upload export (.sql / .csv)" (`pickDbExport`) parses, in the browser:
- **schema exports (DDL)** with `parseDdl()`: tables, views, procedures, functions, packages and bodies, triggers, Oracle scheduler jobs, SQL Agent jobs, MySQL events, pg_cron jobs, database links, linked servers, foreign servers, and admin grants (`GRANT DBA`, `sysadmin`, `ALL PRIVILEGES`, `SUPERUSER`);
- **catalog query results** with `parseCatalogCsv()`: the columns `record_type, schema_name, name, row_count, size_mb, line_count, detail`, where record types are `table | code | job | link | admin`. Comma, semicolon or tab separators, quoted values, `sqlcmd` dash lines, and Db2 header-less exports are all handled.
- `DB_SCRIPTS` contains the downloadable read-only catalog scripts (Oracle, SQL Server, PostgreSQL, MySQL/MariaDB, Db2) that produce that CSV. If you change their columns, update `parseCatalogCsv()`.
- `applyDbExport()` merges files into `M.dbExport` and creates records with `origin:'export'`: the largest 60 tables, database code (plus the 10 largest programs), jobs, links, and risks for admin accounts, links and heavy stored logic. It also adds the engine to the technology finding and the summed size to the system capacity profile.

Manual evidence (documents, the evidence register, full capacity table, people) stays available in a collapsed "Add evidence manually" section. The older checklist functions (`discoveryInputs`, `inputsPanel`) are no longer shown on the page.

**Code scan, part 2** (code block `// Code scan, part 2`): besides build files, the scan reads configuration and deployment files (`configKind()`, `scanConfig()`) and source patterns (`scanSource()`). It finds database connections (engine and host), caches and the **caching strategy** (`cacheStrategy()`), message brokers, login methods and auth frameworks, the number of authorization checks and endpoints, outbound hosts, Kubernetes, Terraform and container images, and **hard-coded secrets (file and line only, never the value)**. The results become technologies, risks, interfaces and data stores (`detectConfigTech`, `detectConfigRisks`, `applyScanConfig`).

**Collector bundle** (code block `// Discovery Collector bundle`): imported records get `origin:'collector'` and status `extracted`. A re-import replaces unconfirmed collector records only. Telemetry and database size become a system-wide measured capacity profile, which `capOf()` uses for every context until that context has its own.

Core principles. Keep them when extending:

1. **Evidence before conclusions.** Every finding links to evidence (files, citations).
2. **Machines propose, people decide.** Code scans, rules and (later) AI agents only *propose*. Strategic value, criticality, approach and sign-off are human decisions with a named person.
3. **Human decisions are never overwritten.** Statuses `confirmed`, `overridden`, `manual`, `unknown` are protected (`protectedStatus()`). New evidence is shown as "pending" or "recheck" instead.
4. **Everything is traceable.** Every review, override, decision and approval is pushed to `M.history`, with the actor's name.
5. **Code never leaves the browser.** The code scanner reads files locally and stores only metadata.

---

## 2. Running and testing locally

```bash
python -m http.server 8768
```

Then open http://localhost:8768/. State is saved in `localStorage` under the key **`mindsuite-artifact-mockup-v1`**. Clear it to reset, or use **New project** in the app.

The older questionnaire draft uses the key `mindsuite-flow-v1`. Leave it alone; it may hold a user's data.

**Quick manual test path:**
1. 01 → **Load reference scenario** → **Analyze & continue**. This loads the synthetic "HarvestLink" sample. Alternatively, fill the two Start fields, scan a code folder, and import the sample collector bundle (How collection works → Download sample bundle → Import a bundle).
2. 01 → Validate: set **Strategic value**, criticality, availability and consistency for each context, and add the *Business owner*.
3. 06 → check the organization rules.
4. 07 → **Use suggestion** → **Apply all recommendations** → write a rationale → **Record decision**.
5. 08 → set wave dates and check freeze windows. 09 → **Approve plan** → **Export full record**.

**Testing the code scanner without a real folder:** in the browser console, pass fake file objects:
```js
const F=(p,t)=>({webkitRelativePath:'demo/'+p,size:t.length,text:async()=>t});
await runScan('folder',[F('pom.xml','<project><artifactId>x</artifactId><properties><maven.compiler.source>1.8</maven.compiler.source></properties><dependencies><dependency><groupId>org.springframework</groupId><artifactId>spring-webmvc</artifactId><version>4.3.30.RELEASE</version></dependency></dependencies></project>'),F('src/A.java','class A{}\n')]);
M.scan // results
```
For zips, build one with `await loadJSZip()` and `new JSZip()`, then call `runScan('zip', file)`.

Before testing, back up the user's saved state, and restore it afterwards.

---

## 3. How `index.html` is organised (read this first)

The script was built up in **layers**. Later layers *redefine* some earlier functions. **In JavaScript, when two function declarations share a name, the last one wins.** So:

> ⚠️ **Always edit the LAST definition of a function.** Earlier copies are dead code.

Functions defined more than once (the last copy is live): `addFiles`, `artifactList`, `exportData`, `fieldCard`, `invalidateEvidence`, `loadSamples`, `removeFile`, `showEvidence`, `simulate`, `simulatedValues`, `stats`, `uploadPage`.

Find the live copy with a search, and pick the bottom match: `grep -n "function simulate(" index.html`.

Approximate map (line numbers drift; search for the marker text):

| Marker to search | What it is |
|---|---|
| `const TYPES=` | The 8 legacy system types (matches the doctrine) |
| `TECH_CATALOG=[` | 279 technologies `[id, label, hint]` |
| `TECH_LAYER={` | Technology id → layer (languages, databases…) |
| `TECH_PREFERRED={` | **Curated** ordered tech list per system type (drives picker order) |
| `const APPROACH=` | The 7 Rs + "Not decided" |
| `const PURPOSE=` / `const CAPS=` | Business domains and capability candidates per domain |
| `const SUBDOMAIN=` `DRIVERS` `KNOWLEDGE` `LANGUAGE` `RELATIONS` `CRIT` | Option lists for DDD findings |
| `const FIELDS=` | **The 15 summary findings** (key, title, group, options, `human:true` = people only) |
| `const MOCK_KEY` / `emptyMock` / `let M=` | State and persistence |
| `const h=` | DOM helper: `h(tag, attrs, children)` |
| `function renderBase` | Base render: dispatches `screen` → page function |
| `// Evidence-led discovery mockup` | Discovery protocol (8 evidence views), scope, evidence register, review/override logic |
| `// System inventory` | Inventory record types, context map, record dialogs, reference-scenario seeding |
| `// Local code scan` | Scanner: folder/zip reading, build-file parsers, detection rules, risk rules |
| `// Phase 2 (Deciding)` | Target catalog, cloud service mapping, scoring rules, decisions, roadmap, sign-off, nav |
| `function render(){renderBase` | Final `render()`: base render + phase navigation |

---

## 4. State model (`M`)

Everything lives in one object `M`, saved by `persist()` to `localStorage`. Call `persist()` then `render()` after every change.

| Field | Meaning |
|---|---|
| `project` | Discovery name |
| `scope` | `{component, environment, baseline, journey, outOfScope}` |
| `files[]` | Evidence records `{id, name, category (view), sample, scanned, context:{component, environment, asOf, revision, origin, custodian, basis, locator, window, lineage}}` |
| `availability{}` | Per evidence view: `{state:'needed'|'unavailable'|'na', reason}` |
| `values{}` / `status{}` / `evidence{}` | The 15 findings: value (id or id[]), status, evidence file ids |
| `details{}` | Technology id → version text, e.g. `"4.3.30 · pom.xml"` |
| `custom{}` | User-added options per finding |
| `originals{}` / `pending{}` / `recheck{}` | First proposed value; newer proposal not applied; "evidence changed" notes |
| `history[]` | Audit log `{field, at, actor, action, reason, before, after}` |
| `inventory{}` | `contexts, components, dataStores, interfaces, rules, events, terms, risks, capacity`: arrays of records |
| `scan` | Last code-scan result (metadata only) |
| `people[]` | `{id, name, role, owns}` |
| `actor` | Person id the user is "working as" (used by `actorName()`) |
| `decide` | `{rules:{stance, skills, residency, region, contracts, weights}, decisions:{[contextId]: {approach, choices, product?, retireTo?, rationale, decidedBy, at, status, reopened?}}}` |
| `plan` | `{items:{[contextId]: {wave?, pattern?, size?, notes}}, waves:{[1..3]: {start, end}}, signoff:{by, name, at, note, fingerprint}}` |

**Phase 2 decision rules (keep these when changing the code):**
- The bounded context's `approach` (set in the inventory or on the decision card) is the single source of truth. `decisionFor()` syncs it, and reopens a recorded decision if it changes.
- `APPROACH_DIMS` lists which dimensions an approach needs; `APPROACH_FIXED` lists the ones it fixes (e.g. rehost keeps language and database). Always read choices through `effective(dim, d)`, never `d.choices` directly, so choices left over from an earlier approach are ignored.
- `recommended()` returns nothing on a tie. The UI says "No clear winner" instead of picking by list order.
- `violations(c, d)` re-checks choices against the current rules on every render. A recorded decision that now breaks a rule shows "Breaks rules" and blocks sign-off.
- Recording a decision requires: an approach, every needed dimension, the product (replace) or destination (retire), a rationale, and a named person.
- Sign-off stores a `fingerprint` of decisions and plan; any later change shows "Changed since approval".

**Capacity & service levels** (inventory type `capacity`, code block `// Capacity & service levels`):
- One or more profiles per bounded context: users, concurrent users, locations, average/peak transactions per second, peak times, peak months, database size and growth, file storage, availability, RTO, RPO, response time, batch window, and whether the numbers are measured, estimated or targets. `capOf(c)` returns the busiest profile.
- Scoring hooks `capArch/capDb/capInt/capCloud` add reasons, e.g. a peak ≥5× the average favours auto-scaling or serverless; ≥99.99% availability penalises a single data centre; ≥2 TB of data penalises a database change.
- `hostingEstimate(c, d)` is a rule-of-thumb sizing: about 25 requests/s per vCPU plus 30% headroom, 4 GB RAM per vCPU, 3-year database growth plus 20%, and backups at about 2× data. Change these assumptions in one place, and keep them labelled as estimates.
- Phase 3 uses peak months as cut-over freeze windows (`freezeNote`, `peakClash`) and adds resilience, recovery and load-test foundations (`capPrereqs`).

**Consistency (CAP / PACELC)** (code block `// Consistency (CAP / PACELC)`): capacity profiles record `consistency` (strong / eventual / mixed), `staleness` and `partitionPref` (stay correct vs. stay available on a network failure). `consArch/consDb/consInt` add scoring reasons, e.g. strong consistency penalises document stores, events without an outbox, and CDC lag. `capTradeoff()` shows a plain-language trade-off on each decision card (e.g. strong + 99.9% → one primary with a synchronous standby). It flags a conflict when strong consistency is combined with "prefer availability". `consPrereqs()` adds reconciliation, outbox/saga and single-owner items to the roadmap.

**Discovery inputs** (code block `// Discovery inputs`): `INPUT_GUIDE` says who provides each evidence view, in which formats, and whether it is Required or Recommended. `discoveryInputs()` builds the "What we need from you" checklist on step 01 (scope, code, 7 evidence views, capacity, business answers, stakeholders) with live status and a button that jumps to each input. When adding a new input type, add a row there so it stays visible.

**Product framing:** the app is positioned as *scan the current state → decide the target → plan the modernization* (product name "Legacy Modernization Advisor"). When selling, keep the claims accurate: the code scan is real, target ranking is rule-based and explainable, and AI-agent analysis is on the roadmap.

**Statuses** (labels in `statusLabel`):
- `suggested`, `inferred`: from the synthetic reference scenario ("· demo").
- `extracted`, `inferred-code`: from the real code scan.
- `conflict`, `missing`.
- `confirmed`, `overridden`, `manual`, `unknown`: human-protected, never overwritten.

**Inventory record shape:** `{id, name, …type fields, status, method ('extracted'|'inferred'|'reported'|'entered'), origin ('scenario'|'scan'|'user'), cite, evidence:[fileIds]}`.

---

## 5. How to extend (recipes)

**Add a screen:**
1. Write `function myPage(){ return h('div',{},[pageHeader('NN / TITLE','Heading','Description',[buttons]), infoBanner(), …]); }`.
2. Add `screen==='mine'?myPage():` to the dispatch in `renderBase`.
3. Add it to the `groups` array in `renderNav()`.

**Add an inventory record type:**
- Add an entry to `INV_TYPES` (`key, title, one, hint, fields[]`). Field `type` is `select | context | contexts | textarea | text`; `human:true` means people only.
- Add the key to the list in `ensureDiscovery()`.

**Add a scanner rule:**
- Dependency → technology: add `[ecosystem, /regex on name/, techId]` to `DEP_RULES`.
- Language: add to `LANG_EXT`, and to `LANG_TECH` to map it to a catalog id.
- Risk: add a check in `detectRisks()`.
- New build file: add it to `manifestType()` and write a `parseX(text, file)` returning `{type, name, version, deps[], runtime:{lang, version}}`.

**Add a target option or cloud service:**
- Add to `TARGETS[dimension].opts`, add the matching key in each `CLOUD_SVC[cloud]`, and add a scoring rule in `SCORE_RULES[dimension]`.
- Scoring reasons use `add(criterion, value -2..+2, 'plain-language reason')`. The criteria are the `WEIGHT_KEYS`. Use `exclude('reason')` for hard rules.
- Cloud service names change. Re-verify `CLOUD_SVC` against provider documentation when updating.

**Log a human action:** push to `M.history` with `actor: actorName()` (or `personLabel(id)` for decisions).

---

## 6. Conventions

- Use the existing UI building blocks: `h()`, `button()`, `inputControl()`, `selectControl()`, `pageHeader()`, `badge()`, `dialogShell()`, and CSS classes `summary-section`, `finding-card`, `metrics/metric`, `scan-table`, `scan-list`, `muted`, `version-note`, `conflict-note`, `empty-state`.
- Plain, non-technical wording in the UI; the users are business and IT people.
- **Input colour coding** (code block `// Input colour coding`): every `.field` is painted automatically. Orange = required and empty, green = filled, plain = optional. A field is required when its label ends with ` *` or it has `data-req="1"`. Mark new required inputs that way; a MutationObserver keeps colours current, so no extra calls are needed. Show the key with `fieldKey()`.
- No external libraries, except JSZip 3.10.1 lazy-loaded from cdnjs when a zip is scanned.
- Must work at phone width (375px) with no horizontal scroll. Check this after layout changes.
- After editing, syntax-check the script: copy the `<script>` contents to a file and run `node --check file.js`.

---

## 7. What is real vs simulated

| Part | Status |
|---|---|
| Scope, evidence register, inventory, reviews, gaps, rules, decisions, roadmap, sign-off, export | Real (works with user input) |
| Code scan (folder/zip, build files, languages, versions, tables, risks) | Real, runs locally |
| "Load reference scenario" (HarvestLink) | Synthetic sample data |
| Target scoring | Rule-based and explainable (`SCORE_RULES`); not AI |
| Agent backend (`backend/`, Go): code (15), database (14), infrastructure (13), messaging & caching (12), identity & access (12), telemetry (11), batch & scheduling (10), network edge & partners (11), documents & diagrams (12) and incidents & changes (11) teams, each in `backend/internal/teams/<team>/`; shared core in `backend/internal/agents/` | Real; rule-based agents tested; Claude-powered agents need an API key and consent (not yet tested with a key) |
| Reading documents, live databases, logs, interviews | Not built yet |

---

## 8. Backlog (agreed direction, not built yet)

1. **Security analysis in discovery** (see section 9).
2. **Users & usage, cost & contracts, change safety (tests, release process)** as inventory record types or findings.
3. **Assumptions register** (assumption, owner, check-by date).
4. **Scan phase 2:** open-source licence check on the dependency list. (Git history hotspots and bus factor are in the agent backend.)
5. **Connectors:** a GitHub/GitLab/Azure DevOps connection through a small backend (e.g. Cloudflare Worker) holding a read-only GitHub App token. Never put tokens in the page. Export guides for non-Git legacy code (mainframe libraries, IBM i source members, PowerBuilder PBLs, Oracle Forms `.fmb` → XML, ABAP via abapGit).
6. **AI agent:** built as the Go agent backend (`backend/README.md`). The app imports its results with `agentImport()` using `origin:'agent'`; people still confirm. Next: test with a real key, then feed agent findings into the decision rules.
7. **Cleanup:** remove the dead earlier copies of duplicated functions (section 3) once behaviour is covered by manual tests.

---

## 9. Security coverage today and what is missing

**Covered today:**
- The code scan flags known vulnerable or end-of-life versions: Log4Shell (`log4j-core` < 2.17), Log4j 1.x, Struts 1, AngularJS, jQuery < 3.5, and out-of-support Java, .NET, Node and Spring. This is a short rule list, not a full vulnerability scan.
- Data sensitivity per data store (personal / financial / regulated).
- A "Security vulnerability" risk category.
- Data-residency rules in Phase 2, which can exclude public clouds for regulated data.

**Missing (recommended additions):**
- **Now built:** the **Security & identity** evidence view (9th view); code-scan detection of login methods, auth frameworks, authorization checks and **hard-coded secrets (file and line only)**; collector-bundle identity findings (MFA, shared, inactive and privileged accounts, database admin accounts).
- **Still missing:** outdated TLS/crypto settings, a full dependency vulnerability lookup (e.g. against the OSV database), encryption at rest and in transit, audit-logging checks, and the applicable regulations.
- **Security findings feeding Phase 2:** e.g. "no SSO" → identity work in the foundations, "PCI scope" → cloud and region constraints.


## GUI presentation layer (Codex, 2026-11-17)

`assets/workspace.css` loads after the embedded legacy CSS and owns the refreshed appearance. `assets/workspace.js` defines `enhanceWorkspace()`, called after `renderNav()` by the final `render()`. It adds accessible mobile navigation, discovery-step links, source search/filtering and collection-detail disclosures. It does not read or write `M`, localStorage, provider credentials or backend requests. Filter state is ephemeral. Preserve this final render hook. Source-card DOM enrichment moves existing nodes, retaining their handlers. Shared agent explanation styling remains in `teams/shared/team-page.css`.

The current app is a static frontend with an optional Go agent backend. Some older sections above describe the earlier browser-only product. Use the latest HANDOFF entry and actual team registry for current implementation status. Do not use the old Codex mockup builder to regenerate this app.


### Contemporary refinement (2026-11-17)
The final block in `assets/workspace.css` now defines the light sidebar, indigo theme, compact navigation and system-type tiles. `enhanceWorkspace()` calls `enhanceContemporaryUI()`, which adds navigation icons, the header phase switcher and shorter visible source-card action labels (original accessible names and handlers are retained). The phase switcher delegates to existing sidebar buttons. No application-state or localStorage writes. Matching agent explanation-page rules are at the end of `teams/shared/team-page.css`.
