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
    const d = new Date(iso);
    return d.toLocaleString("ru-RU", { timeZone: "Europe/Moscow" });
  } catch { return iso; }
}

async function loadStats() {
  try {
    const st = await api("/api/stats");
    $("#stItems").textContent = st.total_items ?? 0;
    $("#stQty").textContent = st.total_qty ?? 0;
  } catch {}
}

async function loadHealth() {
  try {
    const h = await api("/api/health");
    $("#dbPath").textContent = h.db || "";
  } catch {}
}

async function loadItems() {
  const q = $("#qName").value.trim();
  const cell = $("#qCell").value.trim();
  const kind = $("#qKind").value;
  const params = new URLSearchParams();
  if (q) params.set("q", q);
  if (cell) params.set("cell", cell);
  if (kind && kind !== "all") params.set("kind", kind);
  const data = await api("/api/items?" + params.toString());
  const body = $("#itemsBody");
  body.innerHTML = "";
  const items = data.items || [];
  $("#emptyList").hidden = items.length > 0;
  for (const it of items) {
    const tr = document.createElement("tr");
    tr.innerHTML = `
      <td>
        <div><strong>${escapeHtml(it.name)}</strong></div>
        <div class="muted">${formatDate(it.updated_at)}</div>
      </td>
      <td><span class="badge ${escapeHtml(it.kind)}">${KIND_LABEL[it.kind] || it.kind}</span></td>
      <td><strong>${it.quantity}</strong></td>
      <td class="cell-code">${escapeHtml(it.cell) || "—"}</td>
      <td>${escapeHtml(it.sku) || "—"}</td>
      <td class="muted">${escapeHtml((it.notes || "").slice(0, 80))}${(it.notes || "").length > 80 ? "…" : ""}</td>
      <td>
        <div class="actions">
          <button class="btn sm" data-edit="${it.id}">Изменить</button>
          <button class="btn sm danger" data-del="${it.id}">Удалить</button>
        </div>
      </td>`;
    body.appendChild(tr);
  }
}

function openItemDialog(item) {
  forceSave = false;
  $("#dlgWarn").hidden = true;
  $("#dlgWarn").textContent = "";
  $("#dlgTitle").textContent = item ? "Редактирование" : "Новая позиция";
  $("#fId").value = item?.id || "";
  $("#fName").value = item?.name || "";
  $("#fKind").value = item?.kind || "zapchast";
  $("#fQty").value = item?.quantity ?? 0;
  $("#fCell").value = item?.cell || "";
  $("#fSku").value = item?.sku || "";
  $("#fNotes").value = item?.notes || "";
  $("#btnSave").textContent = "Сохранить";
  $("#itemDialog").showModal();
  $("#fName").focus();
}

async function saveItem(ev) {
  ev.preventDefault();
  const id = $("#fId").value;
  const payload = {
    name: $("#fName").value,
    kind: $("#fKind").value,
    quantity: Number($("#fQty").value) || 0,
    cell: $("#fCell").value,
    sku: $("#fSku").value,
    notes: $("#fNotes").value,
    force: forceSave,
  };
  try {
    if (id) {
      await api("/api/items/" + id, { method: "PUT", body: JSON.stringify(payload) });
      toast("Сохранено", "ok");
    } else {
      await api("/api/items", { method: "POST", body: JSON.stringify(payload) });
      toast("Добавлено", "ok");
    }
    $("#itemDialog").close();
    await Promise.all([loadItems(), loadStats()]);
  } catch (err) {
    if (err.status === 409 && err.data?.duplicates?.length) {
      const names = err.data.duplicates.map((d) => `#${d.id} «${d.name}» (${d.cell || "—"})`).join(", ");
      $("#dlgWarn").hidden = false;
      $("#dlgWarn").innerHTML =
        `Уже есть похожие: ${escapeHtml(names)}.<br>Нажмите ещё раз, чтобы сохранить всё равно, или объедините в «Дубликаты».`;
      forceSave = true;
      $("#btnSave").textContent = "Сохранить всё равно";
      return;
    }
    toast(err.message || "Ошибка", "err");
  }
}

function confirmDelete(id, name) {
  return new Promise((resolve) => {
    $("#confirmTitle").textContent = "Удалить позицию?";
    $("#confirmText").textContent = `«${name}» будет удалена безвозвратно.`;
    $("#confirmYes").textContent = "Удалить";
    const dlg = $("#confirmDialog");
    const onYes = () => { cleanup(); resolve(true); };
    const onNo = () => { cleanup(); resolve(false); };
    function cleanup() {
      $("#confirmYes").removeEventListener("click", onYes);
      $("#confirmNo").removeEventListener("click", onNo);
      dlg.close();
    }
    $("#confirmYes").addEventListener("click", onYes);
    $("#confirmNo").addEventListener("click", onNo);
    dlg.showModal();
  });
}

