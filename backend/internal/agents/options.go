package agents

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// Target options team: AI agents propose technical options for one context from the discovery facts.
//
//	1 Evidence brief    (in the web app, no model)  numbered facts F1…Fn, metadata only
//	2 Option scouts     (balanced model, parallel)  3–5 options per area, every reason cites facts
//	3 Verifier          (rules)                     drops uncited reasons and options; checks product names
//	                                                against a curated catalogue; unknown names "need confirmation"
//	4 Rules guard       (rules)                     applies the organization's technology rules to each option
//	5 Architect         (strong model)              2–4 consistent blueprints with a migration path and estimates
//	6 Challenger        (strong model)              challenges each blueprint
//
// The person deciding accepts options and records decisions; nothing here decides.
// Input is facts and rules only: no business data, names of people, comments or unverified claims.

// Option areas a scout can cover.
var OptionAreas = []string{"hosting", "architecture", "data", "integration", "ui", "products", "legacy"}

// Limits on one options request.
const (
	OptionsMaxFacts = 150
	OptionsMaxRules = 80
	OptionsMaxText  = 300
)

// OptionFact is one numbered discovery fact, e.g. {ID:"F3", Area:"data", Text:"Oracle 12c, 340 tables"}.
type OptionFact struct {
	ID   string `json:"id"`
	Area string `json:"area"`
	Text string `json:"text"`
}

// OptionRule is one organization technology rule.
type OptionRule struct {
	Area   string `json:"area"`
	Option string `json:"option"`
	State  string `json:"state"`
	Reason string `json:"reason"`
}

// OptionsInput is everything the team sees.
type OptionsInput struct {
	Mode        string       `json:"mode"`
	Context     string       `json:"context"`
	Strategic   string       `json:"strategic"`
	Criticality string       `json:"criticality"`
	Approach    string       `json:"approach"`
	Areas       []string     `json:"areas"`
	Facts       []OptionFact `json:"facts"`
	Rules       []OptionRule `json:"rules"`
	// Explore, when set, asks for the concrete choices inside one option (for example the Azure services,
	// tiers and configurations for "Microsoft Azure") instead of a full team run.
	Explore *ExploreTarget `json:"explore,omitempty"`
	// Design, when set, asks the solution designer to detail one blueprint.
	Design *DesignTarget `json:"design,omitempty"`
}

// ExploreTarget is the option to look inside.
type ExploreTarget struct {
	Area    string `json:"area"`
	Name    string `json:"name"`
	Vendor  string `json:"vendor"`
	Summary string `json:"summary"`
}

// OptionReason is one reason with the facts it rests on.
type OptionReason struct {
	Text  string   `json:"text"`
	Facts []string `json:"facts"`
}

// TechOption is one proposed option after verification and the rules guard.
type TechOption struct {
	ID            string         `json:"id"`
	Area          string         `json:"area"`
	Name          string         `json:"name"`
	Vendor        string         `json:"vendor"`
	Kind          string         `json:"kind"`
	Summary       string         `json:"summary"`
	Fit           []OptionReason `json:"fit"`
	Prerequisites []string       `json:"prerequisites"`
	Risks         []string       `json:"risks"`
	Lifecycle     string         `json:"lifecycle"`
	Licence       string         `json:"licence"`
	Skills        []string       `json:"skills"`
	Catalogue     string         `json:"catalogue"` // "known", "needs confirmation" or "pattern"
	// Filled for options found by Explore.
	Parent      string       `json:"parent,omitempty"`
	Tier        string       `json:"tier,omitempty"`
	Sizing      string       `json:"sizing,omitempty"`
	Resilience  string       `json:"resilience,omitempty"`
	ChooseWhen  string       `json:"chooseWhen,omitempty"`
	AvoidWhen   string       `json:"avoidWhen,omitempty"`
	Compute     *ComputeSize `json:"compute,omitempty"`
	Spec        []SpecItem   `json:"spec,omitempty"`
	Guard       string       `json:"guard"` // "allowed", "preferred", "standard", "approval", "excluded"
	GuardReason string       `json:"guardReason"`
}

// SpecAspects are the rows of the architects' comparison matrix for explored choices.
var SpecAspects = []string{"sku", "os-runtime", "scaling", "network", "availability", "disaster-recovery", "deployment",
	"limits", "cost-model", "compliance", "operations", "lock-in"}

// SpecItem is one short, specific value for one aspect of an explored choice.
type SpecItem struct {
	Aspect string `json:"aspect"`
	Value  string `json:"value"`
}

// ComputeSize is a starting size for one explored choice. Basis says whether it comes from the capacity
// facts or is an assumption the people must confirm.
type ComputeSize struct {
	VCPU      string `json:"vcpu"`
	RAMGB     string `json:"ramGb"`
	StorageGB string `json:"storageGb"`
	Instances string `json:"instances"`
	Basis     string `json:"basis"` // "facts" or "assumption"
	Note      string `json:"note"`
}

// TrafficComponent is one component on the path into the system (northbound) or out of it (southbound).
type TrafficComponent struct {
	Name      string         `json:"name"`
	Role      string         `json:"role"`
	Why       []OptionReason `json:"why"`
	Catalogue string         `json:"catalogue"`
	Guard     string         `json:"guard"`
}

