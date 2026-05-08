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
