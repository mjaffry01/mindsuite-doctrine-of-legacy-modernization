package telemetry

import (
	"context"

	"github.com/mjaffry01/mindsuite-doctrine-of-legacy-modernization/backend/internal/agents"
)

// The telemetry team: 11 agents that study how the system is really used, from logs and monitoring exports.

func init() { agents.Register(team{}) }

type team struct{}

// state is this team's working state for one run.
type state struct {
	m               *TModel
	srcIDs, useIDs  map[string]string
	avgRps, peakRps float64
	p95, errRate    float64
	peakHours       []string
	peakMonths      []int
	cutover         string
}

func st(r *Run) *state { return r.State.(*state) }

// FinalizeBatches preserves all-window metrics and checks routes against the full timeline.
func (team) FinalizeBatches(ctx context.Context, r *Run) { executeTel(ctx, r) }

func (team) ID() string                          { return "telemetry" }
func (team) Agents() []AgentInfo                 { return TelTeam }
func (team) StepTitles() []string                { return telStepTitles }
func (team) NewState() any                       { return &state{srcIDs: map[string]string{}, useIDs: map[string]string{}} }
func (team) Execute(ctx context.Context, r *Run) { executeTel(ctx, r) }
func (team) WholeKinds() map[string]bool {
	return map[string]bool{"telemetry-window": true, "migration-approach": true}
}
func (team) Source(r *Run) map[string]any {
	if s, ok := r.State.(*state); ok && s.m != nil {
		return s.m.Summary()
	}
	return nil
}
