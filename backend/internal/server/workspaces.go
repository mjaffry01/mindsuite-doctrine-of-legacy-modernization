package server

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/mjaffry01/mindsuite-doctrine-of-legacy-modernization/backend/internal/agents"
)

// Saved results are laid out per project and per agent team:
//
//	<runs>/workspaces/<project name> (<first 8 characters of the workspace id>)/
//	    workspace.json                 the workspace id and project name
//	    01 Source code/<run id>/       result.json, report.md, messages
//	    02 Database/<run id>/
//	    …
//	    10 Incidents and changes/<run id>/
//
// The workspace id stays the key: the folder is found by its id suffix, and renamed when the project is renamed.
// Results saved before this layout (<runs>/workspaces/<workspace id>/<run id>/) are moved into it on first use.

var workspaceID = regexp.MustCompile("^[a-f0-9]{8}(-[a-f0-9]{4}){3}-[a-f0-9]{12}$")
var savedRunID = regexp.MustCompile("^[0-9]{8}-[0-9]{6}-[a-f0-9]{12}$")

// teamFolders names the folder of each agent team, in the order of the web app.
var teamFolders = []struct{ team, folder string }{
	{"code", "01 Source code"},
	{"database", "02 Database"},
	{"infrastructure", "03 Environment and infrastructure"},
	{"messaging", "04 Messaging and caching"},
	{"identity", "05 Identity and access"},
	{"telemetry", "06 Telemetry"},
	{"batch", "07 Batch and scheduling"},
	{"network", "08 Network and edge"},
	{"documents", "09 Documents and diagrams"},
	{"incidents", "10 Incidents and changes"},
}

func teamFolder(team string) string {
	if team == "" {
		team = "code"
	}
	for _, t := range teamFolders {
		if t.team == team {
			return t.folder
		}
	}
	return safeFolderName(team)
}

var unsafeFolderChars = regexp.MustCompile(`[\\/:*?"<>|\x00-\x1f]+`)

// safeFolderName turns a project name into a folder name that is valid on Windows, macOS and Linux.
func safeFolderName(name string) string {
	n := unsafeFolderChars.ReplaceAllString(name, " ")
	n = strings.Join(strings.Fields(n), " ")
	if len([]rune(n)) > 70 {
		n = string([]rune(n)[:70])
	}
	n = strings.TrimRight(strings.TrimSpace(n), ". ")
	if n == "" {
		return "Project"
	}
	return n
}

type workspaceRef struct {
	id, name string
}

func workspaceFrom(w http.ResponseWriter, r *http.Request) (workspaceRef, bool) {
	q := r.URL.Query()
	ref := workspaceRef{id: q.Get("workspace"), name: strings.TrimSpace(q.Get("project"))}
	if ref.id != "" && !workspaceID.MatchString(ref.id) {
		httpErr(w, 400, "invalid project workspace")
		return ref, false
	}
	if len(ref.name) > 200 {
		ref.name = ref.name[:200]
	}
	return ref, true
}

// workspaceDir returns the project folder of a workspace, creating, renaming or migrating it as needed.
// It returns "" when results are not saved.
func (s *Server) workspaceDir(ref workspaceRef) string {
	if s.cfg.RunsDir == "" || ref.id == "" {
		return ""
	}
	s.dirMu.Lock()
	defer s.dirMu.Unlock()
	root := filepath.Join(s.cfg.RunsDir, "workspaces")
	suffix := " (" + ref.id[:8] + ")"
	existing, legacy := "", ""
	entries, _ := os.ReadDir(root)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		switch {
		case e.Name() == ref.id:
			legacy = filepath.Join(root, e.Name())
		case strings.HasSuffix(e.Name(), suffix) && folderBelongs(filepath.Join(root, e.Name()), ref.id):
			existing = filepath.Join(root, e.Name())
		}
	}
	name := ref.name
	want := filepath.Join(root, safeFolderName(name)+suffix)
	dir := existing
	switch {
	case dir == "" && legacy != "":
		dir = legacy
		if os.Rename(legacy, want) == nil {
			dir = want
		}
	case dir == "":
		dir = want
	case name != "" && dir != want && !s.hasLiveRun(ref.id):
		// The project was renamed: follow it, unless a run is still writing into the old folder.
		if os.Rename(dir, want) == nil {
			dir = want
		}
	}
	os.MkdirAll(dir, 0o755)
	for _, t := range teamFolders {
		os.MkdirAll(filepath.Join(dir, t.folder), 0o755)
	}
	if name != "" || !fileExists(filepath.Join(dir, "workspace.json")) {
		meta, _ := json.MarshalIndent(map[string]string{"workspace": ref.id, "project": name}, "", "  ")
		os.WriteFile(filepath.Join(dir, "workspace.json"), meta, 0o644)
	}
	sortRunsIntoTeams(dir)
	return dir
}

