package agents

import (
	"strings"
	"testing"
)

func TestReviewValidate(t *testing.T) {
	ok := ReviewInput{Approach: "rebuild", Target: []ReviewTarget{{Area: "cloud", Option: "Microsoft Azure"}}, Reasons: []string{" + Core domain ", " "}}
	if err := ok.Validate(); err != nil {
		t.Fatal(err)
	}
	if len(ok.Reasons) != 1 || ok.Reasons[0] != "+ Core domain" {
		t.Fatalf("reasons not trimmed: %q", ok.Reasons)
	}
	long := ReviewInput{Approach: "rehost", Checks: []string{strings.Repeat("x", 900)}}
	if err := long.Validate(); err != nil || len([]rune(long.Checks[0])) != ReviewMaxText {
		t.Fatalf("long text not cut: %v", err)
	}
	for name, in := range map[string]ReviewInput{
		"no approach":      {},
		"unknown approach": {Approach: "teleport"},
		"unverified claim": {Approach: "rehost", Claims: []SuggestClaim{{Text: "x", Verdict: "open"}}},
		"too many reasons": {Approach: "rehost", Reasons: make([]string, ReviewMaxItems+1)},
	} {
		if err := in.Validate(); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestCleanReview(t *testing.T) {
	in := ReviewInput{Approach: "rebuild", Claims: []SuggestClaim{{Text: "a", Verdict: "confirmed"}}}
	out := cleanReview(ReviewOutput{
		Verdict:      "great",
		Challenges:   []ReviewChallenge{{Text: "Data migration", Severity: "huge", Claims: []int{1, 1, 5}}, {Text: " "}},
		Missing:      []string{"a", "", "b", "c", "d", "e", "f"},
		Alternatives: []ReviewAlternative{{Approach: "rebuild"}, {Approach: "teleport"}, {Approach: "refactor"}, {Approach: "refactor"}, {Approach: "replace"}, {Approach: "rehost"}},
	}, in)
	if out.Verdict != "questionable" {
		t.Fatal("unknown verdict kept")
	}
	if len(out.Challenges) != 1 || out.Challenges[0].Severity != "medium" || len(out.Challenges[0].Claims) != 1 {
		t.Fatalf("challenges %+v", out.Challenges)
	}
	if len(out.Missing) != 5 {
		t.Fatalf("missing %q", out.Missing)
	}
	if len(out.Alternatives) != 2 || out.Alternatives[0].Approach != "refactor" || out.Alternatives[1].Approach != "replace" {
		t.Fatalf("alternatives %+v", out.Alternatives)
	}
}
