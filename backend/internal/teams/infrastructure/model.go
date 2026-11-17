package infrastructure

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// InfraModel is what the infrastructure team knows about where the system runs. It is built only from
// inventory exports: VM lists (RVTools, PowerCLI, Hyper-V), kubectl JSON, cloud inventories,
// connection lists (ss, netstat, Get-NetTCPConnection) and the collector bundle. Nothing connects to a server.

// Host is a server or VM.
type Host struct {
	Name, OS, Power, Platform, Cluster, Env string
	CPU                                     int
	RAMGB, DiskGB                           float64
	Software, IPs                           []string
	Role                                    string // from the bundle, else a guess from the name
	RoleGuessed                             bool
	File                                    string
	Line                                    int
}

// Conn is a network connection between two endpoints.
type Conn struct {
	From, To string // host names when known, else addresses
	Port     int
	Count    int
	File     string
	Line     int
}

// Workload is a Kubernetes object.
type Workload struct {
	Kind, Name, Namespace string
	Images                []string
	Replicas              int
	NoLimits, Privileged  bool
	Schedule, ServiceType string
	Hosts                 []string
	File                  string
	Line                  int
}

// CloudRes is one cloud resource.
type CloudRes struct {
	Name, Type, Region, Group, Provider string
	File                                string
	Line                                int
}

// InfraModel holds the parsed estate.
type InfraModel struct {
	Hosts                 map[string]*Host
	byIP                  map[string]string
	Conns                 []*Conn
	Workloads             []*Workload
	Cloud                 []*CloudRes
	Hypervisor            string
	HVFile                string
	HVLine                int
	Files                 []string
	connSeen              map[string]*Conn
	InventoryKnown, Links bool
}

// HostList returns hosts sorted by name.
func (m *InfraModel) HostList() []*Host {
	var out []*Host
	for _, k := range sortedKeys(m.Hosts) {
		out = append(out, m.Hosts[k])
	}
	return out
}

// Summary is shown with the run.
func (m *InfraModel) Summary() map[string]any {
	on, cpu, ram := 0, 0, 0.0
	plat := map[string]int{}
	for _, h := range m.Hosts {
		if h.On() {
			on++
			cpu += h.CPU
			ram += h.RAMGB
		}
		if h.Platform != "" {
			plat[h.Platform]++
		}
	}
	return map[string]any{"hosts": len(m.Hosts), "poweredOn": on, "vcpu": cpu, "ramGb": round1(ram), "connections": len(m.Conns),
		"workloads": len(m.Workloads), "cloudResources": len(m.Cloud), "platforms": plat, "hypervisor": m.Hypervisor, "exportFiles": len(m.Files)}
}

// On reports whether the host runs (unknown counts as running).
func (h *Host) On() bool {
	p := strings.ToLower(h.Power)
	return !(strings.Contains(p, "off") || strings.Contains(p, "stopped") || strings.Contains(p, "saved") || strings.Contains(p, "suspend"))
}

func round1(v float64) float64 { return float64(int(v*10+0.5)) / 10 }

func hostKey(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if i := strings.IndexByte(s, '.'); i > 0 && !ipRe.MatchString(s) {
		s = s[:i] // short name: app-prod-01.corp.local → app-prod-01
	}
	return s
}

var ipRe = regexp.MustCompile(`^\d{1,3}(\.\d{1,3}){3}$`)

func (m *InfraModel) host(name string) *Host {
	k := hostKey(name)
	if k == "" {
		return nil
	}
	if h := m.Hosts[k]; h != nil {
		return h
	}
	h := &Host{Name: strings.TrimSpace(name)}
	if i := strings.IndexByte(h.Name, '.'); i > 0 && !ipRe.MatchString(h.Name) {
		h.Name = h.Name[:i]
	}
	m.Hosts[k] = h
	return h
}