// folderBelongs is true when the folder's workspace.json names this workspace (or there is none to say otherwise).
func folderBelongs(dir, id string) bool {
	b, err := os.ReadFile(filepath.Join(dir, "workspace.json"))
	if err != nil {
		return true
	}
	var meta struct{ Workspace string }
	return json.Unmarshal(b, &meta) != nil || meta.Workspace == "" || meta.Workspace == id
}

func (s *Server) hasLiveRun(id string) bool {
	s.mu.Lock()
	runs := make([]*agents.Run, 0, len(s.runs))
	for _, run := range s.runs {
		if run.Options.Workspace == id {
			runs = append(runs, run)
		}
	}
	s.mu.Unlock()
	for _, run := range runs {
		if run.Snapshot().Run.Finished.IsZero() {
			return true
		}
	}
	return false
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// sortRunsIntoTeams moves run folders saved directly in the project folder (the earlier layout) into their team folder.
func sortRunsIntoTeams(dir string) {
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if !e.IsDir() || !savedRunID.MatchString(e.Name()) {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name(), "result.json"))
		if err != nil {
			continue
		}
		var result struct {
			Run struct {
				Options struct {
					Team string `json:"team"`
				} `json:"options"`
			} `json:"run"`
		}
		if json.Unmarshal(b, &result) != nil {
			continue
		}
		target := filepath.Join(dir, teamFolder(result.Run.Options.Team), e.Name())
		if !fileExists(target) {
			os.Rename(filepath.Join(dir, e.Name()), target)
		}
	}
}

// runFolders lists the folders that hold this workspace's runs: one per team, plus the project folder itself.
func runFolders(dir string) []string {
	out := make([]string, 0, len(teamFolders)+1)
	for _, t := range teamFolders {
		out = append(out, filepath.Join(dir, t.folder))
	}
	return append(out, dir)
}

// runBase is the folder a new run of this team is saved in.
func (s *Server) runBase(ref workspaceRef, team string) string {
	if s.cfg.RunsDir == "" || ref.id == "" {
		return s.cfg.RunsDir
	}
	return filepath.Join(s.workspaceDir(ref), teamFolder(team))
}

func (s *Server) savedResult(ref workspaceRef, id string) (agents.Result, bool) {
	var result agents.Result
	if s.cfg.RunsDir == "" || !savedRunID.MatchString(id) {
		return result, false
	}
	folders := []string{s.cfg.RunsDir}
	if ref.id != "" {
		folders = runFolders(s.workspaceDir(ref))
	}
	for _, folder := range folders {
		file, err := os.Open(filepath.Join(folder, id, "result.json"))
		if err != nil {
			continue
		}
		var found agents.Result
		err = json.NewDecoder(file).Decode(&found)
		file.Close()
		if err == nil && found.Run != nil && found.Run.ID == id && found.Run.Options.Workspace == ref.id {
			return found, true
		}
	}
	return result, false
}

type runListItem struct {
	ID      string    `json:"id"`
	Status  string    `json:"status"`
	Started time.Time `json:"started"`
	Team    string    `json:"team"`
	Source  any       `json:"source"`
}

func listItem(result agents.Result) runListItem {
	run := result.Run
	return runListItem{ID: run.ID, Status: run.Status, Started: run.Started, Team: run.Options.Team, Source: result.Source}
}

func (s *Server) resultsFor(ref workspaceRef) []runListItem {
	current := make(map[string]*agents.Run)
	s.mu.Lock()
	for id, run := range s.runs {
		if run.Options.Workspace == ref.id {
			current[id] = run
		}
	}
	s.mu.Unlock()
	results := make([]runListItem, 0, len(current))
	for _, run := range current {
		results = append(results, listItem(run.Snapshot()))
	}
	if ref.id == "" || s.cfg.RunsDir == "" {
		return results
	}
	seen := map[string]bool{}
	for _, folder := range runFolders(s.workspaceDir(ref)) {
		entries, _ := os.ReadDir(folder)
		for _, entry := range entries {
			if !entry.IsDir() || current[entry.Name()] != nil || seen[entry.Name()] || !savedRunID.MatchString(entry.Name()) {
				continue
			}
			seen[entry.Name()] = true
			if result, ok := s.savedResult(ref, entry.Name()); ok {
				results = append(results, listItem(result))
				continue
			}
			started, _ := time.Parse("20060102-150405", strings.Join(strings.Split(entry.Name(), "-")[:2], "-"))
			results = append(results, runListItem{ID: entry.Name(), Status: "interrupted", Started: started})
		}
	}
	return results
}
