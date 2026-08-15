package tui

import (
	"fmt"
	"os"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	tuimsg "github.com/xifan2333/dmnotifier/internal/common"
	"github.com/xifan2333/dmnotifier/internal/tui/business"
	"github.com/xifan2333/dmnotifier/pkg/api"
)

// Room is a platform/rid pair used as a TUI quick-entry target.
type Room struct {
	Platform string
	RID      string
	Cookie   string
}

// Options controls TUI startup (CLI quick entry).
type Options struct {
	// Connect rooms after the UI is up (start remote + local subscribe).
	Connect []Room
	// Optional one-shot server overrides (not written to disk).
	APIAddress string
	APIToken   string
	WSAddress  string
}

// Run starts the TUI main loop.
func Run(opts ...Options) error {
	var o Options
	if len(opts) > 0 {
		o = opts[0]
	}

	cfg, err := LoadConfig()
	if err != nil {
		cfg = DefaultConfig()
	}
	if o.APIAddress != "" {
		cfg.Server.APIAddress = o.APIAddress
	}
	if o.APIToken != "" {
		cfg.Server.APIToken = o.APIToken
	}
	if o.WSAddress != "" {
		cfg.Server.WSAddress = o.WSAddress
	}

	m := NewRootModel(cfg)
	wrapped := &businessLogicMiddleware{model: m, config: cfg}
	p := tea.NewProgram(wrapped, tea.WithAltScreen())
	wrapped.program = p
	mgr := business.NewManager(p, cfg)
	wrapped.mgr = mgr

	// Quick-entry: auto connect after first frame.
	if len(o.Connect) > 0 {
		go func() {
			// Let the alt-screen settle.
			time.Sleep(150 * time.Millisecond)
			for _, r := range o.Connect {
				svc := &api.Service{Platform: r.Platform, RID: r.RID}
				p.Send(tuimsg.ConnectServiceRequestMsg{Service: svc, Cookie: r.Cookie})
			}
		}()
	}

	if _, err := p.Run(); err != nil {
		mgr.Cleanup()
		return err
	}
	mgr.Cleanup()
	return nil
}

// businessLogicMiddleware 处理业务请求消息
type businessLogicMiddleware struct {
	model     tea.Model
	config    *AppConfig
	saveTimer *time.Timer
	saveMu    sync.Mutex
	program   *tea.Program
	mgr       *business.Manager
}

func (m *businessLogicMiddleware) Init() tea.Cmd { return m.model.Init() }

func (m *businessLogicMiddleware) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tuimsg.ConnectServiceRequestMsg:
		cmds = append(cmds, m.mgr.ConnectToService(msg.Service, msg.Cookie))

	case tuimsg.DisconnectServiceRequestMsg:
		m.mgr.DisconnectService()

	case tuimsg.DisconnectOneRequestMsg:
		cmds = append(cmds, m.mgr.DisconnectOne(msg.Platform, msg.RID))

	case tuimsg.RefreshServicesRequestMsg:
		cmds = append(cmds, m.mgr.FetchServices())

	case tuimsg.StopServiceRequestMsg:
		cmds = append(cmds, m.mgr.StopService(msg.Platform, msg.RID))

	case tuimsg.UpdateServerConfigMsg:
		m.mgr.UpdateServerConfig(msg.APIAddress, msg.APIToken, msg.WSAddress)
		cmds = append(cmds, func() tea.Msg {
			return tuimsg.StatusMsg{Message: "Server config updated"}
		})
		m.scheduleSave()

	case tuimsg.AddServiceRequestMsg:
		cmds = append(cmds, m.mgr.AddService(msg.Platform, msg.RID, msg.Cookie))

	case tuimsg.ReuseHistoryRequestMsg:
		cmds = append(cmds, m.mgr.ReuseHistory(msg.Platform, msg.RID, msg.Cookie))

	case tuimsg.DeleteHistoryEntryRequestMsg:
		cmds = append(cmds, m.mgr.DeleteFromHistory(msg.Platform, msg.RID))

	case tuimsg.SaveConfigRequestMsg:
		m.scheduleSave()

	case tuimsg.UpdatePluginsConfigMsg:
		m.mgr.UpdatePluginsConfig(msg.Plugins)
		m.scheduleSave()

	case tuimsg.ConnectSuccessMsg:
		cmds = append(cmds, func() tea.Msg {
			return tuimsg.StatusMsg{Message: fmt.Sprintf("Connected to %s/%s", msg.Service.Platform, msg.Service.RID)}
		})
	}

	var cmd tea.Cmd
	m.model, cmd = m.model.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

func (m *businessLogicMiddleware) View() string { return m.model.View() }

func (m *businessLogicMiddleware) scheduleSave() {
	m.saveMu.Lock()
	defer m.saveMu.Unlock()
	if m.saveTimer != nil {
		m.saveTimer.Stop()
	}
	m.saveTimer = time.AfterFunc(500*time.Millisecond, func() {
		if err := SaveConfig(m.config); err != nil {
			m.program.Send(tuimsg.ErrorMsg{Err: fmt.Errorf("failed to auto-save config: %w", err)})
		} else {
			m.program.Send(tuimsg.StatusMsg{Message: "Config auto-saved"})
		}
	})
}

// MustRun is like Run but exits the process on error (legacy helper).
func MustRun(opts ...Options) {
	if err := Run(opts...); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
