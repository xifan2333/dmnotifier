package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	tuimsg "github.com/xifan2333/dmnotifier/internal/common"
	"github.com/xifan2333/dmnotifier/internal/plugin"
	"gopkg.in/yaml.v3"
)

// AvailableMessageTypes 默认消息类型
var AvailableMessageTypes = []string{"Chat", "Gift", "Like", "EnterRoom", "Subscribe", "SuperChat", "EndLive"}

// AppConfig 应用配置
type AppConfig struct {
	Server   ServerConfig                 `yaml:"server"`
	Client   ClientConfig                 `yaml:"client"`
	Pipeline PipelineConfig               `yaml:"pipeline"`
	History  []tuimsg.ServiceHistoryEntry `yaml:"history,omitempty"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	APIAddress string `yaml:"api_address"`
	APIToken   string `yaml:"api_token"`
	WSAddress  string `yaml:"ws_address"`
}

// ClientConfig 客户端配置
type ClientConfig struct {
	LogLevel string `yaml:"log_level"`
	Debug    bool   `yaml:"debug"`
}

// PipelineConfig 管道配置
type PipelineConfig struct {
	Plugins []tuimsg.PluginConfig `yaml:"plugins"`
}

// ConfigDir returns the application config directory.
//
// Linux / macOS (XDG Base Directory):
//
//	$XDG_CONFIG_HOME/dmnotifier  or  $HOME/.config/dmnotifier
//
// Windows:
//
//	%AppData%/dmnotifier
func ConfigDir() (string, error) {
	if runtime.GOOS == "windows" {
		if appData := os.Getenv("AppData"); appData != "" {
			return filepath.Join(appData, "dmnotifier"), nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("home directory: %w", err)
		}
		return filepath.Join(home, "AppData", "Roaming", "dmnotifier"), nil
	}

	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "dmnotifier"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("home directory: %w", err)
	}
	return filepath.Join(home, ".config", "dmnotifier"), nil
}

// GetConfigPath returns <ConfigDir>/config.yaml
func GetConfigPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

func ensureConfigDir() error {
	dir, err := ConfigDir()
	if err != nil {
		return err
	}
	return os.MkdirAll(dir, 0o755)
}

// Load reads config.yaml from the XDG path. Missing file → Default().
func Load() (*AppConfig, error) {
	path, err := GetConfigPath()
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return Default(), nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg AppConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &cfg, nil
}

// Save writes config.yaml under the XDG config directory.
func Save(cfg *AppConfig) error {
	if err := ensureConfigDir(); err != nil {
		return err
	}
	path, err := GetConfigPath()
	if err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

// Default is the built-in config (local UniBarrage).
func Default() *AppConfig {
	return &AppConfig{
		Server: ServerConfig{
			APIAddress: "http://127.0.0.1:8080",
			APIToken:   "",
			WSAddress:  "ws://127.0.0.1:7777",
		},
		Client: ClientConfig{
			LogLevel: "INFO",
			Debug:    false,
		},
		Pipeline: PipelineConfig{
			Plugins: loadPluginConfigs(),
		},
	}
}

func loadPluginConfigs() []tuimsg.PluginConfig {
	pluginInfos := plugin.GlobalRegistry.GetAllConsumerPluginInfo()
	configs := make([]tuimsg.PluginConfig, 0, len(pluginInfos))
	for _, info := range pluginInfos {
		var c map[string]interface{}
		if len(info.ConfigTemplate) > 0 {
			c = make(map[string]interface{})
			for _, field := range info.ConfigTemplate {
				c[field.Name] = field.Default
			}
		}
		configs = append(configs, tuimsg.PluginConfig{
			Name:         info.Name,
			Enabled:      true,
			MessageTypes: append([]string{}, AvailableMessageTypes...),
			Config:       c,
		})
	}
	return configs
}
