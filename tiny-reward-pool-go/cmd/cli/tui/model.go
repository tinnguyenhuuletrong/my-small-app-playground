package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	bubbletea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/actor"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
)

var cmdStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#00ff66ff"))

type Model struct {
	system    *actor.System
	viewport  viewport.Model
	textInput textinput.Model
	history   []string
	ready     bool
	error     error
}

func NewModel(system *actor.System) Model {
	ti := textinput.New()
	ti.Placeholder = "Enter command..."
	ti.Focus()
	ti.Width = 80

	return Model{
		system:    system,
		textInput: ti,
		history:   []string{},
	}
}

func (m Model) Init() bubbletea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg bubbletea.Msg) (bubbletea.Model, bubbletea.Cmd) {
	var (
		cmd  bubbletea.Cmd
		cmds []bubbletea.Cmd
	)

	m.textInput, cmd = m.textInput.Update(msg)
	cmds = append(cmds, cmd)

	switch msg := msg.(type) {
	case bubbletea.KeyMsg:
		switch msg.Type {
		case bubbletea.KeyEnter:
			input := m.textInput.Value()
			m.textInput.Reset()

			parts := strings.Fields(input)
			if len(parts) == 0 {
				return m, nil
			}

			command := parts[0]
			// args := parts[1:]

			switch command {
			case "/state":
				state := m.system.State()
				reqId := m.system.GetRequestID()

				output := prettyState(state)
				m.history = append(m.history, cmdStyle.Render("/state"))
				m.history = append(m.history, fmt.Sprintf("LatestRequestId: %d", reqId))
				m.history = append(m.history, output)
				m.viewport.SetContent(strings.Join(m.history, "\n"))
				m.viewport.GotoBottom()
			case "/draw":
				resp := <-m.system.Draw()
				var output string
				if resp.Err != nil {
					output = fmt.Sprintf("[Request %d] Draw failed: %v", resp.RequestID, resp.Err)
				} else {
					output = fmt.Sprintf("[Request %d] You drew: %s", resp.RequestID, resp.Item)
				}
				m.history = append(m.history, cmdStyle.Render("/draw"))
				m.history = append(m.history, output)
				m.viewport.SetContent(strings.Join(m.history, "\n"))
				m.viewport.GotoBottom()
			}
		case bubbletea.KeyCtrlC, bubbletea.KeyEsc:
			return m, bubbletea.Quit
		}
	case bubbletea.WindowSizeMsg:
		headerHeight := lipgloss.Height(m.headerView())
		viewportHeight := 10

		if !m.ready {
			m.viewport = viewport.New(msg.Width-2, viewportHeight)
			m.viewport.YPosition = headerHeight
			m.ready = true
		} else {
			m.viewport.Width = msg.Width - 2
		}
		m.textInput.Width = msg.Width - 4
	}

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, bubbletea.Batch(cmds...)
}

func (m Model) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	return lipgloss.JoinVertical(
		lipgloss.Center,
		m.headerView(),
		m.viewport.View(),
		m.footerView(),
	)
}

func (m Model) headerView() string {
	var style = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4"))
	return style.Render("Reward Pool TUI")
}

func (m Model) footerView() string {
	return m.textInput.View()
}

func prettyState(state []types.PoolReward) string {
	var builder strings.Builder
	for _, item := range state {
		builder.WriteString(fmt.Sprintf("ItemID: %-10s Quantity: %-5d Probability: %d\n", item.ItemID, item.Quantity, item.Probability))
	}
	return builder.String()
}
