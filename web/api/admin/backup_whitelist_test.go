package admin

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyWhitelistedFilesIncludesPluginData(t *testing.T) {
	dataDir := t.TempDir()
	destDir := t.TempDir()
	src := filepath.Join(dataDir, "plugin-data", "demo", "state.json")
	if err := os.MkdirAll(filepath.Dir(src), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src, []byte("persistent-plugin-state"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := copyWhitelistedFilesFrom(dataDir, destDir); err != nil {
		t.Fatalf("copyWhitelistedFilesFrom failed: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(destDir, "plugin-data", "demo", "state.json"))
	if err != nil {
		t.Fatalf("plugin-data was not copied into backup staging: %v", err)
	}
	if string(got) != "persistent-plugin-state" {
		t.Fatalf("copied plugin data = %q", got)
	}
}
