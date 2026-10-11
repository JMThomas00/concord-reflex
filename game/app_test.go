package game

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
)

// fakeEnv records what the app asks for.
type fakeEnv struct {
	timers map[string]time.Duration
	sent   []Up
	sounds []string
}

func (e *fakeEnv) After(id string, d time.Duration) { e.timers[id] = d }
func (e *fakeEnv) Send(up Up)                       { e.sent = append(e.sent, up) }
func (e *fakeEnv) Sound(s string)                   { e.sounds = append(e.sounds, s) }

// fire runs the pending timer whose id starts with name.
func (e *fakeEnv) fire(t *testing.T, a *App, name string) {
	t.Helper()
	for id := range e.timers {
		if strings.HasPrefix(id, name) {
			delete(e.timers, id)
			a.Timer(id)
			return
		}
	}
	t.Fatalf("no %s timer: %v", name, e.timers)
}

func (e *fakeEnv) last(t *testing.T) Up {
	t.Helper()
	if len(e.sent) == 0 {
		t.Fatal("nothing sent")
	}
	return e.sent[len(e.sent)-1]
}

// newApp is an 80 x 24 app with a clock the test moves and dice that
// always roll 0 (the outlaw draws 40 ms faster than his level, the word
// comes after 1.5 s).
func newApp() (*App, *fakeEnv, *time.Time) {
	env := &fakeEnv{timers: map[string]time.Duration{}}
	now := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	a := New(env, nil)
	a.W, a.H = 80, 24
	a.Now = func() time.Time { return now }
	a.Rand = func(int) int { return 0 }
	a.Start()
	return a, env, &now
}

func shown(a *App) string { return ansi.Strip(a.View()) }

func TestQuickDraw(t *testing.T) {
	a, env, now := newApp()
	if env.last(t).T != "hello" {
		t.Fatal("no hello to the server half")
	}
	if !strings.Contains(shown(a), "PRESS ENTER") || a.ClaimsEsc() {
		t.Fatalf("title (claims esc %v):\n%s", a.ClaimsEsc(), shown(a))
	}
	a.Key("enter")
	if !strings.Contains(shown(a), "QUICK DRAW") || !a.ClaimsEsc() {
		t.Fatalf("menu:\n%s", shown(a))
	}
	a.Key("enter") // QUICK DRAW, normal
	a.Key(" ")     // ready
	if a.ph != phWait || env.timers["draw:1"] != 1500*time.Millisecond {
		t.Fatalf("phase %v, timers %v", a.ph, env.timers)
	}
	env.fire(t, a, "draw")
	if a.ph != phDraw {
		t.Fatal("no DRAW!")
	}
	*now = now.Add(205 * time.Millisecond)
	a.Key(" ")
	up := env.last(t)
	if up.T != "result" || up.Mode != ModeDuel || up.MS != 205 || !up.Won || up.Level != 1 {
		t.Fatalf("sent %+v", up) // the outlaw drew in 260 ms
	}
	s := shown(a)
	for _, want := range []string{"205 MS · SHERIFF", "A NEW BEST!", "OUTLAW"} {
		if !strings.Contains(s, want) {
			t.Fatalf("result missing %q:\n%s", want, s)
		}
	}
	if a.score != [2]int{1, 0} {
		t.Fatalf("score %v", a.score)
	}
	// Stale timers from that round do nothing.
	a.Timer("slow:1")
	if a.ph != phResult {
		t.Fatal("a stale timer changed the result")
	}
}

func TestTooSoonAndTooFast(t *testing.T) {
	a, env, now := newApp()
	a.Key("enter")
	a.Key("enter")
	a.Key(" ")
	a.Key(" ") // during WAIT...
	if a.ph != phFoul || !env.last(t).Foul || !strings.Contains(shown(a), "THE DUEL GOES TO THE OUTLAW") {
		t.Fatalf("foul: %v %+v\n%s", a.ph, env.last(t), shown(a))
	}
	a.Key(" ") // again
	env.fire(t, a, "draw")
	*now = now.Add(60 * time.Millisecond) // faster than anyone reacts: a guess
	a.Key(" ")
	if a.ph != phFoul {
		t.Fatal("a 60 ms draw counted")
	}
}

