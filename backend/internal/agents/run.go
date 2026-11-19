package agents

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// InterviewFocus guides local code review prioritization. It is not saved in result JSON or sent to the model.
type InterviewFocus struct {
	Reasons    []string
	Problem    string
	Success    string
	System     string
	Objective  string
	OutOfScope string
	Deadline   string
}

// Options chosen by the client for one run.
type Options struct {
	BatchDir  string          `json:"-"`
	Interview *InterviewFocus `json:"-"`

	Workspace string `json:"workspace,omitempty"` // set by the HTTP server
	// AllowModel is the client's consent to send selected code excerpts to Claude.
	// Without it, only the rule-based agents run and the model-based steps are reported as gaps.
	AllowModel bool `json:"allowModel"`
	// MaxModelFiles caps how many files the business-rules agent reads in depth.
	MaxModelFiles int `json:"maxModelFiles"`
	// GitHistory reads `git log` (read-only) when the code is a local folder with .git.
	GitHistory bool `json:"gitHistory"`
	// Team is the id of a registered team: "code" (default), "database", "infrastructure".
	Team string `json:"team"`
}

// Step is the state of one of the nine steps shown to the client.
type Step struct {
	N      int    `json:"n"`
	Title  string `json:"title"`
	Status string `json:"status"` // waiting | running | done | partial (an agent failed) | skipped | people
	Note   string `json:"note,omitempty"`
}

// Run is one analysis of one codebase.
type Run struct {
	ID       string         `json:"id"`
	Status   string         `json:"status"` // running | done | failed
	Error    string         `json:"error,omitempty"`
	Started  time.Time      `json:"started"`
	Finished time.Time      `json:"finished,omitempty"`
	Options  Options        `json:"options"`
	Steps    []Step         `json:"steps"`
	Report   string         `json:"-"`
	Batches  *BatchProgress `json:"batches,omitempty"`
	// Problems lists agents that failed or did not finish; the run carried on without them.
	Problems []Problem `json:"problems,omitempty"`

	Code    *Codebase `json:"-"`
	Store   *Store    `json:"-"`
	LLM     *LLM      `json:"-"`
	Dir     string    `json:"-"`
	Reading []Reading `json:"-"`
	// Secret lines per file: never quoted, never sent to a model.
	Secrets map[string]map[int]bool `json:"-"`
	// SentToModel lists what left this machine, for transparency.
	SentToModel []Sent `json:"-"`

	attempts        map[string]AgentAttempt
	attemptProblems map[string]bool
	attemptNumber   int
	attemptSink     func(AgentAttempt)
	mu              sync.Mutex
	coverage        map[string]map[string]bool // file -> agents that read it
	titles          []string
	team            Team
	// State is the running team's own working state (its parsed model, finding ids…).
	State any `json:"-"`

	current  int             // the step that is running
	parts    map[string]Part // fact parts by key, for re-runs
	reporter func(context.Context, *Run) string
}

// Reading is a file the orchestrator picked for in-depth reading.
type Reading struct {
	File   string `json:"file"`
	Reason string `json:"reason"`
	// Optional line range and object name (database programs inside one export file).
	From int    `json:"from,omitempty"`
	To   int    `json:"to,omitempty"`
	Name string `json:"name,omitempty"`
}

// Sent records one excerpt sent to a model.
type Sent struct {
	Agent string   `json:"agent"`
	Model string   `json:"model"`
	Files []string `json:"files"`
	Chars int      `json:"chars"`
}

// NewRun prepares a run. dir (optional) is where messages and results are saved.
func NewRun(code *Codebase, opts Options, llm *LLM, baseDir string) (*Run, error) {
	b := make([]byte, 6)
	rand.Read(b)
	id := time.Now().UTC().Format("20060102-150405") + "-" + hex.EncodeToString(b)
	dir := ""
	if baseDir != "" {
		dir = filepath.Join(baseDir, id)
	}
	if opts.BatchDir == "" && dir != "" {
		opts.BatchDir = filepath.Join(dir, "checkpoints")
	}
	st, err := NewStore(dir)
	if err != nil {
		return nil, err
	}
	if opts.MaxModelFiles <= 0 {
		opts.MaxModelFiles = 12
	}
	if opts.Team == "" {
		opts.Team = "code"
	}
	team := TeamByID(opts.Team)
	if team == nil {
		return nil, fmt.Errorf("unknown team %q (known: %v)", opts.Team, TeamIDs())
	}
	titles := team.StepTitles()
	r := &Run{ID: id, Status: "running", Started: time.Now().UTC(), Options: opts, Code: code, Store: st, LLM: llm, Dir: dir,
		Secrets: map[string]map[int]bool{}, coverage: map[string]map[string]bool{}, titles: titles, team: team, State: team.NewState()}
	for i, t := range titles {
		r.Steps = append(r.Steps, Step{N: i + 1, Title: t, Status: "waiting"})
	}
	return r, nil
}

