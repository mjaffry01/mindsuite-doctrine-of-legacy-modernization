package agents

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"
)

// Diagram agent: reads the whole design (platform decision, plan layers, capabilities and their routes,
// owners of shared concerns, quality targets and the rule-based draft diagram) and returns a reviewed
// component diagram: components by zone, the flows between them, components the design is missing,
// and conflicts. The web app draws the answer next to its own rule-based diagram.
// Input is design metadata only: no business data, no names of people, no secrets.
// Checks: zones and ids are valid, flows connect known components, nothing names a product or vendor
// that the design does not already mention; one retry when too little survives.

var DiagramZones = []string{"actors", "edge", "app", "data", "cross", "ext"}
var diagramID = regexp.MustCompile(`^[a-z][a-z0-9-]{0,39}$`)

const (
	DiagramMaxLayers = 40
	DiagramMaxCaps   = 60
	DiagramMaxNodes  = 40
	DiagramMaxEdges  = 80
)

type DiagramNode struct {
	ID    string `json:"id"`
	Zone  string `json:"zone"`
	Title string `json:"title"`
	Sub   string `json:"sub"`
	Why   string `json:"why,omitempty"`
}
type DiagramEdge struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Label string `json:"label"`
	Kind  string `json:"kind,omitempty"`
}
type DiagramLayer struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Design string `json:"design"`
}
type DiagramCap struct {
	Name   string `json:"name"`
	Route  string `json:"route"`
	RunsOn string `json:"runsOn"`
}
type DiagramInput struct {
	Project      string         `json:"project"`
	Kind         string         `json:"kind"`
	Platform     []string       `json:"platform"`
	Targets      []string       `json:"targets"`
	Layers       []DiagramLayer `json:"layers"`
	Capabilities []DiagramCap   `json:"capabilities"`
	Owners       []string       `json:"owners"`
	DraftNodes   []DiagramNode  `json:"draftNodes"`
	DraftEdges   []DiagramEdge  `json:"draftEdges"`
}
type DiagramMissing struct {
	Component string `json:"component"`
	Why       string `json:"why"`
}
type DiagramOutput struct {
	Summary   string           `json:"summary"`
	Nodes     []DiagramNode    `json:"nodes"`
	Edges     []DiagramEdge    `json:"edges"`
	Missing   []DiagramMissing `json:"missing"`
	Conflicts []string         `json:"conflicts"`
	Dropped   int              `json:"dropped"`
	Model     string           `json:"model"`
	Attempts  int              `json:"attempts,omitempty"`
}

func tooLong(s string, n int) bool { return utf8.RuneCountInString(s) > n }

