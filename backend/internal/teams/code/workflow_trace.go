package code

import (
	"fmt"
	"path"
	"regexp"
	"strings"
)

type businessMethod struct {
	file *File
	line int
}

func matchBusinessMethod(receiver string, candidates []businessMethod) (businessMethod, bool) {
	name := strings.TrimLeft(receiver, "_")
	if len(name) > 1 && name[0] == 'I' && name[1] >= 'A' && name[1] <= 'Z' {
		name = name[1:]
	}
	var exact, partial []businessMethod
	for _, method := range candidates {
		base := strings.TrimSuffix(path.Base(method.file.Path), path.Ext(method.file.Path))
		if strings.EqualFold(base, name) {
			exact = append(exact, method)
		} else if strings.Contains(strings.ToLower(name), strings.ToLower(base)) ||
			strings.Contains(strings.ToLower(base), strings.ToLower(name)) {
			partial = append(partial, method)
		}
	}
	if len(exact) == 1 {
		return exact[0], true
	}
	if len(exact) == 0 && len(partial) == 1 {
		return partial[0], true
	}
	return businessMethod{}, false
}

var (
	vbMethodEnd = regexp.MustCompile("(?i)^\\s*end\\s+(sub|function)\\b")
	nextMethod  = regexp.MustCompile("(?i)^\\s*(?:public|private|protected|internal|friend)\\b")
	// Mask quoted literals and comments when looking for braces and control flow.
	csText          = regexp.MustCompile(`"(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*'|//.*$`)
	vbText          = regexp.MustCompile("\"(?:[^\"]|\"\")*\"|'.*$")
	guardLine       = regexp.MustCompile("(?i)^\\s*if\\s*(?:\\(|\\s)")
	exitBranch      = regexp.MustCompile("(?i)^\\s*(throw\\b|return\\b|exit\\s+(?:sub|function)\\b)")
	exceptionBranch = regexp.MustCompile("(?i)^\\s*(?:}\\s*)?catch\\b")
	rollbackCall    = regexp.MustCompile("(?i)\\.\\s*Rollback(?:Async)?\\s*\\(")
	savedData       = regexp.MustCompile("(?i)\\.\\s*(SaveChanges(?:Async)?|Execute(?:SqlRaw|SqlCommand|NonQuery)(?:Async)?|ExecuteUpdate(?:Async)?|ExecuteDelete(?:Async)?)\\s*\\(")
)

func workflowCodeLine(language, line string) string {
	if language == "VB.NET" {
		return vbText.ReplaceAllString(line, "")
	}
	return csText.ReplaceAllString(line, "")
}

// Returns a bounded, 1-based inclusive range. An unknown method body supplies no observations.
func businessBody(m businessMethod) (start, end int) {
	depth := 0
	limit := min(len(m.file.Lines), m.line+160)
	for n := m.line; n <= limit; n++ {
		raw := m.file.Line(n)
		if m.file.Lang == "C#" {
			// A multi-line comment or literal can hide a structural delimiter. Stop conservatively.
			if strings.Contains(raw, "/*") || strings.Contains(raw, "\"\"\"") || strings.Contains(raw, "@\"") {
				return 0, 0
			}
		}
		line := workflowCodeLine(m.file.Lang, raw)
		if n > m.line && nextMethod.MatchString(line) && start == 0 {
			return 0, 0
		}
		if m.file.Lang == "VB.NET" {
			if n > m.line && vbMethodEnd.MatchString(line) {
				return m.line + 1, n - 1
			}
			if n > m.line && nextMethod.MatchString(line) {
				return 0, 0
			}
			continue
		}
		if start == 0 && strings.Contains(line, "=>") {
			return n, n
		}
		if start == 0 && strings.Contains(line, "{") {
			start = n
		}
		if start == 0 && strings.Contains(line, ";") {
			return 0, 0
		}
		depth += strings.Count(line, "{") - strings.Count(line, "}")
		if start != 0 && depth == 0 {
			return start, n
		}
	}
	return 0, 0
}

func observedCondition(c Citation) string {
	if c.Quote == "" {
		return ""
	}
	return " Observed condition: " + strings.ReplaceAll(c.Quote, "\x60", "'")
}

