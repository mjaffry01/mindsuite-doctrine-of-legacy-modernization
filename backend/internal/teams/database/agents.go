package database

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Database team: rule-based agents. They read only the parsed metadata (st(r).db) and cite export lines.

// dbCite cites a line of an export file; lines that may hold a password are never quoted.
func dbCite(r *Run, file string, line int) Citation {
	f := r.Code.File(file)
	if f == nil || line <= 0 {
		return Citation{File: file, Line: line}
	}
	if r.Secrets[file][line] {
		return Citation{File: file, Line: line}
	}
	return cite(f, line)
}

var engineNames = map[string]string{"oracle": "Oracle", "sqlserver": "SQL Server", "postgresql": "PostgreSQL", "mysql": "MySQL / MariaDB", "db2": "Db2", "": "Unknown engine"}

// ---------- Catalog reader ----------

func dbVersionRisk(engine, v string) (string, string) {
	lv := strings.ToLower(v)
	switch engine {
	case "oracle":
		for _, old := range []string{"8i", "9i", "10g", "11g", "12c", "18c", "8", "9", "10", "11", "12", "18", "21"} {
			if strings.HasPrefix(lv, old) && (len(lv) == len(old) || !strings.ContainsAny(lv[len(old):len(old)+1], "0123456789")) {
				return "Oracle " + v + " is out of Premier Support (19c and 23ai are the supported long-term releases)", "high"
			}
		}
	case "sqlserver":
		for _, old := range []string{"2000", "2005", "2008", "2012", "2014", "2016"} {
			if strings.HasPrefix(lv, old) {
				return "SQL Server " + v + " is out of (or near the end of) extended support", "high"
			}
		}
	case "postgresql":
		var major int
		fmt.Sscanf(lv, "%d", &major)
		if major > 0 && major < 13 {
			return fmt.Sprintf("PostgreSQL %s is past community end of life (major versions are supported for 5 years)", v), "high"
		}
	case "mysql":
		if strings.HasPrefix(lv, "5.") || strings.HasPrefix(lv, "4.") {
			return "MySQL " + v + " is past end of life", "high"
		}
	}
	return "", ""
}

func runCatalog(r *Run) {
	s, m := r.Store, st(r).db
	sum := m.Summary()
	s.Add(Finding{Agent: "cat", Kind: "db-engine", Title: "Database engine: " + engineNames[m.Engine] + map[bool]string{true: " " + m.Version, false: ""}[m.Version != ""],
		Detail: fmt.Sprintf("%v tables, %v views, %v programs (%v lines), %v triggers, %v jobs, %v database links, %d sequences, from %d export file(s).",
			sum["tables"], sum["views"], sum["programs"], sum["codeLines"], sum["triggers"], sum["jobs"], sum["links"], m.Sequences, len(m.Files)),
		Basis: BasisCode, Confidence: map[bool]string{true: "high", false: "medium"}[m.Engine != ""], Data: sum})
	if m.Version != "" {
		risk, sev := dbVersionRisk(m.Engine, m.Version)
		d := map[string]any{"engine": m.Engine, "version": m.Version}
		detail := "Version stated in the export header."
		if risk != "" {
			d["risk"], d["severity"] = risk, sev
			detail = risk + "."
		}
		s.Add(Finding{Agent: "cat", Kind: "db-version", Title: engineNames[m.Engine] + " version " + m.Version, Detail: detail,
			Citations: []Citation{dbCite(r, m.VersionFile, m.VersionLine)}, Confidence: "high", Data: d})
	} else {
		s.Question("cat", "", "Which database version runs in production? The export does not say (add the catalog query result, or the first lines of the export header).")
	}
	for _, sc := range m.Schemas() {
		nt, nc := 0, 0
		for _, t := range m.Tables {
			if schemaLabel(t.Schema) == sc && t.File != "" {
				nt++
			}
		}
		for _, c := range m.Code {
			if s2, _ := splitName(c.Key); schemaLabel(s2) == sc {
				nc++
			}
		}
		if nt+nc > 0 {
			s.Add(Finding{Agent: "cat", Kind: "schema", Module: sc, Title: "Schema " + sc, Detail: fmt.Sprintf("%d tables and views, %d programs", nt, nc), Data: map[string]any{"tables": nt, "programs": nc}})
		}
	}
	if !m.ColumnsKnown && len(m.Tables) > 0 {
		s.Question("cat", "", "The export lists tables but no columns or keys. Add a schema-only DDL export (e.g. Data Pump SQLFILE, SSMS Generate Scripts, pg_dump --schema-only, mysqldump --no-data) so the agents can map relationships.")
	}
	if m.DataInserts > 0 {
		s.Log("cat", fmt.Sprintf("The export contains %d INSERT … VALUES statements with data rows (first in %s). They are not read, quoted or sent anywhere.", m.DataInserts, m.DataInsertFile))
		s.Question("cat", "", fmt.Sprintf("The export %s contains data rows (%d INSERT statements). Please use a schema-only export next time: MindSuite needs metadata only.", m.DataInsertFile, m.DataInserts))
	}
}

