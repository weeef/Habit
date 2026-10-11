"use strict";

// ---- tiny helpers ----

const $ = (id) => document.getElementById(id);
const store = {
  get(k) { try { return localStorage.getItem(k); } catch { return null; } },
  set(k, v) { try { localStorage.setItem(k, v); } catch {} },
  del(k) { try { localStorage.removeItem(k); } catch {} },
};

function toast(text, ms = 3500) {
  const el = document.createElement("div");
  el.className = "toast";
  el.textContent = text;
  $("toasts").append(el);
  setTimeout(() => el.remove(), ms);
}

// ---- state ----

const TW = 64, TH = 32;           // tile size in world pixels
const BUBBLE_MS = 7000, EMOTE_MS = 2500;
const EMOJI = { wave: "👋", dance: "💃", laugh: "😂", heart: "❤️", sit: "🪑" };

// Each community gets its own look. Missing colours fall back to the page
// theme, so the Plaza follows light/dark mode.
const THEMES = {
  plaza: { accent: "#6b4de6" },
  gaming: { void: "#14112a", floorA: "#2b2350", floorB: "#241d45", edge: "#7c4dff", accent: "#7c4dff" },
  gym: { floorA: "#5b6770", floorB: "#525d66", edge: "#3d464d", accent: "#e4572e" },
};
function theme(name) { return THEMES[name] || THEMES.plaza; }
function themeColor(key, cssVar) { return theme(state.room?.theme)[key] || css(cssVar); }

const state = {
  ws: null,
  room: null,          // { id, name, kind, rows, doors }
  you: null,           // your avatar id in this room
  avatars: new Map(),  // id -> avatar
  rooms: [],           // navigator summaries
  counts: {},
  account: null,       // { username, ageBand }
  pendingText: null,   // message typed before signing up
  recoveryCode: null,  // shown after the first message
  hover: null,         // hovered tile
  view: { ox: 0, oy: 0, z: 1 },
};

// ---- connection ----

function connect() {
  const proto = location.protocol === "https:" ? "wss" : "ws";
  const ws = new WebSocket(`${proto}://${location.host}/ws`);
  state.ws = ws;
  ws.onopen = () => {
    const token = store.get("habit.token");
    if (token) send({ t: "resume", token });
  };
  ws.onmessage = (e) => onMessage(JSON.parse(e.data));
  ws.onclose = () => {
    $("where").textContent = "Reconnecting…";
    setTimeout(connect, 1500);
  };
}

function send(msg) {
  if (state.ws && state.ws.readyState === WebSocket.OPEN) state.ws.send(JSON.stringify(msg));
}

function onMessage(m) {
  switch (m.t) {
    case "welcome":
      state.rooms = m.rooms;
      for (const r of m.rooms) state.counts[r.id] = r.count;
      renderNav();
      break;

    case "room":
      state.room = m.room;
      state.you = m.you;
      state.avatars.clear();
      for (const a of m.avatars) addAvatar(a);
      $("where").textContent = `You're in ${m.room.name}`;
      $("chatTitle").textContent = m.room.name;
      renderLog(m.history);
      renderNav();
      hideTooltip();
      break;

    case "enter": addAvatar(m.avatar); break;
    case "leave": state.avatars.delete(m.id); break;

    case "update": {
      const a = state.avatars.get(m.avatar.id);
      if (a) Object.assign(a, { name: m.avatar.name, guest: m.avatar.guest, newbie: m.avatar.newbie });
      break;
    }

    case "path": {
      const a = state.avatars.get(m.id);
      if (!a) break;
      a.from = { x: m.from[0], y: m.from[1] };
      a.path = m.path.map(([x, y]) => ({ x, y }));
      a.stepMs = m.stepMs || 1;
      a.start = performance.now();
      break;
    }

    case "chat": {
      appendLog(m.msg);
      const a = state.avatars.get(m.msg.from);
      if (a) a.bubbles.push({ text: m.msg.text, at: performance.now() });
      break;
    }

    case "emote": {
      const a = state.avatars.get(m.id);
      if (a) { a.emote = m.emote; a.emoteAt = performance.now(); }
      break;
    }

    case "counts":
      state.counts = m.counts;
      renderNav();
      break;

    case "peek": showPeek(m); break;

    case "need_account": openAuth("signup"); break;

    case "authed":
      state.account = { username: m.username, ageBand: m.ageBand };
      store.set("habit.token", m.token);
      if (m.recoveryCode) state.recoveryCode = m.recoveryCode;
      $("authDialog").close();
      $("named").disabled = false;
      $("accountBtn").textContent = m.username;
      if (state.pendingText) {
        send({ t: "chat", text: state.pendingText });
        state.pendingText = null;
        $("say").value = "";
      }
      break;

    case "auth_error":
      $("authError").textContent = m.message;
      $("authSubmit").disabled = false;
      break;

    case "first_message":
      // Ask them to save the recovery code only after they've said hi, so it
      // never slows down the first message.
      if (state.recoveryCode) {
        $("recoveryCode").textContent = state.recoveryCode;
        setTimeout(() => $("recoveryDialog").showModal(), 1200);
      }
      break;

    case "notice": toast(m.message); break;
  }
}

