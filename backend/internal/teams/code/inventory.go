package code

import (
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"
)

// ---------- 1 Inventory ----------

var buildFiles = map[string]string{"pom.xml": "Maven", "build.gradle": "Gradle", "build.gradle.kts": "Gradle", "package.json": "npm", "requirements.txt": "pip",
	"pyproject.toml": "Python project", "setup.py": "Python setup", "go.mod": "Go modules", "gemfile": "Bundler", "composer.json": "Composer", "build.xml": "Ant", "cargo.toml": "Cargo"}

func runInventory(r *Run) {
	s := r.Store
	type agg struct{ files, lines int }
	byLang := map[string]*agg{}
	for _, f := range r.Code.Files {
		r.Cover("inv", f)
		a := byLang[f.Lang]
		if a == nil {
			a = &agg{}
			byLang[f.Lang] = a
		}
		a.files++
		a.lines += len(f.Lines)
		base := strings.ToLower(path.Base(f.Path))
		tool := buildFiles[base]
		if tool == "" && (strings.HasSuffix(base, ".csproj") || strings.HasSuffix(base, ".vbproj") || strings.HasSuffix(base, ".sln")) {
			tool = "MSBuild / .NET"
		}
		if tool != "" {
			s.Add(Finding{Agent: "inv", Kind: "build-file", Module: f.Module, Title: tool + " build file: " + f.Path, Citations: []Citation{{File: f.Path, Line: 1}}, Confidence: "high", Data: map[string]any{"tool": tool}})
		}
	}
	langs := sortedKeys(byLang)
	sort.SliceStable(langs, func(i, j int) bool { return byLang[langs[i]].lines > byLang[langs[j]].lines })
	for _, l := range langs {
		s.Add(Finding{Agent: "inv", Kind: "language", Title: fmt.Sprintf("%s: %d files, %d lines", l, byLang[l].files, byLang[l].lines), Confidence: "high",
			Data: map[string]any{"language": l, "files": byLang[l].files, "lines": byLang[l].lines}})
	}
	if r.Code.Commit != "" {
		s.Add(Finding{Agent: "inv", Kind: "version", Title: "Analysed version: branch " + r.Code.Branch + ", commit " + short(r.Code.Commit, 12), Confidence: "high",
			Data: map[string]any{"branch": r.Code.Branch, "commit": r.Code.Commit}})
	} else {
		s.Log("inv", "No Git metadata: the exact version could not be recorded. Upload a folder with .git, or note the version yourself.")
	}
	legacy := map[string]int{}
	for _, f := range r.Code.Files {
		if LegacyLangs[f.Lang] {
			legacy[f.Lang]++
		}
	}
	if len(legacy) > 0 {
		s.Handoff("inv", "Legacy languages found; route them to the legacy-language specialists.", legacy)
	}
	if len(r.Code.Skipped) > 0 {
		s.Log("inv", fmt.Sprintf("%d files were not read (binary, too large or unrecognised); they are listed as coverage gaps.", len(r.Code.Skipped)))
	}
}

// ---------- 2 Supply chain ----------

type dep struct {
	eco, name, version string
	file               *File
	line               int
}

var (
	pomDepRe     = regexp.MustCompile(`(?s)<(dependency|parent|plugin)>\s*(.*?)</(dependency|parent|plugin)>`)
	pomTagRe     = regexp.MustCompile(`<(groupId|artifactId|version)>\s*([^<\s]+)\s*</`)
	pomPropRe    = regexp.MustCompile(`<([\w.\-]+)>\s*([^<\s]+)\s*</([\w.\-]+)>`)
	gradleDepRe  = regexp.MustCompile(`(?:implementation|api|compile|runtime|compileOnly|runtimeOnly|testImplementation|classpath)\s*\(?\s*['"]([\w.\-]+):([\w.\-]+):([\w.\-${}]+)['"]`)
	reqRe        = regexp.MustCompile(`^\s*([A-Za-z0-9_.\-\[\]]+)\s*(==|>=|~=|<=)\s*([\w.\-*]+)`)
	nugetRe      = regexp.MustCompile(`<PackageReference\s+Include="([^"]+)"\s+Version="([^"]+)"`)
	packagesCfg  = regexp.MustCompile(`<package\s+id="([^"]+)"\s+version="([^"]+)"`)
	targetFwRe   = regexp.MustCompile(`<TargetFramework(?:Version)?s?>([^<]+)</`)
	goReqRe      = regexp.MustCompile(`^\s*(?:require\s+)?([\w.\-/]+\.[\w.\-/]+)\s+(v[\w.\-+]+)`)
	javaVerRe    = regexp.MustCompile(`<(java\.version|maven\.compiler\.source|maven\.compiler\.release|release)>\s*([\d.]+)\s*</`)
	gradleJavaRe = regexp.MustCompile(`sourceCompatibility\s*=\s*['"]?(?:JavaVersion\.VERSION_)?([\d._]+)`)
)

