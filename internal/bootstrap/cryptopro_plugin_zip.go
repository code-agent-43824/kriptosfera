package bootstrap

import (
	"io"
	"path/filepath"
	"strings"
)

func unzipCryptoProPlugin(readerAt io.ReaderAt, size int64, dest string) error {
	return unzipReaderAtMapped(readerAt, size, dest, mapCryptoProPluginZipEntry)
}

func shouldSkipCryptoProPluginZipEntry(name string) bool {
	cleanName := filepath.ToSlash(filepath.Clean(name))
	for _, part := range strings.Split(cleanName, "/") {
		if strings.Contains(part, ":") {
			return true
		}
	}
	return false
}

func mapCryptoProPluginZipEntry(name string) (string, bool) {
	if shouldSkipCryptoProPluginZipEntry(name) {
		return "", true
	}
	cleanName := filepath.ToSlash(filepath.Clean(name))
	parts := strings.Split(cleanName, "/")
	if len(parts) >= 2 && parts[0] == "CAdES Browser Plug-in" {
		return cleanName, false
	}
	for i := 0; i+2 < len(parts); i++ {
		if parts[i] == "Program Files" && parts[i+1] == "Crypto Pro" {
			rel := strings.Join(parts[i+2:], "/")
			if rel == "" || rel == "." {
				return "", true
			}
			return rel, false
		}
	}
	return "", true
}
