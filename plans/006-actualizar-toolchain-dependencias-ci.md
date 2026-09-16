# Plan 006: Actualizar Go, dependencias y la cadena de suministro de CI

> **Instrucciones para el ejecutor**: Las versiones son temporales. Verifica cada
> una contra fuentes oficiales el día de ejecución y no adoptes prereleases salvo
> instrucción explícita. No hagas upgrades masivos sin revisar diff/changelog.
>
> **Drift check (primero)**:
> `git diff --stat df5abb2..HEAD -- go.mod go.sum examples/standalone/go.mod examples/standalone/go.sum .github/workflows/ci.yml .golangci.yml Makefile README.md && git status --short -- go.mod go.sum examples/standalone/go.mod examples/standalone/go.sum .github/workflows/ci.yml .golangci.yml Makefile README.md`
> Confirma que la base usa Go 1.26.3, properties 1.8.9, yaml.v3 3.0.1,
> golangci-lint 1.62.2 y actions v4/v5/v6.

## Estado

- **Prioridad**: P1
- **Esfuerzo**: L
- **Riesgo**: HIGH
- **Depende de**: planes 001–005
- **Categoría**: security / migration / tech-debt
- **Planificado en**: commit `df5abb2`, 2026-09-15, más cambios locales del propietario

## Por qué importa

El módulo declara un patch antiguo de Go como mínimo, usa una dependencia con
versiones posteriores y CI/linter en majors antiguos. La matriz no prueba la
versión mínima real y las actions por tags mutables amplían riesgo de supply
chain. El upgrade debe separar política de compatibilidad, seguridad y adopción
de prereleases.

## Estado actual y datos verificados al planificar

- `go.mod`: `go 1.26.3`, properties `v1.8.9`, yaml.v3 `v3.0.1`.
- Último Go estable verificado el 2026-09-15: `1.26.8` (publicado 2026-09-01).
- Properties ofrece `v1.8.10` y `v1.18.11`; `v1.18.11` declara Go 1.19. El salto
  inusual exige revisar tags/diff y procedencia antes de adoptar.
- YAML v3.0.1 es la última estable v3. YAML v4 disponible es `v4.0.0-rc.6`:
  mantener v3 en este plan.
- golangci-lint estable verificado: `v2.13.2`; requiere configuración v2.
- Majors verificados de actions: checkout v7, setup-go v7,
  upload-artifact v7; golangci-lint-action v9. Resolver release y SHA completos
  oficiales al ejecutar.
- La documentación de consumidores indica Go 1.24 como base; el código debe
  probarlo antes de bajar el directive.
- Herramientas locales `golangci-lint` y `govulncheck` no estaban instaladas.

Fuentes permitidas: `go.dev`, repositorios/releases oficiales de GitHub para cada
action/dependencia y documentación oficial de golangci-lint. No usar blogs para
elegir versiones o SHAs.

## Comandos necesarios

| Propósito | Comando | Resultado esperado |
|---|---|---|
| Versiones módulo | `GOWORK=off go list -m -versions github.com/magiconair/properties go.yaml.in/yaml/v4` | lista oficial, exit 0 |
| Tidy | `GOWORK=off go mod tidy` | exit 0 y diff explicado |
| Suite | `go test -count=20 ./...` | exit 0 |
| Race/coverage | `go test -race -coverprofile=coverage.out -covermode=atomic ./...` | exit 0 |
| Vulns | `govulncheck ./...` (versión pinneada/registrada) | sin vulnerabilidades alcanzables |
| Lint | `golangci-lint run ./...` (v2 pinneado) | exit 0 |
| Ejemplo | `(cd examples/standalone && GOWORK=off go mod tidy && GOWORK=off go test ./...)` | exit 0 |

## Alcance

**Dentro del alcance**:
- `go.mod`, `go.sum`
- `examples/standalone/go.mod`, `examples/standalone/go.sum`
- `.github/workflows/ci.yml`, `.golangci.yml`, `.github/dependabot.yml` (nuevo)
- `Makefile`, `README.md`, `CHANGELOG.md`
- cambios mecánicos exigidos por linter sólo en `*.go`, sin refactor funcional
- `plans/README.md`

**Fuera del alcance**:
- Migrar a YAML v4 RC.
- Cambiar API o comportamiento para satisfacer un linter.
- Publicar tags/releases o hacer push.
- Usar `@main`, `@master`, `latest` o SHA sin verificar.
- Exigir Go 1.26 si Go 1.24 pasa y es la política de consumidores.

## Flujo Git

- Rama sugerida: `codex/006-toolchain-deps-ci`.
- Commits: `update Go and module dependencies`; `modernize lint and CI supply chain`.
- No push/PR sin orden explícita.

## Pasos

### 1. Confirmar política de Go mínimo y toolchain

Probar la suite completa con el último patch oficial de Go 1.24 y con Go 1.26.8
en entornos limpios. Si ambos pasan, usar `go 1.24.0` como mínimo de lenguaje y
`toolchain go1.26.8` como toolchain recomendado; alinear ejemplo. Si Go 1.24 no
pasa, detenerse y reportar la feature/dependencia exacta antes de elevar mínimo.

