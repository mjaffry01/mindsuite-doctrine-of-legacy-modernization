package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/mjaffry01/mindsuite-doctrine-of-legacy-modernization/backend/internal/agents"
)

type evidenceUpload struct {
	ID      string           `json:"id"`
	Team    string           `json:"team"`
	Name    string           `json:"name"`
	Files   []string         `json:"files"`
	Zip     bool             `json:"zip"`
	Profile *evidenceProfile `json:"profile,omitempty"`
}

type evidenceResult struct {
	evidenceUpload
	Placed        bool   `json:"placed"`
	RunID         string `json:"runId,omitempty"`
	Error         string `json:"error,omitempty"`
	CheckpointRun string `json:"checkpointRun,omitempty"`
}

var evidenceID = regexp.MustCompile("^[a-f0-9]{32}$")

func (s *Server) evidenceWorkspace(w http.ResponseWriter, r *http.Request) (workspaceRef, bool) {
	ref, ok := workspaceFrom(w, r)
	if !ok {
		return ref, false
	}
	if ref.id == "" || s.cfg.RunsDir == "" {
		httpErr(w, 400, "evidence needs a project workspace and a backend configured to save files")
		return ref, false
	}
	return ref, true
}

func (s *Server) stagingDir(ref workspaceRef) string {
	return filepath.Join(s.cfg.RunsDir, "staging", ref.id)
}

// Upload names are display metadata. Relative paths are validated independently
// and every batch has its own directory, so repeated filenames never overwrite.
func (s *Server) stageEvidence(w http.ResponseWriter, r *http.Request) {
	ref, ok := s.evidenceWorkspace(w, r)
	if !ok {
		return
	}
	team := r.URL.Query().Get("team")
	if team == "" || agents.TeamByID(team) == nil {
		httpErr(w, 400, "choose a known evidence type")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, MaxUpload)
	reader, err := r.MultipartReader()
	if err != nil {
		httpErr(w, 400, "upload a file, folder, or ZIP")
		return
	}
	dir := s.stagingDir(ref)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		httpErr(w, 500, "could not prepare temporary storage")
		return
	}
	tmp, err := os.MkdirTemp(dir, ".upload-")
	if err != nil {
		httpErr(w, 500, "could not prepare temporary storage")
		return
	}
	defer os.RemoveAll(tmp)
	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		httpErr(w, 500, "could not allocate an upload ID")
		return
	}
	upload := evidenceUpload{ID: hex.EncodeToString(random), Team: team}
	if err := receiveEvidence(tmp, reader.NextPart, &upload); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			httpErr(w, 413, "upload exceeds 512 MB")
		} else {
			httpErr(w, 400, err.Error())
		}
		return
	}
	if team == "code" {
		profile := profileSource(tmp, upload)
		upload.Profile = &profile
	}
	if err := writeEvidenceJSON(filepath.Join(tmp, "upload.json"), upload); err != nil {
		httpErr(w, 500, "could not record the upload")
		return
	}
	if err := os.Rename(tmp, filepath.Join(dir, upload.ID)); err != nil {
		httpErr(w, 500, "could not finish staging the upload")
		return
	}
	writeJSON(w, 201, upload)
}

func receiveEvidence(dir string, next func() (*multipart.Part, error), input *evidenceUpload) error {
	root := filepath.Join(dir, "files")
	if err := os.Mkdir(root, 0o700); err != nil {
		return err
	}
	seen := map[string]bool{}
	for {
		p, err := next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if p.FormName() == "name" {
			data, err := io.ReadAll(io.LimitReader(p, 601))
			p.Close()
			if err != nil {
				return err
			}
			if len(data) > 600 {
				return errors.New("upload name is too long")
			}
			input.Name = safeFolderName(string(data))
			continue
		}
		if p.FormName() != "file" && p.FormName() != "zip" {
			p.Close()
			continue
		}
		if input.Zip || (p.FormName() == "zip" && len(input.Files) != 0) || len(input.Files) >= 20000 {
			return errors.New("upload one archive or a set of files")
		}
		_, params, err := mime.ParseMediaType(p.Header.Get("Content-Disposition"))
		name := params["filename"]
		local, bad := filepath.Localize(name)
		if err != nil || bad != nil || name == "" || name == "." || strings.ContainsAny(name, "\\\\:") {
			return errors.New("invalid evidence filename")
		}
		key := strings.ToLower(name)
		if seen[key] {
			return errors.New("duplicate evidence filename")
		}
		seen[key] = true
		if p.FormName() == "zip" {
			if strings.Contains(name, "/") || !strings.EqualFold(path.Ext(name), ".zip") {
				return errors.New("choose a ZIP archive")
			}
			input.Zip = true
		}
		target := filepath.Join(root, local)
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			return errors.New("could not prepare evidence path")
		}
		file, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			return errors.New("could not save evidence file")
		}
		_, copyErr := io.Copy(file, p)
		closeErr := file.Close()
		p.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		input.Files = append(input.Files, name)
	}
	if len(input.Files) == 0 {
		return errors.New("no evidence files were uploaded")
	}
	if input.Name == "" {
		base := path.Base(input.Files[0])
		input.Name = safeFolderName(strings.TrimSuffix(base, path.Ext(base)))
	}
	return nil
}

