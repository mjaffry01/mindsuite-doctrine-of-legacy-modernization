package code

import (
	"context"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/mjaffry01/mindsuite-doctrine-of-legacy-modernization/backend/internal/agents"
)

func init() { agents.Register(team{}) }

type team struct{}

func (team) ID() string                          { return "code" }
func (team) Agents() []AgentInfo                 { return Team }
func (team) StepTitles() []string                { return stepTitles }
func (team) NewState() any                       { return nil }
func (team) Execute(ctx context.Context, r *Run) { executeCode(ctx, r) }
func (team) WholeKinds() map[string]bool {
	return map[string]bool{"language": true, "version": true, "bus-factor": true, "architecture-style": true, "extraction-order": true, "module": true, "module-tests": true, "document": true}
}

// ReconcileBatches builds an import index across the repository after the detailed reviews.
func (team) ReconcileBatches(r *Run, batchOf func(Citation) int) {
	known := map[[2]string]bool{}
	for _, finding := range r.Store.Findings("module-edge") {
		if finding.Status == StatusRejected {
			continue
		}
		known[[2]string{fmt.Sprint(finding.Data["from"]), fmt.Sprint(finding.Data["to"])}] = true
	}
	edges := buildModuleGraph(r, false)
	keys := make([][2]string, 0, len(edges))
	for key := range edges {
		if !known[key] {
			keys = append(keys, key)
		}
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i][0]+"\x00"+keys[i][1] < keys[j][0]+"\x00"+keys[j][1] })
	for _, key := range keys {
		edge := edges[key]
		r.Store.Add(Finding{
			Agent: "str", Kind: "cross-batch-link", Title: edge.from + " imports " + edge.to,
			Detail:    "The repository-wide import index links these modules.",
			Citations: []Citation{cite(edge.file, edge.line)}, Status: StatusAccepted,
		})
	}
	traceSourceBusiness(r, func(citations []Citation) bool {
		first := batchOf(citations[0])
		for _, c := range citations[1:] {
			if next := batchOf(c); next > 0 && first > 0 && next != first {
				return true
			}
		}
		return false
	})
}

func (team) BatchPriority(r *Run, context []string) int {
	return reviewPriority(r).score(Finding{Title: strings.Join(context, " ")})
}

func (team) BatchReviewContext(r *Run) string {
	var b strings.Builder
	var findings []Finding
	for _, finding := range r.Store.Findings("workflow-trace", "business-rule", "journey") {
		if finding.Status != StatusRejected {
			findings = append(findings, finding)
		}
	}
	priority := reviewPriority(r)
	writeInterviewFocus(&b, r, priority, findings)
	writeOperatingModel(&b, r, priority)
	return b.String()
}

func (team) Source(r *Run) map[string]any {
	return map[string]any{"root": r.Code.Root, "branch": r.Code.Branch, "commit": r.Code.Commit, "files": len(r.Code.Files), "lines": r.Code.TotalLines()}
}

var stepTitles = []string{"Receive the code", "Plan the work", "Collect the facts", "Choose what to read in depth",
	"Understand the business", "Assess for modernization", "Verify every claim", "Write the report", "Validate with your people"}

// RunPlatform is the platform & config agent; the infrastructure team reuses it for infrastructure-as-code files.
func RunPlatform(r *Run) { runPlatform(r) }

