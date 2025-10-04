package plugin

import (
	"context"
	"fmt"
	"sync"
)

// PluginFactory 插件工厂函数
type PluginFactory func() Plugin

// Registry 插件注册中心
type Registry struct {
	factories map[string]PluginFactory
	infos     map[string]PluginInfo // 缓存插件信息
	mu        sync.RWMutex
}

// NewRegistry 创建新的插件注册中心
func NewRegistry() *Registry {
	return &Registry{
		factories: make(map[string]PluginFactory),
		infos:     make(map[string]PluginInfo),
	}
}

// Register 注册插件工厂和信息
func (r *Registry) Register(name string, factory PluginFactory, info PluginInfo) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.factories[name]; exists {
		return fmt.Errorf("plugin %s already registered", name)
	}

	r.factories[name] = factory
	r.infos[name] = info

	return nil
}

// Create 创建插件实例
func (r *Registry) Create(name string) (Plugin, error) {
	r.mu.RLock()
	factory, exists := r.factories[name]
	r.mu.RUnlock()

	if !exists {
		// 列出所有可用插件，帮助调试
		available := r.List()
		return nil, fmt.Errorf("plugin %s not found (available: %v)", name, available)
	}

	return factory(), nil
}

// List 列出所有已注册的插件
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.factories))
	for name := range r.factories {
		names = append(names, name)
	}
	return names
}

// PluginInfo 插件信息
type PluginInfo struct {
	Name           string
	Type           PluginType
	ConfigTemplate []ConfigField
}

// GetAllPluginInfo 获取所有插件的信息
func (r *Registry) GetAllPluginInfo() []PluginInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	infos := make([]PluginInfo, 0, len(r.infos))
	for _, info := range r.infos {
		infos = append(infos, info)
	}
	return infos
}

// GetAllConsumerPluginInfo 获取所有消费者插件的信息
func (r *Registry) GetAllConsumerPluginInfo() []PluginInfo {
	allInfos := r.GetAllPluginInfo()
	consumerInfos := make([]PluginInfo, 0)
	for _, info := range allInfos {
		if info.Type == TypeConsumer {
			consumerInfos = append(consumerInfos, info)
		}
	}
	return consumerInfos
}

// Has 检查插件是否已注册
func (r *Registry) Has(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.factories[name]
	return exists
}

// GlobalRegistry 全局插件注册中心
var GlobalRegistry = NewRegistry()

// Register 注册插件到全局注册中心
func Register(name string, factory PluginFactory, info PluginInfo) error {
	return GlobalRegistry.Register(name, factory, info)
}

// Create 从全局注册中心创建插件
func Create(name string) (Plugin, error) {
	return GlobalRegistry.Create(name)
}

// PluginManager 插件管理器
type PluginManager struct {
	registry *Registry
	plugins  map[string]Plugin
	mu       sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewPluginManager 创建插件管理器
func NewPluginManager(registry *Registry) *PluginManager {
	if registry == nil {
		registry = GlobalRegistry
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &PluginManager{
		registry: registry,
		plugins:  make(map[string]Plugin),
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Load 加载插件
func (m *PluginManager) Load(name string, config map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查是否已加载
	if _, exists := m.plugins[name]; exists {
		return fmt.Errorf("plugin %s already loaded", name)
	}

	// 创建插件实例
	plugin, err := m.registry.Create(name)
	if err != nil {
		return fmt.Errorf("create plugin %s: %w", name, err)
	}

	// 初始化插件
	if err := plugin.Init(m.ctx, config); err != nil {
		return fmt.Errorf("init plugin %s: %w", name, err)
	}

	m.plugins[name] = plugin
	return nil
}

// Start 启动插件
func (m *PluginManager) Start(name string) error {
	m.mu.RLock()
	plugin, exists := m.plugins[name]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("plugin %s not loaded", name)
	}

	return plugin.Start(m.ctx)
}

// Stop 停止插件
func (m *PluginManager) Stop(name string) error {
	m.mu.RLock()
	plugin, exists := m.plugins[name]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("plugin %s not loaded", name)
	}

	return plugin.Stop(m.ctx)
}

// Get 获取插件实例
func (m *PluginManager) Get(name string) (Plugin, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	plugin, exists := m.plugins[name]
	if !exists {
		return nil, fmt.Errorf("plugin %s not loaded", name)
	}

	return plugin, nil
}

// GetAll 获取所有已加载的插件
func (m *PluginManager) GetAll() []Plugin {
	m.mu.RLock()
	defer m.mu.RUnlock()

	plugins := make([]Plugin, 0, len(m.plugins))
	for _, plugin := range m.plugins {
		plugins = append(plugins, plugin)
	}
	return plugins
}

// StopAll 停止所有插件
func (m *PluginManager) StopAll() error {
	m.mu.RLock()
	plugins := make([]Plugin, 0, len(m.plugins))
	for _, plugin := range m.plugins {
		plugins = append(plugins, plugin)
	}
	m.mu.RUnlock()

	var lastErr error
	for _, plugin := range plugins {
		if err := plugin.Stop(m.ctx); err != nil {
			lastErr = err
		}
	}

	m.cancel()
	return lastErr
}
