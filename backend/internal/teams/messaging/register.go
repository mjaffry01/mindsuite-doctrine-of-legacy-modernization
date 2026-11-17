package messaging

import (
	"context"

	"github.com/mjaffry01/mindsuite-doctrine-of-legacy-modernization/backend/internal/agents"
)

// The messaging & caching team: 12 agents that study brokers, queues, topics and caches from admin-command output.

func init() { agents.Register(team{}) }

type team struct{}

// state is this team's working state for one run.
type state struct {
	m                            *MsgModel
	ids                          map[string]string // queue/topic name -> finding id
	brokerIDs, cacheIDs, chanIDs map[string]string
	flowIDs                      map[string]string
}

func st(r *Run) *state { return r.State.(*state) }

func (team) ID() string           { return "messaging" }
func (team) Agents() []AgentInfo  { return MsgTeam }
func (team) StepTitles() []string { return msgStepTitles }
func (team) NewState() any {
	return &state{ids: map[string]string{}, brokerIDs: map[string]string{}, cacheIDs: map[string]string{}, chanIDs: map[string]string{}, flowIDs: map[string]string{}}
}
func (team) Execute(ctx context.Context, r *Run) { executeMsg(ctx, r) }
func (team) WholeKinds() map[string]bool {
	return map[string]bool{"messaging-estate": true, "migration-order": true, "migration-approach": true, "not-in-code": true}
}
func (team) Source(r *Run) map[string]any {
	if s, ok := r.State.(*state); ok && s.m != nil {
		return s.m.Summary()
	}
	return nil
}
