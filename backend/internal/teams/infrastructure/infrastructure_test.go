package infrastructure

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mjaffry01/mindsuite-doctrine-of-legacy-modernization/backend/internal/agents"
)

// TestInfrastructureRun runs the team (without a model) on testdata/sample: an RVTools export,
// three connection lists (ss and Get-NetTCPConnection), kubectl JSON, an Azure inventory and Terraform.
func TestInfrastructureRun(t *testing.T) {
	code, err := agents.LoadDir("testdata/sample")
	if err != nil {
		t.Fatal(err)
	}
	r, err := agents.NewRun(code, agents.Options{Team: "infrastructure"}, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	r.Execute(context.Background())
	snap := r.Snapshot()
	if snap.Run.Status != "done" {
		t.Fatalf("status %s: %s", snap.Run.Status, snap.Run.Error)
	}
	if len(snap.Agents) != 13 || snap.Agents[1].ID != "est" {
		t.Errorf("infrastructure team expected, got %d agents", len(snap.Agents))
	}
	byKind := map[string][]agents.Finding{}
	for _, f := range snap.Findings {
		byKind[f.Kind] = append(byKind[f.Kind], f)
		if f.Status == StatusProposed {
			t.Errorf("not verified: %s %s", f.ID, f.Title)
		}
		if f.Status == StatusRejected {
			t.Errorf("rejected: %s %s — %s", f.ID, f.Title, f.Verdict)
		}
	}
	has := func(kind, sub string) {
		t.Helper()
		for _, f := range byKind[kind] {
			if strings.Contains(f.Title+" "+f.Detail, sub) {
				return
			}
		}
		titles := []string{}
		for _, f := range byKind[kind] {
			titles = append(titles, f.Title)
		}
		t.Errorf("missing %s containing %q; have %v", kind, sub, titles)
	}
	has("estate", "7 hosts (6 running)")
	has("hypervisor", "6.7.3")
	has("host", "db-prod-01: Oracle Linux 7.9, 16 vCPU, 128 GB")
	has("os-eol", "Red Hat Enterprise Linux 7.9 on 2 host(s)")
	has("os-eol", "Windows Server 2012 R2")
	has("os-eol", "Windows Server 2008 R2")
	has("platform-eol", "out of support")
	has("powered-off", "old-report-01")
	has("net-connection", "app-prod-01 → db-prod-01:1521 (Oracle Database)")
	has("net-connection", "web-prod-01 → app-prod-01:8080")
	has("external-endpoint", "203.0.113.10 (22 SSH / SFTP)")
	has("external-endpoint", "10.0.1.60 (6379 Redis)")
	has("hub-host", "db-prod-01 is used by 3 hosts")
	has("move-group", "app-prod-01, app-prod-02, batch-prd-01, db-prod-01, web-prod-01")
	has("no-connections", "mon-01")
	has("plaintext-protocol", "Telnet")
	has("plaintext-protocol", "AJP")
	has("single-point-of-failure", "Only one database server: db-prod-01")
	has("workload", "Deployment harvest/grower-portal")
	has("image-latest", "harvest/grower-portal")
	has("single-replica", "harvest/grower-portal")
	has("no-limits", "harvest/grower-portal")
	has("k8s-exposed", "harvest/grower-portal-svc")
	has("image-eol", "openjdk:8-jre")
	has("cloud-summary", "3 cloud resources")
	has("public-endpoint", "harvest-pip")
	has("iac-resource", "azurerm_kubernetes_cluster")
	has("secret", "main.tf line 10")
	has("host-strategy", "Retire: 1 host(s)")
	has("host-strategy", "Replatform")
	has("host-strategy", "Exclude: 1 host(s)")
	has("migration-order", "Wave 1")
	has("migration-approach", "")
	has("target-sizing", "vCPU")
	for _, f := range byKind["single-point-of-failure"] {
		if strings.Contains(f.Title, "application") {
			t.Errorf("two application servers are not a single point of failure: %s", f.Title)
		}
	}
	b, _ := json.Marshal(snap)
	if strings.Contains(string(b), "Sup3rS3cret") || strings.Contains(r.Report, "Sup3rS3cret") {
		t.Error("the Terraform password leaked into the result")
	}
	if snap.Run.Steps[4].Status != "skipped" {
		t.Errorf("step 5 should be skipped without a model, got %s", snap.Run.Steps[4].Status)
	}
	if !strings.Contains(r.Report, "# Infrastructure analysis report") || !strings.Contains(r.Report, "Dependencies and move groups") {
		t.Error("report incomplete")
	}
}
