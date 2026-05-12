# Remote payload mode — implementation blueprint

## Goal

Добавить в `Kriptosfera` второй режим поставки:
- `embedded` — текущий режим, сохраняется;
- `remote` — thin launcher, который скачивает immutable `payload.zip`, проверяет SHA-256 и использует тот же extraction pipeline.

Это документ не про product-vision, а про конкретную реализацию в текущем репозитории.

## Target result for v0.3

На выходе этапа `v0.3` должны существовать две сборки launcher:

1. **Embedded launcher**
   - содержит `payload.zip` через `go:embed`;
   - работает как сейчас;
   - нужен для офлайн-демо, отладки и закрытых контуров.

2. **Thin launcher (remote)**
   - не содержит Chromium/runtime внутри;
   - содержит runtime-config с `payloadUrl`, `payloadSha256`, `payloadSize`, `payloadVersion`;
   - при первом запуске скачивает payload во временный файл;
   - проверяет hash;
   - распаковывает в staging;
   - проверяет `manifest.json`;
   - атомарно публикует payload в рабочий каталог;
   - на повторных запусках не скачивает заново, если локальный payload уже валиден.

## Design constraints

- Не использовать пользовательский системный Chrome как основной runtime.
- Не дублировать extraction/state/manifest logic для `embedded` и `remote`.
- Не держать крупный archive целиком в памяти в `remote` mode.
- Launcher должен доверять встроенному `payloadSha256`, а не только `payload.json` на сервере.
- Пути публикации payload должны быть immutable.
- Поведение double-click должно оставаться без wizard-установщика.

## Repo changes

### 1) Новый runtime config для launcher

Добавить новый тип конфигурации launcher, отдельно от `payload-template/config/app-config.json`.

Новый файл:
- `internal/config/runtime_config.go`

```go
type RuntimePayloadConfig struct {
    Mode          string `json:"mode"` // embedded | remote
    Version       string `json:"version"`
    URL           string `json:"url,omitempty"`
    SHA256        string `json:"sha256,omitempty"`
    Size          int64  `json:"size,omitempty"`
}

type RuntimeConfig struct {
    ProductName string               `json:"productName"`
    Version     string               `json:"version"`
    Payload     RuntimePayloadConfig `json:"payload"`
}
```

Загрузка:
- для MVP runtime config можно генерировать на build-этапе рядом с launcher code и вшивать через `go:embed` как сейчас вшивается `app-version.txt`;
- `embedded` launcher получает `payload.mode=embedded`;
- `remote` launcher получает полный блок `payload.*`.

### 2) Payload source abstraction

Новые файлы:
- `internal/bootstrap/payload_source.go`
- `internal/bootstrap/payload_source_embedded.go`
- `internal/bootstrap/payload_source_remote.go`

Интерфейс:

```go
type PayloadSource interface {
    Mode() string
    Version() string
    ExpectedSHA256() string
    Open(ctx context.Context, logger *logging.Logger) (PayloadArchive, error)
}

type PayloadArchive struct {
    Reader io.ReadCloser
    Size   int64
    Close  func() error
}
```

Примечания:
- `EmbeddedPayloadSource` открывает `embeddedPayload` через `io.NopCloser(bytes.NewReader(...))`.
- `RemotePayloadSource` скачивает archive во временный файл, параллельно считает SHA-256, сверяет hash и только потом открывает temp file как `Reader`.
- Удаление temp file должно происходить через `Close()`.

### 3) Общий payload manager

Новый файл:
- `internal/bootstrap/payload_manager.go`

Ответственность `PayloadManager`:
- вычислить root/appDir;
- проверить, готов ли локальный payload;
- если готов — вернуть `reused=true`;
- если не готов — запросить archive у `PayloadSource`;
- распаковать archive в staging;
- проверить `manifest.json`;
- записать `.payload-state.json`;
- записать `.payload-ready`;
- атомарно переместить staging → appDir.

Предлагаемый API:

```go
type PrepareResult struct {
    AppDir string
    Reused bool
}

type PayloadManager struct {}

func (m *PayloadManager) Prepare(ctx context.Context, source PayloadSource, cfg RuntimeConfig, logger *logging.Logger) (PrepareResult, error)
```

Важно:
- текущую логику `ensurePayload`, `isPreparedPayload`, `verifyExtractedPayload`, `writePayloadState`, `unzip` нужно перенести сюда, а не копировать.
- `bootstrap.Run()` должен перестать знать, embedded там payload или remote.

