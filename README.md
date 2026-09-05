# Точка Склада

Локальный учёт склада для товароведа: запчасти, устройства, комплектующие — быстро, по-русски, **без интернета и без облака**.

Окно Windows через **WebView2** (не вкладка браузера). Данные в **SQLite (WAL)** на диске. HTTP только на `127.0.0.1` внутри процесса.

**Разработано в Victimok Labs.**

## Зачем это

Одна «точка» правды по остаткам: ячейки, темы, баланс и временное хранение, отчёты на печать, бэкап одной кнопкой. Поставил Setup — и работаешь офлайн.

## Возможности

- **Баланс** и **Временное хранение** + свои **темы**
- Поиск по названию и ячейке, мин. остаток, «Мало», быстрые +/−
- CSV, backup/restore, дубликаты и объединение
- **Сервис**: печать отчётов (тема / всё + дата), проверка БД, VACUUM, папка данных, журнал
- Пустой старт без демо-данных
- Установщик **`TochkaSklada-Setup.exe`** + нормальная иконка

## Установка (Windows 10/11)

1. Скачайте [`TochkaSklada-Setup.exe`](https://github.com/Johny-Silverhand/sklad-uchet/releases/latest) из Releases
2. Установите через тёмный мастер Victimok Labs
3. Запускайте ярлык **«Точка Склада»**

Нужен **Microsoft Edge WebView2 Runtime** (обычно уже стоит).

База: `%APPDATA%\VictimokLabs\SkladUchet\sklad.db` (сохраняется при удалении программы).

## Режимы exe

| Флаг | Что делает |
|------|------------|
| по умолчанию | setup, если ещё не установлен; иначе app |
| `--setup` | мастер установки |
| `--app` | обычный запуск |
| `--uninstall` | удаление |

## Сборка из исходников

```bash
git clone https://github.com/Johny-Silverhand/sklad-uchet.git
cd sklad-uchet
go mod tidy
go run . --app -no-browser -addr 127.0.0.1:17890
./scripts/build-windows.sh   # → dist/TochkaSklada-Setup.exe
```

Go **1.23+**, CGO не нужен (`modernc.org/sqlite`).

## Лицензия

© Victimok Labs. **Разработано в Victimok Labs.** См. `LICENSE`.
