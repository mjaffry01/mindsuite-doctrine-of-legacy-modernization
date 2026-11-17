package database

import (
	"context"
	"encoding/json"
	"github.com/mjaffry01/mindsuite-doctrine-of-legacy-modernization/backend/internal/agents"
	"strings"
	"testing"
)

func TestDatabaseRun(t *testing.T) {
	code, err := agents.LoadDir("testdata/sample")
	if err != nil {
		t.Fatal(err)
	}
	r, err := agents.NewRun(code, agents.Options{Team: "database"}, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	r.Execute(context.Background())
	snap := r.Snapshot()
	if snap.Run.Status != "done" {
		t.Fatalf("status %s: %s", snap.Run.Status, snap.Run.Error)
	}
	if len(snap.Agents) != 14 || snap.Agents[1].ID != "cat" {
		t.Errorf("database team expected, got %d agents", len(snap.Agents))
	}
	byKind := map[string][]Finding{}
	for _, f := range snap.Findings {
		byKind[f.Kind] = append(byKind[f.Kind], f)
		if f.Status == StatusProposed {
			t.Errorf("not verified: %s %s", f.ID, f.Title)
		}
		if f.Status == StatusRejected {
			t.Errorf("rejected: %s %s — %s", f.ID, f.Title, f.Verdict)
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
			titles = append(titles, f.Title)
		}
		t.Errorf("missing %s containing %q; have %v", kind, sub, titles)
	}
	has("db-engine", "Oracle 12c")
	has("db-version", "out of Premier Support")
	has("table", "HARVEST.DELIVERIES")
	has("table", "refers to GROWER_ID → HARVEST.GROWERS")
	has("constraint-rule", "STATUS IN ('NEW','TESTED','PAID','REJECTED')")
	has("constraint-rule", "HECTARES > 0")
	has("cross-schema-fk", "SETTLE.PAYMENTS refers to HARVEST.DELIVERIES")
	has("data-cluster", "HARVEST.DELIVERIES")
	has("db-program", "package body SETTLE.SETTLEMENT_PKG")
	has("db-program", "writes HARVEST.DELIVERIES, SETTLE.PAYMENTS")
	has("db-program", "calls SETTLE.LOG_AUDIT")
	has("trigger", "on HARVEST.DELIVERIES")
	has("shared-write", "SETTLE.PAYMENTS is written by 2 programs")
	has("dynamic-sql", "SETTLE.SETTLEMENT_PKG")
	has("external-call", "UTL_HTTP")
	has("db-job", "runs SETTLE.SETTLEMENT_PKG")
	has("db-link", "BANK_LINK")
	has("admin-account", "HARVEST_APP has DBA role")
	has("public-grant", "HARVEST.GROWERS")
	has("sensitive-column", "bank account: BANK_IBAN")
	has("password-column", "SETTLE.APP_USERS.PASSWORD")
	has("secret", "jobs_and_security.sql line 10")
	has("vendor-feature", "CONNECT BY")
	has("vendor-feature", "Autonomous transactions")
	has("vendor-type", "LONG")
	has("no-primary-key", "SETTLE.AUDIT_LOG")
	has("unindexed-fk", "HARVEST.QUALITY_TESTS(DELIVERY_ID)")
	has("generic-columns", "HARVEST.OLD_IMPORT")
	has("db-size", "GB")
	has("large-table", "SETTLE.AUDIT_LOG")
	has("empty-tables", "HARVEST.OLD_IMPORT")
	has("migration-approach", "")
	has("migration-order", "")
	if len(byKind["db-job"]) != 1 {
		t.Errorf("the job is in the DDL and the catalog: want 1 finding, got %d", len(byKind["db-job"]))
	}
	if len(byKind["db-assessment"]) == 0 {
		t.Error("no migration assessment")
	}
	b, _ := json.Marshal(snap)
	if strings.Contains(string(b), "Secr3tPw") || strings.Contains(r.Report, "Secr3tPw") {
		t.Error("the link password leaked into the result")
	}
	if strings.Contains(string(b), "'WHEAT', 'A', 210") {
		t.Error("a data row leaked into the result")
	}
	if snap.Run.Steps[4].Status != "skipped" {
		t.Errorf("step 5 should be skipped without a model, got %s", snap.Run.Steps[4].Status)
	}
	if len(r.Reading) == 0 || r.Reading[0].From == 0 {
		t.Errorf("reading list should hold program line ranges: %+v", r.Reading)
	}
	q := ""
	for _, m := range snap.Questions {
		q += m.Text + "\n"
	}
	if !strings.Contains(q, "data rows (1 INSERT") {
		t.Errorf("the data row in the export should raise a question; questions:\n%s", q)
	}
	if !strings.Contains(r.Report, "# Database analysis report") || !strings.Contains(r.Report, "Migration assessment per domain") {
		t.Error("report incomplete")
	}
}
