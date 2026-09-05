const KIND_LABEL = {
  zapchast: "Запчасть",
  ustroystvo: "Устройство",
  komplektuyushchee: "Комплектующее",
};

const STORAGE_LABEL = {
  balance: "Баланс",
  temporary: "Временное",
};

const $ = (sel) => document.querySelector(sel);
const $$ = (sel) => [...document.querySelectorAll(sel)];

let forceSave = false;
let toastTimer = null;
let currentStorage = "balance"; // balance | temporary
let themesCache = [];

async function api(path, opts = {}) {
  const res = await fetch(path, {
    headers: { "Content-Type": "application/json", ...(opts.headers || {}) },
    ...opts,
  });
  let data = null;
  const text = await res.text();
  try { data = text ? JSON.parse(text) : null; } catch { data = { error: text }; }
  if (!res.ok) {
    const err = new Error(data?.error || res.statusText);
    err.status = res.status;
    err.data = data;
    throw err;
  }
  return data;
}

function toast(msg, kind = "") {
  const el = $("#toast");
  el.textContent = msg;
  el.className = "toast " + kind;
  el.hidden = false;
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => { el.hidden = true; }, 3200);
}

function escapeHtml(s) {
  return String(s ?? "")
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}

function formatDate(iso) {
  if (!iso) return "";
  try {
    return new Date(iso).toLocaleString("ru-RU", { timeZone: "Europe/Moscow" });
  } catch { return iso; }
}

async function loadThemes() {
  const data = await api("/api/themes");
  themesCache = data.themes || [];
  fillThemeSelects();
  return themesCache;
}

function fillThemeSelects() {
  const q = $("#qTheme");
  const f = $("#fTheme");
  const r = $("#rptTheme");
  const qVal = q.value;
  const fVal = f.value;
  const rVal = r ? r.value : "all";
  const themeOpts = themesCache.map((t) => `<option value="${t.id}">${escapeHtml(t.name)}</option>`).join("");
  q.innerHTML = `<option value="all">Все темы</option><option value="none">Без темы</option>` + themeOpts;
  f.innerHTML = `<option value="">Без темы</option>` + themeOpts;
  if (r) {
    r.innerHTML = `<option value="all">Все</option><option value="none">Без темы</option>` + themeOpts;
    if ([...r.options].some((o) => o.value === rVal)) r.value = rVal;
  }
  if ([...q.options].some((o) => o.value === qVal)) q.value = qVal;
  if ([...f.options].some((o) => o.value === fVal)) f.value = fVal;
}

async function loadStats() {
  try {
    const st = await api("/api/stats");
    const by = st.by_storage || {};
    $("#stBalance").textContent = by.balance ?? 0;
    $("#stTemp").textContent = by.temporary ?? 0;
    $("#stLow").textContent = st.low_stock ?? 0;
    return st;
  } catch { return null; }
}

async function loadHealth() {
  try {
    const h = await api("/api/health");
    $("#dbPath").textContent = h.db || "";
    if (h.version) $("#aboutVer").textContent = h.version;
  } catch {}
}

