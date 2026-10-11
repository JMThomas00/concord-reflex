package game

// Up is a message from a player's client code to the server half.
type Up struct {
	// T is what it's about:
	//   hello      the client code started: send me my state
	//   result     a quick draw or practice finished (Mode, MS, Foul, Level, Won)
	//   equip      wear something owned (ID)
	//   deal       show me my offer of three
	//   pick       take one from the offer (ID)
	//   options    my sound and effects (Muted, Effects)
	//   people     who could I challenge?
	//   challenge  challenge someone (To: a user id)
	//   answer     accept or decline a challenge (ID, Accept), or cancel mine
	//   ready      ready for the next showdown round (ID)
	//   shot       my showdown time (ID, MS, Foul)
	//   leave      I'm out of the showdown (ID)
	T string `json:"t"`

	Mode    string `json:"mode,omitempty"`
	MS      int    `json:"ms,omitempty"`
	Foul    bool   `json:"foul,omitempty"`
	Won     bool   `json:"won,omitempty"`
	Level   int    `json:"level,omitempty"`
	ID      string `json:"id,omitempty"`
	To      string `json:"to,omitempty"`
	Accept  bool   `json:"accept,omitempty"`
	Muted   bool   `json:"muted,omitempty"`
	Effects string `json:"effects,omitempty"`
}

// Down is a message from the server half to one player's client code.
type Down struct {
	// T is what it's about:
	//   state    the player's profile and the Hall of Fame (Me, Top, Invites)
	//   people   who's here to challenge (People)
	//   duel     a showdown is on (Duel), or over (Duel nil, Note why)
	//   go       DRAW! comes in Delay ms (ID)
	//   outcome  the round's result (ID, Mine, Theirs, Won, Draw, MyFoul, TheirFoul)
	T string `json:"t"`

	Me      *Profile `json:"me,omitempty"`
	Top     []Entry  `json:"top,omitempty"`
	Invites []Invite `json:"invites,omitempty"`
	People  []Person `json:"people,omitempty"`
	Duel    *Duel    `json:"duel,omitempty"`
	Note    string   `json:"note,omitempty"`

	ID        string `json:"id,omitempty"`
	Delay     int    `json:"delay,omitempty"`
	Mine      int    `json:"mine,omitempty"`
	Theirs    int    `json:"theirs,omitempty"`
	Won       bool   `json:"won,omitempty"`
	Draw      bool   `json:"draw,omitempty"`
	MyFoul    bool   `json:"my_foul,omitempty"`
	TheirFoul bool   `json:"their_foul,omitempty"`
}

// Profile is one player's record and gear, as their client code shows it.
type Profile struct {
	Best      int               `json:"best,omitempty"` // fastest ms, 0 for none yet
	Duels     int               `json:"duels"`
	Wins      int               `json:"wins"`
	Streak    int               `json:"streak"`
	Bounties  int               `json:"bounties"`
	Owned     []string          `json:"owned,omitempty"`
	Offer     []string          `json:"offer,omitempty"`
	Equipped  map[string]string `json:"equipped,omitempty"` // kind -> id
	Muted     bool              `json:"muted,omitempty"`
	Effects   string            `json:"effects,omitempty"`    // "" (full), "calm" or "off"
	Earned    int               `json:"earned,omitempty"`     // Bounties from the last result
	NewRecord bool              `json:"new_record,omitempty"` // the last result was their fastest yet
}

// Entry is one line of the Hall of Fame.
type Entry struct {
	Name string `json:"name"`
	MS   int    `json:"ms"`
	Wins int    `json:"wins"`
	Me   bool   `json:"me,omitempty"`
}

// Person is someone in the channel who could be challenged.
type Person struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Online bool   `json:"online"`
	Best   int    `json:"best,omitempty"`
}

// Invite is a challenge waiting for an answer, to or from this player.
type Invite struct {
	ID   string `json:"id"`
	Name string `json:"name"` // the other player
	Mine bool   `json:"mine"` // sent by this player
}

// Duel is a showdown in progress, as one player sees it.
type Duel struct {
	ID        string `json:"id"`
	Opponent  string `json:"opponent"`
	ScoreUs   int    `json:"score_us"`
	ScoreThem int    `json:"score_them"`
}
