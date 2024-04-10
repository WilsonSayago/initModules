package initModules

import "sync"

type BaseInstance[T any] struct {
	instance *T
	once     sync.Once
}

func (b *BaseInstance[T]) GetInstance(constructor func() T) *T {
	b.once.Do(func() {
		instance := constructor()
		b.instance = &instance
	})
	return b.instance
}

func NewInstance[T any]() *BaseInstance[T] {
	return &BaseInstance[T]{}
}