// Blueprint is one complete target built from the options: always one Safe, one Balanced, one Ambitious.
type Blueprint struct {
	Tier       string         `json:"tier"`
	Name       string         `json:"name"`
	Approach   string         `json:"approach"`
	Options    []string       `json:"options"`
	Why        []OptionReason `json:"why"`
	Migration  string         `json:"migration"`
	Effort     string         `json:"effort"`
	CostTrend  string         `json:"costTrend"`
	Deadline   string         `json:"deadline"`
	Verdict    string         `json:"verdict"`
	Challenges []string       `json:"challenges"`
}

// AgentStep is the telemetry of one agent.
type AgentStep struct {
	Agent    string `json:"agent"`
	Model    string `json:"model"`
	Requests int    `json:"requests"`
	Ms       int64  `json:"ms"`
	Note     string `json:"note"`
}

// OptionsOutput is the team's answer.
type OptionsOutput struct {
	Design  *SolutionDesign `json:"design,omitempty"`
	Options []TechOption    `json:"options"`
	// Filled by Explore: components for traffic coming in from users and partners (north) and going out
	// to databases, legacy systems and partners (south).
	North      []TrafficComponent `json:"north,omitempty"`
	South      []TrafficComponent `json:"south,omitempty"`
	Blueprints []Blueprint        `json:"blueprints"`
	Dropped    []string           `json:"dropped"`
	Steps      []AgentStep        `json:"steps"`
	Requests   int                `json:"requests"`
}

func optCut(s string, n int) string {
	s = strings.TrimSpace(s)
	if r := []rune(s); len(r) > n {
		s = string(r[:n])
	}
	return s
}

var factIDRe = regexp.MustCompile(`^F\d{1,3}$`)

// Validate checks and trims the input.
func (in *OptionsInput) Validate() error {
	if len(in.Facts) == 0 {
		return errors.New("no discovery facts: run discovery first")
	}
	if len(in.Facts) > OptionsMaxFacts || len(in.Rules) > OptionsMaxRules {
		return errors.New("too many facts or rules in the request")
	}
	known := map[string]bool{}
	for _, a := range OptionAreas {
		known[a] = true
	}
	var areas []string
	seen := map[string]bool{}
	for _, a := range in.Areas {
		if !known[a] {
			return fmt.Errorf("unknown option area %q", a)
		}
		if !seen[a] {
			seen[a] = true
			areas = append(areas, a)
		}
	}
	if len(areas) == 0 {
		return errors.New("choose at least one option area")
	}
	in.Areas = areas
	ids := map[string]bool{}
	for i := range in.Facts {
		f := &in.Facts[i]
		f.ID, f.Area, f.Text = strings.TrimSpace(f.ID), optCut(f.Area, 30), optCut(f.Text, OptionsMaxText)
		if !factIDRe.MatchString(f.ID) || ids[f.ID] {
			return fmt.Errorf("fact %d has a missing or repeated id", i+1)
		}
		ids[f.ID] = true
		if f.Text == "" {
			return fmt.Errorf("fact %s has no text", f.ID)
		}
	}
	for i := range in.Rules {
		r := &in.Rules[i]
		r.Area, r.Option, r.State, r.Reason = optCut(r.Area, 40), optCut(r.Option, 80), optCut(r.State, 20), optCut(r.Reason, 200)
		switch r.State {
		case "standard", "preferred", "approval", "retiring", "excluded":
		default:
			return fmt.Errorf("rule %d has an unknown state %q", i+1, r.State)
		}
	}
	in.Context, in.Strategic, in.Criticality = optCut(in.Context, 80), optCut(in.Strategic, 40), optCut(in.Criticality, 40)
	switch in.Mode {
	case "":
		in.Mode = "modernize"
	case "modernize", "new", "extend":
	default:
		return fmt.Errorf("unknown project type %q", in.Mode)
	}
	if in.Design != nil {
		if in.Explore != nil {
			return errors.New("ask for an exploration or a design, not both")
		}
		if err := in.Design.validate(); err != nil {
			return err
		}
	}
	if e := in.Explore; e != nil {
		if !known[e.Area] {
			return fmt.Errorf("unknown area to explore %q", e.Area)
		}
		e.Name, e.Vendor, e.Summary = optCut(e.Name, 80), optCut(e.Vendor, 60), optCut(e.Summary, 300)
		if e.Name == "" {
			return errors.New("name the option to explore")
		}
		in.Areas = []string{e.Area}
	}
	if in.Approach != "" {
		ok := false
		for _, a := range Approaches {
			ok = ok || a == in.Approach
		}
		if !ok {
			in.Approach = ""
		}
	}
	return nil
}

// JSONCaller is the model call the team uses; the LLM provides it, tests replace it.
type JSONCaller func(ctx context.Context, model, system, user string, schema map[string]any, maxTokens int64, out any) error

// TargetOptions runs the team with Claude.
func (l *LLM) TargetOptions(ctx context.Context, in OptionsInput) (OptionsOutput, error) {
	return RunOptionsTeam(ctx, in, l.JSON)
}