func runSupplyChain(r *Run) {
	deps := []dep{}
	for _, f := range r.Code.Files {
		base := strings.ToLower(path.Base(f.Path))
		text := strings.Join(f.Lines, "\n")
		switch {
		case base == "pom.xml":
			r.Cover("sup", f)
			props := map[string]string{}
			for _, m := range pomPropRe.FindAllStringSubmatch(text, -1) {
				if m[1] == m[3] {
					props[m[1]] = m[2]
				}
			}
			for _, loc := range pomDepRe.FindAllStringSubmatchIndex(text, -1) {
				body := text[loc[4]:loc[5]]
				var g, a, v string
				for _, t := range pomTagRe.FindAllStringSubmatch(body, -1) {
					switch t[1] {
					case "groupId":
						g = t[2]
					case "artifactId":
						a = t[2]
					case "version":
						v = t[2]
					}
				}
				if strings.HasPrefix(v, "${") {
					if pv, ok := props[strings.Trim(v, "${}")]; ok {
						v = pv
					}
				}
				if a != "" {
					deps = append(deps, dep{"Maven", g + ":" + a, v, f, lineOf(f, loc[0])})
				}
			}
			for _, m := range javaVerRe.FindAllStringSubmatchIndex(text, -1) {
				deps = append(deps, dep{"Java", "java", text[m[4]:m[5]], f, lineOf(f, m[0])})
			}
		case base == "build.gradle" || base == "build.gradle.kts":
			r.Cover("sup", f)
			for i, l := range f.Lines {
				if m := gradleDepRe.FindStringSubmatch(l); m != nil {
					deps = append(deps, dep{"Maven", m[1] + ":" + m[2], m[3], f, i + 1})
				}
				if m := gradleJavaRe.FindStringSubmatch(l); m != nil {
					deps = append(deps, dep{"Java", "java", strings.ReplaceAll(m[1], "_", "."), f, i + 1})
				}
			}
		case base == "package.json":
			r.Cover("sup", f)
			var pj map[string]any
			if json.Unmarshal([]byte(text), &pj) == nil {
				for _, sec := range []string{"dependencies", "devDependencies"} {
					if m, ok := pj[sec].(map[string]any); ok {
						for name, v := range m {
							ln := 1
							for i, l := range f.Lines {
								if strings.Contains(l, `"`+name+`"`) {
									ln = i + 1
									break
								}
							}
							deps = append(deps, dep{"npm", name, fmt.Sprint(v), f, ln})
						}
					}
				}
				if e, ok := pj["engines"].(map[string]any); ok {
					if v, ok := e["node"]; ok {
						deps = append(deps, dep{"Node.js", "node", fmt.Sprint(v), f, 1})
					}
				}
			}
		case base == "requirements.txt":
			r.Cover("sup", f)
			for i, l := range f.Lines {
				if m := reqRe.FindStringSubmatch(l); m != nil {
					deps = append(deps, dep{"PyPI", strings.ToLower(m[1]), m[3], f, i + 1})
				}
			}
		case strings.HasSuffix(base, ".csproj") || strings.HasSuffix(base, ".vbproj") || base == "packages.config":
			r.Cover("sup", f)
			for i, l := range f.Lines {
				if m := nugetRe.FindStringSubmatch(l); m != nil {
					deps = append(deps, dep{"NuGet", m[1], m[2], f, i + 1})
				}
				if m := packagesCfg.FindStringSubmatch(l); m != nil {
					deps = append(deps, dep{"NuGet", m[1], m[2], f, i + 1})
				}
				if m := targetFwRe.FindStringSubmatch(l); m != nil {
					for _, fw := range strings.Split(m[1], ";") {
						deps = append(deps, dep{".NET", "dotnet", fw, f, i + 1})
					}
				}
			}
		case base == "go.mod":
			r.Cover("sup", f)
			for i, l := range f.Lines {
				if m := goReqRe.FindStringSubmatch(l); m != nil && !strings.HasPrefix(strings.TrimSpace(l), "module") {
					deps = append(deps, dep{"Go", m[1], m[2], f, i + 1})
				}
			}
		}
	}
	risky := 0
	for _, d := range deps {
		risk, sev := depRisk(d)
		conf := "high"
		if d.version == "" || strings.Contains(d.version, "$") {
			conf = "medium"
		}
		title := d.eco + " " + d.name + " " + d.version
		data := map[string]any{"ecosystem": d.eco, "name": d.name, "version": d.version}
		kind := "dependency"
		if d.eco == "Java" || d.eco == ".NET" || d.eco == "Node.js" {
			kind = "runtime"
			title = map[string]string{"Java": "Java ", ".NET": ".NET target ", "Node.js": "Node.js "}[d.eco] + d.version
		}
		if risk != "" {
			data["risk"], data["severity"] = risk, sev
			risky++
		}
		r.Store.Add(Finding{Agent: "sup", Kind: kind, Module: d.file.Module, Title: title, Detail: risk, Data: data, Citations: []Citation{cite(d.file, d.line)}, Confidence: conf})
	}
	r.Store.Log("sup", fmt.Sprintf("%d dependencies, %d with a known support or security risk. Licences and a vulnerability database lookup (OSV) are not run: that sends dependency names outside and needs the owner's OK.", len(deps), risky))
}

