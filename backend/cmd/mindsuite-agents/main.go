// Command mindsuite-agents runs the MindSuite code-analysis agent team.
//
//	mindsuite-agents serve [-addr 127.0.0.1:8090] [-runs ./runs] [-cors origins]
//	mindsuite-agents run -dir ./path/to/code [-model] [-git] [-out result.json] [-report report.md]
//
// Settings come from mindsuite.json (see config.go); command-line flags override them.
// The Claude API key comes from ANTHROPIC_API_KEY or from the encrypted key file named by
// apiKeyFile in mindsuite.json. It enables the model-based agents (business rules, domain,
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

  mindsuite-agents serve [-config mindsuite.json] [-addr 127.0.0.1:8090] [-runs ./runs] [-cors origins] [-allow-path]
  mindsuite-agents run -dir <folder> [-config mindsuite.json] [-model] [-git] [-max-files 12] [-out result.json] [-report report.md]

Settings: mindsuite.json (flags override it). Claude API key: ANTHROPIC_API_KEY, or the encrypted
key file named by apiKeyFile in mindsuite.json (see backend/README.md).`)
}

func serve(args []string) {
	c := setup(args)
	defAddr, defRuns, defCORS, defAllow := "127.0.0.1:8090", "runs", defaultOrigins, true
	if c.Addr != "" {
		defAddr = c.Addr
	}
	if c.Runs != nil {
		defRuns = *c.Runs
	}
	if len(c.CORS) > 0 {
		defCORS = strings.Join(c.CORS, ",")
	}
	if c.AllowPath != nil {
		defAllow = *c.AllowPath
	}
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	fs.String("config", configName, "settings file")
	addr := fs.String("addr", defAddr, "address to listen on (keep it on 127.0.0.1 unless you secure it)")
	runs := fs.String("runs", defRuns, "folder for saved run results (empty = keep in memory only)")
	cors := fs.String("cors", defCORS, "comma-separated browser origins allowed to call the API")
	allowPath := fs.Bool("allow-path", defAllow, "allow runs that read a folder on this machine (only when listening on loopback)")
	fs.Parse(args)
	key, keyFrom := apiKey(c)
	host, _, _ := net.SplitHostPort(*addr)
	loopback := host == "127.0.0.1" || host == "localhost" || host == "::1"
	if !loopback {
		log.Printf("WARNING: listening on %s. Anyone who can reach it can send code for analysis and use your API key. Put it behind authentication.", *addr)
	}
	cfg := server.Config{RunsDir: *runs, AllowedOrigins: strings.Split(*cors, ","), AllowLocalPath: *allowPath && loopback, APIKey: key, Version: version}
	srv := server.New(cfg)
	model := "off (no key: set ANTHROPIC_API_KEY or apiKeyFile in mindsuite.json to enable the Claude-powered agents)"
	if cfg.APIKey != "" {
		model = "available (" + agents.Resolve(agents.ModelStrong) + ", " + agents.Resolve(agents.ModelBalanced) + "; key from " + keyFrom + ")"
	}
	log.Printf("MindSuite agent backend %s on http://%s  · Claude: %s", version, *addr, model)
	hs := &http.Server{Addr: *addr, Handler: srv.Handler(), ReadHeaderTimeout: 20 * time.Second}
	log.Fatal(hs.ListenAndServe())
}

func runCLI(args []string) {
	c := setup(args)
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	fs.String("config", configName, "settings file")
	dir := fs.String("dir", "", "folder with the code to analyse (read-only)")
	model := fs.Bool("model", false, "allow sending selected code excerpts to Claude (needs a Claude API key)")
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
	run, err := agents.NewRun(code, agents.Options{AllowModel: *model, GitHistory: *git, MaxModelFiles: *maxFiles, Team: *team}, agents.NewLLM(func() string { k, _ := apiKey(c); return k }()), "")
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

// setup reads the config file and applies the model choices.
func setup(args []string) fileConfig {
	c := loadConfig(args)
	if c.path != "" {
		log.Printf("Settings from %s", c.path)
	}
	agents.SetModels(c.Models.Strong, c.Models.Balanced, c.Models.Fast)
	if agents.Resolve(agents.ModelStrong) == agents.Resolve(agents.ModelBalanced) {
		log.Printf("WARNING: models.strong and models.balanced are the same model, so the verifier checks the agents with the model they used.")
	}
	return c
}
