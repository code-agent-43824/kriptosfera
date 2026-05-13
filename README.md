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
- шаблон payload;
- PowerShell-скрипты сборки под GitHub Actions;
- Windows CI workflow на бесплатных GitHub-hosted runners;
- модель публикации артефактов без дополнительного ручного zip на release-тегах.

Сейчас ещё нет:
- CryptoPro extension;
- native messaging host;
- CSP Lite / библиотек CryptoPro;
- рабочего сценария подписи.

Сейчас уже есть:
- foundation launcher первого этапа;
- managed Chromium runtime второго этапа, который подготавливается в payload из pinned Chrome for Testing build;
- отдельный `user-data-dir` для запуска встроенного браузера;
- cache-friendly подготовка Chromium runtime в CI.

Следующий этап:
- CryptoPro extension;
- затем native messaging;
- затем crypto stack.

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
1. скачивает **стабильный опубликованный payload** по `build/payload-lock.json`
2. проверяет SHA-256 и размер скачанного payload
3. собирает `dist/KriptosferaDemo.exe` (embedded)
4. собирает `dist/KriptosferaDemo-remote.exe` (remote)
5. публикует один launcher artifact / release assets

`build-payload.yml`:
1. готовит payload (включая Chromium runtime)
2. упаковывает `dist/payload.zip` и `dist/payload.json`
3. публикует payload на сервер по SSH
4. публикует **один** payload artifact / release assets

### Почему payload больше не пересобирается при каждой сборке приложения

Payload тяжёлый и меняется редко. Поэтому launcher-сборка больше не тратит время на его пересборку.

Источником истины теперь служит `build/payload-lock.json`:
- там зафиксированы `payloadVersion`, `sha256`, `size`, `url`, `metadataUrl`
- пока payload не меняется, launcher всегда собирается против одного и того же стабильного payload
- если payload когда-нибудь обновляется, нужно обновить этот lock-файл на новый immutable URL/хеш

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
6. CryptoPro components + Рутокен.
7. Минимальная диагностика.
8. macOS PoC.

## Текущий следующий шаг

**Этап 4: переход к CryptoPro extension / native messaging.**

Что закрыто внутри этапа 3:
- выделен runtime/payload abstraction layer;
- добавлен remote runtime core (`RemotePayloadSource`, temp download, SHA-256 verify, cache reuse);
- добавлены build/runtime-config generation и immutable payload artifact layout;
- workflow уже собирает и embedded launcher, и thin launcher;
- для remote first-run добавлен minimal progress UX с маленьким progress window на Windows.

Что дальше:
- загрузка CryptoPro extension;
- native messaging host;
- затем интеграция crypto stack.

## Ближайшие инженерные задачи

- подготовить следующий этап MVP: CryptoPro extension;
- затем идти в native messaging и crypto stack;
- при необходимости позже вернуться к UX-polish progress окна и richer diagnostics.
