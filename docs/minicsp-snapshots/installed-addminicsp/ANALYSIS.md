# Phase 3 analysis — reinstall with `ADDMINICSP=1`

Snapshot captured `2026-05-31T17:08:55Z` on `VIRTUALPC`, after uninstalling the
flagless plug-in (step 2.5) and reinstalling with
`cadesplugin.exe -cadesargs "ADDMINICSP=1"`. Diffed against phase 2
(`installed-noflags`) and the `clean` baseline. Raw evidence: `registry/`,
`files/` (incl. `minicsp-config.ini`, `minicsp-license.ini`), `summary.txt`.

## 1. `ADDMINICSP=1` adds NO registry keys (the decisive result)

Every registry dump is **byte-identical between phase 2 and phase 3** — verified
with `Compare-Object` over all six keys, including the two `Crypto Pro` keys:

```
IDENTICAL: hklm-cryptopro.txt          IDENTICAL: hklm-wow64-cryptopro.txt
IDENTICAL: hklm-capi-defaults.txt      IDENTICAL: hklm-capi-defaults-types.txt
IDENTICAL: hklm-wow64-capi-defaults.txt IDENTICAL: hklm-wow64-capi-types.txt
```

So the Mini CSP flag changes **only the filesystem**. The `Crypto Pro` branch is
exactly what the flagless install wrote (PKI `OCSPAPI`/`TSPAPI` `ProductID`s); no
CSP / provider key is added. This confirms the owner's observation and kills the
registry hypothesis for good — provider registration is **not** in the Windows
registry.

## 2. GOST providers are defined in `Mini CSP\config.ini`, not the registry

`HKLM\…\Cryptography\Defaults\Provider(+Types)` (native + WOW6432Node) are
**byte-identical to the clean phase** — no GOST provider was registered in the
Windows registry. Instead `Mini CSP\config.ini` carries CryptoPro's own
`Defaults\Provider` table (its config-as-registry abstraction):

| config.ini provider | `Image Path` | `Type` |
| --- | --- | --- |
| Crypto-Pro ECDSA and AES CSP | `cpcspi.dll` | 16 |
| Crypto-Pro Enhanced RSA and AES CSP | `cpcspi.dll` | 24 |
| Crypto-Pro GOST R 34.10-2001 Cryptographic Service Provider | `cpcspi.dll` | **75** |
| Crypto-Pro GOST R 34.10-2012 Cryptographic Service Provider | `cpcspi.dll` | **80** |
| Crypto-Pro GOST R 34.10-2012 Strong Cryptographic Service Provider | `cpcspi.dll` | **81** |

…with matching `[Defaults\"Provider Types"\"Type 075/080/081"]` sections (GOST R
34.10-2001 / 2012-256 / 2012-512). This is exactly the provider set that
`CSPName(80)` needs and that is missing on a clean machine.

## 3. `Mini CSP` folder — added in BOTH bitnesses

`ADDMINICSP=1` laid down two Mini CSP trees (full file lists + SHA-256 in
`files/pf-x86-cryptopro.txt`):

- **32-bit:** `C:\Program Files (x86)\Crypto Pro\CAdES Browser Plug-in\Mini CSP\`
  — `config.ini`, `capi20.dll` (x86), `cpcspi.dll` (x86), `capi10/cplib/cpasn1/
  asn1*/bio/fat12/dsrf/cpsuprt/cpui/cpconfig.exe`, token providers
  `rutoken.dll`/`jacarta.dll`/`safenet.dll`/`pcsc.dll`, `license.ini`.
- **64-bit (new):** `C:\Program Files\Crypto Pro\CAdES Browser Plug-in\Mini CSP\`
  — same set but 64-bit, with **`config64.ini`** instead of `config.ini`. The
  64-bit `CAdES Browser Plug-in` otherwise has no root binaries; only the
  `Mini CSP` subfolder was added (phase 2 had no `C:\Program Files\Crypto Pro`
  at all).

`config.ini` (32-bit) and `config64.ini` (64-bit) are **byte-identical**
(SHA-256 `405a209a…`), and `license.ini` is identical in both
(SHA-256 `771d551b…`). The committed `minicsp-config.ini` is the 32-bit
`config.ini`; the snapshot script matches `config.ini` by name, so `config64.ini`
is captured only via the listing (same hash, so no information lost).

## 4. Binary bitness

| Binary | Bitness |
| --- | --- |
| `CAdES Browser Plug-in\nmcades.exe` / `npcades.dll` / `cplib.dll` | x86 (32) |
| `…(x86)\…\Mini CSP\capi20.dll` / `cpcspi.dll` | x86 (32) |
| `…\Program Files\…\Mini CSP\capi20.dll` / `cpcspi.dll` | x64 (64) |

The native host is 32-bit, so it loads the **32-bit** `Mini CSP\capi20.dll` and
reads **`config.ini`** (not `config64.ini`) — consistent with the binary
analysis. The 64-bit tree is for 64-bit CAPI consumers.

## 5. `license.ini`

```
[v30ProductID\"{50F91F80-D397-437C-B0C8-62128DE3B55E}"]
ProductID = "5050EF301001FCZCT9M0HNAUV"
```

Matches the investigation's note (`npcades.dll` reads
`\local\license\ProductID\{50F91F80-…}`). Committed verbatim (expired /
non-secret, per the owner).

## Diff vs phase 2 — what `ADDMINICSP=1` added

- **Registry:** nothing (identical, proven above).
- **Disk:** the 32-bit `Mini CSP` subtree under the existing `Program Files (x86)`
  plug-in, **and** a new 64-bit `C:\Program Files\Crypto Pro\…\Mini CSP` tree
  (`config64.ini`). That is the entire delta.
- **Net:** Mini CSP activation is purely an on-disk concern (`Mini CSP\` next to
  the host, with `config.ini` + `license.ini`); the Windows registry is untouched.
