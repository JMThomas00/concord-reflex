package main

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"github.com/JMThomas00/Concord/sdk/plugin"
	"github.com/JMThomas00/Concord/sdk/plugintest"
)

// Reported times reach everyone viewing the channel as a leaderboard;
// impossible ones are ignored.
func TestLeaderboard(t *testing.T) {
	srv := plugintest.NewServer(t)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go plugin.Run(ctx, srv.Config(), newServer().handler())
	srv.WaitReady()
	ch := uuid.New()
	alice := srv.Enter(ch, "alice", 60, 20)
	srv.Enter(ch, "bob", 60, 20)
	srv.FrameContaining(alice, "runs on your own computer")

	srv.ClientMessage(alice, map[string]int{"ms": 5}) // not human: ignored
	srv.ClientMessage(alice, map[string]int{"ms": 240})
	got := map[uuid.UUID]bool{}
	for len(got) < 2 {
		m, to := srv.NextClientMessage()
		var board struct {
			Top []struct {
				Name string
				MS   int
			}
		}
		_ = json.Unmarshal(m.Data, &board)
		if len(board.Top) == 0 {
			continue // the empty boards sent on Enter
		}
		if len(board.Top) != 1 || board.Top[0].Name != "alice" || board.Top[0].MS != 240 {
			t.Fatalf("board %+v", board)
		}
		got[to] = true
	}
}
