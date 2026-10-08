package tts

import (
	"context"
	"fmt"

	"github.com/xifan2333/dmnotifier/internal/plugin"
	"github.com/xifan2333/dmnotifier/pkg/models"
)

const (
	defaultBaseURL   = "https://api.xiaomimimo.com/v1"
	defaultModel     = "mimo-v2.5-tts"
	defaultVoice     = "茉莉"
	defaultProvider  = "mimo"
	defaultEdgeVoice = "zh-CN-XiaoxiaoNeural"
)

type audioItem struct {
	prefix  string
	content string
}

type Consumer struct {
	*plugin.BasePlugin

	queue  chan *audioItem
	ctx    context.Context
	cancel context.CancelFunc

	engine   engine
	onError  func(error)
	player   string
	done     chan struct{}
	onTiming func(Timing)
}

func New() plugin.Plugin {
	return &Consumer{BasePlugin: plugin.NewBasePlugin("tts", plugin.TypeConsumer)}
}

func (c *Consumer) Init(ctx context.Context, config map[string]interface{}) error {
	if err := c.BasePlugin.Init(ctx, config); err != nil {
		return err
	}

	if f, ok := config["on_timing"].(func(Timing)); ok {
		c.onTiming = f
	}
	definition, err := findEngine(config)
	if err != nil {
		return err
	}
	c.engine, err = definition.create(config, c.emitTiming)
	if err != nil {
		return err
	}
	if err := c.checkPlayer(); err != nil {
		_ = c.engine.Close()
		return err
	}

	queueSize := 100
	if size, ok := config["queue_size"].(int); ok && size > 0 {
		queueSize = size
	}
	c.queue = make(chan *audioItem, queueSize)

	c.ctx, c.cancel = context.WithCancel(ctx)
	c.done = make(chan struct{})

	go c.playLoop()

	return nil
}

func (c *Consumer) Consume(ctx context.Context, msg *models.Message) error {
	if c.ctx.Err() != nil {
		return nil
	}
	prefix, content := c.formatMessage(msg)
	if prefix == "" && content == "" {
		return nil
	}

	// Wait for queue space in the background consumer task instead of dropping.
	select {
	case c.queue <- &audioItem{prefix: prefix, content: content}:
	case <-c.ctx.Done():
	case <-ctx.Done():
	}

	return nil
}

func (c *Consumer) playLoop() {
	defer close(c.done)
	defer c.engine.Close()
	for {
		select {
		case <-c.ctx.Done():
			return
		case item := <-c.queue:
			if c.ctx.Err() != nil {
				return
			}
			if err := c.playMessage(item); err != nil {
				c.reportError(err)
			}
		}
	}
}

func (c *Consumer) Stop(ctx context.Context) error {
	if c.cancel == nil {
		return nil
	}
	c.cancel()
	select {
	case <-c.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *Consumer) reportError(err error) {
	if c.ctx.Err() == nil && c.onError != nil {
		c.onError(fmt.Errorf("tts: %w", err))
	}
}

// formatMessage 拆出 (prefix, content)：chat 类型 prefix 用默认语气，content 保留观众原文
func (c *Consumer) formatMessage(msg *models.Message) (prefix, content string) {
	formatted, ok := msg.Data.(*models.FormattedMessage)
	if !ok {
		return "", ""
	}

	switch formatted.Type {
	case "chat":
		return fmt.Sprintf("%s说：", formatted.UserName), formatted.Content
	case "superchat", "gift", "subscribe", "like", "enterroom":
		return "", fmt.Sprintf("%s%s", formatted.UserName, formatted.Content)
	case "endlive":
		return "", formatted.Content
	default:
		return "", ""
	}
}

func (c *Consumer) SetErrorHandler(handler func(error)) { c.onError = handler }
