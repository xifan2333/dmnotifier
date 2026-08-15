package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	tuimsg "github.com/xifan2333/dmnotifier/internal/common"
	"github.com/xifan2333/dmnotifier/internal/config"
	"github.com/xifan2333/dmnotifier/internal/subscribe"
	"github.com/xifan2333/dmnotifier/internal/tui"
	"github.com/xifan2333/dmnotifier/pkg/models"

	// plugin registration (TUI pipeline)
	_ "github.com/xifan2333/dmnotifier/plugins/consumers/notify"
	_ "github.com/xifan2333/dmnotifier/plugins/consumers/tts"
	_ "github.com/xifan2333/dmnotifier/plugins/consumers/tui"
	_ "github.com/xifan2333/dmnotifier/plugins/consumers/webview"
	_ "github.com/xifan2333/dmnotifier/plugins/filters/message_type"
	_ "github.com/xifan2333/dmnotifier/plugins/transforms/format"
)

// version is overridden at link time:
//
//	go build -ldflags "-X main.version=v1.2.0" ./cmd/dmnotifier
var version = "dev"

const maxHistorySize = 50

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) == 0 {
		return runTUI()
	}

	switch args[0] {
	case "-h", "--help", "help":
		printHelp()
		return 0
	case "-v", "--version", "version":
		fmt.Println("dmnotifier", version)
		return 0
	case "start":
		return cmdStart(args[1:])
	case "stop":
		return cmdStop(args[1:])
	case "list", "ls":
		return cmdList(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", args[0])
		printHelp()
		return 2
	}
}

func printHelp() {
	fmt.Printf(`Work with UniBarrage live danmaku from the terminal.

USAGE
  dmnotifier
  dmnotifier <command> [arguments]

  No command opens the TUI (human UI).
  start / stop / list are for scripts and other programs (no TUI).

COMMANDS
  start <room>...   Start remote listener(s); save history
  stop  <room>...   Stop remote listener(s)
  list              List running remote listeners

  help              Show this help
  version           Show version

FLAGS
  -h, --help        Show this help
  -v, --version     Show version

ROOM
  platform:rid
  platform:rid:cookie

  Only the first two ':' separate fields; the rest is cookie as-is.
  Cookie is a plain string (compose with the shell if needed):

    bilibili:689422:"$(cat ~/secrets/bili.cookie)"

PLATFORMS
  bilibili  douyin  xiaohongshu  kuaishou  douyu  huya

EXIT CODES
  0   success
  1   runtime error (API / network)
  2   usage error

OUTPUT
  start   one line per room:  platform<TAB>rid
  stop    one line per room:  platform<TAB>rid
  list    one line per room:  platform<TAB>rid
  list --json   JSON array on stdout

EXAMPLES
  $ dmnotifier
  $ dmnotifier start xiaohongshu:570409528162863169
  $ dmnotifier start bilibili:689422:"$(cat ~/secrets/bili.cookie)"
  $ dmnotifier list
  $ dmnotifier list --json
  $ dmnotifier stop bilibili:689422 xiaohongshu:570409528162863169

CONFIG
  Edit server/plugins in the TUI (keys c / p). File:
    %s
`, configPathDisplay())
}

