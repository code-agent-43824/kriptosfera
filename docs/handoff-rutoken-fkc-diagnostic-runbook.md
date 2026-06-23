# RUNBOOK — diagnose why Mini CSP shows only `csp` carriers (FKC / PKCS#11 invisible)

Audience: a **Claude Code session running on the owner's Windows box** (no branches —
work on `main`), driving the already-running Kriptosfera launcher. Goal: determine, with
evidence, **why the bundled Mini CSP enumerates only the passive `csp` certificate
format** and not the **FKC** and **PKCS#11-active** formats of a Rutoken ЭЦП — when the
full **system** CryptoPro CSP on the same machine sees all three.

You are deciding between two hypotheses:

- **(A) Config / placement problem on our side** — the FKC/PKCS#11 carrier entries (or
  the reader DLLs) are missing, malformed, or in a Mini CSP folder the runtime never
  loads. Fixable by us.
- **(B) Mini CSP feature gap** — the shipped Mini CSP core does not implement these
  reader devices, no matter how correct the config is. → vendor bug report.

## ⛔ Guardrail boundary — stay inside it

Interop **configuration + passive observation** of licensed software on the owner's own
machine. That is fine.

- ✅ Allowed: `ListDLLs` / `Process Monitor` / `reg export`, read `config.ini`, `dir`
  vendor folders, copy vendor DLLs the machine already has, append carrier sections to
  `config.ini` via CryptoPro's **own documented** mechanism, restart the launcher,
  observe the demo page.
- ⛔ Do **NOT**: disassemble, byte-patch, or otherwise modify any CryptoPro/Rutoken
  binary; defeat licensing/anti-tamper; bypass the user-confirmation dialog by any hack.
  If diagnosis ever appears to *require* opening/modifying a vendor binary, **stop** —
  that is the signal the road ends on our side and the result is a vendor bug report,
  not a workaround. Report back instead of proceeding.

## Reference material (don't re-derive)

- Carrier config fragment + DLL roles: `docs/cryptopro-rutoken-fkc-pkcs11.md`. The
  `2.0.15000` vendor `config.ini` already contains `rutokenfkc` / `rutokenfkc_nfc`; only
  the PKCS#11 device `cryptoki_rutoken` is documented as missing — **verify this against
  the real Program Files `config.ini` in Phase 2, don't assume.**
- The crypto runs inside `nmcades.exe` (PE32/x86), spawned by Chrome.
- The Windows full CSP keeps carriers/devices/readers in the registry
  (`HKLM\SOFTWARE\Crypto Pro\Cryptography\CurrentVersion\…`); Mini CSP mirrors that tree
  in `config.ini` (`[KeyDevices\…]` = a registry subkey). So the working system CSP's
  registry is the proven Windows ground-truth to compare our `config.ini` against.

---

## Phase 1 — DONE (2026-06-24): which Mini CSP actually loads

`ListDLLs` of the live `nmcades.exe` (launched from our overlay) showed a **split load
path**:

