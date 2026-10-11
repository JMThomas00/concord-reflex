package game

import (
	"fmt"
	"strings"

	"github.com/JMThomas00/Concord/sdk/arcade"
)

// scr is an 80 x 24 design frame centred in the pane.
type scr struct {
	c      *arcade.Canvas
	ox, oy int
	w      int
}

func (s scr) text(x, y int, str, fg, bg string, bold bool) int {
	return s.c.Text(s.ox+x, s.oy+y, str, fg, bg, bold) - s.ox
}
func (s scr) center(y int, str, fg, bg string, bold bool) {
	s.c.CenterIn(s.ox, s.w, s.oy+y, str, fg, bg, bold)
}
func (s scr) centerIn(x, w, y int, str, fg string, bold bool) {
	s.c.CenterIn(s.ox+x, w, s.oy+y, str, fg, "", bold)
}
func (s scr) box(x, y, w, h int, role, title, titleRole string) {
	s.c.Box(s.ox+x, s.oy+y, w, h, role, title, titleRole)
}
func (s scr) logo(text string, row int, shade []string) {
	s.c.Logo(text, s.ox+(s.w-arcade.LogoWidth(text, 1))/2, 2*(s.oy+row), 1, shade)
}
func (s scr) keys(keys ...arcade.Key) { s.c.Keys(s.ox+1, s.oy+23, keys...) }

// callout is a word in the logo font on a cleared backing (pixels under it
// are wiped, so nothing shows through the letters).
func (s scr) callout(text string, row int, shade []string) {
	w := arcade.LogoWidth(text, 1)
	x := (s.w - w) / 2
	for j := row - 1; j < row+arcade.LogoHeight(1)+1; j++ {
		s.c.Fill(s.ox+x-2, s.oy+j, w+4, " ", "fg", "bg")
	}
	s.c.Logo(text, s.ox+x, 2*(s.oy+row), 1, shade)
}

func padRight(s string, n int) string {
	if w := arcade.TextWidth(s); w < n {
		return s + strings.Repeat(" ", n-w)
	}
	return s
}

func cut(s string, n int) string {
	if r := []rune(s); len(r) > n && n > 1 {
		return string(r[:n-1]) + "…"
	}
	return s
}

var (
	shadeGold = []string{"yellow", "yellow", "yellow", "orange", "orange", "orange", "orange"}
	shadeRed  = []string{"red", "red", "red", "orange", "orange", "yellow", "yellow"}
	shadeGrey = []string{"comment", "comment", "comment", "dim", "dim", "dim", "dim"}
	shadeFoul = []string{"red", "red", "red", "redD", "redD", "redD", "redD"}
)

// View draws the app.
func (a *App) View() string {
	w, h := a.W, a.H
	if w <= 0 {
		w = 80
	}
	if h <= 0 {
		h = 24
	}
	if !a.big() {
		return a.smallView(w, h)
	}
	c := arcade.New(w, h, a.Pal)
	fw := min(w, 80)
	s := scr{c: c, ox: (w - fw) / 2, oy: (h - 24) / 2, w: fw}
	switch a.scr {
	case scrTitle:
		a.drawTitle(s)
	case scrMenu:
		a.drawMenu(s)
	case scrDuel:
		a.drawDuel(s)
	case scrPeople:
		a.drawPeople(s)
	case scrStore:
		a.drawStore(s)
	case scrDraft:
		a.drawDraft(s)
	case scrHOF:
		a.drawHOF(s)
	case scrHowTo:
		a.drawHowTo(s)
	case scrOptions:
		a.drawOptions(s)
	}
	return c.String()
}

func (a *App) topBar(s scr, where string) {
	x := s.text(1, 0, "1UP ", "pink", "", true)
	best := "NO TIME YET"
	if a.me.Best > 0 {
		best = fmt.Sprintf("BEST %d MS", a.me.Best)
	}
	s.text(x, 0, best, "fg", "", false)
	if len(a.top) > 0 && s.w >= 80 {
		x = s.text(31, 0, "HI-SCORE ", "pink", "", true)
		s.text(x, 0, fmt.Sprintf("%d MS %s", a.top[0].MS, cut(strings.ToUpper(a.top[0].Name), 12)), "fg", "", false)
	}
	s.text(s.w-1-arcade.TextWidth(where), 0, where, "dim", "", false)
}

