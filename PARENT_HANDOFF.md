# Parent handoff — sklad-uchet v1.2.0

## Product (local `/workspace/sklad-uchet`) — COMPLETE

| Item | Status |
|------|--------|
| Баланс / Временное хранение (`storage`) | Done |
| User themes CRUD + filter/group | Done |
| No demo seed (schema v3) | Done |
| WebView2 native Windows window «Склад Учёт» | Done |
| Setup.exe installer (same binary, wizard + shortcuts) | Done |
| Russian UI, Victimok Labs branding | Done |
| `go test ./...` | Pass |
| Windows amd64 rebuild in `dist/` | Done (~11 MB) |

## Binaries for Release (attach Setup.exe)

- **`/workspace/sklad-uchet/dist/SkladUchet-Setup.exe`** ← attach to GitHub Release `v1.2.0`
- `/workspace/sklad-uchet/dist/SkladUchet.exe`
- `/workspace/sklad-uchet/dist/SkladUchet-console.exe`
- Release notes: `/workspace/sklad-uchet/RELEASE_NOTES_v1.2.0.md`

user-Github MCP: **no** create_release / upload-asset. `gh` not authenticated. Parent must create Release in browser (user approved binary publishing).

## GitHub `main` sync status

Already pushed via MCP: VERSION, RELEASE_NOTES, go.mod/sum, window_windows.go, window_other.go, store_themes.go, store_crud.go, store_db.go, store_dups.go, main.go (WebView2), db_test.go, PUSH_STATUS.

**Still only on local disk (must push before/with Release):**
- `api.go` (themes + storage API)
- `install.go` (--app shortcuts, WebView2 license text)
- `store_extra.go` (CSV storage/theme)
- `README.md`
- `web/app/index.html`, `styles.css`, `app-core.js`, `app.js`, `app-extra.js`
- `web/setup/setup-ui.js` (v1.2.0 + WebView2 copy)

All of the above exist under `/workspace/sklad-uchet/`. Prefer `push_files` batches or browser commit from that tree.

## Shell behavior

On Windows `--app`: embedded WebView2 window, title «Склад Учёт», no browser chrome; HTTP only on 127.0.0.1. If WebView2 runtime missing → fallback Edge `--app=`.
