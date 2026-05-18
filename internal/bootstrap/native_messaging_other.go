//go:build !windows

package bootstrap

func registerCryptoProNativeMessagingHost(string) error {
	return nil
}
