package game

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/JMThomas00/Concord/sdk/arcade"
	"github.com/JMThomas00/Concord/sdk/wire"
)

// Env is what the app needs from wherever it runs: Concord client code, or
// a terminal on its own.
type Env interface {
	// After asks for Timer(id) after d.
	After(id string, d time.Duration)
	// Send hands a message to the server half.
	Send(up Up)
	// Sound plays a sound from the plugin's client/ folder.
	Sound(asset string)
}

// Sounds: the arcade kit's, and Reflex's own (tools/gen.go).
const (
	sndBlip   = arcade.SoundBlip
	sndSelect = arcade.SoundSelect
	sndBack   = arcade.SoundBack
	sndCoin   = arcade.SoundCoin
	sndFoul   = arcade.SoundRed
	sndWin    = arcade.SoundFinish
	sndRecord = arcade.SoundRecord
	sndTick   = "sounds/tick.wav"
	sndDraw   = "sounds/draw.wav"
	sndBang   = "sounds/bang.wav"
	sndLose   = "sounds/lose.wav"
)

type screen int

const (
	scrTitle screen = iota
	scrMenu
	scrDuel
	scrPeople
	scrStore
	scrDraft
	scrHOF
	scrHowTo
	scrOptions
)

// phase is where a duel is.
type phase int

const (
	phReady   phase = iota // press Space when ready
	phWaitOpp              // showdown: waiting for the other player to be ready
	phWait                 // WAIT...
	phDraw                 // DRAW!
	phShot                 // showdown: drawn, waiting for the other player's time
	phResult               // a time to show
	phFoul                 // drew too soon
)

const (
	tickEvery    = 100 * time.Millisecond
	attractFor   = 30 * time.Second
	revealFor    = 2500 * time.Millisecond
	tooSlowFor   = 3 * time.Second
	minWait      = 1500
	waitSpread   = 2500
	outlawJitter = 40
)

// App is one player's Reflex: every screen, and the duels.
type App struct {
	W, H int
	Pal  arcade.Palette

	// Standalone is a terminal game with no one to duel: no showdowns.
	Standalone bool

	// Now and Rand are the clock and the dice; tests replace them.
	Now  func() time.Time
	Rand func(n int) int

	env     Env
	scr     screen
	frame   int
	lastKey time.Time
	ticking bool
	cursor  int
	level   int
	notice  string

	me      Profile
	top     []Entry
	invites []Invite
	people  []Person
	asked   bool // people requested

	// the duel
	mode      string
	ph        phase
	round     int // stale timers from earlier rounds are ignored
	drawAt    time.Time
	theirsAt  int
	mine      int
	theirs    int
	won, tie  bool
	myFoul    bool
	theirFoul bool
	record    bool // a new best
	score     [2]int
	showdown  *Duel

	// the store and the draft
	row    int
	idx    [2]int
	sel    int
	picked string
	reveal time.Time
}

// New starts the app on the title screen.
func New(env Env, theme *wire.PaneTheme) *App {
	a := &App{env: env, Pal: arcade.NewPalette(theme), level: 1, Now: time.Now, Rand: rand.IntN}
	a.lastKey = a.Now()
	return a
}

// Start says hello to the server half and starts the title's animation.
func (a *App) Start() {
	a.env.Send(Up{T: "hello"})
	a.kick()
}

// SetTheme follows the viewer's Concord theme.
func (a *App) SetTheme(t *wire.PaneTheme) {
	if t != nil {
		a.Pal = arcade.NewPalette(t)
	}
}

func (a *App) sound(s string) {
	if !a.me.Muted {
		a.env.Sound(s)
	}
}

func (a *App) owns(id string) bool {
	return slices.Contains(a.me.Owned, id) || itemTier(id) == arcade.Starter
}

func itemTier(id string) arcade.Tier {
	for _, u := range Unlockables() {
		if u.ID == id {
			return u.Tier
		}
	}
	return arcade.Common
}

func (a *App) hat() string {
	if id := a.me.Equipped[KindHat]; id != "" && a.owns(id) {
		return id
	}
	return Hats[0].ID
}

