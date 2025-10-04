package notify

import (
	"crypto/md5"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
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

// NewAvatarCache 创建头像缓存管理器
func NewAvatarCache(cacheDir string) (*AvatarCache, error) {
	// 创建缓存目录
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
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

// Get 获取头像本地路径（如果不存在则下载）
func (a *AvatarCache) Get(avatarURL string) string {
	if avatarURL == "" {
		return ""
	}

	// 检查缓存
	a.mu.RLock()
	if localPath, ok := a.cache[avatarURL]; ok {
		a.mu.RUnlock()
		return localPath
	}
	a.mu.RUnlock()

	// 下载头像
	localPath, err := a.download(avatarURL)
	if err != nil {
		return ""
	}

	// 保存到缓存
	a.mu.Lock()
	a.cache[avatarURL] = localPath
	a.mu.Unlock()

	return localPath
}

// download 下载头像
func (a *AvatarCache) download(avatarURL string) (string, error) {
	// 生成本地文件名（使用 URL 的 MD5）
	hash := md5.Sum([]byte(avatarURL))
	filename := fmt.Sprintf("%x.png", hash)
	localPath := filepath.Join(a.cacheDir, filename)

	// 检查文件是否已存在
	if _, err := os.Stat(localPath); err == nil {
		return localPath, nil
	}

	// 下载文件
	resp, err := a.httpClient.Get(avatarURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status: %s", resp.Status)
	}

	// 保存到本地
	file, err := os.Create(localPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		os.Remove(localPath)
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
