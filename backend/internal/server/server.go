// Package server exposes the agent team over HTTP for the MindSuite web app.
package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mjaffry01/mindsuite-doctrine-of-legacy-modernization/backend/internal/agents"
	_ "github.com/mjaffry01/mindsuite-doctrine-of-legacy-modernization/backend/internal/teams" // registers every agent team
)

// MaxUpload is the largest request body accepted for a run.
const MaxUpload = 512 << 20

// Config of the server.
type Config struct {
	RunsDir        string   // where messages, results and reports are saved ("" = memory only)
	AllowedOrigins []string // browser origins allowed to call the API; "*" in a port means any port
	AllowLocalPath bool     // allow {"path": "..."} runs that read a folder on this machine
	APIKey         string   // Anthropic API key ("" = rule-based agents only)
	Version        string
}

// Server holds runs in memory.
type Server struct {
	cfg   Config
	llm   *agents.LLM
	mu    sync.Mutex
	dirMu sync.Mutex // serialises creating and renaming project folders
	runs  map[string]*agents.Run
}

// New creates a server.
func New(cfg Config) *Server {
	return &Server{cfg: cfg, llm: agents.NewLLM(cfg.APIKey), runs: map[string]*agents.Run{}}
}

// Handler returns the HTTP handler.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.health)
	mux.HandleFunc("GET /api/agents", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("team")
		if id == "" {
			id = "code"
		}
		t := agents.TeamByID(id)
		if t == nil {
			httpErr(w, 404, "unknown team; known: "+strings.Join(agents.TeamIDs(), ", "))
			return
		}
		writeJSON(w, 200, agents.ResolvedAgents(t.Agents()))
	})
	mux.HandleFunc("POST /api/runs", s.createRun)
	mux.HandleFunc("GET /api/runs", s.listRuns)
	mux.HandleFunc("GET /api/runs/{id}", s.getRun)
	mux.HandleFunc("GET /api/runs/{id}/events", s.events)
	mux.HandleFunc("GET /api/runs/{id}/report.md", s.report)
	return s.cors(mux)
}

func (s *Server) originAllowed(origin string) bool {
	if origin == "" {
		return true // not a browser request (curl, CLI)
	}
	for _, a := range s.cfg.AllowedOrigins {
		if a == origin {
			return true
		}
		if strings.HasSuffix(a, ":*") && strings.HasPrefix(origin, strings.TrimSuffix(a, "*")) {
			if _, err := strconv.Atoi(strings.TrimPrefix(origin, strings.TrimSuffix(a, "*"))); err == nil {
				return true
			}
		}
		if strings.HasSuffix(a, ":*") && origin == strings.TrimSuffix(a, ":*") {
			return true
		}
	}
	return false
}

// cors allows only the configured web-app origins. POST requests must carry the
// X-MindSuite header, which forces a browser preflight, so other websites cannot start runs.
func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if !s.originAllowed(origin) {
			http.Error(w, "origin not allowed: start the backend with -cors "+origin, http.StatusForbidden)
			return
		}
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-MindSuite")
			w.Header().Set("Access-Control-Allow-Private-Network", "true")
			w.Header().Set("Access-Control-Max-Age", "600")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method == http.MethodPost && r.Header.Get("X-MindSuite") == "" {
			http.Error(w, "missing X-MindSuite header", http.StatusBadRequest)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{
		"ok": true, "version": s.cfg.Version, "teams": agents.TeamIDs(), "modelAvailable": s.llm != nil, "allowLocalPath": s.cfg.AllowLocalPath,
		"projectWorkspaces": true, "projectFolders": true,
		"models": map[string]string{"strong": agents.Resolve(agents.ModelStrong), "balanced": agents.Resolve(agents.ModelBalanced), "fast": agents.Resolve(agents.ModelFast)},
	})
}

// createRun accepts one of:
//   - multipart/form-data with an "options" field (JSON) and either many "file" parts
//     (the part's filename is the file's relative path) or one "zip" part;
//   - application/json {"path": "C:/code/app", "options": {...}} when AllowLocalPath is on.
func (s *Server) createRun(w http.ResponseWriter, r *http.Request) {
	workspace, ok := workspaceFrom(w, r)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, MaxUpload)
	var code *agents.Codebase
	var opts agents.Options
	var err error
	ct := r.Header.Get("Content-Type")
	switch {
	case strings.HasPrefix(ct, "application/json"):
		var req struct {
			Path    string         `json:"path"`
			Options agents.Options `json:"options"`
		}
		if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpErr(w, 400, "bad JSON: "+err.Error())
			return
		}
		if !s.cfg.AllowLocalPath {
			httpErr(w, 403, "reading a local folder is turned off on this backend; upload the files instead")
			return
		}
		opts = req.Options
		code, err = agents.LoadDir(req.Path)
	case strings.HasPrefix(ct, "multipart/form-data"):
		code, opts, err = readMultipart(r)
	default:
		httpErr(w, 415, "send multipart/form-data (files or a zip) or application/json with a path")
		return
	}
	if err != nil {
		var mbe *http.MaxBytesError
		if errors.As(err, &mbe) {
			httpErr(w, 413, fmt.Sprintf("upload larger than %d MB", MaxUpload>>20))
			return
		}
		httpErr(w, 400, err.Error())
		return
	}
	if len(code.Files) == 0 {
		httpErr(w, 400, "no readable source files were received")
		return
	}
	opts.Workspace = workspace.id
	run, err := agents.NewRun(code, opts, s.llm, s.runBase(workspace, opts.Team))
	if err != nil {
		httpErr(w, 500, err.Error())
		return
	}
	s.mu.Lock()
	s.runs[run.ID] = run
	s.mu.Unlock()
	log.Printf("run %s: %d files from %q, model allowed=%v", run.ID, len(code.Files), code.Root, opts.AllowModel)
	go run.Execute(context.Background())
	writeJSON(w, 202, map[string]any{"id": run.ID, "files": len(code.Files), "skipped": len(code.Skipped)})
}