// Resolve turns an address or name into a known host name ("" when outside the inventory).
func (m *InfraModel) Resolve(addr string) string {
	a := strings.Trim(strings.ToLower(addr), "[]")
	if a == "" {
		return ""
	}
	if n, ok := m.byIP[a]; ok {
		return n
	}
	if h := m.Hosts[hostKey(a)]; h != nil {
		return h.Name
	}
	return ""
}

var (
	envRe = []struct {
		env string
		re  *regexp.Regexp
	}{
		{"production", regexp.MustCompile(`(?i)(^|[-_.])(prod|prd|live|p\d{0,2})([-_.\d]|$)`)},
		{"test", regexp.MustCompile(`(?i)(^|[-_.])(test|tst|qa|uat|acc|t\d{0,2})([-_.\d]|$)`)},
		{"development", regexp.MustCompile(`(?i)(^|[-_.])(dev|d\d{0,2}|sandbox)([-_.\d]|$)`)},
		{"staging", regexp.MustCompile(`(?i)(^|[-_.])(stg|stage|staging|preprod|pp)([-_.\d]|$)`)},
	}
	roleRe = []struct {
		role string
		re   *regexp.Regexp
	}{
		{"database", regexp.MustCompile(`(?i)(^|[-_.])(db|sql|ora|oracle|pg|postgres|mysql|maria|mongo|dbs)\d*([-_.]|$)|oracle database|sql server|postgresql|mysql|db2`)},
		{"web", regexp.MustCompile(`(?i)(^|[-_.])(web|www|iis|nginx|apache|httpd|proxy|lb)\d*([-_.]|$)|\bnginx\b|apache http|\biis\b`)},
		{"application", regexp.MustCompile(`(?i)(^|[-_.])(app|as|tomcat|jboss|wildfly|weblogic|websphere|api)\d*([-_.]|$)|tomcat|jboss|wildfly|weblogic|websphere`)},
		{"batch", regexp.MustCompile(`(?i)(^|[-_.])(batch|job|jobs|sched|etl)\d*([-_.]|$)|control-m|autosys`)},
		{"messaging", regexp.MustCompile(`(?i)(^|[-_.])(mq|kafka|rabbit|esb|broker)\d*([-_.]|$)|ibm mq|kafka|rabbitmq`)},
		{"cache", regexp.MustCompile(`(?i)(^|[-_.])(redis|cache|memcache)\d*([-_.]|$)|redis|memcached`)},
		{"file transfer", regexp.MustCompile(`(?i)(^|[-_.])(sftp|ftp|mft|file)\d*([-_.]|$)`)},
		{"monitoring", regexp.MustCompile(`(?i)(^|[-_.])(mon|monitor|nagios|zabbix|grafana|prom|splunk|elk|log)\d*([-_.]|$)`)},
		{"directory", regexp.MustCompile(`(?i)(^|[-_.])(dc|ad|ldap|adfs)\d*([-_.]|$)`)},
	}
)

func guessEnv(name string) string {
	for _, e := range envRe {
		if e.re.MatchString(name) {
			return e.env
		}
	}
	return ""
}

func guessRole(h *Host) string {
	text := h.Name + " " + strings.Join(h.Software, " ")
	for _, r := range roleRe {
		if r.re.MatchString(text) {
			return r.role
		}
	}
	return ""
}

// well-known server ports
var portService = map[int]string{20: "FTP data", 21: "FTP", 22: "SSH / SFTP", 23: "Telnet", 25: "SMTP mail", 53: "DNS", 80: "HTTP", 88: "Kerberos", 110: "POP3", 111: "RPC", 135: "Windows RPC", 139: "NetBIOS", 143: "IMAP",
	389: "LDAP", 443: "HTTPS", 445: "SMB file share", 465: "SMTPS", 514: "Syslog", 587: "SMTP submission", 636: "LDAPS", 873: "rsync", 990: "FTPS", 1414: "IBM MQ", 1433: "SQL Server", 1521: "Oracle Database",
	2049: "NFS", 2181: "ZooKeeper", 3268: "Global catalog (AD)", 3306: "MySQL", 3389: "Remote Desktop", 4369: "RabbitMQ (epmd)", 5432: "PostgreSQL", 5671: "AMQPS", 5672: "RabbitMQ / AMQP", 5985: "WinRM", 5986: "WinRM (TLS)", 6379: "Redis",
	7001: "WebLogic", 8009: "AJP (Tomcat)", 8080: "HTTP (app server)", 8443: "HTTPS (app server)", 9042: "Cassandra", 9092: "Kafka", 9200: "Elasticsearch", 9300: "Elasticsearch", 11211: "Memcached", 27017: "MongoDB", 50000: "Db2"}

