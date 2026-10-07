package tts

import (
	_ "embed"
	"encoding/json"
	"strings"
)

// Upstream voice catalog snapshot, fetched 2026-10-08. The selector exposes
// Chinese locales only and works offline.
// Source: https://speech.platform.bing.com/consumer/speech/synthesize/readaloud/voices/list?trustedclienttoken=6A5AA1D4EAFF4E9FB37E23D68491D6F4
//
//go:embed edge_voices.json
var edgeVoiceCatalog []byte

var edgeVoiceOptions, edgeVoiceLabels = parseEdgeVoices()

func parseEdgeVoices() ([]string, map[string]string) {
	var voices []struct{ ShortName, Gender, Locale string }
	if err := json.Unmarshal(edgeVoiceCatalog, &voices); err != nil {
		panic(err)
	}
	options := make([]string, 0, len(voices))
	labels := make(map[string]string, len(voices))
	for _, v := range voices {
		if !strings.HasPrefix(v.Locale, "zh-") {
			continue
		}
		options = append(options, v.ShortName)
		labels[v.ShortName] = v.ShortName + " · " + v.Gender
	}

	return options, labels
}

// https://mimo.mi.com/static/docs/quick-start/usage-guide/audio/speech-synthesis-v2.5.md
var mimoVoiceLabels = map[string]string{
	"茉莉": "茉莉 · Chinese · Female", "冰糖": "冰糖 · Chinese · Female",
	"苏打": "苏打 · Chinese · Male", "白桦": "白桦 · Chinese · Male",
}
