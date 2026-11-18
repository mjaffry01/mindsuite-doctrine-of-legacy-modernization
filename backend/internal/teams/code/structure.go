package code

import (
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"
)

// ---------- 3 Structure ----------

var (
	javaImportRe = regexp.MustCompile(`^\s*import\s+(?:static\s+)?([\w.]+)\s*;`)
	csUsingRe    = regexp.MustCompile(`^\s*using\s+(?:static\s+)?([\w.]+)\s*;`)
	jsImportRe   = regexp.MustCompile(`(?:import\s[^'"]*from\s*|import\s*\(|require\s*\()\s*['"](\.{1,2}/[^'"]+)['"]`)
	pyImportRe   = regexp.MustCompile(`^\s*(?:from\s+([\w.]+)\s+import|import\s+([\w.]+))`)
	goImportRe   = regexp.MustCompile(`^\s*(?:import\s+)?(?:\w+\s+)?"([\w.\-/]+)"\s*$`)
)

type edge struct {
	from, to string
	count    int
	file     *File
	line     int
}

// moduleGraph builds module → module edges from imports.
func moduleGraph(r *Run) map[[2]string]*edge {
	c := r.Code
	mods := map[string]bool{}
	for _, m := range c.Modules() {
		mods[m] = true
	}
	byNoExt := map[string]*File{}
	for _, f := range c.Files {
		if f.Lang == "JavaScript" || f.Lang == "TypeScript" {
			byNoExt[strings.TrimSuffix(f.Path, path.Ext(f.Path))] = f
		}
	}
	edges := map[[2]string]*edge{}
	add := func(f *File, n int, to string) {
		if to == "" || to == f.Module || !mods[to] {
			return
		}
		k := [2]string{f.Module, to}
		e := edges[k]
		if e == nil {
			e = &edge{from: f.Module, to: to, file: f, line: n}
			edges[k] = e
		}
		e.count++
	}
	for _, f := range c.Files {
		if !CodeLangs[f.Lang] {
			continue
		}
		r.Cover("str", f)
		for i, l := range f.Lines {
			n := i + 1
			switch f.Lang {
			case "Java", "Kotlin", "Scala", "Groovy":
				if m := javaImportRe.FindStringSubmatch(l); m != nil {
					pkg := m[1]
					if j := strings.LastIndex(pkg, "."); j > 0 {
						pkg = pkg[:j]
					}
					add(f, n, c.ModuleOfPkg(pkg))
				}
			case "C#":
				if m := csUsingRe.FindStringSubmatch(l); m != nil {
					add(f, n, c.ModuleOfPkg(m[1]))
				}
			case "JavaScript", "TypeScript":
				for _, m := range jsImportRe.FindAllStringSubmatch(l, -1) {
					target := path.Clean(path.Join(path.Dir(f.Path), m[1]))
					target = strings.TrimSuffix(target, path.Ext(target))
					if t := byNoExt[target]; t != nil {
						add(f, n, t.Module)
					} else if t := byNoExt[target+"/index"]; t != nil {
						add(f, n, t.Module)
					}
				}
			case "Python":
				if m := pyImportRe.FindStringSubmatch(l); m != nil {
					name := m[1] + m[2]
					parts := strings.Split(strings.TrimLeft(name, "."), ".")
					for _, p := range parts {
						if mods[p] {
							add(f, n, p)
							break
						}
					}
				}
			case "Go":
				if m := goImportRe.FindStringSubmatch(l); m != nil {
					segs := strings.Split(m[1], "/")
					for j := len(segs) - 1; j >= 0; j-- {
						if mods[segs[j]] {
							add(f, n, segs[j])
							break
						}
					}
				}
			}
		}
	}
	return edges
}

