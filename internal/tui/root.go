package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	tuimsg "github.com/xifan2333/dmnotifier/internal/common"
	"github.com/xifan2333/dmnotifier/internal/tui/components"
	"github.com/xifan2333/dmnotifier/internal/tui/popups"
	"github.com/xifan2333/dmnotifier/pkg/api"
)

// RootModel 根模型，协调所有子模型
type RootModel struct {
	// 子模型
	messagePanel  components.MessagePanelModel
	servicesPopup popups.ServicesPopupModel
	serverConfig  popups.ServerConfigModel
	addService    popups.AddServiceModel
	pluginsConfig popups.PluginsConfigModel

	// 当前连接（多路）
	selectedService *api.Service // 最近连接的一个，兼容旧 UI
	connectedKeys   []string     // platform/rid 列表

	// 配置
	config *AppConfig

	// 状态
	statusMessage string
	width         int
	height        int
}

// NewRootModel 创建根模型
func NewRootModel(config *AppConfig) RootModel {
	serverConfig := popups.NewServerConfig()
	serverConfig.SetConfig(
		config.Server.APIAddress,
		config.Server.APIToken,
		config.Server.WSAddress,
	)

	return RootModel{
		messagePanel:  components.NewMessagePanel(),
		servicesPopup: popups.NewServicesPopup(),
		serverConfig:  serverConfig,
		addService:    popups.NewAddService(),
		pluginsConfig: popups.NewPluginsConfig(),
		config:        config,
		statusMessage: "Ready",
	}
}

// Init 初始化
func (m RootModel) Init() tea.Cmd {
	return tea.Batch(
		m.messagePanel.Init(),
		m.servicesPopup.Init(),
		m.serverConfig.Init(),
		m.addService.Init(),
		m.pluginsConfig.Init(),
		// 发送请求刷新服务列表
		func() tea.Msg {
			return tuimsg.RefreshServicesRequestMsg{}
		},
	)
}

// Update 更新
func (m RootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		// 如果有弹窗显示，优先处理弹窗按键
		if m.servicesPopup.IsVisible() {
			var cmd tea.Cmd
			m.servicesPopup, cmd = m.servicesPopup.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}

			// Esc 关闭弹窗
			if msg.String() == "esc" {
				m.servicesPopup, _ = m.servicesPopup.Update(tuimsg.HidePopupMsg{})
			}

			return m, tea.Batch(cmds...)
		}

		if m.serverConfig.IsVisible() {
			var cmd tea.Cmd
			m.serverConfig, cmd = m.serverConfig.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}

			// Esc 关闭弹窗（只有在非编辑状态）
			if msg.String() == "esc" {
				m.serverConfig, _ = m.serverConfig.Update(tuimsg.HidePopupMsg{})
			}

			return m, tea.Batch(cmds...)
		}

		if m.addService.IsVisible() {
			var cmd tea.Cmd
			m.addService, cmd = m.addService.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}

			// Esc 关闭弹窗（只有在非编辑状态）
			if msg.String() == "esc" {
				m.addService, _ = m.addService.Update(tuimsg.HidePopupMsg{})
			}

			return m, tea.Batch(cmds...)
		}

		if m.pluginsConfig.IsVisible() {
			var cmd tea.Cmd
			m.pluginsConfig, cmd = m.pluginsConfig.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}

			// Esc 关闭弹窗（只有在非编辑状态）
			if msg.String() == "esc" {
				m.pluginsConfig, _ = m.pluginsConfig.Update(tuimsg.HidePopupMsg{})
			}

			return m, tea.Batch(cmds...)
		}

		// 主界面按键处理
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "s":
			// 显示服务列表
			m.servicesPopup, _ = m.servicesPopup.Update(tuimsg.ShowServicesPopupMsg{})
			return m, nil

		case "c":
			// 显示服务器配置
			m.serverConfig, _ = m.serverConfig.Update(tuimsg.ShowServerConfigPopupMsg{})
			return m, nil

		case "a":
			// 显示添加服务弹窗
			m.addService, _ = m.addService.Update(tuimsg.ShowAddServicePopupMsg{})
			return m, nil

		case "p":
			// 显示插件配置弹窗
			m.pluginsConfig, _ = m.pluginsConfig.Update(tuimsg.ShowPluginsConfigPopupMsg{
				Plugins: m.config.Pipeline.Plugins,
			})
			return m, nil

		case "r":
			// 刷新服务列表
			m.statusMessage = "Refreshing services..."
			return m, func() tea.Msg {
				return tuimsg.RefreshServicesRequestMsg{}
			}

		case "d":
			// 断开全部本地订阅
			if len(m.connectedKeys) > 0 || m.selectedService != nil {
				return m, func() tea.Msg {
					return tuimsg.DisconnectServiceRequestMsg{}
				}
			}
		}

	case tuimsg.StatusMsg:
		m.statusMessage = msg.Message

	case tuimsg.ServiceConnectedMsg:
		m.selectedService = msg.Service
		if len(msg.AllKeys) > 0 {
			m.connectedKeys = msg.AllKeys
		} else if msg.Service != nil {
			key := msg.Service.Platform + "/" + msg.Service.RID
			m.connectedKeys = appendUnique(m.connectedKeys, key)
		}
		m.statusMessage = fmt.Sprintf("Connected to %s/%s (%d total)", msg.Service.Platform, msg.Service.RID, len(m.connectedKeys))

	case tuimsg.ConnectedSnapshotMsg:
		m.connectedKeys = msg.Keys
		if len(msg.Keys) == 0 {
			m.selectedService = nil
		} else {
			// keep selectedService if still present
			found := false
			if m.selectedService != nil {
				cur := m.selectedService.Platform + "/" + m.selectedService.RID
				for _, k := range msg.Keys {
					if k == cur {
						found = true
						break
					}
				}
			}
			if !found {
				parts := splitKey(msg.Keys[0])
				m.selectedService = &api.Service{Platform: parts[0], RID: parts[1]}
			}
		}

	case tuimsg.ServiceDisconnectedMsg:
		m.selectedService = nil
		m.connectedKeys = nil
		m.statusMessage = "Disconnected"

	case tuimsg.ErrorMsg:
		m.statusMessage = fmt.Sprintf("Error: %v", msg.Err)

	case tuimsg.SuccessMsg:
		m.statusMessage = msg.Message

	case tuimsg.UpdateServerConfigMsg:
		// 更新配置
		m.config.Server.APIAddress = msg.APIAddress
		m.config.Server.APIToken = msg.APIToken
		m.config.Server.WSAddress = msg.WSAddress
		// 不关闭弹窗，让用户可以继续编辑或手动按 Esc 关闭

	case tuimsg.UpdatePluginsConfigMsg:
		// 更新插件配置
		m.config.Pipeline.Plugins = msg.Plugins
		// 不关闭弹窗，让用户可以继续编辑或手动按 Esc 关闭
	}

	// 更新子模型
	var cmd tea.Cmd
	m.messagePanel, cmd = m.messagePanel.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	m.servicesPopup, cmd = m.servicesPopup.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	m.serverConfig, cmd = m.serverConfig.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	m.addService, cmd = m.addService.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	m.pluginsConfig, cmd = m.pluginsConfig.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View 渲染
