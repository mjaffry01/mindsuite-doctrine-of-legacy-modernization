package documents

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mjaffry01/mindsuite-doctrine-of-legacy-modernization/backend/internal/agents"
)

// TestDocumentsRun runs the team (without a model) on testdata/sample (made by testdata/make_sample.py):
// Word, PowerPoint, Visio, PDF (with text and scanned), Confluence HTML, Markdown and draw.io (plain and compressed).
func TestDocumentsRun(t *testing.T) {
	code, err := agents.LoadDir("testdata/sample")
	if err != nil {
		t.Fatal(err)
	}
	r, err := agents.NewRun(code, agents.Options{Team: "documents"}, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	r.Execute(context.Background())
	snap := r.Snapshot()
	if snap.Run.Status != "done" {
		t.Fatalf("status %s: %s", snap.Run.Status, snap.Run.Error)
	}
	if len(snap.Agents) != 12 || snap.Agents[1].ID != "drd" {
		t.Errorf("documents team expected, got %d agents", len(snap.Agents))
	}
	byKind := map[string][]agents.Finding{}
	for _, f := range snap.Findings {
		byKind[f.Kind] = append(byKind[f.Kind], f)
		if f.Status == StatusProposed || f.Status == StatusRejected {
			t.Errorf("%s: %s %s — %s", f.Status, f.ID, f.Title, f.Verdict)
		}
	}
	has := func(kind, sub string) {
		t.Helper()
		for _, f := range byKind[kind] {
			if strings.Contains(f.Title+" "+f.Detail, sub) {
				return
			}
		}
		titles := []string{}
		for _, f := range byKind[kind] {
			titles = append(titles, f.Title+" | "+f.Detail)
		}
		t.Errorf("missing %s containing %q; have %v", kind, sub, titles)
	}
	// every format was read
	has("doc-estate", "5 documents")
	has("doc-estate", "3 diagrams")
	has("document", "Settlement specification")              // Word
	has("document", "Word, ")                                // kind
	has("document", "version 2.1")                           // Word version
	has("document", "HarvestLink overview")                  // PowerPoint title
	has("document", "Payment policy")                        // PDF text layer
	has("document", "Operations handbook")                   // Confluence HTML
	has("diagram", "architecture.drawio: 4 shapes, 3 links") // draw.io
	has("diagram", "intake-flow.drawio: 2 shapes, 1 links")  // compressed draw.io
	has("diagram", "system-context.vsdx: 2 shapes, 1 links") // Visio
	// freshness and coverage
	has("old-document", "Settlement specification is dated Mar 2018")
	has("draft-document", "Settlement specification")
	has("topic-coverage", "Topics covered: 4 of 4")
	// rules, glossary, processes, architecture — each with its exact line
	has("rule-statement", "A grower must be paid within 14 days of delivery.")
	has("rule-statement", "Refunds are only allowed within 30 days of the payment.")
	has("doc-term", "Grower")
	has("doc-term", "GRN")
	has("doc-term", "Moisture")
	has("term-conflict", "\"Lot\" is defined differently in Operations handbook and Settlement specification")
	has("doc-process", "Settlement process (4 steps)")
	has("doc-process", "Month-end process (3 steps)")
	has("intended-link", "Grower portal → Settlement service (REST)")
	has("intended-link", "Settlement service → Partner Bank (SFTP)")
	has("intended-link", "HarvestLink → ERP (invoices)")
	has("intended-link", "Delivery → Grading")
	has("documented-tech", "ORACLE")
	has("secret", "notes.properties line 1")
	has("evidence-value", "Settlement specification: use only as a starting point")
	has("validation-plan", "conflicts first")
	for _, f := range byKind["rule-statement"] {
		c := f.Citations[0]
		if c.Quote == "" {
			t.Errorf("rule statement %s has no quote", f.ID)
		}
	}
	q := ""
	for _, m := range snap.Questions {
		q += m.Text
	}
	if !strings.Contains(q, "scanned-contract.pdf") || !strings.Contains(q, "old-manual.doc") {
		t.Errorf("unreadable documents should be asked about; questions: %s", q)
	}
	if len(r.Reading) == 0 || r.Reading[0].From == 0 {
		t.Errorf("the reading list should hold document sections: %+v", r.Reading)
	}
	b, _ := json.Marshal(snap)
	if strings.Contains(string(b), "SharePointPw2026") || strings.Contains(r.Report, "SharePointPw2026") {
		t.Error("the password leaked into the result")
	}
	if snap.Run.Steps[4].Status != "skipped" {
		t.Errorf("step 5 should be skipped without a model, got %s", snap.Run.Steps[4].Status)
	}
}
