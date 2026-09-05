package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func (s *Store) DataDir() string {
	if s.dbPath != "" {
		return filepath.Dir(s.dbPath)
	}
	dir, _ := dataDir()
	return dir
}

func (s *Store) IntegrityCheck() (string, error) {
	var result string
	if err := s.db.QueryRow(`PRAGMA integrity_check`).Scan(&result); err != nil {
		return "", err
	}
	return result, nil
}

func (s *Store) Vacuum() error {
	_, err := s.db.Exec(`VACUUM`)
	return err
}

func (s *Store) ClearMovements() (int64, error) {
	res, err := s.db.Exec(`DELETE FROM movements`)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// CrowdedCells returns cells that hold more than minCount positions (default 1 → cells with >1).
func (s *Store) CrowdedCells(minCount int) ([]CellStat, error) {
	if minCount < 1 {
		minCount = 1
	}
	rows, err := s.db.Query(`
SELECT cell, COUNT(*) AS cnt, COALESCE(SUM(quantity),0) AS qty
FROM items
WHERE TRIM(cell) != ''
GROUP BY cell
HAVING COUNT(*) > ?
ORDER BY cnt DESC, qty DESC
LIMIT 50`, minCount)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []CellStat{}
	for rows.Next() {
		var cs CellStat
		if err := rows.Scan(&cs.Cell, &cs.Count, &cs.Qty); err != nil {
			return nil, err
		}
		out = append(out, cs)
	}
	return out, rows.Err()
}

func revealPath(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("путь пуст")
	}
	if st, err := os.Stat(path); err != nil {
		return fmt.Errorf("путь не найден: %w", err)
	} else if !st.IsDir() {
		path = filepath.Dir(path)
	}
	if runtime.GOOS != "windows" {
		return fmt.Errorf("открытие папки поддерживается только в Windows")
	}
	cmd := exec.Command("explorer", path)
	cmd.SysProcAttr = hideWindow()
	return cmd.Start()
}

func kindLabelRU(k string) string {
	switch k {
	case KindZapchast:
		return "Запчасть"
	case KindUstroystvo:
		return "Устройство"
	case KindKomplektuyushchee:
		return "Комплектующее"
	default:
		return k
	}
}

func storageLabelRU(st string) string {
	switch st {
	case StorageBalance:
		return "Баланс"
	case StorageTemporary:
		return "Временное"
	default:
		return st
	}
}

func htmlEscape(s string) string {
	r := strings.NewReplacer(
		`&`, "&amp;",
		`<`, "&lt;",
		`>`, "&gt;",
		`"`, "&quot;",
	)
	return r.Replace(s)
}

func buildReportHTML(items []Item, filterDesc, generatedISO, generatedLocal string) string {
	var b strings.Builder
	b.WriteString(`<!DOCTYPE html><html lang="ru"><head><meta charset="UTF-8"/>`)
	b.WriteString(`<title>Отчёт — Склад Учёт</title>`)
	b.WriteString(`<style>
@page { size: A4; margin: 14mm; }
* { box-sizing: border-box; }
body { font-family: "Segoe UI", Arial, sans-serif; color: #111; background: #fff; margin: 0; padding: 16px; font-size: 11pt; }
h1 { margin: 0 0 4px; font-size: 18pt; }
.meta { color: #333; margin: 0 0 12px; font-size: 10pt; }
.credit { color: #055; margin: 16px 0 8px; font-weight: 600; }
table { width: 100%; border-collapse: collapse; }
th, td { border: 1px solid #bbb; padding: 5px 7px; text-align: left; vertical-align: top; }
th { background: #eee; font-size: 9.5pt; }
tr.low td { background: #fff3cd; }
.totals { margin-top: 12px; font-size: 10.5pt; }
.noprint { margin: 12px 0; }
@media print {
  .noprint { display: none !important; }
  body { padding: 0; }
  a { color: inherit; text-decoration: none; }
}
</style></head><body>`)
	b.WriteString(`<div class="noprint"><button onclick="window.print()">Печать</button> `)
	b.WriteString(`<button onclick="window.close()">Закрыть</button></div>`)
	b.WriteString(`<h1>Склад Учёт — отчёт</h1>`)
	b.WriteString(`<p class="meta">Дата: <strong>`)
	b.WriteString(htmlEscape(generatedLocal))
	b.WriteString(`</strong> <span style="color:#666">(`)
	b.WriteString(htmlEscape(generatedISO))
	b.WriteString(`)</span><br/>Фильтры: `)
	b.WriteString(htmlEscape(filterDesc))
	b.WriteString(`</p>`)
	b.WriteString(`<p class="credit">Разработано в Victimok Labs</p>`)
	b.WriteString(`<table><thead><tr>`)
	b.WriteString(`<th>#</th><th>Название</th><th>Тема</th><th>Тип</th><th>Хранение</th><th>Кол-во</th><th>Мин</th><th>Ячейка</th><th>Артикул</th>`)
	b.WriteString(`</tr></thead><tbody>`)

	totalQty := 0
	lowN := 0
	for i, it := range items {
		totalQty += it.Quantity
		cls := ""
		if it.LowStock {
			cls = ` class="low"`
			lowN++
		}
		theme := it.ThemeName
		if theme == "" {
			theme = "Без темы"
		}
		fmt.Fprintf(&b, `<tr%s><td>%d</td><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td>%d</td><td>%d</td><td>%s</td><td>%s</td></tr>`,
			cls,
			i+1,
			htmlEscape(it.Name),
			htmlEscape(theme),
			htmlEscape(kindLabelRU(it.Kind)),
			htmlEscape(storageLabelRU(it.Storage)),
			it.Quantity,
			it.MinQty,
			htmlEscape(it.Cell),
			htmlEscape(it.SKU),
		)
	}
	if len(items) == 0 {
		b.WriteString(`<tr><td colspan="9">Нет позиций по выбранным фильтрам.</td></tr>`)
	}
	b.WriteString(`</tbody></table>`)
	fmt.Fprintf(&b, `<p class="totals"><strong>Итого:</strong> позиций — %d; сумма qty — %d; мало — %d</p>`,
		len(items), totalQty, lowN)
	b.WriteString(`<script>
(function(){
  try {
    var el = document.querySelector('.meta strong');
    if (el) el.textContent = new Date().toLocaleString('ru-RU', { timeZone: 'Europe/Moscow' });
  } catch (e) {}
})();
</script>`)
	b.WriteString(`</body></html>`)
	return b.String()
}
