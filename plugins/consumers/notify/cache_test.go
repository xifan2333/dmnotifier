package notify

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultCacheDir(t *testing.T) {
	d := defaultCacheDir()
	want := filepath.Join(os.TempDir(), "dmnotifier", "avatars")
	if d != want {
		t.Fatalf("got %s want %s", d, want)
	}
}

func TestDownloadAvatar(t *testing.T) {
	dir := t.TempDir()
	c, err := NewAvatarCache(dir)
	if err != nil {
		t.Fatal(err)
	}
	path := c.Get("https://i0.hdslb.com/bfs/face/member/noface.jpg")
	if path == "" {
		t.Fatal("empty path")
	}
	st, err := os.Stat(path)
	if err != nil || st.Size() == 0 {
		t.Fatalf("bad file: %v %v", path, err)
	}
	if filepath.Ext(path) != ".jpg" {
		t.Fatalf("ext=%s path=%s", filepath.Ext(path), path)
	}
}
