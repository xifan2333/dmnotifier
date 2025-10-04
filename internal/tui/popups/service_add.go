package popups

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	tuimsg "github.com/xifan/dmnotifier/internal/common"
	"github.com/xifan/dmnotifier/internal/tui/components"
)

type AddServiceModel struct {
	visible        bool
	platforms      []string
	platformCursor int
	inputs         [2]components.FormInputModel // RID, Cookie
	step           int                          // 0=platform, 1=RID, 2=Cookie
	width          int
	height         int
}

func NewAddService() AddServiceModel {
	return AddServiceModel{
		visible:        false,
		platforms:      []string{"bilibili", "douyin", "kuaishou", "douyu", "huya"},
		platformCursor: 0,
		inputs: [2]components.FormInputModel{
			components.NewFormInput("Room ID", "", 50),
			components.NewFormInput("Cookie (optional)", "", 0),
		},
		step: 0,
	}
}

func (m AddServiceModel) Init() tea.Cmd {
	return nil
}

func (m AddServiceModel) Update(msg tea.Msg) (AddServiceModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tuimsg.ShowAddServicePopupMsg:
		m.visible = true
		m.step = 0
		m.platformCursor = 0
		m.inputs[0].SetValue("")
		m.inputs[1].SetValue("")
		return m, nil

	case tuimsg.HidePopupMsg:
		m.visible = false
		m.step = 0
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

		switch m.step {
		case 0: // 平台选择
			switch msg.String() {
			case "up", "k":
				if m.platformCursor > 0 {
					m.platformCursor--
				}
			case "down", "j":
				if m.platformCursor < len(m.platforms)-1 {
					m.platformCursor++
				}
			case "enter":
				// 进入 RID 输入
				m.step = 1
				m.inputs[0].Focus()
				m.inputs[0].StartEdit()
			case "esc":
				return m, func() tea.Msg { return tuimsg.HidePopupMsg{} }
			}

		case 1: // RID 输入
			switch msg.String() {
			case "esc":
				// 返回平台选择
				m.step = 0
				m.inputs[0].StopEdit()
				m.inputs[0].Blur()
				return m, nil
			case "enter":
				rid := m.inputs[0].Value()
				if rid == "" {
					return m, func() tea.Msg {
						return tuimsg.StatusMsg{Message: "Room ID is required"}
					}
				}
				// 进入 Cookie 输入
				m.step = 2
				m.inputs[0].StopEdit()
				m.inputs[0].Blur()
				m.inputs[1].Focus()
				m.inputs[1].StartEdit()
			default:
				var cmd tea.Cmd
				m.inputs[0], cmd = m.inputs[0].Update(msg)
				return m, cmd
			}

		case 2: // Cookie 输入
			switch msg.String() {
			case "esc":
				// 返回 RID 输入
				m.step = 1
				m.inputs[1].StopEdit()
				m.inputs[1].Blur()
				m.inputs[0].Focus()
				m.inputs[0].StartEdit()
				return m, nil
			case "enter":
				// 提交表单
				platform := m.platforms[m.platformCursor]
				rid := m.inputs[0].Value()
				cookie := m.inputs[1].Value()

				m.visible = false
				m.step = 0
				for i := range m.inputs {
					m.inputs[i].Blur()
					m.inputs[i].StopEdit()
				}

				return m, func() tea.Msg {
					return tuimsg.AddServiceRequestMsg{
						Platform: platform,
						RID:      rid,
						Cookie:   cookie,
					}
				}
			default:
				var cmd tea.Cmd
				m.inputs[1], cmd = m.inputs[1].Update(msg)
				return m, cmd
			}
		}
	}

	return m, nil
}

func (m AddServiceModel) View() string {
	if !m.visible {
		return ""
	}

	width := 70
	if m.width > 0 && m.width < 70 {
		width = m.width - 10
	}

	primaryColor := lipgloss.Color("#7D56F4")
	dimColor := lipgloss.Color("#666666")
	foregroundColor := lipgloss.Color("#FFFFFF")

	popupStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(primaryColor).
		Padding(1, 2)

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(primaryColor).
		Padding(0, 1)

	selectedStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(primaryColor)

	normalStyle := lipgloss.NewStyle().
		Foreground(foregroundColor)

	dimStyle := lipgloss.NewStyle().
		Foreground(dimColor)

	header := headerStyle.Render("Add Service")

	var lines []string

	// Step 0: Platform 选择
	if m.step == 0 {
		lines = append(lines, selectedStyle.Render("Select Platform:"))
		for i, platform := range m.platforms {
			cursor := "  "
			style := normalStyle
			if i == m.platformCursor {
				cursor = "• "
				style = selectedStyle
			}
			lines = append(lines, style.Render(fmt.Sprintf("  %s%s", cursor, platform)))
		}
		lines = append(lines, "")
		lines = append(lines, dimStyle.Render("Up/Down: Navigate | Enter: Confirm | Esc: Cancel"))
	} else {
		// 显示已选择的平台
		lines = append(lines, normalStyle.Render(fmt.Sprintf("Platform: %s", m.platforms[m.platformCursor])))
		lines = append(lines, "")
	}

	// Step 1: RID 输入
	switch m.step {
	case 1:
		lines = append(lines, selectedStyle.Render("> "+m.inputs[0].View()))
		lines = append(lines, dimStyle.Render("  "+m.inputs[1].View()))
		lines = append(lines, "")
		lines = append(lines, dimStyle.Render("Enter: Next | Esc: Back"))
	case 2:
		lines = append(lines, dimStyle.Render("  "+m.inputs[0].View()))
		lines = append(lines, selectedStyle.Render("> "+m.inputs[1].View()))
		lines = append(lines, "")
		lines = append(lines, dimStyle.Render("Enter: Submit | Esc: Back"))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)

	body := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		"",
		content,
	)

	return popupStyle.
		Width(width).
		Render(body)
}

func (m AddServiceModel) IsVisible() bool {
	return m.visible
}