// ---------- Schema & relationships ----------

var notNullCheckRe = regexp.MustCompile(`(?i)^\s*"?\w+"?\s+is\s+not\s+null\s*$`)

func runRelationships(r *Run) {
	s, m := r.Store, st(r).db
	tableID := map[string]string{}
	tables := m.sortedTables(true)
	const maxTables = 3000
	for i, t := range tables {
		if i >= maxTables {
			s.Log("rel", fmt.Sprintf("%d more tables are not listed one by one; they are still counted in clusters and checks.", len(tables)-maxTables))
			break
		}
		fks := []string{}
		for _, fk := range t.FKs {
			fks = append(fks, strings.Join(fk.Cols, ",")+" → "+fk.Ref)
		}
		cols := []string{}
		for _, c := range t.Cols {
			cols = append(cols, c.Name)
		}
		detail := fmt.Sprintf("%d columns", len(t.Cols))
		if len(t.PK) > 0 {
			detail += ", key " + strings.Join(t.PK, ", ")
		}
		if len(fks) > 0 {
			detail += ", refers to " + strings.Join(fks, "; ")
		}
		if t.HasStats {
			detail += fmt.Sprintf(", %s rows, %s", fmtRows(t.Rows), fmtMB(t.SizeMB))
		}
		kind := "table"
		if t.View {
			kind = "view"
		}
		tableID[t.Key] = s.Add(Finding{Agent: "rel", Kind: kind, Module: schemaLabel(t.Schema), Title: t.label(), Detail: detail,
			Citations: []Citation{dbCite(r, t.File, t.Line)}, Confidence: "high",
			Data: map[string]any{"table": t.Key, "schema": t.Schema, "columns": cols, "pk": t.PK, "fks": fks, "rows": t.Rows, "sizeMb": t.SizeMB, "hasStats": t.HasStats}})
	}
	st(r).ids = tableID

	// rules written into the schema
	for _, t := range tables {
		for _, c := range t.Checks {
			if notNullCheckRe.MatchString(c.Expr) {
				continue
			}
			s.Add(Finding{Agent: "rel", Kind: "constraint-rule", Module: schemaLabel(t.Schema), Title: t.Name + ": " + short(c.Expr, 120),
				Detail:    "A rule enforced by the database on every write to " + t.Key + ": " + c.Expr,
				Citations: []Citation{dbCite(r, c.File, c.Line)}, Confidence: "high", Data: map[string]any{"table": t.Key, "check": c.Expr}})
		}
	}

	// fan-in and clusters
	refBy := map[string][]string{}
	adj := map[string]map[string]bool{}
	link := func(a, b string) {
		if adj[a] == nil {
			adj[a] = map[string]bool{}
		}
		adj[a][b] = true
	}
	for _, t := range tables {
		for _, fk := range t.FKs {
			if fk.Ref == t.Key {
				continue
			}
			refBy[fk.Ref] = appendUnique(refBy[fk.Ref], t.Key)
			link(t.Key, fk.Ref)
			link(fk.Ref, t.Key)
			ref := m.Tables[fk.Ref]
			if ref != nil && ref.Schema != t.Schema && ref.Schema != "" && t.Schema != "" {
				s.Add(Finding{Agent: "rel", Kind: "cross-schema-fk", Module: schemaLabel(t.Schema), Title: fmt.Sprintf("%s refers to %s in another schema", t.Key, ref.Key),
					Detail:    "A foreign key across schemas ties the two schemas together: they must move together, or the link must be replaced.",
					Citations: []Citation{dbCite(r, fk.File, fk.Line)}, Confidence: "high"})
			}
		}
	}
	for _, k := range sortedByLen(refBy) {
		if len(refBy[k]) < 4 {
			break
		}
		t := m.Tables[k]
		if t == nil || t.File == "" {
			continue
		}
		s.Add(Finding{Agent: "rel", Kind: "hub-table", Module: schemaLabel(t.Schema), Title: fmt.Sprintf("Central table %s: %d tables refer to it", k, len(refBy[k])),
			Detail:    "Referenced by " + strings.Join(firstN(refBy[k], 15), ", ") + ". A core entity: most changes and migrations touch it.",
			Citations: []Citation{dbCite(r, t.File, t.Line)}, Confidence: "high", Data: map[string]any{"table": k, "referencedBy": refBy[k]}})
	}
	seen := map[string]bool{}
	clusters := 0
	for _, start := range sortedKeys(adj) {
		if seen[start] {
			continue
		}
		var comp []string
		queue := []string{start}
		seen[start] = true
		for len(queue) > 0 {
			k := queue[0]
			queue = queue[1:]
			comp = append(comp, k)
			for _, n := range sortedKeys(adj[k]) {
				if !seen[n] {
					seen[n] = true
					queue = append(queue, n)
				}
			}
		}
		if len(comp) < 2 {
			continue
		}
		sort.Strings(comp)
		hub := comp[0]
		for _, k := range comp {
			if len(adj[k]) > len(adj[hub]) {
				hub = k
			}
		}
		ev := []string{}
		for _, k := range comp {
			if id := tableID[k]; id != "" && len(ev) < 40 {
				ev = append(ev, id)
			}
		}
		clusters++
		s.Add(Finding{Agent: "rel", Kind: "data-cluster", Title: fmt.Sprintf("Data cluster around %s (%d tables)", hub, len(comp)),
			Detail:   "Tables joined by foreign keys: " + strings.Join(firstN(comp, 25), ", ") + ". A candidate data domain: these tables usually move together.",
			Evidence: ev, Basis: BasisInferred, Confidence: "medium", Data: map[string]any{"tables": comp, "hub": hub}})
	}
	s.Log("rel", fmt.Sprintf("%d tables and views, %d data clusters, %d tables referenced by 4 or more.", len(tables), clusters, len(refBy)))
}

