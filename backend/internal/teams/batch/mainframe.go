package batch

import (
	"encoding/xml"
	"io"
	"path"
	"regexp"
	"strings"
)

// Mainframe batch: Control-M XML exports (DEFTABLE), CA-7 job listings (LJOB) and the JCL members themselves.
// Like the other readers, these only read definitions; nothing is submitted.

// ---------- Control-M XML (DEFTABLE export) ----------

func (m *BModel) parseControlMXML(f *File) bool {
	text := strings.Join(f.Lines, "\n")
	d := xml.NewDecoder(strings.NewReader(text))
	d.Strict = false
	folder := ""
	var cur *Job
	n := 0
	for {
		off := int(d.InputOffset())
		tok, err := d.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return n > 0
		}
		switch t := tok.(type) {
		case xml.StartElement:
			a := map[string]string{}
			for _, at := range t.Attr {
				a[at.Name.Local] = at.Value
			}
			switch t.Name.Local {
			case "FOLDER", "SMART_FOLDER", "TABLE", "SMART_TABLE", "SCHED_TABLE":
				folder = firstOf(a["FOLDER_NAME"], a["TABLE_NAME"], a["JOBNAME"])
			case "JOB":
				name := firstOf(a["JOBNAME"], a["MEMNAME"])
				j := m.job(name, "Control-M", f, lineOf(f, off+1))
				if j == nil {
					continue
				}
				cur = j
				j.Folder, j.Kind = folder, a["TASKTYPE"]
				j.Desc = a["DESCRIPTION"]
				j.Host = a["NODEID"]
				j.RunAs = firstOf(a["RUN_AS"], a["OWNER"])
				if a["CMDLINE"] != "" {
					j.Command = a["CMDLINE"]
				} else if a["MEMNAME"] != "" {
					j.Command = "JCL member " + a["MEMNAME"]
					if a["MEMLIB"] != "" {
						j.Command += " in " + a["MEMLIB"]
					}
				}
				j.CmdLine = j.Line
				parts := []string{}
				if days := a["DAYS"]; days != "" {
					parts = append(parts, map[bool]string{true: "every day", false: "days " + days}[strings.EqualFold(days, "ALL")])
				}
				if wd := a["WEEKDAYS"]; wd != "" {
					parts = append(parts, "weekdays "+wd)
				}
				if t := a["TIMEFROM"]; len(t) == 4 {
					j.Start = t[:2] + ":" + t[2:]
					parts = append(parts, "from "+j.Start)
				}
				if folder != "" && monthEndRe.MatchString(folder) {
					parts = append(parts, "(folder "+folder+")")
				}
				j.Schedule = strings.Join(parts, " ")
				n++
			case "INCOND":
				if cur != nil && a["NAME"] != "" {
					k := strings.ToLower(cur.Name)
					m.waits[k] = append(m.waits[k], a["NAME"])
				}
			case "OUTCOND":
				if cur != nil && a["NAME"] != "" && a["SIGN"] != "-" {
					m.events[a["NAME"]] = strings.ToLower(cur.Name)
				}
			}
		case xml.EndElement:
			if t.Name.Local == "JOB" {
				cur = nil
			}
		}
	}
	return n > 0
}

func firstOf(xs ...string) string {
	for _, x := range xs {
		if strings.TrimSpace(x) != "" {
			return strings.TrimSpace(x)
		}
	}
	return ""
}

// ---------- CA-7 (LJOB,JOB=…,LIST=ALL listing) ----------

var (
	ca7Re        = regexp.MustCompile(`LJOB,JOB=`)
	ca7JobRe     = regexp.MustCompile(`^\s*\d?LJOB,JOB=([\w@#$]+)`)
	ca7TrigRe    = regexp.MustCompile(`^\s+JOB=([\w@#$]+)\s+SCHID=(\d+)`)
	ca7SectionRe = regexp.MustCompile(`^\s*-{5,}\s*([A-Z][A-Z ]+?)\s*-{5,}`)
)

func (m *BModel) parseCA7(f *File) bool {
	var cur *Job
	section := ""
	n := 0
	for i, l := range f.Lines {
		if mm := ca7JobRe.FindStringSubmatch(l); mm != nil {
			cur = m.job(mm[1], "CA-7", f, i+1)
			section = ""
			if cur != nil {
				if cur.Command == "" {
					cur.Command = "JCL member " + mm[1]
					cur.CmdLine = i + 1
				}
				n++
			}
			continue
		}
		if mm := ca7SectionRe.FindStringSubmatch(l); mm != nil {
			section = mm[1]
			continue
		}
		if cur == nil {
			continue
		}
		// a triggered job runs after the job that triggers it
		if section == "TRIGGERED JOBS" {
			if mm := ca7TrigRe.FindStringSubmatch(l); mm != nil {
				if t := m.job(mm[1], "CA-7", f, i+1); t != nil && t != cur {
					t.After = appendUnique(t.After, cur.Name)
					if t.Schedule == "" {
						t.Schedule = "triggered by " + cur.Name + " (schedule " + mm[2] + ")"
					}
				}
			}
		}
	}
	return n > 0
}

// ---------- JCL members ----------

var (
	jclJobCardRe = regexp.MustCompile(`^//([A-Z@#$][\w@#$]{0,7})\s+JOB\b`)
	jclExecPgmRe = regexp.MustCompile(`^//[\w@#$]*\s+EXEC\s+(?:PGM=([\w@#$]+)|(?:PROC=)?([\w@#$]+))`)
)

// readJCL links each scheduled job to the programs its JCL member runs, and lists JCL jobs that no
// scheduler export mentions (they are run by hand, by another scheduler, or no longer used).
func (m *BModel) readJCL(r *Run) {
	for _, f := range r.Code.Files {
		if f.Lang != "JCL" {
			continue
		}
		name, line := "", 0
		var progs []string
		firstExec := 0
		for i, l := range f.Lines {
			if strings.HasPrefix(l, "//*") {
				continue
			}
			if mm := jclJobCardRe.FindStringSubmatch(l); mm != nil && name == "" {
				name, line = mm[1], i+1
			}
			if mm := jclExecPgmRe.FindStringSubmatch(l); mm != nil {
				p := mm[1] + mm[2]
				if !contains(progs, p) {
					progs = append(progs, p)
				}
				if firstExec == 0 {
					firstExec = i + 1
				}
			}
		}
		if name == "" { // a procedure (PROC), not a job
			continue
		}
		r.Cover("bjr", f)
		member := strings.ToUpper(strings.TrimSuffix(path.Base(f.Path), path.Ext(f.Path)))
		j := m.Jobs[strings.ToLower(member)]
		if j == nil {
			j = m.Jobs[strings.ToLower(name)]
		}
		if j == nil {
			j = m.job(name, "JCL only (in no scheduler export)", f, line)
			j.Schedule = ""
		}
		if len(progs) > 0 {
			run := "runs " + strings.Join(progs, ", ")
			if j.Command == "" || strings.HasPrefix(j.Command, "JCL member") {
				j.Command = strings.TrimSpace(j.Command + " " + run)
				j.CmdLine = firstExec
				if j.File != f.Path {
					// keep the scheduler definition as the job's source; the JCL is where the command is
					j.CmdLine = 0
				}
			}
		}
		if !contains(m.Files, f.Path) {
			m.Files = append(m.Files, f.Path)
		}
	}
}
