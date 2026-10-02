package main

import (
	"os"
	"path/filepath"
	"strconv"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/JMThomas00/concord-reflex/game"
)

// Standalone: the same game in a terminal, with the best time kept in the
// user's config folder.

type timerMsg string

type local struct {
	g    *game.Game
	cmds []tea.Cmd
	w, h int
}

func (m *local) After(id string, d time.Duration) {
	m.cmds = append(m.cmds, tea.Tick(d, func(time.Time) tea.Msg { return timerMsg(id) }))
}

func (m *local) SaveBest(ms int) {
	if p := bestFile(); p != "" {
		_ = os.MkdirAll(filepath.Dir(p), 0o755)
		_ = os.WriteFile(p, []byte(strconv.Itoa(ms)), 0o644)
	}
}

func (m *local) Report(int) {}

func bestFile() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "concord-reflex", "best")
}

func (m *local) Init() tea.Cmd { return nil }

func (m *local) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.cmds = nil
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		m.g.Width, m.g.Height = msg.Width, msg.Height-2
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		}
		m.g.Key(msg.String(), m)
	case timerMsg:
		m.g.Timer(string(msg), m)
	}
	return m, tea.Batch(m.cmds...)
}

func (m *local) View() string {
	body := m.g.View() + "\n\n" + m.g.Colors.Dim("q to quit")
	return lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center, body)
}

func runLocal() error {
	best := 0
	if p := bestFile(); p != "" {
		if b, err := os.ReadFile(p); err == nil {
			best = game.ParseBest(string(b))
		}
	}
	_, err := tea.NewProgram(&local{g: game.New(best)}, tea.WithAltScreen()).Run()
	return err
}
