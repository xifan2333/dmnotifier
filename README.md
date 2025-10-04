# DMNotifier

基于 UniBarrage 的跨平台弹幕通知客户端，支持多种消息消费方式和灵活的插件系统。

## 特性

- **多平台支持**: Bilibili、Douyin、Kuaishou、Douyu、Huya
- **插件化架构**: 灵活的消息过滤、转换和消费插件系统
- **多种消费方式**:
  - TUI 终端界面显示
  - 系统通知
  - TTS 语音播报
  - WebView 弹幕墙
- **消息过滤**: 支持按消息类型过滤（聊天、礼物、SuperChat 等）
- **实时响应**: 异步处理，界面始终流畅

## 安装

### 从 Release 下载

访问 [Releases](https://github.com/xifan23332333/dmnotifier/releases) 页面下载适合您平台的预编译二进制文件。

### 从源码编译

```bash
git clone https://github.com/xifan23332333/dmnotifier.git
cd dmnotifier
go build -o dmnotifier ./cmd/dmnotifier-tui
```

## 使用

### 启动 TUI 客户端

```bash
./dmnotifier
```

### 快捷键

- `s` - 选择服务
- `a` - 添加服务
- `c` - 配置服务器
- `p` - 插件配置
- `r` - 刷新服务列表
- `d` - 断开连接
- `Ctrl+S` - 保存配置
- `q` / `Ctrl+C` - 退出

### 配置文件

配置文件位于 `~/.config/dmnotifier/config.json`

```json
{
  "server": {
    "api_address": "https://danmu.xifan2333.fun",
    "api_token": "your-token",
    "ws_address": "ws://danmu.xifan2333.fun:7777"
  },
  "pipeline": {
    "plugins": [
      {
        "name": "tui",
        "enabled": true,
        "message_types": ["chat", "gift", "superchat"]
      },
      {
        "name": "notify",
        "enabled": true,
        "message_types": ["superchat", "gift"]
      },
      {
        "name": "tts",
        "enabled": false,
        "message_types": ["chat"],
        "config": {
          "voice": "zh-CN-XiaoxiaoNeural",
          "language": "zh-CN"
        }
      },
      {
        "name": "webview",
        "enabled": false,
        "message_types": ["chat", "gift", "superchat"],
        "config": {
          "port": 8080,
          "auto_port": true
        }
      }
    ]
  }
}
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
├── cmd/
│   └── dmnotifier-tui/     # TUI 客户端入口
├── internal/
│   ├── client/              # WebSocket 和 API 客户端
│   ├── pipeline/            # 消息处理管道
│   ├── plugin/              # 插件系统核心
│   └── tui/                 # TUI 界面
├── plugins/
│   ├── consumers/           # 消费者插件
│   ├── filters/             # 过滤器插件
│   └── transforms/          # 转换器插件
└── pkg/
    ├── api/                 # UniBarrage API 客户端
    └── models/              # 数据模型
```

## 开发

### 添加自定义插件

1. 在 `plugins/consumers/` 创建插件目录
2. 实现 `plugin.ConsumerPlugin` 接口
3. 在 `init()` 中注册插件
4. 在 `cmd/dmnotifier-tui/main.go` 中导入插件

示例:

```go
package myplugin

import (
    "context"
    "github.com/xifan23332333/dmnotifier/internal/plugin"
    "github.com/xifan23332333/dmnotifier/pkg/models"
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
