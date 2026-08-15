package notify

import (
	"os"
	"testing"

	"github.com/xifan2333/dmnotifier/internal/platformicons"
	"github.com/xifan2333/dmnotifier/plugins/transforms/format"
)

func TestPlatformIconsEmbedded(t *testing.T) {
	for _, p := range []string{"bilibili", "douyin", "xiaohongshu", "kuaishou", "douyu", "huya"} {
		path := platformicons.Path(p)
		if path == "" {
			t.Fatalf("%s: empty path", p)
		}
		st, err := os.Stat(path)
		if err != nil || st.Size() < 100 {
			t.Fatalf("%s: bad file %v", p, err)
		}
		b, err := platformicons.Bytes(p)
		if err != nil || len(b) < 8 || b[0] != 0x89 {
			t.Fatalf("%s: not png", p)
		}
	}
}

func TestResolveIconMarker(t *testing.T) {
	ref := format.PlatformIconRef("bilibili")
	p, ok := format.ParsePlatformIconRef(ref)
	if !ok || p != "bilibili" {
		t.Fatalf("marker %q", ref)
	}
	path := resolveIcon(nil, ref, "bilibili")
	if path == "" {
		t.Fatal("resolve empty")
	}
}
