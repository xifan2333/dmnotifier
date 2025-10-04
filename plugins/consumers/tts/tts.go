package tts

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"runtime"

	"github.com/lib-x/edgetts"
	"github.com/xifan2333/dmnotifier/internal/plugin"
	"github.com/xifan2333/dmnotifier/pkg/models"
)

// audioItem 音频播放项
type audioItem struct {
	text      string
	audioData []byte
}

// Consumer TTS 语音播报消费者
type Consumer struct {
	*plugin.BasePlugin

	// 播放队列（存储已生成的音频）
	queue  chan *audioItem
	ctx    context.Context
	cancel context.CancelFunc

	// 配置
	voice    string // 音色
	language string // 语言
}

// New 创建 TTS 消费者
func New() plugin.Plugin {
	return &Consumer{
		BasePlugin: plugin.NewBasePlugin("tts", plugin.TypeConsumer),
		voice:      "zh-CN-XiaoxiaoNeural", // 默认音色
		language:   "zh-CN",                // 默认语言
	}
}

// Init 初始化插件
func (c *Consumer) Init(ctx context.Context, config map[string]interface{}) error {

	if err := c.BasePlugin.Init(ctx, config); err != nil {

		return err
	}

	// 读取配置
	if voice, ok := config["voice"].(string); ok && voice != "" {
		c.voice = voice

	}
	if language, ok := config["language"].(string); ok && language != "" {
		c.language = language

	}

	// 读取队列长度配置
	queueSize := 100 // 默认队列长度
	if size, ok := config["queue_size"].(int); ok && size > 0 {
		queueSize = size

	}
	c.queue = make(chan *audioItem, queueSize)

	c.ctx, c.cancel = context.WithCancel(context.Background())

	// 检查 TTS 是否可用

	if err := c.checkTTS(); err != nil {

		return err
	}

	// 启动播放协程

	go c.playLoop()

	return nil
}

// Consume 消费消息 - 异步生成音频
func (c *Consumer) Consume(ctx context.Context, msg *models.Message) error {

	text := c.formatMessage(msg)
	if text == "" {
		return nil
	}

	// 异步生成音频
	go func() {
		audioData, err := c.generateAudio(text)
		if err != nil {

			return
		}

		// 添加到播放队列（非阻塞）
		select {
		case c.queue <- &audioItem{text: text, audioData: audioData}:

		case <-c.ctx.Done():

			return
		default:

		}
	}()

	return nil
}

// playLoop 播放循环 - 从队列中取出音频并播放
func (c *Consumer) playLoop() {
	for {
		select {
		case item := <-c.queue:
			if err := c.speakDirect(item.audioData); err != nil {

			}
		case <-c.ctx.Done():
			return
		}
	}
}

// Stop 停止插件
func (c *Consumer) Stop(ctx context.Context) error {
	// 取消上下文
	c.cancel()

	// 清空队列
	for len(c.queue) > 0 {
		<-c.queue
	}

	return nil
}

// formatMessage 格式化消息为语音文本
func (c *Consumer) formatMessage(msg *models.Message) string {
	// 只处理格式化后的消息
	formatted, ok := msg.Data.(*models.FormattedMessage)
	if !ok {
		return ""
	}

	switch formatted.Type {
	case "chat":
		return fmt.Sprintf("%s说：%s", formatted.UserName, formatted.Content)

	case "superchat", "gift", "subscribe", "like", "enterroom":
		// Content 已经是组装好的描述文本
		return fmt.Sprintf("%s%s", formatted.UserName, formatted.Content)

	case "endlive":
		// 直播结束没有用户名
		return formatted.Content

	default:
		return ""
	}
}

// generateAudio 生成音频数据（每次创建新的 Speech 实例，支持并发）
func (c *Consumer) generateAudio(text string) ([]byte, error) {
	// 创建 Speech 实例
	speech, err := edgetts.NewSpeech(
		edgetts.WithVoice(c.voice),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create speech: %w", err)
	}

	// 使用 buffer 收集音频数据
	var buf bytes.Buffer

	// 添加 TTS 任务，将音频写入 buffer
	if err := speech.AddSingleTask(text, &buf); err != nil {
		return nil, fmt.Errorf("failed to add task: %w", err)
	}

	// 生成音频
	if err := speech.StartTasks(); err != nil {
		return nil, fmt.Errorf("failed to generate audio: %w", err)
	}

	return buf.Bytes(), nil
}

// speakDirect 直接启动播放器进程播放
func (c *Consumer) speakDirect(audioData []byte) error {
	// 根据平台选择音频播放器
	var playerCmd string
	var playerArgs []string

	switch runtime.GOOS {
	case "darwin":
		// macOS: 使用 afplay
		playerCmd = "afplay"
		playerArgs = []string{"-"}
	case "linux":
		// Linux: 使用 mpv 或 ffplay
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
		// Windows: 使用 PowerShell 播放
		playerCmd = "powershell"
		playerArgs = []string{"-Command", "$player = New-Object System.Media.SoundPlayer; $player.Stream = [System.IO.MemoryStream]::new([byte[]]::new(0)); $player.Stream.Write([byte[]]::new(0), 0, 0); $player.Stream.Seek(0, 'Begin'); $player.Load(); $player.PlaySync();"}
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	// 创建音频播放器进程
	cmd := exec.CommandContext(c.ctx, playerCmd, playerArgs...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	// 启动播放器
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start player: %w", err)
	}

	// 写入音频数据
	if _, err := stdin.Write(audioData); err != nil {
		stdin.Close()
		cmd.Process.Kill()
		return fmt.Errorf("failed to write audio data: %w", err)
	}

	// 关闭 stdin，通知播放器没有更多数据
	stdin.Close()

	// 等待播放完成
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("player failed: %w", err)
	}

	return nil
}

// checkTTS 检查系统 TTS 是否可用
func (c *Consumer) checkTTS() error {
	switch runtime.GOOS {
	case "darwin":
		// macOS: 检查 afplay（系统自带）
		if _, err := exec.LookPath("afplay"); err != nil {
			return fmt.Errorf("afplay not found (should be built-in on macOS)")
		}

	case "linux":
		// Linux: 检查 mpv 或 ffplay
		_, err1 := exec.LookPath("mpv")
		_, err2 := exec.LookPath("ffplay")
		if err1 != nil && err2 != nil {
			return fmt.Errorf("mpv or ffplay not found. Install with: sudo pacman -S mpv (Arch) or sudo apt install mpv (Debian/Ubuntu)")
		}

	case "windows":
		// Windows: PowerShell 应该是系统自带的
		cmd := exec.Command("powershell", "-Command", "echo test")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("PowerShell not found")
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
				Name:    "voice",
				Type:    plugin.FieldTypeString,
				Default: "zh-CN-XiaoxiaoNeural",
				Desc:    "TTS 音色（如：zh-CN-XiaoxiaoNeural, zh-CN-YunxiNeural）",
			},
			{
				Name:    "language",
				Type:    plugin.FieldTypeString,
				Default: "zh-CN",
				Desc:    "语言代码（如：zh-CN, en-US）",
			},
			{
				Name:    "queue_size",
				Type:    plugin.FieldTypeNumber,
				Default: 100,
				Desc:    "播放队列长度",
			},
		},
	})
}