async function loadDuplicates() {
  const data = await api("/api/duplicates");
  const box = $("#dupsBox");
  box.innerHTML = "";
  const groups = data.groups || [];
  $("#emptyDups").hidden = groups.length > 0;
  for (const g of groups) {
    const card = document.createElement("div");
    card.className = "dup-card";
    const ids = g.items.map((i) => i.id);
    card.innerHTML = `
      <div class="dup-head">
        <h3>«${escapeHtml(g.normalized_name)}» · ${g.items.length} шт. · сумма ${g.total_qty}</h3>
        <button class="btn primary sm" data-merge>Объединить</button>
      </div>
      <div class="dup-list">
        ${g.items.map((it, idx) => `
          <label class="dup-item ${idx === 0 ? "primary" : ""}">
            <input type="radio" name="pri-${g.normalized_name}" value="${it.id}" ${idx === 0 ? "checked" : ""} />
            <div>
              <strong>${escapeHtml(it.name)}</strong>
              <div class="muted">${KIND_LABEL[it.kind] || it.kind} · ячейка ${escapeHtml(it.cell) || "—"} · SKU ${escapeHtml(it.sku) || "—"}</div>
              <div class="muted">${escapeHtml(it.notes || "")}</div>
            </div>
            <div><strong>${it.quantity}</strong></div>
            <div class="muted">#${it.id}</div>
          </label>`).join("")}
      </div>`;
    card.querySelector("[data-merge]").addEventListener("click", async () => {
      const primary = Number(card.querySelector('input[type=radio]:checked')?.value);
      const others = ids.filter((id) => id !== primary);
      try {
        await api("/api/duplicates/merge", {
          method: "POST",
          body: JSON.stringify({ primary_id: primary, other_ids: others }),
        });
        toast("Объединено: количества суммированы", "ok");
        await Promise.all([loadDuplicates(), loadItems(), loadStats()]);
      } catch (err) {
        toast(err.message || "Ошибка объединения", "err");
      }
    });
    card.querySelectorAll('input[type=radio]').forEach((r) => {
      r.addEventListener("change", () => {
        card.querySelectorAll(".dup-item").forEach((el) => el.classList.remove("primary"));
        r.closest(".dup-item").classList.add("primary");
      });
    });
    box.appendChild(card);
  }
}

function switchView(name) {
  $$(".nav-btn").forEach((b) => b.classList.toggle("active", b.dataset.view === name));
  $("#view-list").hidden = name !== "list";
  $("#view-duplicates").hidden = name !== "duplicates";
  if (name === "duplicates") loadDuplicates().catch((e) => toast(e.message, "err"));
}

function debounce(fn, ms) {
  let t;
  return (...args) => {
    clearTimeout(t);
    t = setTimeout(() => fn(...args), ms);
  };
}

function bind() {
  $$(".nav-btn").forEach((b) => b.addEventListener("click", () => switchView(b.dataset.view)));
  $("#btnAdd").addEventListener("click", () => openItemDialog(null));
  $("#btnCancel").addEventListener("click", () => $("#itemDialog").close());
  $("#itemForm").addEventListener("submit", saveItem);
  $("#btnRefreshDups").addEventListener("click", () => loadDuplicates().catch((e) => toast(e.message, "err")));

  const reload = debounce(() => loadItems().catch((e) => toast(e.message, "err")), 200);
  $("#qName").addEventListener("input", reload);
  $("#qCell").addEventListener("input", reload);
  $("#qKind").addEventListener("change", reload);

  $("#itemsBody").addEventListener("click", async (ev) => {
    const edit = ev.target.closest("[data-edit]");
    const del = ev.target.closest("[data-del]");
    if (edit) {
      try {
        const data = await api("/api/items/" + edit.dataset.edit);
        openItemDialog(data.item);
      } catch (e) { toast(e.message, "err"); }
    }
    if (del) {
      const id = del.dataset.del;
      const row = del.closest("tr");
      const name = row?.querySelector("strong")?.textContent || id;
      if (!(await confirmDelete(id, name))) return;
      try {
        await api("/api/items/" + id, { method: "DELETE" });
        toast("Удалено", "ok");
        await Promise.all([loadItems(), loadStats()]);
      } catch (e) { toast(e.message, "err"); }
    }
  });

  ["fName", "fKind", "fQty", "fCell", "fSku", "fNotes"].forEach((id) => {
    $("#" + id).addEventListener("input", () => {
      forceSave = false;
      $("#btnSave").textContent = "Сохранить";
      $("#dlgWarn").hidden = true;
    });
  });
}

async function init() {
  bind();
  await Promise.all([loadHealth(), loadStats(), loadItems()]);
}

init().catch((e) => toast(e.message || String(e), "err"));
