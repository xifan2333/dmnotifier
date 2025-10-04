package common

import "github.com/xifan2333/dmnotifier/pkg/api"

// PluginConfig 插件配置
type PluginConfig struct {
	Name         string                 `yaml:"name"`
	Enabled      bool                   `yaml:"enabled"`
	MessageTypes []string               `yaml:"messagetypes"`
	Config       map[string]interface{} `yaml:"config,omitempty"`
}

// UI 消息类型
type ShowServicesPopupMsg struct{}
type ShowServerConfigPopupMsg struct{}
type ShowPluginsConfigPopupMsg struct {
	Plugins []PluginConfig
}
type ShowAddServicePopupMsg struct{}
type HidePopupMsg struct{}

// 数据消息类型
type ServicesLoadedMsg struct {
	Services []api.Service
}

type ServiceConnectedMsg struct {
	Service *api.Service
}

type ServiceDisconnectedMsg struct{}

// 内部消息类型
type ConnectSuccessMsg struct {
	Service *api.Service
}

// 请求消息类型（发送给 main.go 处理）
type ConnectServiceRequestMsg struct {
	Service *api.Service
}

type DisconnectServiceRequestMsg struct{}

type StopServiceRequestMsg struct {
	Platform string
	RID      string
}

type RefreshServicesRequestMsg struct{}

type SaveConfigRequestMsg struct{}

// 配置更新消息
type UpdateServerConfigMsg struct {
	APIAddress string
	APIToken   string
	WSAddress  string
}

type UpdatePluginsConfigMsg struct {
	Plugins []PluginConfig
}

type AddServiceRequestMsg struct {
	Platform string
	RID      string
	Cookie   string
}

// 状态消息类型
type StatusMsg struct {
	Message string
}

type ErrorMsg struct {
	Err error
}

type SuccessMsg struct {
	Message string
}
