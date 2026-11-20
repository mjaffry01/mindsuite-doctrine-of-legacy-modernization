package agents

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// Decision review (decision advisor, Phase C): Claude challenges a target the person is about to record.
// It sees the approach, the chosen technology (option labels), the advisor's reasons and checks, the names
// of inputs that are still missing, and claims people confirmed or corrected. It never sees the rationale,
// names, comments, unverified claims or evidence quotes. Claude challenges; the person decides.

// Limits on one review request.
const (
	ReviewMaxItems = 40
	ReviewMaxText  = 300
)

// ReviewTarget is one technology choice: the area (architecture, language, …) and the option label.
type ReviewTarget struct {
	Area   string `json:"area"`
	Option string `json:"option"`
}

// ReviewInput is everything the model sees.
type ReviewInput struct {
	Strategic   string         `json:"strategic"`
	Criticality string         `json:"criticality"`
	Approach    string         `json:"approach"`
	Target      []ReviewTarget `json:"target"`
	Reasons     []string       `json:"reasons"`
	Checks      []string       `json:"checks"`
	Missing     []string       `json:"missing"`
	Claims      []SuggestClaim `json:"claims"`
}

// ReviewChallenge is one weakness, with the claims it rests on (1-based).
type ReviewChallenge struct {
	Text     string `json:"text"`
	Severity string `json:"severity"`
	Claims   []int  `json:"claims"`
}

// ReviewAlternative is an approach the model thinks deserves a second look.
type ReviewAlternative struct {
	Approach string `json:"approach"`
	Why      string `json:"why"`
}

// ReviewOutput is the model's answer.
type ReviewOutput struct {
	Verdict      string              `json:"verdict"`
	Summary      string              `json:"summary"`
	Challenges   []ReviewChallenge   `json:"challenges"`
	Missing      []string            `json:"missing"`
	Questions    []string            `json:"questions"`
	Alternatives []ReviewAlternative `json:"alternatives"`
	Model        string              `json:"model"`
}

func reviewCut(s string, n int) string {
	s = strings.TrimSpace(s)
	if r := []rune(s); len(r) > n {
		s = string(r[:n])
	}
	return s
}

// Validate checks and trims the input.
func (in *ReviewInput) Validate() error {
	known := false
	for _, a := range Approaches {
		if a == in.Approach {
			known = true
		}
	}
	if !known {
		return errors.New("choose an approach before asking for a review")
	}
	if len(in.Target) > 12 || len(in.Reasons) > ReviewMaxItems || len(in.Checks) > ReviewMaxItems || len(in.Missing) > 12 {
		return errors.New("too many items in the request")
	}
	in.Strategic, in.Criticality = reviewCut(in.Strategic, 40), reviewCut(in.Criticality, 40)
	for i := range in.Target {
		in.Target[i].Area, in.Target[i].Option = reviewCut(in.Target[i].Area, 40), reviewCut(in.Target[i].Option, 120)
	}
	for _, l := range []*[]string{&in.Reasons, &in.Checks, &in.Missing} {
		var keep []string
		for _, s := range *l {
			if s = reviewCut(s, ReviewMaxText); s != "" {
				keep = append(keep, s)
			}
		}
		*l = keep
	}
	if len(in.Claims) > 0 {
		claims := SuggestInput{Claims: in.Claims}
		if err := claims.Validate(); err != nil {
			return err
		}
		in.Claims = claims.Claims
	}
	return nil
}

const reviewSystem = `You review a modernization decision for one part of a legacy system, under the MindSuite doctrine.
A rule-based advisor proposed the target below and a person is about to record it. Your job is to challenge it honestly, like an experienced architect on a review board.
- Use only what is given: strategic value, criticality, approach, target technology, the advisor's reasons and checks, inputs still missing, and claims people confirmed or corrected (for a corrected claim the correction is the truth).
- Do not invent facts. If a conclusion depends on something not given, say what is missing instead.
- Look for: an approach that does not fit the strategic value or criticality; technology that does not fit together; risks the checks missed (data migration, cut-over, skills, testing, vendor lock-in, licences, regulation); reasons that are weak or rest on missing inputs.
- Verdict: "sound" if you see no serious problem, "questionable" if important doubts remain, "risky" if the target is likely to fail or cost far more than expected.
- Give at most six challenges, most serious first, each with severity high, medium or low, citing claim numbers when it rests on claims.
- List the most important missing evidence (at most five) and the questions to ask the people who know the system (at most five).
- Name at most two alternative approaches worth a second look, chosen from: rehost, replatform, refactor, rebuild, replace, encapsulate, retire; none if the chosen one is right.
- Summary: one or two sentences. A person makes the decision; you only challenge it.
Write plain, short sentences.`

