package incidents

import (
	"encoding/csv"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// IModel is what the incidents & changes team knows: incidents, changes and problem records of one system,
// read from ServiceNow list exports (incident, change_request, problem) and Jira CSV exports.
// Columns that name people (caller, assignee, reporter, …) are never kept: their values are only used,
// in memory, to remove those names from ticket titles. Long descriptions, comments and work notes are not read.

// Ticket is one incident, change or problem.
type Ticket struct {
	Kind     string // incident | change | problem
	ID       string
	Title    string // short description, cleaned of names, e-mail addresses, addresses and numbers
	CI       string // configuration item / component / service
	Category string
	Priority string // P1..P4 or the export's own value
	State    string
	Type     string // change type: standard | normal | emergency
	Close    string // close code / resolution
	CausedBy string // the change that caused an incident
	Problem  string // the problem record an incident is linked to
	Opened   time.Time
	Resolved time.Time
	Reopened int
	File     string
	Line     int
}

// IModel holds the parsed tickets.
type IModel struct {
	Incidents []*Ticket
	Changes   []*Ticket
	Problems  []*Ticket
	Files     []string
	Ignored   []string // people and free-text columns that were not read
	From, To  time.Time
}

// Summary is shown with the run.
func (m *IModel) Summary() map[string]any {
	s := map[string]any{"incidents": len(m.Incidents), "changes": len(m.Changes), "problems": len(m.Problems), "exportFiles": len(m.Files), "months": m.Months()}
	if !m.From.IsZero() {
		s["from"], s["to"] = m.From.Format("2006-01-02"), m.To.Format("2006-01-02")
	}
	return s
}

// Months is the length of the export window in months (at least 1 when there is data).
func (m *IModel) Months() int {
	if m.From.IsZero() {
		return 0
	}
	n := int(m.To.Sub(m.From).Hours()/24/30.44 + 0.5)
	return max(n, 1)
}

var (
	peopleCols = map[string]bool{"caller_id": true, "caller": true, "opened_by": true, "assigned_to": true, "assignee": true, "reporter": true, "requested_by": true,
		"resolved_by": true, "closed_by": true, "creator": true, "watchers": true, "watch_list": true, "sys_created_by": true, "sys_updated_by": true, "requester": true,
		"u_caller": true, "affected_user": true, "contact": true, "change_manager": true, "implementer": true}
	textCols = map[string]bool{"description": true, "comments": true, "work_notes": true, "close_notes": true, "comment": true, "comments_and_work_notes": true,
		"environment": true, "steps to reproduce": true, "resolution notes": true, "justification": true, "backout_plan": true, "implementation_plan": true, "test_plan": true}
	emailRe  = regexp.MustCompile(`[\w.+-]+@[\w-]+(\.[\w-]+)+`)
	ipRe     = regexp.MustCompile(`\b\d{1,3}(\.\d{1,3}){3}\b`)
	ticketRe = regexp.MustCompile(`(?i)\b(INC|CHG|PRB|RITM|REQ|TASK|CTASK)\d{4,}\b|\b[A-Z][A-Z0-9]+-\d+\b`)
	numRe    = regexp.MustCompile(`\b\d{3,}\b`)
)

func lower(h string) string {
	return strings.ToLower(strings.TrimSpace(strings.TrimPrefix(h, string(rune(0xFEFF)))))
}

func readTable(f *File) (head []string, rows [][]string, lines []int) {
	rd := csv.NewReader(strings.NewReader(strings.Join(f.Lines, "\n")))
	rd.LazyQuotes, rd.FieldsPerRecord, rd.TrimLeadingSpace = true, -1, true
	for {
		row, err := rd.Read()
		if err != nil {
			break
		}
		line, _ := rd.FieldPos(0)
		if head == nil {
			for _, h := range row {
				head = append(head, lower(h))
			}
			continue
		}
		rows, lines = append(rows, row), append(lines, line)
	}
	return
}

func colOf(head []string, names ...string) int {
	for _, n := range names {
		for i, h := range head {
			if h == n {
				return i
			}
		}
	}
	return -1
}

func cellOf(row []string, i int) string {
	if i < 0 || i >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[i])
}

var dateLayouts = []string{"2006-01-02 15:04:05", "2006-01-02 15:04", "2006-01-02T15:04:05Z07:00", "2006-01-02T15:04:05", "2006-01-02",
	"02/Jan/06 3:04 PM", "02/Jan/06 15:04", "2/Jan/06 3:04 PM", "01/02/2006 15:04:05", "01/02/2006 15:04", "1/2/2006 15:04", "01/02/2006", "02.01.2006 15:04:05", "02.01.2006"}

