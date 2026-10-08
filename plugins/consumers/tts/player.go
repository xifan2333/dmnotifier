package tts

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
)

func (c *Consumer) checkPlayer() error {
	names := []string{"mpv", "ffplay"}
	if runtime.GOOS == "windows" {
		names = []string{"ffplay", "mpv"}
	}
	if runtime.GOOS == "darwin" {
		names = append(names, "afplay")
	}
	for _, name := range names {
		if _, err := exec.LookPath(name); err == nil {
			c.player = name
			return nil
		}
	}
	return fmt.Errorf("no audio player found: install mpv or ffplay")
}

func playerArgs(player string, format audioFormat) []string {
	if player == "mpv" {
		args := []string{"--no-config", "--really-quiet", "--no-terminal", "--no-video", "--cache=no", "--audio-buffer=0.1", "--stream-buffer-size=4096", "--demuxer-lavf-buffersize=1024", "--demuxer-lavf-probe-info=no"}
		if format == pcm16 {
			args = append(args, "--demuxer-lavf-format=s16le", "--demuxer-lavf-o=sample_rate=24000,ch_layout=mono")
		} else {
			args = append(args, "--demuxer-lavf-format=mp3", "--demuxer-lavf-probesize=32", "--demuxer-lavf-analyzeduration=0")
		}
		return append(args, "-")
	}
	args := []string{"-hide_banner", "-loglevel", "error", "-nodisp", "-autoexit", "-probesize", "32", "-analyzeduration", "0"}
	if format == pcm16 {
		args = append(args, "-f", "s16le", "-ar", "24000", "-ch_layout", "mono")
	} else {
		args = append(args, "-f", "mp3")
	}
	return append(args, "-i", "pipe:0")
}

func (c *Consumer) playMessage(item *audioItem) error {
	ctx, cancel := context.WithCancel(c.ctx)
	defer cancel()
	stream, err := c.engine.Stream(ctx, item.prefix, item.content)
	if err != nil {
		return err
	}
	defer stream.Close()
	if c.player == "afplay" {
		return c.playFile(ctx, stream)
	}
	cmd := exec.CommandContext(ctx, c.player, playerArgs(c.player, stream.Format)...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	defer stdin.Close()
	if err = cmd.Start(); err != nil {
		return fmt.Errorf("start player: %w", err)
	}
	// Wait concurrently: an early player exit must cancel blocked network reads.
	exited := make(chan error, 1)
	go func() { err := cmd.Wait(); cancel(); exited <- err }()
	_, copyErr := stream.WriteTo(stdin)
	_ = stdin.Close()
	if copyErr != nil {
		cancel()
	}
	waitErr := <-exited
	if cause := context.Cause(ctx); cause != nil && cause != context.Canceled {
		return cause
	}
	if copyErr != nil {
		return fmt.Errorf("audio stream: %w", copyErr)
	}
	if waitErr != nil {
		return fmt.Errorf("player failed: %w", waitErr)
	}
	return nil
}

// afplay requires a seekable file. Only this explicit macOS fallback waits for
// all audio; raw MiMo PCM is wrapped in a proper WAV header after streaming.
func (c *Consumer) playFile(ctx context.Context, stream AudioStream) error {
	ext := ".mp3"
	if stream.Format == pcm16 {
		ext = ".wav"
	}
	f, err := os.CreateTemp("", "dmnotifier-tts-*"+ext)
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	defer f.Close()
	if ext == ".wav" {
		if _, err = f.Write(make([]byte, 44)); err != nil {
			return err
		}
	}
	size, err := io.CopyBuffer(f, struct{ io.Reader }{stream}, make([]byte, audioBuffer))
	if err != nil {
		return err
	}
	if ext == ".wav" {
		if size > int64(^uint32(0))-36 {
			return fmt.Errorf("WAV exceeds 4 GiB")
		}
		h := make([]byte, 44)
		copy(h, "RIFF")
		binary.LittleEndian.PutUint32(h[4:], uint32(size+36))
		copy(h[8:], "WAVEfmt ")
		binary.LittleEndian.PutUint32(h[16:], 16)
		binary.LittleEndian.PutUint16(h[20:], 1)
		binary.LittleEndian.PutUint16(h[22:], 1)
		binary.LittleEndian.PutUint32(h[24:], 24000)
		binary.LittleEndian.PutUint32(h[28:], 48000)
		binary.LittleEndian.PutUint16(h[32:], 2)
		binary.LittleEndian.PutUint16(h[34:], 16)
		copy(h[36:], "data")
		binary.LittleEndian.PutUint32(h[40:], uint32(size))
		if _, err = f.WriteAt(h, 0); err != nil {
			return err
		}
	}
	if err = f.Close(); err != nil {
		return err
	}
	return exec.CommandContext(ctx, "afplay", f.Name()).Run()
}
