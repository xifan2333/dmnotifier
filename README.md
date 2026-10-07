# DMNotifier

基于 UniBarrage 的跨平台弹幕通知客户端，支持多种消息消费方式和灵活的插件系统。

## 特性

- **多平台支持**: Bilibili、Douyin、**Xiaohongshu**、Kuaishou、Douyu、Huya
- **多路同时订阅**: 可同时连接多个 `platform/rid`，消息汇总进同一 pipeline
- **CLI**: `start` / `stop` / `list`（脚本友好，不进 TUI）；无参开 TUI；房间 `platform:rid[:cookie]`
- **插件化架构**: 消息过滤、转换和消费插件
- **多种消费方式**: TUI / 系统通知 / TTS / WebView 弹幕墙

## 安装

### 从 Release 下载

访问 [Releases](https://github.com/xifan2333/dmnotifier/releases) 下载预编译二进制。

### Arch Linux

```bash
yay -S dmnotifier-bin
```

### 从源码编译

```bash
git clone https://github.com/xifan2333/dmnotifier.git
cd dmnotifier
go build -o dmnotifier ./cmd/dmnotifier
```

## 使用

### TUI（人用）

```bash
dmnotifier
```

快捷键：

- `s` - 服务列表（Enter 可**叠加**订阅多路，`*` 表示本地已连；历史区 **Enter** 重连 / **e** 编辑 / **x** 删除）
- `a` - 添加服务
- `c` - 配置 UniBarrage API/WS
- `p` - 插件配置
- `r` - 刷新远端服务列表
- `d` - **断开全部**本地订阅
- `q` - 退出

服务器地址与插件开关在 TUI 内修改（写入 XDG 配置）。

### CLI（给脚本 / 其他程序）

不打开 TUI。稳定机器输出 + exit code，方便 pipe 和调用。

```bash
dmnotifier -h
dmnotifier -v

# start remote listener(s); write history
dmnotifier start xiaohongshu:570409528162863169
dmnotifier start bilibili:689422:"$(cat ~/secrets/bili.cookie)"

# list / stop
dmnotifier list
dmnotifier list --json
dmnotifier stop bilibili:689422 xiaohongshu:570409528162863169
```

房间格式：

```text
platform:rid
platform:rid:cookie          # 第 2 个 : 之后整段都是 cookie
```

平台 id（无别名）：`bilibili` `douyin` `xiaohongshu` `kuaishou` `douyu` `huya`

| 命令 | 作用 | stdout |
|------|------|--------|
| `start` | UniBarrage start + history | `platform\trid` |
| `stop` | UniBarrage stop（保留 history） | `platform\trid` |
| `list` | 当前远端监听 | `platform\trid` 或 `--json` |

看弹幕：开 TUI，在列表/历史里连接本地 WS。


### 配置文件

路径（XDG）：`$XDG_CONFIG_HOME/dmnotifier/config.yaml`（默认 `~/.config/dmnotifier/config.yaml`）

```yaml
server:
  api_address: http://127.0.0.1:8080
  api_token: ""
  ws_address: ws://127.0.0.1:7777
pipeline:
  plugins:
    - name: tui
      enabled: true
      messagetypes: [Chat, Gift, Like, EnterRoom, Subscribe, SuperChat, EndLive]
    - name: notify
      enabled: true
      messagetypes: [Chat, Gift, Like, EnterRoom, Subscribe, SuperChat, EndLive]
    - name: tts
      enabled: false
      messagetypes: [Chat]
history:
  - platform: bilibili
    rid: "689422"
```

## 插件系统

### 内置插件

#### TUI 插件
在终端界面显示弹幕消息，支持平台品牌色标签和时间戳。

#### Notify 插件
发送系统通知前，先取得用户头像并转换为最大 256×256 的 PNG，首次通知也使用头像；后续直接复用缓存，同一头像的并发请求共享下载与转换。无头像或处理失败时使用平台图标，下载超时为 10 秒，失败后一分钟再尝试。头像处理在独立通知任务中执行，不阻塞 TUI 或语音。发送失败会在 TUI 状态栏显示错误。

#### TTS 插件
支持 MiMo（默认）和 Edge 实时流式播报。在插件配置中选中「Speech engine」按 Enter 打开选单，上下键选择、Enter 确认。界面显示 Speech engine、Voice 和 Playback queue size，MiMo 额外显示 API key；模型和 API 地址保留在配置文件中，不在界面展示。切换时保留另一套设置；修改后重新连接订阅生效。MiMo 的 `voice` 和 Edge 的 `edge_voice` 分别保存，Edge 不需要 API Key。

音色也通过选单选择，不提供自定义输入。界面使用英文，音色选单仅显示中文：MiMo 的 4 个中文预置音色，以及 Edge 上游列表中的 14 个中文音色（含大陆、香港、台湾及方言），离线也可选择。输入音色名、语言代码（如 `zh-CN`、`zh-HK`）或 `Male` / `Female` 筛选，Ctrl+U 清空筛选，Esc 取消并返回配置列表。

```yaml
      config:
        provider: mimo            # mimo | edge
        api_key: "你的 MiMo API Key" # 仅 MiMo 必需
        base_url: https://api.xiaomimimo.com/v1
        model: mimo-v2.5-tts
        voice: 茉莉
        edge_voice: zh-CN-XiaoxiaoNeural
        queue_size: 100
```

MiMo 使用 SSE PCM16（24kHz、单声道），Edge 使用流式 MP3，收到音频即可持续送入同一个播放器。MiMo 的用户名和正文并行请求、顺序播放，正文开头的语气标签保持原样；Edge 将用户名和正文合为一次请求。MiMo 用户名音频使用最多 8MiB 的内存 LRU 缓存（单项含键最多 256KiB）；正文受背压限制，不积攒整句。播放队列满时，后台语音消费任务等待空位，不再静默丢弃；停止订阅会取消等待。持续消息多于播报速度时，等待任务和延迟会累积。排队时保存文本，仅当前播报开启网络流。

停止订阅会取消请求并关闭播放器；流中途失败会停止当前播报并在状态栏显示错误，不从头重播。每段请求设有两分钟上限。播放器使用约 100ms 音频缓冲，实际起声还取决于首包大小、解码器和声卡缓冲；首包到起声以 200ms 内为优化目标，不保证云端及网络总延迟。

诊断：TTS 通过 `slog.Debug` 输出 `headers`（MiMo 响应头）、`first_packet`（首段解码音频）、`request`（请求至流结束，包含背压等待）和 `first_write`（首段音频收到至首次写入播放器）的耗时。嵌入使用时也可通过 `on_timing: func(tts.Timing)` 接收；回调可能并发调用，应快速返回；`phase` 区分合成结束和首次交付两类记录。`first_write` 仅代表交付播放器，并非声卡实际起声时间；正文的该值包含等待前缀的时间。

详细测量与验证范围见 [流式播报验证记录](docs/tts-streaming-validation.md)。

**系统依赖**:
- macOS: mpv 或 ffplay 支持流式；只有系统自带 afplay 时回退为整段文件播放
- Windows: mpv 或 ffplay，需在 PATH 中
- Linux: mpv 或 ffplay
  ```bash
  # Arch Linux
  sudo pacman -S mpv

  # Debian/Ubuntu
  sudo apt install mpv
  ```

#### WebView 插件
提供 Web 界面的弹幕墙，支持自动端口查找。

访问 `http://localhost:8080` 查看弹幕墙。

### 消息类型

- `chat` - 聊天消息
- `gift` - 礼物
- `superchat` - SuperChat/SC
- `subscribe` - 订阅/关注
- `like` - 点赞
- `enterroom` - 进入直播间
- `endlive` - 直播结束

## 架构

```
消息流: WebSocket → Pipeline → [Filters] → [Transforms] → [Consumers]
```

### 项目结构

```
dmnotifier/
├── cmd/dmnotifier/          # 唯一入口：TUI + CLI
├── internal/
│   ├── config/              # 默认本地 UniBarrage 地址
│   ├── subscribe/           # 多路订阅
│   ├── client/              # WS / HTTP
│   ├── pipeline/            # 消息管道
│   ├── plugin/              # 插件核心
│   └── tui/                 # TUI
├── plugins/                 # consumers / filters / transforms
└── pkg/                     # api + models
```

## 开发

### 添加自定义插件

1. 在 `plugins/consumers/` 创建插件目录
2. 实现 `plugin.ConsumerPlugin` 接口
3. 在 `init()` 中注册插件
4. 在 `cmd/dmnotifier/main.go` 中导入插件

示例:

```go
package myplugin

import (
    "context"
    "github.com/xifan2333/dmnotifier/internal/plugin"
    "github.com/xifan2333/dmnotifier/pkg/models"
)

type Consumer struct {
    *plugin.BasePlugin
}

func New() plugin.Plugin {
    return &Consumer{
        BasePlugin: plugin.NewBasePlugin("myplugin", plugin.TypeConsumer),
    }
}

func (c *Consumer) Consume(ctx context.Context, msg *models.Message) error {
    // 处理消息
    return nil
}

func init() {
    plugin.Register("myplugin", New, plugin.PluginInfo{
        Name: "myplugin",
        Type: plugin.TypeConsumer,
        ConfigTemplate: []plugin.ConfigField{},
    })
}
```

## 依赖项目

- [UniBarrage](https://github.com/BarryWangQwQ/UniBarrage) - 统一弹幕代理服务
- [Bubbletea](https://github.com/charmbracelet/bubbletea) - TUI 框架
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - 样式库

## License

MIT License
