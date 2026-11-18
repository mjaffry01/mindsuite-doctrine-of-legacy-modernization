package agents

import "testing"

// TestSecretMatch checks the password rule: COBOL / PL/I data names are not secrets, but the same
// shape in a config file is a real value and must be reported.
func TestSecretMatch(t *testing.T) {
	cases := []struct {
		lang, line string
		want       bool
	}{
		{"Properties", "db.password=PROD-DB-2024", true}, // regression: was missed
		{"YAML", "password: PROD-DB-2024", true},
		{"", "db.password=PROD-DB-2024", true},
		{"Properties", "db.password=Prod2024x", true},
		{"COBOL", "           IF SEC-USR-PWD = WS-USER-PWD", false},
		{"COBOL copybook", "           IF PASSWORD = CU-PASSWORD OF CUST-REC.", false},
		{"PL/I", "  IF PWD = USER-PWD THEN", false},
		{"COBOL", `           IF PWD = "SECRET-2024"`, true}, // a quoted value is a value
		{"Java", "if (pwd==input1) {", false},                // a comparison
		{"Properties", "password=${DB_PASS}", false},         // placeholder
	}
	for _, c := range cases {
		got := false
		for _, sr := range SecretPatterns {
			if SecretMatchIn(sr, c.lang, c.line) {
				got = true
			}
		}
		if got != c.want {
			t.Errorf("%s %q: secret=%v, want %v", c.lang, c.line, got, c.want)
		}
	}
	// without a file type the check stays strict, so a data-name line is still kept from the model
	if !LooksSecret("IF SEC-USR-PWD = WS-USER-PWD") {
		t.Error("LooksSecret without a language must not apply the COBOL exception")
	}
}

// TestCiteQuotesCOBOLDataNames: a COBOL comparison of data names is quoted; a config password is not.
func TestCiteQuotesCOBOLDataNames(t *testing.T) {
	c := LoadFiles("x", map[string][]byte{
		"src/LOGIN.cbl":       []byte("           IF SEC-USR-PWD = WS-USER-PWD\n"),
		"conf/app.properties": []byte("db.password=PROD-DB-2024\n"),
	})
	for _, f := range c.Files {
		q := Cite(f, 1).Quote
		switch f.Lang {
		case "COBOL":
			if q == "" {
				t.Errorf("%s: COBOL comparison should be quoted", f.Path)
			}
		default:
			if q != "" {
				t.Errorf("%s (%s): secret line was quoted: %q", f.Path, f.Lang, q)
			}
		}
	}
}