func TestTooSlowAndPractice(t *testing.T) {
	a, env, now := newApp()
	a.Key("enter")
	a.Key("enter")
	a.Key(" ")
	env.fire(t, a, "draw")
	env.fire(t, a, "slow")
	if up := env.last(t); a.ph != phResult || up.Won || up.MS != 0 {
		t.Fatalf("too slow: %v %+v", a.ph, up)
	}
	a.Key("esc")
	for a.menuItems()[a.cursor].label != "PRACTICE" {
		a.Key("down")
	}
	a.Key("enter")
	a.Key(" ")
	env.fire(t, a, "draw")
	*now = now.Add(300 * time.Millisecond)
	a.Key(" ")
	if up := env.last(t); up.Mode != ModePractice || up.MS != 300 || !strings.Contains(shown(a), "300 MS · GREENHORN") {
		t.Fatalf("practice: %+v\n%s", up, shown(a))
	}
}

// A showdown: the server starts the round, each client times its own
// player, the server judges.
func TestShowdown(t *testing.T) {
	a, env, now := newApp()
	a.Server(Down{T: "state", Invites: []Invite{{ID: "i1", Name: "juniper"}}})
	a.Key("enter")
	if !strings.Contains(shown(a), "★ 1 CHALLENGE FOR YOU!") {
		t.Fatalf("menu:\n%s", shown(a))
	}
	a.Key("down")
	a.Key("enter") // SHOWDOWN
	if env.last(t).T != "people" {
		t.Fatal("didn't ask who's here")
	}
	a.Server(Down{T: "people", People: []Person{{ID: "u2", Name: "bob", Online: true, Best: 240}}})
	s := shown(a)
	if !strings.Contains(s, "JUNIPER CHALLENGES YOU") || !strings.Contains(s, "BOB") {
		t.Fatalf("lobby:\n%s", s)
	}
	a.Key("enter") // accept juniper's
	if up := env.last(t); up.T != "answer" || up.ID != "i1" || !up.Accept {
		t.Fatalf("sent %+v", up)
	}
	a.Server(Down{T: "duel", Duel: &Duel{ID: "d1", Opponent: "juniper"}})
	if a.scr != scrDuel || a.mode != ModeShowdown || !strings.Contains(shown(a), "SHOWDOWN · VS JUNIPER") {
		t.Fatalf("duel:\n%s", shown(a))
	}
	a.Key(" ")
	if up := env.last(t); up.T != "ready" || a.ph != phWaitOpp {
		t.Fatalf("ready: %+v", up)
	}
	a.Server(Down{T: "go", ID: "d1", Delay: 2200})
	if a.ph != phWait || env.timers["draw:1"] != 2200*time.Millisecond {
		t.Fatalf("go: %v %v", a.ph, env.timers)
	}
	env.fire(t, a, "draw")
	*now = now.Add(205 * time.Millisecond)
	a.Key(" ")
	if up := env.last(t); up.T != "shot" || up.MS != 205 || a.ph != phShot {
		t.Fatalf("shot: %+v", up)
	}
	a.Server(Down{T: "outcome", ID: "d1", Mine: 205, Theirs: 231, Won: true})
	a.Server(Down{T: "duel", Duel: &Duel{ID: "d1", Opponent: "juniper", ScoreUs: 1}})
	if s := shown(a); !strings.Contains(s, "205 MS · SHERIFF") || a.score != [2]int{1, 0} {
		t.Fatalf("outcome:\n%s", s)
	}
	a.Server(Down{T: "duel", Note: "your opponent left"})
	if a.scr != scrPeople || !strings.Contains(shown(a), "YOUR OPPONENT LEFT") {
		t.Fatalf("after:\n%s", shown(a))
	}
}

