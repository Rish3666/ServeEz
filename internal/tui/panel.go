package tui

import (
	"log"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Rish3666/ServeEz/internal/apiclient"
)

// Env is the shared context handed to every panel. Panels use it to talk to
// the control plane (actions, chat) and to log.
type Env struct {
	Client     *apiclient.Client
	ControlURL string
	Logger     *log.Logger
}

// Panel is a renderable pane in the dashboard. It is a Bubble Tea model plus a
// small contract for the app shell:
//
//   - Title() is shown in the pane's border.
//   - SetSnapshot() is called by the shell after every poll with the latest
//     cluster state; panels store it and render from it.
//   - Focused/SetFocused track which pane receives key input.
//
// Panels implement tea.Model (Init/Update/View). The shell forwards key
// messages only to the focused panel.
type Panel interface {
	tea.Model
	Title() string
	SetSnapshot(*Snapshot)
	Focused() bool
	SetFocused(bool)
}

// PanelFactory builds a panel with access to the shared Env. Registered via
// RegisterPanel so the shell stays decoupled from specific panes.
type PanelFactory func(env Env) Panel

var (
	panelFactories []PanelFactory
	panelNames     []string
)

// RegisterPanel adds a panel factory to the dashboard. Called from init() in
// each panel file so the app shell can discover panes without imports.
func RegisterPanel(name string, f PanelFactory) {
	panelNames = append(panelNames, name)
	panelFactories = append(panelFactories, f)
}

// buildPanels instantiates all registered panels.
func buildPanels(env Env) []Panel {
	out := make([]Panel, 0, len(panelFactories))
	for _, f := range panelFactories {
		out = append(out, f(env))
	}
	return out
}
