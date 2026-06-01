//go:build !windows

package bootstrap

func setChromePolicyDWORD(string, int) error {
	return nil
}
