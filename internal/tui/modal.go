package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ── modal types ───────────────────────────────────────────────────────────────

type modalKind int

const (
	modalNone modalKind = iota
	modalConfirm
	modalInput
	modalInsight
	modalLoading
)

type modal struct {
	kind    modalKind
	title   string
	body    string   // confirm: description; input: field label
	input   string   // current text for input modal
	onOK    func()   // called when confirmed / submitted
}

func (m *modal) reset() { *m = modal{} }

func (m modal) active() bool { return m.kind != modalNone }

var (
	modalBox = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#FF9900")).
		Padding(1, 3)
	modalTitle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF9900"))
	modalHelp  = lipgloss.NewStyle().Foreground(lipgloss.Color("#555555"))
)

func (m modal) view(width int, extra ...string) string {
	if !m.active() {
		return ""
	}
	var b strings.Builder
	b.WriteString(modalTitle.Render(m.title) + "\n\n")
	b.WriteString(m.body)
	if m.kind == modalLoading && len(extra) > 0 {
		b.WriteString(extra[0])
	}
	b.WriteString("\n")
	switch m.kind {
	case modalInput:
		b.WriteString("\n> " + m.input + "█\n")
		b.WriteString("\n" + modalHelp.Render("enter submit • esc cancel"))
	case modalInsight:
		b.WriteString("\n" + modalHelp.Render("esc close"))
	case modalLoading:
		// no help, just the spinner in body
	default:
		b.WriteString("\n" + modalHelp.Render("y confirm • esc cancel"))
	}
	box := modalBox.Render(b.String())
	// center horizontally
	boxW := lipgloss.Width(box)
	pad := (width - boxW) / 2
	if pad < 0 {
		pad = 0
	}
	lines := strings.Split(box, "\n")
	for i, l := range lines {
		lines[i] = strings.Repeat(" ", pad) + l
	}
	return strings.Join(lines, "\n")
}
