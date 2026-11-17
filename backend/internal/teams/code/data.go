package code

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// ---------- 4 Data access ----------

var (
	sqlReadRe   = regexp.MustCompile(`(?i)\b(?:from|join)\s+([A-Za-z_][\w$#]*(?:\.[A-Za-z_][\w$#]*)?)`)
	sqlWriteRe  = regexp.MustCompile(`(?i)\b(?:insert\s+into|delete\s+from|merge\s+into|truncate\s+table)\s+([A-Za-z_][\w$#]*(?:\.[A-Za-z_][\w$#]*)?)`)
	sqlUpdateRe = regexp.MustCompile(`(?i)\bupdate\s+([A-Za-z_][\w$#]*(?:\.[A-Za-z_][\w$#]*)?)\s+set\b`)
	sqlHintRe   = regexp.MustCompile(`(?i)\b(select|insert|update|delete|merge)\b`)
	jpaTableRe  = regexp.MustCompile(`@Table\s*\(\s*(?:name\s*=\s*)?"([^"]+)"`)
	jpaEntityRe = regexp.MustCompile(`@Entity\b`)
	classNameRe = regexp.MustCompile(`\b(?:class|interface|enum|record)\s+([A-Z]\w*)`)
	efDbSetRe   = regexp.MustCompile(`DbSet<(\w+)>\s+(\w+)`)
	djangoRe    = regexp.MustCompile(`^\s*class\s+(\w+)\s*\(\s*(?:models\.)?Model\s*\)`)
	mongoCollRe = regexp.MustCompile(`(?:getCollection|collection)\s*\(\s*["']([\w.\-]+)["']|@Document\s*\(\s*(?:collection\s*=\s*)?"([^"]+)"`)
	procCallRe  = regexp.MustCompile(`(?i)(?:\{\s*\??=?\s*call\s+([\w.$#]+)|withProcedureName\s*\(\s*"([\w.$#]+)"|\bexec(?:ute)?\s+([A-Za-z_][\w.$#]+)|StoredProcedureQuery\s*\(\s*"([\w.$#]+)"|CommandType\.StoredProcedure)`)
	ddlObjRe    = regexp.MustCompile(`(?i)^\s*create\s+(?:or\s+replace\s+)?(?:editionable\s+)?(table|view|procedure|function|package\s+body|package|trigger|sequence|materialized\s+view)\s+(?:if\s+not\s+exists\s+)?([\w.$#"\[\]]+)`)
	notTables   = map[string]bool{"dual": true, "select": true, "where": true, "set": true, "values": true, "the": true, "a": true, "an": true, "this": true, "that": true, "table": true,
		"each": true, "it": true, "here": true, "there": true, "them": true, "which": true, "what": true, "your": true, "our": true, "my": true, "to": true, "and": true, "or": true, "in": true, "on": true,
		"as": true, "is": true, "by": true, "for": true, "of": true, "with": true, "user_tables": false, "lateral": true, "unnest": true, "now": true, "being": true, "all": true, "any": true, "file": true,
		"disk": true, "memory": true, "cache": true, "server": true, "client": true, "list": true, "map": true, "string": true, "null": true, "true": true, "false": true, "import": true, "java": true, "org": true, "com": true}
)

type tableUse struct {
	module, table, op string
	count             int
	file              *File
	line              int
}

func normTable(t string) string {
	t = strings.Trim(t, `"[]`+"`")
	if t == "" {
		return ""
	}
	l := strings.ToLower(t)
	if notTables[l] || len(l) < 2 || strings.HasPrefix(l, "java.") {
		return ""
	}
	return strings.ToUpper(t)
}

