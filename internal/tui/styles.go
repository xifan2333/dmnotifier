package tui

import "github.com/charmbracelet/lipgloss"

var (
	// 颜色定义
	primaryColor = lipgloss.Color("#7D56F4")

	dimColor = lipgloss.Color("#666666")

	// 标题栏样式
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			Padding(0, 1)

	// 信息栏样式
	infoStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Padding(0, 1)

	// 消息面板样式
	messagePanelStyle = lipgloss.NewStyle().
				Padding(1, 2)

	// 弱化文本样式
	dimStyle = lipgloss.NewStyle().
			Foreground(dimColor)

	// 状态栏样式
	statusStyle = lipgloss.NewStyle().
			Foreground(dimColor).
			Padding(0, 1)

	// 帮助栏样式
	helpStyle = lipgloss.NewStyle().
			Foreground(dimColor).
			Italic(true).
			Padding(0, 1)
)

// GetMessagePanelStyle 获取消息面板样式（供 components 包使用）
func GetMessagePanelStyle() lipgloss.Style {
	return messagePanelStyle
}
