package popups

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	tuimsg "github.com/xifan2333/dmnotifier/internal/common"
	"github.com/xifan2333/dmnotifier/internal/plugin"
)

// All rendering, navigation and editing use this same field list. Hidden
// provider settings remain in Config and are restored when switching back.
func visiblePluginFields(cfg tuimsg.PluginConfig) []plugin.ConfigField {
	fields := getPluginConfigTemplate(cfg.Name)
	if cfg.Name != "tts" {
		return fields
	}
	result := make([]plugin.ConfigField, 0, len(fields))
	for _, f := range fields {
		switch f.Name {
		case "edge_voice":
			if ttsProvider(cfg) != "edge" {
				continue
			}
		case "base_url", "model":
			continue
		case "api_key", "voice":
			if ttsProvider(cfg) == "edge" {
				continue
			}
		}
		result = append(result, f)
	}
	return result
}

func ttsProvider(cfg tuimsg.PluginConfig) string {
	value, _ := cfg.Config["provider"].(string)
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "mimo"
	}
	return value
}
func fieldLabel(f plugin.ConfigField) string {
	if f.Label != "" {
		return f.Label
	}
	return f.Name
}
func optionLabel(f plugin.ConfigField, value string) string {
	if label := f.OptionLabels[value]; label != "" {
		return label
	}
	return value
}
func fieldValue(cfg tuimsg.PluginConfig, f plugin.ConfigField) string {
	value := cfg.Config[f.Name]
	if value == nil {
		value = f.Default
	}
	return fmt.Sprint(value)
}
func (m PluginsConfigModel) selectedEnum() (plugin.ConfigField, bool) {
	if m.pluginCursor < 0 || m.pluginCursor >= len(m.plugins) {
		return plugin.ConfigField{}, false
	}
	fields := visiblePluginFields(m.plugins[m.pluginCursor])
	if m.pluginEditingField < 0 || m.pluginEditingField >= len(fields) {
		return plugin.ConfigField{}, false
	}
	f := fields[m.pluginEditingField]
	return f, f.Type == plugin.FieldTypeEnum
}
func (m PluginsConfigModel) enumOptions(f plugin.ConfigField) []string {
	query := strings.ToLower(strings.TrimSpace(m.enumQuery))
	if query == "" {
		return f.Options
	}
	var result []string
	for _, value := range f.Options {
		if strings.Contains(strings.ToLower(value+" "+optionLabel(f, value)), query) {
			result = append(result, value)
		}
	}
	return result
}
func (m PluginsConfigModel) handleEnumSelection(msg tea.KeyMsg) (PluginsConfigModel, tea.Cmd) {
	f, ok := m.selectedEnum()
	if !ok {
		m.enumSelecting = false
		m.pluginEditingField = -1
		return m, nil
	}
	options := m.enumOptions(f)
	switch msg.String() {
	case "esc":
		m.enumSelecting = false
		m.pluginEditingField = -1
		return m, nil
	case "up":
		m.enumCursor = max(0, m.enumCursor-1)
	case "down":
		m.enumCursor = max(0, min(len(options)-1, m.enumCursor+1))
	case "home":
		m.enumCursor = 0
	case "end":
		m.enumCursor = max(0, len(options)-1)
	case "backspace":
		runes := []rune(m.enumQuery)
		if len(runes) > 0 {
			m.enumQuery = string(runes[:len(runes)-1])
		}
		m.enumCursor = 0
	case "ctrl+u":
		m.enumQuery = ""
		m.enumCursor = 0
	case "enter":
		if m.enumCursor < 0 || m.enumCursor >= len(options) {
			return m, nil
		}
		cfg := &m.plugins[m.pluginCursor]
		if cfg.Config == nil {
			cfg.Config = make(map[string]interface{})
		}
		value := options[m.enumCursor]
		cfg.Config[f.Name] = value
		m.enumSelecting = false
		m.pluginEditingField = -1
		m.pluginItemCursor = min(m.pluginItemCursor, m.getPluginItemCount()-1)
		return m, tea.Batch(
			func() tea.Msg { return tuimsg.UpdatePluginsConfigMsg{Plugins: m.plugins} },
			func() tea.Msg {
				return tuimsg.StatusMsg{Message: fmt.Sprintf("%s: %s. Reconnect to apply.", fieldLabel(f), optionLabel(f, value))}
			},
		)
	default:
		if msg.Type == tea.KeyRunes {
			m.enumQuery += string(msg.Runes)
			m.enumCursor = 0
		}
	}
	return m, nil
}
func (m PluginsConfigModel) renderEnumSelector() string {
	f, ok := m.selectedEnum()
	if !ok {
		return ""
	}
	options := m.enumOptions(f)
	current := fieldValue(m.plugins[m.pluginCursor], f)
	width := 70
	if m.width > 0 {
		width = max(16, min(width, m.width-18))
	}
	lines := []string{
		m.itemStyle.Bold(true).Render("Select " + fieldLabel(f)),
		m.dimStyle.Render(ansi.Truncate("Current: "+optionLabel(f, current), width, "…")),
		m.itemStyle.Render("Filter: " + m.enumQuery),
		"",
	}
	if len(options) == 0 {
		lines = append(lines, m.dimStyle.Render("No matches."))
	} else {
		count := 7
		if m.height > 0 {
			count = max(1, min(count, m.height-16))
		}
		start := max(0, min(m.enumCursor-count/2, len(options)-count))
		end := min(len(options), start+count)
		for i := start; i < end; i++ {
			cursor := "  "
			if i == m.enumCursor {
				cursor = "> "
			}
			mark := ""
			if options[i] == current {
				mark = " ✓"
			}
			text := ansi.Truncate(cursor+optionLabel(f, options[i])+mark, width, "…")
			if i == m.enumCursor {
				lines = append(lines, m.selectedStyle.Render(text))
			} else {
				lines = append(lines, m.itemStyle.Render(text))
			}
		}
		lines = append(lines, m.dimStyle.Render(fmt.Sprintf("%d–%d / %d", start+1, end, len(options))))
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// IsEditing reports whether Escape should return to the plugin list.
func (m PluginsConfigModel) IsEditing() bool {
	return m.enumSelecting || m.pluginEditingField >= 0 || m.pluginEditingTypes
}
