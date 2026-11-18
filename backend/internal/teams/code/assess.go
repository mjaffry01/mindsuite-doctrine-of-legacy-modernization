package code

import (
	"fmt"
	"path"
	"sort"
	"strings"
)

// ---------- 12 Modernization assessment ----------

type modMetrics struct {
	name                     string
	lines, fanIn, fanOut     int
	inCycle                  bool
	shared, endpoints, msgs  int
	tests                    bool
	hotspots, secrets, rules int
	evidence                 []string
	score                    int
	effort                   string
	reasons                  []string
}

func runAssessment(r *Run) {
	s := r.Store
	mm := map[string]*modMetrics{}
	get := func(m string) *modMetrics {
		if m == "" {
			return nil
		}
		x := mm[m]
		if x == nil {
			x = &modMetrics{name: m}
			mm[m] = x
		}
		return x
	}
	for _, f := range s.Findings() {
		switch f.Kind {
		case "module":
			m := get(f.Module)
			m.lines = toInt(f.Data["lines"])
			m.evidence = append(m.evidence, f.ID)
		case "module-edge":
			if a := get(fmt.Sprint(f.Data["from"])); a != nil {
				a.fanOut++
			}
			if b := get(fmt.Sprint(f.Data["to"])); b != nil {
				b.fanIn++
				b.evidence = append(b.evidence, f.ID)
			}
		case "cycle":
			for _, m := range toStrings(f.Data["modules"]) {
				x := get(m)
				x.inCycle = true
				x.evidence = append(x.evidence, f.ID)
			}
		case "shared-table":
			for _, m := range toStrings(f.Data["modules"]) {
				x := get(m)
				x.shared++
				x.evidence = append(x.evidence, f.ID)
			}
		case "endpoint":
			if x := get(f.Module); x != nil {
				x.endpoints++
			}
		case "messaging":
			if x := get(f.Module); x != nil {
				x.msgs++
			}
		case "module-tests":
			if x := get(f.Module); x != nil {
				x.tests = toInt(f.Data["testFiles"]) > 0
				x.evidence = append(x.evidence, f.ID)
			}
		case "hotspot":
			if x := get(f.Module); x != nil {
				x.hotspots++
				x.evidence = append(x.evidence, f.ID)
			}
		case "secret":
			if x := get(f.Module); x != nil {
				x.secrets++
			}
		case "business-rule":
			if x := get(f.Module); x != nil {
				x.rules++
			}
		}
	}
	legacyCoupling(r, get)
	var list []*modMetrics
	for _, name := range sortedKeys(mm) {
		m := mm[name]
		if m.lines == 0 {
			continue
		}
		score := 100
		pen := func(p int, why string) {
			if p > 0 {
				score -= p
				m.reasons = append(m.reasons, why)
			}
		}
		pen(min(30, m.fanIn*6), fmt.Sprintf("%d modules depend on it", m.fanIn))
		pen(min(20, m.fanOut*4), fmt.Sprintf("it depends on %d modules", m.fanOut))
		if m.inCycle {
			pen(20, "part of a dependency cycle")
		}
		pen(min(24, m.shared*8), fmt.Sprintf("shares %d tables with other modules", m.shared))
		if !m.tests {
			pen(12, "no automated tests")
		}
		pen(min(12, m.hotspots*4), fmt.Sprintf("%d complexity hotspots", m.hotspots))
		m.score = max(0, score)
		switch {
		case m.lines < 2000:
			m.effort = "S"
		case m.lines < 10000:
			m.effort = "M"
		case m.lines < 30000:
			m.effort = "L"
		default:
			m.effort = "XL"
		}
		if m.score < 40 && m.effort != "XL" {
			m.effort = map[string]string{"S": "M", "M": "L", "L": "XL"}[m.effort]
		}
		list = append(list, m)
	}
	sort.SliceStable(list, func(i, j int) bool { return list[i].score > list[j].score })
	order := []string{}
	for i, m := range list {
		seams := []string{}
		if m.endpoints > 0 {
			seams = append(seams, fmt.Sprintf("an API facade in front of its %d endpoints", m.endpoints))
		}
		if m.msgs > 0 {
			seams = append(seams, "its message topics/queues")
		}
		if m.shared > 0 {
			seams = append(seams, "a data seam: agree table ownership first")
		}
		if len(seams) == 0 {
			seams = append(seams, "its public classes/functions used by other modules")
		}
		band := "start here"
		if m.score < 70 {
			band = "prepare first"
		}
		if m.score < 40 {
			band = "hard: untangle before moving"
		}
		reasons := "no blocking coupling found"
		if len(m.reasons) > 0 {
			reasons = strings.Join(m.reasons, "; ")
		}
		s.Add(Finding{Agent: "ass", Kind: "assessment", Module: m.name, Title: fmt.Sprintf("%s: extractability %d/100 (%s), effort %s", m.name, m.score, band, m.effort),
			Detail: "Why: " + reasons + ". Where to cut: " + strings.Join(seams, ", ") + ".", Evidence: uniq(m.evidence), Basis: BasisInferred, Confidence: "medium",
			Data: map[string]any{"score": m.score, "effort": m.effort, "band": band, "order": i + 1, "fanIn": m.fanIn, "fanOut": m.fanOut, "inCycle": m.inCycle, "sharedTables": m.shared, "tests": m.tests, "endpoints": m.endpoints, "rules": m.rules, "seams": seams}})
		order = append(order, m.name)
	}
	if len(order) > 0 {
		s.Add(Finding{Agent: "ass", Kind: "extraction-order", Title: "Suggested order: " + strings.Join(order, " → "), Detail: "Easiest and least coupled first; core and tangled parts later, once the team has learned from the first moves.",
			Basis: BasisInferred, Confidence: "medium", Data: map[string]any{"order": order}})
	}
}

