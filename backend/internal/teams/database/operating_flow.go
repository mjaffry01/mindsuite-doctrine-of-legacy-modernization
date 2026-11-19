package database

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

type databaseOperation struct {
	Name, Type, File string
	Line, End        int
}

type databaseStep struct {
	Kind   string `json:"kind"`
	Text   string `json:"text"`
	Ref    int    `json:"ref"`
	Target string `json:"target,omitempty"`
}

var (
	memberDecl        = regexp.MustCompile(`(?i)^\s*(procedure|function)\s+([a-z_][a-z0-9_$#]*)\b`)
	namedEnd          = regexp.MustCompile(`(?i)^\s*end\s+([a-z_][a-z0-9_$#]*)\s*;`)
	flowStop          = regexp.MustCompile(`(?i)^\s*(?:go\b|/\s*$|create\b|alter\b)`)
	flowEnd           = regexp.MustCompile(`(?i)^\s*end\s*(?:;|\$\$|/|$)`)
	routineBody       = regexp.MustCompile(`(?i)\bbegin\b`)
	flowBranch        = regexp.MustCompile(`(?i)^\s*(?:if|elsif|elseif|when)\b`)
	flowCondition     = regexp.MustCompile(`(?i)^\s*(?:if|elsif|elseif|when)\s+(.+?)\s+then\b`)
	flowTSQLCondition = regexp.MustCompile(`(?i)^\s*if\s+(.+?)(?:\s+begin)?\s*$`)
	flowAnd           = regexp.MustCompile(`(?i)\s+and\s+`)
	flowCompare       = regexp.MustCompile(`(?i)^\(?\s*([:@]?[a-z_][a-z0-9_.$#]*)\s*(<=|>=|<>|!=|=|<|>)\s*(-?[0-9]+(?:\.[0-9]+)?|'[a-z]{1,12}')\s*\)?$`)
	flowCalculation   = regexp.MustCompile(`(?i)^\s*([a-z_][a-z0-9_$#]*)\s*:=\s*([a-z_][a-z0-9_$#]*)\s*([*/+-])\s*([0-9]+(?:\.[0-9]+)?)\s*;`)
	flowStatusFilter  = regexp.MustCompile(`(?i)\bwhere\s+(?:[a-z_][a-z0-9_]*\.)?(status|state)\s*=\s*'(new|tested|pending|approved|rejected|paid|closed|cancelled|completed|archived)'`)

	flowWrite       = regexp.MustCompile(`(?i)^\s*(insert\s+(?:all\s+)?into|update|delete\s+from|merge\s+into|truncate\s+table)\s+([\w$#."\x60\[\]@]+)`)
	flowQuery       = regexp.MustCompile(`(?i)^\s*(?:select|from|join)\b`)
	flowTransaction = regexp.MustCompile(`(?i)^\s*(commit|rollback)\b`)
	flowRaise       = regexp.MustCompile(`(?i)^\s*(?:raise(?:_application_error)?|throw|raiserror|signal)\b`)
	flowException   = regexp.MustCompile(`(?i)^\s*(?:exception|begin\s+catch|when\s+others)\b`)
	flowReturn      = regexp.MustCompile(`(?i)^\s*return\b`)
	flowCall        = regexp.MustCompile(`(?i)^\s*(?:(call|exec(?:ute)?)\s+)?([a-z_][a-z0-9_$#.]*)(?:\s*\(|\s*;)`)
	flowTarget      = regexp.MustCompile(`(?i)^\s*(?:begin\s+|call\s+|exec(?:ute)?\s+)?([a-z_][a-z0-9_$#.]*)(?:\s*\([^;'\n]*\))?\s*;?\s*(?:end\s*;?)?\s*$`)
	flowStatusWrite = regexp.MustCompile(`(?i)^\s*update\s+[\w$.#"\x60\[\]@]+\s+set\s+(status|state)\s*=\s*'(new|tested|pending|approved|rejected|paid|closed|cancelled|completed|archived)'`)
	interviewWords  = regexp.MustCompile(`[a-z]{4,}`)
)

