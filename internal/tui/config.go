package tui

import (
	"fmt"
	"os"
	"path/filepath"

	tuimsg "github.com/xifan2333/dmnotifier/internal/common"
	"github.com/xifan2333/dmnotifier/internal/plugin"
	"gopkg.in/yaml.v3"
)

// 可用的消息类型
var availableMessageTypes = []string{"Chat", "Gift", "Like", "EnterRoom", "Subscribe", "SuperChat", "EndLive"}

// AppConfig 应用配置
type AppConfig struct {
	Server   ServerConfig   `yaml:"server"`
	Client   ClientConfig   `yaml:"client"`
	Pipeline PipelineConfig `yaml:"pipeline"`
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

	configDir := filepath.Join(home, ".dmnotifier")
	configFile := filepath.Join(configDir, "config.yaml")

	return configFile, nil
}

// ensureConfigDir 确保配置目录存在
func ensureConfigDir() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(home, ".dmnotifier")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	return nil
}

// LoadConfig 加载配置文件
func LoadConfig() (*AppConfig, error) {
	configFile, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	// 如果配置文件不存在，返回默认配置
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return GetDefaultConfig(), nil
	}

	// 读取配置文件
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// 解析配置
	var config AppConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// SaveConfig 保存配置文件
func SaveConfig(config *AppConfig) error {
	// 确保配置目录存在
	if err := ensureConfigDir(); err != nil {
		return err
	}

	configFile, err := GetConfigPath()
	if err != nil {
		return err
	}

	// 序列化配置
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// 写入配置文件
	if err := os.WriteFile(configFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetDefaultConfig 获取默认配置
func GetDefaultConfig() *AppConfig {
	return &AppConfig{
		Server: ServerConfig{
			APIAddress: "https://danmu.xifan2333.fun",
			APIToken:   "Zhi-583379",
			WSAddress:  "ws://danmu.xifan2333.fun:7777",
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

// loadPluginConfigs 从插件注册表动态加载插件配置
func loadPluginConfigs() []tuimsg.PluginConfig {
	// 获取所有消费者插件信息
	pluginInfos := plugin.GlobalRegistry.GetAllConsumerPluginInfo()

	configs := make([]tuimsg.PluginConfig, 0, len(pluginInfos))
	for _, info := range pluginInfos {
		// 初始化默认配置
		var config map[string]interface{}
		if len(info.ConfigTemplate) > 0 {
			config = make(map[string]interface{})
			for _, field := range info.ConfigTemplate {
				config[field.Name] = field.Default
			}
		} // 如果没有配置项，config 保持为 nil

		configs = append(configs, tuimsg.PluginConfig{
			Name:         info.Name,
			Enabled:      true,
			MessageTypes: append([]string{}, availableMessageTypes...), // 创建副本，避免共享地址
			Config:       config,
		})
	}

	return configs
}
