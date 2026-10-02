package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/JMThomas00/Concord/sdk/client/clienttest"
	"github.com/charmbracelet/x/ansi"
)

// A round through Concord's client-code interface: the time is reported to
// the server half and remembered as the best; the leaderboard is drawn.
func TestARoundAsClientCode(t *testing.T) {
	h := clienttest.New(t, handler(), "pane", "storage", "server")
	h.Start(50, 20)
	if h.ForwardingKeys() || !strings.Contains(ansi.Strip(h.LastFrame()), "Press Space to start") {
		t.Fatalf("start: forwarding=%v\n%s", h.ForwardingKeys(), h.LastFrame())
	}
	h.Key(" ")
	if _, ok := h.Timers()["go"]; !ok {
		t.Fatalf("no NOW timer: %v", h.Timers())
	}
	h.Timer("go")
	if !strings.Contains(ansi.Strip(h.LastFrame()), "NOW!") {
		t.Fatalf("now:\n%s", h.LastFrame())
	}
	h.Key(" ")
	sent := h.Sent()
	var report struct{ MS int }
	if len(sent) != 1 || json.Unmarshal(sent[0], &report) != nil || h.Storage()["best"] == "" {
		t.Fatalf("sent %s, storage %v", sent, h.Storage())
	}
	h.Server(map[string]any{"top": []map[string]any{{"name": "alice", "ms": 199}}})
	if !strings.Contains(ansi.Strip(h.LastFrame()), "1. alice") {
		t.Fatalf("leaderboard:\n%s", h.LastFrame())
	}
}