func (m RootModel) View() string {
	// 顶部标题栏
	title := titleStyle.Width(m.width).Render("DMNotifier")

	// 连接信息（支持多路）
	connectionInfo := ""
	switch len(m.connectedKeys) {
	case 0:
		connectionInfo = dimStyle.Width(m.width).Render("Not connected - Press s to select service(s)")
	case 1:
		connectionInfo = infoStyle.Width(m.width).Render("Connected: " + m.connectedKeys[0])
	default:
		shown := m.connectedKeys
		if len(shown) > 4 {
			shown = append(shown[:4], fmt.Sprintf("+%d more", len(m.connectedKeys)-4))
		}
		connectionInfo = infoStyle.Width(m.width).Render(
			fmt.Sprintf("Connected (%d): %s", len(m.connectedKeys), joinComma(shown)),
		)
	}

	// 消息面板
	messagePanel := m.messagePanel.View()

	// 底部状态栏
	status := statusStyle.Width(m.width).Render(m.statusMessage)

	// 帮助栏
	help := helpStyle.Width(m.width).Render("a:Add | s:Services(multi) | c:Config | p:Plugins | r:Refresh | d:DisconnectAll | q:Quit")

	mainView := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		connectionInfo,
		messagePanel,
		status,
		help,
	)

	// 如果有弹窗，叠加显示
	if m.servicesPopup.IsVisible() {
		popupView := m.servicesPopup.View()
		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			popupView,
			lipgloss.WithWhitespaceChars(" "),
		)
	}

	if m.serverConfig.IsVisible() {
		popupView := m.serverConfig.View()
		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			popupView,
			lipgloss.WithWhitespaceChars(" "),
		)
	}

	if m.addService.IsVisible() {
		popupView := m.addService.View()
		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			popupView,
			lipgloss.WithWhitespaceChars(" "),
		)
	}

	if m.pluginsConfig.IsVisible() {
		popupView := m.pluginsConfig.View()
		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			popupView,
			lipgloss.WithWhitespaceChars(" "),
		)
	}

	return mainView
}

// GetConfig 获取配置
func (m RootModel) GetConfig() *AppConfig {
	return m.config
}

func appendUnique(ss []string, s string) []string {
	for _, x := range ss {
		if x == s {
			return ss
		}
	}
	return append(ss, s)
}

func splitKey(key string) [2]string {
	var out [2]string
	for i := 0; i < len(key); i++ {
		if key[i] == '/' {
			out[0] = key[:i]
			out[1] = key[i+1:]
			return out
		}
	}
	out[0] = key
	return out
}

func joinComma(ss []string) string {
	if len(ss) == 0 {
		return ""
	}
	out := ss[0]
	for i := 1; i < len(ss); i++ {
		out += ", " + ss[i]
	}
	return out
}
