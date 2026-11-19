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
	writeAreas(&b, func(kinds ...string) []Finding { return run.Store.Findings(kinds...) }, nil)
	for _, a := range evidenceAreas {
		if !strings.Contains(b.String(), "## "+a.title+"\n") {
			t.Errorf("report has no section %q", a.title)
		}
	}
	if !strings.Contains(b.String(), "Nothing about this area was found in the code. Run the Environment and infrastructure team") {
		t.Error("an empty area does not say which team to run")
	}
}

func TestAreasNotCheckedWhenAgentFailed(t *testing.T) {
	run, err := agents.NewRun(agents.LoadFiles("x", map[string][]byte{"a.java": []byte("class A {}\n")}), agents.Options{}, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	run.Store.Add(Finding{Agent: "int", Kind: "outbound-call", Title: "Calls the bank", Citations: []Citation{{File: "a.java", Line: 1}}})
	var b strings.Builder
	writeAreas(&b, func(kinds ...string) []Finding { return run.Store.Findings(kinds...) },
		map[string]Problem{"areas": {Name: "Platform & runtime", Part: "areas"}})
	out := b.String()
	tel := out[strings.Index(out, "## Telemetry"):]
	if !strings.HasPrefix(tel, "## Telemetry\n\n**Not checked:** the Platform & runtime agent did not finish") {
		t.Errorf("telemetry should be not checked:\n%s", tel[:200])
	}
	if strings.Contains(tel[:300], "Nothing about this area was found") {
		t.Error("a failed area must not say nothing was found")
	}
	if !strings.Contains(out, "## Network and edge\n\n**Partly checked:** the Platform & runtime agent did not finish") {
		t.Error("network has findings and a failed agent: it should be partly checked")
	}
	if !strings.Contains(out, "## Identity and access\n\nNothing about this area was found") {
		t.Error("areas whose agents finished are unchanged")
	}
}
