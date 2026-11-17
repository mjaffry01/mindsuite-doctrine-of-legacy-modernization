# HANDOFF — running work log

**Purpose:** if the current AI session runs out, another model or developer can continue from here. Keep this file updated after every change. Newest entries go at the top of the change log.

Read in this order: **this file → `PRODUCT-OUTCOMES.md` (what the product must deliver) → `AGENT-DESIGN.md` (the agents behind code analysis) → `DEVELOPER-GUIDE.md` (how the code works) → `COLLECTOR-SPEC.md` (collector to build) → `README.md` (the doctrine).**

---

## 1. Where things are

| What | Where |
|---|---|
| GitHub repo | `mjaffry01/mindsuite-doctrine-of-legacy-modernization` (branch `main`) |
| Live site (GitHub Pages) | https://mjaffry01.github.io/mindsuite-doctrine-of-legacy-modernization/ |
| Working copy (temporary!) | `%TEMP%\lm-repo` (= `C:\Users\X1CARB~1\AppData\Local\Temp\lm-repo`) |
| Permanent backup of the working files | `C:\Users\X1 CARBON\Downloads\Legecy Modernization\` (a copy of `index.html`, the `.md` files; refreshed with each log entry) |
| Local preview | `python -m http.server 8768` in the working copy → http://localhost:8768/ (also the `legacy-modernization` entry in `.claude/launch.json`) |
| The app | `index.html`: a single static file, plain JS, no build, state in `localStorage` key `mindsuite-artifact-mockup-v1` |
| Agent teams | **One folder per team**, backend and web app alike. Backend: shared core `backend/internal/agents/`, teams in `backend/internal/teams/{code,database,infrastructure,messaging,identity,telemetry}`: all six evidence teams built. Web app: `teams/shared/` (backend connection, explanation-page engine), `teams/<team>/panel.js` (the panel in the app) and `teams/<team>/agents.html` (the explanation page). See `backend/README.md`. |
| Agent backend (Go) | `backend/`: runs the code (15), database (14), infrastructure (13), messaging & caching (12), identity & access (12) and telemetry (11) teams. `cd backend && go run ./cmd/mindsuite-agents serve` → http://127.0.0.1:8090 (also `mindsuite-agents-backend` in `.claude/launch.json`). `go test ./...` must pass. See `backend/README.md`. |

**Git state at the last update:** local `main` is level with `origin/main` (`e26b51b`). **All work since then is uncommitted:** `index.html` modified; `agents.html`, `backend/`, `AGENT-DESIGN.md`, `DEVELOPER-GUIDE.md`, `COLLECTOR-SPEC.md`, `HANDOFF.md` and `PRODUCT-OUTCOMES.md` new. The live site still shows the older step-by-step questionnaire.

---

## 2. Owner's preferences (follow these)

- **Do not push to GitHub until the owner says "push it".** Edit and test locally first.
- Plain, non-technical wording in the UI. Clear, colour-coded inputs (orange = required and empty, green = filled).
- Principle of the product: **take the minimum from the user, let the agent (scanner/collector) detect the maximum, then a human validates.**
- Selling story: **scan the current state → decide the target → plan the modernization.** Be honest in claims: the code scan is real, target ranking is rule-based, and the AI agent is still to be built.
- When giving "how to extract" advice, use **verified official documentation links**, not guesses.
- Test every change in the browser (desktop 1440×900 and mobile 375), check the console, and restore the owner's saved browser data afterwards (back up both `localStorage` keys first).

---

## 3. Current product (as of the latest entry)

**Phase 1 · Discovery plan** (header only)
- **01 Record evidence**, with four steps:
  1. **Start:** project name, system being modernized, **system type** (the 8 doctrine types; the agent suggests one from the code scan), and the **Client brief** (why now, hard deadline, success criteria).
  2. **Connect sources:** 10 source cards, tailored by system type (Required / Recommended / hidden). Each card has its upload or scan, **How to extract** (commands + official links), **Facts only you know**, **No access? Answer instead**, and **Not applicable**. One **Import collector bundle** button sits at the top.
  3. **What the agent found:** the 9 evidence views and what was detected automatically.
  4. **Validate:** confirm detections, business owner, regulation & compliance, type-specific questions, and per context strategic value, criticality, availability, consistency, maintainer and key-person risk.
- **02 Code analysis** (report: modules, dependency graph, cycles, code-to-table map, tests, hotspots, extractability, effort, extraction order).
- 03 System inventory · 04 Review findings · 05 Resolve gaps · 06 Discovery summary.

**Phase 2 · Deciding**
- 07 Rules & people: people, technology stances, constraints, budget/risk/standards, business case.
- 08 Target decisions: per context, the approach plus architecture, language, database, integration and cloud. Options are ranked with reasons, cloud services are mapped, and each card shows a hosting estimate and the CAP trade-off.

**Phase 3 · Plan**
- 09 Roadmap: foundations, calendar & capacity, 3 waves, freeze windows.
- 10 Sign-off & export: what the plan must achieve, readiness, approval, print, JSON export.

**Real vs. simulated**
- **Real:**
  - code scan: build files, config, deployment files, auth, caching, secrets;
  - database export parser (DDL and catalog CSV);
  - collector-bundle import (format defined);
  - all forms, decisions, roadmap and sign-off.
- **Not built yet:** the Discovery Collector itself, parsers for the other sources' exports (they're only registered), and the AI agent.
- **Sample data:** "Load reference scenario" (HarvestLink) and the sample collector bundle.

---

## 4. Next steps / backlog (agreed with the owner, not built yet)

1. **Merge 03 System inventory + 04 Review findings + 05 Resolve gaps into one "Validate" page**, organized by the 9 evidence views. Proposed and explained; the owner hasn't confirmed yet.
2. **Parsers for the other source exports**, in the browser, like the database parser: RVTools/PowerCLI CSV, `kubectl` JSON, cloud inventory CSV, scheduler exports (Control-M JSON, AutoSys JIL, crontab, `schtasks` CSV), Jira/ServiceNow CSV, monitoring CSV (requests per hour → capacity).
3. **Build the Discovery Collector** per `COLLECTOR-SPEC.md` (separate `collector/` folder, read-only CLI).
4. **Infrastructure map** (servers, networks, load balancers, DR) in the inventory.
5. **Security:** TLS checks, OSV dependency vulnerability lookup, licence check of dependencies (needs the owner's OK: sends dependency names outside).
6. **Agent backend next:** test the Claude-powered agents on a real codebase with an API key (prompts in `backend/internal/teams/code/domain.go`, `backend/internal/agents/rules.go`, `verify.go`, `report.go`); Oracle Forms / ABAP specialists; authentication if it is ever deployed beyond localhost; documents (PDF/Word) as evidence.
7. ~~AI agent backend~~ built in Go (`backend/`), see the change log.
8. Cleanup: remove the dead earlier copies of functions that are defined twice (see `DEVELOPER-GUIDE.md` §3).

---

## 5. How to continue safely

1. Read `DEVELOPER-GUIDE.md` §3. **Functions defined more than once: the last one wins.** Edit the last one.
2. Make changes with small, exact string replacements. After each change, extract the `<script>` and run `node --check`.
3. Test in the browser. To test the code scanner without a folder, call `runScan('folder', fakeFiles)`; see the guide §2.
4. **Add an entry to the change log below**, and copy the files to the permanent backup folder.
5. Push only when the owner says "push it". Then: `git add -A && git commit` (end the message with the attribution line used in this project), `git push`, wait for the GitHub Pages build (`gh api repos/mjaffry01/mindsuite-doctrine-of-legacy-modernization/pages/builds/latest`), and check the live URL.

---

## 6. Change log (newest first)

### 2026-11-17 — Telemetry agent team (11 agents); all six teams built
- Owner's request: "Telemetry". Built in its own folders: `backend/internal/teams/telemetry/` (register.go, model.go, team.go, telemetry_test.go, testdata/sample + make_sample.py) and `teams/telemetry/` (panel.js, agents.html).
- Agents: orchestrator, telemetry reader (NGINX/Apache/IIS logs, Splunk/App Insights/Datadog CSV, Prometheus, CloudWatch, collector bundle), load & peaks, response times, errors, usage & unused features (incl. code endpoints never called), coverage & gaps, business use (Claude Opus; cleaned endpoint names and numbers only), capacity & cut-over, verifier, report.
- Privacy: client IPs hashed and only counted; query strings dropped; ids/e-mails/tokens in paths replaced by `{id}`; access-log lines never quoted in citations. Overlapping sources (log + monitoring for the same hour) take the higher count instead of adding up.
- Core: `.log` files are now read as text.
- App: Telemetry card → "Run the telemetry agent team"; panel on 02 Code analysis; import fills a measured capacity profile (avg/peak per second, peak hours, peak months, 95th percentile, clients), business functions (components) and risks with `team:'telemetry'`. Team switcher has all six teams.
- Tested: `go test ./...` passes for all six teams (telemetry sample: 12 months generated with known answers, 30 checks incl. no IP address, e-mail, token or record id in the output). Fixed while testing: log lines quoting client IPs, missing password findings, static files counted as an endpoint, one-day log percentiles overriding 12-month monitoring, gap length off by one hour. Browser: upload (21 findings), import, card buttons, six panels, explanation page. **Not tested: the Claude-powered business-use agent (no API key).** localStorage restored. Not pushed.

### 2026-11-17 — Identity & access agent team (12 agents)
- Owner's request: "Identity & access". Built in its own folders: `backend/internal/teams/identity/` and `teams/identity/` (panel.js, agents.html).
- Agents: orchestrator, directory reader (AD user and group-member CSVs, Entra MFA report and Conditional Access CSV, Okta JSON, openssl output, collector bundle), accounts, privileged access, MFA & sign-in policy, groups & roles, certificates, provider & protocols, role model (Claude Opus: target roles from group names and counts; no person names), migration assessment (clean-up list, managed identities, approach, order), verifier, report (model gets finding titles only).
- Privacy: passwords and hashes never read; accounts are not listed one by one, named only in findings that need action, and never sent to Claude.
- App: Identity & access card → "Run the identity & access agent team"; panel on 02 Code analysis; import adds sign-in apps (interfaces), target roles (rules) and risks with `team:'identity'`. Team switcher now has five teams.
- Tested: `go test ./...` passes for all five teams (identity sample: 36 checks incl. disabled accounts not counted as inactive, no password in the output); fixed an empty group export being ignored and the approach calling Active Directory a cloud provider. Browser: upload (30 findings), import, card buttons, all five panels, explanation page. **Not tested: the Claude-powered role-model agent (no API key).** localStorage restored. Not pushed.

### 2026-11-17 — Messaging & caching agent team (12 agents)
- Owner's request: "Messaging & caching team". Built in its own folders: `backend/internal/teams/messaging/` (register.go, model.go, team.go, messaging_test.go, testdata/sample) and `teams/messaging/` (panel.js, agents.html).
- Agents: orchestrator, broker & cache reader (runmqsc DISPLAY output, kafka-topics --describe / consumer groups, redis-cli INFO + CONFIG GET, memcached stats, RabbitMQ definitions, collector bundle), versions & support, flows & usage (sender/reader per queue from the uploaded code and configuration, partner channels, remote queues, consumer groups, waiting messages with no reader, queues not in the code), health & backlog, delivery & reliability, caching, security, integration meaning (Claude Opus: business event per queue, integration map; names only), migration assessment (target per broker and cache, order, approach), verifier, report.
- App: Messaging & caching card → "Run the messaging & caching agent team"; panel on 02 Code analysis; import adds queues/topics and caches (data stores), partner channels, remote queues and integrations (interfaces), business events (domain events) and risks with `team:'messaging'`. Team switcher on every explanation page now has four teams.
- Tested: `go test ./...` passes for all four teams (messaging sample: 40 checks incl. no system queues/channels, no password in the output); fixed echoed `DISPLAY CHANNEL(*)` being read as a channel. Browser: upload of the sample (52 findings), flows, import, card buttons, explanation page. **Not tested: the Claude-powered integration-meaning agent (no API key).** localStorage restored. Not pushed.

### 2026-11-17 — Infrastructure agent team (13 agents); every team in its own folder
- Owner's requests: "Environment & infrastructure: create a team of agents", and "keep all the agents of source code, database, environment & infrastructure, messaging & caching, identity & access and telemetry in separate folders… it will be very difficult to troubleshoot later".
- **Folders.** Backend: `internal/agents/` is now only the shared core (run, `Team` interface + registry, store, loader, verifier, rules reader, Claude client, helpers). Each team is a package under `internal/teams/` with its own agents, parser, report, test and `testdata/sample`; `teams.go` registers them; `messaging/`, `identity/`, `telemetry/` are reserved (doc.go). Web app: the team panels moved out of `index.html` into `teams/shared/agent-backend.js`, `teams/code/panel.js`, `teams/database/panel.js`, `teams/infrastructure/panel.js` (loaded by `<script src>` before the main script). Explanation pages moved to `teams/<team>/agents.html` with a shared `teams/shared/team-page.css` + `team-page.js` and a team switcher; `agents.html` and `db-agents.html` at the top level are gone (never pushed).
- **Infrastructure team** (`internal/teams/infrastructure/`): estate reader (RVTools vInfo, PowerCLI/Hyper-V Get-VM, generic server CSV, kubectl JSON, Azure/AWS/GCP inventories, ss/netstat/Get-NetTCPConnection, collector bundle), lifecycle & support (OS, hypervisor, software, images), capacity & sizing, dependencies (connections resolved to hosts, outside endpoints, hubs, move groups), Kubernetes & containers, cloud resources, infrastructure as code (the code team's platform agent), security & resilience (plain-text protocols, RDP, single points of failure, passwords as file:line), roles & environments (Claude Opus, names/OS/software/connection summaries only), migration assessment (retire/rehost/replatform/exclude, waves, approach, sizing), verifier, report. App: Environment & infrastructure card → "Run the infrastructure agent team"; panel on 02 Code analysis; import adds runtime services and Kubernetes workloads (components), outside endpoints (interfaces) and risks with `team:'infrastructure'`. Code-team import now keeps every `team`-tagged row.
- Tested: `go vet` clean; `go test ./...` passes for all three teams and the server (infrastructure sample: 35 checks, no password in the output). Browser: upload of the sample estate (62 findings), import, card buttons, all three explanation pages at desktop and phone width. Fixed a CSS comment containing `*/` that broke the shared page style. **Not tested: the Claude-powered roles agent (no API key).** localStorage restored. Not pushed.

### 2026-11-17 — Source code card: "Run the code agent team"
- Owner: the Source code card only offered the browser scan, not the agent team. Added "Run the code agent team" (`agentGo()`: opens 02 Code analysis at the agent panel) and renamed the link to "See the code agent team ↗", matching the Database card. "Works now" text now names both: quick scan in the browser, full analysis with the agent backend.
- Tested in the browser: buttons present, click opens the agent panel. Not pushed.

### 2026-11-17 — Database agent team (14 agents) in the Go backend
- Owner's request: "in the same way create a team of database agents" (chose: full team — page, backend agents, app connection).
- Backend: `Options.Team` (`code` | `database`); `Run.Execute` dispatches to `executeCode` / `executeDB`; step titles and the team list per run. New files `internal/agents/db_model.go`, `db_agents.go`, `db_ai.go`, `db_team.go`, `db_report.go`; `runRulesAs` (business rules for any agent, line ranges in the reading list); `wholeKinds` in the verifier; `.csv`/`.ddl`/`.trg` file types; CLI `run -team database`. Agents: orchestrator, catalog reader, schema & relationships, size & usage, stored code, jobs/links/outside calls, security & privacy, portability, design quality, database business rules (Opus), data domains & meaning (Opus), migration assessment, verifier, report writer.
- Safety: metadata only; data rows in an export are counted, never read or sent; password lines (incl. `IDENTIFIED BY`) never quoted or sent; link connect strings with credentials hidden; the data-domains agent sends names and facts only.
- `index.html`: 02 Code analysis has a second panel "Run the database agent team" (`dbAgentPanel()`, state `M.dbAgentRun`, `M.agentBackend.dbRunId/dbAllowModel`); Database card has "Run the database agent team" and "See the database agent team ↗". `dbAgentImport()` adds tables (with sensitivity), programs/jobs, links/outside calls, CHECK and stored-code rules, terms, data domains (as contexts only if none from people) and risks with `origin:'agent', team:'database'`; `agentImport()` now leaves `team:'database'` rows alone.
- New page `db-agents.html` (same layout as `agents.html`, linked both ways).
- Tested: `go test ./...` (sample Oracle export `testdata/dbsample` with known answers: 30+ checks, no password or data row in the output; server upload with `team: database`). Browser: upload of the sample export, live steps, results, import (9 tables, 13 risks, 2 rules), code import keeps database rows, db-agents.html renders. **Not tested: the two Claude-powered database agents (no API key).** localStorage restored. Not pushed.

### 2026-11-17 — "Start" button creates the project folders
- Owner's request: a button called Start that creates the folders for all the inputs, under a folder with the project's name.
- `index.html`, 01 Record evidence → step 1 Start: `startFolderBlock()` / `startProjectFolders()`. The user chooses where (File System Access API, `showDirectoryPicker`, Chrome/Edge). It creates `<Project name>/README.txt` plus one numbered subfolder per evidence source that applies to the chosen system type (`SOURCES` filtered by `sourceNeed`), plus `NN Agent results`. Each subfolder has `What to put here.txt` (Required/Recommended, who provides it, what to put, what the agent detects, no secrets).
- Safe to press again: existing folders and files are never overwritten; only missing ones are added (asks first if the project folder exists). Folder names are cleaned for Windows (`safeFolderName`).
- Browsers without folder access (Firefox, Safari) get `<Project name>.zip` with the same structure (`zipFolders`, a small store-only zip writer). Recorded in `M.projectFolder` and the history.
- Tested in the browser with the Origin Private File System as the chosen folder (11 folders for a web system, second press adds 0), zip opened by Go's `archive/zip` with valid CRCs. The real folder dialog was not clicked through (native dialog). localStorage restored. Not pushed.

### 2026-11-17 — Agent backend in Go: all 15 agents built
- Owner's request: "make all the agents", "I want backend in go". New folder `backend/` (Go 1.26, module `github.com/mjaffry01/mindsuite-doctrine-of-legacy-modernization/backend`, Anthropic Go SDK v2). `backend/README.md` explains running, the API and the safety rules.
- `internal/agents`: evidence store (5 message types, append-only, JSONL per run), codebase loader (folder / zip / uploaded files, in memory only), the 9-step run (`run.go`), and the agents: orchestrator, inventory, supply chain (Maven/Gradle/npm/pip/NuGet/Go + end-of-life rules), structure (imports → module graph, Tarjan cycles, architecture style), data access (SQL, JPA/EF/Django/Mongo mappings, DDL, procedures, shared tables), interfaces (Spring/JAX-RS/ASP.NET/Express/Flask/Django/Go routes, outbound hosts, Kafka/JMS/Rabbit/MQ, SOAP, file transfer), security (login methods, role checks with nearby endpoint, secrets file:line only, weak crypto, SQL concatenation, CSRF off, CORS *), quality (tests per module, hotspots, duplication, dead-code candidates, Git churn + bus factor), platform (Docker, Compose, Kubernetes, Terraform, CI, environments, connections, caches, schedules, app servers), legacy (COBOL, JCL, RPG, PowerBuilder), business rules (Claude Opus 5.5, structured outputs), domain & process (Opus 5.5, facts only, no code), assessment, verifier (re-opens every citation; Sonnet 5.5 second opinion on model claims; evidence check for derived claims) and report writer.
- Claude is used only with `ANTHROPIC_API_KEY` on the backend **and** the per-run consent box; secret lines are never quoted or sent; every request is listed (`sentToModel`). POST needs `X-MindSuite` + an allowed origin.
- `index.html`: 02 Code analysis has "Run the full agent team" (`agentPanel()`): backend address, consent, Git option, choose folder / zip / path, live 9 steps and message log (SSE), results (rules, business story, questions, what was sent), report download, "Add results to the inventory" (`agentImport()`, `origin:'agent'`, never overwrites protected statuses). State: `M.agentBackend`, `M.agentRun`.
- `agents.html`: statuses now "Running today" (supply chain and legacy "Partly"); model table and "What is running today" rewritten.
- Tested: `go test ./...` (sample system in `testdata/sample` with known answers; verifier rejects bad citations; upload keeps folder paths; other origins refused). Browser: path run and folder upload, live steps, import, mobile width, console. **Not tested: the Claude-powered agents (no API key here).** Not pushed.

### 2026-11-17 — How the agents run the analysis (goal + 9 steps)
- `agents.html`: new sections "The goal of the code analysis" (4 goals) and "How the analysis runs, step by step" (`STEPS`, 9 steps: receive → plan → collect facts → choose what to read → understand the business → assess → verify → write report → validate with people; each with the goal it serves, agents as clickable chips, what the client sees, and built/partly/planned status), plus coverage and "what code alone cannot tell". Deep links: `agents.html#<agentId>` selects that agent (`fromHash`). `index.html`: `openAgents(id)` takes an optional agent id; `producedBy(ids)` adds a "Produced by: … agent" line under each Code analysis report heading that opens the agent page at that agent. Not pushed.

