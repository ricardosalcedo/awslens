package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	awsclient "github.com/awslens/awslens/internal/aws"
)

type pickerModel struct {
	profiles  []awsclient.Profile
	cursor    int
	chosen    *awsclient.Profile
	createNew bool
}

func (m pickerModel) Init() tea.Cmd { return nil }

func (m pickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	maxIdx := len(m.profiles) // last index is "create new"
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < maxIdx {
				m.cursor++
			}
		case "enter", " ":
			if m.cursor == len(m.profiles) {
				m.createNew = true
				return m, tea.Quit
			}
			m.chosen = &m.profiles[m.cursor]
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m pickerModel) View() string {
	var b strings.Builder

	banner := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#000")).
		Background(orange).
		Padding(1, 3).
		Render("⬡  AWS Lens")
	b.WriteString(banner + "\n\n")
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#fff")).Bold(true).
		Render("  Browse, inspect, and manage your AWS resources from the terminal.") + "\n")
	b.WriteString(mutedStyle.Render("  AI insights • security audit • cost analysis • export • multi-account") + "\n\n")
	b.WriteString(lipgloss.NewStyle().Foreground(cyan).Bold(true).Render("  Choose an AWS Profile") + "\n")
	b.WriteString(mutedStyle.Render("  Profiles are loaded from ~/.aws/config and ~/.aws/credentials") + "\n\n")

	lastRegion := ""
	for i, p := range m.profiles {
		reg := p.Region
		if reg == "" {
			reg = "no region"
		}
		if reg != lastRegion {
			if lastRegion != "" {
				b.WriteString("\n")
			}
			b.WriteString("  " + lipgloss.NewStyle().Foreground(orange).Bold(true).Render("── "+reg+" ──") + "\n")
			lastRegion = reg
		}
		prefix := "  "
		name := fmt.Sprintf("%-20s", p.Name)
		meta := profileMeta(p)

		if i == m.cursor {
			prefix = "▶ "
			name = selectedRow.Render(fmt.Sprintf("%-20s", p.Name))
		}
		b.WriteString(prefix + name + "  " + mutedStyle.Render(meta) + "\n")
	}

	// "create new" option
	createLabel := "➕ Create new profile"
	if m.cursor == len(m.profiles) {
		b.WriteString("▶ " + selectedRow.Render(createLabel) + "\n")
	} else {
		b.WriteString("  " + lipgloss.NewStyle().Foreground(cyan).Render(createLabel) + "\n")
	}

	b.WriteString(helpStyle.Render("\n↑/↓ navigate • enter select • q quit"))
	return b.String()
}

func profileMeta(p awsclient.Profile) string {
	var parts []string
	if p.RoleARN != "" {
		role := p.RoleARN
		if idx := strings.LastIndex(role, "/"); idx >= 0 {
			role = role[idx+1:]
		}
		parts = append(parts, lipgloss.NewStyle().Foreground(orange).Render("role: "+role))
	}
	if p.SSORole != "" {
		parts = append(parts, lipgloss.NewStyle().Foreground(orange).Render("sso: "+p.SSORole))
	}
	if p.SourceProfile != "" {
		parts = append(parts, "via: "+p.SourceProfile)
	}
	if p.Region != "" {
		parts = append(parts, "region: "+p.Region)
	}
	if len(parts) == 0 {
		parts = append(parts, mutedStyle.Render("static credentials  (IAM user)"))
	}
	return strings.Join(parts, mutedStyle.Render("  │  "))
}

// RunPicker shows the profile picker and returns the chosen profile name and region.
func RunPicker() (profile, region string, err error) {
	for {
		profiles := awsclient.LoadProfiles()

		m := pickerModel{profiles: profiles}
		result, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
		if err != nil {
			return "", "", err
		}

		pm := result.(pickerModel)

		// user chose "create new"
		if pm.createNew {
			wiz := newWizard()
			wizResult, err := tea.NewProgram(wiz, tea.WithAltScreen()).Run()
			if err != nil {
				return "", "", err
			}
			wm := wizResult.(wizardModel)
			if wm.cancel {
				continue // back to picker
			}
			if wm.saved {
				continue // reload profiles and pick again
			}
			continue
		}

		if pm.chosen == nil {
			return "", "", fmt.Errorf("no profile selected")
		}
		return pm.chosen.Name, pm.chosen.Region, nil
	}
}
