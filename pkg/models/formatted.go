package models

import "time"

// FormattedMessage 统一格式的消息
type FormattedMessage struct {
	// 基础字段
	UserName  string    `json:"userName"`  // 用户名
	Platform  string    `json:"platform"`  // 平台
	Avatar    string    `json:"avatar"`    // 头像URL
	Content   string    `json:"content"`   // 内容（已组装好的描述文本）
	Timestamp time.Time `json:"timestamp"` // 时间戳

	// 扩展字段
	Type string `json:"type"` // 消息类型：chat, superchat, gift, subscribe, like, enterroom, endlive

	// 原始消息类型
	MessageType MessageType `json:"-"`
}

// GetType 实现 MessageData 接口
func (f *FormattedMessage) GetType() MessageType {
	return f.MessageType
}
