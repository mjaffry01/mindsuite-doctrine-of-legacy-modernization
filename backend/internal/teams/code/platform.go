package code

import (
	"fmt"
	"path"
	"regexp"
	"strings"
)

// ---------- 8 Platform & config ----------

var (
	dockerFromRe = regexp.MustCompile(`(?i)^\s*FROM\s+(\S+)`)
	composeImgRe = regexp.MustCompile(`^\s*image:\s*["']?([^\s"']+)`)
	k8sKindRe    = regexp.MustCompile(`^kind:\s*(\w+)`)
	cronSchedRe  = regexp.MustCompile(`^\s*schedule:\s*["']?([^"'\n]+)`)
	tfResRe      = regexp.MustCompile(`^\s*resource\s+"([\w-]+)"\s+"([\w-]+)"`)
	jdbcRe       = regexp.MustCompile(`jdbc:(\w+)(?::(?:thin:@|//|@)?(?:[^@/\s;"']*@)?([\w.\-]+))?`)
	dsnRes       = []struct {
		tech string
		re   *regexp.Regexp
	}{
		{"MongoDB", regexp.MustCompile(`mongodb(?:\+srv)?://(?:[^@\s/]*@)?([\w.\-]+)`)},
		{"Redis", regexp.MustCompile(`rediss?://(?:[^@\s/]*@)?([\w.\-]+)`)},
		{"RabbitMQ / AMQP", regexp.MustCompile(`amqps?://(?:[^@\s/]*@)?([\w.\-]+)`)},
		{"Kafka", regexp.MustCompile(`bootstrap[._-]servers["']?\s*[:=]\s*["']?([\w.\-]+)`)},
		{"SQL Server", regexp.MustCompile(`(?i)(?:Data Source|Server)\s*=\s*([\w.\-\\]+)\s*;`)},
		{"Elasticsearch", regexp.MustCompile(`(?i)elasticsearch[\w.]*(?:uris|host)s?\s*[:=]\s*["']?(?:https?://)?([\w.\-]+)`)},
	}
	cacheRes = []struct {
		tech string
		re   *regexp.Regexp
	}{
		{"Spring cache annotations", regexp.MustCompile(`@Cacheable|@EnableCaching|@CacheEvict`)},
		{"Redis", regexp.MustCompile(`(?i)RedisTemplate|StackExchange\.Redis|spring\.redis|spring\.data\.redis|ioredis|redis\.createClient|IDistributedCache`)},
		{"Ehcache", regexp.MustCompile(`(?i)ehcache`)},
		{"Hazelcast", regexp.MustCompile(`(?i)hazelcast`)},
		{"Memcached", regexp.MustCompile(`(?i)memcache`)},
		{"Caffeine / Guava cache", regexp.MustCompile(`Caffeine\.newBuilder|CacheBuilder\.newBuilder`)},
		{"In-process cache (.NET)", regexp.MustCompile(`IMemoryCache|MemoryCache\.Default|HttpRuntime\.Cache`)},
		{"Hibernate second-level cache", regexp.MustCompile(`hibernate\.cache\.use_second_level_cache\s*[=:]\s*true|@Cache\(usage`)},
	}
	schedRes = []struct {
		tech string
		re   *regexp.Regexp
	}{
		{"Spring @Scheduled", regexp.MustCompile(`@Scheduled\s*\(\s*(?:cron\s*=\s*"([^"]+)"|fixed(?:Rate|Delay)(?:String)?\s*=\s*"?([\w${}.]+))`)},
		{"Quartz", regexp.MustCompile(`cronSchedule\s*\(\s*"([^"]+)"|CronTrigger|JobDetail|<cron-expression>([^<]+)<`)},
		{"Hangfire", regexp.MustCompile(`RecurringJob\.AddOrUpdate`)},
		{"node-cron / APScheduler / Celery beat", regexp.MustCompile(`cron\.schedule\s*\(\s*['"]([^'"]+)['"]|add_job\(|crontab\(|beat_schedule`)},
		{"Windows Task / Timer service", regexp.MustCompile(`System\.Timers\.Timer|ServiceBase\b|schtasks`)},
	}
	appServers = map[string]string{"weblogic.xml": "Oracle WebLogic", "weblogic-application.xml": "Oracle WebLogic", "ibm-web-bnd.xml": "IBM WebSphere", "ibm-web-ext.xml": "IBM WebSphere",
		"jboss-web.xml": "JBoss / WildFly", "jboss-deployment-structure.xml": "JBoss / WildFly", "context.xml": "Apache Tomcat", "server.xml": "Apache Tomcat", "web.config": "IIS / ASP.NET", "glassfish-web.xml": "GlassFish"}
	envFileRe = regexp.MustCompile(`(?i)(?:application|appsettings|config|settings)[-._]([a-z]+)\.(?:properties|ya?ml|json|config)$|\.env\.([a-z]+)$`)
)