func runDataAccess(r *Run) {
	s := r.Store
	uses := map[string]*tableUse{}
	add := func(f *File, n int, table, op string) {
		t := normTable(table)
		if t == "" {
			return
		}
		k := f.Module + "|" + t + "|" + op
		u := uses[k]
		if u == nil {
			u = &tableUse{module: f.Module, table: t, op: op, file: f, line: n}
			uses[k] = u
		}
		u.count++
	}
	procs := 0
	keep := func(f *File) bool {
		return CodeLangs[f.Lang] || f.Lang == "SQL" || f.Lang == "PL/SQL" || f.Lang == "XML" || LegacyLangs[f.Lang] || f.Lang == "JSP"
	}
	eachLine(r, "dat", keep, func(f *File, n int, l string) {
		isSQL := f.Lang == "SQL" || f.Lang == "PL/SQL"
		inString := strings.ContainsAny(l, `"'`) && sqlHintRe.MatchString(l)
		if f.Lang == "XML" {
			inString = sqlHintRe.MatchString(l) // MyBatis / iBatis mappers
		}
		if isSQL || inString || (LegacyLangs[f.Lang] && strings.Contains(strings.ToUpper(l), "EXEC SQL")) || strings.Contains(l, "@Query") {
			for _, m := range sqlWriteRe.FindAllStringSubmatch(l, -1) {
				add(f, n, m[1], "write")
			}
			for _, m := range sqlUpdateRe.FindAllStringSubmatch(l, -1) {
				add(f, n, m[1], "write")
			}
			if sqlHintRe.MatchString(l) || isSQL {
				for _, m := range sqlReadRe.FindAllStringSubmatch(l, -1) {
					add(f, n, m[1], "read")
				}
			}
		}
		if m := jpaTableRe.FindStringSubmatch(l); m != nil {
			add(f, n, m[1], "mapped")
		}
		if m := efDbSetRe.FindStringSubmatch(l); m != nil {
			add(f, n, m[2], "mapped")
		}
		if m := djangoRe.FindStringSubmatch(l); m != nil {
			add(f, n, m[1], "mapped")
		}
		if m := mongoCollRe.FindStringSubmatch(l); m != nil {
			add(f, n, m[1]+m[2], "mapped")
		}
		if isSQL {
			if m := ddlObjRe.FindStringSubmatch(l); m != nil {
				typ := strings.ToLower(strings.Join(strings.Fields(m[1]), " "))
				name := strings.ToUpper(strings.Trim(m[2], `"[]`))
				kind := "db-object"
				if typ == "procedure" || typ == "function" || typ == "package body" || typ == "package" || typ == "trigger" {
					kind = "stored-procedure"
					procs++
				}
				s.Add(Finding{Agent: "dat", Kind: kind, Module: f.Module, Title: fmt.Sprintf("%s %s defined", strings.Title(typ), name), Citations: []Citation{cite(f, n)}, Confidence: "high",
					Data: map[string]any{"type": typ, "name": name}})
			}
		} else if m := procCallRe.FindStringSubmatch(l); m != nil && (strings.ContainsAny(l, `"'`) || strings.Contains(l, "CommandType")) {
			name := strings.ToUpper(m[1] + m[2] + m[3] + m[4])
			if name == "" {
				name = "(name set elsewhere)"
			}
			s.Add(Finding{Agent: "dat", Kind: "procedure-call", Module: f.Module, Title: f.Module + " calls stored procedure " + name, Citations: []Citation{cite(f, n)}, Confidence: "medium",
				Data: map[string]any{"name": name}})
		}
		// JPA entity without @Table: the class name is the table
		if jpaEntityRe.MatchString(l) {
			for j := n; j < n+6 && j <= len(f.Lines); j++ {
				if jpaTableRe.MatchString(f.Lines[j-1]) {
					break
				}
				if m := classNameRe.FindStringSubmatch(f.Lines[j-1]); m != nil {
					add(f, j, m[1], "mapped")
					break
				}
			}
		}
	})
	keys := sortedKeys(uses)
	byTable := map[string]map[string]string{} // table -> module -> strongest op
	for _, k := range keys {
		u := uses[k]
		s.Add(Finding{Agent: "dat", Kind: "table-access", Module: u.module, Title: fmt.Sprintf("%s %s %s (%d places)", u.module, map[string]string{"read": "reads", "write": "writes", "mapped": "maps"}[u.op], u.table, u.count),
			Citations: []Citation{cite(u.file, u.line)}, Confidence: map[string]string{"read": "medium", "write": "high", "mapped": "high"}[u.op],
			Data: map[string]any{"table": u.table, "op": u.op, "count": u.count}})
		if byTable[u.table] == nil {
			byTable[u.table] = map[string]string{}
		}
		if u.op == "write" || byTable[u.table][u.module] == "" {
			byTable[u.table][u.module] = u.op
		}
	}
	shared := 0
	for _, t := range sortedKeys(byTable) {
		mods := sortedKeys(byTable[t])
		writers := []string{}
		for _, m := range mods {
			if byTable[t][m] == "write" || byTable[t][m] == "mapped" {
				writers = append(writers, m)
			}
		}
		if len(writers) >= 2 || len(mods) >= 3 {
			shared++
			var ev []string
			for _, f := range s.Findings("table-access") {
				if f.Data["table"] == t {
					ev = append(ev, f.ID)
				}
			}
			sort.Strings(ev)
			s.Add(Finding{Agent: "dat", Kind: "shared-table", Title: fmt.Sprintf("Shared table %s: used by %s", t, strings.Join(mods, ", ")),
				Detail:   "Several modules depend on this table. Before separating them, decide which one owns it and how the others will get the data.",
				Evidence: ev, Basis: BasisInferred, Confidence: "medium", Data: map[string]any{"table": t, "modules": mods, "writers": writers}})
		}
	}
	s.Log("dat", fmt.Sprintf("%d tables found, %d shared between modules, %d database programs defined.", len(byTable), shared, procs))
}

