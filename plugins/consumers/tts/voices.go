package tts

// Chinese voices from the upstream catalog, 2026-10-08.
var edgeVoiceOptions, edgeVoiceLabels = edgeVoices()

func edgeVoices() ([]string, map[string]string) {
	voices := []struct{ name, gender string }{
		{"zh-CN-XiaoxiaoNeural", "Female"},
		{"zh-CN-XiaoyiNeural", "Female"},
		{"zh-CN-YunjianNeural", "Male"},
		{"zh-CN-YunxiNeural", "Male"},
		{"zh-CN-YunxiaNeural", "Male"},
		{"zh-CN-YunyangNeural", "Male"},
		{"zh-CN-liaoning-XiaobeiNeural", "Female"},
		{"zh-CN-shaanxi-XiaoniNeural", "Female"},
		{"zh-HK-HiuGaaiNeural", "Female"},
		{"zh-HK-HiuMaanNeural", "Female"},
		{"zh-HK-WanLungNeural", "Male"},
		{"zh-TW-HsiaoChenNeural", "Female"},
		{"zh-TW-HsiaoYuNeural", "Female"},
		{"zh-TW-YunJheNeural", "Male"},
	}
	options := make([]string, 0, len(voices))
	labels := make(map[string]string, len(voices))
	for _, voice := range voices {
		options = append(options, voice.name)
		labels[voice.name] = voice.name + " · " + voice.gender
	}
	return options, labels
}

// https://mimo.mi.com/static/docs/quick-start/usage-guide/audio/speech-synthesis-v2.5.md
var mimoVoiceLabels = map[string]string{
	"茉莉": "茉莉 · Chinese · Female", "冰糖": "冰糖 · Chinese · Female",
	"苏打": "苏打 · Chinese · Male", "白桦": "白桦 · Chinese · Male",
}
