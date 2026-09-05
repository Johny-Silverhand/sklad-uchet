const STEPS = [
  { id: "welcome", label: "Приветствие" },
  { id: "license", label: "Лицензия" },
  { id: "path", label: "Папка" },
  { id: "options", label: "Параметры" },
  { id: "install", label: "Установка" },
  { id: "done", label: "Готово" },
];
const LICENSE = `ЛИЦЕНЗИОННОЕ СОГЛАШЕНИЕ
Точка Склада · Victimok Labs

© 2026 Victimok Labs. Все права защищены.

Это локальное Windows-приложение (Go + SQLite + WebView2). Собственное окно без адресной строки. Данные только на вашем ПК.

1. Неисключительное право установки и использования одной рабочей копии.
2. Запрещаются копирование дистрибутива для перепродажи, декомпиляция и передача третьим лицам без согласия Victimok Labs.
3. Издатель не отвечает за утрату данных из-за вмешательства в файлы установки или базу SQLite.
4. Устанавливая программу, вы подтверждаете: «Разработано в Victimok Labs.»

Контакты: Victimok Labs, 2026.`;

const state = {
  i: 0, native: false, accepted: false,
  dir: "C:\\Users\\User\\AppData\\Local\\Programs\\Victimok Labs\\SkladUchet",
  desktop: true, startMenu: true, autostart: false,
};

const pane = document.getElementById("pane");
const stepsEl = document.getElementById("steps");
const back = document.getElementById("back");
const next = document.getElementById("next");

function esc(s) {
  const d = document.createElement("div");
  d.textContent = String(s);
  return d.innerHTML;
}

function renderSteps() {
  stepsEl.innerHTML = STEPS.map((s, idx) => {
    const cls = idx === state.i ? "is-on" : idx < state.i ? "is-done" : "";
    return `<div class="step ${cls}"><span class="n">${String(idx + 1).padStart(2, "0")}</span><span>${s.label}</span></div>`;
  }).join("");
}

function view() {
  const id = STEPS[state.i].id;
  next.disabled = false;
  if (id === "welcome") {
    pane.innerHTML = `
      <div class="hero"><span>Victimok Labs · desktop ops</span></div>
      <h1>Установка «Точка Склада»</h1>
      <p class="lead">Локальный учёт: Баланс и Временное хранение, свои темы. SQLite на ПК, окно WebView2 (Victimok Labs).</p>
      <div class="meta">
        <div><small>Издатель</small><b>Victimok Labs</b></div>
        <div><small>Версия</small><b>1.3.0 · 2026</b></div>
      </div>`;
    next.textContent = "Далее";
  }
  if (id === "license") {
    pane.innerHTML = `
      <h1>Лицензия</h1>
      <p class="lead">Без согласия установка не продолжится.</p>
      <div class="license">${esc(LICENSE)}</div>
      <label class="check" style="margin-top:12px">
        <input type="checkbox" id="acc" ${state.accepted ? "checked" : ""} />
        <span><b>Принимаю условия Victimok Labs</b><i>Разработано в Victimok Labs.</i></span>
      </label>`;
    document.getElementById("acc").onchange = (e) => {
      state.accepted = e.target.checked;
      next.disabled = !state.accepted;
    };
    next.disabled = !state.accepted;
    next.textContent = "Принять";
  }
  if (id === "path") {
    pane.innerHTML = `
      <h1>Папка установки</h1>
      <p class="lead">Программа ставится как обычное Windows-приложение. Путь можно сменить.</p>
      <div class="path">
        <input id="dir" value="${esc(state.dir)}" />
        <button type="button" class="ghost" id="browse">Обзор</button>
      </div>
      <div class="meta">
        <div><small>Требуется</small><b>~ 25 МБ на диске</b></div>
        <div><small>Права</small><b>текущий пользователь</b></div>
      </div>`;
    document.getElementById("dir").oninput = (e) => { state.dir = e.target.value; };
    document.getElementById("browse").onclick = async () => {
      if (!state.native) return;
      const r = await fetch("/api/browse", { method: "POST" });
      if (r.ok) {
        const j = await r.json();
        if (j.dir) { state.dir = j.dir; document.getElementById("dir").value = j.dir; }
      }
    };
    next.textContent = "Далее";
  }
  if (id === "options") {
    pane.innerHTML = `
      <h1>Параметры</h1>
      <p class="lead">Ярлыки. Ядро приложения ставится всегда.</p>
      <div class="checks">
        <label class="check"><input type="checkbox" checked disabled /><span><b>Ядро «Точка Склада»</b><i>Исполняемый модуль, локальная БД, подпись Victimok Labs</i></span></label>
        <label class="check"><input type="checkbox" id="sm" ${state.startMenu ? "checked" : ""} /><span><b>Меню «Пуск»</b><i>Victimok Labs → Точка Склада</i></span></label>
        <label class="check"><input type="checkbox" id="dt" ${state.desktop ? "checked" : ""} /><span><b>Ярлык на рабочем столе</b><i>Быстрый запуск</i></span></label>
        <label class="check"><input type="checkbox" id="au" ${state.autostart ? "checked" : ""} /><span><b>Автозапуск со Windows</b><i>По желанию</i></span></label>
      </div>`;
    document.getElementById("sm").onchange = (e) => state.startMenu = e.target.checked;
    document.getElementById("dt").onchange = (e) => state.desktop = e.target.checked;
    document.getElementById("au").onchange = (e) => state.autostart = e.target.checked;
    next.textContent = "Установить";
  }
  if (id === "install") {
    pane.innerHTML = `
      <h1>Идёт установка</h1>
      <p class="lead">Копируем модули Victimok Labs и регистрируем программу.</p>
      <div class="progress"><i id="bar"></i></div>
      <div class="log" id="log"></div>
      <p class="err" id="ierr" hidden></p>`;
    next.disabled = true;
    back.disabled = true;
    next.textContent = "Далее";
    void runInstall();
  }
  if (id === "done") {
    pane.innerHTML = `
      <div class="done-mark">OK</div>
      <h1>«Точка Склада» установлен</h1>
      <p class="lead">Разработано в Victimok Labs. Данные — только на этом ПК. Запуск через ярлык или меню Пуск.</p>
      <div class="checks">
        <label class="check"><input type="checkbox" id="launch" checked /><span><b>Запустить сейчас</b><i>Открыть складской учёт</i></span></label>
      </div>`;
    next.disabled = false;
    back.disabled = true;
    next.textContent = "Готово";
  }
  back.disabled = state.i === 0 || id === "install" || id === "done";
  renderSteps();
}

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