func runTUI() int {
	if err := tui.Run(tui.Options{}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func cmdStart(args []string) int {
	if hasHelp(args) {
		fmt.Print(`Start remote UniBarrage listener(s).

Does not open the TUI. Suitable for scripts and other programs.
Saves platform/rid/cookie to history for later TUI reuse.

USAGE
  dmnotifier start <room>...

ROOM
  platform:rid
  platform:rid:cookie

OUTPUT
  platform<TAB>rid   (one line per started room)

EXAMPLES
  $ dmnotifier start xiaohongshu:570409528162863169
  $ dmnotifier start bilibili:689422:"SESSDATA=..." xiaohongshu:2
`)
		return 0
	}
	rooms, err := parseRooms(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	if len(rooms) == 0 {
		fmt.Fprintln(os.Stderr, "usage: dmnotifier start <room>...")
		return 2
	}

	cfg := loadCfg()
	sub := newSub(cfg)
	for _, r := range rooms {
		if err := sub.StartRemote(r.Platform, r.RID, r.Cookie); err != nil {
			fmt.Fprintf(os.Stderr, "start %s: %v\n", r.Key(), err)
			return 1
		}
		prependHistory(cfg, r.Platform, r.RID, r.Cookie)
		fmt.Printf("%s\t%s\n", r.Platform, r.RID)
	}
	if err := config.Save(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "warning: save history: %v\n", err)
	}
	return 0
}

func cmdStop(args []string) int {
	if hasHelp(args) {
		fmt.Print(`Stop remote UniBarrage listener(s).

Does not open the TUI. History is kept.

USAGE
  dmnotifier stop <room>...

ROOM
  platform:rid
  platform:rid:cookie   (cookie ignored)

OUTPUT
  platform<TAB>rid   (one line per stopped room)

EXAMPLES
  $ dmnotifier stop bilibili:689422 xiaohongshu:570409528162863169
`)
		return 0
	}
	rooms, err := parseRooms(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	if len(rooms) == 0 {
		fmt.Fprintln(os.Stderr, "usage: dmnotifier stop <room>...")
		return 2
	}

	cfg := loadCfg()
	sub := newSub(cfg)
	for _, r := range rooms {
		if err := sub.StopRemote(r.Platform, r.RID); err != nil {
			fmt.Fprintf(os.Stderr, "stop %s: %v\n", r.Key(), err)
			return 1
		}
		fmt.Printf("%s\t%s\n", r.Platform, r.RID)
	}
	return 0
}

func cmdList(args []string) int {
	asJSON := false
	for _, a := range args {
		switch a {
		case "-h", "--help", "help":
			fmt.Print(`List running remote UniBarrage listeners.

USAGE
  dmnotifier list [--json]

OUTPUT
  default   platform<TAB>rid
  --json    JSON array
`)
			return 0
		case "--json":
			asJSON = true
		default:
			fmt.Fprintf(os.Stderr, "unknown flag %q\n", a)
			return 2
		}
	}

	cfg := loadCfg()
	sub := newSub(cfg)
	services, err := sub.API().GetAllServices()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if asJSON {
		b, err := json.MarshalIndent(services, "", "  ")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		fmt.Println(string(b))
		return 0
	}
	for _, s := range services {
		fmt.Printf("%s\t%s\n", s.Platform, s.RID)
	}
	return 0
}

// ---------- room parse ----------

// parseRoom splits platform:rid or platform:rid:cookie.
// Only the first two ':' are separators; everything after the second is cookie.
func parseRoom(s string) (subscribe.Target, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return subscribe.Target{}, fmt.Errorf("empty room")
	}
	i := strings.IndexByte(s, ':')
	if i <= 0 || i == len(s)-1 {
		return subscribe.Target{}, fmt.Errorf("invalid room %q (want platform:rid or platform:rid:cookie)", s)
	}
	platRaw := s[:i]
	rest := s[i+1:]
	plat, ok := models.ParsePlatform(platRaw)
	if !ok {
		return subscribe.Target{}, fmt.Errorf("unknown platform %q", platRaw)
	}

	rid := rest
	cookie := ""
	if j := strings.IndexByte(rest, ':'); j >= 0 {
		rid = rest[:j]
		cookie = rest[j+1:]
	}
	if rid == "" {
		return subscribe.Target{}, fmt.Errorf("invalid room %q (empty rid)", s)
	}
	return subscribe.Target{
		Platform: string(plat),
		RID:      rid,
		Cookie:   cookie,
	}, nil
}

func parseRooms(args []string) ([]subscribe.Target, error) {
	var out []subscribe.Target
	for _, a := range args {
		if strings.HasPrefix(a, "-") {
			return nil, fmt.Errorf("unexpected flag %q", a)
		}
		t, err := parseRoom(a)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, nil
}

// ---------- helpers ----------

func hasHelp(args []string) bool {
	for _, a := range args {
		if a == "-h" || a == "--help" || a == "help" {
			return true
		}
	}
	return false
}

func loadCfg() *config.AppConfig {
	cfg, err := config.Load()
	if err != nil || cfg == nil {
		return config.Default()
	}
	return cfg
}

func newSub(cfg *config.AppConfig) *subscribe.Manager {
	return subscribe.New(subscribe.Config{
		APIAddress: cfg.Server.APIAddress,
		APIToken:   cfg.Server.APIToken,
		WSAddress:  cfg.Server.WSAddress,
	})
}

func prependHistory(cfg *config.AppConfig, platform, rid, cookie string) {
	entry := tuimsg.ServiceHistoryEntry{Platform: platform, RID: rid, Cookie: cookie}
	next := make([]tuimsg.ServiceHistoryEntry, 0, len(cfg.History)+1)
	next = append(next, entry)
	for _, h := range cfg.History {
		if h.Platform == platform && h.RID == rid {
			continue
		}
		next = append(next, h)
	}
	if len(next) > maxHistorySize {
		next = next[:maxHistorySize]
	}
	cfg.History = next
}

func configPathDisplay() string {
	p, err := config.GetConfigPath()
	if err != nil || p == "" {
		return "$XDG_CONFIG_HOME/dmnotifier/config.yaml"
	}
	return p
}
