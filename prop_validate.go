package initModules

import (
	"fmt"
	"reflect"
)

func validatePropTarget(p interface{}) error {
	if p == nil {
		return fmt.Errorf("AddProp: parameter must be a pointer to struct, got nil")
	}
	value := reflect.ValueOf(p)
	if value.Kind() != reflect.Pointer || value.IsNil() || value.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("AddProp: parameter must be a pointer to struct, got %T", p)
	}
	return nil
}

// processLoadedProp validates p only after a successful decode/unmarshal.
func processLoadedProp(p interface{}, decodeErr error) error {
	if decodeErr != nil {
		return decodeErr
	}
	return validateLoadedProp(p)
}

func validateLoadedProp(p interface{}) error {
	if v, ok := p.(PropValidator); ok {
		if err := v.Validate(); err != nil {
			return fmt.Errorf("validate %T: %w", p, err)
		}
		return nil
	}
	if v, ok := p.(Prop); ok {
		v.Validate()
	}
	return nil
}
