package tts

import (
	"context"
	"io"

	"github.com/xifan2333/dmnotifier/plugins/consumers/tts/internal/edge"
)

type edgeEngine struct {
	client edge.Client
	emit   func(Timing)
}

func newEdge(config map[string]interface{}, emit func(Timing)) (engine, error) {
	return &edgeEngine{client: edge.Client{Voice: configString(config, "edge_voice", defaultEdgeVoice)}, emit: emit}, nil
}
func (e *edgeEngine) Close() error { return nil }
func (e *edgeEngine) Stream(ctx context.Context, prefix, content string) (AudioStream, error) {
	parts := []audioPart{{name: "body", produce: func(ctx context.Context, w io.Writer, _ *Timing) error { return e.client.Write(ctx, prefix+content, w) }}}
	return newAudioStream(ctx, mp3, "edge", parts, e.emit), nil
}
