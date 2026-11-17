package identity

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

// IDModel is what the identity & access team knows: accounts, groups and their members, MFA registration,
// sign-in policies, applications in the identity provider, and certificates. It is built only from
// read-only directory exports (Get-ADUser / Search-ADAccount / Get-ADGroupMember CSV, Entra ID reports,
// Okta API JSON), openssl output and the collector bundle. Passwords and password hashes are never read.

// Account is a user or service account.
type Account struct {
	Name, UPN, Description       string
	Enabled                      bool
	EnabledKnown                 bool
	LastLogon, PwdLastSet        time.Time
	PwdNeverExpires, MarkedStale bool
	MFAKnown, MFA, IsAdminFlag   bool
	Groups                       []string
	Source, File                 string
	Line                         int
}

// Group is a directory group with its members.
type Group struct {
	Name         string
	Members      []string // account names (users)
	NestedGroups []string
	File         string
	Line         int
}

// Policy is a sign-in (Conditional Access) policy.
type Policy struct {
	Name, State string
	File        string
	Line        int
}

// App is an application registered in the identity provider.
type App struct {
	Name, SignOn, Status string
	File                 string
	Line                 int
}

// Cert is a certificate.
type Cert struct {
	Subject, Issuer, SigAlg string
	NotAfter                time.Time
	KeyBits                 int
	File                    string
	Line                    int
}

// IDModel holds the parsed identity estate.
type IDModel struct {
	Provider, Protocols []string
	ProvFile            string
	ProvLine            int
	Accounts            map[string]*Account
	Groups              map[string]*Group
	Policies            []*Policy
	Apps                []*App
	Certs               []*Cert
	BundleCounts        map[string]int
	BundleMFA           *bool
	Files               []string
	RefDate             time.Time // latest sign-in seen: the "as of" date of the exports
	MFAReport           bool
}

// Summary is shown with the run.
func (m *IDModel) Summary() map[string]any {
	enabled := 0
	for _, a := range m.Accounts {
		if !a.EnabledKnown || a.Enabled {
			enabled++
		}
	}
	return map[string]any{"providers": m.Provider, "accounts": len(m.Accounts), "enabled": enabled, "groups": len(m.Groups), "policies": len(m.Policies),
		"apps": len(m.Apps), "certificates": len(m.Certs), "exportFiles": len(m.Files), "asOf": m.RefDate.Format("2006-01-02")}
}

func key(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

func (m *IDModel) account(name, upn string, f *File, line int) *Account {
	n := strings.TrimSpace(name)
	if n == "" && upn != "" {
		n = strings.SplitN(upn, "@", 2)[0]
	}
	if n == "" {
		return nil
	}
	if a := m.Accounts[key(n)]; a != nil {
		if a.UPN == "" {
			a.UPN = upn
		}
		return a
	}
	if upn != "" { // match an account first seen by user principal name
		if a := m.Accounts[key(strings.SplitN(upn, "@", 2)[0])]; a != nil {
			return a
		}
	}
	a := &Account{Name: n, UPN: upn, File: f.Path, Line: line}
	m.Accounts[key(n)] = a
	return a
}

func (m *IDModel) addProvider(p string, f *File, line int) {
	if p == "" || contains(m.Provider, p) {
		return
	}
	m.Provider = append(m.Provider, p)
	if m.ProvFile == "" {
		m.ProvFile, m.ProvLine = f.Path, line
	}
}

// AccountList returns accounts sorted by name.
func (m *IDModel) AccountList() []*Account {
	var out []*Account
	for _, k := range sortedKeys(m.Accounts) {
		out = append(out, m.Accounts[k])
	}
	return out
}

// ParseIdentity reads every export file of the run.
func ParseIdentity(r *Run) *IDModel {
	m := &IDModel{Accounts: map[string]*Account{}, Groups: map[string]*Group{}, BundleCounts: map[string]int{}}
	for _, f := range r.Code.Files {
		for i, l := range f.Lines {
			if looksSecret(l) {
				r.MarkSecret(f.Path, i+1)
			}
		}
		used := false
		switch f.Lang {
		case "CSV":
			used = m.parseCSV(f)
		case "JSON":
			used = m.parseOkta(f) || m.parseBundle(f)
		case "Text":
			used = m.parseCerts(f)
		}
		if !used && (f.Lang == "Text" || f.Lang == "") {
			used = m.parseCerts(f)
		}
		if used {
			r.Cover("dir", f)
			m.Files = append(m.Files, f.Path)
		}
	}
	for _, a := range m.Accounts {
		if a.LastLogon.After(m.RefDate) {
			m.RefDate = a.LastLogon
		}
	}
	// group membership seen from the account side (MemberOf)
	for _, a := range m.Accounts {
		for _, g := range a.Groups {
			gr := m.group(g, a.File, a.Line)
			gr.Members = appendUnique(gr.Members, a.Name)
		}
	}
	return m
}

func (m *IDModel) group(name, file string, line int) *Group {
	if g := m.Groups[key(name)]; g != nil {
		return g
	}
	g := &Group{Name: name, File: file, Line: line}
	m.Groups[key(name)] = g
	return g
}

// ---------- CSV exports ----------

type table struct {
	head  []string
	rows  [][]string
	lines []int
}

func readTable(f *File) *table {
	if len(f.Lines) == 0 {
		return nil
	}
	first := f.Lines[0]
	if strings.HasPrefix(first, "#TYPE") && len(f.Lines) > 1 {
		first = f.Lines[1]
	}
	rd := csv.NewReader(strings.NewReader(strings.Join(f.Lines, "\n")))
	rd.LazyQuotes, rd.FieldsPerRecord, rd.TrimLeadingSpace = true, -1, true
	if strings.Count(first, ";") > strings.Count(first, ",") {
		rd.Comma = ';'
	} else if strings.Count(first, "\t") > strings.Count(first, ",") {
		rd.Comma = '\t'
	}
	t := &table{}
	for {
		row, err := rd.Read()
		if err != nil {
			break
		}
		line, _ := rd.FieldPos(0)
		if len(row) > 0 && strings.HasPrefix(row[0], "#TYPE") {
			continue
		}
		if t.head == nil {
			for _, h := range row {
				t.head = append(t.head, strings.ToLower(strings.TrimSpace(strings.TrimPrefix(h, string(rune(0xFEFF))))))
			}
			continue
		}
		t.rows = append(t.rows, row)
		t.lines = append(t.lines, line)
	}
	if t.head == nil {
		return nil
	}
	return t
}

func (t *table) col(names ...string) int {
	for _, n := range names {
		for i, h := range t.head {
			if h == n {
				return i
			}
		}
	}
	return -1
}

func val(row []string, i int) string {
	if i < 0 || i >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[i])
}

