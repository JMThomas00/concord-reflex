package main

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/JMThomas00/Concord/sdk/plugin"
	"github.com/JMThomas00/Concord/sdk/plugintest"
	"github.com/JMThomas00/Concord/sdk/wire"
	"github.com/JMThomas00/concord-reflex/game"
)

func startServer(t *testing.T) (*plugintest.Server, uuid.UUID) {
	t.Helper()
	srv := plugintest.NewServer(t)
	cfg := srv.Config() // before the cleanup below, so the plugin stops before its data folder goes
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { plugin.Run(ctx, cfg, newServer().handler()); close(done) }()
	t.Cleanup(func() { cancel(); <-done })
	srv.WaitReady()
	ch := uuid.New()
	srv.Channel(wire.Channel{ID: ch, Name: "reflex"})
	return srv, ch
}

// records are the record events seen while waiting for messages.
var records []wire.PluginRecord

// next is the next message the server half sends to `to` with type kind.
// Every event is read in order (client messages and records share one
// queue), and records are kept.
func next(t *testing.T, srv *plugintest.Server, to uuid.UUID, kind string) game.Down {
	t.Helper()
	deadline := time.Now().Add(plugintest.Timeout)
	for time.Now().Before(deadline) {
		e := srv.NextEvent()
		switch e.Kind {
		case wire.PluginEventRecord:
			var r wire.PluginRecord
			_ = json.Unmarshal(e.Payload, &r)
			records = append(records, r)
		case wire.PluginEventClientMessage:
			var m wire.PluginClientMessagePayload
			var d game.Down
			_ = json.Unmarshal(e.Payload, &m)
			if json.Unmarshal(m.Data, &d) == nil && e.ViewerID == to && d.T == kind {
				return d
			}
		}
	}
	t.Fatalf("no %q message for %s", kind, to)
	return game.Down{}
}

// Results become a record, the Hall of Fame and achievements; impossible
// times are ignored; Bounties buy gear.
func TestRecords(t *testing.T) {
	srv, ch := startServer(t)
	alice := srv.Enter(ch, "alice", 80, 24)
	srv.FrameContaining(alice, "runs on your own computer")
	srv.ClientMessage(alice, game.Up{T: "hello"})
	if d := next(t, srv, alice.ID, "state"); d.Me == nil || d.Me.Best != 0 || len(d.Me.Owned) != 4 {
		t.Fatalf("first state %+v", d.Me) // two starter hats, two starter backdrops
	}

	srv.ClientMessage(alice, game.Up{T: "result", Mode: game.ModeDuel, MS: 40, Won: true, Level: 1}) // a guess
	if d := next(t, srv, alice.ID, "state"); d.Me.Best != 0 || d.Me.Wins != 0 || d.Me.Duels != 1 {
		t.Fatalf("a 40 ms win counted: %+v", d.Me)
	}
	srv.ClientMessage(alice, game.Up{T: "result", Mode: game.ModeDuel, MS: 205, Won: true, Level: 2})
	d := next(t, srv, alice.ID, "state")
	if d.Me.Best != 205 || d.Me.Wins != 1 || !d.Me.NewRecord || len(d.Top) != 1 || d.Top[0].MS != 205 || !d.Top[0].Me {
		t.Fatalf("after a 205 ms win: %+v top %+v", d.Me, d.Top)
	}
	// First draw, first win, deputy, sheriff and the hard outlaw: five
	// Bounties.
	if d.Me.Bounties != 5 || d.Me.Earned != 5 {
		t.Fatalf("bounties %d (earned %d)", d.Me.Bounties, d.Me.Earned)
	}
	if len(records) == 0 {
		t.Fatal("no record sent to Concord")
	}
	rec := records[len(records)-1]
	if rec.Stats[0].Key != "best" || rec.Stats[0].Num != -205 || rec.Stats[0].Value != "205 ms" || len(rec.Unlocked) == 0 {
		t.Fatalf("record %+v", rec)
	}

	srv.ClientMessage(alice, game.Up{T: "deal"})
	d = next(t, srv, alice.ID, "state")
	if len(d.Me.Offer) != 3 {
		t.Fatalf("offer %v", d.Me.Offer)
	}
	pick := d.Me.Offer[0]
	srv.ClientMessage(alice, game.Up{T: "pick", ID: pick})
	d = next(t, srv, alice.ID, "state")
	if d.Me.Bounties != 4 || len(d.Me.Owned) != 5 || (d.Me.Equipped[game.KindHat] != pick && d.Me.Equipped[game.KindScene] != pick) {
		t.Fatalf("after picking %s: %+v", pick, d.Me)
	}
}

