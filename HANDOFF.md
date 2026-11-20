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

**Phase 1 · Discovery**
- **Project → Evidence → Reports**
  1. **Project interview:** enter the project name first. Ask for modernization reasons, the current problem and what success looks like. Deadline, system type and scope are also available. Interview answers are saved with the project.
  2. **Evidence:** add files by source. Each card explains its team, runs analysis for that card’s staged files, and shows progress and completed reports. An optional Claude setting under More evidence options applies to the next team launch.
  3. **Reports:** workspace runs and team results, with advanced settings for one team at a time.
- **03 Validate** (one page, three tabs: System inventory · Field findings · Gaps) and 04 Discovery summary. The old page ids `inventory`, `review` and `gaps` still work in `go()` and open Validate on that tab.

**Phase 2 · Deciding**
- 05 Rules & people, in three tabs (`assets/rules-page.js`): 1 Who decides (people) · 2 Technology rules (chips that open a rule editor: Standard, Preferred, Allowed, Needs approval, Retiring, Excluded, each with reason, owner and review date; presets; tech-radar CSV import and export; your own options) · 3 Priorities & limits (weights first, then risk/budget/standards, team skills/data/contracts, and a collapsible Money section with the current run cost).
- 06 Target decisions: each card opens with the **decision advisor** (`assets/decision-advisor.js`): recommended approach with reasons, confidence and missing inputs, one deciding question, three scenarios (Safe / Balanced / Ambitious) with effort, risk and checks, cross-checks on the current target, and Draft rationale; the detailed choices fold under "Adjust in detail". Per context, the approach plus architecture, language, database, integration and cloud. Options are ranked with reasons, cloud services are mapped, and each card shows a hosting estimate and the CAP trade-off.

**Phase 3 · Plan**
- 07 Roadmap: foundations, calendar & capacity, 3 waves, freeze windows.
- 08 Sign-off & export: what the plan must achieve, readiness, approval, print, JSON export.

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

