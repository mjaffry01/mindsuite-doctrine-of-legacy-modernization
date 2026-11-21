package server

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDiagramRefusals(t *testing.T) {
	post := func(s *Server, body string) int {
		req := httptest.NewRequest("POST", "/api/design/diagram", strings.NewReader(body))
		req.Header.Set("X-MindSuite", "1")
		rec := httptest.NewRecorder()
		s.Handler().ServeHTTP(rec, req)
		return rec.Code
	}
	ok := `{"project":"P","kind":"new","layers":[{"id":"database","label":"Database","design":"PostgreSQL"}]}`
	if c := post(New(Config{}), ok); c != 503 {
		t.Fatalf("no key: %d", c)
	}
	s := New(Config{APIKey: "test-not-used"})
	for name, body := range map[string]string{
		"unknown field": `{"kind":"new","layers":[{"id":"d","label":"D","design":"x"}],"owner":"Asha"}`,
		"no layers":     `{"kind":"new","layers":[]}`,
		"bad kind":      `{"kind":"x","layers":[{"id":"d","label":"D","design":"x"}]}`,
		"bad zone":      `{"kind":"new","layers":[{"id":"d","label":"D","design":"x"}],"draftNodes":[{"id":"a","zone":"moon","title":"x","sub":""}]}`,
		"bad json":      `{`,
	} {
		if c := post(s, body); c != 400 {
			t.Errorf("%s: %d", name, c)
		}
	}
}