func (a *App) drawTitle(s scr) {
	scenes := []string{"scene:noon", "scene:sunset", "scene:night", "scene:canyon"}
	sc := scenes[a.frame/150%len(scenes)]
	sky := SceneByID(sc).Sky
	drawScene(s.c, sc, s.ox, s.oy+1, s.w, 16, a.frame, false)
	a.topBar(s, "#reflex")
	cycle := a.frame % 60
	s.logo("REFLEX", 2, nil)
	s.center(6, "·  H O W   F A S T   A R E   Y O U ?  ·", "fg", sky, false)
	gunslinger(s.c, s.ox+12, s.oy+7, Hats[0].ID, cycle > 44, false, "cyan", false)
	gunslinger(s.c, s.ox+s.w-24, s.oy+7, Hats[1].ID, cycle > 40, true, "red", false)
	tumbleweed(s.c, s.ox, s.w, s.oy+15, a.frame)
	if cycle > 40 {
		word := "DRAW!"
		if cycle >= 50 {
			word = "BANG!"
		}
		s.center(10, word, "red", sky, true)
	}
	if a.blink() {
		s.center(19, "▶  PRESS ENTER  ◀", "yellow", "", true)
	}
	if len(a.top) > 0 {
		s.center(20, fmt.Sprintf("FASTEST HERE: %s %d MS", strings.ToUpper(a.top[0].Name), a.top[0].MS), "cyan", "", false)
	}
	s.text(1, 23, "© CONCORD ARCADE", "dim", "", false)
	s.c.Keys(s.ox+s.w-26, s.oy+23, arcade.Key{Key: "Enter", Does: "start"}, arcade.Key{Key: "Esc", Does: "leave"})
}

func (a *App) drawMenu(s scr) {
	a.topBar(s, "#reflex")
	s.logo("REFLEX", 2, nil)
	items := a.menuItems()
	a.cursor = min(a.cursor, len(items)-1)
	for i, it := range items {
		y, sel := 8+i, i == a.cursor
		if sel {
			s.text(3, y, "▶", "purple", "", true)
		}
		fg, bg := "fg", ""
		if sel {
			fg, bg = "yellow", "line"
		}
		s.text(5, y, " "+padRight(it.label, 15), fg, bg, sel)
		d, role := it.desc()
		if sel && role == "dim" {
			role = "fg"
		}
		s.text(22, y, cut(d, 40), role, "", role == "yellow")
	}
	if s.w >= 80 {
		sc := SceneByID(a.scene())
		s.c.Shade(s.ox+64, s.oy+7, 14, 7, sc.Sky)
		s.c.Shade(s.ox+64, s.oy+14, 14, 2, sc.Ground)
		gunslinger(s.c, s.ox+65, s.oy+8, a.hat(), false, false, "cyan", false)
		if a.me.Best > 0 {
			r := RankOf(a.me.Best)
			s.centerIn(62, 18, 17, r.Title, r.Role, true)
		}
	}
	if a.notice != "" {
		s.center(19, a.notice, "green", "", true)
	}
	s.keys(arcade.Key{Key: "↑↓", Does: "choose"}, arcade.Key{Key: "←→", Does: "level"}, arcade.Key{Key: "Enter", Does: "select"}, arcade.Key{Key: "Esc", Does: "title"})
}

func (a *App) opponentName() string {
	switch {
	case a.mode == ModeShowdown && a.showdown != nil:
		return strings.ToUpper(a.showdown.Opponent)
	case a.mode == ModeShowdown:
		return "THEM"
	}
	return "THE OUTLAW"
}

