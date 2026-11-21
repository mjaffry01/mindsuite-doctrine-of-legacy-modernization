package agents

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestFormfillerProtectsAnswersAndRequiresGrounding(t *testing.T) {
	in := BriefFillInput{Project: "Dispatch scheduling", Mode: "new", Answers: map[string]string{"problem": "Dispatchers schedule 30 deliveries per day."}, Missing: []string{"users", "availability", "where"}}
	if err := in.Validate(); err != nil {
		t.Fatal(err)
	}
	out := in.verifiedSuggestions([]BriefSuggestion{
		{"users", "Dispatchers plan each route.", "Daily scheduling requires dispatchers.", "Dispatchers schedule", ""},
		{"users", "Duplicate", "Reason", "Dispatchers", ""},
		{"problem", "Changed", "Reason", "Dispatchers", ""},
		{"where", "invalid", "Reason", "Dispatchers", ""},
		{"availability", "99.99", "Reason", "30 deliveries", ""},
		{"where", "public", "Reason", "a fabricated citation", ""},
	})
	if len(out) != 1 || out[0].Field != "users" {
		t.Fatalf("unexpected suggestions: %+v", out)
	}
	in.Missing = []string{"problem"}
	if err := in.Validate(); err == nil {
		t.Fatal("accepted overwrite request")
	}
}

func TestNextPredictionUsesTentativeContext(t *testing.T) {
	in := BriefFillInput{Mode: "new", Project: "Cinema", Next: true, Previous: []BriefPrediction{{"users", "Viewers browse 200 titles"}}, Missing: []string{"users_count", "journeys", "mvp"}}
	if err := in.Validate(); err != nil {
		t.Fatal(err)
	}
	got := in.verifiedSuggestions([]BriefSuggestion{
		{Field: "users_count", Value: "200", Reason: "guess", Quote: "200 titles"},
		{Field: "journeys", Value: "Browse and select a title", Reason: "A viewer chooses a film", Quote: "Viewers browse"},
		{Field: "mvp", Value: "Film browsing", Reason: "first release", Quote: "Cinema"},
	})
	if len(got) != 1 || got[0].Field != "journeys" || got[0].QuoteFrom != "users" {
		t.Fatalf("unexpected step: %+v", got)
	}
	in.Answers = map[string]string{"users": "Editors grade screenings"}
	if in.Validate() == nil {
		t.Fatal("accepted a prediction for an answered field")
	}
	in.Answers = nil
	in.Missing = append(in.Missing, "users")
	if in.Validate() == nil {
		t.Fatal("accepted a prediction as its own input")
	}
}

func TestFormfillIntegrations(t *testing.T) {
	in := BriefFillInput{Project: "StockPilot", Mode: "new", Answers: map[string]string{"problem": "Shops track stock on paper and pay suppliers late"},
		Missing: []string{"integrations"}, Next: true,
		Previous: []BriefPrediction{{Field: "mvp", Value: "Stock in and out, card payments at checkout, daily sales posted to accounting"}}}
	if err := in.Validate(); err != nil {
		t.Fatal(err)
	}
	ok := in.verifiedSuggestions([]BriefSuggestion{{Field: "integrations", Value: "Accounting system: post daily sales; card payment provider", Reason: "The first release posts sales and takes cards", Quote: "card payments at checkout"}})
	if len(ok) != 1 || ok[0].QuoteFrom != "mvp" {
		t.Fatalf("a kind of system from an earlier draft should be accepted: %+v", ok)
	}
	named := in.verifiedSuggestions([]BriefSuggestion{{Field: "integrations", Value: "Xero accounting and Stripe", Reason: "x", Quote: "card payments at checkout"}})
	if len(named) != 0 {
		t.Fatalf("a product the brief never names must be rejected: %+v", named)
	}
	in.Answers["problem"] += "; we already use Xero"
	stated := in.verifiedSuggestions([]BriefSuggestion{{Field: "integrations", Value: "Xero: post daily sales", Reason: "x", Quote: "we already use Xero"}})
	if len(stated) != 1 {
		t.Fatalf("a product the brief names should be accepted: %+v", stated)
	}
}

func TestFormfillBuyPreference(t *testing.T) {
	in := BriefFillInput{Project: "StockPilot", Mode: "new", Answers: map[string]string{"problem": "Shops track stock on paper"},
		Missing: []string{"buyPref"}, Next: true,
		Previous: []BriefPrediction{{Field: "integrations", Value: "Card payment provider: take card payments at checkout; Accounting system: daily sales"}}}
	if err := in.Validate(); err != nil {
		t.Fatal(err)
	}
	ok := in.verifiedSuggestions([]BriefSuggestion{{Field: "buyPref", Value: "Suggested starting point: Build the stock-count workflow; Buy card payments and accounting", Reason: "Payments and accounting are common", Quote: "Card payment provider: take card payments at checkout"}})
	if len(ok) != 1 || ok[0].QuoteFrom != "integrations" {
		t.Fatalf("a suggested starting point from an earlier draft should be accepted: %+v", ok)
	}
	named := in.verifiedSuggestions([]BriefSuggestion{{Field: "buyPref", Value: "Buy Razorpay for payments", Reason: "x", Quote: "Card payment provider: take card payments at checkout"}})
	if len(named) != 0 {
		t.Fatalf("a vendor the brief never names must be rejected: %+v", named)
	}
}

func TestFormfillRetriesOnce(t *testing.T) {
	in := BriefFillInput{Project: "StockPilot", Mode: "new", Answers: map[string]string{"problem": "Shops track stock on paper"}, Missing: []string{"users"}, Next: true}
	if err := in.Validate(); err != nil {
		t.Fatal(err)
	}
	calls := 0
	answers := []string{`{"suggestions":[]}`, `{"suggestions":[{"field":"users","value":"Shop staff count stock","reason":"Shops track stock","quote":"Shops track stock"}]}`}
	fake := func(ctx context.Context, model, system, user string, schema map[string]any, max int64, out any) error {
		calls++
		if calls == 2 && !strings.Contains(system, "first attempt returned no usable suggestion") {
			t.Error("the retry should say why it asks again")
		}
		return json.Unmarshal([]byte(answers[calls-1]), out)
	}
	out, err := RunFillProductBrief(context.Background(), in, fake)
	if err != nil || calls != 2 || len(out.Suggestions) != 1 || out.Attempts != 2 {
		t.Fatalf("retry once: calls %d, out %+v, err %v", calls, out, err)
	}
	calls = 0
	answers = []string{`{"suggestions":[{"field":"users","value":"Shop staff","reason":"r","quote":"Shops track stock"}]}`, `{}`}
	if out, _ := RunFillProductBrief(context.Background(), in, fake); calls != 1 || out.Attempts != 1 {
		t.Errorf("no retry when the first answer is usable: calls %d", calls)
	}
	calls = 0
	answers = []string{`{"suggestions":[]}`, `{"suggestions":[]}`}
	if out, _ := RunFillProductBrief(context.Background(), in, fake); calls != 2 || len(out.Suggestions) != 0 {
		t.Errorf("at most one retry: calls %d", calls)
	}
}
