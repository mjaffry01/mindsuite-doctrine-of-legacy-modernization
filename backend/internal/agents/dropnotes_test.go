package agents

import (
	"strings"
	"testing"
)

func TestPolicyDropNotes(t *testing.T) {
	d := policyIn().verify(DataPolicyOutput{
		Data: []DataRule{
			{Object: "Menu", Owner: "Ops", AuthoritativeWriter: "x", Consistency: "eventual"},
			{Object: "Payment", Owner: "Finance", AuthoritativeWriter: "Payments on Azure Functions", Consistency: "strong"},
		},
		Questions: []string{"Should we move to Azure Functions?"},
	})
	notes := strings.Join(d.DropNotes, "\n")
	for _, want := range []string{"data rule for “Menu”: this business object was not sent", "data rule for Payment: names azure functions", "question 1: names azure functions"} {
		if !strings.Contains(notes, want) {
			t.Errorf("missing %q in\n%s", want, notes)
		}
	}
	if d.Dropped != len(d.DropNotes) {
		t.Errorf("dropped %d but %d notes", d.Dropped, len(d.DropNotes))
	}

	s := sessIn().verify(SessionPolicyOutput{
		Cookies: []CookieRule{
			{Category: "analytics", Purpose: "Usage measured with Azure Monitor", Consent: "required"},
			{Category: "tracking", Purpose: "x", Consent: "required"},
		},
		Sessions: []SessionRule{{UserType: "Robot", Store: "server", MFA: "required", IdleTimeout: "1", AbsoluteLifetime: "1"}},
	})
	notes = strings.Join(s.DropNotes, "\n")
	for _, want := range []string{"cookie rule “analytics”: names azure monitor", "cookie rule “tracking”: not one of", "session rule for “Robot”: this user type was not sent"} {
		if !strings.Contains(notes, want) {
			t.Errorf("missing %q in\n%s", want, notes)
		}
	}
	// Reasons never repeat the dropped text itself beyond the object name.
	if strings.Contains(notes, "Usage measured") {
		t.Errorf("note repeats dropped content: %s", notes)
	}
}