func serviceName(port int) string {
	if s, ok := portService[port]; ok {
		return s
	}
	return fmt.Sprintf("port %d", port)
}

func serverPort(p int) bool { _, ok := portService[p]; return ok || p < 1024 }

// ParseInfra reads every export file of the run.
func ParseInfra(r *Run) *InfraModel {
	m := &InfraModel{Hosts: map[string]*Host{}, byIP: map[string]string{}, connSeen: map[string]*Conn{}}
	var connFiles []*File
	// inventories first, so connections can be resolved to host names
	for _, f := range r.Code.Files {
		for i, l := range f.Lines {
			if looksSecret(l) {
				r.MarkSecret(f.Path, i+1)
			}
		}
		used := false
		switch f.Lang {
		case "CSV":
			if m.parseVMCSV(f) || m.parseCloudCSV(f) {
				used = true
			} else if isConnCSV(f) {
				connFiles = append(connFiles, f)
				continue
			}
		case "JSON":
			used = m.parseKubectl(f) || m.parseBundle(f) || m.parseCloudJSON(f)
		case "Text":
			if connLineRe.MatchString(strings.Join(f.Lines[:min(len(f.Lines), 40)], "\n")) {
				connFiles = append(connFiles, f)
				continue
			}
		}
		if used {
			r.Cover("est", f)
			m.Files = append(m.Files, f.Path)
		}
	}
	for _, h := range m.Hosts {
		for _, ip := range h.IPs {
			m.byIP[strings.ToLower(ip)] = h.Name
		}
	}
	for _, f := range connFiles {
		if f.Lang == "CSV" {
			m.parseConnCSV(f)
		} else {
			m.parseConnText(f)
		}
		r.Cover("est", f)
		m.Files = append(m.Files, f.Path)
	}
	for _, h := range m.Hosts {
		if h.Env == "" {
			h.Env = guessEnv(h.Name)
		}
		if h.Role == "" {
			h.Role, h.RoleGuessed = guessRole(h), true
		}
	}
	m.InventoryKnown = len(m.Hosts) > 0
	m.Links = len(m.Conns) > 0
	return m
}

// ---------- VM and server inventories (RVTools vInfo, PowerCLI Get-VM, Hyper-V Get-VM, generic server lists) ----------

type csvTable struct {
	head  []string
	rows  [][]string
	lines []int
}

func readCSV(f *File) *csvTable {
	text := strings.Join(f.Lines, "\n")
	if len(f.Lines) == 0 {
		return nil
	}
	first := f.Lines[0]
	if strings.HasPrefix(first, "#TYPE") { // Export-Csv without -NoTypeInformation
		first = f.Lines[min(1, len(f.Lines)-1)]
	}
	rd := csv.NewReader(strings.NewReader(text))
	rd.LazyQuotes, rd.FieldsPerRecord, rd.TrimLeadingSpace = true, -1, true
	switch {
	case strings.Count(first, "\t") > strings.Count(first, ","):
		rd.Comma = '\t'
	case strings.Count(first, ";") > strings.Count(first, ","):
		rd.Comma = ';'
	}
	t := &csvTable{}
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

func (t *csvTable) col(names ...string) int {
	for _, n := range names {
		for i, h := range t.head {
			if h == n {
				return i
			}
		}
	}
	return -1
}

func (t *csvTable) colLike(subs ...string) int {
	for _, s := range subs {
		for i, h := range t.head {
			if strings.Contains(h, s) {
				return i
			}
		}
	}
	return -1
}

func cell(row []string, i int) string {
	if i < 0 || i >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[i])
}

