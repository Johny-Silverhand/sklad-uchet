# Склад Учёт

Десктопное приложение для товароведа склада: учёт **запчастей**, **устройств** и **комплектующих**.

Интерфейс на русском. Данные **локально** в SQLite (WAL). На Windows — **нативное окно WebView2** (без адресной строки и хрома браузера), заголовок «Склад Учёт». HTTP только на `127.0.0.1` внутри процесса.

**Разработано в Victimok Labs.**

## Возможности

- Разделы **Баланс** (постоянное хранение) и **Временное хранение**
- Пользовательские **темы** (создать / переименовать / удалить, назначить позиции)
- CRUD позиций, поиск по названию и ячейке, фильтр по типу и теме
- **Мин. остаток** + «Мало на складе», быстрые +/−, перенос между балансом и временным
- CSV экспорт/импорт (поля включая `storage`, `theme`), backup/restore SQLite
- Дубликаты и объединение, обзор, журнал movements
- Пустая БД при первом запуске (**без демо-данных**)
- Тёмный UI + **SkladUchet-Setup.exe** (мастер установки, ярлыки, uninstall)

## Установка на Windows 11

1. Скачайте `SkladUchet-Setup.exe` из `dist/` (или Releases).
2. Запустите двойным щелчком — откроется тёмный мастер установки Victimok Labs.
3. Выберите папку, ярлыки (рабочий стол / меню Пуск) → «Установить».
4. После установки запускайте ярлык **«Склад Учёт»** (режим `--app`).

Нужен **Microsoft Edge WebView2 Runtime** (предустановлен на актуальных Windows 10/11).

Удаление: «Удалить Склад Учёт» в меню Пуск или Параметры → Приложения. База в `%APPDATA%` сохраняется.

Режимы одного exe:

| Флаг | Назначение |
|------|------------|
| (по умолчанию) | `--setup`, если ещё не установлен; иначе `--app` |
| `--setup` | мастер установки |
| `--app` | обычный запуск приложения |
| `--uninstall` | удаление |

## Поля позиции

| Поле | Описание |
|------|----------|
| название | name |
| тип | `zapchast` / `ustroystvo` / `komplektuyushchee` |
| хранение | `balance` (баланс) / `temporary` (временное) |
| тема | пользовательская категория или «Без темы» |
| кол-во | целое ≥ 0 |
| мин. остаток | `min_qty`, по умолчанию 0 |
| ячейка | например `A-12` |
| артикул | SKU |
| заметки | опционально |

## Где лежит база

- **Windows:** `%APPDATA%\\VictimokLabs\\SkladUchet\\sklad.db`
- **Linux / macOS:** `~/.local/share/VictimokLabs/SkladUchet/sklad.db`

Миграции: `schema_version` (v3), `storage`/`theme_id`, таблица `themes`, `movements`. Демо-seed отсутствует.

## Требования

- Go **1.23+** (для сборки)
- Windows 10/11: **WebView2 Runtime** (обычно уже установлен с Edge)
- CGO **не нужен** — `modernc.org/sqlite`

## Запуск из исходников

```bash
git clone https://github.com/Johny-Silverhand/sklad-uchet.git
cd sklad-uchet
go mod tidy
go run . --app -no-browser -addr 127.0.0.1:17890
```

Откройте `http://127.0.0.1:17890/` в браузере.

## Сборка установщика Windows

```bash
./scripts/build-windows.sh
```

Или вручную:

```bash
mkdir -p dist
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w -H windowsgui" -o dist/SkladUchet-Setup.exe .
```

Готовые файлы:

- `dist/SkladUchet-Setup.exe` — установщик + приложение (один бинарник)
- `dist/SkladUchet.exe` — то же (алиас)
- `dist/SkladUchet-console.exe` — с консолью (отладка)

## API (локально)

| Метод | Путь | Описание |
|-------|------|----------|
| GET | `/api/items?q=&cell=&kind=&low=1` | список / поиск / мало |
| POST | `/api/items` | создать |
| PUT | `/api/items/{id}` | обновить |
| POST | `/api/items/{id}/adjust` | `{ "delta": ±1 }` |
| POST | `/api/items/{id}/move` | `{ "cell": "B-12" }` |
| DELETE | `/api/items/{id}` | удалить |
| GET | `/api/duplicates` | группы дубликатов |
| POST | `/api/duplicates/merge` | объединение |
| GET | `/api/export.csv` | экспорт CSV |
| POST | `/api/import.csv` | импорт (multipart `file` или body) |
| POST | `/api/backup` | `{ "path": "" }` |
| POST | `/api/restore` | `{ "path": "..." }` |
| GET | `/api/stats` | обзор |
| GET | `/api/movements` | журнал |

## Стек

- **Go** — сервер, установщик, бизнес-логика
- **modernc.org/sqlite** — локальная БД без CGO
- **embed HTML/CSS/JS** — UI приложения и мастера установки
- **WebView2** — нативное окно без браузерного хрома

## Лицензия

© Victimok Labs. **Разработано в Victimok Labs.** Использование по согласованию с автором репозитория. См. файл `LICENSE`.
