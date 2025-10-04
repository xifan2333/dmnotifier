package notify

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gen2brain/beeep"
	"github.com/xifan2333/dmnotifier/internal/plugin"
	"github.com/xifan2333/dmnotifier/pkg/models"
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

	// 使用系统临时目录
	cacheDir := filepath.Join(os.TempDir(), "dmnotifier", "avatars")

	avatarCache, err := NewAvatarCache(cacheDir)
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

	return c.sendNotification(title, message, formatted.Avatar)
}

// sendNotification 发送系统通知
func (c *Consumer) sendNotification(title, message, iconURL string) error {
	// 获取头像本地路径
	iconPath := ""
	if c.avatarCache != nil && iconURL != "" {
		iconPath = c.avatarCache.Get(iconURL)
	}

	// 使用 beeep 发送跨平台通知
	err := beeep.Notify(title, message, iconPath)
	if err != nil {

		return err
	}

	return nil
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