func toF(s string) float64 {
	v, _ := strconv.ParseFloat(strings.NewReplacer(",", "", " ", "").Replace(s), 64)
	return v
}

func (m *InfraModel) parseVMCSV(f *File) bool {
	t := readCSV(f)
	if t == nil {
		return false
	}
	name := t.col("vm", "name", "vmname", "server", "hostname", "computername", "host name")
	cpu := t.col("cpus", "numcpu", "processorcount", "vcpu", "vcpus", "cpu", "cpu count", "cores")
	if name < 0 || cpu < 0 {
		return false
	}
	platform := "Servers"
	switch {
	case t.colLike("os according to") >= 0:
		platform = "VMware vSphere"
	case t.col("numcpu") >= 0:
		platform = "VMware vSphere"
	case t.col("processorcount") >= 0:
		platform = "Hyper-V"
	}
	mem, memUnit := -1, 1.0
	for _, c := range []struct {
		name string
		unit float64
	}{{"memorygb", 1}, {"memory gb", 1}, {"ram gb", 1}, {"ram (gb)", 1}, {"ramgb", 1}, {"memory (gb)", 1}, {"memorymb", 1.0 / 1024}, {"memory mb", 1.0 / 1024}, {"memory mib", 1.0 / 1024},
		{"memory", 0}, {"memoryassigned", 1.0 / (1 << 30)}, {"memorystartup", 1.0 / (1 << 30)}, {"ram", 0}} {
		if i := t.col(c.name); i >= 0 {
			mem, memUnit = i, c.unit
			break
		}
	}
	os := t.col("os according to the vmware tools", "os according to the configuration file", "guest", "guestos", "guest os", "os", "operatingsystem", "operating system", "os name")
	power := t.col("powerstate", "power state", "state", "status")
	host := t.col("host", "vmhost", "esx host", "computername")
	cluster := t.col("cluster", "datacenter")
	disk := t.col("provisioned mib", "provisioned mb", "disk gb", "provisionedspacegb", "disk (gb)", "storage gb")
	ip := t.col("primary ip address", "ipaddress", "ip address", "ip", "ipv4")
	sw := t.col("software", "installed software", "middleware")
	env := t.col("environment", "env")
	role := t.col("role", "function", "purpose")
	sdk, sdkVer := t.col("vi sdk server type", "vi sdk server"), t.col("vi sdk api version")
	cfgOS := t.col("os according to the configuration file")
	used := false
	for i, row := range t.rows {
		n := cell(row, name)
		if n == "" {
			continue
		}
		h := m.host(n)
		used = true
		h.File, h.Line, h.Platform = f.Path, t.lines[i], platform
		h.CPU = int(toF(cell(row, cpu)))
		if mem >= 0 {
			v := toF(cell(row, mem))
			u := memUnit
			if u == 0 { // unitless "Memory": RVTools gives MiB
				u = 1
				if v > 512 {
					u = 1.0 / 1024
				}
			}
			h.RAMGB = round1(v * u)
		}
		h.OS = cell(row, os)
		if h.OS == "" {
			h.OS = cell(row, cfgOS)
		}
		h.Power = cell(row, power)
		if c := cell(row, cluster); c != "" {
			h.Cluster = c
		} else {
			h.Cluster = cell(row, host)
		}
		if disk >= 0 {
			v := toF(cell(row, disk))
			if strings.Contains(t.head[disk], "mib") || strings.Contains(t.head[disk], "mb") {
				v /= 1024
			}
			h.DiskGB = round1(v)
		}
		for _, x := range strings.FieldsFunc(cell(row, ip), func(r rune) bool { return r == ',' || r == ';' || r == ' ' }) {
			h.IPs = appendUnique(h.IPs, x)
		}
		for _, x := range strings.Split(cell(row, sw), ";") {
			if x = strings.TrimSpace(x); x != "" {
				h.Software = appendUnique(h.Software, x)
			}
		}
		h.Env = strings.ToLower(cell(row, env))
		if rl := strings.ToLower(cell(row, role)); rl != "" {
			h.Role = rl
		}
		if v := strings.TrimSpace(cell(row, sdk) + " " + cell(row, sdkVer)); v != "" && m.Hypervisor == "" {
			m.Hypervisor, m.HVFile, m.HVLine = v, f.Path, t.lines[i]
		}
	}
	return used
}

