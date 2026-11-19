package code

import (
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"
)

// ---------- Evidence areas seen in the code ----------
//
// The specialist teams (database, infrastructure, messaging, identity, telemetry, batch, network,
// documents, incidents) read their own exports. Source code often shows part of the same picture,
// so the code team records what it can see here and the report groups it per evidence area.

type areaPattern struct {
	tech string
	re   *regexp.Regexp
}

var (
	telemetryPatterns = []areaPattern{
		{"OpenTelemetry", regexp.MustCompile(`(?i)opentelemetry`)},
		{"Application Insights", regexp.MustCompile(`(?i)applicationinsights|\bTelemetryClient\b`)},
		{"Prometheus / Micrometer", regexp.MustCompile(`(?i)\bprometheus\b|io\.micrometer`)},
		{"Datadog", regexp.MustCompile(`(?i)\bdatadog\b|\bddtrace\b|dd-trace`)},
		{"New Relic", regexp.MustCompile(`(?i)\bnewrelic\b|new-relic`)},
		{"Dynatrace", regexp.MustCompile(`(?i)\bdynatrace\b|oneagent`)},
		{"AppDynamics", regexp.MustCompile(`(?i)\bappdynamics\b`)},
		{"Elastic APM", regexp.MustCompile(`(?i)elastic-apm|elasticapm|co\.elastic\.apm`)},
		{"Sentry", regexp.MustCompile(`(?i)\bio\.sentry\b|@sentry/|sentry_sdk|\bSentrySdk\b|sentry-`)},
		{"Log4j", regexp.MustCompile(`org\.apache\.log4j|org\.apache\.logging\.log4j|\blog4j2?\.(xml|properties)\b`)},
		{"SLF4J / Logback", regexp.MustCompile(`org\.slf4j|\blogback\b`)},
		{"Serilog", regexp.MustCompile(`\bSerilog\b`)},
		{"NLog", regexp.MustCompile(`\bNLog\b`)},
		{"log4net", regexp.MustCompile(`\blog4net\b`)},
		{"Microsoft.Extensions.Logging", regexp.MustCompile(`Microsoft\.Extensions\.Logging|\bILogger<`)},
		{"Windows Event Log", regexp.MustCompile(`EventLog\.WriteEntry|\bEventLog\.CreateEventSource`)},
		{"winston / pino", regexp.MustCompile(`require\(['"](winston|pino)['"]\)|from ['"](winston|pino)['"]`)},
		{"Python logging", regexp.MustCompile(`^\s*import logging\b|logging\.getLogger\(`)},
		{"Go logging (zap / logrus / slog)", regexp.MustCompile(`go\.uber\.org/zap|sirupsen/logrus|"log/slog"`)},
		{"Health check endpoint", regexp.MustCompile(`["'/](actuator/health|healthz|health)["'/?]|\bAddHealthChecks\(|\bMapHealthChecks\(`)},
		{"CICS operator messages", regexp.MustCompile(`(?i)\bEXEC\s+CICS\s+WRITE\s+OPERATOR\b`)},
		{"CICS journal", regexp.MustCompile(`(?i)\bEXEC\s+CICS\s+WRITE\s+JOURNALNAME\b`)},
		{"Operator console (WTO)", regexp.MustCompile(`^\S*\s+WTOR?\s`)},
	}

	networkPatterns = []struct {
		kind, tech string
		re         *regexp.Regexp
	}{
		{"tls-config", "TLS / certificates", regexp.MustCompile(`(?i)\b(keystore|truststore|ssl\.?(enabled|protocol|key)|server\.ssl\.|\.jks\b|\.pfx\b|\.p12\b|\.pem\b|requireSSL|RequireHttps|UseHttpsRedirection|HSTS|UseHsts)`)},
		{"edge-config", "URL rewrite / proxy rules", regexp.MustCompile(`(?i)<rewrite>|\bproxy_pass\b|\bProxyPass\b|\bRewriteRule\b|\bupstream\s+\w+\s*\{|\bX-Forwarded-For\b|ForwardedHeaders`)},
		{"edge-config", "CICS web / TCP/IP service", regexp.MustCompile(`(?i)\bDEFINE\s+(TCPIPSERVICE|URIMAP)\b|\b(TCPIPSERVICE|URIMAP)\s*\(`)},
		{"edge-config", "IIS bindings / site", regexp.MustCompile(`(?i)<binding\s+protocol=|<site\s+name=`)},
	}
	configURLRe = regexp.MustCompile(`(?i)\b(https?|ftps?|sftp|ldaps?|wss?)://([a-z0-9][a-z0-9.\-]*[a-z0-9])(:\d{2,5})?`)

	mqVerbRe      = regexp.MustCompile(`(?i)\bCALL\s+['"](MQOPEN|MQGET|MQPUT1?|MQCLOSE|MQCONNX?|MQINQ|MQSET|MQCMIT|MQBACK|MQSUB)['"]`)
	cicsQueueRe   = regexp.MustCompile(`(?i)\bEXEC\s+CICS\s+(WRITEQ|READQ|DELETEQ)\s+(TD|TS)\b`)
	cicsQueueName = regexp.MustCompile(`(?i)\b(QUEUE|QNAME)\s*\(\s*['"]?([A-Z0-9$#@\-]+)['"]?\s*\)`)

	ticketRe    = regexp.MustCompile(`\b(INC|CHG|CRQ|PRB|RFC|RITM|SR)\d{5,}\b`)
	defectRe    = regexp.MustCompile(`(?i)\b(bug|defect|incident|hotfix|problem|change request|CR)\s*(#|no\.?|number)?\s*\d{3,}\b`)
	knownIssue  = regexp.MustCompile(`\b(TODO|FIXME|HACK|XXX|WORKAROUND|KLUDGE)\b`)
	changeLogRe = regexp.MustCompile(`(?i)^(change ?log|changes|history|release[-_ ]?notes|news)(\.[a-z]+)?$`)
	mermaidRe   = regexp.MustCompile("^\\s*```\\s*(mermaid|plantuml|puml)\\b|^\\s*@startuml\\b")
)

