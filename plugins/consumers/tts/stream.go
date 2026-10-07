package tts

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type audioFormat string

const (
	pcm16       audioFormat = "pcm16"
	mp3         audioFormat = "mp3"
	audioBuffer             = 4800 // 100 ms of 24 kHz mono PCM16.
)

// Timing separates service latency from delivery to the player. FirstWrite is
// stdin delivery, not a measurement of sound reaching the audio device.
type Timing struct {
	Provider, Part, Phase                     string
	Headers, FirstPacket, Request, FirstWrite time.Duration
	Cached                                    bool
}

type audioStream struct {
	io.ReadCloser
	Format  audioFormat
	firstAt *atomic.Int64
	done    <-chan struct{}
}

// stream starts immediately; the pipe applies backpressure without collecting
// an utterance. Closing it cancels both the request and any blocked pipe write.
func (c *Consumer) stream(parent context.Context, text, part string, cached bool, fail func(error)) audioStream {
	ctx, cancel := context.WithTimeout(parent, 2*time.Minute)
	pr, pw := io.Pipe()
	stop := context.AfterFunc(ctx, func() { _ = pw.CloseWithError(ctx.Err()) })
	done := make(chan struct{})
	firstAt := &atomic.Int64{}
	format := pcm16
	if c.provider == "edge" {
		format = mp3
	}
	go func() {
		defer close(done)
		defer stop()
		defer cancel()
		start := time.Now()
		timing := Timing{Provider: c.provider, Part: part, Phase: "synthesis"}
		key := c.provider + "\x00" + c.baseURL + "\x00" + c.model + "\x00" + c.voice + "\x00" + c.edgeVoice + "\x00" + text
		var err error
		if data := c.cache.get(key); cached && data != nil {
			timing.Cached = true
			firstAt.Store(time.Now().UnixNano())
			_, err = pw.Write(data)
		} else {
			w := &captureWriter{dst: pw, cache: cached, first: func() { firstAt.Store(time.Now().UnixNano()); timing.FirstPacket = time.Since(start) }}
			if c.provider == "edge" {
				var r io.ReadCloser
				r, err = c.edgeClient.Stream(ctx, text)
				if err == nil {
					_, err = io.CopyBuffer(w, r, make([]byte, audioBuffer))
					_ = r.Close()
				}
			} else {
				err = c.mimo(ctx, text, w, func() { timing.Headers = time.Since(start) })
			}
			if err == nil && !w.seen {
				err = fmt.Errorf("empty audio response")
			}
			if err == nil && cached && w.cache {
				c.cache.put(key, w.data)
			}
		}
		timing.Request = time.Since(start)
		_ = pw.CloseWithError(err)
		if err != nil {
			fail(fmt.Errorf("%s %s synthesis: %w", c.provider, part, err))
		}
		c.emitTiming(timing)
	}()
	return audioStream{ReadCloser: &cancelReader{ReadCloser: pr, cancel: cancel}, Format: format, firstAt: firstAt, done: done}
}

type cancelReader struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (r *cancelReader) Close() error { r.cancel(); return r.ReadCloser.Close() }

// A single cache entry is capped too, so an unusually long username cannot
// grow a temporary capture without bound.
const cacheLimit = 8 << 20
const entryLimit = 256 << 10

type captureWriter struct {
	dst         io.Writer
	cache, seen bool
	data        []byte
	first       func()
}

func (w *captureWriter) Write(p []byte) (int, error) {
	if len(p) > 0 && !w.seen {
		w.seen = true
		w.first()
	}
	if w.cache {
		if len(w.data)+len(p) > entryLimit {
			w.cache = false
			w.data = nil
		} else {
			w.data = append(w.data, p...)
		}
	}
	return w.dst.Write(p)
}

type prefixEntry struct {
	key  string
	data []byte
}
type prefixCache struct {
	mu      sync.Mutex
	entries []prefixEntry
	size    int
}

func (c *prefixCache) get(key string) []byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i, e := range c.entries {
		if e.key == key {
			copy(c.entries[i:], c.entries[i+1:])
			c.entries[len(c.entries)-1] = e
			return e.data
		}
	}
	return nil
}
func (c *prefixCache) put(key string, data []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	size := len(key) + len(data)
	if len(data) == 0 || size > entryLimit {
		return
	}
	for _, e := range c.entries {
		if e.key == key {
			return
		}
	}
	for c.size+size > cacheLimit {
		e := c.entries[0]
		c.size -= len(e.key) + len(e.data)
		c.entries[0] = prefixEntry{}
		c.entries = c.entries[1:]
	}
	data = bytes.Clone(data)
	c.entries = append(c.entries, prefixEntry{key, data})
	c.size += size
}

func (c *Consumer) emitTiming(t Timing) {
	slog.Debug("tts timing", "provider", t.Provider, "part", t.Part, "phase", t.Phase, "headers", t.Headers, "first_packet", t.FirstPacket, "request", t.Request, "first_write", t.FirstWrite, "cached", t.Cached)
	if c.onTiming != nil {
		c.onTiming(t)
	}
}

func (c *Consumer) mimo(ctx context.Context, text string, w io.Writer, headers func()) error {
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
