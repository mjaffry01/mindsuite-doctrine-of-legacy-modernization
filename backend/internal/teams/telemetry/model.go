package telemetry

import (
	"crypto/sha1"
	"encoding/csv"
	"encoding/json"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// TModel is what the telemetry team knows about real use of the system: requests, errors and response
// times over time and per endpoint. It is built from web server access logs (NGINX/Apache combined,
// IIS W3C), monitoring exports (Splunk / Application Insights / Datadog CSV, Prometheus query_range JSON,
// CloudWatch GetMetricData JSON) and the collector bundle.
// Privacy: client addresses are only counted (hashed, in memory), never stored or reported; query
// strings are dropped; ids, e-mail addresses and tokens in URL paths are replaced by placeholders.

// Bucket is one time slot (usually an hour). Several sources often measure the same traffic (an access
// log and a monitoring export), so each source is counted separately and the highest count is used.
type Bucket struct {
	Req, Err, DurSum float64
	req, err         map[string]float64
	DurN             int
	P95              []float64 // p95 values given by a monitoring export (ms)
}

func (b *Bucket) addReq(src string, v float64) {
	if b.req == nil {
		b.req = map[string]float64{}
	}
	b.req[src] += v
	if b.req[src] > b.Req {
		b.Req = b.req[src]
	}
}

func (b *Bucket) addErr(src string, v float64) {
	if b.err == nil {
		b.err = map[string]float64{}
	}
	b.err[src] += v
	if b.err[src] > b.Err {
		b.Err = b.err[src]
	}
}

// Endpoint is one URL or operation.
type Endpoint struct {
	Name     string
	Req, Err float64
	Durs     []float64 // ms, sample
	P95      []float64 // ms, given by an export
	Last     time.Time
	File     string
	Line     int
}

// Source is one export file.
type Source struct {
	File, Kind string
	Rows       int
	From, To   time.Time
	Line       int
}

// Route is an endpoint declared in the uploaded code.
type Route struct {
	Path, File string
	Line       int
}

// Reported are numbers stated by the collector bundle (not measured here).
type Reported struct {
	Window, PeakTimes                 string
	AvgTps, PeakTps, P95Ms, ErrorRate float64
	Users, ConcurrentUsers            int
	PeakMonths                        []int
	UnusedFeatures                    []string
	File                              string
	Line                              int
}

// TModel holds the parsed telemetry.
type TModel struct {
	Buckets   map[time.Time]*Bucket
	Step      time.Duration
	Endpoints map[string]*Endpoint
	Durs      []float64 // all durations (ms), sample
	Sources   []*Source
	Routes    []Route
	Reported  *Reported
	From, To  time.Time
	clients   map[[8]byte]bool
	// LogFiles are raw access logs: their lines hold client addresses, so they are cited by file and line only.
	LogFiles map[string]bool
	steps    map[time.Duration]int
	Files    []string
}

// Clients is the number of distinct client addresses in the access logs (a rough user count).
func (m *TModel) Clients() int { return len(m.clients) }

// Totals returns requests and errors over the whole window.
func (m *TModel) Totals() (req, err float64) {
	for _, b := range m.Buckets {
		req += b.Req
		err += b.Err
	}
	return
}

// Summary is shown with the run.
func (m *TModel) Summary() map[string]any {
	req, errs := m.Totals()
	days := 0.0
	if !m.From.IsZero() {
		days = m.To.Sub(m.From).Hours() / 24
	}
	kinds := []string{}
	for _, s := range m.Sources {
		kinds = appendUnique(kinds, s.Kind)
	}
	return map[string]any{"requests": int64(req), "errors": int64(errs), "endpoints": len(m.Endpoints), "from": dstr(m.From), "to": dstr(m.To), "days": int(days + 0.5),
		"sources": kinds, "clients": m.Clients(), "exportFiles": len(m.Files), "routesInCode": len(m.Routes)}
}

func dstr(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02")
}

func (m *TModel) bucket(t time.Time) *Bucket {
	t = t.UTC().Truncate(time.Hour)
	if m.From.IsZero() || t.Before(m.From) {
		m.From = t
	}
	if t.After(m.To) {
		m.To = t
	}
	b := m.Buckets[t]
	if b == nil {
		b = &Bucket{}
		m.Buckets[t] = b
	}
	return b
}

func (m *TModel) endpoint(name string, f *File, line int) *Endpoint {
	if e := m.Endpoints[name]; e != nil {
		return e
	}
	e := &Endpoint{Name: name, File: f.Path, Line: line}
	m.Endpoints[name] = e
	return e
}

func (m *TModel) addDur(e *Endpoint, ms float64) {
	if e != nil && len(e.Durs) < 2000 {
		e.Durs = append(e.Durs, ms)
	}
	if len(m.Durs) < 100000 {
		m.Durs = append(m.Durs, ms)
	}
}

// ---------- URL normalisation (privacy) ----------

var (
	idSeg    = regexp.MustCompile(`^(\d+|[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}|[0-9a-fA-F]{16,}|[A-Za-z0-9_\-]{25,})$`)
	mailSeg  = regexp.MustCompile(`@`)
	staticRe = regexp.MustCompile(`(?i)\.(css|js|map|png|jpe?g|gif|ico|svg|woff2?|ttf|eot|webp)$`)
	paramSeg = regexp.MustCompile(`^(\{[^}]*\}|:[A-Za-z_]\w*|<[^>]*>)$`)
)

// NormPath removes the query string and replaces ids, e-mail addresses and tokens with {id}.
func NormPath(p string) string {
	if i := strings.IndexAny(p, "?#;"); i >= 0 {
		p = p[:i]
	}
	if p == "" {
		return "/"
	}
	if staticRe.MatchString(p) {
		return "(static files)"
	}
	segs := strings.Split(p, "/")
	for i, s := range segs {
		switch {
		case s == "":
		case mailSeg.MatchString(s) || idSeg.MatchString(s) || paramSeg.MatchString(s):
			segs[i] = "{id}"
		}
	}
	out := strings.TrimRight(strings.Join(segs, "/"), "/")
	if out == "" {
		return "/"
	}
	return out
}

// ParseTelemetry reads every file of the run.
func ParseTelemetry(r *Run) *TModel {
	m := &TModel{Buckets: map[time.Time]*Bucket{}, Endpoints: map[string]*Endpoint{}, clients: map[[8]byte]bool{}, steps: map[time.Duration]int{}, LogFiles: map[string]bool{}}
	for _, f := range r.Code.Files {
		for i, l := range f.Lines {
			if looksSecret(l) {
				r.MarkSecret(f.Path, i+1)
			}
		}
		head := strings.Join(f.Lines[:min(len(f.Lines), 30)], "\n")
		used := false
		switch {
		case strings.Contains(head, "#Fields:"):
			used = m.parseIIS(f)
			m.LogFiles[f.Path] = true
		case accessRe.MatchString(head):
			used = m.parseAccess(f)
			m.LogFiles[f.Path] = true
		case f.Lang == "CSV":
			used = m.parseCSV(f)
		case f.Lang == "JSON":
			used = m.parseProm(f) || m.parseCloudWatch(f) || m.parseBundle(f)
		case agentsCodeLang(f.Lang):
			m.parseRoutes(f)
			continue
		}
		if used {
			r.Cover("ing", f)
			m.Files = append(m.Files, f.Path)
		}
	}
	// the most common distance between consecutive time slots is the step
	times := make([]time.Time, 0, len(m.Buckets))
	for t := range m.Buckets {
		times = append(times, t)
	}
	sort.Slice(times, func(i, j int) bool { return times[i].Before(times[j]) })
	for i := 1; i < len(times); i++ {
		if d := times[i].Sub(times[i-1]); d > 0 && d <= 24*time.Hour {
			m.steps[d]++
		}
	}
	m.Step = time.Hour
	best := 0
	for d, n := range m.steps {
		if n > best {
			m.Step, best = d, n
		}
	}
	return m
}

// ---------- access logs ----------

var (
	accessRe = regexp.MustCompile(`(?m)^\S+ \S+ \S+ \[[^\]]+\] "[A-Z]+ \S+`)
	combined = regexp.MustCompile(`^(\S+) \S+ \S+ \[([^\]]+)\] "([A-Z]+) (\S+)[^"]*" (\d{3}) \S+(?: "[^"]*" "[^"]*")?(?: (\d+(?:\.\d+)?))?`)
)

func hashClient(ip string) [8]byte {
	h := sha1.Sum([]byte("mindsuite:" + ip))
	var k [8]byte
	copy(k[:], h[:8])
	return k
}

func (m *TModel) parseAccess(f *File) bool {
	n := 0
	src := &Source{File: f.Path, Kind: "web server access log"}
	for i, l := range f.Lines {
		mm := combined.FindStringSubmatch(l)
		if mm == nil {
			continue
		}
		t, err := time.Parse("02/Jan/2006:15:04:05 -0700", mm[2])
		if err != nil {
			continue
		}
		if src.Line == 0 {
			src.Line = i + 1
		}
		m.clients[hashClient(mm[1])] = true
		status, _ := strconv.Atoi(mm[5])
		ep := m.endpoint(mm[3]+" "+NormPath(mm[4]), f, i+1)
		b := m.bucket(t)
		b.addReq(f.Path, 1)
		ep.Req++
		if status >= 500 {
			b.addErr(f.Path, 1)
			ep.Err++
		}
		if t.After(ep.Last) {
			ep.Last = t
		}
		if mm[6] != "" {
			v, _ := strconv.ParseFloat(mm[6], 64)
			if strings.Contains(mm[6], ".") { // NGINX $request_time is in seconds
				v *= 1000
			} else {
				v /= 1000 // Apache %D is in microseconds
			}
			b.DurSum += v
			b.DurN++
			m.addDur(ep, v)
		}
		src.span(t)
		n++
	}
	src.Rows = n
	if n > 0 {
		m.Sources = append(m.Sources, src)
	}
	return n > 0
}

func (s *Source) span(t time.Time) {
	if s.From.IsZero() || t.Before(s.From) {
		s.From = t
	}
	if t.After(s.To) {
		s.To = t
	}
}

func (m *TModel) parseIIS(f *File) bool {
	var fields []string
	n := 0
	src := &Source{File: f.Path, Kind: "IIS log"}
	for i, l := range f.Lines {
		if strings.HasPrefix(l, "#Fields:") {
			fields = strings.Fields(strings.TrimPrefix(l, "#Fields:"))
			continue
		}
		if strings.HasPrefix(l, "#") || fields == nil {
			continue
		}
		vals := strings.Fields(l)
		get := func(k string) string {
			for j, x := range fields {
				if x == k && j < len(vals) {
					return vals[j]
				}
			}
			return ""
		}
		t, err := time.Parse("2006-01-02 15:04:05", get("date")+" "+get("time"))
		if err != nil {
			continue
		}
		if src.Line == 0 {
			src.Line = i + 1
		}
		if ip := get("c-ip"); ip != "" {
			m.clients[hashClient(ip)] = true
		}
		status, _ := strconv.Atoi(get("sc-status"))
		ep := m.endpoint(get("cs-method")+" "+NormPath(get("cs-uri-stem")), f, i+1)
		b := m.bucket(t)
		b.addReq(f.Path, 1)
		ep.Req++
		if status >= 500 {
			b.addErr(f.Path, 1)
			ep.Err++
		}
		if t.After(ep.Last) {
			ep.Last = t
		}
		if v, err := strconv.ParseFloat(get("time-taken"), 64); err == nil {
			b.DurSum += v
			b.DurN++
			m.addDur(ep, v)
		}
		src.span(t)
		n++
	}
	src.Rows = n
	if n > 0 {
		m.Sources = append(m.Sources, src)
	}
	return n > 0
}

// ---------- monitoring CSV (Splunk timechart, Application Insights, Datadog, generic) ----------

var timeLayouts = []string{time.RFC3339, "2006-01-02T15:04:05.000Z07:00", "2006-01-02T15:04:05.000Z", "2006-01-02 15:04:05", "2006-01-02T15:04:05", "2006-01-02 15:04", "1/2/2006 3:04:05 PM", "1/2/2006, 3:04:05.000 PM", "1/2/2006 15:04", "2006-01-02"}

func parseTime(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	for _, l := range timeLayouts {
		if t, err := time.Parse(l, s); err == nil {
			return t, true
		}
	}
	if v, err := strconv.ParseFloat(s, 64); err == nil && v > 1e9 { // unix seconds
		return time.Unix(int64(v), 0).UTC(), true
	}
	return time.Time{}, false
}

func (m *TModel) parseCSV(f *File) bool {
	rd := csv.NewReader(strings.NewReader(strings.Join(f.Lines, "\n")))
	rd.LazyQuotes, rd.FieldsPerRecord, rd.TrimLeadingSpace = true, -1, true
	var head []string
	col := func(subs ...string) int {
		for _, s := range subs {
			for i, h := range head {
				if h == s {
					return i
				}
			}
		}
		for _, s := range subs {
			for i, h := range head {
				if strings.Contains(h, s) {
					return i
				}
			}
		}
		return -1
	}
	var tc, cc, ec, pc, ac, nc int
	secs := false
	n := 0
	src := &Source{File: f.Path, Kind: "monitoring export"}
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
			tc = col("_time", "time", "timestamp", "datetime", "date", "hour", "bin")
			cc = col("count", "requests", "requestcount", "count_", "hits", "total")
			ec = col("errors", "error", "5xx", "failed", "errorcount", "failures")
			pc = col("p95", "perc95", "percentile_duration_95", "duration_p95", "95th")
			ac = col("avg_duration", "avg(duration)", "duration_avg", "avgduration", "mean")
			nc = col("name", "operation", "url", "uri", "endpoint", "route", "operation_name")
			if tc < 0 || cc < 0 {
				return false
			}
			for _, i := range []int{pc, ac} {
				if i >= 0 && (strings.Contains(head[i], "(s)") || strings.Contains(head[i], "sec") || strings.HasSuffix(head[i], "_s")) {
					secs = true
				}
			}
			continue
		}
		get := func(i int) string {
			if i < 0 || i >= len(row) {
				return ""
			}
			return strings.TrimSpace(row[i])
		}
		t, ok := parseTime(get(tc))
		if !ok {
			continue
		}
		num := func(i int) (float64, bool) {
			v, err := strconv.ParseFloat(strings.ReplaceAll(get(i), ",", ""), 64)
			return v, err == nil
		}
		req, ok := num(cc)
		if !ok {
			continue
		}
		if src.Line == 0 {
			src.Line = line
		}
		b := m.bucket(t)
		b.addReq(f.Path, req)
		var ep *Endpoint
		if nc >= 0 && get(nc) != "" {
			name := get(nc)
			if strings.HasPrefix(name, "/") || strings.Contains(name, " /") {
				parts := strings.SplitN(name, " ", 2)
				if len(parts) == 2 {
					name = parts[0] + " " + NormPath(parts[1])
				} else {
					name = NormPath(name)
				}
			}
			ep = m.endpoint(name, f, line)
			ep.Req += req
			if t.After(ep.Last) {
				ep.Last = t
			}
		}
		if e, ok := num(ec); ok {
			b.addErr(f.Path, e)
			if ep != nil {
				ep.Err += e
			}
		}
		conv := func(v float64) float64 {
			if secs {
				return v * 1000
			}
			return v
		}
		if p, ok := num(pc); ok {
			b.P95 = append(b.P95, conv(p))
			if ep != nil {
				ep.P95 = append(ep.P95, conv(p))
			}
		}
		if a, ok := num(ac); ok && req > 0 {
			b.DurSum += conv(a) * req
			b.DurN += int(req)
		}
		src.span(t)
		n++
	}
	src.Rows = n
	if n > 0 {
		m.Sources = append(m.Sources, src)
	}
	return n > 0
}

