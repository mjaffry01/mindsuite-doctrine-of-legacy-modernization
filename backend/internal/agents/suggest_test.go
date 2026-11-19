package agents

import "testing"

func TestSuggestValidate(t *testing.T) {
	ok := SuggestInput{Claims: []SuggestClaim{{Kind: "business-rule", Text: "Bills close on the 25th", Verdict: "confirmed"}}}
	if err := ok.Validate(); err != nil {
		t.Fatal(err)
	}
	for name, in := range map[string]SuggestInput{
		"none":          {},
		"unverified":    {Claims: []SuggestClaim{{Text: "x", Verdict: "open"}}},
		"rejected":      {Claims: []SuggestClaim{{Text: "x", Verdict: "rejected"}}},
		"no correction": {Claims: []SuggestClaim{{Text: "x", Verdict: "corrected"}}},
		"empty text":    {Claims: []SuggestClaim{{Text: "  ", Verdict: "confirmed"}}},
		"too many":      {Claims: make([]SuggestClaim, SuggestMaxClaims+1)},
	} {
		if err := in.Validate(); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestCleanSuggestions(t *testing.T) {
	out := cleanSuggestions(SuggestOutput{Suggestions: []Suggestion{
		{Approach: "replace", Fit: "strong", Reasons: []SuggestReason{{Text: "Generic", Claims: []int{1, 1, 9, 0}}, {Text: " "}}},
		{Approach: "replace", Fit: "weak"},
		{Approach: "teleport", Fit: "strong"},
		{Approach: "retire", Fit: "great"},
		{Approach: "rehost"}, {Approach: "rebuild"}, {Approach: "refactor"},
	}}, 2)
	if len(out.Suggestions) != 4 || out.Suggestions[0].Approach != "replace" || out.Suggestions[1].Approach != "retire" {
		t.Fatalf("%+v", out.Suggestions)
	}
	if r := out.Suggestions[0].Reasons; len(r) != 1 || len(r[0].Claims) != 1 || r[0].Claims[0] != 1 {
		t.Fatalf("reasons %+v", r)
	}
	if out.Suggestions[1].Fit != "possible" {
		t.Fatal("unknown fit kept")
	}
}
