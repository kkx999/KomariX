package upload

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCleanupExpiredRemovesOnlyStaleSessions(t *testing.T) {
	store := &Store{Root: t.TempDir(), MaxSize: 32 * 1024 * 1024}

	stale, err := store.Init(PurposeTheme, "stale.zip", 3)
	if err != nil {
		t.Fatal(err)
	}
	fresh, err := store.Init(PurposePlugin, "fresh.zip", 3)
	if err != nil {
		t.Fatal(err)
	}

	old := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(filepath.Join(stale.Directory, "upload.json"), old, old); err != nil {
		t.Fatal(err)
	}

	removed, err := store.CleanupExpired(24 * time.Hour)
	if err != nil {
		t.Fatalf("CleanupExpired failed: %v", err)
	}
	if removed != 1 {
		t.Fatalf("removed = %d, want 1", removed)
	}
	if _, err := os.Stat(stale.Directory); !os.IsNotExist(err) {
		t.Fatalf("stale upload session still exists: %v", err)
	}
	if _, err := os.Stat(fresh.Directory); err != nil {
		t.Fatalf("fresh upload session was removed: %v", err)
	}
}

func TestSaveChunkRefreshesSessionActivity(t *testing.T) {
	store := &Store{Root: t.TempDir(), MaxSize: 32 * 1024 * 1024}
	session, err := store.Init(PurposePlugin, "plugin.zip", 3)
	if err != nil {
		t.Fatal(err)
	}
	metaPath := filepath.Join(session.Directory, "upload.json")
	old := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(metaPath, old, old); err != nil {
		t.Fatal(err)
	}

	if err := store.SaveChunk(session.ID, 0, bytes.NewBufferString("abc")); err != nil {
		t.Fatalf("SaveChunk failed: %v", err)
	}
	info, err := os.Stat(metaPath)
	if err != nil {
		t.Fatal(err)
	}
	if !info.ModTime().After(old.Add(24 * time.Hour)) {
		t.Fatalf("metadata activity time was not refreshed: %v", info.ModTime())
	}
}
