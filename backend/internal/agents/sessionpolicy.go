package agents

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
)

// Session policy agent: reads the design summary (platform, targets including regulations and availability,
// the identity, cache, security and recovery layers), the user types (customer or staff) and the channels
// (web, mobile app, API) and drafts a cookie policy (by category, with consent and attributes) and a session
// policy per user type (store, timeouts, re-authentication, MFA, sign-out everywhere, rotation, concurrent
// sessions, remember-me, behaviour during a cache or region outage), plus conflicts and questions.
// Input is design metadata only: no business records, no names of people, no secrets.
// Checks: enums are valid, session rules name a user type that was sent, lengths are capped, nothing names a
// product the design does not mention; staff without required MFA is reported as a conflict; one retry when no
// session rule survives.

var (
	CookieCategories = []string{"necessary", "preferences", "analytics", "marketing"}
	CookieConsent    = []string{"required", "not-required"}
	SessionStores    = []string{"server", "token", "hybrid"}
	SessionMFA       = []string{"required", "optional", "not-required"}
)

const SessionMaxUserTypes = 12

type SessionUserType struct {
	Name  string `json:"name"`
	Staff bool   `json:"staff"`
}
type SessionPolicyInput struct {
	Project   string            `json:"project"`
	Kind      string            `json:"kind"`
	Platform  []string          `json:"platform"`
	Targets   []string          `json:"targets"`
	Layers    []DiagramLayer    `json:"layers"`
	UserTypes []SessionUserType `json:"userTypes"`
	Channels  []string          `json:"channels"`
}
type CookieRule struct {
	Category   string `json:"category"`
	Purpose    string `json:"purpose"`
	Consent    string `json:"consent"`
	Attributes string `json:"attributes"`
	Lifetime   string `json:"lifetime"`
	DomainPath string `json:"domainPath"`
	Why        string `json:"why"`
}
type SessionRule struct {
	UserType          string `json:"userType"`
	Channels          string `json:"channels"`
	Store             string `json:"store"`
	IdleTimeout       string `json:"idleTimeout"`
	AbsoluteLifetime  string `json:"absoluteLifetime"`
	Reauth            string `json:"reauth"`
	MFA               string `json:"mfa"`
	SignOutEverywhere string `json:"signOutEverywhere"`
	Rotation          string `json:"rotation"`
	Concurrent        string `json:"concurrent"`
	RememberMe        string `json:"rememberMe"`
	Outage            string `json:"outage"`
	Why               string `json:"why"`
}
type SessionPolicyOutput struct {
	Summary   string        `json:"summary"`
	Cookies   []CookieRule  `json:"cookies"`
	Sessions  []SessionRule `json:"sessions"`
	Conflicts []string      `json:"conflicts"`
	Questions []string      `json:"questions"`
	Dropped   int           `json:"dropped"`
	DropNotes []string      `json:"dropNotes"`
	Model     string        `json:"model"`
	Attempts  int           `json:"attempts,omitempty"`
}

func (in SessionPolicyInput) Validate() error {
	if in.Kind != "new" && in.Kind != "modernize" && in.Kind != "extend" {
		return errors.New("unknown project kind")
	}
	if tooLong(in.Project, 180) || len(in.Platform) > 12 || len(in.Targets) > 16 || len(in.Layers) > 12 || len(in.Channels) > 8 {
		return errors.New("design summary too large")
	}
	if len(in.UserTypes) == 0 || len(in.UserTypes) > SessionMaxUserTypes {
		return errors.New("send between 1 and 12 user types")
	}
	for _, list := range [][]string{in.Platform, in.Targets, in.Channels} {
		for _, s := range list {
			if strings.TrimSpace(s) == "" || tooLong(s, 300) {
				return errors.New("invalid summary line")
			}
		}
	}
	for _, l := range in.Layers {
		if l.ID == "" || tooLong(l.ID, 40) || tooLong(l.Label, 80) || strings.TrimSpace(l.Design) == "" || tooLong(l.Design, 1500) {
			return errors.New("invalid plan layer")
		}
	}
	seen := map[string]bool{}
	for _, u := range in.UserTypes {
		k := strings.ToLower(strings.TrimSpace(u.Name))
		if k == "" || tooLong(u.Name, 60) || seen[k] {
			return errors.New("invalid user type")
		}
		seen[k] = true
	}
	return nil
}

