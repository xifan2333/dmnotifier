package webview

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/xifan/dmnotifier/internal/plugin"
	"github.com/xifan/dmnotifier/pkg/models"
)

//go:embed assets/*
var assetsFS embed.FS

// Consumer WebView 消费者，提供弹幕展示页面
type Consumer struct {
	*plugin.BasePlugin

	// HTTP 服务器
	server   *http.Server
	port     int
	autoPort bool // 自动寻找可用端口

	// WebSocket 客户端管理
	clients   map[*websocket.Conn]bool
	clientsMu sync.RWMutex
	upgrader  websocket.Upgrader

	// 消息广播通道
	broadcast chan *models.FormattedMessage
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

// New 创建 WebView 消费者
func New() plugin.Plugin {
	return &Consumer{
		BasePlugin: plugin.NewBasePlugin("webview_consumer", plugin.TypeConsumer),
		port:       8080,
		autoPort:   true, // 默认启用自动端口
		clients:    make(map[*websocket.Conn]bool),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // 允许所有来源
			},
		},
	}
}

// Init 初始化插件
func (c *Consumer) Init(ctx context.Context, config map[string]interface{}) error {
	if err := c.BasePlugin.Init(ctx, config); err != nil {
		return err
	}

	// 读取端口配置
	if port, ok := config["port"].(float64); ok {
		c.port = int(port)
	}

	// 读取自动端口配置
	if autoPort, ok := config["auto_port"].(bool); ok {
		c.autoPort = autoPort
	}

	// 读取队列长度配置
	queueSize := 100 // 默认队列长度
	if size, ok := config["queue_size"].(int); ok && size > 0 {
		queueSize = size
	}
	c.broadcast = make(chan *models.FormattedMessage, queueSize)

	c.ctx, c.cancel = context.WithCancel(context.Background())

	return nil
}

// Start 启动插件
func (c *Consumer) Start(ctx context.Context) error {
	// 设置路由
	mux := http.NewServeMux()
	mux.HandleFunc("/", c.handleIndex)
	mux.HandleFunc("/ws", c.handleWebSocket)
	mux.HandleFunc("/proxy/image", c.handleImageProxy)
	mux.HandleFunc("/avatar/default/", c.handleDefaultAvatar)
	mux.Handle("/assets/", http.FileServer(http.FS(assetsFS)))

	// 尝试启动服务器，如果启用了自动端口，则在端口占用时尝试其他端口
	startPort := c.port
	maxAttempts := 100
	var listener net.Listener
	var err error

	for attempt := 0; attempt < maxAttempts; attempt++ {
		addr := fmt.Sprintf(":%d", c.port)
		listener, err = net.Listen("tcp", addr)
		if err == nil {
			// 端口可用，成功监听
			break
		}

		// 如果不是端口占用错误，或未启用自动端口，直接返回错误
		if !c.autoPort {
			return fmt.Errorf("failed to listen on port %d: %w", c.port, err)
		}

		// 自动尝试下一个端口
		c.port++
		if attempt == maxAttempts-1 {
			return fmt.Errorf("failed to find available port (tried %d-%d): %w", startPort, c.port, err)
		}
	}

	if c.port != startPort {

	}

	// 创建 HTTP 服务器
	c.server = &http.Server{
		Handler: mux,
	}

	// 启动广播协程
	c.wg.Add(1)
	go c.broadcastWorker()

	// 启动 HTTP 服务器
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()

		if err := c.server.Serve(listener); err != nil && err != http.ErrServerClosed {

		}
	}()

	return nil
}

// Stop 停止插件
func (c *Consumer) Stop(ctx context.Context) error {

	// 停止上下文（停止接收新消息）
	c.cancel()

	// 清空广播队列
	for len(c.broadcast) > 0 {
		<-c.broadcast
	}

	// 关闭广播通道
	close(c.broadcast)

	// 关闭 HTTP 服务器
	if c.server != nil {
		if err := c.server.Shutdown(ctx); err != nil {

		}
	}

	// 关闭所有 WebSocket 连接
	c.clientsMu.Lock()
	for client := range c.clients {
		client.Close()
	}
	c.clientsMu.Unlock()

	// 等待协程结束
	c.wg.Wait()

	return nil
}

// Consume 消费消息
func (c *Consumer) Consume(ctx context.Context, msg *models.Message) error {
	// 检查消息是否已经是格式化的
	formatted, ok := msg.Data.(*models.FormattedMessage)
	if !ok {

		return nil
	}

	// 发送到广播通道（非阻塞）
	select {
	case c.broadcast <- formatted:

	default:

	}

	return nil
}

// broadcastWorker 广播工作协程
func (c *Consumer) broadcastWorker() {
	defer c.wg.Done()

	for {
		select {
		case <-c.ctx.Done():
			return
		case msg, ok := <-c.broadcast:
			if !ok {
				return
			}

			// 广播消息到所有客户端
			data, err := json.Marshal(msg)
			if err != nil {

				continue
			}

			c.clientsMu.RLock()
			for client := range c.clients {
				if err := client.WriteMessage(websocket.TextMessage, data); err != nil {

					// 连接错误会在读取循环中处理
				}
			}
			c.clientsMu.RUnlock()
		}
	}
}

