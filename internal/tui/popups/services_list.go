package popups

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	tuimsg "github.com/xifan2333/dmnotifier/internal/common"
	"github.com/xifan2333/dmnotifier/pkg/api"
)

type ServicesPopupModel struct {
	visible  bool
	services []api.Service
	cursor   int
	width    int
	height   int
}

func NewServicesPopup() ServicesPopupModel {
	return ServicesPopupModel{
		visible:  false,
		services: []api.Service{},
		cursor:   0,
	}
}

func (m ServicesPopupModel) Init() tea.Cmd {
	return nil
}

func (m ServicesPopupModel) Update(msg tea.Msg) (ServicesPopupModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tuimsg.ShowServicesPopupMsg:
		m.visible = true
		return m, nil

	case tuimsg.HidePopupMsg:
		m.visible = false
		return m, nil

	case tuimsg.ServicesLoadedMsg:
		m.services = msg.Services
		if m.cursor >= len(m.services) {
			m.cursor = 0
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

		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.services)-1 {
				m.cursor++
			}

		case "enter":
			if len(m.services) > 0 && m.cursor < len(m.services) {
				service := m.services[m.cursor]
				m.visible = false
				return m, func() tea.Msg {
					return tuimsg.ConnectServiceRequestMsg{Service: &service}
				}
			}

		case "x", "delete":
			if len(m.services) > 0 && m.cursor < len(m.services) {
				service := m.services[m.cursor]
				m.visible = false
				return m, func() tea.Msg {
					return tuimsg.StopServiceRequestMsg{Platform: service.Platform, RID: service.RID}
				}
			}
		}
	}

	return m, nil
}

func (m ServicesPopupModel) View() string {
	if !m.visible {
		return ""
	}

	width := 60
	if m.width > 0 && m.width < 60 {
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
		Foreground(primaryColor).
		Underline(true).
		Padding(0, 1)

	normalStyle := lipgloss.NewStyle().
		Foreground(foregroundColor)

	dimStyle := lipgloss.NewStyle().
		Foreground(dimColor)

	header := headerStyle.Width(width - 4).Render("Services")

	content := ""
	if len(m.services) == 0 {
		content = dimStyle.Render("No services running")
	} else {
		for i, svc := range m.services {
			cursor := " "
			itemStyle := normalStyle
			if m.cursor == i {
				cursor = ">"
				itemStyle = selectedStyle
			}

			line := fmt.Sprintf("%s %s/%s", cursor, svc.Platform, svc.RID)
			content += itemStyle.Render(line) + "\n"
		}
	}

	help := dimStyle.Render("Up/Down: Select | Enter: Connect | x: Remove | Esc: Close")

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

func (m ServicesPopupModel) IsVisible() bool {
	return m.visible
}