async function loadItems() {
  const q = $("#qName").value.trim();
  const cell = $("#qCell").value.trim();
  const kind = $("#qKind").value;
  const theme = $("#qTheme").value;
  const low = $("#qLow").checked;
  const params = new URLSearchParams();
  params.set("storage", currentStorage);
  if (q) params.set("q", q);
  if (cell) params.set("cell", cell);
  if (kind && kind !== "all") params.set("kind", kind);
  if (theme && theme !== "all") params.set("theme_id", theme);
  if (low) params.set("low", "1");
  const data = await api("/api/items?" + params.toString());
  const body = $("#itemsBody");
  body.innerHTML = "";
  const items = data.items || [];
  $("#emptyList").hidden = items.length > 0;

  // Group by theme name
  const groups = new Map();
  for (const it of items) {
    const key = it.theme_name || "Без темы";
    if (!groups.has(key)) groups.set(key, []);
    groups.get(key).push(it);
  }
  for (const [themeName, list] of groups) {
    const hr = document.createElement("tr");
    hr.className = "theme-group";
    hr.innerHTML = `<td colspan="8"><strong>${escapeHtml(themeName)}</strong> · ${list.length}</td>`;
    body.appendChild(hr);
    for (const it of list) {
      const tr = document.createElement("tr");
      if (it.low_stock) tr.classList.add("low-stock");
      const otherStorage = it.storage === "balance" ? "temporary" : "balance";
      const moveLabel = otherStorage === "balance" ? "→ Баланс" : "→ Временное";
      tr.innerHTML = `
        <td>
          <div><strong>${escapeHtml(it.name)}</strong>
            ${it.low_stock ? '<span class="badge low">мало</span>' : ""}
          </div>
          <div class="muted">${formatDate(it.updated_at)}</div>
        </td>
        <td><span class="badge theme">${escapeHtml(it.theme_name || "Без темы")}</span></td>
        <td><span class="badge ${escapeHtml(it.kind)}">${KIND_LABEL[it.kind] || it.kind}</span></td>
        <td>
          <div class="qty-ctrl">
            <button class="btn sm" data-adj="${it.id}" data-delta="-1" title="-1">−</button>
            <strong>${it.quantity}</strong>
            <button class="btn sm" data-adj="${it.id}" data-delta="1" title="+1">+</button>
          </div>
        </td>
        <td>${it.min_qty || 0}</td>
        <td class="cell-code">${escapeHtml(it.cell) || "—"}</td>
        <td>${escapeHtml(it.sku) || "—"}</td>
        <td>
          <div class="actions">
            <button class="btn sm" data-storage="${it.id}" data-to="${otherStorage}">${moveLabel}</button>
            <button class="btn sm" data-move="${it.id}" data-name="${escapeHtml(it.name)}" data-cell="${escapeHtml(it.cell)}">Ячейка</button>
            <button class="btn sm" data-edit="${it.id}">Изменить</button>
            <button class="btn sm danger" data-del="${it.id}">Удалить</button>
          </div>
        </td>`;
      body.appendChild(tr);
    }
  }
}

async function loadOverview() {
  const st = await api("/api/stats");
  const box = $("#overviewBox");
  const kinds = st.by_kind || {};
  const qtys = st.qty_by_kind || {};
  const bySt = st.by_storage || {};
  const qtySt = st.qty_by_storage || {};
  const cells = st.top_cells || [];
  box.innerHTML = `
    <div class="ov-grid">
      <div class="ov-card"><h4>Баланс</h4><div class="big">${bySt.balance || 0}</div><div class="muted">${qtySt.balance || 0} шт.</div></div>
      <div class="ov-card"><h4>Временное</h4><div class="big">${bySt.temporary || 0}</div><div class="muted">${qtySt.temporary || 0} шт.</div></div>
      <div class="ov-card"><h4>Мало на складе</h4><div class="big" style="color:var(--warn)">${st.low_stock || 0}</div></div>
    </div>
    <div class="ov-grid">
      <div class="ov-card"><h4>Запчасти</h4><div class="big">${kinds.zapchast || 0}</div><div class="muted">${qtys.zapchast || 0} шт.</div></div>
      <div class="ov-card"><h4>Устройства</h4><div class="big">${kinds.ustroystvo || 0}</div><div class="muted">${qtys.ustroystvo || 0} шт.</div></div>
      <div class="ov-card"><h4>Комплектующие</h4><div class="big">${kinds.komplektuyushchee || 0}</div><div class="muted">${qtys.komplektuyushchee || 0} шт.</div></div>
    </div>
    <div class="ov-card">
      <h4>Топ ячеек</h4>
      ${cells.length ? `<ul class="ov-list">${cells.map(c => `<li><b class="cell-code">${escapeHtml(c.cell)}</b> — ${c.count} поз., ${c.qty} шт.</li>`).join("")}</ul>` : `<p class="muted">Пока нет заполненных ячеек.</p>`}
    </div>
    <p class="credit-line">Разработано в Victimok Labs</p>`;
}

async function renderThemesView() {
  await loadThemes();
  const box = $("#themesBox");
  box.innerHTML = "";
  $("#emptyThemes").hidden = themesCache.length > 0;
  for (const t of themesCache) {
    const card = document.createElement("div");
    card.className = "tool-card";
    card.innerHTML = `
      <div class="dup-head">
        <div>
          <h3>${escapeHtml(t.name)}</h3>
          <p class="muted">Позиций: ${t.item_count} · порядок ${t.sort_order}</p>
        </div>
        <div class="actions">
          <button class="btn sm" data-edit-theme="${t.id}">Переименовать</button>
          <button class="btn sm danger" data-del-theme="${t.id}">Удалить</button>
        </div>
      </div>`;
    box.appendChild(card);
  }
}