// ---------- cloud inventories (Azure Resource Graph, AWS Resource Explorer, generic) ----------

func providerOf(typ, id string) string {
	l := strings.ToLower(typ + " " + id)
	switch {
	case strings.Contains(l, "microsoft."):
		return "Azure"
	case strings.Contains(l, "arn:aws") || strings.Contains(l, "aws::") || regexp.MustCompile(`^[a-z0-9-]+:[a-z-]+`).MatchString(strings.ToLower(typ)):
		return "AWS"
	case strings.Contains(l, "googleapis.com"):
		return "Google Cloud"
	}
	return "Cloud"
}

func (m *InfraModel) parseCloudCSV(f *File) bool {
	t := readCSV(f)
	if t == nil {
		return false
	}
	typ := t.col("type", "resource type", "resourcetype", "asset type", "assettype")
	reg := t.col("location", "region", "aws region")
	if typ < 0 || reg < 0 {
		return false
	}
	name := t.col("name", "resource name", "identifier", "displayname", "id", "arn")
	grp := t.col("resourcegroup", "resource group", "project", "account", "owning account id")
	for i, row := range t.rows {
		res := &CloudRes{Name: cell(row, name), Type: cell(row, typ), Region: cell(row, reg), Group: cell(row, grp), File: f.Path, Line: t.lines[i]}
		if res.Type == "" {
			continue
		}
		res.Provider = providerOf(res.Type, res.Name)
		m.Cloud = append(m.Cloud, res)
	}
	return len(t.rows) > 0
}

func (m *InfraModel) parseCloudJSON(f *File) bool {
	// gcloud asset export / generic: lines of {"name":..., "assetType":..., "resource":{"location":...}}
	n := 0
	for i, l := range f.Lines {
		l = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(l), ","))
		if !strings.HasPrefix(l, "{") || !strings.Contains(l, "assetType") {
			continue
		}
		var a struct {
			Name      string `json:"name"`
			AssetType string `json:"assetType"`
			Resource  struct {
				Location string `json:"location"`
			} `json:"resource"`
		}
		if json.Unmarshal([]byte(l), &a) == nil && a.AssetType != "" {
			m.Cloud = append(m.Cloud, &CloudRes{Name: path.Base(a.Name), Type: a.AssetType, Region: a.Resource.Location, Provider: "Google Cloud", File: f.Path, Line: i + 1})
			n++
		}
	}
	return n > 0
}

// ---------- Kubernetes (kubectl get … -o json) ----------

func lineOfText(f *File, needle string, from int) int {
	for i := max(0, from); i < len(f.Lines); i++ {
		if strings.Contains(f.Lines[i], needle) {
			return i + 1
		}
	}
	return 0
}

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

