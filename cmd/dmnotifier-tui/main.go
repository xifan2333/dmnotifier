package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	tuimsg "github.com/xifan/dmnotifier/internal/common"
	"github.com/xifan/dmnotifier/internal/tui"
	"github.com/xifan/dmnotifier/internal/tui/business"

	// 导入插件以触发注册
	_ "github.com/xifan/dmnotifier/plugins/consumers/notify"
	_ "github.com/xifan/dmnotifier/plugins/consumers/tts"
	_ "github.com/xifan/dmnotifier/plugins/consumers/tui"
	_ "github.com/xifan/dmnotifier/plugins/consumers/webview"
	_ "github.com/xifan/dmnotifier/plugins/filters/message_type"
	_ "github.com/xifan/dmnotifier/plugins/transforms/format"
)

var businessManager *business.Manager

func main() {
	// 加载配置
	config, err := tui.LoadConfig()
	if err != nil {
		config = tui.GetDefaultConfig()
	}

	// 创建 TUI 模型
	m := tui.NewRootModel()

	// 使用中间件包装模型以处理业务逻辑
	wrappedModel := &BusinessLogicMiddleware{model: m}

	p := tea.NewProgram(wrappedModel, tea.WithAltScreen())

	// 创建业务逻辑管理器
	businessManager = business.NewManager(p, config)

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// 清理资源
	businessManager.Cleanup()
}

// BusinessLogicMiddleware 业务逻辑中间件
type BusinessLogicMiddleware struct {
	model tea.Model
}

func (m *BusinessLogicMiddleware) Init() tea.Cmd {
	return m.model.Init()
}

func (m *BusinessLogicMiddleware) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// 处理业务逻辑请求
	switch msg := msg.(type) {
	case tuimsg.ConnectServiceRequestMsg:
		cmds = append(cmds, businessManager.ConnectToService(msg.Service))

	case tuimsg.DisconnectServiceRequestMsg:
		businessManager.DisconnectService()

	case tuimsg.RefreshServicesRequestMsg:
		cmds = append(cmds, businessManager.FetchServices())

	case tuimsg.StopServiceRequestMsg:
		cmds = append(cmds, businessManager.StopService(msg.Platform, msg.RID))

	case tuimsg.UpdateServerConfigMsg:
		businessManager.UpdateServerConfig(msg.APIAddress, msg.APIToken, msg.WSAddress)
		cmds = append(cmds, func() tea.Msg {
			return tuimsg.StatusMsg{Message: "Server config updated (restart to apply)"}
		})

	case tuimsg.AddServiceRequestMsg:
		cmds = append(cmds, businessManager.AddService(msg.Platform, msg.RID, msg.Cookie))

	case tuimsg.UpdatePluginsConfigMsg:
		businessManager.UpdatePluginsConfig(msg.Plugins)

	case tuimsg.ConnectSuccessMsg:
		// 连接成功，发送后续消息
		cmds = append(cmds,
			func() tea.Msg {
				return tuimsg.ServiceConnectedMsg{Service: msg.Service}
			},
			func() tea.Msg {
				return tuimsg.StatusMsg{Message: fmt.Sprintf("Connected to %s/%s", msg.Service.Platform, msg.Service.RID)}
			},
		)

	case tuimsg.SaveConfigRequestMsg:
		if rootModel, ok := m.model.(tui.RootModel); ok {
			config := rootModel.GetConfig()
			if err := tui.SaveConfig(config); err != nil {
				cmds = append(cmds, func() tea.Msg {
					return tuimsg.ErrorMsg{Err: err}
				})
			} else {
				configPath, _ := tui.GetConfigPath()
				cmds = append(cmds, func() tea.Msg {
					return tuimsg.StatusMsg{Message: fmt.Sprintf("Config saved to %s", configPath)}
				})
			}
		}
	}

	// 传递给实际的模型
	var cmd tea.Cmd
	m.model, cmd = m.model.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *BusinessLogicMiddleware) View() string {
	return m.model.View()
}
