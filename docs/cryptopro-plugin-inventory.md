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

Current temporary legacy compatibility bundle:

```text
url: https://mescheryakov.pro/kriptosfera/cryptopro/plugin/2.0.15000/4590391e35c251cd4685d839ab62fad69e08716335931ac1c1b753b0cd346c6a/cryptopro-plugin.zip
sha256: 4590391e35c251cd4685d839ab62fad69e08716335931ac1c1b753b0cd346c6a
size: 24052329
metadata: https://mescheryakov.pro/kriptosfera/cryptopro/plugin/2.0.15000/4590391e35c251cd4685d839ab62fad69e08716335931ac1c1b753b0cd346c6a/cryptopro-plugin.json
```

Legacy source mirror for audit/rebuild:

```text
installer: https://mescheryakov.pro/kriptosfera/cryptopro/special/legacy-cades-2.0.1500-mv2/cadesplugin_2_0_1500.exe
installer sha256: 7c43d41482684ff3d98fe45c741c6a14b63055c88721f0207ab2b605dbc28cb2
installer size: 11781256
extension: https://mescheryakov.pro/kriptosfera/cryptopro/special/legacy-cades-2.0.1500-mv2/extension_1.2.13.crx
extension sha256: cf9bd5ce31d8ae6e50038dc742b4fd900a87c854cccb5db69a39976cccbf07c9
extension size: 70909
```

The `2.0.15000` bundle and its source mirror were restored on 2026-06-04 after a
static-storage cleanup left `build/cryptopro-plugin-lock.json` pointing at a 404.
Public re-download verification matched the lock file exactly.

Previous bundle retained for audit/history:

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

The build consumes `build/cryptopro-plugin-lock.json`, downloads the full static
`cryptopro-plugin.zip`, verifies its original size/SHA-256, then normalizes it
before embedding into launcher variants. The embedded archive keeps only the
current runtime subtree as `CAdES Browser Plug-in/...`.

The main runtime uncertainty remains whether the extracted AppData-only layout is enough for plugin detection, or whether CryptoPro components require additional COM/CSP/system registration. We should test this empirically after native messaging registration is implemented.

Runtime extraction note: the MSI extraction output contains pseudo-path entries such as `.:Common`. These names are valid as MSI table abstractions but invalid as Windows filesystem path components. The launcher skips archive entries with `:` in any path component.

The source archive keeps CryptoPro's normal 32-bit install subtree:
`Program Files/Crypto Pro/CAdES Browser Plug-in/`. Runtime layout version 3
strips the archive's top-level extraction folder and `Program Files` wrapper,
then places that subtree under the versioned AppData application directory as:

```text
<appDir>/Crypto Pro/CAdES Browser Plug-in/
```

The resulting Mini CSP path is:

```text
<appDir>/Crypto Pro/CAdES Browser Plug-in/Mini CSP/
```

The AppData layout is now validated for the native host, CAdES runtime, and core Mini CSP files:

```text
Crypto Pro/CAdES Browser Plug-in/nmcades.exe
Crypto Pro/CAdES Browser Plug-in/nmcades.json
Crypto Pro/CAdES Browser Plug-in/npcades.dll
Crypto Pro/CAdES Browser Plug-in/cades.dll
Crypto Pro/CAdES Browser Plug-in/xades.dll
Crypto Pro/CAdES Browser Plug-in/cplib.dll
Crypto Pro/CAdES Browser Plug-in/Mini CSP/capi10.dll
Crypto Pro/CAdES Browser Plug-in/Mini CSP/capi20.dll
Crypto Pro/CAdES Browser Plug-in/Mini CSP/cpcspi.dll
Crypto Pro/CAdES Browser Plug-in/Mini CSP/cpsuprt.dll
Crypto Pro/CAdES Browser Plug-in/Mini CSP/cpui.dll
```

Token-specific DLLs such as `rutoken.dll`, `jacarta.dll`, `pcsc.dll`, and `safenet.dll` are present in the bundle and should be reported by runtime diagnostics, but they are not hard blockers yet because the first Mini CSP prototype must prove provider activation before token-specific behavior.

`cadescom.dll` is present in MSI pseudo-path locations (`.:Common` / `.:Common64`) rather than the current safe AppData Browser Plug-in layout. Do not make it a required extracted file until we decide how to map those MSI pseudo-paths intentionally.

Runtime diagnostics are written by the launcher to:

```text
<appDir>/diagnostics/cryptopro-runtime.json
```

The file captures the extracted plugin root, selected extension id, native messaging manifest/host paths, expected HKCU native messaging key, plugin bundle metadata, and SHA-256 values for the required CAdES/Mini CSP files. It is a read-only snapshot for the next two-machine comparison and does not attempt CSP activation.

## 2.0.15000 bundle inventory and pruning notes

The restored `2.0.15000` archive has 112 files, `61,991,999` bytes raw and
`24,052,329` bytes as a zip. Top-level raw sizes:

```text
27.6 MB  Program Files/
11.8 MB  Program Files 64/
10.7 MB  Common64/
10.0 MB  Common/
0.94 MB  cadescom-x64.msi
0.91 MB  cadescom-win32.msi
0.11 MB  System64/
0.01 MB  Windows/
0       CommonAppData/
```

Current launcher integration uses the 32-bit native messaging host and browser
plug-in path under AppData:

```text
Crypto Pro/CAdES Browser Plug-in/
```

Important files in that path:

```text
nmcades.exe
nmcades.json
npcades.dll
cades.dll
xades.dll
asn1.dll
cpasn1.dll
cplib.dll
mydss.dll
Mini CSP/
```

`Mini CSP/` is about `10.0 MB` raw and includes `capi10.dll`, `capi20.dll`,
`cpcspi.dll`, `cpsuprt.dll`, `cpui.dll`, `cpconfig.exe`, `config.ini`, and token
modules such as `rutoken.dll`, `jacarta.dll`, `pcsc.dll`, and `safenet.dll`.

The launcher embed step now prunes everything outside the current 32-bit
Browser Plug-in runtime subtree. The full static archive remains immutable on
the project server for provenance and future repinning, but
`internal/bootstrap/cryptopro-plugin.zip` is a slim normalized archive.

Current slim archive inventory from the pinned `2.0.15000` source:

```text
61 files
27,618,881 bytes raw
~11,247,865 bytes zipped
```

Pruned from launcher embed:

```text
Program Files 64/
Common/
Common64/
CommonAppData/
System64/
Windows/
*.msi
MSI pseudo-path entries such as .:Common
```

Do **not** prune files inside `CAdES Browser Plug-in/` or `Mini CSP/` until a
fixed vendor build exists and can be checked on Windows. That subtree contains
the native host, plug-in DLLs, CAdES runtime libraries, Mini CSP provider chain,
and token modules. Future deeper pruning must use an explicit comparison on:

- machine with no system CryptoPro install;
- machine with MSI-installed plug-in/CSP for control;
- Rutoken certificate enumeration and `SignCades`;
- diagnostics file with all required runtime hashes present.

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