func (a *App) scene() string {
	if id := a.me.Equipped[KindScene]; id != "" && a.owns(id) {
		return id
	}
	return Scenes[0].ID
}

// ── Animation ──────────────────────────────────────────────────────────────

// animated reports whether the screen moves on its own now.
func (a *App) animated() bool {
	if !a.reveal.IsZero() {
		return true // the draft's reveal moves on by itself
	}
	if a.me.Effects != "" {
		return false
	}
	switch a.scr {
	case scrTitle:
		return a.Now().Sub(a.lastKey) < attractFor
	case scrDuel:
		return a.ph == phWait || a.ph == phDraw || a.ph == phShot || a.ph == phWaitOpp ||
			((a.ph == phResult || a.ph == phFoul) && a.Now().Sub(a.lastKey) < 10*time.Second)
	}
	return false
}

func (a *App) kick() {
	if !a.ticking && a.animated() {
		a.ticking = true
		a.env.After("tick", tickEvery)
	}
}

// blink is on half of each second (always on with effects off).
func (a *App) blink() bool { return a.me.Effects != "" || a.frame/5%2 == 0 }

// ── Keys ───────────────────────────────────────────────────────────────────

// Key handles a keypress, named as Bubble Tea names it.
func (a *App) Key(key string) {
	a.lastKey = a.Now()
	defer a.kick()
	if key == "enter" && (a.scr == scrDuel) {
		key = " " // Enter draws too
	}
	if !a.big() {
		a.smallKey(key)
		return
	}
	switch a.scr {
	case scrTitle:
		if key == "enter" || key == " " {
			a.sound(sndCoin)
			a.goTo(scrMenu)
		}
	case scrMenu:
		a.menuKey(key)
	case scrDuel:
		a.duelKey(key)
	case scrPeople:
		a.peopleKey(key)
	case scrStore:
		a.storeKey(key)
	case scrDraft:
		a.draftKey(key)
	case scrHOF, scrHowTo:
		if key == "esc" || key == "enter" || key == "q" {
			a.back()
		}
	case scrOptions:
		a.optionsKey(key)
	}
}

func (a *App) goTo(s screen) {
	a.scr, a.cursor, a.notice = s, 0, ""
}

func (a *App) back() {
	a.sound(sndBack)
	a.goTo(scrMenu)
}

type menuItem struct {
	label string
	desc  func() (string, string)
	enter func()
	turn  func(d int)
}

func (a *App) menuItems() []menuItem {
	items := []menuItem{{
		label: "QUICK DRAW",
		desc: func() (string, string) {
			return fmt.Sprintf("◂ %s ▸  THE OUTLAW DRAWS IN ~%d MS", Levels[a.level].Name, Levels[a.level].MS), "fg"
		},
		enter: func() { a.startDuel(ModeDuel) },
		turn:  func(d int) { a.level = min(len(Levels)-1, max(0, a.level+d)) },
	}}
	if !a.Standalone {
		items = append(items, menuItem{label: "SHOWDOWN", desc: func() (string, string) {
			if n := a.incoming(); n > 0 {
				return fmt.Sprintf("★ %d %s FOR YOU!", n, plural(n, "CHALLENGE")), "yellow"
			}
			return "DUEL SOMEONE IN THE CHANNEL", "dim"
		}, enter: func() { a.openPeople() }})
	}
	items = append(items,
		menuItem{label: "PRACTICE", desc: func() (string, string) { return "JUST YOU AND THE CLOCK", "dim" }, enter: func() { a.startDuel(ModePractice) }},
		menuItem{label: "GENERAL STORE", desc: func() (string, string) {
			if n := a.me.Bounties; n > 0 {
				return fmt.Sprintf("★ %d %s · NEW GEAR!", n, plural(n, "BOUNTY", "BOUNTIES")), "yellow"
			}
			return fmt.Sprintf("%d OF %d COLLECTED", a.collected(), len(Unlockables())), "dim"
		}, enter: func() { a.goTo(scrStore) }},
		menuItem{label: "HALL OF FAME", desc: func() (string, string) { return "THE FASTEST DRAWS IN THIS CHANNEL", "dim" }, enter: func() { a.goTo(scrHOF) }},
		menuItem{label: "HOW TO PLAY", desc: func() (string, string) { return "WAIT FOR IT...", "dim" }, enter: func() { a.goTo(scrHowTo) }},
		menuItem{label: "OPTIONS", desc: func() (string, string) {
			return "SOUND " + onOff(!a.me.Muted) + " · EFFECTS " + effectsName(a.me.Effects), "dim"
		}, enter: func() { a.goTo(scrOptions) }},
	)
	return items
}

