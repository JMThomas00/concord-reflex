// Package game is Reflex itself: a reaction-time game. Press Space to
// start, wait, and press it again the moment the screen says NOW. The same
// game runs as Concord client code (in each player's client) and as a
// standalone terminal game; Env is the difference between them.
package game

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

// Phase is where a round is.
type Phase int

const (
	Idle    Phase = iota // before the first round
	Waiting              // started; NOW is coming
	Now                  // press!
	Result               // a time to show
	Early                // pressed before NOW
)

// Entry is one line of a leaderboard.
type Entry struct {
	Name string `json:"name"`
	MS   int    `json:"ms"`
}

// Env is what the game needs from wherever it runs.
type Env interface {
	// After asks for Timer(id) after d.
	After(id string, d time.Duration)
	// SaveBest remembers a new personal best.
	SaveBest(ms int)
	// Report hands a finished time to a leaderboard, if there is one.
	Report(ms int)
}

// Game is one player's game.
type Game struct {
	Width, Height int
	Phase         Phase
	Last, Best    int
	Board         []Entry // the channel's leaderboard (Concord only)
	Colors        Colors

	goAt time.Time
	spin int

	// Now and Delay are the clock and the wait before NOW; tests replace them.
	Now   func() time.Time
	Delay func() time.Duration
}

// New starts a game with a personal best (0 for none).
func New(best int) *Game {
	return &Game{
		Phase: Idle, Best: best, Colors: DefaultColors(),
		Now:   time.Now,
		Delay: func() time.Duration { return time.Duration(1200+rand.Intn(2600)) * time.Millisecond },
	}
}

// ParseBest reads a stored personal best.
func ParseBest(s string) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || n < 0 {
		return 0
	}
	return n
}

// Timer ids.
const (
	timerGo   = "go"
	timerSpin = "spin"
)

// spinEvery is how often the waiting animation moves.
const spinEvery = 90 * time.Millisecond

// Key handles a keypress (Bubble Tea's name for it).
func (g *Game) Key(key string, env Env) {
	if key != " " && key != "space" && key != "enter" {
		return
	}
	switch g.Phase {
	case Idle, Result, Early:
		g.Phase, g.spin = Waiting, 0
		env.After(timerGo, g.Delay())
		env.After(timerSpin, spinEvery)
	case Waiting:
		g.Phase = Early
	case Now:
		g.Last = int(g.Now().Sub(g.goAt).Milliseconds())
		g.Phase = Result
		if g.Best == 0 || g.Last < g.Best {
			g.Best = g.Last
			env.SaveBest(g.Best)
		}
		env.Report(g.Last)
	}
}

// Timer handles a timer set with Env.After.
func (g *Game) Timer(id string, env Env) {
	if g.Phase != Waiting {
		return // a timer from a round that ended early
	}
	switch id {
	case timerGo:
		g.Phase, g.goAt = Now, g.Now()
	case timerSpin:
		g.spin++
		env.After(timerSpin, spinEvery)
	}
}

// Rating says how good a time is.
func Rating(ms int) string {
	switch {
	case ms < 180:
		return "Lightning!"
	case ms < 230:
		return "Excellent"
	case ms < 280:
		return "Good"
	case ms < 350:
		return "Average"
	}
	return "Keep practising"
}

// View draws the game in Width x Height cells.
func (g *Game) View() string {
	c := g.Colors
	var lines []string
	add := func(s ...string) { lines = append(lines, s...) }

	add(c.Accent("R E F L E X"), "")
	switch g.Phase {
	case Idle:
		add("Press "+c.Key("Space")+" to start, then press it again",
			"the moment the screen says "+c.Good("NOW")+".")
	case Waiting:
		add(c.Bad("Wait for it…"), "", c.Dim(dots(g.spin, 11)))
	case Now:
		w := min(max(g.Width-4, 12), 30)
		bar := strings.Repeat("█", w)
		label := center("NOW!", w)
		add(c.Good(bar), c.GoodBlock(label), c.Good(bar))
	case Result:
		add(c.Good(fmt.Sprintf("%d ms", g.Last))+"  "+c.Dim(Rating(g.Last)), "",
			"Press "+c.Key("Space")+" to go again.")
	case Early:
		add(c.Bad("Too soon!"), "", "Press "+c.Key("Space")+" to try again.")
	}
	add("")
	if g.Best > 0 {
		add(c.Dim(fmt.Sprintf("Your best: %d ms", g.Best)))
	}
	if len(g.Board) > 0 {
		add("", c.Accent("Leaderboard"))
		for i, e := range g.Board {
			name := []rune(e.Name)
			if len(name) > 16 {
				name = append(name[:15], '…')
			}
			add(fmt.Sprintf("%s %-16s %s", c.Dim(fmt.Sprintf("%d.", i+1)), string(name), c.Good(fmt.Sprintf("%4d ms", e.MS))))
		}
	}
	if g.Height > 0 && len(lines) > g.Height {
		lines = lines[:g.Height]
	}
	return strings.Join(lines, "\n")
}

// dots is the waiting animation: a dot sweeping across n places.
func dots(step, n int) string {
	pos := step % (2*n - 2)
	if pos >= n {
		pos = 2*n - 2 - pos
	}
	row := []rune(strings.Repeat("·", n))
	row[pos] = '●'
	return string(row)
}

func center(s string, w int) string {
	pad := max(w-len([]rune(s)), 0)
	return strings.Repeat(" ", pad/2) + s + strings.Repeat(" ", pad-pad/2)
}
