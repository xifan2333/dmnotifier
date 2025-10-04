package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/xifan2333/dmnotifier/pkg/models"
)

// MessageHandler 消息处理函数类型
type MessageHandler func(*models.Message) error

// WSClient WebSocket 客户端
type WSClient struct {
	// 连接配置
	host string
	port int
	url  string

	// WebSocket 连接
	conn   *websocket.Conn
	connMu sync.RWMutex

	// 消息处理
	handler MessageHandler

	// 重连配置
	enableReconnect   bool
	reconnectDelay    time.Duration
	maxReconnectTries int

	// 状态管理
	connected bool
	closed    bool
	closeMu   sync.RWMutex

	// 上下文控制
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// WSClientConfig WebSocket 客户端配置
type WSClientConfig struct {
	URL               string // 完整 URL（优先）
	Host              string
	Port              int
	Handler           MessageHandler
	EnableReconnect   bool
	ReconnectDelay    time.Duration
	MaxReconnectTries int
}

// NewWSClient 创建新的 WebSocket 客户端
func NewWSClient(config WSClientConfig) *WSClient {
	if config.ReconnectDelay == 0 {
		config.ReconnectDelay = 5 * time.Second
	}
	if config.MaxReconnectTries == 0 {
		config.MaxReconnectTries = -1 // -1 表示无限重试
	}

	var wsURL string
	if config.URL != "" {
		wsURL = config.URL
	} else {
		u := url.URL{
			Scheme: "ws",
			Host:   fmt.Sprintf("%s:%d", config.Host, config.Port),
			Path:   "/",
		}
		wsURL = u.String()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &WSClient{
		host:              config.Host,
		port:              config.Port,
		url:               wsURL,
		handler:           config.Handler,
		enableReconnect:   config.EnableReconnect,
		reconnectDelay:    config.ReconnectDelay,
		maxReconnectTries: config.MaxReconnectTries,
		ctx:               ctx,
		cancel:            cancel,
	}
}

// Connect 连接到 WebSocket 服务器
func (c *WSClient) Connect() error {
	c.closeMu.RLock()
	if c.closed {
		c.closeMu.RUnlock()
		return fmt.Errorf("client is closed")
	}
	c.closeMu.RUnlock()

	conn, _, err := websocket.DefaultDialer.Dial(c.url, nil)
	if err != nil {
		return fmt.Errorf("dial websocket: %w", err)
	}

	c.connMu.Lock()
	c.conn = conn
	c.connected = true
	c.connMu.Unlock()

	// 启动消息接收协程
	c.wg.Add(1)
	go c.readLoop()

	return nil
}

// Start 启动客户端（带重连）
func (c *WSClient) Start() error {
	if err := c.Connect(); err != nil {
		if !c.enableReconnect {
			return err
		}

	}

	// 如果启用重连，启动重连监控
	if c.enableReconnect {
		c.wg.Add(1)
		go c.reconnectLoop()
	}

	return nil
}

// readLoop 读取消息循环
func (c *WSClient) readLoop() {
	defer c.wg.Done()
	defer c.setDisconnected()

	for {
		select {
		case <-c.ctx.Done():

			return
		default:
		}

		c.connMu.RLock()
		conn := c.conn
		c.connMu.RUnlock()

		if conn == nil {

			return
		}

		// 设置读取超时
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))

		// 读取消息
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {

			}
			return
		}

		// 解析消息
		if err := c.handleMessage(message); err != nil {

		}
	}
}

// handleMessage 处理接收到的消息
func (c *WSClient) handleMessage(rawMsg []byte) error {
	var msg models.Message
	if err := json.Unmarshal(rawMsg, &msg); err != nil {
		return fmt.Errorf("unmarshal message: %w", err)
	}

	// 调用消息处理器
	if c.handler != nil {
		if err := c.handler(&msg); err != nil {
			return fmt.Errorf("message handler error: %w", err)
		}
	}

	return nil
}

// reconnectLoop 重连循环
func (c *WSClient) reconnectLoop() {
	defer c.wg.Done()

	retries := 0

	for {
		select {
		case <-c.ctx.Done():

			return
		default:
		}

		// 检查是否已连接
		c.connMu.RLock()
		connected := c.connected
		c.connMu.RUnlock()

		if connected {
			time.Sleep(1 * time.Second)
			continue
		}

		// 检查是否超过最大重试次数
		if c.maxReconnectTries > 0 && retries >= c.maxReconnectTries {

			return
		}

		retries++

		if err := c.Connect(); err != nil {

			time.Sleep(c.reconnectDelay)
			continue
		}

		retries = 0
	}
}

// setDisconnected 设置为未连接状态
func (c *WSClient) setDisconnected() {
	c.connMu.Lock()
	defer c.connMu.Unlock()

	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	c.connected = false

}

// IsConnected 检查是否已连接
func (c *WSClient) IsConnected() bool {
	c.connMu.RLock()
	defer c.connMu.RUnlock()
	return c.connected
}

// Close 关闭客户端
func (c *WSClient) Close() error {
	c.closeMu.Lock()
	if c.closed {
		c.closeMu.Unlock()
		return nil
	}
	c.closed = true
	c.closeMu.Unlock()

	// 取消上下文
	c.cancel()

	// 关闭连接
	c.setDisconnected()

	// 等待所有协程结束
	c.wg.Wait()

	return nil
}

// SetMessageHandler 设置消息处理器
func (c *WSClient) SetMessageHandler(handler MessageHandler) {
	c.handler = handler
}
