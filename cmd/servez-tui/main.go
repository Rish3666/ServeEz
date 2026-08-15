// servez-tui is the ServeEz terminal dashboard (htop-meets-k9s): cluster map,
// services list, status, alerts, and an AI chat sidebar. Panes register
// themselves; the shell lays them out and polls /v1/state.
package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Rish3666/ServeEz/internal/tui"
)

func main() {
	url := flag.String("url", "", "control plane URL (default http://localhost:8443)")
	flag.Parse()

	p := tea.NewProgram(tui.NewApp(*url), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "servez-tui:", err)
		os.Exit(1)
	}
}
