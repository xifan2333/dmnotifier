package pipeline

import (
	"context"
	"sync"

	"github.com/xifan/dmnotifier/internal/plugin"
	"github.com/xifan/dmnotifier/pkg/models"
)

// Pipeline 消息处理管道
type Pipeline struct {
	name       string
	enabled    bool
	filters    []plugin.FilterPlugin
	transforms []plugin.TransformPlugin
	consumers  []plugin.ConsumerPlugin
	mu         sync.RWMutex
	wg         sync.WaitGroup // 追踪正在处理的消费者任务
}

// PipelineConfig 管道配置
type PipelineConfig struct {
	Name    string
	Enabled bool
}

// NewPipeline 创建新的管道
func NewPipeline(config PipelineConfig) *Pipeline {
	return &Pipeline{
		name:       config.Name,
		enabled:    config.Enabled,
		filters:    make([]plugin.FilterPlugin, 0),
		transforms: make([]plugin.TransformPlugin, 0),
		consumers:  make([]plugin.ConsumerPlugin, 0),
	}
}

// Name 返回管道名称
func (p *Pipeline) Name() string {
	return p.name
}

// IsEnabled 返回管道是否启用
func (p *Pipeline) IsEnabled() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.enabled
}

// SetEnabled 设置管道启用状态
func (p *Pipeline) SetEnabled(enabled bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.enabled = enabled
}

// AddFilter 添加过滤器
func (p *Pipeline) AddFilter(f plugin.FilterPlugin) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.filters = append(p.filters, f)

}

// AddTransform 添加转换器
func (p *Pipeline) AddTransform(t plugin.TransformPlugin) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.transforms = append(p.transforms, t)

}

// AddConsumer 添加消费者
func (p *Pipeline) AddConsumer(c plugin.ConsumerPlugin) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.consumers = append(p.consumers, c)

}

// Process 处理消息
func (p *Pipeline) Process(ctx context.Context, msg *models.Message) error {
	// 检查管道是否启用
	if !p.IsEnabled() {
		return nil
	}

	p.mu.RLock()
	filters := p.filters
	transforms := p.transforms
	consumers := p.consumers
	p.mu.RUnlock()

	// 阶段 1: 通过所有过滤器
	for _, filter := range filters {
		if !filter.Filter(ctx, msg) {

			return nil // 被过滤，不继续处理
		}
	}

	// 阶段 2: 应用所有转换器
	transformedMsg := msg
	for _, transform := range transforms {
		var err error
		transformedMsg, err = transform.Transform(ctx, transformedMsg)
		if err != nil {

			return err
		}
	}

	// 阶段 3: 分发给所有消费者（并行且异步，不等待完成）
	if len(consumers) == 0 {
		return nil
	}

	for _, consumer := range consumers {
		p.wg.Add(1)
		go func(c plugin.ConsumerPlugin, msg *models.Message) {
			defer p.wg.Done()
			if err := c.Consume(ctx, msg); err != nil {

			}
		}(consumer, transformedMsg)
	}

	// 立即返回，不等待消费者完成
	return nil
}

// GetStats 获取管道统计信息
func (p *Pipeline) GetStats() map[string]int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return map[string]int{
		"filters":    len(p.filters),
		"transforms": len(p.transforms),
		"consumers":  len(p.consumers),
	}
}

// Shutdown 关闭管道，停止所有插件
func (p *Pipeline) Shutdown(ctx context.Context) error {

	// 等待所有正在处理的消费者任务完成
	p.wg.Wait()

	p.mu.Lock()
	defer p.mu.Unlock()

	// 停止所有消费者
	for _, consumer := range p.consumers {
		if err := consumer.Stop(ctx); err != nil {

		}
	}

	// 停止所有转换器
	for _, transform := range p.transforms {
		if err := transform.Stop(ctx); err != nil {

		}
	}

	// 停止所有过滤器
	for _, filter := range p.filters {
		if err := filter.Stop(ctx); err != nil {

		}
	}

	return nil
}