func (in SessionPolicyInput) stated() string {
	var b strings.Builder
	b.WriteString(in.Project)
	for _, s := range append(append(append([]string{}, in.Platform...), in.Targets...), in.Channels...) {
		b.WriteString(" " + s)
	}
	for _, l := range in.Layers {
		b.WriteString(" " + l.Label + " " + l.Design)
	}
	for _, u := range in.UserTypes {
		b.WriteString(" " + u.Name)
	}
	return b.String()
}

func (in SessionPolicyInput) userType(s string) (SessionUserType, bool) {
	s = strings.TrimSpace(s)
	for _, u := range in.UserTypes {
		if strings.EqualFold(strings.TrimSpace(u.Name), s) {
			u.Name = strings.TrimSpace(u.Name)
			return u, true
		}
	}
	return SessionUserType{}, false
}

func (in SessionPolicyInput) verify(out SessionPolicyOutput) SessionPolicyOutput {
	guard := BriefFillInput{Project: in.Project, Answers: map[string]string{"x": in.stated()}}
	clean := SessionPolicyOutput{Summary: capText(out.Summary, 600), Cookies: []CookieRule{}, Sessions: []SessionRule{}, Conflicts: []string{}, Questions: []string{}, DropNotes: []string{}}
	drop := func(what, why string) { clean.Dropped++; clean.DropNotes = addDropNote(clean.DropNotes, what+": "+why) }
	seenCat := map[string]bool{}
	for _, c := range out.Cookies {
		what := "cookie rule " + quoteName(c.Category)
		switch {
		case !slices.Contains(CookieCategories, c.Category):
			drop(what, "not one of necessary, preferences, analytics or marketing")
			continue
		case !slices.Contains(CookieConsent, c.Consent):
			drop(what, "consent "+quoteName(c.Consent)+" is not required or not-required")
			continue
		case seenCat[c.Category]:
			drop(what, "a second rule for the same category")
			continue
		}
		c.Purpose, c.Attributes, c.Lifetime, c.DomainPath, c.Why = capText(c.Purpose, 240), capText(c.Attributes, 160), capText(c.Lifetime, 120), capText(c.DomainPath, 120), capText(c.Why, 300)
		if c.Purpose == "" {
			drop(what, "no purpose given")
			continue
		}
		if prod := guard.unstatedProduct(c.Purpose + " " + c.Attributes); prod != "" {
			drop(what, "names "+prod+", which the design does not mention; add the tool to the design or write this rule yourself")
			continue
		}
		// Strictly necessary cookies never need consent; every other category does.
		if c.Category == "necessary" {
			c.Consent = "not-required"
		} else if c.Consent != "required" {
			c.Consent = "required"
			clean.Conflicts = append(clean.Conflicts, "Consent is needed for "+c.Category+" cookies; the draft said it was not (corrected to required).")
		}
		seenCat[c.Category] = true
		clean.Cookies = append(clean.Cookies, c)
	}
	seenU := map[string]bool{}
	for _, s := range out.Sessions {
		u, ok := in.userType(s.UserType)
		what := "session rule for " + quoteName(s.UserType)
		switch {
		case !ok:
			drop(what, "this user type was not sent")
			continue
		case seenU[u.Name]:
			drop(what, "a second rule for the same user type")
			continue
		case !slices.Contains(SessionStores, s.Store):
			drop(what, "store "+quoteName(s.Store)+" is not server, token or hybrid")
			continue
		case !slices.Contains(SessionMFA, s.MFA):
			drop(what, "MFA "+quoteName(s.MFA)+" is not required, optional or not-required")
			continue
		}
		s.UserType = u.Name
		for _, f := range []*string{&s.Channels, &s.IdleTimeout, &s.AbsoluteLifetime, &s.Reauth, &s.SignOutEverywhere, &s.Rotation, &s.Concurrent, &s.RememberMe, &s.Outage, &s.Why} {
			*f = capText(*f, 240)
		}
		if s.IdleTimeout == "" || s.AbsoluteLifetime == "" {
			drop(what, "no idle timeout or no absolute lifetime")
			continue
		}
		if prod := guard.unstatedProduct(s.Outage + " " + s.Reauth + " " + s.Channels); prod != "" {
			drop(what, "names "+prod+", which the design does not mention")
			continue
		}
		if u.Staff && s.MFA != "required" {
			clean.Conflicts = append(clean.Conflicts, fmt.Sprintf("%s are staff: multi-factor sign-in should be required, the draft says %s.", u.Name, s.MFA))
		}
		seenU[u.Name] = true
		clean.Sessions = append(clean.Sessions, s)
	}
	keepLines(guard, out.Conflicts, &clean.Conflicts, "conflict", drop)
	keepLines(guard, out.Questions, &clean.Questions, "question", drop)
	return clean
}

