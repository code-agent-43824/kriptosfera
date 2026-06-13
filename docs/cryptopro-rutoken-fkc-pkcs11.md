# Rutoken ЭЦП — active (PKCS#11) and FKC carriers for Mini CSP

Source of truth: the **Linux** CryptoPro CSP 5.0.13800 reader packages
(`cprocsp-rdr-rutoken`, `cprocsp-rdr-cpfkc`, `cprocsp-rdr-cryptoki`) extracted from
the pinned `linux-amd64_all.zip` mirror (see `docs/cryptopro-static-bundles.md`). The
carrier definitions live in each package's `postinst` (`cpconfig … -add …`), which is
the Linux equivalent of the Windows `config.ini` `KeyCarriers` / `KeyDevices` entries.

## What Mini CSP already has vs. what is missing

The bundled Mini CSP `config.ini` already defines the **passive** Rutoken carriers
(file keys on the token, read by CryptoPro itself) — all via `DLL = "rutoken.dll"`:
`Rutoken`, `RutokenECP` (ATR `3B 8B 01 …20 C1` = "Rutoken DS"), `RutokenECPM`,
`RutokenECPMSC`, `RutokenECPSC`. For a Rutoken ЭЦП that stores CryptoPro GOST keys as
files this **already works**.

Missing (the user's request):
- **FKC** (функциональный ключевой носитель — token computes GOST itself): Linux carrier
  `rutokenfkc` via `librdrcpfkc.so`.
- **PKCS#11 active** (CryptoPro talks to the token's own PKCS#11): Linux key device
  `cryptoki_rutoken` via `librdrcryptoki.so` + the vendor `librtpkcs11ecp.so`.

## ⚠️ Hard blocker: required DLLs are NOT in the Mini CSP bundle

The slim Mini CSP ships only:
`asn1*, bio, capi10, capi20, cpasn1, cpcspi, cplib, cpsuprt, cpui, dsrf, fat12,
jacarta, pcsc, rutoken, safenet`. It does **not** ship the reader DLLs these modes need.
Adding the config below **without** the DLLs does nothing (the carrier fails to load).

| Mode | Config DLL needed | Source | In Mini CSP? |
| --- | --- | --- | --- |
| FKC | `cpfkc.dll` (Linux `librdrcpfkc.so`) | full CryptoPro CSP for Windows | ❌ |
| PKCS#11 reader | `cryptoki.dll` (Linux `librdrcryptoki.so`) | full CryptoPro CSP for Windows | ❌ |
| PKCS#11 token lib | `rtPKCS11ECP.dll` (Linux `librtpkcs11ecp.so`) | **Rutoken drivers** (Aktiv), not CryptoPro | ❌ |

So enabling these is a **two-part change**: (1) add the DLLs next to the other Mini CSP
DLLs, (2) add the config below. Implemented 2026-06-13: the DLLs are pinned in
`build/rutoken-fkc-lock.json` and overlaid into the embedded Mini CSP archive at
build time. They are still never committed to Git. During implementation we found
that the current `2.0.15000` `config.ini` already contains `rutokenfkc` /
`rutokenfkc_nfc`; the overlay still ships `cpfkc.dll` and adds the missing
`cryptoki_rutoken` PKCS#11-active device when absent.

## Windows-adapted config fragment (drop into Mini CSP `config.ini`)

Adapted for Windows from the Linux `cpconfig` commands: `librdr<X>.so → <X>.dll`,
`librtpkcs11ecp.so → rtPKCS11ECP.dll`; ATR/mask bytes are 1:1; `-connect <name>` becomes
the `\<name>` subkey with `Name`; `KeyDevices` uses the quoted-value style already used by
the `PCSC` device. Append under the existing `[KeyCarriers]` / `[KeyDevices]` trees.

```ini
; --- FKC: Рутокен ЭЦП как функциональный ключевой носитель (cpfkc.dll required) ---
[KeyCarriers\rutokenfkc]
DLL = "cpfkc.dll"

[KeyCarriers\rutokenfkc\Default]
atr = hex: 3B,8B,01,52,75,74,6F,6B,65,6E,20,44,53,20,C1
mask = hex: FF,FF,FF,FF,FF,FF,FF,FF,FF,FF,FF,FF,FF,FF,FF
Name = "Rutoken FKC"

; Rutoken ЭЦП NFC (dual-interface) — optional, same FKC reader
[KeyCarriers\rutokenfkc_nfc]
DLL = "cpfkc.dll"

[KeyCarriers\rutokenfkc_nfc\Default]
atr = hex: 3B,88,80,01,52,74,53,43,77,81,83,20,6A
mask = hex: FF,FF,FF,FF,FF,FF,FF,FF,FF,FF,FF,FF,FF
Name = "Rutoken FKC NFC"

[KeyCarriers\rutokenfkc_nfc\Contact]
atr = hex: 3B,9C,96,80,11,40,52,75,74,6F,6B,65,6E,45,43,50,73,63,C0
mask = hex: FF,FF,FE,FF,FF,FF,FF,FF,FF,FF,FF,FF,FF,FF,FF,FF,FF,FF,FE
Name = "Rutoken FKC NFC"

; --- PKCS#11 active: Рутокен ЭЦП через собственную PKCS#11-библиотеку ---
; needs cryptoki.dll (CryptoPro reader) + rtPKCS11ECP.dll (Rutoken driver)
[KeyDevices\cryptoki_rutoken]
"DLL"="cryptoki.dll"
"Group"=1

[KeyDevices\cryptoki_rutoken\"PNP cryptoki"]
[KeyDevices\cryptoki_rutoken\"PNP cryptoki"\Default]
pkcs11_dll = "rtPKCS11ECP.dll"
```

Notes:
- The FKC `rutokenfkc` ATR (`3B 8B 01 …20 C1`) is **identical** to the passive
  `RutokenECP`. Same physical Rutoken ЭЦП; FKC vs passive is chosen by which reader DLL
  binds it. Keep both; CryptoPro picks the FKC carrier when the token is in active mode.
- `rtPKCS11ECP.dll` is the 32-bit Rutoken PKCS#11 library (Mini CSP host `nmcades.exe`
  is 32-bit, so use the x86 build).
- The Linux `cpfkc` postinst also defines `smartparkfkc` and (commented) `gemfkc`/`nxpfkc`
  — not Rutoken, intentionally omitted.

## Integration path in this repo

`config.ini` lives inside the (uncommitted) vendor bundle, so the real change is a
**build-time overlay** in `build/fetch-cryptopro-plugin.ps1` (where the slim archive is
assembled): inject the three DLLs (SHA-256-pinned) into `Mini CSP\` and append this
fragment to `Mini CSP\config.ini`. This overlay is now implemented. A Rutoken ЭЦП
smoke test (FKC + pkcs11) is still required — and remember the portable-provider
blocker in `docs/cryptopro-portable-plugin-findings.md` still gates the clean-machine
path.