// ---------- Size & usage ----------

func runVolume(r *Run) {
	s, m := r.Store, st(r).db
	if !m.StatsKnown {
		s.Question("vol", "", "Sizes and row counts are not in the export. Run the read-only catalog query (Record evidence → Database → How to extract) and add its result, so the agents can find the largest and the unused tables.")
		return
	}
	var tabs []*DBTable
	totalRows := 0.0
	for _, t := range m.sortedTables(false) {
		if t.HasStats {
			tabs = append(tabs, t)
			totalRows += t.Rows
		}
	}
	s.Add(Finding{Agent: "vol", Kind: "db-size", Title: fmt.Sprintf("Database size: %.1f GB in %d tables", m.SizeGB(), len(tabs)),
		Detail: fmt.Sprintf("About %s rows in total, from catalog statistics (estimates, as of the export date).", fmtRows(totalRows)),
		Data:   map[string]any{"sizeGb": m.SizeGB(), "rows": totalRows, "tables": len(tabs)}})
	sort.SliceStable(tabs, func(i, j int) bool {
		if tabs[i].SizeMB != tabs[j].SizeMB {
			return tabs[i].SizeMB > tabs[j].SizeMB
		}
		return tabs[i].Rows > tabs[j].Rows
	})
	for i, t := range tabs {
		if i >= 15 || (t.SizeMB == 0 && t.Rows == 0) {
			break
		}
		s.Add(Finding{Agent: "vol", Kind: "large-table", Module: schemaLabel(t.Schema), Title: fmt.Sprintf("%s: %s, %s rows", t.Key, fmtMB(t.SizeMB), fmtRows(t.Rows)),
			Detail:    "One of the largest tables: plan its data move (time, downtime, archiving) early.",
			Citations: []Citation{dbCite(r, t.File, t.Line)}, Confidence: "high", Data: map[string]any{"table": t.Key, "sizeMb": t.SizeMB, "rows": t.Rows}})
	}
	var empty []string
	var cites []Citation
	for _, t := range tabs {
		if t.Rows == 0 {
			empty = append(empty, t.Key)
			if len(cites) < 8 {
				cites = append(cites, dbCite(r, t.File, t.Line))
			}
		}
	}
	if len(empty) > 0 {
		s.Add(Finding{Agent: "vol", Kind: "empty-tables", Title: fmt.Sprintf("%d empty tables", len(empty)),
			Detail:    "No rows according to the statistics: " + strings.Join(firstN(empty, 30), ", ") + ". Candidates to retire instead of migrate (confirm the statistics are current).",
			Citations: cites, Confidence: "medium", Data: map[string]any{"tables": empty}})
	}
}

// ---------- Stored code ----------

