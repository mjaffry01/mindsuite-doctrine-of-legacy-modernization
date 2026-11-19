package code

import (
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"
)

type operatingStep struct {
	Kind string `json:"kind"`
	Text string `json:"text"`
	Refs []int  `json:"refs"`
}

type operatingFlow struct {
	Steps     []operatingStep `json:"steps"`
	Questions []string        `json:"questions,omitempty"`
}

func flowData(f Finding) (operatingFlow, bool) {
	var flow operatingFlow
	switch value := f.Data["operatingFlow"].(type) {
	case operatingFlow:
		flow = value
	case map[string]any:
		data, err := json.Marshal(value)
		if err != nil || json.Unmarshal(data, &flow) != nil {
			return flow, false
		}
	}
	return flow, len(flow.Steps) > 0
}

func displayIdentifier(name string) string {
	words := businessWords.FindAllString(name, -1)
	if len(words) == 0 {
		return name
	}
	return strings.Join(words, " ")
}

func displayEntry(file string) string {
	name := path.Base(file)
	name = strings.TrimSuffix(name, path.Ext(name))
	return displayIdentifier(strings.TrimSuffix(strings.TrimSuffix(name, ".aspx"), ".razor"))
}

var simpleLimit = regexp.MustCompile("(?i)^(?:\\s*if\\s*\\(?\\s*)([a-z_][a-z0-9_.]*)\\s*(<=|>=|<|>|==|=|<>|!=)\\s*(-?[0-9]+(?:\\.[0-9]+)?)(?:\\s*\\)?\\s*(?:then\\b|\\{|$))")
var structuredRoleName = regexp.MustCompile("^[A-Za-z][A-Za-z0-9 _-]{0,49}$")

func describedGuard(c Citation) string {
	match := simpleLimit.FindStringSubmatch(c.Quote)
	if match == nil {
		return ""
	}
	field := match[1]
	if dot := strings.LastIndexByte(field, '.'); dot >= 0 {
		field = field[dot+1:]
	}
	compare := map[string]string{"<": "is less than", "<=": "is at or below", ">": "is greater than", ">=": "is at or above", "=": "equals", "==": "equals", "!=": "does not equal", "<>": "does not equal"}[match[2]]
	return strings.ToLower(displayIdentifier(field)) + " " + compare + " " + match[3]
}

func evidenceNames(value any) []string {
	var candidates []string
	switch values := value.(type) {
	case []string:
		candidates = values
	case []any:
		for _, v := range values {
			if name, ok := v.(string); ok {
				candidates = append(candidates, name)
			}
		}
	}
	var names []string
	for _, name := range candidates {
		if structuredRoleName.MatchString(name) {
			names = append(names, name)
		}
	}
	return names
}

func operatingRefs(r *Run, finding Finding, indexes []int) ([]Citation, bool) {
	var cites []Citation
	for _, index := range indexes {
		if index < 0 {
			return nil, false
		}
		if finding.Kind == "operating-flow" {
			if index >= len(finding.Evidence) {
				return nil, false
			}
			evidence, ok := r.Store.Get(finding.Evidence[index])
			if !ok || evidence.Status != StatusAccepted {
				return nil, false
			}
			if len(evidence.Citations) > 0 {
				cites = append(cites, evidence.Citations[0])
			}
		} else {
			if index >= len(finding.Citations) {
				return nil, false
			}
			cites = append(cites, finding.Citations[index])
		}
	}
	return cites, len(cites) > 0
}

func lineReferences(citations []Citation) string {
	var refs []string
	for _, c := range citations {
		if c.File != "" {
			refs = appendUnique(refs, sourceLocation(c))
		}
	}
	return strings.Join(refs, ", ")
}

func flowHasSource(r *Run, finding Finding, flow operatingFlow) bool {
	for _, step := range flow.Steps {
		if _, ok := operatingRefs(r, finding, step.Refs); !ok {
			return false
		}
	}
	return true
}

