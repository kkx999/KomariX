package plugin

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/kkx999/KomariX/database/models"
)

func findPluginManifest(files []*zip.File) (*zip.File, string, error) {
	var rootLegacy *zip.File
	for _, f := range files {
		name := filepath.ToSlash(f.Name)
		switch name {
		case manifestFile:
			return f, "", nil
		case legacyManifestFile:
			rootLegacy = f
		}
	}
	if rootLegacy != nil {
		return rootLegacy, "", nil
	}

	type candidate struct {
		manifest *zip.File
		legacy   bool
	}
	candidates := map[string]candidate{}
	for _, f := range files {
		name := filepath.ToSlash(f.Name)
		if strings.Count(name, "/") != 1 {
			continue
		}
		parts := strings.SplitN(name, "/", 2)
		if parts[0] == "" {
			continue
		}
		if parts[1] != manifestFile && parts[1] != legacyManifestFile {
			continue
		}
		current, exists := candidates[parts[0]]
		isLegacy := parts[1] == legacyManifestFile
		if !exists || (current.legacy && !isLegacy) {
			candidates[parts[0]] = candidate{manifest: f, legacy: isLegacy}
		}
	}
	if len(candidates) == 0 {
		return nil, "", nil
	}
	if len(candidates) > 1 {
		return nil, "", fmt.Errorf("plugin ZIP contains multiple top-level plugin roots")
	}
	for root, item := range candidates {
		return item.manifest, root + "/", nil
	}
	return nil, "", nil
}

// InspectZip validates a plugin archive and returns its manifest without changing installed plugins.
func InspectZip(zipPath string) (models.Plugin, error) {
	var info models.Plugin
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return info, fmt.Errorf("failed to open ZIP file: %v", err)
	}
	defer r.Close()
	if err := validatePluginArchive(r.File); err != nil {
		return info, err
	}
	manifest, _, err := findPluginManifest(r.File)
	if err != nil {
		return info, err
	}
	if manifest == nil {
		return info, fmt.Errorf("plugin manifest %s / %s not found, not a valid plugin package", manifestFile, legacyManifestFile)
	}
	rc, err := manifest.Open()
	if err != nil {
		return info, fmt.Errorf("failed to read plugin manifest: %v", err)
	}
	configData, readErr := io.ReadAll(io.LimitReader(rc, maxPluginManifestSize+1))
	_ = rc.Close()
	if readErr != nil {
		return info, fmt.Errorf("failed to read plugin manifest: %v", readErr)
	}
	if len(configData) > maxPluginManifestSize {
		return info, fmt.Errorf("plugin manifest exceeds the %d byte limit", maxPluginManifestSize)
	}
	if err := json.Unmarshal(configData, &info); err != nil {
		return info, fmt.Errorf("invalid plugin manifest: %v", err)
	}
	if err := validateManifest(&info); err != nil {
		return info, err
	}
	if err := CheckKomariXVersion(info.KomariX); err != nil {
		return info, err
	}
	return info, nil
}
// InstallZip validates a plugin ZIP and extracts it into DataDir/<short>.
// The archive uses komarix-plugin.json; third-party legacy packages are also accepted. Archive limits
// mirror the theme package format; path-traversal entries reject the whole
// package instead of being skipped. Reinstalling over a running plugin
// unloads it first and restores it to its persisted enabled state when the
// extraction succeeds.
func InstallZip(zipPath string) (models.Plugin, error) {
	var info models.Plugin
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return info, fmt.Errorf("failed to open ZIP file: %v", err)
	}
	defer r.Close()

	if err := validatePluginArchive(r.File); err != nil {
		return info, err
	}

	manifest, archiveRoot, err := findPluginManifest(r.File)
	if err != nil {
		return info, err
	}
	if manifest == nil {
		return info, fmt.Errorf("plugin manifest %s / %s not found, not a valid plugin package", manifestFile, legacyManifestFile)
	}

	rc, err := manifest.Open()
	if err != nil {
		return info, fmt.Errorf("failed to read plugin manifest: %v", err)
	}
	configData, readErr := io.ReadAll(io.LimitReader(rc, maxPluginManifestSize+1))
	_ = rc.Close()
	if readErr != nil {
		return info, fmt.Errorf("failed to read plugin manifest: %v", readErr)
	}
	if len(configData) > maxPluginManifestSize {
		return info, fmt.Errorf("plugin manifest exceeds the %d byte limit", maxPluginManifestSize)
	}
	if err := json.Unmarshal(configData, &info); err != nil {
		return info, fmt.Errorf("invalid plugin manifest: %v", err)
	}
	if err := validateManifest(&info); err != nil {
		return info, err
	}
	if err := CheckKomariXVersion(info.KomariX); err != nil {
		return info, err
	}

	if err := os.MkdirAll(DataDir, 0755); err != nil {
		return info, fmt.Errorf("failed to create plugin data directory: %v", err)
	}
	stageDir, err := os.MkdirTemp(DataDir, "."+info.Short+"-install-*")
	if err != nil {
		return info, fmt.Errorf("failed to create plugin staging directory: %v", err)
	}
	stagePublished := false
	defer func() {
		if !stagePublished {
			_ = os.RemoveAll(stageDir)
		}
	}()

	if err := extractPluginArchive(r.File, stageDir, archiveRoot); err != nil {
		return info, err
	}
	normalizedManifest, err := normalizePluginManifestJSON(configData)
	if err != nil {
		return info, fmt.Errorf("failed to normalize plugin manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(stageDir, manifestFile), normalizedManifest, 0644); err != nil {
		return info, fmt.Errorf("failed to write KomariX plugin manifest: %v", err)
	}
	_ = os.Remove(filepath.Join(stageDir, legacyManifestFile))
	if _, err := os.Stat(filepath.Join(stageDir, info.Entry)); err != nil {
		return info, fmt.Errorf("plugin entry %s does not exist", info.Entry)
	}
	for _, page := range info.Pages {
		if _, err := os.Stat(filepath.Join(stageDir, page.File)); err != nil {
			return info, fmt.Errorf("plugin page %s does not exist", page.File)
		}
	}

	dir := filepath.Join(DataDir, info.Short)
	enabled := global.stateStore().get(info.Short).Enabled
	if err := global.unload(info.Short); err != nil && !errors.Is(err, errNotLoaded) {
		return info, fmt.Errorf("failed to unload running plugin %q before reinstall: %w", info.Short, err)
	}

	rollbackDir := stageDir + ".rollback"
	_ = os.RemoveAll(rollbackDir)
	hadOld := false
	if _, err := os.Stat(dir); err == nil {
		if err := os.Rename(dir, rollbackDir); err != nil {
			if enabled {
				_ = global.restartPlugin(info.Short)
			}
			return info, fmt.Errorf("failed to preserve existing plugin before update: %v", err)
		}
		hadOld = true
	} else if !os.IsNotExist(err) {
		return info, fmt.Errorf("failed to inspect existing plugin directory: %v", err)
	}

	rollback := func(cause error) error {
		_ = os.RemoveAll(dir)
		if hadOld {
			if err := os.Rename(rollbackDir, dir); err != nil {
				return fmt.Errorf("%v; rollback failed: %w", cause, err)
			}
			if enabled {
				if err := global.restartPlugin(info.Short); err != nil {
					return fmt.Errorf("%v; old plugin restored but reload failed: %w", cause, err)
				}
			}
		}
		return cause
	}

	if err := os.Rename(stageDir, dir); err != nil {
		return info, rollback(fmt.Errorf("failed to publish plugin update: %w", err))
	}
	stagePublished = true
	if enabled {
		if err := global.restartPlugin(info.Short); err != nil {
			return info, rollback(fmt.Errorf("new plugin failed to reload: %w", err))
		}
	}
	if hadOld {
		_ = os.RemoveAll(rollbackDir)
	}
	return info, nil
}

