package code

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mjaffry01/mindsuite-doctrine-of-legacy-modernization/backend/internal/agents"
)

func TestSourceBusinessWithoutModel(t *testing.T) {
	input := agents.LoadFiles("inventory", map[string][]byte{
		"Inventory/Forms/ProductForm.vb":      []byte("Public Class ProductForm\n    productRepository.AddProduct(item)\n    orderRepository.CancelOrder(order)\nEnd Class"),
		"Inventory/Data/ProductRepository.vb": []byte("Public Class ProductRepository\nPublic Sub AddProduct(item As Product)\nEnd Sub\nEnd Class"),
		"Scripts/jquery-3.js":                 []byte(strings.Repeat("if (x > 2) { return; }\n", 1000)),
	})
	run, err := agents.NewRun(input, agents.Options{}, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	run.Execute(context.Background())
	result := run.Snapshot()
	for _, read := range result.Reading {
		if strings.Contains(read.File, "jquery") {
			t.Fatalf("selected bundled code: %s", read.File)
		}
	}
	purpose := run.Store.Findings("purpose")
	if len(purpose) != 1 || purpose[0].Basis != BasisInferred || purpose[0].Status != StatusAccepted ||
		len(purpose[0].Citations) != 2 || !strings.Contains(purpose[0].Detail, "product inventory") ||
		strings.Contains(result.Report, "Likely business use: orders") || len(result.Sent) != 0 {
		t.Fatalf("unexpected source-based result: %+v", purpose)
	}
	if !strings.Contains(result.Report, "Likely business use: product inventory") {
		t.Fatal("business purpose did not reach the written report")
	}
}

func TestCodeReviewUsesInterviewAndTracesBusinessRules(t *testing.T) {
	input := agents.LoadFiles("inventory", map[string][]byte{
		"Inventory/Inventory.vbproj":     []byte("<Project />"),
		"Inventory/Forms/ProductForm.vb": []byte("Public Class ProductForm\n productService.AddProduct(item)\n productService.ReceiveStock(item)\nEnd Class"),
		"Inventory/ProductService.vb":    []byte("Public Class ProductService\nPublic Sub AddProduct(item As Product)\n ctx.SaveChanges()\nEnd Sub\nPublic Sub ReceiveStock(item As Product)\n If item.Quantity <= 0 Then\n  Exit Sub\n End If\n stockRepository.SaveStock(item)\nEnd Sub\nEnd Class"),
		"Inventory/StockRepository.vb":   []byte("Public Class StockRepository\nPublic Sub SaveStock(item As Product)\n Try\n  cmd.CommandText = \"INSERT INTO StockLedger (SKU, Quantity) VALUES (@sku, @qty)\"\n  cmd.ExecuteNonQuery()\n Catch\n  transaction.Rollback()\n End Try\nEnd Sub\nEnd Class"),
	})
	run, err := agents.NewRun(input, agents.Options{Interview: &agents.InterviewFocus{
		Problem: "Receiving discrepancies", Success: "Accurate stock counts", Deadline: "2027-02-01",
	}}, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	run.Execute(context.Background())
	report := run.Report
	for _, text := range []string{"## Interview priorities", "Interview deadline: 2027-02-01", "## Workflow traces", "Further service or repository call", "StockRepository.vb:5", "Observed condition: If item.Quantity <= 0 Then"} {
		if !strings.Contains(report, text) {
			t.Errorf("report did not contain %q", text)
		}
	}
	focus, _, _ := strings.Cut(report, "## Workflow traces")
	if !strings.Contains(focus, "Workflow trace: receive product inventory") || strings.Contains(focus, "Receiving discrepancies") {
		t.Fatal("report should prioritize receiving without copying the interview problem into the output")
	}
	_, operating, found := strings.Cut(report, "## How the business operates")
	if !found {
		t.Fatal("missing operating report")
	}
	operating, _, _ = strings.Cut(operating, "## Workflow traces")
	for _, want := range []string{
		"Trigger — Product Form starts this operation.",
		"Decision — When quantity is at or below 0",
		"Data change — This path contains a write statement targeting STOCKLEDGER.",
		"Exception — A path calls for a transaction rollback.",
		"Who performs or approves this operation in practice?",
		"StockRepository.vb:4",
	} {
		if !strings.Contains(operating, want) {
			t.Errorf("missing operating evidence %q", want)
		}
	}
	traces := run.Store.Findings("workflow-trace")
	if len(traces) != 2 {
		t.Fatalf("wanted two independent traces, got %d", len(traces))
	}
	for _, tr := range traces {
		if tr.Status != StatusAccepted || len(tr.Citations) < 3 {
			t.Fatalf("uncited trace: %+v", tr)
		}
	}
}

func TestWorkflowConnectsSourceAcrossBatches(t *testing.T) {
	input := agents.LoadFiles("stock", map[string][]byte{
		"Stock/Stock.vbproj":       []byte("<Project/>"),
		"Stock/Forms/StockForm.vb": []byte("Public Class StockForm\n stockService.ReceiveStock(item)\n" + strings.Repeat("' comment\n", 3000) + "End Class"),
		"Stock/StockService.vb":    []byte("Public Class StockService\n Public Sub ReceiveStock(item As Item)\n ctx.SaveChanges()\n End Sub\n" + strings.Repeat("' comment\n", 3000) + "End Class"),
	})
	run, err := agents.NewRun(input, agents.Options{}, nil, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	run.Execute(context.Background())
	if run.Batches == nil || run.Batches.Completed != 2 || !strings.Contains(run.Report, "### receive product inventory (across batches)") {
		t.Fatalf("cross-batch workflow was not included: %+v", run.Batches)
	}
	traces := run.Store.Findings("workflow-trace")
	if len(traces) != 1 || traces[0].Status != StatusAccepted || len(traces[0].Citations) < 3 {
		t.Fatalf("expected a single supported cross-batch trace: %+v", traces)
	}
}

func TestOperatingFlowRequiresEvidenceForEveryStep(t *testing.T) {
	var flow domainOperatingFlow
	data := []byte(`{"name":"Accept a receipt","steps":[{"kind":"Trigger","text":"A receipt is submitted","evidence":["int-1"]},{"kind":"Decision","text":"Check quantity","evidence":["rul-1"]}],"questions":["Who approves it?"]}`)
	if err := json.Unmarshal(data, &flow); err != nil {
		t.Fatal(err)
	}
	if _, _, ok := groundedOperation(flow, map[string]bool{"int-1": true}); ok {
		t.Fatal("accepted an operation with an unrecognized evidence ID")
	}
	result, ids, ok := groundedOperation(flow, map[string]bool{"int-1": true, "rul-1": true})
	if !ok || len(ids) != 2 || result.Steps[1].Refs[0] != 1 {
		t.Fatalf("did not retain step-specific evidence positions: %+v %v", result, ids)
	}
}

func TestDeclaredLegacyJobOrderInOperatingReport(t *testing.T) {
	input := agents.LoadFiles("warehouse", map[string][]byte{
		"Warehouse/NIGHTLY.jcl": []byte("//NIGHTLY JOB\n//IMPORT EXEC PGM=LOADREC\n//POST EXEC PGM=POSTLEDG\n"),
	})
	run, err := agents.NewRun(input, agents.Options{}, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	run.Execute(context.Background())
	first := strings.Index(run.Report, "Job step IMPORT runs LOADREC")
	second := strings.Index(run.Report, "Job step POST runs POSTLEDG")
	if first < 0 || second <= first || !strings.Contains(run.Report, "Confirm the schedule and any conditional execution") {
		t.Fatalf("missing verified job sequence or operational uncertainty: %s", run.Report)
	}
}
