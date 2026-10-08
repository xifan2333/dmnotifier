package tts

import (
	"context"
	"fmt"
	"strings"

	"github.com/xifan2333/dmnotifier/internal/plugin"
)

type engine interface {
	Stream(context.Context, string, string) (AudioStream, error)
	Close() error
}
type engineDefinition struct {
	id, label string
	fields    []plugin.ConfigField
	defaults  []plugin.ConfigField
	create    func(map[string]interface{}, func(Timing)) (engine, error)
}

var engines = []engineDefinition{
	{id: "mimo", label: "MiMo", create: newMiMo,
		fields: []plugin.ConfigField{
			{Name: "api_key", Label: "API key", Type: plugin.FieldTypeString, Default: ""},
			{Name: "voice", Label: "Voice", Type: plugin.FieldTypeEnum, Default: defaultVoice, Options: []string{"茉莉", "冰糖", "苏打", "白桦"}, OptionLabels: mimoVoiceLabels},
		},
		defaults: []plugin.ConfigField{
			{Name: "base_url", Type: plugin.FieldTypeString, Default: defaultBaseURL},
			{Name: "model", Type: plugin.FieldTypeEnum, Default: defaultModel, Options: []string{defaultModel}},
		}},
	{id: "edge", label: "Edge", create: newEdge, fields: []plugin.ConfigField{
		{Name: "edge_voice", Label: "Voice", Type: plugin.FieldTypeEnum, Default: defaultEdgeVoice, Options: edgeVoiceOptions, OptionLabels: edgeVoiceLabels},
	}},
}
var queueField = plugin.ConfigField{Name: "queue_size", Label: "Playback queue size", Type: plugin.FieldTypeNumber, Default: 100}

func configString(config map[string]interface{}, key, fallback string) string {
	value, _ := config[key].(string)
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}
func findEngine(config map[string]interface{}) (engineDefinition, error) {
	id := strings.ToLower(configString(config, "provider", defaultProvider))
	for _, definition := range engines {
		if definition.id == id {
			return definition, nil
		}
	}
	return engineDefinition{}, fmt.Errorf("unknown TTS provider %q", id)
}
func providerField() plugin.ConfigField {
	f := plugin.ConfigField{Name: "provider", Label: "Speech engine", Type: plugin.FieldTypeEnum, Default: defaultProvider, OptionLabels: map[string]string{}}
	for _, e := range engines {
		f.Options = append(f.Options, e.id)
		f.OptionLabels[e.id] = e.label
	}
	return f
}
func configFields(config map[string]interface{}) []plugin.ConfigField {
	fields := []plugin.ConfigField{providerField()}
	if definition, err := findEngine(config); err == nil {
		fields = append(fields, definition.fields...)
	}
	return append(fields, queueField)
}
func configTemplate() []plugin.ConfigField {
	fields := []plugin.ConfigField{providerField()}
	for _, e := range engines {
		fields = append(fields, e.fields...)
		fields = append(fields, e.defaults...)
	}
	return append(fields, queueField)
}
func init() {
	plugin.Register("tts", New, plugin.PluginInfo{Name: "tts", Type: plugin.TypeConsumer, ConfigTemplate: configTemplate(), ConfigFields: configFields})
}
