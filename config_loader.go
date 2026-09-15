package initModules

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/magiconair/properties"
	"gopkg.in/yaml.v3"
)

// loadSettings holds resolved options for a property load operation.
type loadSettings struct {
	filePath   string
	format     PropType
	expandEnv  bool
	strictYAML bool
	strictEnv  bool
}

// Option configures LoadProperties or NewConfigLoader.
type Option func(*loadSettings)

// WithFilePath sets the configuration file path.
func WithFilePath(path string) Option {
	return func(s *loadSettings) {
		s.filePath = path
	}
}

// WithFormat sets the configuration file format (YML or PROPERTIES).
func WithFormat(format PropType) Option {
	return func(s *loadSettings) {
		s.format = format
	}
}

// WithExpandEnv enables or disables environment variable expansion for YAML content (default: true).
func WithExpandEnv(expand bool) Option {
	return func(s *loadSettings) {
		s.expandEnv = expand
	}
}

// WithStrictYAML rejects unknown YAML fields. Default false preserves v1 compatibility.
// Strict YAML is supported only when a single target is registered; combine sections
// into one root struct when more than one target is needed.
func WithStrictYAML(strict bool) Option {
	return func(s *loadSettings) {
		s.strictYAML = strict
	}
}

// WithStrictEnv expands only ${NAME} placeholders, errors if NAME is unset, and
// treats $$ as a literal $. Default false preserves v1 os.ExpandEnv semantics
// (including $NAME). When true, this option performs expansion even if
// WithExpandEnv(false) was set.
func WithStrictEnv(strict bool) Option {
	return func(s *loadSettings) {
		s.strictEnv = strict
	}
}

// ConfigLoader loads configuration into registered property structs without using package-level state.
type ConfigLoader struct {
	filePath   string
	format     PropType
	expandEnv  bool
	strictYAML bool
	strictEnv  bool
	props      []interface{}
	optErr     error
}

// NewConfigLoader creates a loader. Defaults: YML, ExpandEnv true, path resources/properties.yml.
func NewConfigLoader(opts ...Option) *ConfigLoader {
	settings := defaultLoadSettings()
	err := applyOptions(&settings, opts)
	return &ConfigLoader{
		filePath:   settings.filePath,
		format:     settings.format,
		expandEnv:  settings.expandEnv,
		strictYAML: settings.strictYAML,
		strictEnv:  settings.strictEnv,
		props:      make([]interface{}, 0),
		optErr:     err,
	}
}

// AddProp registers a pointer to struct that will be populated on Load.
func (c *ConfigLoader) AddProp(p interface{}) error {
	if err := validatePropTarget(p); err != nil {
		return err
	}
	c.props = append(c.props, p)
	return nil
}

// Load reads the configuration file and decodes it into all registered properties.
func (c *ConfigLoader) Load() error {
	if c.optErr != nil {
		return c.optErr
	}
	return loadPropsFromFile(loadSettings{
		filePath:   c.filePath,
		format:     c.format,
		expandEnv:  c.expandEnv,
		strictYAML: c.strictYAML,
		strictEnv:  c.strictEnv,
	}, c.props)
}

func defaultLoadSettings() loadSettings {
	return loadSettings{
		filePath:  propPath,
		format:    propType,
		expandEnv: true,
	}
}

func applyOptions(settings *loadSettings, opts []Option) error {
	for i, opt := range opts {
		if opt == nil {
			return fmt.Errorf("load properties: option %d is nil", i)
		}
		opt(settings)
	}
	return nil
}

func resolveLoadSettings(opts []Option) (loadSettings, error) {
	settings := defaultLoadSettings()
	if err := applyOptions(&settings, opts); err != nil {
		return settings, err
	}
	return settings, nil
}

// AddPropE registers a property target on the global loader registry.
func AddPropE(p interface{}) error {
	if err := validatePropTarget(p); err != nil {
		return err
	}
	props = append(props, p)
	return nil
}

// LoadProperties loads all globally registered properties (see AddPropE / AddProp).
// Options override path, format, and ExpandEnv for this call only.
func LoadProperties(opts ...Option) error {
	settings, err := resolveLoadSettings(opts)
	if err != nil {
		return err
	}
	return loadPropsFromFile(settings, props)
}

// RunLoadPropertiesE is an alias for LoadProperties with global registry and settings.
func RunLoadPropertiesE() error {
	return LoadProperties()
}

