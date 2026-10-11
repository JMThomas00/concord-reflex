package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	mrand "math/rand/v2"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/JMThomas00/Concord/sdk/arcade"
	"github.com/JMThomas00/Concord/sdk/plugin"
	"github.com/JMThomas00/Concord/sdk/wire"
	"github.com/JMThomas00/concord-reflex/game"
)

// The server half. The duels run in each player's client code, so times are
// exact; this half keeps everyone's record and gear, awards achievements
// and Bounties, ranks the Hall of Fame, and runs showdowns: it tells both
// players' clients when the word comes and compares their two times.
//
// Client code runs on players' own computers, so what it reports can't be
// fully trusted: impossible times are ignored, and a showdown is judged
// here from both players' shots.

// record is one player's, kept across games and restarts.
type record struct {
	Name       string               `json:"name"`
	Best       int                  `json:"best,omitempty"`
	Duels      int                  `json:"duels"`
	Wins       int                  `json:"wins"`
	Streak     int                  `json:"streak"`
	BestStreak int                  `json:"best_streak"`
	Rewards    arcade.Rewards       `json:"rewards"`
	Equipped   map[string]string    `json:"equipped,omitempty"`
	Muted      bool                 `json:"muted,omitempty"`
	Effects    string               `json:"effects,omitempty"`
	Unlocked   map[string]time.Time `json:"unlocked,omitempty"`

	earned    int  // Bounties from the last result
	newRecord bool // the last result was their fastest
}

// Achievements, as declared in plugin.toml.
const (
	achFirstDraw  = "first_draw"
	achFirstWin   = "first_win"
	achDeputy     = "deputy"
	achSheriff    = "sheriff"
	achLegend     = "legend"
	achHardOutlaw = "hard_outlaw"
	achStreak5    = "streak_5"
	achShowdown   = "showdown_win"
	achWins50     = "wins_50"
)

// The slowest time worth keeping.
const slowest = 5000

type viewer struct{ channel, id uuid.UUID }

// invite is a challenge waiting for an answer.
type invite struct {
	id       string
	channel  uuid.UUID
	from, to uuid.UUID
}

// showdown is a duel between two players in a channel.
type showdown struct {
	id      string
	channel uuid.UUID
	players [2]uuid.UUID
	score   [2]int
	ready   [2]bool
	shot    [2]*game.Up // this round's times, as they come in
	round   int         // counts rounds, so a late timeout can't end the next one
}

type server struct {
	conn    *plugin.Conn
	dir     string
	records map[uuid.UUID]*record
	viewers map[viewer]bool
	invites map[string]*invite
	duels   map[string]*showdown
	items   []arcade.Unlockable
	rnd     *mrand.Rand

	// post runs work on the event loop (Conn.Post); out delivers a message
	// to a player's client code (Conn.SendToClient). Standalone, both are
	// local.
	post func(func())
	out  func(channel, to uuid.UUID, d game.Down)
}

func newServer() *server {
	return &server{
		records: map[uuid.UUID]*record{},
		viewers: map[viewer]bool{},
		invites: map[string]*invite{},
		duels:   map[string]*showdown{},
		items:   game.Unlockables(),
		rnd:     mrand.New(mrand.NewPCG(uint64(time.Now().UnixNano()), 7)),
	}
}

func (s *server) handler() plugin.Handler {
	return plugin.Handler{
		OnReady: func(c *plugin.Conn, _ *wire.User) {
			first := s.conn == nil
			s.conn, s.dir = c, c.DataDir()
			s.post = c.Post
			s.out = func(channel, to uuid.UUID, d game.Down) { _ = c.SendToClient(channel, to, d) }
			if first {
				s.load()
			}
		},
		OnEnter: func(c *plugin.Conn, e wire.PluginPaneEnterPayload) {
			s.viewers[viewer{e.ChannelID, e.ViewerID}] = true
			if r := s.record(e.ViewerID); e.ViewerDisplayName != "" {
				r.Name = e.ViewerDisplayName
			}
			// Players who turned plugin code off see this instead.
			_ = c.Frame(e.ChannelID, e.ViewerID, "REFLEX: a quick-draw duel at high noon.\n\n"+
				"It runs on your own computer, so the time is exact.\n"+
				"Allow its code to play (or turn plugin code on in\nSettings > Display > Plugin Code).")
		},
		OnLeave: func(c *plugin.Conn, e wire.PluginPaneLeavePayload) {
			delete(s.viewers, viewer{e.ChannelID, e.ViewerID})
			for _, d := range s.duels {
				if d.channel == e.ChannelID && slices.Contains(d.players[:], e.ViewerID) {
					s.endDuel(d, "your opponent left")
				}
			}
		},
		OnClientMessage: func(c *plugin.Conn, from uuid.UUID, m wire.PluginClientMessagePayload) {
			var up game.Up
			if json.Unmarshal(m.Data, &up) != nil {
				return
			}
			if m.ViewerDisplayName != "" {
				s.record(from).Name = m.ViewerDisplayName
			}
			s.message(m.ChannelID, from, up)
		},
	}
}

