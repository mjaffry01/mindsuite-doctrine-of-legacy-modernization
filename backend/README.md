# MindSuite agent backend (Go)

Runs the MindSuite agent teams and serves their results to the web app. Each team lives in **its own folder**, so a problem can be traced to one team without reading the others.

| Team | Folder | Agents | Reads | In the app |
|---|---|---|---|---|
| Source code | `internal/teams/code/` | 15 | a code folder or zip | Record evidence → Source code → **Run the code agent team** (or 02 Code analysis) |
| Database | `internal/teams/database/` | 14 | the DBA's schema export (.sql/.ddl) and catalog query result (.csv) | Record evidence → Database → **Run the database agent team** |
| Environment & infrastructure | `internal/teams/infrastructure/` | 13 | VM inventory CSV (RVTools, PowerCLI, Hyper-V), connection lists (ss, Get-NetTCPConnection), kubectl JSON, cloud inventories, infrastructure as code, collector bundle | Record evidence → Environment & infrastructure → **Run the infrastructure agent team** |
| Messaging & caching | `internal/teams/messaging/` | 12 | runmqsc DISPLAY output, kafka-topics / kafka-consumer-groups output, redis-cli INFO, memcached stats, RabbitMQ definitions, collector bundle; optionally the application's code | Record evidence → Messaging & caching → **Run the messaging & caching agent team** |
| Identity & access | `internal/teams/identity/` | 12 | AD user and group-member CSVs, Entra MFA report and Conditional Access policies, Okta JSON, openssl certificate output, collector bundle | Record evidence → Identity & access → **Run the identity & access agent team** |
| Telemetry | `internal/teams/telemetry/` | 11 | access logs (NGINX/Apache, IIS), Splunk / Application Insights / Datadog CSV, Prometheus and CloudWatch JSON, collector bundle; optionally the application's code | Record evidence → Telemetry → **Run the telemetry agent team** |
| Batch & scheduling | `internal/teams/batch/` | 10 | Control-M JSON, AutoSys JIL, TWS composer output, schtasks CSV, crontab, run-history CSV | Record evidence → Batch & scheduling → **Run the batch & scheduling agent team** |
| Network edge & partners | `internal/teams/network/` | 11 | nginx -T, HAProxy configuration, ingress/service JSON, cloud inventory, DNS zone or CSV, openssl certificate output, partner CSV, firewall rule CSV | Record evidence → Network edge & partners → **Run the network edge & partners agent team** |
| Documents & diagrams | `internal/teams/documents/` | 12 | Word (.docx), PowerPoint (.pptx), PDF with text, Confluence HTML, Markdown/text, draw.io (.drawio), Visio (.vsdx); up to 20 MB per file | Record evidence → Documents & diagrams → **Run the documents & diagrams agent team** |
| Incidents & changes | `internal/teams/incidents/` | 11 | ServiceNow incident, change_request and problem list exports (CSV), Jira CSV exports | Record evidence → Incidents & changes → **Run the incidents & changes agent team** |

The explanation pages for clients are in the web app, one per team: `../teams/<team>/agents.html`.

## Folder layout

```
backend/
  cmd/mindsuite-agents/      the program: serve (HTTP API) and run (command line)
  internal/agents/           SHARED CORE, used by every team
    run.go                   a run: options, the 9 steps, coverage, result, saving
    team.go                  the Team interface and the registry
    store.go                 the evidence store (task, finding, hand-off, verification, question)
    codebase.go              loads files from a folder, zip or upload (in memory, read-only)
    verify.go                the verifier (step 7 of every team)
    rules.go                 the business-rules reader (used by the code and database teams)
    llm.go                   the Claude client (structured outputs)
    helpers.go               citations, secret detection, small helpers
  internal/teams/            ONE FOLDER PER TEAM
    teams.go                 imports every team so it registers itself
    code/                    team.go (agents, steps, orchestration) + one file per agent group + code_test.go + testdata/sample
    database/                register.go, team.go, model.go (parser), agents.go, ai.go, report.go + database_test.go + testdata/sample
    infrastructure/          register.go, team.go, model.go (parser) + infrastructure_test.go + testdata/sample
    messaging/               register.go, team.go, model.go (parser) + messaging_test.go + testdata/sample
    identity/                register.go, team.go, model.go (parser) + identity_test.go + testdata/sample
    telemetry/               register.go, team.go, model.go (parser) + telemetry_test.go + testdata/sample (+ make_sample.py)
    batch/                   register.go, team.go, model.go (parser), aliases.go + batch_test.go + testdata (make_sample.py)
    network/                 register.go, team.go, model.go (parser), aliases.go + network_test.go + testdata (make_sample.py)
    documents/               register.go, team.go, model.go (documents and diagrams), aliases.go + documents_test.go + testdata (make_sample.py)
    incidents/               register.go, team.go, model.go (tickets), aliases.go + incidents_test.go + testdata (make_sample.py)
  internal/server/           the HTTP API
```

