package agents

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func sessIn() SessionPolicyInput {
	return SessionPolicyInput{Project: "Food ordering", Kind: "new",
		Platform:  []string{"Cloud: AWS"},
		Targets:   []string{"Availability: 99.9%", "Regulations: PCI DSS, Digital Personal Data Protection Act 2023"},
		Layers:    []DiagramLayer{{ID: "cache", Label: "Cache", Design: "Amazon ElastiCache (Valkey or Redis), 2 GB"}},
		UserTypes: []SessionUserType{{Name: "Customer"}, {Name: "Restaurant staff", Staff: true}},
		Channels:  []string{"web", "mobile"},
	}
}

func TestSessionPolicyValidate(t *testing.T) {
	if err := sessIn().Validate(); err != nil {
		t.Fatal(err)
	}
	for name, mut := range map[string]func(*SessionPolicyInput){
		"kind":       func(in *SessionPolicyInput) { in.Kind = "x" },
		"no users":   func(in *SessionPolicyInput) { in.UserTypes = nil },
		"duplicate":  func(in *SessionPolicyInput) { in.UserTypes = append(in.UserTypes, SessionUserType{Name: "customer"}) },
		"empty line": func(in *SessionPolicyInput) { in.Channels = []string{""} },
		"huge layer": func(in *SessionPolicyInput) { in.Layers[0].Design = strings.Repeat("x", 1600) },
	} {
		in := sessIn()
		mut(&in)
		if in.Validate() == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestSessionPolicyVerify(t *testing.T) {
	got := sessIn().verify(SessionPolicyOutput{
		Cookies: []CookieRule{
			{Category: "necessary", Purpose: "Session id", Consent: "required", Attributes: "Secure; HttpOnly; SameSite=Lax"},
			{Category: "marketing", Purpose: "Campaign attribution", Consent: "not-required"},
			{Category: "tracking", Purpose: "x", Consent: "required"},
			{Category: "analytics", Purpose: "Usage on Azure Functions", Consent: "required"},
		},
		Sessions: []SessionRule{
			{UserType: "customer", Store: "token", MFA: "optional", IdleTimeout: "30 minutes", AbsoluteLifetime: "30 days"},
			{UserType: "Restaurant staff", Store: "server", MFA: "optional", IdleTimeout: "15 minutes", AbsoluteLifetime: "12 hours"},
			{UserType: "Admin", Store: "server", MFA: "required", IdleTimeout: "10 minutes", AbsoluteLifetime: "8 hours"},
			{UserType: "Customer", Store: "cookie", MFA: "optional", IdleTimeout: "1", AbsoluteLifetime: "1"},
		},
		Conflicts: []string{"Sessions live in one cache node"},
	})
	if len(got.Cookies) != 2 || got.Cookies[0].Consent != "not-required" || got.Cookies[1].Consent != "required" {
		t.Fatalf("cookies: %+v", got.Cookies)
	}
	if len(got.Sessions) != 2 || got.Sessions[0].UserType != "Customer" {
		t.Fatalf("sessions: %+v", got.Sessions)
	}
	// marketing consent corrected + staff MFA + the model's own conflict
	if len(got.Conflicts) != 3 || got.Dropped != 4 {
		t.Fatalf("conflicts %v dropped %d", got.Conflicts, got.Dropped)
	}
}

func TestSessionPolicyRetriesOnce(t *testing.T) {
	calls := 0
	fake := func(_ context.Context, model, system, user string, _ map[string]any, _ int64, out any) error {
		calls++
		if !strings.Contains(user, "Restaurant staff") || model != ModelStrong {
			t.Fatalf("bad call: %s", model)
		}
		body := `{"summary":"","cookies":[],"sessions":[{"userType":"Robot","store":"server","mfa":"required","idleTimeout":"1","absoluteLifetime":"1"}],"conflicts":[],"questions":[]}`
		if calls == 2 {
			if !strings.Contains(system, "no valid session rule") {
				t.Fatal("retry prompt missing")
			}
			body = `{"summary":"ok","cookies":[],"sessions":[{"userType":"Customer","store":"token","mfa":"optional","idleTimeout":"30 minutes","absoluteLifetime":"30 days"}],"conflicts":[],"questions":[]}`
		}
		return json.Unmarshal([]byte(body), out)
	}
	out, err := RunSessionPolicy(context.Background(), sessIn(), fake)
	if err != nil || calls != 2 || out.Attempts != 2 || len(out.Sessions) != 1 || out.Model == "" {
		t.Fatalf("calls %d out %+v err %v", calls, out, err)
	}
}
