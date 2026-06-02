# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project aims to follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html).
Version numbers track the launcher/payload (`internal/config/app-version.txt`).

## [Unreleased]

### Mini CSP / CSP Lite
- Identified the clean-machine blocker: the pinned CAdES plug-in build
  `2.0.15700` was broken (per CryptoPro); plug-in `2.0.15000` activates the
  bundled Mini CSP with no system CSP. The working combination requires the
  Manifest V2 extension (`1.2.13`) and a Manifest V2-capable Chromium. See
  `docs/cryptopro-csp-lite-plan.md`.
- Pinned the bundled Chromium to the last Manifest V2-capable Chrome for Testing
  milestone (`138.0.7204.183`); Chrome 139+ removed `ExtensionManifestV2Availability`.
  Future goal: return to the latest Chromium + Manifest V3 once CryptoPro ships a
  fixed plug-in build.
- Re-pinned the embedded CryptoPro plug-in bundle to the legacy `2.0.15000`
  archive and switched the payload extension to CryptoPro Manifest V2 `1.2.13`.
  This is a temporary compatibility profile, not the long-term Chromium/extension
  baseline.
- The launcher now applies the per-user Chrome policy
  `ExtensionManifestV2Availability=2` only when a loadable Manifest V2 extension
  is present, keeping the future Manifest V3/latest-Chromium path reversible.

### Performance
- Payload reuse now only checks that manifest files exist instead of re-hashing
  the entire payload (Chromium included) on every launch; full SHA-256
  verification still runs once at extraction time. Startup no longer pays a
  hundreds-of-MB hashing cost on each run.
- `validateCryptoProPluginLayout` walks the plugin directory once (matching all
  required suffixes in a single pass) instead of once per required file.
- Native messaging skips re-writing the manifest and re-registering the HKCU
  host when nothing changed since the last run (gated by a state file).

### Changed
- The demo `startUrl` now points at the internal-csp test page
  (`mescheryakov.pro/.../internal-csp/...`), which sets `EnableInternalCSP`, so a
  launcher run exercises the bundled Mini CSP; `allowedOrigins` updated to match.
- Added `docs/worklog.md` as a per-chunk handoff log for the multi-agent workflow.
- The launcher again starts Chromium as a standalone app window (`--app=<startUrl>`
  in `windowMode: "app"`); the diagnostics behavior that opened the target page
  and a diagnostics page as two tabs was removed.
- Removed the Windows investigation scripts and snapshot/handoff material used
  while diagnosing the Mini CSP issue (payload assets and build scripts kept).
- A second launch of an already-prepared app no longer fails with "bootstrap
  already in progress": preparation now checks for an existing payload/plugin
  before taking the lock, the lock waits (with a bounded timeout) for a
  concurrent first run instead of erroring immediately, and the lock is
  heartbeated so a slow first run is not mistaken for a stale lock.
- `selectCryptoProExtensionID` now signals when it falls back to a non-canonical
  extension, and the launcher logs a warning in that case.
- `build-windows` CI now publishes two separate workflow artifacts
  (`kriptosfera-windows-embedded` and `kriptosfera-windows-remote`) instead of
  one combined archive, so the thin/remote launcher can be downloaded without
  the large embedded build.

### Fixed
- `TestCryptoProPluginManagerSkipsInvalidMSIPseudoPaths` is now Windows-portable:
  it no longer relies on `os.IsNotExist` for a path containing `:` (which Windows
  reports as a syntax error, not "not found"), so CI on Windows runners passes.
- The embedded launcher build now fails when `go test` fails (previously a test
  failure did not stop the PowerShell build step, so it slipped through CI).
- CI now verifies the embedded CryptoPro bundle contains every required file
  (`TestEmbeddedCryptoProBundleContainsRequiredFiles`), so a bad bundle pin is
  caught at build time instead of failing on a user's machine.

### Added
- Documented the Linux CryptoPro CSP/CAdES installer archive and extracted
  package layout published on project static storage for experiments. The
  archives and packages stay on the server only and are not committed to Git.
- Apache-2.0 `LICENSE` and `NOTICE` clarifying that third-party runtime
  components (Chromium, CryptoPro plugin/CSP) keep their own terms.
- Repository documentation: `CONTRIBUTING.md`, `docs/README.md` index,
  `docs/architecture.md` (launcher flow + AppData layout), and package-level
  godoc comments for `bootstrap`, `config`, and `logging`.
- Validation of `profileName` as a safe single path segment so the per-app
  profile directory cannot escape the app root.
- Size cap on remote payload downloads (pinned size or a 1 GiB absolute limit)
  with early abort, plus tests for both new guards.
- Committed zero-byte placeholder `payload.zip` / `cryptopro-plugin.zip` so the
  launcher compiles and `go test ./...` runs on a clean checkout.

## [0.5.0]

### Added
- Remote payload mode for the thin launcher: HTTPS download, SHA-256
  verification, immutable version/sha layout, and cache reuse.
- Embedded CryptoPro CAdES Browser extension (`1.3.17`) with a stable
  extension id derived from `manifest.key`.
- Embedded CryptoPro Browser Plugin bundle (`2.0.15700`), pinned by a lock file
  and verified by SHA-256/size, extracted into AppData at runtime.
- Native messaging manifest generation and per-user HKCU registration for
  `ru.cryptopro.nmcades`.
- Hosted diagnostics page using CryptoPro's official `cadesplugin_api.js`, and
  a read-only `inspect-cryptopro-modules.ps1` diagnostics script.
- Windows CI on GitHub-hosted runners and a single-artifact publish model.

## [0.1.0]

### Added
- Initial Go launcher skeleton with the single-file embedded bootstrapper
  (embedded `payload.zip`), embedded Chromium runtime launch, a dedicated
  browser `user-data-dir`, and the PowerShell-based Windows build scripts.

[Unreleased]: https://github.com/code-agent-43824/kriptosfera/compare/v0.5.0...HEAD
[0.5.0]: https://github.com/code-agent-43824/kriptosfera/compare/v0.1.0...v0.5.0
[0.1.0]: https://github.com/code-agent-43824/kriptosfera/releases/tag/v0.1.0