// Mask comments and string literals while retaining SQL identifiers.
func flowSQLLine(raw string, comment, quoted *bool) string {
	var b strings.Builder
	for i := 0; i < len(raw); i++ {
		if *comment {
			if strings.HasPrefix(raw[i:], "*/") {
				*comment = false
				i++
			}
			continue
		}
		if *quoted {
			if raw[i] == '\'' {
				if i+1 < len(raw) && raw[i+1] == '\'' {
					i++
				} else {
					*quoted = false
				}
			}
			continue
		}
		switch {
		case strings.HasPrefix(raw[i:], "--"):
			return b.String()
		case strings.HasPrefix(raw[i:], "/*"):
			*comment = true
			i++
			b.WriteByte(' ')
		case raw[i] == '\'':
			*quoted = true
			b.WriteByte(' ')
		default:
			b.WriteByte(raw[i])
		}
	}
	return b.String()
}

func routineOperations(r *Run, m *DBModel) []databaseOperation {
	var ops []databaseOperation
	for _, p := range m.sortedCode() {
		f := r.Code.File(p.File)
		if f == nil || p.Line < 1 || p.Line >= p.End || p.Type == "PACKAGE" {
			continue
		}
		end := min(p.End, len(f.Lines))
		comment, quoted := false, false
		if p.Type == "PACKAGE BODY" {
			var current *databaseOperation
			for n := p.Line + 1; n <= end; n++ {
				text := flowSQLLine(f.Line(n), &comment, &quoted)
				if match := memberDecl.FindStringSubmatch(text); match != nil {
					current = &databaseOperation{Name: p.Name + "." + strings.ToUpper(match[2]), Type: strings.ToUpper(match[1]), File: f.Path, Line: n}
				}
				if match := namedEnd.FindStringSubmatch(text); match != nil && current != nil && strings.HasSuffix(current.Name, "."+strings.ToUpper(match[1])) {
					current.End = n
					ops = append(ops, *current)
					current = nil
				}
			}
			continue
		}
		hasBody, closed := false, false
		_, base := splitName(p.Name)
		for n := p.Line; n <= end; n++ {
			text := flowSQLLine(f.Line(n), &comment, &quoted)
			if n > p.Line && flowStop.MatchString(text) {
				end = n - 1
				closed = true
				break
			}
			hasBody = hasBody || routineBody.MatchString(text)
			if match := namedEnd.FindStringSubmatch(text); match != nil && strings.EqualFold(match[1], base) {
				end = n
				closed = true
				break
			}
			if hasBody && m.Engine != "sqlserver" && flowEnd.MatchString(text) {
				end = n
				closed = true
				break
			}
		}
		if !closed {
			// The metadata parser ends a routine before its GO or slash separator.
			for n := end + 1; n <= min(end+4, len(f.Lines)); n++ {
				next := strings.TrimSpace(f.Line(n))
				if next == "" {
					continue
				}
				closed = flowStop.MatchString(next)
				break
			}
		}
		if hasBody && closed && end > p.Line {
			ops = append(ops, databaseOperation{Name: p.Name, Type: p.Type, File: f.Path, Line: p.Line, End: end})
		}
	}
	return ops
}

func cleanTarget(text string) string {
	if match := flowTarget.FindStringSubmatch(text); match != nil {
		return strings.ToUpper(match[1])
	}
	return ""
}

func resolveOperation(names map[string]bool, owner, target string) string {
	if target == "" {
		return ""
	}
	if names[target] {
		return target
	}
	if i := strings.LastIndexByte(owner, '.'); i >= 0 && names[owner[:i+1]+target] {
		return owner[:i+1] + target
	}
	if parts := strings.Split(owner, "."); len(parts) == 3 && names[parts[0]+"."+target] {
		return parts[0] + "." + target
	}
	// Unqualified names without a same-owner definition require confirmation.
	return ""
}

