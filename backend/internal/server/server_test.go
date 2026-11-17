package server

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"
	"time"
)

func upload(t *testing.T, h http.Handler, origin string, withHeader bool) *httptest.ResponseRecorder {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	mw.WriteField("options", `{"allowModel":false}`)
	for p, c := range map[string]string{
		"shop/src/main/java/com/acme/order/OrderService.java": "package com.acme.order;\nimport com.acme.billing.Invoice;\nclass OrderService { void a(){ jdbc.update(\"UPDATE ORDERS SET X=1\"); } }\n",
		"shop/src/main/java/com/acme/billing/Invoice.java":    "package com.acme.billing;\nimport com.acme.order.OrderService;\nclass Invoice { void b(){ jdbc.update(\"UPDATE ORDERS SET Y=1\"); } }\n",
	} {
		hd := textproto.MIMEHeader{}
		hd.Set("Content-Disposition", `form-data; name="file"; filename="`+p+`"`)
		w, _ := mw.CreatePart(hd)
		w.Write([]byte(c))
	}
	mw.Close()
	req := httptest.NewRequest("POST", "/api/runs", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	if withHeader {
		req.Header.Set("X-MindSuite", "1")
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestUploadRun(t *testing.T) {
	s := New(Config{AllowedOrigins: []string{"http://localhost:*", "https://mjaffry01.github.io"}})
	h := s.Handler()
	if rec := upload(t, h, "https://evil.example", true); rec.Code != 403 {
		t.Fatalf("other origins must be refused, got %d", rec.Code)
	}
	if rec := upload(t, h, "http://localhost:8768", false); rec.Code != 400 {
		t.Fatalf("POST without X-MindSuite must be refused, got %d", rec.Code)
	}
	rec := upload(t, h, "http://localhost:8768", true)
	if rec.Code != 202 {
		t.Fatalf("upload: %d %s", rec.Code, rec.Body)
	}
	var out struct{ ID string }
	json.Unmarshal(rec.Body.Bytes(), &out)
	var snap struct {
		Run      struct{ Status string }
		Findings []struct{ Kind, Title string }
	}
	for i := 0; i < 50; i++ {
		r := httptest.NewRecorder()
		h.ServeHTTP(r, httptest.NewRequest("GET", "/api/runs/"+out.ID, nil))
		json.Unmarshal(r.Body.Bytes(), &snap)
		if snap.Run.Status != "running" {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if snap.Run.Status != "done" {
		t.Fatalf("run status %q", snap.Run.Status)
	}
	found := map[string]bool{}
	for _, f := range snap.Findings {
		found[f.Kind+"|"+f.Title] = true
	}
	for _, k := range []string{"cycle|Dependency cycle: billing ↔ order", "shared-table|Shared table ORDERS: used by billing, order"} {
		if !found[k] {
			t.Errorf("missing %s (folder paths must survive the upload)", k)
		}
	}
}

func TestDatabaseUpload(t *testing.T) {
	s := New(Config{AllowedOrigins: []string{"http://localhost:*"}})
	h := s.Handler()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	mw.WriteField("options", `{"team":"database"}`)
	hd := textproto.MIMEHeader{}
	hd.Set("Content-Disposition", `form-data; name="file"; filename="export/schema.sql"`)
	w, _ := mw.CreatePart(hd)
	w.Write([]byte("CREATE TABLE ORDERS (ID NUMBER PRIMARY KEY, CUSTOMER_ID NUMBER REFERENCES CUSTOMERS(ID), STATUS VARCHAR2(10) CHECK (STATUS IN ('NEW','SHIPPED')));\nCREATE TABLE CUSTOMERS (ID NUMBER PRIMARY KEY, EMAIL VARCHAR2(100));\n"))
	mw.Close()
	req := httptest.NewRequest("POST", "/api/runs", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("X-MindSuite", "1")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 202 {
		t.Fatalf("upload: %d %s", rec.Code, rec.Body)
	}
	var out struct{ ID string }
	json.Unmarshal(rec.Body.Bytes(), &out)
	var snap struct {
		Run      struct{ Status string }
		Agents   []struct{ ID string }
		Findings []struct{ Kind, Title string }
	}
	for i := 0; i < 50; i++ {
		r := httptest.NewRecorder()
		h.ServeHTTP(r, httptest.NewRequest("GET", "/api/runs/"+out.ID, nil))
		json.Unmarshal(r.Body.Bytes(), &snap)
		if snap.Run.Status != "running" {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	kinds := map[string]bool{}
	for _, f := range snap.Findings {
		kinds[f.Kind] = true
	}
	for _, k := range []string{"db-engine", "table", "constraint-rule", "data-cluster", "sensitive-column", "migration-approach"} {
		if !kinds[k] {
			t.Errorf("missing %s finding", k)
		}
	}
	if len(snap.Agents) != 14 {
		t.Errorf("want the 14 database agents, got %d", len(snap.Agents))
	}
}
