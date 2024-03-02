# init-modules

# Go Init Component Library

La biblioteca Init Component proporciona una solución integral para la inicialización de configuraciones y la gestión de procesos en aplicaciones Go. Facilita la carga dinámica de propiedades desde archivos y permite el registro y ejecución concurrente de procesos, todo mientras maneja señales del sistema operativo para una terminación controlada.

## Características

- Carga dinámica de propiedades de configuración desde archivos YAML o Properties.
- Registro y ejecución concurrente de procesos definidos por el usuario.
- Manejo de señales del sistema operativo para terminación de la aplicación.
- Validación de propiedades de configuración mediante la implementación de una interfaz específica.

## Instalación

Para utilizar esta biblioteca, asegúrate de que Go esté instalado en tu sistema. Luego, incorpora los archivos de la biblioteca en tu proyecto según sea necesario, respetando la estructura de paquetes de Go.

## Uso

### Inicialización y Ejecución

Para inicializar y ejecutar la carga de propiedades y procesos:

1. **Inicializa la carga de propiedades y procesos**:

```go
package main

import "tu/proyecto/initComponent"

func main() {
    initComponent.Run(true, true) // Habilita tanto la carga de propiedades como la de procesos
}
```

2. **Manejo de señales del sistema operativo**:

La función `Run` inicia la carga de propiedades y procesos, y luego espera señales del sistema operativo (como `SIGINT` o `SIGTERM`) para terminar la aplicación de manera controlada.

### Trabajando con Propiedades

1. **Define y valida tus propiedades de configuración**:

```go
type AppConfig struct {
    Port int `yaml:"port"`
}

func (c *AppConfig) Validate() {
    if c.Port <= 0 {
        log.Fatal("Port must be greater than 0")
    }
}
```

2. **Carga tus propiedades de configuración**:

Configura el tipo y la ruta del archivo de propiedades y añade tu estructura de configuración para cargarla y validarla:

```go
initComponent.SetFilePath(initComponent.YML, "path/to/your/config.yml")
var myConfig AppConfig
initComponent.AddProp(&myConfig)
```

### Registro y Ejecución de Procesos

1. **Define tus procesos**:

```go
type MyProcess struct {}

func (p *MyProcess) Start() {
    // Lógica de inicio del proceso
}
```

2. **Registra y ejecuta tus procesos**:

```go
initComponent.RegisterProcess(&MyProcess{})
```

La función `RunProcesses` se llama automáticamente si `enableLoadProcesses` se establece en `true` durante la llamada a `Run`.

## Ejemplo Completo

Un ejemplo completo demostrando la inicialización, carga de propiedades, registro y ejecución de procesos, y el manejo de señales del sistema operativo está incluido en los archivos de código fuente.

## Contribuciones

Las contribuciones son bienvenidas. Por favor, siente libre de fork el repositorio, realizar cambios y enviar un pull request para su revisión.