// executeCode runs the code team (15 agents, 9 steps).
func executeCode(ctx context.Context, r *Run) {
	s := r.Store

	// 1 Receive the code
	r.Step(1, "running", "")
	s.Task("inv", "Record what is being analysed: languages, size, build files, version.", nil)
	r.MeasureAgent("inv", func() { runInventory(r) })
	r.Step(1, "done", fmt.Sprintf("%d files, %d lines", len(r.Code.Files), r.Code.TotalLines()))

	// 2 Plan the work
	r.Step(2, "running", "")
	packages := plan(r)
	r.Step(2, "done", fmt.Sprintf("%d work packages (modules)", len(packages)))

	// 3 Collect the facts: tool agents in parallel
	r.Step(3, "running", "")
	// The platform agent does two parts; the second (evidence areas) has its own key so it can fail and re-run alone.
	fact := []agents.Part{{ID: "sup", Fn: runSupplyChain}, {ID: "str", Fn: runStructure}, {ID: "dat", Fn: runDataAccess},
		{ID: "int", Fn: runInterfaces}, {ID: "sec", Fn: runSecurity}, {ID: "qua", Fn: runQuality},
		{ID: "pla", Fn: runPlatform}, {ID: "pla", Key: "areas", Name: "Evidence areas (part of Platform & config)", Fn: runAreas}}
	if hasLegacy(r) {
		fact = append(fact, agents.Part{ID: "leg", Fn: runLegacy})
	}
	for _, a := range fact {
		s.Task(a.ID, "Collect facts for all work packages.", map[string]any{"modules": packages})
	}
	r.RunParts(ctx, fact)
	s.Task("dom", "Identify likely business uses from code links between screens and services.", nil)
	sourcePurposes := 0
	r.RunPart(ctx, agents.Part{ID: "dom", Key: "source-business", Name: "Business uses from source", Fn: func(r *Run) { sourcePurposes = runSourceBusiness(r) }})
	r.Step(3, "done", fmt.Sprintf("%d findings", len(s.Findings())))

	// 4 Choose what to read in depth
	r.Step(4, "running", "")
	chooseReading(r)
	r.Step(4, "done", fmt.Sprintf("%d files picked", len(r.Reading)))

	// 5 Understand the business
	if ok, why := r.ModelReady(); ok {
		r.Step(5, "running", "")
		s.Task("rul", "Write the business rules in the reading list in plain language, with exact lines.", map[string]any{"files": r.Reading})
		r.MeasureAgent("rul", func() { runBusinessRules(ctx, r) })
		s.Task("dom", "Connect screens, rules, data and events into the business story.", nil)
		r.MeasureAgent("dom", func() { runDomain(ctx, r) })
		r.Step(5, "done", fmt.Sprintf("%d source-based business purposes and %d rules", sourcePurposes, len(s.Findings("business-rule"))))
	} else {
		if sourcePurposes == 0 {
			r.Step(5, "skipped", "Source did not establish a business workflow. Claude skipped: "+why)
		} else {
			r.Step(5, "done", fmt.Sprintf("%d business purposes inferred from source. Claude skipped: %s", sourcePurposes, why))
		}
	}

	// 6 Assess
	r.Step(6, "running", "")
	s.Task("ass", "Score each module for extraction and suggest the order.", nil)
	r.MeasureAgent("ass", func() { runAssessment(r) })
	r.Step(6, "done", "")

	// 7 Verify
	r.Step(7, "running", "")
	s.Task("ver", "Check every claim against its cited evidence.", nil)
	agents.RunVerifier(ctx, r)
	acc := 0
	for _, f := range s.Findings() {
		if f.Status == StatusAccepted {
			acc++
		}
	}
	r.Step(7, "done", fmt.Sprintf("%d of %d findings accepted, %d questions for your experts", acc, len(s.Findings()), len(s.Messages(MsgQuestion))))

	// 8 Report
	r.Step(8, "running", "")
	s.Task("rep", "Write the Code analysis report and the Business story from verified findings.", nil)
	r.WriteReport(ctx, runReport)
	r.Step(8, "done", "")

	// 9 People
	r.Step(9, "people", "Open the results in MindSuite: your experts confirm, correct or reject each finding.")
}

// plan splits the code into work packages per module and routes legacy code.
func plan(r *Run) []string {
	mods := r.Code.Modules()
	for _, m := range mods {
		n := 0
		for _, f := range r.Code.Files {
			if f.Module == m {
				n++
			}
		}
		r.Store.Log("orch", fmt.Sprintf("Work package %q: %d files", m, n))
	}
	return mods
}

func hasLegacy(r *Run) bool {
	for _, f := range r.Code.Files {
		if LegacyLangs[f.Lang] {
			return true
		}
	}
	return false
}

