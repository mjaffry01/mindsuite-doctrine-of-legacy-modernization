package database

import (
	"encoding/csv"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// DBModel is what the database team knows about one database. It is built only from
// metadata: schema exports (DDL) and catalog query results (CSV). Rows of business data are never read.

// DBColumn is one column of a table.
type DBColumn struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	NotNull bool   `json:"notNull,omitempty"`
	File    string `json:"-"`
	Line    int    `json:"-"`
}

// DBFK is a foreign key.
type DBFK struct {
	Cols    []string
	Ref     string // referenced table key
	RefCols []string
	File    string
	Line    int
}

// DBCheck is a CHECK constraint: often a business rule written in the schema.
type DBCheck struct {
	Expr string
	File string
	Line int
}

// DBTable is a table or view.
type DBTable struct {
	Key, Schema, Name string
	View              bool
	File              string
	Line              int
	Cols              []DBColumn
	PK                []string
	FKs               []DBFK
	Indexes           [][]string
	Checks            []DBCheck
	Rows, SizeMB      float64
	HasStats          bool
}

// DBCode is a procedure, function, package, trigger or type body.
type DBCode struct {
	Key, Name, Type string
	File            string
	Line, End       int
	Lines           int
	Reads, Writes   []string // table keys
	RemoteWrites    []string
	Calls           []string // code keys
	Dynamic         int      // line of the first dynamic SQL, 0 = none
	Complexity      int
	OnTable         string // triggers
	Outside         []DBObj
}

// DBObj is a job, link, account, grant, synonym or similar object.
type DBObj struct {
	Name, Detail, Extra string
	File                string
	Line                int
}

// DBModel holds the parsed database.
type DBModel struct {
	Engine, Version          string
	VersionFile              string
	VersionLine              int
	Tables                   map[string]*DBTable
	Code                     map[string]*DBCode
	Jobs, Links, Admins      []DBObj
	PublicGrants, Synonyms   []DBObj
	Sequences                int
	DataInserts              int
	DataInsertFile           string
	Files                    []string
	byBase                   map[string][]string
	codeByBase               map[string]string
	ColumnsKnown, StatsKnown bool
	// EmbeddedSQL lists application programs with SQL in their code (COBOL EXEC SQL): the logic that
	// uses the database lives there, not in stored procedures.
	EmbeddedSQL []EmbeddedSQL
}

// Summary is shown with the run.
func (m *DBModel) Summary() map[string]any {
	tables, views, progs, trig, lines := 0, 0, 0, 0, 0
	for _, t := range m.Tables {
		if t.View {
			views++
		} else {
			tables++
		}
	}
	for _, c := range m.Code {
		if c.Type == "PACKAGE" && m.Code[strings.TrimSuffix(c.Key, " (spec)")] != nil {
			continue // counted with its body
		}
		if c.Type == "TRIGGER" {
			trig++
		} else {
			progs++
		}
		lines += c.Lines
	}
	return map[string]any{"engine": m.Engine, "version": m.Version, "tables": tables, "views": views, "programs": progs, "triggers": trig,
		"codeLines": lines, "jobs": len(m.Jobs), "links": len(m.Links), "sizeGb": m.SizeGB(), "exportFiles": len(m.Files)}
}

// SizeGB sums table sizes from catalog statistics (0 when unknown).
func (m *DBModel) SizeGB() float64 {
	mb := 0.0
	for _, t := range m.Tables {
		mb += t.SizeMB
	}
	return float64(int(mb/1024*10+0.5)) / 10
}

// Schemas lists schema names ("(default)" when the export has none).
func (m *DBModel) Schemas() []string {
	set := map[string]bool{}
	for _, t := range m.Tables {
		set[schemaLabel(t.Schema)] = true
	}
	for _, c := range m.Code {
		s, _ := splitName(c.Key)
		set[schemaLabel(s)] = true
	}
	return sortedKeys(set)
}

func schemaLabel(s string) string {
	if s == "" {
		return "(default)"
	}
	return s
}

var nameCleaner = strings.NewReplacer(`"`, "", "`", "", "[", "", "]", "")

// splitName returns SCHEMA and NAME in upper case.
func splitName(s string) (string, string) {
	s = strings.ToUpper(strings.TrimSpace(nameCleaner.Replace(s)))
	s = strings.TrimRight(s, ";,(")
	if i := strings.LastIndex(s, "."); i >= 0 {
		schema := s[:i]
		if j := strings.LastIndex(schema, "."); j >= 0 { // db.schema.name
			schema = schema[j+1:]
		}
		return schema, s[i+1:]
	}
	return "", s
}

func joinName(schema, name string) string {
	if schema == "" {
		return name
	}
	return schema + "." + name
}

// table returns (creating if needed) the table with this name.
func (m *DBModel) table(raw string) *DBTable {
	schema, name := splitName(raw)
	key := joinName(schema, name)
	if t := m.Tables[key]; t != nil {
		return t
	}
	if keys := m.byBase[name]; len(keys) > 0 {
		if schema == "" { // an unqualified name matches an existing qualified table
			return m.Tables[keys[0]]
		}
		if t := m.Tables[name]; t != nil && t.Schema == "" { // a qualified name matches a table first seen unqualified
			return t
		}
	}
	t := &DBTable{Key: key, Schema: schema, Name: name}
	m.Tables[key] = t
	m.byBase[name] = append(m.byBase[name], key)
	return t
}

// Find resolves a table or view name used in code ("" when unknown).
func (m *DBModel) Find(raw string) string {
	schema, name := splitName(raw)
	if name == "" {
		return ""
	}
	if t := m.Tables[joinName(schema, name)]; t != nil {
		return t.Key
	}
	if keys := m.byBase[name]; len(keys) > 0 {
		return keys[0]
	}
	return ""
}

