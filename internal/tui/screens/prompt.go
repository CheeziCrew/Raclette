package screens

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/CheeziCrew/curd"
	"github.com/CheeziCrew/raclette/internal/maven"
)

var (
	promptColorBrYlow = lipgloss.Color("11")

	promptTitle = lipgloss.NewStyle().Bold(true).Foreground(promptColorBrYlow).MarginBottom(1)
	promptLabel = lipgloss.NewStyle().Bold(true).Foreground(promptColorBrYlow)
)

// PromptModel collects text inputs for commands that need user-provided values.
type PromptModel struct {
	command maven.Command
	inputs  []textinput.Model
	cursor  int
	width   int
	height  int
}

// NewPrompt creates a prompt screen for the given command's prompts.
func NewPrompt(cmd maven.Command) PromptModel {
	var inputs []textinput.Model
	for i, p := range cmd.Prompts {
		ti := textinput.New()
		ti.Placeholder = p.Placeholder
		ti.CharLimit = 120
		ti.SetWidth(50)
		if i == 0 {
			ti.Focus()
		}
		inputs = append(inputs, ti)
	}

	return PromptModel{
		command: cmd,
		inputs:  inputs,
	}
}

func (m PromptModel) Init() tea.Cmd {
	if len(m.inputs) > 0 {
		return m.inputs[0].Focus()
	}
	return nil
}

func (m PromptModel) Update(msg tea.Msg) (PromptModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("tab", "down"))):
			if m.cursor < len(m.inputs)-1 {
				m.inputs[m.cursor].Blur()
				m.cursor++
				return m, m.inputs[m.cursor].Focus()
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("shift+tab", "up"))):
			if m.cursor > 0 {
				m.inputs[m.cursor].Blur()
				m.cursor--
				return m, m.inputs[m.cursor].Focus()
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			if m.cursor == len(m.inputs)-1 || len(m.inputs) == 0 {
				return m, m.submit()
			}
			m.inputs[m.cursor].Blur()
			m.cursor++
			return m, m.inputs[m.cursor].Focus()
		case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
			return m, func() tea.Msg { return BackToMenuMsg{} }
		}
	}

	// Update the focused input
	if m.cursor < len(m.inputs) {
		var cmd tea.Cmd
		m.inputs[m.cursor], cmd = m.inputs[m.cursor].Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m PromptModel) submit() tea.Cmd {
	inputs := make(map[string]string)
	for i, p := range m.command.Prompts {
		inputs[p.Key] = strings.TrimSpace(m.inputs[i].Value())
	}
	return func() tea.Msg {
		return PromptDoneMsg{Inputs: inputs}
	}
}

func (m PromptModel) View() string {
	var b strings.Builder

	b.WriteString(promptTitle.Render(fmt.Sprintf("%s %s", m.command.Icon, m.command.Name)))
	b.WriteString("\n\n")

	for i, p := range m.command.Prompts {
		label := promptLabel.Render(p.Label)
		if i == m.cursor {
			label = "▸ " + label
		} else {
			label = "  " + label
		}
		b.WriteString(label)
		b.WriteString("\n  ")
		b.WriteString(m.inputs[i].View())
		b.WriteString("\n\n")
	}

	st := curd.RaclettePalette.Styles()
	b.WriteString(curd.RenderHintBar(st, []curd.Hint{
		{Key: "tab/↓", Desc: "next"},
		{Key: "shift+tab/↑", Desc: "prev"},
		{Key: "enter", Desc: "submit"},
		{Key: "esc", Desc: "back"},
	}))

	return b.String()
}

// Inputs returns the collected input values.
func (m PromptModel) Inputs() map[string]string {
	inputs := make(map[string]string)
	for i, p := range m.command.Prompts {
		inputs[p.Key] = strings.TrimSpace(m.inputs[i].Value())
	}
	return inputs
}