func writeOperatingModel(b *strings.Builder, r *Run, priority interviewPriority) {
	fmt.Fprint(b, "## How the business operates\n\n")
	fmt.Fprint(b, "The report follows the paths seen in the source. Decision and exception steps may be alternate paths. Confirm staff roles, manual work and production outcomes with the process owner.\n\n")
	purposes := r.Store.Findings("purpose", "capability")
	priority.sort(purposes)
	written := 0
	for _, f := range purposes {
		if f.Status != StatusAccepted || f.Detail == "" {
			continue
		}
		if written == 0 {
			fmt.Fprint(b, "### What the system supports\n\n")
		}
		fmt.Fprintf(b, "- %s [%s]", f.Detail, f.ID)
		if refs := lineReferences(f.Citations); refs != "" {
			fmt.Fprintf(b, " — %s", refs)
		}
		fmt.Fprintln(b)
		written++
		if written == 6 {
			break
		}
	}
	if written > 0 {
		fmt.Fprintln(b)
	}

	findings := r.Store.Findings("workflow-trace", "operating-flow")
	priority.sort(findings)
	// Prefer a connected trace where a batch-local trace starts at the same action.
	crossBatch := map[string]bool{}
	for _, f := range findings {
		flow, ok := flowData(f)
		if f.Kind == "workflow-trace" && ok && f.Status == StatusAccepted && strings.HasSuffix(f.Title, " (across batches)") && flowHasSource(r, f, flow) {
			crossBatch[strings.TrimSuffix(f.Title, " (across batches)")+"\x00"+sourceLocation(f.Citations[0])] = true
		}
	}
	written = 0
	for _, finding := range findings {
		flow, ok := flowData(finding)
		if !ok || finding.Status != StatusAccepted || !flowHasSource(r, finding, flow) {
			continue
		}
		if finding.Kind == "workflow-trace" && !strings.HasSuffix(finding.Title, " (across batches)") && crossBatch[finding.Title+"\x00"+sourceLocation(finding.Citations[0])] {
			continue
		}
		name := strings.TrimPrefix(finding.Title, "Workflow trace: ")
		fmt.Fprintf(b, "### %s\n\n", name)
		origin := "Source trace"
		if finding.Kind == "operating-flow" {
			origin = "AI interpretation of cited findings · confirm with the process owner"
		}
		fmt.Fprintf(b, "%s [%s]\n\n", origin, finding.ID)
		for i, step := range flow.Steps {
			citations, _ := operatingRefs(r, finding, step.Refs)
			fmt.Fprintf(b, "%d. %s — %s [%s]\n", i+1, step.Kind, step.Text, lineReferences(citations))
		}
		if len(flow.Questions) > 0 {
			fmt.Fprint(b, "\nWhat to confirm\n")
			for _, question := range flow.Questions {
				fmt.Fprintf(b, "- %s\n", question)
			}
		}
		fmt.Fprintln(b)
		written++
		if written == 12 {
			break
		}
	}
	written += writeScheduledOperations(b, r, 12-written)
	if written == 0 {
		fmt.Fprint(b, "The source did not establish a connected operating flow. Ask the process owner for a representative transaction, then match its entry point, decision rules and outcome to the code.\n\n")
	}
}

func writeScheduledOperations(b *strings.Builder, r *Run, limit int) int {
	if limit <= 0 {
		return 0
	}
	byFile := map[string][]Finding{}
	for _, f := range r.Store.Findings("batch-job", "job-step") {
		if f.Status == StatusAccepted && len(f.Citations) > 0 {
			file := f.Citations[0].File
			byFile[file] = append(byFile[file], f)
		}
	}
	count := 0
	for _, file := range sortedKeys(byFile) {
		items := byFile[file]
		sort.SliceStable(items, func(i, j int) bool { return items[i].Citations[0].Line < items[j].Citations[0].Line })
		for i, job := range items {
			if job.Kind != "batch-job" || i+1 >= len(items) || items[i+1].Kind != "job-step" {
				continue
			}
			fmt.Fprintf(b, "### %s\n\nJob steps as declared. Confirm the schedule and any conditional execution with the operations owner. [%s]\n\n", job.Title, job.ID)
			fmt.Fprintf(b, "1. Trigger — The job starts at this declaration. [%s]\n", sourceLocation(job.Citations[0]))
			for j := i + 1; j < len(items) && items[j].Kind == "job-step"; j++ {
				step := items[j]
				fmt.Fprintf(b, "%d. Batch action — %s [%s]\n", j-i+1, step.Title, sourceLocation(step.Citations[0]))
			}
			fmt.Fprint(b, "\nWhat to confirm\n- What starts this job, what records change, and how is a failed step recovered?\n\n")
			count++
			if count == limit {
				return count
			}
		}
	}
	return count
}