var (
	reCreateTable = regexp.MustCompile("(?i)\\bcreate\\s+(?:global\\s+temporary\\s+|temporary\\s+|temp\\s+|unlogged\\s+)?table\\s+(?:if\\s+not\\s+exists\\s+)?([\\w$#.\"`\\[\\]]+)\\s*\\(")
	reCreateView  = regexp.MustCompile("(?i)\\bcreate\\s+(?:or\\s+(?:replace|alter)\\s+)?(?:force\\s+|noforce\\s+)?(?:materialized\\s+)?view\\s+(?:if\\s+not\\s+exists\\s+)?([\\w$#.\"`\\[\\]]+)")
	reCode        = regexp.MustCompile("(?i)\\bcreate\\s+(?:or\\s+(?:replace|alter)\\s+)?(?:(?:editionable|noneditionable)\\s+)?(?:definer\\s*=\\s*\\S+\\s+)?(procedure|proc|function|package\\s+body|package|trigger|type\\s+body)\\s+(?:if\\s+not\\s+exists\\s+)?([\\w$#.\"`\\[\\]]+)")
	reStmtStart   = regexp.MustCompile(`(?im)^[ \t]*(?:create\b|alter\s+table\b|grant\b|go[ \t]*$|/[ \t]*$|begin\s+dbms_scheduler|exec(?:ute)?\s+(?:msdb\.)?(?:dbo\.)?sp_add)`)
	reAlterCons   = regexp.MustCompile("(?i)\\balter\\s+table\\s+(?:only\\s+)?(?:if\\s+exists\\s+)?([\\w$#.\"`\\[\\]]+)\\s+(?:with\\s+(?:no)?check\\s+)?add\\s+(?:constraint\\s+[\\w$#\"`\\[\\]]+\\s+)?((?:primary\\s+key|foreign\\s+key|unique|check)\\b)")
	reIndex       = regexp.MustCompile("(?i)\\bcreate\\s+(?:unique\\s+)?(?:clustered\\s+|nonclustered\\s+|bitmap\\s+)?index\\s+(?:if\\s+not\\s+exists\\s+)?[\\w$#.\"`\\[\\]]+\\s+on\\s+([\\w$#.\"`\\[\\]]+)\\s*\\(([^)]*)\\)")
	reSequence    = regexp.MustCompile(`(?i)\bcreate\s+sequence\b`)
	reSynonym     = regexp.MustCompile("(?i)\\bcreate\\s+(?:or\\s+replace\\s+)?(?:public\\s+)?synonym\\s+([\\w$#.\"`]+)\\s+for\\s+([\\w$#.\"`@]+)")
	reDataInsert  = regexp.MustCompile(`(?im)^[ \t]*insert\s+into\s+\S+.*\bvalues\b`)

	reJobOracle   = regexp.MustCompile(`(?i)create_job\s*\(\s*job_name\s*=>\s*'([^']+)'`)
	reJobRepeat   = regexp.MustCompile(`(?i)repeat_interval\s*=>\s*'([^']*)'`)
	reJobAction   = regexp.MustCompile(`(?i)job_action\s*=>\s*'([^']*)'`)
	reJobSubmit   = regexp.MustCompile(`(?i)dbms_job\.submit\s*\(\s*[:\w]+\s*,\s*'([^']+)'(?:\s*,\s*[^,]+,\s*'([^']*)')?`)
	reJobMSSQL    = regexp.MustCompile(`(?i)sp_add_job\s+@job_name\s*=\s*N?'([^']+)'`)
	reJobStepCmd  = regexp.MustCompile(`(?i)@command\s*=\s*N?'([^']*)'`)
	reJobSchedule = regexp.MustCompile(`(?i)sp_add_(?:job)?schedule\b[^;]*?@(?:freq_type|name)\s*=\s*N?'?([^',\s]+)`)
	reJobEvent    = regexp.MustCompile("(?i)\\bcreate\\s+(?:definer\\s*=\\s*\\S+\\s+)?event\\s+(?:if\\s+not\\s+exists\\s+)?([\\w$`.]+)\\s+on\\s+schedule\\s+([^\\n]+?)(?:\\s+do\\b|$)")
	reEventDo     = regexp.MustCompile(`(?i)\bdo\s+(?:call\s+)?([\w$.` + "`" + `]+)`)
	reJobCron     = regexp.MustCompile(`(?i)cron\.schedule\s*\(\s*'([^']+)'\s*,\s*'([^']+)'\s*,\s*'([^']*)'`)

	reLinkOracle = regexp.MustCompile(`(?i)\bcreate\s+(?:shared\s+)?(?:public\s+)?database\s+link\s+([\w$#."]+)`)
	reLinkUsing  = regexp.MustCompile(`(?i)\busing\s+'([^']+)'`)
	reLinkMSSQL  = regexp.MustCompile(`(?i)sp_addlinkedserver\s+@server\s*=\s*N?'([^']+)'(?:[^;]*?@datasrc\s*=\s*N?'([^']+)')?`)
	reLinkPG     = regexp.MustCompile(`(?i)\bcreate\s+server\s+([\w"]+)\s+foreign\s+data\s+wrapper\s+(\w+)(?:[^;]*?\bhost\s+'([^']+)')?`)
	reIdentified = regexp.MustCompile(`(?i)\bidentified\s+by\b`)

	reAdminDBA    = regexp.MustCompile(`(?i)\bgrant\s+dba\s+to\s+([\w$#"]+)`)
	reAdminSys    = regexp.MustCompile(`(?i)alter\s+server\s+role\s+\[?sysadmin\]?\s+add\s+member\s+\[?([^\];\s]+)`)
	reAdminSys2   = regexp.MustCompile(`(?i)sp_addsrvrolemember\s+@loginame\s*=\s*N?'([^']+)'\s*,\s*@rolename\s*=\s*N?'sysadmin'`)
	reAdminMySQL  = regexp.MustCompile(`(?i)grant\s+all\s+privileges\s+on\s+\*\.\*\s+to\s+([^\s;]+)`)
	reAdminPG     = regexp.MustCompile(`(?i)(?:create|alter)\s+(?:role|user)\s+([\w"]+)[^;]*\bsuperuser\b`)
	reAdminAny    = regexp.MustCompile(`(?i)\bgrant\s+(?:select|insert|update|delete|execute|create|alter|drop)\s+any\s+\w+(?:\s+\w+)?\s+to\s+([\w$#"]+)`)
	rePublicGrant = regexp.MustCompile("(?i)\\bgrant\\s+([\\w\\s,]+?)\\s+on\\s+([\\w$#.\"`\\[\\]]+)\\s+to\\s+public\\b")

	reVersions = []struct {
		engine string
		re     *regexp.Regexp
	}{
		{"oracle", regexp.MustCompile(`(?i)Oracle Database\s+(\d+[a-z]?)(?:\s+(?:Enterprise|Standard|Express)\s+Edition)?(?:\s+Release\s+([\d.]+))?`)},
		{"sqlserver", regexp.MustCompile(`(?i)Microsoft SQL Server\s+(\d{4})`)},
		{"postgresql", regexp.MustCompile(`(?i)PostgreSQL\s+(?:database\s+(?:dump|version)\s*)?(\d+(?:\.\d+)?)`)},
		{"mysql", regexp.MustCompile(`(?i)(?:Distrib|Server version:?)\s+(\d+\.\d+(?:\.\d+)?)`)},
		{"postgresql", regexp.MustCompile(`(?i)Dumped from database version\s+(\d+(?:\.\d+)?)`)},
		{"db2", regexp.MustCompile(`(?i)DB2\s+(?:for\s+\S+\s+)?(?:v(?:ersion)?\s*)?(\d+\.\d+)`)},
	}
)

