package game

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
)

// fakeEnv records what the game asks of where it runs.
type fakeEnv struct {
	timers  map[string]time.Duration
	best    int
	reports []int
}

func (e *fakeEnv) After(id string, d time.Duration) { e.timers[id] = d }
func (e *fakeEnv) SaveBest(ms int)                  { e.best = ms }
func (e *fakeEnv) Report(ms int)                    { e.reports = append(e.reports, ms) }

func newGame(best int) (*Game, *fakeEnv, *time.Time) {
	now := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	g := New(best)
	g.Width, g.Height = 40, 20
	g.Now = func() time.Time { return now }
	g.Delay = func() time.Duration { return 2 * time.Second }
	return g, &fakeEnv{timers: map[string]time.Duration{}}, &now
}

func view(g *Game) string { return ansi.Strip(g.View()) }

func TestARound(t *testing.T) {
	g, env, now := newGame(0)
	if !strings.Contains(view(g), "Press Space to start") {
		t.Fatalf("idle:\n%s", view(g))
	}
	g.Key(" ", env)
	if g.Phase != Waiting || env.timers["go"] != 2*time.Second || env.timers["spin"] == 0 {
		t.Fatalf("start: phase %v timers %v", g.Phase, env.timers)
	}
	g.Timer("spin", env)
	if !strings.Contains(view(g), "Wait for it") {
		t.Fatalf("waiting:\n%s", view(g))
	}
	g.Timer("go", env)
	if !strings.Contains(view(g), "NOW!") {
		t.Fatalf("now:\n%s", view(g))
	}
	*now = now.Add(213 * time.Millisecond)
	g.Key("space", env)
	if g.Last != 213 || env.best != 213 || len(env.reports) != 1 || env.reports[0] != 213 {
		t.Fatalf("result: last %d best %d reports %v", g.Last, env.best, env.reports)
	}
	if v := view(g); !strings.Contains(v, "213 ms") || !strings.Contains(v, "Excellent") || !strings.Contains(v, "Your best: 213 ms") {
		t.Fatalf("result:\n%s", v)
	}
}

// Pressing before NOW is "too soon", and the round's timers are ignored.
func TestTooSoon(t *testing.T) {
	g, env, _ := newGame(180)
	g.Key("enter", env)
	g.Key("enter", env)
	if g.Phase != Early || !strings.Contains(view(g), "Too soon") {
		t.Fatalf("early: %v\n%s", g.Phase, view(g))
	}
	g.Timer("go", env)
	if g.Phase != Early || len(env.reports) != 0 {
		t.Fatal("a stale timer started NOW after a false start")
	}
}

// A slower time doesn't replace the best.
func TestBestKeepsTheFastest(t *testing.T) {
	g, env, now := newGame(150)
	g.Key(" ", env)
	g.Timer("go", env)
	*now = now.Add(300 * time.Millisecond)
	g.Key(" ", env)
	if g.Best != 150 || env.best != 0 {
		t.Fatalf("best %d saved %d", g.Best, env.best)
	}
}

func TestLeaderboardAndSmallPanes(t *testing.T) {
	g, _, _ := newGame(0)
	g.Board = []Entry{{"alice", 190}, {"a-very-long-player-name", 240}}
	v := view(g)
	if !strings.Contains(v, "1. alice") || !strings.Contains(v, "a-very-long-pla…") {
		t.Fatalf("board:\n%s", v)
	}
	g.Height = 3
	if n := len(strings.Split(g.View(), "\n")); n > 3 {
		t.Fatalf("%d lines in a 3-line pane", n)
	}
}

func TestColorsFollowThePalette(t *testing.T) {
	c := FromPalette(map[string]string{"green": "#50fa7b"})
	if got := c.Good("x"); got != "\x1b[1;38;2;80;250;123mx\x1b[0m" {
		t.Fatalf("hex green: %q", got)
	}
	if got := FromPalette(map[string]string{"red": "9"}).Bad("x"); got != "\x1b[1;91mx\x1b[0m" {
		t.Fatalf("ANSI bright red: %q", got)
	}
	if got := FromPalette(map[string]string{"comment": "nonsense"}).Dim("x"); got != "\x1b[90mx\x1b[0m" {
		t.Fatalf("unreadable color should fall back: %q", got)
	}
}
