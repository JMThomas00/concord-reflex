// Package game is what Reflex's two halves share: the gear players unlock,
// the titles their times earn, the outlaw's speeds, and the messages the
// client code and the server half send each other.
package game

import "github.com/JMThomas00/Concord/sdk/arcade"

const (
	KindHat   = "hat"
	KindScene = "scene"
	// Backdrop ids are prefixed so they can't clash with a hat's.
	ScenePrefix = "scene:"
)

// Hat is something to wear: its top three rows of pixels and their roles.
type Hat struct {
	arcade.Unlockable
	Rows []string
	Pal  map[rune]string
}

// Scene is a backdrop: sky, ground and far hills, and what's in the sky.
type Scene struct {
	arcade.Unlockable
	Sky, Ground, Far string
	Sun              string // a sun of this role, if any
	Moon, Stars      bool
	Snow, Vines      bool
	Earth            bool
}

func hat(id, name string, tier arcade.Tier, blurb string, rows []string, pal map[rune]string) Hat {
	return Hat{arcade.Unlockable{ID: id, Name: name, Tier: tier, Kind: KindHat, Blurb: blurb}, rows, pal}
}

var brim = []string{"....HHHH....", "..HHHHHHHH..", "....hhhh...."}

// Hats are in catalogue order: starters first.
var Hats = []Hat{
	hat("stetson", "STETSON", arcade.Starter, "Wide brim, wider sky.", brim, map[rune]string{'H': "orangeD", 'h': "tire"}),
	hat("black", "BLACK HAT", arcade.Starter, "Every outlaw's first.", brim, map[rune]string{'H': "tire", 'h': "red"}),
	hat("white", "WHITE HAT", arcade.Common, "The good guy, probably.", brim, map[rune]string{'H': "fg", 'h': "cyan"}),
	hat("bowler", "BOWLER", arcade.Common, "From back East.", []string{".....HH.....", "....HHHH....", "...hhhhhh..."}, map[rune]string{'H': "tire", 'h': "comment"}),
	hat("sombrero", "SOMBRERO", arcade.Common, "Shade for the whole town.", []string{".....HH.....", "HHHHHHHHHHHH", "..hhhhhhhh.."}, map[rune]string{'H': "yellow", 'h': "red"}),
	hat("tophat", "TOP HAT", arcade.Rare, "The undertaker's favourite.", []string{"....HHHH....", "....hhhh....", "..HHHHHHHH.."}, map[rune]string{'H': "tire", 'h': "purple"}),
	hat("feather", "FEATHER HAT", arcade.Rare, "Showing off, mostly.", []string{"....HHHHp...", "..HHHHHHHHp.", "....hhhh...."}, map[rune]string{'H': "orange", 'h': "tire", 'p': "pink"}),
	hat("grape", "GRAPE HEAD", arcade.Legendary, "The fastest vine in the West.", []string{".....gg.....", "...PPpPP....", "..PPPPPPP..."}, map[rune]string{'P': "purple", 'p': "hi", 'g': "green"}),
	hat("robot", "ROBOT", arcade.Legendary, "Reaction time: yes.", []string{".....r......", "...HHHHHH...", "...HrHHrH..."}, map[rune]string{'H': "comment", 'r': "red"}),
}

func scene(id, name string, tier arcade.Tier, blurb string, s Scene) Scene {
	s.Unlockable = arcade.Unlockable{ID: ScenePrefix + id, Name: name, Tier: tier, Kind: KindScene, Blurb: blurb}
	return s
}

// Scenes are in catalogue order: starters first.
var Scenes = []Scene{
	scene("noon", "HIGH NOON", arcade.Starter, "Not a cloud in the sky.", Scene{Sky: "cyanB", Ground: "yellowB", Far: "orangeB", Sun: "yellow"}),
	scene("sunset", "SUNSET", arcade.Starter, "Ride off into it.", Scene{Sky: "orangeB", Ground: "redB", Far: "pinkB", Sun: "orange"}),
	scene("night", "MIDNIGHT", arcade.Common, "Only the moon is watching.", Scene{Sky: "bg", Ground: "line", Far: "purpleB", Moon: true, Stars: true}),
	scene("snow", "SNOWED IN", arcade.Common, "Cold hands, slow draw.", Scene{Sky: "line", Ground: "fgD", Far: "cyanB", Snow: true}),
	scene("canyon", "RED CANYON", arcade.Rare, "Echoes for miles.", Scene{Sky: "orangeB", Ground: "redD", Far: "redB", Sun: "yellow"}),
	scene("vineyard", "VINEYARD", arcade.Legendary, "Pistols at the harvest.", Scene{Sky: "cyanB", Ground: "greenB", Far: "purpleB", Sun: "yellow", Vines: true}),
	scene("moon", "ON THE MOON", arcade.Legendary, "One small draw.", Scene{Sky: "bg", Ground: "dim", Far: "line", Stars: true, Earth: true}),
}

// Unlockables is every hat and backdrop, for the rewards.
func Unlockables() []arcade.Unlockable {
	var out []arcade.Unlockable
	for _, h := range Hats {
		out = append(out, h.Unlockable)
	}
	for _, s := range Scenes {
		out = append(out, s.Unlockable)
	}
	return out
}

// HatByID and SceneByID look gear up (the first starter if unknown).
func HatByID(id string) *Hat {
	for i := range Hats {
		if Hats[i].ID == id {
			return &Hats[i]
		}
	}
	return &Hats[0]
}

func SceneByID(id string) *Scene {
	for i := range Scenes {
		if Scenes[i].ID == id {
			return &Scenes[i]
		}
	}
	return &Scenes[0]
}

// Rank is the title a time earns.
type Rank struct {
	Under int // milliseconds
	Title string
	Role  string
}

// Ranks, fastest first.
var Ranks = []Rank{
	{170, "LEGEND", "yellow"},
	{210, "SHERIFF", "pink"},
	{250, "DEPUTY", "cyan"},
	{300, "RANGER", "green"},
	{1 << 30, "GREENHORN", "dim"},
}

// RankOf is the title for ms.
func RankOf(ms int) Rank {
	for _, r := range Ranks {
		if ms < r.Under {
			return r
		}
	}
	return Ranks[len(Ranks)-1]
}

// Levels are the outlaw's: his name for it and roughly how fast he draws.
var Levels = []struct {
	Name string
	MS   int
}{{"EASY", 430}, {"NORMAL", 300}, {"HARD", 215}}

// Anticipated is the fastest time anyone really reacts in: anything faster
// was a guess, and counts as drawing too soon.
const Anticipated = 100

// Modes of play.
const (
	ModeDuel     = "duel"     // quick draw against the outlaw
	ModeShowdown = "showdown" // against someone in the channel
	ModePractice = "practice" // just the clock
)
