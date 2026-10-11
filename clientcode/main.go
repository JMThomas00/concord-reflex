// Reflex's client code: the whole arcade runs in each player's Concord
// client, so the time is exact and the animation smooth whatever the
// network does. Records, Bounties, the Hall of Fame and showdowns go
// through the server half.
//
// It's built for WebAssembly by release.go (GOOS=wasip1 GOARCH=wasm) and
// signed with the publisher key; Concord runs nothing unsigned.
package main

import (
	"encoding/json"
	"time"

	"github.com/JMThomas00/Concord/sdk/client"
	"github.com/JMThomas00/Concord/sdk/wire"
	"github.com/JMThomas00/concord-reflex/game"
)

// env is how the app reaches Concord.
type env struct{}

func (env) After(id string, d time.Duration) { _ = client.After(id, d) }
func (env) Send(up game.Up)                  { _ = client.Send(up) }
func (env) Sound(asset string)               { _ = client.PlaySound(asset, 0) }

func handler() client.Handler {
	app := game.New(env{}, nil)
	claimed := false
	draw := func() {
		_ = client.Frame(app.View())
		app.Shown() // on screen now: DRAW!'s clock starts here
		if want := app.ClaimsEsc(); want != claimed {
			claimed = want
			if want {
				_ = client.ClaimKeys(wire.PaneKeyEsc)
			} else {
				_ = client.ClaimKeys()
			}
		}
	}
	sized := func(e client.Event) {
		app.W, app.H = e.Width, e.Height
		app.SetTheme(e.Theme)
	}
	return client.Handler{
		OnStart: func(e client.Event) {
			sized(e)
			_ = client.ForwardKeys(false) // the whole game is here
			app.Start()
			draw()
		},
		OnResize: func(e client.Event) { sized(e); draw() },
		OnKey:    func(e client.Event) { app.Key(e.Key); draw() },
		OnTimer: func(id string) {
			app.Timer(id)
			if !app.Quiet() { // a bell or a sound: nothing to redraw
				draw()
			}
		},
		OnServer: func(data json.RawMessage) {
			var d game.Down
			if json.Unmarshal(data, &d) == nil {
				app.Server(d)
				draw()
			}
		},
	}
}

func main() { client.Run(handler()) }
