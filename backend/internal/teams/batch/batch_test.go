package batch

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mjaffry01/mindsuite-doctrine-of-legacy-modernization/backend/internal/agents"
)

// TestBatchRun runs the team (without a model) on testdata/sample (made by testdata/make_sample.py):
// Control-M JSON, AutoSys JIL, a crontab, schtasks CSV, 30 days of run history and one script.
func TestBatchRun(t *testing.T) {
	code, err := agents.LoadDir("testdata/sample")
	if err != nil {
		t.Fatal(err)
	}
	r, err := agents.NewRun(code, agents.Options{Team: "batch"}, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	r.Execute(context.Background())
	snap := r.Snapshot()
	if snap.Run.Status != "done" {
		t.Fatalf("status %s: %s", snap.Run.Status, snap.Run.Error)
	}
	if len(snap.Agents) != 10 || snap.Agents[1].ID != "bjr" {
		t.Errorf("batch team expected, got %d agents", len(snap.Agents))
	}
	byKind := map[string][]agents.Finding{}
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
			titles = append(titles, f.Title+" | "+f.Detail)
		}
		t.Errorf("missing %s containing %q; have %v", kind, sub, titles)
	}
	has("batch-estate", "14 jobs on 4 host(s) in AutoSys, Control-M, cron, Windows Task Scheduler")
	has("job", "CALC_SETTLEMENT (HL_NIGHTLY)")
	has("job", "after EXTRACT_DELIVERIES")
	has("job", "HL_DAILY_REPORT (HL_REPORTS)")
	has("job", "batch01:poll_files.sh")
	has("job-stream", "EXTRACT_DELIVERIES → CALC_SETTLEMENT")
	has("job-stream", "4 jobs, 3 steps deep")
	has("job-stream", "HL_LOAD → HL_DAILY_REPORT")
	has("missing-predecessor", "event GL_FEED-TO-CLOSE (needed by MONTH_END_CLOSE)")
	has("batch-window", "23:00–06:00")
	has("daytime-batch", "HL_DAILY_REPORT (10:00)")
	has("frequent-jobs", "poll_files.sh")
	has("month-end-jobs", "MONTH_END_CLOSE")
	has("month-end-jobs", "month_end.sh")
	has("run-summary", "185 runs")
	has("failing-job", "CALC_SETTLEMENT failed 5 of 35 runs, rerun 5 time(s)")
	has("long-job", "It grew from about")
	has("never-ran", "MONTH_END_CLOSE")
	has("admin-job", "SEND_PAYMENTS (root)")
	has("admin-job", "Archive Scans (SYSTEM)")
	has("disabled-jobs", "HL_OLD_EXPORT")
	has("disabled-jobs", "old_cleanup.sh")
	has("secret", "crontab-batch01-appuser.txt line 6")
	has("job-script", "poll_files.sh → scripts/poll_files.sh")
	has("scheduler-target", "Control-M (5 jobs): Keep Control-M")
	has("scheduler-target", "cron (4 jobs): Move")
	has("migration-order", "Move the stream EXTRACT_DELIVERIES")
	has("migration-approach", "stream by stream")
	if len(byKind["secret"]) != 1 {
		t.Errorf("only the sqlplus line holds a password, got %d secret findings", len(byKind["secret"]))
	}
	for _, f := range byKind["never-ran"] {
		if strings.Contains(f.Detail, "poll_files") {
			t.Error("cron jobs have no history in the sample and must not be called never-ran")
		}
	}
	for _, f := range byKind["job"] {
		if strings.Contains(f.Title, "ScheduledDefrag") {
			t.Error("Windows' own tasks must be left out")
		}
	}
	b, _ := json.Marshal(snap)
	if strings.Contains(string(b), "Tiger123") || strings.Contains(r.Report, "Tiger123") {
		t.Error("the database password in the crontab leaked into the result")
	}
	if snap.Run.Steps[4].Status != "skipped" {
		t.Errorf("step 5 should be skipped without a model, got %s", snap.Run.Steps[4].Status)
	}
	if !strings.Contains(r.Report, "# Batch & scheduling analysis report") || !strings.Contains(r.Report, "Failures and reruns") {
		t.Error("report incomplete")
	}
}