var areaBrief = map[string]string{
	"hosting":      "where it runs: cloud provider and the runtime service (containers, platform services, virtual machines, serverless, private or in-country cloud, staying on-premises)",
	"architecture": "how the application is structured and moved: modular monolith, strangler fig, services, event-driven, API facade, and concrete frameworks",
	"data":         "the target database engine and how the data moves: managed engines, migration and replication tools, change data capture, archiving of history",
	"integration":  "how it connects: API management, event streaming, message queues, integration platforms, managed file transfer, retiring an existing ESB",
	"ui":           "the user interface and channels: web front end, mobile, low-code, or modernizing a desktop or terminal client",
	"products":     "commercial products or SaaS that could replace this part, with what customization they would need",
	"legacy":       "specialist routes for the legacy runtime: mainframe or midrange rehosting and emulation, automated code conversion, refactoring tools",
}

const scoutSystem = `You are an option scout on a modernization team under the MindSuite doctrine.
You receive numbered facts from the discovery of one part of a system (an existing legacy system, a new product described by its product brief, or both), the organization's technology rules, and one area to cover.
Propose three to five concrete, currently available options for that area that fit these facts.
- Name real products or services as their vendors name them in 2026. Do not invent products. If you are not sure a product exists, do not propose it.
- Every fit reason cites the fact ids it rests on, for example ["F3","F7"]. A reason you cannot tie to a fact must not be given.
- Respect the rules: do not propose excluded or retiring technology; prefer standard and preferred technology when it fits.
- Respect the deployment facts: where it must run (public, private, on-premises, hybrid, edge), the tenancy, the channels and any data residency. Do not propose hosting that contradicts them.
- Kind is product, cloud-service, framework or pattern. A pattern (for example strangler fig, modular monolith) is a way of working, not a product: name it plainly and put any product it uses in the summary.
- Lifecycle: the support status as you know it (for example "generally available, supported", "extended support until 2029"). Say "confirm with the vendor" when unsure.
- Licence is one of: open-source, subscription, cloud-service, perpetual, unknown.
- Keep each text short and plain. You only propose; people decide.`

const architectSystem = `You are the architect on a modernization team under the MindSuite doctrine.
You receive numbered facts about one part of a legacy system, the organization's technology rules, and options the scouts proposed (ids O1, O2, …), already checked against the rules.
Build exactly three complete target blueprints, each a set of options that work together:
- tier "Safe": least change and risk;
- tier "Balanced": the best overall fit for these facts;
- tier "Ambitious": the most long-term benefit, accepting more change.
- Use only the given option ids. Never use an option marked excluded. One blueprint may use an option marked approval, and must say so in the reasons.
- The approach must be one of those listed for the project type: for an existing system rehost, replatform, refactor, rebuild, replace, encapsulate or retire; for a new product build, buy or partner (partner means extending a platform, low-code product or partner solution); when modernizing and extending, either set, choosing build, buy or partner for what is new.
- Every reason cites fact ids. Migration is one or two sentences on how to get there safely (for example strangler fig, change data capture, parallel run).
- Effort is S, M, L or XL. Cost trend compared with today is lower, similar, higher or unknown. Deadline says in one sentence whether it fits the deadline in the facts, or "no deadline given".
- Give each a short plain name, for example "Replatform on managed Azure services".
- Make the three tiers genuinely different routes where the facts allow, not three variants of one. For a new product, consider buy or partner for the Safe tier when a suitable product or platform exists among the options.
Write plain, short sentences.`

const challengerSystem = `You challenge modernization blueprints for one part of a legacy system, like an experienced reviewer.
You receive the facts, the options and the blueprints. For each blueprint give a verdict (sound, questionable or risky) and at most three short challenges: what could go wrong or what is missing. Use only the given facts; when a conclusion needs a missing fact, say which. Write plain, short sentences.`

