package message_type

import (
	"context"

	"github.com/xifan2333/dmnotifier/internal/plugin"
	"github.com/xifan2333/dmnotifier/pkg/models"
)

// Filter 消息类型过滤器
type Filter struct {
	*plugin.BasePlugin
	allowedTypes map[models.MessageType]bool
}

// New 创建消息类型过滤器
func New() plugin.Plugin {
	return &Filter{
		BasePlugin:   plugin.NewBasePlugin("message_type_filter", plugin.TypeFilter),
		allowedTypes: make(map[models.MessageType]bool),
	}
}

// Init 初始化插件
func (f *Filter) Init(ctx context.Context, config map[string]interface{}) error {
	if err := f.BasePlugin.Init(ctx, config); err != nil {
		return err
	}

	// 从配置中读取允许的消息类型
	if types, ok := config["types"].([]interface{}); ok {
		for _, t := range types {
			if typeStr, ok := t.(string); ok {
				f.allowedTypes[models.MessageType(typeStr)] = true
			}
		}
	}

	// 如果没有配置，则允许所有类型
	if len(f.allowedTypes) == 0 {
		f.allowedTypes[models.TypeChat] = true
		f.allowedTypes[models.TypeGift] = true
		f.allowedTypes[models.TypeLike] = true
		f.allowedTypes[models.TypeEnterRoom] = true
		f.allowedTypes[models.TypeSubscribe] = true
		f.allowedTypes[models.TypeSuperChat] = true
		f.allowedTypes[models.TypeEndLive] = true
	}

	return nil
}

// Filter 过滤消息
func (f *Filter) Filter(ctx context.Context, msg *models.Message) bool {
	return f.allowedTypes[msg.Type]
}

func init() {
	plugin.Register("message_type_filter", New, plugin.PluginInfo{
		Name:           "message_type_filter",
		Type:           plugin.TypeFilter,
		ConfigTemplate: []plugin.ConfigField{},
	})
}
