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
- шаблон payload с pinned Chromium runtime и временно откатанным CryptoPro CAdES Browser Plug-in extension Manifest V2 `1.2.13`;
- hosted diagnostics page для проверки CryptoPro extension, Browser Plugin и CSP/provider state через официальный `cadesplugin_api.js`;
- PowerShell-скрипты сборки под GitHub Actions;
- Windows CI workflow на бесплатных GitHub-hosted runners;
- публикация Windows-сборок двумя раздельными артефактами (embedded и тонкий remote), чтобы тонкую версию можно было скачать отдельно, плюс отдельные release assets на тегах.

Сейчас в процессе интеграции:
- сборка рабочей связки в launcher. Причина блокера найдена: пиннутая сборка плагина `2.0.15700` была битой; рабочая — `2.0.15000` + extension Manifest V2 `1.2.13` + Chromium с поддержкой MV2 (Chrome 138). Mini CSP при этом активируется без системного CSP. Плагин, extension и MV2 policy уже перепиннуты как временный legacy compatibility profile; остаётся проверить подпись с Рутокеном — см. `docs/cryptopro-csp-lite-plan.md`;
- сквозная проверка подписи с Рутокеном через нашу сборку.

Сейчас уже есть:
- foundation launcher первого этапа;
- managed Chromium runtime второго этапа, который подготавливается в payload из pinned Chrome for Testing build;
- отдельный `user-data-dir` для запуска встроенного браузера;
- cache-friendly подготовка Chromium runtime в CI;
- CryptoPro extension layer: unpacked extension доставляется в payload, launcher добавляет Chromium extension flags, extension id стабилен через `manifest.key`;
- CryptoPro Browser Plugin bundle закреплён отдельным lock-файлом, скачивается с project static storage, проверяется по SHA-256/size, нормализуется до runtime-поддерева `CAdES Browser Plug-in` и встраивается в оба launcher variants без MSI/Common/64-bit деревьев;
- launcher разворачивает встроенный CryptoPro Browser Plugin bundle в AppData рядом с Chromium под vendor-style путём `Crypto Pro\CAdES Browser Plug-in`, пропускает MSI pseudo-path entries с Windows-недопустимыми именами и проверяет наличие `nmcades.exe`, `nmcades.json`, `npcades.dll`;
- launcher генерирует native messaging manifest `ru.cryptopro.nmcades.json` и регистрирует его в HKCU для текущего пользователя;
- ручная проверка показала, что на машине с установленным обычным CryptoPro CSP приложение ведёт себя как настроенный Chrome: видит extension, Browser Plugin, plugin version, системный CSP, стандартное окно подтверждения доступа и сертификаты;
- минимальная app-config validation: `startUrl` должен быть валидным URL и соответствовать `allowedOrigins` (если список задан), `diagnosticsUrl` обязан быть HTTPS, а `profileName` проверяется как безопасный одиночный сегмент пути (без `..`, разделителей путей и `:`), чтобы каталог профиля не мог выйти за пределы app-root;
- launcher запускает Chromium как отдельное приложение (`--app=startUrl`) при `windowMode: "app"`; диагностика в demo-конфиге выключена.

Полная доменная политика Chromium после старта — не часть текущего MVP. Это future product hardening для клиентских/брендированных сборок; сейчас `allowedOrigins` используется как guard от неправильного стартового URL в конфиге.

## Текущие выводы по CSP Lite / Mini CSP

Блокер «провайдер не грузится на чистой машине» **решён по первопричине**: пиннутая
сборка CAdES-плагина `2.0.15700` оказалась **битой** (подтверждено CryptoPro), и
её внутренний Mini CSP не активировался. Откат на плагин **`2.0.15000`** поднимает
Mini CSP **без системного CSP**: страница диагностики показывает «Криптопровайдер
загружен», провайдер `Crypto-Pro GOST R 34.10-2012 …`, CSP `5.0.13001`. Наш подход
был верным.

Рабочая связка: плагин **2.0.15000** + extension **Manifest V2 `1.2.13`** +
Chromium с поддержкой MV2 (**Chrome 138**, последняя версия с
`ExtensionManifestV2Availability`). Что осталось интегрировать в launcher и какие
дальнейшие цели (возврат к свежему Chromium + Manifest V3, когда CryptoPro выпустит
исправленную сборку плагина) — в [`docs/cryptopro-csp-lite-plan.md`](docs/cryptopro-csp-lite-plan.md).

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

Что выяснено по первопричине:
- блокер был в **битой сборке плагина `2.0.15700`**; рабочая — `2.0.15000`, и с ней Mini CSP активируется на чистой машине без системного CSP (провайдер `Crypto-Pro GOST R 34.10-2012 …`, CSP `5.0.13001`).
- рабочая связка требует extension **Manifest V2 `1.2.13`** и Chromium с поддержкой MV2 (**Chrome 138**).

Что дальше (интеграция рабочей связки в launcher):
- плагин перепиннут на `2.0.15000` (lock + версия + required-files + bump layout);
- bundled extension заменён на Manifest V2 `1.2.13`;
- Chromium уже запиннут на 138.x; launcher выставляет `ExtensionManifestV2Availability=2` только когда в payload найдено loadable MV2-расширение;
- сквозная проверка подписи с Рутокеном через нашу сборку.

Детальный план и будущие цели — в [`docs/cryptopro-csp-lite-plan.md`](docs/cryptopro-csp-lite-plan.md).

## Ближайшие инженерные задачи

- собрать рабочую связку (плагин 2.0.15000 + extension MV2 1.2.13 + Chromium 138) в payload и проверить подпись с Рутокеном;
- в будущем — вернуться к свежему Chromium + Manifest V3, когда CryptoPro выпустит исправленную сборку плагина.

Launcher запускает встроенный Chromium как отдельное приложение (`--app=startUrl`) при `windowMode: "app"`; диагностика в demo-конфиге выключена, локальный diagnostics server не используется.

## Документация

- [`docs/README.md`](docs/README.md) — индекс всей документации.
- [`docs/architecture.md`](docs/architecture.md) — архитектура launcher и runtime-раскладка.
- [`docs/cryptopro-csp-lite-plan.md`](docs/cryptopro-csp-lite-plan.md) — текущий ключевой план (CSP Lite / Mini CSP).
- [`docs/payload-slimming-plan.md`](docs/payload-slimming-plan.md) — текущий размер payload и план аккуратного сокращения Chromium.
- [`CONTRIBUTING.md`](CONTRIBUTING.md) — локальная сборка, тесты, соглашения.
- [`CHANGELOG.md`](CHANGELOG.md) — история изменений.

## Лицензия

Исходный код Kriptosfera распространяется под лицензией Apache 2.0 — см. [`LICENSE`](LICENSE).

Сторонние компоненты, доставляемые в runtime (Chromium, CryptoPro CAdES Browser Plug-in, native messaging host, CryptoPro CSP / Mini CSP), **не покрываются** этой лицензией и регулируются собственными условиями правообладателей; в репозитории они не хранятся. Подробности — в [`NOTICE`](NOTICE).