func isConfigLang(lang string) bool {
	switch lang {
	case "XML", "JSON", "YAML", "Properties", "Config", "TOML":
		return true
	}
	return false
}

func commentLine(f *File, l string) bool {
	t := strings.TrimSpace(l)
	if f.Lang == "COBOL" || f.Lang == "COBOL copybook" {
		return len(l) > 6 && (l[6] == '*' || l[6] == '/')
	}
	return strings.HasPrefix(t, "//") || strings.HasPrefix(t, "#") || strings.HasPrefix(t, "*") || strings.HasPrefix(t, "/*") ||
		strings.HasPrefix(t, "--") || strings.HasPrefix(t, "'") || strings.HasPrefix(t, "REM ") || strings.HasPrefix(t, "<!--")
}

// runAreas records what the code shows about telemetry, network and edge, documents, incidents and
// changes, mainframe queues and scheduler exports.
func runAreas(r *Run) {
	s := r.Store
	seen := map[string]bool{}
	type tally struct {
		f     *File
		line  int
		count int
	}
	issues := map[string]*tally{}  // module → TODO/FIXME
	tickets := map[string]*tally{} // file → ticket references
	telem := map[string]*tally{}   // tech|module
	telemOrder := []string{}
	docs := 0

	for _, f := range r.Code.Files {
		base := strings.ToLower(path.Base(f.Path))
		code := CodeLangs[f.Lang] || LegacyLangs[f.Lang]
		config := isConfigLang(f.Lang)

		// Documents and diagrams kept with the code.
		if f.Lang == "Markdown" || f.Lang == "Document" || f.Lang == "Diagram" || strings.HasPrefix(base, "readme") {
			if docs < 80 && !changeLogRe.MatchString(strings.TrimSuffix(base, path.Ext(base))) {
				docs++
				kind := "Document"
				if f.Lang == "Diagram" {
					kind = "Diagram"
				}
				heading := ""
				diagrams := 0
				for _, l := range f.Lines {
					t := strings.TrimSpace(l)
					if heading == "" && strings.HasPrefix(t, "#") {
						heading = strings.TrimSpace(strings.TrimLeft(t, "#"))
					}
					if mermaidRe.MatchString(l) {
						diagrams++
					}
				}
				title := kind + ": " + f.Path
				if heading != "" {
					title += " — " + short(heading, 80)
				}
				detail := fmt.Sprintf("%d lines.", len(f.Lines))
				if diagrams > 0 {
					detail += fmt.Sprintf(" Contains %d diagram(s) as text (Mermaid / PlantUML).", diagrams)
				}
				fd := Finding{Agent: "inv", Kind: "document", Module: f.Module, Title: title, Detail: detail, Confidence: "high",
					Data: map[string]any{"type": strings.ToLower(kind), "lines": len(f.Lines), "diagrams": diagrams}}
				if len(f.Lines) > 0 {
					fd.Citations = []Citation{cite(f, 1)}
				}
				s.Add(fd)
				r.Cover("inv", f)
			}
		}

		// Change logs and release notes.
		if changeLogRe.MatchString(strings.TrimSuffix(base, path.Ext(base))) && len(f.Lines) > 0 {
			s.Add(Finding{Agent: "qua", Kind: "change-record", Module: f.Module, Title: "Change log: " + f.Path, Detail: fmt.Sprintf("%d lines of recorded changes.", len(f.Lines)),
				Citations: []Citation{cite(f, 1)}, Confidence: "high", Data: map[string]any{"lines": len(f.Lines)}})
			r.Cover("qua", f)
		}

		// Scheduler exports shipped with the code (Control-M, CA-7).
		if f.Lang == "Scheduler export" && len(f.Lines) > 0 {
			s.Add(Finding{Agent: "pla", Kind: "schedule", Module: f.Module, Title: "Scheduler export: " + f.Path,
				Detail:    "A job schedule is kept with the code. The Batch & scheduling team reads it in full: jobs, conditions and calendars.",
				Citations: []Citation{cite(f, 1)}, Confidence: "high", Data: map[string]any{"tech": "scheduler export", "file": f.Path}})
			r.Cover("pla", f)
		}

		if !code && !config {
			continue
		}
		var mq *struct {
			line  int
			calls []string
		}
		for i, l := range f.Lines {
			n := i + 1
			comment := commentLine(f, l)

			// Incidents and changes: ticket references and known issues, written in comments.
			if comment || strings.Contains(l, "//") || strings.Contains(l, "#") || f.Lang == "COBOL" {
				if ticketRe.MatchString(l) || (comment && defectRe.MatchString(l)) {
					t := tickets[f.Path]
					if t == nil {
						t = &tally{f: f, line: n}
						tickets[f.Path] = t
					}
					t.count++
				}
				if knownIssue.MatchString(l) && (comment || f.Lang != "COBOL") {
					t := issues[f.Module]
					if t == nil {
						t = &tally{f: f, line: n}
						issues[f.Module] = t
					}
					t.count++
				}
			}
			if comment {
				continue
			}

			// Telemetry: logging, metrics, tracing, health checks.
			for _, p := range telemetryPatterns {
				if p.re.MatchString(l) {
					k := p.tech + "|" + f.Module
					t := telem[k]
					if t == nil {
						t = &tally{f: f, line: n}
						telem[k] = t
						telemOrder = append(telemOrder, k)
					}
					t.count++
				}
			}

			// Network and edge.
			for _, p := range networkPatterns {
				if p.re.MatchString(l) && once(seen, p.kind+"|"+p.tech+"|"+f.Module) {
					s.Add(Finding{Agent: "pla", Kind: p.kind, Module: f.Module, Title: p.tech + " in " + f.Path, Citations: []Citation{cite(f, n)}, Confidence: "medium", Data: map[string]any{"tech": p.tech}})
					r.Cover("pla", f)
				}
			}
			if config {
				for _, m := range configURLRe.FindAllStringSubmatch(l, -1) {
					host := strings.ToLower(m[2])
					if host == "localhost" || strings.HasPrefix(host, "127.") || strings.HasSuffix(host, "w3.org") || strings.HasSuffix(host, "apache.org") ||
						strings.HasSuffix(host, "maven.org") || strings.HasSuffix(host, "microsoft.com") || strings.HasSuffix(host, "springframework.org") ||
						strings.HasSuffix(host, "xmlsoap.org") || strings.HasSuffix(host, "oracle.com") || strings.HasSuffix(host, "sun.com") || strings.HasSuffix(host, "schemas.android.com") || !once(seen, "url|"+host) {
						continue
					}
					s.Add(Finding{Agent: "pla", Kind: "network-target", Module: f.Module, Title: "Configured address: " + strings.ToLower(m[1]) + "://" + host + m[3],
						Detail:    "A host this system is set up to reach. Check it against the firewall rules and load balancer configuration.",
						Citations: []Citation{cite(f, n)}, Confidence: "medium", Data: map[string]any{"protocol": strings.ToLower(m[1]), "host": host, "port": strings.TrimPrefix(m[3], ":")}})
					r.Cover("pla", f)
				}
			}

			// Messaging in mainframe programs: IBM MQ calls and CICS queues.
			if f.Lang == "COBOL" || f.Lang == "PL/I" {
				if m := mqVerbRe.FindStringSubmatch(l); m != nil {
					if mq == nil {
						mq = &struct {
							line  int
							calls []string
						}{line: n}
					}
					if c := strings.ToUpper(m[1]); !contains(mq.calls, c) {
						mq.calls = append(mq.calls, c)
					}
				}
				if m := cicsQueueRe.FindStringSubmatch(l); m != nil {
					verb, typ := strings.ToUpper(m[1]), strings.ToUpper(m[2])
					name := ""
					for j := i; j < len(f.Lines) && j < i+6; j++ {
						if q := cicsQueueName.FindStringSubmatch(f.Lines[j]); q != nil {
							name = strings.ToUpper(q[2])
							break
						}
					}
					prog := programName(f)
					if typ == "TS" {
						if once(seen, "ts|"+prog+"|"+name) {
							s.Add(Finding{Agent: "pla", Kind: "cache", Module: f.Module, Title: "Caching: CICS temporary storage queue " + orUnknown(name) + " in " + prog,
								Detail:    "CICS temporary storage keeps data between transactions in memory or auxiliary storage. A modern design needs a cache or session store for it.",
								Citations: []Citation{cite(f, n)}, Confidence: "medium", Data: map[string]any{"tech": "CICS temporary storage", "queue": name}})
						}
					} else if once(seen, "td|"+prog+"|"+verb+"|"+name) {
						dir := "publishes"
						if verb == "READQ" {
							dir = "consumes"
						}
						s.Add(Finding{Agent: "int", Kind: "messaging", Module: f.Module, Title: fmt.Sprintf("%s %s %s (CICS transient data queue)", prog, dir, orUnknown(name)),
							Citations: []Citation{cite(f, n)}, Confidence: "high", Data: map[string]any{"tech": "CICS transient data", "direction": dir, "destination": name}})
					}
					r.Cover("int", f)
				}
			}
		}
		if mq != nil {
			prog := programName(f)
			reads, writes := contains(mq.calls, "MQGET"), contains(mq.calls, "MQPUT") || contains(mq.calls, "MQPUT1")
			dir := "uses queues on"
			switch {
			case reads && writes:
				dir = "consumes and publishes messages on"
			case reads:
				dir = "consumes messages from"
			case writes:
				dir = "publishes messages to"
			}
			s.Add(Finding{Agent: "int", Kind: "messaging", Module: f.Module, Title: prog + " " + dir + " IBM MQ",
				Detail:    "Calls " + strings.Join(mq.calls, ", ") + ". The queue names are usually set at run time; take them from the queue manager.",
				Citations: []Citation{cite(f, mq.line)}, Confidence: "high", Data: map[string]any{"tech": "IBM MQ", "direction": dir, "calls": mq.calls}})
			r.Cover("int", f)
		}
	}

	for _, k := range telemOrder {
		t := telem[k]
		tech := strings.SplitN(k, "|", 2)[0]
		s.Add(Finding{Agent: "pla", Kind: "telemetry", Module: t.f.Module, Title: tech + " in " + t.f.Module, Detail: fmt.Sprintf("%d lines in this module.", t.count),
			Citations: []Citation{cite(t.f, t.line)}, Confidence: "medium", Data: map[string]any{"tech": tech, "lines": t.count}})
		r.Cover("pla", t.f)
	}
	for _, m := range sortedKeys(issues) {
		t := issues[m]
		s.Add(Finding{Agent: "qua", Kind: "known-issue", Module: m, Title: fmt.Sprintf("%d TODO / FIXME notes in %s", t.count, m),
			Detail: "Known problems or unfinished work written by developers. Compare them with the incident history.", Citations: []Citation{cite(t.f, t.line)}, Confidence: "high", Data: map[string]any{"count": t.count}})
	}
	n := 0
	for _, p := range sortedKeys(tickets) {
		if n >= 40 {
			break
		}
		n++
		t := tickets[p]
		s.Add(Finding{Agent: "qua", Kind: "incident-ref", Module: t.f.Module, Title: fmt.Sprintf("%d incident, defect or change references in %s", t.count, p),
			Detail: "Comments that name tickets or defects. Match them with the ticket system to see which code fails or changes most.", Citations: []Citation{cite(t.f, t.line)}, Confidence: "medium", Data: map[string]any{"count": t.count}})
	}
}

