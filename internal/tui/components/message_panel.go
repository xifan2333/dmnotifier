package components

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// MessagePanelModel 消息面板模型
type MessagePanelModel struct {
	viewport    viewport.Model
	messages    []string
	maxMessages int
	width       int
	height      int
}

// AddMessageMsg 添加消息的消息类型
type AddMessageMsg struct {
	Content string
}

// NewMessagePanel 创建消息面板
func NewMessagePanel() MessagePanelModel {
	vp := viewport.New(80, 20)
	vp.SetContent("")

	return MessagePanelModel{
		viewport:    vp,
		messages:    []string{},
		maxMessages: 100,
	}
}

// Init 初始化
func (m MessagePanelModel) Init() tea.Cmd {
	return nil
}

// Update 更新
func (m MessagePanelModel) Update(msg tea.Msg) (MessagePanelModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = m.width - 4
		m.viewport.Height = m.height - 2

	case AddMessageMsg:
		m.addMessage(msg.Content)
		return m, nil
	}

	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// View 渲染
func (m MessagePanelModel) View() string {
	messagePanelStyle := lipgloss.NewStyle().Padding(1, 2)

	return messagePanelStyle.
		Width(m.width - 4).
		Height(m.height).
		Render(m.viewport.View())
}

// addMessage 添加消息
func (m *MessagePanelModel) addMessage(msg string) {
	m.messages = append(m.messages, msg)
	if len(m.messages) > m.maxMessages {
		m.messages = m.messages[1:]
	}

	// 更新 viewport 内容
	content := ""
	for _, message := range m.messages {
		content += message + "\n"
	}
	m.viewport.SetContent(content)
	m.viewport.GotoBottom()
}
