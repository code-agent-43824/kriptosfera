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

## Phase 2 — DONE (2026-06-24): inventory of the authoritative Program Files Mini CSP

Read-only inventory (full table in `docs/worklog.md`). Result:

- **All three reader DLLs are MISSING from disk** — `cpfkc.dll`, `cryptoki.dll`,
  `rtPKCS11ECP.dll` are not in the Program Files Mini CSP (nor next to `nmcades.exe`). The
  MSI plug-in ships only the documented slim set; it does **not** add the FKC/cryptoki
  readers.
- **FKC config is ALREADY present and correct** — `[KeyCarriers\rutokenfkc]`,
  `rutokenfkc_nfc`, `RutokenFkcOld` all exist and point at `DLL = "cpfkc.dll"`.
- **PKCS#11 config is absent** — no `[KeyDevices\cryptoki_rutoken]` section anywhere.
- Passive `rutoken.dll` + its carriers present (the working control).

**Verdict: hypothesis A (placement / missing-DLL). The B wall is NOT reached** — there is
a simpler sufficient cause than a Mini CSP feature gap: the reader DLLs were never on
disk. This also retires the old "version skew" theory for FKC: the file isn't the *wrong*
version, it is simply **absent**.

**Key consequence — FKC is now a clean single-variable experiment.** Its carrier config
is already correct, so dropping **only `cpfkc.dll`** (no config edit) either makes FKC
appear or it doesn't. Nothing else changes. That isolates the real question — *does this
Mini CSP honor the FKC carrier once its reader DLL is present?* — better than any other
step. So we run FKC first (Phase 4a), then PKCS#11 (Phase 4b).

---

## Phase 3 — token + DLL sourcing + proven-good reference (admin steps by the owner)

Ask the owner (admin-gated; you cannot do these). The full-CSP + Rutoken-driver installs
double as the **DLL source** for Phase 4, so do them before the edit:
1. Install **Rutoken drivers** (provides x86 `rtPKCS11ECP.dll`); insert the **Rutoken
   ЭЦП** token holding certs in all three formats (csp / pkcs11 / fkc).
2. Install the **full system CryptoPro CSP** (provides `cpfkc.dll` + `cryptoki.dll`, and
   is the working all-three-formats reference).

Then you capture, no admin needed to read:

**(a) Note the Mini CSP core version you're matching against** (cheap, already on disk):
```powershell
(Get-Item "C:\Program Files (x86)\Crypto Pro\CAdES Browser Plug-in\Mini CSP\cpcspi.dll").VersionInfo.FileVersion
```
Record it. The owner's position (CryptoPro keeps component interop stable across builds)
means we proceed regardless; this is only a fallback lead if a placed reader loads but
refuses to bind.

**(b) Positive control — prove the harness + token work in the passive path:**
```powershell
# with token inserted, trigger certificate enumeration on the internal-csp demo page,
# then snapshot:
C:\Tools\listdlls.exe -accepteula nmcades > C:\Tools\nmcades-dlls-token.txt
# expect rutoken.dll to now be loaded and the csp cert to enumerate — confirms the
# token, ATR match, and passive reader path are all alive before we add anything.
```

**(c) Derive the verified PKCS#11 config from the working system CSP** (don't rely only
on our hand-translation):
```powershell
reg export "HKLM\SOFTWARE\Crypto Pro\Cryptography\CurrentVersion" C:\Tools\sys-csp.reg /y
reg export "HKLM\SOFTWARE\WOW6432Node\Crypto Pro\Cryptography\CurrentVersion" C:\Tools\sys-csp-wow.reg /y
```
In the export, find the **working** `KeyDevices\…cryptoki…` device + its `\PNP …\Default`
`pkcs11_dll`, and translate that registry subtree 1:1 into `config.ini` section syntax for
Phase 4b. This replaces the last unverified piece (our `cryptoki_rutoken` was adapted from
Linux, never checked against a real Windows config). Also confirm the `rutokenfkc` carrier
config the Mini CSP already has matches the working registry's FKC carrier.

---

## Phase 4 — the decisive edit, run INCREMENTALLY (elevated shell; target = Program Files Mini CSP)