func (a *App) drawDuel(s scr) {
	sc := a.scene()
	drawScene(s.c, sc, s.ox, s.oy+1, s.w, 16, a.frame, false)
	where := "QUICK DRAW · " + Levels[a.level].Name
	switch a.mode {
	case ModePractice:
		where = "PRACTICE"
	case ModeShowdown:
		where = "SHOWDOWN · VS " + a.opponentName()
	}
	a.topBar(s, where)
	resultIn := a.ph == phResult || a.ph == phFoul
	mineDrawn := a.mine > 0 && a.ph != phFoul
	theirsDrawn := a.mode != ModePractice && (resultIn || (a.ph == phDraw && a.mode == ModeDuel &&
		int(a.Now().Sub(a.drawAt).Milliseconds()) > a.theirsAt))
	gunslinger(s.c, s.ox+12, s.oy+7, a.hat(), mineDrawn, false, "cyan", false)
	ground := SceneByID(sc).Ground
	s.c.Text(s.ox+16, s.oy+15, "YOU", "fg", ground, true)
	if a.mode == ModePractice {
		target(s.c, s.ox+s.w-22, s.oy+8)
	} else {
		them := Hats[1].ID // the outlaw's black hat
		if a.mode == ModeShowdown {
			them = Hats[2].ID
		}
		gunslinger(s.c, s.ox+s.w-24, s.oy+7, them, theirsDrawn, true, "red", false)
		s.c.CenterIn(s.ox+s.w-28, 20, s.oy+15, a.opponentName(), "fg", ground, true)
	}
	switch a.ph {
	case phReady:
		s.callout("READY?", 2, nil)
		note := "SPACE WHEN YOU'RE READY · THEN WAIT FOR THE WORD"
		if a.notice != "" {
			note = a.notice
		}
		s.center(17, note, "fg", "line", false)
	case phWaitOpp:
		s.callout("READY", 2, shadeGrey)
		s.center(17, "WAITING FOR "+a.opponentName(), "dim", "line", false)
	case phWait:
		s.callout("WAIT...", 2, shadeGrey)
		s.center(17, "HANDS STILL", "dim", "line", false)
	case phDraw:
		s.callout("DRAW!", 2, shadeRed)
	case phShot:
		s.callout("BANG!", 2, shadeGold)
		s.center(17, "WAITING FOR "+a.opponentName(), "dim", "line", false)
	case phFoul:
		s.callout("TOO SOON!", 2, shadeFoul)
		msg := "A FOUL · SPACE TO TRY AGAIN"
		if a.mode != ModePractice {
			msg = "A FOUL: THE DUEL GOES TO " + a.opponentName() + " · SPACE FOR ANOTHER"
		}
		s.center(17, msg, "red", "line", true)
	case phResult:
		word, shade := "BANG!", shadeGold
		switch {
		case a.mode == ModePractice:
		case a.tie:
			word, shade = "A DRAW", shadeGrey
		case a.won:
			word = "YOU WIN!"
		case a.mine == 0:
			word, shade = "TOO SLOW!", shadeGrey
		default:
			word, shade = "TOO SLOW!", shadeGrey
		}
		s.callout(word, 2, shade)
		if a.mine > 0 {
			r := RankOf(a.mine)
			line := fmt.Sprintf("%d MS · %s", a.mine, r.Title)
			if a.record {
				line += " · A NEW BEST!"
			}
			if a.theirFoul {
				line += " · THEY DREW TOO SOON"
			}
			s.center(17, line, r.Role, "line", true)
		} else if a.theirFoul {
			s.center(17, a.opponentName()+" DREW TOO SOON", "green", "line", true)
		}
	}
	// the readouts
	s.text(4, 19, "YOU", "pink", "", true)
	mine := -1
	if a.mine > 0 {
		mine = a.mine
	}
	on := "pink"
	if a.ph == phResult && (a.won || a.mode == ModePractice) {
		on = "green"
	}
	a.readout(s, mine, 9, on)
	s.text(25, 20, "MS", "dim", "", false)
	if a.mode != ModePractice {
		name := "OUTLAW"
		if a.mode == ModeShowdown {
			name = cut(a.opponentName(), 9)
		}
		s.text(48, 19, name, "pink", "", true)
		theirs := -1
		if resultIn && a.theirs > 0 {
			theirs = a.theirs
		}
		a.readout(s, theirs, 57, "cyan")
		s.text(73, 20, "MS", "dim", "", false)
		s.text(35, 18, "SCORE", "pink", "", true)
		s.c.Digit(min(9, a.score[0]), s.ox+34, 2*(s.oy+19), "pink", 1)
		s.text(39, 20, "-", "dim", "", false)
		s.c.Digit(min(9, a.score[1]), s.ox+41, 2*(s.oy+19), "cyan", 1)
	}
	if resultIn || a.ph == phReady {
		s.keys(arcade.Key{Key: "Space", Does: "again"}, arcade.Key{Key: "Esc", Does: "menu"})
	} else {
		s.keys(arcade.Key{Key: "Space", Does: "draw!"}, arcade.Key{Key: "Esc", Does: "menu"})
	}
}

// readout is three seven-segment digits (unlit for none).
func (a *App) readout(s scr, ms, x int, on string) {
	for k := 0; k < 3; k++ {
		d := -1
		if ms >= 0 {
			d = []int{ms / 100 % 10, ms / 10 % 10, ms % 10}[k]
			if ms > 999 {
				d = 9
			}
		}
		s.c.Digit(d, s.ox+x+k*5, 2*(s.oy+19), on, 1)
	}
}

