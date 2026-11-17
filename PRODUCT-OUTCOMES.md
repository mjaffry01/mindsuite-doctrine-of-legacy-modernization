# MindSuite Legacy Modernization Advisor — What the client gets

This is the reference definition of the product's output. Build toward it, and judge every feature by whether it improves one of these deliverables.

**One sentence:** from a legacy system's code, data, environment and a few business answers, MindSuite produces a **validated current-state dossier**, a **target decision record** per business area, and a **signed modernization roadmap**.

**Quality bar for everything below:**
- every statement cites its evidence (file and line, document section, measurement, or named person);
- it states its basis (*Observed in code · Extracted from export · Measured · Documented · Reported by a person · Assumed*) and a confidence;
- it is confirmed, corrected or rejected by a named person;
- unknowns stay visible as gaps, never guessed.

Status key: ✅ built · ◐ partly built · ⬜ not built yet

---

## Deliverable 1 — Current-State Dossier (end of Phase 1: Discovery)

| # | Section | What it contains | Status |
|---|---|---|---|
| 1.1 | **Executive summary** | System and scope, system type, why now, hard deadline, success criteria, top risks | ◐ (inputs captured; no summary page yet) |
| 1.2 | **Business story** | Purpose; actors and roles; capabilities (bounded contexts); end-to-end journeys step by step; business rules catalogue; business calendar (batches, month-end, season); key business objects and their lifecycle; partners and hand-offs; manual workarounds; open questions for domain experts | ◐ built in the agent backend (business rules + domain agents with Claude); not yet tested with a real key |
| 1.3 | **System inventory** | Technology stack with versions and support status; components and modules; data stores and ownership (incl. shared tables); interfaces (APIs, outbound calls, queues, files, database links); batch jobs; infrastructure (servers, platforms, connections); identity and access | ◐ code scan, database export and collector bundle are real; infrastructure map not yet |
| 1.4 | **Code analysis report** | Size per language/module; module dependency graph, cycles and coupling; code-to-table map; tests per module; complexity hotspots; dead code; dependencies with licences and vulnerabilities; hard-coded secrets; login and authorization checks; caching pattern; Git hotspots and bus factor | ◐ built: stack, size, modules, dependency graph and cycles, code-to-table map, tests per module, hotspots, extractability, effort, extraction order, config, auth, secrets, caching (step 02 Code analysis). In the agent backend also: Git hotspots and bus factor, dead-code candidates, duplication, endpoints and role checks. Not yet: licences/vulnerabilities |
| 1.5 | **Capacity & service levels** | Users, average and peak traffic, peak months, data size and growth, availability, RTO/RPO, consistency need (CAP), batch windows | ✅ (manual, sample bundle, collector format; telemetry parsing not yet) |
| 1.6 | **Risk register** | End of life, security, coupling, data, capacity, knowledge, compliance; each with severity, citation and mitigation | ✅ (rule-based) |
| 1.7 | **Evidence register & gaps** | Every source per evidence view: provided, extracted, answered by a person, not applicable (with reason), or missing; who provided it and when | ✅ |

## Deliverable 2 — Target Decision Record (end of Phase 2: Deciding)

Per bounded context:

| # | Section | What it contains | Status |
|---|---|---|---|
| 2.1 | **Approach** | One of the 7 Rs (rehost, replatform, refactor, rebuild, replace, encapsulate, retire), with strategic value and criticality | ✅ |
| 2.2 | **Target design** | Architecture, language/runtime, database, integration, cloud/hosting; replacement product or retirement destination | ✅ |
| 2.3 | **Options considered** | Every option ranked, with +/− reasons from the evidence and the organization's rules; rejected and excluded options with why | ✅ |
| 2.4 | **Target services** | The chosen cloud's actual services (database, integration, hosting) | ✅ |
| 2.5 | **Hosting estimate** | Compute, database, storage and backups, sized for year-3 growth; resilience for the availability target | ✅ (rule of thumb) |
| 2.6 | **Consistency trade-off** | CAP/PACELC consequence and required patterns (single primary, outbox, sagas) | ✅ |
| 2.7 | **Decision** | Rationale, named decider, date; rule compliance re-checked | ✅ |
| 2.8 | **Cost comparison** | Target run cost and one-off cost vs. today's run cost | ⬜ (today's run cost is captured; target cost not yet) |

## Deliverable 3 — Modernization Roadmap & Approval (end of Phase 3: Plan)

| # | Section | What it contains | Status |
|---|---|---|---|
| 3.1 | **Foundations** | Landing zone, identity, CI/CD, tests safety net, data migration, compliance controls, knowledge transfer, interim patching | ✅ (generated from the evidence) |
| 3.2 | **Waves** | Contexts sequenced into waves with dates, migration pattern (strangler fig, parallel run…), size, dependencies | ✅ |
| 3.3 | **Constraints check** | Freeze windows (peaks and company freezes), hard deadline fit, delivery capacity | ✅ |
| 3.4 | **What the plan must achieve** | Why now, deadline, run-cost baseline, success criteria | ✅ |
| 3.5 | **Approval** | Readiness checks; named approver (business owner); "changed since approval" detection | ✅ |

## Formats

| Format | Status |
|---|---|
| Interactive pages in the app (each phase and step) | ✅ |
| Printable report / PDF (sign-off page) | ◐ (sign-off page prints; full dossier report not yet) |
| Machine-readable JSON export of everything, including the full review history | ✅ |

---

## What the client must provide (minimum)
- **Start:** project name, system being modernized, system type, why now, hard deadline, success criteria.
- **Access to evidence:** at least the source code. The database export, environment inventory, telemetry and the rest raise quality, and come through exports or the Discovery Collector.
- **Human answers:** strategic value, criticality, availability and consistency per business area; regulation; business owner; budget, risk appetite and run cost when deciding; freeze periods and team capacity when planning.

## Biggest gaps to close (in order)
1. **Code analysis report** (1.4): ✅ built (browser scan + agent backend); remaining: licences and vulnerabilities.
2. **Business story** (1.2): ✅ agents built in the backend; next: test with a real API key on a real codebase, and add documents as input.
3. **Parsers for the other source exports** (inventory, scheduler, telemetry, tickets), and the **Discovery Collector**.
4. **Cost comparison** (2.8) and the **full dossier report** (PDF).