// RunOptionsTeam runs scouts, verifier, rules guard, architect and challenger.
func RunOptionsTeam(ctx context.Context, in OptionsInput, call JSONCaller) (OptionsOutput, error) {
	var out OptionsOutput
	var brief strings.Builder
	fmt.Fprintf(&brief, "Project type: %s\nPart of the system: %s\nStrategic value: %s\nBusiness criticality: %s\nApproach being considered: %s\n\nFacts:\n",
		modeText[in.Mode], orUnknown(in.Context), orUnknown(in.Strategic), orUnknown(in.Criticality), orUnknown(in.Approach))
	facts := map[string]bool{}
	for _, f := range in.Facts {
		facts[f.ID] = true
		fmt.Fprintf(&brief, "%s [%s] %s\n", f.ID, f.Area, f.Text)
	}
	brief.WriteString("\nOrganization technology rules:\n")
	if len(in.Rules) == 0 {
		brief.WriteString("- none: everything is allowed\n")
	}
	for _, r := range in.Rules {
		fmt.Fprintf(&brief, "- %s %s: %s%s\n", r.Area, r.Option, r.State, map[bool]string{true: " (" + r.Reason + ")", false: ""}[r.Reason != ""])
	}

	if in.Explore != nil {
		return runExplore(ctx, in, brief.String(), facts, call)
	}

	// 2 Scouts, in parallel.
	type scoutResult struct {
		area string
		opts []TechOption
		err  error
		ms   int64
	}
	scoutSchema := obj(map[string]any{"options": arr(obj(map[string]any{
		"name": str(), "vendor": str(), "kind": enum("product", "cloud-service", "framework", "pattern"), "summary": str(),
		"fit":           arr(obj(map[string]any{"text": str(), "facts": arr(str())})),
		"prerequisites": arr(str()), "risks": arr(str()), "lifecycle": str(),
		"licence": enum("open-source", "subscription", "cloud-service", "perpetual", "unknown"), "skills": arr(str()),
	}))})
	results := make([]scoutResult, len(in.Areas))
	var wg sync.WaitGroup
	for i, area := range in.Areas {
		wg.Add(1)
		go func(i int, area string) {
			defer wg.Done()
			t := time.Now()
			var res struct {
				Options []TechOption `json:"options"`
			}
			user := brief.String() + "\nArea to cover: " + area + " — " + areaBrief[area] + "\n"
			err := call(ctx, ModelBalanced, scoutSystem, user, scoutSchema, 6000, &res)
			for j := range res.Options {
				res.Options[j].Area = area
			}
			results[i] = scoutResult{area, res.Options, err, time.Since(t).Milliseconds()}
		}(i, area)
	}
	wg.Wait()
	var raw []TechOption
	var failed []string
	for _, r := range results {
		note := fmt.Sprintf("%d options", len(r.opts))
		if r.err != nil {
			note = "failed: " + r.err.Error()
			failed = append(failed, r.area)
		}
		out.Steps = append(out.Steps, AgentStep{Agent: "Scout · " + r.area, Model: Resolve(ModelBalanced), Requests: 1, Ms: r.ms, Note: note})
		out.Requests++
		raw = append(raw, r.opts...)
	}
	if len(failed) == len(in.Areas) {
		return out, fmt.Errorf("every scout failed (%s)", results[0].err)
	}

	// 3 Verifier.
	t := time.Now()
	verified, dropped := verifyOptions(raw, facts)
	out.Dropped = dropped
	out.Steps = append(out.Steps, AgentStep{Agent: "Verifier", Ms: time.Since(t).Milliseconds(), Note: fmt.Sprintf("%d kept, %d dropped", len(verified), len(dropped))})

	// 4 Rules guard.
	t = time.Now()
	guarded := guardOptions(verified, in.Rules)
	for i := range guarded {
		guarded[i].ID = fmt.Sprintf("O%d", i+1)
	}
	out.Options = guarded
	out.Steps = append(out.Steps, AgentStep{Agent: "Rules guard", Ms: time.Since(t).Milliseconds(), Note: fmt.Sprintf("%d rules applied", len(in.Rules))})
	usable := 0
	for _, o := range guarded {
		if o.Guard != "excluded" {
			usable++
		}
	}
	if usable == 0 {
		out.Steps = append(out.Steps, AgentStep{Agent: "Architect", Note: "skipped: no usable options"})
		return out, nil
	}

	// 5 Architect.
	var ob strings.Builder
	ob.WriteString(brief.String())
	ob.WriteString("\nOptions (checked against the rules):\n")
	for _, o := range guarded {
		fmt.Fprintf(&ob, "%s [%s] %s (%s) — %s. Rule check: %s%s\n", o.ID, o.Area, o.Name, o.Vendor, o.Summary, o.Guard, map[bool]string{true: ", " + o.GuardReason, false: ""}[o.GuardReason != ""])
	}
	archSchema := obj(map[string]any{"blueprints": arr(obj(map[string]any{
		"tier": enum("Safe", "Balanced", "Ambitious"), "name": str(), "approach": enum(Approaches...), "options": arr(str()),
		"why":       arr(obj(map[string]any{"text": str(), "facts": arr(str())})),
		"migration": str(), "effort": enum("S", "M", "L", "XL"), "costTrend": enum("lower", "similar", "higher", "unknown"), "deadline": str(),
	}))})
	t = time.Now()
	var arch struct {
		Blueprints []Blueprint `json:"blueprints"`
	}
	err := call(ctx, ModelStrong, architectSystem, ob.String(), archSchema, 8000, &arch)
	out.Requests++
	if err != nil {
		out.Steps = append(out.Steps, AgentStep{Agent: "Architect", Model: Resolve(ModelStrong), Requests: 1, Ms: time.Since(t).Milliseconds(), Note: "failed: " + err.Error()})
		return out, nil
	}
	out.Blueprints = cleanBlueprints(arch.Blueprints, guarded, facts, in.Mode)
	out.Steps = append(out.Steps, AgentStep{Agent: "Architect", Model: Resolve(ModelStrong), Requests: 1, Ms: time.Since(t).Milliseconds(), Note: fmt.Sprintf("%d blueprints", len(out.Blueprints))})
	if len(out.Blueprints) == 0 {
		return out, nil
	}

	// 6 Challenger.
	var cb strings.Builder
	cb.WriteString(ob.String())
	cb.WriteString("\nBlueprints:\n")
	for i, b := range out.Blueprints {
		fmt.Fprintf(&cb, "B%d %s: %s — approach %s; options %s; migration: %s; effort %s\n", i+1, b.Tier, b.Name, b.Approach, strings.Join(b.Options, ", "), b.Migration, b.Effort)
	}
	chSchema := obj(map[string]any{"reviews": arr(obj(map[string]any{"blueprint": integer(), "verdict": enum("sound", "questionable", "risky"), "challenges": arr(str())}))})
	t = time.Now()
	var ch struct {
		Reviews []struct {
			Blueprint  int      `json:"blueprint"`
			Verdict    string   `json:"verdict"`
			Challenges []string `json:"challenges"`
		} `json:"reviews"`
	}
	err = call(ctx, ModelStrong, challengerSystem, cb.String(), chSchema, 4000, &ch)
	out.Requests++
	note := "reviewed"
	if err != nil {
		note = "failed: " + err.Error()
	} else {
		for _, r := range ch.Reviews {
			if r.Blueprint < 1 || r.Blueprint > len(out.Blueprints) {
				continue
			}
			b := &out.Blueprints[r.Blueprint-1]
			if r.Verdict == "sound" || r.Verdict == "questionable" || r.Verdict == "risky" {
				b.Verdict = r.Verdict
			}
			b.Challenges = nil
			for _, c := range r.Challenges {
				if c = strings.TrimSpace(c); c != "" && len(b.Challenges) < 3 {
					b.Challenges = append(b.Challenges, c)
				}
			}
		}
	}
	out.Steps = append(out.Steps, AgentStep{Agent: "Challenger", Model: Resolve(ModelStrong), Requests: 1, Ms: time.Since(t).Milliseconds(), Note: note})
	return out, nil
}

