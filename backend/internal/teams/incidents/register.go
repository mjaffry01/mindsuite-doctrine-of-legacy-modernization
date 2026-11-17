package incidents

import (
	"context"
	"time"

	"github.com/mjaffry01/mindsuite-doctrine-of-legacy-modernization/backend/internal/agents"
)

// The incidents & changes team: 11 agents that find where the system breaks most often, recurring problems,
// change frequency and failed changes, from ServiceNow and Jira exports.

func init() { agents.Register(team{}) }

type team struct{}

// state is this team's working state for one run.
type state struct {
	m         *IModel
	hotIDs    map[string]string // component -> hotspot finding id
	recIDs    []string
	groups    []*group
	changeIDs []string
	failedBy  map[string]int
	perMonth  float64
	failRate  int
	mttr      time.Duration
}

func st(r *Run) *state { return r.State.(*state) }

func (team) ID() string           { return "incidents" }
func (team) Agents() []AgentInfo  { return IncTeam }
func (team) StepTitles() []string { return incStepTitles }
func (team) NewState() any {
	return &state{hotIDs: map[string]string{}, failedBy: map[string]int{}}
}
func (team) Execute(ctx context.Context, r *Run) { executeInc(ctx, r) }
func (team) WholeKinds() map[string]bool {
	return map[string]bool{"ticket-estate": true, "data-window": true}
}
func (team) Source(r *Run) map[string]any {
	if s, ok := r.State.(*state); ok && s.m != nil {
		return s.m.Summary()
	}
	return nil
}