func (a *App) incoming() int {
	n := 0
	for _, in := range a.invites {
		if !in.Mine {
			n++
		}
	}
	return n
}

func (a *App) collected() int {
	n := 0
	for _, u := range Unlockables() {
		if a.owns(u.ID) {
			n++
		}
	}
	return n
}

func plural(n int, one string, many ...string) string {
	if n == 1 {
		return one
	}
	if len(many) > 0 {
		return many[0]
	}
	return one + "S"
}

func onOff(on bool) string {
	if on {
		return "ON"
	}
	return "OFF"
}

func effectsName(e string) string {
	switch e {
	case "calm":
		return "CALM"
	case "off":
		return "OFF"
	}
	return "FULL"
}

func (a *App) menuKey(key string) {
	items := a.menuItems()
	a.cursor = min(a.cursor, len(items)-1)
	switch key {
	case "up", "k":
		a.cursor = (a.cursor + len(items) - 1) % len(items)
		a.sound(sndBlip)
	case "down", "j":
		a.cursor = (a.cursor + 1) % len(items)
		a.sound(sndBlip)
	case "left", "h", "right", "l":
		if it := items[a.cursor]; it.turn != nil {
			d := 1
			if key == "left" || key == "h" {
				d = -1
			}
			it.turn(d)
			a.sound(sndBlip)
		}
	case "enter", " ":
		a.sound(sndSelect)
		items[a.cursor].enter()
	case "esc", "q":
		a.sound(sndBack)
		a.goTo(scrTitle)
	}
}

// ── Duels ──────────────────────────────────────────────────────────────────

func (a *App) startDuel(mode string) {
	if a.mode != mode {
		a.score = [2]int{}
	}
	a.mode, a.ph, a.mine, a.theirs = mode, phReady, 0, 0
	a.goTo(scrDuel)
}

// startRound is Space at READY?: a quick draw or practice starts its own
// clock; a showdown tells the server, which starts both players together.
func (a *App) startRound() {
	a.mine, a.theirs, a.won, a.tie, a.myFoul, a.theirFoul, a.record = 0, 0, false, false, false, false, false
	if a.mode == ModeShowdown {
		if a.showdown == nil {
			return
		}
		a.ph = phWaitOpp
		a.env.Send(Up{T: "ready", ID: a.showdown.ID})
		return
	}
	a.theirsAt = Levels[a.level].MS + a.Rand(2*outlawJitter+1) - outlawJitter
	a.wait(minWait + a.Rand(waitSpread))
}

// wait starts WAIT...: the word comes after delay ms.
func (a *App) wait(delay int) {
	a.round++
	a.ph = phWait
	a.env.After(a.timer("draw"), time.Duration(delay)*time.Millisecond)
	a.env.After(a.timer("bell"), 500*time.Millisecond)
	a.sound(sndTick)
}

func (a *App) timer(name string) string { return name + ":" + strconv.Itoa(a.round) }

func (a *App) duelKey(key string) {
	switch key {
	case "esc", "q":
		if a.mode == ModeShowdown && a.showdown != nil {
			a.env.Send(Up{T: "leave", ID: a.showdown.ID})
			a.showdown = nil
		}
		a.round++ // any timers still to come are stale now
		a.back()
		return
	case " ":
	default:
		return
	}
	switch a.ph {
	case phReady, phResult, phFoul:
		a.sound(sndSelect)
		a.startRound()
	case phWait:
		a.foul()
	case phDraw:
		ms := int(a.Now().Sub(a.drawAt).Milliseconds())
		if ms < Anticipated {
			a.foul() // a guess, not a reaction
			return
		}
		a.shoot(ms)
	}
}