// ParseDB reads every export file of the run into a DBModel.
func ParseDB(r *Run) *DBModel {
	m := &DBModel{Tables: map[string]*DBTable{}, Code: map[string]*DBCode{}, byBase: map[string][]string{}, codeByBase: map[string]string{}}
	engines := map[string]int{}
	type seg struct {
		f   *File
		c   *DBCode
		off int
		end int
	}
	var segs []seg
	for _, f := range r.Code.Files {
		text := strings.Join(f.Lines, "\n")
		for i, l := range f.Lines {
			if looksSecret(l) || reIdentified.MatchString(l) {
				r.MarkSecret(f.Path, i+1)
			}
		}
		if f.Lang == "CSV" {
			if m.parseCatalogCSV(r, f) {
				r.Cover("cat", f)
				m.Files = append(m.Files, f.Path)
				engines[guessDBEngine(text)]++
			}
			continue
		}
		if f.Lang == "COBOL" || f.Lang == "COBOL copybook" || f.Lang == "JCL" || f.Lang == "Control cards" || f.Lang == "PL/I" {
			// DB2 for z/OS shows in the programs and jobs around the database, not in plain CREATE TABLE text
			if db2MarkerRe.MatchString(text) {
				engines["db2"]++
			}
			if f.Lang == "COBOL" || f.Lang == "PL/I" {
				m.noteEmbeddedSQL(f)
			}
			continue
		}
		if !(f.Lang == "SQL" || f.Lang == "PL/SQL" || f.Lang == "Text") {
			continue
		}
		r.Cover("cat", f)
		m.Files = append(m.Files, f.Path)
		engines[guessDBEngine(text)]++
		m.findVersion(f)
		starts := reStmtStart.FindAllStringIndex(text, -1)
		nextStart := func(after int) int {
			for _, s := range starts {
				if s[0] > after {
					return s[0]
				}
			}
			return len(text)
		}
		for _, loc := range reCreateTable.FindAllStringSubmatchIndex(text, -1) {
			t := m.table(text[loc[2]:loc[3]])
			t.File, t.Line = f.Path, lineOf(f, loc[0])
			open := loc[1] - 1
			close := matchParen(text, open)
			if close < 0 {
				continue
			}
			m.parseTableBody(f, t, text, open+1, close)
		}
		for _, loc := range reCreateView.FindAllStringSubmatchIndex(text, -1) {
			t := m.table(text[loc[2]:loc[3]])
			t.View, t.File, t.Line = true, f.Path, lineOf(f, loc[0])
		}
		for _, loc := range reAlterCons.FindAllStringSubmatchIndex(text, -1) {
			t := m.table(text[loc[2]:loc[3]])
			end := strings.IndexByte(text[loc[4]:], ';')
			if end < 0 || end > 4000 {
				end = min(len(text)-loc[4], 4000)
			}
			m.parseItem(f, t, text[loc[4]:loc[4]+end], loc[4], true)
		}
		for _, loc := range reIndex.FindAllStringSubmatchIndex(text, -1) {
			t := m.table(text[loc[2]:loc[3]])
			t.Indexes = append(t.Indexes, colList(text[loc[4]:loc[5]]))
		}
		m.Sequences += len(reSequence.FindAllStringIndex(text, -1))
		hits := reCode.FindAllStringSubmatchIndex(text, -1)
		for _, loc := range hits {
			typ := strings.ToUpper(strings.Join(strings.Fields(text[loc[2]:loc[3]]), " "))
			if typ == "PROC" {
				typ = "PROCEDURE"
			}
			schema, name := splitName(text[loc[4]:loc[5]])
			key := joinName(schema, name)
			switch typ {
			case "PACKAGE":
				key += " (spec)"
			case "TYPE BODY":
				key += " (type body)"
			}
			end := nextStart(loc[1])
			c := &DBCode{Key: key, Name: joinName(schema, name), Type: typ, File: f.Path, Line: lineOf(f, loc[0]), End: lineOf(f, max(loc[0], end-1))}
			c.Lines = c.End - c.Line + 1
			m.Code[key] = c
			if typ != "PACKAGE" {
				m.codeByBase[name] = key
			} else if _, ok := m.codeByBase[name]; !ok {
				m.codeByBase[name] = key
			}
			segs = append(segs, seg{f, c, loc[0], end})
		}
		// INSERT ... VALUES outside stored code means the export carries data rows: count them, never read them.
		for _, loc := range reDataInsert.FindAllStringIndex(text, -1) {
			inCode := false
			for _, sg := range segs {
				if sg.f == f && loc[0] >= sg.off && loc[0] < sg.end {
					inCode = true
					break
				}
			}
			if !inCode {
				m.DataInserts++
				if m.DataInsertFile == "" {
					m.DataInsertFile = f.Path
				}
			}
		}
		for _, loc := range reSynonym.FindAllStringSubmatchIndex(text, -1) {
			m.Synonyms = append(m.Synonyms, DBObj{Name: strings.ToUpper(nameCleaner.Replace(text[loc[2]:loc[3]])), Detail: strings.ToUpper(nameCleaner.Replace(text[loc[4]:loc[5]])), File: f.Path, Line: lineOf(f, loc[0])})
		}
		m.parseJobsLinksAccounts(f, text)
	}
	// second pass: what each program reads, writes and calls (tables are all known now)
	for _, sg := range segs {
		m.analyseCode(sg.f, sg.c)
	}
	// the same job can come from the DDL and from the catalog result: keep the richer one
	seenJob := map[string]int{}
	var jobs []DBObj
	for _, j := range m.Jobs {
		_, base := splitName(j.Name)
		if i, ok := seenJob[base]; ok {
			if jobs[i].Extra == "" && j.Extra != "" {
				jobs[i] = j
			}
			continue
		}
		seenJob[base] = len(jobs)
		jobs = append(jobs, j)
	}
	m.Jobs = jobs
	best, n := "", 0
	for e, c := range engines {
		if e != "" && c > n {
			best, n = e, c
		}
	}
	if m.Engine == "" {
		m.Engine = best
	}
	for _, t := range m.Tables {
		if len(t.Cols) > 0 {
			m.ColumnsKnown = true
		}
		if t.HasStats {
			m.StatsKnown = true
		}
	}
	return m
}

