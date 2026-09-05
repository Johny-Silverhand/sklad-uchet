# Parent handoff — Точка Склада v1.3.0

## Local product READY (ship this)

| Item | Path / status |
|------|----------------|
| Setup binary | **`/workspace/sklad-uchet/dist/TochkaSklada-Setup.exe`** (~12 MB) |
| Compat alias | `dist/SkladUchet-Setup.exe` (same bytes) |
| Source tarball | `dist/tochka-sklada-v1.3.0-sources.tgz` (full text+icons for MCP/git sync) |
| Notes | `RELEASE_NOTES_v1.3.0.md` |
| Tests | `go test ./...` PASS |

## Features in the Setup.exe (embedded)
- Rename **Точка Склада** / exe **TochkaSklada**; credit Victimok Labs
- DB path unchanged: `%APPDATA%\VictimokLabs\SkladUchet`
- Icon `app.ico` embedded → shortcuts/uninstall DisplayIcon
- Setup wizard logo: `icon-256.png`
- Service: printable report, low-stock print, integrity, VACUUM, paths/reveal, clear movements, crowded cells, stats cards, aboutVer from health

## GitHub main (MCP partial)
Pushed: VERSION, RELEASE_NOTES, README, main.go (Точка Склада 1.3.0), store_service.go, build-windows.sh, handoff docs.
**Not fully synced yet:** api.go (Service routes), web/app/*, web/setup/*, binary icons, install.go, db_test.go — use `dist/tochka-sklada-v1.3.0-sources.tgz` or local tree to finish push_files.

## Release
user-Github MCP has **no** create_release/upload-asset. `gh` not authenticated. No computerUse.
→ Parent: create Release **v1.3.0** in browser and attach **TochkaSklada-Setup.exe**.
