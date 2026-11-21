package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/mjaffry01/mindsuite-doctrine-of-legacy-modernization/backend/internal/agents"
)

// designDataPolicy asks the Data policy agent to draft a data and caching policy per business object.
// The web app sends it only when the owner presses the button for one request.
func (s *Server) designDataPolicy(w http.ResponseWriter, r *http.Request) {
	var in agents.DataPolicyInput
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 256<<10))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		httpErr(w, http.StatusBadRequest, "Invalid data policy request")
		return
	}
	if err := in.Validate(); err != nil {
		httpErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if s.llm == nil {
		httpErr(w, http.StatusServiceUnavailable, "Configure Claude on the project backend to use the Data policy agent.")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 4*time.Minute)
	defer cancel()
	log.Printf("data policy agent: %d objects, %d journeys, %d layers sent to Claude", len(in.Objects), len(in.Journeys), len(in.Layers))
	out, err := s.llm.DataPolicy(ctx, in)
	if err != nil {
		httpErr(w, http.StatusBadGateway, "Claude could not draft the data policy: "+err.Error())
		return
	}
	writeJSON(w, 200, out)
}
