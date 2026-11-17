package documents

import (
	"bytes"
	"compress/flate"
	"encoding/base64"
	"encoding/xml"
	"html"
	"io"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"
)

// DModel is what the documents & diagrams team knows: each document with its headings and sections, and
// each diagram with its shapes and links. Word, PowerPoint, Visio and PDF text is extracted by the core
// loader (internal/agents/documents.go); HTML (e.g. a Confluence export), Markdown, text and draw.io files
// are read here. Line numbers always refer to the text the agents read, so every citation can be re-opened.

// Heading is a section heading.
type Heading struct {
	Level int
	Text  string
	Line  int
}

// Doc is one document.
type Doc struct {
	F        *File
	Kind     string
	Title    string
	Text     []string // the readable text of each line (HTML tags removed)
	Headings []Heading
	Words    int
	Date     time.Time
	DateLine int
	Version  string
	Drafts   []int // lines with DRAFT / TBD / TODO
}

// Node and Link are diagram elements.
type Node struct {
	ID, Label string
	Line      int
}

// Link connects two nodes.
type Link struct {
	From, To, Label string
	Line            int
}

// Diagram is one draw.io or Visio diagram.
type Diagram struct {
	F     *File
	Kind  string
	Nodes []Node
	Links []Link
}

// DModel holds the parsed documents.
type DModel struct {
	Docs     []*Doc
	Diagrams []*Diagram
	Files    []string
	NotRead  []agentsSkip
}

type agentsSkip struct{ Path, Why string }

// Summary is shown with the run.
func (m *DModel) Summary() map[string]any {
	words := 0
	for _, d := range m.Docs {
		words += d.Words
	}
	return map[string]any{"documents": len(m.Docs), "diagrams": len(m.Diagrams), "words": words, "notRead": len(m.NotRead), "exportFiles": len(m.Files)}
}

var (
	tagRe      = regexp.MustCompile(`<[^>]*>`)
	htmlHead   = regexp.MustCompile(`(?i)<h([1-6])[^>]*>(.*?)</h[1-6]>`)
	mdHead     = regexp.MustCompile(`^(#{1,6})\s+(.+)$`)
	numHead    = regexp.MustCompile(`^(\d+(?:\.\d+){0,3})\.?\s+([A-Z][^.!?]{2,70})$`)
	draftRe    = regexp.MustCompile(`(?i)\b(draft|tbd|to be decided|todo|to do:|work in progress|wip)\b`)
	versionRe  = regexp.MustCompile(`(?i)\b(?:version|rev(?:ision)?|v)\s*[:.]?\s*(\d+(?:\.\d+){0,2})\b`)
	dateLineRe = regexp.MustCompile(`(?i)(last (?:updated|modified|changed|reviewed)|updated|date|revised|as of|created)\s*[:\-–]?\s*(\d{4}-\d{2}-\d{2}|\d{1,2}[./]\d{1,2}[./]\d{4}|\d{1,2}\s+[A-Za-z]{3,9}\s+\d{4}|[A-Za-z]{3,9}\s+\d{1,2},?\s+\d{4}|[A-Za-z]{3,9}\s+\d{4})`)
)

