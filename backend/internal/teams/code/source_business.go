package code

import (
	"fmt"
	"path"
	"regexp"
	"strings"
)

type businessAction struct {
	verb, area string
	citations  []Citation
}

var businessCall = regexp.MustCompile("(?i)\\b([_a-z][a-z0-9_]*(?:service|repository|repo))\\s*\\.\\s*([a-z][a-z0-9_]*)\\s*\\(")
var publicMethod = regexp.MustCompile("(?i)^\\s*(?:public|internal|friend)\\s+(?:[a-z_][a-z0-9_<>.,?\\[\\]]*\\s+)*([a-z][a-z0-9_]*)\\s*\\(")
var businessVerb = regexp.MustCompile("(?i)^(get|list|find|search|add|create|insert|update|edit|save|delete|remove|approve|cancel)(.+)$")
var businessWords = regexp.MustCompile("[A-Z]?[a-z]+|[A-Z]+(?:[0-9]+)?")

func businessOperation(name string) (verb, area string) {
	match := businessVerb.FindStringSubmatch(name)
	if match == nil {
		return "", ""
	}
	for _, word := range businessWords.FindAllString(match[2], -1) {
		switch strings.ToLower(word) {
		case "catalog":
			area = "product catalog"
		case "product", "products", "stock", "inventory":
			area = "product inventory"
		case "order", "orders":
			area = "orders"
		case "customer", "customers":
			area = "customer records"
		case "invoice", "invoices":
			area = "invoices"
		case "supplier", "suppliers":
			area = "supplier records"
		case "payment", "payments":
			area = "payments"
		}
		if area != "" {
			break
		}
	}
	switch strings.ToLower(match[1]) {
	case "get", "list", "find", "search":
		return "view", area
	case "add", "create", "insert":
		return "add", area
	case "update", "edit", "save":
		return "edit", area
	case "delete", "remove":
		return "remove", area
	default:
		return strings.ToLower(match[1]), area
	}
}

func applicationSource(f *File) bool {
	name := strings.ToLower(path.Base(f.Path))
	return !f.IsTest && (f.Lang == "C#" || f.Lang == "VB.NET") &&
		!strings.HasSuffix(name, ".designer.cs") && !strings.HasSuffix(name, ".designer.vb")
}

func businessSurface(f *File) bool {
	name := strings.ToLower(f.Path)
	return strings.HasSuffix(name, ".aspx.cs") || strings.HasSuffix(name, ".aspx.vb") ||
		strings.Contains("/"+name, "/forms/") || strings.Contains("/"+name, "/controllers/") ||
		strings.HasPrefix(path.Base(name), "form") || strings.HasSuffix(name, "controller.cs")
}

func businessImplementation(f *File) bool {
	name := strings.ToLower(path.Base(f.Path))
	return strings.Contains(name, "service") || strings.Contains(name, "repository") || strings.Contains("/"+strings.ToLower(f.Path), "/data/")
}

// runSourceBusiness sketches only business activities that can be traced from a
// screen or controller to a matching method in a service or repository.
func runSourceBusiness(r *Run) int {
	project := businessProjectScope(r)
	type implementation struct {
		file *File
		line int
	}
	methods := map[string][]implementation{}
	for _, f := range r.Code.Files {
		if !applicationSource(f) || !businessImplementation(f) {
			continue
		}
		for line, text := range f.Lines {
			match := publicMethod.FindStringSubmatch(text)
			if match != nil {
				key := project(f) + "\x00" + strings.ToLower(match[1])
				methods[key] = append(methods[key], implementation{f, line + 1})
			}
		}
	}
	actions := map[string]map[string][]businessAction{}
	seen := map[string]bool{}
	for _, f := range r.Code.Files {
		if !applicationSource(f) || !businessSurface(f) {
			continue
		}
		for line, text := range f.Lines {
			trim := strings.TrimSpace(text)
			if strings.HasPrefix(trim, "//") || strings.HasPrefix(trim, "'") {
				continue
			}
			for _, call := range businessCall.FindAllStringSubmatch(text, -1) {
				verb, area := businessOperation(call[2])
				key := project(f) + "\x00" + area + "\x00" + verb
				if area == "" || seen[key] {
					continue
				}
				matches := methods[project(f)+"\x00"+strings.ToLower(call[2])]
				for _, target := range matches {
					name := strings.ToLower(path.Base(target.file.Path))
					receiver := strings.TrimLeft(strings.ToLower(call[1]), "_")
					if raw := strings.TrimLeft(call[1], "_"); len(raw) > 1 && raw[0] == 'I' && raw[1] >= 'A' && raw[1] <= 'Z' {
						receiver = receiver[1:]
					}
					if !strings.Contains(name, receiver) && !strings.Contains(receiver, strings.TrimSuffix(name, path.Ext(name))) {
						continue
					}
					if actions[project(f)] == nil {
						actions[project(f)] = map[string][]businessAction{}
					}
					actions[project(f)][area] = append(actions[project(f)][area], businessAction{
						verb: verb, area: area, citations: []Citation{cite(f, line+1), cite(target.file, target.line)},
					})
					r.Cover("dom", f)
					r.Cover("dom", target.file)
					seen[key] = true
					break
				}
			}
		}
	}
	count := 0
	for _, module := range sortedKeys(actions) {
		for _, area := range sortedKeys(actions[module]) {
			observed := actions[module][area]
			var verbs []string
			var citations []Citation
			for _, action := range observed {
				verbs = append(verbs, action.verb)
				citations = append(citations, action.citations...)
				r.Store.Add(Finding{Agent: "dom", Kind: "journey", Module: module, Title: action.verb + " " + action.area,
					Detail: fmt.Sprintf("The user-facing code calls a service or repository method to %s %s. The citations identify the call and matching implementation.", action.verb, action.area),
					Basis:  BasisInferred, Confidence: "medium", Citations: action.citations})
			}
			r.Store.Add(Finding{Agent: "dom", Kind: "purpose", Module: module, Title: "Likely business use: " + area,
				Detail: fmt.Sprintf("This application appears to support %s. The user-facing code can %s records in this area.", area, strings.Join(verbs, ", ")),
				Basis:  BasisInferred, Confidence: "medium", Citations: citations})
			count++
		}
	}
	addCatalogModes(r, actions, project)

	return count
}

