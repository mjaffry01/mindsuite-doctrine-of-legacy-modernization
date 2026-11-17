package incidents

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mjaffry01/mindsuite-doctrine-of-legacy-modernization/backend/internal/agents"
)

// TestIncidentsRun runs the team (without a model) on testdata/sample (made by testdata/make_sample.py):
// ServiceNow incident, change_request and problem exports and a Jira export.
func TestIncidentsRun(t *testing.T) {
	code, err := agents.LoadDir("testdata/sample")
	if err != nil {
		t.Fatal(err)
	}
	r, err := agents.NewRun(code, agents.Options{Team: "incidents"}, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	r.Execute(context.Background())
	snap := r.Snapshot()
	if snap.Run.Status != "done" {
		t.Fatalf("status %s: %s", snap.Run.Status, snap.Run.Error)
	}
	if len(snap.Agents) != 11 || snap.Agents[1].ID != "ird" {
		t.Errorf("incidents team expected, got %d agents", len(snap.Agents))
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
	has("ticket-estate", "42 incidents, 30 changes, 2 problem records over 12 months")
	has("hotspot", "HarvestLink Settlement")
	has("pareto", "components cause")
	has("peak-period", "Month-end")
	has("recurring", "Settlement batch failed at month end run {n}\" (8 times)")
	has("recurring", "Label printer not printing at intake scale\" (5 times)")
	has("recurring", "Problem record PRB0001001")
	has("recurring", "Grower portal login very slow\" (4 times)") // Jira
	has("no-problem-record", "Settlement batch failed")
	has("reopened", "1 incidents were reopened")
	has("problem-records", "2 problem records, 1 still open")
	has("restore-time", "P1 incidents")
	has("stale-incident", "1 incidents open for more than 30 days")
	has("change-frequency", "30 changes")
	has("change-frequency", "emergency 6")
	has("emergency-changes", "20% of changes are emergency changes")
	has("change-failure-rate", "Change failure rate 23%: 7 of 30")
	has("change-caused-incident", "CHG0030010")
	has("change-profile", "23% fail")
	has("modernization-priority", "1. HarvestLink Settlement")
	has("freeze-period", "month-end")
	has("secret", "incident.csv line")
	// people's names and the e-mail address are gone from every title, and nothing is quoted from ticket rows
	b, _ := json.Marshal(snap)
	out := string(b) + r.Report
	for _, leak := range []string{"Jane", "Smith", "Okafor", "Fischer", "jane.smith@", "Harvest2025x", "customer details"} {
		if strings.Contains(out, leak) {
			t.Errorf("%q leaked into the result", leak)
		}
	}
	for _, f := range snap.Findings {
		for _, c := range f.Citations {
			if c.Quote != "" {
				t.Errorf("%s quotes a ticket row: %q", f.ID, c.Quote)
			}
		}
	}
	if len(r.Reading) < 3 {
		t.Errorf("the recurring problems should be on the reading list: %+v", r.Reading)
	}
	if testing.Verbose() {
		t.Log(r.Report)
	}
	if snap.Run.Steps[4].Status != "skipped" {
		t.Errorf("step 5 should be skipped without a model, got %s", snap.Run.Steps[4].Status)
	}
}
