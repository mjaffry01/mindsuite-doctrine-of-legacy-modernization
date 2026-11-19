// Package agents runs the MindSuite code-analysis agent team.
//
// Agents never talk to each other directly. They exchange five message types
// (task, finding, hand-off, verification, question) through an append-only
// evidence store. Every finding cites its evidence (file and line) and states
// its basis and confidence; the verifier accepts, rejects or flags it.
package agents

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// Message types exchanged through the evidence store.
const (
	MsgTask         = "task"
	MsgFinding      = "finding"
	MsgHandoff      = "handoff"
	MsgVerification = "verification"
	MsgQuestion     = "question"
	MsgLog          = "log"
)

// Basis of a finding.
const (
	BasisCode     = "observed-in-code"
	BasisInferred = "inferred"
	BasisModel    = "model"
	BasisHistory  = "git-history"
)

// Finding status, set by the verifier.
const (
	StatusProposed = "proposed"
	StatusAccepted = "accepted"
	StatusRejected = "rejected"
	StatusConflict = "conflict"
)

// Citation points to the evidence for a claim. Quotes are short code lines;
// lines flagged as containing a secret are never quoted.
type Citation struct {
	File    string `json:"file"`
	Line    int    `json:"line,omitempty"`
	EndLine int    `json:"endLine,omitempty"`
	Quote   string `json:"quote,omitempty"`
}

// Finding is one claim made by an agent.
type Finding struct {
	ID         string         `json:"id"`
	Agent      string         `json:"agent"`
	Kind       string         `json:"kind"`
	Module     string         `json:"module,omitempty"`
	Title      string         `json:"title"`
	Detail     string         `json:"detail,omitempty"`
	Data       map[string]any `json:"data,omitempty"`
	Citations  []Citation     `json:"citations,omitempty"`
	Evidence   []string       `json:"evidence,omitempty"` // ids of other findings this one builds on
	Basis      string         `json:"basis"`
	Confidence string         `json:"confidence"` // high | medium | low
	Status     string         `json:"status"`
	Verdict    string         `json:"verdict,omitempty"`
}

// Message is one entry in the append-only evidence store.
type Message struct {
	Seq     int       `json:"seq"`
	Time    time.Time `json:"time"`
	Type    string    `json:"type"`
	From    string    `json:"from"`
	To      string    `json:"to,omitempty"`
	Text    string    `json:"text,omitempty"`
	Finding *Finding  `json:"finding,omitempty"`
	Ref     string    `json:"ref,omitempty"`    // finding id for verifications
	Status  string    `json:"status,omitempty"` // verification result
	Data    any       `json:"data,omitempty"`
}

// Store is the append-only evidence store of one run.
type Store struct {
	mu       sync.Mutex
	msgs     []Message
	findings map[string]*Finding
	order    []string
	counter  map[string]int
	file     *os.File
	dir      string
	subs     map[chan Message]struct{}
}

// NewStore creates a store; if dir is not empty, messages are also written to dir/messages.jsonl.
func NewStore(dir string) (*Store, error) {
	s := &Store{findings: map[string]*Finding{}, counter: map[string]int{}, subs: map[chan Message]struct{}{}}
	s.dir = dir
	if dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
		f, err := os.Create(filepath.Join(dir, "messages.jsonl"))
		if err != nil {
			return nil, err
		}
		s.file = f
	}
	return s, nil
}

func (s *Store) append(m Message) Message {
	s.mu.Lock()
	m.Seq = len(s.msgs) + 1
	m.Time = time.Now().UTC()
	s.msgs = append(s.msgs, m)
	if s.file != nil {
		b, _ := json.Marshal(m)
		s.file.Write(append(b, '\n'))
	}
	subs := make([]chan Message, 0, len(s.subs))
	for c := range s.subs {
		subs = append(subs, c)
	}
	s.mu.Unlock()
	for _, c := range subs {
		select {
		case c <- m:
		default: // slow subscriber: it can re-read with Since
		}
	}
	return m
}

// Task records an orchestrator task for an agent.
func (s *Store) Task(to, text string, data any) {
	s.append(Message{Type: MsgTask, From: "orch", To: to, Text: text, Data: data})
}

// Log records progress text from an agent.
func (s *Store) Log(from, text string) { s.append(Message{Type: MsgLog, From: from, Text: text}) }

