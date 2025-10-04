package models

import (
	"encoding/json"
	"fmt"
)

// Message WebSocket 消息结构
type Message struct {
	RID      string          `json:"rid"`      // 房间号
	Platform Platform        `json:"platform"` // 平台
	Type     MessageType     `json:"type"`     // 消息类型
	Data     MessageData     `json:"-"`        // 解析后的数据
	RawData  json.RawMessage `json:"data"`     // 原始 JSON 数据
}

// MessageData 消息数据接口
type MessageData interface {
	GetType() MessageType
}

// ChatData 聊天消息
type ChatData struct {
	Name     string          `json:"name"`     // 发送者名称
	Avatar   string          `json:"avatar"`   // 发送者头像
	Content  string          `json:"content"`  // 聊天内容
	Emoticon []string        `json:"emoticon"` // 表情 URL 列表
	Raw      json.RawMessage `json:"raw"`      // 原始数据
}

func (c *ChatData) GetType() MessageType { return TypeChat }

// GiftData 礼物消息
type GiftData struct {
	Name     string          `json:"name"`     // 赠送者名称
	Avatar   string          `json:"avatar"`   // 赠送者头像
	Item     string          `json:"item"`     // 礼物名称
	Num      int             `json:"num"`      // 礼物数量
	Price    float64         `json:"price"`    // 礼物单价
	GiftIcon string          `json:"giftIcon"` // 礼物图标
	Raw      json.RawMessage `json:"raw"`      // 原始数据
}

func (g *GiftData) GetType() MessageType { return TypeGift }

// LikeData 点赞消息
type LikeData struct {
	Name   string          `json:"name"`   // 点赞者名称
	Avatar string          `json:"avatar"` // 点赞者头像
	Count  int             `json:"count"`  // 点赞次数
	Raw    json.RawMessage `json:"raw"`    // 原始数据
}

func (l *LikeData) GetType() MessageType { return TypeLike }

// EnterRoomData 进入房间消息
type EnterRoomData struct {
	Name   string          `json:"name"`   // 进入者名称
	Avatar string          `json:"avatar"` // 进入者头像
	Raw    json.RawMessage `json:"raw"`    // 原始数据
}

func (e *EnterRoomData) GetType() MessageType { return TypeEnterRoom }

// SubscribeData 订阅消息
type SubscribeData struct {
	Name   string          `json:"name"`   // 订阅者名称
	Avatar string          `json:"avatar"` // 订阅者头像
	Item   string          `json:"item"`   // 订阅项
	Num    int             `json:"num"`    // 订阅次数
	Price  float64         `json:"price"`  // 订阅单价
	Raw    json.RawMessage `json:"raw"`    // 原始数据
}

func (s *SubscribeData) GetType() MessageType { return TypeSubscribe }

// SuperChatData 超级聊天消息
type SuperChatData struct {
	Name    string          `json:"name"`    // 发送者名称
	Avatar  string          `json:"avatar"`  // 发送者头像
	Content string          `json:"content"` // 超级聊天内容
	Price   float64         `json:"price"`   // 金额
	Raw     json.RawMessage `json:"raw"`     // 原始数据
}

func (s *SuperChatData) GetType() MessageType { return TypeSuperChat }

// EndLiveData 结束直播消息
type EndLiveData struct {
	Raw json.RawMessage `json:"raw"` // 原始数据
}

func (e *EndLiveData) GetType() MessageType { return TypeEndLive }

// ParseMessage 解析消息数据
func (m *Message) ParseMessage() error {
	if !m.Type.IsValid() {
		return fmt.Errorf("invalid message type: %s", m.Type)
	}

	switch m.Type {
	case TypeChat:
		var data ChatData
		if err := json.Unmarshal(m.RawData, &data); err != nil {
			return fmt.Errorf("parse chat data: %w", err)
		}
		m.Data = &data

	case TypeGift:
		var data GiftData
		if err := json.Unmarshal(m.RawData, &data); err != nil {
			return fmt.Errorf("parse gift data: %w", err)
		}
		m.Data = &data

	case TypeLike:
		var data LikeData
		if err := json.Unmarshal(m.RawData, &data); err != nil {
			return fmt.Errorf("parse like data: %w", err)
		}
		m.Data = &data

	case TypeEnterRoom:
		var data EnterRoomData
		if err := json.Unmarshal(m.RawData, &data); err != nil {
			return fmt.Errorf("parse enter room data: %w", err)
		}
		m.Data = &data

	case TypeSubscribe:
		var data SubscribeData
		if err := json.Unmarshal(m.RawData, &data); err != nil {
			return fmt.Errorf("parse subscribe data: %w", err)
		}
		m.Data = &data

	case TypeSuperChat:
		var data SuperChatData
		if err := json.Unmarshal(m.RawData, &data); err != nil {
			return fmt.Errorf("parse super chat data: %w", err)
		}
		m.Data = &data

	case TypeEndLive:
		var data EndLiveData
		if err := json.Unmarshal(m.RawData, &data); err != nil {
			return fmt.Errorf("parse end live data: %w", err)
		}
		m.Data = &data
	}

	return nil
}

// UnmarshalJSON 自定义 JSON 反序列化
func (m *Message) UnmarshalJSON(data []byte) error {
	type Alias Message
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(m),
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	return m.ParseMessage()
}
