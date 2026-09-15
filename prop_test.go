package initModules

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

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

func resetPropsForTest(t *testing.T) {
	t.Helper()
	props = nil
	propPath = "resources/properties.yml"
	propType = YML
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
		name        string
		file        string
		format      PropType
		expandEnv   bool
		env         map[string]string
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
			name:     "invalid yaml",
			file:     "invalid.yml",
			format:   YML,
			wantErr:  true,
		},
		{
			name:       "yaml env expand",
			file:       "env_expand.yml",
			format:     YML,
			expandEnv:  true,
			env:        map[string]string{"TEST_INITMODULES_PORT": "7777"},
			wantPort:   7777,
			wantValidate: true,
		},
		{
			name:     "valid properties",
			file:     "valid.properties",
			format:   PROPERTIES,
			wantPort: 9090,
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
