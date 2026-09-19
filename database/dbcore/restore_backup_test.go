package dbcore

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeRestoreTestZip(t *testing.T, path string, entries map[string]string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(f)
	for name, body := range entries {
		entry, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestRestoreBackupArchiveRejectsInvalidBackupWithoutTouchingLiveData(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}
	livePath := filepath.Join(dataDir, "live.txt")
	if err := os.WriteFile(livePath, []byte("keep-me"), 0o644); err != nil {
		t.Fatal(err)
	}
	backupPath := filepath.Join(dataDir, "backup.zip")
	writeRestoreTestZip(t, backupPath, map[string]string{"new.txt": "untrusted"})

	if err := restoreBackupArchive(dataDir, backupPath); err == nil {
		t.Fatal("backup without markup was accepted")
	}
	got, err := os.ReadFile(livePath)
	if err != nil || string(got) != "keep-me" {
		t.Fatalf("live data changed after rejected restore: data=%q err=%v", got, err)
	}
	if _, err := os.Stat(backupPath); err != nil {
		t.Fatalf("rejected backup should remain available for inspection: %v", err)
	}
}

func TestRestoreBackupArchiveCommitsValidBackupAndArchivesInput(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "data")
	backupDir := filepath.Join(dataDir, "backup")
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "old.txt"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(backupDir, "keep.zip"), []byte("history"), 0o644); err != nil {
		t.Fatal(err)
	}

	backupPath := filepath.Join(dataDir, "backup.zip")
	writeRestoreTestZip(t, backupPath, map[string]string{
		"komarix-backup-markup":       "marker",
		"new.txt":                     "new",
		"plugin-data/demo/state.json": "plugin-state",
	})

	if err := restoreBackupArchive(dataDir, backupPath); err != nil {
		t.Fatalf("restoreBackupArchive failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dataDir, "old.txt")); !os.IsNotExist(err) {
		t.Fatalf("old live file still exists after successful restore: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dataDir, "new.txt"))
	if err != nil || string(got) != "new" {
		t.Fatalf("restored file = %q err=%v", got, err)
	}
	if _, err := os.Stat(filepath.Join(backupDir, "keep.zip")); err != nil {
		t.Fatalf("existing backup history was not preserved: %v", err)
	}
	if _, err := os.Stat(backupPath); !os.IsNotExist(err) {
		t.Fatalf("consumed backup.zip still present: %v", err)
	}

	entries, err := os.ReadDir(backupDir)
	if err != nil {
		t.Fatal(err)
	}
	var havePre, haveRestored bool
	for _, entry := range entries {
		havePre = havePre || strings.HasPrefix(entry.Name(), "pre-restore-")
		haveRestored = haveRestored || strings.HasPrefix(entry.Name(), "restored-")
	}
	if !havePre || !haveRestored {
		t.Fatalf("backup history missing pre/restored archives: pre=%v restored=%v", havePre, haveRestored)
	}
}
