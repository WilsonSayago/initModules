package initModules

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

var errInvalidPort = errors.New("invalid port")

type testAppConfig struct {
	Port int `yaml:"port"`
}

func (c *testAppConfig) Validate() {
	if c.Port <= 0 {
		panic("validate called with invalid port")
	}
}

type validateTracker struct {
	Port            int `yaml:"port" properties:"port"`
	ValidateInvoked bool
}

func (v *validateTracker) Validate() {
	v.ValidateInvoked = true
}

type propertiesOnlyConfig struct {
	Port int `properties:"port"`
}

type validatorConfig struct {
	Port int `yaml:"port"`
}

func (c *validatorConfig) Validate() error {
	if c.Port <= 0 {
		return errInvalidPort
	}
	return nil
}

type validatorTracker struct {
	Port            int `yaml:"port"`
	ValidateInvoked bool
}

func (v *validatorTracker) Validate() error {
	v.ValidateInvoked = true
	return nil
}

func resetPropsForTest(t *testing.T) {
	t.Helper()
	reset := func() {
		props = nil
		propPath = "resources/properties.yml"
		propType = YML
	}
	reset()
	t.Cleanup(reset)
}

func testdataPath(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join("testdata", name)
}

func TestValidatePropTarget(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   interface{}
		wantErr bool
	}{
		{name: "pointer to struct", input: &testAppConfig{}, wantErr: false},
		{name: "struct value", input: testAppConfig{}, wantErr: true},
		{name: "pointer to string", input: new(string), wantErr: true},
		{name: "nil", input: nil, wantErr: true},
		{name: "typed nil", input: (*testAppConfig)(nil), wantErr: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validatePropTarget(tt.input)
			if tt.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantErr && err != nil && !strings.Contains(err.Error(), "pointer to struct") {
				t.Fatalf("unexpected error message: %v", err)
			}
		})
	}
}

func TestAddPropE(t *testing.T) {
	resetPropsForTest(t)

	if err := AddPropE(&testAppConfig{}); err != nil {
		t.Fatalf("AddPropE: %v", err)
	}
	if err := AddPropE(testAppConfig{}); err == nil {
		t.Fatal("expected error for non-pointer")
	}
}

func TestLoadProperties(t *testing.T) {
	tests := []struct {
		name           string
		file           string
		format         PropType
		expandEnv      bool
		env            map[string]string
		wantPort       int
		wantErr        bool
		wantValidate   bool
		usePropsConfig bool
	}{
		{
			name:         "valid yaml",
			file:         "valid.yml",
			format:       YML,
			wantPort:     8080,
			wantValidate: true,
		},
		{
			name:    "invalid yaml",
			file:    "invalid.yml",
			format:  YML,
			wantErr: true,
		},
		{
			name:         "yaml env expand",
			file:         "env_expand.yml",
			format:       YML,
			expandEnv:    true,
			env:          map[string]string{"TEST_INITMODULES_PORT": "7777"},
			wantPort:     7777,
			wantValidate: true,
		},
		{
			name:           "valid properties",
			file:           "valid.properties",
			format:         PROPERTIES,
			wantPort:       9090,
			usePropsConfig: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			resetPropsForTest(t)
			for k, v := range tt.env {
				t.Setenv(k, v)
			}

			var target interface{} = &validateTracker{}
			if tt.usePropsConfig {
				target = &propertiesOnlyConfig{}
			}
			if err := AddPropE(target); err != nil {
				t.Fatal(err)
			}

			opts := []Option{
				WithFilePath(testdataPath(t, tt.file)),
				WithFormat(tt.format),
			}
			if tt.format == YML {
				opts = append(opts, WithExpandEnv(tt.expandEnv || len(tt.env) > 0 || tt.file == "valid.yml"))
			}

			err := LoadProperties(opts...)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				if tr, ok := target.(*validateTracker); ok && tr.ValidateInvoked {
					t.Fatal("Validate must not run when load fails")
				}
				return
			}
			if err != nil {
				t.Fatalf("LoadProperties: %v", err)
			}
			switch cfg := target.(type) {
			case *validateTracker:
				if tt.wantValidate && !cfg.ValidateInvoked {
					t.Fatal("expected Validate to run")
				}
				if cfg.Port != tt.wantPort {
					t.Fatalf("port: got %d want %d", cfg.Port, tt.wantPort)
				}
			case *propertiesOnlyConfig:
				if cfg.Port != tt.wantPort {
					t.Fatalf("port: got %d want %d", cfg.Port, tt.wantPort)
				}
			}
		})
	}
}

