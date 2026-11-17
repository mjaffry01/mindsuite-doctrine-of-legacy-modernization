package identity

import (
	"context"

	"github.com/mjaffry01/mindsuite-doctrine-of-legacy-modernization/backend/internal/agents"
)

// The identity & access team: 12 agents that study accounts, groups, MFA, sign-in policies and certificates
// from read-only directory and provider exports.

func init() { agents.Register(team{}) }

type team struct{}

// state is this team's working state for one run.
type state struct {
	m        *IDModel
	groupIDs map[string]string // group name (lower case) -> finding id
	stale    []*Account
	svc      []*Account
	admins   int
}

func st(r *Run) *state { return r.State.(*state) }

func (team) ID() string                          { return "identity" }
func (team) Agents() []AgentInfo                 { return IDTeam }
func (team) StepTitles() []string                { return idStepTitles }
func (team) NewState() any                       { return &state{groupIDs: map[string]string{}} }
func (team) Execute(ctx context.Context, r *Run) { executeID(ctx, r) }
func (team) WholeKinds() map[string]bool {
	return map[string]bool{"identity-estate": true, "role-model": true, "migration-order": true, "migration-approach": true}
}
func (team) Source(r *Run) map[string]any {
	if s, ok := r.State.(*state); ok && s.m != nil {
		return s.m.Summary()
	}
	return nil
}
