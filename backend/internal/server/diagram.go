package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/mjaffry01/mindsuite-doctrine-of-legacy-modernization/backend/internal/agents"
)

// designDiagram asks the Diagram agent to read the whole design and return a reviewed component diagram.
// The web app sends it only when the owner presses the button for one request.
func (s *Server) designDiagram(w http.ResponseWriter, r *http.Request) {
	var in agents.DiagramInput
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 256<<10))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		httpErr(w, http.StatusBadRequest, "Invalid diagram request")
		return
	}
	if err := in.Validate(); err != nil {
		httpErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if s.llm == nil {
		httpErr(w, http.StatusServiceUnavailable, "Configure Claude on the project backend to use the Diagram agent.")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 4*time.Minute)
	defer cancel()
	log.Printf("diagram agent: %d layers, %d capabilities, %d draft components sent to Claude", len(in.Layers), len(in.Capabilities), len(in.DraftNodes))
	out, err := s.llm.DesignDiagram(ctx, in)
	if err != nil {
		httpErr(w, http.StatusBadGateway, "Claude could not draw the diagram: "+err.Error())
		return
	}
	writeJSON(w, 200, out)
}
