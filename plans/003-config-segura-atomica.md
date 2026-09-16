# Plan 003: Hacer la configuración validable, estricta y atómica

> **Instrucciones para el ejecutor**: Este es un cambio delicado de API y datos.
> Agrega primero caminos compatibles, luego cambia internals y finalmente docs.
> Ejecuta cada gate. No conviertas warnings en aceptación silenciosa.
>
> **Drift check (primero)**:
> `git diff --stat df5abb2..HEAD -- config_loader.go prop.go prop_validate.go prop_test.go testdata README.md docs/MIGRATION.md examples/standalone && git status --short -- config_loader.go prop.go prop_validate.go prop_test.go testdata README.md docs/MIGRATION.md examples/standalone`
> Deben existir `ConfigLoader`, `Option`, `Prop.Validate()` y el ejemplo local.

## Estado

- **Prioridad**: P1
- **Esfuerzo**: L
- **Riesgo**: HIGH
- **Depende de**: `plans/001-estabilizar-verificacion.md`
- **Categoría**: security / bug / migration
- **Planificado en**: commit `df5abb2`, 2026-09-15, más cambios locales del propietario

## Por qué importa

La validación actual no puede devolver error, YAML ignora claves desconocidas y
`os.ExpandEnv` sustituye variables ausentes por cadena vacía, incluso dentro de
valores con `$` legítimos. Además, al cargar varios targets, los primeros quedan
mutados si uno posterior falla. En bootstrap esto produce configuraciones
parciales o inseguras difíciles de diagnosticar.

## Estado actual

- `prop.go:9-11`: `type Prop interface { Validate() }`.
- `prop_validate.go:17-24`: después de decodificar, detecta `Prop` por reflexión
  y llama `Validate()` sin posibilidad de error.
- `config_loader.go:12-20`: settings sólo contienen path, format y `expandEnv`;
  las opciones no devuelven error.
- `config_loader.go:83-88`: expansión YAML activada por defecto.
- `config_loader.go:143-145`: usa `os.ExpandEnv(yamlPayload)`; missing env queda
  vacío y `$NAME` también se expande.
- `config_loader.go:155-166`: decodifica directamente cada target y retorna al
  primer error, dejando anteriores mutados.
- `yaml.Unmarshal` no activa `Decoder.KnownFields(true)`.
- `prop_test.go` usa panic para una validación inválida y no cubre rollback,
  claves desconocidas, variable ausente ni dólar literal.

Restricciones de diseño: mantener compatibilidad v1 con `Prop.Validate()` y las
opciones actuales. `gopkg.in/yaml.v3` sigue siendo la dependencia estable; no
migrar a v4 release candidate.

## Comandos necesarios

| Propósito | Comando | Resultado esperado |
|---|---|---|
| Tests config | `go test -count=20 -run 'Test.*Prop|Test.*Config|TestLoadProperties' ./...` | todos pasan |
| Suite/race | `go test -count=20 ./... && go test -race ./...` | exit 0 en toolchain sano |
| Ejemplo | `(cd examples/standalone && GOWORK=off go test ./... && GOWORK=off go run .)` | test OK y config cargada |
| Vet/formato | `gofmt -w config_loader.go prop.go prop_validate.go prop_test.go examples/standalone/*.go && go vet ./...` | exit 0 |

## Alcance

**Dentro del alcance**:
- `config_loader.go`, `prop.go`, `prop_validate.go`, `prop_test.go`
- fixtures nuevos bajo `testdata/`
- `examples/standalone/*.go`
- `README.md`, `docs/MIGRATION.md`, `CHANGELOG.md`
- `plans/README.md`

**Fuera del alcance**:
- Cambiar la sintaxis o semántica de `.properties` salvo atomicidad/validación.
- Eliminar `Prop`, `AddProp`, `SetFilePath` o APIs globales.
- Leer configuración remota, secrets managers o flags.
- Migrar a YAML v4 RC.
- Elegir defaults breaking sin una opción explícita de compatibilidad.

## Flujo Git

- Rama sugerida: `codex/003-config-segura-atomica`.
- Commits lógicos: `add error-returning config validation`; `make config loading strict and atomic`; `document safe config options`.
- No push/PR sin orden explícita.

## Pasos

### 1. Agregar validación con error sin romper `Prop`

Introducir una interfaz exportada con nombre claro, por ejemplo:

```go
type PropValidator interface { Validate() error }
```

Conservar `Prop` y marcarlo como deprecado en godoc. Si un target implementa
ambas interfaces, usar primero `PropValidator`; si devuelve error, envolverlo
con tipo/target y propagarlo. Sólo llamar al `Prop` heredado cuando la nueva
interfaz no aplique. No recuperar panics del código del consumidor.

**Verificar**: tests para éxito, error sentinel (`errors.Is`), compatibilidad
legacy y precedencia de la interfaz nueva.

### 2. Decodificar y validar sobre copias temporales