A team never imports another team, with one exception: the infrastructure team reuses the code team's platform & config agent (`code.RunPlatform`) for infrastructure-as-code files.

### Adding a team

1. Create `internal/teams/<team>/` with a `register.go` implementing `agents.Team` (`ID`, `Agents`, `StepTitles`, `WholeKinds`, `NewState`, `Execute`, `Source`) and calling `agents.Register` in `init()`. Copy `database/register.go` as a start.
2. Keep the team's parser, agents, assessment and report in that folder. Use the core for citations (`agents.Cite`), secrets (`agents.LooksSecret`, `r.MarkSecret`), steps (`r.Step`), what was sent to Claude (`r.AddSent`) and verification (`agents.RunVerifier`).
3. Add `testdata/sample` with known answers and a test; add the import to `internal/teams/teams.go`.
4. In the web app: `teams/<team>/panel.js` and `teams/<team>/agents.html`.

## Run it

```bash
cd backend
go run ./cmd/mindsuite-agents serve            # http://127.0.0.1:8090
```

- **Rule-based agents only (default):** nothing leaves the machine.
- **With Claude:** give the backend your Claude API key (see *Claude API key* below), then tick the consent box in the app for each run.

**Running it on a server:** see [DEPLOY.md](DEPLOY.md) (for DevOps: the key as `ANTHROPIC_API_KEY` from a secret store, `server.example.json`, HTTPS and login in front).

### Settings: `backend/mindsuite.json`
The backend reads `mindsuite.json` from the folder it starts in (`backend/` with `go -C backend run …`), or next to the program; `-config <file>` picks another one. Command-line flags override it. Fields: `addr`, `runs`, `cors`, `allowPath`, `models` (`strong`, `balanced`, `fast`: the Claude model of each tier; keep `strong` and `balanced` different so the verifier never checks a model with itself) and `apiKeyFile`.

**Never put the Claude API key in `mindsuite.json`**: it is committed and published. The backend refuses to start if the file contains something that looks like a key.

### Claude API key
The backend takes the key from `ANTHROPIC_API_KEY` if it is set, else from the file named by `apiKeyFile` (default `~/.mindsuite/claude-key.xml`). That file is encrypted with Windows DPAPI: only the Windows user who saved it, on the same machine, can decrypt it, and it lives in your user folder, outside the repo. Save it once in PowerShell (paste the key at the prompt):

```powershell
New-Item -ItemType Directory -Force "$HOME\.mindsuite" | Out-Null; Read-Host "Paste Claude key" -AsSecureString | Export-Clixml "$HOME\.mindsuite\claude-key.xml"
```

Then start the backend as usual; the start line says `Claude: available (…; key from encrypted key file …)`. To change the key, run the same line again. The key itself is never logged, saved in results or sent to the web app. Outside Windows, use `ANTHROPIC_API_KEY`.

Flags for `serve`: `-addr` (keep it on 127.0.0.1 unless you put authentication in front), `-runs` (folder for saved results, default `runs/`), `-cors` (allowed web-app origins; default localhost on any port and `https://mjaffry01.github.io`), `-allow-path` (allow runs on a folder path on this machine; loopback only).

Command line, without the app:

```bash
go run ./cmd/mindsuite-agents run -dir /path/to/code -report code-report.md
go run ./cmd/mindsuite-agents run -team database -dir /path/to/db-export -report db-report.md
go run ./cmd/mindsuite-agents run -team infrastructure -dir /path/to/inventory -report infra-report.md
go run ./cmd/mindsuite-agents run -team messaging -dir /path/to/broker-exports -report messaging-report.md
go run ./cmd/mindsuite-agents run -team identity -dir /path/to/identity-exports -report identity-report.md
go run ./cmd/mindsuite-agents run -team telemetry -dir /path/to/telemetry -report telemetry-report.md
```

Tests (each team has its own sample with known answers):

