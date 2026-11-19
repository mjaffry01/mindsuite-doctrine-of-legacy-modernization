package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/mjaffry01/mindsuite-doctrine-of-legacy-modernization/backend/internal/agents"
)

// suggestApproaches asks Claude which approaches fit one context, from verified claims only.
// The web app sends it only after the owner ticks the consent box for this one request.
func (s *Server) suggestApproaches(w http.ResponseWriter, r *http.Request) {
	if s.llm == nil {
		httpErr(w, http.StatusServiceUnavailable, "Claude is not configured on this backend")
		return
	}
	var in agents.SuggestInput
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	dec.DisallowUnknownFields() // anything beyond the verified claims (names, comments) is refused, not ignored
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
	log.Printf("approach suggestion: %d verified claims sent to Claude", len(in.Claims))
	out, err := s.llm.SuggestApproaches(ctx, in)
	if err != nil {
		httpErr(w, http.StatusBadGateway, "Claude could not suggest approaches: "+err.Error())
		return
	}
	writeJSON(w, 200, out)
}
