package initModules

import (
	"strings"
	"testing"
)

type envFieldsConfig struct {
	Port    int    `yaml:"port"`
	Bare    string `yaml:"bare"`
	Literal string `yaml:"literal"`
}

type envMultiConfig struct {
	Host string `yaml:"host"`
	Port string `yaml:"port"`
}

type envSecretConfig struct {
	Password string `yaml:"password"`
	Note     string `yaml:"note"`
}

type envTokenConfig struct {
	Token string `yaml:"token"`
}

func (c *envTokenConfig) Validate() error { return nil }

func TestLoadProperties_StrictYAMLKnownField(t *testing.T) {
	resetPropsForTest(t)

	target := &validatorConfig{Port: 1}
	if err := AddPropE(target); err != nil {
		t.Fatal(err)
	}
	err := LoadProperties(
		WithFilePath(testdataPath(t, "valid.yml")),
		WithFormat(YML),
		WithStrictYAML(true),
		WithExpandEnv(false),
	)
	if err != nil {
		t.Fatalf("LoadProperties: %v", err)
	}
	if target.Port != 8080 {
		t.Fatalf("port = %d, want 8080", target.Port)
	}
}

func TestLoadProperties_StrictYAMLUnknownField(t *testing.T) {
	resetPropsForTest(t)

	target := &validatorConfig{Port: 1}
	if err := AddPropE(target); err != nil {
		t.Fatal(err)
	}
	err := LoadProperties(
		WithFilePath(testdataPath(t, "unknown_field.yml")),
		WithFormat(YML),
		WithStrictYAML(true),
		WithExpandEnv(false),
	)
	if err == nil {
		t.Fatal("expected unknown field error")
	}
	if !strings.Contains(err.Error(), "unknown_typo") {
		t.Fatalf("error = %v, want unknown field name", err)
	}
	if target.Port != 1 {
		t.Fatalf("port = %d, want original 1", target.Port)
	}
}

func TestLoadProperties_NonStrictUnknownField(t *testing.T) {
	resetPropsForTest(t)

	target := &validatorConfig{Port: 1}
	if err := AddPropE(target); err != nil {
		t.Fatal(err)
	}
	err := LoadProperties(
		WithFilePath(testdataPath(t, "unknown_field.yml")),
		WithFormat(YML),
		WithExpandEnv(false),
	)
	if err != nil {
		t.Fatalf("legacy non-strict load: %v", err)
	}
	if target.Port != 8080 {
		t.Fatalf("port = %d, want 8080", target.Port)
	}
}

func TestLoadProperties_StrictYAMLMultipleTargets(t *testing.T) {
	resetPropsForTest(t)

	first := &validatorConfig{Port: 1}
	second := &validatorConfig{Port: 2}
	if err := AddPropE(first); err != nil {
		t.Fatal(err)
	}
	if err := AddPropE(second); err != nil {
		t.Fatal(err)
	}
	err := LoadProperties(
		WithFilePath(testdataPath(t, "does-not-exist.yml")),
		WithFormat(YML),
		WithStrictYAML(true),
	)
	if err == nil {
		t.Fatal("expected multi-target strict YAML error")
	}
	if !strings.Contains(err.Error(), "single configuration target") {
		t.Fatalf("error = %v, want explicit multi-target message", err)
	}
	if strings.Contains(err.Error(), "no such file") {
		t.Fatalf("I/O must not run before strict YAML multi-target check: %v", err)
	}
	if first.Port != 1 || second.Port != 2 {
		t.Fatalf("targets mutated: %d %d", first.Port, second.Port)
	}
}

