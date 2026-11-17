package code

import (
	"fmt"
	"github.com/mjaffry01/mindsuite-doctrine-of-legacy-modernization/backend/internal/agents"
	"regexp"
	"strings"
)

// ---------- 6 Security ----------

var (
	authMethods = []struct {
		name string
		re   *regexp.Regexp
	}{
		{"Spring Security", regexp.MustCompile(`@EnableWebSecurity|SecurityFilterChain|WebSecurityConfigurerAdapter|spring-boot-starter-security`)},
		{"OAuth2 / OpenID Connect", regexp.MustCompile(`(?i)oauth2|openid|oidc|AddOpenIdConnect|issuer-uri|\.well-known/openid`)},
		{"JWT tokens", regexp.MustCompile(`(?i)\bjwt\b|JwtBearer|io\.jsonwebtoken|jsonwebtoken|Bearer `)},
		{"SAML", regexp.MustCompile(`(?i)\bsaml2?\b`)},
		{"LDAP / Active Directory", regexp.MustCompile(`(?i)\bldap\b|ActiveDirectory|DirectoryEntry|ldaps?://`)},
		{"Windows / Kerberos", regexp.MustCompile(`(?i)kerberos|spnego|<authentication\s+mode="Windows"|NegotiateDefaults|WindowsAuthentication`)},
		{"Forms / session login", regexp.MustCompile(`(?i)formLogin\(|<authentication\s+mode="Forms"|FormsAuthentication|j_security_check|HttpSession|session\[['"]user`)},
		{"Basic authentication", regexp.MustCompile(`httpBasic\(|(?i)basic\s+realm|BasicAuthentication`)},
		{"API keys", regexp.MustCompile(`(?i)x-api-key|api[_-]?key\s*(?:header|filter)|ApiKeyAuth`)},
		{"Database-stored passwords", regexp.MustCompile(`(?i)(select|where)\b.*\bpassword\b.*\bfrom\b|BCryptPasswordEncoder|PasswordHasher|password_hash\(`)},
	}
	roleCheckRe  = regexp.MustCompile(`@PreAuthorize\s*\(\s*"([^"]+)"|@Secured\s*\(\s*\{?\s*"([^"]+)"|@RolesAllowed\s*\(\s*\{?\s*"([^"]+)"|hasRole\s*\(\s*['"]([^'"]+)['"]|hasAuthority\s*\(\s*['"]([^'"]+)['"]|\[Authorize\s*\(\s*Roles\s*=\s*"([^"]+)"|isUserInRole\s*\(\s*"([^"]+)"|IsInRole\s*\(\s*"([^"]+)"|@permission_required\s*\(\s*['"]([^'"]+)['"]|has_perm\s*\(\s*['"]([^'"]+)['"]`)
	authOnlyRe   = regexp.MustCompile(`\[Authorize\]|@login_required|isAuthenticated\(\)|\.authenticated\(\)|@PermitAll|\.permitAll\(\)|\[AllowAnonymous\]`)
	weakCryptoRe = regexp.MustCompile(`(?i)MessageDigest\.getInstance\(\s*"(MD5|SHA-?1)"|\b(MD5CryptoServiceProvider|SHA1Managed|DESCryptoServiceProvider|TripleDES)\b|Cipher\.getInstance\(\s*"(DES|RC4|[^"]*ECB[^"]*)|hashlib\.(md5|sha1)\(|createHash\(\s*['"](md5|sha1)['"]`)
	sqlConcatRe  = regexp.MustCompile(`(?i)"(?:select|insert|update|delete)\b[^"]*"\s*\+\s*[A-Za-z_]|'(?:select|insert|update|delete)\b[^']*'\s*\+\s*[A-Za-z_]|(?:executeQuery|executeUpdate|execute|createQuery|createNativeQuery|ExecuteSqlRaw|cursor\.execute)\s*\(\s*["'][^"']*["']\s*\+`)
	csrfOffRe    = regexp.MustCompile(`csrf\(\)\s*\.disable\(\)|csrf\(.*disable|@csrf_exempt|IgnoreAntiforgeryToken`)
	corsAnyRe    = regexp.MustCompile(`allowedOrigins\s*\(\s*"\*"|AllowAnyOrigin\(\)|Access-Control-Allow-Origin["']?\s*[:,]\s*["']\*|origins\s*=\s*"\*"`)
)

