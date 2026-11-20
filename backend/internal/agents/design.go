package agents

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Solution designer (Target options team): details one blueprint into a solution design: database
// strategy, caching strategy and components, design patterns, traffic in (north) and out (south),
// cross-cutting components (identity, secrets, monitoring, delivery, backup and recovery, security),
// and a starting size per component. Every reason cites discovery facts; sizes say if they are assumptions.

// DesignTarget is the blueprint to design.
type DesignTarget struct {
	Tier      string   `json:"tier"`
	Name      string   `json:"name"`
	Approach  string   `json:"approach"`
	Options   []string `json:"options"`
	Migration string   `json:"migration"`
}

// DesignItem is one decision in the design.
type DesignItem struct {
	Name      string         `json:"name"`
	Detail    string         `json:"detail"`
	Why       []OptionReason `json:"why"`
	Catalogue string         `json:"catalogue"`
}

// DesignComponent is a cross-cutting component, grouped by category.
type DesignComponent struct {
	Category  string         `json:"category"`
	Name      string         `json:"name"`
	Role      string         `json:"role"`
	Why       []OptionReason `json:"why"`
	Catalogue string         `json:"catalogue"`
}

// SizedComponent is a starting size for one runtime component of the design.
type SizedComponent struct {
	Component string `json:"component"`
	ComputeSize
}

// SolutionDesign is the designer's answer.
type SolutionDesign struct {
	Summary    string             `json:"summary"`
	Database   []DesignItem       `json:"database"`
	Caching    []DesignItem       `json:"caching"`
	Patterns   []DesignItem       `json:"patterns"`
	North      []TrafficComponent `json:"north"`
	South      []TrafficComponent `json:"south"`
	Components []DesignComponent  `json:"components"`
	Sizing     []SizedComponent   `json:"sizing"`
	Gaps       []string           `json:"gaps"`
}

// DesignCategories are the cross-cutting component groups the designer must consider.
var DesignCategories = []string{"identity", "secrets", "monitoring", "delivery", "backup-recovery", "security", "configuration", "messaging", "other"}

func (d *DesignTarget) validate() error {
	ok := false
	for _, a := range Approaches {
		ok = ok || a == d.Approach
	}
	if !ok {
		return errors.New("the blueprint to design has no known approach")
	}
	d.Tier, d.Name, d.Migration = optCut(d.Tier, 20), optCut(d.Name, 100), optCut(d.Migration, 400)
	var opts []string
	for _, o := range d.Options {
		if o = optCut(o, 120); o != "" && len(opts) < 15 {
			opts = append(opts, o)
		}
	}
	d.Options = opts
	return nil
}

const designerSystem = `You are the solution designer on a modernization team under the MindSuite doctrine.
You receive numbered facts about one part of a legacy system, the organization's technology rules, and one target blueprint (approach, chosen options, migration path).
Turn the blueprint into a solution design an engineering team can start from. Use real product names as vendors name them in 2026; never invent products. Respect the rules: no excluded or retiring technology.
- database: two to five decisions: engine and edition, high availability, backup and point-in-time restore, how the data moves from the current system, and anything the facts call for (encryption, retention, read replicas, partitioning).
- caching: one to four decisions: what to cache and where (for example reference data, sessions, output, static files at the edge), the caching component, expiry and invalidation. Say plainly if caching is not worth it for these facts.
- patterns: three to seven design patterns that fit, each with where it applies (for example strangler fig at the edge, repository and unit of work in data access, outbox for reliable events, retry with circuit breaker on outbound calls, CQRS only if the facts justify it).
- north: components for traffic coming in from users and partners. south: components the application uses to reach databases, files, legacy and partner systems.
- components: the cross-cutting components a design like this needs, by category: identity, secrets, monitoring, delivery (build and release), backup-recovery, security, configuration, messaging, other. Include the ones teams often forget.
- sizing: a starting production size for each runtime component (application, database, cache and any others) as vCPU, RAM in GB, storage in GB and instances. Basis is "facts" when it follows from capacity facts, otherwise "assumption" with a note on what is assumed.
- Answer as one list of items, each with its section (database, caching, pattern, north, south, or a component category), a name, the detail, one short reason and the fact ids the reason rests on. Leave the reason and facts empty when no fact supports it.
- gaps: the facts still missing that would change this design (at most six).
- summary: two sentences.
Write plain, short sentences. You only propose; people decide.`

