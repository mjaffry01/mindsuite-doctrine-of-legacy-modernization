package database

import (
	"context"

	"github.com/mjaffry01/mindsuite-doctrine-of-legacy-modernization/backend/internal/agents"
)

// The database team: 14 agents that study a database from the DBA's exports (metadata only).

func init() { agents.Register(team{}) }

type team struct{}

// state is this team's working state for one run.
type state struct {
	db      *DBModel
	ids     map[string]string // table key -> finding id
	codeIDs map[string]string // program key -> finding id
}

func st(r *Run) *state { return r.State.(*state) }

func (team) ID() string                          { return "database" }
func (team) Agents() []AgentInfo                 { return DBTeam }
func (team) StepTitles() []string                { return dbStepTitles }
func (team) NewState() any                       { return &state{ids: map[string]string{}, codeIDs: map[string]string{}} }
func (team) Execute(ctx context.Context, r *Run) { executeDB(ctx, r) }
func (team) WholeKinds() map[string]bool {
	return map[string]bool{"db-engine": true, "db-size": true, "schema": true, "migration-order": true, "migration-approach": true}
}
func (team) Source(r *Run) map[string]any {
	if s, ok := r.State.(*state); ok && s.db != nil {
		return s.db.Summary()
	}
	return nil
}
