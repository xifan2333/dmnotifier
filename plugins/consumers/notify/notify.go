package notify

import (
	"context"
	"fmt"

	"github.com/gen2brain/beeep"
	"github.com/xifan2333/dmnotifier/internal/plugin"
	"github.com/xifan2333/dmnotifier/pkg/models"
	"github.com/xifan2333/dmnotifier/plugins/transforms/format"
)

// Consumer 系统通知消费者
type Consumer struct {
	*plugin.BasePlugin

	// 头像缓存
	avatarCache *AvatarCache

	// 上下文控制
	ctx    context.Context
	cancel context.CancelFunc
}

// New 创建系统通知消费者
func New() plugin.Plugin {
	return &Consumer{
		BasePlugin: plugin.NewBasePlugin("notify", plugin.TypeConsumer),
	}
}

// Init 初始化插件
func (c *Consumer) Init(ctx context.Context, config map[string]interface{}) error {
	if err := c.BasePlugin.Init(ctx, config); err != nil {
		return err
	}

	// 创建上下文
	c.ctx, c.cancel = context.WithCancel(context.Background())

	// 系统临时目录：$TMPDIR/dmnotifier/avatars
	avatarCache, err := NewAvatarCache("")
	if err != nil {

		// 继续运行，只是不缓存头像
	} else {
		c.avatarCache = avatarCache

	}

	return nil
}

// Consume 消费消息
func (c *Consumer) Consume(ctx context.Context, msg *models.Message) error {
	// 检查是否已停止
	select {
	case <-c.ctx.Done():
		return nil
	default:
	}

	// 只处理格式化后的消息
	formatted, ok := msg.Data.(*models.FormattedMessage)
	if !ok {
		return nil
	}

	title := fmt.Sprintf("%s | %s", formatted.Platform, formatted.UserName)
	message := formatted.Content

	return c.sendNotification(title, message, formatted.Avatar, formatted.Platform)
}

// sendNotification 发送系统通知。
// 图标优先级：用户头像 → 平台 simple-icons PNG → 无图标。
func (c *Consumer) sendNotification(title, message, iconURL, platform string) error {
	iconPath := ""
	if c.avatarCache != nil {
		if iconURL != "" {
			iconPath = c.avatarCache.Get(iconURL)
		}
		// 头像缺失或下载失败 → 平台 logo（simple-icons PNG）
		if iconPath == "" {
			if logo := format.PlatformIcon(platform); logo != "" && logo != iconURL {
				iconPath = c.avatarCache.Get(logo)
			}
		}
	}

	return beeep.Notify(title, message, iconPath)
}

// Stop 停止插件
func (c *Consumer) Stop(ctx context.Context) error {
	// 取消上下文，阻止新的通知发送
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
