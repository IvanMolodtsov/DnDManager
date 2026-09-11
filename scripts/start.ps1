# Run the web server from the repo root.
$ErrorActionPreference = "Stop"
Set-Location (Split-Path -Parent $PSScriptRoot)
go run ./cmd/web
