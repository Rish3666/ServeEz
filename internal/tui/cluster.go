package tui

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func init() {
	RegisterPanel("Cluster", func(env Env) Panel { return &clusterPane{env: env} })
}

// clusterPane lists cluster nodes with health + utilization.
type clusterPane struct {
	env      Env
	snap     *Snapshot
	focused  bool
	selected int
}

func (c *clusterPane) Init() tea.Cmd { return nil }

func (c *clusterPane) Title() string { return "Cluster" }

func (c *clusterPane) SetSnapshot(s *Snapshot) { c.snap = s }

func (c *clusterPane) Focused() bool    { return c.focused }
func (c *clusterPane) SetFocused(f bool) { c.focused = f }

func (c *clusterPane) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		if !c.focused || c.snap == nil {
			return c, nil
		}
		switch key.String() {
		case "down", "j":
			if c.selected < len(c.snap.Nodes)-1 {
				c.selected++
			}
		case "up", "k":
			if c.selected > 0 {
				c.selected--
			}
		}
	}
	return c, nil
}

func (c *clusterPane) View() string {
	if c.snap == nil || len(c.snap.Nodes) == 0 {
		return styleHeader.Render("no nodes — run `servez join`")
	}
	nodes := append([]NodeRow(nil), c.snap.Nodes...)
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].Name < nodes[j].Name })

	var b strings.Builder
	for i, n := range nodes {
		marker := " "
		if i == c.selected && c.focused {
			marker = "▸"
		}
		dot := styleGood.Render("●")
		switch n.State {
		case "degraded":
			dot = styleWarn.Render("●")
		case "unhealthy", "disconnected", "cordoned":
			dot = styleBad.Render("●")
		case "pending":
			dot = styleHeader.Render("●")
		}
		mem := fmt.Sprintf("mem %.1f%%", n.MemPct)
		if n.MemCap == 0 {
			mem = "mem n/a"
		}
		b.WriteString(fmt.Sprintf("%s %s %-12s %-11s cpu %5.1f%% %s\n",
			marker, dot, n.Name, n.State, n.CPU, mem))
	}
	return strings.TrimRight(b.String(), "\n")
}

var _ = lipgloss.NewStyle
