package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	awsclient "github.com/awslens/awslens/internal/aws"
)

// wizard steps
const (
	wzName = iota
	wzAuthType
	wzAccessKey
	wzSecretKey
	wzRoleARN
	wzSourceProfile
	wzSSOStartURL
	wzSSORegion
	wzSSOAccount
	wzSSORole
	wzRegion
	wzConfirm
	wzDone
)

var authTypes = []string{"Access Keys (IAM user)", "Assume Role", "SSO"}

type wizardModel struct {
	step   int
	cursor int // for auth type selection
	input  string

	// collected values
	name          string
	authType      int // 0=keys, 1=role, 2=sso
	accessKey     string
	secretKey     string
	roleARN       string
	sourceProfile string
	ssoStartURL   string
	ssoRegion     string
	ssoAccount    string
	ssoRole       string
	region        string

	err    string
	saved  bool
	cancel bool
}

func newWizard() wizardModel { return wizardModel{step: wzName} }

func (w wizardModel) Init() tea.Cmd { return nil }

func (w wizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return w, tea.Quit
		case "esc":
			w.cancel = true
			return w, tea.Quit
		}

		// auth type selection step uses arrow keys
		if w.step == wzAuthType {
			switch msg.String() {
			case "up", "k":
				if w.cursor > 0 {
					w.cursor--
				}
				return w, nil
			case "down", "j":
				if w.cursor < len(authTypes)-1 {
					w.cursor++
				}
				return w, nil
			case "enter":
				w.authType = w.cursor
				switch w.authType {
				case 0:
					w.step = wzAccessKey
				case 1:
					w.step = wzRoleARN
				case 2:
					w.step = wzSSOStartURL
				}
				return w, nil
			}
			return w, nil
		}

		// confirm step
		if w.step == wzConfirm {
			switch msg.String() {
			case "enter", "y", "Y":
				err := awsclient.SaveProfile(
					w.name, w.region,
					w.accessKey, w.secretKey,
					w.roleARN, w.sourceProfile,
					w.ssoStartURL, w.ssoRegion, w.ssoAccount, w.ssoRole,
				)
				if err != nil {
					w.err = err.Error()
				} else {
					w.saved = true
				}
				w.step = wzDone
				return w, nil
			case "n", "N":
				w.cancel = true
				return w, tea.Quit
			}
			return w, nil
		}

		// done step
		if w.step == wzDone {
			if msg.String() == "enter" {
				return w, tea.Quit
			}
			return w, nil
		}

		// text input steps
		switch msg.String() {
		case "enter":
			w.err = ""
			w = w.submitField()
			return w, nil
		case "backspace":
			if len(w.input) > 0 {
				w.input = w.input[:len(w.input)-1]
			}
			return w, nil
		default:
			if len(msg.String()) == 1 {
				w.input += msg.String()
			}
			return w, nil
		}
	}
	return w, nil
}

func (w wizardModel) submitField() wizardModel {
	val := strings.TrimSpace(w.input)
	w.input = ""

	switch w.step {
	case wzName:
		if val == "" {
			w.err = "Profile name is required"
			return w
		}
		w.name = val
		w.step = wzAuthType
	case wzAccessKey:
		if val == "" {
			w.err = "Access key is required"
			return w
		}
		w.accessKey = val
		w.step = wzSecretKey
	case wzSecretKey:
		if val == "" {
			w.err = "Secret key is required"
			return w
		}
		w.secretKey = val
		w.step = wzRegion
	case wzRoleARN:
		if val == "" || !strings.HasPrefix(val, "arn:aws:iam::") && !strings.Contains(val, ":role/") {
			// be lenient, just check non-empty
			if val == "" {
				w.err = "Role ARN is required (arn:aws:iam::<account>:role/<name>)"
				return w
			}
		}
		w.roleARN = val
		w.step = wzSourceProfile
	case wzSourceProfile:
		if val == "" {
			val = "default"
		}
		w.sourceProfile = val
		w.step = wzRegion
	case wzSSOStartURL:
		if val == "" {
			w.err = "SSO start URL is required"
			return w
		}
		w.ssoStartURL = val
		w.step = wzSSORegion
	case wzSSORegion:
		if val == "" {
			val = "us-east-1"
		}
		w.ssoRegion = val
		w.step = wzSSOAccount
	case wzSSOAccount:
		if val == "" {
			w.err = "SSO account ID is required"
			return w
		}
		w.ssoAccount = val
		w.step = wzSSORole
	case wzSSORole:
		if val == "" {
			w.err = "SSO role name is required"
			return w
		}
		w.ssoRole = val
		w.step = wzRegion
	case wzRegion:
		if val == "" {
			val = "us-east-1"
		}
		w.region = val
		w.step = wzConfirm
	}
	return w
}

