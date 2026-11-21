package server

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCapabilityRefusals(t *testing.T) {
	post := func(s *Server, body string) int {
		req := httptest.NewRequest("POST", "/api/brief/capabilities", strings.NewReader(body))
		req.Header.Set("X-MindSuite", "1")
		rec := httptest.NewRecorder()
		s.Handler().ServeHTTP(rec, req)
		return rec.Code
	}
	ok := `{"project":"StockPilot","mode":"new","answers":{"mvp":"Stock in/out"}}`
	if c := post(New(Config{}), ok); c != 503 {
		t.Fatalf("no key: %d", c)
	}
	s := New(Config{APIKey: "test-not-used"})
	for name, body := range map[string]string{
		"unknown field": `{"mode":"new","answers":{"mvp":"x"},"owner":"Asha"}`,
		"number field":  `{"mode":"new","answers":{"users_count":"400"}}`,
		"no scope":      `{"mode":"new","answers":{}}`,
		"bad mode":      `{"mode":"modernize","answers":{"mvp":"x"}}`,
		"bad json":      `{`,
	} {
		if c := post(s, body); c != 400 {
			t.Errorf("%s: %d", name, c)
		}
	}
}
