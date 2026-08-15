package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigPathXDG(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg-test-dmn")
	dir, err := ConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	if dir != "/tmp/xdg-test-dmn/dmnotifier" {
		t.Fatalf("dir=%s", dir)
	}
	p, err := GetConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	if p != filepath.Join("/tmp/xdg-test-dmn/dmnotifier", "config.yaml") {
		t.Fatalf("path=%s", p)
	}
	// without XDG
	os.Unsetenv("XDG_CONFIG_HOME")
	home, _ := os.UserHomeDir()
	dir, _ = ConfigDir()
	if !strings.HasSuffix(dir, filepath.Join(".config", "dmnotifier")) && dir != filepath.Join(home, ".config", "dmnotifier") {
		t.Fatalf("expected ~/.config/dmnotifier got %s", dir)
	}
}
