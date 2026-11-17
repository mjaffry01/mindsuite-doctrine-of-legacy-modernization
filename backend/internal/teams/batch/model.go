package batch

import (
	"encoding/csv"
	"encoding/json"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// BModel is what the batch & scheduling team knows: jobs, their schedules, the order they run in, the hosts
// and accounts they run on, and (optionally) their run history. It is built only from scheduler exports that
// read definitions (Control-M Automation API JSON, AutoSys JIL, IBM Workload Scheduler composer output,
// Windows schtasks CSV, crontab listings) and a run-history CSV. Nothing is started or changed.

// Job is one scheduled job.
type Job struct {
	Name, Scheduler, Folder, Kind string
	Command, Host, RunAs, Desc    string
	Schedule, Start               string // human-readable schedule, start time "HH:MM"
	After                         []string
	Disabled                      bool
	MonthEnd                      bool
	File                          string
	Line                          int
	CmdLine                       int // line of the command (for secrets and quotes)
}

// RunRec is one run from the history.
type RunRec struct {
	Job        string
	Start, End time.Time
	OK         bool
	File       string
	Line       int
}

// BModel holds the parsed batch estate.
type BModel struct {
	Jobs       map[string]*Job
	Runs       []*RunRec
	Schedulers []string
	Files      []string
	Scripts    map[string]*File // uploaded scripts by base name
	events     map[string]string
	waits      map[string][]string
	// MissingEvents are Control-M events a job waits for that no job in the export adds.
	MissingEvents []string
}

// JobList returns jobs sorted by name.
func (m *BModel) JobList() []*Job {
	var out []*Job
	for _, k := range sortedKeys(m.Jobs) {
		out = append(out, m.Jobs[k])
	}
	return out
}

// Summary is shown with the run.
func (m *BModel) Summary() map[string]any {
	hosts := map[string]bool{}
	dis := 0
	for _, j := range m.Jobs {
		if j.Host != "" {
			hosts[j.Host] = true
		}
		if j.Disabled {
			dis++
		}
	}
	return map[string]any{"jobs": len(m.Jobs), "disabled": dis, "schedulers": m.Schedulers, "hosts": len(hosts), "runs": len(m.Runs), "exportFiles": len(m.Files)}
}

func (m *BModel) job(name, sched string, f *File, line int) *Job {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	if j := m.Jobs[strings.ToLower(name)]; j != nil {
		return j
	}
	j := &Job{Name: name, Scheduler: sched, File: f.Path, Line: line}
	m.Jobs[strings.ToLower(name)] = j
	if !contains(m.Schedulers, sched) {
		m.Schedulers = append(m.Schedulers, sched)
	}
	return j
}

var monthEndRe = regexp.MustCompile(`(?i)month[-_ ]?end|monthend|last[-_ ]?day|lastday|eom|year[-_ ]?end|\b28-31\b|\bL\b`)

// ParseBatch reads every file of the run.
func ParseBatch(r *Run) *BModel {
	m := &BModel{Jobs: map[string]*Job{}, Scripts: map[string]*File{}, events: map[string]string{}, waits: map[string][]string{}}
	for _, f := range r.Code.Files {
		for i, l := range f.Lines {
			if looksSecret(l) || cmdPwdRe.MatchString(l) {
				r.MarkSecret(f.Path, i+1)
			}
		}
		head := strings.Join(f.Lines[:min(len(f.Lines), 60)], "\n")
		used := false
		switch {
		case f.Lang == "JSON":
			used = m.parseControlM(f)
		case jilRe.MatchString(head):
			used = m.parseJIL(f)
		case twsRe.MatchString(head):
			used = m.parseTWS(f)
		case f.Lang == "CSV":
			used = m.parseSchtasks(f) || m.parseHistory(f)
		case cronLineRe.MatchString(head) && (f.Lang == "Text" || f.Lang == "" || strings.Contains(strings.ToLower(f.Path), "cron")):
			used = m.parseCron(f)
		}
		if used {
			r.Cover("bjr", f)
			m.Files = append(m.Files, f.Path)
		} else if f.Lang == "Shell" || f.Lang == "PowerShell" || f.Lang == "Batch" || f.Lang == "SQL" || f.Lang == "Python" || f.Lang == "JCL" {
			m.Scripts[strings.ToLower(path.Base(f.Path))] = f
		}
	}
	// Control-M dependencies: a job waiting for an event follows the job that adds it
	for job, evs := range m.waits {
		for _, ev := range evs {
			if src := m.events[ev]; src != "" && src != job {
				if j := m.Jobs[job]; j != nil {
					j.After = appendUnique(j.After, m.Jobs[src].Name)
				}
			} else if src == "" && m.Jobs[job] != nil {
				m.MissingEvents = append(m.MissingEvents, "event "+ev+" (needed by "+m.Jobs[job].Name+")")
			}
		}
	}
	for _, j := range m.Jobs {
		if monthEndRe.MatchString(j.Schedule + " " + j.Name) {
			j.MonthEnd = true
		}
	}
	return m
}

var cmdPwdRe = regexp.MustCompile(`(?i)((^|\s)-p\s*[^\s-]\S{3,}|--password[= ]\S+|\s/p(?:assword)?[: ]\S{4,}|\b\w+/\w{4,}@\w+)`)

// ---------- Control-M (ctm deploy jobs::get) ----------

func lineOf2(f *File, needle string) int {
	for i, l := range f.Lines {
		if strings.Contains(l, needle) {
			return i + 1
		}
	}
	return 1
}

func (m *BModel) parseControlM(f *File) bool {
	var doc map[string]any
	if json.Unmarshal([]byte(strings.Join(f.Lines, "\n")), &doc) != nil {
		return false
	}
	n := 0
	var walk func(name string, v map[string]any, folder string)
	walk = func(name string, v map[string]any, folder string) {
		typ, _ := v["Type"].(string)
		switch {
		case strings.HasPrefix(typ, "Folder") || strings.HasPrefix(typ, "SimpleFolder") || strings.HasPrefix(typ, "SubFolder"):
			for k, c := range v {
				if cm, ok := c.(map[string]any); ok {
					walk(k, cm, name)
				}
			}
		case strings.HasPrefix(typ, "Job"):
			j := m.job(name, "Control-M", f, lineOf2(f, `"`+name+`"`))
			j.Folder, j.Kind = folder, strings.TrimPrefix(typ, "Job:")
			j.Command, _ = v["Command"].(string)
			if j.Command == "" {
				if fp, _ := v["FilePath"].(string); fp != "" {
					fn, _ := v["FileName"].(string)
					j.Command = strings.TrimRight(fp, "/") + "/" + fn
				}
			}
			if j.Command != "" {
				j.CmdLine = lineOf2(f, j.Command)
			}
			j.Host, _ = v["Host"].(string)
			j.RunAs, _ = v["RunAs"].(string)
			j.Desc, _ = v["Description"].(string)
			if w, ok := v["When"].(map[string]any); ok {
				parts := []string{}
				if s, _ := w["Schedule"].(string); s != "" {
					parts = append(parts, s)
				}
				if d, ok := w["WeekDays"].([]any); ok {
					ds := []string{}
					for _, x := range d {
						ds = append(ds, x.(string))
					}
					parts = append(parts, strings.Join(ds, ","))
				}
				if d, ok := w["MonthDays"].([]any); ok {
					ds := []string{}
					for _, x := range d {
						ds = append(ds, x.(string))
					}
					parts = append(parts, "month days "+strings.Join(ds, ","))
				}
				if t, _ := w["FromTime"].(string); len(t) == 4 {
					j.Start = t[:2] + ":" + t[2:]
					parts = append(parts, "from "+j.Start)
				}
				j.Schedule = strings.Join(parts, " ")
			}
			for k, ev := range v {
				em, ok := ev.(map[string]any)
				if !ok {
					continue
				}
				et, _ := em["Type"].(string)
				evs, _ := em["Events"].([]any)
				for _, e := range evs {
					en, _ := e.(map[string]any)["Event"].(string)
					switch {
					case et == "WaitForEvents" || strings.HasPrefix(k, "eventsToWaitFor"):
						m.waits[strings.ToLower(name)] = append(m.waits[strings.ToLower(name)], en)
					case et == "AddEvents" || strings.HasPrefix(k, "eventsToAdd"):
						m.events[en] = strings.ToLower(name)
					}
				}
			}
			n++
		}
	}
	for k, v := range doc {
		if vm, ok := v.(map[string]any); ok {
			walk(k, vm, "")
		}
	}
	return n > 0
}

// ---------- AutoSys JIL ----------

var (
	jilRe     = regexp.MustCompile(`(?m)^\s*(insert_job|update_job)\s*:`)
	jilAttrRe = regexp.MustCompile(`([a-z_]+)\s*:\s*("[^"]*"|\S+)`)
	condJobRe = regexp.MustCompile(`(?i)\b(?:s|success|d|done|f|failure|t|terminated|n|notrunning)\(\s*([\w.#-]+)`)
)

func (m *BModel) parseJIL(f *File) bool {
	var j *Job
	n := 0
	for i, l := range f.Lines {
		t := strings.TrimSpace(l)
		if t == "" || strings.HasPrefix(t, "/*") {
			continue
		}
		for _, a := range jilAttrRe.FindAllStringSubmatch(t, -1) {
			k, v := a[1], strings.Trim(a[2], `"`)
			switch k {
			case "insert_job", "update_job":
				j = m.job(v, "AutoSys", f, i+1)
				n++
				continue
			}
			if j == nil {
				continue
			}
			switch k {
			case "job_type":
				j.Kind = map[string]string{"c": "command", "cmd": "command", "b": "box", "box": "box", "f": "file watcher", "fw": "file watcher"}[strings.ToLower(v)]
				if j.Kind == "" {
					j.Kind = v
				}
			case "box_name":
				j.Folder = v
			case "command":
				j.Command, j.CmdLine = strings.TrimSpace(strings.SplitN(t, ":", 2)[1]), i+1
			case "machine":
				j.Host = v
			case "owner":
				j.RunAs = strings.SplitN(v, "@", 2)[0]
			case "description":
				j.Desc = strings.Trim(strings.TrimSpace(strings.SplitN(t, ":", 2)[1]), `"`)
			case "condition":
				for _, c := range condJobRe.FindAllStringSubmatch(strings.SplitN(t, ":", 2)[1], -1) {
					j.After = appendUnique(j.After, c[1])
				}
			case "days_of_week", "run_calendar", "start_times", "start_mins":
				val := strings.Trim(strings.TrimSpace(strings.SplitN(t, ":", 2)[1]), `"`)
				j.Schedule = strings.TrimSpace(j.Schedule + " " + k + "=" + val)
				if k == "start_times" && len(val) >= 4 {
					j.Start = strings.SplitN(val, ",", 2)[0]
				}
			case "status":
				if strings.EqualFold(v, "on_ice") || strings.EqualFold(v, "on_hold") {
					j.Disabled = true
				}
			}
		}
	}
	return n > 0
}

// ---------- IBM Workload Scheduler (composer extract) ----------

var (
	twsRe       = regexp.MustCompile(`(?im)^\s*SCHEDULE\s+\S+`)
	twsSchedRe  = regexp.MustCompile(`(?i)^\s*SCHEDULE\s+(\S+)`)
	twsJobRe    = regexp.MustCompile(`^\s*([\w#@.-]+)(?:\s+(.*))?$`)
	twsFollows  = regexp.MustCompile(`(?i)\bFOLLOWS\s+([\w#@.,-]+)`)
	twsAtRe     = regexp.MustCompile(`(?i)\bAT\s+(\d{4})`)
	twsRunCycle = regexp.MustCompile(`(?i)\bON\s+(RUNCYCLE\s+\S+\s+)?"?([^"\n]+)"?`)
)

func (m *BModel) parseTWS(f *File) bool {
	stream, sched, start := "", "", ""
	n := 0
	inBody := false
	for i, l := range f.Lines {
		t := strings.TrimSpace(l)
		switch {
		case twsSchedRe.MatchString(t):
			stream = strings.SplitN(twsSchedRe.FindStringSubmatch(t)[1], "#", 2)[len(strings.SplitN(twsSchedRe.FindStringSubmatch(t)[1], "#", 2))-1]
			sched, start, inBody = "", "", false
		case strings.HasPrefix(strings.ToUpper(t), "ON "):
			if mm := twsRunCycle.FindStringSubmatch(t); mm != nil {
				sched = strings.TrimSpace(mm[2])
			}
		case strings.HasPrefix(strings.ToUpper(t), "AT "):
			if mm := twsAtRe.FindStringSubmatch(t); mm != nil {
				start = mm[1][:2] + ":" + mm[1][2:]
			}
		case t == ":":
			inBody = true
		case strings.EqualFold(t, "END"):
			inBody = false
		case inBody && t != "":
			mm := twsJobRe.FindStringSubmatch(t)
			if mm == nil {
				continue
			}
			name := mm[1]
			if k := strings.LastIndex(name, "#"); k >= 0 {
				name = name[k+1:]
			}
			j := m.job(name, "IBM Workload Scheduler", f, i+1)
			j.Folder, j.Schedule, j.Start = stream, sched, start
			if fm := twsFollows.FindStringSubmatch(t); fm != nil {
				for _, p := range strings.Split(fm[1], ",") {
					if k := strings.LastIndex(p, "#"); k >= 0 {
						p = p[k+1:]
					}
					p = strings.SplitN(p, ".", 2)[len(strings.SplitN(p, ".", 2))-1]
					j.After = appendUnique(j.After, p)
				}
			}
			n++
		}
	}
	return n > 0
}

// ---------- Windows Task Scheduler (schtasks /query /fo csv /v) ----------

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
				head = append(head, strings.ToLower(strings.TrimSpace(strings.TrimPrefix(h, string(rune(0xFEFF))))))
			}
			continue
		}
		if len(row) > 0 && strings.EqualFold(row[0], "HostName") { // schtasks repeats the header per folder
			continue
		}
		rows = append(rows, row)
		lines = append(lines, line)
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

