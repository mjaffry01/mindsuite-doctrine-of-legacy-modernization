package agents

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func diagIn() DiagramInput {
	return DiagramInput{Project: "Equipment monitoring", Kind: "new",
		Platform: []string{"Tier: Balanced", "Cloud: AWS"},
		Targets:  []string{"Peak: 27 requests/s", "Availability: 99.9%"},
		Layers: []DiagramLayer{{ID: "database", Label: "Database", Design: "Amazon RDS or Aurora PostgreSQL · 2 vCPU"},
			{ID: "messaging", Label: "Messaging", Design: "Queue: Amazon SQS"}},
		Capabilities: []DiagramCap{{Name: "Pay an invoice", Route: "Buy", RunsOn: "Vendor product"}},
		DraftNodes:   []DiagramNode{{ID: "app", Zone: "app", Title: "App"}, {ID: "db", Zone: "data", Title: "Database"}},
	}
}

func TestDiagramValidate(t *testing.T) {
	if err := diagIn().Validate(); err != nil {
		t.Fatal(err)
	}
	bad := diagIn()
	bad.Kind = "x"
	if bad.Validate() == nil {
		t.Error("accepted unknown kind")
	}
	bad = diagIn()
	bad.Layers = nil
	if bad.Validate() == nil {
		t.Error("accepted no layers")
	}
	bad = diagIn()
	bad.DraftNodes = []DiagramNode{{ID: "a", Zone: "moon", Title: "x"}}
	if bad.Validate() == nil {
		t.Error("accepted bad zone")
	}
	bad = diagIn()
	bad.Layers[0].Design = strings.Repeat("x", 1600)
	if bad.Validate() == nil {
		t.Error("accepted huge layer")
	}
}

func TestDiagramVerify(t *testing.T) {
	in := diagIn()
	got := in.verify(DiagramOutput{
		Summary: "A modular app on AWS.",
		Nodes: []DiagramNode{
			{ID: "app", Zone: "app", Title: "App runtime"},
			{ID: "db", Zone: "data", Title: "Database", Sub: "Amazon RDS or Aurora PostgreSQL"},
			{ID: "db", Zone: "data", Title: "Duplicate"},
			{ID: "Bad Id", Zone: "data", Title: "x"},
			{ID: "q", Zone: "space", Title: "Queue"},
			{ID: "pay", Zone: "ext", Title: "Stripe checkout"},
			{ID: "mq", Zone: "data", Title: "Queue", Sub: "Amazon SQS"},
		},
		Edges: []DiagramEdge{{From: "app", To: "db", Label: "SQL over TLS", Kind: "data"}, {From: "app", To: "pay", Label: "API"},
			{From: "app", To: "app", Label: "self"}, {From: "app", To: "mq", Label: "jobs", Kind: "weird"}, {From: "app", To: "db", Label: "dup"}},
		Missing:   []DiagramMissing{{Component: "Backup vault", Why: "99.9% needs tested restores"}, {Component: "", Why: "x"}},
		Conflicts: []string{"Payments bought and built"},
	})
	if len(got.Nodes) != 3 || got.Nodes[2].ID != "mq" {
		t.Fatalf("nodes: %+v", got.Nodes)
	}
	if len(got.Edges) != 2 || got.Edges[1].Kind != "request" {
		t.Fatalf("edges: %+v", got.Edges)
	}
	if len(got.Missing) != 1 || len(got.Conflicts) != 1 || got.Dropped < 6 {
		t.Fatalf("missing %v conflicts %v dropped %d", got.Missing, got.Conflicts, got.Dropped)
	}
}

func TestDiagramRetriesOnce(t *testing.T) {
	calls := 0
	fake := func(_ context.Context, model, system, user string, _ map[string]any, _ int64, out any) error {
		calls++
		if !strings.Contains(user, "Equipment monitoring") || model != ModelStrong {
			t.Fatalf("bad call: %s %s", model, user[:40])
		}
		body := `{"summary":"","nodes":[{"id":"app","zone":"app","title":"App"}],"edges":[],"missing":[],"conflicts":[]}`
		if calls == 2 {
			if !strings.Contains(system, "too few") {
				t.Fatal("retry prompt missing")
			}
			body = `{"summary":"ok","nodes":[{"id":"app","zone":"app","title":"App"},{"id":"db","zone":"data","title":"Database"},{"id":"mq","zone":"data","title":"Queue","sub":"Amazon SQS"}],"edges":[{"from":"app","to":"db","label":"SQL","kind":"data"}],"missing":[],"conflicts":[]}`
		}
		return json.Unmarshal([]byte(body), out)
	}
	out, err := RunDesignDiagram(context.Background(), diagIn(), fake)
	if err != nil || calls != 2 || out.Attempts != 2 || len(out.Nodes) != 3 || out.Model == "" {
		t.Fatalf("calls %d out %+v err %v", calls, out, err)
	}
}
