package roadrunner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.uber.org/zap"
)

func TestValidWatchDirsSkipsMissingAndNonDirectoryPaths(t *testing.T) {
	tempDir := t.TempDir()
	validDir := filepath.Join(tempDir, "results")
	if err := os.Mkdir(validDir, 0755); err != nil {
		t.Fatalf("failed to create valid watch dir: %v", err)
	}
	filePath := filepath.Join(tempDir, "not-a-dir")
	if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to create non-directory path: %v", err)
	}

	plugin := &Plugin{
		cfg: &Config{Dirs: []string{
			filepath.Join(tempDir, "missing"),
			filePath,
			validDir,
		}},
		log: zap.NewNop(),
	}

	dirs, err := plugin.validWatchDirs()
	if err != nil {
		t.Fatalf("expected one valid watch dir, got error: %v", err)
	}
	if len(dirs) != 1 || dirs[0] != validDir {
		t.Fatalf("expected only %q to remain, got %#v", validDir, dirs)
	}
}

func TestValidWatchDirsFailsWhenNoConfiguredDirectoryIsUsable(t *testing.T) {
	tempDir := t.TempDir()
	plugin := &Plugin{
		cfg: &Config{Dirs: []string{filepath.Join(tempDir, "missing")}},
		log: zap.NewNop(),
	}

	_, err := plugin.validWatchDirs()
	if err == nil {
		t.Fatal("expected no usable watch directories to fail")
	}
	if !strings.Contains(err.Error(), "no configured watch directories") {
		t.Fatalf("unexpected error: %v", err)
	}
}
