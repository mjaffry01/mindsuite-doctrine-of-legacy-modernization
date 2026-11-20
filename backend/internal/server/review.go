package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/mjaffry01/mindsuite-doctrine-of-legacy-modernization/backend/internal/agents"
)

// reviewDecision asks Claude to challenge one decision before it is recorded.
// The web app sends it only after the owner ticks the consent box for this one request.
func (s *Server) reviewDecision(w http.ResponseWriter, r *http.Request) {
	if s.llm == nil {
		httpErr(w, http.StatusServiceUnavailable, "Claude is not configured on this backend")
		return
	}
	var in agents.ReviewInput
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	dec.DisallowUnknownFields() // a rationale, names or comments are refused, not ignored
	if err := dec.Decode(&in); err != nil {
		httpErr(w, http.StatusBadRequest, "could not read the request: "+err.Error())
		return
	}
	if err := in.Validate(); err != nil {
		httpErr(w, http.StatusBadRequest, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Minute)
	defer cancel()
	log.Printf("decision review: approach %s, %d target choices, %d reasons, %d verified claims sent to Claude", in.Approach, len(in.Target), len(in.Reasons), len(in.Claims))
	out, err := s.llm.ReviewDecision(ctx, in)
	if err != nil {
		httpErr(w, http.StatusBadGateway, "Claude could not review the decision: "+err.Error())
		return
	}
	writeJSON(w, 200, out)
}
