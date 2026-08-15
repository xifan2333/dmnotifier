# DMNotifier

基于 UniBarrage 的跨平台弹幕通知客户端，支持多种消息消费方式和灵活的插件系统。

## 特性

- **多平台支持**: Bilibili、Douyin、**Xiaohongshu (xhs)**、Kuaishou、Douyu、Huya
- **多路同时订阅**: 可同时连接多个 `platform/rid`，消息汇总进同一 pipeline
- **CLI 管道友好**: `subscribe` / `add` / `stop` / `list`，支持 `-json` 行输出
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
# 兼容旧入口（仅 TUI）
go build -o dmnotifier-tui ./cmd/dmnotifier-tui
```

## 使用

### TUI（默认）

```bash
./dmnotifier
# 或
./dmnotifier tui
```

快捷键：

- `s` - 服务列表（Enter 可**叠加**订阅多路，`*` 表示本地已连）
- `a` - 添加服务（含小红书）
- `c` - 配置 UniBarrage API/WS
- `p` - 插件配置
- `r` - 刷新远端服务列表
- `d` - **断开全部**本地订阅
- `q` - 退出

### CLI（管道 / 脚本）

```bash
# 支持的平台与别名
./dmnotifier platforms

# 同时订阅 B 站 + 小红书（阻塞收消息，Ctrl+C 退出）
./dmnotifier subscribe \
  -room bilibili:689422 \
  -room xhs:570409528162863169

# 只打 JSON 行（适合 jq / 下游管道）
./dmnotifier subscribe -room xhs:ROOM_ID -json -no-plugins | jq .

# 仅在 UniBarrage 上启动监听（不连本地 WS）
./dmnotifier add -platform xiaohongshu -rid ROOM_ID

# 列出 / 停止远端
./dmnotifier list
./dmnotifier stop -platform xhs -rid ROOM_ID

# 覆盖服务器地址
./dmnotifier subscribe -api http://127.0.0.1:8080 -ws ws://127.0.0.1:7777 -room dy:xxxx
```

`subscribe` 会：`POST` 启动 UniBarrage 监听 → 连接 `ws://.../{platform}/{rid}` → 消息进入插件 pipeline。

### 配置文件

路径：`~/.dmnotifier/config.yaml`

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