func TestStoreAndDraft(t *testing.T) {
	a, env, _ := newApp()
	a.Server(Down{T: "state", Me: &Profile{Bounties: 1, Offer: []string{"grape", "scene:night", "bowler"}}})
	a.Key("enter")
	for a.menuItems()[a.cursor].label != "GENERAL STORE" {
		a.Key("down")
	}
	if !strings.Contains(shown(a), "★ 1 BOUNTY · NEW GEAR!") {
		t.Fatalf("menu:\n%s", shown(a))
	}
	a.Key("enter")
	if s := shown(a); !strings.Contains(s, "◂ STETSON ▸") || !strings.Contains(s, "✓ IN USE") {
		t.Fatalf("store:\n%s", s)
	}
	a.Key("right")
	a.Key("right") // WHITE HAT: locked
	a.Key("enter")
	if env.last(t).T != "deal" || a.scr != scrDraft {
		t.Fatal("no draft")
	}
	if s := shown(a); !strings.Contains(s, "GRAPE HEAD") || !strings.Contains(s, "MIDNIGHT") {
		t.Fatalf("draft:\n%s", s)
	}
	a.Key("enter")
	if up := env.last(t); up.T != "pick" || up.ID != "grape" {
		t.Fatalf("pick %+v", up)
	}
	a.Server(Down{T: "state", Me: &Profile{Owned: []string{"grape"}, Equipped: map[string]string{KindHat: "grape"}}})
	a.reveal = a.reveal.Add(-revealFor)
	env.fire(t, a, "tick")
	if s := shown(a); a.scr != scrStore || !strings.Contains(s, "◂ GRAPE HEAD ▸") {
		t.Fatalf("after the reveal:\n%s", s)
	}
}

// Every screen draws at full width, at 80 x 24 and bigger, and a small pane
// gets practice.
func TestScreens(t *testing.T) {
	for _, size := range [][2]int{{80, 24}, {120, 40}, {64, 24}} {
		a, _, _ := newApp()
		a.W, a.H = size[0], size[1]
		a.Server(Down{T: "state", Me: &Profile{Best: 199, Bounties: 2, Offer: []string{"robot", "scene:moon", "tophat"}},
			Top: []Entry{{Name: "merlot", MS: 163, Wins: 9}, {Name: "you", MS: 199, Me: true}}})
		for s := scrTitle; s <= scrOptions; s++ {
			a.scr = s
			for i, line := range strings.Split(a.View(), "\n") {
				if w := ansi.StringWidth(line); w != size[0] {
					t.Fatalf("screen %d at %v: line %d is %d wide", s, size, i, w)
				}
			}
		}
	}
	a, env, now := newApp()
	a.W, a.H = 40, 12
	a.Key(" ")
	env.fire(t, a, "draw")
	*now = now.Add(250 * time.Millisecond)
	a.Key(" ")
	if s := shown(a); !strings.Contains(s, "250 ms · RANGER") {
		t.Fatalf("small pane:\n%s", s)
	}
}

// The clock starts when DRAW! is on screen, not when the app decided to
// show it: drawing takes time (a lot, in client code), and that time isn't
// the player's. Nothing redraws during WAIT... or DRAW!, so a key press
// never waits behind a frame.
func TestTheClockStartsWhenDrawIsShown(t *testing.T) {
	a, env, now := newApp()
	a.Key("enter")
	a.Key("enter")
	a.Key(" ")
	if a.animated() {
		t.Fatal("animating during WAIT...: a redraw would hold up the key press")
	}
	env.fire(t, a, "bell")
	if !a.Quiet() {
		t.Fatal("the bell asked for a redraw")
	}
	env.fire(t, a, "draw")
	if a.Quiet() || a.animated() {
		t.Fatal("DRAW! must be drawn once, then nothing else")
	}
	*now = now.Add(150 * time.Millisecond) // drawing it
	a.Shown()
	*now = now.Add(240 * time.Millisecond) // the player
	a.Key(" ")
	if a.mine != 240 {
		t.Fatalf("timed %d ms, want 240", a.mine)
	}
}
