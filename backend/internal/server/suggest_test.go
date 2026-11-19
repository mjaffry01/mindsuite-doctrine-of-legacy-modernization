package server

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSuggestRefusals(t *testing.T) {
	post := func(s *Server, body string) int {
		req := httptest.NewRequest("POST", "/api/decide/suggest", strings.NewReader(body))
		req.Header.Set("X-MindSuite", "1")
		rec := httptest.NewRecorder()
		s.Handler().ServeHTTP(rec, req)
		return rec.Code
	}
	claim := `{"claims":[{"kind":"business-rule","role":"process","text":"Bills close on the 25th","verdict":"confirmed","correction":""}]}`
	if c := post(New(Config{}), claim); c != 503 {
		t.Fatalf("no key: %d", c)
	}
	s := New(Config{APIKey: "test-not-used"})
	for name, body := range map[string]string{
		"names":      `{"claims":[{"text":"x","verdict":"confirmed","byName":"Asha"}]}`,
		"comment":    `{"claims":[{"text":"x","verdict":"confirmed","comment":"hi"}]}`,
		"unverified": `{"claims":[{"text":"x","verdict":"open"}]}`,
		"empty":      `{"claims":[]}`,
		"bad json":   `{`,
	} {
		if c := post(s, body); c != 400 {
			t.Errorf("%s: %d", name, c)
		}
	}
}