// Cover marks a file as read by an agent.
func (r *Run) Cover(agent string, f *File) {
	r.mu.Lock()
	if r.coverage[f.Path] == nil {
		r.coverage[f.Path] = map[string]bool{}
	}
	r.coverage[f.Path][agent] = true
	r.mu.Unlock()
}

// MarkSecret records a line that may hold a credential: it is never quoted or sent to a model.
func (r *Run) MarkSecret(file string, line int) {
	r.mu.Lock()
	if r.Secrets[file] == nil {
		r.Secrets[file] = map[int]bool{}
	}
	r.Secrets[file][line] = true
	r.mu.Unlock()
}

// Step updates one of the 9 steps and logs it.
func (r *Run) Step(n int, status, note string) {
	r.mu.Lock()
	if status == "running" {
		r.current = n
	}
	if status == "done" {
		// A step whose agents did not all finish is only partly done.
		var failed []string
		for _, p := range r.Problems {
			if p.Step == n {
				failed = append(failed, p.Name)
			}
		}
		if len(failed) > 0 {
			status = "partial"
			note = strings.TrimPrefix(note+"; did not finish: "+strings.Join(failed, ", "), "; ")
		}
	}
	r.Steps[n-1].Status = status
	if note != "" {
		r.Steps[n-1].Note = note
	}
	r.mu.Unlock()
	r.Store.Log("orch", fmt.Sprintf("Step %d · %s: %s%s", n, r.titles[n-1], status, map[bool]string{true: " — " + note, false: ""}[note != ""]))
}

// ModelReady reports whether model-based agents may run.
func (r *Run) ModelReady() (bool, string) {
	if !r.Options.AllowModel {
		return false, "the client did not allow sending excerpts to Claude"
	}
	if r.LLM == nil {
		return false, "no Claude API key is configured on the backend (ANTHROPIC_API_KEY)"
	}
	return true, ""
}

// Execute runs the whole team. It is safe to call in a goroutine.
func (r *Run) Execute(ctx context.Context) { r.execute(ctx, true) }

func (r *Run) executeSingle(ctx context.Context) { r.execute(ctx, false) }

func (r *Run) execute(ctx context.Context, allowBatches bool) {
	finish := r.beginAgent("orch", "run", 0)
	status := "failed"
	defer func() {
		if p := recover(); p != nil {
			r.mu.Lock()
			r.Status, r.Error = "failed", fmt.Sprint(p)
			r.mu.Unlock()
			r.Store.Log("orch", fmt.Sprint("Run failed: ", p))
		}
		r.mu.Lock()
		r.Finished = time.Now().UTC()
		r.mu.Unlock()
		finish(status)
		r.save()
		r.Store.Close()
	}()
	var batches []evidenceBatch
	if allowBatches {
		batches = planEvidenceBatches(r.Code, r.Options.Team)
	}
	if len(batches) > 0 {
		r.executeBatches(ctx, batches)
	} else {
		r.team.Execute(ctx, r)
	}
	note := r.problemsNote()
	r.mu.Lock()
	r.Report = withProblemsNote(r.Report, note)
	if r.Status == "running" {
		r.Status = "done"
	}
	status = r.Status
	r.mu.Unlock()
}

// AddSent records what was sent to a model, for transparency.
func (r *Run) AddSent(x Sent) {
	r.mu.Lock()
	x.Model = Resolve(x.Model)
	r.SentToModel = append(r.SentToModel, x)
	for id, a := range r.attempts {
		if a.Agent == x.Agent && a.Status == "running" {
			a.ModelRequests++
			r.attempts[id] = a
		}
	}
	r.mu.Unlock()
}

// Team returns the team of this run.
func (r *Run) Team() Team { return r.team }

func (r *Run) IsRunning() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.Status == "running"
}