func sourceLocation(c Citation) string { return fmt.Sprintf("%s:%d", c.File, c.Line) }

func workflowEvidence(r *Run) map[string]map[int][]Finding {
	out := map[string]map[int][]Finding{}
	for _, finding := range r.Store.Findings("table-access", "role-check", "outbound-call", "messaging") {
		for _, c := range finding.Citations {
			if out[c.File] == nil {
				out[c.File] = map[int][]Finding{}
			}
			out[c.File][c.Line] = append(out[c.File][c.Line], finding)
		}
	}
	return out
}

func recordWorkflowTrace(r *Run, module string, action businessAction, method businessMethod, methods map[string][]businessMethod, evidence map[string]map[int][]Finding, crossBatch func([]Citation) bool) {
	title := action.verb + " " + action.area
	citations := append([]Citation(nil), action.citations...)
	lines := []string{
		"Entry point calls a service or repository at " + sourceLocation(citations[0]) + ".",
		"Matching implementation at " + sourceLocation(citations[1]) + ".",
	}
	flow := operatingFlow{Steps: []operatingStep{
		{Kind: "Trigger", Text: displayEntry(citations[0].File) + " starts this operation.", Refs: []int{0}},
		{Kind: "Application action", Text: "The application calls " + displayIdentifier(action.method) + ".", Refs: []int{0, 1}},
	}}
	observe := func(kind, text string, refs ...Citation) {
		indexes := make([]int, 0, len(refs))
		for _, c := range refs {
			indexes = append(indexes, len(citations))
			citations = append(citations, c)
		}
		flow.Steps = append(flow.Steps, operatingStep{Kind: kind, Text: text, Refs: indexes})
	}
	writes, namedWrites, role, exception := false, false, false, false
	observed := map[string]bool{}
	var rules []Finding
	bodyRead := false
	inspectedGuard := false
	var inspect func(businessMethod, bool)
	inspect = func(target businessMethod, followCalls bool) {
		start, end := businessBody(target)
		if start == 0 {
			return
		}
		bodyRead = true
		r.Cover("dom", target.file)
		for n := start; n <= end && len(citations) < 24; n++ {
			line := workflowCodeLine(target.file.Lang, target.file.Line(n))
			if guardLine.MatchString(line) {
				for exit := n; exit <= min(end, n+3); exit++ {
					statement := workflowCodeLine(target.file.Lang, target.file.Line(exit))
					if exit == n {
						// Recognize single-line VB and simple C# single-line guards.
						if _, after, ok := strings.Cut(strings.ToLower(statement), " then "); ok {
							statement = after
						} else if i := strings.LastIndex(statement, ")"); i >= 0 {
							statement = statement[i+1:]
						} else {
							continue
						}
					}
					statement = strings.TrimLeft(strings.TrimSpace(statement), "{ \t")
					if statement == "" {
						continue
					}
					outcome := exitBranch.FindStringSubmatch(statement)
					if len(outcome) > 0 {
						condition, branch := cite(target.file, n), cite(target.file, exit)
						ev := []Citation{condition}
						if exit != n {
							ev = append(ev, branch)
						}
						text := "A condition can end this part of the operation early."
						if conditionText := describedGuard(condition); conditionText != "" {
							text = "When " + conditionText + ", this part of the operation exits before later steps."
						}
						if strings.EqualFold(outcome[1], "throw") {
							text = "This branch raises an error. " + text
						}
						rules = append(rules, Finding{Agent: "dom", Kind: "business-rule", Module: module,
							Title: "Conditional exit in " + title, Detail: text + " Confirm the business meaning of this branch." + observedCondition(condition),
							Basis: BasisInferred, Confidence: "medium", Citations: ev})
						lines = append(lines, "Conditional exit at "+sourceLocation(condition)+": "+text)
						observe("Decision", text, ev...)
						inspectedGuard = true
					}
					break
				}
			}
			if savedData.MatchString(line) {
				c := cite(target.file, n)
				lines = append(lines, "Persistence or database execution call at "+sourceLocation(c)+".")
				observe("Data and outcome", "The code calls a database execution or save operation. This is an attempted action, and the production outcome still needs confirmation.", c)
				writes = true
			}
			if rollbackCall.MatchString(line) {
				observe("Exception", "A path calls for a transaction rollback.", cite(target.file, n))
				exception = true
			} else if exceptionBranch.MatchString(line) {
				observe("Exception", "A catch branch handles errors here. Confirm what the user sees and whether the operation can be retried.", cite(target.file, n))
				exception = true
			}
			for _, finding := range evidence[target.file.Path][n] {
				if observed[finding.ID] || len(citations) >= 24 {
					continue
				}
				c := cite(target.file, n)
				switch finding.Kind {
				case "table-access":
					table, ok := finding.Data["table"].(string)
					if !ok || table == "" {
						continue
					}
					if finding.Data["op"] == "write" {
						writes, namedWrites = true, true
						lines = append(lines, "Table write evidence for "+table+" at "+sourceLocation(c)+".")
						observe("Data change", "This path contains a write statement targeting "+table+".", c)
					} else if finding.Data["op"] == "read" {
						observe("Input", "This path contains a read from "+table+".", c)
					} else {
						continue
					}
				case "role-check":
					lines = append(lines, "Access check at "+sourceLocation(c)+".")
					text := "The code checks access at this point."
					if names := evidenceNames(finding.Data["roles"]); len(names) > 0 {
						role = true
						text += " Roles referenced by the check: " + strings.Join(names, ", ") + "."
					}
					observe("Role or access", text, c)
				default:
					lines = append(lines, "Integration reference at "+sourceLocation(c)+". Confirm the downstream effect.")
					observe("Handoff", "This path contains an integration reference. Confirm what the receiving system does with it.", c)
				}
				observed[finding.ID] = true
			}
			if !followCalls {
				continue
			}
			for _, call := range businessCall.FindAllStringSubmatch(line, 2) {
				next, ok := matchBusinessMethod(call[1], methods[module+"\x00"+strings.ToLower(call[2])])
				if !ok || next == target {
					continue
				}
				at, destination := cite(target.file, n), cite(next.file, next.line)
				lines = append(lines, "Further service or repository call at "+sourceLocation(at)+" leads to "+sourceLocation(destination)+".")
				observe("Handoff", "The code hands this operation to "+displayIdentifier(call[2])+".", at, destination)
				inspect(next, false)
				break
			}
		}
	}
	inspect(method, true)
	if !bodyRead {
		lines = append(lines, "The method body could not be traced safely.")
	} else if !inspectedGuard {
		lines = append(lines, "No conditional early exit was established in this trace.")
	}
	if !writes {
		lines = append(lines, "No persistence call was established in this trace.")
	}
	lines = append(lines, "Confirm authorization, exact records changed, downstream effects and failure handling where they are still unclear.")
	if !role {
		flow.Questions = append(flow.Questions, "Who performs or approves this operation in practice?")
	}
	if !inspectedGuard {
		flow.Questions = append(flow.Questions, "What business conditions allow or block this operation?")
	}
	if !namedWrites {
		flow.Questions = append(flow.Questions, "Which records should this operation read or change?")
	}
	if !exception {
		flow.Questions = append(flow.Questions, "What happens on duplicate input, rejection or a failed save?")
	}
	flow.Questions = append(flow.Questions, "What tells the operator or downstream system that the operation completed?")
	status := ""
	if crossBatch != nil {
		if !crossBatch(citations) {
			return
		}
		// These raw citations refer to the full source and are checked here, after reconciling.
		status = StatusAccepted
		for _, c := range citations {
			if r.CheckCitation(c) != "" {
				return
			}
		}
		title += " (across batches)"
	}
	if crossBatch == nil {
		for _, rule := range rules {
			r.Store.Add(rule)
		}
	}
	r.Store.Add(Finding{Agent: "dom", Kind: "workflow-trace", Module: module, Title: "Workflow trace: " + title,
		Detail: strings.Join(lines, "\n"), Basis: BasisInferred, Confidence: "medium", Citations: citations, Status: status, Data: map[string]any{"operatingFlow": flow}})
}
