# Kriptosfera

Криптосфера — концепт и MVP-каркас для десктопного приложения, которое поставляет специализированную Chromium-оболочку и российский клиентский криптостек в режиме «скачал один файл → запустил → вставил токен → работаешь».

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
- remote payload mode для thin launcher с HTTPS-загрузкой, SHA-256 проверкой и cache reuse;
- шаблон payload с pinned Chromium runtime и CryptoPro CAdES Browser Plug-in extension `1.3.17`;
- hosted diagnostics page для проверки CryptoPro extension, Browser Plugin и CSP/provider state через официальный `cadesplugin_api.js`;
- read-only Windows script `tools/windows/inspect-cryptopro-modules.ps1` для фиксации фактически загруженных модулей `nmcades.exe`;
- PowerShell-скрипты сборки под GitHub Actions;
- Windows CI workflow на бесплатных GitHub-hosted runners;
- модель публикации артефактов без дополнительного ручного zip на release-тегах.

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
- минимальная app-config validation: `startUrl` должен соответствовать `allowedOrigins`, если список задан;
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

## Сборка на GitHub Actions

Основные workflow:
- `.github/workflows/build-windows.yml` — обычная сборка launcher'ов
- `.github/workflows/build-payload.yml` — отдельная редкая сборка/publish payload

### Как теперь устроен pipeline

`build-windows.yml`:
1. собирает payload package из текущего commit
2. собирает `dist/KriptosferaDemo.exe` (embedded)
3. собирает `dist/KriptosferaDemo-remote.exe` (remote)
4. публикует один launcher artifact / release assets

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

Теперь модель проще:
- обычный launcher CI-run публикует только **один** workflow artifact: `launchers`
- payload workflow публикует только **один** workflow artifact: `payload`
- tag build (`v*`) дополнительно прикрепляет сырые `.exe` и payload-файлы как **GitHub Release assets**

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
- снять расширенный ProcMon-трейс на чистой машине во время `Store.Open` и `SignCades`, с фокусом на `capi`, `csp`, `cpcsp`, `cplib`, `config`, `Crypto Pro`, `NAME NOT FOUND`;
- собрать такой же ProcMon-трейс на машине с системным CSP для сравнения successful path;
- проверить состав bundled MiniCSP/CSP Lite, особенно наличие 32-bit `capi10.dll`, `capi20.dll`, `cpcspi.dll`, `cpsuprt.dll`, `cpui.dll`, `csp*.dll` и config/layout files;
- попробовать app-local/PATH activation так, чтобы MiniCSP DLL лежали в search path процесса `nmcades.exe`, и фиксировать, меняется ли `0x80090017` / `0x80090014`.

## Ближайшие инженерные задачи

- расширить диагностику/скрипты так, чтобы они фиксировали не только loaded modules, но и ProcMon-derived failed DLL/config lookups;
- после подтверждения нужного DLL/config layout реализовать минимальный обратимый activation step для bundled CSP Lite / Mini CSP;
- затем вернуться к reference signing flow с Рутокеном;
- при необходимости позже вернуться к UX-polish progress окна и richer diagnostics.

Пока `diagnosticsEnabled=true` и задан `diagnosticsUrl`, launcher открывает целевую страницу и публичную HTTPS-страницу диагностики рядом в обычном Chromium window-mode. Локальный diagnostics server в launcher не используется.
