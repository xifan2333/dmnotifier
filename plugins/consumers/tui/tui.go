package tui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/xifan/dmnotifier/internal/plugin"
	"github.com/xifan/dmnotifier/internal/tui/components"
	"github.com/xifan/dmnotifier/pkg/models"
)

// Consumer TUI 消费者插件
type Consumer struct {
	*plugin.BasePlugin

	// Bubble Tea 程序实例
	program *tea.Program

	// 平台样式
	platformStyles map[string]lipgloss.Style

	// 上下文控制
	ctx    context.Context
	cancel context.CancelFunc
}

// New 创建 TUI 消费者
func New() plugin.Plugin {
	return &Consumer{
		BasePlugin: plugin.NewBasePlugin("tui", plugin.TypeConsumer),
		platformStyles: map[string]lipgloss.Style{
			"bilibili": lipgloss.NewStyle().
				Background(lipgloss.Color("#00a1d6")).
				Foreground(lipgloss.Color("#ffffff")).
				Bold(true).
				Padding(0, 1),
			"douyin": lipgloss.NewStyle().
				Background(lipgloss.Color("#fe2c55")).
				Foreground(lipgloss.Color("#ffffff")).
				Bold(true).
				Padding(0, 1),
			"kuaishou": lipgloss.NewStyle().
				Background(lipgloss.Color("#ff6600")).
				Foreground(lipgloss.Color("#ffffff")).
				Bold(true).
				Padding(0, 1),
			"douyu": lipgloss.NewStyle().
				Background(lipgloss.Color("#ff7500")).
				Foreground(lipgloss.Color("#ffffff")).
				Bold(true).
				Padding(0, 1),
			"huya": lipgloss.NewStyle().
				Background(lipgloss.Color("#f60")).
				Foreground(lipgloss.Color("#ffffff")).
				Bold(true).
				Padding(0, 1),
		},
	}
}

// Init 初始化插件
func (c *Consumer) Init(ctx context.Context, config map[string]interface{}) error {
	if err := c.BasePlugin.Init(ctx, config); err != nil {
		return err
	}

	// 创建上下文
	c.ctx, c.cancel = context.WithCancel(context.Background())

	// 从配置中获取 Bubble Tea 程序实例
	if program, ok := config["program"].(*tea.Program); ok {
		c.program = program
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

	if c.program == nil {
		return nil
	}

	// 只处理格式化后的消息
	formatted, ok := msg.Data.(*models.FormattedMessage)
	if !ok {
		return nil
	}

	// 格式化消息内容
	content := c.formatMessage(formatted)

	// 异步发送到 TUI，避免阻塞
	go c.program.Send(components.AddMessageMsg{Content: content})

	return nil
}

// formatMessage 格式化消息为显示文本
func (c *Consumer) formatMessage(msg *models.FormattedMessage) string {
	// 获取时间（格式：HH:MM:SS）
	timeStr := msg.Timestamp.Format("15:04:05")

	// 获取平台样式，如果没有则使用默认样式
	platformStyle, ok := c.platformStyles[msg.Platform]
	if !ok {
		platformStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#666666")).
			Foreground(lipgloss.Color("#ffffff")).
			Bold(true).
			Padding(0, 1)
	}

	// 渲染平台标签
	platformTag := platformStyle.Render(msg.Platform)

	// 时间样式
	timeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))

	// 组装：[时间] 平台 | 昵称: 内容
	return fmt.Sprintf("%s %s | %s: %s",
		timeStyle.Render(fmt.Sprintf("[%s]", timeStr)),
		platformTag,
		msg.UserName,
		msg.Content,
	)
}

// Stop 停止插件
func (c *Consumer) Stop(ctx context.Context) error {
	// 取消上下文，阻止新消息发送到 TUI
	if c.cancel != nil {
		c.cancel()
	}
	return nil
}

func init() {
	plugin.Register("tui", New, plugin.PluginInfo{
		Name:           "tui",
		Type:           plugin.TypeConsumer,
		ConfigTemplate: []plugin.ConfigField{},
	})
}
