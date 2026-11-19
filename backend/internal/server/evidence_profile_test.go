package server

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestStagedZipSuggestsSystemFromProjectManifest(t *testing.T) {
	dir := t.TempDir()
	files := filepath.Join(dir, "files")
	if err := os.Mkdir(files, 0o700); err != nil {
		t.Fatal(err)
	}
	archive, err := os.Create(filepath.Join(files, "source.zip"))
	if err != nil {
		t.Fatal(err)
	}
	z := zip.NewWriter(archive)
	for name, body := range map[string]string{
		"repo/StockDesk.vbproj":           "<Project><PropertyGroup><UseWindowsForms>true</UseWindowsForms></PropertyGroup></Project>",
		"repo/tests/TestWeb/Default.aspx": "<html/>",
		"repo/StockRules.vb":              "Public Class StockRules\nEnd Class",
	} {
		f, err := z.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	upload := evidenceUpload{Team: "code", Zip: true, Files: []string{"source.zip"}}
	got := profileSource(dir, upload)
	if got.System != "StockDesk" || got.NameFile != "repo/StockDesk.vbproj" || len(got.Types) != 1 || got.Types[0].Type != "desktop" {
		t.Fatalf("unexpected project suggestions: %+v", got)
	}
	upload.Team = "database"
	if got := profileSource(dir, upload); got.System != "" || len(got.Types) != 0 {
		t.Fatalf("a database upload should not classify the application: %+v", got)
	}
}
