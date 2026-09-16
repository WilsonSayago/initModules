package initModules

// BaseInstance provides a typed singleton by reflect.Type in the global registry.
//
// Deprecated: use Once or OnceValue instead.
type BaseInstance[T any] struct{}

// NewInstance creates a BaseInstance helper.
//
// Deprecated: use Once or OnceValue instead.
func NewInstance[T any]() *BaseInstance[T] {
	return &BaseInstance[T]{}
}

// GetInstance returns the singleton for T, invoking constructor on first use.
//
// Deprecated: use OnceValue(constructor) when the factory returns a struct value.
func (b *BaseInstance[T]) GetInstance(constructor func() T) *T {
	return OnceValue(constructor)
}
