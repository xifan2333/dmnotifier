package pipeline

import (
	"context"
	"fmt"
	"sync"

	"github.com/xifan/dmnotifier/pkg/models"
)

// Manager 管道管理器
type Manager struct {
	pipelines []*Pipeline
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewManager 创建新的管道管理器
func NewManager() *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		pipelines: make([]*Pipeline, 0),
		ctx:       ctx,
		cancel:    cancel,
	}
}

// AddPipeline 添加管道
func (m *Manager) AddPipeline(p *Pipeline) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pipelines = append(m.pipelines, p)

}

// GetPipeline 获取管道
func (m *Manager) GetPipeline(name string) (*Pipeline, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, p := range m.pipelines {
		if p.Name() == name {
			return p, nil
		}
	}

	return nil, fmt.Errorf("pipeline %s not found", name)
}

// GetAllPipelines 获取所有管道
func (m *Manager) GetAllPipelines() []*Pipeline {
	m.mu.RLock()
	defer m.mu.RUnlock()

	pipelines := make([]*Pipeline, len(m.pipelines))
	copy(pipelines, m.pipelines)
	return pipelines
}

// RemovePipeline 移除管道
func (m *Manager) RemovePipeline(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, p := range m.pipelines {
		if p.Name() == name {
			m.pipelines = append(m.pipelines[:i], m.pipelines[i+1:]...)

			return nil
		}
	}

	return fmt.Errorf("pipeline %s not found", name)
}

// Dispatch 分发消息到所有管道（完全异步，不等待）
func (m *Manager) Dispatch(ctx context.Context, msg *models.Message) {
	m.mu.RLock()
	pipelines := make([]*Pipeline, len(m.pipelines))
	copy(pipelines, m.pipelines)
	m.mu.RUnlock()

	// 并行处理所有管道，完全异步
	for _, pipeline := range pipelines {
		if !pipeline.IsEnabled() {
			continue
		}

		go func(p *Pipeline) {
			if err := p.Process(ctx, msg); err != nil {

			}
		}(pipeline)
	}
}

// EnablePipeline 启用管道
func (m *Manager) EnablePipeline(name string) error {
	pipeline, err := m.GetPipeline(name)
	if err != nil {
		return err
	}

	pipeline.SetEnabled(true)

	return nil
}

// DisablePipeline 禁用管道
func (m *Manager) DisablePipeline(name string) error {
	pipeline, err := m.GetPipeline(name)
	if err != nil {
		return err
	}

	pipeline.SetEnabled(false)

	return nil
}

// GetStats 获取管道管理器统计信息
func (m *Manager) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["total_pipelines"] = len(m.pipelines)

	enabledCount := 0
	pipelineStats := make([]map[string]interface{}, 0)

	for _, p := range m.pipelines {
		if p.IsEnabled() {
			enabledCount++
		}

		pipelineStats = append(pipelineStats, map[string]interface{}{
			"name":    p.Name(),
			"enabled": p.IsEnabled(),
			"stats":   p.GetStats(),
		})
	}

	stats["enabled_pipelines"] = enabledCount
	stats["pipelines"] = pipelineStats

	return stats
}

// Shutdown 关闭管道管理器
func (m *Manager) Shutdown() {

	// 先取消 context
	m.cancel()

	// 关闭所有 pipeline（每个 pipeline 会等待自己的任务完成）
	ctx := context.Background()
	m.mu.Lock()
	for _, p := range m.pipelines {
		if err := p.Shutdown(ctx); err != nil {

		}
	}
	m.mu.Unlock()

}