// handleIndex 处理首页请求
func (c *Consumer) handleIndex(w http.ResponseWriter, r *http.Request) {
	data, err := assetsFS.ReadFile("assets/templates/index.html")
	if err != nil {
		http.Error(w, "Page not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
}

// handleWebSocket 处理 WebSocket 连接
func (c *Consumer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := c.upgrader.Upgrade(w, r, nil)
	if err != nil {

		return
	}

	// 添加客户端
	c.clientsMu.Lock()
	c.clients[conn] = true
	c.clientsMu.Unlock()

	// 读取消息循环（保持连接）
	defer func() {
		c.clientsMu.Lock()
		delete(c.clients, conn)
		c.clientsMu.Unlock()
		conn.Close()

	}()

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

// handleImageProxy 代理图片请求，绕过防盗链
func (c *Consumer) handleImageProxy(w http.ResponseWriter, r *http.Request) {
	imageURL := r.URL.Query().Get("url")
	if imageURL == "" {
		http.Error(w, "Missing url parameter", http.StatusBadRequest)
		return
	}

	// 创建请求
	req, err := http.NewRequest("GET", imageURL, nil)
	if err != nil {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	// 设置请求头，模拟正常的图片请求
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Referer", "https://www.bilibili.com/")
	req.Header.Set("Accept", "image/avif,image/webp,image/apng,image/svg+xml,image/*,*/*;q=0.8")

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Failed to fetch image", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		http.Error(w, "Failed to fetch image", resp.StatusCode)
		return
	}

	// 设置响应头
	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.Header().Set("Cache-Control", "public, max-age=86400") // 缓存1天

	// 将图片内容复制到响应
	_, err = io.Copy(w, resp.Body)
	if err != nil {

	}
}

// handleDefaultAvatar 提供默认头像
func (c *Consumer) handleDefaultAvatar(w http.ResponseWriter, r *http.Request) {
	// 从 URL 路径获取平台名称
	platform := r.URL.Path[len("/avatar/default/"):]

	// 各平台默认头像 SVG
	avatars := map[string]string{
		"bilibili": `<svg xmlns='http://www.w3.org/2000/svg' width='48' height='48' viewBox='0 0 48 48'><circle cx='24' cy='24' r='24' fill='#00a1d6'/><text x='24' y='32' text-anchor='middle' fill='white' font-size='20' font-weight='bold' font-family='Arial'>B</text></svg>`,
		"douyu":    `<svg xmlns='http://www.w3.org/2000/svg' width='48' height='48' viewBox='0 0 48 48'><circle cx='24' cy='24' r='24' fill='#ff7500'/><text x='24' y='32' text-anchor='middle' fill='white' font-size='20' font-weight='bold' font-family='Arial'>D</text></svg>`,
		"huya":     `<svg xmlns='http://www.w3.org/2000/svg' width='48' height='48' viewBox='0 0 48 48'><circle cx='24' cy='24' r='24' fill='#ff6600'/><text x='24' y='32' text-anchor='middle' fill='white' font-size='20' font-weight='bold' font-family='Arial'>H</text></svg>`,
		"youtube":  `<svg xmlns='http://www.w3.org/2000/svg' width='48' height='48' viewBox='0 0 48 48'><circle cx='24' cy='24' r='24' fill='#ff0000'/><text x='24' y='32' text-anchor='middle' fill='white' font-size='20' font-weight='bold' font-family='Arial'>Y</text></svg>`,
		"twitch":   `<svg xmlns='http://www.w3.org/2000/svg' width='48' height='48' viewBox='0 0 48 48'><circle cx='24' cy='24' r='24' fill='#9146ff'/><text x='24' y='32' text-anchor='middle' fill='white' font-size='20' font-weight='bold' font-family='Arial'>T</text></svg>`,
	}

	// 默认头像（用户图标）
	defaultSVG := `<svg xmlns='http://www.w3.org/2000/svg' width='48' height='48' viewBox='0 0 48 48'><circle cx='24' cy='24' r='24' fill='#cccccc'/><circle cx='24' cy='18' r='8' fill='#ffffff'/><path d='M12 38c0-6.627 5.373-12 12-12s12 5.373 12 12' fill='#ffffff'/></svg>`

	svg, ok := avatars[platform]
	if !ok {
		svg = defaultSVG
	}

	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "public, max-age=31536000") // 缓存1年
	w.Write([]byte(svg))
}

func init() {
	plugin.Register("webview", New, plugin.PluginInfo{
		Name: "webview",
		Type: plugin.TypeConsumer,
		ConfigTemplate: []plugin.ConfigField{
			{
				Name:    "port",
				Type:    plugin.FieldTypeNumber,
				Default: 8080,
				Desc:    "WebView 服务端口",
			},
			{
				Name:    "auto_port",
				Type:    plugin.FieldTypeBool,
				Default: true,
				Desc:    "端口被占用时自动寻找可用端口",
			},
			{
				Name:    "queue_size",
				Type:    plugin.FieldTypeNumber,
				Default: 100,
				Desc:    "消息广播队列长度",
			},
		},
	})
}
