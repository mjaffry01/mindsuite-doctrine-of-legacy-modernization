package network

import (
	"encoding/csv"
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// NModel is what the network edge & partners team knows: entry points (load balancers, reverse proxies,
// API gateways, Kubernetes ingress, CDN), DNS entries, TLS certificates, partner connections and firewall
// openings. It is built from configuration and inventory exports (nginx -T, HAProxy config, kubectl JSON,
// DNS zone files or CSV, cloud inventories, openssl output, partner and firewall rule lists). Nothing is
// contacted.

// Entry is an entry point.
type Entry struct {
	Kind, Name    string // load balancer | reverse proxy | API gateway | ingress | CDN
	Listen        []string
	Hosts         []string
	Backends      []string
	TLS, Redirect bool
	Protocols     string // ssl_protocols
	CertFile      string
	File          string
	Line          int
}

// DNSRec is one DNS record.
type DNSRec struct {
	Name, Type, Value string
	TTL               int
	File              string
	Line              int
}

// Cert is a TLS certificate.
type Cert struct {
	Subject, Issuer, SigAlg string
	NotAfter                time.Time
	KeyBits                 int
	File                    string
	Line                    int
}

// Partner is a connection to an outside party.
type Partner struct {
	Name, Host, Protocol, Direction, Owner string
	Port                                   int
	File                                   string
	Line                                   int
}

// Rule is one firewall rule.
type Rule struct {
	Name, Src, Dst, Port, Proto, Action, Desc string
	File                                      string
	Line                                      int
}

// NModel holds the parsed edge.
type NModel struct {
	Entries  []*Entry
	DNS      []*DNSRec
	Certs    []*Cert
	Partners []*Partner
	Rules    []*Rule
	Files    []string
	upstream map[string][]string
}

// Summary is shown with the run.
func (m *NModel) Summary() map[string]any {
	return map[string]any{"entryPoints": len(m.Entries), "dnsRecords": len(m.DNS), "certificates": len(m.Certs), "partners": len(m.Partners), "firewallRules": len(m.Rules), "exportFiles": len(m.Files)}
}

var vpnKeyRe = regexp.MustCompile(`(?i)\b(psk|pre-shared-key|preshared[-_ ]?key|ike[-_ ]?key|shared[-_ ]?secret)\s*[=:]?\s*\S{4,}`)

// ParseNetwork reads every file of the run.
func ParseNetwork(r *Run) *NModel {
	m := &NModel{upstream: map[string][]string{}}
	for _, f := range r.Code.Files {
		for i, l := range f.Lines {
			if looksSecret(l) || vpnKeyRe.MatchString(l) {
				r.MarkSecret(f.Path, i+1)
			}
		}
		head := strings.Join(f.Lines[:min(len(f.Lines), 80)], "\n")
		used := false
		switch {
		case nginxRe.MatchString(head):
			used = m.parseNginx(f)
		case haproxyRe.MatchString(head):
			used = m.parseHAProxy(f)
		case f.Lang == "JSON":
			used = m.parseKube(f)
		case f.Lang == "CSV":
			used = m.parseCSV(f)
		case zoneRe.MatchString(head):
			used = m.parseZone(f)
		}
		if !used && (strings.Contains(head, "subject=") || (strings.Contains(head, "Subject:") && strings.Contains(head, "Not After"))) {
			used = m.parseCerts(f)
		}
		if used {
			r.Cover("nrd", f)
			m.Files = append(m.Files, f.Path)
		}
	}
	// resolve proxy_pass to upstream servers
	for _, e := range m.Entries {
		var out []string
		for _, b := range e.Backends {
			name := strings.TrimPrefix(strings.TrimPrefix(b, "http://"), "https://")
			name = strings.SplitN(name, "/", 2)[0]
			if ups, ok := m.upstream[name]; ok {
				out = append(out, ups...)
			} else {
				out = append(out, name)
			}
		}
		e.Backends = uniq(out)
	}
	return m
}

// ---------- nginx -T ----------

var (
	nginxRe   = regexp.MustCompile(`(?m)^\s*(server|upstream|http)\s*\{|^# configuration file .*nginx|nginx: the configuration file`)
	directive = regexp.MustCompile(`^\s*([a-z_]+)\s*([^;{]*)[;{]`)
	stmtRe    = regexp.MustCompile(`[^{};]*[{};]`)
)

func (m *NModel) parseNginx(f *File) bool {
	depth := 0
	var srv *Entry
	srvDepth := -1
	upName, upDepth := "", -1
	for i, raw := range f.Lines {
		line := strings.TrimSpace(strings.SplitN(raw, "#", 2)[0])
		// one line may hold several statements: "location / { proxy_pass http://x; }"
		for _, l := range stmtRe.FindAllString(line, -1) {
			l = strings.TrimSpace(l)
			if l == "}" {
				depth--
				if srv != nil && depth <= srvDepth {
					srv, srvDepth = nil, -1
				}
				if upName != "" && depth <= upDepth {
					upName, upDepth = "", -1
				}
				continue
			}
			if mm := directive.FindStringSubmatch(l); mm != nil {
				k, v := mm[1], strings.TrimSpace(mm[2])
				switch {
				case k == "server" && strings.HasSuffix(l, "{") && v == "":
					srv = &Entry{Kind: "reverse proxy", Name: "nginx server", File: f.Path, Line: i + 1}
					m.Entries = append(m.Entries, srv)
					srvDepth = depth
				case k == "upstream" && strings.HasSuffix(l, "{"):
					upName, upDepth = v, depth
				case k == "server" && upName != "":
					m.upstream[upName] = append(m.upstream[upName], strings.Fields(v)[0])
				case srv != nil && k == "listen":
					srv.Listen = append(srv.Listen, strings.Fields(v)[0])
					if strings.Contains(v, "ssl") || strings.Contains(v, "443") {
						srv.TLS = true
					}
				case srv != nil && k == "server_name":
					srv.Hosts = append(srv.Hosts, strings.Fields(v)...)
					if len(srv.Hosts) > 0 {
						srv.Name = srv.Hosts[0]
					}
				case srv != nil && k == "ssl_certificate":
					srv.CertFile, srv.TLS = v, true
				case srv != nil && k == "ssl_protocols":
					srv.Protocols = v
				case srv != nil && k == "return" && strings.Contains(v, "https://"):
					srv.Redirect = true
				case srv != nil && (k == "proxy_pass" || k == "fastcgi_pass" || k == "uwsgi_pass" || k == "grpc_pass"):
					srv.Backends = appendUnique(srv.Backends, v)
				}
			}
			if strings.HasSuffix(l, "{") {
				depth++
			}
		}
	}
	return len(m.Entries) > 0 || len(m.upstream) > 0
}

// ---------- HAProxy ----------

var haproxyRe = regexp.MustCompile(`(?m)^\s*(frontend|backend|listen)\s+\S+`)

func (m *NModel) parseHAProxy(f *File) bool {
	var cur *Entry
	first := len(m.Entries)
	global := ""
	backends := map[string][]string{}
	uses := map[*Entry][]string{}
	be := ""
	for i, raw := range f.Lines {
		l := strings.TrimSpace(strings.SplitN(raw, "#", 2)[0])
		fs := strings.Fields(l)
		if len(fs) == 0 {
			continue
		}
		switch fs[0] {
		case "frontend", "listen":
			cur = &Entry{Kind: "load balancer", Name: "haproxy " + fs[1], File: f.Path, Line: i + 1}
			m.Entries = append(m.Entries, cur)
			be = ""
			if fs[0] == "listen" {
				be = fs[1]
				uses[cur] = append(uses[cur], fs[1])
			}
		case "backend":
			cur, be = nil, fs[1]
		case "bind":
			if cur != nil && len(fs) > 1 {
				cur.Listen = append(cur.Listen, fs[1])
				if strings.Contains(l, " ssl") {
					cur.TLS = true
				}
				if i := strings.Index(l, " crt "); i >= 0 {
					cur.CertFile = strings.Fields(l[i+5:])[0]
				}
			}
		case "default_backend", "use_backend":
			if cur != nil && len(fs) > 1 {
				uses[cur] = append(uses[cur], fs[1])
			}
		case "server":
			if be != "" && len(fs) > 2 {
				backends[be] = append(backends[be], fs[2])
			}
		case "http-request", "redirect":
			if cur != nil && strings.Contains(l, "https") {
				cur.Redirect = true
			}
		case "ssl-default-bind-options":
			global = strings.Join(fs[1:], " ")
		}
	}
	for _, e := range m.Entries[first:] { // the global setting applies to this file's frontends only
		if e.Protocols == "" {
			e.Protocols = global
		}
	}
	for e, bs := range uses {
		for _, b := range bs {
			e.Backends = append(e.Backends, backends[b]...)
		}
	}
	return len(m.Entries) > 0
}

// ---------- Kubernetes ingress and services (kubectl JSON) ----------

func dig(v any, keys ...string) any {
	for _, k := range keys {
		mm, ok := v.(map[string]any)
		if !ok {
			return nil
		}
		v = mm[k]
	}
	return v
}

func lineOf2(f *File, needle string) int {
	for i, l := range f.Lines {
		if strings.Contains(l, needle) {
			return i + 1
		}
	}
	return 1
}

func (m *NModel) parseKube(f *File) bool {
	var doc map[string]any
	if json.Unmarshal([]byte(strings.Join(f.Lines, "\n")), &doc) != nil {
		return false
	}
	items, _ := doc["items"].([]any)
	if items == nil && doc["kind"] != nil {
		items = []any{doc}
	}
	n := 0
	for _, it := range items {
		kind, _ := dig(it, "kind").(string)
		name, _ := dig(it, "metadata", "name").(string)
		ns, _ := dig(it, "metadata", "namespace").(string)
		switch kind {
		case "Ingress":
			e := &Entry{Kind: "ingress", Name: ns + "/" + name, File: f.Path, Line: lineOf2(f, `"`+name+`"`)}
			if rules, ok := dig(it, "spec", "rules").([]any); ok {
				for _, ru := range rules {
					if h, _ := dig(ru, "host").(string); h != "" {
						e.Hosts = append(e.Hosts, h)
					}
					if paths, ok := dig(ru, "http", "paths").([]any); ok {
						for _, p := range paths {
							if s, _ := dig(p, "backend", "service", "name").(string); s != "" {
								e.Backends = appendUnique(e.Backends, "service "+s)
							}
						}
					}
				}
			}
			if tls, ok := dig(it, "spec", "tls").([]any); ok && len(tls) > 0 {
				e.TLS = true
			}
			m.Entries = append(m.Entries, e)
			n++
		case "Service":
			if t, _ := dig(it, "spec", "type").(string); t == "LoadBalancer" {
				m.Entries = append(m.Entries, &Entry{Kind: "load balancer", Name: ns + "/" + name + " (Kubernetes LoadBalancer)", File: f.Path, Line: lineOf2(f, `"`+name+`"`)})
				n++
			}
		}
	}
	return n > 0
}

// ---------- CSV: DNS records, partner connections, firewall rules, cloud inventory ----------

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

var cloudEdgeRe = regexp.MustCompile(`(?i)loadbalancer|applicationgateway|frontdoor|cdn|cloudfront|apimanagement|apigateway|elasticloadbalancing|trafficmanager|elb`)

func (m *NModel) parseCSV(f *File) bool {
	head, rows, lines := readTable(f)
	if head == nil {
		return false
	}
	switch {
	case colOf(head, "type", "record type", "recordtype") >= 0 && colOf(head, "value", "data", "target", "content") >= 0 && colOf(head, "name", "hostname", "record") >= 0:
		nc, tc, vc, ttl := colOf(head, "name", "hostname", "record"), colOf(head, "type", "record type", "recordtype"), colOf(head, "value", "data", "target", "content"), colOf(head, "ttl")
		for i, row := range rows {
			t, _ := strconv.Atoi(cellOf(row, ttl))
			m.DNS = append(m.DNS, &DNSRec{Name: cellOf(row, nc), Type: strings.ToUpper(cellOf(row, tc)), Value: cellOf(row, vc), TTL: t, File: f.Path, Line: lines[i]})
		}
		return len(rows) > 0
	case colOf(head, "action", "permission", "decision") >= 0 && colOf(head, "source", "src", "source address", "from") >= 0:
		nc, sc, dc := colOf(head, "name", "rule", "rule name", "id"), colOf(head, "source", "src", "source address", "from"), colOf(head, "destination", "dst", "destination address", "to")
		pc, prc, ac, dsc := colOf(head, "port", "ports", "service", "destination port", "dport"), colOf(head, "protocol", "proto"), colOf(head, "action", "permission", "decision"), colOf(head, "description", "comment", "remark", "owner")
		for i, row := range rows {
			m.Rules = append(m.Rules, &Rule{Name: cellOf(row, nc), Src: cellOf(row, sc), Dst: cellOf(row, dc), Port: cellOf(row, pc), Proto: cellOf(row, prc), Action: strings.ToLower(cellOf(row, ac)), Desc: cellOf(row, dsc), File: f.Path, Line: lines[i]})
		}
		return len(rows) > 0
	case colOf(head, "partner", "counterparty", "organisation", "organization") >= 0 || (colOf(head, "protocol") >= 0 && colOf(head, "host", "address", "endpoint") >= 0):
		nc, hc, pc := colOf(head, "partner", "counterparty", "organisation", "organization", "name"), colOf(head, "host", "address", "endpoint", "ip"), colOf(head, "port")
		prc, dc, oc := colOf(head, "protocol", "type"), colOf(head, "direction"), colOf(head, "owner", "contact", "responsible")
		for i, row := range rows {
			p, _ := strconv.Atoi(cellOf(row, pc))
			m.Partners = append(m.Partners, &Partner{Name: cellOf(row, nc), Host: cellOf(row, hc), Port: p, Protocol: strings.ToUpper(cellOf(row, prc)), Direction: strings.ToLower(cellOf(row, dc)), Owner: cellOf(row, oc), File: f.Path, Line: lines[i]})
		}
		return len(rows) > 0
	case colOf(head, "type", "resource type", "resourcetype") >= 0:
		tc, nc := colOf(head, "type", "resource type", "resourcetype"), colOf(head, "name", "resource name", "identifier")
		n := 0
		for i, row := range rows {
			t := cellOf(row, tc)
			if !cloudEdgeRe.MatchString(t) {
				continue
			}
			kind := "load balancer"
			switch {
			case regexp.MustCompile(`(?i)cdn|frontdoor|cloudfront`).MatchString(t):
				kind = "CDN"
			case regexp.MustCompile(`(?i)apimanagement|apigateway`).MatchString(t):
				kind = "API gateway"
			}
			m.Entries = append(m.Entries, &Entry{Kind: kind, Name: cellOf(row, nc) + " (" + t + ")", File: f.Path, Line: lines[i]})
			n++
		}
		return n > 0
	}
	return false
}

// ---------- BIND zone files ----------

var (
	zoneRe   = regexp.MustCompile(`(?m)^\$ORIGIN|^\$TTL|\bIN\s+(SOA|A|CNAME|MX)\b`)
	zoneLine = regexp.MustCompile(`^(\S+)?\s+(?:(\d+)\s+)?IN\s+(A|AAAA|CNAME|MX|TXT|SRV|NS)\s+(.+)$`)
)

func (m *NModel) parseZone(f *File) bool {
	origin, defTTL, last := "", 0, ""
	n := 0
	for i, l := range f.Lines {
		t := strings.TrimSpace(strings.SplitN(l, ";", 2)[0])
		switch {
		case strings.HasPrefix(t, "$ORIGIN"):
			origin = strings.TrimSuffix(strings.Fields(t)[1], ".")
			continue
		case strings.HasPrefix(t, "$TTL"):
			defTTL, _ = strconv.Atoi(strings.Fields(t)[1])
			continue
		}
		mm := zoneLine.FindStringSubmatch(l)
		if mm == nil {
			continue
		}
		name := mm[1]
		if name == "" {
			name = last
		}
		last = name
		full := name
		switch {
		case name == "@":
			full = origin
		case !strings.HasSuffix(name, "."):
			full = name + map[bool]string{true: "." + origin, false: ""}[origin != ""]
		}
		ttl, _ := strconv.Atoi(mm[2])
		if ttl == 0 {
			ttl = defTTL
		}
		m.DNS = append(m.DNS, &DNSRec{Name: strings.TrimSuffix(full, "."), Type: mm[3], Value: strings.TrimSuffix(strings.TrimSpace(mm[4]), "."), TTL: ttl, File: f.Path, Line: i + 1})
		n++
	}
	return n > 0
}

// ---------- certificates (openssl output) ----------

var (
	certSubjRe   = regexp.MustCompile(`^\s*(?:subject=|Subject:)\s*(.+)$`)
	certIssRe    = regexp.MustCompile(`^\s*(?:issuer=|Issuer:)\s*(.+)$`)
	certAfterRe  = regexp.MustCompile(`^\s*(?:notAfter=|Not After\s*:)\s*(.+)$`)
	certBitsRe   = regexp.MustCompile(`Public-Key:\s*\((\d+) bit\)`)
	certSigAlgRe = regexp.MustCompile(`Signature Algorithm:\s*(\S+)`)
)

func (m *NModel) parseCerts(f *File) bool {
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
			for _, lay := range []string{"Jan 2 15:04:05 2006 MST", "Jan 2 15:04:05 2006"} {
				if t, err := time.Parse(lay, strings.Join(strings.Fields(mm[1]), " ")); err == nil {
					c.NotAfter = t
					break
				}
			}
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
