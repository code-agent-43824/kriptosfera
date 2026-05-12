package main

import (
    "os"

    "github.com/code-agent-43824/kriptosfera/internal/bootstrap"
)

func main() {
    cfg, err := bootstrap.DefaultConfig()
    if err != nil {
        bootstrap.ReportFatal("Не удалось подготовить конфигурацию запуска", err)
        os.Exit(1)
    }
    if err := bootstrap.Run(cfg); err != nil {
        bootstrap.ReportFatal("Не удалось запустить приложение", err)
        os.Exit(1)
    }
}
