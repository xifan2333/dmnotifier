package config

import (
	"fmt"
	"os"
	"path/filepath"

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

// GetConfigPath 获取配置文件路径
func GetConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".dmnotifier", "config.yaml"), nil
}

func ensureConfigDir() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	return os.MkdirAll(filepath.Join(home, ".dmnotifier"), 0755)
}

// Load 加载配置
func Load() (*AppConfig, error) {
	configFile, err := GetConfigPath()
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return Default(), nil
	}
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	var config AppConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}
	return &config, nil
}

// Save 保存配置
func Save(config *AppConfig) error {
	if err := ensureConfigDir(); err != nil {
		return err
	}
	configFile, err := GetConfigPath()
	if err != nil {
		return err
	}
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	if err := os.WriteFile(configFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	return nil
}

// Default 默认配置
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
		var cfg map[string]interface{}
		if len(info.ConfigTemplate) > 0 {
			cfg = make(map[string]interface{})
			for _, field := range info.ConfigTemplate {
				cfg[field.Name] = field.Default
			}
		}
		configs = append(configs, tuimsg.PluginConfig{
			Name:         info.Name,
			Enabled:      true,
			MessageTypes: append([]string{}, AvailableMessageTypes...),
			Config:       cfg,
		})
	}
	return configs
}
