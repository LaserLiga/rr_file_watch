package roadrunner

import "testing"

func TestConfigDefaultDebounceIsValid(t *testing.T) {
	cfg := &Config{}
	cfg.InitDefaults()

	debounce, err := cfg.DebounceDuration()
	if err != nil {
		t.Fatalf("default debounce should be valid: %v", err)
	}
	if debounce <= 0 {
		t.Fatalf("default debounce should be positive, got %s", debounce)
	}
}

func TestConfigDefaultsSingleWatchDir(t *testing.T) {
	cfg := &Config{}
	cfg.InitDefaults()

	dirs := cfg.WatchDirs()
	if len(dirs) != 1 {
		t.Fatalf("expected one default watch dir, got %d", len(dirs))
	}
	if dirs[0] != "./lmx/results" {
		t.Fatalf("unexpected default watch dir %q", dirs[0])
	}
}

func TestConfigUsesConfiguredDirs(t *testing.T) {
	cfg := &Config{Dirs: []string{"./lmx/results", "./lmx6/results"}}
	cfg.InitDefaults()

	dirs := cfg.WatchDirs()
	if len(dirs) != 2 {
		t.Fatalf("expected two watch dirs, got %d", len(dirs))
	}
	if dirs[0] != "./lmx/results" || dirs[1] != "./lmx6/results" {
		t.Fatalf("unexpected watch dirs %#v", dirs)
	}
}

func TestConfigKeepsLegacyDirWithDirs(t *testing.T) {
	cfg := &Config{Dir: "./lmx/results", Dirs: []string{"./lmx/results", "./lmx6/results"}}
	cfg.InitDefaults()

	dirs := cfg.WatchDirs()
	if len(dirs) != 2 {
		t.Fatalf("expected duplicate dirs to be removed, got %#v", dirs)
	}
	if dirs[0] != "./lmx/results" || dirs[1] != "./lmx6/results" {
		t.Fatalf("unexpected watch dirs %#v", dirs)
	}
}

func TestConfigRejectsInvalidDebounce(t *testing.T) {
	cfg := &Config{Debounce: "soon"}
	cfg.InitDefaults()

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected invalid debounce to fail validation")
	}
}

func TestConfigRejectsNegativeDebounce(t *testing.T) {
	cfg := &Config{Debounce: "-1s"}
	cfg.InitDefaults()

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected negative debounce to fail validation")
	}
}

func TestConfigAllowsZeroDebounce(t *testing.T) {
	cfg := &Config{Debounce: "0s"}
	cfg.InitDefaults()

	debounce, err := cfg.DebounceDuration()
	if err != nil {
		t.Fatalf("zero debounce should be valid: %v", err)
	}
	if debounce != 0 {
		t.Fatalf("expected zero debounce, got %s", debounce)
	}
}