func (m *DBModel) findVersion(f *File) {
	if m.Version != "" {
		return
	}
	for i, l := range f.Lines {
		if i > 80 {
			return
		}
		for _, v := range reVersions {
			if mm := v.re.FindStringSubmatch(l); mm != nil {
				m.Engine, m.Version, m.VersionFile, m.VersionLine = v.engine, mm[1], f.Path, i+1
				if len(mm) > 2 && mm[2] != "" {
					m.Version += " (" + mm[2] + ")"
				}
				return
			}
		}
	}
}

func guessDBEngine(text string) string {
	switch {
	case regexp.MustCompile(`(?i)VARCHAR2|NUMBER\(|PL/SQL|CREATE\s+(OR\s+REPLACE\s+)?PACKAGE|DBMS_`).MatchString(text):
		return "oracle"
	case regexp.MustCompile(`(?im)\bNVARCHAR\b|\[dbo\]|^\s*GO\s*$|sp_add|IDENTITY\(`).MatchString(text):
		return "sqlserver"
	case regexp.MustCompile("(?i)ENGINE\\s*=\\s*InnoDB|AUTO_INCREMENT|`\\w+`").MatchString(text):
		return "mysql"
	case regexp.MustCompile(`(?i)\bSERIAL\b|::\w+|LANGUAGE\s+plpgsql|CREATE\s+EXTENSION`).MatchString(text):
		return "postgresql"
	case regexp.MustCompile(`(?i)db2look|CONNECT\s+TO|ORGANIZE\s+BY|SYSCAT\.`).MatchString(text):
		return "db2"
	}
	return ""
}

// matchParen returns the index of the parenthesis closing the one at open, skipping quotes and comments.
func matchParen(text string, open int) int {
	depth := 0
	for i := open; i < len(text); i++ {
		switch text[i] {
		case '\'':
			for i++; i < len(text) && text[i] != '\''; i++ {
			}
		case '-':
			if i+1 < len(text) && text[i+1] == '-' {
				for ; i < len(text) && text[i] != '\n'; i++ {
				}
			}
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// splitTop splits s on commas outside parentheses and quotes, returning each part's offset.
func splitTop(s string) ([]string, []int) {
	var parts []string
	var offs []int
	depth, start := 0, 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\'':
			for i++; i < len(s) && s[i] != '\''; i++ {
			}
		case '-':
			if i+1 < len(s) && s[i+1] == '-' {
				for ; i < len(s) && s[i] != '\n'; i++ {
				}
			}
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				parts, offs = append(parts, s[start:i]), append(offs, start)
				start = i + 1
			}
		}
	}
	return append(parts, s[start:]), append(offs, start)
}

func colList(s string) []string {
	var out []string
	for _, c := range strings.Split(s, ",") {
		fs := strings.Fields(c)
		if len(fs) == 0 {
			continue
		}
		_, n := splitName(fs[0])
		if n != "" {
			out = append(out, n)
		}
	}
	return out
}

