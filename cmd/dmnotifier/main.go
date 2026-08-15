package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/xifan2333/dmnotifier/internal/pipeline"
	"github.com/xifan2333/dmnotifier/internal/subscribe"
	"github.com/xifan2333/dmnotifier/internal/config"
	"github.com/xifan2333/dmnotifier/internal/tui"
	"github.com/xifan2333/dmnotifier/internal/tui/business"
	"github.com/xifan2333/dmnotifier/pkg/models"

	// 插件注册
	_ "github.com/xifan2333/dmnotifier/plugins/consumers/notify"
	_ "github.com/xifan2333/dmnotifier/plugins/consumers/tts"
	_ "github.com/xifan2333/dmnotifier/plugins/consumers/tui"
	_ "github.com/xifan2333/dmnotifier/plugins/consumers/webview"
	_ "github.com/xifan2333/dmnotifier/plugins/filters/message_type"
	_ "github.com/xifan2333/dmnotifier/plugins/transforms/format"
)

func main() {
	if len(os.Args) < 2 {
		tui.Run()
		return
	}

	cmd := os.Args[1]
	switch cmd {
	case "tui", "ui":
		os.Args = append([]string{os.Args[0]}, os.Args[2:]...)
		tui.Run()
	case "subscribe", "sub":
		runSubscribe(os.Args[2:])
	case "add":
		runAdd(os.Args[2:])
	case "stop", "rm", "remove":
		runStop(os.Args[2:])
	case "list", "ls":
		runList(os.Args[2:])
	case "platforms":
		runPlatforms()
	case "help", "-h", "--help":
		printHelp()
	case "version", "-v", "--version":
		fmt.Println("dmnotifier 1.1.2")
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", cmd)
		printHelp()
		os.Exit(2)
	}
}

func printHelp() {
	cfgPath, _ := config.GetConfigPath()
	if cfgPath == "" {
		cfgPath = "$XDG_CONFIG_HOME/dmnotifier/config.yaml"
	}
	fmt.Printf(`dmnotifier — UniBarrage 弹幕通知客户端

USAGE
  dmnotifier [command] [flags]

  无子命令时启动 TUI。

COMMANDS
  tui                     启动终端界面（与无参数相同）
  subscribe, sub          订阅一个或多个直播间并阻塞收消息（管道友好）
  add                     仅在 UniBarrage 上启动远端监听（不连本地 WS）
  stop, rm                停止 UniBarrage 上的远端监听
  list, ls                列出 UniBarrage 当前运行中的服务
  platforms               列出支持的平台 id 与别名
  help, -h, --help        显示本帮助
  version, -v, --version  显示版本

SUBSCRIBE FLAGS
  -room platform:rid      订阅目标，可重复；也支持逗号分隔多个
  -platform NAME          平台 id/别名，与 -rid 成对（可多次，按顺序配对）
  -rid ID                 房间号，与 -platform 成对
  -cookie STRING          可选 Cookie，应用到本次列出的全部房间
  -api URL                覆盖配置里的 api_address
  -ws URL                 覆盖配置里的 ws_address
  -token TOKEN            覆盖配置里的 api_token
  -plugins a,b,c          只启用列出的消费者插件（默认读配置文件）
  -json                   每条消息向 stdout 打一行 JSON
  -no-plugins             不跑插件 pipeline（适合纯 -json 管道）
  -timeout DURATION       到时自动退出，例如 30s、5m

ADD / STOP FLAGS
  与 subscribe 相同的 -room / -platform / -rid / -cookie / -api / -token

LIST FLAGS
  -api URL                覆盖 api_address
  -token TOKEN            覆盖 api_token

PLATFORMS（id 与别名）
  bilibili      bili
  douyin        dy
  xiaohongshu   xhs, red
  kuaishou      ks
  douyu
  huya

EXAMPLES
  # TUI
  dmnotifier
  dmnotifier tui

  # 同时听 B 站 + 小红书
  dmnotifier subscribe -room bilibili:689422 -room xhs:570409528162863169

  # 管道：JSON 行输出给 jq
  dmnotifier subscribe -room xhs:ROOM -json -no-plugins | jq .

  # 只启动远端监听 / 停止 / 列表
  dmnotifier add -platform xiaohongshu -rid ROOM
  dmnotifier stop -platform xhs -rid ROOM
  dmnotifier list

  # 覆盖本机 UniBarrage 地址
  dmnotifier subscribe -api http://127.0.0.1:8080 -ws ws://127.0.0.1:7777 -room dy:xxxx

CONFIG
  路径（Linux / macOS，XDG）:
    %s
  即 $XDG_CONFIG_HOME/dmnotifier/config.yaml
  未设置 XDG_CONFIG_HOME 时为 ~/.config/dmnotifier/config.yaml

  默认内容指向本机 UniBarrage:
    api_address: http://127.0.0.1:8080
    ws_address:  ws://127.0.0.1:7777
    api_token:   ""

  子命令 -api / -ws / -token 仅覆盖当次进程，不写回文件。
  TUI 里改服务器配置会保存到上述路径。
`, cfgPath)
}

