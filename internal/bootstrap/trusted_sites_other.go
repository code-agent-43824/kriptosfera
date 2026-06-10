//go:build !windows

package bootstrap

// writeCryptoProTrustedSitesRegistry is a no-op on non-Windows hosts: the
// CryptoPro trusted-sites list lives in the per-user Windows registry. The
// launcher only does a diagnostics dry-run off Windows.
func writeCryptoProTrustedSitesRegistry(keyPath, valueName string, sites []string) error {
	return nil
}
