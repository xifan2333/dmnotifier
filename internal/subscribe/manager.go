package subscribe

import (
	"context"
	"fmt"
	"sync"

	"github.com/xifan2333/dmnotifier/internal/client"
	"github.com/xifan2333/dmnotifier/internal/pipeline"
	"github.com/xifan2333/dmnotifier/pkg/api"
	"github.com/xifan2333/dmnotifier/pkg/models"
)

// Target 一个订阅目标（平台 + 房间）
type Target struct {
	Platform string
	RID      string
	Cookie   string
}

// Key platform/rid
func (t Target) Key() string {
	return t.Platform + "/" + t.RID
}

// Config 订阅管理器配置
type Config struct {
	APIAddress string
	APIToken   string
	WSAddress  string // e.g. ws://127.0.0.1:7777
	// BuildPipeline 构建消息处理管道；nil 则只收消息不处理
	BuildPipeline func() (*pipeline.Manager, error)
	// OnMessage 可选额外回调（在 pipeline 之后）
	OnMessage func(*models.Message)
	// OnStatus 可选状态回调
	OnStatus func(string)
	// OnError 可选错误回调
	OnError func(error)
}

// Manager 多路订阅管理器：可同时连多个 platform/rid
type Manager struct {
	cfg      Config
	api      *api.Client
	pipeline *pipeline.Manager

	mu      sync.RWMutex
	conns   map[string]*client.WSClient // key = platform/rid
	targets map[string]Target
}

// New 创建订阅管理器
func New(cfg Config) *Manager {
	return &Manager{
		cfg:     cfg,
		api:     api.NewClient(cfg.APIAddress, cfg.APIToken),
		conns:   make(map[string]*client.WSClient),
		targets: make(map[string]Target),
	}
}

// UpdateServer 更新 API/WS 地址（断开现有连接后由调用方重连）
func (m *Manager) UpdateServer(apiAddress, apiToken, wsAddress string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cfg.APIAddress = apiAddress
	m.cfg.APIToken = apiToken
	m.cfg.WSAddress = wsAddress
	m.api = api.NewClient(apiAddress, apiToken)
}

// API 暴露 API 客户端
func (m *Manager) API() *api.Client {
	return m.api
}

// ensurePipeline 懒初始化共享 pipeline
func (m *Manager) ensurePipeline() error {
	if m.pipeline != nil {
		return nil
	}
	if m.cfg.BuildPipeline == nil {
		m.pipeline = pipeline.NewManager()
		return nil
	}
	p, err := m.cfg.BuildPipeline()
	if err != nil {
		return err
	}
	m.pipeline = p
	return nil
}

func (m *Manager) status(s string) {
	if m.cfg.OnStatus != nil {
		m.cfg.OnStatus(s)
	}
}

func (m *Manager) errf(err error) {
	if err != nil && m.cfg.OnError != nil {
		m.cfg.OnError(err)
	}
}

// StartRemote 在 UniBarrage 上启动监听（HTTP POST）
func (m *Manager) StartRemote(platform, rid, cookie string) error {
	_, err := m.api.StartService(platform, rid, cookie)
	return err
}

// StopRemote 停止 UniBarrage 上的监听
func (m *Manager) StopRemote(platform, rid string) error {
	_, err := m.api.StopService(platform, rid)
	return err
}

// Connect 连接某个房间的 WS 并开始收消息（幂等）
func (m *Manager) Connect(t Target) error {
	if t.Platform == "" || t.RID == "" {
		return fmt.Errorf("platform and rid required")
	}
	key := t.Key()

	m.mu.Lock()
	if _, ok := m.conns[key]; ok {
		m.mu.Unlock()
		return nil // already connected
	}
	if err := m.ensurePipeline(); err != nil {
		m.mu.Unlock()
		return err
	}
	wsURL := fmt.Sprintf("%s/%s/%s", trimSlash(m.cfg.WSAddress), t.Platform, t.RID)
	pl := m.pipeline
	onMsg := m.cfg.OnMessage
	ws := client.NewWSClient(client.WSClientConfig{
		URL:             wsURL,
		EnableReconnect: true,
		Handler: func(msg *models.Message) error {
			if pl != nil {
				pl.Dispatch(context.Background(), msg)
			}
			if onMsg != nil {
				onMsg(msg)
			}
			return nil
		},
	})
	m.conns[key] = ws
	m.targets[key] = t
	m.mu.Unlock()

	if err := ws.Start(); err != nil {
		m.mu.Lock()
		delete(m.conns, key)
		delete(m.targets, key)
		m.mu.Unlock()
		return fmt.Errorf("ws start %s: %w", key, err)
	}
	m.status(fmt.Sprintf("connected %s", key))
	return nil
}

// Disconnect 断开某个订阅（不停止远端 UniBarrage）
func (m *Manager) Disconnect(platform, rid string) {
	key := platform + "/" + rid
	m.mu.Lock()
	ws, ok := m.conns[key]
	if ok {
		delete(m.conns, key)
		delete(m.targets, key)
	}
	empty := len(m.conns) == 0
	pl := m.pipeline
	if empty {
		m.pipeline = nil
	}
	m.mu.Unlock()

	if ok && ws != nil {
		_ = ws.Close()
		m.status(fmt.Sprintf("disconnected %s", key))
	}
	if empty && pl != nil {
		pl.Shutdown()
	}
}

// DisconnectAll 断开全部本地 WS
func (m *Manager) DisconnectAll() {
	m.mu.Lock()
	conns := m.conns
	m.conns = make(map[string]*client.WSClient)
	m.targets = make(map[string]Target)
	pl := m.pipeline
	m.pipeline = nil
	m.mu.Unlock()

	for key, ws := range conns {
		if ws != nil {
			_ = ws.Close()
			m.status(fmt.Sprintf("disconnected %s", key))
		}
	}
	if pl != nil {
		pl.Shutdown()
	}
}

// Subscribe = StartRemote + Connect（管道一步到位）
func (m *Manager) Subscribe(t Target) error {
	if err := m.StartRemote(t.Platform, t.RID, t.Cookie); err != nil {
		// 若已在监听，UniBarrage 会报已存在——仍尝试连 WS
		// 只有非“已存在”类错误才真正失败时，继续尝试 connect 更利于幂等
		m.status(fmt.Sprintf("start remote %s: %v (still connecting ws)", t.Key(), err))
	}
	return m.Connect(t)
}

// Unsubscribe = Disconnect + StopRemote
func (m *Manager) Unsubscribe(platform, rid string) error {
	m.Disconnect(platform, rid)
	if err := m.StopRemote(platform, rid); err != nil {
		return err
	}
	m.status(fmt.Sprintf("unsubscribed %s/%s", platform, rid))
	return nil
}

// List 当前已连接目标
func (m *Manager) List() []Target {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Target, 0, len(m.targets))
	for _, t := range m.targets {
		out = append(out, t)
	}
	return out
}

// IsConnected 是否已连
func (m *Manager) IsConnected(platform, rid string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.conns[platform+"/"+rid]
	return ok
}

// ConnectedCount 连接数
func (m *Manager) ConnectedCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.conns)
}

// Close 释放全部
func (m *Manager) Close() {
	m.DisconnectAll()
}

func trimSlash(s string) string {
	for len(s) > 0 && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}
	return s
}