var (
	reLeadComment = regexp.MustCompile(`^(?:\s*--[^\n]*\n|\s*/\*[\s\S]*?\*/)*\s*`)
	reConsName    = regexp.MustCompile("(?i)^constraint\\s+[\\w$#\"`\\[\\]]+\\s+")
	rePK          = regexp.MustCompile(`(?i)^primary\s+key\s*(?:clustered\s+|nonclustered\s+)?\(([^)]*)\)`)
	reFK          = regexp.MustCompile("(?i)^foreign\\s+key\\s*\\(([^)]*)\\)\\s*references\\s+([\\w$#.\"`\\[\\]]+)\\s*(?:\\(([^)]*)\\))?")
	reUnique      = regexp.MustCompile(`(?i)^unique(?:\s+(?:key|index))?(?:\s+[\w$#"` + "`" + `]+)?\s*\(([^)]*)\)`)
	reKeyIdx      = regexp.MustCompile(`(?i)^(?:key|index|fulltext\s+key)\s+[\w$#"` + "`" + `]*\s*\(([^)]*)\)`)
	reCheck       = regexp.MustCompile(`(?i)^check\s*\(`)
	reColDef      = regexp.MustCompile("^([\\w$#\"`\\[\\]]+)\\s+([A-Za-z_][\\w]*(?:\\s+(?:varying|precision|unsigned|zone|time|with|local))*(?:\\s*\\([^)]*\\))?(?:\\s+(?:char|byte))?)")
	reInlineRef   = regexp.MustCompile("(?i)\\breferences\\s+([\\w$#.\"`\\[\\]]+)\\s*(?:\\(([^)]*)\\))?")
	reInlineCheck = regexp.MustCompile(`(?i)\bcheck\s*\(`)
	reNotNull     = regexp.MustCompile(`(?i)\bnot\s+null\b|\bprimary\s+key\b`)
	reInlinePK    = regexp.MustCompile(`(?i)\bprimary\s+key\b`)
)

func (m *DBModel) parseTableBody(f *File, t *DBTable, text string, from, to int) {
	parts, offs := splitTop(text[from:to])
	for i, p := range parts {
		m.parseItem(f, t, p, from+offs[i], false)
	}
}

// parseItem reads one column definition or table constraint. off is the offset of item in the file text.
func (m *DBModel) parseItem(f *File, t *DBTable, item string, off int, constraintOnly bool) {
	lead := reLeadComment.FindString(item)
	off += len(lead)
	it := strings.TrimSpace(item[len(lead):])
	if it == "" {
		return
	}
	line := lineOf(f, off)
	it = reConsName.ReplaceAllString(it, "")
	switch {
	case rePK.MatchString(it):
		t.PK = colList(rePK.FindStringSubmatch(it)[1])
		return
	case reFK.MatchString(it):
		mm := reFK.FindStringSubmatch(it)
		ref := m.table(mm[2])
		t.FKs = append(t.FKs, DBFK{Cols: colList(mm[1]), Ref: ref.Key, RefCols: colList(mm[3]), File: f.Path, Line: line})
		return
	case reUnique.MatchString(it):
		t.Indexes = append(t.Indexes, colList(reUnique.FindStringSubmatch(it)[1]))
		return
	case reCheck.MatchString(it):
		t.Checks = append(t.Checks, DBCheck{Expr: checkExpr(it), File: f.Path, Line: line})
		return
	case reKeyIdx.MatchString(it):
		t.Indexes = append(t.Indexes, colList(reKeyIdx.FindStringSubmatch(it)[1]))
		return
	}
	if constraintOnly {
		return
	}
	mm := reColDef.FindStringSubmatch(it)
	if mm == nil {
		return
	}
	_, name := splitName(mm[1])
	switch name {
	case "CONSTRAINT", "PRIMARY", "FOREIGN", "UNIQUE", "CHECK", "KEY", "INDEX", "PERIOD", "LIKE":
		return
	}
	col := DBColumn{Name: name, Type: strings.ToUpper(strings.Join(strings.Fields(mm[2]), " ")), NotNull: reNotNull.MatchString(it), File: f.Path, Line: line}
	t.Cols = append(t.Cols, col)
	if reInlinePK.MatchString(it) {
		t.PK = []string{name}
	}
	if rm := reInlineRef.FindStringSubmatch(it); rm != nil {
		ref := m.table(rm[1])
		t.FKs = append(t.FKs, DBFK{Cols: []string{name}, Ref: ref.Key, RefCols: colList(rm[2]), File: f.Path, Line: line})
	}
	if loc := reInlineCheck.FindStringIndex(it); loc != nil {
		t.Checks = append(t.Checks, DBCheck{Expr: checkExpr(it[loc[0]:]), File: f.Path, Line: line})
	}
}

func checkExpr(s string) string {
	i := strings.IndexByte(s, '(')
	if i < 0 {
		return s
	}
	j := matchParen(s, i)
	if j < 0 {
		return short(s[i+1:], 200)
	}
	return strings.Join(strings.Fields(s[i+1:j]), " ")
}

