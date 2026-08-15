package business

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	tuimsg "github.com/xifan2333/dmnotifier/internal/common"
	"github.com/xifan2333/dmnotifier/internal/pipeline"
	"github.com/xifan2333/dmnotifier/internal/subscribe"
	"github.com/xifan2333/dmnotifier/internal/config"
	"github.com/xifan2333/dmnotifier/internal/tui/components"
	"github.com/xifan2333/dmnotifier/pkg/api"
	"github.com/xifan2333/dmnotifier/pkg/models"
)

// Manager 业务逻辑管理器（支持同时订阅多个平台/房间）
type Manager struct {
	program *tea.Program
	sub     *subscribe.Manager
	config  *config.AppConfig
}

const maxHistorySize = 50

func (m *Manager) addToHistory(platform, rid, cookie string) {
	entry := tuimsg.ServiceHistoryEntry{Platform: platform, RID: rid, Cookie: cookie}
	filtered := make([]tuimsg.ServiceHistoryEntry, 0, len(m.config.History)+1)
	filtered = append(filtered, entry)
	for _, h := range m.config.History {
		if h.Platform == platform && h.RID == rid {
			continue
		}
		filtered = append(filtered, h)
	}
	if len(filtered) > maxHistorySize {
		filtered = filtered[:maxHistorySize]
	}
	m.config.History = filtered
}

func (m *Manager) removeFromHistory(platform, rid string) {
	filtered := make([]tuimsg.ServiceHistoryEntry, 0, len(m.config.History))
	for _, h := range m.config.History {
		if h.Platform == platform && h.RID == rid {
			continue
		}
		filtered = append(filtered, h)
	}
	m.config.History = filtered
}

func (m *Manager) newSub() *subscribe.Manager {
	cfg := m.config
	prog := m.program
	return subscribe.New(subscribe.Config{
		APIAddress: cfg.Server.APIAddress,
		APIToken:   cfg.Server.APIToken,
		WSAddress:  cfg.Server.WSAddress,
		BuildPipeline: func() (*pipeline.Manager, error) {
			return BuildPipelines(cfg, prog)
		},
		OnStatus: func(s string) {
			if prog != nil {
				prog.Send(tuimsg.StatusMsg{Message: s})
			}
		},
		OnError: func(err error) {
			if prog != nil {
				prog.Send(tuimsg.ErrorMsg{Err: err})
			}
		},
	})
}

// NewManager 创建业务逻辑管理器
func NewManager(program *tea.Program, cfg *config.AppConfig) *Manager {
	m := &Manager{program: program, config: cfg}
	m.sub = m.newSub()
	return m
}

// GetAPIClient 获取 API 客户端
func (m *Manager) GetAPIClient() *api.Client {
	return m.sub.API()
}

// UpdateServerConfig 更新服务器配置
func (m *Manager) UpdateServerConfig(apiAddress, apiToken, wsAddress string) {
	m.config.Server.APIAddress = apiAddress
	m.config.Server.APIToken = apiToken
	m.config.Server.WSAddress = wsAddress
	// 重建 sub（保留不自动重连旧房间，调用方需重连）
	m.sub.Close()
	m.sub = m.newSub()
}

// UpdatePluginsConfig 更新插件配置。
// 已建立的订阅仍使用旧 pipeline；需断开后重新连接才会按新配置生效。
func (m *Manager) UpdatePluginsConfig(plugins []tuimsg.PluginConfig) {
	m.config.Pipeline.Plugins = plugins
}

// GetConfig 获取配置
func (m *Manager) GetConfig() *config.AppConfig {
	return m.config
}

// ConnectedTargets 当前已连接列表
func (m *Manager) ConnectedTargets() []subscribe.Target {
	return m.sub.List()
}

// FetchServices 获取服务列表
func (m *Manager) FetchServices() tea.Cmd {
	return func() tea.Msg {
		services, err := m.sub.API().GetAllServices()
		if err != nil {
			return tuimsg.ErrorMsg{Err: err}
		}
		return tuimsg.ServicesLoadedMsg{
			Services:  services,
			History:   m.config.History,
			Connected: m.connectedKeys(),
		}
	}
}

func (m *Manager) connectedKeys() []string {
	ts := m.sub.List()
	out := make([]string, 0, len(ts))
	for _, t := range ts {
		out = append(out, t.Key())
	}
	return out
}

func (m *Manager) emitConnectedSnapshot() {
	keys := m.connectedKeys()
	m.program.Send(tuimsg.ConnectedSnapshotMsg{Keys: keys})
	if len(keys) == 0 {
		m.program.Send(tuimsg.ServiceDisconnectedMsg{})
		return
	}
	// 用第一个作为“主”显示；完整列表在 snapshot
	parts := strings.SplitN(keys[0], "/", 2)
	plat, rid := parts[0], ""
	if len(parts) > 1 {
		rid = parts[1]
	}
	m.program.Send(tuimsg.ServiceConnectedMsg{
		Service:   &api.Service{Platform: plat, RID: rid},
		AllKeys:   keys,
		Connected: len(keys),
	})
}