// ── Records ────────────────────────────────────────────────────────────────

func (s *server) record(id uuid.UUID) *record {
	r := s.records[id]
	if r == nil {
		r = &record{}
		s.records[id] = r
	}
	return r
}

func (s *server) path() string {
	dir := s.dir
	if dir == "" {
		dir = "."
	}
	return filepath.Join(dir, "records.json")
}

func (s *server) load() {
	if data, err := os.ReadFile(s.path()); err == nil {
		if err := json.Unmarshal(data, &s.records); err != nil {
			log.Printf("reflex: ignoring unreadable records: %v", err)
			s.records = map[uuid.UUID]*record{}
		}
	}
}

func (s *server) save() {
	data, _ := json.MarshalIndent(s.records, "", "  ")
	path := s.path()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		log.Printf("reflex: can't save records: %v", err)
		return
	}
	if err := os.WriteFile(path+".tmp", data, 0o600); err == nil {
		_ = os.Rename(path+".tmp", path)
	}
}

func (r *record) unlock(id string, now time.Time) bool {
	if r.Unlocked == nil {
		r.Unlocked = map[string]time.Time{}
	}
	if _, done := r.Unlocked[id]; done {
		return false
	}
	r.Unlocked[id] = now
	return true
}

// counted adds a finished duel or practice draw to a player's record: the
// time, a win or a loss, achievements, and Bounties for each new
// achievement, every third win in a row and every tenth duel.
func (s *server) counted(id uuid.UUID, mode string, ms int, foul, won bool, level int) {
	r := s.record(id)
	now := time.Now().UTC()
	r.earned, r.newRecord = 0, false
	fresh := 0
	unlock := func(a string) {
		if r.unlock(a, now) {
			fresh++
		}
	}
	if !foul && ms >= game.Anticipated && ms < slowest {
		if r.Best == 0 || ms < r.Best {
			r.Best, r.newRecord = ms, true
		}
		unlock(achFirstDraw)
		if ms < 250 {
			unlock(achDeputy)
		}
		if ms < 210 {
			unlock(achSheriff)
		}
		if ms < 170 {
			unlock(achLegend)
		}
	}
	bonus := 0
	if mode != game.ModePractice {
		r.Duels++
		if won {
			r.Wins++
			r.Streak++
			r.BestStreak = max(r.BestStreak, r.Streak)
			unlock(achFirstWin)
			if mode == game.ModeDuel && level == len(game.Levels)-1 {
				unlock(achHardOutlaw)
			}
			if mode == game.ModeShowdown {
				unlock(achShowdown)
			}
			if r.Streak%3 == 0 {
				bonus++
			}
		} else {
			r.Streak = 0
		}
		if r.Streak >= 5 {
			unlock(achStreak5)
		}
		if r.Wins >= 50 {
			unlock(achWins50)
		}
		if r.Duels%10 == 0 {
			bonus++
		}
	}
	r.earned = r.Rewards.Grant(fresh+bonus, s.items)
	if s.conn != nil {
		_ = s.conn.SendRecord(r.wire(id))
	}
	s.save()
}

// wire is the record as Concord takes it. The leaderboard ranks highest
// first, so the fastest draw's number is its time made negative.
func (r *record) wire(id uuid.UUID) wire.PluginRecord {
	best, title := "-", "-"
	if r.Best > 0 {
		best, title = fmt.Sprintf("%d ms", r.Best), game.RankOf(r.Best).Title
	}
	rec := wire.PluginRecord{UserID: id, Stats: []wire.PluginStat{
		{Key: "best", Label: "Fastest draw", Value: best, Num: -float64(r.Best)},
		{Key: "title", Label: "Title", Value: title},
		{Key: "wins", Label: "Duels won", Value: fmt.Sprint(r.Wins), Num: float64(r.Wins)},
		{Key: "duels", Label: "Duels", Value: fmt.Sprint(r.Duels), Num: float64(r.Duels)},
		{Key: "best_streak", Label: "Best run", Value: fmt.Sprint(r.BestStreak), Num: float64(r.BestStreak)},
	}}
	for a, at := range r.Unlocked {
		rec.Unlocked = append(rec.Unlocked, wire.PluginUnlock{ID: a, At: at})
	}
	return rec
}

