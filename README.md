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
- thin launcher build/runtime-config generation;
- immutable payload artifact layout для remote delivery;
- следующий оставшийся хвост этапа — minimal progress UX.

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
5. упаковка `dist/payload.zip` и генерация `dist/payload.json`
6. публикация immutable layout в `dist/published/payloads/win64/demo/<version>/<sha256>/...`
7. генерация runtime config для embedded launcher
8. `go test ./...` для embedded path
9. сборка `dist/KriptosferaDemo.exe`
10. генерация runtime config для remote launcher
11. сборка `dist/KriptosferaDemo-remote.exe` с build tag `remote`
12. публикация артефактов

### Важный момент про «без лишнего зазиповывания»

GitHub Actions workflow artifacts технически скачиваются GitHub'ом как zip-контейнер — это ограничение самой платформы.

Поэтому добавлен практичный обходной путь:
- обычный CI-run публикует workflow artifact;
- tag build (`v*`) дополнительно прикрепляет **сырой `KriptosferaDemo.exe`**, **`KriptosferaDemo-remote.exe`**, `payload.zip`, `payload.json` и `README.txt` как **GitHub Release assets**.

То есть для реального скачивания итоговых бинарников и payload без дополнительной упаковки нужно брать **release assets**, а не workflow artifact.

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
