# Run all Go tests from the repo root.
$ErrorActionPreference = "Stop"
Set-Location (Split-Path -Parent $PSScriptRoot)
go test ./...