// ---------- 5 Interfaces ----------

var (
	springMapRe    = regexp.MustCompile(`@(Get|Post|Put|Delete|Patch|Request)Mapping\s*(?:\(\s*(?:value\s*=\s*|path\s*=\s*)?\{?\s*"([^"]*)")?`)
	springMethodRe = regexp.MustCompile(`RequestMethod\.(GET|POST|PUT|DELETE|PATCH)`)
	jaxPathRe      = regexp.MustCompile(`@Path\s*\(\s*"([^"]*)"`)
	jaxVerbRe      = regexp.MustCompile(`^\s*@(GET|POST|PUT|DELETE|PATCH)\b`)
	expressRe      = regexp.MustCompile(`\b(?:app|router|server)\.(get|post|put|delete|patch|all)\s*\(\s*['"\x60]([^'"\x60]+)['"\x60]`)
	flaskRe        = regexp.MustCompile(`@\w+\.(route|get|post|put|delete|patch)\s*\(\s*['"]([^'"]+)['"](.*)`)
	djangoUrlRe    = regexp.MustCompile(`\b(?:path|re_path|url)\s*\(\s*r?['"]([^'"]*)['"]`)
	aspVerbRe      = regexp.MustCompile(`\[Http(Get|Post|Put|Delete|Patch)(?:\s*\(\s*"([^"]*)")?`)
	aspRouteRe     = regexp.MustCompile(`\[Route\s*\(\s*"([^"]*)"`)
	goHandleRe     = regexp.MustCompile(`\.(?:HandleFunc|Handle|GET|POST|PUT|DELETE|PATCH|Get|Post|Put|Delete)\s*\(\s*"(/[^"]*)"`)
	servletRe      = regexp.MustCompile(`@WebServlet\s*\(\s*(?:urlPatterns\s*=\s*|value\s*=\s*)?\{?\s*"([^"]+)"|<url-pattern>([^<]+)</url-pattern>`)
	soapRe         = regexp.MustCompile(`@WebService\b|\[ServiceContract\]|\[WebMethod\]|<wsdl:definitions|\.asmx\b`)
	urlRe          = regexp.MustCompile(`https?://([A-Za-z0-9.\-]+)(?::\d+)?`)
	httpClientRe   = regexp.MustCompile(`\b(RestTemplate|WebClient|HttpClient|OkHttpClient|Feign(?:Client)?|axios|fetch\s*\(|requests\.(?:get|post|put|delete)|HttpWebRequest|WebRequest\.Create|urllib|http\.(?:Get|Post|NewRequest)|SoapClient|JAXWS|Service\.create)\b`)
	feignRe        = regexp.MustCompile(`@FeignClient\s*\(\s*(?:name\s*=\s*|value\s*=\s*)?"([^"]+)"`)
	kafkaListenRe  = regexp.MustCompile(`@KafkaListener\s*\(.*?topics\s*=\s*\{?\s*"([^"]+)"`)
	jmsListenRe    = regexp.MustCompile(`@JmsListener\s*\(.*?destination\s*=\s*"([^"]+)"`)
	rabbitListenRe = regexp.MustCompile(`@RabbitListener\s*\(.*?queues\s*=\s*\{?\s*"([^"]+)"`)
	sendTopicRe    = regexp.MustCompile(`(?:kafkaTemplate|jmsTemplate|rabbitTemplate|producer)\.(?:send|convertAndSend)\s*\(\s*"([^"]+)"`)
	mqTechRe       = regexp.MustCompile(`\b(IBM ?MQ|MQQueueManager|com\.ibm\.mq|TIBCO|tibjms|ActiveMQ|RabbitMQ|AmazonSQS|SqsClient|ServiceBusClient|Azure\.Messaging\.ServiceBus|MSMQ|MessageQueue\(|PubSub|EventHub)`)
	fileXRe        = regexp.MustCompile(`(?i)\b(sftp|ftps?|JSch|FTPClient|ChannelSftp|SmbFile|WinSCP)\b`)
	ignoreHosts    = regexp.MustCompile(`(?i)(^|\.)(w3\.org|apache\.org|xmlsoap\.org|java\.sun\.com|sun\.com|oracle\.com/xml|springframework\.org|maven\.apache\.org|schemas\.microsoft\.com|json-schema\.org|example\.(com|org)|localhost|127\.0\.0\.1|xml\.org|jcp\.org|hibernate\.org|github\.com|openxmlformats\.org|purl\.org|w3c\.org|eclipse\.org|jboss\.org|mozilla\.org|ietf\.org|opensource\.org|gnu\.org|swagger\.io|googleapis\.com/auth)$`)
)