func (m *InfraModel) parseKubectl(f *File) bool {
	var doc map[string]any
	if json.Unmarshal([]byte(strings.Join(f.Lines, "\n")), &doc) != nil {
		return false
	}
	var items []any
	if it, ok := doc["items"].([]any); ok {
		items = it
	} else if doc["kind"] != nil && doc["apiVersion"] != nil {
		items = []any{doc}
	}
	if len(items) == 0 || dig(items[0], "kind") == nil {
		return false
	}
	cursor := 0
	for _, it := range items {
		kind, _ := dig(it, "kind").(string)
		name, _ := dig(it, "metadata", "name").(string)
		ns, _ := dig(it, "metadata", "namespace").(string)
		if kind == "" || name == "" {
			continue
		}
		line := lineOfText(f, `"name": "`+name+`"`, cursor)
		if line == 0 {
			line = lineOfText(f, `"name":"`+name+`"`, 0)
		}
		if line > 0 {
			cursor = line
		}
		w := &Workload{Kind: kind, Name: name, Namespace: ns, File: f.Path, Line: line}
		pod := dig(it, "spec", "template", "spec")
		switch kind {
		case "CronJob":
			pod = dig(it, "spec", "jobTemplate", "spec", "template", "spec")
			w.Schedule, _ = dig(it, "spec", "schedule").(string)
		case "Pod":
			pod = dig(it, "spec")
		case "Service":
			w.ServiceType, _ = dig(it, "spec", "type").(string)
		case "Ingress":
			if rules, ok := dig(it, "spec", "rules").([]any); ok {
				for _, ru := range rules {
					if hst, _ := dig(ru, "host").(string); hst != "" {
						w.Hosts = append(w.Hosts, hst)
					}
				}
			}
		}
		if rep, ok := dig(it, "spec", "replicas").(float64); ok {
			w.Replicas = int(rep)
		} else if kind == "Deployment" || kind == "StatefulSet" {
			w.Replicas = 1
		}
		if cs, ok := dig(pod, "containers").([]any); ok {
			for _, c := range cs {
				if img, _ := dig(c, "image").(string); img != "" {
					w.Images = append(w.Images, img)
				}
				if dig(c, "resources", "limits") == nil {
					w.NoLimits = true
				}
				if p, _ := dig(c, "securityContext", "privileged").(bool); p {
					w.Privileged = true
				}
			}
		}
		m.Workloads = append(m.Workloads, w)
	}
	return len(m.Workloads) > 0
}

// ---------- collector bundle (COLLECTOR-SPEC.md) ----------

func (m *InfraModel) parseBundle(f *File) bool {
	var b struct {
		Infrastructure *struct {
			Hosts []struct {
				Name, OS, Virtualization, Role string
				CPU                            int     `json:"cpu"`
				RAMGb                          float64 `json:"ramGb"`
				Software                       []string
				IP                             string `json:"ip"`
			} `json:"hosts"`
			Connections []struct {
				From, To string
				Port     int
			} `json:"connections"`
		} `json:"infrastructure"`
		Environment string `json:"environment"`
	}
	if json.Unmarshal([]byte(strings.Join(f.Lines, "\n")), &b) != nil || b.Infrastructure == nil {
		return false
	}
	for _, x := range b.Infrastructure.Hosts {
		h := m.host(x.Name)
		if h == nil {
			continue
		}
		h.File, h.Line = f.Path, lineOfText(f, `"`+x.Name+`"`, 0)
		h.OS, h.CPU, h.RAMGB, h.Software, h.Role = x.OS, x.CPU, x.RAMGb, x.Software, x.Role
		h.Platform = x.Virtualization
		if h.Platform == "" {
			h.Platform = "Physical or unknown"
		}
		if x.IP != "" {
			h.IPs = append(h.IPs, x.IP)
		}
		if b.Environment != "" {
			h.Env = strings.ToLower(b.Environment)
		}
		if strings.Contains(x.Virtualization, "vSphere") && m.Hypervisor == "" {
			m.Hypervisor, m.HVFile, m.HVLine = x.Virtualization, f.Path, h.Line
		}
	}
	for _, c := range b.Infrastructure.Connections {
		line := lineOfText(f, `"from": "`+c.From+`"`, 0)
		if line == 0 {
			line = lineOfText(f, `"`+c.To+`"`, 0)
		}
		m.addConn(c.From, c.To, c.Port, f, line)
	}
	return true
}

// ---------- connections ----------

var connLineRe = regexp.MustCompile(`(?m)(\[?[0-9a-fA-F:.]+\]?):(\d{1,5})\s+(\[?[0-9a-fA-F:.*]+\]?):(\d{1,5}|\*)`)

func isConnCSV(f *File) bool {
	t := readCSV(f)
	return t != nil && t.col("remoteaddress", "remote address", "foreign address") >= 0 && t.col("remoteport", "remote port") >= 0
}

