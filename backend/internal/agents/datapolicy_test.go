package agents

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func policyIn() DataPolicyInput {
	return DataPolicyInput{Project: "Food ordering", Kind: "new",
		Platform: []string{"Cloud: AWS", "Database: PostgreSQL (managed)"},
		Targets:  []string{"Availability: 99.9%", "RPO: 15 minutes"},
		Layers:   []DiagramLayer{{ID: "database", Label: "Database", Design: "Amazon RDS or Aurora PostgreSQL"}, {ID: "cache", Label: "Cache", Design: "Amazon ElastiCache (Valkey or Redis)"}},
		Objects:  []PolicyObject{{Name: "Stock", Writers: []string{"Ordering", "Warehouse"}}, {Name: "Payment", Sensitivity: "financial"}},
		Journeys: []PolicyJourney{{Journey: "A customer pays for the last item in stock", Measure: "Exactly one paid order"}},
	}
}

func TestDataPolicyValidate(t *testing.T) {
	if err := policyIn().Validate(); err != nil {
		t.Fatal(err)
	}
	for name, mut := range map[string]func(*DataPolicyInput){
		"kind":       func(in *DataPolicyInput) { in.Kind = "x" },
		"no objects": func(in *DataPolicyInput) { in.Objects = nil },
		"duplicate":  func(in *DataPolicyInput) { in.Objects = append(in.Objects, PolicyObject{Name: "stock"}) },
		"long name":  func(in *DataPolicyInput) { in.Objects[0].Name = strings.Repeat("x", 61) },
		"empty line": func(in *DataPolicyInput) { in.Targets = []string{" "} },
		"huge layer": func(in *DataPolicyInput) { in.Layers[0].Design = strings.Repeat("x", 1600) },
		"no journey": func(in *DataPolicyInput) { in.Journeys = []PolicyJourney{{Journey: ""}} },
	} {
		in := policyIn()
		mut(&in)
		if in.Validate() == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestDataPolicyVerify(t *testing.T) {
	got := policyIn().verify(DataPolicyOutput{
		Summary: "Stock must be exact at checkout.",
		Data: []DataRule{
			{Object: "stock", Owner: "Operations", AuthoritativeWriter: "Ordering service", Consistency: "strong"},
			{Object: "Stock", Owner: "Ops", AuthoritativeWriter: "x", Consistency: "strong"},                                 // duplicate
			{Object: "Menu", Owner: "Ops", AuthoritativeWriter: "x", Consistency: "eventual"},                                // not sent
			{Object: "Payment", Owner: "Finance", AuthoritativeWriter: "Payments", Consistency: "sometimes"},                 // bad enum
			{Object: "Payment", Owner: "Finance", AuthoritativeWriter: "Payments on Azure Functions", Consistency: "strong"}, // unstated product
		},
		Cache: []CacheRule{
			{Object: "Stock", Cacheable: "no", Why: "must be exact"},
			{Object: "Payment", Cacheable: "maybe"},
			{Object: "Payment", Cacheable: "short", Invalidation: "on write via ElastiCache"},
		},
		Conflicts: []string{"Stock is cached while checkout needs exactly one order", ""},
		Questions: []string{"How long are payments kept?"},
	})
	if len(got.Data) != 1 || got.Data[0].Object != "Stock" {
		t.Fatalf("data: %+v", got.Data)
	}
	if len(got.Cache) != 2 || got.Cache[1].Cacheable != "short" {
		t.Fatalf("cache: %+v", got.Cache)
	}
	if len(got.Conflicts) != 1 || len(got.Questions) != 1 || got.Dropped != 5 {
		t.Fatalf("conflicts %v questions %v dropped %d", got.Conflicts, got.Questions, got.Dropped)
	}
}

func TestDataPolicyRetriesOnce(t *testing.T) {
	calls := 0
	fake := func(_ context.Context, model, system, user string, _ map[string]any, _ int64, out any) error {
		calls++
		if !strings.Contains(user, "Food ordering") || model != ModelStrong {
			t.Fatalf("bad call: %s", model)
		}
		body := `{"summary":"","data":[{"object":"Cutlery","owner":"x","authoritativeWriter":"y","consistency":"strong"}],"cache":[],"conflicts":[],"questions":[]}`
		if calls == 2 {
			if !strings.Contains(system, "no valid data rule") {
				t.Fatal("retry prompt missing")
			}
			body = `{"summary":"ok","data":[{"object":"Stock","owner":"Operations","authoritativeWriter":"Ordering","consistency":"strong"}],"cache":[{"object":"Stock","cacheable":"no"}],"conflicts":[],"questions":[]}`
		}
		return json.Unmarshal([]byte(body), out)
	}
	out, err := RunDataPolicy(context.Background(), policyIn(), fake)
	if err != nil || calls != 2 || out.Attempts != 2 || len(out.Data) != 1 || out.Model == "" {
		t.Fatalf("calls %d out %+v err %v", calls, out, err)
	}
}