func programName(f *File) string {
	return strings.ToUpper(strings.TrimSuffix(path.Base(f.Path), path.Ext(f.Path)))
}

func orUnknown(s string) string {
	if s == "" {
		return "(name set at run time)"
	}
	return s
}

// evidenceArea is one section of the "What the code shows per evidence area" part of the report.
// The order and names follow the agent teams in the web app.
type evidenceArea struct {
	title, team, other string
	kinds              []string
	keep               func(Finding) bool
	// parts are the fact agents (part keys) that look for this area; when one fails, the area is not fully checked.
	parts []string
}

func dbConnection(f Finding) bool { return strings.HasPrefix(f.Title, "Database connection") }

var evidenceAreas = []evidenceArea{
	{"Database", "database", "schema exports, DDL and query statistics",
		[]string{"shared-table", "table-access", "legacy-sql", "stored-procedure", "procedure-call", "db-object", "ims-database", "ims-view", "ims-use", "connection", "legacy-file", "dataset"}, nil, []string{"dat", "pla", "leg"}},
	{"Environment and infrastructure", "infrastructure", "server, cloud and infrastructure-as-code exports",
		[]string{"app-server", "container", "k8s-object", "iac-resource", "ci-pipeline", "environment"}, nil, []string{"pla"}},
	{"Messaging and caching", "messaging", "queue manager and broker exports",
		[]string{"messaging", "messaging-tech", "file-exchange", "cache"}, nil, []string{"int", "areas", "pla"}},
	{"Identity and access", "identity", "directory, RACF / ACF2 and role exports",
		[]string{"auth-method", "role-check", "secret", "weak-crypto", "injection-risk", "csrf-disabled", "cors-any"}, nil, []string{"sec"}},
	{"Telemetry", "telemetry", "monitoring, logging and APM exports",
		[]string{"telemetry"}, nil, []string{"areas"}},
	{"Batch and scheduling", "batch", "scheduler exports and run history",
		[]string{"schedule", "batch-job", "job-step"}, nil, []string{"pla", "leg", "areas"}},
	{"Network and edge", "network", "firewall, load balancer, DNS and proxy exports",
		[]string{"outbound-call", "soap-service", "network-target", "tls-config", "edge-config", "connection"}, nil, []string{"int", "areas", "pla"}},
	{"Documents and diagrams", "documents", "design documents, runbooks and diagrams",
		[]string{"document"}, nil, []string{"areas"}},
	{"Incidents and changes", "incidents", "ticket, incident and change exports",
		[]string{"change-record", "incident-ref", "known-issue", "churn-hotspot", "bus-factor"}, nil, []string{"areas", "qua"}},
}

