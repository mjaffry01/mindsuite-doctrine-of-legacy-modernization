package code

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha1"
	"fmt"
	"math"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"
)

// ---------- 7 Quality & change safety ----------

type fileScore struct {
	f          *File
	complexity int
	score      float64
}

var (
	branchRe  = regexp.MustCompile(`\b(if|else\s+if|elif|for|foreach|while|case|catch|except|when)\b|&&|\|\||\?\s*[^:]+:`)
	wordRe    = regexp.MustCompile(`[A-Za-z_]\w{2,}`)
	declRe    = regexp.MustCompile(`^\s*(?:public\s+|internal\s+|protected\s+|private\s+)?(?:abstract\s+|final\s+|static\s+|sealed\s+|partial\s+)*(?:class|interface|enum|record)\s+([A-Z]\w*)`)
	wiredRe   = regexp.MustCompile(`@(RestController|Controller|Component|Service|Repository|Configuration|SpringBootApplication|Entity|WebServlet|Path|Named|Singleton|Stateless|MessageDriven|WebService|Mapper|ControllerAdvice)\b|\[ApiController\]|: *(Controller|ControllerBase|Hub|BackgroundService|DbContext|Page)\b|static\s+void\s+[Mm]ain\s*\(|extends\s+(HttpServlet|Action|TestCase)\b`)
	trivialRe = regexp.MustCompile(`^\s*([{}()\[\];,]*|import .*|using .*|package .*|from .* import .*|#include.*|//.*|/?\*.*|@\w+(\(.*\))?|return;|break;|else|try|finally|end|\}\s*else\s*\{|<\/?\w+>)\s*$`)
)

func runQuality(r *Run) {
	s := r.Store
	c := r.Code
	type mod struct{ code, tests, testLines, codeLines int }
	mods := map[string]*mod{}
	var scores []fileScore
	for _, f := range c.Files {
		if !CodeLangs[f.Lang] && !LegacyLangs[f.Lang] {
			continue
		}
		r.Cover("qua", f)
		m := mods[f.Module]
		if m == nil {
			m = &mod{}
			mods[f.Module] = m
		}
		if f.IsTest {
			m.tests++
			m.testLines += len(f.Lines)
			continue
		}
		m.code++
		m.codeLines += len(f.Lines)
		cx := 0
		for _, l := range f.Lines {
			cx += len(branchRe.FindAllStringIndex(l, -1))
		}
		scores = append(scores, fileScore{f, cx, float64(cx) * math.Log10(float64(len(f.Lines))+10)})
	}
	for _, name := range sortedKeys(mods) {
		m := mods[name]
		if m.code == 0 {
			continue
		}
		ratio := float64(m.testLines) / float64(max(1, m.codeLines))
		conf, detail := "high", ""
		if m.tests == 0 {
			detail = "No automated tests: changes to this module cannot be checked automatically. Add characterisation tests before modernizing it."
		}
		s.Add(Finding{Agent: "qua", Kind: "module-tests", Module: name, Title: fmt.Sprintf("%s: %d test files for %d code files (test lines %.0f%% of code lines)", name, m.tests, m.code, ratio*100), Detail: detail,
			Confidence: conf, Data: map[string]any{"testFiles": m.tests, "codeFiles": m.code, "testRatio": math.Round(ratio*100) / 100}})
	}
	sort.Slice(scores, func(i, j int) bool { return scores[i].score > scores[j].score })
	for i, fs := range scores {
		if i >= 15 || fs.complexity < 15 {
			break
		}
		s.Add(Finding{Agent: "qua", Kind: "hotspot", Module: fs.f.Module, Title: fmt.Sprintf("Complexity hotspot: %s (%d decision points, %d lines)", fs.f.Path, fs.complexity, len(fs.f.Lines)),
			Citations: []Citation{{File: fs.f.Path, Line: 1}}, Confidence: "high", Data: map[string]any{"complexity": fs.complexity, "lines": len(fs.f.Lines), "rank": i + 1}})
	}
	duplication(r)
	deadCode(r)
	if r.Options.GitHistory && r.Code.LocalDir != "" && r.Code.HasGit {
		gitHistory(r, scores)
	} else {
		s.Log("qua", "Git history not read (needs a local folder with .git and the Git history option). Churn hotspots and bus factor are a gap.")
	}
}

// duplication finds identical blocks of 6 meaningful lines in different files.
func duplication(r *Run) {
	const win = 6
	type loc struct {
		f    *File
		line int
	}
	seen := map[[20]byte][]loc{}
	for _, f := range r.Code.Files {
		if !CodeLangs[f.Lang] || f.IsTest {
			continue
		}
		var lines []string
		var nums []int
		for i, l := range f.Lines {
			t := strings.Join(strings.Fields(l), " ")
			if len(t) < 8 || trivialRe.MatchString(t) {
				continue
			}
			lines = append(lines, t)
			nums = append(nums, i+1)
		}
		last := -win
		for i := 0; i+win <= len(lines); i++ {
			h := sha1.Sum([]byte(strings.Join(lines[i:i+win], "\n")))
			if i-last < win && len(seen[h]) > 0 && seen[h][len(seen[h])-1].f == f {
				continue
			}
			seen[h] = append(seen[h], loc{f, nums[i]})
			last = i
		}
	}
	type dup struct {
		locs []loc
	}
	var dups []dup
	files := map[string]bool{}
	for _, l := range seen {
		distinct := map[string]bool{}
		for _, x := range l {
			distinct[x.f.Path] = true
		}
		if len(distinct) >= 2 {
			key := ""
			for _, p := range sortedKeys(distinct) {
				key += p + "|"
			}
			if files[key] {
				continue // one finding per set of files
			}
			files[key] = true
			dups = append(dups, dup{l})
		}
	}
	sort.Slice(dups, func(i, j int) bool { return len(dups[i].locs) > len(dups[j].locs) })
	for i, d := range dups {
		if i >= 20 {
			break
		}
		var cits []Citation
		mods := map[string]bool{}
		for j, x := range d.locs {
			if j < 4 {
				cits = append(cits, Citation{File: x.f.Path, Line: x.line, EndLine: x.line + win})
			}
			mods[x.f.Module] = true
		}
		r.Store.Add(Finding{Agent: "qua", Kind: "duplication", Module: d.locs[0].f.Module, Title: fmt.Sprintf("Duplicated code in %d places (%s)", len(d.locs), strings.Join(sortedKeys(mods), ", ")),
			Detail: "The same logic exists in several places; a change must be made in all of them.", Citations: cits, Confidence: "high"})
	}
	r.Store.Log("qua", fmt.Sprintf("%d duplicated blocks found.", len(dups)))
}

