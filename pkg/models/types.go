package models

// MessageType 消息类型枚举
type MessageType string

const (
	TypeChat      MessageType = "Chat"
	TypeGift      MessageType = "Gift"
	TypeLike      MessageType = "Like"
	TypeEnterRoom MessageType = "EnterRoom"
	TypeSubscribe MessageType = "Subscribe"
	TypeSuperChat MessageType = "SuperChat"
	TypeEndLive   MessageType = "EndLive"
)

// String 返回消息类型名称
func (t MessageType) String() string {
	return string(t)
}

// IsValid 检查消息类型是否有效
func (t MessageType) IsValid() bool {
	switch t {
	case TypeChat, TypeGift, TypeLike, TypeEnterRoom,
		TypeSubscribe, TypeSuperChat, TypeEndLive:
		return true
	}
	return false
}
