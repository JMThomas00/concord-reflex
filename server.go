package main

import (
	"encoding/json"
	"sort"

	"github.com/google/uuid"

	"github.com/JMThomas00/Concord/sdk/plugin"
	"github.com/JMThomas00/Concord/sdk/wire"
	"github.com/JMThomas00/concord-reflex/game"
)

// The server half keeps each channel's leaderboard. The game itself runs in
// players' clients; anyone who doesn't run its code sees what this draws.

// Times outside this range aren't human, or are made up: client code runs
// on the player's own computer, so what it reports can't be trusted.
const minMS, maxMS = 80, 10000

type viewer struct{ channel, id uuid.UUID }

type server struct {
	best    map[uuid.UUID]map[string]int // channel → player → best ms
	viewers map[viewer]bool
}

func newServer() *server {
	return &server{best: map[uuid.UUID]map[string]int{}, viewers: map[viewer]bool{}}
}

// top is a channel's five best players.
func (s *server) top(ch uuid.UUID) []game.Entry {
	var out []game.Entry
	for name, ms := range s.best[ch] {
		out = append(out, game.Entry{Name: name, MS: ms})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].MS != out[j].MS {
			return out[i].MS < out[j].MS
		}
		return out[i].Name < out[j].Name
	})
	return out[:min(len(out), 5)]
}

const fallback = "R E F L E X\n\nReflex runs on your own computer.\n\nAllow its code when Concord asks, or turn plugin\ncode on in Settings > Display > Plugin Code."

func (s *server) handler() plugin.Handler {
	return plugin.Handler{
		OnEnter: func(c *plugin.Conn, e wire.PluginPaneEnterPayload) {
			s.viewers[viewer{e.ChannelID, e.ViewerID}] = true
			_ = c.Frame(e.ChannelID, e.ViewerID, fallback)
			_ = c.SendToClient(e.ChannelID, e.ViewerID, map[string]any{"top": s.top(e.ChannelID)})
		},
		OnLeave: func(c *plugin.Conn, e wire.PluginPaneLeavePayload) {
			delete(s.viewers, viewer{e.ChannelID, e.ViewerID})
		},
		OnChannelDelete: func(c *plugin.Conn, e wire.ChannelDeletePayload) {
			delete(s.best, e.ChannelID)
		},
		OnClientMessage: func(c *plugin.Conn, from uuid.UUID, m wire.PluginClientMessagePayload) {
			var report struct {
				MS int `json:"ms"`
			}
			if json.Unmarshal(m.Data, &report) != nil || report.MS < minMS || report.MS > maxMS {
				return
			}
			name := m.ViewerDisplayName
			if name == "" {
				name = m.ViewerName
			}
			if s.best[m.ChannelID] == nil {
				s.best[m.ChannelID] = map[string]int{}
			}
			if old, ok := s.best[m.ChannelID][name]; ok && old <= report.MS {
				return // not a new best: the board hasn't changed
			}
			s.best[m.ChannelID][name] = report.MS
			board := map[string]any{"top": s.top(m.ChannelID)}
			for v := range s.viewers {
				if v.channel == m.ChannelID {
					_ = c.SendToClient(v.channel, v.id, board)
				}
			}
		},
	}
}
