# CryptoPro plugin inventory

## Scope

This file records the first binary inventory for the CryptoPro CAdES Browser Plugin bundle used by Kriptosfera.

No CryptoPro binaries are stored in Git. The binaries are stored only on the project static server and referenced from `build/cryptopro-plugin-lock.json`.

## Official source files

Downloaded from CryptoPro's public `current_release_2_0` directory:

```text
https://cryptopro.ru/sites/default/files/products/cades/current_release_2_0/cadesplugin.exe
https://cryptopro.ru/sites/default/files/products/cades/current_release_2_0/cadescom-x64.msi
https://cryptopro.ru/sites/default/files/products/cades/current_release_2_0/cades-x64.msi
https://cryptopro.ru/sites/default/files/products/cades/current_release_2_0/cades-win32.msi
```

SHA-256:

```text
b33c7d84c842fb97e6c7c13dd46a49242031b5c52539ec6f90d759282749b0c7  cadesplugin.exe
9f73456b6db5d793947f31bf2faabae742cf80fe161b9a21e97035067dd8cf5e  cadescom-x64.msi
54e03480024f5b74a5ba785b41b529f9e073a2e6605743d5688820e59eee56cb  cades-x64.msi
30b308f48cc03096ece4ce2584328ed72d478face0ce92babb8ce37a8a2f3226  cades-win32.msi
```

Signature verification with `osslsigncode` reported `Signature verification: ok` for all four files. The signer certificate subject is `CRYPTO-PRO LLC`, issued by `GlobalSign GCC R45 EV CodeSigning CA 2020`.

## Published static bundles

Plugin runtime bundle:

```text
url: https://mescheryakov.pro/kriptosfera/cryptopro/plugin/2.0.15700/c35327b079022f123a8c31e5656891d61a7e493312010a0893b76a25f15feebe/cryptopro-plugin.zip
sha256: c35327b079022f123a8c31e5656891d61a7e493312010a0893b76a25f15feebe
size: 21699162
```

Official installer mirror for audit/rebuild:

```text
url: https://mescheryakov.pro/kriptosfera/cryptopro/sources/2.0.15700/8b4b1bfbe801c4569c3bf23107110263a344a9b1bce30c575b9b92ff77f2c2d4/cryptopro-cades-official-2.0.15700.zip
sha256: 8b4b1bfbe801c4569c3bf23107110263a344a9b1bce30c575b9b92ff77f2c2d4
size: 38472470
```

## Extracted MSI facts

Primary extraction source:

```text
cadescom-x64.msi
Title: Installation Database for CryptoPro CADESCOM
Subject: PKIpro2: CryptoPro CADESCOM
Comments: 2.0.15700
Template: AMD64;1049
```

Relevant files inside the extracted MSI:

```text
Program Files/Crypto Pro/CAdES Browser Plug-in/nmcades.exe
Program Files/Crypto Pro/CAdES Browser Plug-in/nmcades.json
Program Files/Crypto Pro/CAdES Browser Plug-in/nmcades_firefox.json
Program Files/Crypto Pro/CAdES Browser Plug-in/npcades.dll
Program Files/Crypto Pro/CAdES Browser Plug-in/Mini CSP/
```

Relevant registry rows from MSI inventory:

```text
SOFTWARE\Google\Chrome\NativeMessagingHosts\ru.cryptopro.nmcades
SOFTWARE\Mozilla\NativeMessagingHosts\ru.cryptopro.nmcades
```

The Chrome native messaging manifest template uses:

```json
{
  "name": "ru.cryptopro.nmcades",
  "description": "Chrome and Opera Native Messaging Host for CAdES Browser plug-in",
  "path": "<HOST_PATH>",
  "type": "stdio",
  "allowed_origins": [
    "chrome-extension://iifchhfnnmpdbibifmljnfjhpififfog/",
    "chrome-extension://epebfcehmdedogndhlcacafjaacknbcm/",
    "chrome-extension://pfhgbfnnjiafkhfdkmpiflachepdcjod/"
  ]
}
```

For MVP, the launcher should generate or patch the manifest at runtime so `path` points to the extracted AppData copy of `nmcades.exe`.

## Current conclusion

The next implementation step can consume `build/cryptopro-plugin-lock.json`, download `cryptopro-plugin.zip`, verify size/SHA-256, and embed the verified archive into both launcher variants.

The main runtime uncertainty remains whether the extracted AppData-only layout is enough for plugin detection, or whether CryptoPro components require additional COM/CSP/system registration. We should test this empirically after native messaging registration is implemented.

Runtime extraction note: the MSI extraction output contains pseudo-path entries such as `.:Common`. These names are valid as MSI table abstractions but invalid as Windows filesystem path components. The launcher skips archive entries with `:` in any path component and then validates the required native host files (`nmcades.exe`, `nmcades.json`, `npcades.dll`) in the AppData layout.
