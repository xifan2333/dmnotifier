package platformicons

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Supported live platforms only (see pkg/models.AllPlatforms).
// All PNGs are 256×256.
//
// Used by:
//   - plugins/consumers/notify  (notification icon fallback)
//   - plugins/consumers/webview (/avatar/default/{platform})
//   - plugins/transforms/format (Avatar = "platform-icon:{id}" marker)

//go:embed icons/*.png
var iconFS embed.FS

var (
	once   sync.Once
	dir    string
	initErr error

	byPlat = map[string]string{
		"bilibili":    "bilibili.png",
		"douyin":      "douyin.png",
		"xiaohongshu": "xiaohongshu.png",
		"xhs":         "xhs.png",
		"kuaishou":    "kuaishou.png",
		"douyu":       "douyu.png",
		"huya":        "huya.png",
	}
)

func ensureDir() (string, error) {
	once.Do(func() {
		d := filepath.Join(os.TempDir(), "dmnotifier", "platform-icons")
		if err := os.MkdirAll(d, 0o755); err != nil {
			initErr = err
			return
		}
		entries, err := iconFS.ReadDir("icons")
		if err != nil {
			initErr = err
			return
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			data, err := iconFS.ReadFile("icons/" + name)
			if err != nil {
				initErr = err
				return
			}
			dst := filepath.Join(d, name)
			if st, err := os.Stat(dst); err != nil || st.Size() != int64(len(data)) {
				if err := os.WriteFile(dst, data, 0o644); err != nil {
					initErr = err
					return
				}
			}
		}
		dir = d
	})
	return dir, initErr
}

// Path returns a local filesystem path for the platform PNG, or "".
func Path(platform string) string {
	d, err := ensureDir()
	if err != nil || d == "" {
		return ""
	}
	file, ok := byPlat[platform]
	if !ok {
		return ""
	}
	p := filepath.Join(d, file)
	if _, err := os.Stat(p); err != nil {
		return ""
	}
	return p
}

// Bytes returns raw PNG bytes for the platform.
func Bytes(platform string) ([]byte, error) {
	file, ok := byPlat[platform]
	if !ok {
		return nil, fmt.Errorf("no icon for platform %q", platform)
	}
	return iconFS.ReadFile("icons/" + file)
}

// Has reports whether an icon exists for platform.
func Has(platform string) bool {
	_, ok := byPlat[platform]
	return ok
}
