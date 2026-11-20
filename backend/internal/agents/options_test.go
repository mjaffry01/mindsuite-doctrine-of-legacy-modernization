package agents

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func optionsInput() OptionsInput {
	return OptionsInput{
		Context: "Whole system", Strategic: "core", Areas: []string{"data", "hosting", "data"},
		Facts: []OptionFact{{ID: "F1", Area: "data", Text: "SQL Server LocalDB, 6 tables"}, {ID: "F2", Area: "stack", Text: "VB.NET Windows Forms on .NET Framework 4.8"}},
		Rules: []OptionRule{{Area: "cloud", Option: "Amazon Web Services", State: "excluded", Reason: "group policy"}, {Area: "database", Option: "PostgreSQL (managed)", State: "preferred"}},
	}
}

func TestOptionsValidate(t *testing.T) {
	in := optionsInput()
	if err := in.Validate(); err != nil {
		t.Fatal(err)
	}
	if len(in.Areas) != 2 {
		t.Fatalf("areas not de-duplicated: %v", in.Areas)
	}
	for name, mod := range map[string]func(*OptionsInput){
		"no facts":     func(i *OptionsInput) { i.Facts = nil },
		"bad fact id":  func(i *OptionsInput) { i.Facts[0].ID = "X1" },
		"repeated id":  func(i *OptionsInput) { i.Facts[1].ID = "F1" },
		"unknown area": func(i *OptionsInput) { i.Areas = []string{"weather"} },
		"no area":      func(i *OptionsInput) { i.Areas = nil },
		"bad state":    func(i *OptionsInput) { i.Rules[0].State = "maybe" },
	} {
		in := optionsInput()
		mod(&in)
		if err := in.Validate(); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestKnownProduct(t *testing.T) {
	for name, want := range map[string]bool{
		"Azure Database for PostgreSQL Flexible Server": true,
		"Amazon Aurora PostgreSQL":                      true,
		"Micro Focus Enterprise Server":                 true,
		"Ledger Cloud for Azure":                        false,
		"Azure App Service (Web Apps)":                  true,
		"Azure App Service Environment v3":              true,
		"Azure Static Web Apps":                         true,
		"QuantumBase":                                   false,
	} {
		if got := knownProduct(name, ""); got != want {
			t.Errorf("%s: %v", name, got)
		}
	}
}

// TestOptionsTeam runs the whole team with a fake model: scouts, verifier, rules guard, architect, challenger.
func TestOptionsTeam(t *testing.T) {
	in := optionsInput()
	if err := in.Validate(); err != nil {
		t.Fatal(err)
	}
	calls := 0
	fake := func(ctx context.Context, model, system, user string, schema map[string]any, max int64, out any) error {
		calls++
		var js string
		switch {
		case strings.Contains(system, "option scout") && strings.Contains(user, "Area to cover: data"):
			js = `{"options":[
			 {"name":"Azure Database for PostgreSQL Flexible Server","vendor":"Microsoft","kind":"managed database","summary":"Managed PostgreSQL","fit":[{"text":"Small relational schema","facts":["F1","F9"]}],"prerequisites":[],"risks":[],"lifecycle":"supported","licence":"cloud-service","skills":["PostgreSQL"]},
			 {"name":"Amazon Aurora PostgreSQL","vendor":"AWS","kind":"managed database","summary":"Aurora","fit":[{"text":"Relational","facts":["F1"]}],"prerequisites":[],"risks":[],"lifecycle":"supported","licence":"cloud-service","skills":[]},
			 {"name":"QuantumBase","vendor":"Nobody","kind":"database","summary":"Made up","fit":[{"text":"Fits","facts":["F1"]}],"prerequisites":[],"risks":[],"lifecycle":"","licence":"weird","skills":[]},
			 {"name":"Uncited DB","vendor":"X","kind":"database","summary":"","fit":[{"text":"Fits","facts":["F99"]}],"prerequisites":[],"risks":[],"lifecycle":"","licence":"unknown","skills":[]}]}`
		case strings.Contains(system, "option scout"):
			js = `{"options":[{"name":"Azure App Service","vendor":"Microsoft","kind":"platform service","summary":"Web hosting","fit":[{"text":".NET runs well","facts":["F2"]}],"prerequisites":[],"risks":[],"lifecycle":"supported","licence":"cloud-service","skills":[]}]}`
		case strings.Contains(system, "architect"):
			if !strings.Contains(user, "excluded") {
				t.Error("architect did not see the rule check")
			}
			js = `{"blueprints":[
			 {"tier":"Ambitious","name":"Rebuild","approach":"rebuild","options":["O1"],"why":[],"migration":"","effort":"L","costTrend":"unknown","deadline":""},
			 {"tier":"Safe","name":"Replatform","approach":"replatform","options":["O1","O2","O9"],"why":[{"text":"Keeps .NET","facts":["F2"]},{"text":"No fact","facts":[]}],"migration":"Move the database first","effort":"M","costTrend":"lower","deadline":"no deadline given"},
			 {"tier":"Safe","name":"Second safe","approach":"rehost","options":["O1"],"why":[],"migration":"","effort":"S","costTrend":"lower","deadline":""},
			 {"tier":"Balanced","name":"Bad","approach":"teleport","options":["O1"],"why":[],"migration":"","effort":"S","costTrend":"lower","deadline":""}]}`
		case strings.Contains(system, "challenge"):
			js = `{"reviews":[{"blueprint":1,"verdict":"questionable","challenges":["Data migration not planned","",""]},{"blueprint":7,"verdict":"risky","challenges":["x"]}]}`
		case strings.Contains(system, "one level deeper"):
			t.Fatal("explore must not run in a team run")
		default:
			t.Fatalf("unexpected call: %s", system[:40])
		}
		return json.Unmarshal([]byte(js), out)
	}
	out, err := RunOptionsTeam(context.Background(), in, fake)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 4 || out.Requests != 4 {
		t.Fatalf("calls %d requests %d", calls, out.Requests)
	}
	names := map[string]TechOption{}
	for _, o := range out.Options {
		names[o.Name] = o
	}
	if _, ok := names["Uncited DB"]; ok {
		t.Error("an option with no real fact was kept")
	}
	if len(out.Dropped) != 1 {
		t.Errorf("dropped %v", out.Dropped)
	}
	pg := names["Azure Database for PostgreSQL Flexible Server"]
	if pg.Catalogue != "known" || pg.Guard != "preferred" || len(pg.Fit) != 1 || len(pg.Fit[0].Facts) != 1 {
		t.Errorf("postgres option %+v", pg)
	}
	if a := names["Amazon Aurora PostgreSQL"]; a.Guard != "excluded" {
		t.Errorf("AWS rule not applied: %+v", a)
	}
	if q := names["QuantumBase"]; q.Catalogue != "needs confirmation" || q.Licence != "unknown" {
		t.Errorf("unknown product %+v", q)
	}
	if out.Options[0].Guard != "preferred" {
		t.Errorf("preferred options should come first: %s", out.Options[0].Guard)
	}
	if len(out.Blueprints) != 2 || out.Blueprints[0].Tier != "Safe" || out.Blueprints[1].Tier != "Ambitious" {
		t.Fatalf("blueprints should be one per tier, Safe first: %+v", out.Blueprints)
	}
	b := out.Blueprints[0]
	for _, id := range b.Options {
		for _, o := range out.Options {
			if o.ID == id && o.Guard == "excluded" {
				t.Errorf("blueprint uses excluded option %s", o.Name)
			}
		}
	}
	if len(b.Why) != 1 || b.Verdict != "questionable" || len(b.Challenges) != 1 {
		t.Errorf("blueprint %+v", b)
	}
	if len(out.Steps) != 6 {
		t.Errorf("steps %+v", out.Steps)
	}
}

func TestExplore(t *testing.T) {
	in := optionsInput()
	in.Explore = &ExploreTarget{Area: "hosting", Name: "Microsoft Azure", Vendor: "Microsoft"}
	if err := in.Validate(); err != nil {
		t.Fatal(err)
	}
	if len(in.Areas) != 1 || in.Areas[0] != "hosting" {
		t.Fatalf("explore areas %v", in.Areas)
	}
	calls := 0
	fake := func(ctx context.Context, model, system, user string, schema map[string]any, max int64, out any) error {
		calls++
		if !strings.Contains(system, "one level deeper") || !strings.Contains(user, "Microsoft Azure") {
			t.Fatalf("not an explore call: %s", system[:30])
		}
		if calls == 1 { // first answer has nothing usable: the team asks again
			return json.Unmarshal([]byte(`{"options":[{"name":"_placeholder","fit":[]}],"north":[],"south":[]}`), out)
		}
		return json.Unmarshal([]byte(`{"north":[{"name":"Azure Front Door","role":"Entry for browser users","why":[{"text":"Web access","facts":["F2"]}]},{"name":"Amazon CloudFront","role":"CDN","why":[{"text":"x","facts":["F1"]}]},{"name":"","role":"x","why":[]}],
		 "south":[{"name":"Private endpoint","role":"Private path to the database","why":[{"text":"Personal data","facts":["F99"]}]}],
		 "options":[
		 {"name":"Azure App Service","vendor":"Microsoft","kind":"cloud-service","summary":"Web hosting","tier":"Premium v3","compute":{"vcpu":"2","ramGb":"8","storageGb":"250","instances":"2","basis":"guess","note":"about 50 users"},"spec":[{"aspect":"availability","value":"99.95% SLA; zone redundant needs 3 instances"},{"aspect":"sku","value":"P1v3 start"},{"aspect":"sku","value":"dup"},{"aspect":"weird","value":"x"},{"aspect":"limits","value":"  "}],"sizing":"P1v3, 2 instances","resilience":"zone redundant","chooseWhen":"web front end","avoidWhen":"desktop client stays","fit":[{"text":".NET","facts":["F2"]}],"prerequisites":[],"risks":[],"lifecycle":"supported","licence":"cloud-service","skills":[]},
		 {"name":"Azure Virtual Desktop","vendor":"Microsoft","kind":"cloud-service","summary":"Hosts the desktop client","tier":"","sizing":"","resilience":"","chooseWhen":"","avoidWhen":"","compute":{"vcpu":"","ramGb":"","storageGb":"","instances":"","basis":"facts","note":""},"fit":[{"text":"Desktop app","facts":["F2"]}],"prerequisites":[],"risks":[],"lifecycle":"","licence":"subscription","skills":[]},
		 {"name":"Strangler fig","vendor":"","kind":"pattern","summary":"","tier":"","sizing":"","resilience":"","chooseWhen":"","avoidWhen":"","fit":[{"text":"x","facts":["F1"]}],"prerequisites":[],"risks":[],"lifecycle":"","licence":"unknown","skills":[]}]}`), out)
	}
	out, err := RunOptionsTeam(context.Background(), in, fake)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 || out.Requests != 2 || len(out.Options) != 3 || len(out.Blueprints) != 0 {
		t.Fatalf("calls %d options %d blueprints %d", calls, len(out.Options), len(out.Blueprints))
	}
	o := out.Options[0]
	if o.Parent != "Microsoft Azure" || o.Tier != "Premium v3" || o.ID != "X1" || o.Catalogue != "known" {
		t.Errorf("explored option %+v", o)
	}
	if c := o.Compute; c == nil || c.VCPU != "2" || c.RAMGB != "8" || c.Basis != "assumption" {
		t.Errorf("compute %+v", o.Compute)
	}
	if sp := o.Spec; len(sp) != 2 || sp[0].Aspect != "sku" || sp[0].Value != "P1v3 start" || sp[1].Aspect != "availability" {
		t.Errorf("spec should keep known aspects once, in matrix order: %+v", sp)
	}
	if out.Options[1].Compute != nil {
		t.Error("an empty compute size was kept")
	}
	// AWS is excluded by the rules, the nameless one is dropped; a role alone keeps a component.
	if len(out.North) != 1 || out.North[0].Name != "Azure Front Door" || out.North[0].Catalogue != "known" {
		t.Errorf("north %+v", out.North)
	}
	if len(out.South) != 1 || len(out.South[0].Why) != 0 {
		t.Errorf("south %+v", out.South)
	}
	if out.Options[2].Catalogue != "pattern" {
		t.Errorf("a pattern is not a product: %+v", out.Options[2])
	}
	bad := optionsInput()
	bad.Explore = &ExploreTarget{Area: "weather", Name: "x"}
	if err := bad.Validate(); err == nil {
		t.Error("unknown explore area accepted")
	}
}

func TestOptionsModes(t *testing.T) {
	in := optionsInput()
	in.Mode = "weird"
	if err := in.Validate(); err == nil {
		t.Error("unknown project type accepted")
	}
	in = optionsInput()
	if err := in.Validate(); err != nil || in.Mode != "modernize" {
		t.Fatalf("empty mode should mean modernize: %v %q", err, in.Mode)
	}
	opts := []TechOption{{ID: "O1", Name: "Azure App Service", Guard: "allowed"}}
	facts := map[string]bool{"F1": true}
	bps := []Blueprint{
		{Tier: "Safe", Approach: "buy", Options: []string{"O1"}},
		{Tier: "Balanced", Approach: "rebuild", Options: []string{"O1"}},
		{Tier: "Ambitious", Approach: "build", Options: []string{"O1"}},
	}
	got := cleanBlueprints(bps, opts, facts, "new")
	if len(got) != 2 || got[0].Approach != "buy" || got[1].Approach != "build" {
		t.Errorf("a new product keeps only build / buy / partner: %+v", got)
	}
	if got := cleanBlueprints(bps, opts, facts, "modernize"); len(got) != 1 || got[0].Approach != "rebuild" {
		t.Errorf("modernize keeps only the 7 Rs: %+v", got)
	}
	if got := cleanBlueprints(bps, opts, facts, "extend"); len(got) != 3 {
		t.Errorf("extend keeps both: %+v", got)
	}
}

func TestCitedFacts(t *testing.T) {
	facts := map[string]bool{"F2": true, "F3": true, "F7": true}
	for name, c := range map[string]struct {
		r    OptionReason
		want string
	}{
		"plain":       {OptionReason{Facts: []string{"F2"}}, "F2"},
		"lower, zero": {OptionReason{Facts: []string{"f02", "[F3]"}}, "F2,F3"},
		"joined":      {OptionReason{Facts: []string{"F2, F7"}}, "F2,F7"},
		"in text":     {OptionReason{Text: "Fits the users (F3) and scale (F7)"}, "F3,F7"},
		"unknown":     {OptionReason{Facts: []string{"F99"}}, ""},
		"duplicates":  {OptionReason{Facts: []string{"F2", "F2"}}, "F2"},
	} {
		if got := strings.Join(citedFacts(c.r, facts), ","); got != c.want {
			t.Errorf("%s: got %q want %q", name, got, c.want)
		}
	}
}
