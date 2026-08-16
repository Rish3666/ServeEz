package tui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Rish3666/ServeEz/internal/api"
)

func init() {
	RegisterPanel("Chat", func(env Env) Panel { return &chatPane{env: env} })
}

type chatPane struct {
	env Env

	snap    *Snapshot
	focused bool
	history []chatEntry

	input     []rune
	cursor    int
	historyOn bool
	scroll    int

	pending *pendingAction
}

type chatEntry struct {
	Kind string
	Text string
	At   time.Time
}

type pendingAction struct {
	act  api.Action
	kind string
}

type chatResultMsg struct {
	lines     []string
	pending   *pendingAction
	clearPend bool
}

func (c *chatPane) Init() tea.Cmd { return nil }

func (c *chatPane) Title() string { return "Chat" }

func (c *chatPane) SetSnapshot(s *Snapshot) { c.snap = s }

func (c *chatPane) Focused() bool     { return c.focused }
func (c *chatPane) SetFocused(f bool) { c.focused = f }

func (c *chatPane) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case tea.KeyMsg:
		if !c.focused {
			return c, nil
		}
		return c.handleKey(m)
	case chatResultMsg:
		for _, line := range m.lines {
			c.push("system", line)
		}
		if m.clearPend {
			c.pending = nil
		}
		if m.pending != nil {
			c.pending = m.pending
		}
		return c, nil
	default:
		return c, nil
	}
}

func (c *chatPane) View() string {
	var b strings.Builder
	if c.snap == nil {
		b.WriteString(styleHeader.Render("awaiting first poll…"))
		b.WriteString("\n")
	} else {
		b.WriteString(c.snapshotSummary())
		b.WriteString("\n")
	}

	if len(c.history) == 0 {
		b.WriteString(styleHelp.Render("type `help` for commands"))
		b.WriteString("\n")
	} else {
		for _, entry := range c.visibleHistory() {
			b.WriteString(c.renderEntry(entry))
			b.WriteString("\n")
		}
	}

	b.WriteString(c.renderInput())
	return strings.TrimRight(b.String(), "\n")
}

func (c *chatPane) handleKey(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch k.Type {
	case tea.KeyEsc:
		c.historyOn = !c.historyOn
		return c, nil
	case tea.KeyEnter:
		text := strings.TrimSpace(string(c.input))
		if text == "" {
			return c, nil
		}
		c.push("user", text)
		c.input = nil
		c.cursor = 0
		c.historyOn = false
		c.scroll = 0
		return c, c.submit(text)
	case tea.KeyBackspace:
		if c.cursor > 0 {
			c.input = deleteRune(c.input, c.cursor-1)
			c.cursor--
		}
	case tea.KeyDelete:
		if c.cursor < len(c.input) {
			c.input = deleteRune(c.input, c.cursor)
		}
	case tea.KeyLeft:
		if !c.historyOn && c.cursor > 0 {
			c.cursor--
		}
	case tea.KeyRight:
		if !c.historyOn && c.cursor < len(c.input) {
			c.cursor++
		}
	case tea.KeyHome:
		if !c.historyOn {
			c.cursor = 0
		}
	case tea.KeyEnd:
		if !c.historyOn {
			c.cursor = len(c.input)
		}
	case tea.KeyUp:
		if c.historyOn {
			if c.scroll < max(0, len(c.history)-1) {
				c.scroll++
			}
		} else {
			c.historyOn = true
		}
	case tea.KeyDown:
		if c.historyOn {
			if c.scroll > 0 {
				c.scroll--
			} else {
				c.historyOn = false
			}
		}
	default:
		if !c.historyOn && k.Type == tea.KeyRunes && len(k.Runes) > 0 {
			c.input = insertRunes(c.input, c.cursor, k.Runes)
			c.cursor += len(k.Runes)
		}
	}
	return c, nil
}

func (c *chatPane) submit(text string) tea.Cmd {
	return func() tea.Msg {
		lines, pending, clearPend := c.runCommand(text)
		return chatResultMsg{lines: lines, pending: pending, clearPend: clearPend}
	}
}

