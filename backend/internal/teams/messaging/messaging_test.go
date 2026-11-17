package messaging

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mjaffry01/mindsuite-doctrine-of-legacy-modernization/backend/internal/agents"
)

// TestMessagingRun runs the team (without a model) on testdata/sample: runmqsc output of an IBM MQ
// queue manager, Redis INFO, Kafka topics and consumer groups, RabbitMQ definitions and a little
// application code that sends to and reads from the queues.
func TestMessagingRun(t *testing.T) {
	code, err := agents.LoadDir("testdata/sample")
	if err != nil {
		t.Fatal(err)
	}
	r, err := agents.NewRun(code, agents.Options{Team: "messaging"}, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	r.Execute(context.Background())
	snap := r.Snapshot()
	if snap.Run.Status != "done" {
		t.Fatalf("status %s: %s", snap.Run.Status, snap.Run.Error)
	}
	if len(snap.Agents) != 12 || snap.Agents[1].ID != "brk" {
		t.Errorf("messaging team expected, got %d agents", len(snap.Agents))
	}
	byKind := map[string][]agents.Finding{}
	for _, f := range snap.Findings {
		byKind[f.Kind] = append(byKind[f.Kind], f)
		if f.Status == StatusProposed {
			t.Errorf("not verified: %s %s", f.ID, f.Title)
		}
		if f.Status == StatusRejected {
			t.Errorf("rejected: %s %s — %s", f.ID, f.Title, f.Verdict)
		}
	}
	has := func(kind, sub string) {
		t.Helper()
		for _, f := range byKind[kind] {
			if strings.Contains(f.Title+" "+f.Detail, sub) {
				return
			}
		}
		titles := []string{}
		for _, f := range byKind[kind] {
			titles = append(titles, f.Title)
		}
		t.Errorf("missing %s containing %q; have %v", kind, sub, titles)
	}
	has("broker", "IBM MQ QM_HL 9.1.0.0")
	has("broker", "Kafka")
	has("broker", "RabbitMQ")
	has("destination", "queue HL.LOT.GRADED on QM_HL")
	has("destination", "topic grading.results")
	has("channel", "HL.TO.BANK (sender) → mq.partnerbank.example(1414)")
	has("cache", "Redis")
	has("broker-eol", "IBM MQ 9.2 and earlier")
	has("broker-eol", "RabbitMQ before 3.13")
	has("cache-eol", "Redis 6.x and earlier")
	has("flow", "HL.LOT.GRADED: sent by")
	has("flow", "read by")
	has("flow", "grading.results: sent by")
	has("partner-channel", "mq.partnerbank.example(1414)")
	has("remote-queue", "BANK.PAYMENTS forwards to PAY.IN @ QM_BANK")
	has("consumer-group", "settlement-svc reads grading.results")
	has("no-reader", "HL.PAYMENT.OUT holds 4600")
	has("queue-near-full", "HL.PAYMENT.OUT is 92% full")
	has("dead-letters", "37 message(s) waiting on HL.DLQ")
	has("consumer-lag", "reporting is 25000 messages behind")
	has("no-backout-queue", "HL.LOT.GRADED")
	has("non-persistent", "HL.PAYMENT.OUT")
	has("non-durable", "portal.notifications")
	has("no-dead-letter", "portal.notifications")
	has("single-replica", "grading.results")
	has("low-hit-ratio", "hit ratio 72%")
	has("cache-noeviction", "noeviction")
	has("cache-near-full", "3.7 GB of 4.0 GB")
	has("cache-no-ttl", "182800 of 184000 keys never expire")
	has("cache-as-store", "append-only file")
	has("channel-no-tls", "HL.TO.BANK")
	has("admin-channel", "HL.APP.SVRCONN")
	has("default-user", "guest")
	has("secret", "application.properties line 2")
	has("broker-assessment", "Keep IBM MQ first")
	has("cache-assessment", "Migrate to a managed Redis service with its data")
	has("migration-order", "Last: IBM MQ QM_HL")
	has("migration-order", "starting with: With the application waves: Kafka")
	has("migration-approach", "partners depend on it")
	if len(byKind["channel"]) != 2 {
		t.Errorf("want the 2 non-system channels, got %d", len(byKind["channel"]))
	}
	for _, f := range byKind["destination"] {
		if strings.Contains(f.Title, "SYSTEM.") {
			t.Errorf("system queues must not be listed: %s", f.Title)
		}
	}
	for _, f := range byKind["channel-no-tls"] {
		if strings.Contains(f.Detail, "SYSTEM.DEF") {
			t.Errorf("system channels must not be listed: %s", f.Detail)
		}
	}
	b, _ := json.Marshal(snap)
	if strings.Contains(string(b), "Redis5ecret99") || strings.Contains(r.Report, "Redis5ecret99") {
		t.Error("the Redis password leaked into the result")
	}
	if snap.Run.Steps[4].Status != "skipped" {
		t.Errorf("step 5 should be skipped without a model, got %s", snap.Run.Steps[4].Status)
	}
	if !strings.Contains(r.Report, "# Messaging & caching analysis report") || !strings.Contains(r.Report, "Messages at risk") {
		t.Error("report incomplete")
	}
}
