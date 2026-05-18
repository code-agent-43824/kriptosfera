package bootstrap

import (
	"io"
	"path/filepath"
	"strings"
)

func unzipCryptoProPlugin(readerAt io.ReaderAt, size int64, dest string) error {
	return unzipReaderAtFiltered(readerAt, size, dest, shouldSkipCryptoProPluginZipEntry)
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
