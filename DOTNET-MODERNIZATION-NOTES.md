# MindSuite handoff notes on legacy .NET modernization

Research reference for a future .NET discovery pilot. Reviewed 2026-11-18. The workflow and outputs below are proposals for MindSuite.

## What the article contributes

Niharika Gupta walks through a .NET Framework 4.8 Web API migration. The walkthrough combines mechanical project conversion with AI assistance for API, dependency injection and configuration changes. It suggests migrating in slices and comparing behavior before and after each change. The examples are a customer lookup endpoint and payment gateway configuration. [C# Corner article](https://www.c-sharpcorner.com/article/ai-assisted-legacy-net-modernization-using-github-copilot-app/)

Microsoft has deprecated the Upgrade Assistant used in the walkthrough. The current .NET Copilot upgrade workflow produces an assessment, a record of upgrade choices, a plan and a task list with validation criteria. Its documented repository artifacts include assessment.md, upgrade-options.md, plan.md and tasks.md under .github/upgrades/{scenarioId}. Use the artifact sequence as a reference for MindSuite's own export. [Upgrade Assistant deprecation](https://learn.microsoft.com/en-us/dotnet/core/porting/upgrade-assistant-overview) · [Copilot upgrade walkthrough](https://learn.microsoft.com/en-us/dotnet/core/porting/github-copilot-upgrade/dotnet-how-to-upgrade-with-github-copilot)

The article uses .NET 8 and 9 as examples. When planning a real migration, check Microsoft's live support table at decision time and record the chosen version, support end date and rationale. Evaluate the application's host OS and patch level separately from the framework targeted in its project file. Microsoft ties support for .NET Framework 4.8 and 4.8.1 to the parent Windows lifecycle. Confirm the deployed versions with the client. [Modern .NET support policy](https://dotnet.microsoft.com/en-us/platform/support/policy/dotnet-core) · [.NET Framework lifecycle FAQ](https://learn.microsoft.com/en-us/lifecycle/faq/dotnet-framework)

## Evidence to collect in the client's project workspace

| Area | Start with | Confirm with the client |
| --- | --- | --- |
| Build and dependencies | Solutions, C# and VB project files, project references, packages.config, Directory.Build.props/targets, Directory.Packages.props, global.json, lock files and CI build definitions. Record every target framework separately, including conditional targets where resolvable. | Actual SDK and build environment, inaccessible private packages, vendor support and dependencies absent from the checkout. |
| Web architecture and behavior | System.Web use, Global.asax, Web.config and transforms, routes, filters, HTTP modules and handlers, authentication, authorization, session state, ViewState, Web Forms code-behind and WCF contracts where present. | Critical journeys, partner clients, required SOAP or HTTP contracts, and behavior implemented by the host or a separately deployed component. |
| Deployment and platform | Sanitized IIS site and app pool inventory, Windows Server build, deployment manifests and runtime inventory. Look for COM, registry access, native DLLs, Windows authentication, file shares, jobs and integrations as candidate constraints. | What runs in each environment, which dependencies must stay on Windows, whether Linux is required, and whether parallel operation is possible. |
| Data and operations | Database access and stored-procedure references, sanitized database exports, tests, build results and measured traffic or recovery requirements. | Transaction boundaries, data ownership, cutover windows, rollback limits and the behaviors each owner will sign off. |

Keep evidence and agent runs tied to the active workspace and record the repository revision or export date. Label source observations, runtime observations, assumptions and owner confirmations separately. Redact credential values and keep customer code read-only during discovery. Preserve existing user overrides when evidence is reprocessed.

## Map the findings to MindSuite's workflow

1. **Discover.** Build a per-project compatibility inventory with exact source citations. Separate mechanical candidates such as project file conversion from behavior changes such as session or authorization migration. Keep unresolved references and unknown runtime facts visible. The current browser scan already reads basic project files, and the Go Code team in backend/internal/teams/code has basic NuGet and target-framework extraction in inventory.go plus platform hints in platform.go. A future pilot should deepen these paths rather than introduce a separate scanner.
2. **Decide.** Compare an interim Windows-hosted path, an in-place upgrade and a side-by-side migration when those options fit. Choose upgrade order from the dependency graph and business risk. Evaluate framework, hosting and database changes as separate decisions. For ASP.NET Framework applications, check the scope of System.Web adapters and the incremental migration tooling. For Web Forms, explicitly assess pages, controls, postbacks and state before proposing a new UI. [Microsoft ASP.NET migration tooling](https://learn.microsoft.com/en-us/aspnet/core/migration/fx-to-core/tooling) · [Microsoft Copilot scenarios and skills](https://learn.microsoft.com/en-us/dotnet/core/porting/github-copilot-app-modernization/scenarios-and-skills)
3. **Plan.** Propose a small first slice, ideally with a clear interface and a measurable baseline. Export a task list from accepted findings. Each task should carry a finding ID and citation, affected projects and contracts, dependencies, owner, completion criteria, rollback approach and status. Store the plan and later verification evidence with the same project. An execution team could use this handoff in Copilot or its chosen development tool.
4. **Validate.** Capture current behavior before code changes. Confirm build and test results in the appropriate environment and compare behavior on representative data. Check authorization, API and data contracts, configuration precedence, cancellation and timeouts, and operational measures where they matter. Estimate effort and hosting savings from the pilot's measured work and telemetry. The article provides no reproducible baseline for its automation or cost claims.

## Example task for a future pilot

**Port one customer lookup route.** Cite its current route, handler, authorization policy and repository. Establish examples for a known customer, a missing customer, a malformed ID and an unauthorized request. Preserve the agreed verb, public path, status codes and response contract. The article's rewritten example adds an integer route constraint and introduces asynchronous data access, both of which deserve explicit parity checks. If using parallel routing, define how to return that route to the old application and what evidence authorizes switching it back.

**Configuration follow-up.** Inventory which gateway settings come from Web.config, deployment transforms or runtime overrides. Bind the settings in the target app and check override precedence, missing values and retry behavior. Treat IIS modules, authentication, rewrites and binding redirects as their own findings. A JSON settings file covers only part of what Web.config can hold.

## Suggested first implementation slice

Use a synthetic old-style ASP.NET solution with a library, packages.config, a Web.config transform and a controller. Verify that MindSuite records each project and target independently, links dependencies, distinguishes observed deployment evidence from a source hint, and emits one reviewable migration task with a citation and parity tests. A second pass should retain a human override. Keep Linux readiness and lifecycle status unknown until the necessary evidence is recorded.

## Candidate public test fixture

[eShopModernizing](https://github.com/dotnet-architecture/eShopModernizing) is an archived, MIT-licensed sample of back-office product catalog management. Start with eShopLegacyWebFormsSolution alone in a dedicated test workspace and record the source commit. Add the MVC solution, the separate WinForms client and WCF service, and Windows-container manifests as separate inputs or runs. This makes it possible to measure whether MindSuite distinguishes hosting changes from application modernization. Keep the upstream notice with any bundled copy. [Upstream license](https://github.com/dotnet-architecture/eShopModernizing/blob/main/LICENSE)

- Run the browser scan and Go Code team with Claude off to establish the deterministic baseline. Check framework and application type, IIS configuration hints and the SQL Server dependency. Test WCF and WinForms recognition as coverage probes.
- Run the same sample with Claude enabled for that run when authorized. The expected semantic result is a cited explanation of product catalog management and how UseMockData selects an in-memory catalog or the SQL-backed implementation through dependency injection. Review what was sent to the model. [Upstream walkthrough](https://github.com/dotnet-architecture/eShopModernizing/wiki/01.-Tour-of-the-ASP.NET-MVC-and-WebForms-apps-implementation-code)
- Compare the reports for correct findings, false positives and unknowns. Running the sample is unnecessary for source discovery. Production topology, load and incident claims require separate evidence. The upstream modernized variants demonstrate Windows-container lift-and-shift. A modern .NET port and its parity checks would be a separate exercise.

Status: suitability reviewed. MindSuite and Claude runs on this sample are pending.