func (a *App) drawPeople(s scr) {
	a.topBar(s, "SHOWDOWN")
	s.box(1, 1, s.w-2, 21, "comment", "SHOWDOWN · WHO'S UP FOR IT?", "purple")
	lines := a.lobby()
	switch {
	case !a.asked && len(lines) == 0:
	case len(lines) == 0 && a.people == nil:
		s.text(5, 4, "LOOKING…", "dim", "", false)
	case len(lines) == 0:
		s.text(5, 4, "NOBODY ELSE CAN SEE THIS CHANNEL YET.", "dim", "", false)
	}
	a.cursor = min(a.cursor, max(0, len(lines)-1))
	top := max(0, a.cursor-14)
	for i := top; i < len(lines) && i < top+15; i++ {
		y, sel := 3+i-top, i == a.cursor
		if sel {
			s.text(3, y, "▶", "purple", "", true)
			s.text(5, y, " "+cut(lines[i].label, s.w-10)+" ", "yellow", "line", true)
		} else {
			s.text(6, y, cut(lines[i].label, s.w-10), lines[i].role, "", false)
		}
	}
	if a.notice != "" {
		s.center(20, a.notice, "green", "", true)
	} else {
		s.center(20, "EACH OF YOU IS TIMED ON YOUR OWN COMPUTER: LAG CAN'T COST YOU", "dim", "", false)
	}
	s.keys(arcade.Key{Key: "↑↓", Does: "choose"}, arcade.Key{Key: "Enter", Does: "challenge / accept"}, arcade.Key{Key: "D", Does: "decline"}, arcade.Key{Key: "Esc", Does: "back"})
}

func (a *App) drawStore(s scr) {
	a.topBar(s, "GENERAL STORE")
	s.box(1, 1, s.w-2, 21, "comment", "GENERAL STORE", "purple")
	hatU, hi := a.storeItem(0)
	sceneU, si := a.storeItem(1)
	ownHat, ownScene := a.owns(hatU.ID), a.owns(sceneU.ID)
	// a window onto the street: the backdrop being looked at (or the one in
	// use), and the hat on you
	showScene, showHat := a.scene(), a.hat()
	if a.row == 1 {
		showScene = Scenes[si].ID
	} else {
		showHat = Hats[hi].ID
	}
	drawScene(s.c, showScene, s.ox+3, s.oy+3, 38, 15, a.frame, a.row == 1 && !ownScene)
	gunslinger(s.c, s.ox+16, s.oy+7, showHat, false, false, "cyan", a.row == 0 && !ownHat)
	if (a.row == 0 && !ownHat) || (a.row == 1 && !ownScene) {
		s.text(18, 19, " LOCKED ", "bg", "comment", true)
	}
	for row, u := range []arcade.Unlockable{hatU, sceneU} {
		y, sel := 3+row*3, row == a.row
		if sel {
			s.text(45, y, "▶", "purple", "", true)
		}
		labelRole := "pink"
		if sel {
			labelRole = "yellow"
		}
		s.text(47, y, []string{"HAT", "BACKDROP"}[row], labelRole, "", true)
		if (row == 0 && u.ID == a.hat()) || (row == 1 && u.ID == a.scene()) {
			s.text(s.w-12, y, "✓ IN USE", "green", "", false)
		}
		name, role := "? ? ?", "dim"
		if a.owns(u.ID) {
			name, role = u.Name, "fg"
		}
		end := s.text(47, y+1, "◂ "+cut(name, 18)+" ▸", role, "", true)
		if u.Tier != arcade.Starter {
			s.text(end+1, y+1, u.Tier.Stars(), u.Tier.Role(), "", true)
		}
	}
	focus := []arcade.Unlockable{hatU, sceneU}[a.row]
	s.text(47, 9, focus.Tier.Name(), focus.Tier.Role(), "", true)
	if a.owns(focus.ID) {
		s.text(47, 10, cut("\""+focus.Blurb+"\"", s.w-50), "dim", "", false)
	} else {
		s.text(47, 10, "LOCKED: EARN BOUNTIES", "dim", "", false)
	}
	all := len(Unlockables())
	have := a.collected()
	s.text(47, 14, "COLLECTED", "pink", "", true)
	for k := 0; k < all && 47+k < s.w-3; k++ {
		ch, role := "▯", "ghost"
		if k < have {
			ch, role = "▮", "purple"
		}
		s.text(47+k, 15, ch, role, "", false)
	}
	s.text(47, 16, fmt.Sprintf("%d OF %d", have, all), "fg", "", false)
	s.text(47, 18, "BOUNTIES", "pink", "", true)
	passRole := "dim"
	if a.me.Bounties > 0 {
		passRole = "yellow"
	}
	end := s.text(47, 19, fmt.Sprintf("★ %d", a.me.Bounties), passRole, "", true)
	hint := "NONE YET"
	if a.me.Bounties > 0 {
		hint = "ENTER ON A LOCKED ONE"
	}
	s.text(end+1, 19, hint, "dim", "", false)
	if a.notice != "" {
		s.center(21, a.notice, "green", "", true)
	}
	s.keys(arcade.Key{Key: "↑↓", Does: "row"}, arcade.Key{Key: "←→", Does: "change"}, arcade.Key{Key: "Enter", Does: "wear / trade"}, arcade.Key{Key: "Esc", Does: "back"})
}

