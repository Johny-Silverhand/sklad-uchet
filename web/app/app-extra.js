async function loadMovements() {
  try {
    const data = await api("/api/movements?limit=40");
    const lines = (data.movements || []).map((m) => {
      const t = formatDate(m.created_at);
      return `${t}  #${m.item_id}  ${m.kind}  Δ${m.delta}  ${m.from_cell || "—"}→${m.to_cell || "—"}  ${m.note || ""}`;
    });
    $("#movementsBox").textContent = lines.length ? lines.join("\n") : "Пока нет движений.";
  } catch (e) {
    $("#movementsBox").textContent = e.message;
  }
}


function reportURL(extraLow) {
  const params = new URLSearchParams();
  const theme = $("#rptTheme")?.value || "all";
  const storage = $("#rptStorage")?.value || "all";
  const low = extraLow || ($("#rptLow")?.checked);
  if (theme && theme !== "all") params.set("theme_id", theme);
  if (storage && storage !== "all") params.set("storage", storage);
  if (low) params.set("low", "1");
  const q = params.toString();
  return "/api/report.html" + (q ? "?" + q : "");
}

function openReport(printAfter, forceLow) {
  const url = reportURL(forceLow);
  const w = window.open(url, "_blank");
  if (!w) {
    toast("Не удалось открыть окно отчёта", "err");
    return;
  }
  if (printAfter) {
    const tryPrint = () => {
      try { w.focus(); w.print(); } catch (_) {}
    };
    // WebView2 / Edge: print after load
    setTimeout(tryPrint, 400);
  }
}

async function loadServiceStats() {
  try {
    const st = await api("/api/stats");
    const by = st.by_storage || {};
    $("#svcTotal").textContent = st.total_items ?? 0;
    $("#svcQty").textContent = st.total_qty ?? 0;
    $("#svcLow").textContent = st.low_stock ?? 0;
    $("#svcStor").textContent = (by.balance ?? 0) + " / " + (by.temporary ?? 0);
  } catch (e) {
    $("#svcTotal").textContent = "—";
  }
}

async function loadPaths() {
  try {
    const p = await api("/api/paths");
    $("#pathsBox").textContent = "БД: " + (p.db || "—") + "\nПапка: " + (p.data_dir || "—") + "\nОС: " + (p.os || "—");
    window.__dataDir = p.data_dir || "";
    window.__dbPath = p.db || "";
  } catch (e) {
    $("#pathsBox").textContent = e.message;
  }
}

async function loadCrowded() {
  try {
    const data = await api("/api/cells/crowded?min=1");
    const cells = data.cells || [];
    if (!cells.length) {
      $("#crowdedBox").textContent = "Нет ячеек с более чем одной позицией.";
      return;
    }
    $("#crowdedBox").innerHTML = "<ul class=\"ov-list\">" + cells.map((c) =>
      `<li><b class="cell-code">${escapeHtml(c.cell)}</b> — ${c.count} поз., ${c.qty} шт.</li>`
    ).join("") + "</ul>";
  } catch (e) {
    $("#crowdedBox").textContent = e.message;
  }
}

async function loadServicePage() {
  fillThemeSelects();
  await Promise.all([
    loadServiceStats(),
    loadMovements(),
    loadPaths(),
    loadCrowded(),
    loadHealth(),
  ]);
}

function setListHeaders() {
  if (currentStorage === "balance") {
    $("#listTitle").textContent = "Баланс";
    $("#listSubtitle").textContent = "Постоянное хранение на складе";
  } else {
    $("#listTitle").textContent = "Временное хранение";
    $("#listSubtitle").textContent = "Позиции на временном складе";
  }
}