// Coverage summarises which files were analysed.
type Coverage struct {
	Files       int            `json:"files"`
	Analysed    int            `json:"analysed"`
	ByAgent     map[string]int `json:"byAgent"`
	NotRead     []Skip         `json:"notRead"`
	Skipped     []Skip         `json:"skipped"`
	ReadByModel int            `json:"readByModel"`
}

// CoverageSummary returns coverage of the run.
func (r *Run) CoverageSummary() Coverage {
	r.mu.Lock()
	defer r.mu.Unlock()
	cv := Coverage{Files: len(r.Code.Files), ByAgent: map[string]int{}, Skipped: r.Code.Skipped}
	modelAgents := map[string]bool{}
	for _, a := range r.team.Agents() {
		if a.Kind == "model" {
			modelAgents[a.ID] = true
		}
	}
	for _, f := range r.Code.Files {
		ag := r.coverage[f.Path]
		if len(ag) > 0 {
			cv.Analysed++
		} else {
			reason := "no agent reads " + f.Lang + " files yet"
			cv.NotRead = append(cv.NotRead, Skip{f.Path, reason})
		}
		for a := range ag {
			cv.ByAgent[a]++
		}
		for a := range ag {
			if modelAgents[a] {
				cv.ReadByModel++
				break
			}
		}
	}
	if len(cv.NotRead) > 200 {
		cv.NotRead = cv.NotRead[:200]
	}
	if len(cv.Skipped) > 200 {
		cv.Skipped = cv.Skipped[:200]
	}
	return cv
}

// Result is the full output of a run.
type Result struct {
	Telemetry []AgentAttempt `json:"telemetry,omitempty"`
	Run       *Run           `json:"run"`
	Source    any            `json:"source"`
	Coverage  Coverage       `json:"coverage"`
	Findings  []Finding      `json:"findings"`
	Questions []Message      `json:"questions"`
	Handoffs  []Message      `json:"handoffs"`
	Reading   []Reading      `json:"reading"`
	Sent      []Sent         `json:"sentToModel"`
	Counts    map[string]int `json:"counts"`
	Report    string         `json:"report"`
	Agents    []AgentInfo    `json:"agents"`
}

// ListSummary returns lightweight status for a batch run without copying its findings.
func (r *Run) ListSummary() Result {
	r.mu.Lock()
	progress := r.Batches
	if progress == nil {
		r.mu.Unlock()
		return r.Snapshot()
	}
	summary := &Run{ID: r.ID, Status: r.Status, Started: r.Started, Options: r.Options, Batches: progress, Problems: append([]Problem(nil), r.Problems...)}
	r.mu.Unlock()
	return Result{Run: summary, Source: map[string]any{"root": r.Code.Root, "files": len(r.Code.Files), "batches": progress.Total}}
}

// Snapshot returns the current result.
func (r *Run) Snapshot() Result {
	r.mu.Lock()
	telemetry := sortAttempts(r.attempts)
	sent := append([]Sent(nil), r.SentToModel...)
	reading := append([]Reading(nil), r.Reading...)
	report := r.Report
	runCopy := &Run{ID: r.ID, Status: r.Status, Error: r.Error, Started: r.Started, Finished: r.Finished, Options: r.Options, Batches: r.Batches, Steps: append([]Step(nil), r.Steps...), Problems: append([]Problem(nil), r.Problems...)}
	r.mu.Unlock()
	source := map[string]any{"root": r.Code.Root, "files": len(r.Code.Files)}
	if runCopy.Batches == nil {
		for k, v := range r.team.Source(r) {
			source[k] = v
		}
	} else {
		source["lines"] = r.Code.TotalLines()
		source["batches"] = runCopy.Batches.Total
	}
	return Result{
		Run:       runCopy,
		Telemetry: telemetry,
		Source:    source,
		Coverage:  r.CoverageSummary(),
		Findings:  r.Store.Findings(),
		Questions: r.Store.Messages(MsgQuestion),
		Handoffs:  r.Store.Messages(MsgHandoff),
		Reading:   reading,
		Sent:      sent,
		Counts:    r.Store.CountByAgent(),
		Report:    report,
		Agents:    ResolvedAgents(r.team.Agents()),
	}
}

func (r *Run) save() {
	if r.Dir == "" {
		return
	}
	b, err := json.MarshalIndent(r.Snapshot(), "", " ")
	if err == nil {
		os.WriteFile(filepath.Join(r.Dir, "result.json"), b, 0o644)
	}
	os.WriteFile(filepath.Join(r.Dir, "report.md"), []byte(r.Report), 0o644)
}