// DesignBlueprint runs the solution designer for one blueprint with Claude.
func (l *LLM) DesignBlueprint(ctx context.Context, in OptionsInput) (SolutionDesign, AgentStep, error) {
	return RunDesign(ctx, in, l.JSON)
}

// RunDesign builds the brief, asks the designer and cleans the answer.
func RunDesign(ctx context.Context, in OptionsInput, call JSONCaller) (SolutionDesign, AgentStep, error) {
	b := in.Design
	var u strings.Builder
	facts := map[string]bool{}
	fmt.Fprintf(&u, "Part of the system: %s\nStrategic value: %s\nBusiness criticality: %s\n\nFacts:\n", orUnknown(in.Context), orUnknown(in.Strategic), orUnknown(in.Criticality))
	for _, f := range in.Facts {
		facts[f.ID] = true
		fmt.Fprintf(&u, "%s [%s] %s\n", f.ID, f.Area, f.Text)
	}
	u.WriteString("\nOrganization technology rules:\n")
	if len(in.Rules) == 0 {
		u.WriteString("- none: everything is allowed\n")
	}
	for _, r := range in.Rules {
		fmt.Fprintf(&u, "- %s %s: %s\n", r.Area, r.Option, r.State)
	}
	fmt.Fprintf(&u, "\nBlueprint (%s): %s\nApproach: %s\nOptions: %s\nMigration: %s\n", orUnknown(b.Tier), b.Name, b.Approach, strings.Join(b.Options, "; "), orUnknown(b.Migration))

	// One flat list keeps the structured-output grammar small; it is regrouped below.
	sections := append([]string{"database", "caching", "pattern", "north", "south"}, DesignCategories...)
	schema := obj(map[string]any{
		"summary": str(),
		"items":   arr(obj(map[string]any{"section": enum(sections...), "name": str(), "detail": str(), "reason": str(), "facts": arr(str())})),
		"sizing":  arr(obj(map[string]any{"component": str(), "vcpu": str(), "ramGb": str(), "storageGb": str(), "instances": str(), "basis": enum("facts", "assumption"), "note": str()})),
		"gaps":    arr(str()),
	})
	t := time.Now()
	var flat struct {
		Summary string `json:"summary"`
		Items   []struct {
			Section string   `json:"section"`
			Name    string   `json:"name"`
			Detail  string   `json:"detail"`
			Reason  string   `json:"reason"`
			Facts   []string `json:"facts"`
		} `json:"items"`
		Sizing []SizedComponent `json:"sizing"`
		Gaps   []string         `json:"gaps"`
	}
	err := call(ctx, ModelStrong, designerSystem, u.String(), schema, 16000, &flat)
	step := AgentStep{Agent: "Solution designer · " + orUnknown(b.Tier), Model: Resolve(ModelStrong), Requests: 1, Ms: time.Since(t).Milliseconds()}
	if err != nil {
		step.Note = "failed: " + err.Error()
		return SolutionDesign{}, step, err
	}
	raw := SolutionDesign{Summary: flat.Summary, Sizing: flat.Sizing, Gaps: flat.Gaps}
	for _, it := range flat.Items {
		why := []OptionReason{{Text: it.Reason, Facts: it.Facts}}
		switch it.Section {
		case "database":
			raw.Database = append(raw.Database, DesignItem{Name: it.Name, Detail: it.Detail, Why: why})
		case "caching":
			raw.Caching = append(raw.Caching, DesignItem{Name: it.Name, Detail: it.Detail, Why: why})
		case "pattern":
			raw.Patterns = append(raw.Patterns, DesignItem{Name: it.Name, Detail: it.Detail, Why: why})
		case "north":
			raw.North = append(raw.North, TrafficComponent{Name: it.Name, Role: it.Detail, Why: why})
		case "south":
			raw.South = append(raw.South, TrafficComponent{Name: it.Name, Role: it.Detail, Why: why})
		default:
			raw.Components = append(raw.Components, DesignComponent{Category: it.Section, Name: it.Name, Role: it.Detail, Why: why})
		}
	}
	d := cleanDesign(raw, facts, in.Rules)
	step.Note = fmt.Sprintf("%d database, %d caching, %d patterns, %d components", len(d.Database), len(d.Caching), len(d.Patterns), len(d.Components))
	return d, step, nil
}

