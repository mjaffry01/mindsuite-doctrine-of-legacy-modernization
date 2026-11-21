package agents

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
)

// Data policy agent: reads the design summary (platform decision, quality and recovery targets, the
// database, cache, messaging and backup layers), the business objects the product keeps (with the systems
// that write them) and the critical journeys with their acceptance measures, and drafts a data policy and a
// caching policy per business object, plus conflicts and questions for the data owner.
// Input is design metadata only: no business records, no names of people, no secrets.
// Checks: every rule names an object that was sent, enums are valid, lengths are capped, and nothing names a
// product or vendor the design does not already mention; one retry when no data rule survives.

var (
	PolicyConsistency = []string{"strong", "eventual", "mixed"}
	PolicyCacheable   = []string{"no", "short", "yes"}
)

const (
	PolicyMaxObjects  = 30
	PolicyMaxJourneys = 30
)

type PolicyObject struct {
	Name        string   `json:"name"`
	Source      string   `json:"source"`
	Writers     []string `json:"writers"`
	Sensitivity string   `json:"sensitivity"`
	Consistency string   `json:"consistency"`
}
type PolicyJourney struct {
	Journey string `json:"journey"`
	Failure string `json:"failure"`
	Measure string `json:"measure"`
}
type DataPolicyInput struct {
	Project  string          `json:"project"`
	Kind     string          `json:"kind"`
	Platform []string        `json:"platform"`
	Targets  []string        `json:"targets"`
	Layers   []DiagramLayer  `json:"layers"`
	Objects  []PolicyObject  `json:"objects"`
	Journeys []PolicyJourney `json:"journeys"`
}
type DataRule struct {
	Object              string `json:"object"`
	Owner               string `json:"owner"`
	AuthoritativeWriter string `json:"authoritativeWriter"`
	Consistency         string `json:"consistency"`
	Retention           string `json:"retention"`
	Classification      string `json:"classification"`
	Backup              string `json:"backup"`
	Access              string `json:"access"`
	Why                 string `json:"why"`
}
type CacheRule struct {
	Object         string `json:"object"`
	Cacheable      string `json:"cacheable"`
	MaxStaleness   string `json:"maxStaleness"`
	Invalidation   string `json:"invalidation"`
	OnCacheFailure string `json:"onCacheFailure"`
	PersonalData   string `json:"personalData"`
	Why            string `json:"why"`
}
type DataPolicyOutput struct {
	Summary   string      `json:"summary"`
	Data      []DataRule  `json:"data"`
	Cache     []CacheRule `json:"cache"`
	Conflicts []string    `json:"conflicts"`
	Questions []string    `json:"questions"`
	Dropped   int         `json:"dropped"`
	DropNotes []string    `json:"dropNotes"`
	Model     string      `json:"model"`
	Attempts  int         `json:"attempts,omitempty"`
}

func (in DataPolicyInput) Validate() error {
	if in.Kind != "new" && in.Kind != "modernize" && in.Kind != "extend" {
		return errors.New("unknown project kind")
	}
	if tooLong(in.Project, 180) || len(in.Platform) > 12 || len(in.Targets) > 16 || len(in.Layers) > 12 {
		return errors.New("design summary too large")
	}
	if len(in.Objects) == 0 || len(in.Objects) > PolicyMaxObjects {
		return errors.New("send between 1 and 30 business objects")
	}
	if len(in.Journeys) > PolicyMaxJourneys {
		return errors.New("too many journeys")
	}
	for _, list := range [][]string{in.Platform, in.Targets} {
		for _, s := range list {
			if strings.TrimSpace(s) == "" || tooLong(s, 300) {
				return errors.New("invalid summary line")
			}
		}
	}
	for _, l := range in.Layers {
		if l.ID == "" || tooLong(l.ID, 40) || tooLong(l.Label, 80) || strings.TrimSpace(l.Design) == "" || tooLong(l.Design, 1500) {
			return errors.New("invalid plan layer")
		}
	}
	seen := map[string]bool{}
	for _, o := range in.Objects {
		k := strings.ToLower(strings.TrimSpace(o.Name))
		if k == "" || tooLong(o.Name, 60) || seen[k] || tooLong(o.Source, 120) || tooLong(o.Sensitivity, 40) || tooLong(o.Consistency, 40) || len(o.Writers) > 12 {
			return errors.New("invalid business object")
		}
		seen[k] = true
		for _, w := range o.Writers {
			if strings.TrimSpace(w) == "" || tooLong(w, 120) {
				return errors.New("invalid writer")
			}
		}
	}
	for _, j := range in.Journeys {
		if strings.TrimSpace(j.Journey) == "" || tooLong(j.Journey, 200) || tooLong(j.Failure, 400) || tooLong(j.Measure, 400) {
			return errors.New("invalid journey")
		}
	}
	return nil
}

