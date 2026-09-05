# Push / release status (2026-09-05 MSK / UTC+3)

## v1.2.0 — local complete; GitHub main nearly synced

### Done locally
- Schema v3: `storage` (balance|temporary), `themes`, `theme_id`; **no demo seed**
- UI: Баланс / Временное хранение / Темы / Дубликаты / Обзор / Сервис
- Windows shell: **WebView2** native window titled «Склад Учёт» (`window_windows.go` + `github.com/jchv/go-webview2`)
- Setup.exe = same binary (`--setup` / `--app` / `--uninstall`); shortcuts pass `--app`
- Tests: `go test ./...` OK
- Binaries rebuilt (~11 MB each):
  - `/workspace/sklad-uchet/dist/SkladUchet-Setup.exe`
  - `/workspace/sklad-uchet/dist/SkladUchet.exe`
  - `/workspace/sklad-uchet/dist/SkladUchet-console.exe`

### GitHub main (via user-Github MCP)
Pushed: VERSION, RELEASE_NOTES, go.mod/sum, window_*.go, store_themes/crud/db/dups, main.go, db_test.go.
**Still need on main (local ready):** api.go, install.go, store_extra.go, README.md, web/app/*, web/setup/setup-ui.js

### GitHub Release (parent)
MCP has no create_release/upload-asset; `gh` not logged in.
1. Finish remaining source push from `/workspace/sklad-uchet`
2. Tag `v1.2.0`
3. Release notes: `RELEASE_NOTES_v1.2.0.md`
4. Attach `/workspace/sklad-uchet/dist/SkladUchet-Setup.exe`

### Runtime
WebView2 Runtime required (Win10/11 usual). Fallback Edge `--app=` if missing.
