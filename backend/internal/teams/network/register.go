package network

import (
	"context"

	"github.com/mjaffry01/mindsuite-doctrine-of-legacy-modernization/backend/internal/agents"
)

// The network edge & partners team: 11 agents that study entry points, DNS, certificates, partner connections
// and firewall openings from configuration and inventory exports.

func init() { agents.Register(team{}) }

type team struct{}

// state is this team's working state for one run.
type state struct {
	m                    *NModel
	entryIDs, partnerIDs map[string]string
}

func st(r *Run) *state { return r.State.(*state) }

func (team) ID() string           { return "network" }
func (team) Agents() []AgentInfo  { return NetTeam }
func (team) StepTitles() []string { return netStepTitles }
func (team) NewState() any {
	return &state{entryIDs: map[string]string{}, partnerIDs: map[string]string{}}
}
func (team) Execute(ctx context.Context, r *Run) { executeNet(ctx, r) }
func (team) WholeKinds() map[string]bool {
	return map[string]bool{"edge-estate": true, "migration-order": true, "migration-approach": true}
}
func (team) Source(r *Run) map[string]any {
	if s, ok := r.State.(*state); ok && s.m != nil {
		return s.m.Summary()
	}
	return nil
}
