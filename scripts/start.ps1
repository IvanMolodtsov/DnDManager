# Run the web server with hot reload (Air).
# Rebuilds on .go / .html / .json / .sql / .css changes.
$ErrorActionPreference = "Stop"
Set-Location (Split-Path -Parent $PSScriptRoot)
New-Item -ItemType Directory -Force -Path tmp | Out-Null
Write-Host "Hot reload: http://127.0.0.1:8080  (Ctrl+C to stop)"
go run github.com/air-verse/air@v1.61.7
