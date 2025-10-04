package popups

import (
	"fmt"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	tuimsg "github.com/xifan/dmnotifier/internal/common"
	"github.com/xifan/dmnotifier/internal/plugin"
	"github.com/xifan/dmnotifier/internal/tui/components"
)

// 可用的消息类型
var availableMessageTypes = []string{"Chat", "Gift", "Like", "EnterRoom", "Subscribe", "SuperChat", "EndLive"}

// PluginsConfigModel 插件配置弹窗模型
type PluginsConfigModel struct {
	visible bool
	width   int
	height  int

	// 插件列表
	plugins []tuimsg.PluginConfig

	// 导航状态
	pluginCursor     int // 当前选中的插件索引
	pluginItemCursor int // 当前插件内的项目索引 (0=name, 1=types, 2+=config fields)

	// 编辑状态
	pluginEditingTypes bool // 是否正在编辑消息类型
	pluginTypesCursor  int  // 消息类型列表光标

	pluginEditingField int                       // 正在编辑的字段索引 (-1 表示未编辑)
	pluginConfigInput  components.FormInputModel // 配置字段输入框

	// 样式
	primaryColor    lipgloss.Color
	foregroundColor lipgloss.Color
	dimColor        lipgloss.Color
	popupStyle      lipgloss.Style
	headerStyle     lipgloss.Style
	itemStyle       lipgloss.Style
	selectedStyle   lipgloss.Style
	dimStyle        lipgloss.Style
}

// NewPluginsConfig 创建插件配置弹窗
func NewPluginsConfig() PluginsConfigModel {
	primaryColor := lipgloss.Color("#7D56F4")
	foregroundColor := lipgloss.Color("#FFFFFF")
	dimColor := lipgloss.Color("#666666")

	return PluginsConfigModel{
		visible:            false,
		plugins:            []tuimsg.PluginConfig{},
		pluginCursor:       0,
		pluginItemCursor:   0,
		pluginEditingTypes: false,
		pluginTypesCursor:  0,
		pluginEditingField: -1,
		pluginConfigInput:  components.NewFormInput("", "", 200),
		primaryColor:       primaryColor,
		foregroundColor:    foregroundColor,
		dimColor:           dimColor,
		popupStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(1, 2),
		headerStyle: lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			Padding(0, 1),
		itemStyle: lipgloss.NewStyle().
			Foreground(foregroundColor).
			Padding(0, 1),
		selectedStyle: lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			Underline(true).
			Padding(0, 1),
		dimStyle: lipgloss.NewStyle().
			Foreground(dimColor),
	}
}

// Init 初始化
func (m PluginsConfigModel) Init() tea.Cmd {
	return nil
}

// Update 更新
func (m PluginsConfigModel) Update(msg tea.Msg) (PluginsConfigModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tuimsg.ShowPluginsConfigPopupMsg:
		m.visible = true
		m.plugins = msg.Plugins
		m.pluginCursor = 0
		m.pluginItemCursor = 0
		m.pluginEditingTypes = false
		m.pluginEditingField = -1
		return m, nil

	case tuimsg.HidePopupMsg:
		m.visible = false
		m.pluginEditingTypes = false
		m.pluginEditingField = -1
		m.pluginConfigInput.Blur()
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if !m.visible {
			return m, nil
		}

		return m.handleKeyPress(msg)
	}

	return m, nil
}

// handleKeyPress 处理按键
func (m PluginsConfigModel) handleKeyPress(msg tea.KeyMsg) (PluginsConfigModel, tea.Cmd) {
	// 如果正在编辑某个字段
	if m.pluginEditingField >= 0 {
		return m.handleFieldEditing(msg)
	}

	// 如果在编辑消息类型
	if m.pluginEditingTypes {
		return m.handleMessageTypesEditing(msg)
	}

	// 普通导航模式
	return m.handleNavigation(msg)
}