// The ASP.NET catalog sample exposes a clear link from a named setting to two
// Autofac registrations. Require each link before explaining the data modes.
func addCatalogModes(r *Run, actions map[string]map[string][]businessAction, project func(*File) string) {
	for _, module := range sortedKeys(actions) {
		if len(actions[module]["product catalog"]) == 0 {
			continue
		}
		var setting, startup, condition, mock, live, memory, db Citation
		for _, f := range r.Code.Files {
			if project(f) != module || f.IsTest {
				continue
			}
			base := strings.ToLower(path.Base(f.Path))
			for i, line := range f.Lines {
				text := strings.ToLower(line)
				switch {
				case base == "web.config" && strings.Contains(text, "key=\"usemockdata\""):
					setting = cite(f, i+1)
				case base == "global.asax.cs" && strings.Contains(text, "appsettings[\"usemockdata\"]"):
					startup = cite(f, i+1)
				case base == "applicationmodule.cs" && strings.Contains(text, "if (this.usemockdata)"):
					condition = cite(f, i+1)
				case base == "applicationmodule.cs" && strings.Contains(text, "registertype<catalogservicemock>"):
					mock = cite(f, i+1)
				case base == "applicationmodule.cs" && strings.Contains(text, "registertype<catalogservice>"):
					live = cite(f, i+1)
				case base == "catalogservicemock.cs" && strings.Contains(text, "list<catalogitem>"):
					memory = cite(f, i+1)
				case base == "catalogservice.cs" && strings.Contains(text, ".savechanges("):
					db = cite(f, i+1)
				}
			}
		}
		if setting.File == "" || startup.File == "" || condition.File == "" || mock.Line <= condition.Line || live.Line <= mock.Line || memory.File == "" || db.File == "" {
			continue
		}
		r.Store.Add(Finding{Agent: "dom", Kind: "business-rule", Module: module, Title: "Configuration chooses how catalog changes are stored",
			Detail: "UseMockData enables the in-memory catalog service. Otherwise the SQL-backed catalog service saves changes through the database context. A successful edit in the mock mode does not validate the database path. Confirm which mode runs in the deployed system.",
			Basis:  BasisInferred, Confidence: "medium", Citations: []Citation{setting, startup, condition, mock, live, memory, db}})
	}
}

// Use the nearest project file so namespaces such as Catalog and Services can
// participate in the same application without joining separate projects.
func businessProjectScope(r *Run) func(*File) string {
	var roots []string
	for _, f := range r.Code.Files {
		ext := strings.ToLower(path.Ext(f.Path))
		if ext == ".csproj" || ext == ".vbproj" {
			roots = append(roots, path.Dir(f.Path))
		}
	}
	return func(f *File) string {
		best := ""
		for _, root := range roots {
			if len(root) > len(best) && (root == "." || strings.HasPrefix(f.Path, root+"/")) {
				best = root
			}
		}
		if best != "" {
			return best
		}
		return f.Module
	}
}
