package agents

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// guardTeam has three fact agents: one works, one crashes on its first try, one hangs.
type guardTeam struct {
	tries   *atomic.Int32
	release chan struct{}
}

func (guardTeam) ID() string { return "guard-test" }
func (guardTeam) Agents() []AgentInfo {
	return []AgentInfo{{ID: "ok", Name: "Steady agent"}, {ID: "boom", Name: "Crashing agent"}, {ID: "slow", Name: "Stuck agent"}}
}
func (guardTeam) StepTitles() []string         { return []string{"Collect", "Verify", "Report"} }
func (guardTeam) WholeKinds() map[string]bool  { return map[string]bool{"fact": true} }
func (guardTeam) NewState() any                { return nil }
func (guardTeam) Source(r *Run) map[string]any { return nil }
func (t guardTeam) Execute(ctx context.Context, r *Run) {
	r.Step(1, "running", "")
	r.RunParts(ctx, []Part{
		{ID: "ok", Fn: func(r *Run) { r.Store.Add(Finding{Agent: "ok", Kind: "fact", Title: "steady fact"}) }},
		{ID: "boom", Fn: func(r *Run) {
			if t.tries.Add(1) == 1 {
				var m map[string]int
				m["x"] = 1 // crashes: assignment to a nil map
			}
			r.Store.Add(Finding{Agent: "boom", Kind: "fact", Title: "fact found on the second try"})
		}},
		{ID: "slow", Fn: func(r *Run) { <-t.release }},
	})
	r.Step(1, "done", "collected")
	r.Step(2, "running", "")
	RunVerifier(ctx, r)
	r.Step(2, "done", "")
	r.Step(3, "running", "")
	r.WriteReport(ctx, func(ctx context.Context, r *Run) string {
		return "# Guard report\n\nFacts: " + titles(r.Store.Findings()) + "\n"
	})
	r.Step(3, "done", "")
}

func titles(fs []Finding) string {
	out := []string{}
	for _, f := range fs {
		out = append(out, f.Title+" ("+f.Status+")")
	}
	return strings.Join(out, ", ")
}

func TestFailingAgentIsSkippedAndReported(t *testing.T) {
	old := AgentTimeLimit
	AgentTimeLimit = 200 * time.Millisecond
	defer func() { AgentTimeLimit = old }()
	team := guardTeam{tries: &atomic.Int32{}, release: make(chan struct{})}
	defer close(team.release)
	Register(team)

	dir := t.TempDir()
	r, err := NewRun(LoadFiles("x", map[string][]byte{"a.txt": []byte("a\n")}), Options{Team: "guard-test"}, nil, dir)
	if err != nil {
		t.Fatal(err)
	}
	r.Execute(context.Background())

	snap := r.Snapshot()
	wantAttempts := map[string]string{"boom": "failed", "slow": "timed out", "ok": "done", "ver": "done", "rep": "done"}
	for _, a := range snap.Telemetry {
		if want, ok := wantAttempts[a.Agent]; ok {
			if a.Status != want || a.Ended == nil || a.Ended.Before(a.Started) {
				t.Fatalf("incorrect agent timing: %+v", a)
			}
			delete(wantAttempts, a.Agent)
		}
	}
	if len(wantAttempts) > 0 {
		t.Fatalf("missing agent timings: %+v", wantAttempts)
	}

	if snap.Run.Status != "done" {
		t.Fatalf("the run should carry on and finish, got %q (%s)", snap.Run.Status, snap.Run.Error)
	}
	if len(snap.Run.Problems) != 2 {
		t.Fatalf("want 2 problems, got %+v", snap.Run.Problems)
	}
	byPart := r.FailedParts()
	if p := byPart["boom"]; p.Name != "Crashing agent" || p.Step != 1 || !strings.Contains(p.Error, "nil map") || !p.CanRerun {
		t.Fatalf("crash not recorded well: %+v", p)
	}
	if p := byPart["slow"]; !p.TimedOut || !strings.Contains(p.Error, "did not finish within") {
		t.Fatalf("hang not recorded as a time-out: %+v", p)
	}
	if s := snap.Run.Steps[0]; s.Status != "partial" || !strings.Contains(s.Note, "did not finish: ") {
		t.Fatalf("step 1 should be partial, got %+v", s)
	}
	if !strings.Contains(r.Report, "steady fact") {
		t.Fatalf("the steady agent's work is lost:\n%s", r.Report)
	}
	if !strings.HasPrefix(r.Report, "# Guard report\n\n> **Warning: this report is incomplete.** 2 agents did not finish") ||
		!strings.Contains(r.Report, "**Crashing agent** (step 1)") || !strings.Contains(r.Report, "**Stuck agent** (step 1)") {
		t.Fatalf("report warning missing:\n%s", r.Report)
	}
	saved, _ := os.ReadFile(filepath.Join(dir, r.ID, "result.json"))
	if !strings.Contains(string(saved), `"problems"`) {
		t.Fatal("problems are not saved with the result")
	}

	// Run the crashed agent again: it works the second time, its finding is verified and the report rewritten.
	if err := r.Rerun(context.Background(), "nope"); err != ErrNoSuchPart {
		t.Fatalf("unknown part: %v", err)
	}
	if err := r.Rerun(context.Background(), "boom"); err != nil {
		t.Fatal(err)
	}
	if err := r.Rerun(context.Background(), "slow"); err != ErrRunBusy {
		t.Fatalf("a second re-run while one is going should wait: %v", err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for r.Snapshot().Run.Status == "running" {
		if time.Now().After(deadline) {
			t.Fatal("re-run did not finish")
		}
		time.Sleep(10 * time.Millisecond)
	}
	snap = r.Snapshot()
	if len(snap.Run.Problems) != 1 || snap.Run.Problems[0].Part != "slow" {
		t.Fatalf("only the stuck agent should be left: %+v", snap.Run.Problems)
	}
	if !strings.Contains(r.Report, "fact found on the second try (accepted)") || !strings.Contains(r.Report, "One agent did not finish") {
		t.Fatalf("report not rewritten after the re-run:\n%s", r.Report)
	}
	if snap.Run.Steps[0].Status != "partial" {
		t.Fatal("step 1 is still partial while the stuck agent is missing")
	}
	msgs, _ := os.ReadFile(filepath.Join(dir, r.ID, "messages.jsonl"))
	if !strings.Contains(string(msgs), "Crashing agent finished on the second try") {
		t.Fatal("the re-run is not in the saved message log")
	}
}

func TestStepDoneWithoutProblems(t *testing.T) {
	team := guardTeam{tries: &atomic.Int32{}, release: make(chan struct{})}
	Register(team)
	r, _ := NewRun(LoadFiles("x", map[string][]byte{"a.txt": []byte("a\n")}), Options{Team: "guard-test"}, nil, "")
	r.Step(2, "running", "")
	r.Step(2, "done", "fine")
	if r.Steps[1].Status != "done" || r.Steps[1].Note != "fine" {
		t.Fatalf("%+v", r.Steps[1])
	}
	if withProblemsNote("# T\n\nbody", "") != "# T\n\nbody" {
		t.Fatal("report changed without problems")
	}
}
