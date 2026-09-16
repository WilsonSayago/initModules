package initModules

import (
	"log"
)

type PropType int

// Prop is a configuration target that validates itself after a successful decode.
//
// Deprecated: implement PropValidator so validation can return an error.
type Prop interface {
	Validate()
}

// PropValidator is a configuration target that validates itself after a successful decode.
// If a target implements PropValidator, that method is used and Prop.Validate is not called.
type PropValidator interface {
	Validate() error
}

const (
	YML PropType = iota
	PROPERTIES
)

var propPath = "resources/properties.yml"
var propType = YML
var props = make([]interface{}, 0)

// SetFilePath sets the global configuration file path and format.
//
// Deprecated: prefer LoadProperties(initModules.WithFilePath(...), initModules.WithFormat(...)).
func SetFilePath(pt PropType, p string) {
	propType = pt
	propPath = p
}

// AddProp registers a property on the global loader.
//
// Deprecated: use AddPropE; handle the returned error in main instead of log.Fatal inside the library.
func AddProp(p interface{}) {
	if err := AddPropE(p); err != nil {
		log.Fatal(err)
	}
}

// RunLoadProperties loads globally registered properties using global path and format settings.
//
// Deprecated: use LoadProperties; handle errors in application main.
func RunLoadProperties() {
	log.Println("Started load properties from file: ", propPath)
	if err := LoadProperties(); err != nil {
		log.Fatal("Error load properties: ", err)
	}
	log.Println("Finished load properties from file: ", propPath)
}