function addAvatar(a) {
  const pos = { x: a.pos[0], y: a.pos[1] };
  state.avatars.set(a.id, {
    ...a, from: pos, path: [], stepMs: 1, start: 0,
    bubbles: [], emote: null, emoteAt: 0, you: a.id === state.you || a.you,
  });
}

// ---- side panel ----

function renderNav() {
  const nav = $("nav");
  nav.replaceChildren();
  for (const r of state.rooms) {
    const li = document.createElement("li");
    const b = document.createElement("button");
    b.type = "button";
    const name = document.createElement("span");
    name.textContent = r.name;
    const count = document.createElement("span");
    count.className = "count";
    const n = state.counts[r.id] ?? 0;
    count.textContent = `${n} inside`;
    b.append(name, count);
    if (state.room && state.room.id === r.id) b.setAttribute("aria-current", "true");
    b.onclick = () => send({ t: "teleport", room: r.id });
    li.append(b);
    nav.append(li);
  }
}

function renderLog(history) {
  const log = $("log");
  log.replaceChildren();
  if (!history || history.length === 0) {
    const li = document.createElement("li");
    li.className = "empty";
    li.textContent = "It's quiet in here. Say hi?";
    log.append(li);
    return;
  }
  for (const m of history) appendLog(m);
}

function appendLog(m) {
  const log = $("log");
  log.querySelector(".empty")?.remove();
  const li = document.createElement("li");
  if (m.newbie) {
    const n = document.createElement("span");
    n.className = "new";
    n.textContent = "new";
    li.append(n);
  }
  const name = document.createElement("span");
  name.className = "name";
  name.textContent = m.name;
  li.append(name, document.createTextNode(m.text));
  const nearBottom = log.scrollHeight - log.scrollTop - log.clientHeight < 40;
  log.append(li);
  if (nearBottom) log.scrollTop = log.scrollHeight;
}

function say(text) {
  text = text.trim();
  if (!text) return;
  if (!state.account) {
    state.pendingText = text;
    openAuth("signup");
    return;
  }
  send({ t: "chat", text });
  $("say").value = "";
}

$("sayForm").addEventListener("submit", (e) => { e.preventDefault(); say($("say").value); });
$("say").addEventListener("focus", () => { if (!state.account) openAuth("signup"); });
for (const b of $("prompts").querySelectorAll("button")) b.onclick = () => say(b.textContent);
for (const b of document.querySelectorAll("[data-emote]")) b.onclick = () => send({ t: "emote", emote: b.dataset.emote });
$("named").addEventListener("change", (e) => send({ t: "named", on: e.target.checked }));

$("viewToggle").onclick = () => {
  const on = document.body.classList.toggle("list-view");
  $("viewToggle").setAttribute("aria-pressed", String(on));
  $("viewToggle").textContent = on ? "World view" : "List view";
  store.set("habit.listView", on ? "1" : "");
};
if (store.get("habit.listView")) $("viewToggle").click();

// ---- sign up / log in ----

let authMode = "signup";

function openAuth(mode) {
  setAuthMode(mode);
  $("authError").textContent = "";
  $("authSubmit").disabled = false;
  if (!$("authDialog").open) $("authDialog").showModal();
}

function setAuthMode(mode) {
  authMode = mode;
  $("authForm").classList.toggle("login", mode === "login");
  for (const t of document.querySelectorAll(".tabs [data-mode]")) t.setAttribute("aria-selected", String(t.dataset.mode === mode));
  $("authTitle").textContent = mode === "signup" ? "Pick a name to chat" : "Welcome back";
  $("authSubmit").textContent = mode === "signup" ? "Start chatting" : "Log in";
  $("authForm").password.autocomplete = mode === "signup" ? "new-password" : "current-password";
  $("authForm").birthYear.required = mode === "signup";
}