// foul is drawing before the word.
func (a *App) foul() {
	a.round++
	a.myFoul = true
	a.sound(sndFoul)
	switch a.mode {
	case ModeShowdown:
		a.ph = phShot
		a.env.Send(Up{T: "shot", ID: a.showdown.ID, Foul: true})
	case ModeDuel:
		a.ph, a.theirs = phFoul, a.theirsAt
		a.score[1] = min(9, a.score[1]+1)
		a.env.Send(Up{T: "result", Mode: ModeDuel, Foul: true, Level: a.level})
	default:
		a.ph = phFoul
	}
}

// shoot is drawing ms after the word.
func (a *App) shoot(ms int) {
	a.round++
	a.mine = ms
	a.sound(sndBang)
	a.record = a.me.Best == 0 || ms < a.me.Best
	switch a.mode {
	case ModeShowdown:
		a.ph = phShot
		a.env.Send(Up{T: "shot", ID: a.showdown.ID, MS: ms})
		return
	case ModeDuel:
		a.theirs = a.theirsAt
		a.won = ms < a.theirs
		if a.won {
			a.score[0] = min(9, a.score[0]+1)
		} else {
			a.score[1] = min(9, a.score[1]+1)
		}
		a.env.Send(Up{T: "result", Mode: ModeDuel, MS: ms, Won: a.won, Level: a.level})
	default:
		a.won = true
		a.env.Send(Up{T: "result", Mode: ModePractice, MS: ms})
	}
	a.ph = phResult
	a.fanfare()
}

func (a *App) fanfare() {
	switch {
	case a.record && a.mine > 0:
		a.sound(sndRecord)
	case a.won:
		a.sound(sndWin)
	default:
		a.sound(sndLose)
	}
}

// Timer handles a timer set with Env.After.
func (a *App) Timer(id string) {
	if id == "tick" {
		a.ticking = false
		a.frame++
		if !a.reveal.IsZero() && a.Now().Sub(a.reveal) >= revealFor {
			a.endReveal()
		}
		a.kick()
		return
	}
	name, round, _ := strings.Cut(id, ":")
	if round != strconv.Itoa(a.round) || a.scr != scrDuel {
		return // from a round that's over
	}
	defer a.kick()
	switch name {
	case "bell":
		if a.ph == phWait {
			a.sound(sndTick)
			a.env.After(a.timer("bell"), 500*time.Millisecond)
		}
	case "draw":
		if a.ph != phWait {
			return
		}
		a.ph, a.drawAt = phDraw, a.Now()
		a.sound(sndDraw)
		if a.mode == ModeDuel {
			a.env.After(a.timer("outlaw"), time.Duration(a.theirsAt)*time.Millisecond)
		}
		a.env.After(a.timer("slow"), tooSlowFor)
	case "outlaw":
		if a.ph == phDraw {
			a.sound(sndBang) // he's drawn; you're still too slow
		}
	case "slow":
		if a.ph != phDraw {
			return
		}
		a.round++
		switch a.mode {
		case ModeShowdown:
			a.ph = phShot
			a.env.Send(Up{T: "shot", ID: a.showdown.ID, MS: 5000})
		case ModeDuel:
			a.ph, a.theirs, a.won = phResult, a.theirsAt, false
			a.score[1] = min(9, a.score[1]+1)
			a.env.Send(Up{T: "result", Mode: ModeDuel, Level: a.level})
			a.sound(sndLose)
		default:
			a.ph, a.notice = phReady, "TOO SLOW: SPACE TO TRY AGAIN"
		}
	}
}

// ── The server half ────────────────────────────────────────────────────────