func loadCfg() *config.AppConfig {
	cfg, err := config.Load()
	if err != nil || cfg == nil {
		return config.Default()
	}
	return cfg
}

type multiFlag []string

func (m *multiFlag) String() string { return strings.Join(*m, ",") }
func (m *multiFlag) Set(v string) error {
	for _, p := range strings.Split(v, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			*m = append(*m, p)
		}
	}
	return nil
}

func parseRoom(s string) (subscribe.Target, error) {
	s = strings.TrimSpace(s)
	i := strings.IndexByte(s, ':')
	if i <= 0 || i == len(s)-1 {
		return subscribe.Target{}, fmt.Errorf("invalid -room %q (want platform:rid)", s)
	}
	plat, ok := models.NormalizePlatform(s[:i])
	if !ok {
		return subscribe.Target{}, fmt.Errorf("unknown platform %q", s[:i])
	}
	return subscribe.Target{Platform: string(plat), RID: s[i+1:]}, nil
}

func parseRoomList(fs *flag.FlagSet, args []string) (rooms []subscribe.Target, apiURL, wsURL, token, cookie string, plugins string, asJSON, noPlugins bool, timeout time.Duration, err error) {
	var roomFlags multiFlag
	var platforms multiFlag
	var rids multiFlag
	fs.Var(&roomFlags, "room", "platform:rid (repeatable)")
	fs.Var(&platforms, "platform", "platform (pair with -rid)")
	fs.Var(&rids, "rid", "room id (pair with -platform)")
	fs.StringVar(&apiURL, "api", "", "api base url")
	fs.StringVar(&wsURL, "ws", "", "websocket base url")
	fs.StringVar(&token, "token", "", "api bearer token")
	fs.StringVar(&cookie, "cookie", "", "cookie for all rooms")
	fs.StringVar(&plugins, "plugins", "", "comma plugins to enable")
	fs.BoolVar(&asJSON, "json", false, "print JSON lines")
	fs.BoolVar(&noPlugins, "no-plugins", false, "disable plugin pipeline")
	fs.DurationVar(&timeout, "timeout", 0, "exit after duration")
	if err = fs.Parse(args); err != nil {
		return
	}
	for _, r := range roomFlags {
		var t subscribe.Target
		t, err = parseRoom(r)
		if err != nil {
			return
		}
		t.Cookie = cookie
		rooms = append(rooms, t)
	}
	if len(platforms) != len(rids) {
		err = fmt.Errorf("-platform and -rid count mismatch (%d vs %d)", len(platforms), len(rids))
		return
	}
	for i := range platforms {
		p, ok := models.NormalizePlatform(platforms[i])
		if !ok {
			err = fmt.Errorf("unknown platform %q", platforms[i])
			return
		}
		rooms = append(rooms, subscribe.Target{Platform: string(p), RID: rids[i], Cookie: cookie})
	}
	if len(rooms) == 0 {
		err = fmt.Errorf("no rooms: use -room platform:rid or -platform + -rid")
	}
	return
}

func applyServerFlags(cfg *config.AppConfig, apiURL, wsURL, token string) {
	if apiURL != "" {
		cfg.Server.APIAddress = apiURL
	}
	if wsURL != "" {
		cfg.Server.WSAddress = wsURL
	}
	if token != "" {
		cfg.Server.APIToken = token
	}
}

func enableOnly(cfg *config.AppConfig, names []string) {
	set := map[string]bool{}
	for _, n := range names {
		n = strings.TrimSpace(n)
		if n != "" {
			set[n] = true
		}
	}
	for i := range cfg.Pipeline.Plugins {
		cfg.Pipeline.Plugins[i].Enabled = set[cfg.Pipeline.Plugins[i].Name]
	}
}

