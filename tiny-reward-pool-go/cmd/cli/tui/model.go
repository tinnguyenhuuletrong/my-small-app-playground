package tui

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/actor"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
)

var (
	cmdStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("#00ff66ff"))
	logMsgStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#a8a8a8ff"))
	headerTextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FAFAFA")).Background(lipgloss.Color("#7D56F4"))
	statusStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#FAFAFA")).Background(lipgloss.Color("#A569BD"))
)

type Model struct {
	system       *actor.System
	viewport     viewport.Model
	textInput    textinput.Model
	history      []string
	ready        bool
	logChan      <-chan string
	debugView    viewport.Model
	logOutput    []string
	ShouldReload bool
	Quitting     bool
}

func NewModel(system *actor.System, logChan <-chan string) Model {
	ti := textinput.New()
	ti.Placeholder = "Enter command..."
	ti.Focus()
	ti.Width = 80

	dv := viewport.New(80, 10)

	return Model{
		system:       system,
		viewport:     viewport.New(80, 20),
		textInput:    ti,
		history:      []string{},
		logChan:      logChan,
		debugView:    dv,
		logOutput:    []string{},
		ShouldReload: false,
		Quitting:     false,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, waitForLog(m.logChan))
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	m.textInput, cmd = m.textInput.Update(msg)
	cmds = append(cmds, cmd)
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)
	m.debugView, cmd = m.debugView.Update(msg)
	cmds = append(cmds, cmd)

	switch msg := msg.(type) {
	case logMessage:
		m.logOutput = append(m.logOutput, string(msg))
		m.debugView.SetContent(logMsgStyle.Render(strings.Join(m.logOutput, "")))
		m.debugView.GotoBottom()
		cmds = append(cmds, waitForLog(m.logChan))

	case concurrentDrawsFinishedMsg:
		responses := []actor.DrawResponse(msg)
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

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			input := m.textInput.Value()
			m.textInput.Reset()
			parts := strings.Fields(input)

			// fallback to help cmd
			if len(parts) == 0 {
				parts = []string{"h"}
			}
			command := parts[0]
			args := parts[1:]
			cmds = m.onUserCommand(command, args, cmds)
		case tea.KeyCtrlC, tea.KeyEsc:
			m.Quitting = true
			cmds = append(cmds, tea.Quit)
		}

	case tea.WindowSizeMsg:
		m.onResize(msg)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if m.Quitting {
		return ""
	}
	if !m.ready {
		return "\n  Initializing..."
	}

	mainView := lipgloss.JoinVertical(
		lipgloss.Left,
		m.viewport.View(),
		m.footerView(),
	)

	debugRender := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), true, false, false, false).
		Render(m.debugView.View())

	return lipgloss.JoinVertical(
		lipgloss.Center,
		m.headerView(),
		mainView,
		debugRender,
	)
}

func (m *Model) onUserCommand(command string, args []string, cmds []tea.Cmd) []tea.Cmd {
	m.history = append(m.history, cmdStyle.Render(command+" "+strings.Join(args, " ")))
	switch command {
	case "h":
		m.history = append(m.history, getHelp())
	case "s":
		state := m.system.State()
		m.history = append(m.history, m.getStatus())
		m.history = append(m.history, prettyState(state))
	case "d":
		n := 1
		if len(args) > 0 {
			val, err := strconv.Atoi(args[0])
			if err == nil && val > 0 {
				n = val
			}
		}
		cmds = append(cmds, doConcurrentDraws(m.system, n))
	case "u":
		if len(args) != 3 {
			m.history = append(m.history, "Usage: u <id> <quantity> <weight>")
			break
		}
		id := args[0]
		quantity, errQty := strconv.Atoi(args[1])
		weight, errWeight := strconv.ParseInt(args[2], 10, 64)
		if errQty != nil || errWeight != nil {
			m.history = append(m.history, "Invalid quantity or weight")
			break
		}
		if err := m.system.UpdateItem(id, quantity, weight); err != nil {
			m.history = append(m.history, fmt.Sprintf("Failed to update item %s: %v", id, err))
		} else {
			m.history = append(m.history, fmt.Sprintf("Updated item %s", id))
		}
	case "r":
		m.ShouldReload = true
		return append(cmds, tea.Quit)
	case "q":
		m.Quitting = true
		return append(cmds, tea.Quit)
	default:
		m.history = append(m.history, "Unknown command. Type 'h' for help.")
	}
	m.viewport.SetContent(strings.Join(m.history, "\n"))
	m.viewport.GotoBottom()
	return cmds
}

func (m *Model) onResize(msg tea.WindowSizeMsg) {
	headerHeight := lipgloss.Height(m.headerView())
	windowWidth := msg.Width
	viewportHeight := 30
	debugViewHeight := 10

	if !m.ready {
		m.viewport = viewport.New(windowWidth, viewportHeight)
		m.viewport.YPosition = headerHeight
		m.debugView = viewport.New(windowWidth, debugViewHeight)
		m.ready = true
	} else {
		m.viewport.Width = windowWidth
		m.debugView.Width = windowWidth
	}
	m.textInput.Width = windowWidth
	logMsgStyle = logMsgStyle.Width(windowWidth)
}

func (m Model) headerView() string {
	return headerTextStyle.Render("Reward Pool TUI") + " " + statusStyle.Render(fmt.Sprintf("Request ID: %d", m.system.GetRequestID()))
}

func (m Model) footerView() string {
	return m.textInput.View()
}

func getHelp() string {
	return "\nCommands:\n" +
		"  h          - Show this help message\n" +
		"  s          - Show pool status\n" +
		"  d [n]      - Draw [n] items (default: 1)\n" +
		"  u <id> <qty> <w> - Update item quantity and weight\n" +
		"  r          - Reload pool from config\n" +
		"  q          - Quit\n"
}

func (m *Model) getStatus() string {
	// More detailed status can be added here
	return fmt.Sprintf("Actor System is running. Last Request ID: %d", m.system.GetRequestID())
}

func prettyState(state []types.PoolReward) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("% -15s % -10s % -10s\n", "ItemID", "Quantity", "Weight"))
	builder.WriteString(strings.Repeat("-", 37) + "\n")
	for _, item := range state {
		builder.WriteString(fmt.Sprintf("% -15s % -10d % -10d\n", item.ItemID, item.Quantity, item.Probability))
	}
	return builder.String()
}

type concurrentDrawsFinishedMsg []actor.DrawResponse
type logMessage string

func waitForLog(ch <-chan string) tea.Cmd {
	return func() tea.Msg {
		return logMessage(<-ch)
	}
}

func doConcurrentDraws(system *actor.System, n int) tea.Cmd {
	return func() tea.Msg {
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
