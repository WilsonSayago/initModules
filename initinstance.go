package initModules

import (
	"sync"
)

var instances = make(map[string]interface{})
var once sync.Once

func GetInstance(key string, newInstance func() interface{}) interface{} {
	once.Do(func() {
		instances[key] = newInstance()
	})
	return instances[key]
}
