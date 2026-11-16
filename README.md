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