func parseDate(s string) time.Time {
	s = strings.TrimSpace(strings.ReplaceAll(s, ",", ""))
	for _, l := range []string{"2006-01-02", "02.01.2006", "2.1.2006", "02/01/2006", "2/1/2006", "2 January 2006", "2 Jan 2006", "January 2 2006", "Jan 2 2006", "January 2006", "Jan 2006"} {
		if t, err := time.Parse(l, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

// ParseDocuments reads every document and diagram of the run.
func ParseDocuments(r *Run) *DModel {
	m := &DModel{}
	for _, s := range r.Code.Skipped {
		ext := strings.ToLower(path.Ext(s.Path))
		if ext == ".docx" || ext == ".pptx" || ext == ".vsdx" || ext == ".pdf" || ext == ".doc" || ext == ".vsd" || ext == ".ppt" {
			m.NotRead = append(m.NotRead, agentsSkip{s.Path, s.Reason})
		}
	}
	for _, f := range r.Code.Files {
		for i, l := range f.Lines {
			if looksSecret(l) {
				r.MarkSecret(f.Path, i+1)
			}
		}
		ext := strings.ToLower(path.Ext(f.Path))
		switch {
		case f.Lang == "Diagram" || (f.Lang == "XML" && strings.Contains(strings.Join(f.Lines[:min(5, len(f.Lines))], " "), "<mxfile")):
			if d := parseDrawio(f); d != nil {
				m.Diagrams = append(m.Diagrams, d)
				m.Files = append(m.Files, f.Path)
				r.Cover("drd", f)
			}
		case f.Lang == "Document" && ext == ".vsdx":
			d := &Diagram{F: f, Kind: "Visio"}
			for i, l := range f.Lines {
				switch {
				case strings.HasPrefix(l, "shape: "):
					d.Nodes = append(d.Nodes, Node{ID: l[7:], Label: l[7:], Line: i + 1})
				case strings.HasPrefix(l, "link: "):
					body := l[6:]
					label := ""
					if j := strings.LastIndex(body, " ("); j > 0 && strings.HasSuffix(body, ")") {
						label, body = body[j+2:len(body)-1], body[:j]
					}
					if a, z, ok := strings.Cut(body, " -> "); ok {
						d.Links = append(d.Links, Link{From: a, To: z, Label: label, Line: i + 1})
					}
				}
			}
			m.Diagrams = append(m.Diagrams, d)
			m.Files = append(m.Files, f.Path)
			r.Cover("drd", f)
		case f.Lang == "Document" || f.Lang == "Markdown" || f.Lang == "HTML" || (f.Lang == "Text" && !strings.Contains(strings.ToLower(f.Path), "readme")):
			m.Docs = append(m.Docs, readDoc(f, ext))
			m.Files = append(m.Files, f.Path)
			r.Cover("drd", f)
		}
	}
	return m
}

func readDoc(f *File, ext string) *Doc {
	d := &Doc{F: f, Kind: map[string]string{".docx": "Word", ".pptx": "PowerPoint", ".pdf": "PDF", ".md": "Markdown", ".html": "Web page", ".htm": "Web page", ".txt": "Text"}[ext]}
	if d.Kind == "" {
		d.Kind = f.Lang
	}
	d.Text = make([]string, len(f.Lines))
	for i, l := range f.Lines {
		t := l
		if f.Lang == "HTML" {
			if mm := htmlHead.FindStringSubmatch(l); mm != nil {
				d.Headings = append(d.Headings, Heading{Level: int(mm[1][0] - '0'), Text: strings.TrimSpace(html.UnescapeString(tagRe.ReplaceAllString(mm[2], ""))), Line: i + 1})
			}
			if strings.Contains(strings.ToLower(l), "<title>") && d.Title == "" {
				d.Title = strings.TrimSpace(html.UnescapeString(tagRe.ReplaceAllString(l, "")))
			}
			t = strings.NewReplacer("</td>", " | ", "</th>", " | ", "<li>", "- ").Replace(l)
			t = html.UnescapeString(tagRe.ReplaceAllString(t, ""))
			t = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(t), "|"))
		} else {
			t = strings.TrimSpace(l)
			if mm := mdHead.FindStringSubmatch(t); mm != nil {
				d.Headings = append(d.Headings, Heading{Level: len(mm[1]), Text: strings.TrimSpace(mm[2]), Line: i + 1})
			} else if mm := numHead.FindStringSubmatch(t); mm != nil && f.Lang != "Document" || mm != nil && d.Kind == "PDF" {
				d.Headings = append(d.Headings, Heading{Level: strings.Count(mm[1], ".") + 1, Text: strings.TrimSpace(mm[2]), Line: i + 1})
			}
		}
		d.Text[i] = t
		d.Words += len(strings.Fields(t))
		if draftRe.MatchString(t) {
			d.Drafts = append(d.Drafts, i+1)
		}
		if d.Date.IsZero() && i < 60 {
			if mm := dateLineRe.FindStringSubmatch(t); mm != nil {
				if dt := parseDate(mm[2]); !dt.IsZero() {
					d.Date, d.DateLine = dt, i+1
				}
			}
		}
		if d.Version == "" && i < 60 {
			if mm := versionRe.FindStringSubmatch(t); mm != nil {
				d.Version = mm[1]
			}
		}
	}
	if d.Title == "" && (d.Kind == "PowerPoint" || d.Kind == "PDF" || len(d.Headings) == 0) {
		for _, t := range d.Text { // the first real line is the title (slide headings are just "Slide N")
			if t != "" && !strings.HasPrefix(t, "#") && len(t) <= 80 {
				d.Title = t
				break
			}
		}
	}
	if d.Title == "" && len(d.Headings) > 0 {
		d.Title = d.Headings[0].Text
	}
	if d.Title == "" {
		d.Title = strings.TrimSuffix(path.Base(f.Path), path.Ext(f.Path))
	}
	return d
}

// Section returns the line range [from, to] of the section under heading i.
func (d *Doc) Section(i int) (int, int) {
	h := d.Headings[i]
	end := len(d.Text)
	for _, n := range d.Headings[i+1:] {
		if n.Level <= h.Level {
			end = n.Line - 1
			break
		}
	}
	return h.Line, end
}

// ---------- draw.io ----------

type mxCell struct {
	ID     string `xml:"id,attr"`
	Value  string `xml:"value,attr"`
	Vertex string `xml:"vertex,attr"`
	Edge   string `xml:"edge,attr"`
	Source string `xml:"source,attr"`
	Target string `xml:"target,attr"`
	Parent string `xml:"parent,attr"`
}

var diagramRe = regexp.MustCompile(`(?s)<diagram[^>]*>(.*?)</diagram>`)

func inflateDrawio(s string) string {
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(s))
	if err != nil {
		return ""
	}
	b, err := io.ReadAll(flate.NewReader(bytes.NewReader(raw)))
	if err != nil {
		return ""
	}
	out, err := url.QueryUnescape(string(b))
	if err != nil {
		return string(b)
	}
	return out
}

