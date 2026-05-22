package tts

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"runtime"
	"time"

	"github.com/xifan2333/dmnotifier/internal/plugin"
	"github.com/xifan2333/dmnotifier/pkg/models"
)

const (
	defaultBaseURL = "https://token-plan-cn.xiaomimimo.com/v1"
	defaultModel   = "mimo-v2.5-tts"
	defaultVoice   = "茉莉"
)

type audioItem struct {
	text      string
	audioData []byte
}

type Consumer struct {
	*plugin.BasePlugin

	queue  chan *audioItem
	ctx    context.Context
	cancel context.CancelFunc

	apiKey  string
	baseURL string
	model   string
	voice   string
	style   string

	httpClient *http.Client
}

func New() plugin.Plugin {
	return &Consumer{
		BasePlugin: plugin.NewBasePlugin("tts", plugin.TypeConsumer),
		baseURL:    defaultBaseURL,
		model:      defaultModel,
		voice:      defaultVoice,
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
	if v, ok := config["style"].(string); ok {
		c.style = v
	}

	queueSize := 100
	if size, ok := config["queue_size"].(int); ok && size > 0 {
		queueSize = size
	}
	c.queue = make(chan *audioItem, queueSize)

	c.ctx, c.cancel = context.WithCancel(context.Background())
	c.httpClient = &http.Client{Timeout: 30 * time.Second}

	if c.apiKey == "" {
		return fmt.Errorf("api_key is required")
	}

	if err := c.checkPlayer(); err != nil {
		return err
	}

	go c.playLoop()

	return nil
}

func (c *Consumer) Consume(ctx context.Context, msg *models.Message) error {
	text := c.formatMessage(msg)
	if text == "" {
		return nil
	}

	go func() {
		audioData, err := c.generateAudio(text)
		if err != nil {
			return
		}

		select {
		case c.queue <- &audioItem{text: text, audioData: audioData}:
		case <-c.ctx.Done():
			return
		default:
		}
	}()

	return nil
}

func (c *Consumer) playLoop() {
	for {
		select {
		case item := <-c.queue:
			if err := c.speakDirect(item.audioData); err != nil {
				_ = err
			}
		case <-c.ctx.Done():
			return
		}
	}
}

func (c *Consumer) Stop(ctx context.Context) error {
	c.cancel()
	for len(c.queue) > 0 {
		<-c.queue
	}
	return nil
}

func (c *Consumer) formatMessage(msg *models.Message) string {
	formatted, ok := msg.Data.(*models.FormattedMessage)
	if !ok {
		return ""
	}

	switch formatted.Type {
	case "chat":
		return fmt.Sprintf("%s说：%s", formatted.UserName, formatted.Content)
	case "superchat", "gift", "subscribe", "like", "enterroom":
		return fmt.Sprintf("%s%s", formatted.UserName, formatted.Content)
	case "endlive":
		return formatted.Content
	default:
		return ""
	}
}

type mimoMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type mimoAudio struct {
	Format string `json:"format"`
	Voice  string `json:"voice"`
}

type mimoRequest struct {
	Model    string        `json:"model"`
	Messages []mimoMessage `json:"messages"`
	Audio    mimoAudio     `json:"audio"`
}

type mimoResponse struct {
	Choices []struct {
		Message struct {
			Audio struct {
				Data string `json:"data"`
			} `json:"audio"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (c *Consumer) generateAudio(text string) ([]byte, error) {
	style := c.style
	if style == "" {
		style = "自然亲切"
	}

	reqBody := mimoRequest{
		Model: c.model,
		Messages: []mimoMessage{
			{Role: "user", Content: style},
			{Role: "assistant", Content: text},
		},
		Audio: mimoAudio{Format: "mp3", Voice: c.voice},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(c.ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode, string(respBody))
	}

	var parsed mimoResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	if parsed.Error != nil {
		return nil, fmt.Errorf("api error: %s", parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 || parsed.Choices[0].Message.Audio.Data == "" {
		return nil, fmt.Errorf("empty audio data in response")
	}

	audio, err := base64.StdEncoding.DecodeString(parsed.Choices[0].Message.Audio.Data)
	if err != nil {
		return nil, fmt.Errorf("decode base64: %w", err)
	}

	return audio, nil
}

func (c *Consumer) speakDirect(audioData []byte) error {
	var playerCmd string
	var playerArgs []string

	switch runtime.GOOS {
	case "darwin":
		playerCmd = "afplay"
		playerArgs = []string{"-"}
	case "linux":
		if _, err := exec.LookPath("mpv"); err == nil {
			playerCmd = "mpv"
			playerArgs = []string{"--really-quiet", "--no-terminal", "-"}
		} else if _, err := exec.LookPath("ffplay"); err == nil {
			playerCmd = "ffplay"
			playerArgs = []string{"-nodisp", "-autoexit", "-"}
		} else {
			return fmt.Errorf("no audio player found (mpv or ffplay)")
		}
	case "windows":
		if _, err := exec.LookPath("ffplay"); err == nil {
			playerCmd = "ffplay"
			playerArgs = []string{"-nodisp", "-autoexit", "-"}
		} else if _, err := exec.LookPath("mpv"); err == nil {
			playerCmd = "mpv"
			playerArgs = []string{"--really-quiet", "--no-terminal", "-"}
		} else {
			return fmt.Errorf("no audio player found (install ffplay or mpv)")
		}
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	cmd := exec.CommandContext(c.ctx, playerCmd, playerArgs...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("create stdin pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start player: %w", err)
	}

	if _, err := stdin.Write(audioData); err != nil {
		stdin.Close()
		cmd.Process.Kill()
		return fmt.Errorf("write audio: %w", err)
	}
	stdin.Close()

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("player failed: %w", err)
	}

	return nil
}

func (c *Consumer) checkPlayer() error {
	switch runtime.GOOS {
	case "darwin":
		if _, err := exec.LookPath("afplay"); err != nil {
			return fmt.Errorf("afplay not found (should be built-in on macOS)")
		}
	case "linux":
		_, err1 := exec.LookPath("mpv")
		_, err2 := exec.LookPath("ffplay")
		if err1 != nil && err2 != nil {
			return fmt.Errorf("mpv or ffplay not found. Install with: sudo pacman -S mpv (Arch) or sudo apt install mpv (Debian/Ubuntu)")
		}
	case "windows":
		_, err1 := exec.LookPath("ffplay")
		_, err2 := exec.LookPath("mpv")
		if err1 != nil && err2 != nil {
			return fmt.Errorf("ffplay or mpv not found on PATH (install ffmpeg or mpv)")
		}
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	return nil
}

func init() {
	plugin.Register("tts", New, plugin.PluginInfo{
		Name: "tts",
		Type: plugin.TypeConsumer,
		ConfigTemplate: []plugin.ConfigField{
			{
				Name:    "api_key",
				Type:    plugin.FieldTypeString,
				Default: "",
				Desc:    "Xiaomi MiMo API Key (required)",
			},
			{
				Name:    "base_url",
				Type:    plugin.FieldTypeString,
				Default: defaultBaseURL,
				Desc:    "API Base URL",
			},
			{
				Name:    "model",
				Type:    plugin.FieldTypeString,
				Default: defaultModel,
				Desc:    "TTS model",
			},
			{
				Name:    "voice",
				Type:    plugin.FieldTypeString,
				Default: defaultVoice,
				Desc:    "Voice (preset timbre, e.g. 茉莉/冰糖/苏打/白桦/Mia/Chloe/Milo/Dean/mimo_default)",
			},
			{
				Name:    "style",
				Type:    plugin.FieldTypeString,
				Default: "",
				Desc:    "Style prompt (e.g. natural, lively); leave empty for default",
			},
			{
				Name:    "queue_size",
				Type:    plugin.FieldTypeNumber,
				Default: 100,
				Desc:    "Playback queue size",
			},
		},
	})
}
