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

csp/
  linux/
    <csp-version>/
      <source-sha256>/
        manifest.json
        SHA256SUMS
        amd64/
          deb/
          rpm/
        arm64/
          deb/
          rpm/
        archives/

sources/
  linux/
    <csp-version>-<cades-version>/
      <source-sha256>/
        <source-archive>
```

Current scaffold URLs:

```text
https://mescheryakov.pro/kriptosfera/cryptopro/README.txt
https://mescheryakov.pro/kriptosfera/cryptopro/plugin/README.txt
https://mescheryakov.pro/kriptosfera/cryptopro/csp-lite/README.txt
```

Current CryptoPro CAdES Browser Plugin bundle:

```text
https://mescheryakov.pro/kriptosfera/cryptopro/plugin/2.0.15000/4590391e35c251cd4685d839ab62fad69e08716335931ac1c1b753b0cd346c6a/cryptopro-plugin.zip
https://mescheryakov.pro/kriptosfera/cryptopro/plugin/2.0.15000/4590391e35c251cd4685d839ab62fad69e08716335931ac1c1b753b0cd346c6a/cryptopro-plugin.json
```

This is a temporary legacy compatibility bundle extracted from the supplied
CryptoPro CAdES Browser Plug-in installer:

```text
https://mescheryakov.pro/kriptosfera/cryptopro/special/legacy-cades-2.0.1500-mv2/cadesplugin_2_0_1500.exe
```

Source installer mirror for audit/rebuild:

```text
https://mescheryakov.pro/kriptosfera/cryptopro/special/legacy-cades-2.0.1500-mv2/cadesplugin_2_0_1500.exe
https://mescheryakov.pro/kriptosfera/cryptopro/special/legacy-cades-2.0.1500-mv2/extension_1.2.13.crx
```

Current CryptoPro CSP Linux installer assets for cross-platform experiments:

```text
https://mescheryakov.pro/kriptosfera/cryptopro/sources/linux/5.0.13800-2.0.15700/6928220796ea0bbf36985b15bbf4f1d673c971c337833220ab6511fb6b481bc5/linux-amd64_all.zip
https://mescheryakov.pro/kriptosfera/cryptopro/csp/linux/5.0.13800/6928220796ea0bbf36985b15bbf4f1d673c971c337833220ab6511fb6b481bc5/manifest.json
https://mescheryakov.pro/kriptosfera/cryptopro/csp/linux/5.0.13800/6928220796ea0bbf36985b15bbf4f1d673c971c337833220ab6511fb6b481bc5/SHA256SUMS
```

Expanded Linux layout:

```text
cryptopro/csp/linux/5.0.13800/6928220796ea0bbf36985b15bbf4f1d673c971c337833220ab6511fb6b481bc5/
  amd64/deb/
  amd64/rpm/
  arm64/deb/
  arm64/rpm/
  archives/
  manifest.json
  SHA256SUMS
```

## Publication rules

- Publish each binary bundle under a directory containing its version and SHA-256.
- Treat published version/sha256 directories as immutable.
- Do not overwrite an existing archive in place.
- Do not commit CryptoPro binary archives to GitHub.
- Do not commit extracted CryptoPro installer packages to GitHub.
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

The static storage scaffold exists and is reachable over HTTPS. CryptoPro CAdES Browser Plugin 2.0.15000 has been uploaded as an immutable, checksum-addressed archive for the legacy MV2 / Chrome 138 compatibility profile. The previous 2.0.15700 archive remains published for audit/history, but the launcher lock no longer points to it.

Known bundle values:

- plugin archive: `cryptopro-plugin.zip`
- plugin archive size: `24052329`
- plugin archive SHA-256: `4590391e35c251cd4685d839ab62fad69e08716335931ac1c1b753b0cd346c6a`
- source installer mirror size: `11781256`
- source installer mirror SHA-256: `7c43d41482684ff3d98fe45c741c6a14b63055c88721f0207ab2b605dbc28cb2`
- source installer bootstrapper file version: `2.0.15002.0`
- Linux source archive: `linux-amd64_all.zip`
- Linux source archive size: `144025183`
- Linux source archive SHA-256: `6928220796ea0bbf36985b15bbf4f1d673c971c337833220ab6511fb6b481bc5`
- Linux CSP version: `5.0.13800`
- Linux CAdES/package source version marker: `2.0.15700`

The binary archives stay on the static server only. GitHub stores this documentation and the pinned lock file, not the CryptoPro binaries.
