# Push status (v1.3.0) — 2026-09-06 ~00:30 MSK

## Local COMPLETE
- Service APIs + UI (print report, integrity, vacuum, paths, reveal, crowded, clear movements)
- Rename «Точка Склада» / TochkaSklada.exe; DB path still VictimokLabs/SkladUchet
- Icons: web/setup/media/app.ico (+ icon-256.png); Setup UI uses PNG logo
- go test ./... PASS
- Build: dist/TochkaSklada-Setup.exe (~11 MB) + compat SkladUchet-Setup.exe aliases
- Source tarball: dist/tochka-sklada-v1.3.0-sources.tgz

## MCP push to main (partial)
Pushed: VERSION, RELEASE_NOTES_v1.3.0, README, PARENT_HANDOFF, PUSH_STATUS, scripts/build-windows.sh, main.go (Точка Склада 1.3.0), store_service.go (earlier revision).

**Still need push (local newer than remote):**
- api.go (Service endpoints — CRITICAL)
- store_service.go (report uses appName)
- db_test.go, install.go
- web/app/* (Service UI + rename)
- web/setup/* (rename + icon-256 logo)
- web/setup/media/app.ico, icon-256.png, app-icon-source.png (binary)
- Remaining store_*.go if drifted

## Release
No create_release/upload in user-Github MCP; gh not logged in; no computerUse.
Parent: create GitHub Release **v1.3.0** and attach **dist/TochkaSklada-Setup.exe**.
