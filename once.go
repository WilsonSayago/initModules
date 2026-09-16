package initModules

import (
	"reflect"
	"sync"
)

type singletonData struct {
	once     sync.Once
	instance interface{}
}

var globalSingletons sync.Map // reflect.Type -> *singletonData

// Once returns a singleton *T where T is a struct type. The constructor must return *T.
func Once[T any](constructor func() *T) *T {
	return onceWithRegistry(&globalSingletons, constructor)
}

// OnceValue returns a singleton *T where the constructor provides a struct value (copied once).
func OnceValue[T any](constructor func() T) *T {
	if constructor == nil {
		panic("initModules: nil constructor; pass a function that returns T")
	}
	return Once(func() *T {
		v := constructor()
		return &v
	})
}

func onceWithRegistry[T any](registry *sync.Map, constructor func() *T) *T {
	if constructor == nil {
		panic("initModules: nil constructor; pass a function that returns *T")
	}
	tType := reflect.TypeOf((*T)(nil)).Elem()

	val, _ := registry.LoadOrStore(tType, &singletonData{})
	data := val.(*singletonData)

	data.once.Do(func() {
		inst := constructor()
		if inst == nil {
			panic("initModules: constructor returned nil; Once requires a non-nil *T")
		}
		data.instance = inst
	})

	return data.instance.(*T)
}

func isNilValue(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Ptr, reflect.Interface, reflect.Slice, reflect.Map, reflect.Chan, reflect.Func:
		return rv.IsNil()
	default:
		return false
	}
}

func typeName(v any) string {
	if isNilValue(v) {
		return "nil"
	}
	t := reflect.TypeOf(v)
	if t == nil {
		return "nil"
	}
	if t.Kind() == reflect.Ptr {
		return t.Elem().Name()
	}
	if name := t.Name(); name != "" {
		return name
	}
	return t.String()
}
