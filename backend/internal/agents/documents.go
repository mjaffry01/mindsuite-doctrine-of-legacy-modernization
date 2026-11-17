package agents

import (
	"archive/zip"
	"bytes"
	"compress/zlib"
	"encoding/xml"
	"fmt"
	"io"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Document text extraction for the loader. Office files and PDFs are binary; their text is extracted once,
// in memory, so every team can read and cite them by line like any text file. Nothing is converted on disk.
//
// Word (.docx): one line per paragraph; headings start with "#" marks ("## Settlement"); table rows become
//               "cell | cell | cell".
// PowerPoint (.pptx): "Slide N" lines, then one line per paragraph.
// Visio (.vsdx): "Page: name", then "shape: text" lines and "link: A -> B" lines for connectors.
// PDF: text from the page content streams (Tj/TJ operators). PDFs without a text layer (scans) or with
//      custom font encodings give little or no text: they are reported as not readable.

// MaxDocBytes is the largest office document or PDF that is read.
const MaxDocBytes = 20 << 20

var docExts = map[string]bool{".docx": true, ".pptx": true, ".vsdx": true, ".pdf": true}

// ExtractDocument returns the text of an office document or PDF.
func ExtractDocument(name string, data []byte) (string, error) {
	switch strings.ToLower(path.Ext(name)) {
	case ".docx":
		return docxText(data)
	case ".pptx":
		return pptxText(data)
	case ".vsdx":
		return vsdxText(data)
	case ".pdf":
		return pdfText(data)
	}
	return "", fmt.Errorf("not a document")
}

func zipFiles(data []byte) (map[string]*zip.File, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	out := map[string]*zip.File{}
	for _, f := range zr.File {
		out[f.Name] = f
	}
	return out, nil
}

func readZip(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(io.LimitReader(rc, 64<<20))
}

func attr(se xml.StartElement, local string) string {
	for _, a := range se.Attr {
		if a.Name.Local == local {
			return a.Value
		}
	}
	return ""
}

func docxText(data []byte) (string, error) {
	files, err := zipFiles(data)
	if err != nil {
		return "", err
	}
	f := files["word/document.xml"]
	if f == nil {
		return "", fmt.Errorf("no word/document.xml")
	}
	raw, err := readZip(f)
	if err != nil {
		return "", err
	}
	dec := xml.NewDecoder(bytes.NewReader(raw))
	var lines []string
	var para strings.Builder
	level := 0
	inTable := 0
	var cell strings.Builder
	var row []string
	flushPara := func() {
		t := strings.TrimSpace(para.String())
		para.Reset()
		if inTable > 0 {
			if t != "" {
				if cell.Len() > 0 {
					cell.WriteString(" ")
				}
				cell.WriteString(t)
			}
			return
		}
		if t != "" {
			if level > 0 {
				t = strings.Repeat("#", level) + " " + t
			}
			lines = append(lines, t)
		}
		level = 0
	}
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch x := tok.(type) {
		case xml.StartElement:
			switch x.Name.Local {
			case "pStyle":
				v := strings.ReplaceAll(attr(x, "val"), " ", "")
				if m := regexp.MustCompile(`(?i)^(heading|berschrift|titre)(\d)$`).FindStringSubmatch(v); m != nil {
					level, _ = strconv.Atoi(m[2])
				} else if strings.EqualFold(v, "Title") {
					level = 1
				}
			case "tab":
				para.WriteString("\t")
			case "br", "cr":
				para.WriteString(" ")
			case "tbl":
				inTable++
			}
		case xml.CharData:
			para.Write(x)
		case xml.EndElement:
			switch x.Name.Local {
			case "p":
				flushPara()
			case "tc":
				row = append(row, strings.TrimSpace(cell.String()))
				cell.Reset()
			case "tr":
				if len(row) > 0 && strings.TrimSpace(strings.Join(row, "")) != "" {
					lines = append(lines, strings.Join(row, " | "))
				}
				row = nil
			case "tbl":
				inTable--
			}
		}
	}
	// CharData outside <w:t> (e.g. instrText) is rare in body text; accept it.
	return strings.Join(lines, "\n"), nil
}

func numbered(files map[string]*zip.File, re *regexp.Regexp) []string {
	type nf struct {
		n    int
		name string
	}
	var out []nf
	for name := range files {
		if m := re.FindStringSubmatch(name); m != nil {
			n, _ := strconv.Atoi(m[1])
			out = append(out, nf{n, name})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].n < out[j].n })
	names := []string{}
	for _, x := range out {
		names = append(names, x.name)
	}
	return names
}

func paragraphsOf(raw []byte, paraTag string) []string {
	dec := xml.NewDecoder(bytes.NewReader(raw))
	var lines []string
	var b strings.Builder
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch x := tok.(type) {
		case xml.CharData:
			b.Write(x)
		case xml.EndElement:
			if x.Name.Local == paraTag {
				if t := strings.TrimSpace(b.String()); t != "" {
					lines = append(lines, t)
				}
				b.Reset()
			}
		}
	}
	return lines
}

func pptxText(data []byte) (string, error) {
	files, err := zipFiles(data)
	if err != nil {
		return "", err
	}
	var lines []string
	for i, name := range numbered(files, regexp.MustCompile(`^ppt/slides/slide(\d+)\.xml$`)) {
		raw, err := readZip(files[name])
		if err != nil {
			continue
		}
		lines = append(lines, fmt.Sprintf("# Slide %d", i+1))
		lines = append(lines, paragraphsOf(raw, "p")...)
	}
	if len(lines) == 0 {
		return "", fmt.Errorf("no slides")
	}
	return strings.Join(lines, "\n"), nil
}

