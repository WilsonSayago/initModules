package initModules

import "sync"

type instanceEntry struct {
	once     sync.Once
	instance interface{}
}

var instances sync.Map // map[string]*instanceEntry

// GetInstance returns a singleton for the given key, creating it with newInstance on first call.
// It is safe for concurrent use. If newInstance is nil, the stored value may remain nil.
//
// Deprecated: prefer Once[T] for type-safe singletons or Container.Once in a composition root.
// String keys are error-prone and shared across the whole process.
func GetInstance(key string, newInstance func() interface{}) interface{} {
	val, _ := instances.LoadOrStore(key, &instanceEntry{})
	entry := val.(*instanceEntry)

	entry.once.Do(func() {
		if newInstance != nil {
			entry.instance = newInstance()
		}
	})

	return entry.instance
}
