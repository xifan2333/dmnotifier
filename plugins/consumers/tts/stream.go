package tts

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

type audioFormat string

const (
	pcm16       audioFormat = "pcm16" // 24 kHz, mono, signed 16-bit little endian.
	mp3         audioFormat = "mp3"
	audioBuffer             = 4800
)

type Timing struct {
	Provider, Part, Phase                     string
	Headers, FirstPacket, Request, FirstWrite time.Duration
	Cached                                    bool
}

type AudioStream struct {
	io.ReadCloser
	Format audioFormat
}

func (s AudioStream) WriteTo(w io.Writer) (int64, error) {
	if r, ok := s.ReadCloser.(io.WriterTo); ok {
		return r.WriteTo(w)
	}
	return io.CopyBuffer(w, s.ReadCloser, make([]byte, audioBuffer))
}

type audioPart struct {
	name    string
	produce func(context.Context, io.Writer, *Timing) error
}
type runningPart struct {
	reader    *io.PipeReader
	firstAt   atomic.Int64
	name      string
	delivered bool
}
type streamReader struct {
	ctx      context.Context
	cancel   context.CancelCauseFunc
	parts    []*runningPart
	index    int
	workers  sync.WaitGroup
	once     sync.Once
	provider string
	emit     func(Timing)
}

func newAudioStream(parent context.Context, format audioFormat, provider string, parts []audioPart, emit func(Timing)) AudioStream {
	ctx, cancel := context.WithCancelCause(parent)
	r := &streamReader{ctx: ctx, cancel: cancel, provider: provider, emit: emit}
	for _, part := range parts {
		pr, pw := io.Pipe()
		running := &runningPart{reader: pr, name: part.name}
		r.parts = append(r.parts, running)
		r.workers.Add(1)
		go func() {
			defer r.workers.Done()
			requestCtx, requestCancel := context.WithTimeout(ctx, 2*time.Minute)
			defer requestCancel()
			stop := context.AfterFunc(requestCtx, func() { _ = pw.CloseWithError(context.Cause(requestCtx)) })
			defer stop()
			start := time.Now()
			timing := Timing{Provider: provider, Part: part.name, Phase: "synthesis"}
			w := &firstPacketWriter{dst: pw, first: func() { running.firstAt.Store(time.Now().UnixNano()); timing.FirstPacket = time.Since(start) }}
			err := part.produce(requestCtx, w, &timing)
			if err == nil && !w.seen {
				err = fmt.Errorf("empty audio response")
			}
			if err != nil {
				cancel(fmt.Errorf("%s %s synthesis: %w", provider, part.name, err))
			}
			_ = pw.CloseWithError(err)
			timing.Request = time.Since(start)
			emit(timing)
		}()
	}
	return AudioStream{ReadCloser: r, Format: format}
}
func (r *streamReader) Read(b []byte) (int, error) {
	if len(b) == 0 {
		return 0, nil
	}
	for r.index < len(r.parts) {
		if err := context.Cause(r.ctx); err != nil {
			return 0, err
		}
		n, err := r.parts[r.index].reader.Read(b)
		if n > 0 {
			return n, err
		}
		if err != io.EOF {
			return n, err
		}
		r.index++
	}
	return 0, io.EOF
}
func (r *streamReader) WriteTo(w io.Writer) (int64, error) {
	b := make([]byte, audioBuffer)
	var total int64
	for {
		n, err := r.Read(b)
		if n > 0 {
			written, writeErr := w.Write(b[:n])
			total += int64(written)
			p := r.parts[r.index]
			if written > 0 && !p.delivered {
				p.delivered = true
				r.emit(Timing{Provider: r.provider, Part: p.name, Phase: "delivery", FirstWrite: time.Since(time.Unix(0, p.firstAt.Load()))})
			}
			if writeErr != nil {
				return total, writeErr
			}
			if written != n {
				return total, io.ErrShortWrite
			}
		}
		if err == io.EOF {
			return total, nil
		}
		if err != nil {
			return total, err
		}
	}
}
func (r *streamReader) Close() error {
	r.once.Do(func() {
		r.cancel(context.Canceled)
		for _, p := range r.parts {
			_ = p.reader.Close()
		}
		r.workers.Wait()
	})
	return nil
}

type firstPacketWriter struct {
	dst   io.Writer
	first func()
	seen  bool
}

func (w *firstPacketWriter) Write(b []byte) (int, error) {
	if len(b) > 0 && !w.seen {
		w.seen = true
		w.first()
	}
	return w.dst.Write(b)
}
func (c *Consumer) emitTiming(t Timing) {
	slog.Debug("tts timing", "provider", t.Provider, "part", t.Part, "phase", t.Phase, "headers", t.Headers, "first_packet", t.FirstPacket, "request", t.Request, "first_write", t.FirstWrite, "cached", t.Cached)
	if c.onTiming != nil {
		c.onTiming(t)
	}
}