```bash
go test ./...
go test ./internal/teams/infrastructure/   # one team only
```

## API

| Method | Path | Purpose |
|---|---|---|
| GET | `/api/health` | Version, the registered teams, whether Claude is configured, whether folder paths are allowed |
| GET | `/api/agents?team=code` | One team: id, name, kind, model |
| POST | `/api/runs` | Start a run. `multipart/form-data` with `options` (JSON) and many `file` parts (filename = relative path) or one `zip` part; or `application/json` `{"path": "...", "options": {...}}`. Needs header `X-MindSuite: 1`. |
| GET | `/api/runs` | Runs held in memory |
| GET | `/api/runs/{id}` | Status, the 9 steps, coverage, findings, questions, reading list, what was sent to Claude, report |
| GET | `/api/runs/{id}/events?after=N` | Server-Sent Events: every message in the evidence store |
| GET | `/api/runs/{id}/report.md` | The Markdown report |

`options`: `team` (`code` default, `database`, `infrastructure`, `messaging`, `identity`, `telemetry`), `allowModel` (consent to send excerpts to Claude), `gitHistory` (code team, local folders only), `maxModelFiles` (files or programs read in depth; default 12).

## Project workspaces

The web app attaches `?workspace=<workspace-id>` to run creation, listing, snapshots, live events and report downloads. The server sets the workspace from the request URL and scopes every lookup to it. With a runs folder configured, project results, reports and logs live under `workspaces/<workspace-id>/<run-id>/` inside that folder. Completed runs and reports remain available after restart. Requests without a workspace continue to use legacy runs. Workspace IDs are for project organization. The deployment still requires authentication for network access.

## How a run works

Every team runs the same 9 steps (`Team.Execute`), writes only to the evidence store, and is checked by the shared verifier:

1. **Receive** the input (loaded in memory, read-only).
2. **Plan** work packages (modules, schemas, platforms).
3. **Collect the facts**: the team's rule-based agents, in parallel.
4. **Choose what to read in depth** (files, programs, hosts).
5. **Understand the meaning**: the team's Claude agents, only with a key and consent; otherwise reported as skipped.
6. **Assess** for modernization or migration.
7. **Verify** every claim: cited lines re-opened, model claims second-checked, derived claims need accepted evidence.
8. **Report** (Markdown).
9. **People** confirm in the web app.

### Code team (`internal/teams/code/`)
Inventory, supply chain, structure, data access, interfaces, security, quality, platform & config, legacy specialists; business rules and domain & process (Claude); assessment; report.

### Database team (`internal/teams/database/`)
Catalog reader, schema & relationships, size & usage, stored code, jobs/links/outside calls, security & privacy, portability, design quality; database business rules and data domains (Claude); migration assessment; report. Data rows in an export are counted, never read. Catalog CSV columns: `record_type, schema_name, name, row_count, size_mb, line_count, detail`; record types `table`, `view`, `code`, `job`, `link`, `admin`, plus `column` (`NAME TYPE [NOT NULL]`), `pk` / `index` (`COL1,COL2`), `fk` (`COL -> REF_TABLE(REF_COL)`), `grant` (`PRIV TO GRANTEE`), `version`.

### Infrastructure team (`internal/teams/infrastructure/`)
Estate reader, lifecycle & support, capacity & sizing, dependencies (move groups, outside endpoints, hubs), Kubernetes & containers, cloud resources, infrastructure as code, security & resilience; roles & environments (Claude); migration assessment (strategy per host, waves, approach, sizing); report. Connection files are named after their server (`connections-<server>.txt/.csv`); addresses are resolved to host names through the inventory's IP column.

### Messaging & caching team (`internal/teams/messaging/`)
Broker & cache reader (IBM MQ queue managers, queues, remote queues and channels; Kafka topics and consumer groups; RabbitMQ queues, exchanges and user names; Redis and Memcached statistics), versions & support, flows & usage (who sends to and reads from each queue, from the uploaded code and configuration; partner channels; queues with waiting messages and no reader), health & backlog, delivery & reliability, caching (hit ratio, eviction, memory, keys without expiry, caches used as stores), security (channels without TLS, clients as `mqm`, guest user, passwords as file:line); integration meaning (Claude: business event per queue and the integration map, names only); migration assessment (target per broker and cache, order, approach); report. Name each Redis/Memcached file after its instance (`redis-cache-01-info.txt`). Message contents and cached values are never read.

