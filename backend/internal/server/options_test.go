package server

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOptionsRefusals(t *testing.T) {
	post := func(s *Server, body string) int {
		req := httptest.NewRequest("POST", "/api/decide/options", strings.NewReader(body))
		req.Header.Set("X-MindSuite", "1")
		rec := httptest.NewRecorder()
		s.Handler().ServeHTTP(rec, req)
		return rec.Code
	}
	ok := `{"context":"Whole system","areas":["data"],"facts":[{"id":"F1","area":"data","text":"SQL Server"}],"rules":[]}`
	if c := post(New(Config{}), ok); c != 503 {
		t.Fatalf("no key: %d", c)
	}
	s := New(Config{APIKey: "test-not-used"})
	for name, body := range map[string]string{
		"names":    `{"areas":["data"],"facts":[{"id":"F1","area":"data","text":"x"}],"owner":"Asha"}`,
		"rows":     `{"areas":["data"],"facts":[{"id":"F1","area":"data","text":"x","rows":[1]}]}`,
		"no facts": `{"areas":["data"],"facts":[]}`,
		"no area":  `{"areas":[],"facts":[{"id":"F1","area":"data","text":"x"}]}`,
		"bad json": `{`,
	} {
		if c := post(s, body); c != 400 {
			t.Errorf("%s: %d", name, c)
		}
	}
}
