package notify

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultCacheDir(t *testing.T) {
	d := defaultCacheDir()
	if filepath.Base(d) != "dmnotifier-avatars" {
		t.Fatalf("got %s", d)
	}
	if filepath.Dir(d) != os.TempDir() && filepath.Clean(filepath.Dir(d)) != filepath.Clean(os.TempDir()) {
		// TempDir may have trailing slash differences
		t.Logf("dir=%s temp=%s", filepath.Dir(d), os.TempDir())
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
