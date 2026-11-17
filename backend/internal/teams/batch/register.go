package batch

import (
	"context"

	"github.com/mjaffry01/mindsuite-doctrine-of-legacy-modernization/backend/internal/agents"
)

// The batch & scheduling team: 10 agents that study job streams, schedules, dependencies, failures and reruns
// from scheduler exports and run history.

func init() { agents.Register(team{}) }

type team struct{}

// state is this team's working state for one run.
type state struct {
	m       *BModel
	ids     map[string]string // job name (lower case) -> finding id
	streams [][]string
	window  string
}

func st(r *Run) *state { return r.State.(*state) }

func (team) ID() string                          { return "batch" }
func (team) Agents() []AgentInfo                 { return BatchTeam }
func (team) StepTitles() []string                { return batchStepTitles }
func (team) NewState() any                       { return &state{ids: map[string]string{}} }
func (team) Execute(ctx context.Context, r *Run) { executeBatch(ctx, r) }
func (team) WholeKinds() map[string]bool {
	return map[string]bool{"batch-estate": true, "migration-order": true, "migration-approach": true, "scheduler-target": true}
}
func (team) Source(r *Run) map[string]any {
	if s, ok := r.State.(*state); ok && s.m != nil {
		return s.m.Summary()
	}
	return nil
}