func displayDBField(field string) string {
	field = strings.ToLower(strings.TrimLeft(field, ":@"))
	if i := strings.LastIndexByte(field, '.'); i >= 0 {
		field = field[i+1:]
	}
	return strings.ReplaceAll(strings.TrimPrefix(strings.TrimPrefix(field, "v_"), "p_"), "_", " ")
}

func describeCondition(raw string) string {
	unknown := "A condition controls which path runs here. Confirm the exact business rule."
	match := flowCondition.FindStringSubmatch(raw)
	if match == nil {
		match = flowTSQLCondition.FindStringSubmatch(raw)
	}
	if match == nil {
		return unknown
	}
	var checks []string
	for _, part := range flowAnd.Split(match[1], -1) {
		d := flowCompare.FindStringSubmatch(strings.TrimSpace(part))
		if d == nil {
			return unknown
		}
		field := displayDBField(d[1])
		value := d[3]
		if strings.HasPrefix(value, "'") {
			value = strings.ToUpper(strings.Trim(value, "'"))
			isGrade := strings.HasSuffix(field, "grade") && strings.Contains("|A|B|C|D|", "|"+value+"|")
			isState := (field == "status" || field == "state") && strings.Contains("|NEW|TESTED|PENDING|APPROVED|REJECTED|PAID|CLOSED|CANCELLED|COMPLETED|ARCHIVED|", "|"+value+"|")
			if !isGrade && !isState {
				return unknown
			}
		}
		comparator := map[string]string{"<": "is below", "<=": "is at or below", ">": "is above", ">=": "is at or above", "=": "equals", "<>": "does not equal", "!=": "does not equal"}[d[2]]
		checks = append(checks, field+" "+comparator+" "+value)
	}
	return "This branch checks whether " + strings.Join(checks, " and ") + "."
}