func (c *chatPane) runCommand(text string) ([]string, *pendingAction, bool) {
	if text == "" {
		return nil, nil, false
	}
	if isConfirm(text) {
		if c.pending == nil {
			return []string{"no pending action to confirm"}, nil, false
		}
		return c.executePending(c.pending)
	}

	if cmd, ok := parseCommand(text); ok {
		switch cmd.kind {
		case "help":
			return []string{helpText()}, nil, false
		case "status":
			return []string{c.snapshotSummary()}, nil, false
		case "deploy":
			return c.runDeploy(cmd)
		default:
			return c.runAction(cmd)
		}
	}
	return []string{helpText()}, nil, false
}

func (c *chatPane) runDeploy(cmd commandSpec) ([]string, *pendingAction, bool) {
	if c.env.Client == nil {
		return []string{"control client unavailable"}, nil, false
	}
	spec := &api.WorkloadSpec{
		Image:         cmd.image,
		Replicas:      cmd.replicas,
		Type:          "service",
		RestartPolicy: "always",
		Strategy:      "rolling",
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := c.env.Client.Deploy(ctx, cmd.name, spec); err != nil {
		return []string{"deploy failed: " + err.Error()}, nil, false
	}
	return []string{fmt.Sprintf("deployed %s with image %s", cmd.name, cmd.image)}, nil, false
}

func (c *chatPane) runAction(cmd commandSpec) ([]string, *pendingAction, bool) {
	if c.env.Client == nil {
		return []string{"control client unavailable"}, nil, false
	}
	act := api.Action{
		Type:       cmd.kind,
		Target:     cmd.target,
		Confidence: 0.9,
		Initiator:  "human:tui",
		Parameters: map[string]any{},
	}
	if cmd.kind == "scale" {
		act.Target = "workload:" + cmd.target
		act.Parameters["replicas"] = cmd.replicas
	}
	if cmd.kind == "kill" {
		act.Target = "cluster"
	}
	if cmd.kind == "migrate" && act.Target == "" {
		act.Target = cmd.target
	}

	lines := []string{}
	if isGatedAction(cmd.kind) {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		sim, err := c.env.Client.Simulate(ctx, act)
		if err != nil {
			return []string{"simulate failed: " + err.Error()}, nil, false
		}
		lines = append(lines, "simulated: "+sim.Recommendation)
		if sim.Recommendation == "requires_approval" {
			lines = append(lines, "type yes or confirm to execute")
			return lines, &pendingAction{act: act, kind: cmd.kind}, false
		}
	}
	extra, err := c.executeAction(act)
	if err != nil {
		return append(lines, "execute failed: "+err.Error()), nil, false
	}
	lines = append(lines, extra...)
	return lines, nil, true
}

func (c *chatPane) executePending(p *pendingAction) ([]string, *pendingAction, bool) {
	if p == nil {
		return []string{"no pending action to confirm"}, nil, false
	}
	extra, err := c.executeAction(p.act)
	if err != nil {
		return []string{"execute failed: " + err.Error()}, nil, false
	}
	return append([]string{"confirmed"}, extra...), nil, true
}

func (c *chatPane) executeAction(act api.Action) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	out, err := c.env.Client.Execute(ctx, act)
	if err != nil {
		return nil, err
	}
	if out == nil {
		return []string{"action completed"}, nil
	}
	lines := []string{}
	if out.Status != "" {
		lines = append(lines, "status: "+out.Status)
	}
	if out.Message != "" {
		lines = append(lines, out.Message)
	}
	if len(lines) == 0 {
		lines = append(lines, "action completed")
	}
	return lines, nil
}

func (c *chatPane) push(kind, text string) {
	if text == "" {
		return
	}
	if len(c.history) >= 100 {
		copy(c.history, c.history[1:])
		c.history = c.history[:99]
	}
	c.history = append(c.history, chatEntry{Kind: kind, Text: text, At: time.Now()})
}

func (c *chatPane) visibleHistory() []chatEntry {
	if len(c.history) == 0 {
		return nil
	}
	maxLines := 10
	end := len(c.history) - c.scroll
	if end < 0 {
		end = 0
	}
	start := end - maxLines
	if start < 0 {
		start = 0
	}
	out := make([]chatEntry, 0, end-start)
	for i := start; i < end; i++ {
		out = append(out, c.history[i])
	}
	return out
}

func (c *chatPane) renderEntry(e chatEntry) string {
	prefix := styleHelp.Render("chat")
	switch e.Kind {
	case "user":
		prefix = styleGood.Render("you")
	case "system":
		prefix = styleWarn.Render("srv")
	}
	return fmt.Sprintf("%s %s", prefix, e.Text)
}

