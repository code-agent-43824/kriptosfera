param(
  [string]$Version = "0.1.0",
  [string]$OutputDir = "dist"
)

$ErrorActionPreference = "Stop"

New-Item -ItemType Directory -Force -Path $OutputDir | Out-Null
Set-Content -Path "internal/bootstrap/app-version.txt" -Value $Version -NoNewline
Set-Content -Path "internal/config/app-version.txt" -Value $Version -NoNewline

pwsh ./build/prepare-payload.ps1
pwsh ./build/embed-payload.ps1

go test ./...
$env:GOOS = "windows"
$env:GOARCH = "amd64"
go build -trimpath -ldflags "-H=windowsgui -s -w" -o (Join-Path $OutputDir "KriptosferaDemo.exe") ./cmd/kriptosfera-launcher

Copy-Item README.md (Join-Path $OutputDir "README.txt")
Write-Host "Build completed: $(Join-Path $OutputDir 'KriptosferaDemo.exe')"