- **From our overlay** (`%LOCALAPPDATA%\Kriptosfera\…\CAdES Browser Plug-in\`):
  `nmcades.exe`, `npcades.dll`, CAdES runtime, and Mini CSP **helper** DLLs
  (`Mini CSP\capi20.dll`, `asn1*.dll`, `cpsuprt.dll`) — direct process imports, resolved
  from the process dir.
- **From `C:\Program Files (x86)\Crypto Pro\CAdES Browser Plug-in\Mini CSP\`**: the
  actual CSP provider core `cpcspi.dll` (+ `capi10.dll`, a second `capi20.dll`,
  `cpsuprt.dll`, `pcsc.dll`). `cpcspi` is pulled in via the **HKLM MSI provider
  registration**, not by process-dir search, and reads `config.ini` relative to **its
  own** directory.

**Conclusion:** the **authoritative Mini CSP = `C:\Program Files (x86)\Crypto Pro\CAdES
Browser Plug-in\Mini CSP\`**. Our `%LOCALAPPDATA%` overlay copy of `cpcspi` + `config.ini`
is **dead weight** on this machine — which directly explains the prior failed manual
edits (the edited copies were never the loaded ones). The three reader DLLs
(`cpfkc`/`cryptoki`/`rtPKCS11ECP`) were absent (expected pre-fix), and `rutoken.dll`
was not yet loaded because no token op had been triggered at capture time.

> **Product-architecture note (not part of the bug, but record it):** when the plug-in is
> MSI-installed, the HKLM-registered Program Files Mini CSP wins; our portable overlay
> provider is bypassed entirely. This reinforces the planned **two-mode** behavior (ride
> the installed CSP when present; only activate our overlay on a clean machine). Whatever
> Phase 4 proves about FKC here is proven about the **MSI** Mini CSP; re-confirm on the
> overlay path once the portable blocker is fixed.

---

## Phase 2 — NEXT: read-only inventory of the authoritative Mini CSP (no admin, no token)

Do this **now** — it's free and may already decide A vs B. Just read the Program Files
Mini CSP (reading needs no admin; only writing does).

```powershell
$mini = "C:\Program Files (x86)\Crypto Pro\CAdES Browser Plug-in\Mini CSP"

# (a) Which reader DLLs are already present?
Get-ChildItem $mini -Filter *.dll | Select-Object Name, Length,
  @{n='Ver';e={(Get-Item $_.FullName).VersionInfo.FileVersion}} | Sort-Object Name
# look specifically for: cpfkc.dll, cryptoki.dll, rutoken.dll, rtPKCS11ECP.dll

# (b) Which carrier/device sections does its config.ini already define?
$enc  = [System.Text.Encoding]::GetEncoding(1251)
$cfg  = Join-Path $mini 'config.ini'
$text = [System.IO.File]::ReadAllText($cfg, $enc)
# dump just the section headers:
[regex]::Matches($text, '(?m)^\s*\[(KeyCarriers|KeyDevices)\\[^\]]+\]') |
  ForEach-Object { $_.Value }
# look specifically for: [KeyCarriers\rutokenfkc...], [KeyDevices\cryptoki_rutoken]
```

Record the answers to these four questions — they drive the next step:

| Present in Program Files Mini CSP? | |
| --- | --- |
| `cpfkc.dll` (FKC reader) | ? |
| `cryptoki.dll` (PKCS#11 reader) | ? |
| `rtPKCS11ECP.dll` (Rutoken PKCS#11 lib) | ? |
| `[KeyCarriers\rutokenfkc]` section in config.ini | ? |
| `[KeyDevices\cryptoki_rutoken]` section in config.ini | ? |

**Pre-diagnosis from the inventory (before touching anything):**
- If `rutokenfkc` IS in config.ini **and** `cpfkc.dll` IS present, yet FKC certs are
  invisible → strong lean toward **(B)** (or an ATR/token-mode mismatch — note the token
  model). This is the most informative outcome and worth flagging immediately.
- If `cpfkc.dll` / `cryptoki.dll` / the `cryptoki_rutoken` section are simply **absent**
  → consistent with **(A)**; proceed to Phase 4 to add them and re-test.

---

## Phase 3 — token + control + proven-good reference (admin steps by the owner)

Ask the owner (admin-gated; you cannot do these):
1. Install **Rutoken drivers**; insert the **Rutoken ЭЦП** token holding certs in all
   three formats (csp / pkcs11 / fkc).
2. Install the **full system CryptoPro CSP** (the one that already sees all three).

Then you capture, no admin needed to read:

**(a) Positive control — prove the harness + token work in the passive path:**
```powershell
# with token inserted, trigger certificate enumeration on the internal-csp demo page,
# then snapshot again:
C:\Tools\listdlls.exe -accepteula nmcades > C:\Tools\nmcades-dlls-token.txt
# expect rutoken.dll to now be loaded and the csp cert to enumerate — confirms the
# token, ATR match, and passive reader path are all alive.
```

**(b) Ground-truth config from the working system CSP:**
```powershell
reg export "HKLM\SOFTWARE\Crypto Pro\Cryptography\CurrentVersion" C:\Tools\sys-csp.reg /y
reg export "HKLM\SOFTWARE\WOW6432Node\Crypto Pro\Cryptography\CurrentVersion" C:\Tools\sys-csp-wow.reg /y
```
Locate the working `KeyDevices\cryptoki...`, `KeyCarriers\rutokenfkc...` and reader
subkeys; diff them section-for-section against the **Program Files** `config.ini` from
Phase 2. Note every difference (key names, `Group`, `DLL`, `pkcs11_dll`, ATR/mask).

**(c) Optional gold trace** — if you can drive a working all-three-formats enumeration
through the system CSP (e.g. its own tooling / a system-CSP browser session), `ListDLLs`
that process to see exactly which reader DLLs it loads, from where, and their versions.
That is the direct "working trace" to compare against the failing Mini CSP one.

---

## Phase 4 — the decisive edit (elevated shell; target = Program Files Mini CSP)

Only add what Phase 2 showed missing. Target the **Program Files** Mini CSP (writing
there needs the elevated shell the owner granted; otherwise dictate the steps).

1. **Back up first:** copy `<Program Files Mini CSP>\config.ini` → `config.ini.bak`.
2. **Place the missing x86 reader DLLs**, sourced locally (not downloaded): `cpfkc.dll`
   / `cryptoki.dll` from the installed full CryptoPro CSP (`…\Crypto Pro\CSP\`);
   `rtPKCS11ECP.dll` (32-bit) from the installed Rutoken drivers.
   - `cpfkc.dll`, `cryptoki.dll` → `<Program Files Mini CSP>\`
   - `rtPKCS11ECP.dll` → the `CAdES Browser Plug-in\` dir (next to `nmcades.exe`, bare-name
     load = process dir). If unsure, also drop a copy in `Mini CSP\` (harmless).
3. **Append the missing config sections** (only those absent per Phase 2), preserving
   **Windows-1251** encoding. Idempotent append of the PKCS#11 device:

   ```powershell
   $enc = [System.Text.Encoding]::GetEncoding(1251)
   $cfg = "C:\Program Files (x86)\Crypto Pro\CAdES Browser Plug-in\Mini CSP\config.ini"
   $text = [System.IO.File]::ReadAllText($cfg, $enc)
   $pkcs11 = @"

[KeyDevices\cryptoki_rutoken]
"DLL"="cryptoki.dll"
"Group"=1
[KeyDevices\cryptoki_rutoken\"PNP cryptoki"]
[KeyDevices\cryptoki_rutoken\"PNP cryptoki"\Default]
pkcs11_dll = "rtPKCS11ECP.dll"
"@
   if ($text -notmatch '\[KeyDevices\\cryptoki_rutoken\]') {
     [System.IO.File]::WriteAllText($cfg, $text + "`r`n" + $pkcs11, $enc)
   }
   # If Phase 2 showed rutokenfkc is ALSO absent, append the FKC block from
   # docs/cryptopro-rutoken-fkc-pkcs11.md the same way. If it was PRESENT and FKC is
   # still invisible after adding cpfkc.dll, that is the (B) signal — see decision tree.
   ```

4. **Fully restart** the launcher (close Chrome + any lingering `nmcades.exe`), reopen
   the demo page, re-trigger enumeration with the token in.
5. **Re-snapshot** to confirm the readers now bind:
   ```powershell
   C:\Tools\listdlls.exe -accepteula nmcades > C:\Tools\nmcades-dlls-after.txt
   # expect cpfkc.dll / cryptoki.dll / rtPKCS11ECP.dll to now appear, with full paths
   ```

---

## Decision tree

- **FKC / PKCS#11 certs now enumerate** → hypothesis **A**. Root cause was
  placement/missing entries. Fix on our side: make the launcher overlay actually be the
  loaded provider (depends on the portable blocker) and ship `cryptoki_rutoken` + the
  reader DLLs. **No vendor report.**
- **Reader DLLs are loaded (`ListDLLs` confirms) but certs still don't enumerate** →
  reader binds, device init fails. Capture a ProcMon trace of the load and compare the
  failing calls against the system-CSP gold trace before concluding.
- **`config.ini` matches the working system-CSP registry 1:1, the reader DLLs are
  present in the authoritative folder, yet `ListDLLs` shows Mini CSP never even loads
  `cpfkc.dll` / `cryptoki.dll`** → hypothesis **B**: the Mini CSP core ignores these
  device sections → **genuine vendor defect, file the bug report** with: `ListDLLs`
  before/after, the `config.ini`-vs-registry diff, the gold trace, and the fact that the
  full CSP of the same line sees all three formats on the same token.

## Report back

Append findings to `docs/worklog.md` and summarize to the owner: the Phase 2 inventory
table, the config-vs-registry diff, the before/after `ListDLLs`, and which branch of the
decision tree we landed on. Commit docs straight to `main` with `[skip ci]`. Do **not**
commit any vendor binaries or `.reg`/trace dumps that contain them.