// verifyOptions keeps reasons that cite real facts, options with at least one such reason, one option
// per name, and marks names that are not in the curated catalogue.
func verifyOptions(raw []TechOption, facts map[string]bool) ([]TechOption, []string) {
	var keep []TechOption
	var dropped []string
	seen := map[string]bool{}
	perArea := map[string]int{}
	for _, o := range raw {
		o.Name, o.Vendor, o.Summary = optCut(o.Name, 80), optCut(o.Vendor, 60), optCut(o.Summary, 300)
		key := normName(o.Name)
		if key == "" || seen[key] {
			continue
		}
		var fit []OptionReason
		for _, r := range o.Fit {
			fs := citedFacts(r, facts)
			if t := optCut(r.Text, 300); t != "" && len(fs) > 0 {
				fit = append(fit, OptionReason{Text: t, Facts: fs})
			}
		}
		if len(fit) == 0 {
			dropped = append(dropped, o.Name+": no reason tied to a discovery fact")
			continue
		}
		if limit := map[bool]int{true: 6, false: 5}[o.Parent != ""]; perArea[o.Area] == limit {
			continue
		}
		seen[key] = true
		perArea[o.Area]++
		o.Fit = fit
		o.Prerequisites, o.Risks, o.Skills = trimList(o.Prerequisites, 5), trimList(o.Risks, 5), trimList(o.Skills, 6)
		o.Lifecycle = optCut(o.Lifecycle, 160)
		switch o.Licence {
		case "open-source", "subscription", "cloud-service", "perpetual":
		default:
			o.Licence = "unknown"
		}
		switch {
		case o.Kind == "pattern":
			o.Catalogue = "pattern"
		case knownProduct(o.Name, o.Vendor):
			o.Catalogue = "known"
		default:
			o.Catalogue = "needs confirmation"
		}
		keep = append(keep, o)
	}
	return keep, dropped
}

func trimList(l []string, max int) []string {
	var out []string
	for _, s := range l {
		if s = optCut(s, 200); s != "" && len(out) < max {
			out = append(out, s)
		}
	}
	return out
}

var nonWord = regexp.MustCompile(`[^a-z0-9#+.]+`)

func normName(s string) string {
	return strings.TrimSpace(nonWord.ReplaceAllString(strings.ToLower(s), " "))
}

// guardOptions applies the organization's rules. A rule matches when its option name (or the part before a
// bracket, e.g. "PostgreSQL" from "PostgreSQL (managed)") appears in the option's name or vendor as words.
func guardOptions(opts []TechOption, rules []OptionRule) []TechOption {
	rank := map[string]int{"excluded": 5, "retiring": 4, "approval": 3, "standard": 2, "preferred": 1}
	for i := range opts {
		o := &opts[i]
		o.Guard = "allowed"
		hay := " " + normName(o.Name+" "+o.Vendor) + " "
		best := 0
		for _, r := range rules {
			if !ruleApplies(r.Area, o.Area) {
				continue
			}
			key := normName(strings.SplitN(r.Option, "(", 2)[0])
			for _, alias := range ruleAliases(key) {
				if alias != "" && strings.Contains(hay, " "+alias+" ") && rank[r.State] > best {
					best = rank[r.State]
					st := r.State
					if st == "retiring" {
						st = "excluded"
					}
					o.Guard = st
					o.GuardReason = r.State + " under your rule for " + r.Option + map[bool]string{true: ": " + r.Reason, false: ""}[r.Reason != ""]
				}
			}
		}
	}
	sort.SliceStable(opts, func(a, b int) bool { return guardOrder(opts[a].Guard) < guardOrder(opts[b].Guard) })
	return opts
}