Target the **Program Files** Mini CSP (writing there needs the elevated shell the owner
granted; otherwise dictate the steps). Back up first:
`copy "<Program Files Mini CSP>\config.ini" config.ini.bak`.

Run 4a and 4b as **separate** experiments — don't change both variables at once.

### Phase 4a — FKC (single variable: just the DLL)

The FKC carrier config is already present (Phase 2). Add **only** the reader DLL:

1. Copy x86 `cpfkc.dll` (from the installed full CSP) → `<Program Files Mini CSP>\`.
   **No config edit.**
2. Fully restart the launcher (close Chrome + any lingering `nmcades.exe`), reopen the
   demo page, trigger enumeration with the token in **FKC mode**.
3. Re-snapshot: `C:\Tools\listdlls.exe -accepteula nmcades > C:\Tools\nmcades-dlls-fkc.txt`
   — expect `cpfkc.dll` to now be loaded, and the FKC certificate to enumerate.

**This is the pivotal test.** FKC appearing proves the Mini CSP *does* honor these carrier
sections once the reader DLL exists → hypothesis A confirmed, and B becomes unlikely for
PKCS#11 too. FKC *not* appearing despite `cpfkc.dll` being loaded → the (B) signal.

### Phase 4b — PKCS#11 active (needs both DLLs + config)

Only after 4a is conclusive:

1. Copy x86 `cryptoki.dll` (full CSP) → `<Program Files Mini CSP>\`; copy x86
   `rtPKCS11ECP.dll` (Rutoken drivers) → the `CAdES Browser Plug-in\` dir (next to
   `nmcades.exe`; bare-name load = process dir). If unsure, also drop a copy in `Mini CSP\`.
2. Append the `cryptoki_rutoken` device section — **preferably the version derived from
   the Phase 3(c) registry export**; the Linux-adapted fallback is below. Preserve
   **Windows-1251** encoding:

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
   ```

3. Fully restart, trigger enumeration with the token in **PKCS#11 mode**, re-snapshot:
   `C:\Tools\listdlls.exe -accepteula nmcades > C:\Tools\nmcades-dlls-pkcs11.txt`
   — expect `cryptoki.dll` + `rtPKCS11ECP.dll` loaded and the pkcs11 certificate to enumerate.

---

## Decision tree

Read primarily off **Phase 4a (FKC)** — it's the single-variable test:

- **FKC enumerates after dropping `cpfkc.dll`** → hypothesis **A** confirmed. Root cause
  was the missing reader DLL (vendor config was already correct). Mini CSP honors these
  carriers when the DLL exists. Then expect 4b (PKCS#11) to behave the same; finish it.
  Fix on our side: ship the reader DLLs into the Mini CSP that actually loads, and add
  `cryptoki_rutoken`. **No vendor report.** (Re-confirm later on the portable overlay path
  once the `GetModuleFileName` blocker is fixed — see the Phase-1 product note.)
- **`ListDLLs` shows `cpfkc.dll` is loaded but FKC still doesn't enumerate** → reader
  binds, device init fails. Capture a ProcMon trace of the load and compare the failing
  calls against the system-CSP gold trace (and reconsider the `cpcspi` version delta from
  Phase 3a) before concluding.
- **`cpfkc.dll` is present in the authoritative folder, the carrier config is correct
  (it is, per Phase 2), yet `ListDLLs` shows Mini CSP never even attempts to load
  `cpfkc.dll`** → hypothesis **B**: the Mini CSP core ignores these carrier/device
  sections → **genuine vendor defect, file the bug report** with: `ListDLLs` before/after,
  the `config.ini`-vs-registry diff, the gold trace, and the fact that the full CSP of the
  same line sees all three formats on the same token.

## Report back

Append findings to `docs/worklog.md` and summarize to the owner: the Phase 2 inventory
table, the config-vs-registry diff, the before/after `ListDLLs`, and which branch of the
decision tree we landed on. Commit docs straight to `main` with `[skip ci]`. Do **not**
commit any vendor binaries or `.reg`/trace dumps that contain them.
