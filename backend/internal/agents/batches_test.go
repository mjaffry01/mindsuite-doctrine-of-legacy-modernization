package agents

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

type batchTestTeam struct {
	after func()
	calls *int
}

func (b batchTestTeam) ID() string                  { return "test-batches" }
func (b batchTestTeam) Agents() []AgentInfo         { return nil }
func (b batchTestTeam) StepTitles() []string        { return make([]string, 9) }
func (b batchTestTeam) WholeKinds() map[string]bool { return nil }
func (b batchTestTeam) NewState() any               { return nil }
func (b batchTestTeam) Source(*Run) map[string]any  { return nil }
func (b batchTestTeam) Execute(_ context.Context, r *Run) {
	(*b.calls)++
	for _, file := range r.Code.Files {
		r.Cover("test", file)
		id := r.Store.Add(Finding{Agent: "test", Title: "Shared source", Kind: "table-access", Data: map[string]any{"table": "StockLedger"}, Citations: []Citation{Cite(file, 1)}, Status: StatusAccepted})
		r.Store.Add(Finding{Agent: "test", Title: "Derived", Kind: "business-rule", Evidence: []string{id}, Status: StatusAccepted})
	}
	b.after()
}

func TestBatchCheckpointResume(t *testing.T) {
	calls := 0
	ctx, cancel := context.WithCancel(context.Background())
	Register(batchTestTeam{after: cancel, calls: &calls})
	files := map[string][]byte{}
	for i := 1; i <= 3; i++ {
		files[fmt.Sprintf("file%d.cs", i)] = []byte(strings.Repeat("// stock\n", 3200))
	}
	code := LoadFiles("warehouse", files)
	first, err := NewRun(code, Options{Team: "test-batches"}, nil, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	first.Execute(ctx)
	if calls != 1 || first.Batches.Completed != 1 || first.Status != "failed" {
		t.Fatalf("expected one saved batch after interruption: calls=%v, %+v", calls, first.Batches)
	}
	Register(batchTestTeam{after: func() {}, calls: &calls})
	resumed, err := NewRun(code, Options{Team: "test-batches", BatchDir: filepath.Join(first.Dir, "checkpoints")}, nil, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	resumed.Execute(context.Background())
	if calls != 3 || resumed.Batches.Completed != 3 || resumed.Batches.Reused != 1 || resumed.Status != "done" {
		t.Fatalf("expected reuse of one batch: calls=%v, %+v", calls, resumed.Batches)
	}
	linked := resumed.Store.Findings("cross-batch-link")
	if len(linked) != 1 || len(linked[0].Evidence) != 3 {
		t.Fatalf("missing linked batches: %+v", linked)
	}
	for _, rule := range resumed.Store.Findings("business-rule") {
		if len(rule.Evidence) != 1 {
			t.Fatalf("missing evidence: %+v", rule)
		}
		finding, ok := resumed.Store.Get(rule.Evidence[0])
		if !ok || finding.Kind != "table-access" || strings.Split(rule.ID, "-")[0] != strings.Split(finding.ID, "-")[0] {
			t.Fatalf("incorrect linked ID: %+v", rule)
		}
	}
}

func TestBatchCitationOffsetsAndAtomicExports(t *testing.T) {
	code := LoadFiles("warehouse", map[string][]byte{"Warehouse/Stock.vb": []byte(strings.Repeat("' stock\n", 4200) + "End Sub\n" + strings.Repeat("' ledger\n", 1500))})
	batches := planEvidenceBatches(code, "code")
	if len(batches) != 2 || batches[1].files[0].start != 4201 {
		t.Fatalf("unexpected source boundaries: %+v", batches)
	}
	child := batches[1].codebase(code)
	f := child.Files[0]
	store, err := NewStore("")
	if err != nil {
		t.Fatal(err)
	}
	r := &Run{Store: store, coverage: map[string]map[string]bool{}}
	r.addBatchResult(2, batches[1], Result{Run: &Run{Status: "done"}, Findings: []Finding{{
		ID: "test-1", Status: StatusAccepted, Citations: []Citation{Cite(f, 2)}, Detail: f.Path + ":2. " + f.Path + ":2:",
	}}}, map[string][]Finding{})
	got, _ := store.Get("b2-test-1")
	if got.Citations[0].Line != 4203 || strings.Count(got.Detail, f.Path+":4203") != 2 {
		t.Fatalf("expected original citation locations: %+v", got)
	}
	export := LoadFiles("jobs", map[string][]byte{"jobs.json": []byte("[\n" + strings.Repeat("  {},\n", 6000) + "  {}\n]")})
	plan := planEvidenceBatches(export, "batch")
	if len(plan) != 1 || plan[0].lines <= batchTargetLines {
		t.Fatalf("structured export must remain intact: %+v", plan)
	}
}