// ruleApplies says whether a rule from one area of the rules page covers an option area.
// Cloud rules name vendors, so they apply everywhere (an excluded cloud also excludes its database service).
func ruleApplies(ruleArea, optArea string) bool {
	switch strings.ToLower(ruleArea) {
	case "cloud", "cloud / hosting":
		return true
	case "database":
		return optArea == "data" || optArea == "products"
	case "language", "language / runtime":
		return optArea == "architecture" || optArea == "ui" || optArea == "legacy"
	case "integration":
		return optArea == "integration"
	case "architecture":
		return optArea == "architecture"
	}
	return true
}

func guardOrder(g string) int {
	return map[string]int{"standard": 0, "preferred": 1, "allowed": 2, "approval": 3, "excluded": 4}[g]
}

// ruleAliases widens a few common rule names so "Microsoft Azure" also matches "Azure SQL Database".
func ruleAliases(key string) []string {
	alias := map[string][]string{
		"microsoft azure":         {"microsoft azure", "azure"},
		"amazon web services":     {"amazon web services", "aws", "amazon"},
		"google cloud":            {"google cloud", "gcp"},
		"oracle cloud":            {"oracle cloud", "oci"},
		"postgresql":              {"postgresql", "postgres"},
		"sql server":              {"sql server", "azure sql"},
		".net":                    {".net", "asp.net", "c#"},
		"node.js typescript":      {"node.js", "node", "typescript"},
		"api gateway + rest apis": {"api gateway", "api management"},
		"event streaming":         {"kafka", "event hubs", "event streaming", "kinesis", "pub sub"},
		"message queue":           {"rabbitmq", "service bus", "sqs", "ibm mq", "message queue"},
		"integration platform":    {"mulesoft", "boomi", "logic apps", "integration platform"},
		"change data capture":     {"debezium", "goldengate", "change data capture", "dms"},
	}
	if a, ok := alias[key]; ok {
		return a
	}
	return []string{key}
}

// cleanBlueprints keeps known approaches, real option ids (never excluded ones), cited reasons and the limits.
func cleanBlueprints(in []Blueprint, opts []TechOption, facts map[string]bool, mode string) []Blueprint {
	byID := map[string]TechOption{}
	for _, o := range opts {
		byID[o.ID] = o
	}
	known := map[string]bool{}
	allowed := ModeApproaches[mode]
	if allowed == nil {
		allowed = ModeApproaches["modernize"]
	}
	for _, a := range allowed {
		known[a] = true
	}
	tierOrder := map[string]int{"Safe": 0, "Balanced": 1, "Ambitious": 2}
	byTier := map[string]bool{}
	var out []Blueprint
	for _, b := range in {
		if _, ok := tierOrder[b.Tier]; !ok || byTier[b.Tier] || !known[b.Approach] {
			continue
		}
		var ids []string
		used := map[string]bool{}
		for _, id := range b.Options {
			id = strings.TrimSpace(id)
			if o, ok := byID[id]; ok && o.Guard != "excluded" && !used[id] {
				used[id] = true
				ids = append(ids, id)
			}
		}
		if len(ids) == 0 && b.Approach != "retire" {
			continue
		}
		var why []OptionReason
		for _, r := range b.Why {
			fs := citedFacts(r, facts)
			if t := optCut(r.Text, 300); t != "" && len(fs) > 0 {
				why = append(why, OptionReason{Text: t, Facts: fs})
			}
		}
		b.Name, b.Migration, b.Deadline = optCut(b.Name, 100), optCut(b.Migration, 400), optCut(b.Deadline, 200)
		b.Options, b.Why = ids, why
		switch b.Effort {
		case "S", "M", "L", "XL":
		default:
			b.Effort = ""
		}
		switch b.CostTrend {
		case "lower", "similar", "higher":
		default:
			b.CostTrend = "unknown"
		}
		b.Verdict, b.Challenges = "", nil
		byTier[b.Tier] = true
		out = append(out, b)
	}
	sort.SliceStable(out, func(a, c int) bool { return tierOrder[out[a].Tier] < tierOrder[out[c].Tier] })
	return out
}