// deadCode lists classes no other file mentions (candidates only: reflection and configuration can still use them).
func deadCode(r *Run) {
	refs := map[string]map[string]bool{}
	for _, f := range r.Code.Files {
		if f.Lang == "Markdown" || f.Lang == "Text" {
			continue
		}
		for _, l := range f.Lines {
			for _, w := range wordRe.FindAllString(l, -1) {
				if refs[w] == nil {
					refs[w] = map[string]bool{}
				}
				refs[w][f.Path] = true
			}
		}
	}
	n := 0
	for _, f := range r.Code.Files {
		if f.IsTest || !(f.Lang == "Java" || f.Lang == "C#" || f.Lang == "Kotlin" || f.Lang == "Scala") {
			continue
		}
		text := strings.Join(f.Lines, "\n")
		if wiredRe.MatchString(text) {
			continue
		}
		for i, l := range f.Lines {
			m := declRe.FindStringSubmatch(l)
			if m == nil {
				continue
			}
			used := false
			for p := range refs[m[1]] {
				if p != f.Path {
					used = true
					break
				}
			}
			if !used {
				n++
				if n <= 40 {
					r.Store.Add(Finding{Agent: "qua", Kind: "dead-code-candidate", Module: f.Module, Title: "Possibly unused: " + m[1], Detail: "No other file mentions this type. Check reflection, configuration and external callers before removing it.",
						Citations: []Citation{cite(f, i+1)}, Basis: BasisInferred, Confidence: "low"})
				}
			}
			break
		}
	}
	r.Store.Log("qua", fmt.Sprintf("%d dead-code candidates.", n))
}

// gitHistory reads `git log` read-only: churn per file and authors per module.
func gitHistory(r *Run, scores []fileScore) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "-C", r.Code.LocalDir, "log", "--since=24.months", "--no-merges", "--numstat", "--format=@@%an")
	out, err := cmd.Output()
	if err != nil {
		r.Store.Log("qua", "Git history could not be read: "+err.Error())
		return
	}
	commits := map[string]int{}
	authors := map[string]map[string]int{}
	author := ""
	sc := bufio.NewScanner(bytes.NewReader(out))
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		l := sc.Text()
		if strings.HasPrefix(l, "@@") {
			author = strings.TrimPrefix(l, "@@")
			continue
		}
		parts := strings.Split(l, "\t")
		if len(parts) != 3 {
			continue
		}
		f := r.Code.File(parts[2])
		if f == nil {
			continue
		}
		commits[f.Path]++
		if authors[f.Module] == nil {
			authors[f.Module] = map[string]int{}
		}
		authors[f.Module][author]++
	}
	cx := map[string]int{}
	for _, s := range scores {
		cx[s.f.Path] = s.complexity
	}
	files := sortedKeys(commits)
	sort.Slice(files, func(i, j int) bool {
		return commits[files[i]]*(cx[files[i]]+1) > commits[files[j]]*(cx[files[j]]+1)
	})
	for i, p := range files {
		if i >= 10 {
			break
		}
		f := r.Code.File(p)
		r.Store.Add(Finding{Agent: "qua", Kind: "churn-hotspot", Module: f.Module, Title: fmt.Sprintf("Changed often and complex: %s (%d commits in 24 months)", p, commits[p]),
			Citations: []Citation{{File: p, Line: 1}}, Basis: BasisHistory, Confidence: "high", Data: map[string]any{"commits": commits[p], "complexity": cx[p]}})
	}
	for _, m := range sortedKeys(authors) {
		a := authors[m]
		total := 0
		names := sortedKeys(a)
		for _, n := range names {
			total += a[n]
		}
		sort.Slice(names, func(i, j int) bool { return a[names[i]] > a[names[j]] })
		cum, bus := 0, 0
		for _, n := range names {
			cum += a[n]
			bus++
			if cum*2 >= total {
				break
			}
		}
		detail := ""
		if bus == 1 {
			detail = "One person made most of the changes: knowledge risk. Involve them early."
		}
		r.Store.Add(Finding{Agent: "qua", Kind: "bus-factor", Module: m, Title: fmt.Sprintf("%s: bus factor %d (%d people changed it)", m, bus, len(names)), Detail: detail, Basis: BasisHistory, Confidence: "medium",
			Data: map[string]any{"busFactor": bus, "people": len(names)}})
	}
}