func parseTime(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}
	}
	for _, l := range dateLayouts {
		if t, err := time.Parse(l, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

// priority maps "1 - Critical", "Highest", "P2", "Blocker" … to P1..P4.
func priority(s string) string {
	l := strings.ToLower(strings.TrimSpace(s))
	switch {
	case l == "":
		return ""
	case strings.HasPrefix(l, "1") || strings.HasPrefix(l, "p1") || strings.Contains(l, "critical") || strings.Contains(l, "highest") || strings.Contains(l, "blocker"):
		return "P1"
	case strings.HasPrefix(l, "2") || strings.HasPrefix(l, "p2") || l == "high" || strings.Contains(l, "- high"):
		return "P2"
	case strings.HasPrefix(l, "3") || strings.HasPrefix(l, "p3") || strings.Contains(l, "moderate") || strings.Contains(l, "medium"):
		return "P3"
	default:
		return "P4"
	}
}

// ParseIncidents reads every CSV of the run.
func ParseIncidents(r *Run) *IModel {
	m := &IModel{}
	ignored := map[string]bool{}
	names := map[string]bool{}
	for _, f := range r.Code.Files {
		for i, l := range f.Lines {
			if looksSecret(l) {
				r.MarkSecret(f.Path, i+1)
			}
		}
		if f.Lang != "CSV" {
			continue
		}
		head, rows, lines := readTable(f)
		cNum := colOf(head, "number", "issue key", "key", "id")
		cTitle := colOf(head, "short_description", "summary", "title", "short description")
		if cNum < 0 || cTitle < 0 {
			continue
		}
		cIType := colOf(head, "issue type", "issuetype", "sys_class_name")
		cCI := colOf(head, "cmdb_ci", "configuration item", "business_service", "service", "component/s", "components", "component", "application")
		cCat := colOf(head, "category", "subcategory", "labels")
		cPri := colOf(head, "priority", "severity", "impact")
		cState := colOf(head, "state", "status", "incident_state")
		cType := colOf(head, "type", "change type", "chg_model")
		cClose := colOf(head, "close_code", "resolution", "close code", "u_close_code")
		cCaused := colOf(head, "caused_by", "caused by", "caused by change", "rfc")
		cProb := colOf(head, "problem_id", "problem", "linked problem")
		cOpen := colOf(head, "opened_at", "created", "sys_created_on", "start_date", "planned start")
		cRes := colOf(head, "resolved_at", "resolved", "closed_at", "end_date", "work_end", "resolutiondate")
		cReopen := colOf(head, "reopen_count", "reopened", "reopen count")
		var pcols []int
		for i, h := range head {
			if peopleCols[h] {
				pcols = append(pcols, i)
				ignored[h] = true
			} else if textCols[h] {
				ignored[h] = true
			}
		}
		base := strings.ToLower(path.Base(f.Path))
		used := false
		for k, row := range rows {
			id := cellOf(row, cNum)
			if id == "" {
				continue
			}
			for _, i := range pcols {
				for _, w := range strings.FieldsFunc(cellOf(row, i), func(r rune) bool { return r == ' ' || r == ',' || r == '.' || r == ';' }) {
					if len(w) >= 3 && !strings.Contains(w, "@") {
						names[strings.ToLower(w)] = true
					}
				}
			}
			t := &Ticket{ID: id, Title: cellOf(row, cTitle), CI: cellOf(row, cCI), Category: cellOf(row, cCat), Priority: priority(cellOf(row, cPri)), State: cellOf(row, cState),
				Type: strings.ToLower(cellOf(row, cType)), Close: cellOf(row, cClose), CausedBy: cellOf(row, cCaused), Problem: cellOf(row, cProb),
				Opened: parseTime(cellOf(row, cOpen)), Resolved: parseTime(cellOf(row, cRes)), File: f.Path, Line: lines[k]}
			t.Reopened, _ = strconv.Atoi(cellOf(row, cReopen))
			if r.Secrets[f.Path][t.Line] {
				t.Title = "(title hidden: it may hold a password)"
			}
			it := strings.ToLower(cellOf(row, cIType))
			up := strings.ToUpper(id)
			switch {
			case strings.HasPrefix(up, "CHG") || strings.Contains(it, "change"):
				t.Kind = "change"
			case strings.HasPrefix(up, "PRB") || strings.Contains(it, "problem"):
				t.Kind = "problem"
			case strings.HasPrefix(up, "INC") || strings.Contains(it, "incident") || strings.Contains(it, "bug"):
				t.Kind = "incident"
			case strings.Contains(base, "change"):
				t.Kind = "change"
			case strings.Contains(base, "problem"):
				t.Kind = "problem"
			case strings.Contains(base, "incident"):
				t.Kind = "incident"
			default:
				continue
			}
			if t.Kind == "change" {
				switch {
				case strings.Contains(t.Type, "emerg") || strings.Contains(t.Type, "expedit"):
					t.Type = "emergency"
				case strings.Contains(t.Type, "standard") || strings.Contains(t.Type, "pre-approved"):
					t.Type = "standard"
				default:
					t.Type = "normal"
				}
			}
			used = true
			switch t.Kind {
			case "incident":
				m.Incidents = append(m.Incidents, t)
			case "change":
				m.Changes = append(m.Changes, t)
			default:
				m.Problems = append(m.Problems, t)
			}
			for _, d := range []time.Time{t.Opened, t.Resolved} {
				if d.IsZero() {
					continue
				}
				if m.From.IsZero() || d.Before(m.From) {
					m.From = d
				}
				if d.After(m.To) {
					m.To = d
				}
			}
		}
		if used {
			r.Cover("ird", f)
			m.Files = append(m.Files, f.Path)
		}
	}
	for _, ts := range [][]*Ticket{m.Incidents, m.Changes, m.Problems} {
		for _, t := range ts {
			t.Title = clean(t.Title, names)
		}
	}
	for _, ts := range [][]*Ticket{m.Incidents, m.Changes, m.Problems} {
		sort.SliceStable(ts, func(i, j int) bool { return ts[i].Opened.Before(ts[j].Opened) })
	}
	m.Ignored = sortedKeys(ignored)
	sort.Strings(m.Ignored)
	return m
}

// clean removes people's names (taken from the people columns), e-mail addresses, IP addresses, ticket numbers and long numbers.
func clean(s string, names map[string]bool) string {
	s = emailRe.ReplaceAllString(s, "{email}")
	s = ipRe.ReplaceAllString(s, "{address}")
	s = ticketRe.ReplaceAllString(s, "{ticket}")
	s = numRe.ReplaceAllString(s, "{n}")
	ws := strings.Fields(s)
	for i, w := range ws {
		if names[strings.ToLower(strings.Trim(w, ".,;:()'\"!?"))] {
			ws[i] = "{person}"
		}
	}
	return strings.Join(ws, " ")
}