// ---------- Prometheus query_range and CloudWatch GetMetricData ----------

func (m *TModel) parseProm(f *File) bool {
	var d struct {
		Status string
		Data   struct {
			ResultType string
			Result     []struct {
				Metric map[string]string
				Values [][2]any
			}
		}
	}
	if json.Unmarshal([]byte(strings.Join(f.Lines, "\n")), &d) != nil || d.Data.ResultType != "matrix" {
		return false
	}
	base := strings.ToLower(path.Base(f.Path))
	kind := "requests"
	switch {
	case strings.Contains(base, "error") || strings.Contains(base, "5xx"):
		kind = "errors"
	case strings.Contains(base, "latency") || strings.Contains(base, "duration") || strings.Contains(base, "p95"):
		kind = "p95"
	}
	src := &Source{File: f.Path, Kind: "Prometheus", Line: 1}
	var times []float64
	for _, res := range d.Data.Result {
		for _, v := range res.Values {
			ts, _ := v[0].(float64)
			times = append(times, ts)
		}
	}
	sort.Float64s(times)
	step := 3600.0
	for i := 1; i < len(times); i++ {
		if dlt := times[i] - times[i-1]; dlt > 0 {
			step = dlt
			break
		}
	}
	for _, res := range d.Data.Result {
		name := ""
		for _, k := range []string{"handler", "route", "path", "uri", "endpoint", "operation"} {
			if v := res.Metric[k]; v != "" {
				name = NormPath(v)
			}
		}
		var ep *Endpoint
		if name != "" {
			ep = m.endpoint(name, f, 1)
		}
		for _, v := range res.Values {
			ts, _ := v[0].(float64)
			s, _ := v[1].(string)
			val, err := strconv.ParseFloat(s, 64)
			if err != nil {
				continue
			}
			t := time.Unix(int64(ts), 0).UTC()
			b := m.bucket(t)
			switch kind {
			case "requests":
				b.addReq(f.Path, val*step)
				if ep != nil {
					ep.Req += val * step
				}
			case "errors":
				b.addErr(f.Path, val*step)
				if ep != nil {
					ep.Err += val * step
				}
			case "p95":
				b.P95 = append(b.P95, val*1000)
				if ep != nil {
					ep.P95 = append(ep.P95, val*1000)
				}
			}
			src.span(t)
			src.Rows++
		}
	}
	if src.Rows > 0 {
		m.Sources = append(m.Sources, src)
	}
	return src.Rows > 0
}

