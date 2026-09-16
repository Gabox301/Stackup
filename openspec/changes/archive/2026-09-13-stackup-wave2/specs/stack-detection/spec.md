# Delta for stack-detection

## ADDED Requirements

### Requirement: Wave-2 ecosystem detectors

The system MUST run `csharp`, `java`, `ruby`, `erlang` after `rust` in registry order. Each MUST grade manifest-alone Medium, manifest plus lock-or-pin High, hint-alone Low.

| Ecosystem | Medium (manifest) | High (+lock/pin) | Low (hint alone) |
|---|---|---|---|
| C# | `*.csproj` root glob, `*.sln`/`*.slnx` | `packages.lock.json`, `global.json` pin | `.config/dotnet-tools.json`, `NuGet.config`, `Directory.Build.props` |
| Java | `pom.xml` or `build.gradle(.kts)` + `settings.gradle(.kts)`; gradle wins, both in `Signals` | `gradle.lockfile`, `verification-metadata.xml`, `gradlew`/`mvnw`, `.mvn/`, `.java-version`/`.sdkmanrc`/`.tool-versions:java` | version-hint files alone |
| Ruby | `Gemfile`, `*.gemspec` | `Gemfile.lock`, `.ruby-version`/`.rbenv-version`/`.tool-versions:ruby` pin; `PackageManager=bundler` | version files alone |
| Erlang | `rebar.config` | `rebar.lock`, `.tool-versions:erlang` pin; `PackageManager=rebar3` | `.tool-versions` alone |

The system MUST NOT fire on `*.app.src`/`*.erl` alone and MUST ignore `mix.exs`.

#### Scenario: C# glob upgrades with pin

- GIVEN root `app.csproj` alone
- WHEN detection runs
- THEN C# evidence grades Medium; with `global.json` it grades High

#### Scenario: Java duel resolves gradle-wins

- GIVEN `pom.xml` plus `build.gradle`/`settings.gradle`
- WHEN detection runs
- THEN single Java evidence reports `PackageManager=gradle` with both manifests in `Signals`

#### Scenario: Erlang weak signals stay silent

- GIVEN only `foo.app.src`, `bar.erl`, `mix.exs`
- WHEN detection runs
- THEN no Erlang evidence returns

### Requirement: Optional Runtime/Frameworks evidence fields

The system MUST add optional `Runtime` and `Frameworks` to Evidence; both MUST be additive (zero-value safe) and MUST NOT affect ranking.

#### Scenario: Additive fields preserve ranking

- GIVEN an old consumer of Evidence JSON
- WHEN `Runtime`/`Frameworks` are absent or empty
- THEN decode succeeds and sort order is unchanged

### Requirement: Node Bun runtime and framework signals

The system MUST set `Runtime=bun` on Bun repos and MUST resolve `PackageManager` from `packageManager` first, else `bun`. Frameworks MUST be presence-only from root `package.json` `dependencies` + `devDependencies` (`react`, `vue`, `svelte`, `astro`, `next`, `nuxt`, `sveltekit`); meta-frameworks record both (e.g. `next`→`react`). The system MUST keep a single Node evidence.

#### Scenario: Bun coexistence keeps single evidence

- GIVEN `package.json` with `package-lock.json` and `bun.lock`
- WHEN detection runs
- THEN one Node evidence returns with `Runtime=bun` (`packageManager`-first)

#### Scenario: Framework presence without semver

- GIVEN root `package.json` with `next@^14`
- WHEN detection runs
- THEN `Frameworks` contains `react` and `next` regardless of range