Para cada target puntero-a-struct validado, crear una copia temporal independiente
del valor actual, no un struct inicializado en cero. La copia debe conservar
defaults de constructor en campos ausentes del archivo y no debe compartir mapas,
slices ni punteros mutables que el decoder pueda modificar antes del commit.
Decodificar y validar exclusivamente esas copias. Sólo cuando todos los targets
hayan pasado, asignar cada struct temporal al destino original. La fase de commit
no debe fallar. Esto garantiza rollback de valores; documentar que efectos externos
causados por un `Validate` del consumidor no son reversibles.

**Verificar**: iniciar dos targets con valores conocidos; forzar fallo de decode
y luego de validación en el segundo; ambos deben conservar exactamente sus
valores originales. Añadir un load exitoso con un campo omitido que conserva su
default previo y casos de fallo con mapa, slice y puntero preinicializados que no
mutan a través de aliasing.

### 3. Añadir modo YAML estricto

Agregar una opción explícita `WithStrictYAML(bool)` y su campo en settings/loader.
Cuando esté activa, usar `yaml.NewDecoder` con `KnownFields(true)`. Como el diseño
actual decodifica el mismo documento en varios structs parciales, soportar modo
estricto sólo con un target: con más de uno devolver un error claro antes de
mutar nada, recomendando un root config que agrupe secciones. El default v1 debe
seguir siendo `false`; recomendar `true` en docs y ejemplo.

**Verificar**: clave conocida pasa; typo desconocido falla incluyendo nombre de
campo; múltiples targets + strict falla antes de commit; legacy no estricto
conserva compatibilidad.

### 4. Añadir interpolación de entorno segura

Agregar `WithStrictEnv(bool)` conservando `WithExpandEnv`. En strict env,
expandir sólo `${NAME}` y devolver error si `NAME` no existe (distinguir unset de
valor explícitamente vacío con `os.LookupEnv`). Definir y probar escape para un
dólar literal, por ejemplo `$$` → `$`; documentarlo. No incluir valores de env
en errores/logs. En modo no estricto conservar la semántica legacy para no romper
v1; recomendar strict en el ejemplo.

**Verificar**: variable presente, presente-vacía, ausente, `${NAME}`, `$NAME`,
`$$`, múltiples placeholders y contenido parecido a contraseña. Missing debe
nombrar sólo la variable, nunca el secreto.

### 5. Validar opciones y targets antes de I/O

Rechazar loader sin targets, opción `nil`, formato inválido y target tipado-nil
con errores, sin panic. Ejecutar esas validaciones antes de leer el archivo.
Mantener errores envueltos con operación y filename, pero no datos sensibles.

**Verificar**: tests de cada input inválido; usar un path inexistente para probar
que el error de target/opción tiene precedencia sobre el I/O.

### 6. Migrar ejemplo y documentación

El ejemplo debe implementar `Validate() error` y activar strict YAML/env. README
debe explicar defaults de compatibilidad, opciones recomendadas y atomicidad.
Migration debe mostrar cambio incremental de `Prop` a `PropValidator` sin afirmar
que la interfaz vieja desaparece en v1. Changelog debe dejar estas entradas en
`Unreleased` hasta el plan 007.

**Verificar**: compilar y ejecutar ejemplo; `rg -n 'PropValidator|WithStrictYAML|WithStrictEnv|atomic' README.md docs/MIGRATION.md CHANGELOG.md` → todas las APIs documentadas.

## Plan de pruebas

- Decode YAML y properties exitoso, con validación nueva y legacy.
- Error de validación preservado con `errors.Is`.
- Rollback total con fallo de decode o validación en cualquier target.
- Load exitoso parcial conserva defaults preexistentes de campos ausentes.
- Fallos no mutan mapas, slices ni valores apuntados preexistentes por aliasing.
- YAML strict: typo rechazado, multi-target rechazado de forma explícita.
- Env strict: presente, vacío, ausente, literal `$`, y no filtración de valor.
- Target nil/tipado-nil, cero targets, option nil y formato inválido.
- Ejemplo independiente con opciones recomendadas.

## Criterios de término

- [ ] Los consumidores pueden devolver errores de validación sin romper `Prop`.
- [ ] Ningún target se modifica cuando cualquier fase falla.
- [ ] Un load exitoso conserva los defaults preexistentes de campos no presentes
  en YAML o `.properties`, igual que antes del cambio atómico.
- [ ] Strict YAML y strict env son opt-in en v1 y están documentados.
- [ ] Errores no exponen contenido de configuración ni valores de entorno.
- [ ] suite repetida, race, vet, formato y ejemplo pasan.
- [ ] sólo archivos dentro del alcance cambiaron.

## Condiciones de parada

- Se descubre un consumidor publicado que ya define simultáneamente ambos
  métodos `Validate` de forma incompatible.
- El modo strict requeriría aceptar silenciosamente claves desconocidas con
  múltiples targets; mantener el error explícito y reportar.
- La atomicidad requiere cambiar punteros/identidad externa del target.
- Una opción estricta cambia el default legacy accidentalmente.
- Un gate falla dos veces tras una corrección razonable.

## Notas de mantenimiento

Al revisar, buscar aliasing: sólo el valor del struct se copia; mapas, slices o
punteros anidados decodificados deben ser nuevos para evitar mutación anticipada.
En v2 se podrá hacer strict por defecto y eliminar `Prop` legacy.
