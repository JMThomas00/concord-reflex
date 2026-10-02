// Reflex's client code: the game runs in each player's Concord client, so
// timing is exact and the animation smooth whatever the network does. Only
// finished times go to the server half, for the channel's leaderboard.
//
// It's built for WebAssembly by release.go (GOOS=wasip1 GOARCH=wasm) and
// signed with the publisher key; Concord runs nothing unsigned.
package main

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/JMThomas00/Concord/sdk/client"
	"github.com/JMThomas00/concord-reflex/game"
)

// env is how the game reaches Concord.
type env struct{}

func (env) After(id string, d time.Duration) { _ = client.After(id, d) }
func (env) SaveBest(ms int)                  { _ = client.Set("best", strconv.Itoa(ms)) }
func (env) Report(ms int)                    { _ = client.Send(map[string]int{"ms": ms}) }

func handler() client.Handler {
	g := game.New(0)
	draw := func() { _ = client.Frame(g.View()) }
	sized := func(e client.Event) {
		g.Width, g.Height = e.Width, e.Height
		if e.Theme != nil {
			g.Colors = game.FromPalette(e.Theme.Palette)
		}
	}
	return client.Handler{
		OnStart: func(e client.Event) {
			sized(e)
			if v, ok, _ := client.Get("best"); ok {
				g.Best = game.ParseBest(v)
			}
			_ = client.ForwardKeys(false) // the whole game is here
			draw()
		},
		OnResize: func(e client.Event) { sized(e); draw() },
		OnKey:    func(e client.Event) { g.Key(e.Key, env{}); draw() },
		OnTimer:  func(id string) { g.Timer(id, env{}); draw() },
		OnServer: func(data json.RawMessage) {
			var m struct {
				Top []game.Entry `json:"top"`
			}
			if json.Unmarshal(data, &m) == nil {
				g.Board = m.Top
				draw()
			}
		},
	}
}

func main() { client.Run(handler()) }
