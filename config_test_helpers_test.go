package initModules

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

var errInvalidPort = errors.New("invalid port")

type validatorConfig struct {
	Port int `yaml:"port"`
}

func (c *validatorConfig) Validate() error {
	if c.Port <= 0 {
		return errInvalidPort
	}
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

func writeTempYAML(t *testing.T, contents string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
