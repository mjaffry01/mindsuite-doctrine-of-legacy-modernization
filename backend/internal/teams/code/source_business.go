package code

import (
	"fmt"
	"path"
	"regexp"
	"strings"
)

type businessAction struct {
	verb, area, method string
	citations          []Citation
}

var businessCall = regexp.MustCompile("(?i)\\b([_a-z][a-z0-9_]*(?:service|repository|repo))\\s*\\.\\s*([a-z][a-z0-9_]*)\\s*\\(")
var publicMethod = regexp.MustCompile("(?i)^\\s*(?:public|internal|friend)\\s+(?:[a-z_][a-z0-9_<>.,?\\[\\]]*\\s+)*([a-z][a-z0-9_]*)\\s*\\(")
var businessVerb = regexp.MustCompile("(?i)^(get|list|find|search|add|create|insert|update|edit|save|delete|remove|approve|cancel|receive|adjust|ship|transfer|return|reserve|reconcile|submit|settle|inspect|grade|pack|dispatch|issue|renew|refund|post|register|schedule|enroll|process|calculate|pay|accept|reject)(.+)$")
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
		case "claim", "claims":
			area = "claims"
		case "policy", "policies":
			area = "policies"
		case "account", "accounts":
			area = "accounts"
		case "lot", "lots", "batch", "batches":
			area = "lots or batches"
		case "harvest", "crop":
			area = "harvest records"
		case "shipment", "delivery", "deliveries":
			area = "deliveries"
		case "booking", "bookings":
			area = "bookings"
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
func runSourceBusiness(r *Run) int { return traceSourceBusiness(r, nil) }

func traceSourceBusiness(r *Run, crossBatch func([]Citation) bool) int {
	project := businessProjectScope(r)
	evidence := workflowEvidence(r)
	methods := map[string][]businessMethod{}
	for _, f := range r.Code.Files {
		if !applicationSource(f) || !businessImplementation(f) {
			continue
		}
		for line, text := range f.Lines {
			match := publicMethod.FindStringSubmatch(text)
			if match != nil {
				key := project(f) + "\x00" + strings.ToLower(match[1])
				methods[key] = append(methods[key], businessMethod{f, line + 1})
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
			code := workflowCodeLine(f.Lang, text)
			for _, call := range businessCall.FindAllStringSubmatch(code, -1) {
				verb, area := businessOperation(call[2])
				key := f.Path + "\x00" + strings.ToLower(call[2])
				if area == "" || seen[key] {
					continue
				}
				matches := methods[project(f)+"\x00"+strings.ToLower(call[2])]
				target, ok := matchBusinessMethod(call[1], matches)
				if !ok {
					continue
				}
				module := project(f)
				if actions[module] == nil {
					actions[module] = map[string][]businessAction{}
				}
				action := businessAction{
					verb: verb, area: area, method: call[2], citations: []Citation{cite(f, line+1), cite(target.file, target.line)},
				}
				actions[module][area] = append(actions[module][area], action)
				r.Cover("dom", f)
				r.Cover("dom", target.file)
				seen[key] = true
				recordWorkflowTrace(r, module, action, target, methods, evidence, crossBatch)
			}
		}
	}
	if crossBatch != nil {
		return 0
	}
	count := 0
	for _, module := range sortedKeys(actions) {
		for _, area := range sortedKeys(actions[module]) {
			observed := actions[module][area]
			var verbs []string
			var citations []Citation
			for _, action := range observed {
				verbs = appendUnique(verbs, action.verb)
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