const explorerSystem = `You are a specialist on a modernization team under the MindSuite doctrine.
You receive numbered facts about one part of a legacy system, the organization's technology rules, and one option the team is considering.
Go one level deeper: list the three to six concrete choices inside that option that fit these facts. Stay inside the option: only its own services, plans, tiers or editions; do not list other services (for example databases when exploring a hosting service). For a cloud provider these are its specific services and their plans or tiers (for example for hosting on Microsoft Azure: App Service on a Premium plan, Container Apps, Azure Kubernetes Service, Virtual Machines, Azure Virtual Desktop for a desktop client); for a database, the deployment options and tiers; for a framework, the hosting and UI models.
- Name real services, plans and tiers as the vendor names them in 2026. Do not invent any. If unsure a plan exists, leave tier empty.
- For each choice give: tier (plan, SKU family or edition), sizing (how it scales for the facts), resilience (zones, regions, backups, recovery for the facts), chooseWhen and avoidWhen (one short sentence each).
- For each choice give spec: terse, specific values an architect can compare side by side, one per aspect, at most 140 characters each, no prose:
  sku (exact plan or SKU names and the starting one, e.g. "Premium v3: P1v3 start, P2v3 if memory-bound"),
  os-runtime (OS, runtime stack, container or code deploy),
  scaling (min and max instances, autoscale rule, scale-out time),
  network (VNet integration, private endpoints, inbound and outbound IP, subnet size),
  availability (published SLA %, zone redundancy and its minimum instance count, regions),
  disaster-recovery (pair region or second region, RTO and RPO you can reach, backup),
  deployment (slots, blue-green or rolling, CI/CD target),
  limits (hard limits that matter: request timeout, max instances, max memory, connection limits),
  cost-model (how it is billed: per instance-hour, vCore, reserved or savings plan, what drives cost),
  compliance (certifications relevant to the facts, e.g. HIPAA eligibility under a BAA, ISO 27001, data residency),
  operations (who patches OS and runtime, monitoring hooks, team effort),
  lock-in (what is portable and what ties you to the vendor).
  Leave out aspects that do not apply. Use the vendor's 2026 names and numbers; when unsure of a number, say "confirm" instead of guessing.
- For each choice give compute: a starting production size as vCPU, RAM in GB, storage (disk) in GB and number of instances, each as a number or a short range. Basis is "facts" when it follows from the capacity facts, otherwise "assumption"; when it is an assumption, the note says what it assumes (for example "about 50 users, low traffic").
- Then list the traffic components for running the explored option for this system, with real product names:
  north: what traffic coming in from users and partners passes through (for example DNS, CDN, web application firewall, global or regional load balancer, API management, identity and sign-in);
  south: what the application uses to reach databases, files, legacy and partner systems (for example private endpoints, virtual network integration, NAT gateway, VPN or private connection to on-premises, queues).
  Give each a role (one short sentence) and reasons that cite fact ids. Two to six each.
- Every fit reason cites fact ids. A reason you cannot tie to a fact must not be given.
- Respect the rules: do not propose excluded or retiring technology.
- Kind is product, cloud-service, framework or pattern. Licence is open-source, subscription, cloud-service, perpetual or unknown.
Keep each text short and plain. You only propose; people decide.`

// runExplore asks one specialist for the concrete choices inside one option, then verifies and guards them.
func runExplore(ctx context.Context, in OptionsInput, brief string, facts map[string]bool, call JSONCaller) (OptionsOutput, error) {
	var out OptionsOutput
	e := in.Explore
	traffic := obj(map[string]any{"name": str(), "role": str(), "why": arr(obj(map[string]any{"text": str(), "facts": arr(str())}))})
	schema := obj(map[string]any{"options": arr(obj(map[string]any{
		"name": str(), "vendor": str(), "kind": enum("product", "cloud-service", "framework", "pattern"), "summary": str(),
		"tier": str(), "sizing": str(), "resilience": str(), "chooseWhen": str(), "avoidWhen": str(),
		"compute":       obj(map[string]any{"vcpu": str(), "ramGb": str(), "storageGb": str(), "instances": str(), "basis": enum("facts", "assumption"), "note": str()}),
		"spec":          arr(obj(map[string]any{"aspect": enum(SpecAspects...), "value": str()})),
		"fit":           arr(obj(map[string]any{"text": str(), "facts": arr(str())})),
		"prerequisites": arr(str()), "risks": arr(str()), "lifecycle": str(),
		"licence": enum("open-source", "subscription", "cloud-service", "perpetual", "unknown"), "skills": arr(str()),
	})),
		"north": arr(traffic), "south": arr(traffic)})
	user := brief + "\nOption to explore (" + e.Area + "): " + e.Name + map[bool]string{true: " by " + e.Vendor, false: ""}[e.Vendor != ""] +
		map[bool]string{true: " — " + e.Summary, false: ""}[e.Summary != ""] + "\n"
	type exploreAnswer struct {
		Options []TechOption       `json:"options"`
		North   []TrafficComponent `json:"north"`
		South   []TrafficComponent `json:"south"`
	}
	var res exploreAnswer
	// The specialist sometimes answers with nothing usable; one retry, then report what came back.
	for attempt := 1; attempt <= 2; attempt++ {
		t := time.Now()
		res = exploreAnswer{}
		err := call(ctx, ModelStrong, explorerSystem, user, schema, 12000, &res)
		out.Requests++
		if err != nil {
			return out, fmt.Errorf("the specialist could not explore %s: %w", e.Name, err)
		}
		usable, _ := verifyOptions(append([]TechOption(nil), res.Options...), facts)
		note := fmt.Sprintf("%d choices", len(res.Options))
		if len(usable) == 0 && attempt == 1 {
			note += ", none usable: asked again"
		}
		out.Steps = append(out.Steps, AgentStep{Agent: "Specialist · " + e.Name, Model: Resolve(ModelStrong), Requests: 1, Ms: time.Since(t).Milliseconds(), Note: note})
		if len(usable) > 0 {
			break
		}
	}
	out.North, out.South = cleanTraffic(res.North, facts, in.Rules, e.Area), cleanTraffic(res.South, facts, in.Rules, e.Area)
	for i := range res.Options {
		o := &res.Options[i]
		o.Area, o.Parent = e.Area, e.Name
		o.Tier, o.Sizing, o.Resilience = optCut(o.Tier, 120), optCut(o.Sizing, 300), optCut(o.Resilience, 300)
		o.ChooseWhen, o.AvoidWhen = optCut(o.ChooseWhen, 200), optCut(o.AvoidWhen, 200)
		o.Spec = cleanSpec(o.Spec)
		if c := o.Compute; c != nil {
			c.VCPU, c.RAMGB, c.StorageGB, c.Instances, c.Note = optCut(c.VCPU, 40), optCut(c.RAMGB, 40), optCut(c.StorageGB, 40), optCut(c.Instances, 40), optCut(c.Note, 200)
			if c.Basis != "facts" {
				c.Basis = "assumption"
			}
			if c.VCPU == "" && c.RAMGB == "" && c.StorageGB == "" {
				o.Compute = nil
			}
		}
	}
	t := time.Now()
	verified, dropped := verifyOptions(res.Options, facts)
	out.Dropped = dropped
	out.Steps = append(out.Steps, AgentStep{Agent: "Verifier", Ms: time.Since(t).Milliseconds(), Note: fmt.Sprintf("%d kept, %d dropped", len(verified), len(dropped))})
	guarded := guardOptions(verified, in.Rules)
	for i := range guarded {
		guarded[i].ID = fmt.Sprintf("X%d", i+1)
	}
	out.Options = guarded
	out.Steps = append(out.Steps, AgentStep{Agent: "Rules guard", Note: fmt.Sprintf("%d rules applied", len(in.Rules))})
	return out, nil
}

