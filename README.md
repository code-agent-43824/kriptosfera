# Kriptosfera

[![build-windows](https://github.com/code-agent-43824/kriptosfera/actions/workflows/build-windows.yml/badge.svg)](https://github.com/code-agent-43824/kriptosfera/actions/workflows/build-windows.yml)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)

Криптосфера — концепт и MVP-каркас для десктопного приложения, которое поставляет специализированную Chromium-оболочку и российский клиентский криптостек в режиме «скачал один файл → запустил → вставил токен → работаешь».

## Содержание

- [Что зафиксировано по документам](#что-зафиксировано-по-документам)
- [MVP scope](#mvp-scope)
- [Текущие выводы по CSP Lite / Mini CSP](#текущие-выводы-по-csp-lite--mini-csp)
- [Репозиторий](#репозиторий)
- [Локальная разработка](#локальная-разработка)
- [Сборка на GitHub Actions](#сборка-на-github-actions)
- [Документно подтверждённые этапы MVP](#документно-подтверждённые-этапы-mvp)
- [Текущий следующий шаг](#текущий-следующий-шаг)
- [Ближайшие инженерные задачи](#ближайшие-инженерные-задачи)
- [Документация](#документация)
- [Лицензия](#лицензия)

Полный обзор архитектуры — в [`docs/architecture.md`](docs/architecture.md); индекс документации — в [`docs/README.md`](docs/README.md).

## Что зафиксировано по документам

Из входных документов проекта следует такой базовый замысел:
- первый приоритет — Windows MVP;
- поставка пользователю как один `.exe` без wizard-установщика и без admin rights;
- внутри — launcher на Go, управляемый payload, Chromium runtime, отдельный профиль браузера, CryptoPro extension, native host и криптографические библиотеки;
- первый референсный сценарий — тестовая страница CryptoPro CAdES Browser Plug-in;
- критерий успеха MVP — успешная тестовая подпись с Рутокеном без системной установки CryptoPro CSP.

После первых запусков MVP-план уточнён: embedded payload mode сохраняется, но основной продуктовый вектор теперь — **thin launcher + remote payload mode**.

## MVP scope

В текущем стартовом репозитории подготовлен именно **каркас MVP**, а не готовая интеграция с CryptoPro.

Сейчас есть:
- каркас `launcher` на Go;
- логика single-file bootstrapper с embedded `payload.zip`;
- remote payload mode для thin launcher с HTTPS-загрузкой, SHA-256 проверкой, ограничением размера загрузки (cap по pinned size / абсолютный предел) и cache reuse;
- шаблон payload с pinned Chromium runtime и CryptoPro CAdES Browser Plug-in extension `1.3.17`;
- hosted diagnostics page для проверки CryptoPro extension, Browser Plugin и CSP/provider state через официальный `cadesplugin_api.js`;
- deployed Mini CSP test mirrors on `mescheryakov.pro`, including
  `internal-csp-early`, where `EnableInternalCSP` is set before
  `cadesplugin_api.js` and then re-asserted for clean-machine experiments;
- read-only Windows script `tools/windows/inspect-cryptopro-modules.ps1` для фиксации фактически загруженных модулей `nmcades.exe`;
- PowerShell-скрипты сборки под GitHub Actions;
- Windows CI workflow на бесплатных GitHub-hosted runners;
- публикация Windows-сборок двумя раздельными артефактами (embedded и тонкий remote), чтобы тонкую версию можно было скачать отдельно, плюс отдельные release assets на тегах.

Сейчас ещё нет:
- активированного bundled CSP Lite / Mini CSP режима для чистой машины;
- рабочего сценария подписи без установленного системного CryptoPro CSP.

Сейчас уже есть:
- foundation launcher первого этапа;
- managed Chromium runtime второго этапа, который подготавливается в payload из pinned Chrome for Testing build;
- отдельный `user-data-dir` для запуска встроенного браузера;
- cache-friendly подготовка Chromium runtime в CI;
- CryptoPro extension layer: unpacked extension доставляется в payload, launcher добавляет Chromium extension flags, extension id стабилен через `manifest.key`;
- CryptoPro Browser Plugin bundle закреплён отдельным lock-файлом, скачивается с project static storage, проверяется по SHA-256/size и встраивается в оба launcher variants;
- launcher разворачивает встроенный CryptoPro Browser Plugin bundle в AppData рядом с Chromium, пропускает MSI pseudo-path entries с Windows-недопустимыми именами и проверяет наличие `nmcades.exe`, `nmcades.json`, `npcades.dll`;
- launcher генерирует native messaging manifest `ru.cryptopro.nmcades.json` и регистрирует его в HKCU для текущего пользователя;
- ручная проверка показала, что на машине с установленным обычным CryptoPro CSP приложение ведёт себя как настроенный Chrome: видит extension, Browser Plugin, plugin version, системный CSP, стандартное окно подтверждения доступа и сертификаты;
- минимальная app-config validation: `startUrl` должен быть валидным URL и соответствовать `allowedOrigins` (если список задан), `diagnosticsUrl` обязан быть HTTPS, а `profileName` проверяется как безопасный одиночный сегмент пути (без `..`, разделителей путей и `:`), чтобы каталог профиля не мог выйти за пределы app-root;
- diagnostics остаётся включённой для MVP; `diagnosticsUrl` включает открытие публичной HTTPS-страницы диагностики рядом с целевой страницей.

Полная доменная политика Chromium после старта — не часть текущего MVP. Это future product hardening для клиентских/брендированных сборок; сейчас `allowedOrigins` используется как guard от неправильного стартового URL в конфиге.

Текущая точка:
- двухмашинная diagnostics matrix по `CAdESCOM.About` / CSP state снята;
- extension + native Browser Plugin delivery подтверждены;
- ProcMon на чистой машине показал, что `nmcades.exe` реально загружает bundled `npcades.dll` и `cplib.dll` из AppData;
- следующий этап — аккуратная активация bundled CSP Lite / Mini CSP через CryptoPro runtime/CAPI layer, а не через замену CAdES-плагина.

## Текущие выводы по CSP Lite / Mini CSP

Исследование на 2026-05-24:

- На машине без системного CSP запускаются extension, native host и `CAdESCOM.About`; `About.CSPName(80)` / `CSPVersion("", 80)` падают с `0x80090017` («Тип поставщика не определен»).
- На чистой машине `CAdESCOM.Store` может открыть `MY` и увидеть сертификат с приватным ключом через `Microsoft Smart Card Key Storage Provider`, но `SignCades` падает с `0x80090014` (`wrong provider type`). Одна видимость CNG/KSP-ключа не равна совместимости с CAdESCOM-подписью.
- Ранний `cryptopro-modules.json` был недостаточен: он не показал `npcades.dll` / `cplib.dll`, хотя ProcMon позже подтвердил их загрузку. Для DLL-загрузки и failed lookup основным инструментом теперь считается ProcMon.
- Linux/Unix-модель CryptoPro указывает на собственный runtime/config слой CryptoPro: `cpconfig`, `capi10`, `capi20`, `csp`, `pcsc`, `cng`. Поэтому MiniCSP/CSP Lite может не регистрировать полноценный Windows provider напрямую, а ожидать вызова через CryptoPro CAPI/CSP libraries.
- Рабочая гипотеза: цепочка должна быть `nmcades.exe -> npcades.dll -> cplib.dll -> capi10/capi20/cpcspi/Mini CSP runtime -> token/container`. Сейчас подтверждена только первая часть до `cplib.dll`.

Скорректированное направление:

- Не начинать с ручной записи в системный Windows CSP registry.
- Сначала выяснить, какие DLL/config/registry paths `cplib.dll` и CAdES runtime пытаются открыть при `Store.Open` и `SignCades`.
- Проверять именно 32-bit слой: текущий `nmcades.exe` работает под WOW64, значит x64-only MiniCSP DLL не подцепятся.
- Подключение MiniCSP делать маленькими обратимыми шагами: сначала DLL search path / app-local layout / environment, затем только при необходимости HKCU/HKLM config or registry.

## Репозиторий

```text
cmd/kriptosfera-launcher/   entrypoint launcher
internal/bootstrap/         распаковка payload и запуск runtime
internal/config/            JSON-конфиг приложения
internal/logging/           launcher log
payload-template/           шаблон payload
build/                      PowerShell scripts для CI/локальной сборки
.github/workflows/          GitHub Actions
```

## Локальная разработка

Реальные `payload.zip` и `cryptopro-plugin.zip` собираются/скачиваются build-скриптами и в Git не хранятся. Чтобы launcher компилировался и `go test ./...` проходил на чистом checkout (в т.ч. на Linux/macOS dev-машинах), в репозиторий закоммичены **пустые placeholder-файлы нулевого размера** `internal/bootstrap/payload.zip` и `internal/bootstrap/cryptopro-plugin.zip`. Они нужны только для удовлетворения директив `go:embed`, не содержат бинарников CryptoPro (launcher трактует пустой embed как «bundle не встроен») и во время Windows-сборки перезаписываются настоящими артефактами (`build/embed-payload.ps1`, `build/fetch-cryptopro-plugin.ps1`).

Базовая проверка кода:

```sh
go vet ./...
go test ./...
GOOS=windows GOARCH=amd64 go build ./...
```

## Сборка на GitHub Actions

Основные workflow:
- `.github/workflows/build-windows.yml` — обычная сборка launcher'ов
- `.github/workflows/build-payload.yml` — отдельная редкая сборка/publish payload

### Как теперь устроен pipeline

`build-windows.yml`:
1. собирает payload package из текущего commit
2. собирает `dist/KriptosferaDemo.exe` (embedded)
3. собирает `dist/KriptosferaDemo-remote.exe` (remote)
4. публикует **два отдельных** workflow artifact'а — embedded и remote — плюс release assets на тегах

`build-payload.yml`:
1. готовит payload (включая Chromium runtime)
2. упаковывает `dist/payload.zip` и `dist/payload.json`
3. публикует payload на сервер по SSH
4. публикует **один** payload artifact / release assets

### Payload source для launcher build

Обычный `build-windows.yml` собирает embedded launcher против payload из текущего commit. Remote launcher в этом workflow собирается против уже опубликованного immutable payload из `build/payload-lock.json`, чтобы release artifact не ссылался на payload URL, который ещё не загружен на сервер.

Для отдельного stable-payload сценария в `build/build-windows.ps1` оставлен флаг `-UseStablePayload`: он скачивает payload по `build/payload-lock.json`, проверяет SHA-256/size и собирает launchers против уже опубликованного immutable payload.

### Важный момент про артефакты

GitHub Actions workflow artifacts технически скачиваются GitHub'ом как zip-контейнер — это ограничение самой платформы.

Модель публикации:
- обычный launcher CI-run публикует **два независимых** workflow artifact'а, чтобы каждую сборку можно было скачать отдельно (тонкую — без большого embedded-файла):
  - `kriptosfera-windows-embedded` — `KriptosferaDemo.exe` (payload вшит, большой файл) + `README.txt` + диагностический скрипт;
  - `kriptosfera-windows-remote` — `KriptosferaDemo-remote.exe` (тонкий launcher) + `README.txt` + диагностический скрипт;
- payload workflow публикует только **один** workflow artifact: `payload`;
- tag build (`v*`) дополнительно прикрепляет сырые `.exe` и payload-файлы как отдельные **GitHub Release assets** (каждый качается независимо).

## Документно подтверждённые этапы MVP

1. Embedded single-file bootstrapper.
2. Embedded Chromium runtime launch.
3. Remote payload mode / thin launcher.
4. CryptoPro extension.
5. Native messaging.
6. CryptoPro CSP Lite / Mini CSP activation.
7. Рутокен и тестовая подпись.
8. Минимальная диагностика.
9. macOS PoC.

## Текущий следующий шаг

**Этап 6: переход к CSP Lite / Mini CSP.**

Что закрыто внутри этапов 3-4:
- выделен runtime/payload abstraction layer;
- добавлен remote runtime core (`RemotePayloadSource`, temp download, SHA-256 verify, cache reuse);
- добавлены build/runtime-config generation и immutable payload artifact layout;
- workflow уже собирает и embedded launcher, и thin launcher;
- для remote first-run добавлен minimal progress UX с маленьким progress window на Windows.
- CryptoPro extension добавлен в payload и проверен через launcher/runtime diagnostics.
- native messaging manifest генерируется и регистрируется в HKCU;
- Browser Plugin на машине с системным CryptoPro CSP подтверждён ручной проверкой через CryptoPro demo page.

Что закрыто диагностикой:
- на машине с системным CSP diagnostics показывает plugin `2.0.15700`, CSP `5.0.13455`, provider name и целевая страница видит сертификаты;
- на чистой машине extension/API и `CAdESCOM.About` доступны, но plugin/CSP state остаётся `0.0.0` / `0x80090017`.
- launcher пишет `cryptopro-runtime.json`, а отдельный read-only script может записать `cryptopro-modules.json` со списком загруженных CryptoPro/CAdES/Mini CSP модулей.
- ProcMon на чистой машине подтвердил загрузку `npcades.dll` и `cplib.dll` из bundled Browser Plug-in каталога; значит следующий пробел — не native messaging и не CAdES plugin bootstrap, а переход от `cplib.dll` к CAPI/MiniCSP runtime.

Что дальше:
- на чистой машине без системного CSP открыть свежий
  `internal-csp-early` mirror и/или hosted diagnostics page из текущей сборки,
  чтобы проверить гипотезу раннего `EnableInternalCSP`;
- если ранний флаг не активирует провайдеры, снять расширенный ProcMon-трейс на чистой машине во время `Store.Open` и `SignCades`, с фокусом на `capi`, `csp`, `cpcsp`, `cplib`, `config`, `Crypto Pro`, `NAME NOT FOUND`;
- собрать такой же ProcMon-трейс на машине с системным CSP для сравнения successful path;
- проверить состав bundled MiniCSP/CSP Lite, особенно наличие 32-bit `capi10.dll`, `capi20.dll`, `cpcspi.dll`, `cpsuprt.dll`, `cpui.dll`, `csp*.dll` и config/layout files;
- попробовать app-local/PATH activation так, чтобы MiniCSP DLL лежали в search path процесса `nmcades.exe`, и фиксировать, меняется ли `0x80090017` / `0x80090014`.

## Ближайшие инженерные задачи

- расширить диагностику/скрипты так, чтобы они фиксировали не только loaded modules, но и ProcMon-derived failed DLL/config lookups;
- после подтверждения нужного DLL/config layout реализовать минимальный обратимый activation step для bundled CSP Lite / Mini CSP;
- затем вернуться к reference signing flow с Рутокеном;
- при необходимости позже вернуться к UX-polish progress окна и richer diagnostics.

Пока `diagnosticsEnabled=true` и задан `diagnosticsUrl`, launcher открывает целевую страницу и публичную HTTPS-страницу диагностики рядом в обычном Chromium window-mode. Локальный diagnostics server в launcher не используется.

## Документация

- [`docs/README.md`](docs/README.md) — индекс всей документации.
- [`docs/architecture.md`](docs/architecture.md) — архитектура launcher и runtime-раскладка.
- [`docs/cryptopro-csp-lite-plan.md`](docs/cryptopro-csp-lite-plan.md) — текущий ключевой план (CSP Lite / Mini CSP).
- [`CONTRIBUTING.md`](CONTRIBUTING.md) — локальная сборка, тесты, соглашения.
- [`CHANGELOG.md`](CHANGELOG.md) — история изменений.

## Лицензия

Исходный код Kriptosfera распространяется под лицензией Apache 2.0 — см. [`LICENSE`](LICENSE).

Сторонние компоненты, доставляемые в runtime (Chromium, CryptoPro CAdES Browser Plug-in, native messaging host, CryptoPro CSP / Mini CSP), **не покрываются** этой лицензией и регулируются собственными условиями правообладателей; в репозитории они не хранятся. Подробности — в [`NOTICE`](NOTICE).
