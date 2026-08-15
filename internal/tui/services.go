package tui

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func init() {
	RegisterPanel("Services", func(env Env) Panel { return &servicesPane{env: env} })
}

// servicesPane lists workloads and their placement.
type servicesPane struct {
	env      Env
	snap     *Snapshot
	focused  bool
	selected int
}

func (s *servicesPane) Init() tea.Cmd { return nil }

func (s *servicesPane) Title() string { return "Services" }

func (s *servicesPane) SetSnapshot(snap *Snapshot) { s.snap = snap }

func (s *servicesPane) Focused() bool    { return s.focused }
func (s *servicesPane) SetFocused(f bool) { s.focused = f }

func (s *servicesPane) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		if !s.focused || s.snap == nil {
			return s, nil
		}
		switch key.String() {
		case "down", "j":
			if s.selected < len(s.snap.Workloads)-1 {
				s.selected++
			}
		case "up", "k":
			if s.selected > 0 {
				s.selected--
			}
		}
	}
	return s, nil
}

func (s *servicesPane) View() string {
	if s.snap == nil || len(s.snap.Workloads) == 0 {
		return styleHeader.Render("no workloads — run `servez deploy`")
	}
	ws := append([]WorkloadRow(nil), s.snap.Workloads...)
	sort.Slice(ws, func(i, j int) bool { return ws[i].Name < ws[j].Name })

	var b strings.Builder
	for i, w := range ws {
		marker := " "
		if i == s.selected && s.focused {
			marker = "▸"
		}
		state := styleGood.Render(w.State)
		switch w.State {
		case "unschedulable", "failed":
			state = styleBad.Render(w.State)
		case "declared", "pending":
			state = styleWarn.Render(w.State)
		}
		node := w.Node
		if node == "" {
			node = "unscheduled"
		}
		b.WriteString(fmt.Sprintf("%s %-12s %-16s %2d× %-14s %s\n",
			marker, w.Name, w.Image, w.Replicas, state, node))
	}
	return strings.TrimRight(b.String(), "\n")
}