func writeEvidenceJSON(file string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(file), ".metadata-")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	_, err = f.Write(data)
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(f.Name(), file)
}

func readEvidenceJSON(file string, value any) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, value)
}

func evidenceCode(dir string, upload evidenceUpload) (*agents.Codebase, error) {
	root := filepath.Join(dir, "files")
	if !upload.Zip {
		code, err := agents.LoadDir(root)
		if err == nil {
			code.Root = upload.Name
		}
		return code, err
	}
	file, err := os.Open(filepath.Join(root, upload.Files[0]))
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	return agents.LoadZip(file, info.Size(), upload.Name)
}

func (s *Server) startEvidence(w http.ResponseWriter, r *http.Request) {
	ref, ok := s.evidenceWorkspace(w, r)
	if !ok {
		return
	}
	var req struct {
		Name       string
		Uploads    []string
		AllowModel bool
		Interview  *agents.InterviewFocus
		Reanalyze  bool
		Resume     bool
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 32768)).Decode(&req); err != nil || len(req.Uploads) > 100 {
		httpErr(w, 400, "invalid evidence start request")
		return
	}
	if req.Interview != nil && !validInterviewFocus(req.Interview) {
		httpErr(w, 400, "interview answers are too long or the deadline format is invalid")
		return
	}
	ref.name = strings.TrimSpace(req.Name)
	if ref.name == "" || len(ref.name) > 600 {
		httpErr(w, 400, "enter a project name before starting")
		return
	}
	for _, id := range req.Uploads {
		if !evidenceID.MatchString(id) {
			httpErr(w, 400, "invalid evidence upload")
			return
		}
	}
	s.evidenceMu.Lock()
	defer s.evidenceMu.Unlock()
	project := s.workspaceDir(ref)
	if !fileExists(filepath.Join(project, "workspace.json")) {
		httpErr(w, 500, "could not create the project workspace folders")
		return
	}
	results := make([]evidenceResult, 0, len(req.Uploads))
	seen := map[string]bool{}
	for _, id := range req.Uploads {
		if !seen[id] {
			results = append(results, s.placeAndRun(ref, project, id, agents.Options{AllowModel: req.AllowModel, Interview: req.Interview}, req.Reanalyze, req.Resume))
			seen[id] = true
		}
	}
	writeJSON(w, 200, map[string]any{"project": ref.name, "folder": filepath.Base(project), "uploads": results})
}

func validInterviewFocus(focus *agents.InterviewFocus) bool {
	if len(focus.Reasons) > 40 {
		return false
	}
	for _, reason := range focus.Reasons {
		if len(reason) > 64 {
			return false
		}
	}
	for _, value := range []string{focus.Problem, focus.Success, focus.System, focus.Objective, focus.OutOfScope} {
		if utf8.RuneCountInString(value) > 1000 {
			return false
		}
	}
	if focus.Deadline != "" {
		_, err := time.Parse("2006-01-02", focus.Deadline)
		return err == nil
	}
	return true
}

