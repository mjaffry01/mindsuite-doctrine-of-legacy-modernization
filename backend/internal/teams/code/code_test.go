package code

import (
	"context"
	"encoding/json"
	"github.com/mjaffry01/mindsuite-doctrine-of-legacy-modernization/backend/internal/agents"
	"strings"
	"testing"
)

// TestSampleRun runs the whole team (without a model) on testdata/sample, a small
// Spring + COBOL + JCL system with known answers.
func TestSampleRun(t *testing.T) {
	code, err := agents.LoadDir("testdata/sample")
	if err != nil {
		t.Fatal(err)
	}
	run, err := agents.NewRun(code, agents.Options{}, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	run.Execute(context.Background())
	snap := run.Snapshot()
	if snap.Run.Status != "done" {
		t.Fatalf("status %s: %s", snap.Run.Status, snap.Run.Error)
	}
	has := func(kind, contains string) bool {
		for _, f := range snap.Findings {
			if f.Kind == kind && strings.Contains(f.Title, contains) && f.Status != StatusRejected {
				return true
			}
		}
		return false
	}
	want := []struct{ kind, text string }{
		{"module", "order"}, {"module", "billing"}, {"module", "customer"},
		{"cycle", "billing ↔ order"},
		{"module-edge", "order depends on customer"},
		{"table-access", "order writes ORDERS"}, {"table-access", "billing writes ORDERS"},
		{"shared-table", "ORDERS"},
		{"endpoint", "GET /api/orders/{id}"}, {"endpoint", "POST /api/orders/{id}/approve"},
		{"role-check", "hasRole('MANAGER')"},
		{"auth-method", "Spring Security"},
		{"secret", "password"},
		{"weak-crypto", "MD5"},
		{"injection-risk", "SQL"},
		{"outbound-call", "payments.partnerbank.com"},
		{"runtime", "Java 1.8"},
		{"dependency", "log4j:log4j"},
		{"schedule", "0 0 2 * * SUN"},
		{"connection", "oracle"},
		{"cache", "Redis"},
		{"container", "openjdk:8-jre"},
		{"environment", "prod"},
		{"stored-procedure", "CLOSE_MONTH"},
		{"legacy-program", "BILLRUN"}, {"legacy-call", "RISKCHK"}, {"copybook-use", "CUSTREC"},
		{"batch-job", "NIGHTLY"}, {"job-step", "BILLRUN"},
		{"module-tests", "order"},
		{"dead-code-candidate", "LegacyHelper"},
		{"assessment", "customer"},
		{"extraction-order", "→"},
	}
	for _, w := range want {
		if !has(w.kind, w.text) {
			t.Errorf("missing %s finding containing %q", w.kind, w.text)
		}
	}
	// secrets: never quoted anywhere in the output
	b, _ := json.Marshal(snap)
	if strings.Contains(string(b), "Pa55word") {
		t.Error("a secret value leaked into the result")
	}
	// everything the verifier accepted must cite real lines
	for _, f := range snap.Findings {
		if f.Status == StatusProposed {
			t.Errorf("finding %s was never verified", f.ID)
		}
		for _, c := range f.Citations {
			if p := run.CheckCitation(c); p != "" && f.Status == StatusAccepted {
				t.Errorf("accepted finding %s has a bad citation: %s", f.ID, p)
			}
		}
	}
	// model steps are reported as skipped, not silently missing
	if snap.Run.Steps[4].Status != "skipped" {
		t.Errorf("step 5 should be skipped without a model, got %s", snap.Run.Steps[4].Status)
	}
	if snap.Coverage.Analysed == 0 || snap.Coverage.Files == 0 {
		t.Error("coverage not recorded")
	}
	if !strings.Contains(snap.Report, "# Code analysis report") || !strings.Contains(snap.Report, "Where to start") {
		t.Error("report incomplete")
	}
	if len(run.Reading) == 0 || !strings.Contains(run.Reading[0].File, "OrderService") && !strings.Contains(run.Reading[0].File, "BILLRUN") {
		t.Errorf("reading list should start with rule-heavy code, got %+v", run.Reading)
	}
}

// TestVerifierRejectsBadCitations checks that made-up evidence does not pass.
func TestVerifierRejectsBadCitations(t *testing.T) {
	code := agents.LoadFiles("x", map[string][]byte{"src/a/A.java": []byte("package a;\nclass A {\n  int limit = 5;\n}\n")})
	run, _ := agents.NewRun(code, agents.Options{}, nil, "")
	ok := run.Store.Add(Finding{Agent: "rul", Kind: "business-rule", Title: "Limit is five", Citations: []Citation{{File: "src/a/A.java", Line: 3, Quote: "int limit = 5;"}}, Basis: BasisCode})
	wrongQuote := run.Store.Add(Finding{Agent: "rul", Kind: "business-rule", Title: "Limit is ten", Citations: []Citation{{File: "src/a/A.java", Line: 3, Quote: "int limit = 10;"}}, Basis: BasisCode})
	noFile := run.Store.Add(Finding{Agent: "rul", Kind: "business-rule", Title: "Ghost", Citations: []Citation{{File: "nope.java", Line: 1}}, Basis: BasisCode})
	noEvidence := run.Store.Add(Finding{Agent: "dom", Kind: "journey", Title: "Made up", Evidence: []string{"int-99"}, Basis: BasisModel})
	agents.RunVerifier(context.Background(), run)
	for id, want := range map[string]string{ok: StatusAccepted, wrongQuote: StatusRejected, noFile: StatusRejected, noEvidence: StatusRejected} {
		if f, _ := run.Store.Get(id); f.Status != want {
			t.Errorf("%s: got %s, want %s (%s)", id, f.Status, want, f.Verdict)
		}
	}
}
