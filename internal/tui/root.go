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

	// 当前连接的服务
	selectedService *api.Service

	// 配置
	config *AppConfig

	// 状态
	statusMessage string
	width         int
	height        int
}

// NewRootModel 创建根模型
func NewRootModel() RootModel {
	// 加载配置
	config, err := LoadConfig()
	if err != nil {
		config = GetDefaultConfig()
	}

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
			// 断开连接
			if m.selectedService != nil {
				return m, func() tea.Msg {
					return tuimsg.DisconnectServiceRequestMsg{}
				}
			}

		case "ctrl+s":
			// 保存配置
			return m, func() tea.Msg {
				return tuimsg.SaveConfigRequestMsg{}
			}
		}

	case tuimsg.StatusMsg:
		m.statusMessage = msg.Message

	case tuimsg.ServiceConnectedMsg:
		m.selectedService = msg.Service
		m.statusMessage = fmt.Sprintf("Connected to %s/%s", msg.Service.Platform, msg.Service.RID)

	case tuimsg.ServiceDisconnectedMsg:
		m.selectedService = nil
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
		m.statusMessage = "Server config updated"

		// 关闭弹窗
		m.serverConfig, _ = m.serverConfig.Update(tuimsg.HidePopupMsg{})

	case tuimsg.UpdatePluginsConfigMsg:
		// 更新插件配置
		m.config.Pipeline.Plugins = msg.Plugins
		m.statusMessage = "Plugins config updated"
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

	// 连接信息
	connectionInfo := ""
	if m.selectedService != nil {
		connectionInfo = infoStyle.Width(m.width).Render(
			fmt.Sprintf("Connected: %s/%s", m.selectedService.Platform, m.selectedService.RID),
		)
	} else {
		connectionInfo = dimStyle.Width(m.width).Render("Not connected - Press s to select service")
	}

	// 消息面板
	messagePanel := m.messagePanel.View()

	// 底部状态栏
	status := statusStyle.Width(m.width).Render(m.statusMessage)

	// 帮助栏
	help := helpStyle.Width(m.width).Render("a:Add | s:Services | c:Config | p:Plugins | r:Refresh | d:Disconnect | Ctrl+S:Save | q:Quit")

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
