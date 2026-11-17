package database

import (
	"context"
	"fmt"
	"github.com/mjaffry01/mindsuite-doctrine-of-legacy-modernization/backend/internal/agents"
	"sort"
	"strings"
	"sync"
)

// DBTeam is the database agent team in the order of a run (ids match db-agents.html in the web app).
var DBTeam = []AgentInfo{
	{ID: "orch", Name: "Database orchestrator", Kind: "coordinator", Model: "", Does: "Reads the export files, splits the work per schema, starts the agents, picks the programs to read in depth."},
	{ID: "cat", Name: "Catalog reader", Kind: "tool", Model: "", Does: "Engine and version, schemas, and every table, view, column, key, index, program, job, link and account in the export."},
	{ID: "rel", Name: "Schema & relationships", Kind: "tool", Model: "", Does: "Primary and foreign keys, central tables, data clusters (candidate data domains), cross-schema keys, rules written as CHECK constraints."},
	{ID: "vol", Name: "Size & usage", Kind: "tool", Model: "", Does: "Database size, largest tables, empty tables (retire candidates), from catalog statistics."},
	{ID: "prc", Name: "Stored code", Kind: "tool", Model: "", Does: "Procedures, functions, packages and triggers: size, complexity, tables read and written, calls, dynamic SQL; tables written by several programs."},
	{ID: "job", Name: "Jobs, links & outside calls", Kind: "tool", Model: "", Does: "Scheduled jobs and what they run, database links, remote synonyms, calls out of the database (HTTP, mail, files, shell)."},
	{ID: "dsec", Name: "Security & privacy", Kind: "tool", Model: "", Does: "Administrator accounts, grants to PUBLIC, columns with personal or sensitive data (by name), password columns, passwords in the export (file and line only)."},
	{ID: "prt", Name: "Portability", Kind: "tool", Model: "", Does: "Engine-specific features and data types a move to another engine must convert, with effort."},
	{ID: "dq", Name: "Design quality", Kind: "tool", Model: "", Does: "Tables without primary keys, foreign keys without indexes, very wide tables, generic and name/value columns, isolated tables."},
	{ID: "drul", Name: "Database business rules", Kind: "model", Model: ModelStrong, Does: "Reads the programs the orchestrator picked and writes each business rule in plain language with its exact lines."},
	{ID: "ddom", Name: "Data domains & meaning", Kind: "model", Model: ModelStrong, Does: "From table and column names, keys, programs and jobs (never data): data domains, what each main table means, vocabulary, data lifecycle."},
	{ID: "dass", Name: "Migration assessment", Kind: "tool", Model: "", Does: "Complexity, effort and order per data domain, and the overall migration approach."},
	{ID: "ver", Name: "Verifier", Kind: "tool+model", Model: ModelBalanced, Does: "Re-opens every cited line of the export; a second model checks the rules; doubts become questions for your DBA and experts."},
	{ID: "rep", Name: "Report writer", Kind: "tool+model", Model: ModelBalanced, Does: "Writes the Database analysis report from verified findings."},
}

var dbStepTitles = []string{"Receive the database export", "Plan the work", "Collect the facts", "Choose what to read in depth",
	"Understand the business", "Assess for migration", "Verify every claim", "Write the report", "Validate with your people"}

