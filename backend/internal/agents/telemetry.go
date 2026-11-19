package agents

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"
)

type AgentAttempt struct {
	ID            string     `json:"id"`
	Name          string     `json:"name,omitempty"`
	Agent         string     `json:"agent"`
	Part          string     `json:"part,omitempty"`
	Step          int        `json:"step"`
	Batch         int        `json:"batch,omitempty"`
	Reused        bool       `json:"reused,omitempty"`
	Status        string     `json:"status"`
	Started       time.Time  `json:"started"`
	Ended         *time.Time `json:"ended,omitempty"`
	DurationMS    int64      `json:"durationMs"`
	ModelRequests int        `json:"modelRequests"`
}

func (r *Run) recordAttempt(a AgentAttempt) {
	r.mu.Lock()
	if r.attempts == nil {
		r.attempts = map[string]AgentAttempt{}
	}
	r.attempts[a.ID] = a
	if r.Dir != "" {
		if f, err := os.OpenFile(filepath.Join(r.Dir, "telemetry.jsonl"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600); err == nil {
			_ = json.NewEncoder(f).Encode(a)
			_ = f.Close()
		}
	}
	sink := r.attemptSink
	r.mu.Unlock()
	if sink != nil {
		sink(a)
	}
}

func (r *Run) beginAgent(agent, part string, step int) func(string) {
	now := time.Now()
	r.mu.Lock()
	r.attemptNumber++
	n := r.attemptNumber
	r.mu.Unlock()
	a := AgentAttempt{ID: r.ID + "-" + strconv.Itoa(n), Agent: agent, Name: r.agentName(agent), Part: part, Step: step, Status: "running", Started: now.UTC()}
	r.recordAttempt(a)
	return func(status string) {
		ended := time.Now().UTC()
		r.mu.Lock()
		a = r.attempts[a.ID]
		if status == "done" && r.attemptProblems[a.ID] {
			status = "partial"
		}
		delete(r.attemptProblems, a.ID)
		r.mu.Unlock()
		a.Ended = &ended
		a.DurationMS = time.Since(now).Milliseconds()
		a.Status = status
		r.recordAttempt(a)
	}
}

func (r *Run) MeasureAgent(agent string, fn func()) {
	finish := r.beginAgent(agent, "", r.currentStep())
	status := "failed"
	defer func() { finish(status) }()
	fn()
	status = "done"
}

func MeasureInput[T any](r *Run, agent string, parse func(*Run) T, record func(T)) T {
	var input T
	r.MeasureAgent(agent, func() { input = parse(r); record(input) })
	return input
}

func (r *Run) flagAttemptProblem(agent string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.attemptProblems == nil {
		r.attemptProblems = map[string]bool{}
	}
	for id, a := range r.attempts {
		if a.Agent == agent && a.Status == "running" {
			r.attemptProblems[id] = true
		}
	}
}

func sortAttempts(attempts map[string]AgentAttempt) []AgentAttempt {
	out := make([]AgentAttempt, 0, len(attempts))
	for _, attempt := range attempts {
		out = append(out, attempt)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Started.Equal(out[j].Started) {
			return out[i].ID < out[j].ID
		}
		return out[i].Started.Before(out[j].Started)
	})
	return out
}

func (r *Run) AgentTelemetry() (string, []AgentAttempt) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.Status, sortAttempts(r.attempts)
}

// ReadAgentTelemetry recovers the last recorded state of each attempt after a restart.
func ReadAgentTelemetry(dir string) ([]AgentAttempt, bool) {
	f, err := os.Open(filepath.Join(dir, "telemetry.jsonl"))
	if err != nil {
		return nil, false
	}
	defer f.Close()
	attempts := map[string]AgentAttempt{}
	decoder := json.NewDecoder(f)
	for {
		var a AgentAttempt
		if decoder.Decode(&a) != nil {
			break
		}
		attempts[a.ID] = a
	}
	for id, a := range attempts {
		if a.Status == "running" {
			a.Status = "interrupted"
			attempts[id] = a
		}
	}
	return sortAttempts(attempts), true
}
