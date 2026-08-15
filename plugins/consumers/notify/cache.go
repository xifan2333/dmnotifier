package notify

import (
	"crypto/md5"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// AvatarCache 头像缓存管理器
type AvatarCache struct {
	cacheDir   string
	httpClient *http.Client
	mu         sync.RWMutex
	cache      map[string]string // URL -> 本地路径
}

// defaultCacheDir uses the OS temp dir.
// Name is "dmnotifier-avatars" (flat), not "dmnotifier/avatars" — the latter collides
// when a local build binary is placed at $TMPDIR/dmnotifier.
func defaultCacheDir() string {
	return filepath.Join(os.TempDir(), "dmnotifier-avatars")
}

// NewAvatarCache 创建头像缓存管理器
func NewAvatarCache(cacheDir string) (*AvatarCache, error) {
	if cacheDir == "" {
		cacheDir = defaultCacheDir()
	}
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return nil, fmt.Errorf("create cache dir: %w", err)
	}
	return &AvatarCache{
		cacheDir: cacheDir,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		cache: make(map[string]string),
	}, nil
}

// Get 获取头像本地路径（不存在则下载）
func (a *AvatarCache) Get(avatarURL string) string {
	if avatarURL == "" {
		return ""
	}

	a.mu.RLock()
	if localPath, ok := a.cache[avatarURL]; ok {
		a.mu.RUnlock()
		if _, err := os.Stat(localPath); err == nil {
			return localPath
		}
		// stale map entry — fall through to re-download
	} else {
		a.mu.RUnlock()
	}

	localPath, err := a.download(avatarURL)
	if err != nil {
		return ""
	}

	a.mu.Lock()
	a.cache[avatarURL] = localPath
	a.mu.Unlock()
	return localPath
}

func extFromURLOrCT(avatarURL, contentType string) string {
	// content-type first
	ct := strings.ToLower(contentType)
	switch {
	case strings.Contains(ct, "jpeg"), strings.Contains(ct, "jpg"):
		return ".jpg"
	case strings.Contains(ct, "png"):
		return ".png"
	case strings.Contains(ct, "webp"):
		return ".webp"
	case strings.Contains(ct, "gif"):
		return ".gif"
	case strings.Contains(ct, "svg"):
		return ".svg"
	}

	u := strings.ToLower(strings.Split(avatarURL, "?")[0])
	for _, ext := range []string{".jpg", ".jpeg", ".png", ".webp", ".gif", ".svg"} {
		if strings.HasSuffix(u, ext) {
			if ext == ".jpeg" {
				return ".jpg"
			}
			return ext
		}
	}
	// sniff later; default jpg (common for live avatars)
	return ".jpg"
}

func sniffExt(head []byte) string {
	if len(head) >= 3 && head[0] == 0xff && head[1] == 0xd8 && head[2] == 0xff {
		return ".jpg"
	}
	if len(head) >= 8 && head[0] == 0x89 && head[1] == 0x50 && head[2] == 0x4e && head[3] == 0x47 {
		return ".png"
	}
	if len(head) >= 4 && head[0] == 0x47 && head[1] == 0x49 && head[2] == 0x46 {
		return ".gif"
	}
	if len(head) >= 12 && string(head[0:4]) == "RIFF" && string(head[8:12]) == "WEBP" {
		return ".webp"
	}
	s := strings.TrimSpace(string(head))
	if strings.HasPrefix(s, "<svg") || strings.HasPrefix(s, "<?xml") {
		return ".svg"
	}
	return ""
}

func (a *AvatarCache) download(avatarURL string) (string, error) {
	hash := md5.Sum([]byte(avatarURL))
	id := fmt.Sprintf("%x", hash)

	// reuse any existing extension for this hash
	if matches, _ := filepath.Glob(filepath.Join(a.cacheDir, id+".*")); len(matches) > 0 {
		return matches[0], nil
	}

	req, err := http.NewRequest(http.MethodGet, avatarURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) dmnotifier/1.1")
	req.Header.Set("Accept", "image/avif,image/webp,image/apng,image/*,*/*;q=0.8")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status: %s", resp.Status)
	}

	// read a bit for sniff + rest
	head := make([]byte, 512)
	n, _ := io.ReadFull(resp.Body, head)
	head = head[:n]
	ext := sniffExt(head)
	if ext == "" {
		ext = extFromURLOrCT(avatarURL, resp.Header.Get("Content-Type"))
	}
	// mako/notify often weak on svg — skip caching pure svg as icon if possible
	if ext == ".svg" {
		// still save; notify may ignore, but better than wrong .png
	}

	localPath := filepath.Join(a.cacheDir, id+ext)
	tmpPath := localPath + ".part"
	file, err := os.Create(tmpPath)
	if err != nil {
		return "", err
	}
	if _, err := file.Write(head); err != nil {
		file.Close()
		os.Remove(tmpPath)
		return "", err
	}
	if _, err := io.Copy(file, resp.Body); err != nil {
		file.Close()
		os.Remove(tmpPath)
		return "", err
	}
	if err := file.Close(); err != nil {
		os.Remove(tmpPath)
		return "", err
	}
	if err := os.Rename(tmpPath, localPath); err != nil {
		os.Remove(tmpPath)
		return "", err
	}
	return localPath, nil
}

// Clear 清空缓存
func (a *AvatarCache) Clear() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.cache = make(map[string]string)
	return os.RemoveAll(a.cacheDir)
}