func runStoredCode(r *Run) {
	s, m := r.Store, st(r).db
	writers := map[string][]string{}
	writerIDs := map[string][]string{}
	ids := map[string]string{}
	codes := m.sortedCode()
	for _, c := range codes {
		if c.Type == "PACKAGE" && m.Code[strings.TrimSuffix(c.Key, " (spec)")] != nil {
			continue // the body carries the logic
		}
		kind, title := "db-program", fmt.Sprintf("%s %s", strings.ToLower(c.Type), c.Name)
		detail := fmt.Sprintf("%d lines", c.Lines)
		if c.Type == "TRIGGER" {
			kind = "trigger"
			if c.OnTable != "" {
				title += " on " + c.OnTable
			}
		}
		if c.Complexity > 0 {
			detail += fmt.Sprintf(", %d decision points", c.Complexity)
		}
		if len(c.Writes) > 0 {
			detail += ", writes " + strings.Join(firstN(c.Writes, 8), ", ")
		}
		if len(c.Reads) > 0 {
			detail += ", reads " + strings.Join(firstN(c.Reads, 8), ", ")
		}
		if len(c.Calls) > 0 {
			detail += ", calls " + strings.Join(firstN(c.Calls, 8), ", ")
		}
		s2, _ := splitName(c.Name)
		id := s.Add(Finding{Agent: "prc", Kind: kind, Module: schemaLabel(s2), Title: title, Detail: detail, Citations: []Citation{dbCite(r, c.File, c.Line)}, Confidence: "high",
			Data: map[string]any{"program": c.Key, "type": c.Type, "lines": c.Lines, "complexity": c.Complexity, "reads": c.Reads, "writes": c.Writes, "calls": c.Calls,
				"onTable": c.OnTable, "dynamicSql": c.Dynamic > 0, "remoteWrites": c.RemoteWrites}})
		ids[c.Key] = id
		for _, t := range c.Writes {
			writers[t] = append(writers[t], c.Key)
			writerIDs[t] = append(writerIDs[t], id)
		}
		if c.Dynamic > 0 {
			s.Add(Finding{Agent: "prc", Kind: "dynamic-sql", Module: schemaLabel(s2), Title: "Dynamic SQL in " + c.Name,
				Detail:    "SQL built at run time: what it touches cannot be read from the code alone, it is harder to convert to another engine, and it is an injection risk if input is concatenated.",
				Citations: []Citation{dbCite(r, c.File, c.Dynamic)}, Confidence: "high", Data: map[string]any{"program": c.Key}})
		}
		for _, t := range c.RemoteWrites {
			s.Add(Finding{Agent: "prc", Kind: "remote-write", Module: schemaLabel(s2), Title: c.Name + " writes to " + t + " in another database",
				Detail:    "A write over a database link: two databases change in one transaction. Replace with an API or event before moving either one.",
				Citations: []Citation{dbCite(r, c.File, c.Line)}, Confidence: "high"})
		}
	}
	st(r).codeIDs = ids
	for _, t := range sortedByLen(writers) {
		if len(writers[t]) < 2 {
			break
		}
		s.Add(Finding{Agent: "prc", Kind: "shared-write", Title: fmt.Sprintf("%s is written by %d programs", t, len(writers[t])),
			Detail:   "Written by " + strings.Join(firstN(writers[t], 12), ", ") + ". Rules about this table are spread over several programs: extract them together.",
			Evidence: writerIDs[t], Basis: BasisInferred, Confidence: "high", Data: map[string]any{"table": t, "programs": writers[t]}})
	}
	byCx := append([]*DBCode{}, codes...)
	sort.SliceStable(byCx, func(i, j int) bool { return byCx[i].Complexity > byCx[j].Complexity })
	for i, c := range byCx {
		if i >= 10 || c.Complexity < 10 {
			break
		}
		s.Add(Finding{Agent: "prc", Kind: "db-hotspot", Title: fmt.Sprintf("%s: %d decision points in %d lines", c.Name, c.Complexity, c.Lines),
			Detail:    "One of the most complex programs: likely to hold business rules; read it first and test it well before changing it.",
			Citations: []Citation{dbCite(r, c.File, c.Line)}, Confidence: "high", Data: map[string]any{"program": c.Key, "complexity": c.Complexity}})
	}
}

// ---------- Jobs, links & outside calls ----------

