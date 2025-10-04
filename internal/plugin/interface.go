package plugin

import (
	"context"

	"github.com/xifan2333/dmnotifier/pkg/models"
)

// PluginType 插件类型
type PluginType string

const (
	TypeFilter    PluginType = "filter"    // 过滤器
	TypeTransform PluginType = "transform" // 转换器
	TypeConsumer  PluginType = "consumer"  // 消费者
)

// ConfigFieldType 配置字段类型
type ConfigFieldType string

const (
	FieldTypeBool   ConfigFieldType = "bool"   // 开关
	FieldTypeString ConfigFieldType = "string" // 文本输入
	FieldTypeNumber ConfigFieldType = "number" // 数字输入
	FieldTypeEnum   ConfigFieldType = "enum"   // 单选（枚举）
	FieldTypeArray  ConfigFieldType = "array"  // 多选
)

// ConfigField 配置字段定义
type ConfigField struct {
	Name    string          // 字段名称
	Type    ConfigFieldType // 字段类型
	Default interface{}     // 默认值
	Desc    string          // 字段描述
	Options []string        // 选项列表（用于 enum 和 array 类型）
}

// Plugin 插件基础接口
type Plugin interface {
	// Name 返回插件名称（唯一标识）
	Name() string

	// Type 返回插件类型
	Type() PluginType

	// ConfigTemplate 返回配置模板
	ConfigTemplate() []ConfigField

	// Init 初始化插件
	Init(ctx context.Context, config map[string]interface{}) error

	// Start 启动插件
	Start(ctx context.Context) error

	// Stop 停止插件
	Stop(ctx context.Context) error
}

// FilterPlugin 过滤器插件（返回 true 表示通过）
type FilterPlugin interface {
	Plugin
	// Filter 过滤消息，返回 true 表示通过，false 表示拦截
	Filter(ctx context.Context, msg *models.Message) bool
}

// TransformPlugin 转换器插件（可以修改消息）
type TransformPlugin interface {
	Plugin
	// Transform 转换消息，返回转换后的消息
	Transform(ctx context.Context, msg *models.Message) (*models.Message, error)
}

// ConsumerPlugin 消费者插件（最终处理）
type ConsumerPlugin interface {
	Plugin
	// Consume 消费消息
	Consume(ctx context.Context, msg *models.Message) error
}

// BasePlugin 插件基础实现（可选继承）
type BasePlugin struct {
	name   string
	pType  PluginType
	config map[string]interface{}
}

// NewBasePlugin 创建基础插件
func NewBasePlugin(name string, pType PluginType) *BasePlugin {
	return &BasePlugin{
		name:  name,
		pType: pType,
	}
}

func (p *BasePlugin) Name() string {
	return p.name
}

func (p *BasePlugin) Type() PluginType {
	return p.pType
}

func (p *BasePlugin) ConfigTemplate() []ConfigField {
	// BasePlugin 不提供配置模板，由注册时提供
	return []ConfigField{}
}

func (p *BasePlugin) Init(ctx context.Context, config map[string]interface{}) error {
	p.config = config
	return nil
}

func (p *BasePlugin) Start(ctx context.Context) error {
	return nil
}

func (p *BasePlugin) Stop(ctx context.Context) error {
	return nil
}

func (p *BasePlugin) GetConfig() map[string]interface{} {
	return p.config
}