// ReviewDecision asks the model to challenge a decision and drops anything that does not fit the input.
func (l *LLM) ReviewDecision(ctx context.Context, in ReviewInput) (ReviewOutput, error) {
	var b strings.Builder
	fmt.Fprintf(&b, "Strategic value: %s\nBusiness criticality: %s\nApproach: %s\n\nTarget technology:\n", orUnknown(in.Strategic), orUnknown(in.Criticality), in.Approach)
	if len(in.Target) == 0 {
		b.WriteString("- none chosen\n")
	}
	for _, t := range in.Target {
		fmt.Fprintf(&b, "- %s: %s\n", t.Area, t.Option)
	}
	list := func(title string, items []string) {
		fmt.Fprintf(&b, "\n%s:\n", title)
		if len(items) == 0 {
			b.WriteString("- none\n")
		}
		for _, s := range items {
			fmt.Fprintf(&b, "- %s\n", s)
		}
	}
	list("Advisor reasons (+ for, − against)", in.Reasons)
	list("Advisor checks on this target", in.Checks)
	list("Inputs still missing", in.Missing)
	b.WriteString("\nVerified claims:\n")
	if len(in.Claims) == 0 {
		b.WriteString("- none yet\n")
	}
	for i, c := range in.Claims {
		fmt.Fprintf(&b, "%d. [%s, %s] %s\n", i+1, c.Kind, c.Verdict, c.Text)
		if c.Verdict == "corrected" {
			fmt.Fprintf(&b, "   Correction (true version): %s\n", c.Correction)
		}
	}
	schema := obj(map[string]any{
		"verdict":      enum("sound", "questionable", "risky"),
		"summary":      str(),
		"challenges":   arr(obj(map[string]any{"text": str(), "severity": enum("high", "medium", "low"), "claims": arr(integer())})),
		"missing":      arr(str()),
		"questions":    arr(str()),
		"alternatives": arr(obj(map[string]any{"approach": enum(Approaches...), "why": str()})),
	})
	var out ReviewOutput
	if err := l.JSON(ctx, ModelStrong, reviewSystem, b.String(), schema, 4000, &out); err != nil {
		return ReviewOutput{}, err
	}
	out.Model = Resolve(ModelStrong)
	return cleanReview(out, in), nil
}

// cleanReview keeps known values, the stated limits, and only claim numbers that exist.
func cleanReview(out ReviewOutput, in ReviewInput) ReviewOutput {
	if out.Verdict != "sound" && out.Verdict != "questionable" && out.Verdict != "risky" {
		out.Verdict = "questionable"
	}
	n := len(in.Claims)
	var ch []ReviewChallenge
	for _, c := range out.Challenges {
		if strings.TrimSpace(c.Text) == "" || len(ch) == 6 {
			continue
		}
		if c.Severity != "high" && c.Severity != "medium" && c.Severity != "low" {
			c.Severity = "medium"
		}
		var cs []int
		used := map[int]bool{}
		for _, k := range c.Claims {
			if k >= 1 && k <= n && !used[k] {
				used[k] = true
				cs = append(cs, k)
			}
		}
		c.Claims = cs
		ch = append(ch, c)
	}
	out.Challenges = ch
	trimList := func(l []string, max int) []string {
		var keep []string
		for _, s := range l {
			if s = strings.TrimSpace(s); s != "" && len(keep) < max {
				keep = append(keep, s)
			}
		}
		return keep
	}
	out.Missing, out.Questions = trimList(out.Missing, 5), trimList(out.Questions, 5)
	known := map[string]bool{}
	for _, a := range Approaches {
		known[a] = true
	}
	var alts []ReviewAlternative
	seen := map[string]bool{}
	for _, a := range out.Alternatives {
		if !known[a.Approach] || a.Approach == in.Approach || seen[a.Approach] || len(alts) == 2 {
			continue
		}
		seen[a.Approach] = true
		alts = append(alts, a)
	}
	out.Alternatives = alts
	return out
}
