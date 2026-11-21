package server

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDataPolicyRefusals(t *testing.T) {
	post := func(s *Server, body string) int {
		req := httptest.NewRequest("POST", "/api/design/datapolicy", strings.NewReader(body))
		req.Header.Set("X-MindSuite", "1")
		rec := httptest.NewRecorder()
		s.Handler().ServeHTTP(rec, req)
		return rec.Code
	}
	ok := `{"project":"P","kind":"new","objects":[{"name":"Stock"}]}`
	if c := post(New(Config{}), ok); c != 503 {
		t.Fatalf("no key: %d", c)
	}
	s := New(Config{APIKey: "test-not-used"})
	for name, body := range map[string]string{
		"unknown field": `{"kind":"new","objects":[{"name":"Stock"}],"owner":"Asha"}`,
		"no objects":    `{"kind":"new","objects":[]}`,
		"bad kind":      `{"kind":"x","objects":[{"name":"Stock"}]}`,
		"duplicate":     `{"kind":"new","objects":[{"name":"Stock"},{"name":"stock"}]}`,
		"bad json":      `{`,
	} {
		if c := post(s, body); c != 400 {
			t.Errorf("%s: %d", name, c)
		}
	}
}
