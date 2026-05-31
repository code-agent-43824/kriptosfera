# Handoff: Windows snapshots of CryptoPro Mini CSP install states

Adressee: a Claude Code instance running in a **local session on a clean
Windows machine**. You have `git` and a `GH_TOKEN` for pushing. Author: the
diagnostic agent from the web session (no Windows/SSH there).

Goal: capture **factual** before/after snapshots of the registry and
`Program Files (x86)\Crypto Pro` across three install states, analyze each, and
commit the evidence. We are settling — by observation, not guesswork — how the
CryptoPro plugin and its Mini CSP register providers, and what `ADDMINICSP=1`
actually changes.

## Why this matters (context in one paragraph)

On a clean machine without system CryptoPro CSP, the plugin loads but providers
are not enumerated (`About.CSPName(80)` → `0x80090017`), so signing pages can't
proceed. Binary analysis of the bundle (see
`docs/cryptopro-csp-lite-plan.md` → "Ground truth from binary analysis") showed
Mini CSP activation is **module-relative and flag-gated** (`npcades.dll` reads
`cadesplugin.EnableInternalCSP` and `LoadLibraryEx`-loads `Mini CSP\capi20.dll`),
and that the owner observed **no `Crypto Pro` registry branch at all** on a real
`ADDMINICSP=1` machine. These snapshots verify exactly what install writes,
where, and at which bitness — the ground truth the rest of the plan depends on.

## Ground rules

- **Do not download anything.** Use only the plugin installer the owner already
  placed in Downloads (`cadesplugin.exe`). If you can't find it, stop and ask.
- **Do not edit the registry or "fix" anything.** This task is read-only except
  for writing snapshot files and committing them.
- Run each snapshot with the SAME script so phases diff cleanly:
  `tools/windows/snapshot-cryptopro-state.ps1`.
- Run PowerShell **as Administrator** (registry export + Program Files install
  need it). The snapshot script itself is harmless; admin is for completeness.
- Commit after each phase (don't batch all three) so evidence is preserved even
  if a later step goes wrong. Push to `main` with your `GH_TOKEN`.
- This machine reads `config.ini`, not the Windows registry, for provider config
  (CryptoPro uses its own config as a "registry"). So "registry is empty" is an
  expected, important finding — record it, don't treat it as failure.

## Repository prep

```powershell
cd <repo>
git pull origin main
# snapshots land under docs/minicsp-snapshots/<phase>/
```

The snapshot script writes per phase:
- `registry/*.reg` + `*.txt` — `reg export` and `reg query /s` of every relevant
  key (HKLM/HKCU, native + WOW6432Node, CryptoPro + CryptoAPI Defaults\Provider
  + Chrome native-messaging host).
- `files/pf-x86-cryptopro.txt` — recursive `Program Files (x86)\Crypto Pro`
  listing with size + SHA-256 per file (and `Program Files\Crypto Pro` too).
- `files/minicsp-config.ini`, `files/minicsp-license.ini` — verbatim copies.
- `summary.txt` — which keys/paths existed, bitness of key binaries.

## Phase 1 — clean machine (no plugin installed)

```powershell
./tools/windows/snapshot-cryptopro-state.ps1 -Phase clean
```

Analyze and write findings to `docs/minicsp-snapshots/clean/ANALYSIS.md`:
- Confirm there is NO `Crypto Pro` key (HKLM/HKCU, native/WOW6432Node).
- Record what `HKLM\...\Microsoft\Cryptography\Defaults\Provider Types` already
  contains on a stock Windows (baseline; usually only Microsoft providers).
- Confirm `Program Files (x86)\Crypto Pro` is absent.

Commit:
```powershell
git add docs/minicsp-snapshots/clean tools/windows/snapshot-cryptopro-state.ps1
git commit -m "docs: snapshot clean Windows CryptoPro state (phase 1)"
git push origin HEAD:main
```

## Phase 2 — plugin installed WITHOUT flags

Install the owner's installer with no extra args (silent if it supports it,
otherwise click through with all defaults):

```powershell
# from the folder containing the installer (use the real filename):
& "$env:USERPROFILE\Downloads\cadesplugin.exe"
```

