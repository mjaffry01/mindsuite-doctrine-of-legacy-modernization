package agents

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// Approach suggestions (README "Verify before deciding", principle 5): the AI suggests and a person decides.
// The model sees only claims that people confirmed or corrected, and the context's strategic value and criticality.
// Names, comments and unverified findings are never part of the input.

// Approaches the model may suggest (the web app's APPROACH list, without "undecided").
var Approaches = []string{"rehost", "replatform", "refactor", "rebuild", "replace", "encapsulate", "retire", "build", "buy", "partner"}

// ModeApproaches are the approaches each project type uses (README "Three kinds of project"):
// the 7 Rs for an existing system, build / buy / partner for new capabilities, both when extending.
var ModeApproaches = map[string][]string{
	"modernize": {"rehost", "replatform", "refactor", "rebuild", "replace", "encapsulate", "retire"},
	"new":       {"build", "buy", "partner"},
	"extend":    {"rehost", "replatform", "refactor", "rebuild", "replace", "encapsulate", "retire", "build", "buy", "partner"},
}

// Limits on what one suggestion request may carry.
const (
	SuggestMaxClaims = 300
	SuggestMaxText   = 600
)

// SuggestClaim is one verified claim. Correction is the person's right version of a corrected claim.
type SuggestClaim struct {
	Kind       string `json:"kind"`
	Role       string `json:"role"`
	Text       string `json:"text"`
	Verdict    string `json:"verdict"`
	Correction string `json:"correction"`
}

// SuggestInput is everything the model sees.
type SuggestInput struct {
	Strategic   string         `json:"strategic"`
	Criticality string         `json:"criticality"`
	Claims      []SuggestClaim `json:"claims"`
}

// SuggestReason is one reason, with the numbers (1-based) of the claims it rests on.
type SuggestReason struct {
	Text   string `json:"text"`
	Claims []int  `json:"claims"`
}

// Suggestion is one approach the model suggests.
type Suggestion struct {
	Approach string          `json:"approach"`
	Fit      string          `json:"fit"`
	Reasons  []SuggestReason `json:"reasons"`
	Changes  string          `json:"changes"`
}

// SuggestOutput is the model's answer.
type SuggestOutput struct {
	Summary     string       `json:"summary"`
	Suggestions []Suggestion `json:"suggestions"`
	Model       string       `json:"model"`
}

// Validate checks and trims the input. It refuses anything that is not a verified claim.
func (in *SuggestInput) Validate() error {
	if len(in.Claims) == 0 {
		return errors.New("no verified claims: confirm or correct claims with people first")
	}
	if len(in.Claims) > SuggestMaxClaims {
		return fmt.Errorf("too many claims: at most %d", SuggestMaxClaims)
	}
	cut := func(s string) string {
		s = strings.TrimSpace(s)
		if r := []rune(s); len(r) > SuggestMaxText {
			s = string(r[:SuggestMaxText])
		}
		return s
	}
	for i := range in.Claims {
		c := &in.Claims[i]
		if c.Verdict != "confirmed" && c.Verdict != "corrected" {
			return fmt.Errorf("claim %d is not confirmed or corrected", i+1)
		}
		c.Kind, c.Role, c.Text, c.Correction = cut(c.Kind), cut(c.Role), cut(c.Text), cut(c.Correction)
		if c.Text == "" {
			return fmt.Errorf("claim %d has no text", i+1)
		}
		if c.Verdict == "corrected" && c.Correction == "" {
			return fmt.Errorf("claim %d is corrected but has no right version", i+1)
		}
	}
	in.Strategic, in.Criticality = cut(in.Strategic), cut(in.Criticality)
	return nil
}

const suggestSystem = `You advise on modernizing legacy systems under the MindSuite doctrine.
You receive claims about one part of a legacy system. The people who know that part have confirmed each claim, or corrected it; for a corrected claim, the correction is the true version and the original text is wrong where they differ.
Suggest which modernization approaches fit, chosen from: rehost, replatform, refactor, rebuild, replace, encapsulate, retire.
- Use only the claims and the stated strategic value and criticality. Do not assume facts that are not given; if something important is unknown, say so in the reasons.
- Rules of thumb: a core domain is worth custom work (rebuild or refactor); a supporting subdomain usually moves to managed platforms with little change (replatform); a generic subdomain is usually bought (replace); a part nobody uses or that duplicates another can be retired; a stable core that new systems need can be wrapped with APIs (encapsulate); rehost is for urgent infrastructure moves with no business change.
- Give one to four approaches, best first. Fit is strong, possible or weak.
- Every reason cites the numbers of the claims it rests on. A reason based only on the strategic value or criticality cites no claims.
- For each approach say, in one sentence, what would change the suggestion.
- The summary is one or two sentences. A person makes the decision; you only suggest.
Write plain, short sentences.`

// SuggestApproaches asks the model for approach suggestions and drops anything that does not fit the input.
func (l *LLM) SuggestApproaches(ctx context.Context, in SuggestInput) (SuggestOutput, error) {
	var b strings.Builder
	fmt.Fprintf(&b, "Strategic value: %s\nBusiness criticality: %s\n\nVerified claims:\n", orUnknown(in.Strategic), orUnknown(in.Criticality))
	for i, c := range in.Claims {
		fmt.Fprintf(&b, "%d. [%s, %s] %s\n", i+1, c.Kind, c.Verdict, c.Text)
		if c.Verdict == "corrected" {
			fmt.Fprintf(&b, "   Correction (true version): %s\n", c.Correction)
		}
	}
	schema := obj(map[string]any{
		"summary": str(),
		"suggestions": arr(obj(map[string]any{
			"approach": enum(Approaches...),
			"fit":      enum("strong", "possible", "weak"),
			"reasons":  arr(obj(map[string]any{"text": str(), "claims": arr(integer())})),
			"changes":  str(),
		})),
	})
	var out SuggestOutput
	if err := l.JSON(ctx, ModelStrong, suggestSystem, b.String(), schema, 4000, &out); err != nil {
		return SuggestOutput{}, err
	}
	out.Model = Resolve(ModelStrong)
	return cleanSuggestions(out, len(in.Claims)), nil
}

// cleanSuggestions keeps known approaches once each, at most four, and only claim numbers that exist.
func cleanSuggestions(out SuggestOutput, n int) SuggestOutput {
	known := map[string]bool{}
	for _, a := range Approaches {
		known[a] = true
	}
	seen := map[string]bool{}
	var keep []Suggestion
	for _, s := range out.Suggestions {
		if !known[s.Approach] || seen[s.Approach] || len(keep) == 4 {
			continue
		}
		seen[s.Approach] = true
		if s.Fit != "strong" && s.Fit != "possible" && s.Fit != "weak" {
			s.Fit = "possible"
		}
		var rs []SuggestReason
		for _, r := range s.Reasons {
			if strings.TrimSpace(r.Text) == "" {
				continue
			}
			var cs []int
			used := map[int]bool{}
			for _, c := range r.Claims {
				if c >= 1 && c <= n && !used[c] {
					used[c] = true
					cs = append(cs, c)
				}
			}
			rs = append(rs, SuggestReason{Text: r.Text, Claims: cs})
		}
		s.Reasons = rs
		keep = append(keep, s)
	}
	out.Suggestions = keep
	return out
}

func orUnknown(s string) string {
	if s == "" {
		return "not recorded"
	}
	return s
}
