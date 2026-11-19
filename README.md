# MindSuite Doctrine of Legacy Modernization

## Purpose

We are defining a web system for legacy modernization. The first step is to classify the legacy systems it will assess, so that discovery can account for the different technologies, architectures and business contexts involved.

## Legacy system categories

The initial classification groups systems by technology and architecture. These categories can overlap; a single business platform may contain components belonging to several categories.

| Category | What it includes | Typical examples |
| --- | --- | --- |
| Mainframe and midrange systems | Centralized systems running core business transactions and batch jobs | COBOL, CICS, JCL, IBM AS/400 |
| Desktop applications | Software installed on individual computers, sometimes connected to shared databases | VB6, Windows Forms, older Java Swing applications |
| Client-server systems | Installed client applications communicating with central application or database servers | PowerBuilder, Oracle Forms, thick-client business applications |
| Web applications and monoliths | Browser-based systems and applications with tightly coupled business functions | Classic ASP, older ASP.NET, Java EE, PHP monoliths |
| Packaged enterprise systems | Commercial business products with aging versions or extensive customizations | ERP, CRM, HR and finance systems |
| Data and reporting systems | Databases, data pipelines, warehouses and reports that need modernization | Stored-procedure-heavy databases, legacy ETL, data warehouses |
| Integration and batch systems | Components that connect applications or run scheduled processing | ESBs, file transfers, SOAP services, scheduled scripts |
| Embedded and industrial systems | Software tied closely to equipment, devices or operational environments | Factory controllers, device firmware, industrial monitoring systems |

## Applying the classification

Classify components individually when a business platform spans multiple categories. For example, an insurance platform may include a COBOL mainframe, a desktop application, an Oracle database and nightly file integrations. Each component receives its relevant classification while remaining linked to the overall platform.

Record business purpose and business criticality separately from technology and architecture. Business purpose describes what the system supports, such as finance, orders, inventory or HR. Business criticality describes how important the system is to operations and the impact of its unavailability.

Technology examples identify the type of system; their presence alone does not establish that modernization is necessary.

## Verify before deciding

Discovery produces findings; deciding needs facts. Between the two, every finding is checked by the people who know the system. This step applies to every project and every category of legacy system.

### Principles

1. **A finding is a claim, not a fact.** Whatever the agents or the AI produce is a proposal until the right person confirms it. Evidence shows where a claim came from, not that it is true.
2. **Every claim belongs to a role, not a person.** The doctrine defines the roles; each project assigns people to them. Each type of finding goes to one role:

   | Finding type | Verified by |
   | --- | --- |
   | Business rules, processes and workflows | Process owner |
   | Data meaning, ownership and quality | Data owner |
   | Users, roles and access | Business owner, with security |
   | Interfaces, batch jobs and schedules | System custodian (IT or operations) |
   | Security and compliance | Security or compliance officer |
   | Product versions, licences and support | Vendor or contract owner |

3. **One standard verification form.** The form has the same structure in every project; only its content changes. For each claim it shows the claim in plain words and the evidence behind it, and records the answer, the person and the date. The answer is one of:
   - **Confirmed**: the claim is true.
   - **Corrected**: the claim is partly wrong; the person gives the right version.
   - **Rejected**: the claim is wrong; the person gives the reason.
   - **Not mine**: the person names who should answer instead.

   The questions follow the system's category: a mainframe's batch schedule raises different questions from a desktop application's local database.
4. **Verification is a gate.** A decision may rest only on confirmed or corrected claims. Rejected claims are kept with their reason, because they show where the evidence misled. A claim nobody answered stays an open assumption and is listed as a risk in the decision record.
5. **The AI suggests and a person decides.** Approach suggestions (retain, rehost, replatform, refactor, rebuild, replace, retire) are drawn only from verified claims, using stated rules: the system's category, its business criticality, how strategic the area is, and the evidence. Each suggestion gives its reasons and what would change it. A named person makes the decision and records why.
6. **Everything traces back.** Each decision links to the claims it rests on; each claim links to its evidence and to who confirmed it. When a claim changes, every decision that rests on it reopens.
7. **The method improves over time.** The doctrine tracks how often each type of finding is confirmed or rejected across projects. Types that are often wrong get a lower confidence or a different check.

### Policy

Decided once, for all projects:

- **How the form reaches people:** as a file. MindSuite makes one self-contained HTML form per person with only their claims. The person opens it in any browser, answers, and saves an answers file that they send back. Nothing is hosted.
- **Who sends it:** the project owner, by their own email or chat. MindSuite only makes the files; it sends nothing.
- **What the AI may see:** only verified claims and their verdicts, and only for a run the owner allows. Names and comments never leave the owner's machine.
- **When nobody answers:** the claim stays an open assumption. The owner may accept it with a written reason; it is then listed as a risk in the decision record.

In the web app this is **Verify with people**, after the Discovery summary: collect the claims from the agent reports, choose a person for each role, make each person's form, and import the answers files.

In **Target decisions** each decision card shows the claims it rests on: those that name the context (its name, its components or their paths) and those that name no context, which apply to the whole system. A decision cannot be recorded while any of them is unanswered, unless the owner accepts them as open assumptions with a reason; the decision record then lists them as risks, and Sign-off warns about them. Rejected claims are listed with their reason but not used. When a claim the decision rests on is answered, changes, disappears or a new one appears, the decision reopens with what changed.

Each card can also ask Claude to **suggest approaches**. Only the confirmed and corrected claims are sent (claim text, kind, role, verdict and the right version of a corrected claim), with the context's strategic value and criticality; names, comments, who answered, evidence quotes and file paths, rejected and unverified claims stay on the machine, and the backend refuses any other field. The owner sees exactly what will be sent and ticks a box that covers one request. Claude gives up to four approaches with a fit, reasons that cite the claims they rest on, and what would change each one; the suggestion is marked out of date when the verified claims change. "Use this approach" only fills in the approach: the named person still records the decision and its rationale.
