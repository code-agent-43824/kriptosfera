package main

import (
    "fmt"
    "os"

    "github.com/code-agent-43824/kriptosfera/internal/bootstrap"
)

func main() {
    cfg, err := bootstrap.DefaultConfig()
    if err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
    if err := bootstrap.Run(cfg); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
