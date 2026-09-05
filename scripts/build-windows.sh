#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
mkdir -p dist
echo "Building SkladUchet-Setup.exe (amd64, windowsgui)…"
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w -H windowsgui" -o dist/SkladUchet-Setup.exe .
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w -H windowsgui" -o dist/SkladUchet.exe .
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o dist/SkladUchet-console.exe .
ls -lh dist/*.exe
echo "Done. Installer: dist/SkladUchet-Setup.exe (modes: --setup / --app)"
