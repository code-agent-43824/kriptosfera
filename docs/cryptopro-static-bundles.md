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

The static storage scaffold exists and is reachable over HTTPS. No CryptoPro binary archive has been uploaded yet.

