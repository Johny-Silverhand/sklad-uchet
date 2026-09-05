# Push / release status (2026-09-05 MSK)

## v1.2.0

### Source
Pushed to `main` via user-Github MCP (`push_files`).

### Local Windows binaries (rebuild)
- `/workspace/sklad-uchet/dist/SkladUchet-Setup.exe` — installer / setup wizard
- `/workspace/sklad-uchet/dist/SkladUchet.exe` — same binary alias
- `/workspace/sklad-uchet/dist/SkladUchet-console.exe` — console debug build

### GitHub Release
Tag: `v1.2.0`  
Attach: `SkladUchet-Setup.exe`  
user-Github MCP has no `create_release` / upload-asset tool; `gh` not authenticated on this box. Parent should publish Release in browser (user approved binary publishing).

### Shell
Windows: embedded **WebView2** window titled «Склад Учёт» (no browser chrome). Fallback to Edge `--app=` only if WebView2 runtime missing.