func runSecurity(r *Run) {
	s := r.Store
	methodSeen := map[string]bool{}
	checks := 0
	keep := func(f *File) bool {
		return !f.IsTest && f.Lang != "Markdown" && f.Lang != "Text" && f.Lang != "HTML" && f.Lang != "CSS"
	}
	eachLine(r, "sec", keep, func(f *File, n int, l string) {
		trim := strings.TrimSpace(l)
		comment := strings.HasPrefix(trim, "//") || strings.HasPrefix(trim, "#") || strings.HasPrefix(trim, "*") || strings.HasPrefix(trim, "<!--")
		for _, a := range authMethods {
			if !methodSeen[a.name] && a.re.MatchString(l) && !comment {
				methodSeen[a.name] = true
				s.Add(Finding{Agent: "sec", Kind: "auth-method", Title: "Login method: " + a.name, Citations: []Citation{cite(f, n)}, Confidence: "medium", Data: map[string]any{"method": a.name}})
			}
		}
		if m := roleCheckRe.FindStringSubmatch(l); m != nil && !comment {
			expr := strings.Join(m[1:], "")
			checks++
			roles := rolesIn(expr)
			s.Add(Finding{Agent: "sec", Kind: "role-check", Module: f.Module, Title: "Access check: " + short(expr, 80), Citations: []Citation{cite(f, n)}, Confidence: "high",
				Data: map[string]any{"expression": expr, "roles": roles, "endpoint": nearEndpoint(f, n)}})
		} else if authOnlyRe.MatchString(l) && !comment {
			checks++
		}
		if CodeLangs[f.Lang] || f.Lang == "Properties" || f.Lang == "YAML" || f.Lang == "Config" || f.Lang == "XML" || f.Lang == "JSON" || f.Lang == "Shell" || f.Lang == "Dockerfile" || f.Lang == "Terraform" {
			for _, sr := range agents.SecretPatterns {
				if sr.Re.MatchString(l) && !agents.Placeholder.MatchString(l) {
					r.MarkSecret(f.Path, n)
					// never quote a secret line
					s.Add(Finding{Agent: "sec", Kind: "secret", Module: f.Module, Title: "Possible hard-coded " + sr.Kind, Detail: "The value is not recorded. Move it to a secret store and rotate it.",
						Citations: []Citation{{File: f.Path, Line: n}}, Confidence: "medium", Data: map[string]any{"type": sr.Kind, "severity": "high"}})
					break
				}
			}
		}
		if comment {
			return
		}
		if m := weakCryptoRe.FindStringSubmatch(l); m != nil {
			s.Add(Finding{Agent: "sec", Kind: "weak-crypto", Module: f.Module, Title: "Weak cryptography: " + strings.Trim(strings.Join(m[1:], ""), " "), Citations: []Citation{cite(f, n)}, Confidence: "high", Data: map[string]any{"severity": "medium"}})
		}
		if sqlConcatRe.MatchString(l) {
			s.Add(Finding{Agent: "sec", Kind: "injection-risk", Module: f.Module, Title: "SQL built by string concatenation (possible SQL injection)", Citations: []Citation{cite(f, n)}, Confidence: "medium", Data: map[string]any{"severity": "high"}})
		}
		if csrfOffRe.MatchString(l) {
			s.Add(Finding{Agent: "sec", Kind: "csrf-disabled", Module: f.Module, Title: "CSRF protection disabled", Citations: []Citation{cite(f, n)}, Confidence: "high", Data: map[string]any{"severity": "medium"}})
		}
		if corsAnyRe.MatchString(l) {
			s.Add(Finding{Agent: "sec", Kind: "cors-any", Module: f.Module, Title: "Any website may call this API (CORS *)", Citations: []Citation{cite(f, n)}, Confidence: "high", Data: map[string]any{"severity": "medium"}})
		}
	})
	if len(methodSeen) == 0 {
		s.Log("sec", "No login method found in the code: it may be handled by a gateway, the app server or a separate system. Ask the team.")
		s.Question("sec", "", "No login method was found in the code. Where are users authenticated (app server, gateway, single sign-on product)?")
	}
	s.Log("sec", fmt.Sprintf("%d login methods, %d access checks.", len(methodSeen), checks))
}

var roleNameRe = regexp.MustCompile(`['"]?(?:ROLE_)?([A-Z][A-Z0-9_]{1,40}|[a-z][a-z0-9_.:-]{2,40})['"]?`)

func rolesIn(expr string) []string {
	out := []string{}
	for _, part := range regexp.MustCompile(`[,|&()\s]+|hasRole|hasAnyRole|hasAuthority|and|or`).Split(expr, -1) {
		p := strings.Trim(part, `'" `)
		if p == "" || p == "isAuthenticated" {
			continue
		}
		if roleNameRe.MatchString(p) {
			out = appendUnique(out, strings.TrimPrefix(p, "ROLE_"))
		}
	}
	return out
}

// nearEndpoint finds the endpoint annotation next to an access check (within 4 lines).
func nearEndpoint(f *File, n int) string {
	for j := max(1, n-4); j <= min(len(f.Lines), n+4); j++ {
		l := f.Lines[j-1]
		if m := springMapRe.FindStringSubmatch(l); m != nil {
			return m[1] + " " + m[2]
		}
		if m := aspVerbRe.FindStringSubmatch(l); m != nil {
			return m[1] + " " + m[2]
		}
	}
	return ""
}