// stated is all text the owner's design already contains (products may only be named if they appear here).
func (in DataPolicyInput) stated() string {
	var b strings.Builder
	b.WriteString(in.Project)
	for _, s := range append(append([]string{}, in.Platform...), in.Targets...) {
		b.WriteString(" " + s)
	}
	for _, l := range in.Layers {
		b.WriteString(" " + l.Label + " " + l.Design)
	}
	for _, o := range in.Objects {
		b.WriteString(" " + o.Name + " " + o.Source + " " + strings.Join(o.Writers, " "))
	}
	for _, j := range in.Journeys {
		b.WriteString(" " + j.Journey + " " + j.Failure + " " + j.Measure)
	}
	return b.String()
}

func (in DataPolicyInput) objectName(s string) (string, bool) {
	s = strings.TrimSpace(s)
	for _, o := range in.Objects {
		if strings.EqualFold(strings.TrimSpace(o.Name), s) {
			return strings.TrimSpace(o.Name), true
		}
	}
	return "", false
}

func capText(s string, n int) string {
	s = strings.TrimSpace(s)
	if tooLong(s, n) {
		return string([]rune(s)[:n])
	}
	return s
}

func (in DataPolicyInput) verify(out DataPolicyOutput) DataPolicyOutput {
	guard := BriefFillInput{Project: in.Project, Answers: map[string]string{"x": in.stated()}}
	clean := DataPolicyOutput{Summary: capText(out.Summary, 600), Data: []DataRule{}, Cache: []CacheRule{}, Conflicts: []string{}, Questions: []string{}, DropNotes: []string{}}
	drop := func(what, why string) { clean.Dropped++; clean.DropNotes = addDropNote(clean.DropNotes, what+": "+why) }
	seen := map[string]bool{}
	for _, d := range out.Data {
		name, ok := in.objectName(d.Object)
		switch {
		case !ok:
			drop("data rule for "+quoteName(d.Object), "this business object was not sent")
			continue
		case seen[name]:
			drop("data rule for "+name, "a second rule for the same object")
			continue
		case !slices.Contains(PolicyConsistency, d.Consistency):
			drop("data rule for "+name, "consistency "+quoteName(d.Consistency)+" is not strong, eventual or mixed")
			continue
		}
		d.Object = name
		d.Owner, d.AuthoritativeWriter, d.Retention, d.Classification = capText(d.Owner, 120), capText(d.AuthoritativeWriter, 160), capText(d.Retention, 200), capText(d.Classification, 200)
		d.Backup, d.Access, d.Why = capText(d.Backup, 200), capText(d.Access, 240), capText(d.Why, 300)
		if d.Owner == "" || d.AuthoritativeWriter == "" {
			drop("data rule for "+name, "no owner or no authoritative writer")
			continue
		}
		if prod := guard.unstatedProduct(d.AuthoritativeWriter + " " + d.Backup + " " + d.Access + " " + d.Retention); prod != "" {
			drop("data rule for "+name, "names "+prod+", which the design does not mention")
			continue
		}
		seen[name] = true
		clean.Data = append(clean.Data, d)
	}
	seenC := map[string]bool{}
	for _, c := range out.Cache {
		name, ok := in.objectName(c.Object)
		switch {
		case !ok:
			drop("cache rule for "+quoteName(c.Object), "this business object was not sent")
			continue
		case seenC[name]:
			drop("cache rule for "+name, "a second rule for the same object")
			continue
		case !slices.Contains(PolicyCacheable, c.Cacheable):
			drop("cache rule for "+name, "cacheable "+quoteName(c.Cacheable)+" is not no, short or yes")
			continue
		}
		c.Object = name
		c.MaxStaleness, c.Invalidation, c.OnCacheFailure = capText(c.MaxStaleness, 120), capText(c.Invalidation, 240), capText(c.OnCacheFailure, 240)
		c.PersonalData, c.Why = capText(c.PersonalData, 200), capText(c.Why, 300)
		if prod := guard.unstatedProduct(c.Invalidation + " " + c.OnCacheFailure + " " + c.MaxStaleness); prod != "" {
			drop("cache rule for "+name, "names "+prod+", which the design does not mention")
			continue
		}
		seenC[name] = true
		clean.Cache = append(clean.Cache, c)
	}
	keepLines(guard, out.Conflicts, &clean.Conflicts, "conflict", drop)
	keepLines(guard, out.Questions, &clean.Questions, "question", drop)
	return clean
}