// handleFieldEditing 处理字段编辑
func (m PluginsConfigModel) handleFieldEditing(msg tea.KeyMsg) (PluginsConfigModel, tea.Cmd) {
	if m.pluginCursor >= len(m.plugins) {
		return m, nil
	}

	pluginCfg := &m.plugins[m.pluginCursor]
	template := getPluginConfigTemplate(pluginCfg.Name)

	if m.pluginEditingField >= len(template) {
		m.pluginEditingField = -1
		return m, nil
	}

	switch msg.String() {
	case "esc":
		// 退出编辑模式
		m.pluginEditingField = -1
		m.pluginConfigInput.Blur()
		return m, func() tea.Msg {
			return tuimsg.StatusMsg{Message: "Cancelled"}
		}

	case "enter":
		// 保存值
		field := template[m.pluginEditingField]
		newValue := m.pluginConfigInput.Value()

		// 根据类型转换值
		switch field.Type {
		case plugin.FieldTypeString:
			pluginCfg.Config[field.Name] = newValue
		case plugin.FieldTypeNumber:
			// 尝试解析为数字
			if intVal, err := strconv.Atoi(newValue); err == nil {
				pluginCfg.Config[field.Name] = intVal
			} else if floatVal, err := strconv.ParseFloat(newValue, 64); err == nil {
				pluginCfg.Config[field.Name] = floatVal
			} else {
				return m, func() tea.Msg {
					return tuimsg.StatusMsg{Message: fmt.Sprintf("Invalid number: %s", newValue)}
				}
			}
		}

		m.pluginEditingField = -1
		m.pluginConfigInput.Blur()

		// 发送更新消息
		return m, tea.Batch(
			func() tea.Msg {
				return tuimsg.StatusMsg{Message: fmt.Sprintf("Saved %s = %s", field.Name, newValue)}
			},
			func() tea.Msg {
				return tuimsg.UpdatePluginsConfigMsg{Plugins: m.plugins}
			},
		)

	default:
		// 传递按键给输入框
		var cmd tea.Cmd
		m.pluginConfigInput, cmd = m.pluginConfigInput.Update(msg)
		return m, cmd
	}
}

// handleMessageTypesEditing 处理消息类型编辑
func (m PluginsConfigModel) handleMessageTypesEditing(msg tea.KeyMsg) (PluginsConfigModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// 退出编辑模式
		m.pluginEditingTypes = false
		return m, func() tea.Msg {
			return tuimsg.StatusMsg{Message: "Exit message types editing"}
		}

	case "up", "k":
		if m.pluginTypesCursor > 0 {
			m.pluginTypesCursor--
		}

	case "down", "j":
		if m.pluginTypesCursor < len(availableMessageTypes)-1 {
			m.pluginTypesCursor++
		}

	case "enter", " ":
		// 切换消息类型
		if m.pluginCursor < len(m.plugins) {
			plugin := &m.plugins[m.pluginCursor]
			msgType := availableMessageTypes[m.pluginTypesCursor]

			// 检查是否已选中
			found := -1
			for i, t := range plugin.MessageTypes {
				if t == msgType {
					found = i
					break
				}
			}

			if found >= 0 {
				// 已选中，移除
				plugin.MessageTypes = append(plugin.MessageTypes[:found], plugin.MessageTypes[found+1:]...)
			} else {
				// 未选中，添加
				plugin.MessageTypes = append(plugin.MessageTypes, msgType)
			}

			return m, tea.Batch(
				func() tea.Msg {
					return tuimsg.StatusMsg{Message: fmt.Sprintf("Toggled %s for %s", msgType, plugin.Name)}
				},
				func() tea.Msg {
					return tuimsg.UpdatePluginsConfigMsg{Plugins: m.plugins}
				},
			)
		}
	}
	return m, nil
}

