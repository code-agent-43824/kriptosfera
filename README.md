# Kriptosfera

Криптосфера — концепт и MVP-каркас для десктопного приложения, которое поставляет специализированную Chromium-оболочку и российский клиентский криптостек в режиме «скачал один файл → запустил → вставил токен → работаешь».

## Что зафиксировано по документам

Из входных документов проекта следует такой базовый замысел:
- первый приоритет — Windows MVP;
- поставка пользователю как один `.exe` без wizard-установщика и без admin rights;
- внутри — launcher на Go, встроенный payload, Chromium runtime, отдельный профиль браузера, CryptoPro extension, native host и криптографические библиотеки;
- первый референсный сценарий — тестовая страница CryptoPro CAdES Browser Plug-in;
- критерий успеха MVP — успешная тестовая подпись с Рутокеном без системной установки CryptoPro CSP.

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

1. Single-file bootstrapper без Chromium.
2. Запуск встроенного Chromium runtime.
3. Загрузка CryptoPro extension.
4. Native messaging.
5. CryptoPro components + Рутокен.
6. Минимальная диагностика.
7. macOS PoC.

## Первый разумный шаг реализации MVP

**Этап 1: доказать single-file bootstrapper без Chromium.**

Почему именно он:
- это первый формальный этап из ТЗ;
- он отрезает сразу большой класс рисков по распаковке, versioning и layout каталогов;
- его можно сделать и стабилизировать без зависимости от лицензирования/доставки CryptoPro-компонентов;
- он даст основу, поверх которой уже удобно вешать Chromium, extension и native host.

Конкретный критерий готовности первого шага:
- один `KriptosferaDemo.exe`;
- при первом запуске тихо раскладывает payload в пользовательский каталог;
- при повторном запуске не распаковывает заново;
- пишет `launcher.log`;
- открывает тестовый внутренний ресурс или делает dry-run запуска runtime.

## Ближайшие инженерные задачи

- заменить заглушечный payload на versioned payload manifest;
- добавить checksum payload;
- зафиксировать layout `LOCALAPPDATA/Kriptosfera/...`;
- оформить smoke-test для first run / second run;
- затем подключать runtime Chromium.
