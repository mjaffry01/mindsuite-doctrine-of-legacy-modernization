package agents

import (
	"regexp"
	"strings"
)

var bracketRe = regexp.MustCompile(`\([^)]*\)`)

var skuRe = regexp.MustCompile(`^(v?\d+(\.\d+)*|[a-z]{1,2}\d+[a-z]*\d*)$`)

// Curated product catalogue for the options Verifier. A proposed option whose name matches one of these
// (as whole words, case-insensitive) is marked "known"; anything else is kept but marked
// "needs confirmation", so a product the model made up is never presented as established.
// Extend the list as the organization confirms products. Names as vendors use them in 2026.
var productCatalogue = []string{
	// Clouds and hosting
	"amazon web services", "aws", "microsoft azure", "azure", "google cloud", "oracle cloud infrastructure", "oci", "ibm cloud",
	"alibaba cloud", "huawei cloud", "openshift", "red hat openshift", "vmware", "vmware cloud foundation", "nutanix",
	"kubernetes", "amazon eks", "amazon ecs", "aws fargate", "aws lambda", "aws elastic beanstalk", "amazon ec2", "aws app runner",
	"azure kubernetes service", "aks", "azure container apps", "azure app service", "azure functions", "azure virtual machines",
	"google kubernetes engine", "gke", "cloud run", "cloud functions", "compute engine", "app engine",
	"oci kubernetes engine", "oke", "oci functions", "oci compute", "ibm cloud code engine", "ibm power virtual server",
	"docker", "rancher", "tanzu", "knative", "heroku", "cloud foundry",
	// Data
	"postgresql", "amazon aurora", "aurora postgresql", "amazon rds", "azure database for postgresql", "azure sql database",
	"azure sql managed instance", "sql server", "cloud sql", "alloydb", "cloud spanner", "spanner", "mysql", "mariadb",
	"oracle database", "autonomous database", "exadata", "db2", "ibm db2", "mongodb", "mongodb atlas", "cosmos db", "azure cosmos db",
	"dynamodb", "amazon dynamodb", "documentdb", "firestore", "cassandra", "redis", "elasticsearch", "opensearch", "snowflake",
	"databricks", "bigquery", "amazon redshift", "azure synapse", "microsoft fabric", "cockroachdb", "yugabytedb", "tidb",
	"aws database migration service", "aws dms", "azure database migration service", "database migration service", "datastream",
	"oracle goldengate", "goldengate", "debezium", "qlik replicate", "fivetran", "aws schema conversion tool", "ora2pg", "flyway", "liquibase",
	// Integration
	"apache kafka", "kafka", "confluent", "confluent cloud", "amazon msk", "azure event hubs", "event hubs", "google pub sub", "pub sub",
	"rabbitmq", "ibm mq", "activemq", "amazon sqs", "amazon sns", "azure service bus", "service bus", "nats",
	"amazon api gateway", "azure api management", "api management", "apigee", "kong", "kong gateway", "mulesoft", "mulesoft anypoint",
	"boomi", "dell boomi", "azure logic apps", "logic apps", "ibm app connect", "tibco", "webmethods", "biztalk", "oracle integration",
	"aws step functions", "camunda", "apache camel", "spring integration", "aws transfer family", "goanywhere", "ibm sterling", "axway",
	"graphql", "grpc", "apollo", "eventbridge", "amazon eventbridge", "azure event grid",
	// Application and frameworks
	"spring boot", "spring", "quarkus", "micronaut", "jakarta ee", "asp.net core", ".net", "blazor", "entity framework", "node.js",
	"nestjs", "express", "fastapi", "django", "go", "kotlin", "java", "python", "typescript", "react", "angular", "vue", "next.js",
	"flutter", "react native", ".net maui", "maui", "power apps", "microsoft power apps", "outsystems", "mendix", "appian",
	"servicenow app engine", "retool",
	// Legacy runtimes and conversion
	"micro focus enterprise server", "rocket enterprise server", "rocket software", "micro focus", "lzlabs", "lzlabs software defined mainframe",
	"aws mainframe modernization", "aws blu age", "blu age", "ibm z", "ibm zsystems", "ibm watsonx code assistant for z", "ibm z and cloud modernization stack",
	"heirloom computing", "tmaxsoft", "openframe", "modern systems", "ensono", "kyndryl", "mobilize.net", "gapvelocity", "visual basic upgrade companion",
	"appeon powerbuilder", "powerbuilder", "powerserver", "ibm i", "profound logic", "fresche", "lansa", "delphi", "rad studio",
	"app service", "app service environment", "azure static web apps", "static web apps", "azure spring apps", "azure red hat openshift", "azure virtual desktop", "windows forms", "winforms", "wpf", "razor pages", "asp.net core razor pages",
	"asp.net core web api", "asp.net core mvc", "web api", "microsoft entra id", "entra id", "azure front door", "front door", "azure application gateway",
	"application gateway", "azure firewall", "azure dns", "azure cdn", "azure private link", "private link", "private endpoint", "azure nat gateway",
	"nat gateway", "azure vpn gateway", "vpn gateway", "expressroute", "azure expressroute", "azure blob storage", "blob storage", "azure files",
	"azure key vault", "key vault", "azure monitor", "application insights", "azure bastion", "azure load balancer", "cloudflare", "amazon cloudfront",
	"cloudfront", "aws waf", "elastic load balancing", "application load balancer", "aws privatelink", "aws direct connect", "cloud armor", "cloud load balancing",
	"cics", "ibm cics", "jcl", "control-m", "bmc control-m", "autosys", "apache airflow", "airflow", "spring batch",
	// Products and SaaS
	"sap s 4hana", "s 4hana", "sap", "oracle fusion", "oracle netsuite", "netsuite", "microsoft dynamics 365", "dynamics 365",
	"dynamics 365 business central", "business central", "salesforce", "salesforce sales cloud", "salesforce service cloud", "workday",
	"servicenow", "odoo", "sage", "sage intacct", "xero", "quickbooks", "infor", "epicor", "ifs", "zoho", "zoho inventory",
	"freshservice", "jira", "stripe", "razorpay", "paytm", "phonepe", "paypal", "adyen", "square", "braintree", "twilio", "sendgrid", "mailchimp", "shopify", "woocommerce", "tally", "tallyprime", "whatsapp business", "slack", "microsoft teams", "okta", "auth0", "google workspace", "zendesk", "hubspot", "guidewire", "duck creek", "temenos", "finastra", "oracle flexcube", "infosys finacle", "finacle",
}

