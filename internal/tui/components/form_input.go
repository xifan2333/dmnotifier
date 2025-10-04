package components

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// FormInputModel 表单输入组件
type FormInputModel struct {
	Label      string
	Input      textinput.Model
	Validator  func(string) error
	IsFocused  bool
	IsEditing  bool
	maskValue  bool // 是否遮罩显示（用于密码等）
}

// NewFormInput 创建新的表单输入
func NewFormInput(label, placeholder string, charLimit int) FormInputModel {
	input := textinput.New()
	input.Placeholder = placeholder
	input.CharLimit = charLimit

	return FormInputModel{
		Label:     label,
		Input:     input,
		IsFocused: false,
		IsEditing: false,
		maskValue: false,
	}
}

// NewPasswordInput 创建密码输入框
func NewPasswordInput(label, placeholder string, charLimit int) FormInputModel {
	input := textinput.New()
	input.Placeholder = placeholder
	input.CharLimit = charLimit
	input.EchoMode = textinput.EchoPassword
	input.EchoCharacter = '*'

	return FormInputModel{
		Label:     label,
		Input:     input,
		IsFocused: false,
		IsEditing: false,
		maskValue: true,
	}
}

// Focus 获得焦点
func (m *FormInputModel) Focus() {
	m.IsFocused = true
}

// Blur 失去焦点
func (m *FormInputModel) Blur() {
	m.IsFocused = false
	m.IsEditing = false
	m.Input.Blur()
}

// StartEdit 开始编辑
func (m *FormInputModel) StartEdit() {
	m.IsEditing = true
	m.Input.Focus()
}

// StopEdit 停止编辑
func (m *FormInputModel) StopEdit() {
	m.IsEditing = false
	m.Input.Blur()
}

// SetValue 设置值
func (m *FormInputModel) SetValue(value string) {
	m.Input.SetValue(value)
}

// Value 获取值
func (m *FormInputModel) Value() string {
	return m.Input.Value()
}

// Validate 验证输入
func (m *FormInputModel) Validate() error {
	if m.Validator != nil {
		return m.Validator(m.Input.Value())
	}
	return nil
}

// Update 更新
func (m FormInputModel) Update(msg tea.Msg) (FormInputModel, tea.Cmd) {
	if !m.IsEditing {
		return m, nil
	}

	var cmd tea.Cmd
	m.Input, cmd = m.Input.Update(msg)
	return m, cmd
}

// View 渲染
func (m FormInputModel) View() string {
	var labelStyle, valueStyle lipgloss.Style

	if m.IsFocused && !m.IsEditing {
		// 有焦点但未编辑
		labelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
		if m.maskValue && m.Input.Value() != "" {
			maskedValue := maskToken(m.Input.Value())
			valueStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
			return labelStyle.Render("> "+m.Label+": ") + valueStyle.Render(maskedValue)
		}
		valueStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
		return labelStyle.Render("> "+m.Label+": ") + valueStyle.Render(m.Input.Value())
	} else if m.IsEditing {
		// 正在编辑
		labelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
		return labelStyle.Render("  "+m.Label+": ") + m.Input.View()
	} else {
		// 无焦点
		labelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		if m.maskValue && m.Input.Value() != "" {
			maskedValue := maskToken(m.Input.Value())
			valueStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
			return labelStyle.Render("  "+m.Label+": ") + valueStyle.Render(maskedValue)
		}
		valueStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		return labelStyle.Render("  "+m.Label+": ") + valueStyle.Render(m.Input.Value())
	}
}

// maskToken 遮罩令牌
func maskToken(token string) string {
	if len(token) <= 8 {
		return "***"
	}
	return token[:4] + "***" + token[len(token)-4:]
}