func runDatabaseOperations(r *Run) {
	ops := routineOperations(r, st(r).db)
	names := map[string]bool{}
	for _, op := range ops {
		names[op.Name] = true
	}
	for _, op := range ops {
		f := r.Code.File(op.File)
		var steps []databaseStep
		var citations []Citation
		add := func(kind, text, target string, line int) {
			if len(steps) >= 24 {
				return
			}
			steps = append(steps, databaseStep{kind, text, len(citations), target})
			citations = append(citations, Citation{File: f.Path, Line: line})
		}
		if op.Type == "TRIGGER" {
			text, line := "The export declares this trigger. Confirm the event and enabled state.", op.Line
			for n := op.Line; n <= min(op.End, op.Line+10); n++ {
				if match := reTrigOn.FindStringSubmatch(f.Line(n)); match != nil {
					text, line = "Declared trigger event: "+strings.Join(strings.Fields(match[0]), " ")+".", n
					break
				}
				if match := reTrigOn2.FindStringSubmatch(f.Line(n)); match != nil {
					text, line = "Declared trigger on "+strings.ToUpper(nameCleaner.Replace(match[1]))+". Confirm its event.", n
					break
				}
			}
			add("Trigger", text, "", line)
		} else {
			add("Entry point", "Another part of the system can call "+op.Name+".", "", op.Line)
		}

		comment, quoted := false, false
		for n := op.Line + 1; n <= op.End && len(steps) < 24; n++ {
			raw := f.Line(n)
			line := strings.TrimSpace(flowSQLLine(raw, &comment, &quoted))
			switch {
			case flowException.MatchString(line):
				add("Exception", "This routine defines an error-handling branch. Confirm the result the caller receives.", "", n)
			case reDynamic.MatchString(line):
				add("Dynamic SQL", "The database builds or runs SQL at runtime. Confirm the records it affects.", "", n)
			case flowBranch.MatchString(line):
				add("Decision", describeCondition(raw), "", n)
			case flowRaise.MatchString(line):
				add("Exception", "This path can signal an error back to the caller.", "", n)
			case flowReturn.MatchString(line):
				add("Return", "This path returns to the caller. Confirm what the caller treats as success.", "", n)
			case flowCalculation.MatchString(line):
				items := flowCalculation.FindStringSubmatch(line)
				if strings.EqualFold(items[1], items[2]) {
					word := map[string]string{"*": "multiplies", "/": "divides", "+": "increases", "-": "decreases"}[items[3]]
					add("Calculation", "This path "+word+" "+displayDBField(items[1])+" by "+items[4]+".", "", n)
				}
			case flowTransaction.MatchString(line):
				match := flowTransaction.FindStringSubmatch(line)
				add("Transaction", "This path issues "+strings.ToUpper(match[1])+". Confirm which steps share the transaction.", "", n)
			}
			if match := flowWrite.FindStringSubmatch(line); match != nil {
				table := strings.ToUpper(nameCleaner.Replace(match[2]))
				action := strings.ToUpper(strings.Fields(match[1])[0])
				text := "This path contains a " + action + " statement targeting " + table + "."
				if status := flowStatusWrite.FindStringSubmatch(raw); action == "UPDATE" && status != nil {
					text += " It sets " + strings.ToLower(status[1]) + " to " + strings.ToUpper(status[2]) + "."
				}
				add("Data change", text, "", n)
			}
			if flowQuery.MatchString(line) {
				for _, match := range reRead.FindAllStringSubmatch(line, 4) {
					text := "This query reads from " + strings.ToUpper(nameCleaner.Replace(match[1])) + "."
					if status := flowStatusFilter.FindStringSubmatch(raw); status != nil {
						text += " It selects records where " + strings.ToLower(status[1]) + " is " + strings.ToUpper(status[2]) + "."
					}
					add("Input", text, "", n)
				}
			}
			if call := flowCall.FindStringSubmatch(line); call != nil {
				target := strings.ToUpper(call[2])
				resolved := resolveOperation(names, op.Name, target)
				if resolved != "" && resolved != op.Name {
					add("Handoff", "The routine calls "+resolved+".", resolved, n)
				} else if resolved == "" && (call[1] != "" || strings.Contains(target, ".")) && target != "IMMEDIATE" && target != "FORMAT" {
					add("Handoff", "The routine calls "+target+". Confirm its implementation.", target, n)
				}
			}
			for _, outside := range outsideCalls {
				if outside.re.MatchString(line) {
					add("External handoff", outside.name+" is referenced here. Confirm the receiving system and its response.", "", n)
					break
				}
			}
		}
		if len(steps) < 2 {
			continue
		}
		var detail []string
		for _, step := range steps {
			detail = append(detail, step.Kind+" — "+step.Text)
		}
		r.Store.Add(Finding{Agent: "ddom", Kind: "db-operating-flow", Title: op.Name, Detail: strings.Join(detail, "\n"),
			Basis: BasisInferred, Confidence: "medium", Citations: citations, Data: map[string]any{"steps": steps}})
	}
}

func databaseSteps(f Finding) []databaseStep {
	if values, ok := f.Data["steps"].([]databaseStep); ok {
		return values
	}
	payload, err := json.Marshal(f.Data["steps"])
	if err != nil {
		return nil
	}
	var values []databaseStep
	if json.Unmarshal(payload, &values) != nil {
		return nil
	}
	return values
}

func databaseInterviewScore(r *Run, text string) int {
	focus := r.Options.Interview
	if focus == nil {
		return 0
	}
	haystack := strings.ToLower(text)
	for _, value := range strings.FieldsFunc(focus.OutOfScope, func(r rune) bool { return r == ',' || r == ';' || r == '\n' }) {
		if token := strings.TrimSpace(strings.ToLower(value)); token != "" && strings.Contains(haystack, token) {
			return -10
		}
	}
	words := map[string]bool{}
	for _, word := range interviewWords.FindAllString(strings.ToLower(focus.Problem+" "+focus.Success+" "+focus.Objective), -1) {
		switch word {
		case "that", "from", "with", "this", "currently", "should", "would", "system":
			continue
		}
		words[word] = true
	}
	score := 0
	for word := range words {
		if strings.Contains(haystack, word) {
			score++
		}
	}
	return score
}

