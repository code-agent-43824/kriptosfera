# HANDOFF — add Rutoken ЭЦП FKC + PKCS#11(active) to the bundled Mini CSP

Audience: an agent that can modify the **payload / plugin bundle** and run the
Windows build. Goal: make the bundled Mini CSP recognize Rutoken ЭЦП in **FKC**
(функциональный ключевой носитель) and **PKCS#11 active** modes — not just the
passive file-key mode it already supports.

Status 2026-06-13: implemented in the repo as a build-time overlay. The pinned
static DLL lock is `build/rutoken-fkc-lock.json`; `build/fetch-cryptopro-plugin.ps1`
injects `cpfkc.dll` and `cryptoki.dll` into `CAdES Browser Plug-in\Mini CSP\`,
keeps a spare `Mini CSP\rtPKCS11ECP.dll` copy, also places
`rtPKCS11ECP.dll` beside `nmcades.exe` at
`CAdES Browser Plug-in\rtPKCS11ECP.dll`, and appends missing config fragments to
`config.ini`. The current `2.0.15000` config already had `rutokenfkc` /
`rutokenfkc_nfc`, so the implemented overlay adds the missing
`cryptoki_rutoken` PKCS#11-active device; `cryptoProPluginLayout` is now `4`.
Remaining work is Windows CI/log review and hardware smoke testing with Rutoken
ЭЦП in FKC and PKCS#11-active modes.

Prereq reading (don't re-derive): `docs/cryptopro-rutoken-fkc-pkcs11.md` has the full
analysis and the **ready, Windows-adapted `config.ini` fragment**. This file is the
*how to ship it* checklist.

## The whole task in one sentence

Drop the **32-bit** CryptoPro reader DLLs into the Mini CSP folder, place the
Rutoken PKCS#11 DLL beside `nmcades.exe` (plus a harmless spare Mini CSP copy),
and append the prepared carrier fragment to `Mini CSP\config.ini`,
fetched+SHA-pinned the same way as every other binary (never committed to Git).

## 1. Acquire the three DLLs (all x86 — host `nmcades.exe` is PE32)

| File | Purpose | Where to get it | Notes |
| --- | --- | --- | --- |
| `cpfkc.dll` | CryptoPro FKC reader (Linux `librdrcpfkc.so`) | **full CryptoPro CSP 5.0 for Windows** (x86) | owner has the CSP distribution |
| `cryptoki.dll` | CryptoPro PKCS#11 reader (Linux `librdrcryptoki.so`) | **full CryptoPro CSP 5.0 for Windows** (x86) | same source |
| `rtPKCS11ECP.dll` | Rutoken's own PKCS#11 library (Linux `librtpkcs11ecp.so`) | **"Драйверы Рутокен"** from the official Rutoken site (rutoken.ru) — **NOT CryptoPro** | take the **x86 (32-bit)** build; the x64 build will NOT load into the 32-bit host |

Get the exact Windows file names from a machine where the full CryptoPro CSP +
Rutoken drivers are installed (e.g. under `C:\Program Files (x86)\Crypto Pro\CSP\` and
the Rutoken driver install dir). The Linux `.so` files in our mirror are reference-only
(version/ATR confirmation) — they are not usable on Windows.

License: `rtPKCS11ECP.dll` is Aktiv-Soft's; `cpfkc.dll`/`cryptoki.dll` are CryptoPro's.
Confirm redistribution terms before hosting. Keep them out of Git either way.

## 2. Host + pin (follow `docs/cryptopro-static-bundles.md`)

- Upload each DLL to the project static storage under a version+sha256 path
  (immutable, never overwrite).
- Add a new lock file `build/rutoken-fkc-lock.json` listing, per DLL: `sha256`,
  `size`, `url`. Mirror the shape of `build/cryptopro-plugin-lock.json`.
- The build must fetch each DLL, verify sha256+size, and **fail closed** on mismatch.

## 3. Inject into the slim plugin bundle (build step)

The embedded bundle is assembled by **`build/fetch-cryptopro-plugin.ps1`** (it
downloads the vendor archive, verifies it, then writes the slim
`internal/bootstrap/cryptopro-plugin.zip` keeping only the `CAdES Browser Plug-in\...`
subtree). Extend that script, after slimming, to overlay:

1. **DLL placement matters** (verified against the bundle layout — `nmcades.exe` lives
   in `CAdES Browser Plug-in\`, the Mini CSP DLLs live in the `Mini CSP\` subfolder):
   - `cpfkc.dll`, `cryptoki.dll` → **`CAdES Browser Plug-in\Mini CSP\`**. These are the
     CSP **reader** DLLs named by the `DLL = "..."` config value; `cpcspi.dll`/`capi20.dll`
     load them relative to their own (Mini CSP) directory, exactly like `rutoken.dll` does.
   - `rtPKCS11ECP.dll` → **`CAdES Browser Plug-in\`** (next to `nmcades.exe`). It is
     loaded **by bare name** (`pkcs11_dll = "rtPKCS11ECP.dll"`), and a bare-name
     `LoadLibrary` searches the **process directory** = the `nmcades.exe` dir, **not**
     `Mini CSP\`. Keep the `Mini CSP\rtPKCS11ECP.dll` spare copy too; placing it
     in both dirs is harmless.
2. Append the fragment from `docs/cryptopro-rutoken-fkc-pkcs11.md` to
   `CAdES Browser Plug-in\Mini CSP\config.ini`.
   - **Encoding matters:** `config.ini` is **Windows-1251 (CP1251)**. Read it as cp1251,
     append the fragment (ASCII-only, so safe), write back as cp1251. Do **not** convert
     the file to UTF-8.
3. Keep the slim-archive guard tests happy (they reject `Program Files`, `Common*`,
   `.msi`, MSI pseudo-paths). Adding `Mini CSP\*.dll` and a top-level `rtPKCS11ECP.dll`
   is fine; just don't reintroduce a `Program Files` prefix.

Why NOT `[apppath]`: on Linux the `\config\apppath` map points names at absolute paths,
but our runtime dir is dynamic
(`%LOCALAPPDATA%\Kriptosfera\apps\demo\<ver>\Crypto Pro\CAdES Browser Plug-in\…`), so a
**static** absolute `[apppath]` entry can't be baked into the shipped `config.ini`.
Rely on DLL-search placement above instead. (If `[apppath]` is truly required, the
launcher would have to write it at runtime with the resolved path — only do that if
test shows bare-name loading fails.)

## 4. Force re-extraction + rebuild

- Bump the plugin layout version in `internal/bootstrap/cryptopro_plugin_manager.go`
  (`cryptoProPluginLayout`, currently `3` → `4`) so machines with an older extraction
  (without the new DLLs) re-extract instead of reusing.
- Optional: add the three DLLs to `requiredCryptoProPluginFiles` so the build/CI guard
  fails if the overlay was dropped. Update the corresponding test.
- Run `gofmt`, `go vet`, `go test ./...`, and `GOOS=windows GOARCH=amd64 go build ./...`
  (+ `-tags remote`). The Mini CSP/config.ini lives in the **embedded** plugin bundle, so
  **both** launcher variants get it from a fresh build — **no `payload-lock.json`
  re-pin is needed** (that lock is only for `payload.zip` = Chromium + extension +
  app-config).

## 5. Verify (needs real hardware)

On a Windows box with a **Rutoken ЭЦП** token, run a fresh launcher and on the
internal-csp demo page confirm: provider loads, the token's certificate **enumerates**,
and `SignCades` succeeds — once in **FKC** mode and once in **PKCS#11** mode. The passive
`RutokenECP` path already works and is the control.

## Gotchas / context

- The **clean-machine provider blocker** (`docs/cryptopro-portable-plugin-findings.md`:
  CryptoPro `GetModuleFileName(0x10000000)` bug) still gates no-MSI machines. This FKC/
  PKCS#11 work is **orthogonal** — testable now on an MSI-installed machine, and ready
  for when the vendor fix lands.
- ATR for `rutokenfkc` is identical to passive `RutokenECP` (`3B 8B 01 …20 C1`) — same
  token, mode chosen by reader DLL. Keep both carriers.
- Don't touch `capi20.dll`/`cpcspi.dll` (certified CSP). We only add reader DLLs + config.
