package network

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mjaffry01/mindsuite-doctrine-of-legacy-modernization/backend/internal/agents"
)

// TestNetworkRun runs the team (without a model) on testdata/sample (made by testdata/make_sample.py).
func TestNetworkRun(t *testing.T) {
	code, err := agents.LoadDir("testdata/sample")
	if err != nil {
		t.Fatal(err)
	}
	r, err := agents.NewRun(code, agents.Options{Team: "network"}, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	r.Execute(context.Background())
	snap := r.Snapshot()
	if snap.Run.Status != "done" {
		t.Fatalf("status %s: %s", snap.Run.Status, snap.Run.Error)
	}
	if len(snap.Agents) != 11 || snap.Agents[1].ID != "nrd" {
		t.Errorf("network team expected, got %d agents", len(snap.Agents))
	}
	byKind := map[string][]agents.Finding{}
	for _, f := range snap.Findings {
		byKind[f.Kind] = append(byKind[f.Kind], f)
		if f.Status == StatusProposed || f.Status == StatusRejected {
			t.Errorf("%s: %s %s — %s", f.Status, f.ID, f.Title, f.Verdict)
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
			titles = append(titles, f.Title+" | "+f.Detail)
		}
		t.Errorf("missing %s containing %q; have %v", kind, sub, titles)
	}
	has("edge-estate", "6 entry points, 4 DNS records, 2 certificates, 4 partner connections, 5 firewall rules")
	has("entry-point", "forwards to 10.0.1.11:8080, 10.0.1.12:8080")
	has("entry-point", "haproxy api_front")
	has("entry-point", "10.0.1.31:9000, 10.0.1.32:9000")
	has("entry-point", "ingress: harvest/grower-portal")
	has("entry-point", "CDN: harvest-fd")
	has("plain-http", "reports.harvest.local (8081)")
	has("old-tls", "portal.harvest.example (TLSv1 TLSv1.1 TLSv1.2)")
	has("single-backend", "reports.harvest.local → 10.0.1.21:8080")
	has("cdn", "harvest-fd.azurefd.net")
	has("cdn", "cloudfront.net")
	has("long-ttl", "portal.harvest.example (TTL 86400 s)")
	has("certificate-issue", "portal.harvest.example: expired on 10 Jan 2024")
	has("partner", "Partner Bank: SFTP sftp.partnerbank.example:22")
	has("partner-unencrypted", "Old Haulier (FTP)")
	has("partner-no-owner", "Grain Co-op")
	has("firewall-summary", "5 firewall rules, 4 allowing traffic")
	has("wide-rule", "legacy-all")
	has("exposed-admin-port", "rdp-admin")
	has("rule-no-description", "rdp-admin")
	has("secret", "ipsec.conf line 5")
	has("migration-order", "tell the 4 partner(s)")
	has("migration-order", "lower the TTL")
	has("migration-approach", "switch with DNS")
	for _, f := range byKind["old-tls"] {
		if strings.Contains(f.Detail, "api_front") || strings.Contains(f.Detail, "harvest-fd") {
			t.Errorf("TLS 1.2-only entry points flagged as old: %s", f.Detail)
		}
	}
	b, _ := json.Marshal(snap)
	if strings.Contains(string(b), "RegionVpnKey2026") || strings.Contains(r.Report, "RegionVpnKey2026") {
		t.Error("the VPN key leaked into the result")
	}
	if snap.Run.Steps[4].Status != "skipped" {
		t.Errorf("step 5 should be skipped without a model, got %s", snap.Run.Steps[4].Status)
	}
}
