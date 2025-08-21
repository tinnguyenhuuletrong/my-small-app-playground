package tui

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	bubbletea "github.com/charmbracelet/bubbletea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/actor"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
)

var cmdStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#00ff66ff"))

type Model struct {
	system       *actor.System
	viewport     viewport.Model
	textInput    textinput.Model
	history      []string
	ready        bool
	logChan      <-chan string
	debugView    viewport.Model
	debugHistory []string
}

func NewModel(system *actor.System, logChan <-chan string) Model {

	ti := textinput.New()

	ti.Placeholder = "Enter command..."

	ti.Focus()

	ti.Width = 80

	dv := viewport.New(80, 10)
	dv.SetContent("Debug log...")

	return Model{
		system:       system,
		textInput:    ti,
		history:      []string{},
		logChan:      logChan,
		debugView:    dv,
		debugHistory: []string{},
	}
}

func (m Model) Init() bubbletea.Cmd {
	return tea.Batch(textinput.Blink, waitForLog(m.logChan))
}

type concurrentDrawsFinishedMsg []actor.DrawResponse
type logMessage string

func waitForLog(ch <-chan string) bubbletea.Cmd {
	return func() bubbletea.Msg {
		return logMessage(<-ch)
	}
}

func doConcurrentDraws(system *actor.System, n int) bubbletea.Cmd {
	return func() bubbletea.Msg {
		var responses []actor.DrawResponse
		var wg sync.WaitGroup
		var mu sync.Mutex

		wg.Add(n)

		for i := 0; i < n; i++ {
			go func() {
				defer wg.Done()
				resp := <-system.Draw()
				mu.Lock()
				responses = append(responses, resp)
				mu.Unlock()
			}()
		}

		wg.Wait()
		return concurrentDrawsFinishedMsg(responses)
	}
}

func (m Model) Update(msg bubbletea.Msg) (bubbletea.Model, bubbletea.Cmd) {
	var (
		cmd  bubbletea.Cmd
		cmds []bubbletea.Cmd
	)

	switch msg := msg.(type) {
	case logMessage:
		m.debugHistory = append(m.debugHistory, string(msg))
		m.debugView.SetContent(strings.Join(m.debugHistory, ""))
		m.debugView.GotoBottom()
		return m, waitForLog(m.logChan)

	case concurrentDrawsFinishedMsg:
		responses := []actor.DrawResponse(msg)
		// Sort responses by RequestID
		sort.Slice(responses, func(i, j int) bool {
			return responses[i].RequestID < responses[j].RequestID
		})

		for _, resp := range responses {
			var output string
			if resp.Err != nil {
				output = fmt.Sprintf("[Request %d] Draw failed: %v", resp.RequestID, resp.Err)
			} else {
				output = fmt.Sprintf("[Request %d] You drew: %s", resp.RequestID, resp.Item)
			}
			m.history = append(m.history, output)
		}
		m.viewport.SetContent(strings.Join(m.history, "\n"))
		m.viewport.GotoBottom()
		return m, nil

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
			args := parts[1:]

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
				return m, nil
			case "/draw":
				n := 1
				if len(args) > 0 {
					val, err := strconv.Atoi(args[0])
					if err == nil && val > 0 {
						n = val
					}
				}

				m.history = append(m.history, cmdStyle.Render(fmt.Sprintf("/draw %d", n)))

				return m, doConcurrentDraws(m.system, n)
			}
		case bubbletea.KeyCtrlC, bubbletea.KeyEsc:
			m.system.Stop()
			return m, bubbletea.Batch(waitForLog(m.logChan), bubbletea.Quit)
		}
	case bubbletea.WindowSizeMsg:
		headerHeight := lipgloss.Height(m.headerView())
		viewportHeight := 10
		debugViewHeight := 10

		if !m.ready {
			m.viewport = viewport.New(msg.Width-2, viewportHeight)
			m.viewport.YPosition = headerHeight
			m.debugView = viewport.New(msg.Width-2, debugViewHeight)
			m.ready = true
		} else {
			m.viewport.Width = msg.Width - 2
			m.debugView.Width = msg.Width - 2
		}
		m.textInput.Width = msg.Width - 4
	}

	m.textInput, cmd = m.textInput.Update(msg)
	cmds = append(cmds, cmd)

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	m.debugView, cmd = m.debugView.Update(msg)
	cmds = append(cmds, cmd)

	return m, bubbletea.Batch(cmds...)
}

func (m Model) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	mainView := lipgloss.JoinVertical(
		lipgloss.Left,
		m.viewport.View(),
		m.footerView(),
	)

	debugView := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), true).
		SetString("Debug Log").
		Render(m.debugView.View())

	return lipgloss.JoinVertical(
		lipgloss.Center,
		m.headerView(),
		mainView,
		debugView,
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
