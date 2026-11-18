package server

import (
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestProjectWorkspaceIsolation(t *testing.T) {
	const alpha = "8aae19c0-6c8c-404d-ba55-4c860ce09291"
	const beta = "6ac069f6-aea9-4925-871d-dac0be8af004"
	dir := t.TempDir()
	server := New(Config{RunsDir: dir})
	api := server.Handler()

	get := func(handler http.Handler, path string, want int) *httptest.ResponseRecorder {
		t.Helper()
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
		if rec.Code != want {
			t.Fatalf("GET %s returned %d, want %d", path, rec.Code, want)
		}
		return rec
	}

	post := func(workspace string, want int) string {
		t.Helper()
		var body strings.Builder
		writer := multipart.NewWriter(&body)
		writer.WriteField("options", "{\"team\":\"code\",\"workspace\":\""+beta+"\"}")
		part, err := writer.CreateFormFile("file", "Demo.java")
		if err != nil {
			t.Fatal(err)
		}
		part.Write([]byte("class Demo { int value() { return 1; } }"))
		writer.Close()
		req := httptest.NewRequest("POST", "/api/runs?workspace="+workspace, strings.NewReader(body.String()))
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("X-MindSuite", "1")
		rec := httptest.NewRecorder()
		api.ServeHTTP(rec, req)
		if rec.Code != want {
			t.Fatalf("create returned %d, want %d: %s", rec.Code, want, rec.Body)
		}
		var response struct{ ID string }
		json.Unmarshal(rec.Body.Bytes(), &response)
		return response.ID
	}

	post("..%2Fescape", 400)
	id := post(alpha, 202)
	runPath := "/api/runs/" + id
	for _, suffix := range []string{"", "/report.md", "/events"} {
		get(api, runPath+suffix+"?workspace="+beta, 404)
		get(api, runPath+suffix, 404)
	}
	if strings.Contains(get(api, "/api/runs?workspace="+beta, 200).Body.String(), id) {
		t.Fatal("project B received a run from project A")
	}
	if !strings.Contains(get(api, "/api/runs?workspace="+alpha, 200).Body.String(), id) {
		t.Fatal("project A cannot find its run")
	}
	live := server.liveRun(alpha, id)
	saved := filepath.Join(dir, "workspaces", "Project ("+alpha[:8]+")", "01 Source code", id, "result.json")
	deadline := time.Now().Add(5 * time.Second)
	for !func() bool {
		_, err := os.Stat(saved)
		return err == nil && !live.Snapshot().Run.Finished.IsZero()
	}() {
		if time.Now().After(deadline) {
			t.Fatal("result was not saved in the project directory")
		}
		time.Sleep(20 * time.Millisecond)
	}
	restarted := New(Config{RunsDir: dir}).Handler()
	get(restarted, runPath+"?workspace="+alpha, 200)
	get(restarted, runPath+"/report.md?workspace="+alpha, 200)
	get(restarted, runPath+"/events?workspace="+alpha, 204)
	get(restarted, runPath+"?workspace="+beta, 404)
	if !strings.Contains(get(restarted, "/api/runs?workspace="+alpha, 200).Body.String(), id) {
		t.Fatal("project A cannot find its run after restart")
	}
}

func TestProjectFoldersByNameAndTeam(t *testing.T) {
	const id = "8aae19c0-6c8c-404d-ba55-4c860ce09291"
	dir := t.TempDir()
	s := New(Config{RunsDir: dir})

	// A result saved in the earlier layout (<runs>/workspaces/<id>/<run>/) moves into the team folder.
	old := filepath.Join(dir, "workspaces", id, "20261118-142836-25e85c518dc9")
	os.MkdirAll(old, 0o755)
	os.WriteFile(filepath.Join(old, "result.json"), []byte(`{"run":{"id":"20261118-142836-25e85c518dc9","options":{"team":"batch","workspace":"`+id+`"}}}`), 0o644)

	got := s.workspaceDir(workspaceRef{id: id, name: `CardDemo: test/1?`})
	want := filepath.Join(dir, "workspaces", "CardDemo test 1 ("+id[:8]+")")
	if got != want {
		t.Fatalf("project folder %q, want %q", got, want)
	}
	for _, f := range teamFolders {
		if !fileExists(filepath.Join(got, f.folder)) {
			t.Fatalf("missing team folder %s", f.folder)
		}
	}
	if !fileExists(filepath.Join(got, "07 Batch and scheduling", "20261118-142836-25e85c518dc9", "result.json")) {
		t.Fatal("earlier run was not moved into its team folder")
	}
	if _, ok := s.savedResult(workspaceRef{id: id}, "20261118-142836-25e85c518dc9"); !ok {
		t.Fatal("moved run cannot be found")
	}

	// Renaming the project renames the folder; a request without a name keeps it.
	renamed := s.workspaceDir(workspaceRef{id: id, name: "CardDemo mainframe"})
	if filepath.Base(renamed) != "CardDemo mainframe ("+id[:8]+")" || fileExists(got) {
		t.Fatalf("rename gave %q", renamed)
	}
	if s.workspaceDir(workspaceRef{id: id}) != renamed {
		t.Fatal("a request without a project name moved the folder")
	}
	if s.runBase(workspaceRef{id: id}, "messaging") != filepath.Join(renamed, "04 Messaging and caching") {
		t.Fatal("messaging runs are not saved in the messaging folder")
	}
}
