package initModules

import (
	"fmt"
	"reflect"
)

func validatePropTarget(p interface{}) error {
	value := reflect.ValueOf(p)
	if value.Kind() != reflect.Ptr || value.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("AddProp: parameter must be a pointer to struct, got %T", p)
	}
	return nil
}

// processLoadedProp validates p only after a successful decode/unmarshal.
func processLoadedProp(p interface{}, decodeErr error) error {
	if decodeErr != nil {
		return decodeErr
	}
	if value := reflect.TypeOf(p); value.Implements(reflect.TypeOf((*Prop)(nil)).Elem()) {
		p.(Prop).Validate()
	}
	return nil
}
