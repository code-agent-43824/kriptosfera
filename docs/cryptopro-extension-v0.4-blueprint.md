# CryptoPro extension — v0.4 technical blueprint

## Goal

Следующий продуктовый шаг после `v0.3`:
- доставить CryptoPro CAdES Browser Plug-in extension внутри payload;
- загружать extension в pinned Chromium runtime без admin rights;
- получить стабильную и проверяемую точку контроля, что extension действительно поднялся.

Это **не** документ про native messaging и **не** про криптобиблиотеки/CSP. Только extension layer.

## Out of scope for v0.4

Не делаем на этом этапе:
- native messaging host;
- регистрацию native host manifest;
- CryptoPro CSP / crypto libraries;
- работу с Рутокеном;
- реальную подпись.

## Success criteria

Этап `v0.4` считаем закрытым, если:
1. extension лежит в payload в фиксированном layout;
2. Chromium запускается с этим extension и не ломает текущий bootstrap/runtime flow;
3. есть служебная страница/диагностика, по которой можно понять:
   - extension загружен;
   - extension id известен;
   - базовый API/признак живости доступен;
4. одинаково работает и в `embedded`, и в `remote` launcher mode.

## Key constraints

- Только user-space delivery, без admin rights.
- Не полагаться на системный Chrome.
- Не смешивать extension layer с native messaging layer.
- Layout extension внутри payload должен быть стабильным и version-controlled.
- По возможности обеспечить стабильный `extension id`.

## Proposed payload layout

Добавить в payload новый корень:

```text
payload/
  chromium/
  config/
  diagnostics/
  extensions/
    cryptopro-cades/
      manifest.json
      ...extension files...
```

Для первого шага достаточно одного extension:
- `extensions/cryptopro-cades/`

Если позже понадобится больше extension'ов, layout уже будет готов.

## Source of truth for extension files

Нужно зафиксировать один канонический источник extension payload:

Варианты:
1. committed files inside repo;
2. pinned external archive downloaded at build time;
3. private asset mirrored into project-controlled location.

Для MVP рекомендую **вариант 1**:
- положить unpacked extension files прямо в репозиторий;
- зафиксировать версию в отдельном note;
- не тащить сетевую нестабильность в build.

Если размер/лицензия мешают, запасной путь — pinned archive + deterministic unpack на build.

## Config changes

### App config

Текущий `payload-template/config/app-config.json` уже умеет принимать `chromiumArgs`.

Для `v0.4` рекомендую **не зашивать** extension args в статический JSON напрямую как единственное место правды, а собирать их в launcher на базе payload layout.

Причина:
- путь к extension зависит от `appDir` после extraction;
- жёсткие абсолютные пути в payload config неудобны.

### Runtime decision

В `internal/bootstrap/bootstrap.go` перед `exec.Command(...)` добавить построение extension args из файловой структуры payload.

Принцип:
- launcher определяет `extensionsRoot := filepath.Join(appDir, "extensions")`
- если существует `extensions/cryptopro-cades`, добавляет Chromium args

## Chromium launch integration

### New helper

Добавить новый helper, например:
- `internal/bootstrap/extensions.go`

Предлагаемое API:

```go
type ExtensionSpec struct {
    Name string
    Path string
}

func detectExtensions(appDir string) ([]ExtensionSpec, error)
func buildExtensionArgs(exts []ExtensionSpec) []string
```

### Initial launch flags

Для первого прохода достаточно такого подхода:
- `--disable-extensions-except=<abs-path-to-extension>`
- `--load-extension=<abs-path-to-extension>`

Если extension'ов станет несколько:
- передавать comma-separated absolute paths.

### Integration point

В `buildChromiumArgs(...)` или рядом с ним:
- оставить существующие app/browser args;
- дополнительно append extension args.

Важно:
- extension args должны добавляться **до** пользовательских override-аргументов только если хотим запретить случайную поломку;
- для MVP лучше append их **до** `appCfg.ChromiumArgs`, чтобы project-owned args были стабильными.

## Stable extension identity

Это отдельный мини-риск v0.4.

Нужно проверить, от чего зависит extension id:
- есть ли в extension `key` в `manifest.json`;
- если нет, можно ли закрепить стабильный id предсказуемым способом;
- если id плавает, diagnostics/page checks будут хрупкими.

