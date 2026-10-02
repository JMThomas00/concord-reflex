// Command concord-reflex is a reaction-time game: press Space, wait, and
// press it again the moment the screen says NOW.
//
// In Concord the game runs in each player's client (clientcode/, as signed
// WebAssembly) and this program keeps each channel's leaderboard. Run from
// a terminal, it's the same game on its own.
package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"

	"github.com/JMThomas00/Concord/sdk/plugin"
)

func main() {
	cfg, underConcord := plugin.ConfigFromEnv()
	if !underConcord {
		if err := runLocal(); err != nil {
			log.Fatal(err)
		}
		return
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := plugin.Run(ctx, cfg, newServer().handler()); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatal(err)
	}
}
