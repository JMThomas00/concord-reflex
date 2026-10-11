package main

import (
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"

	"github.com/JMThomas00/concord-reflex/game"
)

// Standalone: the same arcade in a terminal. The server half runs in the
// same program (records, Bounties and the store, kept in the user's config
// folder); there's no one to have a showdown with.

type timerMsg string
type downMsg game.Down

var (
	localChannel = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	localPlayer  = uuid.MustParse("00000000-0000-0000-0000-000000000002")
)

type local struct {
	app  *game.App
	srv  *server
	cmds []tea.Cmd
	w, h int
}

func (m *local) After(id string, d time.Duration) {
	m.cmds = append(m.cmds, tea.Tick(d, func(time.Time) tea.Msg { return timerMsg(id) }))
}

// Send hands a message to the in-process server half; its answers come
// back as messages, like the network would bring them.
func (m *local) Send(up game.Up) { m.srv.message(localChannel, localPlayer, up) }

func (m *local) Sound(string) {} // a terminal has no sounds

func (m *local) Init() tea.Cmd {
	m.app.Start()
	return tea.Batch(m.take()...)
}

func (m *local) take() []tea.Cmd {
	cmds := m.cmds
	m.cmds = nil
	return cmds
}

func (m *local) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		m.app.W, m.app.H = msg.Width, msg.Height
	case tea.KeyMsg:
		if k := msg.String(); k == "ctrl+c" {
			return m, tea.Quit
		}
		if k := msg.String(); k == "esc" || k == "q" {
			if m.app.OnTitle() {
				return m, tea.Quit // Esc on the title leaves, as it does in Concord
			}
		}
		m.app.Key(msg.String())
	case timerMsg:
		m.app.Timer(string(msg))
	case downMsg:
		m.app.Server(game.Down(msg))
	}
	m.app.Shown() // Bubble Tea draws this update next
	return m, tea.Batch(m.take()...)
}

func (m *local) View() string { return m.app.View() }

func runLocal() error {
	m := &local{srv: newServer()}
	if dir, err := os.UserConfigDir(); err == nil {
		m.srv.dir = filepath.Join(dir, "concord-reflex")
	}
	m.srv.load()
	m.srv.record(localPlayer).Name = playerName()
	m.srv.viewers[viewer{localChannel, localPlayer}] = true
	// The server half answers by queueing a message for the next update.
	m.srv.out = func(_, _ uuid.UUID, d game.Down) {
		m.cmds = append(m.cmds, func() tea.Msg { return downMsg(d) })
	}
	m.srv.post = func(fn func()) { fn() }
	m.app = game.New(m, nil)
	m.app.Standalone = true
	_, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
	return err
}

func playerName() string {
	for _, k := range []string{"USER", "USERNAME"} {
		if v := os.Getenv(k); v != "" {
			return v
		}
	}
	return "You"
}