### 4) Payload state format

Текущий `.payload-state.json` расширить.

```go
type PayloadState struct {
    Version       string `json:"version"`
    PayloadMode   string `json:"payloadMode"`
    PayloadSHA256 string `json:"payloadSha256"`
    SourceURL     string `json:"sourceUrl,omitempty"`
}
```

Смысл:
- `Version` — логическая версия приложения/payload;
- `PayloadSHA256` — канонический идентификатор содержимого;
- `PayloadMode` — полезен для диагностики;
- `SourceURL` — полезен в remote mode для логов и support-разбора.

### 5) Remote downloader

Новый файл:
- `internal/bootstrap/downloader.go`

Минимальный API:

```go
type DownloadResult struct {
    TempPath string
    Bytes    int64
    SHA256   string
}

func DownloadFile(ctx context.Context, url string, expectedSize int64, logger *logging.Logger, progress func(done, total int64)) (DownloadResult, error)
```

Требования:
- `http.Client` с timeout;
- скачивание только по `https://`;
- запись в temp file внутри user-writable temp dir;
- hash считать потоково через `io.MultiWriter(file, hash)`;
- если `expectedSize > 0`, логировать mismatch как warning/error;
- частично скачанный файл удалять при любой ошибке;
- launcher не должен удалять уже валидный локальный payload, если новая загрузка не удалась.

### 6) Error model

Новый файл:
- `internal/bootstrap/errors.go`

Коды:

```go
const (
    ErrPayloadDownloadFailed = "PAYLOAD_DOWNLOAD_FAILED"
    ErrPayloadHashMismatch   = "PAYLOAD_HASH_MISMATCH"
    ErrPayloadExtractFailed  = "PAYLOAD_EXTRACT_FAILED"
    ErrPayloadManifestInvalid = "PAYLOAD_MANIFEST_INVALID"
    ErrPayloadNotFound       = "PAYLOAD_NOT_FOUND"
)
```

Ошибка должна уметь отдавать:
- user-facing message;
- code;
- wrapped technical error.

Например:

```go
type LauncherError struct {
    Code    string
    Message string
    Err     error
}
```

`ReportFatal(...)` должен показывать человеку короткий текст с кодом ошибки, а технические детали писать в лог.

### 7) Logging changes

В логах remote flow должны появиться как минимум такие события:

```text
launcher start version=0.3.0 mode=remote
payload check version=0.3.0 sha256=<sha>
payload cache miss path=<appDir>
download start url=<url> expected_size=<size>
download progress bytes=<n> total=<n>
download complete bytes=<n> sha256=<sha>
payload archive verified sha256=<sha>
extract start staging=<dir>
manifest verify ok files=<count>
payload published appDir=<dir>
launch chromium path=<path>
```

Для `embedded` mode формат логов должен остаться близким, но с `mode=embedded`.

### 8) Minimal progress UX

Для `v0.3` не нужен полноценный installer.

Два уровня допустимости:

1. **Technical MVP minimum**
   - логирование прогресса;
   - `MessageBox` только при ошибках.

2. **Preferred for demo**
   - отдельное маленькое progress window/overlay:
     - «Подготовка рабочей среды...»
     - «Загрузка компонентов: 42%»

Рекомендация для первого прохода:
- не блокировать архитектурный этап GUI-обвязкой;
- сначала сделать callback `progress(done, total)` в downloader;
- UI можно повесить поверх него следующим коммитом.

### 9) Build pipeline split

Нужно разделить pipeline на понятные фазы.

#### Embedded build

Текущий поток почти сохраняется:
1. prepare payload dir
2. prepare Chromium
3. create manifest
4. pack `payload.zip`
5. embed `payload.zip`
6. build embedded launcher

#### Remote build

Новые шаги:
1. prepare payload dir
2. prepare Chromium
3. create manifest
4. pack `payload.zip`
5. compute `payload.zip` SHA-256
6. generate immutable publish path
7. generate runtime config for thin launcher
8. build thin launcher without embedded payload
9. publish `payload.zip`
10. publish `payload.json`
11. publish thin launcher artifact

### 10) Build scripts changes

Текущие файлы:
- `build/prepare-payload.ps1`
- `build/embed-payload.ps1`
- `build/build-windows.ps1`

Предлагаемое расширение:

Новые скрипты:
- `build/package-payload.ps1`
- `build/generate-runtime-config.ps1`
- `build/publish-payload.ps1`

