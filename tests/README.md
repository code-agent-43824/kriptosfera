# Tests

## Сейчас есть

- **Go unit tests** в `internal/bootstrap` (`*_test.go`) — покрывают подготовку
  и кэширование payload, защиту от zip path traversal, remote-загрузку
  (HTTPS-проверка, mismatch SHA-256, лимит размера), извлечение CryptoPro
  Browser Plugin, обнаружение расширений, генерацию native messaging manifest,
  сборку аргументов Chromium и валидацию app-config (включая `profileName`).

Запуск: `go test ./...` (работает на любой платформе; на не-Windows launcher
делает diagnostics dry-run вместо запуска браузера).

## Планируется

- PowerShell smoke-тесты для сценариев first run / second run.
- Playwright smoke для запуска Chromium и открытия стартового URL.
- Ручной сценарий с Рутокеном и тестовой подписью на демо-странице CryptoPro.

См. [`../docs/architecture.md`](../docs/architecture.md) про поток launcher и
[`../CONTRIBUTING.md`](../CONTRIBUTING.md) про локальную сборку.
