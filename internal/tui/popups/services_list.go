package popups

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	tuimsg "github.com/xifan2333/dmnotifier/internal/common"
	"github.com/xifan2333/dmnotifier/pkg/api"
)

type ServicesPopupModel struct {
	visible   bool
	services  []api.Service
	history   []tuimsg.ServiceHistoryEntry
	connected map[string]bool // platform/rid
	cursor    int
	width     int
	height    int
}

func NewServicesPopup() ServicesPopupModel {
	return ServicesPopupModel{
		visible:  false,
		services: []api.Service{},
		history:  []tuimsg.ServiceHistoryEntry{},
		cursor:   0,
	}
}

func (m ServicesPopupModel) Init() tea.Cmd {
	return nil
}

// totalItems 返回合并后的条目数（运行中 + 历史）
func (m ServicesPopupModel) totalItems() int {
	return len(m.services) + len(m.history)
}

// isHistoryCursor 当前光标是否落在历史区
func (m ServicesPopupModel) isHistoryCursor() bool {
	return m.cursor >= len(m.services)
}

// historyIndex 历史区中的索引（仅在 isHistoryCursor() 为 true 时有效）
func (m ServicesPopupModel) historyIndex() int {
	return m.cursor - len(m.services)
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
		m.history = msg.History
		m.connected = map[string]bool{}
		for _, k := range msg.Connected {
			m.connected[k] = true
		}
		if m.cursor >= m.totalItems() {
			m.cursor = 0
		}
		return m, nil

	case tuimsg.ConnectedSnapshotMsg:
		m.connected = map[string]bool{}
		for _, k := range msg.Keys {
			m.connected[k] = true
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

		total := m.totalItems()

		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < total-1 {
				m.cursor++
			}

		case "enter":
			if total == 0 {
				return m, nil
			}
			if m.isHistoryCursor() {
				entry := m.history[m.historyIndex()]
				m.visible = false
				return m, func() tea.Msg {
					return tuimsg.ReuseHistoryRequestMsg{
						Platform: entry.Platform,
						RID:      entry.RID,
						Cookie:   entry.Cookie,
					}
				}
			}
			service := m.services[m.cursor]
			m.visible = false
			return m, func() tea.Msg {
				return tuimsg.ConnectServiceRequestMsg{Service: &service}
			}

		case "x", "delete":
			if total == 0 {
				return m, nil
			}
			if m.isHistoryCursor() {
				entry := m.history[m.historyIndex()]
				return m, func() tea.Msg {
					return tuimsg.DeleteHistoryEntryRequestMsg{
						Platform: entry.Platform,
						RID:      entry.RID,
					}
				}
			}
			service := m.services[m.cursor]
			m.visible = false
			return m, func() tea.Msg {
				return tuimsg.StopServiceRequestMsg{Platform: service.Platform, RID: service.RID}
			}

		case "e", "edit":
			// 编辑历史记录
			if total == 0 || !m.isHistoryCursor() {
				return m, nil
			}
			entry := m.history[m.historyIndex()]
			m.visible = false
			return m, func() tea.Msg {
				return tuimsg.ShowEditHistoryPopupMsg{Entry: entry}
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
	runningSectionColor := lipgloss.Color("#10B981")
	historySectionColor := lipgloss.Color("#F59E0B")

	popupStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(primaryColor).
		Padding(1, 2)

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(primaryColor).
		Padding(0, 1)

	runningSectionStyle := lipgloss.NewStyle().Bold(true).Foreground(runningSectionColor)
	historySectionStyle := lipgloss.NewStyle().Bold(true).Foreground(historySectionColor)

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
	if m.totalItems() == 0 {
		content = dimStyle.Render("No services and no history")
	} else {
		if len(m.services) > 0 {
			content += runningSectionStyle.Render("Running (Enter adds local subscribe)") + "\n"
			for i, svc := range m.services {
				cursor := " "
				itemStyle := normalStyle
				if m.cursor == i {
					cursor = ">"
					itemStyle = selectedStyle
				}
				mark := " "
				if m.connected[svc.Platform+"/"+svc.RID] {
					mark = "*"
				}
				line := fmt.Sprintf("%s%s %s/%s", cursor, mark, svc.Platform, svc.RID)
				content += itemStyle.Render(line) + "\n"
			}
		}
		if len(m.history) > 0 {
			if len(m.services) > 0 {
				content += "\n"
			}
			content += historySectionStyle.Render("History") + "\n"
			for i, h := range m.history {
				globalIdx := len(m.services) + i
				cursor := " "
				itemStyle := normalStyle
				if m.cursor == globalIdx {
					cursor = ">"
					itemStyle = selectedStyle
				}
				line := fmt.Sprintf("%s %s/%s", cursor, h.Platform, h.RID)
				content += itemStyle.Render(line) + "\n"
			}
		}
	}

	var helpText string
	if m.isHistoryCursor() && len(m.history) > 0 {
		helpText = "Up/Down: Select | Enter: Reconnect | e: Edit | x: Remove from history | Esc: Close"
	} else {
		helpText = "Up/Down: Select | Enter: Subscribe(+multi) | x: Stop remote | Esc: Close"
	}
	help := dimStyle.Render(helpText)

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
