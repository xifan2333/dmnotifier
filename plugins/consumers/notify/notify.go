package notify

import (
	"context"
	"fmt"

	"github.com/gen2brain/beeep"
	"github.com/xifan2333/dmnotifier/internal/platformicons"
	"github.com/xifan2333/dmnotifier/internal/plugin"
	"github.com/xifan2333/dmnotifier/pkg/models"
	"github.com/xifan2333/dmnotifier/plugins/transforms/format"
)

// Consumer 系统通知
// 图标：用户头像（下载缓存）→ 嵌入平台 logo（internal/platformicons）
type Consumer struct {
	*plugin.BasePlugin
	avatarCache *AvatarCache
	ctx         context.Context
	cancel      context.CancelFunc
}

func New() plugin.Plugin {
	return &Consumer{BasePlugin: plugin.NewBasePlugin("notify", plugin.TypeConsumer)}
}

func (c *Consumer) Init(ctx context.Context, config map[string]interface{}) error {
	if err := c.BasePlugin.Init(ctx, config); err != nil {
		return err
	}
	c.ctx, c.cancel = context.WithCancel(context.Background())
	// warm extract embedded icons
	_ = platformicons.Path("bilibili")

	if ac, err := NewAvatarCache(""); err == nil {
		c.avatarCache = ac
	}
	return nil
}

func (c *Consumer) Consume(ctx context.Context, msg *models.Message) error {
	select {
	case <-c.ctx.Done():
		return nil
	default:
	}
	formatted, ok := msg.Data.(*models.FormattedMessage)
	if !ok {
		return nil
	}
	title := fmt.Sprintf("%s | %s", formatted.Platform, formatted.UserName)
	icon := resolveIcon(c.avatarCache, formatted.Avatar, formatted.Platform)
	return beeep.Notify(title, formatted.Content, icon)
}

func resolveIcon(cache *AvatarCache, avatar, platform string) string {
	if p, ok := format.ParsePlatformIconRef(avatar); ok {
		if path := platformicons.Path(p); path != "" {
			return path
		}
	}
	if cache != nil && len(avatar) > 4 && avatar[:4] == "http" {
		if path := cache.Get(avatar); path != "" {
			return path
		}
	}
	return platformicons.Path(platform)
}

func (c *Consumer) Stop(ctx context.Context) error {
	if c.cancel != nil {
		c.cancel()
	}
	return nil
}

func init() {
	plugin.Register("notify", New, plugin.PluginInfo{
		Name:           "notify",
		Type:           plugin.TypeConsumer,
		ConfigTemplate: []plugin.ConfigField{},
	})
}
