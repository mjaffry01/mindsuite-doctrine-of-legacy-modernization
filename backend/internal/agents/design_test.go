package agents

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestDesign(t *testing.T) {
	in := optionsInput()
	in.Design = &DesignTarget{Tier: "Balanced", Name: "Rebuild as a web app", Approach: "rebuild", Options: []string{"Azure App Service", "Azure SQL Database"}}
	if err := in.Validate(); err != nil {
		t.Fatal(err)
	}
	fake := func(ctx context.Context, model, system, user string, schema map[string]any, max int64, out any) error {
		if !strings.Contains(system, "solution designer") || !strings.Contains(user, "Rebuild as a web app") {
			t.Fatalf("not a design call")
		}
		return json.Unmarshal([]byte(`{"summary":"Web app on Azure.","items":[
		 {"section":"database","name":"Azure SQL Database","detail":"General Purpose, zone redundant","reason":"Relational data","facts":["F1","F42"]},
		 {"section":"database","name":"Amazon Aurora","detail":"","reason":"","facts":[]},
		 {"section":"caching","name":"Azure Cache for Redis","detail":"Reference data, 10 min expiry","reason":"no fact","facts":[]},
		 {"section":"pattern","name":"Strangler fig","detail":"At the edge","reason":"Desktop app replaced step by step","facts":["F2"]},
		 {"section":"north","name":"Azure Front Door","detail":"Entry and WAF","reason":"","facts":[]},
		 {"section":"south","name":"Private endpoint","detail":"To the database","reason":"","facts":[]},
		 {"section":"secrets","name":"Azure Key Vault","detail":"Connection strings","reason":"","facts":[]},
		 {"section":"weird","name":"Azure Monitor","detail":"Logs","reason":"","facts":[]},
		 {"section":"other","name":"AWS WAF","detail":"x","reason":"","facts":[]}],
		 "sizing":[{"component":"App","vcpu":"2","ramGb":"8","storageGb":"50","instances":"2","basis":"maybe","note":"50 users"},{"component":"Empty","vcpu":"","ramGb":"","storageGb":"","instances":"","basis":"facts","note":""},{"component":"CDN","vcpu":"0","ramGb":"0","storageGb":"n/a","instances":"0","basis":"facts","note":""}],
		 "gaps":["Peak load","","Data size"]}`), out)
	}
	d, step, err := RunDesign(context.Background(), in, fake)
	if err != nil {
		t.Fatal(err)
	}
	if step.Requests != 1 || !strings.Contains(step.Note, "1 database") {
		t.Errorf("step %+v", step)
	}
	if len(d.Database) != 1 || d.Database[0].Catalogue != "known" || len(d.Database[0].Why) != 1 || len(d.Database[0].Why[0].Facts) != 1 {
		t.Errorf("database %+v (AWS is excluded, unknown fact dropped)", d.Database)
	}
	if len(d.Caching) != 1 || len(d.Caching[0].Why) != 0 {
		t.Errorf("caching %+v", d.Caching)
	}
	if len(d.Patterns) != 1 || d.Patterns[0].Catalogue != "pattern" {
		t.Errorf("patterns %+v", d.Patterns)
	}
	if len(d.Components) != 2 || d.Components[1].Category != "other" {
		t.Errorf("components %+v", d.Components)
	}
	if len(d.Sizing) != 1 || d.Sizing[0].Basis != "assumption" || d.Sizing[0].RAMGB != "8" {
		t.Errorf("sizing %+v", d.Sizing)
	}
	if len(d.Gaps) != 2 || len(d.North) != 1 || len(d.South) != 1 {
		t.Errorf("gaps %v north %v south %v", d.Gaps, d.North, d.South)
	}
	bad := optionsInput()
	bad.Design = &DesignTarget{Approach: "teleport"}
	if err := bad.Validate(); err == nil {
		t.Error("unknown approach accepted")
	}
}