// Handoff records an agent telling the orchestrator something that changes the plan.
func (s *Store) Handoff(from, text string, data any) {
	s.append(Message{Type: MsgHandoff, From: from, To: "orch", Text: text, Data: data})
}

// Question records a question for the client's experts.
func (s *Store) Question(from, ref, text string) {
	s.append(Message{Type: MsgQuestion, From: from, To: "people", Ref: ref, Text: text})
}

// Add stores a new finding and returns its id.
func (s *Store) Add(f Finding) string {
	s.mu.Lock()
	s.counter[f.Agent]++
	f.ID = fmt.Sprintf("%s-%d", f.Agent, s.counter[f.Agent])
	if f.Status == "" {
		f.Status = StatusProposed
	}
	if f.Confidence == "" {
		f.Confidence = "medium"
	}
	if f.Basis == "" {
		f.Basis = BasisCode
	}
	cp := f
	s.findings[f.ID] = &cp
	s.order = append(s.order, f.ID)
	s.mu.Unlock()
	s.append(Message{Type: MsgFinding, From: f.Agent, Finding: &f})
	return f.ID
}

// addImported preserves IDs from a reviewed batch and their supporting links.
func (s *Store) addImported(f Finding) {
	s.mu.Lock()
	s.findings[f.ID] = &f
	s.order = append(s.order, f.ID)
	s.mu.Unlock()
	s.append(Message{Type: MsgFinding, From: f.Agent, Finding: &f})
}

// Verify records the verifier's result for a finding.
func (s *Store) Verify(ref, status, reason string) {
	s.mu.Lock()
	if f, ok := s.findings[ref]; ok {
		f.Status = status
		f.Verdict = reason
	}
	s.mu.Unlock()
	s.append(Message{Type: MsgVerification, From: "ver", Ref: ref, Status: status, Text: reason})
}

// Findings returns a snapshot of all findings, optionally filtered by kind.
func (s *Store) Findings(kinds ...string) []Finding {
	s.mu.Lock()
	defer s.mu.Unlock()
	want := map[string]bool{}
	for _, k := range kinds {
		want[k] = true
	}
	out := []Finding{}
	for _, id := range s.order {
		f := s.findings[id]
		if len(want) == 0 || want[f.Kind] {
			out = append(out, *f)
		}
	}
	return out
}

// Get returns one finding.
func (s *Store) Get(id string) (Finding, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, ok := s.findings[id]
	if !ok {
		return Finding{}, false
	}
	return *f, true
}

// Since returns messages with Seq > after.
func (s *Store) Since(after int) []Message {
	s.mu.Lock()
	defer s.mu.Unlock()
	if after < 0 {
		after = 0
	}
	if after >= len(s.msgs) {
		return nil
	}
	return append([]Message(nil), s.msgs[after:]...)
}

// Messages of one type.
func (s *Store) Messages(typ string) []Message {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []Message{}
	for _, m := range s.msgs {
		if m.Type == typ {
			out = append(out, m)
		}
	}
	return out
}

// Subscribe returns a channel receiving new messages; call the func to stop.
func (s *Store) Subscribe() (chan Message, func()) {
	c := make(chan Message, 256)
	s.mu.Lock()
	s.subs[c] = struct{}{}
	s.mu.Unlock()
	return c, func() {
		s.mu.Lock()
		delete(s.subs, c)
		s.mu.Unlock()
	}
}

// reopen opens the message file again to add to it (a re-run after the run finished).
func (s *Store) reopen() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.file == nil && s.dir != "" {
		if f, err := os.OpenFile(filepath.Join(s.dir, "messages.jsonl"), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o644); err == nil {
			s.file = f
		}
	}
}

// Close closes the message file.
func (s *Store) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.file != nil {
		s.file.Close()
		s.file = nil
	}
}

// CountByAgent returns finding counts per agent.
func (s *Store) CountByAgent() map[string]int {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := map[string]int{}
	for _, f := range s.findings {
		out[f.Agent]++
	}
	return out
}

func sortedKeys[V any](m map[string]V) []string {
	k := make([]string, 0, len(m))
	for x := range m {
		k = append(k, x)
	}
	sort.Strings(k)
	return k
}
