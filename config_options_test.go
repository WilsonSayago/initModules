package initModules

import (
	"strings"
	"testing"
)

func TestLoadProperties_NoRegisteredProps(t *testing.T) {
	resetPropsForTest(t)
	if err := LoadProperties(WithFilePath(testdataPath(t, "valid.yml"))); err == nil {
		t.Fatal("expected error when no props registered")
	}
}

func TestLoadProperties_RejectsNilOptionBeforeIO(t *testing.T) {
	resetPropsForTest(t)
	if err := AddPropE(&validatorConfig{}); err != nil {
		t.Fatal(err)
	}
	err := LoadProperties(WithFilePath(testdataPath(t, "does-not-exist.yml")), nil)
	if err == nil || !strings.Contains(err.Error(), "option") {
		t.Fatalf("error = %v, want nil option", err)
	}
	if strings.Contains(err.Error(), "no such file") {
		t.Fatalf("I/O must not run: %v", err)
	}
}

func TestLoadProperties_RejectsInvalidFormatBeforeIO(t *testing.T) {
	resetPropsForTest(t)
	if err := AddPropE(&validatorConfig{Port: 1}); err != nil {
		t.Fatal(err)
	}
	err := LoadProperties(
		WithFilePath(testdataPath(t, "does-not-exist.yml")),
		WithFormat(PropType(99)),
	)
	if err == nil || !strings.Contains(err.Error(), "unsupported format") {
		t.Fatalf("error = %v, want invalid format", err)
	}
	if strings.Contains(err.Error(), "no such file") {
		t.Fatalf("I/O must not run: %v", err)
	}
}

func TestConfigLoader_RejectsNoTargetsBeforeIO(t *testing.T) {
	cfg := NewConfigLoader(WithFilePath(testdataPath(t, "does-not-exist.yml")))
	err := cfg.Load()
	if err == nil || !strings.Contains(err.Error(), "no properties registered") {
		t.Fatalf("error = %v, want missing targets", err)
	}
	if strings.Contains(err.Error(), "no such file") {
		t.Fatalf("I/O must not run: %v", err)
	}
}

func TestConfigLoader_RejectsTypedNilBeforeIO(t *testing.T) {
	cfg := NewConfigLoader(WithFilePath(testdataPath(t, "does-not-exist.yml")))
	err := cfg.AddProp((*validatorConfig)(nil))
	if err == nil || !strings.Contains(err.Error(), "pointer to struct") {
		t.Fatalf("error = %v, want typed-nil target", err)
	}
}
