# Linux enumeration control + the Windows architecture difference

Goal: test, by experiment, the hypothesis that "provider enumeration should work
from `config.ini` alone, with no registry — like CryptoPro CSP on Linux." Run in
the web-session Linux sandbox with CryptoPro CSP Lite (`lsb-cprocsp-capilite-64`)
+ CAdES installed; there is **no registry** on Linux at all.

## Experiment

`enumprov.c` (in this directory) calls exactly the CryptoAPI functions the
browser plugin / `npcades` use to enumerate providers:
`CryptEnumProviderTypes`, `CryptEnumProviders`, `CryptGetDefaultProvider`. Built
against CryptoPro's `libcapi10`/`libcapi20` headers and run three ways.

### Result 1 — with `config64.ini` present (`output-with-config.txt`)

All providers enumerate, by the same CryptoAPI calls the plugin uses:

- `CryptEnumProviderTypes` → types **75 / 80 / 81** (+ 1/16/24/32) with names.
- `CryptEnumProviders` → `Crypto-Pro GOST R 34.10-2012 KC1 CSP`, etc.
- `CryptGetDefaultProvider(80)` → `Crypto-Pro GOST R 34.10-2012 KC1 CSP`.

### Result 2 — `strace` proves the source (`strace-config-open.txt`)

The only config the calls open is:

```
openat(AT_FDCWD, "/etc/opt/cprocsp/config64.ini", O_RDONLY) = 3
```

No registry (there is none on Linux), no other config. **Providers come straight
from `config64.ini`.**

### Result 3 — control: remove the config (`output-without-config.txt`)

With `config64.ini` temporarily renamed away:

- `CryptEnumProviderTypes` → fails immediately (`0x80090020`).
- `CryptGetDefaultProvider(80)` → **`0x80090017`** — the exact error we see on
  Windows (`NTE_PROV_TYPE_NOT_DEF`).

So on Linux, `0x80090017` is precisely "the CryptoPro CryptoAPI layer has no
provider config". The config restored, enumeration returns.

**Conclusion (Linux):** the hypothesis is correct *on Linux* — enumeration is a
pure `config.ini` operation, registry-free, and a missing/unreadable config
yields the same `0x80090017`. `CryptAcquireContext` to type 80 also succeeds
(`PP_NAME = Crypto-Pro GOST R 34.10-2012 KC1 CSP`); only ephemeral key-gen fails
in this headless sandbox (no RNG/token), which is unrelated to enumeration.

## The decisive difference: Windows base CryptoAPI is Microsoft's, not CryptoPro's

Static export analysis of the **Windows** Mini CSP binaries (from our bundle)
explains why the Linux result does **not** transfer to Windows as-is:

- **`Mini CSP\cpcspi.dll`** exports the CSP SPI — `CPAcquireContext`, `CPGenKey`,
  `CPSignHash`, `CPGetProvParam`, … (the GOST engine itself).
- **`Mini CSP\capi20.dll`** exports only the *high-level* CryptoAPI2 surface —
  `Cert*`, `CryptMsg*`, `CryptAcquireCertificatePrivateKey`,
  `CryptFindCertificateKeyProvInfo` — and **does NOT export** the base provider
  functions `CryptAcquireContext`, `CryptEnumProviders`,
  `CryptEnumProviderTypes`, `CryptGetDefaultProvider`.

On **Linux** those base functions live in CryptoPro's own `libcapi20` (there is
no OS CryptoAPI), so every call routes through CryptoPro and reads `config.ini`.
On **Windows** those base functions are **Microsoft's `advapi32.dll`**, which
resolves providers from the **registry** (`HKLM\…\Cryptography\Defaults\
Provider[ Types]`). The Mini CSP is *not* registered there (proven in
`docs/minicsp-snapshots/` — `ADDMINICSP=1` writes nothing to the registry).

Therefore, on Windows:

- `About.CSPName(80)` / `CSPVersion("",80)` ultimately call
  `CryptGetDefaultProvider` / `CryptEnumProviderTypes` via **advapi32 → registry
  → empty → `0x80090017`**. This is **expected by architecture**, not a config
  path bug, and it matches the observed Windows diagnostics exactly.
- The Mini CSP is still usable for actual crypto **in-process**: `npcades` loads
  `Mini CSP\capi20.dll`, which reads `config.ini` and loads `cpcspi.dll`, and the
  plugin calls the **CSP SPI (`CPAcquireContext`/`CPSignHash`) directly**,
  bypassing advapi32 and the registry.

### What this means for the plan

- **Enumeration via `About.CSPName(80)` is a dead end on Windows for an
  unregistered Mini CSP** — by OS architecture. It will read `0x80090017` even on
  the official `ADDMINICSP=1` install (confirmed). It is the **wrong success
  signal**.
- The signal that matters is whether the **in-process** path works:
  `CAdESCOM.Store` → certificate with a Mini-CSP container → `SignCades` routed
  through `npcades → cpcspi`. That needs a real GOST token to test.
- A Windows reverse-test (`tools/windows/reverse-enum-test.ps1`) demonstrates the
  split directly: CryptoPro's own `cpconfig.exe` (config-based) enumerates the
  providers from `Mini CSP\config.ini`, while `certutil -csplist` and an
  `advapi32` `CryptGetDefaultProvider(80)` (registry-based) do not. See
  `docs/handoff-windows-reverse-enum.md`.

## Caveat

This still leaves one open question the reverse-test should settle: whether the
plugin's higher-level calls (e.g. opening a Mini-CSP container, signing) are
routed by `npcades` directly into the loaded `cpcspi.dll` (works without
registry) or whether some path still goes through advapi32 (needs registration).
The token sign test is the decisive end-to-end check.
