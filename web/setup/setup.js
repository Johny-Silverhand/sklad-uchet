const LINES = [
  "Проверка пакета…",
  "Копирование TochkaSklada.exe",
  "Запись LICENSE.txt и app.ico",
  "Маркер installed.json",
  "Регистрация в «Программы и компоненты»",
  "Ярлык Victimok Labs в меню Пуск",
  "Ярлык на рабочем столе",
  "Контроль… OK",
  "Разработано в Victimok Labs.",
];

async function runInstall() {
  const log = document.getElementById("log");
  const bar = document.getElementById("bar");
  const err = document.getElementById("ierr");
  const write = (t, ok) => {
    const d = document.createElement("div");
    d.textContent = "› " + t;
    if (ok) d.className = "ok";
    log.appendChild(d);
    log.scrollTop = log.scrollHeight;
  };
  try {
    if (state.native) {
      write("Связь с установщиком…");
      const r = await fetch("/api/install", {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: JSON.stringify({
          dir: state.dir, desktop: state.desktop,
          startMenu: state.startMenu, autostart: state.autostart,
        }),
      });
      const j = await r.json().catch(() => ({}));
      if (!r.ok) throw new Error(j.error || "Ошибка установки");
    }
    for (let i = 0; i < LINES.length; i++) {
      await wait(state.native ? 80 : 120);
      bar.style.width = ((i + 1) / LINES.length * 100) + "%";
      write(LINES[i], i === LINES.length - 1);
    }
    await wait(200);
    state.i += 1;
    view();
  } catch (e) {
    err.hidden = false;
    err.textContent = e.message || String(e);
    back.disabled = false;
    next.disabled = false;
    next.textContent = "Повторить";
  }
}

function wait(ms) { return new Promise((r) => setTimeout(r, ms)); }

back.onclick = () => { if (state.i > 0) { state.i -= 1; view(); } };
next.onclick = async () => {
  const id = STEPS[state.i].id;
  if (id === "license" && !state.accepted) return;
  if (id === "install") { view(); return; }
  if (id === "done") {
    const launch = document.getElementById("launch")?.checked !== false;
    if (state.native) {
      await fetch("/api/finish", { method: "POST", headers: { "content-type": "application/json" }, body: JSON.stringify({ launch }) });
      try { window.close(); } catch {}
    }
    return;
  }
  if (state.i < STEPS.length - 1) { state.i += 1; view(); }
};

(async () => {
  view();
  try {
    const r = await fetch("/api/meta", { cache: "no-store" });
    if (r.ok) {
      const j = await r.json();
      if (j.native) {
        state.native = true;
        if (j.defaultDir) state.dir = j.defaultDir;
        view();
      }
    }
  } catch {}
})();
