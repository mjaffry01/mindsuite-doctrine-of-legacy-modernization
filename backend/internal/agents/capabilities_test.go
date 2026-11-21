package agents

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func capIn() CapabilityInput {
	return CapabilityInput{Project: "StockPilot", Mode: "new", Answers: map[string]string{
		"users": "Store staff: count and move stock; buyers: reorder",
		"mvp":   "Stock in/out, counts, low-stock alerts. Not yet: forecasting, supplier portal",
	}, Existing: []string{"Count a shelf"}}
}

func TestCapabilityValidate(t *testing.T) {
	if err := capIn().Validate(); err != nil {
		t.Fatal(err)
	}
	bad := map[string]CapabilityInput{
		"mode":        {Mode: "modernize", Answers: map[string]string{"mvp": "x"}},
		"numbers":     {Mode: "new", Answers: map[string]string{"users_count": "400"}},
		"no scope":    {Mode: "new", Answers: map[string]string{"success": "98% accuracy"}},
		"empty value": {Mode: "new", Answers: map[string]string{"mvp": " "}},
		"long name":   {Mode: "new", Answers: map[string]string{"mvp": "x"}, Existing: []string{strings.Repeat("a", 121)}},
	}
	for name, in := range bad {
		if in.Validate() == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestCapabilityDraftsAreGrounded(t *testing.T) {
	in := capIn()
	got := in.verified([]CapabilityDraft{
		{"Move stock", "Record stock moving in and out", "first", "Staff move stock", "count and move stock", ""},
		{"Count a shelf", "Duplicate of existing", "first", "r", "count and move stock", ""},
		{"move  stock", "Duplicate of the first draft", "first", "r", "Stock in/out", ""},
		{"Forecast demand", "Predict what to reorder", "later", "Left out of the first release", "Not yet: forecasting", ""},
		{"Low-stock alerts", "Warn staff", "soon", "bad scope", "low-stock alerts", ""},
		{"Raise orders", "Reorder", "first", "made-up quote", "buyers raise purchase orders daily", ""},
		{"Pay with Stripe", "Take card payments through Stripe", "first", "vendor not in the brief", "Stock in/out", ""},
	})
	if len(got) != 2 || got[0].Name != "Move stock" || got[0].QuoteFrom != "users" || got[1].ReleaseScope != "later" || got[1].QuoteFrom != "mvp" {
		t.Fatalf("unexpected drafts: %+v", got)
	}
}

func TestCapabilityRetriesOnce(t *testing.T) {
	calls := 0
	fake := func(_ context.Context, _, system, user string, _ map[string]any, _ int64, out any) error {
		calls++
		if !strings.Contains(user, "StockPilot") {
			t.Fatalf("brief not sent: %s", user)
		}
		body := `{"capabilities":[]}`
		if calls == 2 {
			if !strings.Contains(system, "Try once more") {
				t.Fatal("retry prompt missing")
			}
			body = `{"capabilities":[{"name":"Receive stock","description":"Record a delivery","releaseScope":"first","reason":"In the MVP","quote":"Stock in/out"}]}`
		}
		return json.Unmarshal([]byte(body), out)
	}
	out, err := RunSuggestCapabilities(context.Background(), capIn(), fake)
	if err != nil || calls != 2 || out.Attempts != 2 || len(out.Capabilities) != 1 || out.Model == "" {
		t.Fatalf("calls %d out %+v err %v", calls, out, err)
	}
}
