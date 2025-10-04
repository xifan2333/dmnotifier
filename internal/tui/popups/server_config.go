package popups

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	tuimsg "github.com/xifan2333/dmnotifier/internal/common"
	"github.com/xifan2333/dmnotifier/internal/tui/components"
)

type ServerConfigModel struct {
	visible      bool
	inputs       [3]components.FormInputModel
	focusedIndex int
	width        int
	height       int
}

func NewServerConfig() ServerConfigModel {
	return ServerConfigModel{
		visible: false,
		inputs: [3]components.FormInputModel{
			components.NewFormInput("API Address", "https://example.com", 200),
			components.NewPasswordInput("API Token", "your-token", 200),
			components.NewFormInput("WS Address", "ws://example.com:7777", 200),
		},
		focusedIndex: 0,
	}
}

func (m *ServerConfigModel) SetConfig(apiAddress, apiToken, wsAddress string) {
	m.inputs[0].SetValue(apiAddress)
	m.inputs[1].SetValue(apiToken)
	m.inputs[2].SetValue(wsAddress)
}

func (m ServerConfigModel) Init() tea.Cmd {
	return nil
}

func (m ServerConfigModel) Update(msg tea.Msg) (ServerConfigModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tuimsg.ShowServerConfigPopupMsg:
		m.visible = true
		m.focusedIndex = 0
		m.inputs[0].Focus()
		return m, nil

	case tuimsg.HidePopupMsg:
		m.visible = false
		for i := range m.inputs {
			m.inputs[i].Blur()
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if !m.visible {
			return m, nil
		}

		// 如果正在编辑
		if m.inputs[m.focusedIndex].IsEditing {
			switch msg.String() {
			case "esc":
				m.inputs[m.focusedIndex].StopEdit()
				return m, func() tea.Msg {
					return tuimsg.StatusMsg{Message: "Cancelled"}
				}

			case "enter":
				m.inputs[m.focusedIndex].StopEdit()
				// 自动触发配置保存
				return m, func() tea.Msg {
					return tuimsg.UpdateServerConfigMsg{
						APIAddress: m.inputs[0].Value(),
						APIToken:   m.inputs[1].Value(),
						WSAddress:  m.inputs[2].Value(),
					}
				}

			default:
				var cmd tea.Cmd
				m.inputs[m.focusedIndex], cmd = m.inputs[m.focusedIndex].Update(msg)
				return m, cmd
			}
		}

		// 非编辑模式
		switch msg.String() {
		case "up", "k":
			m.inputs[m.focusedIndex].Blur()
			if m.focusedIndex > 0 {
				m.focusedIndex--
			}
			m.inputs[m.focusedIndex].Focus()

		case "down", "j", "tab":
			m.inputs[m.focusedIndex].Blur()
			if m.focusedIndex < len(m.inputs)-1 {
				m.focusedIndex++
			}
			m.inputs[m.focusedIndex].Focus()

		case "enter":
			m.inputs[m.focusedIndex].StartEdit()
		}
	}

	return m, nil
}

func (m ServerConfigModel) View() string {
	if !m.visible {
		return ""
	}

	width := 70
	if m.width > 0 && m.width < 70 {
		width = m.width - 10
	}

	primaryColor := lipgloss.Color("#7D56F4")
	dimColor := lipgloss.Color("#666666")

	popupStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(primaryColor).
		Padding(1, 2)

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(primaryColor).
		Padding(0, 1)

	dimStyle := lipgloss.NewStyle().
		Foreground(dimColor)

	header := headerStyle.Width(width - 4).Render("Server Config")

	var content string
	for _, input := range m.inputs {
		content += input.View() + "\n"
	}

	help := dimStyle.Render("Up/Down: Navigate | Enter: Edit/Save | Esc: Cancel/Close")

	body := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		"",
		content,
		"",
		help,
	)

	return popupStyle.
		Width(width).
		Render(body)
}

func (m ServerConfigModel) IsVisible() bool {
	return m.visible
}
