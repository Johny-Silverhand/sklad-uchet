# Parent handoff — Точка Склада v1.3.0

## Product `/workspace/sklad-uchet` — READY

| Item | Status |
|------|--------|
| Rename «Точка Склада» / TochkaSklada.exe | Done |
| Icon app.ico + setup PNG logo | Done |
| DB path stable `%APPDATA%\VictimokLabs\SkladUchet` | Done |
| Service: print report, integrity, vacuum, paths, crowded cells, clear movements | Done |
| `go test ./...` | Pass |
| Windows amd64 Setup | `dist/TochkaSklada-Setup.exe` (+ compat SkladUchet-Setup.exe alias) |

## Release attach
- **`/workspace/sklad-uchet/dist/TochkaSklada-Setup.exe`** ← GitHub Release `v1.3.0`
- Notes: `RELEASE_NOTES_v1.3.0.md`

user-Github MCP: no create_release/upload-asset. `gh` not authenticated. Parent/browser must publish Release with Setup.exe.

## Branding notes
- Display: Точка Склада · credit Victimok Labs
- Install dir default: `%LOCALAPPDATA%\Programs\Victimok Labs\TochkaSklada`
- Data/DB: unchanged `VictimokLabs\SkladUchet`
