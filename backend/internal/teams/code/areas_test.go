package code

import (
	"strings"
	"testing"

	"github.com/mjaffry01/mindsuite-doctrine-of-legacy-modernization/backend/internal/agents"
)

// TestEvidenceAreas: the code team records telemetry, network, documents, changes and mainframe queues,
// and the report has a section per evidence area.
func TestEvidenceAreas(t *testing.T) {
	files := map[string][]byte{
		"src/orders/Orders.java":                    []byte("package orders;\nimport org.slf4j.Logger;\nclass Orders {\n  // TODO: retry when the bank times out (INC0012345)\n  void run() {}\n}\n"),
		"src/main/resources/application.properties": []byte("server.ssl.enabled=true\npayments.url=https://pay.example.net:8443/api\nschema=http://www.w3.org/2001/XMLSchema\n"),
		"cbl/PAYMQ01.cbl":                           []byte("       IDENTIFICATION DIVISION.\n       PROCEDURE DIVISION.\n           CALL 'MQGET' USING W-HCONN.\n           CALL 'MQPUT' USING W-HCONN.\n           EXEC CICS WRITEQ TS QUEUE('PAYTSQ') FROM(W-DATA) END-EXEC.\n           GOBACK.\n"),
		"docs/architecture.md":                      []byte("# Payments architecture\n```mermaid\ngraph TD; A-->B\n```\n"),
		"CHANGELOG.md":                              []byte("## 1.2\n- fixed rounding\n"),
		"scheduler/jobs.controlm":                   []byte("<DEFTABLE>\n<JOB JOBNAME=\"PAYJOB\"/>\n</DEFTABLE>\n"),
	}
	run, err := agents.NewRun(agents.LoadFiles("pay", files), agents.Options{}, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	runAreas(run)
	got := map[string][]string{}
	for _, f := range run.Store.Findings() {
		got[f.Kind] = append(got[f.Kind], f.Title)
	}
	for kind, want := range map[string]string{
		"telemetry":      "SLF4J / Logback",
		"known-issue":    "TODO / FIXME",
		"incident-ref":   "incident, defect or change references",
		"tls-config":     "TLS / certificates",
		"network-target": "https://pay.example.net:8443",
		"messaging":      "PAYMQ01 consumes and publishes messages on IBM MQ",
		"cache":          "CICS temporary storage queue PAYTSQ",
		"document":       "Payments architecture",
		"change-record":  "CHANGELOG.md",
		"schedule":       "Scheduler export: scheduler/jobs.controlm",
	} {
		if !strings.Contains(strings.Join(got[kind], " | "), want) {
			t.Errorf("%s: want %q, got %q", kind, want, got[kind])
		}
	}
	if strings.Contains(strings.Join(got["network-target"], " "), "w3.org") {
		t.Error("schema addresses are not network targets")
	}

	var b strings.Builder
	writeAreas(&b, func(kinds ...string) []Finding { return run.Store.Findings(kinds...) })
	for _, a := range evidenceAreas {
		if !strings.Contains(b.String(), "## "+a.title+"\n") {
			t.Errorf("report has no section %q", a.title)
		}
	}
	if !strings.Contains(b.String(), "Nothing about this area was found in the code. Run the Environment and infrastructure team") {
		t.Error("an empty area does not say which team to run")
	}
}
