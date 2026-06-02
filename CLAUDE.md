# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this project is

Kriptosfera is an MVP scaffold for a Windows desktop app delivered as a single
`.exe`: a Go launcher that prepares a payload (pinned Chromium runtime + CryptoPro
CAdES browser extension + native messaging host + CryptoPro Browser Plugin / Mini
CSP), then starts an isolated Chromium against a target page. The end goal is a
test signature with a Rutoken on the CryptoPro CAdES demo page **without** a
system-installed CryptoPro CSP. The launcher is the only compiled code; everything
else is payload assets and PowerShell build scripts.

MVP stage 6 (bundled **Mini CSP / CSP Lite** on a clean machine) is root-caused
and unblocked: the previously pinned plug-in build `2.0.15700` was broken; the
working combination is plug-in `2.0.15000` + a Manifest V2 extension (`1.2.13`)
+ a Manifest V2-capable Chromium (Chrome 138). Wiring that into the launcher and
the Rutoken signing check are the remaining integration steps. See
`docs/cryptopro-csp-lite-plan.md` (status + plan) and `docs/architecture.md`.

## Build, test, and develop

The Go module needs two `go:embed` artifacts that are NOT real in Git:
`internal/bootstrap/payload.zip` and `internal/bootstrap/cryptopro-plugin.zip` are
committed as **zero-byte placeholders** so the tree compiles on any platform. An
empty embed is treated as "bundle not embedded". Windows build scripts overwrite
them with real artifacts. Do not delete these placeholders, and do not commit real
CryptoPro/Chromium binaries.

```sh
gofmt -l .                                            # must be empty
go vet ./...
go test ./...                                         # full suite (Linux/macOS OK)
go test ./internal/bootstrap/ -run TestName -v        # single test
GOOS=windows GOARCH=amd64 go build ./...              # embedded launcher
GOOS=windows GOARCH=amd64 go build -tags remote ./... # thin/remote launcher
```

On non-Windows hosts the launcher does a diagnostics "dry-run" instead of starting
Chromium, so the full bootstrap path is still exercised by `go test`.

Full Windows build (PowerShell, run in CI on `windows-2025-vs2026`):

```powershell
./build/build-payload.ps1
./build/build-launcher.ps1 -PayloadMode embedded -PayloadZip dist/payload.zip -PayloadMetadata dist/payload.json
./build/build-launcher.ps1 -PayloadMode remote -UsePayloadLock -PayloadLockPath build/payload-lock.json
```

`build/build-launcher.ps1` runs `go test ./...` for the embedded build and fails the
step on test failure — keep tests green or Windows CI goes red even when local
Linux passes (some failures are Windows-only, e.g. paths containing `:`).

## Architecture essentials

Read these together to understand the big picture; they are coupled by file-based
state and conventions, not just function calls.

- **Entry → flow:** `cmd/kriptosfera-launcher/main.go` calls `bootstrap.Run`
  (`internal/bootstrap/bootstrap.go`), which is the ordered pipeline: resolve app
  root → pick `PayloadSource` → `PayloadManager.Prepare` → load+validate
  `AppConfig` → extract CryptoPro plugin → detect extensions → register native
  messaging → write diagnostics → launch Chromium (Windows) or dry-run.

- **Two config layers** (`internal/config`): `RuntimeConfig` is baked into the
  binary at build time (`runtime-config.json`/`app-version.txt`) and selects
  payload mode (embedded vs remote) + version/URL/SHA-256. `AppConfig` ships inside
  the payload (`config/app-config.json`) and drives start URL, allowed origins,
  profile name, window mode, diagnostics. `validateAppConfig` enforces invariants
  (HTTPS diagnostics URL, `profileName` as a safe single path segment, startURL
  within `allowedOrigins`).

- **Payload preparation invariants** (`payload_manager.go`, `bootstrap.go`): every
  prepared component (payload, CryptoPro plugin) uses the same pattern — a bootstrap
  lock (file lock with TTL + heartbeat, waits for a concurrent first run), reuse via
  a ready-marker + state file (`.payload-ready`/`.payload-state.json`,
  `.cryptopro-plugin-ready`/state), extract to a staging dir, verify, then atomic
  rename into the versioned dir under `%LOCALAPPDATA%\Kriptosfera\apps\demo\<ver>`.
  Reuse checks file presence only; full SHA-256 verification runs at extraction
  time. Unzip rejects zip-slip and skips MSI pseudo-path entries (names containing
  `:`).

- **Platform split:** `*_windows.go` vs `*_other.go` via build tags (progress
  window, native messaging registry write, embedded plugin bytes, dialogs). The
  `remote` build tag swaps the embedded payload source for the remote downloader
  (`payload_remote.go`/`payload_source_remote.go`, HTTPS + SHA-256 + size cap).

- **CryptoPro layer** (`cryptopro_plugin_manager.go`, `native_messaging*.go`,
  `extensions.go`): the Browser Plugin bundle is extracted under
  `<appDir>/cryptopro/plugin`, validated against `requiredCryptoProPluginFiles`
  (CAdES runtime + Mini CSP DLLs). Native messaging manifest for
  `ru.cryptopro.nmcades` is written and registered in HKCU (per-user, no admin),
  gated by a state file so it is not rewritten unnecessarily. Extension id is
  derived deterministically from `manifest.key` (SHA-256 → a–p mapping).

## Mini CSP / CSP Lite domain knowledge

Resolution (details in `docs/cryptopro-csp-lite-plan.md`):

- The clean-machine "provider not loaded / `0x80090017`" symptom was a **broken
  CAdES plug-in build (`2.0.15700`)**, confirmed by CryptoPro. Plug-in
  `2.0.15000` activates the bundled Mini CSP with no system CSP.
- The working combination is plug-in **`2.0.15000`** + the **Manifest V2**
  extension (`1.2.13`) + a Manifest V2-capable Chromium (**Chrome 138**, the last
  milestone honoring `ExtensionManifestV2Availability`). `build/chromium-runtime.json`
  is pinned to 138.x for this reason.
- The bundled Mini CSP needs no Windows registry, no extra license (ProductID is
  in `Mini CSP\license.ini`), and no extra runtime; the 32-bit native host reads
  `Mini CSP\config.ini`.
- Remaining integration: re-pin plug-in to `2.0.15000`, ship the MV2 extension,
  apply `ExtensionManifestV2Availability=2`, verify Rutoken signing.
- FUTURE: return to the latest Chromium + a Manifest V3 extension once CryptoPro
  ships a fixed MV3-compatible plug-in build (MV2 + Chrome 138 is temporary).

## Conventions

- **Multi-agent workflow (important):** several coding agents work on this repo,
  sometimes in parallel. Before/with each change — even a small one — record it in
  `docs/worklog.md` (what was planned, what was done, next steps) so the next agent
  can pick up unfinished work. Keep `CHANGELOG.md` and the relevant plan docs
  (e.g. `docs/cryptopro-csp-lite-plan.md`) in sync. Document first, then code.
- Commits use Conventional Commits (`feat:`/`fix:`/`docs:`/`chore:`/`ci:`).
- Exported Go symbols carry godoc comments; run `gofmt` before committing.
- Keep all CryptoPro/Chromium binaries out of Git; fetch/generate via build
  scripts, pinned by SHA-256/size lock files (`build/*-lock.json`).
- Update `CHANGELOG.md` for user-visible changes.
- Two payload modes are first-class: `embedded` (offline/demo) and `remote` (thin
  launcher, the main product direction). CI publishes them as two separate Windows
  artifacts (`kriptosfera-windows-embedded`, `kriptosfera-windows-remote`).
