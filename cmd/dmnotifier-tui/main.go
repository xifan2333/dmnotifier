package main

import (
	"github.com/xifan2333/dmnotifier/internal/tui"

	// 导入插件以触发注册
	_ "github.com/xifan2333/dmnotifier/plugins/consumers/notify"
	_ "github.com/xifan2333/dmnotifier/plugins/consumers/tts"
	_ "github.com/xifan2333/dmnotifier/plugins/consumers/tui"
	_ "github.com/xifan2333/dmnotifier/plugins/consumers/webview"
	_ "github.com/xifan2333/dmnotifier/plugins/filters/message_type"
	_ "github.com/xifan2333/dmnotifier/plugins/transforms/format"
)

func main() {
	tui.Run()
}
