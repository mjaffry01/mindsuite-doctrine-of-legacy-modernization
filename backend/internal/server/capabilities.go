package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/mjaffry01/mindsuite-doctrine-of-legacy-modernization/backend/internal/agents"
)

// suggestCapabilities drafts First-release scope capabilities from the product brief.
// The web app sends it only when the owner presses "Suggest capabilities".
func (s *Server) suggestCapabilities(w http.ResponseWriter, r *http.Request) {
	var in agents.CapabilityInput
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		httpErr(w, http.StatusBadRequest, "Invalid capability request")
		return
	}
	if err := in.Validate(); err != nil {
		httpErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if s.llm == nil {
		httpErr(w, http.StatusServiceUnavailable, "Configure Claude on the project backend to use the Capability agent.")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()
	log.Printf("capability agent: %d brief answers, %d existing capabilities sent to Claude", len(in.Answers), len(in.Existing))
	out, err := s.llm.SuggestCapabilities(ctx, in)
	if err != nil {
		httpErr(w, http.StatusBadGateway, "Claude could not suggest capabilities: "+err.Error())
		return
	}
	writeJSON(w, 200, out)
}