func (m *TModel) parseCloudWatch(f *File) bool {
	var d struct {
		MetricDataResults []struct {
			ID         string `json:"Id"`
			Label      string
			Timestamps []string
			Values     []float64
		}
	}
	if json.Unmarshal([]byte(strings.Join(f.Lines, "\n")), &d) != nil || len(d.MetricDataResults) == 0 {
		return false
	}
	src := &Source{File: f.Path, Kind: "CloudWatch", Line: 1}
	for _, res := range d.MetricDataResults {
		l := strings.ToLower(res.Label + " " + res.ID)
		for i, ts := range res.Timestamps {
			if i >= len(res.Values) {
				break
			}
			t, ok := parseTime(ts)
			if !ok {
				continue
			}
			b := m.bucket(t)
			v := res.Values[i]
			switch {
			case strings.Contains(l, "5xx") || strings.Contains(l, "error"):
				b.addErr(f.Path, v)
			case strings.Contains(l, "responsetime") || strings.Contains(l, "latency") || strings.Contains(l, "duration"):
				b.P95 = append(b.P95, v*1000) // average target response time in seconds, used as an upper-bound proxy
			default:
				b.addReq(f.Path, v)
			}
			src.span(t)
			src.Rows++
		}
	}
	if src.Rows > 0 {
		m.Sources = append(m.Sources, src)
	}
	return src.Rows > 0
}