func truthy(s string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "true", "yes", "1", "enabled", "y", "registered", "capable":
		return true, true
	case "false", "no", "0", "disabled", "n", "notregistered", "not registered":
		return false, true
	}
	return false, false
}

var dateLayouts = []string{"2006-01-02 15:04:05", "2006-01-02T15:04:05Z07:00", "2006-01-02T15:04:05Z", "2006-01-02T15:04:05.000Z", "2006-01-02", "1/2/2006 3:04:05 PM", "1/2/2006 15:04:05", "1/2/2006", "02/01/2006 15:04:05", "Jan 2 15:04:05 2006 MST", "Jan  2 15:04:05 2006 MST", "Jan 2 15:04:05 2006"}

func parseDate(s string) time.Time {
	s = strings.TrimSpace(s)
	for _, l := range dateLayouts {
		if t, err := time.Parse(l, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

func groupFromFile(p string) string {
	b := strings.TrimSuffix(path.Base(p), path.Ext(p))
	b = regexp.MustCompile(`(?i)[-_ ]*(members?|group-?members|export)$`).ReplaceAllString(b, "")
	return strings.Trim(b, "-_ ")
}

func (m *IDModel) parseCSV(f *File) bool {
	t := readTable(f)
	if t == nil {
		return false
	}
	sam := t.col("samaccountname", "sam account name", "username", "user name", "login", "account")
	upn := t.col("userprincipalname", "user principal name", "upn", "mail", "email")
	name := t.col("name", "displayname", "display name")
	switch {
	case t.col("ismfaregistered", "ismfacapable", "mfa registered", "mfastatus", "mfa") >= 0 && upn >= 0:
		return m.parseMFA(f, t, upn)
	case t.col("state") >= 0 && t.col("displayname", "display name", "policy", "policyname") >= 0 && sam < 0 && upn < 0:
		return m.parsePolicies(f, t)
	case t.col("objectclass") >= 0 && (sam >= 0 || name >= 0) && t.col("enabled") < 0:
		return m.parseMembers(f, t, sam, name)
	case sam >= 0 || upn >= 0:
		return m.parseUsers(f, t, sam, upn, name)
	}
	return false
}

func (m *IDModel) parseUsers(f *File, t *table, sam, upn, name int) bool {
	en := t.col("enabled", "accountenabled", "account enabled", "status")
	last := t.col("lastlogondate", "lastlogon", "lastlogontimestamp", "last sign-in", "lastsignin", "lastsignindatetime", "last login", "lastlogin")
	never := t.col("passwordneverexpires", "password never expires", "pwdneverexpires")
	pwdset := t.col("passwordlastset", "pwdlastset", "password last set")
	desc := t.col("description", "title", "jobtitle")
	member := t.col("memberof", "groups", "member of")
	stale := strings.Contains(strings.ToLower(path.Base(f.Path)), "inactive") || strings.Contains(strings.ToLower(path.Base(f.Path)), "stale")
	if strings.Contains(strings.ToLower(path.Base(f.Path)), "-ad") || sam >= 0 {
		m.addProvider("Active Directory", f, 1)
	} else {
		m.addProvider("Directory export", f, 1)
	}
	for i, row := range t.rows {
		a := m.account(val(row, sam), val(row, upn), f, t.lines[i])
		if a == nil {
			a = m.account(val(row, name), "", f, t.lines[i])
		}
		if a == nil {
			continue
		}
		if v := val(row, en); v != "" {
			if b, ok := truthy(v); ok {
				a.Enabled, a.EnabledKnown = b, true
			} else if strings.EqualFold(v, "active") {
				a.Enabled, a.EnabledKnown = true, true
			}
		}
		if d := parseDate(val(row, last)); !d.IsZero() {
			a.LastLogon = d
		}
		if d := parseDate(val(row, pwdset)); !d.IsZero() {
			a.PwdLastSet = d
		}
		if b, ok := truthy(val(row, never)); ok && b {
			a.PwdNeverExpires = true
		}
		if d := val(row, desc); d != "" {
			a.Description = d
		}
		if stale {
			a.MarkedStale = true
		}
		for _, g := range strings.FieldsFunc(val(row, member), func(r rune) bool { return r == ';' || r == '|' }) {
			g = strings.TrimSpace(g)
			if strings.HasPrefix(strings.ToUpper(g), "CN=") { // CN=App-Admins,OU=… -> App-Admins
				g = strings.SplitN(strings.TrimPrefix(g[3:], ""), ",", 2)[0]
			}
			if g != "" {
				a.Groups = appendUnique(a.Groups, g)
			}
		}
	}
	return true
}

func (m *IDModel) parseMembers(f *File, t *table, sam, name int) bool {
	gcol := t.col("group", "groupname", "group name")
	cls := t.col("objectclass")
	m.addProvider("Active Directory", f, 1)
	if gcol < 0 { // one file per group: an export with no rows is an empty group
		m.group(groupFromFile(f.Path), f.Path, 1)
	}
	for i, row := range t.rows {
		gname := val(row, gcol)
		if gname == "" {
			gname = groupFromFile(f.Path)
		}
		g := m.group(gname, f.Path, t.lines[i])
		n := val(row, sam)
		if n == "" {
			n = val(row, name)
		}
		if strings.EqualFold(val(row, cls), "group") {
			g.NestedGroups = appendUnique(g.NestedGroups, n)
			continue
		}
		g.Members = appendUnique(g.Members, n)
		if a := m.account(n, "", f, t.lines[i]); a != nil {
			a.Groups = appendUnique(a.Groups, gname)
		}
	}
	return true
}

func (m *IDModel) parseMFA(f *File, t *table, upn int) bool {
	reg := t.col("ismfaregistered", "mfa registered", "ismfacapable", "mfastatus", "mfa")
	adm := t.col("isadmin", "is admin")
	m.addProvider("Microsoft Entra ID", f, 1)
	m.MFAReport = true
	for i, row := range t.rows {
		a := m.account("", val(row, upn), f, t.lines[i])
		if a == nil {
			continue
		}
		if b, ok := truthy(val(row, reg)); ok {
			a.MFA, a.MFAKnown = b, true
		}
		if b, ok := truthy(val(row, adm)); ok && b {
			a.IsAdminFlag = true
		}
	}
	return true
}

func (m *IDModel) parsePolicies(f *File, t *table) bool {
	n := t.col("displayname", "display name", "policy", "policyname")
	st := t.col("state")
	m.addProvider("Microsoft Entra ID", f, 1)
	for i, row := range t.rows {
		m.Policies = append(m.Policies, &Policy{Name: val(row, n), State: val(row, st), File: f.Path, Line: t.lines[i]})
	}
	return len(t.rows) > 0
}

// ---------- Okta (read-only API JSON) and the collector bundle ----------

func lineOf2(f *File, needle string) int {
	for i, l := range f.Lines {
		if strings.Contains(l, needle) {
			return i + 1
		}
	}
	return 1
}

func (m *IDModel) parseOkta(f *File) bool {
	var items []map[string]any
	if json.Unmarshal([]byte(strings.Join(f.Lines, "\n")), &items) != nil || len(items) == 0 {
		return false
	}
	used := false
	for _, it := range items {
		prof, _ := it["profile"].(map[string]any)
		switch {
		case prof != nil && prof["login"] != nil: // user
			login, _ := prof["login"].(string)
			a := m.account("", login, f, lineOf2(f, login))
			if a == nil {
				continue
			}
			status, _ := it["status"].(string)
			a.Enabled, a.EnabledKnown = status == "ACTIVE" || status == "PASSWORD_EXPIRED" || status == "LOCKED_OUT" || status == "RECOVERY", true
			if s, _ := it["lastLogin"].(string); s != "" {
				a.LastLogon = parseDate(s)
			}
			used = true
		case prof != nil && prof["name"] != nil && it["type"] != nil: // group
			n, _ := prof["name"].(string)
			m.group(n, f.Path, lineOf2(f, `"`+n+`"`))
			used = true
		case it["label"] != nil && it["signOnMode"] != nil: // app
			n, _ := it["label"].(string)
			so, _ := it["signOnMode"].(string)
			stt, _ := it["status"].(string)
			m.Apps = append(m.Apps, &App{Name: n, SignOn: so, Status: stt, File: f.Path, Line: lineOf2(f, `"`+n+`"`)})
			used = true
		}
	}
	if used {
		m.addProvider("Okta", f, 1)
	}
	return used
}

func (m *IDModel) parseBundle(f *File) bool {
	var b struct {
		Identity *struct {
			Provider                                                                   string
			Protocols                                                                  []string
			MfaEnforced                                                                *bool `json:"mfaEnforced"`
			Groups, ServiceAccounts, SharedAccounts, StaleAccounts, PrivilegedAccounts int
		} `json:"identity"`
	}
	if json.Unmarshal([]byte(strings.Join(f.Lines, "\n")), &b) != nil || b.Identity == nil {
		return false
	}
	x := b.Identity
	line := lineOf2(f, `"identity"`)
	for _, p := range strings.Split(x.Provider, "|") {
		m.addProvider(strings.TrimSpace(p), f, line)
	}
	m.Protocols = append(m.Protocols, x.Protocols...)
	m.BundleMFA = x.MfaEnforced
	m.BundleCounts = map[string]int{"groups": x.Groups, "service": x.ServiceAccounts, "shared": x.SharedAccounts, "stale": x.StaleAccounts, "privileged": x.PrivilegedAccounts}
	return true
}

// ---------- certificates (openssl x509 / s_client output) ----------

var (
	certSubjRe   = regexp.MustCompile(`^\s*(?:subject=|Subject:)\s*(.+)$`)
	certIssRe    = regexp.MustCompile(`^\s*(?:issuer=|Issuer:)\s*(.+)$`)
	certAfterRe  = regexp.MustCompile(`^\s*(?:notAfter=|Not After\s*:)\s*(.+)$`)
	certBitsRe   = regexp.MustCompile(`Public-Key:\s*\((\d+) bit\)`)
	certSigAlgRe = regexp.MustCompile(`Signature Algorithm:\s*(\S+)`)
)

func (m *IDModel) parseCerts(f *File) bool {
	var c *Cert
	n := 0
	for i, l := range f.Lines {
		if mm := certSubjRe.FindStringSubmatch(l); mm != nil {
			c = &Cert{Subject: strings.TrimSpace(mm[1]), File: f.Path, Line: i + 1}
			m.Certs = append(m.Certs, c)
			n++
			continue
		}
		if c == nil {
			continue
		}
		if mm := certIssRe.FindStringSubmatch(l); mm != nil {
			c.Issuer = strings.TrimSpace(mm[1])
		}
		if mm := certAfterRe.FindStringSubmatch(l); mm != nil {
			c.NotAfter = parseDate(strings.Join(strings.Fields(mm[1]), " "))
		}
		if mm := certBitsRe.FindStringSubmatch(l); mm != nil {
			c.KeyBits, _ = strconv.Atoi(mm[1])
		}
		if mm := certSigAlgRe.FindStringSubmatch(l); mm != nil && c.SigAlg == "" {
			c.SigAlg = mm[1]
		}
	}
	return n > 0
}

func sortedGroups(m map[string]*Group) []*Group {
	var out []*Group
	for _, k := range sortedKeys(m) {
		out = append(out, m[k])
	}
	sort.SliceStable(out, func(i, j int) bool { return len(out[i].Members) > len(out[j].Members) })
	return out
}