var (
	reWrite   = regexp.MustCompile("(?i)\\b(?:insert\\s+(?:all\\s+)?into|update|delete\\s+from|delete|merge\\s+into|truncate\\s+table)\\s+([\\w$#.\"`\\[\\]@]+)")
	reRead    = regexp.MustCompile("(?i)\\b(?:from|join|using)\\s+([\\w$#.\"`\\[\\]@]+)")
	reToken   = regexp.MustCompile(`[A-Za-z_][\w$#]*(?:\.[A-Za-z_][\w$#]*)?`)
	reDynamic = regexp.MustCompile(`(?i)execute\s+immediate|sp_executesql|\bexec(?:ute)?\s*\(\s*@|\bdbms_sql\.|\bprepare\s+\w+\s+from\b|\bexecute\s+format\s*\(`)
	reBranch  = regexp.MustCompile(`(?i)\b(if|elsif|elseif|case|when|loop|while|for|exception)\b`)
	reEndKw   = regexp.MustCompile(`(?i)\bend\s+(if|loop|case)\b`)
	reTrigOn  = regexp.MustCompile("(?i)\\b(?:before|after|instead\\s+of)\\s+(?:insert|update|delete)[\\w\\s,]*?\\s+on\\s+([\\w$#.\"`\\[\\]]+)")
	reTrigOn2 = regexp.MustCompile("(?i)\\bon\\s+([\\w$#.\"`\\[\\]]+)\\s+(?:after|for|instead\\s+of)\\b")
	reComment = regexp.MustCompile(`--[^\n]*`)

	outsideCalls = []struct {
		name string
		re   *regexp.Regexp
	}{
		{"HTTP call (UTL_HTTP)", regexp.MustCompile(`(?i)\butl_http\.`)},
		{"E-mail (UTL_SMTP / UTL_MAIL)", regexp.MustCompile(`(?i)\butl_(?:smtp|mail)\.`)},
		{"File read/write (UTL_FILE)", regexp.MustCompile(`(?i)\butl_file\.`)},
		{"TCP connection (UTL_TCP)", regexp.MustCompile(`(?i)\butl_tcp\.`)},
		{"Queue (Oracle AQ)", regexp.MustCompile(`(?i)\bdbms_aq\.(?:enqueue|dequeue)`)},
		{"Operating-system command (xp_cmdshell)", regexp.MustCompile(`(?i)\bxp_cmdshell\b`)},
		{"COM / OLE automation (sp_OACreate)", regexp.MustCompile(`(?i)\bsp_OA\w+`)},
		{"E-mail (sp_send_dbmail)", regexp.MustCompile(`(?i)\bsp_send_dbmail\b`)},
		{"Remote query (OPENQUERY / OPENROWSET)", regexp.MustCompile(`(?i)\bopen(?:query|rowset|datasource)\s*\(`)},
		{"Java stored code", regexp.MustCompile(`(?i)\blanguage\s+java\b`)},
		{"External program (DBMS_SCHEDULER executable)", regexp.MustCompile(`(?i)job_type\s*=>\s*'EXECUTABLE'`)},
	}
)

func (m *DBModel) analyseCode(f *File, c *DBCode) {
	body := strings.Join(f.Lines[c.Line-1:min(c.End, len(f.Lines))], "\n")
	clean := reComment.ReplaceAllString(body, "")
	reads, writes := map[string]bool{}, map[string]bool{}
	for _, mm := range reWrite.FindAllStringSubmatch(clean, -1) {
		if strings.Contains(mm[1], "@") {
			c.RemoteWrites = appendUnique(c.RemoteWrites, strings.ToUpper(nameCleaner.Replace(mm[1])))
			continue
		}
		if k := m.Find(mm[1]); k != "" {
			writes[k] = true
		}
	}
	for _, mm := range reRead.FindAllStringSubmatch(clean, -1) {
		if k := m.Find(mm[1]); k != "" && !writes[k] {
			reads[k] = true
		}
	}
	c.Reads, c.Writes = sortedKeys(reads), sortedKeys(writes)
	_, self := splitName(c.Name)
	calls := map[string]bool{}
	for _, tok := range reToken.FindAllString(clean, -1) {
		up := strings.ToUpper(tok)
		first := up
		if i := strings.IndexByte(up, '.'); i >= 0 {
			first = up[:i]
			if k, ok := m.codeByBase[up[i+1:]]; ok && first != "SYS" && m.Find(first) == "" {
				_, base := splitName(k)
				if base != self {
					calls[k] = true
				}
				continue
			}
		}
		if k, ok := m.codeByBase[first]; ok && first != self {
			calls[k] = true
		}
	}
	c.Calls = sortedKeys(calls)
	c.Complexity = max(0, len(reBranch.FindAllStringIndex(clean, -1))-len(reEndKw.FindAllStringIndex(clean, -1)))
	for i, l := range f.Lines[c.Line-1 : min(c.End, len(f.Lines))] {
		if c.Dynamic == 0 && reDynamic.MatchString(l) {
			c.Dynamic = c.Line + i
		}
		for _, oc := range outsideCalls {
			if oc.re.MatchString(l) {
				c.Outside = append(c.Outside, DBObj{Name: oc.name, File: f.Path, Line: c.Line + i})
			}
		}
	}
	if c.Type == "TRIGGER" {
		head := strings.Join(f.Lines[c.Line-1:min(c.Line+8, len(f.Lines))], " ")
		if mm := reTrigOn.FindStringSubmatch(head); mm != nil {
			c.OnTable = m.table(mm[1]).Key
		} else if mm := reTrigOn2.FindStringSubmatch(head); mm != nil {
			c.OnTable = m.table(mm[1]).Key
		}
	}
}