for (const t of document.querySelectorAll(".tabs [data-mode]")) t.onclick = () => setAuthMode(t.dataset.mode);
$("authCancel").onclick = () => { state.pendingText = null; $("authDialog").close(); };
$("accountBtn").onclick = () => { if (!state.account) openAuth("signup"); };

$("authForm").addEventListener("submit", (e) => {
  e.preventDefault();
  const f = e.target;
  $("authError").textContent = "";
  $("authSubmit").disabled = true;
  const msg = { t: authMode, username: f.username.value, password: f.password.value };
  if (authMode === "signup") msg.birthYear = Number(f.birthYear.value);
  send(msg);
});

$("savedCode").onclick = () => { state.recoveryCode = null; $("recoveryDialog").close(); };
$("copyCode").onclick = async () => {
  try { await navigator.clipboard.writeText(state.recoveryCode); toast("Copied"); } catch { toast("Couldn't copy. Please write it down."); }
};

// ---- the world view ----

const canvas = $("world");
const ctx = canvas.getContext("2d");

function css(name) { return getComputedStyle(document.documentElement).getPropertyValue(name).trim(); }

// Tile (x, y) -> screen centre of that tile.
function toScreen(x, y) {
  const { ox, oy, z } = state.view;
  return { sx: ox + (x - y) * (TW / 2) * z, sy: oy + (x + y) * (TH / 2) * z };
}

// Screen point -> tile under it.
function toTile(px, py) {
  const { ox, oy, z } = state.view;
  const a = (px - ox) / ((TW / 2) * z);
  const b = (py - oy) / ((TH / 2) * z);
  return { x: Math.round((a + b) / 2), y: Math.round((b - a) / 2) };
}

function isFloor(x, y) {
  const rows = state.room?.rows;
  return !!rows && y >= 0 && y < rows.length && x >= 0 && x < rows[y].length && rows[y][x] === ".";
}

function doorAt(x, y) {
  return state.room?.doors.find((d) => d.pos[0] === x && d.pos[1] === y);
}

function layout() {
  const dpr = window.devicePixelRatio || 1;
  const w = canvas.clientWidth, h = canvas.clientHeight;
  if (canvas.width !== Math.round(w * dpr) || canvas.height !== Math.round(h * dpr)) {
    canvas.width = Math.round(w * dpr);
    canvas.height = Math.round(h * dpr);
  }
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
  if (!state.room) return { w, h };

  const H = state.room.rows.length, W = state.room.rows[0].length;
  // Room footprint in unscaled pixels, plus headroom for avatars and doors.
  const roomW = (W + H) * (TW / 2);
  const roomH = (W + H) * (TH / 2) + 90;
  const z = Math.min(1.6, Math.max(0.45, Math.min((w - 24) / roomW, (h - 24) / roomH)));
  state.view.z = z;
  state.view.ox = w / 2 - ((W - H) * (TW / 2) * z) / 2;
  state.view.oy = (h - (W + H - 2) * (TH / 2) * z) / 2 + 30 * z;
  return { w, h };
}

function avatarPos(a, now) {
  if (!a.path.length) return a.from;
  const k = (now - a.start) / a.stepMs;
  const i = Math.floor(k);
  if (i >= a.path.length) {
    a.from = a.path[a.path.length - 1];
    a.path = [];
    return a.from;
  }
  const p0 = i === 0 ? a.from : a.path[i - 1];
  const p1 = a.path[i];
  const f = k - i;
  return { x: p0.x + (p1.x - p0.x) * f, y: p0.y + (p1.y - p0.y) * f };
}

function diamond(sx, sy, z) {
  ctx.beginPath();
  ctx.moveTo(sx, sy - (TH / 2) * z);
  ctx.lineTo(sx + (TW / 2) * z, sy);
  ctx.lineTo(sx, sy + (TH / 2) * z);
  ctx.lineTo(sx - (TW / 2) * z, sy);
  ctx.closePath();
}

