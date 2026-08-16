package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func init() {
	RegisterPanel("Alerts", func(env Env) Panel { return &alertsPane{env: env, firstSeen: map[string]int{}} })
}

type alertsPane struct {
	env       Env
	snap      *Snapshot
	focused   bool
	scroll    int
	cycle     int
	firstSeen map[string]int
	alerts    []alertItem
}

type alertItem struct {
	severity string
	text     string
}

func (a *alertsPane) Init() tea.Cmd { return nil }

func (a *alertsPane) Title() string { return "Alerts" }

func (a *alertsPane) SetSnapshot(s *Snapshot) {
	a.snap = s
	if s == nil {
		a.alerts = nil
		return
	}
	a.cycle++
	a.alerts = a.alerts[:0]

	for i := len(s.Nodes) - 1; i >= 0; i-- {
		n := s.Nodes[i]
		switch n.State {
		case "unhealthy", "disconnected", "cordoned":
			a.alerts = append(a.alerts, alertItem{severity: "bad", text: fmt.Sprintf("node %s is %s", n.Name, n.State)})
		case "degraded":
			a.alerts = append(a.alerts, alertItem{severity: "warn", text: fmt.Sprintf("node %s is degraded", n.Name)})
		}
	}

	for i := len(s.Workloads) - 1; i >= 0; i-- {
		w := s.Workloads[i]
		switch w.State {
		case "unschedulable":
			a.alerts = append(a.alerts, alertItem{severity: "bad", text: fmt.Sprintf("workload %s is unschedulable", w.Name)})
		case "declared", "pending":
			if first, ok := a.firstSeen[w.Name]; !ok {
				a.firstSeen[w.Name] = a.cycle
			} else if a.cycle-first > 0 {
				a.alerts = append(a.alerts, alertItem{severity: "info", text: fmt.Sprintf("workload %s not yet scheduled", w.Name)})
			}
		default:
			delete(a.firstSeen, w.Name)
		}
	}

	if len(a.alerts) > 1 {
		reverseAlerts(a.alerts)
	}
	if a.scroll > max(0, len(a.alerts)-1) {
		a.scroll = max(0, len(a.alerts)-1)
	}
}

func (a *alertsPane) Focused() bool     { return a.focused }
func (a *alertsPane) SetFocused(f bool) { a.focused = f }

func (a *alertsPane) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok && a.focused {
		switch key.String() {
		case "j", "down":
			if a.scroll < max(0, len(a.alerts)-1) {
				a.scroll++
			}
		case "k", "up":
			if a.scroll > 0 {
				a.scroll--
			}
		}
	}
	return a, nil
}

func (a *alertsPane) View() string {
	if a.snap == nil {
		return styleHeader.Render("awaiting first poll…")
	}
	if len(a.alerts) == 0 {
		return styleHeader.Render("no alerts")
	}

	start := a.scroll
	if start < 0 {
		start = 0
	}
	if start >= len(a.alerts) {
		start = len(a.alerts) - 1
	}
	end := start + 20
	if end > len(a.alerts) {
		end = len(a.alerts)
	}
	lines := make([]string, 0, end-start)
	for _, al := range a.alerts[start:end] {
		lines = append(lines, renderAlert(al))
	}
	footer := ""
	if len(a.alerts) > 20 {
		footer = fmt.Sprintf("\n%s", styleHelp.Render(fmt.Sprintf("scroll %d/%d", start, len(a.alerts)-1)))
	}
	return strings.Join(lines, "\n") + footer
}

func renderAlert(a alertItem) string {
	dot := styleGood.Render("●")
	switch a.severity {
	case "warn":
		dot = styleWarn.Render("●")
	case "bad":
		dot = styleBad.Render("●")
	case "info":
		dot = styleHeader.Render("●")
	}
	return fmt.Sprintf("%s %s", dot, a.text)
}

func reverseAlerts(items []alertItem) {
	for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
		items[i], items[j] = items[j], items[i]
	}
}

var _ = lipgloss.NewStyle
var _ Panel = (*alertsPane)(nil)
