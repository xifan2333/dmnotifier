package format

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/xifan2333/dmnotifier/internal/plugin"
	"github.com/xifan2333/dmnotifier/pkg/models"
)

// platformIconMarker is a special Avatar value meaning "use embedded platform logo".
// Consumers that need a real image (notify/webview) resolve this to a local asset.
const platformIconPrefix = "platform-icon:"

// PlatformIconRef returns the marker for an embedded platform logo.
func PlatformIconRef(platform string) string {
	return platformIconPrefix + platform
}

// ParsePlatformIconRef extracts platform id from a marker; ok=false if not a marker.
func ParsePlatformIconRef(avatar string) (platform string, ok bool) {
	if strings.HasPrefix(avatar, platformIconPrefix) {
		return strings.TrimPrefix(avatar, platformIconPrefix), true
	}
	return "", false
}

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
		return msg, nil
	}
	return &models.Message{
		Type:     msg.Type,
		Platform: msg.Platform,
		RID:      msg.RID,
		Data:     formatted,
		RawData:  msg.RawData,
	}, nil
}

func (t *Transform) convertToFormatted(msg *models.Message) *models.FormattedMessage {
	timestamp := time.Now()
	platform := string(msg.Platform)

	switch data := msg.Data.(type) {
	case *models.ChatData:
		return &models.FormattedMessage{
			UserName: data.Name, Platform: platform,
			Avatar: t.getAvatar(data.Avatar, platform), Content: data.Content,
			Timestamp: timestamp, Type: "chat", MessageType: models.TypeChat,
		}
	case *models.SuperChatData:
		return &models.FormattedMessage{
			UserName: data.Name, Platform: platform,
			Avatar: t.getAvatar(data.Avatar, platform),
			Content: fmt.Sprintf("%.2f 元: %s", data.Price, data.Content),
			Timestamp: timestamp, Type: "superchat", MessageType: models.TypeSuperChat,
		}
	case *models.GiftData:
		total := data.Price * float64(data.Num)
		return &models.FormattedMessage{
			UserName: data.Name, Platform: platform,
			Avatar: t.getAvatar(data.Avatar, platform),
			Content: fmt.Sprintf("送出了 %d 个 %s (%.2f 元)", data.Num, data.Item, total),
			Timestamp: timestamp, Type: "gift", MessageType: models.TypeGift,
		}
	case *models.SubscribeData:
		return &models.FormattedMessage{
			UserName: data.Name, Platform: platform,
			Avatar: t.getAvatar(data.Avatar, platform),
			Content: fmt.Sprintf("订阅了 %s", data.Item),
			Timestamp: timestamp, Type: "subscribe", MessageType: models.TypeSubscribe,
		}
	case *models.LikeData:
		return &models.FormattedMessage{
			UserName: data.Name, Platform: platform,
			Avatar: t.getAvatar(data.Avatar, platform),
			Content: fmt.Sprintf("点赞了 %d 次", data.Count),
			Timestamp: timestamp, Type: "like", MessageType: models.TypeLike,
		}
	case *models.EnterRoomData:
		return &models.FormattedMessage{
			UserName: data.Name, Platform: platform,
			Avatar: t.getAvatar(data.Avatar, platform), Content: "进入了直播间",
			Timestamp: timestamp, Type: "enterroom", MessageType: models.TypeEnterRoom,
		}
	case *models.EndLiveData:
		return &models.FormattedMessage{
			UserName: "", Platform: platform,
			Avatar: PlatformIconRef(platform), Content: "直播结束",
			Timestamp: timestamp, Type: "endlive", MessageType: models.TypeEndLive,
		}
	default:
		return nil
	}
}

// getAvatar: user avatar URL if present, else platform-icon marker for local logo.
func (t *Transform) getAvatar(avatar string, platform string) string {
	if avatar != "" {
		return avatar
	}
	return PlatformIconRef(platform)
}

func init() {
	plugin.Register("format_transform", New, plugin.PluginInfo{
		Name:           "format_transform",
		Type:           plugin.TypeTransform,
		ConfigTemplate: []plugin.ConfigField{},
	})
}