**Verificar**: misma suite, vet y ejemplo pasan en ambos extremos de la matriz.

### 2. Revisar y actualizar `properties`

Verificar que `v1.18.11` proviene del repo/módulo oficial, leer changelog y
comparar API/comportamiento desde `v1.8.9`. Si es legítima y estable, actualizar,
hacer tidy y ejecutar especialmente pruebas `.properties`. Si el salto no puede
validarse o cambia semántica, usar sólo `v1.8.10` y abrir/registrar follow-up con
la evidencia; no forzar “latest”.

**Verificar**: `go list -m all`, checksum en `go.sum`, tests de PROPERTIES y suite
completa pasan; no aparecen dependencias inesperadas sin explicación.

### 3. Mantener YAML estable y ejecutar escaneo

Conservar `gopkg.in/yaml.v3 v3.0.1`. Instalar/usar una versión explícita y actual
de `govulncheck`; guardar la versión en docs/CI. Corregir sólo vulnerabilidades
alcanzables dentro del alcance; si una exige migrar a prerelease, detenerse.

**Verificar**: `govulncheck ./...` no reporta vulnerabilidades alcanzables.

### 4. Migrar golangci-lint a v2

Convertir `.golangci.yml` a schema v2 (`version: "2"`) usando la guía oficial.
Fijar una versión exacta de golangci-lint (2.13.2 si sigue siendo estable actual).
Revisar cada cambio mecánico producido por nuevas reglas; no silenciar errores
reales ni usar exclusiones globales nuevas sin justificación.

**Verificar**: `golangci-lint config verify` y `golangci-lint run ./...` → exit 0.

### 5. Actualizar y fijar actions por SHA

Actualizar a releases estables actuales de checkout/setup-go/upload-artifact y
golangci-lint-action. En `uses`, pinnear el SHA completo del tag verificado y
añadir comentario con la versión legible. Otorgar sólo `contents: read`; conservar
cache y artefacto de cobertura. Separar jobs/matriz: tests en mínimo y último Go;
lint/vuln en último; ejemplo incluido. No descargar ejecutables desde URLs no
oficiales.

**Verificar**: cada SHA resuelve al tag oficial; validar YAML; ejecutar localmente
los comandos de cada job. La ejecución real de GitHub Actions debe quedar verde
antes de merge.

### 6. Añadir actualización automatizada acotada

Crear `.github/dependabot.yml` para gomod y github-actions con frecuencia
razonable, límites de PR y agrupación conservadora. Prereleases quedan excluidas.
Documentar versión mínima/recomendada real en README y Changelog `Unreleased`.

**Verificar**: YAML parsea; `rg -n 'go 1\.24|1\.26\.8|toolchain|govulncheck' go.mod README.md .github/workflows/ci.yml Makefile` → política consistente.

## Plan de pruebas

- Matriz Go mínimo/último: unit repetida, race/coverage, vet.
- Config YAML y `.properties`, incluidos tests estrictos del plan 003.
- Módulo de ejemplo independiente.
- `govulncheck`, config verify y golangci-lint v2.
- Verificación manual de SHAs contra tags oficiales.

## Criterios de término

- [ ] `go` expresa mínimo validado y `toolchain` el patch recomendado actual.
- [ ] properties está en la última estable verificable o hay evidencia para el
  fallback 1.8.10.
- [ ] YAML permanece en última estable, no RC.
- [ ] CI prueba mínimo/último, race/coverage, lint, vuln y ejemplo.
- [ ] actions están pinneadas por SHA completo con versión comentada.
- [ ] todos los gates locales y CI pasan.

## Condiciones de parada

- La base con planes 001–005 no está verde antes de upgrades.
- Go 1.24 falla; no elevar mínimo sin reportar evidencia al propietario.
- No se puede verificar origen/tag/SHA de una dependencia o action.
- Latest estable implica una regresión de comportamiento o API.
- Sólo una prerelease corrige el problema.
- Un gate falla dos veces tras una corrección razonable.

## Notas de mantenimiento

Revisar mensualmente patch de Go y alertas Dependabot, pero elevar el directive
`go` sólo cuando cambie el mínimo soportado. Renovar SHAs mediante PR revisable,
nunca tags flotantes.

## Corrección requerida tras revisión

La revisión de `1afad81` detectó que la matriz no prueba realmente el mínimo:
la selección predeterminada `GOTOOLCHAIN=auto` lee `toolchain go1.27.1` y cambia
el proceso iniciado con Go 1.24.13 a Go 1.27.1. En el job `test`, establecer
`GOTOOLCHAIN: local` para los comandos de formato, vet y pruebas (a nivel de job
o de steps), de modo que cada fila use exactamente el binario instalado por
`actions/setup-go`. Conservar el `toolchain` directive para consumidores que
quieran la versión recomendada. Antes de marcar el plan como DONE, comprobar en
la ejecución remota que `go version` reporta 1.24.13 y 1.27.1 respectivamente;
añadir un step explícito de diagnóstico si el log no lo deja claro.