func parseDrawio(f *File) *Diagram {
	text := strings.Join(f.Lines, "\n")
	d := &Diagram{F: f, Kind: "draw.io"}
	labels := map[string]string{}
	type edge struct {
		c    mxCell
		line int
	}
	var edges []edge
	scan := func(xmlText string, lineFor func(id string) int) {
		dec := xml.NewDecoder(strings.NewReader(xmlText))
		for {
			tok, err := dec.Token()
			if err != nil {
				return
			}
			se, ok := tok.(xml.StartElement)
			if !ok || se.Name.Local != "mxCell" {
				continue
			}
			var c mxCell
			if dec.DecodeElement(&c, &se) != nil {
				continue
			}
			label := strings.Join(strings.Fields(html.UnescapeString(tagRe.ReplaceAllString(strings.ReplaceAll(c.Value, "<br>", " "), " "))), " ")
			switch {
			case c.Vertex == "1" && label != "":
				labels[c.ID] = label
				d.Nodes = append(d.Nodes, Node{ID: c.ID, Label: label, Line: lineFor(c.ID)})
			case c.Edge == "1":
				c.Value = label
				edges = append(edges, edge{c, lineFor(c.ID)})
			}
		}
	}
	lineOfID := func(id string) int {
		for i, l := range f.Lines {
			if strings.Contains(l, `id="`+id+`"`) {
				return i + 1
			}
		}
		return 1
	}
	if strings.Contains(text, "<mxGraphModel") {
		scan(text, lineOfID)
	} else {
		for _, mm := range diagramRe.FindAllStringSubmatchIndex(text, -1) {
			inner := inflateDrawio(text[mm[2]:mm[3]])
			line := strings.Count(text[:mm[0]], "\n") + 1
			scan(inner, func(string) int { return line })
		}
	}
	for _, e := range edges {
		a, z := labels[e.c.Source], labels[e.c.Target]
		if a != "" && z != "" {
			d.Links = append(d.Links, Link{From: a, To: z, Label: e.c.Value, Line: e.line})
		}
	}
	if len(d.Nodes) == 0 {
		return nil
	}
	return d
}