// legacyCoupling adds the links between modules that COBOL shows: a program using a copybook kept in
// another module, calling a program of another module (CALL, CICS LINK / XCTL), or reading another
// module's IMS database. Mainframe code has no imports, so without this every module looks independent.
func legacyCoupling(r *Run, get func(string) *modMetrics) {
	owner := map[string]string{} // copybook or program name -> module
	for _, f := range r.Code.Files {
		if f.Lang == "COBOL copybook" || f.Lang == "COBOL" || f.Lang == "IMS definition" {
			name := strings.ToUpper(strings.TrimSuffix(path.Base(f.Path), path.Ext(f.Path)))
			if _, dup := owner[name]; !dup {
				owner[name] = f.Module
			}
		}
	}
	for _, f := range r.Store.Findings("legacy-program") {
		if p := strings.ToUpper(fmt.Sprint(f.Data["program"])); p != "" && p != "<NIL>" {
			owner[p] = f.Module
		}
	}
	seen := map[string]bool{}
	for _, f := range r.Store.Findings("copybook-use", "legacy-call", "cics", "ims-use") {
		target := ""
		switch f.Kind {
		case "copybook-use":
			target = fmt.Sprint(f.Data["copybook"])
		case "legacy-call":
			target = fmt.Sprint(f.Data["target"])
		case "ims-use":
			target = fmt.Sprint(f.Data["dbd"])
		case "cics":
			if c := strings.ToUpper(fmt.Sprint(f.Data["command"])); c == "LINK" || c == "XCTL" {
				target = fmt.Sprint(f.Data["target"])
			}
		}
		to := owner[strings.ToUpper(target)]
		if to == "" || to == f.Module || seen[f.Module+">"+to] {
			continue
		}
		seen[f.Module+">"+to] = true
		if a := get(f.Module); a != nil {
			a.fanOut++
			a.evidence = append(a.evidence, f.ID)
		}
		if b := get(to); b != nil {
			b.fanIn++
			b.evidence = append(b.evidence, f.ID)
		}
	}
}