func loadPropsFromFile(settings loadSettings, targets []interface{}) error {
	if err := validateLoadRequest(settings, targets); err != nil {
		return err
	}

	filename, err := filepath.Abs(settings.filePath)
	if err != nil {
		return fmt.Errorf("load properties: absolute path: %w", err)
	}

	temps := make([]interface{}, len(targets))
	for i, target := range targets {
		tmp, err := newTempTarget(target)
		if err != nil {
			return fmt.Errorf("load properties: %w", err)
		}
		temps[i] = tmp
	}

	switch settings.format {
	case YML:
		dataFile, err := os.ReadFile(filename)
		if err != nil {
			return fmt.Errorf("load properties: read file %s: %w", filename, err)
		}
		yamlPayload, err := expandYAMLPayload(string(dataFile), settings)
		if err != nil {
			return fmt.Errorf("load properties: expand env %s: %w", filename, err)
		}
		for _, tmp := range temps {
			if err := decodeYAML(yamlPayload, tmp, settings.strictYAML); err != nil {
				return fmt.Errorf("load properties: decode %s: %w", filename, err)
			}
			if err := validateLoadedProp(tmp); err != nil {
				return fmt.Errorf("load properties: decode %s: %w", filename, err)
			}
		}
	case PROPERTIES:
		propFile, err := properties.LoadFile(filename, properties.UTF8)
		if err != nil {
			return fmt.Errorf("load properties: read file %s: %w", filename, err)
		}
		for _, tmp := range temps {
			if err := processLoadedProp(tmp, propFile.Decode(tmp)); err != nil {
				return fmt.Errorf("load properties: decode %s: %w", filename, err)
			}
		}
	default:
		return fmt.Errorf("load properties: unsupported format %v", settings.format)
	}

	commitTargets(targets, temps)
	return nil
}

func validateLoadRequest(settings loadSettings, targets []interface{}) error {
	if len(targets) == 0 {
		return fmt.Errorf("LoadProperties: no properties registered; use AddPropE first")
	}
	for _, target := range targets {
		if err := validatePropTarget(target); err != nil {
			return fmt.Errorf("load properties: %w", err)
		}
	}
	switch settings.format {
	case YML, PROPERTIES:
	default:
		return fmt.Errorf("load properties: unsupported format %v", settings.format)
	}
	if settings.strictYAML && settings.format == YML && len(targets) > 1 {
		return fmt.Errorf("load properties: strict YAML requires a single configuration target; combine sections into one root struct")
	}
	if settings.filePath == "" {
		return fmt.Errorf("load properties: file path is empty")
	}
	return nil
}

func newTempTarget(target interface{}) (interface{}, error) {
	if err := validatePropTarget(target); err != nil {
		return nil, err
	}
	elemType := reflect.ValueOf(target).Elem().Type()
	return reflect.New(elemType).Interface(), nil
}

func commitTargets(dsts, srcs []interface{}) {
	for i := range dsts {
		reflect.ValueOf(dsts[i]).Elem().Set(reflect.ValueOf(srcs[i]).Elem())
	}
}

func decodeYAML(payload string, target interface{}, strict bool) error {
	if !strict {
		return yaml.Unmarshal([]byte(payload), target)
	}
	dec := yaml.NewDecoder(bytes.NewReader([]byte(payload)))
	dec.KnownFields(true)
	return dec.Decode(target)
}

func expandYAMLPayload(payload string, settings loadSettings) (string, error) {
	if settings.strictEnv {
		return expandEnvStrict(payload)
	}
	if settings.expandEnv {
		return os.ExpandEnv(payload), nil
	}
	return payload, nil
}

func expandEnvStrict(s string) (string, error) {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); {
		if s[i] != '$' {
			b.WriteByte(s[i])
			i++
			continue
		}
		if i+1 < len(s) && s[i+1] == '$' {
			b.WriteByte('$')
			i += 2
			continue
		}
		if i+1 < len(s) && s[i+1] == '{' {
			end := strings.IndexByte(s[i+2:], '}')
			if end < 0 {
				return "", fmt.Errorf("unclosed ${} placeholder")
			}
			name := s[i+2 : i+2+end]
			if err := validateEnvName(name); err != nil {
				return "", err
			}
			val, ok := os.LookupEnv(name)
			if !ok {
				return "", fmt.Errorf("environment variable %s is not set", name)
			}
			b.WriteString(val)
			i += 3 + end
			continue
		}
		b.WriteByte('$')
		i++
	}
	return b.String(), nil
}

func validateEnvName(name string) error {
	if name == "" {
		return fmt.Errorf("empty variable name")
	}
	for i, r := range name {
		if i == 0 && r >= '0' && r <= '9' {
			return fmt.Errorf("invalid environment variable name")
		}
		if r == '_' || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			continue
		}
		return fmt.Errorf("invalid environment variable name")
	}
	return nil
}