// A showdown: a challenge, an acceptance, both ready, one word for both,
// and the faster draw wins.
func TestShowdown(t *testing.T) {
	srv, ch := startServer(t)
	aliceID, bobID := srv.UserID("alice"), srv.UserID("bob")
	alice := srv.EnterAs(ch, aliceID, "alice", 80, 24)
	bob := srv.EnterAs(ch, bobID, "bob", 80, 24)
	srv.FrameContaining(alice, "runs on your own computer")
	srv.FrameContaining(bob, "runs on your own computer")

	srv.ClientMessage(alice, game.Up{T: "challenge", To: bobID.String()})
	d := next(t, srv, bobID, "state")
	if len(d.Invites) != 1 || d.Invites[0].Name != "alice" || d.Invites[0].Mine {
		t.Fatalf("bob's invites %+v", d.Invites)
	}
	srv.ClientMessage(bob, game.Up{T: "answer", ID: d.Invites[0].ID, Accept: true})
	duel := next(t, srv, aliceID, "duel").Duel
	if duel == nil || duel.Opponent != "bob" {
		t.Fatalf("alice's duel %+v", duel)
	}
	srv.ClientMessage(alice, game.Up{T: "ready", ID: duel.ID})
	srv.ClientMessage(bob, game.Up{T: "ready", ID: duel.ID})
	goA, goB := next(t, srv, aliceID, "go"), next(t, srv, bobID, "go")
	if goA.Delay != goB.Delay || goA.Delay < 1500 {
		t.Fatalf("the word comes at %d and %d", goA.Delay, goB.Delay)
	}
	srv.ClientMessage(alice, game.Up{T: "shot", ID: duel.ID, MS: 231})
	srv.ClientMessage(bob, game.Up{T: "shot", ID: duel.ID, MS: 198})
	a, b := next(t, srv, aliceID, "outcome"), next(t, srv, bobID, "outcome")
	if a.Won || !b.Won || a.Mine != 231 || a.Theirs != 198 || b.Mine != 198 {
		t.Fatalf("outcomes %+v / %+v", a, b)
	}
	if d := next(t, srv, bobID, "duel"); d.Duel == nil || d.Duel.ScoreUs != 1 || d.Duel.ScoreThem != 0 {
		t.Fatalf("bob's score %+v", d.Duel)
	}

	// A foul loses, however fast the other was.
	srv.ClientMessage(alice, game.Up{T: "ready", ID: duel.ID})
	srv.ClientMessage(bob, game.Up{T: "ready", ID: duel.ID})
	next(t, srv, aliceID, "go")
	srv.ClientMessage(bob, game.Up{T: "shot", ID: duel.ID, Foul: true})
	srv.ClientMessage(alice, game.Up{T: "shot", ID: duel.ID, MS: 400})
	if a := next(t, srv, aliceID, "outcome"); !a.Won || !a.TheirFoul {
		t.Fatalf("alice after bob's foul %+v", a)
	}
	next(t, srv, aliceID, "duel") // the new score

	// Leaving ends it for the other.
	srv.Leave(bob)
	if d := next(t, srv, aliceID, "duel"); d.Duel != nil || d.Note == "" {
		t.Fatalf("after bob left %+v", d)
	}
}