### Identity & access team (`internal/teams/identity/`)
Directory reader (AD users and group members, Entra MFA registration and Conditional Access policies, Okta users/groups/apps, openssl certificate output, collector bundle), accounts (inactive for 90+ days measured against the latest sign-in in the export, passwords that never expire, service and shared accounts), privileged access (administrator groups incl. nested, service accounts with admin rights, inactive admins, admins without MFA), MFA & sign-in policy, groups & roles, certificates (expired/expiring, weak keys, SHA-1, self-signed; passwords as file:line), provider & protocols; role model (Claude: target roles from group names and counts, **never person names**); migration assessment (clean-up list, service identities, approach, order); report (the model gets finding titles only). Name group-member files after the group (`App-Admins-members.csv`).

### Telemetry team (`internal/teams/telemetry/`)
Telemetry reader (one timeline per hour from access logs, monitoring CSV, Prometheus and CloudWatch JSON and the collector bundle; overlapping sources are not added up: each hour uses the highest count), load & peaks (average/peak per second, busiest hours/weekdays/months, quietest 4-hour window), response times (prefers the longest source), errors, usage & unused features (top endpoints, rarely used, endpoints in the uploaded code never called, distinct clients as a count), coverage & gaps (window under a year, monitoring gaps, passwords as file:line); business use (Claude: business functions and calendar from cleaned endpoint names and numbers); capacity & cut-over (peak × 1.5, service-level targets, freeze months, cut-over window); report. **Privacy:** client addresses are hashed and only counted in memory; query strings dropped; ids, e-mail addresses and tokens in paths become `{id}`; access-log lines are cited by file and line only (never quoted).

### Documents & diagrams team (`internal/teams/documents/`)
Text extraction is in the shared core (`internal/agents/documents.go`: .docx paragraphs, headings and tables; .pptx slides; .vsdx shapes and connectors; PDF text streams; files up to `MaxDocBytes` = 20 MB). Old .doc/.ppt/.vsd and scanned PDFs are listed as not readable and asked about. Document reader (title, date, version, headings; draw.io plain and compressed; Visio), freshness & coverage (documents 3+ years older than the newest, drafts with TBD/TODO, passwords as file:line, which of rules/processes/glossary/architecture are covered), rule statements (must/shall/only/within N days, quoted with their line), glossary (tables and "X means" sentences; terms defined differently in two documents), processes (numbered/listed steps, diagram flows), intended architecture (diagram components and links, named technologies; marked *intended*); business rules (Claude: rewrites only the rule sections the orchestrator picked, password lines removed), business story & conflicts (Claude: purpose, capabilities, contradictions from extracted facts only); evidence value (trust per document, validation plan); report.

### Incidents & changes team (`internal/teams/incidents/`)
Ticket reader (ServiceNow incident/change/problem lists and Jira CSV; kind from the number prefix, the issue type or the file name; priorities mapped to P1–P4), hotspots (incidents per component, the few components behind 60% of incidents, month-end and Monday peaks), recurring problems (similar cleaned titles, 3+ times; linked problem record or not; reopened; open problem records), restore times (median and slowest 10% per priority; incidents open 30+ days measured against the export's last date), change frequency (per month, standard/normal/emergency, weekend), failed changes (close code unsuccessful/backed out/with issues; change failure rate; incidents caused by a change via caused_by or same component within 48 h); failure story (Claude: fragile business functions from counts and cleaned titles only); modernization priorities (delivery profile, ranking per component, periods to avoid); report. **Privacy:** people columns (caller, assignee, reporter, …) are never kept, their values only remove those names from titles in memory; descriptions, comments and work notes are not read; e-mail addresses, IP addresses, ticket numbers and long numbers become placeholders; a title on a line that may hold a password is hidden; ticket rows are cited by file and line only (never quoted).

## Safety rules (keep them)

- Read-only: nothing is executed; nothing connects to a server or database; `git log` is the only command run (code team, local folders).
- Secrets: lines that may hold a credential are never quoted (`agents.Cite` / `LooksSecret` / `r.MarkSecret`) and are removed before any text goes to Claude.
- Claude is used only with a key **and** per-run consent; every request is listed in `sentToModel`.
- POST needs `X-MindSuite` (forces a CORS preflight) and an allowed origin, so other websites cannot start runs on your machine.
- Not done: licences and vulnerability lookup (would send dependency names outside: needs the owner's OK); Oracle Forms and ABAP specialists; utilisation-based sizing; authentication for non-local deployment.
