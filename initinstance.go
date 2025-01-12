package initModules

var instances = make(map[string]interface{})

func GetInstance(key string, newInstance func() interface{}) interface{} {
	if instance, exists := instances[key]; exists {
		return instance
	}
	instances[key] = newInstance()
	return instances[key]
}
