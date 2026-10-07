package notify

import (
	"bytes"
	"context"
	"crypto/md5"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

const (
	maxAvatarBytes   = 8 << 20
	maxAvatarPixels  = 16 << 20
	avatarIconSize   = 256
	avatarRetryDelay = time.Minute
)

type avatarLoad struct {
	done chan struct{}
	path string
}

// AvatarCache stores validated PNG thumbnails, regardless of the source format.
type AvatarCache struct {
	cacheDir   string
	httpClient *http.Client
	mu         sync.Mutex
	cache      map[string]string
	pending    map[string]*avatarLoad
	retryAfter map[string]time.Time
	ctx        context.Context
	cancel     context.CancelFunc
	workers    sync.WaitGroup
}

// defaultCacheDir uses the OS temp dir: $TMPDIR/dmnotifier/avatars
func defaultCacheDir() string {
	return filepath.Join(os.TempDir(), "dmnotifier", "avatars")
}

func NewAvatarCache(cacheDir string) (*AvatarCache, error) {
	if cacheDir == "" {
		cacheDir = defaultCacheDir()
	}
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return nil, fmt.Errorf("create cache dir: %w", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &AvatarCache{
		cacheDir:   cacheDir,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		cache:      make(map[string]string),
		pending:    make(map[string]*avatarLoad),
		retryAfter: make(map[string]time.Time),
		ctx:        ctx,
		cancel:     cancel,
	}, nil
}

// cachedPath requires a.mu to be held.
func (a *AvatarCache) cachedPath(avatarURL string) string {
	if path := a.cache[avatarURL]; path != "" {
		if st, err := os.Stat(path); err == nil && st.Mode().IsRegular() {
			return path
		}
		delete(a.cache, avatarURL)
	}
	return ""
}

// Get converts an avatar once, then reuses its PNG across messages and restarts.
// Concurrent requests for the same URL share the download and conversion.
// An empty result tells the consumer to use the embedded platform icon.
// The notification worker waits here so even the first message uses the avatar.
func (a *AvatarCache) Get(avatarURL string) string {
	if avatarURL == "" {
		return ""
	}

	a.mu.Lock()
	if a.ctx.Err() != nil || time.Now().Before(a.retryAfter[avatarURL]) {
		a.mu.Unlock()
		return ""
	}
	if path := a.cachedPath(avatarURL); path != "" {
		a.mu.Unlock()
		return path
	}
	if load, ok := a.pending[avatarURL]; ok {
		a.mu.Unlock()
		select {
		case <-load.done:
			return load.path
		case <-a.ctx.Done():
			return ""
		}
	}
	load := &avatarLoad{done: make(chan struct{})}
	a.pending[avatarURL] = load
	a.workers.Add(1)
	a.mu.Unlock()
	defer a.workers.Done()

	path, err := a.load(avatarURL)
	a.mu.Lock()
	if err == nil {
		load.path = path
		a.cache[avatarURL] = path
		delete(a.retryAfter, avatarURL)
	} else {
		a.retryAfter[avatarURL] = time.Now().Add(avatarRetryDelay)
	}
	delete(a.pending, avatarURL)
	close(load.done)
	a.mu.Unlock()
	return load.path
}

func (a *AvatarCache) load(avatarURL string) (string, error) {
	hash := md5.Sum([]byte(avatarURL))
	id := fmt.Sprintf("%x", hash)
	// Separate normalized files from legacy downloads, including old .png files.
	localPath := filepath.Join(a.cacheDir, id+".notify.png")
	if file, err := os.Open(localPath); err == nil {
		_, format, err := decodeAvatar(file)
		file.Close()
		if err == nil && format == "png" {
			return localPath, nil
		}
	}

	// Migrate completed legacy downloads without fetching the avatar again.
	// Never return raw images or unfinished .part files to a notification daemon.
	for _, ext := range []string{".jpg", ".jpeg", ".png", ".webp", ".gif"} {
		file, err := os.Open(filepath.Join(a.cacheDir, id+ext))
		if err != nil {
			continue
		}
		path, err := a.storePNG(localPath, file)
		file.Close()
		if err == nil {
			return path, nil
		}
	}

	req, err := http.NewRequestWithContext(a.ctx, http.MethodGet, avatarURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) dmnotifier/1.1")
	req.Header.Set("Accept", "image/png,image/jpeg,image/webp,image/gif")
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status: %s", resp.Status)
	}
	return a.storePNG(localPath, resp.Body)
}

func decodeAvatar(r io.Reader) (image.Image, string, error) {
	data, err := io.ReadAll(io.LimitReader(r, maxAvatarBytes+1))
	if err != nil {
		return nil, "", err
	}
	if len(data) > maxAvatarBytes {
		return nil, "", fmt.Errorf("avatar exceeds %d bytes", maxAvatarBytes)
	}
	config, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, "", err
	}
	if config.Width <= 0 || config.Height <= 0 || config.Width > maxAvatarPixels/config.Height {
		return nil, "", fmt.Errorf("avatar dimensions too large or invalid: %dx%d", config.Width, config.Height)
	}
	return image.Decode(bytes.NewReader(data))
}

func encodeAvatarPNG(w io.Writer, r io.Reader) error {
	img, _, err := decodeAvatar(r)
	if err != nil {
		return err
	}
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width > avatarIconSize || height > avatarIconSize {
		if width >= height {
			height = max(1, height*avatarIconSize/width)
			width = avatarIconSize
		} else {
			width = max(1, width*avatarIconSize/height)
			height = avatarIconSize
		}
		// RGBA enables the scaler's optimized JPEG/YCbCr path.
		thumb := image.NewRGBA(image.Rect(0, 0, width, height))
		draw.ApproxBiLinear.Scale(thumb, thumb.Bounds(), img, bounds, draw.Src, nil)
		img = thumb
	}
	encoder := png.Encoder{CompressionLevel: png.BestSpeed}
	return encoder.Encode(w, img)
}

func (a *AvatarCache) storePNG(localPath string, r io.Reader) (string, error) {
	if err := a.ctx.Err(); err != nil {
		return "", err
	}
	if err := os.MkdirAll(a.cacheDir, 0o755); err != nil {
		return "", err
	}
	file, err := os.CreateTemp(a.cacheDir, "avatar-*.part")
	if err != nil {
		return "", err
	}
	defer os.Remove(file.Name())
	if err := encodeAvatarPNG(file, r); err != nil {
		file.Close()
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	if err := a.ctx.Err(); err != nil {
		return "", err
	}
	if err := os.Rename(file.Name(), localPath); err != nil {
		return "", err
	}
	return localPath, nil
}

// Clear 清空缓存
func (a *AvatarCache) Clear() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.cache = make(map[string]string)
	a.retryAfter = make(map[string]time.Time)
	return os.RemoveAll(a.cacheDir)
}

// Close cancels pending HTTP requests and waits for active conversions.
func (a *AvatarCache) Close() {
	a.mu.Lock()
	a.cancel()
	a.mu.Unlock()
	a.workers.Wait()
}