func fileHost(p string) string {
	b := strings.ToLower(strings.TrimSuffix(path.Base(p), path.Ext(p)))
	for _, pre := range []string{"connections", "connection", "conns", "conn", "netstat", "ss", "tcp"} {
		b = strings.TrimPrefix(b, pre)
	}
	return strings.Trim(b, "-_. ")
}

func (m *InfraModel) addConn(from, to string, port int, f *File, line int) {
	if from == "" || to == "" || from == to {
		return
	}
	if strings.HasPrefix(to, "127.") || to == "::1" || strings.HasPrefix(from, "127.") {
		return
	}
	k := strings.ToLower(from + "|" + to + "|" + strconv.Itoa(port))
	if c := m.connSeen[k]; c != nil {
		c.Count++
		return
	}
	c := &Conn{From: from, To: to, Port: port, Count: 1, File: f.Path, Line: line}
	m.connSeen[k] = c
	m.Conns = append(m.Conns, c)
}

// connection from the point of view of host self: inbound to a server port, or outbound to a peer's server port.
func (m *InfraModel) addObserved(self, local string, lport int, remote string, rport int, f *File, line int) {
	me := self
	if me == "" {
		me = m.Resolve(local)
	}
	if me == "" {
		me = local
	}
	peer := m.Resolve(remote)
	if peer == "" {
		peer = remote
	}
	if serverPort(lport) && !serverPort(rport) {
		m.addConn(peer, me, lport, f, line) // someone connects to us
	} else {
		m.addConn(me, peer, rport, f, line)
	}
}

func splitHostPort(s string) (string, int) {
	s = strings.TrimSpace(s)
	i := strings.LastIndexByte(s, ':')
	if i < 0 {
		return s, 0
	}
	p, _ := strconv.Atoi(s[i+1:])
	return strings.Trim(s[:i], "[]"), p
}

func (m *InfraModel) parseConnText(f *File) {
	self := m.Resolve(fileHost(f.Path))
	if self == "" {
		self = fileHost(f.Path)
	}
	for i, l := range f.Lines {
		if strings.Contains(strings.ToUpper(l), "LISTEN") || strings.Contains(strings.ToUpper(l), "TIME_WAIT") {
			continue
		}
		mm := connLineRe.FindStringSubmatch(l)
		if mm == nil || mm[4] == "*" {
			continue
		}
		lp, _ := strconv.Atoi(mm[2])
		rp, _ := strconv.Atoi(mm[4])
		m.addObserved(self, strings.Trim(mm[1], "[]"), lp, strings.Trim(mm[3], "[]"), rp, f, i+1)
	}
}

func (m *InfraModel) parseConnCSV(f *File) {
	t := readCSV(f)
	la, lp := t.col("localaddress", "local address"), t.col("localport", "local port")
	ra, rp := t.col("remoteaddress", "remote address", "foreign address"), t.col("remoteport", "remote port")
	st, pc := t.col("state"), t.col("pscomputername", "computername", "host")
	self := m.Resolve(fileHost(f.Path))
	if self == "" {
		self = fileHost(f.Path)
	}
	for i, row := range t.rows {
		if s := strings.ToLower(cell(row, st)); s != "" && !strings.HasPrefix(s, "establish") && s != "5" {
			continue
		}
		me := self
		if c := cell(row, pc); c != "" {
			me = c
		}
		l, _ := strconv.Atoi(cell(row, lp))
		r, _ := strconv.Atoi(cell(row, rp))
		m.addObserved(me, cell(row, la), l, cell(row, ra), r, f, t.lines[i])
	}
}

// sortedConns returns connections sorted for stable output.
func (m *InfraModel) sortedConns() []*Conn {
	out := append([]*Conn{}, m.Conns...)
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.From != b.From {
			return a.From < b.From
		}
		if a.To != b.To {
			return a.To < b.To
		}
		return a.Port < b.Port
	})
	return out
}
