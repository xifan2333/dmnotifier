package tts

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/lib-x/edgetts"
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

	apiKey     string
	baseURL    string
	model      string
	voice      string
	provider   string
	edgeVoice  string
	edgeClient *edgetts.Client
	onError    func(error)

	httpClient *http.Client
	player     string
	done       chan struct{}
	cache      prefixCache
	onTiming   func(Timing)
}

func New() plugin.Plugin {
	return &Consumer{
		BasePlugin: plugin.NewBasePlugin("tts", plugin.TypeConsumer),
		baseURL:    defaultBaseURL,
		model:      defaultModel,
		voice:      defaultVoice,
		provider:   defaultProvider,
		edgeVoice:  defaultEdgeVoice,
	}
}

func (c *Consumer) Init(ctx context.Context, config map[string]interface{}) error {
	if err := c.BasePlugin.Init(ctx, config); err != nil {
		return err
	}

	if v, ok := config["api_key"].(string); ok {
		c.apiKey = v
	}
	if v, ok := config["base_url"].(string); ok && v != "" {
		c.baseURL = v
	}
	if v, ok := config["model"].(string); ok && v != "" {
		c.model = v
	}
	if v, ok := config["voice"].(string); ok && v != "" {
		c.voice = v
	}
	if v, ok := config["provider"].(string); ok && strings.TrimSpace(v) != "" {
		c.provider = strings.ToLower(strings.TrimSpace(v))
	}
	if v, ok := config["edge_voice"].(string); ok && strings.TrimSpace(v) != "" {
		c.edgeVoice = strings.TrimSpace(v)
	}
	if onError, ok := config["on_error"].(func(error)); ok {
		c.onError = onError
	}

	switch c.provider {
	case "mimo":
		if c.apiKey == "" {
			return fmt.Errorf("MiMo requires api_key (or select provider=edge)")
		}
	case "edge":
		c.edgeClient = edgetts.New(edgetts.WithVoice(c.edgeVoice))
	default:
		return fmt.Errorf("unknown TTS provider %q (choose mimo or edge)", c.provider)
	}
	if err := c.checkPlayer(); err != nil {
		return err
	}

	queueSize := 100
	if size, ok := config["queue_size"].(int); ok && size > 0 {
		queueSize = size
	}
	c.queue = make(chan *audioItem, queueSize)

	c.ctx, c.cancel = context.WithCancel(ctx)
	c.done = make(chan struct{})
	if f, ok := config["on_timing"].(func(Timing)); ok {
		c.onTiming = f
	}
	c.httpClient = &http.Client{Transport: &http.Transport{
		Proxy: http.ProxyFromEnvironment, MaxIdleConns: 8, MaxIdleConnsPerHost: 4,
		IdleConnTimeout: 90 * time.Second, TLSHandshakeTimeout: 10 * time.Second, ResponseHeaderTimeout: 30 * time.Second,
	}}

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
	defer c.httpClient.CloseIdleConnections()
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

func init() {
	plugin.Register("tts", New, plugin.PluginInfo{
		Name: "tts",
		Type: plugin.TypeConsumer,
		ConfigTemplate: []plugin.ConfigField{
			{
				Name:         "provider",
				Label:        "Speech engine",
				OptionLabels: map[string]string{"mimo": "MiMo", "edge": "Edge"},
				Type:         plugin.FieldTypeEnum,
				Default:      defaultProvider,
				Desc:         "TTS engine (mimo or edge)",
				Options:      []string{"mimo", "edge"},
			},
			{
				Name:         "edge_voice",
				Label:        "Voice",
				Options:      edgeVoiceOptions,
				OptionLabels: edgeVoiceLabels,
				Type:         plugin.FieldTypeEnum,
				Default:      defaultEdgeVoice,
				Desc:         "Edge voice (e.g. zh-CN-XiaoxiaoNeural or zh-CN-YunxiNeural)",
			},
			{
				Name:    "api_key",
				Label:   "API key",
				Type:    plugin.FieldTypeString,
				Default: "",
				Desc:    "Xiaomi MiMo API Key (required only for mimo)",
			},
			{
				Name:    "base_url",
				Label:   "MiMo API URL",
				Type:    plugin.FieldTypeString,
				Default: defaultBaseURL,
				Desc:    "API Base URL",
			},
			{
				Name:    "model",
				Label:   "MiMo model",
				Type:    plugin.FieldTypeEnum,
				Options: []string{defaultModel},
				Default: defaultModel,
				Desc:    "TTS model",
			},
			{
				Name:         "voice",
				Label:        "Voice",
				Options:      []string{"茉莉", "冰糖", "苏打", "白桦"},
				OptionLabels: mimoVoiceLabels,
				Type:         plugin.FieldTypeEnum,
				Default:      defaultVoice,
				Desc:         "MiMo voice (e.g. 茉莉/冰糖/苏打/白桦/Mia/Chloe/Milo/Dean/mimo_default)",
			},
			{
				Name:    "queue_size",
				Label:   "Playback queue size",
				Type:    plugin.FieldTypeNumber,
				Default: 100,
				Desc:    "Playback queue size",
			},
		},
	})
}
