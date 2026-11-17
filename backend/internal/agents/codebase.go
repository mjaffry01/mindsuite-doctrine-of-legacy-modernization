package agents

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Limits keep a run bounded. Files over the size limit are listed as gaps.
const (
	MaxFileBytes  = 2 << 20   // 2 MB per file
	MaxTotalBytes = 300 << 20 // 300 MB per run
)

// File is one text file of the code being analysed. Code is held in memory
// only for the duration of the run and is never written to disk by the backend.
type File struct {
	Path   string   `json:"path"`
	Lang   string   `json:"lang"`
	Module string   `json:"module"`
	Lines  []string `json:"-"`
	Bytes  int      `json:"bytes"`
	IsTest bool     `json:"isTest"`
	Pkg    string   `json:"pkg,omitempty"`
}

// Skip is a file that was not analysed, with the reason.
type Skip struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

// Codebase is the read-only input of a run.
type Codebase struct {
	Root     string  `json:"root"`
	Files    []*File `json:"files"`
	Skipped  []Skip  `json:"skipped"`
	Branch   string  `json:"branch,omitempty"`
	Commit   string  `json:"commit,omitempty"`
	HasGit   bool    `json:"hasGit"`
	LocalDir string  `json:"-"` // set when loaded from a folder on this machine (enables Git history)
	byPath   map[string]*File
	total    int
	pkgRoot  string
}

var skipDirs = map[string]bool{".git": true, "node_modules": true, "target": true, "bin": true, "obj": true, "build": true, "dist": true,
	"vendor": true, ".idea": true, ".vscode": true, ".gradle": true, ".mvn": true, "__pycache__": true, ".venv": true, "venv": true, "out": true, "coverage": true}

var langByExt = map[string]string{
	".java": "Java", ".kt": "Kotlin", ".scala": "Scala", ".groovy": "Groovy", ".cs": "C#", ".vb": "VB.NET", ".fs": "F#",
	".js": "JavaScript", ".jsx": "JavaScript", ".mjs": "JavaScript", ".ts": "TypeScript", ".tsx": "TypeScript",
	".py": "Python", ".rb": "Ruby", ".php": "PHP", ".go": "Go", ".rs": "Rust", ".c": "C", ".h": "C", ".cpp": "C++", ".hpp": "C++", ".cc": "C++",
	".sql": "SQL", ".ddl": "SQL", ".trg": "PL/SQL", ".vw": "SQL", ".csv": "CSV", ".tsv": "CSV", ".pks": "PL/SQL", ".pkb": "PL/SQL", ".pls": "PL/SQL", ".plsql": "PL/SQL", ".prc": "PL/SQL", ".fnc": "PL/SQL",
	".cbl": "COBOL", ".cob": "COBOL", ".cpy": "COBOL copybook", ".jcl": "JCL", ".pli": "PL/I", ".rpg": "RPG", ".rpgle": "RPG", ".sqlrpgle": "RPG", ".clle": "CL",
	".srw": "PowerBuilder", ".srd": "PowerBuilder", ".sru": "PowerBuilder", ".srf": "PowerBuilder", ".fmb": "Oracle Forms", ".abap": "ABAP",
	".jsp": "JSP", ".xhtml": "JSF / XHTML", ".aspx": "ASP.NET", ".cshtml": "Razor", ".html": "HTML", ".css": "CSS", ".scss": "CSS",
	".xml": "XML", ".json": "JSON", ".yml": "YAML", ".yaml": "YAML", ".properties": "Properties", ".config": "Config", ".ini": "Config", ".toml": "TOML",
	".tf": "Terraform", ".sh": "Shell", ".bat": "Batch", ".cmd": "Batch", ".ps1": "PowerShell", ".gradle": "Gradle", ".csproj": "MSBuild", ".vbproj": "MSBuild", ".sln": "MSBuild",
	".md": "Markdown", ".txt": "Text", ".log": "Text", ".env": "Config", ".xsl": "XSLT", ".xslt": "XSLT", ".wsdl": "WSDL", ".xsd": "XSD", ".proto": "Protobuf", ".graphql": "GraphQL",
}

