package documents

import (
	"context"

	"github.com/mjaffry01/mindsuite-doctrine-of-legacy-modernization/backend/internal/agents"
)

// The documents & diagrams team: 12 agents that read business rules, processes, glossary and intended
// architecture from documents and diagrams, with citations.

func init() { agents.Register(team{}) }

type team struct{}

// state is this team's working state for one run.
type state struct {
	m         *DModel
	docIDs    map[string]string // document path -> finding id
	ruleLines map[string][]int  // document path -> lines with rule statements
}

func st(r *Run) *state { return r.State.(*state) }

func (team) ID() string                          { return "documents" }
func (team) Agents() []AgentInfo                 { return DocTeam }
func (team) StepTitles() []string                { return docStepTitles }
func (team) NewState() any                       { return &state{docIDs: map[string]string{}, ruleLines: map[string][]int{}} }
func (team) Execute(ctx context.Context, r *Run) { executeDocs(ctx, r) }
func (team) WholeKinds() map[string]bool {
	return map[string]bool{"doc-estate": true, "topic-coverage": true}
}
func (team) Source(r *Run) map[string]any {
	if s, ok := r.State.(*state); ok && s.m != nil {
		return s.m.Summary()
	}
	return nil
}