func init() {
	// A connection string belongs to the database area when it names a database, otherwise to the network.
	evidenceAreas[0].keep = func(f Finding) bool { return f.Kind != "connection" || dbConnection(f) }
	evidenceAreas[6].keep = func(f Finding) bool { return f.Kind != "connection" || !dbConnection(f) }
}

// writeAreas writes what the code shows for each evidence area, so the source code report also covers
// databases, infrastructure, messaging and the rest when the code reveals them.
// failed lists the fact agents that did not finish (by part key): their areas are marked as not, or only partly, checked.
func writeAreas(b *strings.Builder, accepted func(kinds ...string) []Finding, failed map[string]Problem) {
	w := func(format string, a ...any) { fmt.Fprintf(b, format, a...) }
	w("# What the code shows for each evidence area\n\n")
	w("The source code often shows part of what the other agent teams read from their own evidence. Each area below lists what the code shows; the team for that area confirms and completes it.\n\n")
	for _, a := range evidenceAreas {
		var fs []Finding
		for _, f := range accepted(a.kinds...) {
			if a.keep == nil || a.keep(f) {
				fs = append(fs, f)
			}
		}
		// The area's kinds are listed most telling first (tables before data sets, schedules before job steps).
		rank := map[string]int{}
		for i, k := range a.kinds {
			rank[k] = i
		}
		sort.SliceStable(fs, func(i, j int) bool { return rank[fs[i].Kind] < rank[fs[j].Kind] })
		w("## %s\n\n", a.title)
		var down []string
		for _, p := range a.parts {
			if pr, ok := failed[p]; ok && !contains(down, pr.Name) {
				down = append(down, pr.Name)
			}
		}
		if len(down) > 0 {
			who := strings.Join(down, " and ")
			if len(fs) == 0 {
				w("**Not checked:** the %s agent did not finish, so the code was not fully searched for this area. Run the agent again, or run the %s team with %s.\n\n", who, a.title, a.other)
				continue
			}
			w("**Partly checked:** the %s agent did not finish, so findings may be missing below.\n\n", who)
		}
		if len(fs) == 0 {
			w("Nothing about this area was found in the code. Run the %s team with %s.\n\n", a.title, a.other)
			continue
		}
		noun := "findings"
		if len(fs) == 1 {
			noun = "finding"
		}
		w("%d %s from the code. Confirm them with the %s team (%s).\n\n", len(fs), noun, a.title, a.other)
		for i, f := range fs {
			if i >= 40 {
				w("- … and %d more (see the full findings)\n", len(fs)-40)
				break
			}
			mark := "✓"
			if f.Status != StatusAccepted {
				mark = "?"
			}
			loc := ""
			if len(f.Citations) > 0 {
				c := f.Citations[0]
				loc = fmt.Sprintf(" — `%s:%d`", c.File, c.Line)
			}
			w("- %s %s%s [%s]\n", mark, f.Title, loc, f.ID)
			if f.Detail != "" {
				w("  %s\n", strings.ReplaceAll(short(f.Detail, 300), "\n", " "))
			}
		}
		w("\n")
	}
}
