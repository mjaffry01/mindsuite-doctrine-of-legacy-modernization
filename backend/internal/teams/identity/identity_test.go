package identity

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mjaffry01/mindsuite-doctrine-of-legacy-modernization/backend/internal/agents"
)

// TestIdentityRun runs the team (without a model) on testdata/sample: an AD user export, AD group members,
// the Entra MFA registration report, Conditional Access policies, an Okta app list, openssl certificate
// output and a configuration file with a bind password.
func TestIdentityRun(t *testing.T) {
	code, err := agents.LoadDir("testdata/sample")
	if err != nil {
		t.Fatal(err)
	}
	r, err := agents.NewRun(code, agents.Options{Team: "identity"}, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	r.Execute(context.Background())
	snap := r.Snapshot()
	if snap.Run.Status != "done" {
		t.Fatalf("status %s: %s", snap.Run.Status, snap.Run.Error)
	}
	if len(snap.Agents) != 12 || snap.Agents[1].ID != "dir" {
		t.Errorf("identity team expected, got %d agents", len(snap.Agents))
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
			titles = append(titles, f.Title+" | "+f.Detail)
		}
		t.Errorf("missing %s containing %q; have %v", kind, sub, titles)
	}
	lacks := func(kind, sub string) {
		t.Helper()
		for _, f := range byKind[kind] {
			if strings.Contains(f.Title+" "+f.Detail, sub) {
				t.Errorf("%s should not contain %q: %s", kind, sub, f.Detail)
			}
		}
	}
	has("identity-estate", "8 accounts (7 enabled)")
	has("identity-provider", "Active Directory")
	has("identity-provider", "Okta")
	has("group", "Group HL-Admins: 2 member(s), 1 nested group(s)")
	has("group", "Group HL-Users: 5 member(s)")
	has("inactive-accounts", "bwilson")
	has("inactive-accounts", "mlee")
	lacks("inactive-accounts", "oldtemp") // disabled already
	lacks("inactive-accounts", "jdoe")
	has("password-never-expires", "svc_settlement")
	has("service-accounts", "svc_settlement")
	has("shared-accounts", "admin")
	has("admin-group", "Administrator group Domain Admins: 1 member(s)")
	has("admin-group", "HL-Admins")
	has("service-admin", "svc_settlement")
	has("inactive-admin", "mlee")
	has("admin-no-mfa", "kpatel")
	has("mfa-coverage", "MFA registered for 2 of 4 users (50%)")
	has("no-mfa", "asmith")
	has("policy-not-enforced", "Block legacy authentication (enabledForReportingButNotEnforced)")
	has("policy-not-enforced", "Require compliant device (disabled)")
	has("legacy-auth-open", "No enabled policy blocks legacy authentication")
	has("empty-groups", "HL-Reports")
	has("role-model", "HL-Users (5)")
	has("certificate-issue", "portal.harvest.example: expired on 10 Jan 2024")
	has("certificate-issue", "mq.harvest.local: weak 1024-bit key")
	has("certificate-issue", "sha1WithRSAEncryption")
	has("certificate-issue", "self-signed")
	has("idp-app", "HarvestLink Portal (SAML_2_0")
	has("secret", "ldap.properties line 2")
	has("clean-up", "inactive")
	has("service-identity", "1 service account")
	has("migration-approach", "existing cloud identity provider (Microsoft Entra ID or Okta)")
	has("migration-order", "clean up, then roles")
	b, _ := json.Marshal(snap)
	if strings.Contains(string(b), "B1ndPassw0rd") || strings.Contains(r.Report, "B1ndPassw0rd") {
		t.Error("the LDAP bind password leaked into the result")
	}
	if snap.Run.Steps[4].Status != "skipped" {
		t.Errorf("step 5 should be skipped without a model, got %s", snap.Run.Steps[4].Status)
	}
	if !strings.Contains(r.Report, "# Identity & access analysis report") || !strings.Contains(r.Report, "Privileged access") {
		t.Error("report incomplete")
	}
}
