# Склад Учёт

Десктопное приложение для товароведа склада: учёт **запчастей**, **устройств** и **комплектующих**.

Интерфейс на русском языке. Данные хранятся локально в SQLite (режим WAL).

**Разработано в Victimok Labs.**

## Возможности

- CRUD: список, добавление, редактирование, удаление
- Поиск по названию (подстрока) и по номеру ячейки (префикс)
- Фильтр по типу: Запчасть / Устройство / Комплектующее
- Предупреждение о дубликатах названий (нормализация: trim, lower, схлопывание пробелов)
- Экран **«Дубликаты»**: группы совпадений и **объединение** (сумма количеств, лучший SKU/заметки; разные ячейки → основная + «Также: …» в заметках)
- Win11-friendly UI: локальный HTTP + окно Edge/Chrome в режиме `--app=`

## Поля позиции

| Поле | Описание |
|------|----------|
| название | name |
| тип | `zapchast` / `ustroystvo` / `komplektuyushchee` |
| кол-во | целое ≥ 0 |
| ячейка | строковый номер, например `A-12` |
| артикул | SKU (опционально) |
| заметки | опционально |
| created_at / updated_at | автоматически |

## Где лежит база

- **Windows:** `%APPDATA%\\VictimokLabs\\SkladUchet\\sklad.db`
- **Linux / macOS:** `~/.local/share/VictimokLabs/SkladUchet/sklad.db` (или `$XDG_DATA_HOME/...`)

Индексы: нормализованное имя (`name_norm`), ячейка (`cell`). Включён `PRAGMA journal_mode=WAL`.

## Требования

- Go **1.23+**
- Windows 11: Microsoft Edge или Google Chrome (для окна приложения)
- CGO **не нужен** — используется `modernc.org/sqlite` (pure Go)

## Запуск из исходников

```bash
git clone https://github.com/Johny-Silverhand/sklad-uchet.git
cd sklad-uchet
go mod tidy
go run .
```

Флаги:

- `-no-browser` — только сервер (удобно для отладки в браузере)
- `-addr 127.0.0.1:17890` — фиксированный порт

Пример без окна:

```bash
go run . -no-browser -addr 127.0.0.1:17890
```

Откройте в браузере адрес, который выведет программа (например `http://127.0.0.1:17890/`).

## Сборка .exe для Windows 11

На любой ОС с Go (кросс-компиляция):

```bash
mkdir -p dist
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w -H windowsgui" -o dist/SkladUchet.exe .
```

- `-H windowsgui` скрывает консольное окно (только Windows).
- Для отладки с консолью уберите `-H windowsgui`:

```bash
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o dist/SkladUchet.exe .
```

Готовый файл: `dist/SkladUchet.exe`. Скопируйте на ПК с Windows 11 и запустите двойным щелчком. Нужен установленный Edge или Chrome.

Сборка **на** Windows:

```powershell
go build -ldflags="-s -w -H windowsgui" -o dist\\SkladUchet.exe .
```

## Стек

- **Go** — сервер и бизнес-логика
- **modernc.org/sqlite** — локальная БД без CGO
- **встроенный HTML/CSS/JS** (`embed`) — UI
- **Edge/Chrome `--app=`** — окно как у десктоп-приложения (тот же подход, что у ochag desktop/win)

## API (локально)

| Метод | Путь | Описание |
|-------|------|----------|
| GET | `/api/items?q=&cell=&kind=` | список / поиск |
| POST | `/api/items` | создать (`force: true` — игнор предупреждения о дублях) |
| PUT | `/api/items/{id}` | обновить |
| DELETE | `/api/items/{id}` | удалить |
| GET | `/api/duplicates` | группы дубликатов |
| POST | `/api/duplicates/merge` | `{ "primary_id": 1, "other_ids": [2,3] }` |

## Лицензия

© Victimok Labs. Использование по согласованию с автором репозитория.
