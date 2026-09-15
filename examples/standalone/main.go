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

func (c *AppConfig) Validate() {
	if c.Port <= 0 {
		log.Fatal("port must be greater than 0")
	}
}

func main() {
	cfg := initModules.OnceValue(NewAppConfig)

	if err := initModules.AddPropE(cfg); err != nil {
		log.Fatal(err)
	}
	if err := initModules.LoadProperties(
		initModules.WithFilePath("config.yml"),
		initModules.WithFormat(initModules.YML),
	); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Loaded config: port=%d\n", cfg.Port)
}