func runJobsLinks(r *Run) {
	s, m := r.Store, st(r).db
	for _, j := range m.Jobs {
		runs := ""
		if j.Extra != "" {
			for _, tok := range reToken.FindAllString(strings.ToUpper(j.Extra), -1) {
				base := tok
				if i := strings.LastIndexByte(tok, '.'); i >= 0 {
					if k, ok := m.codeByBase[tok[i+1:]]; ok {
						runs = k
						break
					}
					base = tok[:i]
				}
				if k, ok := m.codeByBase[base]; ok {
					runs = k
					break
				}
			}
		}
		detail := "Schedule: " + map[bool]string{true: j.Detail, false: "not in the export"}[j.Detail != ""]
		if runs != "" {
			detail += "; runs " + runs
		} else if j.Extra != "" {
			detail += "; action: " + short(j.Extra, 120)
		}
		var ev []string
		if id := st(r).codeIDs[runs]; id != "" {
			ev = []string{id}
		}
		s.Add(Finding{Agent: "job", Kind: "db-job", Title: "Scheduled job " + j.Name, Detail: detail, Citations: []Citation{dbCite(r, j.File, j.Line)}, Evidence: ev,
			Confidence: "high", Data: map[string]any{"job": j.Name, "schedule": j.Detail, "runs": runs}})
	}
	for _, l := range m.Links {
		target := l.Detail
		if strings.ContainsAny(target, "@/") && regexp.MustCompile(`(?i)\w+/\w+@`).MatchString(target) {
			target = "(connect string not shown: may contain credentials)"
		}
		s.Add(Finding{Agent: "job", Kind: "db-link", Title: "Database link " + l.Name + map[bool]string{true: " → " + target, false: ""}[target != ""],
			Detail:    "This database reads or writes another database directly. Both sides are coupled at the data level: find who owns the other side and replace the link with an API or event.",
			Citations: []Citation{dbCite(r, l.File, l.Line)}, Confidence: "high", Data: map[string]any{"link": l.Name, "target": target}})
	}
	for _, sy := range m.Synonyms {
		if strings.Contains(sy.Detail, "@") {
			s.Add(Finding{Agent: "job", Kind: "remote-synonym", Title: fmt.Sprintf("Synonym %s points to %s", sy.Name, sy.Detail),
				Detail: "Code that uses this name silently reads another database.", Citations: []Citation{dbCite(r, sy.File, sy.Line)}, Confidence: "high"})
		}
	}
	type agg struct {
		progs []string
		cites []Citation
	}
	out := map[string]*agg{}
	for _, c := range m.sortedCode() {
		for _, o := range c.Outside {
			a := out[o.Name]
			if a == nil {
				a = &agg{}
				out[o.Name] = a
			}
			if !contains(a.progs, c.Name) {
				a.progs = append(a.progs, c.Name)
				if len(a.cites) < 6 {
					a.cites = append(a.cites, dbCite(r, o.File, o.Line))
				}
			}
		}
	}
	for _, k := range sortedKeys(out) {
		a := out[k]
		s.Add(Finding{Agent: "job", Kind: "external-call", Title: fmt.Sprintf("%s from %d program(s)", k, len(a.progs)),
			Detail:    "The database itself talks to the outside world: " + strings.Join(firstN(a.progs, 10), ", ") + ". This interface is easy to miss and must be rebuilt outside the database.",
			Citations: a.cites, Confidence: "high", Data: map[string]any{"feature": k, "programs": a.progs}})
	}
}

// ---------- Security & privacy ----------

var sensitiveCols = []struct {
	cat string
	re  *regexp.Regexp
}{
	{"credential", regexp.MustCompile(`(?i)^(password|passwd|pwd|pass_?word_?hash|user_?password|pin|pin_?code|secret|api_?key|token)$|password`)},
	{"national id", regexp.MustCompile(`(?i)(^|_)(ssn|social_?sec\w*|national_?id\w*|nino|bsn|sin_?no|tax_?id|tin|nif|cpr|aadhaar|passport\w*)(_|$)`)},
	{"payment card", regexp.MustCompile(`(?i)(card_?(no|num|number)|credit_?card|(^|_)(ccn|pan|cvv|cvc)(_|$))`)},
	{"bank account", regexp.MustCompile(`(?i)(iban|bank_?acc|account_?no|routing_?(no|number)|sort_?code|(^|_)(swift|bic)(_|$))`)},
	{"birth date", regexp.MustCompile(`(?i)(birth|(^|_)dob(_|$)|date_?of_?b)`)},
	{"contact", regexp.MustCompile(`(?i)(e_?mail|phone|mobile|\bfax\b|^fax$)`)},
	{"address", regexp.MustCompile(`(?i)(address|street|postcode|post_?code|zip_?code|^zip$)`)},
	{"name", regexp.MustCompile(`(?i)^(first|last|middle|given|family|sur|full)_?name$|^surname$`)},
	{"health", regexp.MustCompile(`(?i)(diagnos|medical|health|allerg|disabilit|blood)`)},
	{"pay", regexp.MustCompile(`(?i)(salary|wage|bonus|payroll)`)},
	{"location", regexp.MustCompile(`(?i)(latitude|longitude|geo_?loc|gps)`)},
}

var hashedName = regexp.MustCompile(`(?i)hash|salt|digest|enc|crypt|bcrypt`)

