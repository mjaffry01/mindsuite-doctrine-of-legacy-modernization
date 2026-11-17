package agents

import (
	"fmt"
	"regexp"
)

// Shared helpers for every team. Team packages use these instead of their own copies.

// SecretPattern is one kind of credential that must never be quoted or sent.
type SecretPattern struct {
	Kind string
	Re   *regexp.Regexp
}

var (
	// SecretPatterns find lines that may hold a credential.
	SecretPatterns = []SecretPattern{
		{"password", regexp.MustCompile(`(?i)(password|passwd|pwd)\s*[:=]\s*["']?[^\s"'$#{}<>]{4,}`)},
		{"secret or token", regexp.MustCompile(`(?i)(secret|token|api[_-]?key|access[_-]?key|client[_-]?secret)\s*[:=]\s*["'][A-Za-z0-9/+_\-.=]{12,}["']`)},
		{"AWS access key", regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`)},
		{"private key", regexp.MustCompile(`-----BEGIN (RSA |EC |DSA |OPENSSH )?PRIVATE KEY-----`)},
		{"connection string with password", regexp.MustCompile(`(?i)(password|pwd)=[^;"'\s$]{3,};|://[^/\s:@]+:[^@\s/$]{3,}@`)},
	}
	// Placeholder marks values that are not real secrets (variables, examples).
	Placeholder = regexp.MustCompile(`(?i)\$\{|%\(|<%=|\{\{|changeme|your[_-]?password|example|placeholder|xxx+|\*\*\*|getenv|environ|process\.env|System\.getProperty|Configuration\[`)
)

// EachLine calls fn for every line of the files accepted by keep, and records coverage.
func EachLine(r *Run, agent string, keep func(*File) bool, fn func(f *File, n int, line string)) {
	for _, f := range r.Code.Files {
		if !keep(f) {
			continue
		}
		r.Cover(agent, f)
		for i, l := range f.Lines {
			fn(f, i+1, l)
		}
	}
}

// Cite points to a line and quotes it, unless the line may contain a secret.
func Cite(f *File, n int) Citation {
	l := f.Line(n)
	if LooksSecret(l) {
		return Citation{File: f.Path, Line: n}
	}
	return Citation{File: f.Path, Line: n, Quote: short(l, 160)}
}

// LooksSecret reports whether a line may hold a credential.
func LooksSecret(l string) bool {
	for _, sr := range SecretPatterns {
		if sr.Re.MatchString(l) && !Placeholder.MatchString(l) {
			return true
		}
	}
	return false
}

func looksSecret(l string) bool { return LooksSecret(l) }

// LineOf returns the line number of a byte offset in a file.
func LineOf(f *File, offset int) int {
	n, pos := 1, 0
	for i, l := range f.Lines {
		if pos+len(l)+1 > offset {
			return i + 1
		}
		pos += len(l) + 1
		n = i + 2
	}
	return n
}

// Contains reports whether s holds v.
func Contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

// AppendUnique appends v unless it is already in s.
func AppendUnique(s []string, v string) []string {
	for _, x := range s {
		if x == v {
			return s
		}
	}
	return append(s, v)
}

func appendUnique(s []string, v string) []string { return AppendUnique(s, v) }

// Uniq removes duplicates, keeping order.
func Uniq(s []string) []string {
	out := []string{}
	for _, x := range s {
		out = AppendUnique(out, x)
	}
	return out
}

// ToInt reads a number from finding data.
func ToInt(v any) int {
	switch x := v.(type) {
	case int:
		return x
	case float64:
		return int(x)
	}
	return 0
}

// ToStrings reads a string list from finding data.
func ToStrings(v any) []string {
	switch x := v.(type) {
	case []string:
		return x
	case []any:
		out := []string{}
		for _, e := range x {
			out = append(out, fmt.Sprint(e))
		}
		return out
	}
	return nil
}

// FirstN keeps the first n items and says how many more there are.
func FirstN(s []string, n int) []string {
	if len(s) <= n {
		return s
	}
	return append(append([]string{}, s[:n]...), fmt.Sprintf("… %d more", len(s)-n))
}

// Short cuts a string to n characters.
func Short(s string, n int) string { return short(s, n) }

// SortedKeys returns the keys of m in order.
func SortedKeys[V any](m map[string]V) []string { return sortedKeys(m) }

// JSON-schema helpers for structured outputs.
func Obj(props map[string]any) map[string]any { return obj(props) }
func Arr(items map[string]any) map[string]any { return arr(items) }
func Str() map[string]any                     { return str() }
func Integer() map[string]any                 { return integer() }
func Enum(v ...string) map[string]any         { return enum(v...) }

// ReportSchema is the structured output of every report writer.
var ReportSchema = obj(map[string]any{"summary": str(), "story": str()})

var _ = fmt.Sprint
