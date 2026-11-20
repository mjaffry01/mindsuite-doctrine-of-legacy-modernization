package server

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReviewRefusals(t *testing.T) {
	post := func(s *Server, body string) int {
		req := httptest.NewRequest("POST", "/api/decide/review", strings.NewReader(body))
		req.Header.Set("X-MindSuite", "1")
		rec := httptest.NewRecorder()
		s.Handler().ServeHTTP(rec, req)
		return rec.Code
	}
	ok := `{"approach":"rebuild","target":[{"area":"cloud","option":"Microsoft Azure"}],"reasons":["+ Core domain"],"checks":[],"missing":[],"claims":[]}`
	if c := post(New(Config{}), ok); c != 503 {
		t.Fatalf("no key: %d", c)
	}
	s := New(Config{APIKey: "test-not-used"})
	for name, body := range map[string]string{
		"rationale":   `{"approach":"rebuild","rationale":"Asha decided"}`,
		"names":       `{"approach":"rebuild","decidedBy":"Asha"}`,
		"no approach": `{"target":[]}`,
		"unverified":  `{"approach":"rebuild","claims":[{"text":"x","verdict":"open"}]}`,
		"bad json":    `{`,
	} {
		if c := post(s, body); c != 400 {
			t.Errorf("%s: %d", name, c)
		}
	}
}