function draw(now) {
  const { w, h } = layout();
  ctx.fillStyle = themeColor("void", "--void");
  ctx.fillRect(0, 0, w, h);
  if (!state.room) { requestAnimationFrame(draw); return; }
  const z = state.view.z;
  const rows = state.room.rows;

  // Floor.
  for (let y = 0; y < rows.length; y++) {
    for (let x = 0; x < rows[y].length; x++) {
      if (!isFloor(x, y)) continue;
      const { sx, sy } = toScreen(x, y);
      diamond(sx, sy, z);
      ctx.fillStyle = (x + y) % 2 ? themeColor("floorA", "--floor-a") : themeColor("floorB", "--floor-b");
      ctx.fill();
      ctx.strokeStyle = themeColor("edge", "--floor-edge");
      ctx.lineWidth = 1;
      ctx.stroke();
    }
  }
  if (state.hover && isFloor(state.hover.x, state.hover.y)) {
    const { sx, sy } = toScreen(state.hover.x, state.hover.y);
    diamond(sx, sy, z);
    ctx.fillStyle = "rgb(255 255 255 / 0.25)";
    ctx.fill();
  }

  // Doors and avatars, drawn back to front.
  const drawables = [];
  for (const d of state.room.doors) drawables.push({ depth: d.pos[0] + d.pos[1] - 0.5, draw: () => drawDoor(d, z) });
  for (const a of state.avatars.values()) {
    const p = avatarPos(a, now);
    drawables.push({ depth: p.x + p.y, draw: () => drawAvatar(a, p, now, z) });
  }
  drawables.sort((p, q) => p.depth - q.depth);
  for (const d of drawables) d.draw();

  // Bubbles go on top of everything.
  for (const a of state.avatars.values()) drawBubbles(a, avatarPos(a, now), now, z);

  requestAnimationFrame(draw);
}

function drawDoor(d, z) {
  const { sx, sy } = toScreen(d.pos[0], d.pos[1]);
  const w = 26 * z, h = 46 * z;
  const accent = theme(d.target).accent;
  ctx.fillStyle = accent;
  ctx.fillRect(sx - w / 2, sy - h, w, h);
  ctx.fillStyle = "#2a2438";
  ctx.fillRect(sx - w / 2 + 4 * z, sy - h + 5 * z, w - 8 * z, h - 5 * z);
  ctx.fillStyle = "#f3a712";
  ctx.beginPath();
  ctx.arc(sx + w / 2 - 8 * z, sy - h / 2, 2 * z, 0, Math.PI * 2);
  ctx.fill();

  const n = state.counts[d.target] ?? 0;
  label(`${d.label} · ${n}`, sx, sy - h - 10 * z, z, accent, "#fff");
}

function label(text, x, y, z, bg, fg) {
  ctx.font = `600 ${Math.round(12 * z)}px system-ui, sans-serif`;
  const tw = ctx.measureText(text).width;
  const pad = 5 * z, hgt = 18 * z;
  ctx.fillStyle = bg;
  roundRect(x - tw / 2 - pad, y - hgt / 2, tw + pad * 2, hgt, 6 * z);
  ctx.fill();
  ctx.fillStyle = fg;
  ctx.textAlign = "center";
  ctx.textBaseline = "middle";
  ctx.fillText(text, x, y + 0.5);
}

function roundRect(x, y, w, h, r) {
  ctx.beginPath();
  ctx.roundRect(x, y, w, h, r);
}

function drawAvatar(a, p, now, z) {
  const { sx, sy } = toScreen(p.x, p.y);
  const emoting = a.emote && now - a.emoteAt < EMOTE_MS;
  let dy = 0;
  if (emoting && a.emote === "sit") dy = 8 * z;
  if (emoting && a.emote === "dance") dy = -Math.abs(Math.sin(now / 120)) * 6 * z;
  if (a.path.length) dy -= Math.abs(Math.sin(now / 90)) * 2 * z;

  // Shadow.
  ctx.fillStyle = "rgb(0 0 0 / 0.22)";
  ctx.beginPath();
  ctx.ellipse(sx, sy, 14 * z, 6 * z, 0, 0, Math.PI * 2);
  ctx.fill();

  const base = sy + dy;
  // Legs.
  ctx.fillStyle = "#2a2438";
  ctx.fillRect(sx - 7 * z, base - 14 * z, 5 * z, 13 * z);
  ctx.fillRect(sx + 2 * z, base - 14 * z, 5 * z, 13 * z);
  // Body.
  ctx.fillStyle = a.color;
  ctx.fillRect(sx - 10 * z, base - 36 * z, 20 * z, 23 * z);
  // Head.
  ctx.fillStyle = "#f1c27d";
  ctx.fillRect(sx - 8 * z, base - 52 * z, 16 * z, 16 * z);
  ctx.fillStyle = "#2a2438";
  ctx.fillRect(sx - 4 * z, base - 46 * z, 2 * z, 3 * z);
  ctx.fillRect(sx + 2 * z, base - 46 * z, 2 * z, 3 * z);
  // Waving arm.
  if (emoting && a.emote === "wave") {
    ctx.fillStyle = a.color;
    const swing = Math.sin(now / 100) * 4 * z;
    ctx.fillRect(sx + 10 * z, base - 50 * z + swing, 5 * z, 16 * z);
  }

  // Name tags show for you and whoever is under the pointer, like Habbo,
  // so a busy room doesn't turn into a wall of labels.
  const hovered = state.hover && Math.round(p.x) === state.hover.x && Math.round(p.y) === state.hover.y;
  if (a.you || hovered) {
    const tag = a.you ? `${a.name} (you)` : a.name;
    label(tag, sx, base - 64 * z, z * 0.85, a.you ? "#6b4de6" : "rgb(42 36 56 / 0.85)", "#fff");
  }
  if (a.newbie) label("new", sx, base - (a.you || hovered ? 80 : 62) * z, z * 0.75, "#3fb27f", "#fff");
  if (emoting) {
    ctx.font = `${Math.round(20 * z)}px system-ui, sans-serif`;
    ctx.textAlign = "center";
    ctx.fillText(EMOJI[a.emote], sx + 20 * z, base - 54 * z);
  }
}

