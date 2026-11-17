# Agents for code analysis — design

How the "agent" behind MindSuite should be split up to produce the **Code analysis report (1.4)** and the **Business story (1.2)** in `PRODUCT-OUTCOMES.md`.

**Principles**
1. **Tools extract facts, models interpret them.** Deterministic parsers (the current in-browser scanner, later the Discovery Collector) find what is certain. Language models explain, connect and name things.
2. **Every claim cites its evidence** (file and line, table, document) and states its basis and confidence. Uncited claims are rejected.
3. **Agents propose; people validate.** Results land in the inventory as "Extracted" or "Inferred" with `origin:'agent'`, and never overwrite a human-confirmed item.
4. **Read-only, metadata first.** Agents read code in a sandbox and never run it against production.

Status key: ✅ built · ◐ partly · ⬜ not built. **All agents are implemented in the Go agent backend (`backend/`, see `backend/README.md`).** A quick rule-based subset also runs in the browser scan.

---

## The agents

| # | Agent | Type | Reads | Produces | Status |
|---|---|---|---|---|---|
| 0 | **Orchestrator** | Coordinator | The whole run | Splits the codebase into work packages, assigns agents, tracks coverage (which files and modules have been analysed), merges results, and re-runs gaps | ✅ (`run.go`: work packages, hand-offs, reading list, coverage) |
| 1 | **Inventory agent** | Tool | All files, build files, `.git` metadata | Languages, size, build files, dependencies, commit and branch | ✅ |
| 2 | **Supply-chain agent** | Tool + lookup | Dependency lists, lockfiles | Versions, end-of-life, **licences**, **known vulnerabilities** (e.g. via OSV), SBOM | ◐ (versions and end-of-life/known-risk rules; no licences or OSV lookup yet: needs the owner's OK) |
| 3 | **Structure agent** | Tool + model | Packages, imports, deployables | Modules, dependency graph, cycles, layers, architecture style; with agent 9, **proposed bounded contexts** | ✅ (`structure.go`; contexts proposed by agent 10) |
| 4 | **Data-access agent** | Tool + model | SQL in code, ORM mappings, stored procedures, database export | Code-to-table map, shared tables, data ownership, what each procedure does | ✅ (`data.go`; procedure meaning via agent 9) |
| 5 | **Interface agent** | Tool + model | Controllers, routes, clients, queue and file code, API specs | Every exposed endpoint (method, path, purpose), outbound calls, topics and queues, file exchanges, batch inputs and outputs | ✅ (`data.go`, `runInterfaces`) |
| 6 | **Security agent** | Tool + model | Auth config, annotations, role checks, crypto, secrets | Login methods, **role → permission → endpoint/screen/table matrix**, privileged paths, hard-coded secrets, weak crypto, missing checks | ✅ (`security.go`; role checks linked to the nearest endpoint) |
| 7 | **Quality & change-safety agent** | Tool | Code, tests, Git history | Tests per module, complexity hotspots, duplication, **dead code**, **Git churn hotspots and bus factor** | ✅ (`quality.go`; Git history needs a local folder) |
| 8 | **Platform & config agent** | Tool | Config, Dockerfiles, Kubernetes, Terraform, schedulers in code | Environments, connection targets, caching strategy, containers and infrastructure as code, scheduled jobs | ✅ |
| 9 | **Business-rules agent** | Model | Rule-heavy code and database code (picked by agents 3, 4, 7) | Each rule in plain language: condition, outcome, constants, exceptions, **citation**, confidence | ✅ (`business.go`, Opus 5.5; needs key + consent) |
| 10 | **Domain & process agent** | Model | Screens, entities, status fields, events, rules, documents | Actors, capabilities, end-to-end journeys, domain events, glossary and naming conflicts, business calendar, the **business story** | ✅ (`business.go`, Opus 5.5, reads facts only) |
| 11 | **Legacy-language specialists** | Tool + model, one per family | COBOL/JCL/PL/I (copybooks, CALL graph, CICS transactions, job streams), RPG/CL, PowerBuilder, Oracle Forms, ABAP | The same outputs as agents 3–9 for code the generic parsers can't read | ◐ (`platform.go`: COBOL, JCL, RPG, PowerBuilder; not Oracle Forms, ABAP) |
| 12 | **Modernization assessment agent** | Rules + model | Outputs of agents 3–11, capacity, consistency needs | Extractability, effort, **seams** (where to cut), extraction order, which approaches are realistic per context | ✅ (`assess.go`, rule-based, with evidence ids) |
| 13 | **Verifier (critic) agent** | Model + tool | Every claim from the other agents | Re-reads the cited lines; rejects uncited or contradicted claims; flags conflicts between agents (e.g. diagram vs code); writes **questions for domain experts** | ✅ (`verify.go`; Sonnet 5.5 second opinion) |
| 14 | **Report writer agent** | Model | Verified results | The Code analysis report and the Business story in plain language, with links to the evidence | ✅ (`report.go`; written summary and story with Sonnet 5.5 when allowed) |

Human validation is **not** an agent. A named person confirms, corrects or rejects each result in the inventory.

## Order of work in one run
```
1 Inventory ─┬─ 2 Supply chain
             ├─ 3 Structure ─┐
             ├─ 4 Data access├─ 9 Business rules ─┐
             ├─ 5 Interfaces │                    ├─ 12 Assessment ─ 13 Verifier ─ 14 Report writer ─▶ people validate
             ├─ 6 Security   ├─ 10 Domain & process┘
             ├─ 7 Quality    │
             ├─ 8 Platform  ─┘
             └─ 11 Legacy specialists (when legacy languages are found)
```
Agents 1–8 and 11 can run in parallel per module. Agents 9 and 10 need their outputs to pick the right code. Agent 13 checks everything before it reaches a person.

The client-facing page explains the same run as **9 steps** (`STEPS` in `teams/code/agents.html`): 1 receive the code, 2 plan the work (orchestrator), 3 collect the facts (agents 2–8), 4 choose what to read in depth, 5 understand the business (9, 10), 6 assess (12), 7 verify every claim (13), 8 write the report (14), 9 validate with people. Each step names the goal it serves and its status. Keep the two in step.

**Client-facing version:** `teams/code/agents.html` shows this design as an interactive page (system diagram, each agent's inputs/outputs/links, message types, a worked example, models, what runs today). It is opened from the app (02 Code analysis and 01 Record evidence → "How the agents study your code"). Keep it in step with this file.

## How agents communicate
Agents do not chat freely. They exchange five message types through the orchestrator and a shared, append-only **evidence store**:
1. **Task** (orchestrator → agent): scope and deadline.
2. **Finding** (agent → store): claim + citation + basis + confidence.
3. **Hand-off** (agent → orchestrator): e.g. "these files are COBOL"; the orchestrator re-plans.
4. **Verification** (verifier → store): accepted, rejected or conflict.
5. **Question for an expert** (verifier → people).

## Model choice: tiers, not one model per agent
The model is a setting per agent, chosen by testing on a real codebase with known answers.
- **Tier 0 · No model** (1, 2, 7, 8 and the parser parts of 3–6, 11, 12): fast, repeatable and free to rerun.
- **Bulk reading** (summarising thousands of files for agents 3–6, 11): a fast, low-cost model, e.g. Claude Haiku 5.5 (`claude-haiku-5-5`).
- **Reasoning agents** (9, 10, 12, the orchestrator): a strong model, e.g. Claude Opus 5.5 (`claude-opus-5-5`) or Sonnet 5.5 (`claude-sonnet-5-5`).
- **Verifier (13):** a different prompt (and ideally a different model) from the agent it checks.

## Where it runs
- **Every team in its own folder:** backend `backend/internal/teams/<team>/`, web app `teams/<team>/`. Teams: code (this design), database (`teams/database/agents.html`), environment & infrastructure (`teams/infrastructure/agents.html`), messaging & caching (`teams/messaging/agents.html`), identity & access (`teams/identity/agents.html`), telemetry (`teams/telemetry/agents.html`).
- **Database team:** a second team of 14 agents (catalog reader, schema & relationships, size & usage, stored code, jobs/links/outside calls, security & privacy, portability, design quality, database business rules, data domains & meaning, migration assessment, verifier, report writer, orchestrator) works from the DBA's exports in the same 9 steps. See `db-agents.html` and `backend/README.md`.
- **Agent backend (Go, `backend/`):** runs all agents next to the code (client's machine or network). The web app calls it from 02 Code analysis. Code is read-only, held in memory, and sent to Claude only with a key plus per-run consent.
- **Browser:** a quick rule-based scan still runs in `index.html` without the backend.
- **Later:** the Discovery Collector (`COLLECTOR-SPEC.md`) for databases and the environment can share the same backend.

## Build order (suggested)
1. **Verifier rules first:** citation format, rejection rules, confidence. Every later agent depends on them.
2. **Business-rules agent (9)**, on database code and the hotspots found by agent 7. Highest value, and a good way to test the verifier.
3. **Domain & process agent (10)** → the business story.
4. **Interface (5) and security (6) depth:** the endpoint list and role matrix.
5. **Supply chain (2):** licences and vulnerabilities (needs the owner's OK for an outside lookup).
6. **Legacy specialists (11)**, starting with COBOL/JCL.
7. **Orchestrator (0)** once more than two model-based agents exist.