func (m *DBModel) parseJobsLinksAccounts(f *File, text string) {
	at := func(off int) int { return lineOf(f, off) }
	window := func(off, n int) string { return text[off:min(len(text), off+n)] }
	for _, loc := range reJobOracle.FindAllStringSubmatchIndex(text, -1) {
		w := window(loc[0], 1200)
		j := DBObj{Name: text[loc[2]:loc[3]], File: f.Path, Line: at(loc[0])}
		if mm := reJobRepeat.FindStringSubmatch(w); mm != nil {
			j.Detail = mm[1]
		}
		if mm := reJobAction.FindStringSubmatch(w); mm != nil {
			j.Extra = mm[1]
		}
		m.Jobs = append(m.Jobs, j)
	}
	for _, loc := range reJobSubmit.FindAllStringSubmatchIndex(text, -1) {
		j := DBObj{Name: "DBMS_JOB: " + short(text[loc[2]:loc[3]], 60), Extra: text[loc[2]:loc[3]], File: f.Path, Line: at(loc[0])}
		if loc[4] >= 0 {
			j.Detail = text[loc[4]:loc[5]]
		}
		m.Jobs = append(m.Jobs, j)
	}
	for _, loc := range reJobMSSQL.FindAllStringSubmatchIndex(text, -1) {
		w := window(loc[0], 3000)
		j := DBObj{Name: text[loc[2]:loc[3]], File: f.Path, Line: at(loc[0])}
		if mm := reJobStepCmd.FindStringSubmatch(w); mm != nil {
			j.Extra = mm[1]
		}
		if mm := reJobSchedule.FindStringSubmatch(w); mm != nil {
			j.Detail = mm[1]
		}
		m.Jobs = append(m.Jobs, j)
	}
	for _, loc := range reJobEvent.FindAllStringSubmatchIndex(text, -1) {
		j := DBObj{Name: nameCleaner.Replace(text[loc[2]:loc[3]]), Detail: strings.TrimSpace(text[loc[4]:loc[5]]), File: f.Path, Line: at(loc[0])}
		if mm := reEventDo.FindStringSubmatch(window(loc[0], 600)); mm != nil {
			j.Extra = mm[1]
		}
		m.Jobs = append(m.Jobs, j)
	}
	for _, loc := range reJobCron.FindAllStringSubmatchIndex(text, -1) {
		m.Jobs = append(m.Jobs, DBObj{Name: text[loc[2]:loc[3]], Detail: text[loc[4]:loc[5]], Extra: text[loc[6]:loc[7]], File: f.Path, Line: at(loc[0])})
	}
	for _, loc := range reLinkOracle.FindAllStringSubmatchIndex(text, -1) {
		l := DBObj{Name: strings.ToUpper(nameCleaner.Replace(text[loc[2]:loc[3]])), File: f.Path, Line: at(loc[0])}
		if mm := reLinkUsing.FindStringSubmatch(window(loc[0], 400)); mm != nil {
			l.Detail = mm[1]
		}
		m.Links = append(m.Links, l)
	}
	for _, loc := range reLinkMSSQL.FindAllStringSubmatchIndex(text, -1) {
		l := DBObj{Name: text[loc[2]:loc[3]], Detail: "linked server", File: f.Path, Line: at(loc[0])}
		if loc[4] >= 0 {
			l.Detail = text[loc[4]:loc[5]]
		}
		m.Links = append(m.Links, l)
	}
	for _, loc := range reLinkPG.FindAllStringSubmatchIndex(text, -1) {
		l := DBObj{Name: nameCleaner.Replace(text[loc[2]:loc[3]]), Detail: text[loc[4]:loc[5]], File: f.Path, Line: at(loc[0])}
		if loc[6] >= 0 {
			l.Detail += " · " + text[loc[6]:loc[7]]
		}
		m.Links = append(m.Links, l)
	}
	addAdmin := func(re *regexp.Regexp, role string) {
		for _, loc := range re.FindAllStringSubmatchIndex(text, -1) {
			m.Admins = append(m.Admins, DBObj{Name: strings.Trim(nameCleaner.Replace(text[loc[2]:loc[3]]), "'"), Detail: role, File: f.Path, Line: at(loc[0])})
		}
	}
	addAdmin(reAdminDBA, "DBA role")
	addAdmin(reAdminSys, "sysadmin")
	addAdmin(reAdminSys2, "sysadmin")
	addAdmin(reAdminMySQL, "ALL PRIVILEGES on *.*")
	addAdmin(reAdminPG, "superuser")
	addAdmin(reAdminAny, "ANY privilege")
	for _, loc := range rePublicGrant.FindAllStringSubmatchIndex(text, -1) {
		m.PublicGrants = append(m.PublicGrants, DBObj{Name: strings.ToUpper(nameCleaner.Replace(text[loc[4]:loc[5]])), Detail: strings.ToUpper(strings.Join(strings.Fields(text[loc[2]:loc[3]]), " ")), File: f.Path, Line: at(loc[0])})
	}
}

// ---------- catalog query results (CSV) ----------

// Record types of the catalog query (01 Record evidence → Database → How to extract).
// Extra types accepted here: column, pk, fk, index, grant, version.
var catalogTypes = map[string]bool{"table": true, "view": true, "code": true, "job": true, "link": true, "admin": true,
	"column": true, "pk": true, "fk": true, "index": true, "grant": true, "version": true}

var reFKDetail = regexp.MustCompile(`^\s*([^-]+?)\s*->\s*([\w$#."]+)\s*(?:\(([^)]*)\))?`)

