package business

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/xifan/dmnotifier/internal/client"
	tuimsg "github.com/xifan/dmnotifier/internal/common"
	"github.com/xifan/dmnotifier/internal/pipeline"
	"github.com/xifan/dmnotifier/internal/tui"
	"github.com/xifan/dmnotifier/internal/tui/components"
	"github.com/xifan/dmnotifier/pkg/api"
	"github.com/xifan/dmnotifier/pkg/models"
)

// Manager 业务逻辑管理器
type Manager struct {
	program         *tea.Program
	apiClient       *api.Client
	wsClient        *client.WSClient
	pipelineManager *pipeline.Manager
	config          *tui.AppConfig
}

// NewManager 创建业务逻辑管理器
func NewManager(program *tea.Program, config *tui.AppConfig) *Manager {
	apiClient := api.NewClient(config.Server.APIAddress, config.Server.APIToken)
	return &Manager{
		program:   program,
		apiClient: apiClient,
		config:    config,
	}
}

// GetAPIClient 获取 API 客户端
func (m *Manager) GetAPIClient() *api.Client {
	return m.apiClient
}

// UpdateServerConfig 更新服务器配置
func (m *Manager) UpdateServerConfig(apiAddress, apiToken, wsAddress string) {
	m.config.Server.APIAddress = apiAddress
	m.config.Server.APIToken = apiToken
	m.config.Server.WSAddress = wsAddress
	m.apiClient = api.NewClient(apiAddress, apiToken)
}

// UpdatePluginsConfig 更新插件配置
func (m *Manager) UpdatePluginsConfig(plugins []tuimsg.PluginConfig) {
	m.config.Pipeline.Plugins = plugins
}

// GetConfig 获取配置
func (m *Manager) GetConfig() *tui.AppConfig {
	return m.config
}

// FetchServices 获取服务列表
func (m *Manager) FetchServices() tea.Cmd {
	return func() tea.Msg {
		services, err := m.apiClient.GetAllServices()
		if err != nil {
			return tuimsg.ErrorMsg{Err: err}
		}
		return tuimsg.ServicesLoadedMsg{Services: services}
	}
}

// StopService 停止服务
func (m *Manager) StopService(platform, rid string) tea.Cmd {
	return func() tea.Msg {
		_, err := m.apiClient.StopService(platform, rid)
		if err != nil {
			return tuimsg.ErrorMsg{Err: err}
		}
		// 刷新服务列表
		services, err := m.apiClient.GetAllServices()
		if err != nil {
			return tuimsg.ErrorMsg{Err: err}
		}
		return tuimsg.ServicesLoadedMsg{Services: services}
	}
}

// AddService 添加服务
func (m *Manager) AddService(platform, rid, cookie string) tea.Cmd {
	return func() tea.Msg {
		_, err := m.apiClient.StartService(platform, rid, cookie)
		if err != nil {
			return tuimsg.ErrorMsg{Err: err}
		}
		// 刷新服务列表
		services, err := m.apiClient.GetAllServices()
		if err != nil {
			return tuimsg.ErrorMsg{Err: err}
		}
		m.program.Send(tuimsg.StatusMsg{Message: fmt.Sprintf("Service %s/%s added", platform, rid)})
		return tuimsg.ServicesLoadedMsg{Services: services}
	}
}

// ConnectToService 连接到服务（返回 Cmd）
func (m *Manager) ConnectToService(service *api.Service) tea.Cmd {
	// 在独立 goroutine 中执行所有初始化，避免阻塞 TUI
	go func() {
		// 如果已有连接，先断开
		if m.wsClient != nil {
			m.disconnectSync()
		}

		// 异步构建 pipeline 管理器，传入 program 实例
		pipelineManager, err := BuildPipelines(m.config, m.program)
		if err != nil {
			pipelineManager = pipeline.NewManager()
		}
		m.pipelineManager = pipelineManager

		// 构建 WebSocket URL
		wsURL := fmt.Sprintf("%s/%s/%s", m.config.Server.WSAddress, service.Platform, service.RID)

		// 创建 WebSocket 客户端
		m.wsClient = client.NewWSClient(client.WSClientConfig{
			URL:             wsURL,
			EnableReconnect: true,
			Handler: func(msg *models.Message) error {
				// 分发消息到 pipeline（包括 TUI 插件）
				if m.pipelineManager != nil {
					m.pipelineManager.Dispatch(context.Background(), msg)
				}

				return nil
			},
		})

		// 启动 WebSocket 连接
		if err := m.wsClient.Start(); err != nil {
			m.wsClient = nil
			if m.pipelineManager != nil {
				m.pipelineManager.Shutdown()
				m.pipelineManager = nil
			}
			m.program.Send(tuimsg.ErrorMsg{Err: fmt.Errorf("failed to start WebSocket client: %w", err)})
			return
		}

		// 连接成功，发送成功消息
		m.program.Send(tuimsg.ConnectSuccessMsg{Service: service})
	}()

	// 立即返回 Cmd，不阻塞事件循环
	return func() tea.Msg {
		return tuimsg.StatusMsg{Message: fmt.Sprintf("Connecting to %s/%s...", service.Platform, service.RID)}
	}
}

// DisconnectService 断开服务连接
func (m *Manager) DisconnectService() {
	// 在独立 goroutine 中执行断开操作和发送消息，避免阻塞 TUI
	go func() {
		m.disconnectSync()

		// 断开完成后通知 TUI
		m.program.Send(tuimsg.ServiceDisconnectedMsg{})
		m.program.Send(components.AddMessageMsg{Content: "[Disconnected]"})
	}()
}

// disconnectSync 同步断开连接（内部使用）
func (m *Manager) disconnectSync() {
	if m.wsClient != nil {
		m.wsClient.Close()
		m.wsClient = nil
	}

	if m.pipelineManager != nil {
		m.pipelineManager.Shutdown()
		m.pipelineManager = nil
	}
}

// Cleanup 清理资源
func (m *Manager) Cleanup() {
	m.DisconnectService()
}