Практический план:
1. сначала загрузить extension как есть;
2. руками/через diagnostics получить реальный id;
3. если id нестабилен — отдельно решить вопрос с `manifest key`.

Не блокировать первый коммит этой задачей, но сделать её обязательной частью validation для `v0.4`.

## Diagnostics plan

### Goal

Нужна минимальная проверка, что extension жив.

### Minimal form

Добавить в payload diagnostics дополнительную служебную страницу или блок на существующей странице:

```text
Extension status:
- expected path exists
- chromium launched with extension args
- extension id: <id or unknown>
- extension API probe: ok/fail
```

### Possible implementation directions

Вариант A — page-side probe:
- стартовая/диагностическая страница выполняет JS probe на наличие ожидаемого extension-related surface.

Вариант B — launcher-side marker:
- launcher пишет диагностический JSON о том, что extension path найден и args добавлены.

Для MVP лучше сделать **оба уровня**:
1. launcher пишет факт wiring;
2. diagnostics page показывает runtime result.

## Build pipeline changes

### Payload preparation

В `build/prepare-payload.ps1`:
- после копирования `payload-template/*` убедиться, что `extensions/cryptopro-cades/**` попадает в payload;
- extension files автоматически войдут в `manifest.json` и `payload.zip`.

### Required files check

Расширить `$required` только после того, как source extension станет каноническим.

Например:

```powershell
"extensions/cryptopro-cades/manifest.json"
```

Но лучше делать эту проверку только когда файлы уже действительно добавлены в репозиторий.

### CI

Отдельных workflow не нужно.

Текущий split уже достаточный:
- `build-payload.yml` соберёт payload с extension;
- `build-windows.yml` подхватит stable payload через `payload-lock.json`.

## Suggested repo changes

### New files

- `internal/bootstrap/extensions.go`
- `docs/cryptopro-extension-v0.4-blueprint.md` *(this file)*
- возможно: `payload-template/extensions/cryptopro-cades/**`
- возможно: `payload-template/diagnostics/extension-check.html` или обновление существующей diagnostics page

### Existing files to change

- `internal/bootstrap/bootstrap.go`
- `internal/config/config.go` *(только если понадобится формализовать extension-specific config)*
- `build/prepare-payload.ps1`
- `payload-template/config/app-config.json` *(если потребуется feature flag / diagnostics URL tweak)*
- `payload-template/diagnostics/diagnostics.html`

## Recommended commit sequence

### Commit 1
`feat: add extension payload layout scaffold`
- добавить каталог `payload-template/extensions/cryptopro-cades/`
- положить туда placeholder/real extension files
- без Chromium wiring

### Commit 2
`feat: load unpacked extension in chromium runtime`
- добавить `extensions.go`
- wiring в launcher startup
- логирование найденных extension paths

### Commit 3
`feat: add extension diagnostics probe`
- diagnostics page / diagnostic JSON
- показать, что extension path найден и runtime probe выполняется

### Commit 4
`ci: include extension files in payload validation`
- расширить required files checks
- обновить docs/summary при необходимости

## Validation checklist for v0.4

### Automated
- build-windows CI green
- build-payload CI green
- payload manifest contains extension files
- remote and embedded launchers still build successfully

### Manual on Windows
- embedded launcher first run
- remote launcher first run
- repeat run after cached payload exists
- Chromium opens successfully
- extension appears loaded
- diagnostics page confirms expected status

## Main risks before coding

1. **Real extension packaging/source**
- нужен точный набор файлов extension и понятный license/supply path

2. **Extension id stability**
- может всплыть необходимость фиксировать `key`

3. **Manifest/Chromium compatibility**
- надо убедиться, что unpacked extension вообще поднимается в выбранной pinned Chromium build

Это и есть три главных технических вопроса этапа `v0.4`.

## Recommended first practical task

Первый безопасный шаг прямо сейчас:

**Сделать scaffold для extension delivery без реальной криптологики**
- добавить `extensions/cryptopro-cades/README-or-placeholder`;
- добавить launcher-side `detectExtensions()`;
- пока только логировать найденные extension dirs, не ломая запуск.

После этого вторым шагом уже подключать реальные extension files и включать `--load-extension`.
