package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/JMThomas00/Concord/sdk/client/clienttest"
	"github.com/JMThomas00/Concord/sdk/wire"
	"github.com/charmbracelet/x/ansi"

	"github.com/JMThomas00/concord-reflex/game"
)

// The arcade through Concord's client-code interface: it says hello, draws
// the title, claims Esc past the title, and a quick draw reaches the server
// half, with its sounds.
func TestTheArcadeAsClientCode(t *testing.T) {
	h := clienttest.New(t, handler(), "pane", "server", "sound")
	h.Start(80, 24)
	if h.ForwardingKeys() || !strings.Contains(ansi.Strip(h.LastFrame()), "PRESS ENTER") {
		t.Fatalf("start: forwarding=%v\n%s", h.ForwardingKeys(), h.LastFrame())
	}
	if len(h.Claimed()) != 0 {
		t.Fatalf("the title claims %v: Esc must leave the pane there", h.Claimed())
	}
	var hello game.Up
	if sent := h.Sent(); len(sent) != 1 || json.Unmarshal(sent[0], &hello) != nil || hello.T != "hello" {
		t.Fatalf("sent %s", h.Sent())
	}
	h.Server(game.Down{T: "state", Me: &game.Profile{Best: 240}, Top: []game.Entry{{Name: "merlot", MS: 163}}})
	if !strings.Contains(ansi.Strip(h.LastFrame()), "FASTEST HERE: MERLOT 163 MS") {
		t.Fatalf("title with the Hall of Fame:\n%s", h.LastFrame())
	}
	h.Key("enter")
	if c := h.Claimed(); len(c) != 1 || c[0] != wire.PaneKeyEsc {
		t.Fatalf("the menu claims %v", c)
	}
	h.Key("enter") // QUICK DRAW
	h.Key(" ")     // ready
	var draw string
	for id := range h.Timers() {
		if strings.HasPrefix(id, "draw:") {
			draw = id
		}
	}
	if !h.Timer(draw) || !strings.Contains(ansi.Strip(h.LastFrame()), "draw!") {
		t.Fatalf("no word: %v\n%s", h.Timers(), h.LastFrame())
	}
	h.Key(" ") // at once: faster than anyone reacts, so a guess, a foul
	sent := h.Sent()
	var result game.Up
	if json.Unmarshal(sent[len(sent)-1], &result) != nil || result.T != "result" || result.Mode != game.ModeDuel || !result.Foul {
		t.Fatalf("sent %s", sent)
	}
	played := map[string]bool{}
	for _, s := range h.Sounds() {
		played[s.Asset] = true
	}
	for _, want := range []string{"sounds/coin.wav", "sounds/tick.wav", "sounds/draw.wav", "sounds/red.wav"} {
		if !played[want] {
			t.Fatalf("no %s among %v", want, h.Sounds())
		}
	}
}