func runPlatform(r *Run) {
	s := r.Store
	seen := map[string]bool{}
	envs := map[string]Citation{}
	keep := func(f *File) bool { return !f.IsTest }
	eachLine(r, "pla", keep, func(f *File, n int, l string) {
		base := strings.ToLower(path.Base(f.Path))
		if n == 1 {
			if srv, ok := appServers[base]; ok && once(seen, "as|"+srv) {
				s.Add(Finding{Agent: "pla", Kind: "app-server", Title: "Application server: " + srv, Citations: []Citation{{File: f.Path, Line: 1}}, Confidence: "medium", Data: map[string]any{"server": srv}})
			}
			if m := envFileRe.FindStringSubmatch(base); m != nil {
				env := strings.ToLower(m[1] + m[2])
				if _, ok := envs[env]; !ok {
					envs[env] = Citation{File: f.Path, Line: 1}
				}
			}
			ci := ""
			switch {
			case strings.Contains(f.Path, ".github/workflows/"):
				ci = "GitHub Actions"
			case base == "jenkinsfile":
				ci = "Jenkins"
			case base == ".gitlab-ci.yml":
				ci = "GitLab CI"
			case base == "azure-pipelines.yml":
				ci = "Azure Pipelines"
			case base == "bitbucket-pipelines.yml":
				ci = "Bitbucket Pipelines"
			}
			if ci != "" && once(seen, "ci|"+ci) {
				s.Add(Finding{Agent: "pla", Kind: "ci-pipeline", Title: "Build pipeline: " + ci, Citations: []Citation{{File: f.Path, Line: 1}}, Confidence: "high", Data: map[string]any{"tool": ci}})
			}
			if base == "chart.yaml" && once(seen, "helm|"+f.Path) {
				s.Add(Finding{Agent: "pla", Kind: "k8s-object", Title: "Helm chart: " + path.Dir(f.Path), Citations: []Citation{{File: f.Path, Line: 1}}, Confidence: "high", Data: map[string]any{"kind": "HelmChart"}})
			}
		}
		if f.Lang == "Dockerfile" {
			if m := dockerFromRe.FindStringSubmatch(l); m != nil {
				s.Add(Finding{Agent: "pla", Kind: "container", Module: f.Module, Title: "Container image built from " + m[1], Citations: []Citation{cite(f, n)}, Confidence: "high", Data: map[string]any{"base": m[1]}})
			}
		}
		if f.Lang == "YAML" {
			if strings.Contains(base, "compose") {
				if m := composeImgRe.FindStringSubmatch(l); m != nil && once(seen, "img|"+m[1]) {
					s.Add(Finding{Agent: "pla", Kind: "container", Title: "Compose service image " + m[1], Citations: []Citation{cite(f, n)}, Confidence: "high", Data: map[string]any{"image": m[1]}})
				}
			}
			if m := k8sKindRe.FindStringSubmatch(l); m != nil {
				s.Add(Finding{Agent: "pla", Kind: "k8s-object", Title: "Kubernetes " + m[1] + " in " + f.Path, Citations: []Citation{cite(f, n)}, Confidence: "high", Data: map[string]any{"kind": m[1]}})
			}
			if m := cronSchedRe.FindStringSubmatch(l); m != nil && strings.Count(m[1], " ") >= 4 {
				s.Add(Finding{Agent: "pla", Kind: "schedule", Title: "Kubernetes CronJob schedule " + strings.TrimSpace(m[1]), Citations: []Citation{cite(f, n)}, Confidence: "high", Data: map[string]any{"tech": "Kubernetes CronJob", "when": strings.TrimSpace(m[1])}})
			}
		}
		if f.Lang == "Terraform" {
			if m := tfResRe.FindStringSubmatch(l); m != nil {
				s.Add(Finding{Agent: "pla", Kind: "iac-resource", Title: "Terraform " + m[1] + " " + m[2], Citations: []Citation{cite(f, n)}, Confidence: "high", Data: map[string]any{"type": m[1], "name": m[2]}})
			}
		}
		if m := jdbcRe.FindStringSubmatch(l); m != nil && once(seen, "db|"+m[1]+"|"+m[2]) {
			target := m[1]
			if m[2] != "" {
				target += " at " + m[2]
			}
			s.Add(Finding{Agent: "pla", Kind: "connection", Module: f.Module, Title: "Database connection: " + target, Citations: []Citation{cite(f, n)}, Confidence: "high", Data: map[string]any{"tech": m[1], "host": m[2]}})
		}
		for _, d := range dsnRes {
			if m := d.re.FindStringSubmatch(l); m != nil && once(seen, "dsn|"+d.tech+"|"+m[1]) {
				s.Add(Finding{Agent: "pla", Kind: "connection", Module: f.Module, Title: d.tech + " connection: " + m[1], Citations: []Citation{cite(f, n)}, Confidence: "medium", Data: map[string]any{"tech": d.tech, "host": m[1]}})
			}
		}
		if CodeLangs[f.Lang] || f.Lang == "XML" || f.Lang == "Properties" || f.Lang == "YAML" || f.Lang == "Config" || f.Lang == "JSON" {
			for _, c := range cacheRes {
				if c.re.MatchString(l) && once(seen, "cache|"+c.tech+"|"+f.Module) {
					s.Add(Finding{Agent: "pla", Kind: "cache", Module: f.Module, Title: "Caching: " + c.tech, Citations: []Citation{cite(f, n)}, Confidence: "medium", Data: map[string]any{"tech": c.tech}})
				}
			}
			for _, sc := range schedRes {
				if m := sc.re.FindStringSubmatch(l); m != nil {
					when := strings.Join(m[1:], "")
					s.Add(Finding{Agent: "pla", Kind: "schedule", Module: f.Module, Title: "Scheduled job (" + sc.tech + ")" + map[bool]string{true: ": " + when, false: ""}[when != ""], Citations: []Citation{cite(f, n)}, Confidence: "high",
						Data: map[string]any{"tech": sc.tech, "when": when}})
				}
			}
		}
	})
	for _, e := range sortedKeys(envs) {
		s.Add(Finding{Agent: "pla", Kind: "environment", Title: "Environment configuration: " + e, Citations: []Citation{envs[e]}, Confidence: "medium", Data: map[string]any{"environment": e}})
	}
	s.Log("pla", fmt.Sprintf("%d environments found in configuration file names.", len(envs)))
}

