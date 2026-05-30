// Package bootstrap implements the Kriptosfera launcher runtime: it prepares
// the application payload, wires up the bundled CryptoPro layer, and starts the
// managed Chromium browser.
//
// The high-level flow performed by [Run] is:
//
//  1. resolve the per-user application root under LOCALAPPDATA (Windows) or
//     ~/.local/share (other platforms) and open the launcher log;
//  2. select a [PayloadSource] (embedded or remote) and let [PayloadManager]
//     extract, verify, and cache the payload under apps/demo/<version>;
//  3. load and validate the bundled app-config.json;
//  4. extract the embedded CryptoPro Browser Plugin bundle, detect browser
//     extensions, and register the native messaging host (HKCU, Windows only);
//  5. build the Chromium command line and launch the browser with a dedicated
//     user-data-dir, or write a diagnostics dry-run on non-Windows hosts.
//
// Platform-specific behavior (progress window, registry registration, native
// dialogs, embedded plugin bytes) is split into _windows.go / _other.go files
// guarded by build tags so the package compiles on every platform.
package bootstrap