### 2026-11-17 — Agent team button moved under the Source code card
- Owner's request: the agent team button belongs under the Source code pane. It is now a full-width button at the bottom of the Source code card on 01 Record evidence ("See the agent team that will study your code ↗"). Removed from the top bar and the Connect-sources heading. Still also on 02 Code analysis.

### 2026-11-17 — "Agent team ↗" button in the top bar
- Always-visible button in the header (next to "New project") opens `agents.html` in its own window; also still on 02 Code analysis and 01 Record evidence → Connect sources. On phones the PREVIEW label is hidden to make room.
- Reminder: none of this is on the live GitHub site until it is pushed.

### 2026-11-17 — Client-facing page: how the agents study the code
- New `agents.html`: a standalone page opened in its own window by "How the agents study your code ↗" (02 Code analysis header and 01 Record evidence → Connect sources). `openAgents()` in `index.html`.
- Contents: four rules (read-only, facts before opinions, every claim cited, people decide); an interactive SVG system diagram (click an agent to highlight its inputs and outputs); a detail panel (does, reads, produces, receives from, sends to, model tier, safeguard, status); the five message types and an example finding; one finding end to end; cards for all agents; model tiers; an honest "what runs today".
- `AGENT-DESIGN.md` now includes the communication model and model tiers.