func (m *BModel) parseSchtasks(f *File) bool {
	head, rows, lines := readTable(f)
	tn, cmd := colOf(head, "taskname"), colOf(head, "task to run")
	if tn < 0 || cmd < 0 {
		return false
	}
	host, user, typ, start := colOf(head, "hostname"), colOf(head, "run as user"), colOf(head, "schedule type"), colOf(head, "start time")
	state, days, comment := colOf(head, "scheduled task state"), colOf(head, "days"), colOf(head, "comment")
	n := 0
	for i, row := range rows {
		name := cellOf(row, tn)
		if name == "" || strings.HasPrefix(name, `\Microsoft\`) {
			continue
		}
		j := m.job(strings.TrimPrefix(name, `\`), "Windows Task Scheduler", f, lines[i])
		j.Command, j.CmdLine = cellOf(row, cmd), lines[i]
		j.Host, j.RunAs, j.Desc = cellOf(row, host), cellOf(row, user), cellOf(row, comment)
		j.Schedule = strings.TrimSpace(cellOf(row, typ) + " " + cellOf(row, days) + " " + cellOf(row, start))
		if s := cellOf(row, start); len(s) >= 5 {
			j.Start = s[:5]
		}
		if strings.EqualFold(cellOf(row, state), "disabled") {
			j.Disabled = true
		}
		n++
	}
	return n > 0
}

// ---------- run history (job, start, end, status) ----------

var histLayouts = []string{"2006-01-02 15:04:05", "2006-01-02T15:04:05Z07:00", "2006-01-02T15:04:05", "2006-01-02 15:04", "1/2/2006 3:04:05 PM", "1/2/2006 15:04", "20060102150405"}

func histTime(s string) time.Time {
	s = strings.TrimSpace(s)
	for _, l := range histLayouts {
		if t, err := time.Parse(l, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

func (m *BModel) parseHistory(f *File) bool {
	head, rows, lines := readTable(f)
	jc := colOf(head, "job", "jobname", "job_name", "job name", "name")
	sc := colOf(head, "start", "starttime", "start_time", "start time", "started")
	ec := colOf(head, "end", "endtime", "end_time", "end time", "ended")
	st := colOf(head, "status", "result", "ended_status", "endstatus")
	if jc < 0 || sc < 0 || ec < 0 || st < 0 {
		return false
	}
	for i, row := range rows {
		s, e := histTime(cellOf(row, sc)), histTime(cellOf(row, ec))
		if s.IsZero() {
			continue
		}
		status := strings.ToUpper(cellOf(row, st))
		ok := strings.Contains(status, "SUCCESS") || strings.Contains(status, "ENDED_OK") || status == "OK" || status == "0" || status == "SU" || status == "COMPLETED"
		m.Runs = append(m.Runs, &RunRec{Job: cellOf(row, jc), Start: s, End: e, OK: ok, File: f.Path, Line: lines[i]})
	}
	return len(m.Runs) > 0
}

// ---------- crontab ----------

var (
	cronLineRe = regexp.MustCompile(`(?m)^\s*#?\s*(@(reboot|yearly|annually|monthly|weekly|daily|hourly)|[\d*/,-]+\s+[\d*/,-]+\s+[\d*/,LW?-]+\s+[\d*/,A-Za-z-]+\s+[\d*/,A-Za-z-]+)\s+\S`)
	cronFields = regexp.MustCompile(`^\s*(#?)\s*(@\w+|([\d*/,-]+)\s+([\d*/,-]+)\s+([\d*/,LW?-]+)\s+([\d*/,A-Za-z-]+)\s+([\d*/,A-Za-z-]+))\s+(.+)$`)
	userRe     = regexp.MustCompile(`^[a-z_][a-z0-9_-]*$`)
)

func cronHuman(min, hour, dom, mon, dow string) (string, string) {
	start := ""
	if regexp.MustCompile(`^\d+$`).MatchString(min) && regexp.MustCompile(`^\d+$`).MatchString(hour) {
		start = pad2(hour) + ":" + pad2(min)
	}
	parts := []string{}
	switch {
	case strings.HasPrefix(min, "*/"):
		parts = append(parts, "every "+strings.TrimPrefix(min, "*/")+" minutes")
	case hour == "*":
		parts = append(parts, "every hour")
	case strings.HasPrefix(hour, "*/"):
		parts = append(parts, "every "+strings.TrimPrefix(hour, "*/")+" hours")
	case start != "":
		parts = append(parts, "at "+start)
	default:
		parts = append(parts, "at minute "+min+" hour "+hour)
	}
	if dow != "*" {
		parts = append(parts, "on weekdays "+dow)
	}
	if dom != "*" {
		parts = append(parts, "on days "+dom)
	}
	if mon != "*" {
		parts = append(parts, "in months "+mon)
	}
	return strings.Join(parts, " "), start
}

func pad2(s string) string {
	if len(s) == 1 {
		return "0" + s
	}
	return s
}

func (m *BModel) parseCron(f *File) bool {
	hint := strings.TrimSuffix(path.Base(f.Path), path.Ext(f.Path))
	for _, p := range []string{"crontab-", "crontab_", "crontab", "cron-", "cron_"} {
		hint = strings.TrimPrefix(hint, p)
	}
	host, user := hint, ""
	if i := strings.LastIndexAny(hint, "-_"); i > 0 {
		host, user = hint[:i], hint[i+1:]
	}
	systemFile := strings.Contains(f.Path, "cron.d") || strings.HasSuffix(f.Path, "/crontab")
	n := 0
	for i, l := range f.Lines {
		mm := cronFields.FindStringSubmatch(l)
		if mm == nil {
			continue
		}
		disabled := mm[1] == "#"
		cmd := strings.TrimSpace(mm[8])
		runAs := user
		if systemFile {
			fs := strings.SplitN(cmd, " ", 2)
			if len(fs) == 2 && userRe.MatchString(fs[0]) {
				runAs, cmd = fs[0], strings.TrimSpace(fs[1])
			}
		}
		sched, start := mm[2], ""
		if mm[3] != "" {
			sched, start = cronHuman(mm[3], mm[4], mm[5], mm[6], mm[7])
		}
		base := strings.Fields(cmd)[0]
		name := path.Base(base)
		if strings.Contains(name, "=") || name == "cd" || name == "." || name == "bash" || name == "sh" {
			for _, w := range strings.Fields(cmd) {
				if strings.Contains(w, "/") && !strings.HasPrefix(w, "-") {
					name = path.Base(w)
					break
				}
			}
		}
		name = host + ":" + name
		if m.Jobs[strings.ToLower(name)] != nil {
			name = name + "#" + strconv.Itoa(i+1) // same script twice: tell them apart by line
		}
		j := m.job(name, "cron", f, i+1)
		j.Command, j.CmdLine, j.Host, j.RunAs, j.Schedule, j.Start, j.Disabled = cmd, i+1, host, runAs, sched, start, disabled
		if mm[5] == "28-31" || strings.Contains(mm[5], "L") || strings.Contains(cmd, "date +%d") && strings.Contains(cmd, "-eq 1") {
			j.MonthEnd = true
		}
		n++
	}
	return n > 0
}

// ---------- graph helpers ----------

// Successors maps each job to the jobs that wait for it.
func (m *BModel) Successors() map[string][]string {
	out := map[string][]string{}
	for _, j := range m.Jobs {
		for _, p := range j.After {
			out[strings.ToLower(p)] = appendUnique(out[strings.ToLower(p)], j.Name)
		}
	}
	for k := range out {
		sort.Strings(out[k])
	}
	return out
}
