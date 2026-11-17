// Command mindsuite-agents runs the MindSuite code-analysis agent team.
//
//	mindsuite-agents serve [-addr 127.0.0.1:8090] [-runs ./runs] [-cors origins]
//	mindsuite-agents run -dir ./path/to/code [-model] [-git] [-out result.json] [-report report.md]
//
// Set ANTHROPIC_API_KEY to enable the model-based agents (business rules, domain,
// verifier second opinion, report summary). Without it, the rule-based agents still run.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/mjaffry01/mindsuite-doctrine-of-legacy-modernization/backend/internal/agents"
	"github.com/mjaffry01/mindsuite-doctrine-of-legacy-modernization/backend/internal/server"
	_ "github.com/mjaffry01/mindsuite-doctrine-of-legacy-modernization/backend/internal/teams" // registers every agent team
)

const version = "0.1.0"

const defaultOrigins = "http://localhost:*,http://127.0.0.1:*,https://mjaffry01.github.io"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "serve":
		serve(os.Args[2:])
	case "run":
		runCLI(os.Args[2:])
	case "version":
		fmt.Println(version)
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `MindSuite agent backend `+version+`

  mindsuite-agents serve [-addr 127.0.0.1:8090] [-runs ./runs] [-cors origins] [-allow-path]
  mindsuite-agents run -dir <folder> [-model] [-git] [-max-files 12] [-out result.json] [-report report.md]

Set ANTHROPIC_API_KEY to enable the Claude-powered agents.`)
}

func serve(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	addr := fs.String("addr", "127.0.0.1:8090", "address to listen on (keep it on 127.0.0.1 unless you secure it)")
	runs := fs.String("runs", "runs", "folder for saved run results (empty = keep in memory only)")
	cors := fs.String("cors", defaultOrigins, "comma-separated browser origins allowed to call the API")
	allowPath := fs.Bool("allow-path", true, "allow runs that read a folder on this machine (only when listening on loopback)")
	fs.Parse(args)
	host, _, _ := net.SplitHostPort(*addr)
	loopback := host == "127.0.0.1" || host == "localhost" || host == "::1"
	if !loopback {
		log.Printf("WARNING: listening on %s. Anyone who can reach it can send code for analysis and use your API key. Put it behind authentication.", *addr)
	}
	cfg := server.Config{RunsDir: *runs, AllowedOrigins: strings.Split(*cors, ","), AllowLocalPath: *allowPath && loopback, APIKey: os.Getenv("ANTHROPIC_API_KEY"), Version: version}
	srv := server.New(cfg)
	model := "off (set ANTHROPIC_API_KEY to enable the Claude-powered agents)"
	if cfg.APIKey != "" {
		model = "available (" + agents.ModelStrong + ", " + agents.ModelBalanced + ")"
	}
	log.Printf("MindSuite agent backend %s on http://%s  · Claude: %s", version, *addr, model)
	hs := &http.Server{Addr: *addr, Handler: srv.Handler(), ReadHeaderTimeout: 20 * time.Second}
	log.Fatal(hs.ListenAndServe())
}

func runCLI(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	dir := fs.String("dir", "", "folder with the code to analyse (read-only)")
	model := fs.Bool("model", false, "allow sending selected code excerpts to Claude (needs ANTHROPIC_API_KEY)")
	git := fs.Bool("git", true, "read Git history (read-only git log) when the folder has .git")
	maxFiles := fs.Int("max-files", 12, "files the business-rules agent reads in depth")
	out := fs.String("out", "", "write the full result as JSON to this file")
	rep := fs.String("report", "", "write the Markdown report to this file")
	team := fs.String("team", "code", `"code" (source code), "database" (schema exports and catalog results) or "infrastructure" (VM, Kubernetes, cloud and connection exports)`)
	fs.Parse(args)
	if *dir == "" {
		usage()
		os.Exit(2)
	}
	code, err := agents.LoadDir(*dir)
	if err != nil {
		log.Fatal(err)
	}
	run, err := agents.NewRun(code, agents.Options{AllowModel: *model, GitHistory: *git, MaxModelFiles: *maxFiles, Team: *team}, agents.NewLLM(os.Getenv("ANTHROPIC_API_KEY")), "")
	if err != nil {
		log.Fatal(err)
	}
	ch, stop := run.Store.Subscribe()
	done := make(chan struct{})
	go func() {
		for {
			select {
			case m := <-ch:
				if m.Type == agents.MsgLog || m.Type == agents.MsgHandoff || m.Type == agents.MsgQuestion {
					fmt.Fprintf(os.Stderr, "[%s] %s: %s\n", m.Type, m.From, m.Text)
				}
			case <-done:
				return
			}
		}
	}()
	run.Execute(context.Background())
	close(done)
	stop()
	snap := run.Snapshot()
	fmt.Fprintf(os.Stderr, "\n%d findings, %d questions, coverage %d/%d files\n", len(snap.Findings), len(snap.Questions), snap.Coverage.Analysed, snap.Coverage.Files)
	if *out != "" {
		b, _ := json.MarshalIndent(snap, "", " ")
		if err := os.WriteFile(*out, b, 0o644); err != nil {
			log.Fatal(err)
		}
	}
	if *rep != "" {
		if err := os.WriteFile(*rep, []byte(snap.Report), 0o644); err != nil {
			log.Fatal(err)
		}
	} else if *out == "" {
		fmt.Println(snap.Report)
	}
}
