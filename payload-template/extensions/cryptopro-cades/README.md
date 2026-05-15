# CryptoPro CAdES extension payload

Канонический unpacked payload для этапа `v0.4`.

Источник:
- Chrome Web Store: `Extension for CAdES Browser Plugin`
- Extension id: `pfhgbfnnjiafkhfdkmpiflachepdcjod`
- Version: `1.3.17`
- Web Store URL: `https://chromewebstore.google.com/detail/extension-for-cades-brows/pfhgbfnnjiafkhfdkmpiflachepdcjod`
- CRX download URL pattern: `https://clients2.google.com/service/update2/crx?...id=pfhgbfnnjiafkhfdkmpiflachepdcjod...`

Примечания:
- В payload кладём unpacked extension целиком, чтобы launcher мог загружать его через `--disable-extensions-except` и `--load-extension`.
- `manifest.json` содержит `key`, поэтому extension id стабилен и может использоваться в diagnostics probe.
- Этот слой проверяет только доставку и загрузку browser extension. Native messaging, CryptoPro plugin/CSP и реальная подпись идут следующими этапами.
