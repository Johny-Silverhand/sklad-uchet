const KIND_LABEL = {
  zapchast: "Запчасть",
  ustroystvo: "Устройство",
  komplektuyushchee: "Комплектующее",
};

const $ = (sel) => document.querySelector(sel);
const $$ = (sel) => [...document.querySelectorAll(sel)];

let forceSave = false;
let toastTimer = null;

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

async function loadStats() {
  try {
    const st = await api("/api/stats");
    $("#stItems").textContent = st.total_items ?? 0;
    $("#stQty").textContent = st.total_qty ?? 0;
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
  const low = $("#qLow").checked;
  const params = new URLSearchParams();
  if (q) params.set("q", q);
  if (cell) params.set("cell", cell);
  if (kind && kind !== "all") params.set("kind", kind);
  if (low) params.set("low", "1");
  const data = await api("/api/items?" + params.toString());
  const body = $("#itemsBody");
  body.innerHTML = "";
  const items = data.items || [];
  $("#emptyList").hidden = items.length > 0;
  for (const it of items) {
    const tr = document.createElement("tr");
    if (it.low_stock) tr.classList.add("low-stock");
    tr.innerHTML = `
      <td>
        <div><strong>${escapeHtml(it.name)}</strong>
          ${it.low_stock ? '<span class="badge low">мало</span>' : ""}
        </div>
        <div class="muted">${formatDate(it.updated_at)}</div>
      </td>
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
          <button class="btn sm" data-move="${it.id}" data-name="${escapeHtml(it.name)}" data-cell="${escapeHtml(it.cell)}">Ячейка</button>
          <button class="btn sm" data-edit="${it.id}">Изменить</button>
          <button class="btn sm danger" data-del="${it.id}">Удалить</button>
        </div>
      </td>`;
    body.appendChild(tr);
  }
}

async function loadOverview() {
  const st = await api("/api/stats");
  const box = $("#overviewBox");
  const kinds = st.by_kind || {};
  const qtys = st.qty_by_kind || {};
  const cells = st.top_cells || [];
  box.innerHTML = `
    <div class="ov-grid">
      <div class="ov-card"><h4>Позиций</h4><div class="big">${st.total_items || 0}</div></div>
      <div class="ov-card"><h4>Всего шт.</h4><div class="big">${st.total_qty || 0}</div></div>
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
