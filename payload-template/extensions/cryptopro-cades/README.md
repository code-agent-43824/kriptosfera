# CryptoPro CAdES extension compatibility pin

This directory currently contains the legacy CryptoPro CAdES extension
`1.2.13` with `manifest_version: 2`.

It is a temporary compatibility pin for the clean-machine Mini CSP path:

- CAdES Browser Plug-in `2.0.15000`
- CryptoPro extension `1.2.13` / Manifest V2
- Chrome for Testing `138.x` with `ExtensionManifestV2Availability=2`

Do not treat this as the long-term extension baseline. When CryptoPro provides a
fixed modern plug-in / extension combination, migrate this directory back to a
Manifest V3 extension and remove the Chrome 138 MV2 policy wiring.
