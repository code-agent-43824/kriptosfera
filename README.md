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

Сейчас в работе:
- первый рефакторинговый шаг под `remote payload mode`;
- выделение `RuntimeConfig.Payload`, `PayloadSource` и общего `PayloadManager`.

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

Workflow: `.github/workflows/build-windows.yml`

Что делает pipeline:
1. checkout
2. setup Go
3. восстановление cache Chromium runtime
4. подготовка payload (включая Chromium runtime)
5. упаковка payload в `internal/bootstrap/payload.zip`
6. `go test ./...`
7. сборка `dist/KriptosferaDemo.exe`
8. публикация артефактов

### Важный момент про «без лишнего зазиповывания»

GitHub Actions workflow artifacts технически скачиваются GitHub'ом как zip-контейнер — это ограничение самой платформы.

Поэтому добавлен практичный обходной путь:
- обычный CI-run публикует workflow artifact;
- tag build (`v*`) дополнительно прикрепляет **сырой `KriptosferaDemo.exe`** и `README.txt` как **GitHub Release assets**.

То есть для реального скачивания итогового бинарника без дополнительной упаковки нужно брать **release asset**, а не workflow artifact.

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

**Этап 3: подготовить каркас remote payload mode / thin launcher.**

Что делается сейчас:
- отделяется launcher runtime config от payload app config;
- вводится `PayloadSource` abstraction;
- общий extraction/state pipeline выносится в `PayloadManager`;
- текущий embedded flow переводится на новый каркас без изменения внешнего поведения.

Зачем это сейчас:
- это готовит правильное основание для thin launcher;
- не даёт размножить payload-логику перед добавлением сети;
- позволяет потом добавить remote downloader и publish flow отдельными маленькими коммитами.

## Ближайшие инженерные задачи

- довести первый рефакторинг `RuntimeConfig.Payload` / `PayloadSource` / `PayloadManager`;
- добавить `RemotePayloadSource` и remote download flow с SHA-256 verification;
- подготовить build split для embedded launcher и thin launcher;
- добавить immutable publish flow для `payload.zip` / `payload.json`;
- затем переходить к загрузке CryptoPro extension.