type eolRule struct {
	eco, name *regexp.Regexp
	version   *regexp.Regexp
	reason    string
	sev       string
}

var eolRules = []eolRule{
	{regexp.MustCompile(`^Java$`), regexp.MustCompile(`^java$`), regexp.MustCompile(`^(1\.[1-8]|[5-8]|1[0-6])(\.|$)`), "Old Java version: public updates ended or ending; plan an upgrade to Java 21+", "high"},
	{regexp.MustCompile(`^Maven$`), regexp.MustCompile(`spring-boot`), regexp.MustCompile(`^(1\.|2\.)`), "Spring Boot 1.x/2.x is out of open-source support", "high"},
	{regexp.MustCompile(`^Maven$`), regexp.MustCompile(`springframework:spring-`), regexp.MustCompile(`^([1-4]\.|5\.[0-2]\.)`), "Old Spring Framework: out of support", "high"},
	{regexp.MustCompile(`^Maven$`), regexp.MustCompile(`^log4j:log4j$`), regexp.MustCompile(`.`), "Log4j 1.x is end of life and has known vulnerabilities", "critical"},
	{regexp.MustCompile(`^Maven$`), regexp.MustCompile(`log4j-core`), regexp.MustCompile(`^2\.(\d|1[0-6])\.`), "Log4j 2 before 2.17 (Log4Shell family of vulnerabilities)", "critical"},
	{regexp.MustCompile(`^Maven$`), regexp.MustCompile(`struts`), regexp.MustCompile(`^(1\.|2\.[0-4]\.)`), "Old Apache Struts: end of life, known vulnerabilities", "critical"},
	{regexp.MustCompile(`^Maven$`), regexp.MustCompile(`commons-collections:commons-collections`), regexp.MustCompile(`^3\.2(\.1)?$`), "commons-collections 3.2.1 deserialisation vulnerability", "high"},
	{regexp.MustCompile(`^Maven$`), regexp.MustCompile(`hibernate`), regexp.MustCompile(`^[1-4]\.`), "Old Hibernate: out of support", "medium"},
	{regexp.MustCompile(`^Maven$`), regexp.MustCompile(`javax\.`), regexp.MustCompile(`.`), "javax.* APIs: moving to Jakarta EE is needed for current app servers", "low"},
	{regexp.MustCompile(`^npm$`), regexp.MustCompile(`^angular$`), regexp.MustCompile(`.`), "AngularJS 1.x reached end of life in January 2022", "high"},
	{regexp.MustCompile(`^npm$`), regexp.MustCompile(`^jquery$`), regexp.MustCompile(`^[\^~]?(1\.|2\.|3\.[0-4]\.)`), "jQuery before 3.5 has known XSS issues", "medium"},
	{regexp.MustCompile(`^npm$`), regexp.MustCompile(`^(request|node-sass|tslint)$`), regexp.MustCompile(`.`), "Deprecated package", "low"},
	{regexp.MustCompile(`^Node\.js$`), regexp.MustCompile(`.`), regexp.MustCompile(`(^|[^\d])(0\.|[4-9]|1[0-7])(\.|$|\D)`), "Old Node.js version: out of support", "high"},
	{regexp.MustCompile(`^\.NET$`), regexp.MustCompile(`.`), regexp.MustCompile(`^(net(20|35|40|45|451|452|46|461)|v(2|3\.5|4\.0|4\.5(\.[12])?|4\.6(\.1)?)$|netcoreapp|net5\.0|net6\.0|net7\.0)`), "Out-of-support .NET version", "high"},
	{regexp.MustCompile(`^\.NET$`), regexp.MustCompile(`.`), regexp.MustCompile(`^(net4\d+|v4\.)`), ".NET Framework 4.x: Windows-only; moving to modern .NET is usually needed for cloud targets", "low"},
	{regexp.MustCompile(`^PyPI$`), regexp.MustCompile(`^django$`), regexp.MustCompile(`^(1\.|2\.|3\.[01])`), "Old Django: out of support", "high"},
}

func depRisk(d dep) (string, string) {
	for _, r := range eolRules {
		if r.eco.MatchString(d.eco) && r.name.MatchString(d.name) && r.version.MatchString(d.version) {
			return r.reason, r.sev
		}
	}
	return "", ""
}