func cleanReasons(in []OptionReason, facts map[string]bool) []OptionReason {
	var out []OptionReason
	for _, r := range in {
		fs := citedFacts(r, facts)
		if t := optCut(r.Text, 300); t != "" && len(fs) > 0 {
			out = append(out, OptionReason{Text: t, Facts: fs})
		}
	}
	return out
}

// cleanDesign trims, limits, keeps only cited reasons, marks catalogue matches and drops excluded technology.
func cleanDesign(d SolutionDesign, facts map[string]bool, rules []OptionRule) SolutionDesign {
	excluded := func(name, area string) bool {
		return guardOptions([]TechOption{{Name: name, Area: area}}, rules)[0].Guard == "excluded"
	}
	items := func(in []DesignItem, max int, area string, products bool) []DesignItem {
		var out []DesignItem
		for _, it := range in {
			it.Name, it.Detail = optCut(it.Name, 100), optCut(it.Detail, 400)
			if it.Name == "" || len(out) == max || (products && excluded(it.Name, area)) {
				continue
			}
			it.Why = cleanReasons(it.Why, facts)
			it.Catalogue = "pattern"
			if products {
				it.Catalogue = "needs confirmation"
				if knownProduct(it.Name, "") {
					it.Catalogue = "known"
				}
			}
			out = append(out, it)
		}
		return out
	}
	out := SolutionDesign{Summary: optCut(d.Summary, 400)}
	out.Database = items(d.Database, 5, "data", true)
	out.Caching = items(d.Caching, 4, "data", true)
	out.Patterns = items(d.Patterns, 7, "architecture", false)
	out.North = cleanTraffic(d.North, facts, rules, "hosting")
	out.South = cleanTraffic(d.South, facts, rules, "hosting")
	cats := map[string]bool{}
	for _, c := range DesignCategories {
		cats[c] = true
	}
	for _, c := range d.Components {
		c.Name, c.Role = optCut(c.Name, 80), optCut(c.Role, 200)
		if c.Name == "" || len(out.Components) == 16 || excluded(c.Name, "hosting") {
			continue
		}
		if !cats[c.Category] {
			c.Category = "other"
		}
		c.Why = cleanReasons(c.Why, facts)
		c.Catalogue = "needs confirmation"
		if knownProduct(c.Name, "") {
			c.Catalogue = "known"
		}
		out.Components = append(out.Components, c)
	}
	for _, s := range d.Sizing {
		s.Component = optCut(s.Component, 80)
		s.VCPU, s.RAMGB, s.StorageGB, s.Instances, s.Note = optCut(s.VCPU, 20), optCut(s.RAMGB, 20), optCut(s.StorageGB, 20), optCut(s.Instances, 20), optCut(s.Note, 200)
		none := func(v string) bool {
			v = strings.TrimSpace(v)
			return v == "" || v == "0" || v == "-" || strings.EqualFold(v, "n/a")
		}
		// Managed services without compute to size (for example a CDN) are left out of the sizing table.
		if s.Component == "" || (none(s.VCPU) && none(s.RAMGB) && none(s.StorageGB)) || len(out.Sizing) == 10 {
			continue
		}
		if s.Basis != "facts" {
			s.Basis = "assumption"
		}
		out.Sizing = append(out.Sizing, s)
	}
	out.Gaps = trimList(d.Gaps, 6)
	return out
}