1. ~~Merge 03 System inventory + 04 Review findings + 05 Resolve gaps into one "Validate" page~~ done 2026-11-20 as three tabs (see the change log). Still open: group the records by the 9 evidence views instead of by record type.
2. **Parsers for the other source exports**, in the browser, like the database parser: RVTools/PowerCLI CSV, `kubectl` JSON, cloud inventory CSV, scheduler exports (Control-M JSON, AutoSys JIL, crontab, `schtasks` CSV), Jira/ServiceNow CSV, monitoring CSV (requests per hour → capacity).
3. **Build the Discovery Collector** per `COLLECTOR-SPEC.md` (separate `collector/` folder, read-only CLI).
4. **Infrastructure map** (servers, networks, load balancers, DR) in the inventory.
5. **Security:** TLS checks, OSV dependency vulnerability lookup, licence check of dependencies (needs the owner's OK: sends dependency names outside).
6. **Agent backend next:** test the Claude-powered agents on a real codebase with an API key (prompts in `backend/internal/teams/code/domain.go`, `backend/internal/agents/rules.go`, `verify.go`, `report.go`); Oracle Forms / ABAP specialists; authentication if it is ever deployed beyond localhost; documents (PDF/Word) as evidence.
7. ~~AI agent backend~~ built in Go (`backend/`), see the change log.
8. Cleanup: remove the dead earlier copies of functions that are defined twice (see `DEVELOPER-GUIDE.md` §3).
9. **Technology rules, step 2 (agreed direction, not built):** new decision areas (hosting model, user interface, packaged/SaaS products, mainframe path, batch and scheduling, identity, reporting and analytics) and missing options (Kotlin, PHP, low-code; Db2, MariaDB, MongoDB, Redis, warehouse; GraphQL, gRPC, existing ESB, EDI; Alibaba, Huawei, in-country and sovereign clouds, hybrid). Each needs its own scoring reasons in `SCORE_RULES` and a `CLOUD_SVC` mapping, otherwise it ranks on rules and skills only.
11. ~~Decision advisor Phases B and C~~ built 2026-11-20 (see the change log). Still open: replace the effort rule of thumb with the roadmap's own estimates once they exist.
10. **Technology rules, step 3:** rules scoped to one business area or to regulated data, and version limits ("Java 21 or later").

---

## 5. How to continue safely

1. Read `DEVELOPER-GUIDE.md` §3. **Functions defined more than once: the last one wins.** Edit the last one.
2. Make changes with small, exact string replacements. After each change, extract the `<script>` and run `node --check`.
3. Test in the browser. To test the code scanner without a folder, call `runScan('folder', fakeFiles)`; see the guide §2.
4. **Add an entry to the change log below**, and copy the files to the permanent backup folder.
5. Push only when the owner says "push it". Then: `git add -A && git commit` (end the message with the attribution line used in this project), `git push`, wait for the GitHub Pages build (`gh api repos/mjaffry01/mindsuite-doctrine-of-legacy-modernization/pages/builds/latest`), and check the live URL.

---

## 6. Change log (newest first)

### 2026-11-20 — Doctrine and app cover new product development (Claude)
- **Doctrine** (`README.md`), renamed *MindSuite Doctrine of Legacy Modernization and Product Development*: three kinds of project (modernize an existing system, build a new product, modernize and extend) with what differs per kind; seven principles shared by every project; the legacy chapter kept as "Modernizing an existing system"; new chapters "Building a new product" (product brief, including **deployment type**: where it runs, tenancy, channels, data residency; build / buy / partner; the first release) and "Modernizing and extending"; *Verify before deciding* now also covers product brief answers. `PRODUCT-OUTCOMES.md` notes the new-product deliverables.
- **Start screen** (`assets/project-mode.js`, cache key `mode-3`): a new, empty project shows "What are you working on?" with three cards; the choice is stored in `M.mode` (projects saved earlier count as modernize) and logged as `project:mode`. A Project type switch on the project interview step changes it later.
- **Product brief** for new and extend projects on the project step: problem, users and jobs, key journeys, first release (MVP), success, integrations, build/buy/partner preferences, expected users, peak load, data volume, availability, and **Deployment type** (where it runs, tenancy, data residency regions, channels). Modernize projects get a short *Target deployment type* section. Stored in `M.product`; every change logged as `product:*`. The sidebar says "01 Product brief · n of 8 answered" for a new product.
- **Decisions:** new approaches `build`, `buy`, `partner` in `APPROACH`, `APPROACH_DIMS` (build: all areas; buy: integration and cloud; partner: architecture, integration and cloud), `autoSize`, `autoWave`, `autoPattern`. Each project type offers only its approaches (`approachAllowed`); `suggestApproach` uses `modeSuggest` (core → build, generic → buy, supporting → partner). "Keep current …" options are excluded for a new product. The advisor scores build / buy / partner from strategic value, skills, deadline, risk appetite, deployment and tenancy; for a new product its confidence uses the brief (problem and users, MVP, scale, deployment) instead of code scans; scenarios use buy / partner / build as the tiers; a new cross-check warns when the chosen cloud contradicts the deployment type.
- **Agent team:** the options brief starts with the product facts and deployment (`productFacts`); areas default to include UI and products for new products; the request carries `mode`. Backend: `Approaches` gains build, buy, partner; `ModeApproaches` per project type; `OptionsInput.Mode` (modernize, new, extend; empty = modernize); blueprints outside the project type's approaches are dropped; scouts must respect deployment facts; the architect must make the three tiers genuinely different routes. The verifier now reads fact references tolerantly (`citedFacts`: "f02", "[F3]", "F2, F7", or references in the reason text).
- Checked: gofmt, go vet and all Go tests pass (new: project types, blueprint filtering per type, fact references). In Chrome on :8768 with storage backed up: New project shows the start screen; Build a new product opens the brief (fixed: the first version did not redraw after the choice); a test product brief (stock tracking SaaS, public cloud, India residency, multi-tenant, web and mobile) gave Build / Buy / Partner only, advisor recommending Build with Safe Buy, Balanced Partner, Ambitious Build; two real team runs on :8091: hosting options all in Indian regions; after the fixes 17 of 19 options kept and tiers Safe partner (Zoho Inventory), Balanced build, Ambitious build. Storage restored exactly and the test project removed.

### 2026-11-20 — Explore gives architects a dense comparison matrix (Claude)
- The owner asked for denser hosting choices for architects. Explore now returns a **spec** per choice: 12 aspects (`SpecAspects`: sku, os-runtime, scaling, network, availability, disaster-recovery, deployment, limits, cost-model, compliance, operations, lock-in), each a terse value of at most 160 characters; the specialist is told to write "confirm" rather than guess a number. `cleanSpec` keeps known aspects once each, in matrix order. Compute values may be up to 40 characters.
- The web app shows the choices as a **comparison matrix** (`exploreMatrix`): one column per choice with name, tier, catalogue and rule badges and Add to my options; rows for compute, each spec aspect present, choose and avoid when, fit with fact references, risks and your rules; sticky header and first column; **Download comparison (CSV)**. Units are added only to plain numbers; n/a and none show nothing.
- Catalogue matching now accepts catalogue names anywhere in the option name (overlaps allowed), so "Azure App Service Environment v3" is known; added Static Web Apps, Spring Apps, Azure Red Hat OpenShift.
- Checked: Go tests pass (spec order and duplicates, catalogue cases). A real Explore of "Microsoft Azure" for the VB.NET sample (test facts in memory, storage restored exactly afterwards) returned 4 choices × 12 aspects in 52 s: App Service Premium v3, Container Apps, App Service Environment v3 (Isolated v2), Static Web Apps; no console errors.

### 2026-11-20 — Target options team: AI agents propose technical options (Claude)
- The owner asked for technical options generated by a team of AI agents from the discovery, then for more detail: drill into an option (for example Azure hosting), concrete vCPU / RAM / disk, northbound and southbound traffic components, database and caching strategy, design patterns, caching components and the components teams usually miss, and blueprints in three tiers.
- **Backend** (`POST /api/decide/options`, health flag `targetOptions`; files `internal/agents/options.go`, `catalogue.go`, `design.go`, `internal/server/options.go`). Strict input: numbered facts `F1…Fn` (≤150, ≤300 characters each), technology rules, areas; unknown fields refused. Three modes:
  - **Team run:** option scouts in parallel (balanced model, one per area: hosting, architecture, data, integration, ui, products, legacy; kind product / cloud-service / framework / pattern) → **verifier** (drops reasons that cite no real fact and options left without one; curated product catalogue in `catalogue.go`: known, needs confirmation, or pattern; SKU and version words and bracketed text are ignored) → **rules guard** (applies the organization's rules by area; cloud rules apply to every area; excluded and retiring options are marked excluded) → **architect** (strong model, exactly one Safe, one Balanced and one Ambitious blueprint, cited reasons, migration, effort, cost trend, deadline fit; never an excluded option) → **challenger** (verdict and up to 3 challenges per blueprint). About 7 model requests.
  - **Explore** (`explore`: one option): a specialist (strong model) lists 3–6 choices inside it (services, plans, tiers) with sizing, resilience, choose and avoid when, **compute** (vCPU, RAM GB, disk GB, instances, basis facts or assumption), and **north** / **south** traffic components. Retries once when nothing usable comes back (seen once in real use). 1–2 requests.
  - **Design** (`design`: one blueprint): the solution designer (strong model) returns database strategy, caching strategy and components, design patterns with where they apply, north and south traffic, cross-cutting components by category (identity, secrets, monitoring, delivery, backup-recovery, security, configuration, messaging, other), starting sizes per component and missing facts. The answer is one flat list regrouped in code: the nested schema was rejected by the API as "compiled grammar is too large". Services without compute (CDN etc.) are left out of the sizing table. 1 request.
- **Web app** (`assets/target-options.js`, cache key `options-5`; added under the advisor by `advisedCard`): the **Evidence brief** builds the facts in the browser (system type, technology and versions, components, data stores with sensitivity, interface protocols and directions, events, rule counts, risks, capacity, regulation, residency, skills, constraints, deadline, drivers, knowledge) and never sends business data, people's or partner names, comments, rule texts or unverified claims; file paths become `[file]`. Areas default from the evidence and can be changed. Consent box per run with "See exactly what is sent"; Explore and Design ask for consent per request. Results: agent steps with model, requests and time; blueprints (Safe / Balanced / Ambitious) with verdict, Use this blueprint (sets the approach and every choice that matches the catalogue) and Design this blueprint; options by area with fit reasons and fact references (hover shows the fact), catalogue and rule badges, lifecycle, licence, skills, risks, Add to my options (architecture, database, integration, cloud) and Explore inside. Stored in `d.optionsRun` (`explore`, `designs`); logged as `ai-options`, `ai-explore`, `ai-design`.
- Checked: gofmt, go vet and all Go tests pass (full pipeline, explore with retry, design, refusals, catalogue). Real runs on 127.0.0.1:8091 with the VB.NET sample plus test facts in memory (storage backed up and restored exactly): team run 46–56 s, 7 requests, 21 options, three tiered blueprints (Safe: port to current .NET on Azure SQL; Balanced: web app on App Service, sound; Ambitious: API-first on Container Apps, risky); Explore of Azure App Service gave plans with sizes (for example Premium v3, 2 vCPU / 8 GB / 250 GB, ×2, assumption) and north / south components; Design of the Balanced blueprint gave a full design in 47 s. No console errors. The :8090 backend does not have these endpoints until it is rebuilt.

### 2026-11-20 — Decision advisor Phases B and C; Technology rules easier to read (Claude)
- **Weighted approach scoring:** every approach reason now names its criterion (fit, effort, risk, cost, skills); the score is Σ value × weight / 3, so What matters most in 05 changes the approach advice as it already changed the technology ranking.
- **Robustness** (`robustness(c,d)`): moves each weight by ±2 and reports when the recommended approach or a chosen technology would change ("If running cost counted more, …"). A choice that already loses under the current weights is reported once ("With your current priorities, X already ranks above …"). Shown as Robust / Sensitive under the current target.
- **Portfolio view** (`portfolioPanel(C)`, top of 06): one row per context (approach decided or suggested, cloud, database, language, effort, confidence, status); platforms to run (warns on more than one cloud); languages not yet a team skill; total effort from `autoSize` with a person-month rule of thumb (S 2–6, M 6–18, L 18–48, XL 48–120); **3-year cost** from today's run cost (business case), cost of one person-month and expected yearly running cost after the move (new fields `personMonth`, `targetRun` in `M.decide.businessCase`), with payback. Labelled as estimates, not a budget.
- **Phase C, second opinion from Claude:** new `POST /api/decide/review` (`backend/internal/agents/review.go`, `internal/server/review.go`, health flag `decisionReview`). Input: strategic value, criticality, approach, target option labels, advisor reasons and checks, names of missing inputs, and verified claims only (reuses `SuggestClaim` validation). Unknown fields (a rationale, names) are refused with 400; at most 40 reasons/checks, text cut to 300 characters. Output: verdict sound / questionable / risky, summary, up to 6 challenges with severity and cited claim numbers, missing evidence, questions to ask, up to 2 alternative approaches; cleaned server-side. Web app: `reviewPanel()` under the current target with a per-request consent box, "See exactly what is sent", result kept in `d.review` (marked out of date when the payload changes), each request logged as `ai-review`. Tests: `review_test.go` in both packages.
- **Technology rules tab:** a "Rules in force" table (rule, area, option, reason, owner, review date, Edit) that flags rules without a reason or owner; presets show what they will set before Apply; each area says what it decides and how many rules it holds; option descriptions on hover; search and "Only options with a rule" filter (`rulesFilter`, filters the page without re-rendering).
- Checked: gofmt, go vet and all Go tests pass. My backend on 127.0.0.1:8091 was rebuilt and restarted (health shows `decisionReview: true`); the :8090 backend was not touched and does not have the review endpoint until it is rebuilt. In Chrome on :8768 with storage backed up, on the VB.NET sample: weights changed the approach scores (Rebuild 1.7, Refactor 0.7); robustness flagged the architecture choice; the portfolio and 3-year cost computed (600k today against 540k–840k, payback 2.3–6.0 years with test numbers); one real review call (target, 11 reasons, 2 checks, 0 claims; claude-opus-5-5) returned "questionable" with challenges about the missing code scan, contexts and language, and two alternatives; the consent cleared after one request; rules tab preview, table, search and filter worked; no console errors. Fixed during testing: cost wording when only the low estimate is cheaper, robustness notes when the choice already loses, an empty preset preview box. Storage restored exactly afterwards.

### 2026-11-20 — Decision advisor on Target decisions, Phase A (Claude)
- The owner asked for smarter, more intuitive decision making. Problems found: six independent dropdowns with no complete target, the approach suggested from strategic value only, frequent ties, no confidence, about 2,400 px per card.
- New `assets/decision-advisor.js` (cache key `advisor-4`); `decidePage()` now maps contexts through `advisedCard(c)`, which builds `decisionCard(c)` and adds the panel. No model calls.
  - `approachAdvice(c)`: scores the 7 approaches from strategic value, criticality, modernization drivers (finding `drivers` plus interview reasons mapped by `REASON_TO_DRIVER`), domain knowledge, deadline (interview date, or the advisor's horizon answer), risk appetite and downtime, tests, shared data, rules count, COBOL and skills. Each + / − is a reason shown to the user. A past interview deadline is ignored and reported.
  - `advisorConfidence()`: 8 key inputs, high ≥ 7, medium ≥ 4; each missing input is a link to where it is entered.
  - `advisorScenarios()`: Safe (best of rehost / encapsulate / replatform), Balanced (best middle option), Ambitious (best of rebuild / refactor / replace), plus Retire when the interview says so. `buildScenario()` fills each area with the top option from `rankOptions` / `recommended`, so organization rules apply. Shows fit, effort (`autoSize`), risk and check count; "Use this scenario" sets the approach and every choice (logged as `advisor:scenario:<id>`).
  - `crossChecks()`: rule violations, effort against deadline, serverless on-premises, big architecture against low risk appetite, zero downtime without CDC, rebuild with undocumented rules, refactor or rebuild without tests or with an unknown current language, another cloud than other contexts, ties.
  - `advisorQuestion()`: one question at a time (strategic value, deadline, risk appetite, who knows the rules, then cloud agreement on a cloud tie). Answers are stored where they belong (context or `M.values`, `constraints().risk`, `M.decide.advisor.horizon`, a Preferred cloud rule) and logged.
  - `draftRationale()`: approach reasons, target with the top reason per choice, accepted warnings and the confidence.
- Verified claims and Claude suggestions stay visible; the approach row, technology choices, services, hosting and CAP fold under "Adjust in detail" (open state kept per context while the page is open).
- Checked in Chrome on :8768 with storage backed up, on the VB.NET sample: Low confidence at first (1 of 8); four answers moved it to Medium and the recommendation to Rebuild; the cloud question set Azure as Preferred and all scenarios followed; Use this scenario filled every choice; Draft rationale wrote a cited rationale; checks showed the passed deadline and the unknown language; fold and unfold kept; no console errors. Fixed during testing: a past deadline counted as "fits", Balanced could pick Rebuild, "best fit" on a zero score, wrong "no technology choices" text. Storage restored exactly afterwards.

### 2026-11-20 — Richer technology rules (Claude)
- The owner asked whether the rules cover larger estates. Review: 5 areas, 31 fixed options, 3 states, no reason/owner/date, no custom options, no bulk entry. Agreed plan in three steps (backlog 9 and 10); this is step 1.
- New `assets/rule-catalog.js` (cache key `rules-rich-2`): six states `standard | preferred | allowed | approval | retiring | excluded` in `M.decide.rules.stance` (old preferred/allowed/excluded still valid); `ruleInfo[area][id]` = {reason, owner, reviewBy, exitBy, at, by}; `custom[area]` = the organization's own options. `syncRuleCatalog(R)` (called at the end of `ensureDecide()`) appends custom options to `TARGETS[area].opts` and gives a custom cloud a generic `CLOUD_SVC` entry. `ruleStanceEffect()` replaces the old check in `rankOptions()`: standard +3, preferred +2, approval −1, retiring and excluded exclude the option, and in single-choice areas a standard counts −1 against the others (not against "keep"). Rule weight stays ×3. The reason is shown in the ranking reasons and in rule violations. Every change is logged as `rule:<area>:<id>`.
- Presets: Azure shop, AWS shop, Google Cloud shop, Regulated (keep data on-premises), Open-source first, Low risk. Import CSV (technology radar or our own export): columns name, ring/status, optional area, reason, owner, review_by, exit_by; Adopt/Trial/Assess/Hold map to Preferred/Allowed/Needs approval/Retiring; names not in the list are added as own options only when an area is given, otherwise reported. Export writes the same format.
- `assets/rules-page.js` (cache key `rules-rich-1`): chips open a rule editor dialog; chips show the state icon, exit date and "review due"; "+ Add your own" per area; toolbar with presets, import, export and CSV help; the tab hint counts states and overdue reviews.
- Also fixed old step numbers left from the renumbering (Target decisions, Roadmap, Sign-off texts now say 05/06), and rule violations no longer lowercase the whole reason.
- Checked in Chrome on :8768 with storage backed up: preset set 8 rules; import matched Kafka to Event streaming and PostgreSQL, reported COBOL as not in the list, added Yotta Cloud under Cloud (a bug that put area-less rows under Architecture was found and fixed); the editor saved Retiring with exit date and reason; Target decisions disabled Azure with the reason and showed services for Yotta Cloud; all pages open; no console errors. Storage restored exactly (11 of 11).

### 2026-11-20 — Rules & people page in three tabs (Claude)
- The owner found the page confusing: 6 sections, 43 dropdowns and 20 fields on one long page, 31 of them identical Allowed dropdowns, the scoring weights at the bottom and skills under "Constraints".
- New `assets/rules-page.js` (cache key `rules-tabs-1`); the router now calls `rulesTabsPage()` for `rules`. It builds the old `rulesPage()` and reuses its sections, so storage, logging and scoring are unchanged. Tabs show progress (people added, preferred/excluded count, limits answered) and a Next button. Technology rules are chips that cycle Allowed → Preferred → Excluded and write the same `M.decide.rules.stance`. Tab 3 order: What matters most, Risk budget and standards, Team skills data and contracts (was "Constraints"), Money (collapsed unless a run cost is entered). Styles are injected by the file (`#rules-tabs-style`).
- `rulesPage()` is still in `index.html` and still used as the source of the sections; edit the sections there, not in the new file.
- Checked in Chrome on :8768 with storage backed up: tabs switch; chip clicks stored preferred and excluded and updated the tab hint; tab 3 shows its four sections; Target decisions still renders; no console errors. Storage restored exactly (11 of 11 entries) afterwards.

### 2026-11-20 — One Validate page instead of three (Claude)
- The owner agreed to merge 03 System inventory, 04 Review findings and 05 Resolve gaps. The sidebar now has **03 Validate**; later steps are renumbered: 04 Discovery summary, 05 Rules & people, 06 Target decisions, 07 Roadmap, 08 Sign-off & export (page kickers and in-text references updated, also in `teams/{code,database,infrastructure}/agents.html`).
- New `assets/validate-page.js` (cache key `validate-3`): `validatePage()` shows one counter row (records to confirm, findings to review, conflicts, owner decisions) and three tabs that reuse `inventoryPage()`, `reviewPage()` and `gapsPage()` unchanged; it strips their title block, their counter row and the buttons that only jumped between the old pages. `go('inventory'|'review'|'gaps')` sets `validateTab` and opens `validate`, so every existing link keeps working. The Record evidence page's review box now says "Validate findings · Open Validate".
- Checked in Chrome on :8768 (storage backed up to `__bk`): sidebar shows 03 Validate and the new numbers; each old id opens the right tab; tab clicks work; Populate from team reports is still on the inventory tab; no console errors. The main script and the new file pass `node --check`.

### 2026-11-20 — Approach suggestions from verified claims (Claude, not committed)
- Doctrine step 4 of "Verify before deciding" (principle 5, policy "what the AI may see").
- **Backend:** new `POST /api/decide/suggest` (`internal/server/suggest.go`; health flag `approachSuggest`). Body `{strategic, criticality, claims:[{kind, role, text, verdict, correction}]}`, at most 300 claims, 1 MB, text cut to 600 characters. Unknown fields are refused (400), so names or comments cannot be sent by mistake; verdict must be confirmed or corrected, and corrected needs the correction. 503 without a key. `internal/agents/suggest.go` asks `ModelStrong` for structured JSON (summary; up to 4 suggestions with approach, fit, reasons citing claim numbers, what would change it) and drops unknown approaches, duplicates and claim numbers that do not exist. Tests: `suggest_test.go` in both packages.
- **Web app** (end of `assets/verify-claims.js`, cache key `suggest-1`; `aiPanel(c,d)` on each decision card after Verified claims): a consent box per card that covers one request, "See exactly what is sent", Suggest approaches / Ask again. The answer is kept in `d.ai` with the claims it used; it is marked out of date when they change. "Use this approach" sets the approach only. Each request is logged as `ai-suggest` with how many claims were sent.
- Checked: Go tests pass. Real call with made-up payroll claims (none of the user's data) on a separate backend built from this code on :8092 (Codex's :8090 untouched): answer in 9 s, reasons cite claims, extra field refused. In Chrome on :8768 with storage backed up and made-up claims: the preview held only the 3 verified claims (no quote, comment, name, rejected or open claim); the button stayed off until the box was ticked and the tick cleared after one request; three suggestions came back; Use this approach set Replace; answering another claim marked the suggestion out of date. Storage restored exactly (11 entries).
- 2026-11-20 00:26: the :8090 backend was rebuilt from this code and restarted (old Codex process from `Documents\Codex\...\work\database-business-review` stopped; no run was in progress). It now runs `Downloads\Legecy Modernizationin\mindsuite-agents.exe serve -config mindsuite.json -runs "Downloads\Legecy Modernization
uns"` from that `bin` folder, hidden, log in `binackend.log`. Health shows `approachSuggest: true`, Claude available from the encrypted key file, and the 3 Mess-Management runs are still listed.

### 2026-11-19 — The gate in Target decisions (Claude, not committed)
- Doctrine step 3 of "Verify before deciding" (principles 4 and 6). Code at the end of `assets/verify-claims.js` (cache key `gate-1`), hooked into `index.html` with `typeof` guards.
- **Which claims:** `claimsForContext(c)`: for the whole system (no bounded contexts) all live claims; otherwise claims whose module, text or evidence files mention the context's name, its components' names or their cited paths, plus claims that mention no context (system-wide). Claims now keep the finding's `module`.
- **Gate:** `claimGate(c,d)` splits them into confirmed/corrected (the basis), rejected (shown with reason, not used), accepted risks and blocking. `recordDecision` refuses while any claim is blocking. "Accept N as risks" needs a reason and stores `d.accepted[claimId] = {reason, by, at}`, logged as `accept-risk` under `decision:<context>`.
- **Record:** on decide, `gateRecord` saves `d.basis` (claim id → status/answer/text signature), `d.risks` and `d.verified`. The card's footer says what the decision rests on. Sign-off readiness warns (not blocks) when decisions rest on accepted risks.
- **Reopen:** `decisionFor` calls `claimsChanged(c,d)`; if a claim in the basis was answered, changed or is gone, or a new claim applies, the decision reopens with `d.reopenWhy` (shown on the card) and a `reopen` log entry. Decisions made before the gate (no `d.basis`) are not reopened.
- With no claims collected the card says so and does not block (projects without agent runs keep working).
- Checked: node test (context matching, gate split, reason required, record, reopen on answer, on a new system-wide claim but not a claim of another context, on a gone claim). In Chrome on :8768 with storage backed up, on Mess-Management (100 claims, whole system): Record was refused with 100 unverified, accept without reason refused, accept with reason → recorded "rests on 0 verified claims and 100 accepted risks"; Sign-off showed the warning; importing one Confirmed answer reopened the decision with the reason. Storage restored exactly (11 entries match).

### 2026-11-19 — Verify with people: claims, roles, form file and answers (Claude, not committed)
- Doctrine steps 1 and 2 of "Verify before deciding". The four policy defaults are now in `README.md` ("Policy"): form as a file, the owner sends it, the AI sees only verified claims with consent, unanswered claims stay open assumptions.
- New `assets/verify-claims.js` (cache key `verify-2`) and a new page **Verify with people** (menu item "✓", after Discovery summary; `screen==='verify'`, `verifyPage()`).
- **Claims** (`M.verify.claims`): "Collect claims" reads the latest finished run of each team in the project (`loadWorkspaceRuns`, `GET /api/runs/{id}`). `CLAIM_KINDS` maps finding kinds to six roles (`VERIFY_ROLES`, the README table); unlisted kinds are technical facts and are not sent to people. Business and process kinds are one claim each; other kinds with more than 5 findings in a run become one claim with examples. Rejected findings are skipped; conflicts are flagged. Re-collecting keeps answers; if a claim's text changed it reopens. Statuses: proposed (not sent), open (waiting), confirmed, corrected, rejected.
- **Roles:** `M.verify.assign` maps each role to a person from Rules & people, suggested from their role. "Not mine" answers move the claim to the named person (`assignee`), or flag that they must be added first.
- **Form file:** "Make form" downloads one self-contained HTML file per person with only their claims (text, evidence file and lines, a short quote; never the quote of a secret). The person picks Confirmed / Corrected / Rejected / Not mine (the last three need a text), can save part and finish later, and "Save my answers" downloads an answers JSON. The form makes no network calls; all text is shown with textContent.
- **Import answers:** checks the file is a MindSuite answers file for a form made in this project, skips claims not in that form or incomplete answers, records who and when on each claim, and logs every change in `M.history` (`verify:<claim id>`).
- Nothing here calls Claude or the network beyond reading run results from the local backend.
- Checked: node test on real Mess-Management runs (104 claims from 361 findings; 94 business rules for the process owner; answers applied; wrong form and bad file refused; no secret quotes). In Chrome on :8768 with storage backed up: collected 100 claims, made the form, answered 4 claims in it, imported them; the claims table showed who confirmed. Storage restored exactly afterwards (11 entries match). The column fix (`.verify-table`) and de-duplicated examples came after that check and were not seen in the browser.

### 2026-11-19 — Doctrine: Verify before deciding (Claude, not committed)
- `README.md` has a new section, "Verify before deciding": the step between Discovery and Deciding that applies to every project. Findings are claims until the right role confirms them; one standard form (Confirmed / Corrected / Rejected / Not mine); verification is a gate for decisions; the AI suggests approaches from verified claims only and a named person decides; decisions trace back to claims; confirm and reject rates improve the method.
- It lists four policy questions still open: how the form travels, who sends it, what the AI may see, and what happens when nobody answers. Nothing is built yet.

### 2026-11-19 — Claude checkbox on each evidence card (Claude, not committed)
- Each evidence card on Upload evidence now has its own checkbox, "Allow selected evidence to be sent to Claude on the next team run", under its buttons. It was one shared box hidden under "More evidence options"; that one is removed.
- `evidenceUI.allowModel` in `assets/evidence-intake.js` is now a map by team. `runEvidenceTeam` sends `allowModel` for that team only, and clears only that team's box after the run starts (consent still covers one run). Resume analysis uses the same team box.
- Cache key `assets/evidence-intake.js?v=card-consent-1`.
- Checked: both scripts parse. Not checked in the browser (the browser pane refused localhost at the time).

### 2026-11-19 — Main backend caught up with Claude’s handoff (Codex)

- Restarted the main local agent service on 8090 after verifying there were no running analyses. Its startup log confirmed it loaded the configured encrypted key file. Both 8090 and Claude’s 8091 service report model availability.
- Verified all four completed workspace reports still appear in the main run listing and open after restart. No client evidence was sent to Claude or rerun during catch-up.
- The latest tracked application changes are at e5d8e1a. The working copy was at 5951ae7 before this handoff update.

### 2026-11-19 — Claude key replaced (Claude)
- Every Claude call had failed with 401 because the saved key file didn't hold an Anthropic key. The owner saved a new Console key (named "L2M") in `~/.mindsuite/claude-key.xml`. Checked: the models endpoint returns 200, and one tiny Haiku request succeeded, so billing is active. The key itself is never logged or stored in the repo.
- A backend reads the key only at start. The backend on 8091 was restarted and reports "Claude: available". **Restart the Codex backend on 8090** so it picks up the new key.

### 2026-11-19 — Pushed: commit e5d8e1a on main
- The 2026-11-19 Codex entries below (interview workflow, evidence intake, batching and resume, telemetry, guard, business operations in the reports). Before the push: `gofmt`, `go vet`, `go test ./...` (12 packages) and `node --check` on every changed script all passed; a secret scan found nothing; the app loaded with no console errors. GitHub Pages run 44474542191 succeeded.
- The Claude VB.NET fixes were already pushed in Codex's commit `f61ae9e`.

### 2026-11-19 — Business operations in Database reports (Codex)

- Database now uses the same Business operations report tab as Source code. The Business operations & data domains agent traces exported procedures, named Oracle package members, triggers, and explicit connections from scheduler job actions, with source file and line per step. The view shows conditions, calculations, reads and writes, routine and external handoffs, errors, returns and transaction boundaries where present.
- The local flow uses schema and routine definitions. The report distinguishes declared schedules and code paths from runtime behaviour and turns missing ownership, confirmation, and failure paths into questions. Claude can optionally interpret the cited facts and data lifecycle.
- Current interview terms prioritize workflows and review targets locally. A new database review can run on previously saved evidence. Large reports link job declarations and routines across batches. Raw interview text does not go to Claude.
- Verified with Oracle settlement, SQL Server procedure boundaries and a split jobs/programs fixture. No client data rows or stored credential values were copied into the new section.

### 2026-11-19 — Business operations in the code report (Codex)

- New Source code reports open on Business operations. The section lays out evidence-backed workflow triggers, actions, decisions, access checks, records, handoffs and exceptions as ordered steps. It keeps manual work, actor ownership and production outcomes as questions when the evidence stops.
- The local reader structures supported C# and VB.NET traces, including recognized numeric decisions and rollback calls. Declared JCL job steps show as operating sequences. Code reports across batches retain this section and prioritize using the existing interview.
- The optional domain model can write richer operating flows. Every step must cite supplied facts, and rendering requires accepted supporting evidence and source locations. Interview free text stays local. Older reports need a fresh code run for the new view. Code batch checkpoints use a newer signature because the findings now carry structured operations.
- Regression coverage includes stock receiving, validation, ledger writes, rollback, missing actor, declared job order, rejected missing evidence and workflow citations across batches. The front-end view was checked with a representative report and a DOM fixture.

### 2026-11-19 — Per-agent run telemetry (Codex)

- Evidence and Reports now expose Telemetry per run. The report viewer also shows its saved attempts. The live view refreshes while the analysis is running and can filter by agent.
- The Go runner records an ID, agent, step, optional batch and part, start, end, elapsed time, outcome and model requests for each measured attempt. Retried agents remain separate. Resumed reviews mark recovered attempts as reused.
- Each run appends the attempt states to telemetry.jsonl and keeps the final snapshot in result.json. The workspace-scoped telemetry endpoint recovers interrupted attempts after restart. Earlier runs may have no timing data.
- Regression checks cover failed and timed-out attempts, saved and reused telemetry after restart, and workspace isolation.

### 2026-11-19 — Shared large-evidence batching and resume (Codex)

- Large inputs use the shared runner for all ten teams. The planner groups code by module, targets up to 5,000 lines, and splits long files at recognized boundaries where possible. Telemetry can use log or CSV row ranges. Structured inputs without a safe split remain intact. Reports count oversized batches and source-loader skips.
- Each successfully reviewed batch saves an atomic checkpoint inside the project review folder. Evidence and Reports show progress and offer Resume analysis after interruption. Resume starts a new report, reuses compatible checkpoints, and leaves previous reports intact.
- The second pass links repeated evidence and, for source code, cross-batch imports and supported workflow traces. Findings preserve original source line numbers. Whole-period Telemetry metrics are calculated from the complete timeline so a partial window cannot make a live endpoint look unused. Other local estimates retain their batch scope.
- The code interview prioritizes batches and findings. The selected excerpt budget is shared across the review, and no raw interview answers are added to model requests. Update the checkpoint version if a change makes existing saved batch results incompatible.
- The full Go suite passed, including checkpoint recovery across a restart, original source citations, cross-batch workflows, intact structured exports, and full-period Telemetry. Frontend resume dispatch was checked with a mock response, and the updated local service passed health checks. Browser visual verification was unavailable. The existing upload and loader caps remain in force.

### 2026-11-19 — Interview-guided workflow and business-rule review (Codex)

- The Source code card sends the current problem, success criteria, reason selections, scope and optional deadline to the local backend. Interview terms rank cited workflow and rule findings and help choose files for closer reading. Interview answers are kept out of Claude requests and saved run JSON. The report shows a supplied deadline as project context.
- The local code team traces supported C# and VB.NET entry points into uniquely matched services and repositories. It records conditional exits, persistence and available integration/access evidence with file and line references. The report calls out what the trace did not establish. Exclusions lower review priority while retaining the source baseline for dependency review.
- Run code review again starts a new review on saved code with the latest interview answers. It retains previous reports and refuses a second review for an upload already running.
- Verified the Go suite, including an inventory workflow with a quantity rule and report prioritization, the saved-evidence repeat review, and local server health. The refreshed service on 8090 uses the existing runs directory. No user project analysis was started.

### 2026-11-19 — Populate System inventory from evidence reports (Codex)

- System inventory can now load the newest completed report for each team in the active workspace. It reuses the team importers to propose records for review and keep citations where available. Confirmed records are preserved. Any reports that fail to load are counted separately.
- The code importer adds modules as components and can propose bounded contexts from cited business-purpose findings when explicit capability findings are absent.
- Verified with a local test fixture covering citations, repeat import, protected records, and partial errors.

### 2026-11-19 — Interview before evidence and individual team runs (Codex)

- The flow opens on Project. A project name is required to proceed to Evidence. Reasons, current problems and success criteria are asked before upload. Existing M.scope.brief data is reused, including whyDetail and success.
- Evidence cards offer About this team and Run agent team. Each launch submits only that team’s pending upload IDs to the existing evidence endpoint and stays on Evidence for report progress. Use Reports to see all runs. The first launch creates the project folders.
- Interview data remains in the workspace. The evidence endpoint still runs baseline analysis and receives no interview text.
- Verified JavaScript syntax, per-team submission, the duplicate guard, and all ten agent-guide paths. Browser automation could not open the preview.

### 2026-11-19 — Multiple modernization reasons (Codex)

- Start offers 33 optional reasons in six groups. Users can choose and clear any combination or specify Other in Goal details.
- New selections are saved as an array in M.scope.brief.why. Existing single-reason strings remain readable. Sign-off lists every reason. Readiness checks the deadline and success criteria independently of the optional reasons.
- Browser tested adding two reasons, reload persistence, Sign-off and clearing the selection in an isolated test project.

### 2026-11-19 — Streamlined discovery workspace (Codex)

- Record evidence now shows one stage at a time. The compact four-step flowchart also appears on Agent reports and fits horizontally at phone width.
- Upload actions and available reports stay visible on each source row. Collection guidance, manual answers and applicability sit under Collection details. Collector import, the reference scenario, the browser scan and the evidence register sit under More evidence options.
- The Start view puts the system name, optional modernization goal and launch controls together. System type and scope can be opened for overrides. Start analysis uses the existing staging and agent launch path.
- Reports starts with workspace runs, with a selector for one team’s setup and run details. Resume and retry links open the appropriate team. Review links to findings and opens validation and evidence coverage on demand.
- Checked at 1440 × 900 and 375 px in the browser. Verified stage navigation, source filtering, Start form, report settings and review access. The UI checks used the open project without starting an agent run or changing its form values.

### 2026-11-19 — Record evidence flowchart (Codex)

- The Record evidence page now links each step in the workflow: Upload evidence → Set a goal (optional) → Start → Agents analyze → Review findings.
- The Start node takes the user to the existing Start control, which creates the project workspace and launches the available teams for staged evidence. Agents analyze opens Code analysis, where the user can open reports by team. Review findings jumps to evidence validation.
- The optional goal updates the flowchart immediately. Goal-based recommendations are shown as planned after review. This change adds no second analysis pass or backend orchestration.
- The main footer opens findings for real evidence. The simulation remains available as Analyze reference scenario when sample evidence is loaded.
- Checked JavaScript syntax and verified desktop layout, goal selection, Start navigation, Code analysis, and the findings link in an isolated browser preview.

### 2026-11-19 — When an agent fails, the run carries on and says so (Claude, not committed)
- **Shared guard** (`backend/internal/agents/guard.go`): every team's fact agents now run through `r.RunParts` / `r.RunPart` instead of their own goroutine loop. All ten `teams/*/team.go` were converted.
  - A part that crashes, or does not finish within `agents.AgentTimeLimit` (10 minutes), is recorded as a `Problem` (agent, name, part key, step, error, timed out, can re-run). The other agents carry on. A stuck agent no longer holds up the run.
  - A failed Claude request is recorded too (`r.AgentProblem`): business rules (one problem per agent, with the count of failed excerpts), business story or meaning, roles and the written summary, in every team.
  - The step becomes `partial`, with "did not finish: <names>" in its note. The run stays `done` and lists `problems` (saved in `result.json`). Steps every later step depends on (reading the input, the verifier, the report) still stop the run as `failed`.
  - Every report opens with "Warning: this report is incomplete", naming each agent, its step and its error (`withProblemsNote`, applied in `Run.Execute`).
- **Code report:** each evidence area knows which fact agents look for it (`parts` in `evidenceAreas`). When one failed, the area says **Not checked** (nothing found) or **Partly checked** (some findings), instead of "Nothing about this area was found".
- **Run one agent again:** `POST /api/runs/{id}/rerun?part=<key>` (health `agentRerun: true`). It re-runs only that part, verifies only its new findings, rewrites the report and saves the result. It needs the run's evidence in memory, so after a backend restart it answers 409 "Run the team again". Steps that used the facts earlier (reading list, assessment) are not redone, and the log says so. Teams register their report writer with `r.WriteReport(ctx, fn)` so a re-run can rewrite it.
- **Web app:** a warning box lists the agents that did not finish, with a **Run again** button each (`agentProblemsBox` / `agentRerun` in `teams/shared/agent-backend.js`; used by `teams/shared/team-panel.js` and `teams/code/panel.js`). Steps show "Done with problems". The report viewer (`assets/agent-report.js`) shows the box and "Finished with problems". The project's run list (`assets/project-workspaces.js`) shows "Finished with problems (N agents did not finish)". Cache keys are `agent-problems-1`. Styles: `.agent-problems`, `.agent-step.partial` in `index.html`.
- **Testing failures:** `MINDSUITE_TEST_FAIL_AGENT=areas,sec` makes those parts crash on their first try (for testing only; leave it unset).
- **Tests:** `agents/guard_test.go` (a crash and a hang are skipped and reported, the report warns, `problems` are saved, re-run fixes one and rewrites the report, a second re-run while one is going is refused). `teams/code/areas_test.go` covers Not checked / Partly checked. `go vet` and `go test ./...` pass.
- **Checked live** on a test backend (port 8093, scratch runs folder) with CardDemo. Security and the evidence-areas part were forced to fail; 1,036 findings from the others were kept; step 3 showed "Done with problems"; the report warned and marked Identity, Telemetry and Messaging "Not checked". In the browser, **Run again** on Security added its finding and left one problem. The run list showed "Finished with problems (1 agent did not finish)". The browser pane's storage was restored and the test backend stopped afterwards.
- The backend on 8090 was a Codex-built exe (`work\evidence-prefill\mindsuite-agents-prefill.exe`). On 2026-11-19 it was stopped and restarted from this source with `go run` (`.claude/launch.json` "mindsuite-agents-backend", same runs folder); health shows `agentRerun: true`.

### 2026-11-19 — Open agent reports from evidence cards (Codex)

- Each evidence card shows reports from its own agent team in the current workspace. The latest completed report is visible directly, with earlier runs under a disclosure. A running analysis refreshes in place. The card opens the existing report reader.
- The cards and report history share the workspace run list request and cache. Browser verified the Source code and Database cards each opened the corresponding saved report. JavaScript syntax checked. No backend changes.



### 2026-11-19 — Recover staged uploads automatically (Codex)

- Record evidence checks for unlisted staged uploads when it opens. Restored files appear in the regular list. Removed the recovery button and empty-recovery message.
- Browser tested an empty page and a staged upload absent from browser state. The file appeared after reload without any recovery action. Removed the test upload afterward.



### 2026-11-19 — Project title and modernization question (Codex)

- Record evidence now shows the detected name in the heading as Record evidence: <project>. Name project and Rename project sit beside the heading. Step 2 keeps the system name and system types and asks for the modernization goal before Start.
- The brief asks Why do you want to modernize this system? It distinguishes source inspection during upload from completed analysis. Existing modernization reason values and saved project data use the same format.
- Browser verified upload-driven heading, manual rename, reason persistence after reload, and a system name that remains independent of renaming the project. JavaScript syntax checked. No backend changes needed.



### 2026-11-19 — Suggest project fields from uploaded evidence (Codex)

- Source intake inspects filenames and a bounded set of project manifests in staged folders or ZIPs. When it can identify one named project, it fills the project and system names from it. Otherwise it uses the upload name for blank fields. App types are selected only when supported by source markers. A database export on its own does not classify the system.
- Step 2 shows the supporting file for suggestions. Manual changes to either name or the type selection remain in place on later uploads. Full analysis remains behind Start, and prefill works locally without Claude.
- Backend coverage includes a WinForms ZIP and an ignored test-only web marker. Browser verification used StockDesk.vbproj in a differently named ZIP, confirmed both names and Desktop applications, then verified that manual names and type selection survived a later Web Forms upload. Temporary test files were removed and the local backend was refreshed.



### 2026-11-19 — Upload evidence before Start (Codex)

- Record evidence now starts with Upload evidence. Selecting a file, folder, or ZIP stages an original copy under runs/staging/<workspace-id>/<upload-id>/files. The first upload names an unnamed draft. Users can change the name or remove/recover staged uploads before Start.
- Start makes the named project and all ten team folders in the configured runs/workspaces directory. It promotes each upload to its team's evidence/<upload-id>/files folder and starts analysis from there. Reports, result JSON and logs remain under the matching team/run directory. The report list refreshes while runs are active. Claude remains opt-in for Start.
- New endpoints in backend/internal/server/evidence.go handle staging, listing, discarding and starting. A repeat Start reuses a run already created for that upload. Read-only history queries no longer create folders for an unstarted draft. The original files are retained after Start. Standard collector bundle import and manual notes remain browser workflows.
- Verified go test ./..., JavaScript syntax, and the browser with separate source and database uploads. No project folder was created during staging in the integration test. The browser suggested IntakeCatalog for the first file, accepted a renamed project on Start, and displayed both completed reports. Confirmed the physical files and reports in separate team folders. Local backend refreshed on port 8090.



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
