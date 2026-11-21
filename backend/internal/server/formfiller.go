package server

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/mjaffry01/mindsuite-doctrine-of-legacy-modernization/backend/internal/agents"
)

func (s *Server) fillProductBrief(w http.ResponseWriter, r *http.Request) {
	var in agents.BriefFillInput
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		httpErr(w, 400, "Invalid product brief request")
		return
	}
	if err := in.Validate(); err != nil {
		httpErr(w, 400, err.Error())
		return
	}
	if s.llm == nil {
		httpErr(w, 503, "Configure Claude on the project backend to use the Formfiller agent.")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()
	out, err := s.llm.FillProductBrief(ctx, in)
	if err != nil {
		httpErr(w, 502, "Claude could not complete the product brief: "+err.Error())
		return
	}
	writeJSON(w, 200, out)
}
