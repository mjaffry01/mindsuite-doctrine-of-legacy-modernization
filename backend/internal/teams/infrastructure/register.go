package infrastructure

import (
	"context"

	"github.com/mjaffry01/mindsuite-doctrine-of-legacy-modernization/backend/internal/agents"
)

// The environment & infrastructure team: 13 agents that study where the system runs, from inventory exports.

func init() { agents.Register(team{}) }

type team struct{}

// state is this team's working state for one run.
type state struct {
	m   *InfraModel
	ids map[string]string // host name (lower case) -> finding id
}

func st(r *Run) *state { return r.State.(*state) }

func (team) ID() string                          { return "infrastructure" }
func (team) Agents() []AgentInfo                 { return InfraTeam }
func (team) StepTitles() []string                { return infraStepTitles }
func (team) NewState() any                       { return &state{ids: map[string]string{}} }
func (team) Execute(ctx context.Context, r *Run) { executeInfra(ctx, r) }
func (team) WholeKinds() map[string]bool {
	return map[string]bool{"estate": true, "capacity": true, "os-mix": true, "cloud-summary": true, "target-sizing": true, "migration-order": true, "migration-approach": true}
}
func (team) Source(r *Run) map[string]any {
	if s, ok := r.State.(*state); ok && s.m != nil {
		return s.m.Summary()
	}
	return nil
}
