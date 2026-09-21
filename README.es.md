# stackup

> 🌐 [Read in English](README.md)

## Español

`stackup` analiza la raíz de un proyecto, detecta sus ecosistemas con evidencia
ordenada por confianza y genera la configuración del editor para VS Code,
Cursor, Devin Desktop y Kiro. Las escrituras se pueden previsualizar
(`--dry-run`), siempre generan una copia de seguridad antes de cada cambio y
son aptas para CI (salida legible por máquinas, nunca se bloquea en prompts).

### Instalación

Requiere Go 1.27.1+.

```sh
go install ./cmd/stackup
# o compilar localmente
go build -o stackup.exe ./cmd/stackup
```

#### Binarios precompilados

Descargá el archivo correspondiente a tu sistema operativo y arquitectura desde
[GitHub Releases](https://github.com/Gabox301/Stackup/releases):

| SO      | Arquitectura   | Archivo de ejemplo                   |
| ------- | -------------- | ------------------------------------ |
| Windows | x86_64 (amd64) | `stackup_0.1.0_Windows_x86_64.zip`   |
| Linux   | x86_64 (amd64) | `stackup_0.1.0_Linux_x86_64.tar.gz`  |
| Linux   | arm64          | `stackup_0.1.0_Linux_arm64.tar.gz`   |
| macOS   | x86_64 (amd64) | `stackup_0.1.0_Darwin_x86_64.tar.gz` |
| macOS   | arm64          | `stackup_0.1.0_Darwin_arm64.tar.gz`  |

O instalá la última versión con Go (requiere Go 1.27.1+):

```sh
go install github.com/Gabox301/Stackup/cmd/stackup@latest
```

### Inicio rápido

```sh
# ¿Qué stacks hay en este repo?
stackup detect --path .

# Previsualizar lo que se generaría (no escribe nada)
stackup generate --dry-run --path .

# Mostrar el diff unificado contra el disco
stackup diff --path .

# Aplicar (respalda cada archivo sobrescrito como <f>.bak.<timestamp>)
stackup apply --path .
```

### Probarlo localmente en Windows

Desde el directorio de tu proyecto, en **Windows Terminal** (o cualquier
consola compatible con ANSI). Compilá el binario y abrí la TUI interactiva:

```powershell
go build -o stackup.exe ./cmd/stackup
.\stackup.exe detect --path .
```

Las pantallas de la TUI — paneles redondeados, badges de confianza y la fila
seleccionada resaltada — solo se muestran en un TTY real:

- Ejecutá `.\stackup.exe detect .` directo; **no** pipees ni redirijas stdout.
- Usá Windows Terminal en vez de la consola legacy para ver colores y
  caracteres de marco (las consolas legacy degradan a ASCII plano a propósito).
- Agregá `--yes` / `--non-interactive`, o `--format json`, para forzar salida
  plana o de máquina (también ocurre automáticamente si pipeás stdout).
- Teclas: `q` sale de cualquier pantalla, `up/down` / `j/k` mueven la
  selección y en `apply` `y` aplica / `n` aborta sin escribir.

Previsualizá todo antes de escribir cualquier cosa:

```powershell
.\stackup.exe generate --dry-run --path .   # solo preview (exit 2 si hay cambios)
.\stackup.exe diff --path .                 # diff unificado, no escribe nada
.\stackup.exe apply --yes --path .          # escritura segura; respalda primero
```

El proyecto `testdata/node` del propio repo es un objetivo de prueba listo:
`.\stackup.exe detect --path testdata\node`.

### Comandos

| Comando    | Qué hace                                                     |
| ---------- | ------------------------------------------------------------ |
| `detect`   | Muestra la evidencia de stacks ordenada para un proyecto     |
| `generate` | Genera configs del IDE (con `--dry-run` solo previsualiza)   |
| `diff`     | Muestra el diff unificado de los cambios, no escribe nada    |
| `apply`    | Escribe configs con respaldos fechados antes de sobrescribir |

#### ¿Qué comandos escriben al disco?

- Solo lectura (nunca escriben): `detect`, `diff` y cualquier comando con `--dry-run`.
- Escriben: `generate` (sin `--dry-run`) y `apply`.
- `apply` es el comando que buscás: crea los archivos faltantes directamente y,
  antes de sobrescribir algo existente, guarda un respaldo
  `<archivo>.bak.<UTC-timestamp>`. Sobrescribir requiere `--force` (o
  confirmación TTY); en CI usá `--yes` / `--non-interactive` (los bloqueos
  salen con `3` en vez de preguntar).

```sh
stackup apply --path <proyecto>          # interactivo (pregunta antes de sobrescribir)
stackup apply --yes --path <proyecto>    # no interactivo
stackup apply --force --path <proyecto>  # también sobrescribe existentes
```

#### Flags

```
--path <dir>            raíz del proyecto (default: .)
--ide <lista>           vscode,cursor,devin,kiro (default: todos)
--format <text|json>    salida humana o legible por máquinas
--dry-run               solo previsualiza, no escribe nada
--force                 permite sobrescribir archivos (igual respalda)
--yes, --non-interactive  omite todos los prompts (modo CI)
--allow-unknown         continúa aunque no se detecte ningún stack
```

#### Códigos de salida

| Código | Significado                                |
| ------ | ------------------------------------------ |
| 0      | ok / sin cambios / abortado por el usuario |
| 2      | el preview tiene cambios                   |
| 3      | bloqueado en modo no interactivo           |
| 4      | stack desconocido                          |
| 1      | error                                      |

### Stacks detectados

| Ecosistema         | Señales (manifiesto → lock/pin → hint)                              | Extensión recomendada      |
| ------------------ | ------------------------------------------------------------------- | -------------------------- |
| Node.js            | `package.json` → lockfiles / `packageManager` / Bun (`bun.lock`)    | `dbaeumer.vscode-eslint`   |
| Bun                | `bun.lock` / `packageManager: bun` (runtime sobre evidencia Node)   | `oven.bun-vscode`          |
| React / Next.js    | `react`, `next` en dependencies                                     | cubierto por ESLint        |
| Vue / Nuxt         | `vue`, `nuxt`                                                       | `Vue.volar`                |
| Svelte / SvelteKit | `svelte`, `sveltekit` / `@sveltejs/kit`                             | `svelte.svelte-vscode`     |
| Astro              | `astro`                                                             | `astro-build.astro-vscode` |
| Go                 | `go.mod` → `go.sum`                                                 | `golang.go`                |
| Python             | `pyproject.toml` / `requirements*.txt` / `Pipfile` → lockfiles      | `ms-python.python`         |
| Rust               | `Cargo.toml` → `Cargo.lock`                                         | `rust-lang.rust-analyzer`  |
| C# (.NET)          | `*.csproj` / `*.sln` → `global.json` / `packages.lock.json`         | `ms-dotnettools.csharp`    |
| Java               | `pom.xml` / `build.gradle` (Gradle gana el duelo) → wrappers / pins | `redhat.java`              |
| Ruby               | `Gemfile` → `Gemfile.lock` (Bundler)                                | `Shopify.ruby-lsp`         |
| Erlang             | `rebar.config` → `rebar.lock` (rebar3)                              | `pgourlain.erlang`         |

La detección es solo-raíz y políglota: un backend Go + frontend Node
devuelve ambas evidencias ordenadas por confianza, nunca un único ganador.

### Targets por IDE

| IDE           | Archivos escritos                                                               |
| ------------- | ------------------------------------------------------------------------------- |
| VS Code       | `.vscode/{settings,extensions,launch,tasks}.json` (JSONC, preserva comentarios) |
| Cursor        | `.cursor/rules/*.mdc` + `.cursor/mcp.json`                                      |
| Devin Desktop | `.devin/rules/*.md` (+ fallback `.windsurf`) + `.devin/mcp_config.json`         |
| Kiro          | `.kiro/steering/*.md` + `.kiro/specs/<feature>/` + `.kiro/settings/mcp.json`    |

Las recomendaciones de extensiones van al array `recommendations` de VS Code
(merge por unión — tus entradas nunca se borran); los demás IDEs reciben
menciones en prosa. Re-generar es byte-idempotente.

### Modelo de seguridad

- `--dry-run` / `diff` nunca tocan el disco.
- Cada sobrescritura está precedida por un respaldo `<archivo>.bak.<UTC-timestamp>`;
  escribir sin respaldo es imposible por construcción.
- Sobrescribir requiere `--force` (o confirmación TTY explícita en `apply`).
- Las corridas no-TTY omiten prompts y salen con `3` en vez de colgar un pipeline.

### Desarrollo

```sh
go build ./...
go test ./... -count=1
gofmt -l . && go vet ./...
golangci-lint run ./...
```

Estructura: `cmd/stackup/` (árbol Cobra) + `internal/{detect,generate,merge,diff,apply,tui}/`
(pipeline agnóstico de UI; la TUI Bubble Tea vive detrás de una interfaz
`Launcher` reemplazable, con stub para tests).