const sessionPolicyPrompt = `You are the MindSuite Session policy agent, a senior security and privacy architect. You receive a design summary as quoted data: the platform decision, the targets (availability, recovery, regulations such as PCI DSS, GDPR or India's DPDP Act, region), the identity, cache, security and recovery layers, the user types (marked staff or not) and the channels (web browser, mobile app, API). Draft a cookie policy and a session policy.
Cookies (one rule per category used: necessary, preferences, analytics, marketing):
- purpose; consent: "not-required" only for strictly necessary cookies, "required" for every other category (ask before setting them, and allow refusing as easily as accepting).
- attributes: Secure, HttpOnly for anything the page script does not need, SameSite (Lax by default, Strict for staff and payment flows, None only with Secure for a stated cross-site need); lifetime; domain and path (as narrow as possible, no parent-domain cookies unless needed).
- Never store personal data, card data or tokens readable by scripts in cookies; a session cookie holds only an opaque identifier.
Sessions (one rule per user type, using the user type name exactly as sent):
- channels it covers; store: "server" (opaque id, session kept server-side), "token" (signed short-lived access token plus refresh token, typical for mobile apps and APIs) or "hybrid".
- idleTimeout and absoluteLifetime (shorter for staff and administrators); reauth: which actions need signing in again (payments, changing email or password, payouts, admin actions); mfa: "required" for staff; signOutEverywhere; rotation (new session id on sign-in and on privilege change, preventing session fixation; refresh-token rotation with reuse detection); concurrent sessions; rememberMe rules (never for staff unless stated); outage: what happens to sessions when the cache or a region fails, consistent with the availability target.
- Use only products and services named in the summary; never name another product, vendor or cloud service.
"conflicts": anything that cannot be true together (for example sessions kept only in a single cache node while the availability target needs failover, marketing cookies without consent where GDPR or DPDP applies, remember-me for administrators).
"questions": what the security owner or privacy owner must decide that the summary does not say.
"summary": two or three sentences.
Keep every field to one or two short sentences; give at most 8 conflicts and 10 questions.`

const sessionPolicyRetry = " A first attempt returned no valid session rule. Use exactly the user type names sent and the allowed values for store and mfa."

func (l *LLM) SessionPolicy(ctx context.Context, in SessionPolicyInput) (SessionPolicyOutput, error) {
	return RunSessionPolicy(ctx, in, l.JSON)
}

// RunSessionPolicy asks the model, verifies the answer and asks once more when no session rule survives.
func RunSessionPolicy(ctx context.Context, in SessionPolicyInput, call JSONCaller) (SessionPolicyOutput, error) {
	data, err := json.Marshal(struct {
		Design SessionPolicyInput `json:"design"`
	}{in})
	if err != nil {
		return SessionPolicyOutput{}, err
	}
	schema := obj(map[string]any{
		"summary": str(),
		"cookies": arr(obj(map[string]any{"category": enum(CookieCategories...), "purpose": str(), "consent": enum(CookieConsent...),
			"attributes": str(), "lifetime": str(), "domainPath": str(), "why": str()})),
		"sessions": arr(obj(map[string]any{"userType": str(), "channels": str(), "store": enum(SessionStores...), "idleTimeout": str(),
			"absoluteLifetime": str(), "reauth": str(), "mfa": enum(SessionMFA...), "signOutEverywhere": str(), "rotation": str(),
			"concurrent": str(), "rememberMe": str(), "outage": str(), "why": str()})),
		"conflicts": arr(str()),
		"questions": arr(str()),
	})
	var out SessionPolicyOutput
	for attempt := 1; attempt <= 2; attempt++ {
		var raw SessionPolicyOutput
		p := sessionPolicyPrompt
		if attempt == 2 {
			p += sessionPolicyRetry
		}
		if err = call(ctx, ModelStrong, p, string(data), schema, 16000, &raw); err != nil {
			return SessionPolicyOutput{}, err
		}
		out = in.verify(raw)
		out.Attempts = attempt
		if len(out.Sessions) >= 1 {
			break
		}
	}
	out.Model = Resolve(ModelStrong)
	return out, nil
}