// executeDB runs the database team (14 agents, 9 steps).
func executeDB(ctx context.Context, r *Run) {
	s := r.Store

	// 1 Receive
	r.Step(1, "running", "")
	s.Task("cat", "Read the export files: tables, columns, keys, programs, jobs, links, accounts, sizes.", nil)
	m := ParseDB(r)
	st(r).db = m
	if len(m.Files) == 0 {
		s.Log("cat", "No schema export (.sql/.ddl) or catalog result (.csv) was recognised.")
	}
	runCatalog(r)
	sum := m.Summary()
	r.Step(1, "done", fmt.Sprintf("%s: %v tables, %v views, %v programs, %v jobs from %d file(s)", engineNames[m.Engine], sum["tables"], sum["views"], sum["programs"], sum["jobs"], len(m.Files)))

	// 2 Plan
	r.Step(2, "running", "")
	schemas := m.Schemas()
	for _, sc := range schemas {
		nt, nc := 0, 0
		for _, t := range m.Tables {
			if schemaLabel(t.Schema) == sc && t.File != "" {
				nt++
			}
		}
		for _, c := range m.Code {
			if s2, _ := splitName(c.Key); schemaLabel(s2) == sc {
				nc++
			}
		}
		s.Log("orch", fmt.Sprintf("Work package %q: %d tables and views, %d programs", sc, nt, nc))
	}
	r.Step(2, "done", fmt.Sprintf("%d work packages (schemas)", len(schemas)))

	// 3 Facts: in parallel; jobs follow stored code because they point to the programs they run.
	r.Step(3, "running", "")
	facts := []struct {
		id string
		fn func(*Run)
	}{{"rel", runRelationships}, {"vol", runVolume}, {"prc", runStoredCode}, {"dsec", runDBSecurity}, {"prt", runPortability}, {"dq", runDesignQuality}}
	var wg sync.WaitGroup
	safe := func(id string, fn func(*Run)) {
		defer func() {
			if p := recover(); p != nil {
				s.Log(id, fmt.Sprint("Agent stopped with an error: ", p))
			}
		}()
		fn(r)
	}
	for _, a := range facts {
		s.Task(a.id, "Collect facts for all schemas.", map[string]any{"schemas": schemas})
		wg.Add(1)
		go func(id string, fn func(*Run)) {
			defer wg.Done()
			safe(id, fn)
		}(a.id, a.fn)
	}
	wg.Wait()
	s.Task("job", "Collect jobs, links and outside calls; link each job to the program it runs.", nil)
	safe("job", runJobsLinks)
	r.Step(3, "done", fmt.Sprintf("%d findings", len(s.Findings())))

	// 4 Reading list
	r.Step(4, "running", "")
	chooseDBReading(r)
	r.Step(4, "done", fmt.Sprintf("%d programs picked", len(r.Reading)))

	// 5 Business meaning
	if ok, why := r.ModelReady(); ok {
		r.Step(5, "running", "")
		if len(r.Reading) > 0 {
			s.Task("drul", "Write the business rules in the picked programs in plain language, with exact lines.", map[string]any{"programs": r.Reading})
			agents.ReadRules(ctx, r, "drul", dbRulesSystem)
		}
		s.Task("ddom", "Name the data domains and explain the main tables from names, keys and programs (no data).", nil)
		runDataDomains(ctx, r)
		r.Step(5, "done", fmt.Sprintf("%d rules, %d data domains", len(s.Findings("business-rule")), len(s.Findings("data-domain"))))
	} else {
		r.Step(5, "skipped", "needs Claude: "+why)
	}

	// 6 Assess
	r.Step(6, "running", "")
	s.Task("dass", "Score each data domain for migration and suggest the order and approach.", nil)
	runDBAssessment(r)
	r.Step(6, "done", "")

	// 7 Verify
	r.Step(7, "running", "")
	s.Task("ver", "Check every claim against the cited export lines.", nil)
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
	s.Task("rep", "Write the Database analysis report from verified findings.", nil)
	r.Report = runDBReport(ctx, r)
	r.Step(8, "done", "")

	r.Step(9, "people", "Open the results in MindSuite: your DBA and experts confirm, correct or reject each finding.")
}

// chooseDBReading picks the programs where business rules most likely live.
func chooseDBReading(r *Run) {
	m := st(r).db
	jobRuns := map[string]bool{}
	for _, f := range r.Store.Findings("db-job") {
		if k, _ := f.Data["runs"].(string); k != "" {
			jobRuns[k] = true
		}
	}
	type cand struct {
		c     *DBCode
		score float64
		why   []string
	}
	var cs []cand
	for _, c := range m.sortedCode() {
		if c.End <= c.Line || c.Type == "PACKAGE" && m.Code[strings.TrimSuffix(c.Key, " (spec)")] != nil {
			continue // no body in the export
		}
		x := cand{c: c}
		if len(c.Writes) > 0 {
			x.score += 1.5 * float64(min(len(c.Writes), 4))
			x.why = append(x.why, "writes "+strings.Join(firstN(c.Writes, 3), ", "))
		}
		if c.Complexity >= 5 {
			x.score += float64(c.Complexity) / 5
			x.why = append(x.why, fmt.Sprintf("%d decision points", c.Complexity))
		}
		if c.Type == "TRIGGER" {
			x.score += 2
			x.why = append(x.why, "trigger on "+c.OnTable)
		}
		if jobRuns[c.Key] {
			x.score += 1.5
			x.why = append(x.why, "run by a scheduled job")
		}
		n := 0
		f := r.Code.File(c.File)
		for i := c.Line; f != nil && i <= c.End && i <= len(f.Lines); i++ {
			if ruleLikeRe.MatchString(f.Lines[i-1]) {
				n++
			}
		}
		if n >= 3 {
			x.score += float64(n) / 4
			x.why = append(x.why, fmt.Sprintf("%d rule-like conditions", n))
		}
		if x.score > 0 {
			cs = append(cs, x)
		}
	}
	sort.SliceStable(cs, func(i, j int) bool { return cs[i].score > cs[j].score })
	if len(cs) > r.Options.MaxModelFiles {
		r.Store.Log("orch", fmt.Sprintf("%d programs qualify for in-depth reading; the top %d are read in this run (raise the number to read more).", len(cs), r.Options.MaxModelFiles))
		cs = cs[:r.Options.MaxModelFiles]
	}
	for _, x := range cs {
		r.Reading = append(r.Reading, Reading{File: x.c.File, From: x.c.Line, To: x.c.End, Name: x.c.Name, Reason: strings.Join(x.why, ", ")})
	}
	r.Store.Log("orch", fmt.Sprintf("Reading list: %d programs", len(r.Reading)))
}

