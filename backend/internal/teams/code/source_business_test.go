package code

import (
	"context"
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