const dataPolicyPrompt = `You are the MindSuite Data policy agent, a senior data architect. You receive a design summary as quoted data: the platform decision, quality and recovery targets (availability, RTO, RPO, regulations, residency), the database, cache, messaging and backup layers, the business objects the product keeps (with the systems that write them and any sensitivity or consistency hints), and the critical journeys with their failure scenarios and acceptance measures. Draft a data policy and a caching policy for each business object.
Rules:
- One "data" rule and one "cache" rule per business object, using the object name exactly as sent. Never add objects that were not sent.
- owner: a role or team (for example "Head of operations", "Ordering team"), never a person's name. authoritativeWriter: the one system or service allowed to write the object; if several writers were sent, say which one is authoritative and how the others change it (through its API or events). For a modernization, say which system writes it while both systems are live.
- consistency: strong when money, stock, bookings or anything that must never be oversold or double-counted; eventual when brief staleness is harmless; mixed when part of it is strong.
- retention, classification (including personal or financial data and residency), backup (tie to the RPO and RTO when given; say "not decided" when they are missing) and access (who may read and change it, least privilege).
- cache: cacheable "no" for data that must be exact at the moment of a decision (stock at checkout, balances, payment status); "short" with a maximum staleness in seconds or minutes; "yes" for reference data. Say how entries are invalidated, what the product does when the cache is down (read from the database, degrade, or refuse) and whether personal data may be cached.
- Use only products and services named in the summary; never name another product, vendor or cloud service.
- "conflicts": anything that cannot be true together, for example a journey that requires exactly one order while stock is cached for minutes, two authoritative writers, or an RPO the backup layer cannot meet.
- "questions": what the data owner must decide that the summary does not say.
- "summary": two or three sentences a data architect would say about this design.
Keep every field to one or two short sentences; give at most 8 conflicts and 10 questions.`

const dataPolicyRetry = " A first attempt returned no valid data rule. Use exactly the business object names sent and the allowed values for consistency and cacheable."

func (l *LLM) DataPolicy(ctx context.Context, in DataPolicyInput) (DataPolicyOutput, error) {
	return RunDataPolicy(ctx, in, l.JSON)
}

// RunDataPolicy asks the model, verifies the answer and asks once more when no data rule survives.
func RunDataPolicy(ctx context.Context, in DataPolicyInput, call JSONCaller) (DataPolicyOutput, error) {
	data, err := json.Marshal(struct {
		Design DataPolicyInput `json:"design"`
	}{in})
	if err != nil {
		return DataPolicyOutput{}, err
	}
	schema := obj(map[string]any{
		"summary": str(),
		"data": arr(obj(map[string]any{"object": str(), "owner": str(), "authoritativeWriter": str(), "consistency": enum(PolicyConsistency...),
			"retention": str(), "classification": str(), "backup": str(), "access": str(), "why": str()})),
		"cache": arr(obj(map[string]any{"object": str(), "cacheable": enum(PolicyCacheable...), "maxStaleness": str(), "invalidation": str(),
			"onCacheFailure": str(), "personalData": str(), "why": str()})),
		"conflicts": arr(str()),
		"questions": arr(str()),
	})
	var out DataPolicyOutput
	for attempt := 1; attempt <= 2; attempt++ {
		var raw DataPolicyOutput
		p := dataPolicyPrompt
		if attempt == 2 {
			p += dataPolicyRetry
		}
		if err = call(ctx, ModelStrong, p, string(data), schema, 16000, &raw); err != nil {
			return DataPolicyOutput{}, err
		}
		out = in.verify(raw)
		out.Attempts = attempt
		if len(out.Data) >= 1 {
			break
		}
	}
	out.Model = Resolve(ModelStrong)
	return out, nil
}

// unstatedProduct returns the first catalogue product named in value that the design does not mention, or "".
func (in BriefFillInput) unstatedProduct(value string) string {
	if !in.namesUnstatedProduct(value) {
		return ""
	}
	stated := " " + normName(strings.ToLower(in.Project)+" "+strings.Join(mapValues(in.Answers), " ")) + " "
	low := " " + normName(value) + " "
	best := ""
	for _, name := range productCatalogue {
		n := normName(name)
		if len(n) >= 4 && len(n) > len(normName(best)) && strings.Contains(low, " "+n+" ") && !strings.Contains(stated, " "+n+" ") {
			best = name // the most specific name wins ("azure functions" over "azure")
		}
	}
	if best == "" {
		return "a product"
	}
	return best
}

func mapValues(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for _, v := range m {
		out = append(out, strings.ToLower(v))
	}
	return out
}

func quoteName(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "(blank)"
	}
	return "“" + capText(s, 40) + "”"
}

// addDropNote keeps at most 20 short reasons (the reason, never the dropped text itself).
func addDropNote(notes []string, note string) []string {
	if len(notes) >= 20 {
		return notes
	}
	return append(notes, capText(note, 200))
}

// keepLines keeps short conflict or question lines and records why the others were dropped.
func keepLines(guard BriefFillInput, in []string, out *[]string, kind string, drop func(what, why string)) {
	for i, s := range in {
		s = strings.TrimSpace(s)
		switch {
		case s == "":
			continue
		case tooLong(s, 300):
			drop(fmt.Sprintf("%s %d", kind, i+1), "longer than 300 characters")
		case len(*out) >= 20:
			drop(fmt.Sprintf("%s %d", kind, i+1), "more than 20 "+kind+"s")
		default:
			if prod := guard.unstatedProduct(s); prod != "" {
				drop(fmt.Sprintf("%s %d", kind, i+1), "names "+prod+", which the design does not mention")
				continue
			}
			*out = append(*out, s)
		}
	}
}
