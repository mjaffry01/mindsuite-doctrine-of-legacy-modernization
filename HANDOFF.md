# HANDOFF — running work log

**Purpose:** if the current AI session runs out, another model or developer can continue from here. Keep this file updated after every change. Newest entries go at the top of the change log.

Read in this order: **this file → `PRODUCT-OUTCOMES.md` (what the product must deliver) → `AGENT-DESIGN.md` (the agents behind code analysis) → `DEVELOPER-GUIDE.md` (how the code works) → `COLLECTOR-SPEC.md` (collector to build) → `README.md` (the doctrine).**

**Before working on any .NET or ASP.NET discovery, target decision or migration plan:** Read the [Legacy .NET modernization notes](DOTNET-MODERNIZATION-NOTES.md). Use the evidence checklist before proposing changes. Confirm the deployed framework and hosting separately from the project file targets. Keep cited findings, human decisions, task acceptance tests and rollback criteria in the active project workspace. Verify current Microsoft tooling and lifecycle guidance when making a recommendation.

---

## 1. Where things are

| What | Where |
|---|---|
| GitHub repo | `mjaffry01/mindsuite-doctrine-of-legacy-modernization` (branch `main`) |
| Live site (GitHub Pages) | https://mjaffry01.github.io/mindsuite-doctrine-of-legacy-modernization/ |
| Working copy (temporary!) | `%TEMP%\lm-repo` (= `C:\Users\X1CARB~1\AppData\Local\Temp\lm-repo`) |
| Permanent backup of the working files | `C:\Users\X1 CARBON\Downloads\Legecy Modernization\` (a copy of `index.html`, the `.md` files; refreshed with each log entry) |
| Local preview | `python -m http.server 8768` in the working copy → http://localhost:8768/ (also the `legacy-modernization` entry in `.claude/launch.json`) |
| The app | `index.html`: a single static file, plain JS, no build, project state in `localStorage` keys `mindsuite-workspace-v1:<id>` (previous draft migrated on first open) |
| Agent teams | **One folder per team**, backend and web app alike. Backend: shared core `backend/internal/agents/`, teams in `backend/internal/teams/{code,database,infrastructure,messaging,identity,telemetry,batch,network,documents,incidents}`: all ten evidence teams implemented. Web app: `teams/shared/` (backend connection, explanation-page engine), `teams/<team>/panel.js` (the panel in the app) and `teams/<team>/agents.html` (the explanation page). See `backend/README.md`. |
| Agent backend (Go) | `backend/`: runs the code (15), database (14), infrastructure (13), messaging & caching (12), identity & access (12) telemetry (11), batch (10), network (11), documents (12) and incidents (11) teams. `cd backend && go run ./cmd/mindsuite-agents serve` → http://127.0.0.1:8090 (also `mindsuite-agents-backend` in `.claude/launch.json`). `go test ./...` must pass. See `backend/README.md`. |

**Git baseline (2026-11-17):** `main` = `origin/main` at `926eec4` (all ten evidence teams, workspace UI refresh). See the latest change log and working tree for changes after that baseline.

---

## 2. Owner's preferences (follow these)

- **Do not push to GitHub until the owner says "push it".** Edit and test locally first.
- Plain, non-technical wording in the UI. Clear, colour-coded inputs (orange = required and empty, green = filled).
- Principle of the product: **take the minimum from the user, let the agent (scanner/collector) detect the maximum, then a human validates.**
- Selling story: **scan the current state → decide the target → plan the modernization.** Be honest in claims: the code scan is real, target ranking is rule-based, and the Claude integration is implemented but real-key validation remains pending.
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
- **Not built yet:** the Discovery Collector itself, Documents & diagrams and Incidents & changes teams; real-key validation of the implemented Claude agents. The other six non-code/database teams now have backend export parsers.
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

### 2026-11-18 — Results saved per project and agent team; code report covers every evidence area (Claude)
- **Runs folder:** the local backend saves results in `C:\Users\X1 CARBON\Downloads\Legecy Modernization\runs` (`-runs` in `.claude/launch.json` "mindsuite-agents-backend", which runs `go run` from this repo).
- **Layout** (`backend/internal/server/workspaces.go`): `runs/workspaces/<project name> (<first 8 of workspace id>)/` holds `workspace.json` (id and name) and ten team folders: `01 Source code`, `02 Database`, `03 Environment and infrastructure`, `04 Messaging and caching`, `05 Identity and access`, `06 Telemetry`, `07 Batch and scheduling`, `08 Network and edge`, `09 Documents and diagrams`, `10 Incidents and changes`. Each run is `<team folder>/<run id>/` with `result.json` and `report.md`.
  - The web app sends `project=<name>` with `workspace=<id>` (`teams/shared/agent-backend.js`, cache key `project-folders-1`). The folder is found by its id suffix and checked against `workspace.json`. When the project is renamed, the folder follows, unless a run is still writing.
  - Folders in the earlier layout (`workspaces/<id>/<run>/`) are renamed and their runs sorted into team folders on first use. Health reports `projectFolders: true`.
- **Code team** (`teams/code/areas.go`, wired in `team.go`): new findings from code:
  - `telemetry`: logging, APM and metrics libraries, health checks, CICS WRITE OPERATOR / journal, WTO;
  - `network-target` (addresses in config files; schema hosts ignored), `tls-config`, `edge-config` (rewrite and proxy rules, CICS TCPIPSERVICE / URIMAP, IIS bindings);
  - `document` (Markdown, documents and diagrams, with Mermaid / PlantUML counted);
  - `change-record` (change logs), `incident-ref` (ticket and defect references in comments), `known-issue` (TODO / FIXME per module);
  - `messaging` for IBM MQ calls and CICS transient data queues in COBOL / PL/I, and `cache` for CICS temporary storage;
  - `schedule` for scheduler exports (Control-M, CA-7) kept with the code.
- **Code report** (`report.go`): new part "What the code shows for each evidence area", one section per team in the order above, each listing its findings with the most telling kinds first. An empty area says which team to run, and with what evidence. The former Data / Interfaces / Security / Platform sections moved into it; endpoints are under "Online entry points and screens".
- Tests: `teams/code/areas_test.go`, plus `TestProjectFoldersByNameAndTeam` (name, team folders, legacy migration, rename). `go vet` and `go test -count=1 ./...` pass.
  - CardDemo without Claude: 1,184 findings, 249/249 files. It shows Database (IMS, DB2, data sets), Messaging (3 MQ programs, 1 CICS TD queue), Identity (sign-on), Batch (2 scheduler exports and the JCL jobs) and Documents (6 READMEs and the data-model diagram). Infrastructure, telemetry, network and incidents are reported as not found in the code.
- Backend restarted from this source on 127.0.0.1:8090. A test run with a throwaway workspace created the expected folders; the test folder was removed.
- The earlier saved runs (93 files) were moved to the Windows Recycle Bin at 22:05 by someone other than Claude, so the runs folder started empty. They can be restored from the Recycle Bin if wanted.

### 2026-11-18 — Infer business use from code without Claude (Codex)

- The code team now traces C# and VB.NET screen or controller calls to matching service or repository methods, scoped to the enclosing project. It reports likely purpose and observable workflows with file citations and medium confidence. Sparse evidence stays an open question in the report. Claude can still enrich the story when it is enabled and available.
- The eShop configuration explanation requires both catalog service registrations, the UseMockData setting and startup code, plus persistence evidence. Deep reading prioritizes files cited by business findings and skips common bundled UI libraries.
- Verified with go test ./... and a fresh full eShop run (116 files) without Claude. Run 20261118-160603-5d5847f761dc produced an inferred product-catalog purpose, four workflow findings and the mock-versus-SQL storage rule. The report is in the existing eShop workspace and under outputs/eshop-experiment in the Codex workspace.
- The local backend is running the updated build and retains the earlier report directory.


### 2026-11-18 — Local runs folder moved (Claude, local setup only)
- The backend's results folder (`-runs`) is now `C:\Users\X1 CARBON\Downloads\Legecy Modernization\runs`. It was in a temporary Claude scratchpad, which can be cleaned up. All 93 files were copied and checked identical, and the old folder was removed.
- Restarted the running backend (`work\project-workspaces\mindsuite-agents.exe serve` on 127.0.0.1:8090) with the new path, and updated the "mindsuite-agents-backend" entry in `.claude/launch.json`. Old runs and workspace runs (`runs/workspaces/<id>/`) open from the new folder.
- No code change. The runs folder holds project results, so it stays out of git.

### 2026-11-18 — Code team: VB.NET desktop sample (Claude)
- Sample: github.com/rabbitDLaLune/vbnet-inventory-management-system (VB.NET Windows Forms, SQL Server LocalDB), run in Chrome as project "Inventory Management System (VB.NET sample)". Before: 22 findings, which missed most of the app. After: 37 findings, all accepted, no expert question.
- Fixes in `backend/internal`:
  - `teams/code/data.go`: SQL in strings that span lines is read (`multiString`: VB.NET, C# `@"…"` / `"""…"""`, Java / Kotlin / Python `"""`, JS / TS / Go backticks), including `UPDATE x` with `SET` on the next line. The sample now shows writes to 4 tables and reads of `Users`.
  - `teams/code/data.go`: Windows Forms screens (VB `Inherits …Form`, C# `class X : Form`) are listed as `screen` findings.
  - `teams/code/security.go`: the login method "Database-stored passwords" also matches `PasswordHash` / `HashPassword(`. New `weakPasswordHashes`: a code file that hashes passwords with MD5 / SHA-1 / SHA-2 and has no salt or KDF gets a `weak-crypto` finding, "Weak password storage" (severity high).
  - `teams/code/platform.go`: SQL Server `Data Source=(LocalDB)\…` is recognised, with a note when the database file ships with the app (`AttachDbFilename`).
  - `agents/codebase.go`: the folder that holds every file (a GitHub zip's `repo-<sha>/`) is not a module; it stops at `src/`, `app/`, `lib/` and mainframe type folders. Database files (`.mdf`, `.ldf`, `.mdb`, `.accdb`, `.sqlite`, `.bak`, `.dmp` …) are skipped as "database file (business data, not read)".
  - `teams/code/structure.go`: a single module with layer folders reads "Single module, layered inside: Data (7 files), Forms (14 files) …".
- Tests: `teams/code/desktop_test.go` (a small VB.NET app in a zip folder), `teams/code/multistring_test.go`. `gofmt`, `go vet` and `go test -count=1 ./...` pass (12 packages).
- **Claude key rejected:** in this run the code team's "send code excerpts to Claude" box was ticked (not by Claude; it was off in the first run). 12 code excerpts, the facts summary and the report request went to api.anthropic.com, and all returned **401 Unauthorized**. The key in `~/.mindsuite/claude-key.xml` does not work: make a new key and store it again. Codex saw the same 401 (entry below). The UI shows "14 requests sent to Claude, 0 business rules" with no error; the 401 is only in the run log.
- Backend: Codex's backend on :8090 was left running (old build). The new build runs on **127.0.0.1:8091** (runs folder in Claude's scratchpad), and this project's backend address is set to it.

### 2026-11-18 — Read agent reports inside MindSuite (Codex)

- Code analysis now loads the workspace report list automatically. View report opens the written report, searchable findings with citations and verdicts, and agent/file coverage. The code panel also opens its latest report and can use a browser copy when offline. Saved MindSuite result JSON can be opened for review without importing it.
- Script versions keep old browser connection code from loading alongside the new app. This stale script had omitted the workspace ID and made saved reports appear missing.
- Verified three saved reports in the eShop workspace, the .NET Framework finding and its exact citation, and the agent coverage view. The other workspace checked had no stored runs.
- eShop experiment results are under outputs/eshop-experiment in the Codex workspace, including a source walkthrough. The baseline ran against 116 files. The Claude pass received HTTP 401 from Anthropic and produced no model findings. Bundled JavaScript occupied all 12 deep-reading slots and generated 64 false stored-procedure findings. A focused source-only run excluding Scripts reduced those false matches to zero. Fix file selection and misleading predicates before repeating the live Claude comparison.


### 2026-11-18 — Fixed the missed-password bug; tests for the mainframe readers (Claude)
- `internal/agents/helpers.go`: new `SecretMatchIn(pattern, lang, line)` and `LooksSecretIn(lang, line)`. The `password = <DATA-NAME>` exception now applies only to COBOL, COBOL copybook and PL/I files; `pwd ==` comparisons are still skipped everywhere. `db.password=PROD-DB-2024` in a config file is reported again.
  - `LooksSecret(line)` (no file type) is strict again: it is used to keep lines away from the model, so it does not apply the COBOL exception. `Cite` and the code team's secret agent (`teams/code/security.go`) pass the file's language.
- `teams/messaging/code.go`: MQ calls were only found in COBOL's `CALL 'MQGET'` form; PL/I (`CALL MQPUT(...)`) and C (`MQGET(...)`) were missed although the code said it read them. Both forms are now found; C headers (`.h`, `.hpp`) are skipped because they only declare the API.
- New tests: `agents/helpers_test.go` (secret rule, including the regression), `teams/batch/mainframe_test.go` (Control-M XML with folder, conditions and a missing event; CA-7 triggered jobs; JCL members, JCL-only jobs, PROCs ignored), `teams/database/embedded_test.go` (EXEC SQL programs and tables, comments ignored), `teams/messaging/code_test.go` (COBOL, PL/I, C; headers and non-MQ programs ignored).
- Checked: `gofmt`, `go vet` clean; `go test -count=1 ./...` passes (12 packages).
- This closes the open issue and the test gap in the entry below.

### 2026-11-18 — Claude caught up with the Codex changes (review only, no code change)
- Codex work in the working tree since `926eec4`: **project workspaces** (entry below) and **mainframe support**, which had no log entry:
  - loader (`internal/agents/codebase.go`): BMS maps, CICS CSD, IMS DBD/PSB, Assembler, control cards, CA-7/Control-M exports, `.proc`/`.dcl`; JCL recognised from content (`.prc` is shared with PL/SQL); mainframe data set copies (`.ps`, `.dat`, VSAM/GDG, `ebcdic/`) skipped as business data; type folders (`cbl/`, `cpy/`, `jcl/` …) are not treated as modules;
  - batch: Control-M XML (DEFTABLE), CA-7 LJOB listings and JCL members linking jobs to programs (`teams/batch/mainframe.go`);
  - database: programs with embedded `EXEC SQL` and the tables they name (`teams/database/embedded.go`);
  - messaging: MQ API calls in COBOL, PL/I and C programs (`teams/messaging/code.go`);
  - code team: assess, data, platform, quality, report, security and structure updated for mainframe code;
  - secrets: new `agents.SecretMatch` ignores `password = <COBOL data name>` and `pwd ==` comparisons.
- Checked: `gofmt`, `go vet` clean, `go test -count=1 ./...` passes. Workspace and run IDs are strictly validated before any path is built (no path traversal).
- **Open issue:** `SecretMatch` also skips real unquoted passwords that look like data names, e.g. `db.password=PROD-DB-2024` in a `.properties` file (tested: not reported; `db.password=Prod2024x` is). Fix: apply the data-name exception only to COBOL / PL/I / copybook files.
- **Gap:** no tests yet for the Control-M XML, CA-7, JCL, embedded-SQL and MQ-in-code readers.
- Agent counts: infrastructure, messaging and identity have 12 agents each (120 in total); an earlier Claude answer said 11 (117) by mistake.

### 2026-11-18 — Legacy .NET modernization research (Codex)

- Read the C# Corner Copilot walkthrough and checked it against Microsoft guidance. See [Legacy .NET modernization notes](DOTNET-MODERNIZATION-NOTES.md) for an evidence checklist, .NET discovery gaps, a proposed migration task format and a first pilot.
- For future discovery work, build on the existing Code team. Keep the assessment, owner-approved decisions, tasks and validation evidence in the client workspace. Evaluate deployed Windows/IIS and runtime evidence separately from what the repository targets.
- The article's Upgrade Assistant CLI is deprecated. Recheck current Copilot capabilities and .NET support dates before choosing a migration path. Research only.
- Added the .NET notes to the read-first instructions for Codex, Claude and future maintainers.



### Project workspaces (Codex)
- New project opens a named workspace and Projects switches between saved workspaces. The old browser draft is copied into the first workspace. The reference scenario opens separately.
- The workspace ID stays fixed for each page instance. The full browser state, including evidence, review history, decisions and sign-off, is stored under that ID. Optional evidence folders use the project name and ID suffix. Existing folder links stay intact.
- All ten agent panels send the workspace ID on runs, polling, live events and downloads. The backend saves workspace results under `runs/workspaces/<workspace-id>/<run-id>/` (relative to the configured runs folder). Code analysis → Project analysis history lists the active workspace's runs and opens pending results or downloads completed reports. The original draft can still retrieve referenced legacy run IDs. Restart the local backend after updating the source.
- Browser saves are local to the current browser and origin. Workspace IDs organize projects and do not provide user authentication. Keep existing deployment authentication guidance for any network server.
- Verified desktop project creation and switching, separate sample workspace, 375 px dialog with no overflow, no console errors, JavaScript syntax and Go tests for project scoping, persistence and restart recovery. The updated local backend is running on port 8090 with the existing runs folder. All 10 prior results were found on disk and an older report result was retrieved after restart. Existing mainframe work was preserved. No publication.


### 2026-11-18 — Pushed: commit 926eec4 on main
- Owner said "Push it". Commit `926eec4` (config file `backend/mindsuite.json`, encrypted local key via DPAPI, configurable tier models, `backend/DEPLOY.md`, `server.example.json`, `.gitignore`) pushed to `origin/main`; tests passed first and no key-like strings were in the files. GitHub Pages run 44273887806 succeeded; `backend/DEPLOY.md` is on GitHub. Only this HANDOFF note is uncommitted.

### 2026-11-18 — DEPLOY.md for DevOps
- Owner wants DevOps to put the Anthropic key in the server environment. That already works (`ANTHROPIC_API_KEY` wins over `apiKeyFile`); no code change.
- New `backend/DEPLOY.md`: key in the platform secret store exposed as `ANTHROPIC_API_KEY`, key-handling rules, build, `server.json` fields, required login + HTTPS in front (no built-in auth), reverse-proxy settings (512 MB uploads, SSE without buffering, `X-MindSuite` header), health check, privacy decision (per-customer backend vs shared server), updating. New `backend/server.example.json`. README links to it.
- Tested: backend started with `server.example.json` and a fake `ANTHROPIC_API_KEY` → log "key from ANTHROPIC_API_KEY", `/api/health` modelAvailable true, allowLocalPath false.
- Hosting not decided yet. Pushed to GitHub on 2026-11-18.

### 2026-11-18 — Config file `backend/mindsuite.json`; Claude key from an encrypted file
- Owner asked for a config file and where to put the key. The plain-text `anthropic.key` option made earlier the same day was removed (owner: "not a good option").
- `backend/mindsuite.json` (committed): `addr`, `runs`, `cors`, `allowPath`, `models` (strong/balanced/fast), `apiKeyFile` (default `~/.mindsuite/claude-key.xml`). Read from the start folder or next to the program, or `-config <file>`; flags override it. The backend refuses a config that contains something like a key.
- Key: `ANTHROPIC_API_KEY` first, else `apiKeyFile`, a PowerShell `Export-Clixml` SecureString file decrypted with Windows DPAPI (`cmd/mindsuite-agents/config.go`, `dpapi_windows.go`; `dpapi_other.go` says to use the env var elsewhere). The key is never logged; the start line names only its source.
- Models: `agents.SetModels`/`agents.Resolve` (`internal/agents/models.go`) map the built-in tier models to the configured ones in the Claude client, `sentToModel`, the run's agent list and `/api/agents`, `/api/health`. Warning if strong = balanced.
- `.gitignore` now ignores `backend/runs/` (saved results).
- Tested: fake encrypted key + model override (`/api/health` and `/api/agents` show the override, start line "available"), key-in-config refused, default config found and decrypted the owner's own key file (key not printed). `go vet` clean, `go test ./...` passes.
- Pushed to GitHub on 2026-11-18.

### 2026-11-17 — Pushed: commit 2b0b40d on main
- Owner said "Push it". Commit `2b0b40d` (batch, network, documents and incidents teams, shared document extraction, team-panel builder, Codex workspace UI refresh) pushed to `origin/main`; GitHub Pages deployed successfully and the new pages and scripts load on the live site. Working tree clean.

### 2026-11-17 — Incidents & changes team (11 agents): all ten evidence cards now have a team
- Owner's request: "Build it" (Incidents & changes). Scope = the card's "Agent detects" line: where the system breaks most often, recurring problems, change frequency and failed changes.
- Backend: `backend/internal/teams/incidents/` (orchestrator, ticket reader, hotspots, recurring problems, restore times, change frequency, failed changes, failure story [Opus], modernization priorities, verifier, report writer). Reads ServiceNow incident/change_request/problem CSV and Jira CSV. Privacy: people columns never kept (only used to scrub names from titles), descriptions/comments not read, ticket rows cited without quotes. Sample by `testdata/make_sample.py` (names, an e-mail and a password planted; the test checks none leak). `go test ./...` passes (12 packages).
- Web app: `teams/incidents/{panel.js,agents.html}` via `makeTeamPanel`; card Incidents & changes has run / see buttons; all 10 team pages link each other. Import adds risks (recurring problems, failed changes, emergency changes, peak periods, …).
- Tested in the browser: 42 incidents, 30 changes, 2 problems → 31 of 31 findings accepted → import added 10 risks; no names, e-mail or password in the result. localStorage restored.
- Pushed later the same day in commit `2b0b40d`.

### 2026-11-17 — Documents & diagrams team (12 agents)
- Owner's request: "Build Documents". Scope = the card's "Agent detects" line: business rules, processes, glossary and intended architecture, read with citations.
- Backend core: `backend/internal/agents/documents.go` extracts text from .docx/.pptx/.vsdx/PDF (no new dependencies); `codebase.go` treats them as `Document` (20 MB limit, `MaxDocBytes`), adds .drawio, .htm, .log, .jil, .cfg/.conf, .zone; the server's upload limit follows `MaxDocBytes`. Old .doc/.ppt/.vsd and scanned PDFs are skipped with a reason and asked about.
- Backend team: `backend/internal/teams/documents/` (orchestrator, document reader, freshness & coverage, rule statements, glossary, processes, intended architecture, business rules [Opus], business story & conflicts [Opus], evidence value, verifier, report writer). Sample made by `testdata/make_sample.py` (Word, PowerPoint, Visio, PDF with and without text, Confluence HTML, Markdown, draw.io plain and compressed, an old .doc, a password in a properties file). `go test ./...` passes (11 packages).
- Shared wording: the step-5 skip reason is now "the client did not allow sending excerpts to Claude" (was "code excerpts"), since teams also send document passages.
- Web app: `teams/documents/{panel.js,agents.html}` via `makeTeamPanel`; `teams/shared/team-panel.js` gained `cfg.maxBytes` (default stays 2 MB; documents use 20 MB). Card Documents & diagrams has run / see buttons; all 9 team pages link each other.
- Tested in the browser: sample upload → 5 documents, 3 diagrams, 2 not readable → 49 of 49 findings accepted, 1 expert question, no password in the result → import added 5 rules, 4 terms, 2 processes (events), 8 intended components, 3 risks. localStorage restored.
- Nothing committed or pushed. Next: Incidents & changes.

### 2026-11-17 — Claude caught up with the Codex GUI work; batch and network tested end to end
- Reviewed the Codex changes: `assets/workspace.css` + `assets/workspace.js` (presentation only; `enhanceWorkspace()` / `enhanceContemporaryUI()` after `renderNav()`), header and skip link in `index.html`, refreshed `teams/shared/team-page.css`. No conflicts with the agent code: the agent panels, card buttons (short visible labels, full accessible names) and team pages work unchanged.
- Checked: `go vet` and `go test ./...` pass (server + 8 teams); all `assets/*.js` and `teams/**/panel.js` parse.
- **Batch and network now tested end to end in the browser** with the redesigned UI (backend restarted): upload of each team's sample → live run → results → import. Batch: 37 findings, 14 jobs, window 23:00–06:00, 5 of 185 runs failed; import added 14 batch components and 10 risks. Network: 29 findings, 6 entry points; import added 10 interfaces and 10 risks. No page overflow; no console errors except the expected health-check retries while the backend started. localStorage restored.
- Nothing committed or pushed (working tree: Codex GUI files, batch, network, team-panel builder). Next: Documents & diagrams, then Incidents & changes.

### 2026-11-17 — Contemporary visual redesign — completed and verified (Codex)
- Owner requested a more contemporary appearance after the first GUI uplift. Switched to a light compact sidebar, indigo accents, restrained lavender surfaces, stronger page titles, icon navigation, tile-style system choices and quieter forms.
- Added a Discover / Decide / Plan switcher in the desktop header. It calls the existing phase navigation and reflects the active phase. Source-card agent actions have concise visible labels while retaining their original accessible names and click handlers.
- Files changed: `assets/workspace.css`, `assets/workspace.js`, `teams/shared/team-page.css`. The final theme rules are marked "Contemporary workspace refinement"; the new function `enhanceContemporaryUI()` is called at the end of `enhanceWorkspace()`. No changes to index.html, business state, scanners or backend logic.
- Verified all 10 workspace screens at 1440×900 and 375×810: no page overflow or console errors. Tested desktop stage switching, active-stage highlighting, mobile menu, source search and network team navigation. Network explanation page checked at both sizes. New JavaScript parses; index.html is byte-for-byte unchanged from this turn’s baseline. No saved answers, consent, backend runs or AI calls changed. No publication.
- Screenshots: `outputs/mindsuite-contemporary-desktop.jpg` and `outputs/mindsuite-contemporary-mobile.jpg` in the Codex chat folder. Handoff and changed source files synchronized to the existing Downloads backup and `outputs/mindsuite-current/`.
- Continuation: the original GUI uplift below is superseded visually by this entry. Keep both presentation enhancements; do not run the old mockup generator. Documents/Incidents teams, actual backend-run validation and live-model testing remain pending.


### 2026-11-17 — Codex GUI uplift — complete and browser-checked
- Owner asked: "uplift the GUI and keep updating the handoff" so Claude and Codex can alternate on the project. Work remains local; do not push without the owner's instruction.
- Added `assets/workspace.css` and `assets/workspace.js`: shared visual tokens, dark phase navigation, responsive mobile menu, readable cards/forms/tables/dialogs, evidence-source search + needs-input filter, clickable discovery steps, accessible focus and reduced-motion behavior. These files are loaded by `index.html`; the final `render()` invokes `enhanceWorkspace()` after `renderNav()`.
- Source collection details are collapsible; all original buttons/handlers remain. UI-only filter state stays in memory and never writes project data. Verification used read-only navigation and presentation controls; no sample reload, project reset, import, form save or consent changes. Existing browser answers were left intact. No scanners, AI prompts, backend logic, catalog or decision rules changed.
- Updated `teams/shared/team-page.css` so all eight agent explanation pages use the same visual language.
- Source of truth is still this working copy, not the old `outputs/index.html` from the earlier Codex mockup. Never run the earlier `work/build-mockup.cjs` against this project: it would replace the newer product.
- Backups: changed files mirrored to `C:\Users\X1 CARBON\Downloads\Legecy Modernization\`; complete source snapshot (frontend, eight teams, backend and docs; excludes Git and runtime data) at `C:\Users\X1 CARBON\Documents\Codex\2026-11-16\https-mjaffry01-github-io-mindsuite-doctrine\outputs\mindsuite-current\`. The active working copy remains `%TEMP%\lm-repo`.
- Validation: 21 frontend/inline JavaScript blocks parse successfully. An exact comparison confirms index.html changed only the stylesheet/script links, brand markup, skip link and final render hook; existing business logic is byte-for-byte unchanged. All ten workspace screens were checked at effective CSS viewports 1440×900 and 375×810 with no horizontal page overflow or console errors. Verified source search, no-match result, needs-input/all toggles, discovery-step jumps, collection-detail disclosure, batch-panel navigation, mobile menu open/close and evidence drawer. Eight agent explanation pages checked at desktop; Network page additionally checked at 375px. Backend runs and Claude calls were NOT tested in this GUI task.



- Next for Claude: continue Documents & diagrams, then Incidents & changes. Batch/network UI rendering was checked here, but their actual export-to-run-to-import workflows still need end-to-end testing. Collector, real-key model validation, repository authorization, licence/vulnerability lookup and full dossier/cost comparison remain separate backlog items.

### 2026-11-17 — Batch & scheduling (10 agents) and Network edge & partners (11 agents); Documents not started
- Owner's requests: "Batch & scheduling", "Network edge & partners", "Documents & diagrams", "Agent detects:" (read as: each team covers exactly its card's "Agent detects" line).
- Backend: `backend/internal/teams/batch/` (Control-M JSON, AutoSys JIL, TWS composer, schtasks CSV, crontab, run-history CSV; streams, window, dependencies, failures and reruns) and `backend/internal/teams/network/` (nginx -T, HAProxy, ingress JSON, DNS zone/CSV, openssl, partner and firewall CSVs, cloud inventory; entry points, DNS/CDN, TLS, partners, firewall, cut-over checklist). Each with a test and `testdata/make_sample.py`. Core loader now reads `.log .jil .cfg .conf .zone`.
- Web app: new shared builder `teams/shared/team-panel.js` (`makeTeamPanel(cfg)`); `teams/batch/` and `teams/network/` (panel.js, agents.html) use it; cards Batch & scheduling and Network edge & partners have run buttons; team switcher has 8 teams.
- Tested: `go test ./...` passes (10 packages). **Not browser-tested** (usage limit). Not pushed.
- **Next:** Documents & diagrams team (plan: core text extraction for .docx/.pdf/.pptx/.vsdx in `internal/agents`, then `teams/documents`: document reader, freshness & coverage, glossary, processes, intended architecture from draw.io/Visio, rules with citations via `agents.ReadRules`, assessment). Incidents & changes team still to do. Browser-test batch and network panels.

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