### 2026-11-17 — Agent design for code analysis
- Added `AGENT-DESIGN.md`: 15 agents (orchestrator; inventory, supply chain, structure, data access, interfaces, security, quality & change safety, platform & config; business rules, domain & process; legacy-language specialists; modernization assessment; verifier; report writer). Covers what each reads and produces, built/partly/not status, run order, model choice, where it runs, and build order (verifier rules first, then the business-rules agent).

### 2026-11-17 — Code analysis report (Deliverable 1.4)
- New step **02 Code analysis**; later steps renumbered (03 inventory … 10 sign-off). A link from the Source code card opens it.
- The scan now analyses each code file (`analyzeFile`) and builds (`analyzeCode`): modules (by package, namespace or folder; shared libraries such as common/util are flagged), the module dependency graph from imports, cycles (Tarjan), a code-to-table map from SQL in string literals plus JPA `@Table` / EF `[Table]` (matched against known tables from SQL files and the database export), stored-procedure calls, test files per module, complexity hotspots (decision points), and per module an extractability score (0–100), effort (S/M/L/XL) and the suggested extraction order.
- It feeds the inventory (`applyCodeAnalysis`): proposed bounded contexts per module (only when no human contexts exist), module components, tables with writers and readers as contexts (so Phase 2 sees shared data), and risks (shared tables, cycles, no tests, very complex files).
- Report page: summary metrics, "Where to start", SVG dependency graph, modules table, data-access table, hotspots.
- Still missing for 1.4: dependency licences and vulnerabilities, Git history hotspots and bus factor, dead code.