function drawBubbles(a, p, now, z) {
  a.bubbles = a.bubbles.filter((b) => now - b.at < BUBBLE_MS);
  if (!a.bubbles.length) return;
  const { sx, sy } = toScreen(p.x, p.y);
  let y = sy - 100 * z;
  ctx.font = `${Math.round(13 * z)}px system-ui, sans-serif`;
  for (let i = a.bubbles.length - 1; i >= 0; i--) {
    const b = a.bubbles[i];
    const age = now - b.at;
    const alpha = age > BUBBLE_MS - 1000 ? (BUBBLE_MS - age) / 1000 : 1;
    const text = b.text.length > 60 ? b.text.slice(0, 57) + "…" : b.text;
    const tw = ctx.measureText(text).width;
    const pad = 8 * z, hgt = 24 * z;
    ctx.globalAlpha = alpha;
    ctx.fillStyle = "#ffffff";
    ctx.strokeStyle = "#2a2438";
    ctx.lineWidth = 1.5;
    roundRect(sx - tw / 2 - pad, y - hgt, tw + pad * 2, hgt, 8 * z);
    ctx.fill();
    ctx.stroke();
    ctx.fillStyle = "#2a2438";
    ctx.textAlign = "center";
    ctx.textBaseline = "middle";
    ctx.fillText(text, sx, y - hgt / 2);
    ctx.globalAlpha = 1;
    y -= hgt + 4 * z;
  }
}

// ---- pointer input and door peeking ----

let lastPeek = { key: "", at: 0 };

canvas.addEventListener("pointermove", (e) => {
  const r = canvas.getBoundingClientRect();
  const t = toTile(e.clientX - r.left, e.clientY - r.top);
  state.hover = t;
  const d = doorAt(t.x, t.y);
  if (!d) { hideTooltip(); return; }
  const key = `${state.room.id}:${d.target}`;
  const now = performance.now();
  if (key !== lastPeek.key || now - lastPeek.at > 2000) {
    lastPeek = { key, at: now };
    send({ t: "peek", room: d.target });
  }
  const tip = $("tooltip");
  tip.style.left = `${Math.min(e.clientX - r.left + 14, r.width - 270)}px`;
  tip.style.top = `${e.clientY - r.top + 14}px`;
});
canvas.addEventListener("pointerleave", () => { state.hover = null; hideTooltip(); });

canvas.addEventListener("click", (e) => {
  const r = canvas.getBoundingClientRect();
  const t = toTile(e.clientX - r.left, e.clientY - r.top);
  if (isFloor(t.x, t.y)) {
    send({ t: "walk", x: t.x, y: t.y });
    $("hint").hidden = true;
  }
});

function showPeek(m) {
  if (!state.hover) return;
  const d = doorAt(state.hover.x, state.hover.y);
  if (!d || d.target !== m.room) return;
  const tip = $("tooltip");
  tip.replaceChildren();
  const title = document.createElement("strong");
  title.textContent = `${m.name} · ${m.count} inside`;
  tip.append(title);
  if (!m.recent.length) {
    const p = document.createElement("p");
    p.textContent = "Quiet right now.";
    tip.append(p);
  }
  for (const msg of m.recent) {
    const p = document.createElement("p");
    p.textContent = `${msg.name}: ${msg.text}`;
    tip.append(p);
  }
  tip.hidden = false;
}

function hideTooltip() { $("tooltip").hidden = true; }

connect();
requestAnimationFrame(draw);
