package database

import (
	"context"
	"encoding/json"
	"github.com/mjaffry01/mindsuite-doctrine-of-legacy-modernization/backend/internal/agents"
	"os"
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
	if snap.Run.Steps[4].Status != "done" || len(snap.Sent) != 0 {
		t.Errorf("want local business operations without model calls: %+v", snap.Run.Steps[4])
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
	_, operating, found := strings.Cut(r.Report, "## How the business operates")
	if !found {
		t.Fatal("missing business report")
	}
	operating, _, _ = strings.Cut(operating, "## Migration approach")
	for _, want := range []string{
		"Scheduled job NIGHTLY_SETTLEMENT is configured to call this routine",
		"Declared schedule: FREQ=DAILY;BYHOUR=2",
		"The routine calls SETTLE.SETTLEMENT_PKG.CALC_PAYMENT",
		"This branch checks whether grade equals A and moist is below 14",
		"This path multiplies price by 1.10",
		"targeting HARVEST.DELIVERIES. It sets status to REJECTED",
		"INSERT statement targeting SETTLE.PAYMENTS",
		"weight kg is above 40000",
		"programs.sql:",
		"What to confirm",
	} {
		if !strings.Contains(operating, want) {
			t.Errorf("missing operational detail %q", want)
		}
	}
	if strings.Contains(operating, "Secr3tPw") || strings.Contains(operating, "WHEAT") {
		t.Fatal("business section contains database data or a credential")
	}

}

func TestDatabaseBusinessFlowAcrossBatches(t *testing.T) {
	files := map[string][]byte{}
	for _, name := range []string{"programs.sql", "jobs_and_security.sql"} {
		data, err := os.ReadFile("testdata/sample/" + name)
		if err != nil {
			t.Fatal(err)
		}
		files[name] = append(data, []byte(strings.Repeat("-- batch fixture\n", 2800))...)
	}
	input := agents.LoadFiles("settlement", files)
	run, err := agents.NewRun(input, agents.Options{Team: "database", Interview: &agents.InterviewFocus{
		Problem: "Grade rejection and payment accuracy",
	}}, nil, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	run.Execute(context.Background())
	snap := run.Snapshot()
	if snap.Run.Batches == nil || snap.Run.Batches.Completed != 2 || !strings.Contains(snap.Report, "Scheduled job NIGHTLY_SETTLEMENT is configured to call this routine") {
		t.Fatalf("expected cross-batch job and routine link, report state %+v", snap.Run.Batches)
	}
}

func TestSQLServerRoutineEndsBeforeDataRows(t *testing.T) {
	sql := []byte(`CREATE PROCEDURE dbo.POST_RECEIPT AS
BEGIN
  IF @quantity <= 0
    RETURN;
  INSERT INTO dbo.STOCK_LEDGER (QTY) VALUES (@quantity);
END;
GO
INSERT INTO dbo.OUTSIDE_DATA VALUES ('fixture row');
`)
	r, err := agents.NewRun(agents.LoadFiles("sqlserver", map[string][]byte{"procedure.sql": sql}), agents.Options{Team: "database"}, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	r.Execute(context.Background())
	_, operating, found := strings.Cut(r.Report, "## How the business operates")
	if !found {
		t.Fatal("missing business section")
	}
	operating, _, _ = strings.Cut(operating, "## Migration approach")
	if !strings.Contains(operating, "quantity is at or below 0") || !strings.Contains(operating, "DBO.STOCK_LEDGER") {
		t.Fatalf("missing procedure evidence: %s", operating)
	}
	if strings.Contains(operating, "fixture row") || strings.Contains(operating, "OUTSIDE_DATA") {
		t.Fatal("a statement outside the stored routine was treated as part of the business workflow")
	}
}