func writeDatabaseOperations(b *strings.Builder, r *Run) {
	fmt.Fprint(b, "## How the business operates\n\n")
	fmt.Fprint(b, "These flows show what the exported database code does. Decisions and exceptions describe alternate paths. Confirm who starts the work, whether jobs and triggers are enabled, and the outcomes in production.\n\n")
	flows := r.Store.Findings("db-operating-flow")
	names := map[string]bool{}
	for _, f := range flows {
		if f.Status == StatusAccepted {
			names[f.Title] = true
		}
	}
	jobs := map[string][]Finding{}
	for _, job := range r.Store.Findings("db-job") {
		if job.Status != StatusAccepted || len(job.Citations) == 0 {
			continue
		}
		action, _ := job.Data["operationTarget"].(string)
		if name := resolveOperation(names, "", action); name != "" {
			jobs[name] = append(jobs[name], job)
		}
	}
	scores := map[string]int{}
	for _, flow := range flows {
		scores[flow.ID] = 10 * databaseInterviewScore(r, flow.Title+" "+flow.Detail)
		if len(jobs[flow.Title]) > 0 {
			scores[flow.ID]++
		}
	}
	sort.SliceStable(flows, func(i, j int) bool { return scores[flows[i].ID] > scores[flows[j].ID] })
	shown := 0
	for _, flow := range flows {
		if flow.Status != StatusAccepted {
			continue
		}
		steps := databaseSteps(flow)
		if len(steps) < 2 {
			continue
		}
		fmt.Fprintf(b, "### %s\n\nDatabase source trace [%s]\n\n", flow.Title, flow.ID)
		n := 1
		for _, job := range jobs[flow.Title] {
			schedule, _ := job.Data["schedule"].(string)
			text := job.Title + " is configured to call this routine."
			if schedule != "" {
				text += " Declared schedule: " + strings.Join(strings.Fields(schedule), " ") + "."
			}
			fmt.Fprintf(b, "%d. Trigger — %s [%s:%d]\n", n, text, job.Citations[0].File, job.Citations[0].Line)
			n++
		}
		hasException, trigger := false, false
		for _, step := range steps {
			if step.Ref < 0 || step.Ref >= len(flow.Citations) {
				continue
			}
			c := flow.Citations[step.Ref]
			text := step.Text
			if target := resolveOperation(names, flow.Title, step.Target); target != "" && target != flow.Title && strings.HasSuffix(text, "Confirm its implementation.") {
				text = "The routine calls " + target + ". See the linked routine for its database steps."
			}
			fmt.Fprintf(b, "%d. %s — %s [%s:%d]\n", n, step.Kind, text, c.File, c.Line)
			n++
			hasException = hasException || step.Kind == "Exception"
			trigger = trigger || step.Kind == "Trigger"
		}
		fmt.Fprint(b, "\nWhat to confirm\n")
		if len(jobs[flow.Title]) == 0 && !trigger {
			fmt.Fprintln(b, "- Which workflow, user action or job initiates this routine?")
		}
		fmt.Fprintln(b, "- What proves the operation completed, and who owns it?")
		if !hasException {
			fmt.Fprintln(b, "- What happens on rejection, duplicate requests, or a database failure?")
		}
		if len(steps) == 24 {
			fmt.Fprintln(b, "- Check the remaining routine for decisions beyond the report’s step limit.")
		}
		fmt.Fprintln(b)
		shown++
		if shown == 12 {
			break
		}
	}
	if shown == 0 {
		fmt.Fprint(b, "No connected stored-code workflow was established from this export. Include stored procedure bodies, trigger definitions, and scheduler job actions to see how the database operates.\n\n")
	}
	if len(names) > shown {
		fmt.Fprintf(b, "%d more traced routines are available in All findings.\n\n", len(names)-shown)
	}
}
