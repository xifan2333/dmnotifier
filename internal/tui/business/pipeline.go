package business

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	tuimsg "github.com/xifan2333/dmnotifier/internal/common"
	"github.com/xifan2333/dmnotifier/internal/pipeline"
	"github.com/xifan2333/dmnotifier/internal/plugin"
	"github.com/xifan2333/dmnotifier/internal/tui"
)

// BuildPipelines 根据配置构建所有 pipeline
func BuildPipelines(config *tui.AppConfig, program *tea.Program) (*pipeline.Manager, error) {
	ctx := context.Background()
	manager := pipeline.NewManager()

	// 为每个启用的消费者插件创建一个 pipeline
	for _, pluginCfg := range config.Pipeline.Plugins {
		if !pluginCfg.Enabled {
			continue
		}

		p, err := buildPipelineForConsumer(ctx, pluginCfg, program)
		if err != nil {
			continue
		}

		manager.AddPipeline(p)
	}

	return manager, nil
}

// buildPipelineForConsumer 为单个消费者插件构建 pipeline
func buildPipelineForConsumer(ctx context.Context, pluginCfg tuimsg.PluginConfig, program *tea.Program) (*pipeline.Pipeline, error) {
	// 创建 pipeline
	p := pipeline.NewPipeline(pipeline.PipelineConfig{
		Name:    fmt.Sprintf("%s_pipeline", pluginCfg.Name),
		Enabled: true,
	})

	// 1. 添加消息类型过滤器
	if len(pluginCfg.MessageTypes) > 0 {
		typeFilter, err := plugin.Create("message_type_filter")
		if err != nil {
			return nil, fmt.Errorf("failed to create message_type_filter: %w", err)
		}

		// 转换 MessageTypes 为 interface{} 切片
		types := make([]interface{}, len(pluginCfg.MessageTypes))
		for i, t := range pluginCfg.MessageTypes {
			types[i] = t
		}

		if err := typeFilter.Init(ctx, map[string]interface{}{
			"types": types,
		}); err != nil {
			return nil, fmt.Errorf("failed to init message_type_filter: %w", err)
		}

		if err := typeFilter.Start(ctx); err != nil {
			return nil, fmt.Errorf("failed to start message_type_filter: %w", err)
		}

		p.AddFilter(typeFilter.(plugin.FilterPlugin))
	}

	// 2. 添加格式化转换器
	formatTransform, err := plugin.Create("format_transform")
	if err != nil {
		return nil, fmt.Errorf("failed to create format_transform: %w", err)
	}

	if err := formatTransform.Init(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to init format_transform: %w", err)
	}

	if err := formatTransform.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start format_transform: %w", err)
	}

	p.AddTransform(formatTransform.(plugin.TransformPlugin))

	// 3. 添加消费者插件
	consumer, err := plugin.Create(pluginCfg.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer %s: %w", pluginCfg.Name, err)
	}

	// 为 TUI 插件传入 program 实例
	config := pluginCfg.Config
	if pluginCfg.Name == "tui" {
		if config == nil {
			config = make(map[string]interface{})
		}
		config["program"] = program
	}

	if err := consumer.Init(ctx, config); err != nil {
		return nil, fmt.Errorf("failed to init consumer %s: %w", pluginCfg.Name, err)
	}

	if err := consumer.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start consumer %s: %w", pluginCfg.Name, err)
	}

	p.AddConsumer(consumer.(plugin.ConsumerPlugin))

	return p, nil
}
