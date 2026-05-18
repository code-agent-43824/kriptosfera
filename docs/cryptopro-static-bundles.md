# CryptoPro static bundle storage

## Purpose

This static storage is the controlled source for CryptoPro binary bundles used by Kriptosfera builds.

CryptoPro binaries are intentionally not committed to GitHub and are not included in the remote Chromium payload. Build scripts will download version/sha256-pinned archives from this storage, verify checksum and size, then embed the verified archive into both launcher variants.

## Base URL

```text
https://mescheryakov.pro/kriptosfera/cryptopro/
```

Server path:

```text
/home/openclaw/sites/mescheryakov.pro/public/kriptosfera/cryptopro/
```

## Immutable layout

```text
plugin/
  <plugin-version>/
    <sha256>/
      cryptopro-plugin.zip
      cryptopro-plugin.json

csp-lite/
  <future-version>/
    <sha256>/
      cryptopro-csp-lite.zip
      cryptopro-csp-lite.json
```

Current scaffold URLs:

```text
https://mescheryakov.pro/kriptosfera/cryptopro/README.txt
https://mescheryakov.pro/kriptosfera/cryptopro/plugin/README.txt
https://mescheryakov.pro/kriptosfera/cryptopro/csp-lite/README.txt
```

Current CryptoPro CAdES Browser Plugin bundle:

```text
https://mescheryakov.pro/kriptosfera/cryptopro/plugin/2.0.15700/c35327b079022f123a8c31e5656891d61a7e493312010a0893b76a25f15feebe/cryptopro-plugin.zip
https://mescheryakov.pro/kriptosfera/cryptopro/plugin/2.0.15700/c35327b079022f123a8c31e5656891d61a7e493312010a0893b76a25f15feebe/cryptopro-plugin.json
```

The bundle was extracted from the official signed CryptoPro CADESCOM MSI:

```text
https://cryptopro.ru/sites/default/files/products/cades/current_release_2_0/cadescom-x64.msi
```

Source installer mirror for audit/rebuild:

```text
https://mescheryakov.pro/kriptosfera/cryptopro/sources/2.0.15700/8b4b1bfbe801c4569c3bf23107110263a344a9b1bce30c575b9b92ff77f2c2d4/cryptopro-cades-official-2.0.15700.zip
https://mescheryakov.pro/kriptosfera/cryptopro/sources/2.0.15700/8b4b1bfbe801c4569c3bf23107110263a344a9b1bce30c575b9b92ff77f2c2d4/cryptopro-cades-official.json
```

## Publication rules

- Publish each binary bundle under a directory containing its version and SHA-256.
- Treat published version/sha256 directories as immutable.
- Do not overwrite an existing archive in place.
- Do not commit CryptoPro binary archives to GitHub.
- Do not put CryptoPro binary archives into `payload.zip`.
- Keep only lock files, checksums, source notes, and build scripts in GitHub.
- The build must verify the archive checksum and size from a pinned lock file.
- The build must fail closed if the static archive is missing or checksum/size differs.
- Both thick and thin launcher variants must embed the same verified CryptoPro plugin bundle.

## Metadata shape

```json
{
  "component": "cryptopro-browser-plugin",
  "version": "2.x.x",
  "platform": "windows-amd64",
  "archive": "cryptopro-plugin.zip",
  "sha256": "<lowercase sha256>",
  "size": 12345678,
  "source": "CryptoPro official package; redistribution permitted by CryptoPro",
  "createdAt": "2026-05-18T00:00:00Z"
}
```

## Current status

The static storage scaffold exists and is reachable over HTTPS. CryptoPro CAdES Browser Plugin 2.0.15700 has been uploaded as an immutable, checksum-addressed archive.

Known bundle values:

- plugin archive: `cryptopro-plugin.zip`
- plugin archive size: `21699162`
- plugin archive SHA-256: `c35327b079022f123a8c31e5656891d61a7e493312010a0893b76a25f15feebe`
- source installer mirror size: `38472470`
- source installer mirror SHA-256: `8b4b1bfbe801c4569c3bf23107110263a344a9b1bce30c575b9b92ff77f2c2d4`
- official `cadescom-x64.msi` SHA-256: `9f73456b6db5d793947f31bf2faabae742cf80fe161b9a21e97035067dd8cf5e`

The binary archives stay on the static server only. GitHub stores this documentation and the pinned lock file, not the CryptoPro binaries.
