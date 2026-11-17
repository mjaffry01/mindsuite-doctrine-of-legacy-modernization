package messaging

import (
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// MsgModel is what the messaging & caching team knows: brokers, queues and topics, channels to other
// queue managers, consumer groups and caches. It is built only from admin-command output that displays
// information (runmqsc DISPLAY, kafka-topics / kafka-consumer-groups, redis-cli INFO / CONFIG GET,
// memcached stats, RabbitMQ definitions) and the collector bundle. Message contents are never read.

// Broker is a queue manager, Kafka cluster, RabbitMQ node or cloud broker.
type Broker struct {
	Kind, Name, Version, DeadQ string
	File                       string
	Line                       int
}

// Dest is a queue, topic or exchange.
type Dest struct {
	Kind, Name, Broker, Type      string // Kind: queue | topic | exchange
	Depth, MaxDepth, In, Out      int
	HasDepth, HasProcs            bool
	Backout, Persist, Remote      string
	Partitions, Replication       int
	Durable, DeadLetter, Explicit bool
	File                          string
	Line                          int
}

// Channel is an IBM MQ channel.
type Channel struct {
	Name, Type, Conn, XmitQ, Cipher, MCAUser, QMgr string
	File                                           string
	Line                                           int
}

// Group is a Kafka consumer group on one topic.
type Group struct {
	Name, Topic string
	Lag         int64
	Active      bool
	File        string
	Line        int
}

// Cache is a Redis or Memcached instance.
type Cache struct {
	Kind, Name, Version, Policy, Role string
	Hits, Misses, Evicted, Clients    int64
	UsedMem, MaxMem, Keys, Expires    int64
	AOF, RDBSaves, HasStats           bool
	File                              string
	Line                              int
	Lines                             map[string]int
}

// MsgModel holds the parsed messaging estate.
type MsgModel struct {
	Brokers    []*Broker
	Dests      []*Dest
	Channels   []*Channel
	Groups     []*Group
	Caches     []*Cache
	Users      []string // broker user names (RabbitMQ); never passwords
	UsersFile  string
	UsersLine  int
	SystemQs   int
	Files      []string
	destByName map[string]*Dest
	cacheByKey map[string]*Cache
}

// Summary is shown with the run.
func (m *MsgModel) Summary() map[string]any {
	kinds := map[string]int{}
	for _, d := range m.Dests {
		kinds[d.Kind]++
	}
	b := []string{}
	for _, x := range m.Brokers {
		b = append(b, strings.TrimSpace(x.Kind+" "+x.Name))
	}
	return map[string]any{"brokers": len(m.Brokers), "brokerNames": b, "queues": kinds["queue"], "topics": kinds["topic"], "exchanges": kinds["exchange"],
		"channels": len(m.Channels), "consumerGroups": len(m.Groups), "caches": len(m.Caches), "exportFiles": len(m.Files)}
}

func (m *MsgModel) broker(kind, name string, f *File, line int) *Broker {
	for _, b := range m.Brokers {
		if b.Kind == kind && (b.Name == name || b.Name == "" || name == "") {
			if b.Name == "" {
				b.Name = name
			}
			return b
		}
	}
	b := &Broker{Kind: kind, Name: name, File: f.Path, Line: line}
	m.Brokers = append(m.Brokers, b)
	return b
}

func (m *MsgModel) dest(kind, name, broker string, f *File, line int) *Dest {
	k := kind + "|" + name
	if d := m.destByName[k]; d != nil {
		return d
	}
	d := &Dest{Kind: kind, Name: name, Broker: broker, File: f.Path, Line: line}
	m.destByName[k] = d
	m.Dests = append(m.Dests, d)
	return d
}

// Names lists every queue, topic and exchange name.
func (m *MsgModel) Names() []string {
	out := []string{}
	for _, d := range m.Dests {
		out = append(out, d.Name)
	}
	sort.Strings(out)
	return out
}

// ParseMsg reads every export file of the run.
func ParseMsg(r *Run) *MsgModel {
	m := &MsgModel{destByName: map[string]*Dest{}, cacheByKey: map[string]*Cache{}}
	for _, f := range r.Code.Files {
		for i, l := range f.Lines {
			if looksSecret(l) {
				r.MarkSecret(f.Path, i+1)
			}
		}
		head := strings.Join(f.Lines[:min(len(f.Lines), 60)], "\n")
		used := false
		switch {
		case mqscRe.MatchString(head):
			used = m.parseMQSC(f)
		case strings.Contains(head, "redis_version:") || strings.Contains(head, "maxmemory") || strings.Contains(head, "keyspace_hits"):
			used = m.parseRedis(f)
		case strings.Contains(head, "STAT version") || strings.Contains(head, "STAT get_hits"):
			used = m.parseMemcached(f)
		case kafkaGroupHead.MatchString(head):
			used = m.parseKafkaGroups(f)
		case strings.Contains(head, "PartitionCount:") || (strings.Contains(head, "Topic:") && strings.Contains(head, "ReplicationFactor")):
			used = m.parseKafkaDescribe(f)
		case f.Lang == "JSON":
			used = m.parseRabbit(f) || m.parseBundle(f)
		case f.Lang == "Text" && strings.Contains(strings.ToLower(path.Base(f.Path)), "topic"):
			used = m.parseKafkaList(f)
		}
		if used {
			r.Cover("brk", f)
			m.Files = append(m.Files, f.Path)
		}
	}
	return m
}

// ---------- IBM MQ (runmqsc DISPLAY output) ----------

var (
	mqscRe   = regexp.MustCompile(`(?m)AMQ84\d\dI|^\s*(QUEUE|CHANNEL|QMNAME)\([^)]*\)|Starting MQSC for queue manager`)
	mqAttrRe = regexp.MustCompile(`([A-Z][A-Z0-9]*)\(((?:[^()']|'[^']*')*)\)`)
	mqQmRe   = regexp.MustCompile(`Starting MQSC for queue manager ([\w.%/]+)`)
)

func mqVersion(v string) string {
	if len(v) == 8 && strings.Trim(v, "0123456789") == "" { // 09010000 -> 9.1.0.0
		a, _ := strconv.Atoi(v[0:2])
		b, _ := strconv.Atoi(v[2:4])
		c, _ := strconv.Atoi(v[4:6])
		d, _ := strconv.Atoi(v[6:8])
		return fmt.Sprintf("%d.%d.%d.%d", a, b, c, d)
	}
	return v
}

func (m *MsgModel) parseMQSC(f *File) bool {
	qm := ""
	var q *Dest
	var ch *Channel
	var b *Broker
	used := false
	for i, l := range f.Lines {
		if mm := mqQmRe.FindStringSubmatch(l); mm != nil {
			qm = strings.TrimSuffix(mm[1], ".")
			b = m.broker("IBM MQ", qm, f, i+1)
			used = true
		}
		if strings.Contains(strings.ToUpper(l), "DISPLAY ") {
			continue // the echoed command, not output
		}
		for _, a := range mqAttrRe.FindAllStringSubmatch(l, -1) {
			k, v := a[1], strings.Trim(strings.TrimSpace(a[2]), "'")
			v = strings.TrimSpace(v)
			switch k {
			case "QMNAME":
				qm = v
				b = m.broker("IBM MQ", qm, f, i+1)
				q, ch, used = nil, nil, true
			case "QUEUE":
				ch = nil
				if strings.HasPrefix(v, "SYSTEM.") || strings.HasPrefix(v, "AMQ.") {
					m.SystemQs++
					q = &Dest{} // attributes of system queues are ignored
					continue
				}
				q = m.dest("queue", v, qm, f, i+1)
				q.Explicit, used = true, true
			case "CHANNEL":
				q = nil
				if strings.HasPrefix(v, "SYSTEM.") {
					ch = &Channel{}
					continue
				}
				ch = &Channel{Name: v, QMgr: qm, File: f.Path, Line: i + 1}
				m.Channels = append(m.Channels, ch)
				used = true
			}
			if b != nil {
				switch k {
				case "VERSION":
					b.Version = mqVersion(v)
				case "DEADQ":
					b.DeadQ = v
				}
			}
			if q != nil && q.Name != "" {
				switch k {
				case "TYPE":
					q.Type = v
				case "CURDEPTH":
					q.Depth, _ = strconv.Atoi(v)
					q.HasDepth = true
				case "MAXDEPTH":
					q.MaxDepth, _ = strconv.Atoi(v)
				case "BOQNAME":
					q.Backout = v
				case "DEFPSIST":
					q.Persist = v
				case "IPPROCS":
					q.In, _ = strconv.Atoi(v)
					q.HasProcs = true
				case "OPPROCS":
					q.Out, _ = strconv.Atoi(v)
					q.HasProcs = true
				case "RNAME", "TARGET":
					q.Remote = v
				case "RQMNAME":
					q.Remote = strings.TrimSpace(q.Remote + " @ " + v)
				}
			}
			if ch != nil && ch.Name != "" {
				switch k {
				case "CHLTYPE":
					ch.Type = v
				case "CONNAME":
					ch.Conn = v
				case "XMITQ":
					ch.XmitQ = v
				case "SSLCIPH":
					ch.Cipher = v
				case "MCAUSER":
					ch.MCAUser = v
				}
			}
		}
	}
	if used && qm == "" {
		m.broker("IBM MQ", "", f, 1)
	}
	return used
}

// ---------- Kafka ----------

var (
	kafkaGroupHead = regexp.MustCompile(`(?m)^\s*GROUP\s+TOPIC\s+PARTITION\s+.*LAG`)
	kafkaTopicRe   = regexp.MustCompile(`^\s*Topic:\s*(\S+)\s.*PartitionCount:\s*(\d+)\s.*ReplicationFactor:\s*(\d+)`)
	kafkaNameRe    = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
)

func (m *MsgModel) kafka(f *File) *Broker { return m.broker("Kafka", "", f, 1) }

func (m *MsgModel) parseKafkaList(f *File) bool {
	n := 0
	for i, l := range f.Lines {
		l = strings.TrimSpace(l)
		if l == "" || !kafkaNameRe.MatchString(l) || strings.HasPrefix(l, "__") {
			continue
		}
		d := m.dest("topic", l, "Kafka", f, i+1)
		d.Explicit = true
		n++
	}
	if n > 0 {
		m.kafka(f)
	}
	return n > 0
}

func (m *MsgModel) parseKafkaDescribe(f *File) bool {
	n := 0
	for i, l := range f.Lines {
		mm := kafkaTopicRe.FindStringSubmatch(l)
		if mm == nil || strings.HasPrefix(mm[1], "__") {
			continue
		}
		d := m.dest("topic", mm[1], "Kafka", f, i+1)
		d.Explicit = true
		d.File, d.Line = f.Path, i+1
		d.Partitions, _ = strconv.Atoi(mm[2])
		d.Replication, _ = strconv.Atoi(mm[3])
		n++
	}
	if n > 0 {
		m.kafka(f)
	}
	return n > 0
}

func (m *MsgModel) parseKafkaGroups(f *File) bool {
	type key struct{ g, t string }
	agg := map[key]*Group{}
	var order []key
	var cols []string
	for i, l := range f.Lines {
		fs := strings.Fields(l)
		if len(fs) == 0 {
			continue
		}
		if fs[0] == "GROUP" {
			cols = fs
			continue
		}
		if cols == nil || len(fs) < 6 {
			continue
		}
		get := func(name string) string {
			for j, c := range cols {
				if c == name && j < len(fs) {
					return fs[j]
				}
			}
			return ""
		}
		k := key{get("GROUP"), get("TOPIC")}
		if k.t == "" || strings.HasPrefix(k.t, "__") {
			continue
		}
		g := agg[k]
		if g == nil {
			g = &Group{Name: k.g, Topic: k.t, File: f.Path, Line: i + 1}
			agg[k] = g
			order = append(order, k)
		}
		lag, _ := strconv.ParseInt(get("LAG"), 10, 64)
		g.Lag += lag
		if c := get("CONSUMER-ID"); c != "" && c != "-" {
			g.Active = true
		}
		m.dest("topic", k.t, "Kafka", f, i+1)
	}
	for _, k := range order {
		m.Groups = append(m.Groups, agg[k])
	}
	if len(order) > 0 {
		m.kafka(f)
	}
	return len(order) > 0
}

// ---------- Redis (INFO, CONFIG GET) and Memcached (stats) ----------

func cacheName(p string) string {
	b := strings.ToLower(strings.TrimSuffix(path.Base(p), path.Ext(p)))
	for _, w := range []string{"redis", "memcached", "memcache", "info", "stats", "memory", "config", "keyspace"} {
		b = strings.ReplaceAll(b, w, "")
	}
	b = strings.Trim(regexp.MustCompile(`[-_.]{2,}`).ReplaceAllString(b, "-"), "-_. ")
	if b == "" {
		return "cache"
	}
	return b
}

func (m *MsgModel) cache(kind string, f *File) *Cache {
	k := kind + "|" + cacheName(f.Path)
	if c := m.cacheByKey[k]; c != nil {
		return c
	}
	c := &Cache{Kind: kind, Name: cacheName(f.Path), File: f.Path, Line: 1, Lines: map[string]int{}}
	m.cacheByKey[k] = c
	m.Caches = append(m.Caches, c)
	return c
}

var dbKeysRe = regexp.MustCompile(`^db\d+:keys=(\d+),expires=(\d+)`)

func (m *MsgModel) parseRedis(f *File) bool {
	c := m.cache("Redis", f)
	n := func(s string) int64 { v, _ := strconv.ParseInt(strings.TrimSpace(s), 10, 64); return v }
	prev := ""
	for i, raw := range f.Lines {
		l := strings.TrimSpace(raw)
		// CONFIG GET output: `1) "maxmemory-policy"` / `2) "noeviction"`, or plain two lines
		plain := strings.Trim(regexp.MustCompile(`^\d+\)\s*`).ReplaceAllString(l, ""), `"`)
		if prev == "maxmemory-policy" && plain != "" {
			c.Policy, c.Lines["maxmemory_policy"] = plain, i+1
		}
		if prev == "maxmemory" && plain != "" && c.MaxMem == 0 {
			c.MaxMem, c.Lines["maxmemory"] = n(plain), i+1
		}
		prev = plain
		if mm := dbKeysRe.FindStringSubmatch(l); mm != nil {
			c.Keys += n(mm[1])
			c.Expires += n(mm[2])
			c.Lines["keyspace"] = i + 1
			continue
		}
		k, v, ok := strings.Cut(l, ":")
		if !ok || strings.HasPrefix(l, "#") {
			continue
		}
		c.HasStats = true
		switch k {
		case "redis_version":
			c.Version = v
		case "keyspace_hits":
			c.Hits = n(v)
		case "keyspace_misses":
			c.Misses = n(v)
		case "evicted_keys":
			c.Evicted = n(v)
		case "used_memory":
			c.UsedMem = n(v)
		case "maxmemory":
			c.MaxMem = n(v)
		case "maxmemory_policy":
			c.Policy = v
		case "aof_enabled":
			c.AOF = v == "1"
		case "rdb_changes_since_last_save", "rdb_last_bgsave_status":
			c.RDBSaves = true
		case "role":
			c.Role = v
		case "connected_clients":
			c.Clients = n(v)
		}
		c.Lines[k] = i + 1
	}
	return c.HasStats || c.Policy != ""
}

func (m *MsgModel) parseMemcached(f *File) bool {
	c := m.cache("Memcached", f)
	c.Policy = "lru"
	for i, l := range f.Lines {
		fs := strings.Fields(l)
		if len(fs) != 3 || fs[0] != "STAT" {
			continue
		}
		v, _ := strconv.ParseInt(fs[2], 10, 64)
		c.HasStats = true
		switch fs[1] {
		case "version":
			c.Version = fs[2]
		case "get_hits":
			c.Hits = v
		case "get_misses":
			c.Misses = v
		case "evictions":
			c.Evicted = v
		case "bytes":
			c.UsedMem = v
		case "limit_maxbytes":
			c.MaxMem = v
		case "curr_items":
			c.Keys = v
		case "curr_connections":
			c.Clients = v
		}
		c.Lines[fs[1]] = i + 1
	}
	return c.HasStats
}

// ---------- RabbitMQ definitions export and the collector bundle ----------

func lineOfText(f *File, needle string) int {
	for i, l := range f.Lines {
		if strings.Contains(l, needle) {
			return i + 1
		}
	}
	return 1
}

func (m *MsgModel) parseRabbit(f *File) bool {
	var d struct {
		Version string `json:"rabbit_version"`
		Queues  []struct {
			Name, Vhost string
			Durable     bool
			Arguments   map[string]any
		} `json:"queues"`
		Exchanges []struct{ Name, Type string } `json:"exchanges"`
		Users     []struct{ Name, Tags any }    `json:"users"`
		Policies  []struct {
			Pattern    string
			Definition map[string]any
		} `json:"policies"`
	}
	if json.Unmarshal([]byte(strings.Join(f.Lines, "\n")), &d) != nil || (d.Version == "" && d.Queues == nil) {
		return false
	}
	b := m.broker("RabbitMQ", "", f, lineOfText(f, "rabbit_version"))
	b.Version = d.Version
	dlxPatterns := []*regexp.Regexp{}
	for _, p := range d.Policies {
		if _, ok := p.Definition["dead-letter-exchange"]; ok {
			if re, err := regexp.Compile(p.Pattern); err == nil {
				dlxPatterns = append(dlxPatterns, re)
			}
		}
	}
	for _, q := range d.Queues {
		x := m.dest("queue", q.Name, "RabbitMQ", f, lineOfText(f, `"name": "`+q.Name+`"`))
		x.Explicit, x.Durable, x.Type = true, q.Durable, "RabbitMQ queue"
		if _, ok := q.Arguments["x-dead-letter-exchange"]; ok {
			x.DeadLetter = true
		}
		for _, re := range dlxPatterns {
			if re.MatchString(q.Name) {
				x.DeadLetter = true
			}
		}
	}
	for _, e := range d.Exchanges {
		if e.Name == "" || strings.HasPrefix(e.Name, "amq.") {
			continue
		}
		x := m.dest("exchange", e.Name, "RabbitMQ", f, lineOfText(f, `"name": "`+e.Name+`"`))
		x.Explicit, x.Type = true, e.Type
	}
	for _, u := range d.Users {
		if n, _ := u.Name.(string); n != "" {
			m.Users = append(m.Users, n)
		}
	}
	if len(m.Users) > 0 {
		m.UsersFile, m.UsersLine = f.Path, lineOfText(f, `"users"`)
	}
	return true
}

func (m *MsgModel) parseBundle(f *File) bool {
	var b struct {
		Middleware *struct {
			Brokers []struct {
				Type, Name, Version string
				Queues              []string
			} `json:"brokers"`
			Caches []struct {
				Type, Name, Version, EvictionPolicy string
				HitRatio                            float64 `json:"hitRatio"`
				MaxMemoryMb                         float64 `json:"maxMemoryMb"`
			} `json:"caches"`
		} `json:"middleware"`
	}
	if json.Unmarshal([]byte(strings.Join(f.Lines, "\n")), &b) != nil || b.Middleware == nil {
		return false
	}
	for _, x := range b.Middleware.Brokers {
		br := m.broker(x.Type, x.Name, f, lineOfText(f, `"`+x.Name+`"`))
		br.Version = x.Version
		for _, q := range x.Queues {
			d := m.dest(map[bool]string{true: "topic", false: "queue"}[x.Type == "Kafka"], q, x.Name, f, lineOfText(f, `"`+q+`"`))
			d.Explicit = true
		}
	}
	for _, x := range b.Middleware.Caches {
		c := &Cache{Kind: x.Type, Name: x.Name, Version: x.Version, Policy: x.EvictionPolicy, MaxMem: int64(x.MaxMemoryMb * 1024 * 1024), File: f.Path, Line: lineOfText(f, `"`+x.Name+`"`), Lines: map[string]int{}}
		if x.HitRatio > 0 {
			c.Hits, c.Misses, c.HasStats = int64(x.HitRatio*1000), int64((1-x.HitRatio)*1000), true
		}
		m.Caches = append(m.Caches, c)
	}
	return true
}
