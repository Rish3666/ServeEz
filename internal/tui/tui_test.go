package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestStrFltIntAccessors(t *testing.T) {
	if strOf("x") != "x" || strOf(3) != "" {
		t.Fatal("strOf failed")
	}
	if fltOf(1.5) != 1.5 || fltOf(3) != 3 {
		t.Fatal("fltOf failed")
	}
	if intOf(2.9) != 2 || intOf(7) != 7 {
		t.Fatal("intOf failed")
	}
	if u64Of(1<<30) != 1<<30 {
		t.Fatal("u64Of failed")
	}
}

func TestPanelsRegistered(t *testing.T) {
	if len(panelFactories) < 3 {
		t.Fatalf("expected at least 3 panels, got %d", len(panelFactories))
	}
	env := Env{}
	panels := buildPanels(env)
	for _, p := range panels {
		if p.Title() == "" {
			t.Fatal("panel with empty title")
		}
		if p.View() == "" {
			t.Fatal("panel with empty view")
		}
	}
}

func TestClusterPaneSelection(t *testing.T) {
	c := &clusterPane{}
	c.snap = &Snapshot{
		Nodes: []NodeRow{{Name: "a"}, {Name: "b"}},
	}
	// Selection shouldn't move when not focused.
	c.SetFocused(false)
	upd, _ := c.Update(tea.KeyMsg{Type: tea.KeyDown})
	if upd.(*clusterPane).selected != 0 {
		t.Fatal("selection moved while unfocused")
	}
	c.SetFocused(true)
	upd, _ = c.Update(tea.KeyMsg{Type: tea.KeyDown})
	if upd.(*clusterPane).selected != 1 {
		t.Fatal("selection should move down")
	}
}
