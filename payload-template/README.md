# Payload template

Шаблон payload, из которого build-скрипты собирают реальный `payload.zip`.

## Уже в репозитории

- `config/app-config.json` — продуктовый конфиг (start URL, allowedOrigins,
  profileName, windowMode, diagnostics).
- `diagnostics/diagnostics.html` — hosted diagnostics page на официальном
  `cadesplugin_api.js`.
- `extensions/cryptopro-cades/` — canonical unpacked CryptoPro CAdES Browser
  extension (`1.3.17`) со стабильным id из `manifest.key`.

## Добавляется на этапе Windows-сборки (в Git не хранится)

- Chromium runtime (pinned Chrome for Testing) — готовится `build/prepare-chromium.ps1`.
- CryptoPro Browser Plugin bundle (native host `nmcades.exe`, `npcades.dll`,
  CSP Lite / Mini CSP) — скачивается и проверяется `build/fetch-cryptopro-plugin.ps1`
  по `build/cryptopro-plugin-lock.json`.

Итоговая раскладка собранного payload и runtime-каталога описана в
[`../docs/architecture.md`](../docs/architecture.md).