func runInterfaces(r *Run) {
	s := r.Store
	hosts := map[string]Citation{}
	hostMod := map[string]string{}
	keep := func(f *File) bool {
		return (CodeLangs[f.Lang] || f.Lang == "XML" || f.Lang == "Properties" || f.Lang == "YAML" || f.Lang == "Config" || f.Lang == "JSON" || f.Lang == "WSDL") && !f.IsTest
	}
	var classPrefix string
	var curFile *File
	pendingVerb := ""
	endpoints := 0
	seen := map[string]bool{}
	eachLine(r, "int", keep, func(f *File, n int, l string) {
		if f != curFile {
			curFile, classPrefix, pendingVerb = f, "", ""
		}
		ep := func(method, p string) {
			endpoints++
			s.Add(Finding{Agent: "int", Kind: "endpoint", Module: f.Module, Title: strings.ToUpper(method) + " " + p, Citations: []Citation{cite(f, n)}, Confidence: "high",
				Data: map[string]any{"method": strings.ToUpper(method), "path": p}})
		}
		trim := strings.TrimSpace(l)
		if m := springMapRe.FindStringSubmatch(l); m != nil {
			// class-level @RequestMapping sets a prefix
			isClassLevel := false
			for j := n + 1; j <= n+4 && j <= len(f.Lines); j++ {
				if classNameRe.MatchString(f.Lines[j-1]) && !strings.Contains(f.Lines[j-1], "(") {
					isClassLevel = true
					break
				}
				if strings.Contains(f.Lines[j-1], "(") && !strings.HasPrefix(strings.TrimSpace(f.Lines[j-1]), "@") {
					break
				}
			}
			if isClassLevel {
				classPrefix = m[2]
			} else {
				method := m[1]
				if method == "Request" {
					method = "ANY"
					if mm := springMethodRe.FindStringSubmatch(l); mm != nil {
						method = mm[1]
					}
				}
				ep(method, joinPath(classPrefix, m[2]))
			}
		}
		if m := jaxPathRe.FindStringSubmatch(l); m != nil {
			if pendingVerb != "" {
				ep(pendingVerb, joinPath(classPrefix, m[1]))
				pendingVerb = ""
			} else if strings.HasPrefix(trim, "@Path") && classPrefix == "" && !nextIsMethod(f, n) {
				classPrefix = m[1]
			} else {
				pendingVerb = "PATH:" + m[1]
			}
		}
		if m := jaxVerbRe.FindStringSubmatch(l); m != nil {
			if strings.HasPrefix(pendingVerb, "PATH:") {
				ep(m[1], joinPath(classPrefix, strings.TrimPrefix(pendingVerb, "PATH:")))
				pendingVerb = ""
			} else if nextIsMethod(f, n) {
				ep(m[1], joinPath(classPrefix, ""))
			} else {
				pendingVerb = m[1]
			}
		}
		if m := aspRouteRe.FindStringSubmatch(l); m != nil && strings.Contains(strings.Join(f.Lines[n:min(n+3, len(f.Lines))], " "), "class ") {
			classPrefix = m[1]
		}
		if m := aspVerbRe.FindStringSubmatch(l); m != nil {
			ep(m[1], joinPath(classPrefix, m[2]))
		}
		for _, m := range expressRe.FindAllStringSubmatch(l, -1) {
			if strings.HasPrefix(m[2], "/") {
				ep(m[1], m[2])
			}
		}
		if m := flaskRe.FindStringSubmatch(l); m != nil && f.Lang == "Python" {
			method := m[1]
			if method == "route" {
				method = "GET"
				if mm := regexp.MustCompile(`methods\s*=\s*\[([^\]]+)\]`).FindStringSubmatch(m[3]); mm != nil {
					method = strings.ToUpper(strings.Trim(strings.Split(mm[1], ",")[0], ` '"`))
				}
			}
			ep(method, m[2])
		}
		if f.Lang == "Python" && strings.HasSuffix(f.Path, "urls.py") {
			if m := djangoUrlRe.FindStringSubmatch(l); m != nil {
				ep("ANY", "/"+m[1])
			}
		}
		if f.Lang == "Go" {
			if m := goHandleRe.FindStringSubmatch(l); m != nil {
				ep("ANY", m[1])
			}
		}
		if m := servletRe.FindStringSubmatch(l); m != nil {
			ep("ANY", m[1]+m[2])
		}
		if soapRe.MatchString(l) {
			s.Add(Finding{Agent: "int", Kind: "soap-service", Module: f.Module, Title: "SOAP / web service in " + f.Path, Citations: []Citation{cite(f, n)}, Confidence: "high"})
		}
		for _, m := range urlRe.FindAllStringSubmatch(l, -1) {
			h := strings.ToLower(m[1])
			if ignoreHosts.MatchString(h) || !strings.Contains(h, ".") {
				continue
			}
			if _, ok := hosts[h]; !ok {
				hosts[h] = cite(f, n)
				hostMod[h] = f.Module
			}
		}
		if m := feignRe.FindStringSubmatch(l); m != nil {
			s.Add(Finding{Agent: "int", Kind: "outbound-call", Module: f.Module, Title: f.Module + " calls service " + m[1] + " (Feign)", Citations: []Citation{cite(f, n)}, Confidence: "high", Data: map[string]any{"target": m[1], "client": "Feign"}})
		}
		for _, pair := range []struct {
			re   *regexp.Regexp
			tech string
			dir  string
		}{{kafkaListenRe, "Kafka", "consumes"}, {jmsListenRe, "JMS", "consumes"}, {rabbitListenRe, "RabbitMQ", "consumes"}, {sendTopicRe, "messaging", "publishes"}} {
			if m := pair.re.FindStringSubmatch(l); m != nil {
				s.Add(Finding{Agent: "int", Kind: "messaging", Module: f.Module, Title: fmt.Sprintf("%s %s %s (%s)", f.Module, pair.dir, m[1], pair.tech), Citations: []Citation{cite(f, n)}, Confidence: "high",
					Data: map[string]any{"tech": pair.tech, "direction": pair.dir, "destination": m[1]}})
			}
		}
		if m := mqTechRe.FindStringSubmatch(l); m != nil && !strings.HasPrefix(trim, "//") && once(seen, f.Module+"|mq|"+m[1]) {
			s.Add(Finding{Agent: "int", Kind: "messaging-tech", Module: f.Module, Title: "Message broker client: " + m[1], Citations: []Citation{cite(f, n)}, Confidence: "medium", Data: map[string]any{"tech": m[1]}})
		}
		if m := fileXRe.FindStringSubmatch(l); m != nil && !strings.HasPrefix(trim, "import") && once(seen, f.Module+"|fx|"+strings.ToLower(m[1])) {
			s.Add(Finding{Agent: "int", Kind: "file-exchange", Module: f.Module, Title: "File transfer (" + strings.ToUpper(m[1]) + ") in " + f.Module, Citations: []Citation{cite(f, n)}, Confidence: "medium", Data: map[string]any{"protocol": m[1]}})
		}
	})
	for _, h := range sortedKeys(hosts) {
		s.Add(Finding{Agent: "int", Kind: "outbound-call", Module: hostMod[h], Title: "Calls or references host " + h, Citations: []Citation{hosts[h]}, Confidence: "medium", Data: map[string]any{"target": h}})
	}
	s.Log("int", fmt.Sprintf("%d endpoints, %d external hosts.", endpoints, len(hosts)))
}

func nextIsMethod(f *File, n int) bool {
	for j := n + 1; j <= n+4 && j <= len(f.Lines); j++ {
		t := strings.TrimSpace(f.Lines[j-1])
		if strings.HasPrefix(t, "@") {
			continue
		}
		return strings.Contains(t, "(") && !classNameRe.MatchString(t)
	}
	return false
}

func joinPath(a, b string) string {
	p := "/" + strings.Trim(a, "/") + "/" + strings.Trim(b, "/")
	p = strings.ReplaceAll(p, "//", "/")
	if len(p) > 1 {
		p = strings.TrimSuffix(p, "/")
	}
	return p
}

// once reports whether key is seen for the first time.
func once(seen map[string]bool, key string) bool {
	if seen[key] {
		return false
	}
	seen[key] = true
	return true
}