After install completes (and the install dialog is fully done), snapshot:

```powershell
./tools/windows/snapshot-cryptopro-state.ps1 -Phase installed-noflags
```

Analyze → `docs/minicsp-snapshots/installed-noflags/ANALYSIS.md`:
- Did a `Crypto Pro` registry branch appear? Where (HKLM vs WOW6432Node)? Dump
  the notable values (especially anything like `AppPath`, `CurrentVersion`).
- Did `Defaults\Provider` / `Provider Types` gain GOST 75/80/81 entries? If yes,
  list provider names + `Image Path`.
- Is there a `Mini CSP` folder? (Default install usually does NOT add it.)
- Bitness of `nmcades.exe`, `npcades.dll`, `Mini CSP\capi20.dll` (if present).
- Diff against phase 1: what exactly was added.

Commit:
```powershell
git add docs/minicsp-snapshots/installed-noflags
git commit -m "docs: snapshot CryptoPro plugin install without flags (phase 2)"
git push origin HEAD:main
```

## Phase 3 — reinstall with ADDMINICSP=1

Uninstall the plugin first (Apps & Features, or the installer's uninstall), then
reinstall with the Mini CSP flag exactly as the owner specified:

```powershell
& "$env:USERPROFILE\Downloads\cadesplugin.exe" -cadesargs "ADDMINICSP=1"
```

After it finishes, snapshot:

```powershell
./tools/windows/snapshot-cryptopro-state.ps1 -Phase installed-addminicsp
```

Analyze → `docs/minicsp-snapshots/installed-addminicsp/ANALYSIS.md`:
- Confirm the `Mini CSP` folder now exists; list its files (the snapshot already
  captures size+sha256). Compare `config.ini` to the one we already analyzed in
  the bundle (provider sections `[Defaults\Provider\...]` Type 75/80/81).
- **Key question:** does `ADDMINICSP=1` add ANY registry keys vs phase 2? The
  owner observed none — verify and record precisely (this confirms the
  config.ini-only model and kills the registry hypothesis for good).
- Diff phase 3 vs phase 2 (files added = the whole `Mini CSP` subtree?) and
  phase 3 registry vs phase 2 registry (expected: identical / no new keys).
- Note `config.ini` vs `config64.ini` presence and the bitness of the binaries
  that would read them.

Commit:
```powershell
git add docs/minicsp-snapshots/installed-addminicsp
git commit -m "docs: snapshot CryptoPro plugin install with ADDMINICSP=1 (phase 3)"
git push origin HEAD:main
```

## Final write-up

Add `docs/minicsp-snapshots/CONCLUSIONS.md` summarizing the cross-phase diff and
what it proves:
- Exactly what each install state writes to the registry (if anything).
- Exactly what `ADDMINICSP=1` adds on disk and whether it touches the registry.
- Whether provider registration lives in the registry or only in
  `Mini CSP\config.ini`.
- Implications for our portable launcher: what we must replicate to make the
  bundled Mini CSP enumerate providers without a full MSI install.

Then report back to the owner with the conclusions and the next experiment
(open the deployed test pages — `internal-csp`, `internal-csp-early` — in plain
Chrome on this `ADDMINICSP=1` machine and record whether providers enumerate;
if not, ProcMon on `nmcades.exe` for `Load Image Mini CSP\capi20.dll` and
`CreateFile config.ini`).

## Notes / gotchas

- `reg export` writes UTF-16; that's fine to commit (diffs are noisy but the
  `*.txt` `reg query` dumps are the readable ones). If a `*.reg` is huge/noisy,
  keep it anyway — it's the authoritative copy.
- File hashes let us later prove our bundle's Mini CSP files are byte-identical
  to a real `ADDMINICSP=1` install (rules out repackaging corruption).
- If the installer filename differs from `cadesplugin.exe`, use the actual name;
  do not fetch a different build.
- Keep commits scoped to `docs/minicsp-snapshots/**` (+ the snapshot script on
  phase 1). Don't touch Go code or build scripts here.
- Verify `build-windows` CI stays green after pushes (these are docs-only, so it
  should be unaffected), but since you're pushing to `main`, glance at Actions.
