package agents

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// When one agent fails, the run carries on without it and says so.
//
// Agents that only collect facts run side by side in RunParts. If one of them crashes, or does not finish within
// AgentTimeLimit, it is recorded as a Problem and the others carry on: their facts do not depend on it.
// The step is then marked "partial", the run lists the problem, and the report opens with a warning.
// Steps that every later step depends on (reading the evidence, the verifier, the report) still stop the run.
//
// A failed part can be run again on its own with Rerun, while the run's evidence is still loaded.

// AgentTimeLimit is how long one fact agent may take before the run carries on without it.
var AgentTimeLimit = 10 * time.Minute

// Part is one agent's piece of work in a step. Key tells two parts of the same agent apart (it defaults to ID).
type Part struct {
	ID   string
	Key  string
	Name string // shown when it fails; defaults to the agent's name
	Fn   func(*Run)
}

func (p Part) key() string {
	if p.Key != "" {
		return p.Key
	}
	return p.ID
}

// Problem is an agent that failed or did not finish.
type Problem struct {
	Agent    string    `json:"agent"`
	Name     string    `json:"name"`
	Part     string    `json:"part"`
	Step     int       `json:"step"`
	Error    string    `json:"error"`
	TimedOut bool      `json:"timedOut,omitempty"`
	CanRerun bool      `json:"canRerun,omitempty"`
	Time     time.Time `json:"time"`
}

// RunParts runs the parts side by side in the current step. A part that crashes or overruns is recorded and skipped.
func (r *Run) RunParts(ctx context.Context, parts []Part) {
	var wg sync.WaitGroup
	for _, p := range parts {
		r.mu.Lock()
		if r.parts == nil {
			r.parts = map[string]Part{}
		}
		r.parts[p.key()] = p
		r.mu.Unlock()
		wg.Add(1)
		go func(p Part) {
			defer wg.Done()
			r.runPart(ctx, r.currentStep(), p)
		}(p)
	}
	wg.Wait()
}

// RunPart runs one part in the current step, recording a failure instead of stopping the run. It reports success.
func (r *Run) RunPart(ctx context.Context, p Part) bool {
	r.mu.Lock()
	if r.parts == nil {
		r.parts = map[string]Part{}
	}
	r.parts[p.key()] = p
	r.mu.Unlock()
	return r.runPart(ctx, r.currentStep(), p)
}

// MINDSUITE_TEST_FAIL_AGENT names part keys (comma separated) that crash on their first try, to test failure handling.
var testFail sync.Map

func (r *Run) runPart(ctx context.Context, step int, p Part) bool {
	finish := r.beginAgent(p.ID, p.key(), step)
	done := make(chan string, 1)
	go func() {
		defer func() {
			if x := recover(); x != nil {
				done <- fmt.Sprint(x)
			}
		}()
		for _, k := range strings.Split(os.Getenv("MINDSUITE_TEST_FAIL_AGENT"), ",") {
			if k = strings.TrimSpace(k); k == p.key() {
				if _, tried := testFail.LoadOrStore(r.ID+"/"+k, true); !tried {
					panic("test failure requested by MINDSUITE_TEST_FAIL_AGENT")
				}
			}
		}
		p.Fn(r)
		done <- ""
	}()
	timer := time.NewTimer(AgentTimeLimit)
	defer timer.Stop()
	select {
	case msg := <-done:
		if msg == "" {
			finish("done")
			return true
		}
		finish("failed")
		r.addProblem(Problem{Agent: p.ID, Name: p.Name, Part: p.key(), Step: step, Error: msg, CanRerun: true})
	case <-timer.C:
		finish("timed out")
		r.addProblem(Problem{Agent: p.ID, Name: p.Name, Part: p.key(), Step: step, TimedOut: true, CanRerun: true,
			Error: fmt.Sprintf("did not finish within %s", AgentTimeLimit)})
	case <-ctx.Done():
		finish("interrupted")
		r.addProblem(Problem{Agent: p.ID, Name: p.Name, Part: p.key(), Step: step, Error: "stopped: " + ctx.Err().Error(), CanRerun: true})
	}
	return false
}

// AgentProblem records that an agent could not do its work (for example a Claude request that failed).
// It cannot be run again on its own: run the team again.
func (r *Run) AgentProblem(agent, msg string) {
	r.flagAttemptProblem(agent)
	r.addProblem(Problem{Agent: agent, Part: agent, Step: r.currentStep(), Error: msg})
}

func (r *Run) addProblem(p Problem) {
	if p.Name == "" {
		p.Name = r.agentName(p.Agent)
	}
	p.Time = time.Now().UTC()
	r.mu.Lock()
	r.Problems = append(r.Problems, p)
	r.mu.Unlock()
	what := "stopped with an error: " + p.Error
	if p.TimedOut {
		what = p.Error
	}
	r.Store.Log(p.Agent, fmt.Sprintf("%s %s. The run carries on without it; its findings are missing.", p.Name, what))
}

func (r *Run) agentName(id string) string {
	for _, a := range r.team.Agents() {
		if a.ID == id {
			return a.Name
		}
	}
	return id
}

