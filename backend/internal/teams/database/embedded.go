package database

import (
	"path"
	"regexp"
	"strings"
)

// EmbeddedSQL is one application program that runs SQL from its own code.
type EmbeddedSQL struct {
	Program, File string
	Line          int
	Statements    int
	Tables        []string
}

var (
	// db2MarkerRe finds DB2 for z/OS around the database: SQL precompile / bind jobs, DB2 utilities,
	// DCLGEN declarations and the DSN command processor.
	db2MarkerRe  = regexp.MustCompile(`(?i)\bDSN(TIAUL|TEP2|TIAD|TEP4|HC|UPROC|ELI|ECP)\b|\bDSN\s+SYSTEM\s*\(|\bBIND\s+(PLAN|PACKAGE)\b|\bDCLGEN\b|\bDSNDB0|\bIKJEFT0[1B]\b[\s\S]{0,400}\bDSN\b`)
	execSQLRe    = regexp.MustCompile(`(?i)\bEXEC\s+SQL\b`)
	sqlTableUse  = regexp.MustCompile(`(?i)\b(?:FROM|INTO|UPDATE|JOIN|TABLE)\s+([A-Z][\w#$@]*(?:\.[A-Z][\w#$@]*)?)`)
	sqlNotTables = map[string]bool{"SQLCA": true, "SQLDA": true, "DUAL": true, "SYSIBM.SYSDUMMY1": true, ":": true}
)

// noteEmbeddedSQL records the programs with EXEC SQL and the tables their statements name.
func (m *DBModel) noteEmbeddedSQL(f *File) {
	var e *EmbeddedSQL
	in := false
	for i, l := range f.Lines {
		if len(l) > 6 && l[6] == '*' {
			continue
		}
		if execSQLRe.MatchString(l) {
			in = true
			if e == nil {
				e = &EmbeddedSQL{Program: strings.ToUpper(strings.TrimSuffix(path.Base(f.Path), path.Ext(f.Path))), File: f.Path, Line: i + 1}
			}
			e.Statements++
		}
		if in {
			for _, mm := range sqlTableUse.FindAllStringSubmatch(l, -1) {
				t := strings.ToUpper(mm[1])
				if !sqlNotTables[t] && !strings.HasPrefix(t, "WS-") && !contains(e.Tables, t) {
					e.Tables = append(e.Tables, t)
				}
			}
			if strings.Contains(strings.ToUpper(l), "END-EXEC") {
				in = false
			}
		}
	}
	if e != nil {
		m.EmbeddedSQL = append(m.EmbeddedSQL, *e)
	}
}