// profile is a record as its player's client code shows it.
func (s *server) profile(r *record) *game.Profile {
	var owned []string
	for _, it := range s.items {
		if r.Rewards.Owns(it.ID, s.items) {
			owned = append(owned, it.ID)
		}
	}
	return &game.Profile{
		Best: r.Best, Duels: r.Duels, Wins: r.Wins, Streak: r.Streak,
		Bounties: r.Rewards.Passes, Owned: owned, Offer: r.Rewards.Offer, Equipped: r.Equipped,
		Muted: r.Muted, Effects: r.Effects, Earned: r.earned, NewRecord: r.newRecord,
	}
}

// top is the Hall of Fame: the fastest draws.
func (s *server) top(me uuid.UUID) []game.Entry {
	var out []game.Entry
	for id, r := range s.records {
		if r.Best > 0 {
			out = append(out, game.Entry{Name: r.Name, MS: r.Best, Wins: r.Wins, Me: id == me})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].MS != out[j].MS {
			return out[i].MS < out[j].MS
		}
		return out[i].Name < out[j].Name
	})
	return out[:min(len(out), 8)]
}

// ── Talking to clients ─────────────────────────────────────────────────────

func (s *server) send(channel, to uuid.UUID, d game.Down) {
	if s.out != nil && s.viewers[viewer{channel, to}] {
		s.out(channel, to, d)
	}
}

func (s *server) sendState(channel, to uuid.UUID) {
	var invites []game.Invite
	for _, in := range s.invites {
		switch {
		case in.channel != channel:
		case in.to == to:
			invites = append(invites, game.Invite{ID: in.id, Name: s.record(in.from).Name})
		case in.from == to:
			invites = append(invites, game.Invite{ID: in.id, Name: s.record(in.to).Name, Mine: true})
		}
	}
	sort.Slice(invites, func(i, j int) bool { return invites[i].ID < invites[j].ID })
	s.send(channel, to, game.Down{T: "state", Me: s.profile(s.record(to)), Top: s.top(to), Invites: invites})
}

// everyone sends the Hall of Fame (and their own state) to everyone
// watching a channel.
func (s *server) everyone(channel uuid.UUID) {
	for v := range s.viewers {
		if v.channel == channel {
			s.sendState(channel, v.id)
		}
	}
}

func newID() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *server) message(channel, from uuid.UUID, up game.Up) {
	r := s.record(from)
	switch up.T {
	case "hello":
		s.sendState(channel, from)
	case "result":
		if up.Mode != game.ModeDuel && up.Mode != game.ModePractice {
			return // showdowns are judged here, from both shots
		}
		s.counted(from, up.Mode, up.MS, up.Foul, up.Won && !up.Foul && up.MS >= game.Anticipated, up.Level)
		s.everyone(channel)
	case "equip":
		if r.Rewards.Owns(up.ID, s.items) {
			for _, it := range s.items {
				if it.ID == up.ID {
					s.equip(r, it)
				}
			}
			s.save()
		}
		s.sendState(channel, from)
	case "deal":
		had := len(r.Rewards.Offer)
		if len(r.Rewards.Deal(s.items, s.rnd)) != had {
			s.save()
		}
		s.sendState(channel, from)
	case "pick":
		if r.Rewards.Pick(up.ID) {
			for _, it := range s.items {
				if it.ID == up.ID {
					s.equip(r, it)
				}
			}
			s.save()
		}
		s.sendState(channel, from)
	case "options":
		r.Muted, r.Effects = up.Muted, up.Effects
		s.save()
	case "people":
		s.people(channel, from)
	case "challenge":
		s.challenge(channel, from, up.To)
	case "answer":
		s.answer(channel, from, up.ID, up.Accept)
	case "ready":
		s.ready(from, up.ID)
	case "shot":
		s.shot(from, up)
	case "leave":
		if d := s.duels[up.ID]; d != nil && slices.Contains(d.players[:], from) {
			s.endDuel(d, "your opponent walked away")
		}
	}
}

func (s *server) equip(r *record, it arcade.Unlockable) {
	if r.Equipped == nil {
		r.Equipped = map[string]string{}
	}
	r.Equipped[it.Kind] = it.ID
}

// ── Showdowns ──────────────────────────────────────────────────────────────

// people looks up who can see the channel, in the background.
func (s *server) people(channel, from uuid.UUID) {
	if s.conn == nil || s.post == nil {
		return
	}
	conn := s.conn
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		members, err := conn.RequestMembers(ctx, channel)
		if err != nil {
			log.Printf("reflex: members lookup failed: %v", err)
			return
		}
		s.post(func() {
			out := []game.Person{}
			for _, m := range members {
				if m.UserID == from {
					continue
				}
				out = append(out, game.Person{ID: m.UserID.String(), Name: m.DisplayName, Online: m.Online, Best: s.record(m.UserID).Best})
			}
			sort.SliceStable(out, func(i, j int) bool { return out[i].Online && !out[j].Online })
			s.send(channel, from, game.Down{T: "people", People: out})
		})
	}()
}

