package code

import "testing"

// TestMultiString: SQL that spans lines in C#, Java and JavaScript strings is recognised;
// ordinary code and one-line strings are not.
func TestMultiString(t *testing.T) {
	cases := []struct {
		lang  string
		lines []string
		want  []bool
	}{
		{"C#", []string{`var sql = @"`, `  SELECT Name`, `  FROM Items"; // done`, `var x = "FROM nothing";`}, []bool{false, true, true, false}},
		{"C#", []string{`var sql = """`, `  UPDATE Items`, `  SET Name = 1`, `  """;`, `Console.WriteLine("x");`}, []bool{false, true, true, true, false}},
		{"C#", []string{`var s = "a \" quote"; var c = '"';`, `FROM Items`}, []bool{false, false}},
		{"Java", []string{`String sql = """`, `    SELECT * FROM Items`, `    """;`}, []bool{false, true, true}},
		{"JavaScript", []string{"const q = `", "  DELETE FROM Items", "`;"}, []bool{false, true, true}},
		{"VB.NET", []string{`Dim s = "He said ""hi"""`, `FROM Items`}, []bool{false, false}},
	}
	for _, c := range cases {
		ms := &multiString{}
		for i, l := range c.lines {
			if got := ms.line(c.lang, l); got != c.want[i] {
				t.Errorf("%s line %d %q: in SQL string = %v, want %v", c.lang, i+1, l, got, c.want[i])
			}
		}
	}
}