// StopService 停止远端服务；若本地已订阅则一并断开
func (m *Manager) StopService(platform, rid string) tea.Cmd {
	return func() tea.Msg {
		if m.sub.IsConnected(platform, rid) {
			m.sub.Disconnect(platform, rid)
			m.emitConnectedSnapshot()
		}
		_, err := m.sub.API().StopService(platform, rid)
		if err != nil {
			return tuimsg.ErrorMsg{Err: err}
		}
		services, err := m.sub.API().GetAllServices()
		if err != nil {
			return tuimsg.ErrorMsg{Err: err}
		}
		return tuimsg.ServicesLoadedMsg{
			Services:  services,
			History:   m.config.History,
			Connected: m.connectedKeys(),
		}
	}
}

// AddService 添加远端服务（不自动连接）
func (m *Manager) AddService(platform, rid, cookie string) tea.Cmd {
	return func() tea.Msg {
		platform = normalizePlatform(platform)
		_, err := m.sub.API().StartService(platform, rid, cookie)
		if err != nil {
			return tuimsg.ErrorMsg{Err: err}
		}
		m.addToHistory(platform, rid, cookie)
		m.program.Send(tuimsg.SaveConfigRequestMsg{})
		services, err := m.sub.API().GetAllServices()
		if err != nil {
			return tuimsg.ErrorMsg{Err: err}
		}
		m.program.Send(tuimsg.StatusMsg{Message: fmt.Sprintf("Service %s/%s added", platform, rid)})
		return tuimsg.ServicesLoadedMsg{
			Services:  services,
			History:   m.config.History,
			Connected: m.connectedKeys(),
		}
	}
}

// ReuseHistory 从历史重新 start + connect（可叠加多路）
func (m *Manager) ReuseHistory(platform, rid, cookie string) tea.Cmd {
	return func() tea.Msg {
		platform = normalizePlatform(platform)
		m.addToHistory(platform, rid, cookie)
		m.program.Send(tuimsg.SaveConfigRequestMsg{})
		services, _ := m.sub.API().GetAllServices()
		m.program.Send(tuimsg.ServicesLoadedMsg{
			Services:  services,
			History:   m.config.History,
			Connected: m.connectedKeys(),
		})
		svc := &api.Service{Platform: platform, RID: rid}
		// 带 cookie 的连接请求
		m.program.Send(tuimsg.ConnectServiceRequestMsg{Service: svc, Cookie: cookie})
		return tuimsg.StatusMsg{Message: fmt.Sprintf("Reusing %s/%s from history...", platform, rid)}
	}
}

// DeleteFromHistory 从历史删除
func (m *Manager) DeleteFromHistory(platform, rid string) tea.Cmd {
	return func() tea.Msg {
		m.removeFromHistory(platform, rid)
		m.program.Send(tuimsg.SaveConfigRequestMsg{})
		services, err := m.sub.API().GetAllServices()
		if err != nil {
			return tuimsg.ServicesLoadedMsg{Services: nil, History: m.config.History, Connected: m.connectedKeys()}
		}
		return tuimsg.ServicesLoadedMsg{Services: services, History: m.config.History, Connected: m.connectedKeys()}
	}
}

// ConnectToService 连接到服务（多路叠加，不踢掉已有连接）
func (m *Manager) ConnectToService(service *api.Service, cookie string) tea.Cmd {
	go func() {
		t := subscribe.Target{
			Platform: normalizePlatform(service.Platform),
			RID:      service.RID,
			Cookie:   cookie,
		}
		// 确保远端在听
		if err := m.sub.StartRemote(t.Platform, t.RID, t.Cookie); err != nil {
			// 可能已在听，继续 connect
			m.program.Send(tuimsg.StatusMsg{Message: fmt.Sprintf("start: %v", err)})
		}
		if err := m.sub.Connect(t); err != nil {
			m.program.Send(tuimsg.ErrorMsg{Err: fmt.Errorf("connect %s: %w", t.Key(), err)})
			return
		}
		m.addToHistory(t.Platform, t.RID, t.Cookie)
		m.program.Send(tuimsg.SaveConfigRequestMsg{})
		m.program.Send(tuimsg.ConnectSuccessMsg{Service: &api.Service{Platform: t.Platform, RID: t.RID}})
		m.emitConnectedSnapshot()
	}()
	return func() tea.Msg {
		return tuimsg.StatusMsg{Message: fmt.Sprintf("Connecting to %s/%s...", service.Platform, service.RID)}
	}
}

// DisconnectService 断开全部本地订阅
func (m *Manager) DisconnectService() {
	go func() {
		m.sub.DisconnectAll()
		m.program.Send(tuimsg.ServiceDisconnectedMsg{})
		m.program.Send(tuimsg.ConnectedSnapshotMsg{Keys: nil})
		m.program.Send(components.AddMessageMsg{Content: "[Disconnected all]"})
	}()
}

// DisconnectOne 断开单路
func (m *Manager) DisconnectOne(platform, rid string) tea.Cmd {
	return func() tea.Msg {
		m.sub.Disconnect(platform, rid)
		m.emitConnectedSnapshot()
		return tuimsg.StatusMsg{Message: fmt.Sprintf("Disconnected %s/%s", platform, rid)}
	}
}

// Cleanup 清理资源
func (m *Manager) Cleanup() {
	if m.sub != nil {
		m.sub.Close()
	}
}

func normalizePlatform(s string) string {
	if p, ok := models.NormalizePlatform(s); ok {
		return string(p)
	}
	return strings.ToLower(strings.TrimSpace(s))
}