func (m *TModel) parseBundle(f *File) bool {
	var b struct {
		Telemetry *struct {
			Window, PeakTimes                 string
			AvgTps, PeakTps, P95Ms, ErrorRate float64
			Users, ConcurrentUsers            int
			PeakMonths                        []int
			UnusedFeatures                    []string
		} `json:"telemetry"`
	}
	if json.Unmarshal([]byte(strings.Join(f.Lines, "\n")), &b) != nil || b.Telemetry == nil {
		return false
	}
	x := b.Telemetry
	line := 1
	for i, l := range f.Lines {
		if strings.Contains(l, `"telemetry"`) {
			line = i + 1
		}
	}
	m.Reported = &Reported{Window: x.Window, PeakTimes: x.PeakTimes, AvgTps: x.AvgTps, PeakTps: x.PeakTps, P95Ms: x.P95Ms, ErrorRate: x.ErrorRate, Users: x.Users, ConcurrentUsers: x.ConcurrentUsers, PeakMonths: x.PeakMonths, UnusedFeatures: x.UnusedFeatures, File: f.Path, Line: line}
	m.Sources = append(m.Sources, &Source{File: f.Path, Kind: "collector bundle", Line: line})
	return true
}

// ---------- routes declared in the uploaded code ----------

var (
	springRe  = regexp.MustCompile(`@(Get|Post|Put|Delete|Patch|Request)Mapping\s*\(\s*(?:value\s*=\s*|path\s*=\s*)?\{?\s*"([^"]*)"`)
	jaxrsRe   = regexp.MustCompile(`@Path\s*\(\s*"([^"]*)"`)
	expressRe = regexp.MustCompile(`\.(get|post|put|delete|patch)\(\s*['"](/[^'"]*)['"]`)
	flaskRe   = regexp.MustCompile(`@\w+\.(?:route|get|post|put|delete)\(\s*['"](/[^'"]*)['"]`)
	aspnetRe  = regexp.MustCompile(`\[(?:Http(?:Get|Post|Put|Delete|Patch)|Route)\(\s*"([^"]*)"\s*\)\]`)
	classLine = regexp.MustCompile(`\b(class|interface)\s+\w+`)
)

