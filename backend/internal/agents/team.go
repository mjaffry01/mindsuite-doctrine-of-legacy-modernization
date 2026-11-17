package agents

import (
	"context"
	"sort"
	"sync"
)

// AgentInfo describes one member of a team (ids match the team's explanation page in the web app).
type AgentInfo struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Kind  string `json:"kind"`  // tool | model | tool+model | coordinator
	Model string `json:"model"` // "" for no model
	Does  string `json:"does"`
}

// Model tiers. The verifier deliberately uses a different model from the agents it checks.
var (
	ModelStrong   = "claude-opus-5-5"
	ModelBalanced = "claude-sonnet-5-5"
	ModelFast     = "claude-haiku-5-5"
)

// Team is one agent team. Each team lives in its own folder under internal/teams/ and registers itself.
// This package is the shared core: the run, the evidence store, the loader, the verifier and the Claude client.
type Team interface {
	ID() string                          // "code", "database", "infrastructure", …
	Agents() []AgentInfo                 // the team, in the order of a run
	StepTitles() []string                // the 9 steps shown to the client
	WholeKinds() map[string]bool         // finding kinds computed from the whole input (no single citation)
	NewState() any                       // the team's own working state for one run (r.State)
	Execute(ctx context.Context, r *Run) // runs steps 1–9
	Source(r *Run) map[string]any        // what was analysed, shown with the result
}

var (
	regMu    sync.Mutex
	registry = map[string]Team{}
)

// Register adds a team. Team packages call it from init().
func Register(t Team) {
	regMu.Lock()
	registry[t.ID()] = t
	regMu.Unlock()
}

// TeamByID returns a registered team (nil when unknown).
func TeamByID(id string) Team {
	regMu.Lock()
	defer regMu.Unlock()
	return registry[id]
}

// TeamIDs lists the registered teams.
func TeamIDs() []string {
	regMu.Lock()
	defer regMu.Unlock()
	ids := []string{}
	for k := range registry {
		ids = append(ids, k)
	}
	sort.Strings(ids)
	return ids
}