func (m *DBModel) parseCatalogCSV(r *Run, f *File) bool {
	text := strings.Join(f.Lines, "\n")
	first := f.Lines[0]
	rd := csv.NewReader(strings.NewReader(text))
	rd.LazyQuotes, rd.FieldsPerRecord, rd.TrimLeadingSpace = true, -1, true
	switch {
	case strings.Count(first, "\t") > strings.Count(first, ","):
		rd.Comma = '\t'
	case strings.Count(first, ";") > strings.Count(first, ","):
		rd.Comma = ';'
	case strings.Count(first, "|") > strings.Count(first, ","):
		rd.Comma = '|'
	}
	cols := []string{"record_type", "schema_name", "name", "row_count", "size_mb", "line_count", "detail"}
	idx := func(k string) int {
		for i, c := range cols {
			if c == k {
				return i
			}
		}
		return -1
	}
	get := func(row []string, k string) string {
		i := idx(k)
		if i < 0 || i >= len(row) {
			return ""
		}
		v := strings.TrimSpace(row[i])
		if strings.EqualFold(v, "NULL") {
			return ""
		}
		return v
	}
	num := func(v string) (float64, bool) {
		n, err := strconv.ParseFloat(strings.NewReplacer(",", "", " ", "").Replace(v), 64)
		return n, err == nil && n >= 0
	}
	used := false
	for rowN := 0; ; rowN++ {
		row, err := rd.Read()
		if err != nil {
			break
		}
		line, _ := rd.FieldPos(0)
		if rowN == 0 {
			head := make([]string, len(row))
			for i, h := range row {
				head[i] = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(h, string(rune(0xFEFF)))))
			}
			if contains(head, "record_type") {
				cols = head
				continue
			}
			if !catalogTypes[head[0]] {
				return false
			}
		}
		typ := strings.ToLower(get(row, "record_type"))
		if !catalogTypes[typ] {
			continue
		}
		used = true
		schema, name, detail := get(row, "schema_name"), get(row, "name"), get(row, "detail")
		full := name
		if schema != "" {
			full = schema + "." + name
		}
		switch typ {
		case "table", "view":
			if name == "" {
				continue
			}
			t := m.table(full)
			if t.File == "" {
				t.File, t.Line = f.Path, line
			}
			if typ == "view" || strings.Contains(strings.ToLower(detail), "view") {
				t.View = true
			}
			if v, ok := num(get(row, "row_count")); ok && get(row, "row_count") != "" {
				t.Rows, t.HasStats = v, true
			}
			if v, ok := num(get(row, "size_mb")); ok && get(row, "size_mb") != "" {
				t.SizeMB, t.HasStats = v, true
			}
		case "code":
			s2, n2 := splitName(full)
			typ := strings.ToUpper(detail)
			if typ == "" {
				typ = "CODE"
			}
			key := joinName(s2, n2)
			if typ == "PACKAGE" {
				key += " (spec)"
			}
			if _, ok := m.Code[key]; ok {
				continue // the DDL export has the body; keep it
			}
			lines, _ := num(get(row, "line_count"))
			m.Code[key] = &DBCode{Key: key, Name: joinName(s2, n2), Type: typ, File: f.Path, Line: line, End: line, Lines: int(lines)}
			if _, ok := m.codeByBase[n2]; !ok || typ != "PACKAGE" {
				m.codeByBase[n2] = key
			}
		case "job":
			m.Jobs = append(m.Jobs, DBObj{Name: full, Detail: detail, File: f.Path, Line: line})
		case "link":
			m.Links = append(m.Links, DBObj{Name: full, Detail: detail, File: f.Path, Line: line})
		case "admin":
			if detail == "" {
				detail = "administrator"
			}
			m.Admins = append(m.Admins, DBObj{Name: name, Detail: detail, File: f.Path, Line: line})
		case "column":
			t := m.table(full)
			fs := strings.Fields(detail)
			if len(fs) == 0 {
				continue
			}
			_, cn := splitName(fs[0])
			typ := ""
			if len(fs) > 1 {
				typ = strings.ToUpper(fs[1])
			}
			t.Cols = append(t.Cols, DBColumn{Name: cn, Type: typ, NotNull: strings.Contains(strings.ToUpper(detail), "NOT NULL"), File: f.Path, Line: line})
			if t.File == "" {
				t.File, t.Line = f.Path, line
			}
		case "pk":
			m.table(full).PK = colList(detail)
		case "index":
			t := m.table(full)
			t.Indexes = append(t.Indexes, colList(detail))
		case "fk":
			if mm := reFKDetail.FindStringSubmatch(detail); mm != nil {
				t := m.table(full)
				ref := m.table(mm[2])
				t.FKs = append(t.FKs, DBFK{Cols: colList(mm[1]), Ref: ref.Key, RefCols: colList(mm[3]), File: f.Path, Line: line})
			}
		case "grant":
			if strings.Contains(strings.ToUpper(detail), "PUBLIC") {
				m.PublicGrants = append(m.PublicGrants, DBObj{Name: strings.ToUpper(full), Detail: strings.ToUpper(detail), File: f.Path, Line: line})
			}
		case "version":
			if m.Version == "" {
				m.Version, m.VersionFile, m.VersionLine = detail, f.Path, line
				if e := strings.ToLower(name); e != "" {
					m.Engine = e
				}
			}
		}
	}
	return used
}

// sortedTables returns tables (not views) sorted by key.
func (m *DBModel) sortedTables(includeViews bool) []*DBTable {
	var out []*DBTable
	for _, k := range sortedKeys(m.Tables) {
		t := m.Tables[k]
		if t.View && !includeViews {
			continue
		}
		if t.File == "" { // only referenced, never defined in the export
			continue
		}
		out = append(out, t)
	}
	return out
}

func (m *DBModel) sortedCode() []*DBCode {
	var out []*DBCode
	for _, k := range sortedKeys(m.Code) {
		out = append(out, m.Code[k])
	}
	return out
}

func (t *DBTable) label() string {
	if t.View {
		return "view " + t.Key
	}
	return t.Key
}

func fmtRows(v float64) string {
	switch {
	case v >= 1e9:
		return fmt.Sprintf("%.1f billion", v/1e9)
	case v >= 1e6:
		return fmt.Sprintf("%.1f million", v/1e6)
	case v >= 1e4:
		return fmt.Sprintf("%.0f thousand", v/1e3)
	}
	return fmt.Sprintf("%.0f", v)
}

func fmtMB(mb float64) string {
	if mb >= 1024 {
		return fmt.Sprintf("%.1f GB", mb/1024)
	}
	return fmt.Sprintf("%.0f MB", mb)
}

func sortedByLen(m map[string][]string) []string {
	keys := sortedKeys(m)
	sort.SliceStable(keys, func(i, j int) bool { return len(m[keys[i]]) > len(m[keys[j]]) })
	return keys
}