// chooseReading picks the files where the business logic is most likely to live.
func chooseReading(r *Run) {
	score := map[string]float64{}
	why := map[string][]string{}
	add := func(file string, s float64, reason string) {
		f := r.Code.File(file)
		if f == nil || f.IsTest || bundledScript(f) {
			return
		}
		score[file] += s
		why[file] = appendUnique(why[file], reason)
	}
	priority := reviewPriority(r)
	for _, f := range r.Store.Findings("journey", "business-rule", "workflow-trace") {
		for _, c := range f.Citations {
			add(c.File, 8, "supports a business workflow or rule")
			if focusScore := priority.score(f); focusScore > 0 {
				add(c.File, float64(min(focusScore, 20)), "matches the project interview")
			}
		}
	}
	for _, f := range r.Store.Findings("hotspot") {
		if len(f.Citations) > 0 {
			add(f.Citations[0].File, 3, "complexity hotspot")
		}
	}
	for _, f := range r.Store.Findings("table-access") {
		if op, _ := f.Data["op"].(string); op == "write" && len(f.Citations) > 0 {
			add(f.Citations[0].File, 1.5, "writes "+fmt.Sprint(f.Data["table"]))
		}
	}
	for _, f := range r.Store.Findings("stored-procedure", "legacy-program") {
		if len(f.Citations) > 0 {
			add(f.Citations[0].File, 2.5, "database or legacy program logic")
		}
	}
	for _, f := range r.Store.Findings("schedule") {
		if len(f.Citations) > 0 {
			add(f.Citations[0].File, 1, "scheduled job")
		}
	}
	// rule-like code: many conditions with business constants
	for _, f := range r.Code.Files {
		if f.IsTest || !(CodeLangs[f.Lang] || LegacyLangs[f.Lang] || f.Lang == "PL/SQL" || f.Lang == "SQL") {
			continue
		}
		n := 0
		for _, l := range f.Lines {
			if ruleLikeRe.MatchString(l) {
				n++
			}
		}
		if n >= 4 {
			add(f.Path, float64(n)/4, fmt.Sprintf("%d rule-like conditions", n))
		}
	}
	files := sortedKeys(score)
	sort.SliceStable(files, func(i, j int) bool { return score[files[i]] > score[files[j]] })
	if len(files) > r.Options.MaxModelFiles {
		r.Store.Log("orch", fmt.Sprintf("%d files qualify for in-depth reading; the top %d are read in this run (raise maxModelFiles to read more).", len(files), r.Options.MaxModelFiles))
		files = files[:r.Options.MaxModelFiles]
	}
	for _, f := range files {
		r.Reading = append(r.Reading, Reading{File: f, Reason: strings.Join(why[f], ", ")})
	}
	r.Store.Log("orch", fmt.Sprintf("Reading list: %d files", len(r.Reading)))
}

// Coverage summarises
// Team is the agent team in the order of a run.
var Team = []AgentInfo{
	{ID: "orch", Name: "Orchestrator", Kind: "coordinator", Model: "", Does: "Splits the code into work packages per module, assigns agents, tracks coverage, picks the reading list."},
	{ID: "inv", Name: "Inventory", Kind: "tool", Model: "", Does: "Languages, size, build files, branch and commit."},
	{ID: "sup", Name: "Supply chain", Kind: "tool", Model: "", Does: "Dependencies and versions from build files; end-of-life and known-risk versions."},
	{ID: "str", Name: "Structure", Kind: "tool", Model: "", Does: "Modules, module dependencies, cycles, architecture style."},
	{ID: "dat", Name: "Data access", Kind: "tool", Model: "", Does: "Which module reads or writes which table, shared tables, stored procedures."},
	{ID: "int", Name: "Interfaces", Kind: "tool", Model: "", Does: "Endpoints, outbound calls, queues and topics, file exchanges, SOAP."},
	{ID: "sec", Name: "Security", Kind: "tool", Model: "", Does: "Login methods, role checks per endpoint, secrets (file and line only), weak crypto, injection risks."},
	{ID: "qua", Name: "Quality & change safety", Kind: "tool", Model: "", Does: "Tests per module, complexity hotspots, duplication, dead-code candidates, Git churn and bus factor."},
	{ID: "pla", Name: "Platform & config", Kind: "tool", Model: "", Does: "Containers, Kubernetes, Terraform, CI, environments, connections, caching, scheduled jobs, app servers."},
	{ID: "leg", Name: "Legacy-language specialists", Kind: "tool", Model: "", Does: "COBOL, JCL, RPG, PowerBuilder: programs, calls, copybooks, SQL, CICS, job steps."},
	{ID: "rul", Name: "Business rules", Kind: "model", Model: ModelStrong, Does: "Reads the reading list and writes each rule in plain language with its exact lines."},
	{ID: "dom", Name: "Domain & process", Kind: "tool+model", Model: ModelStrong, Does: "Traces business activities from source and uses Claude, when available, to expand the story."},
	{ID: "ass", Name: "Modernization assessment", Kind: "tool", Model: "", Does: "Extractability, effort, seams and extraction order per module."},
	{ID: "ver", Name: "Verifier", Kind: "tool+model", Model: ModelBalanced, Does: "Re-opens every cited line; rejects unsupported claims; turns doubts into expert questions."},
	{ID: "rep", Name: "Report writer", Kind: "tool+model", Model: ModelBalanced, Does: "Writes the Code analysis report and the Business story from verified findings."},
}

// Bundled UI libraries are dependencies, not candidates for business rule reading.
func bundledScript(f *File) bool {
	if f.Lang != "JavaScript" {
		return false
	}
	name := strings.ToLower(path.Base(f.Path))
	if strings.HasSuffix(name, ".min.js") || strings.HasSuffix(name, ".intellisense.js") {
		return true
	}
	for _, prefix := range []string{"jquery", "bootstrap", "popper", "modernizr", "respond", "microsoftajax", "webuivalidation"} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return strings.Contains(strings.ToLower(f.Path), "/scripts/webforms/")
}
