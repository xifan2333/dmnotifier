package tui

import "github.com/xifan2333/dmnotifier/internal/config"

// Re-export config types used by TUI package.
type AppConfig = config.AppConfig

func LoadConfig() (*AppConfig, error) { return config.Load() }
func SaveConfig(c *AppConfig) error   { return config.Save(c) }
func DefaultConfig() *AppConfig       { return config.Default() }