func runDBSecurity(r *Run) {
	s, m := r.Store, st(r).db
	for _, a := range m.Admins {
		s.Add(Finding{Agent: "dsec", Kind: "admin-account", Title: fmt.Sprintf("Account %s has %s", a.Name, a.Detail),
			Detail:    "Check whether the application connects with this account; it should use a least-privilege account instead.",
			Citations: []Citation{dbCite(r, a.File, a.Line)}, Confidence: "high", Data: map[string]any{"account": a.Name, "role": a.Detail}})
	}
	for _, g := range m.PublicGrants {
		s.Add(Finding{Agent: "dsec", Kind: "public-grant", Title: fmt.Sprintf("%s on %s granted to PUBLIC", g.Detail, g.Name),
			Detail: "Every database account can use this object.", Citations: []Citation{dbCite(r, g.File, g.Line)}, Confidence: "high"})
	}
	pii := 0
	for _, t := range m.sortedTables(false) {
		cats := map[string][]string{}
		var first *DBColumn
		for i, c := range t.Cols {
			for _, sc := range sensitiveCols {
				if sc.re.MatchString(c.Name) {
					if sc.cat == "credential" && hashedName.MatchString(c.Name) {
						break
					}
					if sc.cat != "credential" {
						cats[sc.cat] = append(cats[sc.cat], c.Name)
						if first == nil {
							first = &t.Cols[i]
						}
					}
					if sc.cat == "credential" {
						s.Add(Finding{Agent: "dsec", Kind: "password-column", Module: schemaLabel(t.Schema), Title: fmt.Sprintf("%s.%s (%s) may store passwords", t.Key, c.Name, c.Type),
							Detail:    "Check that values are hashed with a slow password hash (bcrypt, scrypt, Argon2, PBKDF2), never stored as plain or reversible text.",
							Citations: []Citation{dbCite(r, c.File, c.Line)}, Confidence: "medium"})
					}
					break
				}
			}
		}
		if len(cats) == 0 {
			continue
		}
		pii++
		parts := []string{}
		for _, k := range sortedKeys(cats) {
			parts = append(parts, k+": "+strings.Join(cats[k], ", "))
		}
		s.Add(Finding{Agent: "dsec", Kind: "sensitive-column", Module: schemaLabel(t.Schema), Title: t.Key + " holds personal or sensitive data",
			Detail:    "Judged from column names only (no data read): " + strings.Join(parts, "; ") + ". Needs protection, masking in test copies, and a legal basis to move.",
			Citations: []Citation{dbCite(r, first.File, first.Line)}, Basis: BasisInferred, Confidence: "medium", Data: map[string]any{"table": t.Key, "categories": sortedKeys(cats), "columns": cats}})
	}
	// passwords written into the export (file and line only)
	for _, f := range r.Code.Files {
		lines := []int{}
		for n := range r.Secrets[f.Path] {
			lines = append(lines, n)
		}
		sort.Ints(lines)
		for _, n := range lines {
			s.Add(Finding{Agent: "dsec", Kind: "secret", Title: fmt.Sprintf("Possible password in %s line %d", f.Path, n),
				Detail:    "A password or key appears to be written in the export (not shown). Remove it from shared files and rotate it.",
				Citations: []Citation{{File: f.Path, Line: n}}, Confidence: "medium"})
		}
	}
	s.Log("dsec", fmt.Sprintf("%d tables with personal or sensitive columns (by name).", pii))
}

// ---------- Portability ----------

type dbFeature struct {
	engine, name, effort, note string
	re                         *regexp.Regexp
}

var dbFeatures = []dbFeature{
	{"oracle", "CONNECT BY hierarchical queries", "medium", "rewrite as recursive WITH queries", regexp.MustCompile(`(?i)\bconnect\s+by\b`)},
	{"oracle", "ROWNUM", "low", "replace with LIMIT / FETCH FIRST or ROW_NUMBER()", regexp.MustCompile(`(?i)\brownum\b`)},
	{"oracle", "Old-style (+) outer joins", "low", "rewrite as ANSI LEFT/RIGHT JOIN", regexp.MustCompile(`\(\+\)`)},
	{"oracle", "DECODE / NVL", "low", "replace with CASE / COALESCE", regexp.MustCompile(`(?i)\b(decode|nvl2?)\s*\(`)},
	{"oracle", "Autonomous transactions", "high", "no direct equivalent in most engines; redesign (often logging)", regexp.MustCompile(`(?i)pragma\s+autonomous_transaction`)},
	{"oracle", "Bulk processing (BULK COLLECT / FORALL)", "medium", "rewrite as set-based SQL or arrays", regexp.MustCompile(`(?i)\bbulk\s+collect\b|\bforall\b`)},
	{"oracle", "Oracle packages (DBMS_*/UTL_*)", "medium", "find an equivalent or move the logic to the application", regexp.MustCompile(`(?i)\b(dbms|utl)_\w+\.`)},
	{"oracle", "%TYPE / %ROWTYPE anchors", "low", "supported in PostgreSQL PL/pgSQL; elsewhere rewrite", regexp.MustCompile(`(?i)%(row)?type\b`)},
	{"oracle", "Sequences used as .NEXTVAL", "low", "use sequences / identity columns of the target", regexp.MustCompile(`(?i)\.nextval\b`)},
	{"oracle", "Global temporary tables", "low", "temporary tables behave differently per engine", regexp.MustCompile(`(?i)global\s+temporary\s+table`)},
	{"oracle", "Materialized views", "medium", "refresh options differ per engine", regexp.MustCompile(`(?i)materialized\s+view`)},
	{"oracle", "PL/SQL packages", "medium", "split each package into functions or application modules; check package variables, which keep state per session", regexp.MustCompile(`(?i)^\s*create\s+(or\s+replace\s+)?package\s+body`)},
	{"sqlserver", "Cursors", "medium", "rewrite as set-based SQL where possible", regexp.MustCompile(`(?i)\bdeclare\s+\w+\s+cursor\b|\bfetch\s+next\s+from\b`)},
	{"sqlserver", "NOLOCK hints", "low", "remove: dirty reads; check isolation needs", regexp.MustCompile(`(?i)\bnolock\b`)},
	{"sqlserver", "T-SQL temp tables (#)", "low", "temporary tables differ per engine", regexp.MustCompile(`(?i)\b(create\s+table|into)\s+#\w+`)},
	{"sqlserver", "IDENTITY columns", "low", "use identity / sequences of the target", regexp.MustCompile(`(?i)\bidentity\s*\(`)},
	{"sqlserver", "CLR assemblies", "high", "rewrite outside the database", regexp.MustCompile(`(?i)\bcreate\s+assembly\b|external\s+name\b`)},
	{"sqlserver", "TOP / @@ROWCOUNT / @@IDENTITY", "low", "replace with LIMIT / ROW_COUNT / RETURNING", regexp.MustCompile(`(?i)\bselect\s+top\s+\d|@@(rowcount|identity|error)`)},
	{"mysql", "MySQL-specific (AUTO_INCREMENT, ON DUPLICATE KEY)", "low", "use identity / ON CONFLICT", regexp.MustCompile(`(?i)auto_increment|on\s+duplicate\s+key`)},
	{"postgresql", "PostgreSQL extensions", "medium", "check the extension exists on the target (managed services limit them)", regexp.MustCompile(`(?i)create\s+extension`)},
	{"", "Triggers with business logic", "medium", "move rules to the application or keep the trigger", regexp.MustCompile(`(?i)^\s*create\s+(or\s+replace\s+)?(definer\s*=\s*\S+\s+)?trigger\b`)},
	{"", "XML handling in SQL", "medium", "XML functions differ per engine", regexp.MustCompile(`(?i)\b(xmltype|xmlelement|xmlagg|for\s+xml|openxml|\.nodes\s*\()`)},
}

