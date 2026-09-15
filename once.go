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
	return Once(func() *T {
		v := constructor()
		return &v
	})
}

func onceWithRegistry[T any](registry *sync.Map, constructor func() *T) *T {
	tType := reflect.TypeOf((*T)(nil)).Elem()

	val, _ := registry.LoadOrStore(tType, &singletonData{})
	data := val.(*singletonData)

	data.once.Do(func() {
		data.instance = constructor()
	})

	return data.instance.(*T)
}