func (r *Run) currentStep() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.current
}

// FailedParts returns the open problems by part key.
func (r *Run) FailedParts() map[string]Problem {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := map[string]Problem{}
	for _, p := range r.Problems {
		out[p.Part] = p
	}
	return out
}

// WriteReport writes the run's report with fn and keeps fn, so the report can be rewritten after a re-run.
func (r *Run) WriteReport(ctx context.Context, fn func(context.Context, *Run) string) {
	r.mu.Lock()
	r.reporter = fn
	r.mu.Unlock()
	r.MeasureAgent("rep", func() { r.Report = fn(ctx, r) })
}

// problemsNote is the warning at the top of a report when agents failed.
func (r *Run) problemsNote() string {
	r.mu.Lock()
	ps := append([]Problem(nil), r.Problems...)
	r.mu.Unlock()
	if len(ps) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("> **Warning: this report is incomplete.** ")
	if len(ps) == 1 {
		b.WriteString("One agent did not finish, so its findings are missing:\n")
	} else {
		fmt.Fprintf(&b, "%d agents did not finish, so their findings are missing:\n", len(ps))
	}
	for _, p := range ps {
		fmt.Fprintf(&b, "> - **%s** (step %d): %s\n", p.Name, p.Step, p.Error)
	}
	b.WriteString("> \n> Run the agent again from MindSuite, or run the team again.\n\n")
	return b.String()
}

// withProblemsNote puts the warning right under the report's title.
func withProblemsNote(report, note string) string {
	if note == "" {
		return report
	}
	if i := strings.Index(report, "\n\n"); strings.HasPrefix(report, "# ") && i > 0 {
		return report[:i+2] + note + report[i+2:]
	}
	return note + report
}

// Rerun errors.
var (
	ErrRunBusy     = errors.New("the run is still working")
	ErrNoSuchPart  = errors.New("this agent has no failure to run again")
	ErrCannotRerun = errors.New("this agent cannot be run again on its own: run the team again")
)

// Rerun runs one failed part again, verifies its new findings and rewrites the report.
// Steps that used its facts earlier (the reading list, the assessment) are not redone.
func (r *Run) Rerun(ctx context.Context, part string) error {
	r.mu.Lock()
	if r.Status == "running" || r.Finished.IsZero() {
		r.mu.Unlock()
		return ErrRunBusy
	}
	idx := -1
	for i, p := range r.Problems {
		if p.Part == part {
			idx = i
		}
	}
	if idx < 0 {
		r.mu.Unlock()
		return ErrNoSuchPart
	}
	pr := r.Problems[idx]
	p, ok := r.parts[part]
	if !ok || !pr.CanRerun {
		r.mu.Unlock()
		return ErrCannotRerun
	}
	r.Status, r.Finished = "running", time.Time{}
	r.mu.Unlock()

	r.Store.reopen()
	r.Store.Log("orch", fmt.Sprintf("Running %s again.", pr.Name))
	before := map[string]bool{}
	for _, f := range r.Store.Findings() {
		before[f.ID] = true
	}
	go func() {
		defer func() {
			if x := recover(); x != nil {
				r.Store.Log("orch", fmt.Sprint("Re-run failed: ", x))
			}
			r.mu.Lock()
			r.Status, r.Finished = "done", time.Now().UTC()
			r.mu.Unlock()
			r.save()
			r.Store.Close()
		}()
		r.mu.Lock()
		r.current = pr.Step
		r.mu.Unlock()
		if !r.runPart(ctx, pr.Step, p) {
			// runPart recorded the new failure; drop the earlier one.
			r.mu.Lock()
			r.Problems = append(r.Problems[:idx], r.Problems[idx+1:]...)
			r.mu.Unlock()
			r.refreshReport(ctx)
			return
		}
		r.mu.Lock()
		r.Problems = append(r.Problems[:idx], r.Problems[idx+1:]...)
		still := false
		for _, q := range r.Problems {
			still = still || q.Step == pr.Step
		}
		if !still && r.Steps[pr.Step-1].Status == "partial" {
			r.Steps[pr.Step-1].Status = "done"
		}
		r.mu.Unlock()
		var added []Finding
		for _, f := range r.Store.Findings() {
			if !before[f.ID] {
				added = append(added, f)
			}
		}
		r.MeasureAgent("ver", func() { verifyFindings(ctx, r, added) })
		r.Store.Log("orch", fmt.Sprintf("%s finished on the second try: %d new findings, verified. Steps that used its facts earlier were not redone.", pr.Name, len(added)))
		r.refreshReport(ctx)
	}()
	return nil
}

func (r *Run) refreshReport(ctx context.Context) {
	r.mu.Lock()
	fn := r.reporter
	r.mu.Unlock()
	if fn != nil {
		var report string
		r.MeasureAgent("rep", func() { report = withProblemsNote(fn(ctx, r), r.problemsNote()) })
		r.mu.Lock()
		r.Report = report
		r.mu.Unlock()
	}
}
