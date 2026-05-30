# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project aims to follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html).
Version numbers track the launcher/payload (`internal/config/app-version.txt`).

## [Unreleased]

### Changed
- `build-windows` CI now publishes two separate workflow artifacts
  (`kriptosfera-windows-embedded` and `kriptosfera-windows-remote`) instead of
  one combined archive, so the thin/remote launcher can be downloaded without
  the large embedded build.

### Added
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