// Server handles a message from the server half.
func (a *App) Server(d Down) {
	defer a.kick()
	switch d.T {
	case "state":
		if d.Me != nil {
			a.me = *d.Me
		}
		a.top, a.invites = d.Top, d.Invites
	case "people":
		a.people = d.People
	case "duel":
		switch {
		case d.Duel != nil && (a.showdown == nil || a.showdown.ID != d.Duel.ID):
			a.showdown = d.Duel
			a.score = [2]int{d.Duel.ScoreUs, d.Duel.ScoreThem}
			a.mode, a.ph = ModeShowdown, phReady
			a.goTo(scrDuel)
		case d.Duel != nil:
			a.showdown = d.Duel
			a.score = [2]int{d.Duel.ScoreUs, d.Duel.ScoreThem}
		default:
			if a.mode == ModeShowdown && a.showdown != nil {
				a.showdown = nil
				a.round++
				a.goTo(scrPeople)
				a.env.Send(Up{T: "people"})
			}
			a.notice = strings.ToUpper(d.Note)
		}
	case "go":
		if a.showdown != nil && d.ID == a.showdown.ID && a.ph == phWaitOpp {
			a.wait(d.Delay)
		}
	case "outcome":
		if a.showdown == nil || d.ID != a.showdown.ID {
			return
		}
		a.mine, a.theirs, a.won, a.tie, a.myFoul, a.theirFoul = d.Mine, d.Theirs, d.Won, d.Draw, d.MyFoul, d.TheirFoul
		a.ph = phResult
		if a.myFoul {
			a.ph = phFoul
		} else {
			a.fanfare()
		}
	}
}

// ── Showdown lobby ─────────────────────────────────────────────────────────

type lobbyLine struct {
	label, role string
	enter       func()
	decline     func()
}

func (a *App) openPeople() {
	a.goTo(scrPeople)
	a.asked = true
	a.env.Send(Up{T: "people"})
}

func (a *App) lobby() []lobbyLine {
	var out []lobbyLine
	for _, in := range a.invites {
		in := in
		if in.Mine {
			out = append(out, lobbyLine{label: "… WAITING FOR " + strings.ToUpper(in.Name) + " · ENTER CANCELS", role: "dim",
				enter: func() { a.env.Send(Up{T: "answer", ID: in.ID}) }})
			continue
		}
		out = append(out, lobbyLine{label: "★ " + strings.ToUpper(in.Name) + " CHALLENGES YOU · ENTER ACCEPTS · D DECLINES", role: "yellow",
			enter:   func() { a.env.Send(Up{T: "answer", ID: in.ID, Accept: true}) },
			decline: func() { a.env.Send(Up{T: "answer", ID: in.ID}) }})
	}
	for _, p := range a.people {
		p := p
		best := "-"
		if p.Best > 0 {
			best = strconv.Itoa(p.Best) + " MS"
		}
		dot := "○ "
		if p.Online {
			dot = "● "
		}
		out = append(out, lobbyLine{label: dot + padRight(strings.ToUpper(p.Name), 20) + " BEST " + best, role: "fg",
			enter: func() {
				a.env.Send(Up{T: "challenge", To: p.ID})
				a.notice = "CHALLENGE SENT TO " + strings.ToUpper(p.Name)
			}})
	}
	return out
}

func (a *App) peopleKey(key string) {
	lines := a.lobby()
	switch key {
	case "up", "k":
		a.cursor = max(0, a.cursor-1)
		a.sound(sndBlip)
	case "down", "j":
		a.cursor = min(max(0, len(lines)-1), a.cursor+1)
		a.sound(sndBlip)
	case "enter", " ":
		if a.cursor < len(lines) {
			a.sound(sndSelect)
			lines[a.cursor].enter()
		}
	case "d":
		if a.cursor < len(lines) && lines[a.cursor].decline != nil {
			lines[a.cursor].decline()
		}
	case "esc", "q":
		a.back()
	}
}

// ── The general store ──────────────────────────────────────────────────────

func (a *App) storeItem(row int) (arcade.Unlockable, int) {
	if row == 0 {
		i := a.idx[0] % len(Hats)
		return Hats[i].Unlockable, i
	}
	i := a.idx[1] % len(Scenes)
	return Scenes[i].Unlockable, i
}

