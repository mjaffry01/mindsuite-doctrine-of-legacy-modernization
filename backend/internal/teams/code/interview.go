package code

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var focusCamel = regexp.MustCompile("([a-z0-9])([A-Z])")
var focusWords = regexp.MustCompile("[a-z][a-z0-9]{2,}")
var focusStop = map[string]bool{
	"the": true, "and": true, "for": true, "with": true, "this": true, "that": true,
	"want": true, "need": true, "system": true, "application": true, "modernization": true, "improve": true,
	"better": true, "today": true, "from": true, "more": true, "less": true, "than": true, "would": true,
}
var reasonSignals = map[string]string{
	"support": "framework runtime dependency version", "contract": "hosting licence dependency", "cost": "batch cache database compute",
	"security": "authorization authentication role identity validation", "regulation": "audit retention approval",
	"privacy": "customer identity retention", "risk": "error retry recovery exception", "growth": "batch concurrency database capacity",
	"performance": "batch query cache latency", "speed": "dependency integration test", "recovery": "rollback restore retry transaction",
	"data": "data reporting reconciliation table", "integration": "client interface queue api", "manual-work": "approval manual reconciliation",
	"testing": "test validation", "skills": "legacy framework", "hosting": "deployment runtime environment", "dependencies": "dependency package",
	"compatibility": "client browser runtime 32bit", "accessibility": "form screen", "observability": "logging telemetry", "licensing": "licence dependency",
	"consolidation": "shared migration", "energy": "batch cache", "complexity": "dependency coupling branch", "architecture": "module coupling interface",
	"portability": "platform runtime deployment", "experience": "form screen endpoint", "capabilities": "workflow business",
	"merger": "integration data", "retirement": "dependency usage", "strategy": "platform framework",
}

func wordsForFocus(text string) map[string]bool {
	terms := map[string]bool{}
	for _, term := range focusWords.FindAllString(strings.ToLower(focusCamel.ReplaceAllString(text, "$1 $2")), -1) {
		switch term {
		case "receiving", "receipt", "receipts":
			term = "receive"
		case "adjustment", "adjustments", "adjusting":
			term = "adjust"
		case "counts", "counting":
			term = "count"
		case "reconciliation", "reconciling":
			term = "reconcile"
		}
		if !focusStop[term] {
			terms[term] = true
		}
	}
	return terms
}

type interviewPriority struct {
	clientTerms map[string]bool
	reasonTerms map[string]bool
	exclusions  []string
}

func reviewPriority(r *Run) interviewPriority {
	focus := r.Options.Interview
	if focus == nil {
		return interviewPriority{}
	}
	clientTerms := wordsForFocus(strings.Join([]string{focus.Problem, focus.Success, focus.System, focus.Objective}, " "))
	var signals []string
	for _, reason := range focus.Reasons {
		signals = append(signals, reasonSignals[reason])
	}
	var exclusions []string
	for _, phrase := range strings.FieldsFunc(focus.OutOfScope, func(r rune) bool { return r == ',' || r == ';' || r == '\n' }) {
		value := strings.Join(focusWords.FindAllString(strings.ToLower(phrase), -1), " ")
		if value != "" {
			exclusions = append(exclusions, value)
		}
	}
	return interviewPriority{clientTerms, wordsForFocus(strings.Join(signals, " ")), exclusions}
}

func (p interviewPriority) score(f Finding) int {
	if len(p.clientTerms) == 0 && len(p.reasonTerms) == 0 && len(p.exclusions) == 0 {
		return 0
	}
	text := f.Module + " " + f.Title + " " + f.Detail
	specific := f.Module + " " + f.Title
	for _, citation := range f.Citations {
		text += " " + citation.File + " " + citation.Quote
		specific += " " + citation.File
	}
	keywords := " " + strings.Join(focusWords.FindAllString(strings.ToLower(focusCamel.ReplaceAllString(specific, "$1 $2")), -1), " ") + " "
	for _, exclusion := range p.exclusions {
		if strings.Contains(keywords, " "+exclusion+" ") {
			return -1
		}
	}
	score := 0
	for term := range wordsForFocus(text) {
		if p.clientTerms[term] {
			score += 3
		} else if p.reasonTerms[term] {
			score++
		}
	}
	return score
}

func (p interviewPriority) sort(findings []Finding) {
	scores := make(map[string]int, len(findings))
	for _, finding := range findings {
		scores[finding.ID] = p.score(finding)
	}
	sort.SliceStable(findings, func(i, j int) bool { return scores[findings[i].ID] > scores[findings[j].ID] })
}

func writeInterviewFocus(b *strings.Builder, r *Run, p interviewPriority, findings []Finding) {
	focus := r.Options.Interview
	if focus == nil {
		return
	}
	fmt.Fprint(b, "## Interview priorities\n\n")
	fmt.Fprint(b, "The project interview guides which findings to review first. Findings still need source evidence. Items that match recorded exclusions remain in the full report for dependency review.\n\n")
	if focus.Deadline != "" {
		fmt.Fprintf(b, "Interview deadline: %s. Confirm that the required scope can be validated by then.\n\n", focus.Deadline)
	}
	p.sort(findings)
	count := 0
	for _, finding := range findings {
		if p.score(finding) <= 0 || count == 8 {
			break
		}
		fmt.Fprintf(b, "- %s [%s]\n", finding.Title, finding.ID)
		count++
	}
	if count == 0 {
		fmt.Fprint(b, "No reviewed workflow or rule matched the interview terms. Confirm which source files implement the workflows described in the interview.\n")
	}
	fmt.Fprintln(b)
}
