package tui

import (
	"github.com/xifan2333/dmnotifier/internal/config"
)

// 兼容旧导入路径的类型别名
type (
	AppConfig      = config.AppConfig
	ServerConfig   = config.ServerConfig
	ClientConfig   = config.ClientConfig
	PipelineConfig = config.PipelineConfig
)

// GetConfigPath / LoadConfig / SaveConfig / GetDefaultConfig 兼容包装
func GetConfigPath() (string, error)          { return config.GetConfigPath() }
func LoadConfig() (*AppConfig, error)         { return config.Load() }
func SaveConfig(c *AppConfig) error           { return config.Save(c) }
func GetDefaultConfig() *AppConfig            { return config.Default() }