func (a *App) storeKey(key string) {
	n := []int{len(Hats), len(Scenes)}[a.row]
	a.notice = ""
	switch key {
	case "up", "k", "down", "j":
		a.row ^= 1
		a.sound(sndBlip)
	case "left", "h":
		a.idx[a.row] = (a.idx[a.row] + n - 1) % n
		a.sound(sndBlip)
	case "right", "l":
		a.idx[a.row] = (a.idx[a.row] + 1) % n
		a.sound(sndBlip)
	case "enter", " ":
		u, _ := a.storeItem(a.row)
		switch {
		case a.owns(u.ID):
			if a.me.Equipped == nil {
				a.me.Equipped = map[string]string{}
			}
			a.me.Equipped[u.Kind] = u.ID // shown now; the server keeps it
			a.env.Send(Up{T: "equip", ID: u.ID})
			a.sound(sndSelect)
			a.notice = u.Name + " IT IS."
		case a.me.Bounties > 0:
			a.env.Send(Up{T: "deal"})
			a.sound(sndSelect)
			a.goTo(scrDraft)
		default:
			a.notice = "EARN BOUNTIES TO UNLOCK MORE."
			a.sound(sndBack)
		}
	case "esc", "q":
		a.back()
	}
}

func (a *App) draftKey(key string) {
	if !a.reveal.IsZero() {
		return
	}
	offer := a.me.Offer
	switch key {
	case "left", "h":
		if len(offer) > 0 {
			a.sel = (a.sel + len(offer) - 1) % len(offer)
			a.sound(sndBlip)
		}
	case "right", "l":
		if len(offer) > 0 {
			a.sel = (a.sel + 1) % len(offer)
			a.sound(sndBlip)
		}
	case "enter", " ":
		if a.sel < len(offer) {
			a.picked = offer[a.sel]
			a.env.Send(Up{T: "pick", ID: a.picked})
			a.reveal = a.Now()
			a.sound(sndRecord)
		}
	case "esc", "q":
		a.sound(sndBack)
		a.goTo(scrStore)
	}
}

// endReveal goes back to the store, on what was just picked.
func (a *App) endReveal() {
	for i, h := range Hats {
		if h.ID == a.picked {
			a.row, a.idx[0] = 0, i
		}
	}
	for i, s := range Scenes {
		if s.ID == a.picked {
			a.row, a.idx[1] = 1, i
		}
	}
	a.reveal, a.picked, a.sel = time.Time{}, "", 0
	a.goTo(scrStore)
}

func (a *App) optionsKey(key string) {
	switch key {
	case "up", "k", "down", "j":
		a.cursor ^= 1
		a.sound(sndBlip)
	case "left", "h", "right", "l", "enter", " ":
		if a.cursor == 0 {
			a.me.Muted = !a.me.Muted
		} else {
			order := []string{"", "calm", "off"}
			i := max(0, slices.Index(order, a.me.Effects))
			if key == "left" || key == "h" {
				i += 2
			} else {
				i++
			}
			a.me.Effects = order[i%3]
		}
		a.env.Send(Up{T: "options", Muted: a.me.Muted, Effects: a.me.Effects})
		a.sound(sndBlip)
	case "esc", "q":
		a.back()
	}
}

// ── Small panes ────────────────────────────────────────────────────────────

func (a *App) big() bool { return a.W >= 64 && a.H >= 24 }

// smallKey plays practice rounds in a pane too small for the arcade.
func (a *App) smallKey(key string) {
	if key != " " && key != "enter" {
		return
	}
	if a.mode != ModePractice {
		a.startDuel(ModePractice)
	}
	a.duelKey(" ")
}

// OnTitle reports whether the title screen is showing (where Esc leaves).
func (a *App) OnTitle() bool { return a.scr == scrTitle }

// ClaimsEsc reports whether Esc is the app's: everywhere but the title,
// where it's Concord's, to leave the pane.
func (a *App) ClaimsEsc() bool { return a.scr != scrTitle && a.big() }