var nonPortableTypes = regexp.MustCompile(`(?i)^(LONG|LONG RAW|RAW|BFILE|XMLTYPE|SDO_GEOMETRY|ROWID|UROWID|ANYDATA|IMAGE|NTEXT|TEXT|MONEY|SMALLMONEY|SQL_VARIANT|HIERARCHYID|GEOGRAPHY|GEOMETRY|ENUM|SET|INTERVAL)\b`)

func runPortability(r *Run) {
	s, m := r.Store, st(r).db
	type agg struct {
		n     int
		cites []Citation
	}
	hits := map[string]*agg{}
	feat := map[string]dbFeature{}
	for _, f := range r.Code.Files {
		if !(f.Lang == "SQL" || f.Lang == "PL/SQL" || f.Lang == "Text") {
			continue
		}
		r.Cover("prt", f)
		for i, l := range f.Lines {
			for _, ft := range dbFeatures {
				if ft.engine != "" && m.Engine != "" && ft.engine != m.Engine {
					continue
				}
				if ft.re.MatchString(l) {
					a := hits[ft.name]
					if a == nil {
						a = &agg{}
						hits[ft.name] = a
						feat[ft.name] = ft
					}
					a.n++
					if len(a.cites) < 5 {
						a.cites = append(a.cites, dbCite(r, f.Path, i+1))
					}
				}
			}
		}
	}
	for _, k := range sortedKeys(hits) {
		a, ft := hits[k], feat[k]
		s.Add(Finding{Agent: "prt", Kind: "vendor-feature", Title: fmt.Sprintf("%s: %d place(s)", ft.name, a.n),
			Detail:    fmt.Sprintf("Engine-specific. To move to another engine: %s. Conversion effort: %s.", ft.note, ft.effort),
			Citations: a.cites, Confidence: "high", Data: map[string]any{"feature": ft.name, "count": a.n, "effort": ft.effort}})
	}
	types := map[string][]string{}
	tcite := map[string][]Citation{}
	for _, t := range m.sortedTables(false) {
		for _, c := range t.Cols {
			if mm := nonPortableTypes.FindStringSubmatch(c.Type); mm != nil {
				k := strings.ToUpper(mm[1])
				types[k] = append(types[k], t.Name+"."+c.Name)
				if len(tcite[k]) < 5 {
					tcite[k] = append(tcite[k], dbCite(r, c.File, c.Line))
				}
			}
		}
	}
	for _, k := range sortedKeys(types) {
		s.Add(Finding{Agent: "prt", Kind: "vendor-type", Title: fmt.Sprintf("Data type %s in %d column(s)", k, len(types[k])),
			Detail:    "Not portable as-is: " + strings.Join(firstN(types[k], 12), ", ") + ". Map it to a target type and convert the data.",
			Citations: tcite[k], Confidence: "high", Data: map[string]any{"type": k, "columns": types[k]}})
	}
}

// ---------- Design quality ----------