function switchView(name) {
  $$(".nav-btn").forEach((b) => b.classList.toggle("active", b.dataset.view === name));
  const isList = name === "balance" || name === "temporary";
  $("#view-list").hidden = !isList;
  $("#view-themes").hidden = name !== "themes";
  $("#view-overview").hidden = name !== "overview";
  $("#view-duplicates").hidden = name !== "duplicates";
  $("#view-tools").hidden = name !== "tools";
  if (name === "balance" || name === "temporary") {
    currentStorage = name === "balance" ? "balance" : "temporary";
    setListHeaders();
    loadItems().catch((e) => toast(e.message, "err"));
  }
  if (name === "themes") renderThemesView().catch((e) => toast(e.message, "err"));
  if (name === "duplicates") loadDuplicates().catch((e) => toast(e.message, "err"));
  if (name === "overview") loadOverview().catch((e) => toast(e.message, "err"));
  if (name === "tools") loadServicePage().catch((e) => toast(e.message, "err"));
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
  $("#moveCancel").addEventListener("click", () => $("#moveDialog").close());
  $("#themeCancel").addEventListener("click", () => $("#themeDialog").close());
  $("#btnAddTheme").addEventListener("click", () => openThemeDialog(null));
  $("#themeForm").addEventListener("submit", async (ev) => {
    ev.preventDefault();
    const id = $("#thId").value;
    const payload = { name: $("#thName").value, sort_order: Number($("#thSort").value) || 0 };
    try {
      if (id) {
        await api("/api/themes/" + id, { method: "PUT", body: JSON.stringify(payload) });
        toast("Тема обновлена", "ok");
      } else {
        await api("/api/themes", { method: "POST", body: JSON.stringify(payload) });
        toast("Тема создана", "ok");
      }
      $("#themeDialog").close();
      await renderThemesView();
      fillThemeSelects();
    } catch (e) { toast(e.message, "err"); }
  });

  $("#moveForm").addEventListener("submit", async (ev) => {
    ev.preventDefault();
    const id = $("#moveId").value;
    const cell = $("#moveCell").value;
    try {
      await api("/api/items/" + id + "/move", { method: "POST", body: JSON.stringify({ cell }) });
      $("#moveDialog").close();
      toast("Перемещено", "ok");
      await Promise.all([loadItems(), loadStats()]);
    } catch (e) { toast(e.message, "err"); }
  });

  const reload = debounce(() => loadItems().catch((e) => toast(e.message, "err")), 200);
  $("#qName").addEventListener("input", reload);
  $("#qCell").addEventListener("input", reload);
  $("#qKind").addEventListener("change", reload);
  $("#qTheme").addEventListener("change", reload);
  $("#qLow").addEventListener("change", reload);

  $("#itemsBody").addEventListener("click", async (ev) => {
    const adj = ev.target.closest("[data-adj]");
    const edit = ev.target.closest("[data-edit]");
    const del = ev.target.closest("[data-del]");
    const move = ev.target.closest("[data-move]");
    const stor = ev.target.closest("[data-storage]");
    if (adj) {
      try {
        await api("/api/items/" + adj.dataset.adj + "/adjust", {
          method: "POST",
          body: JSON.stringify({ delta: Number(adj.dataset.delta) }),
        });
        await Promise.all([loadItems(), loadStats()]);
      } catch (e) { toast(e.message, "err"); }
      return;
    }
    if (stor) {
      try {
        await api("/api/items/" + stor.dataset.storage + "/storage", {
          method: "POST",
          body: JSON.stringify({ storage: stor.dataset.to }),
        });
        toast(stor.dataset.to === "balance" ? "Перенесено на баланс" : "Во временное хранение", "ok");
        await Promise.all([loadItems(), loadStats()]);
      } catch (e) { toast(e.message, "err"); }
      return;
    }
    if (move) {
      $("#moveId").value = move.dataset.move;
      $("#moveName").textContent = move.dataset.name || "";
      $("#moveCell").value = move.dataset.cell || "";
      $("#moveDialog").showModal();
      $("#moveCell").focus();
      return;
    }
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
      if (!(await confirmDelete(id, name, "позицию"))) return;
      try {
        await api("/api/items/" + id, { method: "DELETE" });
        toast("Удалено", "ok");
        await Promise.all([loadItems(), loadStats()]);
      } catch (e) { toast(e.message, "err"); }
    }
  });

  $("#themesBox").addEventListener("click", async (ev) => {
    const edit = ev.target.closest("[data-edit-theme]");
    const del = ev.target.closest("[data-del-theme]");
    if (edit) {
      const t = themesCache.find((x) => String(x.id) === edit.dataset.editTheme);
      if (t) openThemeDialog(t);
      return;
    }
    if (del) {
      const t = themesCache.find((x) => String(x.id) === del.dataset.delTheme);
      if (!t) return;
      if (!(await confirmDelete(t.id, t.name, "тему"))) return;
      try {
        await api("/api/themes/" + t.id, { method: "DELETE" });
        toast("Тема удалена", "ok");
        await renderThemesView();
        fillThemeSelects();
      } catch (e) { toast(e.message, "err"); }
    }
  });

  ["fName", "fKind", "fStorage", "fQty", "fMin", "fCell", "fSku", "fNotes", "fTheme"].forEach((id) => {
    const el = $("#" + id);
    if (!el) return;
    el.addEventListener("input", () => {
      forceSave = false;
      $("#btnSave").textContent = "Сохранить";
      $("#dlgWarn").hidden = true;
    });
    el.addEventListener("change", () => {
      forceSave = false;
      $("#btnSave").textContent = "Сохранить";
      $("#dlgWarn").hidden = true;
    });
  });

  $("#importFile").addEventListener("change", async (ev) => {
    const file = ev.target.files?.[0];
    if (!file) return;
    try {
      const fd = new FormData();
      fd.append("file", file);
      const res = await fetch("/api/import.csv?update=1", { method: "POST", body: fd });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error || "Ошибка импорта");
      toast(`Импорт: +${data.created} / ~${data.updated} / skip ${data.skipped}`, "ok");
      await Promise.all([loadThemes(), loadItems(), loadStats()]);
    } catch (e) { toast(e.message, "err"); }
    ev.target.value = "";
  });

  $("#btnBackup").addEventListener("click", async () => {
    try {
      const path = $("#backupPath").value.trim();
      const data = await api("/api/backup", { method: "POST", body: JSON.stringify({ path }) });
      toast("Backup: " + data.path, "ok");
    } catch (e) { toast(e.message, "err"); }
  });

  $("#btnRestore").addEventListener("click", async () => {
    const path = $("#restorePath").value.trim();
    if (!path) { toast("Укажите путь к .db", "err"); return; }
    if (!confirm("Восстановить БД из файла? Текущая будет сохранена как .before-restore")) return;
    try {
      await api("/api/restore", { method: "POST", body: JSON.stringify({ path }) });
      toast("Восстановлено", "ok");
      await Promise.all([loadThemes(), loadItems(), loadStats(), loadHealth()]);
    } catch (e) { toast(e.message, "err"); }
  });

  $("#btnMovements").addEventListener("click", () => loadMovements());

  $("#btnReportPrint")?.addEventListener("click", () => openReport(true, false));
  $("#btnReportOpen")?.addEventListener("click", () => openReport(false, false));
  $("#btnReportLow")?.addEventListener("click", () => openReport(true, true));

  $("#btnCrowded")?.addEventListener("click", () => loadCrowded().catch((e) => toast(e.message, "err")));

  $("#btnIntegrity")?.addEventListener("click", async () => {
    try {
      const data = await api("/api/db/integrity", { method: "POST", body: "{}" });
      $("#dbMaintBox").textContent = data.ok ? "Целостность: OK" : ("Результат: " + (data.result || "?"));
      toast(data.ok ? "БД в порядке" : "Проблемы целостности", data.ok ? "ok" : "err");
    } catch (e) { toast(e.message, "err"); }
  });

  $("#btnVacuum")?.addEventListener("click", async () => {
    if (!confirm("Сжать базу (VACUUM)? Операция может занять несколько секунд.")) return;
    try {
      await api("/api/db/vacuum", { method: "POST", body: "{}" });
      $("#dbMaintBox").textContent = "VACUUM выполнен.";
      toast("VACUUM готов", "ok");
      await loadPaths();
    } catch (e) { toast(e.message, "err"); }
  });

  $("#btnRefreshPaths")?.addEventListener("click", () => loadPaths());
  $("#btnCopyPath")?.addEventListener("click", async () => {
    const text = window.__dataDir || window.__dbPath || "";
    if (!text) { toast("Путь неизвестен", "err"); return; }
    try {
      await navigator.clipboard.writeText(text);
      toast("Путь скопирован", "ok");
    } catch {
      prompt("Скопируйте путь:", text);
    }
  });
  $("#btnRevealData")?.addEventListener("click", async () => {
    try {
      const data = await api("/api/reveal-data", { method: "POST", body: "{}" });
      toast("Открыто: " + (data.path || ""), "ok");
    } catch (e) { toast(e.message, "err"); }
  });

  $("#btnClearMovements")?.addEventListener("click", async () => {
    if (!confirm("Очистить журнал движений? Позиции склада не удаляются.")) return;
    try {
      const data = await api("/api/movements?clear=1", { method: "POST", body: "{}" });
      toast("Удалено записей: " + (data.deleted ?? 0), "ok");
      await loadMovements();
    } catch (e) { toast(e.message, "err"); }
  });
}

async function init() {
  document.title = "Точка Склада";
  bind();
  setListHeaders();
  await Promise.all([loadHealth(), loadThemes(), loadStats(), loadItems()]);
}

init().catch((e) => toast(e.message || String(e), "err"));
