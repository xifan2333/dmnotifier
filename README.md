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

- `s` - 服务列表（Enter 可**叠加**订阅多路，`*` 表示本地已连）
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
发送系统通知，支持头像缓存。

#### TTS 插件
语音播报弹幕消息，基于 Edge TTS。

**系统依赖**:
- macOS: afplay (系统自带)
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