func TestConfigLoader_Load(t *testing.T) {
	cfg := NewConfigLoader(
		WithFilePath(testdataPath(t, "valid.yml")),
		WithFormat(YML),
	)
	target := &testAppConfig{}
	if err := cfg.AddProp(target); err != nil {
		t.Fatal(err)
	}
	if err := cfg.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if target.Port != 8080 {
		t.Fatalf("port: got %d", target.Port)
	}
}

func TestLoadProperties_NoRegisteredProps(t *testing.T) {
	resetPropsForTest(t)
	if err := LoadProperties(WithFilePath(testdataPath(t, "valid.yml"))); err == nil {
		t.Fatal("expected error when no props registered")
	}
}

func TestProcessLoadedProp_SkipsValidateOnDecodeError(t *testing.T) {
	t.Parallel()

	tracker := &validateTracker{}
	err := processLoadedProp(tracker, os.ErrInvalid)
	if err == nil {
		t.Fatal("expected decode error to be returned")
	}
	if tracker.ValidateInvoked {
		t.Fatal("Validate must not run when decode fails")
	}
}

func TestProcessLoadedProp_PropValidatorSuccess(t *testing.T) {
	t.Parallel()

	tracker := &validatorTracker{Port: 8080}
	if err := processLoadedProp(tracker, nil); err != nil {
		t.Fatalf("processLoadedProp: %v", err)
	}
	if !tracker.ValidateInvoked {
		t.Fatal("expected PropValidator.Validate to run")
	}
}

func TestProcessLoadedProp_PropValidatorError(t *testing.T) {
	t.Parallel()

	cfg := &validatorConfig{Port: 0}
	err := processLoadedProp(cfg, nil)
	if !errors.Is(err, errInvalidPort) {
		t.Fatalf("error = %v, want errors.Is(_, errInvalidPort)", err)
	}
	if !strings.Contains(err.Error(), "validate") {
		t.Fatalf("error = %v, want wrapped target context", err)
	}
}

func TestProcessLoadedProp_LegacyPropStillRuns(t *testing.T) {
	t.Parallel()

	tracker := &validateTracker{Port: 8080}
	if err := processLoadedProp(tracker, nil); err != nil {
		t.Fatalf("processLoadedProp: %v", err)
	}
	if !tracker.ValidateInvoked {
		t.Fatal("expected legacy Prop.Validate to run when PropValidator is not implemented")
	}
}

func TestProcessLoadedProp_PrefersPropValidator(t *testing.T) {
	t.Parallel()

	// A single Go type cannot declare both Validate() and Validate() error.
	// Validate() error selects PropValidator and is the path used for new code.
	cfg := &validatorConfig{Port: 8080}
	if _, ok := any(cfg).(PropValidator); !ok {
		t.Fatal("validatorConfig should implement PropValidator")
	}
	if _, ok := any(cfg).(Prop); ok {
		t.Fatal("Validate() error must not satisfy legacy Prop")
	}
	if err := processLoadedProp(cfg, nil); err != nil {
		t.Fatalf("processLoadedProp: %v", err)
	}
}

func TestRunLoadProperties_FailsOnInvalidYAML(t *testing.T) {
	resetPropsForTest(t)

	tracker := &validateTracker{}
	if err := AddPropE(tracker); err != nil {
		t.Fatal(err)
	}

	if err := yamlUnmarshalTest([]byte("port: [not-an-int]\n"), tracker); err == nil {
		t.Fatal("expected unmarshal error")
	}
	if tracker.ValidateInvoked {
		t.Fatal("Validate must not run when unmarshal fails")
	}
}

func yamlUnmarshalTest(data []byte, target interface{}) error {
	return processLoadedProp(target, yaml.Unmarshal(data, target))
}

type nestedConfig struct {
	Nested struct {
		Value int `yaml:"value"`
	} `yaml:"nested"`
}

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

type failingValidator struct {
	Port int `yaml:"port"`
}

func (f *failingValidator) Validate() error {
	return errInvalidPort
}

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

func TestConfigLoader_LoadWithPropValidator(t *testing.T) {
	cfg := NewConfigLoader(
		WithFilePath(testdataPath(t, "valid.yml")),
		WithFormat(YML),
		WithStrictYAML(true),
		WithExpandEnv(false),
	)
	target := &validatorConfig{}
	if err := cfg.AddProp(target); err != nil {
		t.Fatal(err)
	}
	if err := cfg.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if target.Port != 8080 {
		t.Fatalf("port = %d", target.Port)
	}
}

func writeTempYAML(t *testing.T, contents string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
