// Minimal example: load YAML config and validate (no database or HTTP).
//
// Run from this directory:
//
//	go run .
package main

import (
	"fmt"
	"log"

	"github.com/WilsonSayago/initModules"
)

type AppConfig struct {
	Port int `yaml:"port"`
}

func NewAppConfig() AppConfig {
	return AppConfig{}
}

func (c *AppConfig) Validate() error {
	if c.Port <= 0 {
		return fmt.Errorf("port must be greater than 0")
	}
	return nil
}

func main() {
	cfg := initModules.OnceValue(NewAppConfig)

	if err := initModules.AddPropE(cfg); err != nil {
		log.Fatal(err)
	}
	if err := initModules.LoadProperties(
		initModules.WithFilePath("config.yml"),
		initModules.WithFormat(initModules.YML),
		initModules.WithStrictYAML(true),
		initModules.WithStrictEnv(true),
	); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Loaded config: port=%d\n", cfg.Port)
}
