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

Runtime extraction note: the MSI extraction output contains pseudo-path entries such as `.:Common`. These names are valid as MSI table abstractions but invalid as Windows filesystem path components. The launcher skips archive entries with `:` in any path component.

The AppData layout is now validated for the native host, CAdES runtime, and core Mini CSP files:

```text
Program Files/Crypto Pro/CAdES Browser Plug-in/nmcades.exe
Program Files/Crypto Pro/CAdES Browser Plug-in/nmcades.json
Program Files/Crypto Pro/CAdES Browser Plug-in/npcades.dll
Program Files/Crypto Pro/CAdES Browser Plug-in/cades.dll
Program Files/Crypto Pro/CAdES Browser Plug-in/xades.dll
Program Files/Crypto Pro/CAdES Browser Plug-in/cplib.dll
Program Files/Crypto Pro/CAdES Browser Plug-in/Mini CSP/capi10.dll
Program Files/Crypto Pro/CAdES Browser Plug-in/Mini CSP/capi20.dll
Program Files/Crypto Pro/CAdES Browser Plug-in/Mini CSP/cpcspi.dll
Program Files/Crypto Pro/CAdES Browser Plug-in/Mini CSP/cpsuprt.dll
Program Files/Crypto Pro/CAdES Browser Plug-in/Mini CSP/cpui.dll
```

Token-specific DLLs such as `rutoken.dll`, `jacarta.dll`, `pcsc.dll`, and `safenet.dll` are present in the bundle and should be reported by runtime diagnostics, but they are not hard blockers yet because the first Mini CSP prototype must prove provider activation before token-specific behavior.

`cadescom.dll` is present in MSI pseudo-path locations (`.:Common` / `.:Common64`) rather than the current safe AppData Browser Plug-in layout. Do not make it a required extracted file until we decide how to map those MSI pseudo-paths intentionally.

Runtime diagnostics are written by the launcher to:

```text
<appDir>/diagnostics/cryptopro-runtime.json
```

The file captures the extracted plugin root, selected extension id, native messaging manifest/host paths, expected HKCU native messaging key, plugin bundle metadata, and SHA-256 values for the required CAdES/Mini CSP files. It is a read-only snapshot for the next two-machine comparison and does not attempt CSP activation.

## Manual runtime findings

Manual Windows validation on 2026-05-20 clarified the boundary of the current Browser Plugin bundle:

- On a machine with normal system CryptoPro CSP installed, Kriptosfera's bundled extension/native host/plugin layer behaves like regular configured Chrome:
  - extension loaded;
  - plugin loaded;
  - plugin version reported as `2.0.15700`;
  - CSP version reported as `5.0.13455`;
  - provider reported as `Crypto-Pro GOST R 34.10-2012 Cryptographic Service Provider`;
  - the standard CryptoPro access confirmation dialog appears;
  - approving the dialog allows certificate enumeration;
  - denying the dialog returns the expected user-cancelled error `0x000004C7`.
- On a clean machine without system CryptoPro CSP installed:
  - extension loaded;
  - plugin loaded;
  - CSP not loaded;
  - plugin version reported as `0.0.0000`;
  - no access confirmation dialog appears;
  - certificates are not enumerated.

Conclusion: current native messaging and Browser Plugin integration works. The remaining clean-machine gap is CSP/provider activation, likely through the bundled `Mini CSP` / future CSP Lite layer. The `0.0.0000` plugin version should be treated as a symptom of missing provider activation, not as a separate UI formatting bug.