// ---------- 11 Legacy-language specialists ----------

var (
	cblProgRe = regexp.MustCompile(`(?i)PROGRAM-ID\.\s*([\w-]+)`)
	cblCallRe = regexp.MustCompile(`(?i)\bCALL\s+['"]([\w-]+)['"]`)
	cblCopyRe = regexp.MustCompile(`(?i)^\s*(?:\d{6})?\s*COPY\s+([\w-]+)`)
	cicsRe    = regexp.MustCompile(`(?i)EXEC\s+CICS\s+(\w+)(?:\s+(?:PROGRAM|MAP|FILE|DATASET|TRANSID)\s*\(\s*['"]?([\w-]+))?`)
	cblFileRe = regexp.MustCompile(`(?i)^\s*(?:\d{6})?\s*SELECT\s+([\w-]+)\s+ASSIGN\s+TO\s+([\w-]+)`)
	jclJobRe  = regexp.MustCompile(`^//(\w+)\s+JOB\b`)
	jclStepRe = regexp.MustCompile(`^//(\w+)\s+EXEC\s+(?:PGM=(\w+)|(?:PROC=)?(\w+))`)
	jclDsnRe  = regexp.MustCompile(`DSN=([\w.$#@()+&-]+)`)
	rpgFileRe = regexp.MustCompile(`(?i)^\s{5}F(\w{1,10})\s+[IOUC]|^\s*dcl-f\s+(\w+)`)
	rpgCallRe = regexp.MustCompile(`(?i)\bCALLP?\s*\(?\s*'?([\w]+)|^\s{5}C\s.*\bCALL\s+'(\w+)'`)
	pbSqlRe   = regexp.MustCompile(`(?i)retrieve\s*=\s*"\s*(?:PBSELECT|SELECT)|SQLCA|DECLARE\s+\w+\s+CURSOR`)
	pbWinRe   = regexp.MustCompile(`(?i)^\s*(?:global\s+)?type\s+(w_\w+|d_\w+|u_\w+)\s+from\s+(\w+)`)
)