func (c *chatPane) renderInput() string {
	cursor := " "
	if c.focused {
		cursor = styleTitle.Render("█")
	}
	if c.cursor < 0 {
		c.cursor = 0
	}
	if c.cursor > len(c.input) {
		c.cursor = len(c.input)
	}
	left := string(c.input[:c.cursor])
	right := string(c.input[c.cursor:])
	mode := "input"
	if c.historyOn {
		mode = "history"
	}
	return styleHeader.Render(fmt.Sprintf("%s> %s%s%s", mode, left, cursor, right))
}

func (c *chatPane) snapshotSummary() string {
	if c.snap == nil {
		return styleHeader.Render("no snapshot yet")
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("%d nodes, %d workloads", len(c.snap.Nodes), len(c.snap.Workloads)))
	if c.snap.Err != nil {
		b.WriteString(" | ")
		b.WriteString(styleBad.Render(c.snap.Err.Error()))
	}
	if len(c.snap.Nodes) > 0 {
		n := c.snap.Nodes[0]
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("top node: %s [%s]", n.Name, n.State))
	}
	if len(c.snap.Workloads) > 0 {
		w := c.snap.Workloads[0]
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("top workload: %s -> %s", w.Name, w.Node))
	}
	return b.String()
}

func helpText() string {
	return strings.Join([]string{
		"commands:",
		"  status",
		"  help",
		"  deploy <name> image=<img> [replicas=<n>]",
		"  scale <workload> <n>",
		"  restart <target>",
		"  stop <target>",
		"  remove <target>",
		"  migrate <target>",
		"  kill",
	}, "\n")
}

type commandSpec struct {
	kind     string
	name     string
	target   string
	image    string
	replicas int
}

func parseCommand(text string) (commandSpec, bool) {
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return commandSpec{}, false
	}
	switch strings.ToLower(fields[0]) {
	case "help":
		return commandSpec{kind: "help"}, true
	case "status":
		return commandSpec{kind: "status"}, true
	case "kill":
		return commandSpec{kind: "kill"}, true
	case "restart", "stop", "remove", "migrate":
		if len(fields) < 2 {
			return commandSpec{}, false
		}
		return commandSpec{kind: strings.ToLower(fields[0]), target: fields[1]}, true
	case "scale":
		if len(fields) < 3 {
			return commandSpec{}, false
		}
		n, err := strconv.Atoi(fields[2])
		if err != nil {
			return commandSpec{}, false
		}
		return commandSpec{kind: "scale", target: fields[1], replicas: n}, true
	case "deploy":
		if len(fields) < 3 {
			return commandSpec{}, false
		}
		spec := commandSpec{kind: "deploy", name: fields[1], replicas: 1}
		for _, field := range fields[2:] {
			key, val, ok := strings.Cut(field, "=")
			if !ok {
				continue
			}
			switch strings.ToLower(key) {
			case "image":
				spec.image = val
			case "replicas":
				if n, err := strconv.Atoi(val); err == nil && n > 0 {
					spec.replicas = n
				}
			}
		}
		if spec.image == "" {
			return commandSpec{}, false
		}
		return spec, true
	default:
		return commandSpec{}, false
	}
}

func isConfirm(text string) bool {
	switch strings.ToLower(strings.TrimSpace(text)) {
	case "yes", "confirm":
		return true
	default:
		return false
	}
}

func isGatedAction(kind string) bool {
	switch kind {
	case "kill", "stop", "remove", "restart", "migrate", "scale":
		return true
	default:
		return false
	}
}

func deleteRune(rs []rune, idx int) []rune {
	if idx < 0 || idx >= len(rs) {
		return rs
	}
	out := append([]rune(nil), rs[:idx]...)
	return append(out, rs[idx+1:]...)
}

func insertRunes(rs []rune, idx int, add []rune) []rune {
	if idx < 0 {
		idx = 0
	}
	if idx > len(rs) {
		idx = len(rs)
	}
	out := append([]rune(nil), rs[:idx]...)
	out = append(out, add...)
	out = append(out, rs[idx:]...)
	return out
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

var _ Panel = (*chatPane)(nil)
var _ = lipgloss.NewStyle
