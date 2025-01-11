# init-modules

# Go Init Component Library

The Init Component library provides a comprehensive solution for configuration initialization and process management in
Go applications. It facilitates dynamic loading of properties from files and allows the registration and concurrent
execution of processes, all while handling operating system signals for controlled termination.

## Features

## Features

- Dynamic loading of configuration properties from YAML or Properties files.
- Registration and concurrent execution of user-defined processes.
- Handling of operating system signals for application termination.
- Validation of configuration properties through the implementation of a specific interface.
- Singleton pattern implementation for creating and retrieving instances using a unique key with the `GetInstance`
  function.

## Installation

To use this library, ensure that Go is installed on your system. Then, incorporate the library files into your project
as needed, respecting Go's package structure.

## Usage

### Initialization and Execution

To initialize and execute the loading of properties and processes:

1. **Initialize the loading of properties and processes**:

    ```go
    package main
    
    import "your/project/initModules"
    
    func main() {
        initModules.Run(true, true) // Enable both property loading and process loading
    }
    ```

### Working with Properties

1. **Define and validate your configuration properties:**

    ```go
    type AppConfig struct {
      Port int `yaml:"port"`
    }
    
    func NewAppConfig() AppConfig {
      return AppConfig{}
    }
    
    func (c *AppConfig) Validate() {
        if c.Port <= 0 {
          log.Fatal("Port must be greater than 0")
        }
    }
    ```

2. **Load your configuration properties:**

Set the type and path of the properties file and add your configuration structure to load and validate it:

```go
initModules.SetFilePath(initComponent.YML, "path/to/your/config.yml")
initModules.AddProp(initModules.NewInstance[prop.AppConfig]().GetInstance(prop.NewAppConfig))
initModules.RunLoadProperties()
```

### Registering and Executing Processes

1. Create a struct for your processes:

    ```go
    package instance
    
    import "your/project/initModules"
    
    type TestInstance struct {}
    
    func GetTestInstance() *TestInstance {
      instance := initModules.GetInstance("TestInstance", func () interface{} {
        return &TestInstance{}
      })
      return instance.(*TestInstance)
    }
    ```

1. Register your processes:
    
```go
initModules.RegisterProcess(instance.GetTestInstance())
```

The RunProcesses function is called automatically if enableLoadProcesses is set to true during the call to Run.

### Contributions
Contributions are welcome.