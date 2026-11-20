package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/mjaffry01/mindsuite-doctrine-of-legacy-modernization/backend/internal/agents"
)

// targetOptions runs the Target options team for one context.
// The web app sends it only after the owner ticks the consent box for this one run.
func (s *Server) targetOptions(w http.ResponseWriter, r *http.Request) {
	if s.llm == nil {
		httpErr(w, http.StatusServiceUnavailable, "Claude is not configured on this backend")
		return
	}
	var in agents.OptionsInput
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	dec.DisallowUnknownFields() // anything beyond facts and rules (names, comments, rationale) is refused
	if err := dec.Decode(&in); err != nil {
		httpErr(w, http.StatusBadRequest, "could not read the request: "+err.Error())
		return
	}
	if err := in.Validate(); err != nil {
		httpErr(w, http.StatusBadRequest, err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 6*time.Minute)
	defer cancel()
	log.Printf("target options: %d facts, %d rules, areas %v sent to Claude", len(in.Facts), len(in.Rules), in.Areas)
	if in.Design != nil {
		d, step, err := s.llm.DesignBlueprint(ctx, in)
		if err != nil {
			httpErr(w, http.StatusBadGateway, "the solution designer could not finish: "+err.Error())
			return
		}
		writeJSON(w, 200, agents.OptionsOutput{Design: &d, Steps: []agents.AgentStep{step}, Requests: 1})
		return
	}
	out, err := s.llm.TargetOptions(ctx, in)
	if err != nil {
		httpErr(w, http.StatusBadGateway, "the options team could not finish: "+err.Error())
		return
	}
	log.Printf("target options: %d options, %d blueprints, %d model requests", len(out.Options), len(out.Blueprints), out.Requests)
	writeJSON(w, 200, out)
}
