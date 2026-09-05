function openItemDialog(item) {
  forceSave = false;
  $("#dlgWarn").hidden = true;
  $("#dlgWarn").textContent = "";
  $("#dlgTitle").textContent = item ? "Редактирование" : "Новая позиция";
  $("#fId").value = item?.id || "";
  $("#fName").value = item?.name || "";
  $("#fKind").value = item?.kind || "zapchast";
  $("#fQty").value = item?.quantity ?? 0;
  $("#fMin").value = item?.min_qty ?? 0;
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
    min_qty: Number($("#fMin").value) || 0,
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
            <input type="radio" name="pri-${escapeHtml(g.normalized_name)}" value="${it.id}" ${idx === 0 ? "checked" : ""} />
            <div>
              <strong>${escapeHtml(it.name)}</strong>
              <div class="muted">${KIND_LABEL[it.kind] || it.kind} · ячейка ${escapeHtml(it.cell) || "—"} · SKU ${escapeHtml(it.sku) || "—"} · мин ${it.min_qty || 0}</div>
            </div>
            <div><strong>${it.quantity}</strong></div>
            <div class="muted">#${it.id}</div>
          </label>`).join("")}
      </div>`;
    card.querySelector("[data-merge]").addEventListener("click", async () => {
      const primary = Number(card.querySelector("input[type=radio]:checked")?.value);
      const others = ids.filter((id) => id !== primary);
      try {
        await api("/api/duplicates/merge", {
          method: "POST",
          body: JSON.stringify({ primary_id: primary, other_ids: others }),
        });
        toast("Объединено", "ok");
        await Promise.all([loadDuplicates(), loadItems(), loadStats()]);
      } catch (err) {
        toast(err.message || "Ошибка объединения", "err");
      }
    });
    card.querySelectorAll("input[type=radio]").forEach((r) => {
      r.addEventListener("change", () => {
        card.querySelectorAll(".dup-item").forEach((el) => el.classList.remove("primary"));
        r.closest(".dup-item").classList.add("primary");
      });
    });
    box.appendChild(card);
  }
}