func (w wizardModel) View() string {
	var b strings.Builder

	title := lipgloss.NewStyle().Bold(true).Foreground(cyan).Render("  Create New AWS Profile")
	b.WriteString(titleStyle.Render("⬡ awslens") + "\n\n" + title + "\n")
	b.WriteString(mutedStyle.Render("  Press esc to cancel at any time") + "\n\n")

	// show completed fields
	if w.name != "" {
		b.WriteString(doneField("Profile name", w.name))
	}
	if w.step > wzAuthType {
		b.WriteString(doneField("Auth type", authTypes[w.authType]))
	}
	if w.accessKey != "" {
		b.WriteString(doneField("Access Key", maskKey(w.accessKey)))
	}
	if w.secretKey != "" {
		b.WriteString(doneField("Secret Key", "••••••••"))
	}
	if w.roleARN != "" {
		b.WriteString(doneField("Role ARN", w.roleARN))
	}
	if w.sourceProfile != "" && w.step > wzSourceProfile {
		b.WriteString(doneField("Source profile", w.sourceProfile))
	}
	if w.ssoStartURL != "" {
		b.WriteString(doneField("SSO Start URL", w.ssoStartURL))
	}
	if w.ssoRegion != "" && w.step > wzSSORegion {
		b.WriteString(doneField("SSO Region", w.ssoRegion))
	}
	if w.ssoAccount != "" {
		b.WriteString(doneField("SSO Account", w.ssoAccount))
	}
	if w.ssoRole != "" {
		b.WriteString(doneField("SSO Role", w.ssoRole))
	}
	if w.region != "" && w.step > wzRegion {
		b.WriteString(doneField("Region", w.region))
	}

	b.WriteString("\n")

	// current prompt
	switch w.step {
	case wzName:
		b.WriteString(prompt("Profile name", "", w.input))
	case wzAuthType:
		b.WriteString("  " + lipgloss.NewStyle().Foreground(yellow).Bold(true).Render("Authentication type:") + "\n\n")
		for i, at := range authTypes {
			prefix := "    "
			if i == w.cursor {
				prefix = "  ▶ "
				b.WriteString(prefix + selectedRow.Render(at) + "\n")
			} else {
				b.WriteString(prefix + at + "\n")
			}
		}
	case wzAccessKey:
		b.WriteString(prompt("AWS Access Key ID", "", w.input))
	case wzSecretKey:
		b.WriteString(prompt("AWS Secret Access Key", "", strings.Repeat("•", len(w.input))))
	case wzRoleARN:
		b.WriteString(prompt("Role ARN", "arn:aws:iam::<account>:role/<name>", w.input))
	case wzSourceProfile:
		b.WriteString(prompt("Source profile", "default", w.input))
	case wzSSOStartURL:
		b.WriteString(prompt("SSO Start URL", "https://my-org.awsapps.com/start", w.input))
	case wzSSORegion:
		b.WriteString(prompt("SSO Region", "us-east-1", w.input))
	case wzSSOAccount:
		b.WriteString(prompt("SSO Account ID", "123456789012", w.input))
	case wzSSORole:
		b.WriteString(prompt("SSO Role Name", "AdministratorAccess", w.input))
	case wzRegion:
		b.WriteString(prompt("Default region", "us-east-1", w.input))
	case wzConfirm:
		b.WriteString("  " + lipgloss.NewStyle().Foreground(yellow).Bold(true).Render("Save this profile? (y/n)") + "\n")
	case wzDone:
		if w.err != "" {
			b.WriteString("  " + warnStyle.Render("Error: "+w.err) + "\n")
		} else {
			b.WriteString("  " + lipgloss.NewStyle().Foreground(green).Bold(true).Render(
				fmt.Sprintf("✓ Profile '%s' saved! Press enter to continue.", w.name)) + "\n")
		}
	}

	if w.err != "" && w.step != wzDone {
		b.WriteString("\n  " + warnStyle.Render(w.err) + "\n")
	}

	return b.String()
}

func prompt(label, placeholder, value string) string {
	p := "  " + lipgloss.NewStyle().Foreground(yellow).Bold(true).Render(label+": ")
	if placeholder != "" && value == "" {
		p += mutedStyle.Render(placeholder)
	}
	p += value + "█\n"
	return p
}

func doneField(label, value string) string {
	return "  " + lipgloss.NewStyle().Foreground(green).Render("✓ ") +
		mutedStyle.Render(label+": ") + value + "\n"
}

func maskKey(key string) string {
	if len(key) <= 4 {
		return "••••"
	}
	return strings.Repeat("•", len(key)-4) + key[len(key)-4:]
}