// handleNavigation 处理普通导航
func (m PluginsConfigModel) handleNavigation(msg tea.KeyMsg) (PluginsConfigModel, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.pluginItemCursor > 0 {
			// 在当前插件内向上移动
			m.pluginItemCursor--
		} else if m.pluginCursor > 0 {
			// 跳到上一个插件的最后一项
			m.pluginCursor--
			m.pluginItemCursor = m.getPluginItemCount() - 1
		}

	case "down", "j":
		itemCount := m.getPluginItemCount()
		if m.pluginItemCursor < itemCount-1 {
			// 在当前插件内向下移动
			m.pluginItemCursor++
		} else if m.pluginCursor < len(m.plugins)-1 {
			// 跳到下一个插件的第一项
			m.pluginCursor++
			m.pluginItemCursor = 0
		}

	case " ":
		// 只在插件名行才能切换启用/禁用
		if m.pluginItemCursor == 0 && m.pluginCursor < len(m.plugins) {
			m.plugins[m.pluginCursor].Enabled = !m.plugins[m.pluginCursor].Enabled
			status := "disabled"
			if m.plugins[m.pluginCursor].Enabled {
				status = "enabled"
			}
			return m, tea.Batch(
				func() tea.Msg {
					return tuimsg.StatusMsg{Message: fmt.Sprintf("Plugin %s %s", m.plugins[m.pluginCursor].Name, status)}
				},
				func() tea.Msg {
					return tuimsg.UpdatePluginsConfigMsg{Plugins: m.plugins}
				},
			)
		}

	case "enter":
		if m.pluginCursor >= len(m.plugins) {
			return m, nil
		}

		pluginCfg := &m.plugins[m.pluginCursor]
		template := getPluginConfigTemplate(pluginCfg.Name)

		switch m.pluginItemCursor {
		case 0:
			// 在插件名上，不做任何事
			return m, func() tea.Msg {
				return tuimsg.StatusMsg{Message: "Use Space to toggle enable/disable"}
			}
		case 1:
			// 在 Types 行，进入消息类型编辑
			m.pluginEditingTypes = true
			m.pluginTypesCursor = 0
			return m, func() tea.Msg {
				return tuimsg.StatusMsg{Message: "Editing message types (Space: toggle, Esc: exit)"}
			}
		default:
			// 在配置字段行
			fieldIdx := m.pluginItemCursor - 2
			if fieldIdx < len(template) {
				field := template[fieldIdx]

				switch field.Type {
				case plugin.FieldTypeBool:
					// Bool 类型直接切换
					if val, ok := pluginCfg.Config[field.Name].(bool); ok {
						pluginCfg.Config[field.Name] = !val
						return m, tea.Batch(
							func() tea.Msg {
								return tuimsg.StatusMsg{Message: fmt.Sprintf("Toggled %s", field.Name)}
							},
							func() tea.Msg {
								return tuimsg.UpdatePluginsConfigMsg{Plugins: m.plugins}
							},
						)
					}

				case plugin.FieldTypeString, plugin.FieldTypeNumber:
					// String/Number 类型进入输入模式
					currentVal := fmt.Sprintf("%v", pluginCfg.Config[field.Name])
					m.pluginConfigInput.SetValue(currentVal)
					m.pluginConfigInput.Focus()
					m.pluginConfigInput.StartEdit()
					m.pluginEditingField = fieldIdx
					return m, func() tea.Msg {
						return tuimsg.StatusMsg{Message: fmt.Sprintf("Editing %s (Enter: save, Esc: cancel)", field.Name)}
					}

				default:
					return m, func() tea.Msg {
						return tuimsg.StatusMsg{Message: fmt.Sprintf("Type %s not supported yet", field.Type)}
					}
				}
			}
		}
	}

	return m, nil
}

// View 渲染
func (m PluginsConfigModel) View() string {
	if !m.visible {
		return ""
	}

	width := 80
	if m.width > 0 && m.width < 80 {
		width = m.width - 10
	}
	height := 25
	if m.height > 0 && m.height < 30 {
		height = m.height - 10
	}

	header := m.headerStyle.Width(width - 4).Render("Plugins Config")

	var content string
	if m.pluginEditingTypes {
		// 显示消息类型编辑界面
		content = m.renderMessageTypesEditor()
	} else {
		// 显示插件列表
		content = m.renderPluginsList()
	}

	var help string
	if m.pluginEditingTypes {
		help = m.dimStyle.Render("Up/Down: Navigate | Space: Toggle | Esc: Back")
	} else {
		help = m.dimStyle.Render("Up/Down: Navigate | Space: Toggle Enable | Enter: Edit | Esc: Close")
	}

	body := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		"",
		content,
		"",
		help,
	)

	return m.popupStyle.
		Width(width).
		Height(height).
		Render(body)
}

// IsVisible 返回弹窗是否可见
func (m PluginsConfigModel) IsVisible() bool {
	return m.visible
}

// getPluginItemCount 获取当前插件的项目总数
func (m PluginsConfigModel) getPluginItemCount() int {
	if m.pluginCursor >= len(m.plugins) {
		return 0
	}
	template := getPluginConfigTemplate(m.plugins[m.pluginCursor].Name)
	// 0=name, 1=types, 2+=config fields
	return 2 + len(template)
}

// getPluginConfigTemplate 获取插件的配置模板
func getPluginConfigTemplate(pluginName string) []plugin.ConfigField {
	pluginInfos := plugin.GlobalRegistry.GetAllPluginInfo()
	for _, info := range pluginInfos {
		if info.Name == pluginName {
			return info.ConfigTemplate
		}
	}
	return []plugin.ConfigField{}
}

