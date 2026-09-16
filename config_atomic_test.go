package initModules

import (
	"errors"
	"testing"
)

type nestedConfig struct {
	Nested struct {
		Value int `yaml:"value"`
	} `yaml:"nested"`
}

type defaultedConfig struct {
	Port int    `yaml:"port" properties:"port"`
	Host string `yaml:"host" properties:"host"`
}

func (c *defaultedConfig) Validate() error { return nil }

type nestedPtr struct {
	Value int `yaml:"value"`
}

type aliasConfig struct {
	Port   int               `yaml:"port"`
	Labels map[string]string `yaml:"labels"`
	Items  []string          `yaml:"items"`
	Nested *nestedPtr        `yaml:"nested"`
}

func (c *aliasConfig) Validate() error { return nil }

type failingValidator struct {
	Port int `yaml:"port"`
}

func (f *failingValidator) Validate() error {
	return errInvalidPort
}

func TestLoadProperties_AtomicRollbackOnDecodeError(t *testing.T) {
	resetPropsForTest(t)

	first := &validatorConfig{Port: 11}
	second := &nestedConfig{}
	second.Nested.Value = 22
	if err := AddPropE(first); err != nil {
		t.Fatal(err)
	}
	if err := AddPropE(second); err != nil {
		t.Fatal(err)
	}

	err := LoadProperties(
		WithFilePath(testdataPath(t, "mixed_nested.yml")),
		WithFormat(YML),
		WithExpandEnv(false),
	)
	if err == nil {
		t.Fatal("expected decode error")
	}
	if first.Port != 11 {
		t.Fatalf("first.Port = %d, want original 11", first.Port)
	}
	if second.Nested.Value != 22 {
		t.Fatalf("second.Nested.Value = %d, want original 22", second.Nested.Value)
	}
}

func TestLoadProperties_AtomicRollbackOnValidateError(t *testing.T) {
	resetPropsForTest(t)

	first := &validatorConfig{Port: 11}
	secondFail := &failingValidator{Port: 33}
	if err := AddPropE(first); err != nil {
		t.Fatal(err)
	}
	if err := AddPropE(secondFail); err != nil {
		t.Fatal(err)
	}

	err := LoadProperties(
		WithFilePath(testdataPath(t, "valid.yml")),
		WithFormat(YML),
		WithExpandEnv(false),
	)
	if !errors.Is(err, errInvalidPort) {
		t.Fatalf("error = %v, want validation sentinel", err)
	}
	if first.Port != 11 {
		t.Fatalf("first.Port = %d, want original 11", first.Port)
	}
	if secondFail.Port != 33 {
		t.Fatalf("second.Port = %d, want original 33", secondFail.Port)
	}
}

func TestLoadProperties_PreservesOmittedYAMLDefaults(t *testing.T) {
	resetPropsForTest(t)

	target := &defaultedConfig{Port: 1, Host: "localhost"}
	if err := AddPropE(target); err != nil {
		t.Fatal(err)
	}
	err := LoadProperties(
		WithFilePath(testdataPath(t, "partial.yml")),
		WithFormat(YML),
		WithExpandEnv(false),
	)
	if err != nil {
		t.Fatalf("LoadProperties: %v", err)
	}
	if target.Port != 9090 {
		t.Fatalf("port = %d, want 9090", target.Port)
	}
	if target.Host != "localhost" {
		t.Fatalf("host = %q, want preserved default", target.Host)
	}
}

func TestLoadProperties_PreservesOmittedPropertiesDefaults(t *testing.T) {
	resetPropsForTest(t)

	type propsDefault struct {
		Port int    `properties:"port"`
		Name string `properties:"-"`
	}
	cfg := &propsDefault{Port: 1, Name: "svc"}
	if err := AddPropE(cfg); err != nil {
		t.Fatal(err)
	}
	err := LoadProperties(
		WithFilePath(testdataPath(t, "partial.properties")),
		WithFormat(PROPERTIES),
	)
	if err != nil {
		t.Fatalf("LoadProperties: %v", err)
	}
	if cfg.Port != 9090 {
		t.Fatalf("port = %d, want 9090", cfg.Port)
	}
	if cfg.Name != "svc" {
		t.Fatalf("name = %q, want preserved default", cfg.Name)
	}
}

func TestLoadProperties_DecodeFailureDoesNotAliasComposites(t *testing.T) {
	resetPropsForTest(t)

	labels := map[string]string{"keep": "me"}
	items := []string{"orig"}
	nested := &nestedPtr{Value: 7}
	target := &aliasConfig{Port: 1, Labels: labels, Items: items, Nested: nested}
	if err := AddPropE(target); err != nil {
		t.Fatal(err)
	}

	err := LoadProperties(
		WithFilePath(testdataPath(t, "aliasing_fail.yml")),
		WithFormat(YML),
		WithExpandEnv(false),
	)
	if err == nil {
		t.Fatal("expected decode error")
	}
	assertCompositesUnchanged(t, labels, items, nested, target, 1)
}

func TestLoadProperties_ValidateFailureDoesNotAliasComposites(t *testing.T) {
	resetPropsForTest(t)

	labels := map[string]string{"keep": "me"}
	items := []string{"orig"}
	nested := &nestedPtr{Value: 7}
	first := &aliasConfig{Port: 1, Labels: labels, Items: items, Nested: nested}
	second := &failingValidator{Port: 33}
	if err := AddPropE(first); err != nil {
		t.Fatal(err)
	}
	if err := AddPropE(second); err != nil {
		t.Fatal(err)
	}

	err := LoadProperties(
		WithFilePath(testdataPath(t, "aliasing_ok.yml")),
		WithFormat(YML),
		WithExpandEnv(false),
	)
	if !errors.Is(err, errInvalidPort) {
		t.Fatalf("error = %v, want validation sentinel", err)
	}
	assertCompositesUnchanged(t, labels, items, nested, first, 1)
	if second.Port != 33 {
		t.Fatalf("second.Port = %d, want original 33", second.Port)
	}
}

func assertCompositesUnchanged(t *testing.T, labels map[string]string, items []string, nested *nestedPtr, target *aliasConfig, port int) {
	t.Helper()
	if target.Port != port {
		t.Fatalf("port = %d, want original %d", target.Port, port)
	}
	if labels["keep"] != "me" {
		t.Fatalf("original map mutated: %v", labels)
	}
	if _, ok := labels["injected"]; ok {
		t.Fatalf("original map gained aliased key: %v", labels)
	}
	if len(items) != 1 || items[0] != "orig" {
		t.Fatalf("original slice mutated: %v", items)
	}
	if nested.Value != 7 {
		t.Fatalf("original pointer mutated: %+v", nested)
	}
	if target.Labels["keep"] != "me" || len(target.Items) != 1 || target.Items[0] != "orig" || target.Nested.Value != 7 {
		t.Fatalf("target composites mutated: %+v", target)
	}
}