func runStructure(r *Run) {
	s := r.Store
	c := r.Code
	type stats struct {
		files, lines, tests int
		langs               map[string]int
	}
	ms := map[string]*stats{}
	for _, f := range c.Files {
		if !(CodeLangs[f.Lang] || LegacyLangs[f.Lang]) {
			continue
		}
		st := ms[f.Module]
		if st == nil {
			st = &stats{langs: map[string]int{}}
			ms[f.Module] = st
		}
		st.files++
		st.lines += len(f.Lines)
		st.langs[f.Lang]++
		if f.IsTest {
			st.tests++
		}
	}
	for _, m := range sortedKeys(ms) {
		st := ms[m]
		s.Add(Finding{Agent: "str", Kind: "module", Module: m, Title: fmt.Sprintf("Module %s: %d files, %d lines", m, st.files, st.lines), Confidence: "high",
			Data: map[string]any{"files": st.files, "lines": st.lines, "testFiles": st.tests, "languages": st.langs}})
	}
	edges := moduleGraph(r)
	keys := make([][2]string, 0, len(edges))
	for k := range edges {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i][0]+keys[i][1] < keys[j][0]+keys[j][1] })
	adj := map[string][]string{}
	for _, k := range keys {
		e := edges[k]
		adj[e.from] = append(adj[e.from], e.to)
		s.Add(Finding{Agent: "str", Kind: "module-edge", Module: e.from, Title: fmt.Sprintf("%s depends on %s (%d imports)", e.from, e.to, e.count),
			Citations: []Citation{cite(e.file, e.line)}, Confidence: "high", Data: map[string]any{"from": e.from, "to": e.to, "count": e.count}})
	}
	for _, comp := range tarjan(sortedKeys(ms), adj) {
		if len(comp) < 2 {
			continue
		}
		sort.Strings(comp)
		var cits []Citation
		for _, k := range keys {
			e := edges[k]
			if contains(comp, e.from) && contains(comp, e.to) && len(cits) < 6 {
				cits = append(cits, cite(e.file, e.line))
			}
		}
		s.Add(Finding{Agent: "str", Kind: "cycle", Title: "Dependency cycle: " + strings.Join(comp, " ↔ "), Detail: "These modules depend on each other and cannot be separated one at a time without first breaking the cycle.",
			Citations: cits, Confidence: "high", Data: map[string]any{"modules": comp}})
	}
	// architecture style from names
	layerNames := map[string]bool{"controller": true, "controllers": true, "web": true, "api": true, "rest": true, "service": true, "services": true, "repository": true, "repositories": true,
		"dao": true, "dal": true, "data": true, "forms": true, "ui": true, "views": true, "helpers": true, "model": true, "models": true, "entity": true, "entities": true, "domain": true, "dto": true, "util": true, "utils": true, "common": true, "config": true}
	layered := 0
	for m := range ms {
		if layerNames[strings.ToLower(m)] {
			layered++
		}
	}
	style, detail := "Modules by business feature", "Most modules are named after business areas; they are natural candidates for bounded contexts."
	if len(ms) > 0 && layered*2 >= len(ms) {
		style, detail = "Layered by technical layer", "Modules are technical layers (controller, service, repository…). Business areas cut across them, so bounded contexts must be found inside the layers."
	}
	if len(ms) <= 1 {
		style, detail = "Single module", "All code sits in one module; the internal structure needs a closer look to find business areas."
		if inner := innerLayers(r); inner != "" {
			style, detail = "Single module, layered inside", "All code sits in one module, split into technical layers: "+inner+". Business areas (for example one per screen or table) cut across these layers, so they must be found inside them; check with the team."
		}
	}
	legacyLines, allLines := 0, 0
	for _, f := range r.Code.Files {
		allLines += len(f.Lines)
		if LegacyLangs[f.Lang] {
			legacyLines += len(f.Lines)
		}
	}
	if allLines > 0 && legacyLines*2 >= allLines {
		style, detail = "Mainframe application folders", "Mainframe members are grouped by type (programs, copybooks, JCL, maps), so the modules here are the application folders, not business areas. Business areas must be found from the transactions, screens and programs; check this with the team."
	}
	s.Add(Finding{Agent: "str", Kind: "architecture-style", Title: style, Detail: detail, Basis: BasisInferred, Confidence: "medium", Data: map[string]any{"modules": len(ms), "layerNamed": layered}})
}

// tarjan returns strongly connected components.
func tarjan(nodes []string, adj map[string][]string) [][]string {
	index, low := map[string]int{}, map[string]int{}
	on := map[string]bool{}
	stack := []string{}
	out := [][]string{}
	i := 0
	var visit func(v string)
	visit = func(v string) {
		index[v], low[v] = i, i
		i++
		stack = append(stack, v)
		on[v] = true
		for _, w := range adj[v] {
			if _, seen := index[w]; !seen {
				visit(w)
				low[v] = min(low[v], low[w])
			} else if on[w] {
				low[v] = min(low[v], index[w])
			}
		}
		if low[v] == index[v] {
			comp := []string{}
			for {
				w := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				on[w] = false
				comp = append(comp, w)
				if w == v {
					break
				}
			}
			out = append(out, comp)
		}
	}
	for _, v := range nodes {
		if _, seen := index[v]; !seen {
			visit(v)
		}
	}
	return out
}

// innerLayers describes the technical-layer folders inside a single module ("Forms (7 files), Services (6 files) …"),
// or returns "" when the folders are not layer names.
func innerLayers(r *Run) string {
	layer := map[string]bool{"controller": true, "controllers": true, "web": true, "api": true, "service": true, "services": true, "repository": true, "repositories": true,
		"dao": true, "dal": true, "data": true, "dataaccess": true, "forms": true, "ui": true, "views": true, "pages": true, "screens": true, "model": true, "models": true,
		"entity": true, "entities": true, "domain": true, "dto": true, "helpers": true, "util": true, "utils": true, "common": true, "business": true, "bll": true}
	count := map[string]int{}
	other := 0
	for _, f := range r.Code.Files {
		if !CodeLangs[f.Lang] || f.IsTest {
			continue
		}
		parts := strings.Split(f.Path, "/")
		i := indexOf(parts, f.Module)
		if i < 0 || i+2 >= len(parts) {
			continue
		}
		if sub := parts[i+1]; layer[strings.ToLower(sub)] {
			count[sub]++
		} else {
			other++
		}
	}
	if len(count) < 2 || other > len(count)*3 {
		return ""
	}
	var out []string
	for _, k := range sortedKeys(count) {
		out = append(out, fmt.Sprintf("%s (%d files)", k, count[k]))
	}
	return strings.Join(out, ", ")
}

func indexOf(xs []string, x string) int {
	for i, v := range xs {
		if v == x {
			return i
		}
	}
	return -1
}
