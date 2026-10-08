package tts

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type mimoEngine struct {
	apiKey, baseURL, model, voice string
	httpClient                    *http.Client
	cache                         prefixCache
	emit                          func(Timing)
}

func newMiMo(config map[string]interface{}, emit func(Timing)) (engine, error) {
	key := configString(config, "api_key", "")
	if key == "" {
		return nil, fmt.Errorf("MiMo requires api_key")
	}
	return &mimoEngine{apiKey: key, baseURL: configString(config, "base_url", defaultBaseURL), model: configString(config, "model", defaultModel), voice: configString(config, "voice", defaultVoice), emit: emit,
		httpClient: &http.Client{Transport: &http.Transport{Proxy: http.ProxyFromEnvironment, MaxIdleConns: 8, MaxIdleConnsPerHost: 4, IdleConnTimeout: 90 * time.Second, TLSHandshakeTimeout: 10 * time.Second, ResponseHeaderTimeout: 30 * time.Second}}}, nil
}
func (c *mimoEngine) Close() error { c.httpClient.CloseIdleConnections(); return nil }
func (c *mimoEngine) Stream(ctx context.Context, prefix, content string) (AudioStream, error) {
	var parts []audioPart
	for i, text := range []string{prefix, content} {
		if text == "" {
			continue
		}
		part := "body"
		if i == 0 {
			part = "prefix"
		}
		parts = append(parts, audioPart{name: part, produce: func(ctx context.Context, w io.Writer, t *Timing) error {
			key := c.baseURL + "\x00" + c.model + "\x00" + c.voice + "\x00" + text
			if i == 0 {
				if data := c.cache.get(key); data != nil {
					t.Cached = true
					_, err := w.Write(data)
					return err
				}
			}
			capture := &captureWriter{dst: w, cache: i == 0}
			start := time.Now()
			err := c.synthesize(ctx, text, capture, func() { t.Headers = time.Since(start) })
			if err == nil && !capture.seen {
				return fmt.Errorf("empty audio response")
			}
			if err == nil && capture.cache {
				c.cache.put(key, capture.data)
			}
			return err
		}})
	}
	return newAudioStream(ctx, pcm16, "mimo", parts, c.emit), nil
}

func (c *mimoEngine) synthesize(ctx context.Context, text string, w io.Writer, headers func()) error {
	body, err := json.Marshal(map[string]any{
		"model": c.model, "messages": []map[string]string{{"role": "assistant", "content": text}},
		"audio": map[string]string{"format": "pcm16", "voice": c.voice}, "stream": true,
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.baseURL, "/")+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("api-key", c.apiKey)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	headers()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("MiMo HTTP %d: %s", resp.StatusCode, b)
	}
	return readSSE(resp.Body, w)
}

// SSE data may span lines; comments and non-data fields are ignored. A missing
// DONE marker is an interrupted stream, even if some audio already arrived.
func readSSE(r io.Reader, w io.Writer) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 4096), 1<<20)
	var event strings.Builder
	odd := false
	dispatch := func() (bool, error) {
		data := strings.TrimSpace(event.String())
		event.Reset()
		if data == "" {
			return false, nil
		}
		if data == "[DONE]" {
			if odd {
				return false, fmt.Errorf("truncated PCM16 sample")
			}
			return true, nil
		}
		var v struct {
			Choices []struct {
				Delta struct {
					Audio struct {
						Data string `json:"data"`
					} `json:"audio"`
				} `json:"delta"`
			} `json:"choices"`
			Error *struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal([]byte(data), &v); err != nil {
			return false, fmt.Errorf("MiMo SSE: %w", err)
		}
		if v.Error != nil {
			return false, fmt.Errorf("MiMo: %s", v.Error.Message)
		}
		for _, choice := range v.Choices {
			b, err := base64.StdEncoding.DecodeString(choice.Delta.Audio.Data)
			if err != nil {
				return false, fmt.Errorf("MiMo audio: %w", err)
			}
			if len(b)%2 != 0 {
				odd = !odd
			}
			if len(b) > 0 {
				if _, err = w.Write(b); err != nil {
					return false, err
				}
			}
		}
		return false, nil
	}
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			done, err := dispatch()
			if err != nil || done {
				return err
			}
			continue
		}
		if strings.HasPrefix(line, "data:") {
			if event.Len()+len(line) > 1<<20 {
				return fmt.Errorf("MiMo SSE event exceeds 1 MiB")
			}
			event.WriteString(strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " "))
			event.WriteByte('\n')
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if event.Len() > 0 {
		done, err := dispatch()
		if err != nil || done {
			return err
		}
	}
	return io.ErrUnexpectedEOF
}