func (s *Server) placeAndRun(ref workspaceRef, project, id string, opts agents.Options, reanalyze, resume bool) evidenceResult {
	out := evidenceResult{evidenceUpload: evidenceUpload{ID: id}}
	staged := filepath.Join(s.stagingDir(ref), id)
	var placed string
	if err := readEvidenceJSON(filepath.Join(staged, "upload.json"), &out.evidenceUpload); err == nil {
		placed = filepath.Join(project, teamFolder(out.Team), "evidence", id)
		if err := os.MkdirAll(filepath.Dir(placed), 0o700); err != nil || os.Rename(staged, placed) != nil {
			out.Error = "Could not move the upload into the project. Try Start again."
			return out
		}
	} else {
		for _, team := range teamFolders {
			dir := filepath.Join(project, team.folder, "evidence", id)
			if readEvidenceJSON(filepath.Join(dir, "upload.json"), &out.evidenceUpload) == nil {
				placed = dir
				break
			}
		}
	}
	if placed == "" {
		out.Error = "This upload was not found in this workspace. Upload it again."
		return out
	}
	out.Placed = true
	if out.Team != "code" && out.Team != "database" {
		reanalyze = false
		opts.Interview = nil
	}
	var previous evidenceResult
	teamDir := filepath.Join(project, teamFolder(out.Team))
	if readEvidenceJSON(filepath.Join(placed, "run.json"), &previous) == nil && savedRunID.MatchString(previous.RunID) {
		live := s.liveRun(ref.id, previous.RunID)
		if (reanalyze || resume) && live != nil && live.IsRunning() {
			out.Error = "An analysis is already running for this upload."
			return out
		}
		resultPath := filepath.Join(teamDir, previous.RunID, "result.json")
		if resume {
			var saved agents.Result
			if readEvidenceJSON(resultPath, &saved) == nil && saved.Run != nil && saved.Run.Status == "done" {
				out.Error = "This analysis already completed."
				return out
			}
			checkpointRun := previous.CheckpointRun
			if !savedRunID.MatchString(checkpointRun) {
				checkpointRun = previous.RunID
			}
			out.CheckpointRun = checkpointRun
			opts.BatchDir = filepath.Join(teamDir, checkpointRun, "checkpoints")
		} else if !reanalyze && (live != nil || fileExists(resultPath)) {
			out.RunID = previous.RunID
			return out
		}
	}
	code, err := evidenceCode(placed, out.evidenceUpload)
	if err != nil || len(code.Files) == 0 {
		out.Error = "Evidence is saved in the project. The team could not read any files in this upload."
		return out
	}
	opts.Workspace, opts.Team = ref.id, out.Team
	run, err := agents.NewRun(code, opts, s.llm, teamDir)
	if err != nil {
		out.Error = "Evidence is saved in the project. Could not create the analysis run."
		return out
	}
	out.RunID = run.ID
	out.Error = ""
	if err := writeEvidenceJSON(filepath.Join(placed, "run.json"), out); err != nil {
		run.Store.Close()
		out.RunID = ""
		out.Error = "Evidence is saved in the project. Could not record the analysis run."
		return out
	}
	s.mu.Lock()
	s.runs[run.ID] = run
	s.mu.Unlock()
	go run.Execute(context.Background())
	return out
}

func (s *Server) listStagedEvidence(w http.ResponseWriter, r *http.Request) {
	ref, ok := s.evidenceWorkspace(w, r)
	if !ok {
		return
	}
	items := []evidenceUpload{}
	root := s.stagingDir(ref)
	entries, _ := os.ReadDir(root)
	for _, entry := range entries {
		if !entry.IsDir() || !evidenceID.MatchString(entry.Name()) {
			continue
		}
		var item evidenceUpload
		if readEvidenceJSON(filepath.Join(root, entry.Name(), "upload.json"), &item) == nil {
			items = append(items, item)
		}
	}
	writeJSON(w, 200, items)
}

func (s *Server) discardEvidence(w http.ResponseWriter, r *http.Request) {
	ref, ok := s.evidenceWorkspace(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")
	if !evidenceID.MatchString(id) {
		httpErr(w, 400, "invalid evidence upload")
		return
	}
	s.evidenceMu.Lock()
	defer s.evidenceMu.Unlock()
	if err := os.RemoveAll(filepath.Join(s.stagingDir(ref), id)); err != nil {
		httpErr(w, 500, "could not remove the temporary upload")
		return
	}
	writeJSON(w, 200, map[string]bool{"removed": true})
}