Идея:
- `prepare-payload.ps1` — только сборка payload directory;
- `package-payload.ps1` — zip + manifest + sha256 + size;
- `generate-runtime-config.ps1` — создаёт embedded/runtime config для выбранного mode;
- `embed-payload.ps1` использовать только для `embedded` mode;
- `publish-payload.ps1` — заливает immutable payload на host.

### 11) Publishing model for MVP

Для MVP достаточно Caddy + static files на сервере агента.

Immutable layout:

```text
/payloads/win64/demo/0.3.0/<sha256>/payload.zip
/payloads/win64/demo/0.3.0/<sha256>/payload.json
```

`payload.json`:

```json
{
  "appId": "ru.kriptosfera.demo",
  "platform": "win64",
  "payloadVersion": "0.3.0",
  "archive": "payload.zip",
  "sha256": "<sha256>",
  "size": 188000000,
  "createdAt": "<timestamp>"
}
```

Launcher не должен доверять только этому файлу, но публиковать его всё равно нужно.

### 12) bootstrap.Run refactor target

`internal/bootstrap/bootstrap.go` после рефакторинга должен стать тоньше.

Желаемая схема:

```go
func Run(cfg RuntimeConfig) error {
    root := ...
    logger := ...
    source := NewPayloadSource(cfg, logger)
    prepareResult, err := payloadManager.Prepare(ctx, source, cfg, logger)
    if err != nil { return err }
    appCfg := config.Load(filepath.Join(prepareResult.AppDir, "config", "app-config.json"))
    return launchChromium(...)
}
```

То есть:
- выбор source — сверху;
- lifecycle payload — в manager;
- запуск Chromium — в отдельной функции.

## Suggested commit sequence

### Commit 1
`refactor: add runtime payload config and source abstraction`
- `runtime_config.go`
- `payload_source.go`
- wire config loading
- no remote download yet

### Commit 2
`refactor: extract shared payload manager`
- перенос common extraction/state logic
- tests stay green

### Commit 3
`feat: add remote payload downloader and hash verification`
- temp download
- SHA-256 verify
- error codes
- logging

### Commit 4
`build: add thin launcher runtime config generation`
- build scripts split
- remote config generation

### Commit 5
`ci: add remote payload packaging and publish flow`
- payload.zip
- payload.json
- immutable publish path

### Commit 6
`feat: add minimal progress reporting for remote payload`
- progress callback
- optional simple window/message updates

## Test plan

### Unit tests

Добавить сценарии:
- embedded source returns archive;
- remote source fails on non-https URL;
- remote source fails on hash mismatch;
- payload manager reuses existing valid payload;
- payload manager ignores missing `.payload-ready`;
- payload manager rejects broken `manifest.json`;
- payload manager does not destroy valid old payload when remote download fails.

### Integration-ish tests

- local test HTTP server serving payload;
- download → verify → extract → launch-prep;
- second run reuses cached payload without HTTP hit;
- changed SHA causes redownload/reject.

### Manual checks on Windows

- first run thin launcher with internet;
- offline rerun after successful first run;
- first run without internet shows clear error;
- corrupted remote payload gives hash mismatch error;
- double click does not open console window.

## Implementation progress

### Done on 2026-05-12
- введён `RuntimeConfig.Payload`;
- добавлен `PayloadSource` interface;
- текущий embedded flow вынесен в `EmbeddedPayloadSource`;
- добавлен общий `PayloadManager`;
- launcher переведён на новый каркас без изменения внешнего поведения embedded mode;
- добавлены `RemotePayloadSource`, `DownloadFile`, `LauncherError` и remote runtime core;
- launcher теперь умеет выбирать `remote` mode на уровне source selection;
- добавлены тесты на https-download, hash mismatch, non-https reject и cache reuse.
- добавлены build/runtime-config generation и split сборки embedded/remote launcher;
- добавлен immutable payload artifact layout для `payload.zip` / `payload.json`;
- добавлен minimal progress UX для remote first-run: маленькое progress window на Windows с фазами подготовки / загрузки / распаковки / проверки.

### Next coding step
1. считать этап `remote payload mode / thin launcher` закрытым по MVP;
2. перейти к следующему продуктово значимому слою: CryptoPro extension / native messaging / crypto stack;
3. отдельно позже можно улучшать polish: более красивый progress UI, richer diagnostics, update semantics.

Иными словами: этап 3 теперь закрывает и core, и build/publish, и минимальный user-facing UX.