func langOf(p string) string {
	base := strings.ToLower(path.Base(p))
	switch {
	case base == "dockerfile" || strings.HasPrefix(base, "dockerfile."):
		return "Dockerfile"
	case base == "jenkinsfile":
		return "Groovy"
	case base == "makefile":
		return "Make"
	case base == "go.mod":
		return "Go module"
	}
	if l, ok := langByExt[strings.ToLower(path.Ext(p))]; ok {
		return l
	}
	return ""
}

// CodeLangs are languages the structure, quality and assessment agents read.
var CodeLangs = map[string]bool{"Java": true, "Kotlin": true, "Scala": true, "Groovy": true, "C#": true, "VB.NET": true, "JavaScript": true, "TypeScript": true,
	"Python": true, "Ruby": true, "PHP": true, "Go": true, "Rust": true, "C": true, "C++": true}

// LegacyLangs are routed to the legacy-language specialists.
var LegacyLangs = map[string]bool{"COBOL": true, "COBOL copybook": true, "JCL": true, "PL/I": true, "RPG": true, "CL": true, "PowerBuilder": true, "Oracle Forms": true, "ABAP": true}

var testPathRe = regexp.MustCompile(`(?i)(^|/)(test|tests|__tests__|spec|specs)/|(Test|Tests|IT|Spec)\.(java|kt|cs|scala|groovy)$|[._-](test|spec)\.(js|jsx|ts|tsx|py|rb|go)$|(^|/)test_[^/]+\.py$|_test\.go$`)

func (c *Codebase) add(p string, data []byte) {
	p = strings.TrimPrefix(path.Clean(strings.ReplaceAll(p, "\\", "/")), "/")
	if p == "." || p == "" || strings.HasPrefix(p, "../") {
		return
	}
	for _, seg := range strings.Split(path.Dir(p), "/") {
		if skipDirs[seg] {
			return
		}
	}
	lang := langOf(p)
	if lang == "" {
		c.Skipped = append(c.Skipped, Skip{p, "unrecognised file type"})
		return
	}
	if len(data) > MaxFileBytes {
		c.Skipped = append(c.Skipped, Skip{p, fmt.Sprintf("larger than %d MB", MaxFileBytes>>20)})
		return
	}
	if c.total+len(data) > MaxTotalBytes {
		c.Skipped = append(c.Skipped, Skip{p, "run size limit reached"})
		return
	}
	head := data
	if len(head) > 8000 {
		head = head[:8000]
	}
	if bytes.IndexByte(head, 0) >= 0 {
		c.Skipped = append(c.Skipped, Skip{p, "binary file"})
		return
	}
	c.total += len(data)
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	c.Files = append(c.Files, &File{Path: p, Lang: lang, Lines: strings.Split(text, "\n"), Bytes: len(data), IsTest: testPathRe.MatchString(p)})
}

// LoadDir reads a folder on this machine, read-only.
func LoadDir(root string) (*Codebase, error) {
	st, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !st.IsDir() {
		return nil, errors.New("not a folder: " + root)
	}
	c := &Codebase{Root: filepath.Base(root), LocalDir: root}
	err = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if rel != "." && skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		if langOf(rel) == "" {
			c.Skipped = append(c.Skipped, Skip{rel, "unrecognised file type"})
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		if info.Size() > MaxFileBytes {
			c.Skipped = append(c.Skipped, Skip{rel, fmt.Sprintf("larger than %d MB", MaxFileBytes>>20)})
			return nil
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return nil
		}
		c.add(rel, data)
		return nil
	})
	if err != nil {
		return nil, err
	}
	c.readGitHead(root)
	c.finish()
	return c, nil
}

