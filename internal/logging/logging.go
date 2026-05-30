package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Logger is an append-only file logger that writes timestamped lines.
type Logger struct {
	file *os.File
}

// New opens (creating parent directories and the file as needed) the log at
// path for appending and returns a Logger writing to it.
func New(path string) (*Logger, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	return &Logger{file: f}, nil
}

// Close closes the underlying log file.
func (l *Logger) Close() error { return l.file.Close() }

// Info formats its arguments and writes a single RFC 3339 UTC-timestamped line.
func (l *Logger) Info(format string, args ...any) {
	line := fmt.Sprintf(format, args...)
	fmt.Fprintf(l.file, "%s %s\n", time.Now().UTC().Format(time.RFC3339), line)
}
