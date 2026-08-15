package notify

import (
	"os"
	"testing"

	"github.com/xifan2333/dmnotifier/plugins/transforms/format"
)

func TestPlatformLogoPNG(t *testing.T) {
	dir := t.TempDir()
	c, err := NewAvatarCache(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{"bilibili", "douyin", "xiaohongshu", "kuaishou"} {
		url := format.PlatformIcon(p)
		path := c.Get(url)
		if path == "" {
			t.Fatalf("%s empty path url=%s", p, url)
		}
		st, err := os.Stat(path)
		if err != nil || st.Size() < 100 {
			t.Fatalf("%s bad file %v", p, err)
		}
		// raster image (png or jpeg favicon)
		b, _ := os.ReadFile(path)
		if len(b) < 4 {
			t.Fatalf("%s empty", p)
		}
		isPNG := b[0] == 0x89 && b[1] == 0x50
		isJPG := b[0] == 0xff && b[1] == 0xd8
		if !isPNG && !isJPG {
			t.Fatalf("%s not png/jpeg magic=%x", p, b[:4])
		}
	}
}
