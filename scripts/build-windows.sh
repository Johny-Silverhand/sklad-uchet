#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
mkdir -p dist
echo "Building TochkaSklada-Setup.exe (amd64, windowsgui)…"
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w -H windowsgui" -o dist/TochkaSklada-Setup.exe .
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w -H windowsgui" -o dist/TochkaSklada.exe .
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o dist/TochkaSklada-console.exe .
# Compat aliases for older docs/scripts
cp -f dist/TochkaSklada-Setup.exe dist/SkladUchet-Setup.exe
cp -f dist/TochkaSklada.exe dist/SkladUchet.exe
cp -f dist/TochkaSklada-console.exe dist/SkladUchet-console.exe
ls -lh dist/*.exe
echo "Done. Installer: dist/TochkaSklada-Setup.exe (modes: --setup / --app)"