func agentsCodeLang(l string) bool {
	switch l {
	case "Java", "Kotlin", "C#", "JavaScript", "TypeScript", "Python", "Go", "PHP", "Ruby", "Groovy", "Scala":
		return true
	}
	return false
}

func joinRoute(a, b string) string {
	p := "/" + strings.Trim(strings.Trim(a, "/")+"/"+strings.Trim(b, "/"), "/")
	return NormPath(p)
}

func (m *TModel) parseRoutes(f *File) {
	prefix := ""
	seenClass := false
	for i, l := range f.Lines {
		if !seenClass {
			if mm := springRe.FindStringSubmatch(l); mm != nil && mm[1] == "Request" {
				prefix = mm[2]
				continue
			}
			if mm := jaxrsRe.FindStringSubmatch(l); mm != nil {
				prefix = mm[1]
				continue
			}
			if mm := aspnetRe.FindStringSubmatch(l); mm != nil && strings.Contains(l, "Route(") {
				prefix = strings.ReplaceAll(mm[1], "[controller]", strings.TrimSuffix(strings.TrimSuffix(path.Base(f.Path), path.Ext(f.Path)), "Controller"))
				continue
			}
			if classLine.MatchString(l) {
				seenClass = true
				continue
			}
		}
		add := func(p string) {
			m.Routes = append(m.Routes, Route{Path: joinRoute(prefix, p), File: f.Path, Line: i + 1})
		}
		switch {
		case springRe.MatchString(l):
			add(springRe.FindStringSubmatch(l)[2])
		case seenClass && jaxrsRe.MatchString(l):
			add(jaxrsRe.FindStringSubmatch(l)[1])
		case seenClass && aspnetRe.MatchString(l):
			add(aspnetRe.FindStringSubmatch(l)[1])
		case expressRe.MatchString(l):
			add(expressRe.FindStringSubmatch(l)[2])
		case flaskRe.MatchString(l):
			add(flaskRe.FindStringSubmatch(l)[1])
		}
	}
}

// ---------- statistics ----------

func percentile(v []float64, p float64) float64 {
	if len(v) == 0 {
		return 0
	}
	s := append([]float64(nil), v...)
	sort.Float64s(s)
	i := int(p*float64(len(s)-1) + 0.5)
	return s[i]
}

func median(v []float64) float64 { return percentile(v, 0.5) }