var (
	genericColRe = regexp.MustCompile(`(?i)^(attribute|attr|segment|flex|user_?field|udf|custom|col|field|extra)_?\d+$`)
	eavKeyRe     = regexp.MustCompile(`(?i)^(attr(ibute)?_?(name|code|key)?|prop(erty)?_?(name|key)?|key|param(eter)?_?name)$`)
	eavValRe     = regexp.MustCompile(`(?i)^(attr(ibute)?_?value|prop(erty)?_?value|value|val|param(eter)?_?value)$`)
)

func runDesignQuality(r *Run) {
	s, m := r.Store, st(r).db
	tables := m.sortedTables(false)
	if !m.ColumnsKnown {
		s.Log("dq", "No column definitions in the export: design checks skipped.")
		return
	}
	hasIdx := false
	refd := map[string]bool{}
	anyFK := false
	for _, t := range tables {
		if len(t.Indexes) > 0 {
			hasIdx = true
		}
		for _, fk := range t.FKs {
			refd[fk.Ref] = true
			anyFK = true
		}
	}
	var noPK, unindexed, isolated []string
	var noPKc, unIdxC, isoC []Citation
	for _, t := range tables {
		if len(t.Cols) == 0 {
			continue
		}
		if len(t.PK) == 0 {
			noPK = append(noPK, t.Key)
			if len(noPKc) < 8 {
				noPKc = append(noPKc, dbCite(r, t.File, t.Line))
			}
		}
		if hasIdx {
			for _, fk := range t.FKs {
				if !prefixIndexed(t, fk.Cols) {
					unindexed = append(unindexed, t.Key+"("+strings.Join(fk.Cols, ",")+")")
					if len(unIdxC) < 8 {
						unIdxC = append(unIdxC, dbCite(r, fk.File, fk.Line))
					}
				}
			}
		}
		if anyFK && len(t.FKs) == 0 && !refd[t.Key] {
			isolated = append(isolated, t.Key)
			if len(isoC) < 8 {
				isoC = append(isoC, dbCite(r, t.File, t.Line))
			}
		}
		if len(t.Cols) > 60 {
			s.Add(Finding{Agent: "dq", Kind: "wide-table", Module: schemaLabel(t.Schema), Title: fmt.Sprintf("%s has %d columns", t.Key, len(t.Cols)),
				Detail: "Very wide tables often mix several business concepts; split them when the domain is clear.", Citations: []Citation{dbCite(r, t.File, t.Line)}, Confidence: "high"})
		}
		gen := []string{}
		var key, val bool
		for _, c := range t.Cols {
			if genericColRe.MatchString(c.Name) {
				gen = append(gen, c.Name)
			}
			key = key || eavKeyRe.MatchString(c.Name)
			val = val || eavValRe.MatchString(c.Name)
		}
		if len(gen) >= 5 {
			s.Add(Finding{Agent: "dq", Kind: "generic-columns", Module: schemaLabel(t.Schema), Title: fmt.Sprintf("%s has %d generic columns (%s…)", t.Key, len(gen), gen[0]),
				Detail: "Their meaning is configured elsewhere or only known by people: ask what each holds before migrating.", Citations: []Citation{dbCite(r, t.File, t.Line)}, Confidence: "high"})
		}
		if key && val && len(t.Cols) <= 8 {
			s.Add(Finding{Agent: "dq", Kind: "key-value-table", Module: schemaLabel(t.Schema), Title: t.Key + " stores data as name/value pairs",
				Detail: "An entity-attribute-value design: the real structure is hidden in the data. Ask which names are used.", Citations: []Citation{dbCite(r, t.File, t.Line)}, Basis: BasisInferred, Confidence: "medium"})
		}
	}
	add := func(kind, title, detail string, list []string, c []Citation) {
		if len(list) == 0 {
			return
		}
		s.Add(Finding{Agent: "dq", Kind: kind, Title: fmt.Sprintf(title, len(list)), Detail: detail + " " + strings.Join(firstN(list, 25), ", ") + ".", Citations: c, Confidence: "high", Data: map[string]any{"tables": list}})
	}
	add("no-primary-key", "%d tables have no primary key", "Rows cannot be identified reliably; replication and many migration tools need a key:", noPK, noPKc)
	add("unindexed-fk", "%d foreign keys have no index", "Slow joins and lock problems on delete:", unindexed, unIdxC)
	add("isolated-tables", "%d tables have no relationships", "No foreign keys in or out: relationships live in the code, or the tables are unused:", isolated, isoC)
}

func prefixIndexed(t *DBTable, cols []string) bool {
	cand := append([][]string{t.PK}, t.Indexes...)
	for _, ix := range cand {
		if len(ix) < len(cols) {
			continue
		}
		ok := true
		for i, c := range cols {
			if ix[i] != c {
				ok = false
				break
			}
		}
		if ok {
			return true
		}
	}
	return false
}