func TestLoadProperties_StrictEnvPresent(t *testing.T) {
	resetPropsForTest(t)
	t.Setenv("TEST_INITMODULES_PORT", "7777")

	target := &envFieldsConfig{}
	if err := AddPropE(target); err != nil {
		t.Fatal(err)
	}
	err := LoadProperties(
		WithFilePath(testdataPath(t, "strict_env.yml")),
		WithFormat(YML),
		WithStrictEnv(true),
	)
	if err != nil {
		t.Fatalf("LoadProperties: %v", err)
	}
	if target.Port != 7777 {
		t.Fatalf("port = %d, want 7777", target.Port)
	}
	if target.Bare != "$TEST_INITMODULES_PORT" {
		t.Fatalf("bare = %q, want unexpanded $NAME", target.Bare)
	}
	if target.Literal != "$KEEP" {
		t.Fatalf("literal = %q, want $KEEP", target.Literal)
	}
}

func TestLoadProperties_StrictEnvPresentEmpty(t *testing.T) {
	resetPropsForTest(t)
	t.Setenv("TEST_INITMODULES_TOKEN", "")

	target := &envTokenConfig{Token: "old"}
	if err := AddPropE(target); err != nil {
		t.Fatal(err)
	}
	err := LoadProperties(
		WithFilePath(writeTempYAML(t, "token: \"${TEST_INITMODULES_TOKEN}\"\n")),
		WithFormat(YML),
		WithStrictEnv(true),
	)
	if err != nil {
		t.Fatalf("LoadProperties: %v", err)
	}
	if target.Token != "" {
		t.Fatalf("token = %q, want empty string", target.Token)
	}
}

func TestLoadProperties_StrictEnvAbsent(t *testing.T) {
	resetPropsForTest(t)

	target := &validatorConfig{Port: 1}
	if err := AddPropE(target); err != nil {
		t.Fatal(err)
	}
	err := LoadProperties(
		WithFilePath(testdataPath(t, "env_expand.yml")),
		WithFormat(YML),
		WithStrictEnv(true),
	)
	if err == nil {
		t.Fatal("expected missing env error")
	}
	if !strings.Contains(err.Error(), "TEST_INITMODULES_PORT") {
		t.Fatalf("error = %v, want variable name", err)
	}
	if target.Port != 1 {
		t.Fatalf("port = %d, want original 1", target.Port)
	}
}

func TestLoadProperties_StrictEnvMultiplePlaceholders(t *testing.T) {
	resetPropsForTest(t)
	t.Setenv("TEST_INITMODULES_HOST", "localhost")
	t.Setenv("TEST_INITMODULES_PORT", "8080")

	target := &envMultiConfig{}
	if err := AddPropE(target); err != nil {
		t.Fatal(err)
	}
	err := LoadProperties(
		WithFilePath(testdataPath(t, "strict_env_multi.yml")),
		WithFormat(YML),
		WithStrictEnv(true),
	)
	if err != nil {
		t.Fatalf("LoadProperties: %v", err)
	}
	if target.Host != "localhost" || target.Port != "8080" {
		t.Fatalf("got host=%q port=%q", target.Host, target.Port)
	}
}

func TestLoadProperties_StrictEnvDoesNotLeakSecret(t *testing.T) {
	resetPropsForTest(t)
	t.Setenv("UNRELATED_SECRET", "s3cret-value")

	target := &envSecretConfig{Password: "old", Note: "old"}
	if err := AddPropE(target); err != nil {
		t.Fatal(err)
	}
	err := LoadProperties(
		WithFilePath(testdataPath(t, "strict_env_secret.yml")),
		WithFormat(YML),
		WithStrictEnv(true),
	)
	if err == nil {
		t.Fatal("expected missing env error")
	}
	if !strings.Contains(err.Error(), "TEST_INITMODULES_MISSING") {
		t.Fatalf("error = %v, want variable name", err)
	}
	if strings.Contains(err.Error(), "s3cret-value") || strings.Contains(err.Error(), "UNRELATED_SECRET") {
		t.Fatalf("error leaked secret material: %v", err)
	}
	if target.Password != "old" || target.Note != "old" {
		t.Fatalf("target mutated: %+v", target)
	}
}
