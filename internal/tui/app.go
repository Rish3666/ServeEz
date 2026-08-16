package tui

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Rish3666/ServeEz/internal/apiclient"
)

// pollInterval is how often the shell refreshes cluster state.
const pollInterval = 2 * time.Second

// Style kit for the dashboard.
var (
	styleApp      = lipgloss.NewStyle().Padding(0, 1)
	styleTitle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	styleHeader   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	styleGood     = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	styleWarn     = lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	styleBad      = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	stylePane     = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
	stylePaneFoc  = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("45")).Padding(0, 1)
	styleStatus   = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	styleHelp     = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
)

type tickMsg time.Time
type stateMsg struct{ snap *Snapshot }

// App is the root Bubble Tea model.
type App struct {
	env     Env
	panels  []Panel
	focused int
	snap    *Snapshot
	width   int
	height  int
}

// NewApp builds the dashboard root model. controlURL defaults to
// http://localhost:8443 when empty.
func NewApp(controlURL string) *App {
	if controlURL == "" {
		controlURL = "http://localhost:8443"
	}
	env := Env{
		Client:     apiclient.New(controlURL),
		ControlURL: controlURL,
		Logger:     log.New(os.Stderr, "servez-tui: ", 0),
	}
	return &App{
		env:     env,
		panels:  buildPanels(env),
		focused: 0,
		snap:    &Snapshot{},
	}
}

func (a *App) Init() tea.Cmd {
	cmds := []tea.Cmd{tea.Tick(0, func(t time.Time) tea.Msg { return tickMsg(t) })}
	for _, p := range a.panels {
		if c := p.Init(); c != nil {
			cmds = append(cmds, c)
		}
	}
	return tea.Batch(cmds...)
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		a.width, a.height = m.Width, m.Height
	case tickMsg:
		return a, tea.Batch(
			func() tea.Msg {
				ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
				defer cancel()
				return stateMsg{snap: fetchState(ctx, a.env.Client, a.env.ControlURL)}
			},
			tea.Tick(pollInterval, func(t time.Time) tea.Msg { return tickMsg(t) }),
		)
	case stateMsg:
		a.snap = m.snap
		for _, p := range a.panels {
			p.SetSnapshot(m.snap)
		}
	case tea.KeyMsg:
		return a, a.handleKey(m)
	default:
		// Forward to the focused panel so it can react (e.g. chat input).
		if len(a.panels) > 0 {
			idx := a.focused
			updated, cmd := a.panels[idx].Update(msg)
			if p, ok := updated.(Panel); ok {
				a.panels[idx] = p
			}
			return a, cmd
		}
	}
	return a, nil
}

func (a *App) handleKey(m tea.KeyMsg) tea.Cmd {
	switch m.String() {
	case "ctrl+c", "q":
		return tea.Quit
	case "tab", "shift+tab":
		if len(a.panels) == 0 {
			return nil
		}
		a.panels[a.focused].SetFocused(false)
		dir := 1
		if m.String() == "shift+tab" {
			dir = -1
		}
		a.focused = (a.focused + dir + len(a.panels)) % len(a.panels)
		a.panels[a.focused].SetFocused(true)
		return nil
	}
	// All other keys go to the focused panel.
	if len(a.panels) > 0 {
		idx := a.focused
		updated, cmd := a.panels[idx].Update(m)
		if p, ok := updated.(Panel); ok {
			a.panels[idx] = p
		}
		return cmd
	}
	return nil
}

func (a *App) View() string {
	health := "no data"
	state := a.snap
	if state != nil && len(state.Nodes) > 0 {
		up := 0
		for _, n := range state.Nodes {
			if n.State == "healthy" || n.State == "pending" {
				up++
			}
		}
		health = fmt.Sprintf("%d/%d healthy", up, len(state.Nodes))
	}

	header := lipgloss.JoinHorizontal(lipgloss.Top,
		styleTitle.Render("ServeEz"),
		styleHeader.Render(" "+health),
		styleStatus.Render("  "+a.env.ControlURL),
	)

	var body strings.Builder
	// Render panes in a two-column grid, alternating left/right so any number
	// of registered panes is laid out (panes register in a stable order: the
	// shell's own Status pane is always last, hence the 2x2 look).
	left, right := []string{}, []string{}
	for i := range a.panels {
		rendered := a.renderPane(i)
		if i%2 == 0 {
			left = append(left, rendered)
		} else {
			right = append(right, rendered)
		}
	}
	leftCol := lipgloss.JoinVertical(lipgloss.Left, left...)
	rightCol := lipgloss.JoinVertical(lipgloss.Left, right...)
	if len(right) > 0 {
		body.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, leftCol, rightCol))
	} else {
		body.WriteString(leftCol)
	}

	status := a.statusLine()
	help := styleHelp.Render("tab: focus  ↑/↓: navigate  enter: select  q: quit")

	out := lipgloss.JoinVertical(lipgloss.Left,
		header,
		body.String(),
		status,
		help,
	)
	return styleApp.Render(out)
}

func (a *App) renderPane(idx int) string {
	if idx >= len(a.panels) {
		return ""
	}
	p := a.panels[idx]
	style := stylePane
	if p.Focused() {
		style = stylePaneFoc
	}
	return style.Render(p.Title() + "\n" + p.View())
}

func (a *App) statusLine() string {
	s := a.snap
	if s == nil {
		return styleStatus.Render("awaiting first poll…")
	}
	if s.Err != nil {
		return styleBad.Render("connection error: " + s.Err.Error())
	}
	up, tot, cpu, mem := 0, len(s.Nodes), 0.0, 0.0
	for _, n := range s.Nodes {
		if n.State == "healthy" || n.State == "pending" {
			up++
		}
		cpu += n.CPU
		mem += n.MemPct
	}
	var avgCPU, avgMem float64
	if tot > 0 {
		avgCPU, avgMem = cpu/float64(tot), mem/float64(tot)
	}
	return styleStatus.Render(fmt.Sprintf(
		"%s | %d workloads | nodes: %d/%d | cpu avg %.1f%% | mem avg %.1f%% | updated %s",
		healthDots(s.Nodes), len(s.Workloads), up, tot, avgCPU, avgMem,
		s.FetchedAt.Format("15:04:05")))
}

func healthDots(nodes []NodeRow) string {
	var b strings.Builder
	for _, n := range nodes {
		switch n.State {
		case "healthy", "pending":
			b.WriteString(styleGood.Render("●"))
		case "degraded":
			b.WriteString(styleWarn.Render("●"))
		default:
			b.WriteString(styleBad.Render("●"))
		}
	}
	return b.String()
}