func (a *App) drawDraft(s scr) {
	a.topBar(s, "BOUNTY")
	revealing := !a.reveal.IsZero()
	title, shade := "PICK ONE", arcade.LogoShade
	if revealing {
		title, shade = "UNLOCKED!", shadeGold
	}
	s.logo(title, 1, shade)
	offer := a.me.Offer
	if revealing {
		offer = []string{a.picked}
	}
	if len(offer) == 0 {
		s.center(12, "DEALING…", "dim", "", false)
		s.keys(arcade.Key{Key: "Esc", Does: "back"})
		return
	}
	a.sel = min(a.sel, len(offer)-1)
	cardW := min(26, (s.w-2)/len(offer))
	left := (s.w - cardW*len(offer)) / 2
	for i, id := range offer {
		var u arcade.Unlockable
		for _, it := range Unlockables() {
			if it.ID == id {
				u = it
			}
		}
		x := left + i*cardW
		chosen := i == a.sel || revealing
		role := "comment"
		if chosen {
			role = "purple"
			if revealing {
				role = "yellow"
			}
		}
		s.box(x, 6, cardW-1, 14, role, "", "")
		if chosen && !revealing {
			s.text(x+cardW/2-1, 5, "▼", "purple", "", true)
		}
		s.centerIn(x, cardW-1, 7, strings.TrimSpace(u.Tier.Stars()+" "+u.Tier.Name()), u.Tier.Role(), true)
		if u.Kind == KindHat {
			gunslinger(s.c, s.ox+x+(cardW-13)/2, s.oy+8, u.ID, false, false, "cyan", false)
		} else {
			drawScene(s.c, u.ID, s.ox+x+2, s.oy+8, cardW-5, 6, a.frame, false)
		}
		nameRole := "dim"
		if chosen {
			nameRole = "fg"
		}
		s.centerIn(x, cardW-1, 15, cut(u.Name, cardW-3), nameRole, true)
		s.centerIn(x, cardW-1, 16, map[string]string{KindHat: "HAT", KindScene: "BACKDROP"}[u.Kind], "pink", false)
		s.centerIn(x, cardW-1, 18, cut(u.Blurb, cardW-3), "dim", false)
	}
	if revealing {
		s.center(21, "IT'S YOURS.", "green", "", true)
		return
	}
	s.center(21, fmt.Sprintf("SPENDS 1 BOUNTY · YOU HAVE %d · THE OFFER STAYS UNTIL YOU PICK", a.me.Bounties), "dim", "", false)
	s.keys(arcade.Key{Key: "←→", Does: "choose"}, arcade.Key{Key: "Enter", Does: "trade"}, arcade.Key{Key: "Esc", Does: "not now"})
}

func (a *App) drawHOF(s scr) {
	a.topBar(s, "HALL OF FAME")
	s.logo("HALL OF FAME", 2, nil)
	x := (s.w - 58) / 2
	s.text(x, 8, "RANK   NAME            FASTEST    TITLE        DUELS WON", "pink", "", true)
	ranks := []string{"1ST", "2ND", "3RD", "4TH", "5TH", "6TH", "7TH", "8TH"}
	roles := []string{"yellow", "orange", "red", "pink", "cyan", "green", "fg", "fg"}
	for i, e := range a.top {
		if i >= len(ranks) {
			break
		}
		line := fmt.Sprintf("%-6s %-14s  %5d MS    %-12s %5d", ranks[i], cut(strings.ToUpper(e.Name), 14), e.MS, RankOf(e.MS).Title, e.Wins)
		fg, bg := roles[i], ""
		if e.Me {
			fg, bg = "fg", "line"
		}
		s.text(x, 10+i, line, fg, bg, true)
	}
	if len(a.top) == 0 {
		s.center(12, "NO TIMES YET. BE THE FIRST!", "dim", "", false)
	}
	s.center(19, "UNDER 170 MS: LEGEND · 210: SHERIFF · 250: DEPUTY · 300: RANGER", "dim", "", false)
	s.keys(arcade.Key{Key: "Esc", Does: "back"})
}