### 2026-11-17 — What the client gets (re-established)
- Added `PRODUCT-OUTCOMES.md`: the three deliverables (Current-State Dossier, Target Decision Record, Modernization Roadmap & Approval), the quality bar (citation, basis, confidence, named validator), status per section (built / partly / not yet), minimum client inputs, and the biggest gaps in order.
- The backlog in section 4 should follow the "Biggest gaps" list in that file: code analysis report first, then the business story (AI agent).

### 2026-11-17 — Running handoff log
- Added this `HANDOFF.md`, plus a permanent backup copy in `Downloads\Legecy Modernization`.

### 2026-11-17 — Client inputs asked at the right step
- Start: **Client brief** (why now, hard deadline, success criteria required; previous attempts and access optional). Start is incomplete without it.
- Source cards: **"Facts only you know"** (tests, release, environments; years of history, archiving, data quality; user groups, unused features; connected systems, partner notice; audit reports).
- Validate: **Regulation & compliance**; per context: who maintains it, key-person risk.
- 06 Rules: **Budget, risk and standards** (risk appetite affects scoring; growth % sizes hosting for year 3) and **Business case** (yearly run cost).
- 08 Roadmap: **Calendar & capacity** (freeze months, events, delivery teams), with wave warnings (after the deadline, too many contexts per team).
- 09 Sign-off: **"What the plan must achieve"** (why now, deadline, run cost, success criteria) plus readiness checks.
- New foundations from facts: compliance controls, data migration with history/cleansing, a regression safety net, release automation, knowledge transfer.

