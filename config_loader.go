package initModules

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/magiconair/properties"
	"gopkg.in/yaml.v3"
)

// loadSettings holds resolved options for a property load operation.
type loadSettings struct {
	filePath  string
	format    PropType
	expandEnv bool
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

// ConfigLoader loads configuration into registered property structs without using package-level state.
type ConfigLoader struct {
	filePath  string
	format    PropType
	expandEnv bool
	props     []interface{}
}

// NewConfigLoader creates a loader. Defaults: YML, ExpandEnv true, path resources/properties.yml.
func NewConfigLoader(opts ...Option) *ConfigLoader {
	settings := defaultLoadSettings()
	for _, opt := range opts {
		opt(&settings)
	}
	return &ConfigLoader{
		filePath:  settings.filePath,
		format:    settings.format,
		expandEnv: settings.expandEnv,
		props:     make([]interface{}, 0),
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
	return loadPropsFromFile(loadSettings{
		filePath:  c.filePath,
		format:    c.format,
		expandEnv: c.expandEnv,
	}, c.props)
}

func defaultLoadSettings() loadSettings {
	return loadSettings{
		filePath:  propPath,
		format:    propType,
		expandEnv: true,
	}
}

func resolveLoadSettings(opts []Option) loadSettings {
	settings := defaultLoadSettings()
	for _, opt := range opts {
		opt(&settings)
	}
	return settings
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
	settings := resolveLoadSettings(opts)
	if len(props) == 0 {
		return fmt.Errorf("LoadProperties: no properties registered; use AddPropE first")
	}
	return loadPropsFromFile(settings, props)
}

// RunLoadPropertiesE is an alias for LoadProperties with global registry and settings.
func RunLoadPropertiesE() error {
	return LoadProperties()
}

func loadPropsFromFile(settings loadSettings, targets []interface{}) error {
	if settings.filePath == "" {
		return fmt.Errorf("load properties: file path is empty")
	}

	filename, err := filepath.Abs(settings.filePath)
	if err != nil {
		return fmt.Errorf("load properties: absolute path: %w", err)
	}

	var yamlPayload string
	var propFile *properties.Properties

	switch settings.format {
	case YML:
		dataFile, err := os.ReadFile(filename)
		if err != nil {
			return fmt.Errorf("load properties: read file %s: %w", filename, err)
		}
		yamlPayload = string(dataFile)
		if settings.expandEnv {
			yamlPayload = os.ExpandEnv(yamlPayload)
		}
	case PROPERTIES:
		propFile, err = properties.LoadFile(filename, properties.UTF8)
		if err != nil {
			return fmt.Errorf("load properties: read file %s: %w", filename, err)
		}
	default:
		return fmt.Errorf("load properties: unsupported format %v", settings.format)
	}

	for _, target := range targets {
		var decodeErr error
		switch settings.format {
		case YML:
			decodeErr = yaml.Unmarshal([]byte(yamlPayload), target)
		case PROPERTIES:
			decodeErr = propFile.Decode(target)
		}
		if err := processLoadedProp(target, decodeErr); err != nil {
			return fmt.Errorf("load properties: decode %s: %w", filename, err)
		}
	}

	return nil
}