func (a *App) drawHowTo(s scr) {
	a.topBar(s, "HOW TO PLAY")
	s.box(1, 1, s.w-2, 21, "comment", "HOW TO PLAY", "purple")
	lines := []string{
		"Two gunslingers, one word. Press Space when you're ready, keep your",
		"hands still through WAIT..., and press Space the moment it says DRAW!",
		"Your time is measured on your own computer, to the millisecond, so",
		"the network never slows you down.",
		"",
		"Press before DRAW! (or within 0.1 s of it: that's a guess) and it's a",
		"foul: the duel goes to the other side.",
		"",
		"QUICK DRAW   beat the outlaw (easy ~430 ms, normal ~300, hard ~215).",
		"SHOWDOWN     duel someone in the channel: you both see the word together.",
		"PRACTICE     just you and the clock.",
		"",
		"Win duels and set records to earn Bounties for the General Store.",
	}
	for i, l := range lines {
		s.text(4, 3+i, cut(l, s.w-8), "fg", "", false)
	}
	s.text(4, 17, "KEYS", "pink", "", true)
	s.c.Keys(s.ox+4, s.oy+18, arcade.Key{Key: "Space", Does: "ready, then draw"})
	s.c.Keys(s.ox+4, s.oy+19, arcade.Key{Key: "Esc  ", Does: "back one screen (on the title: leave)"})
	s.keys(arcade.Key{Key: "Esc", Does: "back"})
}

func (a *App) drawOptions(s scr) {
	a.topBar(s, "OPTIONS")
	s.logo("OPTIONS", 2, nil)
	opts := [][3]string{
		{"SOUND", onOff(!a.me.Muted), "THE BELL, THE WORD AND THE BANG, FOR YOU"},
		{"EFFECTS", effectsName(a.me.Effects), "FULL: ANIMATION · CALM: STILL · OFF"},
	}
	for i, o := range opts {
		y, sel := 9+i*3, i == a.cursor
		if sel {
			s.text(20, y, "▶", "purple", "", true)
		}
		role := "pink"
		if sel {
			role = "yellow"
		}
		s.text(22, y, o[0], role, "", true)
		s.text(34, y, "◂ "+o[1]+" ▸", "fg", "", true)
		s.text(22, y+1, o[2], "dim", "", false)
	}
	s.center(18, "THESE ARE YOURS: EVERYONE ELSE KEEPS THEIR OWN.", "dim", "", false)
	s.keys(arcade.Key{Key: "↑↓", Does: "choose"}, arcade.Key{Key: "←→", Does: "change"}, arcade.Key{Key: "Esc", Does: "back"})
}

// smallView is practice in a pane too small for the arcade.
func (a *App) smallView(w, h int) string {
	c := arcade.New(w, h, a.Pal)
	lines := []struct{ text, role string }{{"REFLEX", "purple"}, {"", ""}}
	switch {
	case a.scr != scrDuel || a.ph == phReady:
		lines = append(lines, struct{ text, role string }{"Space: wait for NOW, then press it again.", "fg"})
	case a.ph == phWait:
		lines = append(lines, struct{ text, role string }{"Wait for it…", "red"})
	case a.ph == phDraw:
		lines = append(lines, struct{ text, role string }{"███  NOW!  ███", "green"})
	case a.ph == phFoul:
		lines = append(lines, struct{ text, role string }{"Too soon! Space to try again.", "red"})
	case a.ph == phResult && a.mine > 0:
		lines = append(lines, struct{ text, role string }{fmt.Sprintf("%d ms · %s", a.mine, RankOf(a.mine).Title), "green"})
	}
	if a.me.Best > 0 {
		lines = append(lines, struct{ text, role string }{"", ""}, struct{ text, role string }{fmt.Sprintf("Best: %d ms", a.me.Best), "dim"})
	}
	lines = append(lines, struct{ text, role string }{"", ""}, struct{ text, role string }{"(a pane of 64x24 or more shows the arcade)", "dim"})
	top := max(0, (h-len(lines))/2)
	for i, l := range lines {
		c.Center(top+i, l.text, l.role, "", i == 0)
	}
	return c.String()
}
