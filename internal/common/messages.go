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

// ServiceHistoryEntry 历史使用过的服务记录
type ServiceHistoryEntry struct {
	Platform string `yaml:"platform"`
	RID      string `yaml:"rid"`
	Cookie   string `yaml:"cookie,omitempty"`
}

// 数据消息类型
type ServicesLoadedMsg struct {
	Services  []api.Service
	History   []ServiceHistoryEntry
	Connected []string // platform/rid currently subscribed locally
}

type ServiceConnectedMsg struct {
	Service   *api.Service
	AllKeys   []string // all connected keys after this event
	Connected int
}

type ServiceDisconnectedMsg struct{}

// ConnectedSnapshotMsg 多路订阅快照（状态栏用）
type ConnectedSnapshotMsg struct {
	Keys []string
}

// 内部消息类型
type ConnectSuccessMsg struct {
	Service *api.Service
}

// 请求消息类型（发送给 main.go 处理）
type ConnectServiceRequestMsg struct {
	Service *api.Service
	Cookie  string // optional, used when starting remote if needed
}

type DisconnectServiceRequestMsg struct{}

// DisconnectOneRequestMsg 断开单路本地订阅
type DisconnectOneRequestMsg struct {
	Platform string
	RID      string
}

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

// ReuseHistoryRequestMsg 从历史记录重新启用并连接
type ReuseHistoryRequestMsg struct {
	Platform string
	RID      string
	Cookie   string
}

// DeleteHistoryEntryRequestMsg 从历史记录中删除一条
type DeleteHistoryEntryRequestMsg struct {
	Platform string
	RID      string
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