func vsdxText(data []byte) (string, error) {
	files, err := zipFiles(data)
	if err != nil {
		return "", err
	}
	var lines []string
	for i, name := range numbered(files, regexp.MustCompile(`^visio/pages/page(\d+)\.xml$`)) {
		raw, err := readZip(files[name])
		if err != nil {
			continue
		}
		lines = append(lines, fmt.Sprintf("# Page %d", i+1))
		dec := xml.NewDecoder(bytes.NewReader(raw))
		texts := map[string]string{}
		var order []string
		shapeStack := []string{}
		inText := 0
		var b strings.Builder
		type conn struct{ from, cell, to string }
		var conns []conn
		for {
			tok, err := dec.Token()
			if err != nil {
				break
			}
			switch x := tok.(type) {
			case xml.StartElement:
				switch x.Name.Local {
				case "Shape":
					shapeStack = append(shapeStack, attr(x, "ID"))
				case "Text":
					inText++
					b.Reset()
				case "Connect":
					conns = append(conns, conn{attr(x, "FromSheet"), attr(x, "FromCell"), attr(x, "ToSheet")})
				}
			case xml.CharData:
				if inText > 0 {
					b.Write(x)
				}
			case xml.EndElement:
				switch x.Name.Local {
				case "Text":
					inText--
					if len(shapeStack) > 0 {
						id := shapeStack[len(shapeStack)-1]
						if t := strings.Join(strings.Fields(b.String()), " "); t != "" {
							if _, ok := texts[id]; !ok {
								order = append(order, id)
							}
							texts[id] = t
						}
					}
				case "Shape":
					if len(shapeStack) > 0 {
						shapeStack = shapeStack[:len(shapeStack)-1]
					}
				}
			}
		}
		begin, end := map[string]string{}, map[string]string{}
		for _, c := range conns {
			switch c.cell {
			case "BeginX":
				begin[c.from] = c.to
			case "EndX":
				end[c.from] = c.to
			}
		}
		isConnector := map[string]bool{}
		for id := range begin {
			isConnector[id] = true
		}
		for _, id := range order {
			if !isConnector[id] {
				lines = append(lines, "shape: "+texts[id])
			}
		}
		for _, id := range sortedKeys(begin) {
			a, z := texts[begin[id]], texts[end[id]]
			if a == "" || z == "" {
				continue
			}
			l := "link: " + a + " -> " + z
			if t := texts[id]; t != "" {
				l += " (" + t + ")"
			}
			lines = append(lines, l)
		}
	}
	if len(lines) == 0 {
		return "", fmt.Errorf("no pages")
	}
	return strings.Join(lines, "\n"), nil
}

var (
	pdfStreamRe = regexp.MustCompile(`(?s)stream\r?\n(.*?)\r?\nendstream`)
	pdfOpRe     = regexp.MustCompile(`(?s)\((?:\\.|[^\\)])*\)\s*(?:Tj|'|")|\[(?:[^\]\\]|\\.)*\]\s*TJ|T\*|ET|-?[\d.]+\s+-?[\d.]+\s+T[dD]`)
	pdfStrRe    = regexp.MustCompile(`\((?:\\.|[^\\)])*\)`)
	letters     = regexp.MustCompile(`[A-Za-z]{3,}`)
)

func pdfUnescape(s string) string {
	s = s[1 : len(s)-1]
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c != '\\' || i+1 >= len(s) {
			b.WriteByte(c)
			continue
		}
		i++
		switch s[i] {
		case 'n', 'r':
			b.WriteByte(' ')
		case 't':
			b.WriteByte('\t')
		case '(', ')', '\\':
			b.WriteByte(s[i])
		default:
			if s[i] >= '0' && s[i] <= '7' {
				j := i
				for j < len(s) && j < i+3 && s[j] >= '0' && s[j] <= '7' {
					j++
				}
				v, _ := strconv.ParseInt(s[i:j], 8, 32)
				b.WriteByte(byte(v))
				i = j - 1
			}
		}
	}
	return b.String()
}

func pdfText(data []byte) (string, error) {
	var lines []string
	var cur strings.Builder
	flush := func() {
		if t := strings.Join(strings.Fields(cur.String()), " "); t != "" {
			lines = append(lines, t)
		}
		cur.Reset()
	}
	for _, m := range pdfStreamRe.FindAllSubmatch(data, -1) {
		content := m[1]
		if zr, err := zlib.NewReader(bytes.NewReader(content)); err == nil {
			if b, err := io.ReadAll(io.LimitReader(zr, 32<<20)); err == nil || len(b) > 0 {
				content = b
			}
		}
		if !bytes.Contains(content, []byte("Tj")) && !bytes.Contains(content, []byte("TJ")) {
			continue
		}
		for _, op := range pdfOpRe.FindAll(content, -1) {
			s := string(op)
			switch {
			case s == "ET" || s == "T*" || strings.HasSuffix(s, "Td") || strings.HasSuffix(s, "TD"):
				if strings.HasSuffix(s, "Td") || strings.HasSuffix(s, "TD") {
					if f := strings.Fields(s); len(f) == 3 && f[1] == "0" {
						continue // same line, moved right
					}
				}
				flush()
			default:
				for _, str := range pdfStrRe.FindAllString(s, -1) {
					cur.WriteString(pdfUnescape(str))
				}
				if strings.HasSuffix(s, "'") || strings.HasSuffix(s, "\"") {
					flush()
				}
			}
		}
		flush()
	}
	text := strings.Join(lines, "\n")
	if len(letters.FindAllString(text, 20)) < 5 {
		return "", fmt.Errorf("no readable text layer (scanned, or fonts with a custom encoding)")
	}
	return text, nil
}