func readMultipart(r *http.Request) (*agents.Codebase, agents.Options, error) {
	var opts agents.Options
	mr, err := r.MultipartReader()
	if err != nil {
		return nil, opts, err
	}
	files := map[string][]byte{}
	var zipData []byte
	zipName := ""
	root := "upload"
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, opts, err
		}
		switch p.FormName() {
		case "options":
			b, _ := io.ReadAll(io.LimitReader(p, 1<<16))
			if len(bytes.TrimSpace(b)) > 0 {
				if err := json.Unmarshal(b, &opts); err != nil {
					return nil, opts, fmt.Errorf("bad options: %w", err)
				}
			}
		case "root":
			b, _ := io.ReadAll(io.LimitReader(p, 512))
			root = strings.TrimSpace(string(b))
		case "zip":
			zipName = path.Base(p.FileName())
			zipData, err = io.ReadAll(p)
			if err != nil {
				return nil, opts, err
			}
		case "file":
			// Part.FileName() drops folders; the relative path is what the agents need.
			name := p.FileName()
			if _, params, err := mime.ParseMediaType(p.Header.Get("Content-Disposition")); err == nil && params["filename"] != "" {
				name = params["filename"]
			}
			b, err := io.ReadAll(io.LimitReader(p, agents.MaxDocBytes+1)) // the loader applies the limit per file type
			if err != nil {
				return nil, opts, err
			}
			files[name] = b
		}
		p.Close()
	}
	if zipData != nil {
		c, err := agents.LoadZip(bytes.NewReader(zipData), int64(len(zipData)), zipName)
		return c, opts, err
	}
	return agents.LoadFiles(root, files), opts, nil
}

func (s *Server) liveRun(workspace, id string) *agents.Run {
	s.mu.Lock()
	defer s.mu.Unlock()
	if run := s.runs[id]; run != nil && run.Options.Workspace == workspace {
		return run
	}
	return nil
}

func (s *Server) listRuns(w http.ResponseWriter, r *http.Request) {
	workspace, ok := workspaceFrom(w, r)
	if !ok {
		return
	}
	out := s.resultsFor(workspace)
	sort.Slice(out, func(i, j int) bool { return out[i].ID > out[j].ID })
	writeJSON(w, 200, out)
}

func (s *Server) result(w http.ResponseWriter, r *http.Request) (agents.Result, bool) {
	workspace, ok := workspaceFrom(w, r)
	if !ok {
		return agents.Result{}, false
	}
	id := r.PathValue("id")
	if run := s.liveRun(workspace.id, id); run != nil {
		return run.Snapshot(), true
	}
	if result, ok := s.savedResult(workspace, id); ok {
		return result, true
	}
	httpErr(w, 404, "run not found in this workspace")
	return agents.Result{}, false
}

func (s *Server) getRun(w http.ResponseWriter, r *http.Request) {
	if result, ok := s.result(w, r); ok {
		writeJSON(w, 200, result)
	}
}

func (s *Server) report(w http.ResponseWriter, r *http.Request) {
	result, ok := s.result(w, r)
	if !ok {
		return
	}
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+result.Run.Options.Team+"-analysis-"+result.Run.ID+`.md"`)
	io.WriteString(w, result.Report)
}

// events streams the evidence store as Server-Sent Events; ?after=N resumes after message N.
func (s *Server) events(w http.ResponseWriter, r *http.Request) {
	workspace, ok := workspaceFrom(w, r)
	if !ok {
		return
	}
	run := s.liveRun(workspace.id, r.PathValue("id"))
	if run == nil {
		if _, ok := s.savedResult(workspace, r.PathValue("id")); ok {
			w.WriteHeader(http.StatusNoContent)
		} else {
			httpErr(w, 404, "live run not found in this workspace")
		}
		return
	}
	fl, ok := w.(http.Flusher)
	if !ok {
		httpErr(w, 500, "streaming not supported")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	after, _ := strconv.Atoi(r.URL.Query().Get("after"))
	ch, stop := run.Store.Subscribe()
	defer stop()
	send := func(m agents.Message) {
		b, _ := json.Marshal(m)
		fmt.Fprintf(w, "id: %d\nevent: message\ndata: %s\n\n", m.Seq, b)
		after = m.Seq
	}
	for _, m := range run.Store.Since(after) {
		send(m)
	}
	fl.Flush()
	tick := time.NewTicker(2 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case m := <-ch:
			if m.Seq > after {
				for _, x := range run.Store.Since(after) {
					send(x)
				}
				fl.Flush()
			}
		case <-tick.C:
			for _, x := range run.Store.Since(after) {
				send(x)
			}
			if run.Snapshot().Run.Status != "running" {
				fmt.Fprintf(w, "event: done\ndata: {}\n\n")
				fl.Flush()
				return
			}
			fmt.Fprintf(w, ": keep-alive\n\n")
			fl.Flush()
		}
	}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.Encode(v)
}

func httpErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}
