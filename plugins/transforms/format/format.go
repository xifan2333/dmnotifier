package format

import (
	"context"
	"fmt"
	"time"

	"github.com/xifan2333/dmnotifier/internal/plugin"
	"github.com/xifan2333/dmnotifier/pkg/models"
)

// favicon returns a PNG favicon URL for a site (works with notify-send/mako).
func favicon(domain string) string {
	return "https://www.google.com/s2/favicons?domain=" + domain + "&sz=128"
}

// PlatformIcons maps platform id → favicon PNG (default avatar when user has none).
var PlatformIcons = map[string]string{
	"bilibili":    favicon("bilibili.com"),
	"douyin":      favicon("douyin.com"),
	"xiaohongshu": favicon("xiaohongshu.com"),
	"xhs":         favicon("xiaohongshu.com"),
	"kuaishou":    favicon("kuaishou.com"),
	"douyu":       favicon("douyu.com"),
	"huya":        favicon("huya.com"),
	"youtube":     favicon("youtube.com"),
	"twitch":      favicon("twitch.tv"),
}

// DefaultUserAvatar generic fallback favicon.
const DefaultUserAvatar = "https://www.google.com/s2/favicons?domain=live.com&sz=128"

// Transform 格式化转换器
type Transform struct {
	*plugin.BasePlugin
}

// New 创建格式化转换器
func New() plugin.Plugin {
	return &Transform{
		BasePlugin: plugin.NewBasePlugin("format_transform", plugin.TypeTransform),
	}
}

// Transform 转换消息为统一格式
func (t *Transform) Transform(ctx context.Context, msg *models.Message) (*models.Message, error) {
	formatted := t.convertToFormatted(msg)
	if formatted == nil {
		// 不支持的消息类型，返回原消息
		return msg, nil
	}

	// 创建新消息，使用统一格式
	newMsg := &models.Message{
		Type:     msg.Type,
		Platform: msg.Platform,
		RID:      msg.RID,
		Data:     formatted,
		RawData:  msg.RawData,
	}

	return newMsg, nil
}

// convertToFormatted 将消息转换为统一格式
func (t *Transform) convertToFormatted(msg *models.Message) *models.FormattedMessage {
	timestamp := time.Now()
	platform := string(msg.Platform)

	switch data := msg.Data.(type) {
	case *models.ChatData:

		return &models.FormattedMessage{
			UserName:    data.Name,
			Platform:    platform,
			Avatar:      t.getAvatar(data.Avatar, platform),
			Content:     data.Content,
			Timestamp:   timestamp,
			Type:        "chat",
			MessageType: models.TypeChat,
		}

	case *models.SuperChatData:

		content := fmt.Sprintf("%.2f 元: %s", data.Price, data.Content)
		return &models.FormattedMessage{
			UserName:    data.Name,
			Platform:    platform,
			Avatar:      t.getAvatar(data.Avatar, platform),
			Content:     content,
			Timestamp:   timestamp,
			Type:        "superchat",
			MessageType: models.TypeSuperChat,
		}

	case *models.GiftData:

		totalPrice := data.Price * float64(data.Num)
		content := fmt.Sprintf("送出了 %d 个 %s (%.2f 元)", data.Num, data.Item, totalPrice)
		return &models.FormattedMessage{
			UserName:    data.Name,
			Platform:    platform,
			Avatar:      t.getAvatar(data.Avatar, platform),
			Content:     content,
			Timestamp:   timestamp,
			Type:        "gift",
			MessageType: models.TypeGift,
		}

	case *models.SubscribeData:

		content := fmt.Sprintf("订阅了 %s", data.Item)
		return &models.FormattedMessage{
			UserName:    data.Name,
			Platform:    platform,
			Avatar:      t.getAvatar(data.Avatar, platform),
			Content:     content,
			Timestamp:   timestamp,
			Type:        "subscribe",
			MessageType: models.TypeSubscribe,
		}

	case *models.LikeData:

		content := fmt.Sprintf("点赞了 %d 次", data.Count)
		return &models.FormattedMessage{
			UserName:    data.Name,
			Platform:    platform,
			Avatar:      t.getAvatar(data.Avatar, platform),
			Content:     content,
			Timestamp:   timestamp,
			Type:        "like",
			MessageType: models.TypeLike,
		}

	case *models.EnterRoomData:

		content := "进入了直播间"
		return &models.FormattedMessage{
			UserName:    data.Name,
			Platform:    platform,
			Avatar:      t.getAvatar(data.Avatar, platform),
			Content:     content,
			Timestamp:   timestamp,
			Type:        "enterroom",
			MessageType: models.TypeEnterRoom,
		}

	case *models.EndLiveData:

		content := "直播结束"
		return &models.FormattedMessage{
			UserName:    "",
			Platform:    platform,
			Avatar:      t.getAvatar("", platform),
			Content:     content,
			Timestamp:   timestamp,
			Type:        "endlive",
			MessageType: models.TypeEndLive,
		}

	default:
		return nil
	}
}

// PlatformIcon returns the PNG logo URL for a platform id.
func PlatformIcon(platform string) string {
	if icon, ok := PlatformIcons[platform]; ok {
		return icon
	}
	return DefaultUserAvatar
}

// getAvatar：有用户头像用用户头像，否则用平台 logo PNG。
func (t *Transform) getAvatar(avatar string, platform string) string {
	if avatar != "" {
		return avatar
	}
	return PlatformIcon(platform)
}

func init() {
	plugin.Register("format_transform", New, plugin.PluginInfo{
		Name:           "format_transform",
		Type:           plugin.TypeTransform,
		ConfigTemplate: []plugin.ConfigField{},
	})
}