func (in DiagramInput) Validate() error {
	if in.Kind != "new" && in.Kind != "modernize" && in.Kind != "extend" {
		return errors.New("unknown project kind")
	}
	if tooLong(in.Project, 180) || len(in.Platform) > 12 || len(in.Targets) > 16 || len(in.Owners) > 12 {
		return errors.New("design summary too large")
	}
	if len(in.Layers) == 0 || len(in.Layers) > DiagramMaxLayers || len(in.Capabilities) > DiagramMaxCaps {
		return errors.New("send between 1 and 40 plan layers")
	}
	for _, list := range [][]string{in.Platform, in.Targets, in.Owners} {
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
	for _, c := range in.Capabilities {
		if strings.TrimSpace(c.Name) == "" || tooLong(c.Name, 120) || tooLong(c.Route, 80) || tooLong(c.RunsOn, 160) {
			return errors.New("invalid capability")
		}
	}
	if len(in.DraftNodes) > DiagramMaxNodes || len(in.DraftEdges) > DiagramMaxEdges {
		return errors.New("draft diagram too large")
	}
	for _, n := range in.DraftNodes {
		if !slices.Contains(DiagramZones, n.Zone) || n.ID == "" || tooLong(n.ID, 40) || tooLong(n.Title, 120) || tooLong(n.Sub, 300) {
			return errors.New("invalid draft component")
		}
	}
	for _, e := range in.DraftEdges {
		if tooLong(e.From, 40) || tooLong(e.To, 40) || tooLong(e.Label, 80) {
			return errors.New("invalid draft flow")
		}
	}
	return nil
}

// stated is all text the owner's design already contains (products may only be named if they appear here).
func (in DiagramInput) stated() string {
	var b strings.Builder
	b.WriteString(in.Project)
	for _, s := range append(append(append([]string{}, in.Platform...), in.Targets...), in.Owners...) {
		b.WriteString(" " + s)
	}
	for _, l := range in.Layers {
		b.WriteString(" " + l.Label + " " + l.Design)
	}
	for _, c := range in.Capabilities {
		b.WriteString(" " + c.Name + " " + c.Route + " " + c.RunsOn)
	}
	for _, n := range in.DraftNodes {
		b.WriteString(" " + n.Title + " " + n.Sub)
	}
	return b.String()
}

func (in DiagramInput) verify(out DiagramOutput) DiagramOutput {
	guard := BriefFillInput{Project: in.Project, Answers: map[string]string{"x": in.stated()}}
	clean := DiagramOutput{Summary: strings.TrimSpace(out.Summary), Nodes: []DiagramNode{}, Edges: []DiagramEdge{}, Missing: []DiagramMissing{}, Conflicts: []string{}}
	if tooLong(clean.Summary, 600) {
		clean.Summary = string([]rune(clean.Summary)[:600])
	}
	ids := map[string]bool{}
	for _, n := range out.Nodes {
		n.ID, n.Title, n.Sub, n.Why = strings.TrimSpace(n.ID), strings.TrimSpace(n.Title), strings.TrimSpace(n.Sub), strings.TrimSpace(n.Why)
		if !diagramID.MatchString(n.ID) || ids[n.ID] || !slices.Contains(DiagramZones, n.Zone) || n.Title == "" || tooLong(n.Title, 80) || tooLong(n.Sub, 200) || tooLong(n.Why, 240) {
			clean.Dropped++
			continue
		}
		if guard.namesUnstatedProduct(n.Title + " " + n.Sub) {
			clean.Dropped++
			continue
		}
		ids[n.ID] = true
		clean.Nodes = append(clean.Nodes, n)
		if len(clean.Nodes) == DiagramMaxNodes {
			break
		}
	}
	seen := map[string]bool{}
	for _, e := range out.Edges {
		key := e.From + ">" + e.To
		if !ids[e.From] || !ids[e.To] || e.From == e.To || seen[key] || tooLong(e.Label, 60) {
			clean.Dropped++
			continue
		}
		if e.Kind != "request" && e.Kind != "data" && e.Kind != "control" && e.Kind != "callback" {
			e.Kind = "request"
		}
		seen[key] = true
		clean.Edges = append(clean.Edges, e)
		if len(clean.Edges) == DiagramMaxEdges {
			break
		}
	}
	for _, m := range out.Missing {
		m.Component, m.Why = strings.TrimSpace(m.Component), strings.TrimSpace(m.Why)
		if m.Component == "" || m.Why == "" || tooLong(m.Component, 80) || tooLong(m.Why, 240) || guard.namesUnstatedProduct(m.Component) {
			clean.Dropped++
			continue
		}
		clean.Missing = append(clean.Missing, m)
	}
	for _, c := range out.Conflicts {
		if c = strings.TrimSpace(c); c != "" && !tooLong(c, 300) {
			clean.Conflicts = append(clean.Conflicts, c)
		}
	}
	return clean
}

const diagramPrompt = `You are the MindSuite Diagram agent, a senior solution architect. You receive a design summary as quoted data: the platform decision, the plan layers (each with its design text), the capabilities with their route and where they run, who owns each shared concern, the quality targets, and a rule-based draft component diagram. Understand the whole design and return the component diagram an architect would draw for it.
Rules:
- Components are running parts: entry (CDN, WAF, load balancer, API gateway, device hub), application runtimes and workers, data stores, cross-cutting services (identity, secrets, monitoring, release pipeline, backup), and external services (payment provider, bought products, partner platforms, AI model service, other systems). Users and devices are in zone "actors".
- Zones: actors, edge, app, data, cross, ext. Ids: lower-case words joined by hyphens. Reuse the draft's ids for the same component.
- Use the service names and sizes exactly as written in the layers; never name a product, vendor or cloud service that the summary does not mention. If the design needs a component it lacks (for example private endpoints, DNS, a backup vault, a second region for the availability target), list it under "missing" with the reason instead of inventing a product.
- Flows: from, to, a short label with the protocol or purpose (HTTPS, SQL over TLS, MQTT, publish events, OIDC, webhooks), and kind: request, data, control (sign-in, secrets, telemetry, deploy) or callback.
- "conflicts": anything in the design that does not fit together (for example a capability bought while the platform also builds the same thing, an availability target the data tier cannot meet).
- "summary": two or three sentences an architect would say about this design.
Keep it to the components that matter: at most 30 components and 60 flows.`

const diagramRetry = " A first attempt returned too few valid components. Use the draft ids, the zones listed and only names from the summary."

func (l *LLM) DesignDiagram(ctx context.Context, in DiagramInput) (DiagramOutput, error) {
	return RunDesignDiagram(ctx, in, l.JSON)
}

// RunDesignDiagram asks the model, verifies the answer and asks once more when fewer than 3 components survive.
func RunDesignDiagram(ctx context.Context, in DiagramInput, call JSONCaller) (DiagramOutput, error) {
	data, err := json.Marshal(struct {
		Design DiagramInput `json:"design"`
	}{in})
	if err != nil {
		return DiagramOutput{}, err
	}
	schema := obj(map[string]any{
		"summary":   str(),
		"nodes":     arr(obj(map[string]any{"id": str(), "zone": enum(DiagramZones...), "title": str(), "sub": str(), "why": str()})),
		"edges":     arr(obj(map[string]any{"from": str(), "to": str(), "label": str(), "kind": enum("request", "data", "control", "callback")})),
		"missing":   arr(obj(map[string]any{"component": str(), "why": str()})),
		"conflicts": arr(str()),
	})
	var out DiagramOutput
	for attempt := 1; attempt <= 2; attempt++ {
		var raw DiagramOutput
		p := diagramPrompt
		if attempt == 2 {
			p += diagramRetry
		}
		if err = call(ctx, ModelStrong, p, string(data), schema, 8000, &raw); err != nil {
			return DiagramOutput{}, err
		}
		out = in.verify(raw)
		out.Attempts = attempt
		if len(out.Nodes) >= 3 {
			break
		}
	}
	out.Model = Resolve(ModelStrong)
	return out, nil
}