func normalizePluginManifestJSON(data []byte) ([]byte, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	if _, ok := raw["komarix"]; !ok {
		if legacy, ok := raw["komari"]; ok {
			raw["komarix"] = legacy
		}
	}
	delete(raw, "komari")
	return json.MarshalIndent(raw, "", "  ")
}

func validatePluginArchive(files []*zip.File) error {
	if len(files) > maxPluginArchiveFiles {
		return fmt.Errorf("plugin archive has more than %d files", maxPluginArchiveFiles)
	}
	var total uint64
	for _, file := range files {
		if file.FileInfo().IsDir() {
			continue
		}
		if file.UncompressedSize64 > maxPluginFileSize {
			return fmt.Errorf("plugin file %s exceeds the %d byte limit", file.Name, maxPluginFileSize)
		}
		total += file.UncompressedSize64
		if total > maxPluginExtractedSize {
			return fmt.Errorf("plugin archive exceeds the %d byte extraction limit", maxPluginExtractedSize)
		}
	}
	return nil
}

func extractPluginArchive(files []*zip.File, dir, archiveRoot string) error {
	for _, f := range files {
		entryName := filepath.ToSlash(f.Name)
		if archiveRoot != "" {
			if !strings.HasPrefix(entryName, archiveRoot) {
				continue
			}
			entryName = strings.TrimPrefix(entryName, archiveRoot)
			if entryName == "" {
				continue
			}
		}
		path := filepath.Join(dir, filepath.FromSlash(entryName))
		if !withinDir(path, dir) {
			return fmt.Errorf("plugin archive contains an invalid path %q", f.Name)
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(path, 0755); err != nil {
				return fmt.Errorf("failed to create directory: %v", err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return fmt.Errorf("failed to create directory: %v", err)
		}
		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("failed to open archive file: %v", err)
		}
		outFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			_ = rc.Close()
			return fmt.Errorf("failed to create file: %v", err)
		}
		_, copyErr := io.Copy(outFile, rc)
		_ = outFile.Close()
		_ = rc.Close()
		if copyErr != nil {
			return fmt.Errorf("failed to extract file: %v", copyErr)
		}
	}
	return nil
}

// withinDir reports whether path stays inside dir after cleaning.
func withinDir(path, dir string) bool {
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) && !filepath.IsAbs(rel)
}
