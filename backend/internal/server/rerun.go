package server

import (
	"context"
	"errors"
	"net/http"
	"regexp"

	"github.com/mjaffry01/mindsuite-doctrine-of-legacy-modernization/backend/internal/agents"
)

var partKey = regexp.MustCompile("^[a-z0-9-]{1,40}$")

// rerunAgent runs one failed agent of a finished run again: POST /api/runs/{id}/rerun?part=<key>.
// It works while the run's evidence is still loaded, that is until the backend restarts.
func (s *Server) rerunAgent(w http.ResponseWriter, r *http.Request) {
	workspace, ok := workspaceFrom(w, r)
	if !ok {
		return
	}
	part := r.URL.Query().Get("part")
	if !partKey.MatchString(part) {
		httpErr(w, 400, "invalid agent")
		return
	}
	run := s.liveRun(workspace.id, r.PathValue("id"))
	if run == nil {
		httpErr(w, 409, "This run's evidence is no longer loaded (the backend was restarted). Run the team again.")
		return
	}
	switch err := run.Rerun(context.Background(), part); {
	case err == nil:
		writeJSON(w, 202, map[string]any{"id": run.ID, "part": part})
	case errors.Is(err, agents.ErrRunBusy), errors.Is(err, agents.ErrNoSuchPart), errors.Is(err, agents.ErrCannotRerun):
		httpErr(w, 409, err.Error())
	default:
		httpErr(w, 500, err.Error())
	}
}
