package initModules

import (
	"reflect"
	"sync"
)

var registry = make(map[reflect.Type]*baseInstanceData)
var registryLock = sync.Mutex{}

type baseInstanceData struct {
	once     sync.Once
	instance interface{}
}

type BaseInstance[T any] struct {
}

func (b *BaseInstance[T]) GetInstance(constructor func() T) *T {
	// Obtiene el tipo de T.
	tType := reflect.TypeOf((*T)(nil)).Elem()

	registryLock.Lock()
	defer registryLock.Unlock()

	// Encuentra o crea el contenedor de datos base para el tipo T.
	data, exists := registry[tType]
	if !exists {
		data = &baseInstanceData{}
		registry[tType] = data
	}

	// Utiliza sync.Once para garantizar que la instancia solo se cree una vez.
	data.once.Do(func() {
		instance := constructor()
		data.instance = &instance
	})

	// Retorna la instancia de T.
	return data.instance.(*T)
}

func NewInstance[T any]() *BaseInstance[T] {
	return &BaseInstance[T]{}
}
