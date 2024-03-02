package initModules

import (
	"github.com/magiconair/properties"
	"gopkg.in/yaml.v3"
	"log"
	"os"
	"path/filepath"
	"reflect"
)

type PropType int

type Prop interface {
	Validate()
}

const (
	YML PropType = iota
	PROPERTIES
)

var propPath = "resources/properties.yml"
var propType = YML
var props = make([]interface{}, 0)

func SetFilePath(pt PropType, p string) {
	propType = pt
	propPath = p
}

func AddProp(p interface{}) {
	if value := reflect.ValueOf(p); value.Kind() != reflect.Ptr || value.Elem().Kind() != reflect.Struct {
		log.Fatal("The parameter must be a pointer or struct")
	}
	
	props = append(props, p)
}

func RunLoadProperties() {
	log.Println("Started load properties from file: ", propPath)
	
	filename, err := filepath.Abs(propPath)
	if err != nil {
		log.Fatal("Error get absolute path: ", err)
	}
	
	var dataEnv string
	var propEnv *properties.Properties
	
	if propType == YML {
		dataFile, err := os.ReadFile(filename)
		if err != nil {
			log.Fatal("Error read file: ", err)
		}
		dataEnv = os.ExpandEnv(string(dataFile))
	} else if propType == PROPERTIES {
		propEnv = properties.MustLoadFile(filename, properties.UTF8)
	}
	
	for _, p := range props {
		if propType == YML {
			err = yaml.Unmarshal([]byte(dataEnv), p)
		} else if propType == PROPERTIES {
			err = propEnv.Decode(p)
		}
		//reflect.TypeOf((*MyInterface)(nil)) returns a pointer to the interface, and Elem() is used to get the type of the interface.
		if value := reflect.TypeOf(p); value.Implements(reflect.TypeOf((*Prop)(nil)).Elem()) {
			p.(Prop).Validate()
		}
		if err != nil {
			log.Fatal("Error unmarshal properties: ", err)
		}
	}
	
	log.Println("Finished load properties from file: ", propPath)
}