### 2026-11-17 — Database card parser and "How to extract" guides
- Database card: **Upload export (.sql / .csv)** parses DDL and catalog CSV in the browser (tables, views, code, jobs, links, admin grants, rows and sizes) into the inventory, risks, technology and capacity.
- Downloadable read-only catalog scripts for Oracle, SQL Server, PostgreSQL, MySQL/MariaDB and Db2.
- **"How to extract"** guide on every non-code card, with commands and **verified official links** (Oracle, Microsoft, PostgreSQL, MySQL, IBM, Kubernetes, AWS, Google Cloud, Okta, OpenSSL, Prometheus, Datadog, Dynatrace, Splunk, BMC, Broadcom, Atlassian, draw.io).
- Cards show **"From code: …"** when the code scan already found evidence (identity, environment, messaging, network, batch).

### 2026-11-17 — Evidence tailored by system type
- Start asks for the system type (multi-select; the agent suggests it from the scan).
- 10 sources (added batch, network edge, documents, incidents). `SOURCE_NEED` marks each Required / Recommended / Not applicable per type; `TYPE_NOTES` gives type-specific guidance; `TYPE_QUESTIONS` adds type-specific questions in Validate.
- Every card: upload export, "No access? Answer instead" (saved with the person's name), and "Not applicable" (a reason is required).

### 2026-11-16/17 — Record evidence redesign (minimum input, agent detects, human validates)
- Single Phase 1 input page with 4 steps (Start, Connect sources, What the agent found, Validate).
- **Code scan part 2:** config and deployment files, database connections, caches and caching strategy, brokers, login methods, authorization checks, endpoints, outbound hosts, Kubernetes/Terraform/images, **hard-coded secrets (file:line only)**.
- **Collector bundle importer** + sample bundle; `COLLECTOR-SPEC.md` written.
- Added the **Security & identity** evidence view (9th view).

### 2026-11-16 — Earlier work in this session (summary)
- Three phases: Discovery plan / Deciding / Plan, with steps 01–09.
- System inventory (contexts, components, data stores, interfaces, rules, events, glossary, risks, capacity) and context map.
- Phase 2: organization rules, people, ranked target options with reasons, cloud service mapping, decision records. A review fixed 12 issues: ties, stale choices, rule violations, rationale required, and others.
- Phase 3: roadmap waves, foundations, sign-off with fingerprint, print/export.
- Capacity & service levels (traffic, peaks, data, availability, RTO/RPO) → hosting estimate, freeze windows; **CAP/PACELC** consistency trade-offs.
- In-browser code scanner (folder/zip), merged 3 upstream commits (curated `TECH_PREFERRED` lists now order the tech picker).
- Colour-coded inputs; `DEVELOPER-GUIDE.md` written.
