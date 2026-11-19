package server

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mjaffry01/mindsuite-doctrine-of-legacy-modernization/backend/internal/agents"
)

func TestStagedEvidenceMovesIntoTeamFolders(t *testing.T) {
	const ws = "8aae19c0-6c8c-404d-ba55-4c860ce09291"
	const elsewhere = "6ac069f6-aea9-4925-871d-dac0be8af004"
	dir := t.TempDir()
	api := New(Config{RunsDir: dir})
	handler := api.Handler()
	send := func(method, target, contentType string, body []byte, want int) []byte {
		t.Helper()
		req := httptest.NewRequest(method, target, bytes.NewReader(body))
		req.Header.Set("Content-Type", contentType)
		req.Header.Set("X-MindSuite", "1")
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		if res.Code != want {
			t.Fatalf("%s %s: %d %s", method, target, res.Code, res.Body)
		}
		return res.Body.Bytes()
	}
	upload := func(team, name, content string, want int) evidenceUpload {
		t.Helper()
		var buf bytes.Buffer
		form := multipart.NewWriter(&buf)
		part, err := form.CreateFormFile("file", name)
		if err != nil {
			t.Fatal(err)
		}
		part.Write([]byte(content))
		form.Close()
		result := send("POST", "/api/evidence/stage?workspace="+ws+"&team="+team, form.FormDataContentType(), buf.Bytes(), want)
		var record evidenceUpload
		if want == 201 && json.Unmarshal(result, &record) != nil {
			t.Fatalf("invalid upload response: %s", result)
		}
		return record
	}
	upload("code", "../Outside.java", "class Outside {}", 400)
	code := upload("code", "Demo.java", "class Demo { int count() { return 1; } }", 201)
	db := upload("database", "schema.sql", "CREATE TABLE CATALOG (ID INT PRIMARY KEY);", 201)
	if code.Name != "Demo" {
		t.Fatalf("expected a suggested name derived from the upload: %+v", code)
	}
	send("GET", "/api/runs?workspace="+ws, "", nil, 200)
	send("GET", "/api/evidence/staged?workspace="+ws, "", nil, 200)
	if _, err := os.Stat(filepath.Join(dir, "workspaces")); !os.IsNotExist(err) {
		t.Fatalf("upload or read created a project before Start: %v", err)
	}
	start := func(workspace string) []evidenceResult {
		t.Helper()
		payload, _ := json.Marshal(map[string]any{"name": "Catalog pilot", "uploads": []string{code.ID, db.ID}})
		result := send("POST", "/api/evidence/start?workspace="+workspace, "application/json", payload, 200)
		var outcome struct{ Uploads []evidenceResult }
		if err := json.Unmarshal(result, &outcome); err != nil {
			t.Fatal(err)
		}
		return outcome.Uploads
	}
	for _, item := range start(elsewhere) {
		if item.RunID != "" || item.Placed || item.Error == "" {
			t.Fatalf("another workspace could use the evidence: %+v", item)
		}
	}
	launched := start(ws)
	repeated := start(ws)
	project := filepath.Join(dir, "workspaces", "Catalog pilot ("+ws[:8]+")")
	for _, team := range teamFolders {
		if info, err := os.Stat(filepath.Join(project, team.folder)); err != nil || !info.IsDir() {
			t.Fatalf("missing team folder: %s (%v)", team.folder, err)
		}
	}
	for i, item := range launched {
		if !item.Placed || item.RunID == "" || item.RunID != repeated[i].RunID {
			t.Fatalf("expected exactly one run per promoted upload: %+v, %+v", item, repeated[i])
		}
		folder := filepath.Join(project, teamFolder(item.Team))
		if _, err := os.Stat(filepath.Join(folder, "evidence", item.ID, "files", item.Files[0])); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(filepath.Join(dir, "staging", ws, item.ID)); !os.IsNotExist(err) {
			t.Fatalf("evidence is still staged: %v", err)
		}
		report := filepath.Join(folder, item.RunID, "report.md")
		deadline := time.Now().Add(5 * time.Second)
		for !fileExists(report) {
			if time.Now().After(deadline) {
				t.Fatalf("the agent did not save a report: %s", report)
			}
			time.Sleep(20 * time.Millisecond)
		}
	}
	payload, _ := json.Marshal(map[string]any{
		"name": "Catalog pilot", "uploads": []string{code.ID}, "reanalyze": true,
		"interview": map[string]any{"problem": "Focus on inventory", "deadline": "2027-02-01"},
	})
	body := send("POST", "/api/evidence/start?workspace="+ws, "application/json", payload, 200)
	var again struct{ Uploads []evidenceResult }
	if err := json.Unmarshal(body, &again); err != nil || len(again.Uploads) != 1 || again.Uploads[0].RunID == "" || again.Uploads[0].RunID == launched[0].RunID {
		t.Fatalf("reanalyze must create another run: %s (%v)", body, err)
	}
	report := filepath.Join(project, teamFolder("code"), again.Uploads[0].RunID, "report.md")
	for deadline := time.Now().Add(5 * time.Second); ; {
		content, err := os.ReadFile(report)
		if err == nil && bytes.Contains(content, []byte("Interview deadline: 2027-02-01")) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the second run did not report the interview deadline")
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !fileExists(filepath.Join(project, teamFolder("code"), launched[0].RunID, "report.md")) {
		t.Fatal("reanalyze removed the prior report")
	}

	// Simulate an interruption after the batches reached disk but before the final report was saved.
	large := upload("code", "Large.vb", strings.Repeat("' sample\n", 4000)+"End Sub\n"+strings.Repeat("' next\n", 1800), 201)
	launchLarge := func(resume bool) evidenceResult {
		t.Helper()
		payload, _ := json.Marshal(map[string]any{"name": "Catalog pilot", "uploads": []string{large.ID}, "resume": resume})
		data := send("POST", "/api/evidence/start?workspace="+ws, "application/json", payload, 200)
		var result struct{ Uploads []evidenceResult }
		if err := json.Unmarshal(data, &result); err != nil || len(result.Uploads) != 1 || result.Uploads[0].RunID == "" {
			t.Fatalf("run query failed: %s (%v)", data, err)
		}
		return result.Uploads[0]
	}
	waitForBatches := func(id string) agents.Result {
		t.Helper()
		folder := filepath.Join(project, teamFolder("code"), id)
		for deadline := time.Now().Add(8 * time.Second); ; {
			var result agents.Result
			if fileExists(filepath.Join(folder, "report.md")) && readEvidenceJSON(filepath.Join(folder, "result.json"), &result) == nil {
				if result.Run.Batches == nil || result.Run.Batches.Completed != 2 || result.Run.Status != "done" {
					t.Fatalf("unexpected completion: %+v", result.Run)
				}
				return result
			}
			if time.Now().After(deadline) {
				t.Fatal("batch run did not finish")
			}
			time.Sleep(20 * time.Millisecond)
		}
	}
	firstLarge := launchLarge(false)
	waitForBatches(firstLarge.RunID)
	resultFile := filepath.Join(project, teamFolder("code"), firstLarge.RunID, "result.json")
	if err := os.Rename(resultFile, resultFile+".test-saved"); err != nil {
		t.Fatal(err)
	}
	handler = New(Config{RunsDir: dir}).Handler()

	telemetry := func(id, workspace string, want int) []byte {
		t.Helper()
		return send("GET", "/api/runs/"+id+"/telemetry?workspace="+workspace, "", nil, want)
	}
	old := telemetry(firstLarge.RunID, ws, 200)
	if !bytes.Contains(old, []byte("interrupted")) || !bytes.Contains(old, []byte("Source business")) && !bytes.Contains(old, []byte("Inventory")) {
		t.Fatalf("stored telemetry lost across restart: %s", old)
	}
	telemetry(firstLarge.RunID, elsewhere, 404)
	resumed := launchLarge(true)
	recovered := waitForBatches(resumed.RunID)
	reusedAttempts := 0
	for _, attempt := range recovered.Telemetry {
		if attempt.Reused && attempt.Batch > 0 {
			reusedAttempts++
		}
	}
	if reusedAttempts == 0 {
		t.Fatal("resumed run did not preserve reused attempts")
	}

	if resumed.RunID == firstLarge.RunID || recovered.Run.Batches.Reused != 2 {
		t.Fatalf("resume did not reuse both persisted batches: %+v", recovered.Run.Batches)
	}

}
