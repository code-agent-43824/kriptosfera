package logging

import (
    "fmt"
    "os"
    "path/filepath"
    "time"
)

type Logger struct {
    file *os.File
}

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

func (l *Logger) Close() error { return l.file.Close() }

func (l *Logger) Info(format string, args ...any) {
    line := fmt.Sprintf(format, args...)
    fmt.Fprintf(l.file, "%s %s\n", time.Now().UTC().Format(time.RFC3339), line)
}