// ---------- Migration assessment ----------

func runDBAssessment(r *Run) {
	s, m := r.Store, st(r).db
	// units: data domains from the model when present, else data clusters, else schemas
	type unit struct {
		name   string
		tables []string
		ev     []string
	}
	var units []unit
	inUnit := map[string]bool{}
	for _, f := range s.Findings("data-domain") {
		if f.Status == StatusRejected {
			continue
		}
		ts := []string{}
		for _, t := range toStrings(f.Data["tables"]) {
			if k := m.Find(t); k != "" && !inUnit[k] {
				ts = append(ts, k)
				inUnit[k] = true
			}
		}
		if len(ts) > 0 {
			units = append(units, unit{f.Title, ts, []string{f.ID}})
		}
	}
	if len(units) == 0 {
		for _, f := range s.Findings("data-cluster") {
			ts := toStrings(f.Data["tables"])
			for _, t := range ts {
				inUnit[t] = true
			}
			units = append(units, unit{strings.TrimPrefix(strings.SplitN(f.Title, " (", 2)[0], "Data cluster around "), ts, []string{f.ID}})
		}
	}
	rest := map[string][]string{}
	restUnit := map[int]bool{}
	for _, t := range m.sortedTables(false) {
		if !inUnit[t.Key] {
			rest[schemaLabel(t.Schema)] = append(rest[schemaLabel(t.Schema)], t.Key)
		}
	}
	for _, sc := range sortedKeys(rest) {
		name := "Other tables in " + sc
		if len(units) == 0 {
			name = "Schema " + sc
		}
		units = append(units, unit{name, rest[sc], nil})
		restUnit[len(units)-1] = strings.HasPrefix(name, "Other tables")
	}
	unitOf := map[string]int{}
	for i, u := range units {
		for _, t := range u.tables {
			unitOf[t] = i
		}
	}
	type res struct {
		id    string
		name  string
		score int
		rest  bool
	}
	var results []res
	for i, u := range units {
		set := map[string]bool{}
		for _, t := range u.tables {
			set[t] = true
		}
		sizeMB, crossFK := 0.0, 0
		ev := append([]string{}, u.ev...)
		for _, t := range u.tables {
			tb := m.Tables[t]
			if tb == nil {
				continue
			}
			sizeMB += tb.SizeMB
			for _, fk := range tb.FKs {
				if j, ok := unitOf[fk.Ref]; ok && j != i {
					crossFK++
				}
			}
			if id := st(r).ids[t]; id != "" && len(ev) < 30 {
				ev = append(ev, id)
			}
		}
		progs, triggers, lines, dyn := 0, 0, 0, 0
		shared := 0
		for _, c := range m.Code {
			touches, outside := false, false
			for _, t := range append(append([]string{}, c.Writes...), c.Reads...) {
				if set[t] {
					touches = true
				} else {
					outside = true
				}
			}
			if c.OnTable != "" && set[c.OnTable] {
				touches = true
				triggers++
			}
			if !touches {
				continue
			}
			progs++
			lines += c.Lines
			if c.Dynamic > 0 {
				dyn++
			}
			if outside {
				shared++
			}
			if id := st(r).codeIDs[c.Key]; id != "" && len(ev) < 40 {
				ev = append(ev, id)
			}
		}
		score := 100 - min(crossFK*5, 25) - min(lines/500, 25) - min(triggers*3, 15) - min(shared*3, 15) - min(dyn*5, 10)
		if sizeMB > 100*1024 {
			score -= 10
		}
		score = max(score, 0)
		effort := "S"
		switch w := len(u.tables) + lines/200; {
		case w > 150:
			effort = "XL"
		case w > 60:
			effort = "L"
		case w > 15:
			effort = "M"
		}
		band := "easy to move on its own"
		switch {
		case score < 40:
			band = "tightly coupled: move late or together with its neighbours"
		case score < 70:
			band = "moderate: untangle the shared programs and keys first"
		}
		detail := fmt.Sprintf("%d tables%s, %d programs touch it (%d lines, %d triggers, %d also use other domains), %d keys to other domains, %d with dynamic SQL. Score %d/100: %s. Effort %s.",
			len(u.tables), map[bool]string{true: ", " + fmtMB(sizeMB), false: ""}[sizeMB > 0], progs, lines, triggers, shared, crossFK, dyn, score, band, effort)
		if len(ev) == 0 {
			ev = nil
		}
		id := s.Add(Finding{Agent: "dass", Kind: "db-assessment", Module: u.name, Title: "Migration: " + u.name, Detail: detail, Evidence: ev, Basis: BasisInferred, Confidence: "medium",
			Data: map[string]any{"tables": len(u.tables), "programs": progs, "lines": lines, "triggers": triggers, "crossKeys": crossFK, "sharedPrograms": shared, "dynamic": dyn, "sizeMb": sizeMB, "score": score, "effort": effort, "band": band}})
		if len(ev) == 0 {
			s.Verify(id, StatusConflict, "no table findings to cite (export without table definitions)")
		}
		results = append(results, res{id, u.name, score, restUnit[i]})
	}
	// real domains first (most self-contained first); tables without relationships last
	sort.SliceStable(results, func(i, j int) bool {
		if results[i].rest != results[j].rest {
			return !results[i].rest
		}
		return results[i].score > results[j].score
	})
	order := []string{}
	for i, x := range results {
		note := ""
		if x.rest {
			note = " — tables without foreign keys: move them with the domain whose programs use them"
		}
		order = append(order, fmt.Sprintf("%d. %s (%d/100)%s", i+1, x.name, x.score, note))
	}
	if len(order) > 0 {
		s.Add(Finding{Agent: "dass", Kind: "migration-order", Title: "Suggested order: " + results[0].name + " first",
			Detail: "Start with the most self-contained domain and move the tightly coupled ones last:\n" + strings.Join(firstN(order, 30), "\n"), Basis: BasisInferred, Confidence: "medium"})
	}
	codeLines, features, high := 0, 0, 0
	for _, c := range m.Code {
		codeLines += c.Lines
	}
	for _, f := range s.Findings("vendor-feature") {
		features += toInt(f.Data["count"])
		if f.Data["effort"] == "high" {
			high++
		}
	}
	approach, why := "", []string{}
	why = append(why, fmt.Sprintf("%d lines of stored code", codeLines), fmt.Sprintf("%d uses of engine-specific features (%d kinds with high conversion effort)", features, high),
		fmt.Sprintf("%d database links", len(m.Links)), fmt.Sprintf("%d scheduled jobs", len(m.Jobs)))
	switch {
	case codeLines > 20000 || high >= 2:
		approach = "Move first, then extract: keep the same engine (newer version or managed service), then take the business logic out of the stored code domain by domain"
	case codeLines > 3000 || features > 50:
		approach = "Convert with a schema-conversion tool, then rewrite the stored code that does not convert, domain by domain"
	default:
		approach = "Convert directly: little stored code and few engine-specific features, so a move to another engine (e.g. PostgreSQL) is mostly automatic"
	}
	s.Add(Finding{Agent: "dass", Kind: "migration-approach", Title: approach, Detail: "Based on " + strings.Join(why, ", ") + ". Confirm with your DBA and the target platform owner.",
		Basis: BasisInferred, Confidence: "medium", Data: map[string]any{"codeLines": codeLines, "vendorUses": features, "highEffortKinds": high, "links": len(m.Links), "jobs": len(m.Jobs)}})
}
