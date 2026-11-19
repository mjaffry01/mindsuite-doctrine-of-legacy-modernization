package server

import (
	"archive/zip"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

type detectedSystemType struct {
	Type string `json:"type"`
	File string `json:"file"`
}

type evidenceProfile struct {
	System   string               `json:"system,omitempty"`
	NameFile string               `json:"nameFile,omitempty"`
	Types    []detectedSystemType `json:"types,omitempty"`
}

type profileFile struct {
	name string
	open func() (io.ReadCloser, error)
}

// Profile only inspects filenames and a bounded set of project manifests.
// The full analysis still runs from the workspace after Start.
func profileSource(dir string, upload evidenceUpload) evidenceProfile {
	if upload.Team != "code" {
		return evidenceProfile{}
	}
	root := filepath.Join(dir, "files")
	var candidates []profileFile
	if upload.Zip {
		reader, err := zip.OpenReader(filepath.Join(root, upload.Files[0]))
		if err != nil {
			return evidenceProfile{}
		}
		defer reader.Close()
		for _, file := range reader.File {
			if !file.FileInfo().IsDir() {
				candidates = append(candidates, profileFile{file.Name, file.Open})
			}
		}
	} else {
		for _, name := range upload.Files {
			candidates = append(candidates, profileFile{name, func() (io.ReadCloser, error) {
				return os.Open(filepath.Join(root, filepath.FromSlash(name)))
			}})
		}
	}
	return inferEvidenceProfile(candidates)
}

func inferEvidenceProfile(files []profileFile) evidenceProfile {
	var result evidenceProfile
	detected := map[string]bool{}
	names := map[string]string{}
	manifests := 0
	add := func(kind, file string) {
		if !detected[kind] {
			detected[kind] = true
			result.Types = append(result.Types, detectedSystemType{kind, file})
		}
	}
	for _, file := range files {
		name := strings.ReplaceAll(file.name, "\\", "/")
		lower := strings.ToLower(name)
		if skipProfilePath(lower) {
			continue
		}
		switch path.Ext(lower) {
		case ".aspx", ".jsp":
			add("web", name)
		case ".cbl", ".cob", ".jcl", ".rpgle":
			add("mainframe", name)
		case ".fmb", ".svc":
			add("clientserver", name)
		case ".rdl", ".dtsx", ".pbip":
			add("data", name)
		case ".abap":
			add("packaged", name)
		case ".bpel":
			add("integration", name)
		case ".l5x", ".acd":
			add("embedded", name)
		case ".csproj", ".vbproj", ".fsproj":
			base := path.Base(name)
			stem := strings.TrimSuffix(base, path.Ext(base))
			if strings.HasSuffix(strings.ToLower(stem), ".tests") || strings.HasSuffix(strings.ToLower(stem), ".test") {
				continue
			}
			names[stem] = name
			if manifests >= 24 {
				continue
			}
			manifests++
			handle, err := file.open()
			if err != nil {
				continue
			}
			content, err := io.ReadAll(io.LimitReader(handle, 96<<10))
			handle.Close()
			if err != nil {
				continue
			}
			xml := strings.ToLower(string(content))
			if strings.Contains(xml, "<usewindowsforms>true</usewindowsforms>") ||
				strings.Contains(xml, "<usewpf>true</usewpf>") ||
				strings.Contains(xml, "system.windows.forms") ||
				strings.Contains(xml, "<subtype>form</subtype>") {
				add("desktop", name)
			}
			if strings.Contains(xml, "microsoft.net.sdk.web") || strings.Contains(xml, "system.web") {
				add("web", name)
			}
		}
	}
	if len(names) == 1 {
		for name, file := range names {
			result.System, result.NameFile = safeFolderName(name), file
		}
	}
	sort.Slice(result.Types, func(i, j int) bool { return result.Types[i].Type < result.Types[j].Type })
	return result
}

func skipProfilePath(name string) bool {
	for _, part := range strings.Split(name, "/") {
		switch part {
		case "..", ".git", ".vs", "node_modules", "bin", "obj", "vendor", "packages", "test", "tests":
			return true
		}
	}
	return false
}