// renderPluginsList 渲染插件列表
func (m PluginsConfigModel) renderPluginsList() string {
	var items []string

	for i, pluginCfg := range m.plugins {
		template := getPluginConfigTemplate(pluginCfg.Name)

		// 插件名行
		cursor := "  "
		if i == m.pluginCursor && m.pluginItemCursor == 0 {
			cursor = "> "
		}

		status := "[ ]"
		if pluginCfg.Enabled {
			status = "[✓]"
		}

		pluginLine := fmt.Sprintf("%s%s %s", cursor, status, pluginCfg.Name)
		if i == m.pluginCursor && m.pluginItemCursor == 0 {
			items = append(items, m.selectedStyle.Render(pluginLine))
		} else {
			items = append(items, m.itemStyle.Render(pluginLine))
		}

		// Types 行
		cursor = "  "
		if i == m.pluginCursor && m.pluginItemCursor == 1 {
			cursor = "> "
		}

		typesStr := ""
		totalTypes := len(availableMessageTypes)
		selectedCount := len(pluginCfg.MessageTypes)

		if selectedCount == 0 {
			typesStr = "(none)"
		} else if selectedCount == totalTypes {
			// 全选：显示 "All (7)"
			typesStr = fmt.Sprintf("All (%d)", totalTypes)
		} else if selectedCount <= 3 {
			// 少量选中：完整显示
			for j, msgType := range pluginCfg.MessageTypes {
				if j > 0 {
					typesStr += ", "
				}
				typesStr += msgType
			}
		} else {
			// 大量选中：显示前 3 个 + 省略号
			for j := 0; j < 3; j++ {
				if j > 0 {
					typesStr += ", "
				}
				typesStr += pluginCfg.MessageTypes[j]
			}
			typesStr += fmt.Sprintf("... (%d/%d)", selectedCount, totalTypes)
		}

		typesLine := fmt.Sprintf("%sTypes: %s", cursor, typesStr)
		if i == m.pluginCursor && m.pluginItemCursor == 1 {
			items = append(items, m.selectedStyle.Render(typesLine))
		} else {
			items = append(items, m.itemStyle.Foreground(m.dimColor).Render(typesLine))
		}

		// Config fields
		for fieldIdx, field := range template {
			itemIdx := 2 + fieldIdx
			cursor = "  "
			if i == m.pluginCursor && m.pluginItemCursor == itemIdx {
				cursor = "> "
			}

			value := pluginCfg.Config[field.Name]

			// 如果当前字段正在编辑，单独渲染输入框
			if i == m.pluginCursor && m.pluginEditingField == fieldIdx && m.pluginConfigInput.IsEditing {
				label := fmt.Sprintf("%s%s: ", cursor, field.Name)
				line := m.selectedStyle.Render(label) + m.pluginConfigInput.View()
				items = append(items, line)
			} else {
				// 根据类型格式化值
				var valueStr string
				switch field.Type {
				case plugin.FieldTypeBool:
					if value == true {
						valueStr = "[✓]"
					} else {
						valueStr = "[ ]"
					}
				case plugin.FieldTypeNumber:
					valueStr = fmt.Sprintf("%v", value)
				case plugin.FieldTypeString:
					valueStr = fmt.Sprintf("%v", value)
					if valueStr == "" {
						valueStr = "(empty)"
					}
				default:
					valueStr = fmt.Sprintf("%v", value)
				}

				line := fmt.Sprintf("%s%s: %s", cursor, field.Name, valueStr)
				if i == m.pluginCursor && m.pluginItemCursor == itemIdx {
					items = append(items, m.selectedStyle.Render(line))
				} else {
					items = append(items, m.itemStyle.Foreground(m.dimColor).Render(line))
				}
			}
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, items...)
}

// renderMessageTypesEditor 渲染消息类型编辑器
func (m PluginsConfigModel) renderMessageTypesEditor() string {
	if m.pluginCursor >= len(m.plugins) {
		return ""
	}

	plugin := m.plugins[m.pluginCursor]
	var items []string

	items = append(items, m.itemStyle.Bold(true).Render(fmt.Sprintf("Edit message types for: %s", plugin.Name)))
	items = append(items, "")

	for i, msgType := range availableMessageTypes {
		cursor := "  "
		if i == m.pluginTypesCursor {
			cursor = "> "
		}

		// 检查是否已选中
		checked := "[ ]"
		for _, t := range plugin.MessageTypes {
			if t == msgType {
				checked = "[✓]"
				break
			}
		}

		line := fmt.Sprintf("%s%s %s", cursor, checked, msgType)
		if i == m.pluginTypesCursor {
			items = append(items, m.selectedStyle.Render(line))
		} else {
			items = append(items, m.itemStyle.Render(line))
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, items...)
}
