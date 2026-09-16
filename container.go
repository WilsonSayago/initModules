package initModules

import "sync"

// Container holds type-keyed singletons isolated from the global Once registry.
// Embed or wrap it from internal/bootstrap in each microservice.
type Container struct {
	registry sync.Map
}

// NewContainer creates an empty dependency container.
func NewContainer() *Container {
	return &Container{}
}

// OnceIn returns a singleton *T scoped to the container (constructor returns *T).
// c must be created with NewContainer; a nil container panics instead of using
// the global Once registry.
func OnceIn[T any](c *Container, constructor func() *T) *T {
	if c == nil {
		panic("initModules: nil Container; use NewContainer or Once")
	}
	return onceWithRegistry(&c.registry, constructor)
}

// OnceValueIn is the container-scoped variant of OnceValue.
func OnceValueIn[T any](c *Container, constructor func() T) *T {
	if c == nil {
		panic("initModules: nil Container; use NewContainer or Once")
	}
	if constructor == nil {
		panic("initModules: nil constructor; pass a function that returns T")
	}
	return OnceIn(c, func() *T {
		v := constructor()
		return &v
	})
}
