package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

func init() {
	RegisterPanel("Status", func(env Env) Panel { return &statusPane{env: env} })
}

// statusPane summarizes cluster health + the last poll.
type statusPane struct {
	env     Env
	snap    *Snapshot
	focused bool
}

func (s *statusPane) Init() tea.Cmd { return nil }

func (s *statusPane) Title() string { return "Status" }

func (s *statusPane) SetSnapshot(snap *Snapshot) { s.snap = snap }

func (s *statusPane) Focused() bool    { return s.focused }
func (s *statusPane) SetFocused(f bool) { s.focused = f }

func (s *statusPane) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return s, nil }

func (s *statusPane) View() string {
	if s.snap == nil {
		return "awaiting first poll…"
	}
	if s.snap.Err != nil {
		return styleBad.Render("connection error: " + s.snap.Err.Error())
	}
	up, tot, cpu, mem := 0, len(s.snap.Nodes), 0.0, 0.0
	for _, n := range s.snap.Nodes {
		if n.State == "healthy" || n.State == "pending" {
			up++
		}
		cpu += n.CPU
		mem += n.MemPct
	}
	avgCPU, avgMem := "n/a", "n/a"
	if tot > 0 {
		avgCPU = fmt.Sprintf("%.1f%%", cpu/float64(tot))
		avgMem = fmt.Sprintf("%.1f%%", mem/float64(tot))
	}
	return fmt.Sprintf("%d/%d nodes up · %d workloads · cpu avg %s · mem avg %s",
		up, tot, len(s.snap.Workloads), avgCPU, avgMem)
}