func (s *server) challenge(channel, from uuid.UUID, to string) {
	target, err := uuid.Parse(to)
	if err != nil || target == from {
		return
	}
	for _, in := range s.invites {
		if in.channel == channel && in.from == from && in.to == target {
			return
		}
	}
	in := &invite{id: newID(), channel: channel, from: from, to: target}
	s.invites[in.id] = in
	if s.conn != nil {
		_ = s.conn.NotifyUser(target, channel, s.record(from).Name+" challenges you to a showdown in Reflex")
	}
	s.sendState(channel, from)
	s.sendState(channel, target)
}

// answer accepts or declines a challenge; its sender answering cancels it.
func (s *server) answer(channel, from uuid.UUID, id string, accept bool) {
	in := s.invites[id]
	if in == nil || (from != in.to && from != in.from) {
		return
	}
	delete(s.invites, id)
	defer func() {
		s.sendState(channel, in.from)
		s.sendState(channel, in.to)
	}()
	if !accept || from != in.to {
		return
	}
	if !s.viewers[viewer{channel, in.from}] {
		s.send(channel, in.to, game.Down{T: "duel", Note: s.record(in.from).Name + " isn't here any more"})
		return
	}
	d := &showdown{id: newID(), channel: channel, players: [2]uuid.UUID{in.from, in.to}}
	s.duels[d.id] = d
	s.duelState(d)
}

func (s *server) duelState(d *showdown) {
	for i, p := range d.players {
		s.send(d.channel, p, game.Down{T: "duel", Duel: &game.Duel{
			ID: d.id, Opponent: s.record(d.players[1-i]).Name, ScoreUs: d.score[i], ScoreThem: d.score[1-i]}})
	}
}

func (s *server) endDuel(d *showdown, why string) {
	delete(s.duels, d.id)
	for _, p := range d.players {
		s.send(d.channel, p, game.Down{T: "duel", Note: why})
	}
}

// ready starts a round once both players are ready: both clients are told
// together when the word comes, and each times its own player from it.
func (s *server) ready(from uuid.UUID, id string) {
	d := s.duels[id]
	if d == nil {
		return
	}
	i := slices.Index(d.players[:], from)
	if i < 0 {
		return
	}
	d.ready[i] = true
	if !d.ready[0] || !d.ready[1] {
		return
	}
	d.ready, d.shot = [2]bool{}, [2]*game.Up{}
	d.round++
	delay := 1500 + s.rnd.IntN(2500)
	for _, p := range d.players {
		s.send(d.channel, p, game.Down{T: "go", ID: d.id, Delay: delay})
	}
	// Someone who never draws loses the round.
	round := d.round
	time.AfterFunc(time.Duration(delay)*time.Millisecond+6*time.Second, func() {
		if s.post == nil {
			return
		}
		s.post(func() {
			if s.duels[id] == d && d.round == round {
				for k := range d.shot {
					if d.shot[k] == nil {
						d.shot[k] = &game.Up{Foul: true}
					}
				}
				s.judge(d)
			}
		})
	})
}

func (s *server) shot(from uuid.UUID, up game.Up) {
	d := s.duels[up.ID]
	if d == nil {
		return
	}
	i := slices.Index(d.players[:], from)
	if i < 0 || d.shot[i] != nil || d.ready[0] || d.ready[1] {
		return
	}
	if !up.Foul && (up.MS < game.Anticipated || up.MS > slowest) {
		up.Foul, up.MS = up.MS < game.Anticipated, min(up.MS, slowest)
	}
	d.shot[i] = &up
	if d.shot[0] != nil && d.shot[1] != nil {
		s.judge(d)
	}
}

// judge decides a round: a foul loses (both fouling is a draw), otherwise
// the faster time wins.
func (s *server) judge(d *showdown) {
	a, b := d.shot[0], d.shot[1]
	winner := -1
	switch {
	case a.Foul && !b.Foul:
		winner = 1
	case b.Foul && !a.Foul:
		winner = 0
	case !a.Foul && !b.Foul && a.MS < b.MS:
		winner = 0
	case !a.Foul && !b.Foul && b.MS < a.MS:
		winner = 1
	}
	if winner >= 0 {
		d.score[winner]++
	}
	d.round++ // a late timeout for this round does nothing now
	for i, p := range d.players {
		mine, theirs := d.shot[i], d.shot[1-i]
		s.counted(p, game.ModeShowdown, mine.MS, mine.Foul, winner == i, 0)
		s.send(d.channel, p, game.Down{T: "outcome", ID: d.id, Mine: mine.MS, Theirs: theirs.MS,
			Won: winner == i, Draw: winner < 0, MyFoul: mine.Foul, TheirFoul: theirs.Foul})
	}
	d.shot = [2]*game.Up{}
	s.duelState(d)
	s.everyone(d.channel)
}
