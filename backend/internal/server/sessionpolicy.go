package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/mjaffry01/mindsuite-doctrine-of-legacy-modernization/backend/internal/agents"
)

// designSessionPolicy asks the Session policy agent to draft the cookie and session policy.
// The web app sends it only when the owner presses the button for one request.
func (s *Server) designSessionPolicy(w http.ResponseWriter, r *http.Request) {
	var in agents.SessionPolicyInput
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 256<<10))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		httpErr(w, http.StatusBadRequest, "Invalid session policy request")
		return
	}
	if err := in.Validate(); err != nil {
		httpErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if s.llm == nil {
		httpErr(w, http.StatusServiceUnavailable, "Configure Claude on the project backend to use the Session policy agent.")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 4*time.Minute)
	defer cancel()
	log.Printf("session policy agent: %d user types, %d channels, %d layers sent to Claude", len(in.UserTypes), len(in.Channels), len(in.Layers))
	out, err := s.llm.SessionPolicy(ctx, in)
	if err != nil {
		httpErr(w, http.StatusBadGateway, "Claude could not draft the session policy: "+err.Error())
		return
	}
	writeJSON(w, 200, out)
}