func runLegacy(r *Run) {
	s := r.Store
	keep := func(f *File) bool { return LegacyLangs[f.Lang] }
	seen := map[string]bool{}
	eachLine(r, "leg", keep, func(f *File, n int, l string) {
		switch f.Lang {
		case "COBOL", "COBOL copybook":
			if m := cblProgRe.FindStringSubmatch(l); m != nil {
				s.Add(Finding{Agent: "leg", Kind: "legacy-program", Module: f.Module, Title: "COBOL program " + m[1], Citations: []Citation{cite(f, n)}, Confidence: "high", Data: map[string]any{"program": m[1], "lang": "COBOL"}})
			}
			if m := cblCallRe.FindStringSubmatch(l); m != nil && once(seen, f.Path+"|call|"+m[1]) {
				s.Add(Finding{Agent: "leg", Kind: "legacy-call", Module: f.Module, Title: path.Base(f.Path) + " calls " + m[1], Citations: []Citation{cite(f, n)}, Confidence: "high", Data: map[string]any{"target": m[1]}})
			}
			if m := cblCopyRe.FindStringSubmatch(l); m != nil && once(seen, f.Path+"|copy|"+m[1]) {
				s.Add(Finding{Agent: "leg", Kind: "copybook-use", Module: f.Module, Title: path.Base(f.Path) + " uses copybook " + m[1], Citations: []Citation{cite(f, n)}, Confidence: "high", Data: map[string]any{"copybook": m[1]}})
			}
			if m := cicsRe.FindStringSubmatch(l); m != nil && once(seen, f.Path+"|cics|"+m[1]+m[2]) {
				s.Add(Finding{Agent: "leg", Kind: "cics", Module: f.Module, Title: "CICS " + strings.ToUpper(m[1]) + " " + m[2], Citations: []Citation{cite(f, n)}, Confidence: "high", Data: map[string]any{"command": m[1], "target": m[2]}})
			}
			if m := cblFileRe.FindStringSubmatch(l); m != nil {
				s.Add(Finding{Agent: "leg", Kind: "legacy-file", Module: f.Module, Title: "File " + m[1] + " assigned to " + m[2], Citations: []Citation{cite(f, n)}, Confidence: "high"})
			}
		case "JCL":
			if m := jclJobRe.FindStringSubmatch(l); m != nil {
				s.Add(Finding{Agent: "leg", Kind: "batch-job", Module: f.Module, Title: "Batch job " + m[1], Citations: []Citation{cite(f, n)}, Confidence: "high", Data: map[string]any{"job": m[1]}})
			}
			if m := jclStepRe.FindStringSubmatch(l); m != nil {
				s.Add(Finding{Agent: "leg", Kind: "job-step", Module: f.Module, Title: "Job step " + m[1] + " runs " + m[2] + m[3], Citations: []Citation{cite(f, n)}, Confidence: "high", Data: map[string]any{"step": m[1], "program": m[2] + m[3]}})
			}
			if m := jclDsnRe.FindStringSubmatch(l); m != nil && once(seen, f.Path+"|dsn|"+m[1]) {
				s.Add(Finding{Agent: "leg", Kind: "dataset", Module: f.Module, Title: "Dataset " + m[1], Citations: []Citation{cite(f, n)}, Confidence: "high"})
			}
		case "RPG", "CL":
			if m := rpgFileRe.FindStringSubmatch(l); m != nil {
				s.Add(Finding{Agent: "leg", Kind: "legacy-file", Module: f.Module, Title: "RPG file " + m[1] + m[2], Citations: []Citation{cite(f, n)}, Confidence: "medium"})
			}
			if m := rpgCallRe.FindStringSubmatch(l); m != nil && once(seen, f.Path+"|call|"+m[1]+m[2]) {
				s.Add(Finding{Agent: "leg", Kind: "legacy-call", Module: f.Module, Title: path.Base(f.Path) + " calls " + m[1] + m[2], Citations: []Citation{cite(f, n)}, Confidence: "medium"})
			}
			if n == 1 {
				s.Add(Finding{Agent: "leg", Kind: "legacy-program", Module: f.Module, Title: f.Lang + " program " + strings.TrimSuffix(path.Base(f.Path), path.Ext(f.Path)), Citations: []Citation{{File: f.Path, Line: 1}}, Confidence: "high", Data: map[string]any{"lang": f.Lang}})
			}
		case "PowerBuilder":
			if m := pbWinRe.FindStringSubmatch(l); m != nil {
				s.Add(Finding{Agent: "leg", Kind: "legacy-program", Module: f.Module, Title: "PowerBuilder " + m[1] + " (" + m[2] + ")", Citations: []Citation{cite(f, n)}, Confidence: "high", Data: map[string]any{"lang": "PowerBuilder"}})
			}
			if pbSqlRe.MatchString(l) && once(seen, f.Path+"|sql") {
				s.Add(Finding{Agent: "leg", Kind: "legacy-sql", Module: f.Module, Title: "Embedded SQL / DataWindow in " + path.Base(f.Path), Citations: []Citation{cite(f, n)}, Confidence: "medium"})
			}
		}
	})
	s.Log("leg", fmt.Sprintf("%d legacy findings. Oracle Forms (.fmb) and ABAP need a specialist export; they are listed as gaps if present.", len(s.Findings("legacy-program", "legacy-call", "copybook-use", "cics", "batch-job", "job-step"))))
}