var catalogueSet = func() map[string]bool {
	m := map[string]bool{}
	for _, n := range productCatalogue {
		m[normName(n)] = true
	}
	return m
}()

// Words that can follow or join a product name without making it a different product.
var catalogueFiller = map[string]bool{"for": true, "on": true, "with": true, "and": true, "the": true, "of": true, "by": true, "in": true,
	"managed": true, "flexible": true, "server": true, "service": true, "services": true, "edition": true, "enterprise": true,
	"standard": true, "premium": true, "plan": true, "tier": true, "cloud": true, "platform": true, "instance": true, "cluster": true,
	"serverless": true, "dedicated": true, "hyperscale": true, "single": true, "database": true, "apps": true, "app": true,
	"api": true, "apis": true, "gateway": true, "current": true, "via": true, "using": true, "linux": true, "windows": true, "integration": true, "regional": true, "global": true, "runtime": true, "framework": true, "lts": true, "latest": true, "version": true}

// knownProduct reports whether catalogue names cover the option's whole name (vendor prefix allowed),
// apart from filler words. "Azure Database for PostgreSQL Flexible Server" is known; "Ledger Cloud for Azure" is not.
func knownProduct(name, vendor string) bool {
	// Text in brackets is a description, e.g. "Azure App Service (Web Apps)".
	name = bracketRe.ReplaceAllString(name, " ")
	words := strings.Fields(normName(name))
	if len(words) == 0 {
		return false
	}
	covered := make([]bool, len(words))
	matched := false
	// Every catalogue name found anywhere in the option name covers its words; overlapping names are fine,
	// so "Azure App Service Environment" is covered by "azure app service" and "app service environment".
	for i := range words {
		for j := i + 1; j <= len(words); j++ {
			if catalogueSet[strings.Join(words[i:j], " ")] {
				for k := i; k < j; k++ {
					covered[k] = true
				}
				matched = true
			}
		}
	}
	if !matched {
		return false
	}
	v := map[string]bool{}
	for _, w := range strings.Fields(normName(vendor)) {
		v[w] = true
	}
	for i, w := range words {
		// SKU and version tokens such as "v3", "p1v3", "8.0" do not make a different product.
		if !covered[i] && !catalogueFiller[w] && !v[w] && !skuRe.MatchString(w) {
			return false
		}
	}
	return true
}