// LoadZip reads a zip archive in memory. Nothing is extracted to disk.
func LoadZip(r io.ReaderAt, size int64, name string) (*Codebase, error) {
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return nil, err
	}
	c := &Codebase{Root: strings.TrimSuffix(name, ".zip")}
	for _, zf := range zr.File {
		if zf.FileInfo().IsDir() {
			continue
		}
		if strings.Contains(zf.Name, "/.git/") || strings.HasPrefix(zf.Name, ".git/") {
			c.HasGit = true
			continue
		}
		if langOf(zf.Name) == "" {
			c.Skipped = append(c.Skipped, Skip{zf.Name, "unrecognised file type"})
			continue
		}
		if zf.UncompressedSize64 > MaxFileBytes {
			c.Skipped = append(c.Skipped, Skip{zf.Name, fmt.Sprintf("larger than %d MB", MaxFileBytes>>20)})
			continue
		}
		rc, err := zf.Open()
		if err != nil {
			continue
		}
		data, err := io.ReadAll(io.LimitReader(rc, MaxFileBytes+1))
		rc.Close()
		if err != nil {
			continue
		}
		c.add(zf.Name, data)
	}
	c.stripCommonRoot()
	c.finish()
	return c, nil
}

// LoadFiles builds a codebase from uploaded files (path → content).
func LoadFiles(root string, files map[string][]byte) *Codebase {
	c := &Codebase{Root: root}
	for _, k := range sortedKeys(files) {
		c.add(k, files[k])
	}
	c.stripCommonRoot()
	c.finish()
	return c
}

// stripCommonRoot removes a single top folder shared by every file (typical for zips and folder uploads).
func (c *Codebase) stripCommonRoot() {
	if len(c.Files) == 0 {
		return
	}
	first := strings.SplitN(c.Files[0].Path, "/", 2)
	if len(first) < 2 {
		return
	}
	switch strings.ToLower(first[0]) {
	case "src", "app", "lib", "source", "sources", "test", "tests":
		return // a source root, not a wrapper folder
	}
	prefix := first[0] + "/"
	for _, f := range c.Files {
		if !strings.HasPrefix(f.Path, prefix) {
			return
		}
	}
	for _, f := range c.Files {
		f.Path = strings.TrimPrefix(f.Path, prefix)
	}
	for i := range c.Skipped {
		c.Skipped[i].Path = strings.TrimPrefix(c.Skipped[i].Path, prefix)
	}
	if c.Root == "" {
		c.Root = first[0]
	}
}

func (c *Codebase) readGitHead(root string) {
	b, err := os.ReadFile(filepath.Join(root, ".git", "HEAD"))
	if err != nil {
		return
	}
	c.HasGit = true
	head := strings.TrimSpace(string(b))
	if !strings.HasPrefix(head, "ref: ") {
		c.Commit = head
		return
	}
	ref := strings.TrimPrefix(head, "ref: ")
	c.Branch = strings.TrimPrefix(ref, "refs/heads/")
	if h, err := os.ReadFile(filepath.Join(root, ".git", filepath.FromSlash(ref))); err == nil {
		c.Commit = strings.TrimSpace(string(h))
	} else if packed, err := os.ReadFile(filepath.Join(root, ".git", "packed-refs")); err == nil {
		for _, line := range strings.Split(string(packed), "\n") {
			if strings.HasSuffix(line, " "+ref) {
				c.Commit = strings.Fields(line)[0]
			}
		}
	}
}

var (
	javaPkgRe = regexp.MustCompile(`^\s*package\s+([\w.]+)\s*;?`)
	csNsRe    = regexp.MustCompile(`^\s*namespace\s+([\w.]+)`)
)

var sourceRoots = []string{"src/main/java/", "src/main/kotlin/", "src/main/scala/", "src/test/java/", "src/test/kotlin/", "src/main/", "src/test/", "src/", "app/", "lib/", "source/", "sources/"}