// cleanTraffic keeps components with a name and at least one reason tied to a real fact (or a role when
// the facts say nothing about the path), marks catalogue matches and applies the rules.
func cleanTraffic(in []TrafficComponent, facts map[string]bool, rules []OptionRule, area string) []TrafficComponent {
	var out []TrafficComponent
	seen := map[string]bool{}
	for _, t := range in {
		t.Name, t.Role = optCut(t.Name, 80), optCut(t.Role, 200)
		key := normName(t.Name)
		if key == "" || seen[key] || len(out) == 6 {
			continue
		}
		var why []OptionReason
		for _, r := range t.Why {
			fs := citedFacts(r, facts)
			if txt := optCut(r.Text, 300); txt != "" && len(fs) > 0 {
				why = append(why, OptionReason{Text: txt, Facts: fs})
			}
		}
		if len(why) == 0 && t.Role == "" {
			continue
		}
		seen[key] = true
		t.Why = why
		t.Catalogue = "needs confirmation"
		if knownProduct(t.Name, "") {
			t.Catalogue = "known"
		}
		g := guardOptions([]TechOption{{Name: t.Name, Area: area}}, rules)[0]
		t.Guard = g.Guard
		if t.Guard == "excluded" {
			continue
		}
		out = append(out, t)
	}
	return out
}

// cleanSpec keeps known aspects once each, in the matrix order, with short values.
func cleanSpec(in []SpecItem) []SpecItem {
	byAspect := map[string]string{}
	for _, it := range in {
		v := optCut(it.Value, 160)
		if v == "" || byAspect[it.Aspect] != "" {
			continue
		}
		byAspect[it.Aspect] = v
	}
	var out []SpecItem
	for _, a := range SpecAspects {
		if v := byAspect[a]; v != "" {
			out = append(out, SpecItem{Aspect: a, Value: v})
		}
	}
	return out
}

var modeText = map[string]string{
	"modernize": "modernize an existing system",
	"new":       "build a new product (facts come from the product brief)",
	"extend":    "modernize an existing system and extend it with new capabilities",
}

// factRefRe finds fact references however the model wrote them: "F3", "f3", "F03", "[F3]", "F3, F7".
var factRefRe = regexp.MustCompile(`(?i)\bF\s*0*(\d{1,3})\b`)

// citedFacts returns the real fact ids a reason cites, from its facts list or, when that is empty,
// from references in its text. Ids that are not in the brief are dropped.
func citedFacts(r OptionReason, facts map[string]bool) []string {
	var out []string
	used := map[string]bool{}
	take := func(s string) {
		for _, m := range factRefRe.FindAllStringSubmatch(s, -1) {
			id := "F" + m[1]
			if facts[id] && !used[id] {
				used[id] = true
				out = append(out, id)
			}
		}
	}
	for _, f := range r.Facts {
		take(f)
	}
	if len(out) == 0 {
		take(r.Text)
	}
	return out
}