func runSubscribe(args []string) {
	fs := flag.NewFlagSet("subscribe", flag.ExitOnError)
	rooms, apiURL, wsURL, token, _, plugins, asJSON, noPlugins, timeout, err := parseRoomList(fs, args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	cfg := loadCfg()
	applyServerFlags(cfg, apiURL, wsURL, token)
	if plugins != "" {
		enableOnly(cfg, strings.Split(plugins, ","))
	}
	if noPlugins {
		for i := range cfg.Pipeline.Plugins {
			cfg.Pipeline.Plugins[i].Enabled = false
		}
	}

	var build func() (*pipeline.Manager, error)
	if !noPlugins {
		build = func() (*pipeline.Manager, error) {
			return business.BuildPipelines(cfg, nil)
		}
	}

	sub := subscribe.New(subscribe.Config{
		APIAddress:    cfg.Server.APIAddress,
		APIToken:      cfg.Server.APIToken,
		WSAddress:     cfg.Server.WSAddress,
		BuildPipeline: build,
		OnMessage: func(msg *models.Message) {
			if !asJSON {
				return
			}
			out := map[string]any{
				"rid":      msg.RID,
				"platform": msg.Platform,
				"type":     msg.Type,
				"data":     json.RawMessage(msg.RawData),
			}
			b, err := json.Marshal(out)
			if err != nil {
				return
			}
			fmt.Println(string(b))
		},
		OnStatus: func(s string) { fmt.Fprintln(os.Stderr, "[status]", s) },
		OnError:  func(e error) { fmt.Fprintln(os.Stderr, "[error]", e) },
	})
	defer sub.Close()

	for _, r := range rooms {
		fmt.Fprintf(os.Stderr, "subscribe %s ...\n", r.Key())
		if err := sub.Subscribe(r); err != nil {
			fmt.Fprintf(os.Stderr, "subscribe %s failed: %v\n", r.Key(), err)
			os.Exit(1)
		}
	}
	fmt.Fprintf(os.Stderr, "listening %d room(s); Ctrl+C to quit\n", len(rooms))

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	if timeout > 0 {
		select {
		case <-sigCh:
		case <-time.After(timeout):
			fmt.Fprintln(os.Stderr, "timeout")
		}
	} else {
		<-sigCh
	}
	sub.DisconnectAll()
}

func runAdd(args []string) {
	fs := flag.NewFlagSet("add", flag.ExitOnError)
	rooms, apiURL, _, token, _, _, _, _, _, err := parseRoomList(fs, args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	cfg := loadCfg()
	applyServerFlags(cfg, apiURL, "", token)
	sub := subscribe.New(subscribe.Config{
		APIAddress: cfg.Server.APIAddress,
		APIToken:   cfg.Server.APIToken,
		WSAddress:  cfg.Server.WSAddress,
	})
	for _, r := range rooms {
		if err := sub.StartRemote(r.Platform, r.RID, r.Cookie); err != nil {
			fmt.Fprintf(os.Stderr, "add %s: %v\n", r.Key(), err)
			os.Exit(1)
		}
		fmt.Printf("started %s\n", r.Key())
	}
}

func runStop(args []string) {
	fs := flag.NewFlagSet("stop", flag.ExitOnError)
	rooms, apiURL, _, token, _, _, _, _, _, err := parseRoomList(fs, args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	cfg := loadCfg()
	applyServerFlags(cfg, apiURL, "", token)
	sub := subscribe.New(subscribe.Config{
		APIAddress: cfg.Server.APIAddress,
		APIToken:   cfg.Server.APIToken,
		WSAddress:  cfg.Server.WSAddress,
	})
	for _, r := range rooms {
		if err := sub.StopRemote(r.Platform, r.RID); err != nil {
			fmt.Fprintf(os.Stderr, "stop %s: %v\n", r.Key(), err)
			os.Exit(1)
		}
		fmt.Printf("stopped %s\n", r.Key())
	}
}

func runList(args []string) {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	apiURL := fs.String("api", "", "api base")
	token := fs.String("token", "", "token")
	_ = fs.Parse(args)
	cfg := loadCfg()
	applyServerFlags(cfg, *apiURL, "", *token)
	sub := subscribe.New(subscribe.Config{
		APIAddress: cfg.Server.APIAddress,
		APIToken:   cfg.Server.APIToken,
	})
	services, err := sub.API().GetAllServices()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if len(services) == 0 {
		fmt.Println("(no running services)")
		return
	}
	for _, s := range services {
		fmt.Printf("%s\t%s\n", s.Platform, s.RID)
	}
}

func runPlatforms() {
	fmt.Println("id\taliases")
	for _, p := range models.AllPlatforms {
		var aliases []string
		for a, v := range models.PlatformAliases {
			if v == p && a != string(p) {
				aliases = append(aliases, a)
			}
		}
		fmt.Printf("%s\t%s\n", p, strings.Join(aliases, ","))
	}
}