// finish assigns modules and builds indexes.
func (c *Codebase) finish() {
	sort.Slice(c.Files, func(i, j int) bool { return c.Files[i].Path < c.Files[j].Path })
	c.byPath = map[string]*File{}
	pkgs := []string{}
	for _, f := range c.Files {
		c.byPath[f.Path] = f
		re := javaPkgRe
		switch f.Lang {
		case "Java", "Kotlin", "Scala", "Groovy":
		case "C#":
			re = csNsRe
		default:
			continue
		}
		for i, l := range f.Lines {
			if i > 60 {
				break
			}
			if m := re.FindStringSubmatch(l); m != nil {
				f.Pkg = m[1]
				pkgs = append(pkgs, m[1])
				break
			}
		}
	}
	c.pkgRoot = commonPkgPrefix(pkgs)
	for _, f := range c.Files {
		f.Module = c.moduleOf(f)
	}
}

func commonPkgPrefix(pkgs []string) string {
	if len(pkgs) == 0 {
		return ""
	}
	prefix := strings.Split(pkgs[0], ".")
	for _, p := range pkgs[1:] {
		parts := strings.Split(p, ".")
		n := 0
		for n < len(prefix) && n < len(parts) && prefix[n] == parts[n] {
			n++
		}
		prefix = prefix[:n]
	}
	return strings.Join(prefix, ".")
}

// ModuleOfPkg maps a package or namespace to a module name ("" if outside the codebase).
func (c *Codebase) ModuleOfPkg(pkg string) string {
	if c.pkgRoot == "" || !strings.HasPrefix(pkg+".", c.pkgRoot+".") {
		return ""
	}
	rest := strings.TrimPrefix(strings.TrimPrefix(pkg, c.pkgRoot), ".")
	if rest == "" {
		return "(root)"
	}
	return strings.SplitN(rest, ".", 2)[0]
}

func (c *Codebase) moduleOf(f *File) string {
	if f.Pkg != "" {
		if m := c.ModuleOfPkg(f.Pkg); m != "" {
			return m
		}
	}
	return ModuleOfPath(f.Path)
}

// ModuleOfPath derives a module from a file path.
func ModuleOfPath(p string) string {
	q := p
	for changed := true; changed; {
		changed = false
		for _, r := range sourceRoots {
			if strings.HasPrefix(q, r) {
				q = strings.TrimPrefix(q, r)
				changed = true
			}
		}
	}
	parts := strings.Split(q, "/")
	if len(parts) <= 1 {
		return "(root)"
	}
	return parts[0]
}

// File returns the file at path p.
func (c *Codebase) File(p string) *File { return c.byPath[p] }

// Modules returns all module names that contain code.
func (c *Codebase) Modules() []string {
	m := map[string]bool{}
	for _, f := range c.Files {
		if CodeLangs[f.Lang] || LegacyLangs[f.Lang] {
			m[f.Module] = true
		}
	}
	return sortedKeys(m)
}

// TotalLines counts lines across files.
func (c *Codebase) TotalLines() int {
	n := 0
	for _, f := range c.Files {
		n += len(f.Lines)
	}
	return n
}

// Numbered returns lines [from,to] (1-based) prefixed with line numbers; redacted lines are replaced.
func (f *File) Numbered(from, to int, redact map[int]bool) string {
	if from < 1 {
		from = 1
	}
	if to > len(f.Lines) || to <= 0 {
		to = len(f.Lines)
	}
	var b strings.Builder
	for i := from; i <= to; i++ {
		line := f.Lines[i-1]
		if redact[i] {
			line = "[line removed: possible secret]"
		}
		fmt.Fprintf(&b, "%5d| %s\n", i, line)
	}
	return b.String()
}

// Line returns line n (1-based), trimmed.
func (f *File) Line(n int) string {
	if n < 1 || n > len(f.Lines) {
		return ""
	}
	return strings.TrimSpace(f.Lines[n-1])
}

func short(s string, n int) string {
	s = strings.TrimSpace(s)
	if len([]rune(s)) <= n {
		return s
	}
	return string([]rune(s)[:n]) + "…"
}
