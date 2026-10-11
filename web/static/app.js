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
  // An open dialog (like the board) sits above the page, so show the toast
  // inside it or it would be hidden.
  const dialog = [...document.querySelectorAll("dialog[open]")].pop();
  let host = $("toasts");
  if (dialog) {
    host = dialog.querySelector(":scope > .toasts") || dialog.appendChild(Object.assign(document.createElement("div"), { className: "toasts" }));
  }
  host.append(el);
  setTimeout(() => el.remove(), ms);
}

// ---- state ----

const TW = 64, TH = 32;           // tile size in world pixels
const BUBBLE_MS = 7000, EMOTE_MS = 2500;
const EMOJI = { wave: "👋", dance: "💃", laugh: "😂", heart: "❤️", sit: "🪑" };

// Art direction: cute voxels. Everything is built from little cubes:
// characters are small 3D voxel models that turn to face where they're
// going, floors are blocks, props (fountain, benches, trees, lamps, bell,
// noticeboard) are voxel models you can use. Each community gets its own
// palette and floor decorations; missing colours fall back to the page
// theme, so the Plaza follows light/dark mode. Colours must be hex (see
// shade()).
// The Plaza is outdoors in a bright city square with a day/night cycle
// (see dayCycle); its look doesn't follow the page's dark mode. Materials
// map floor characters to colours; "ring" is the inlay round the fountain.
const THEMES = {
  plaza: {
    accent: "#ff6fae", deco: null, outdoor: true, void: "#d6eeff",
    materials: {
      ".": ["#f6f1ea", "#f1ebe3"], ",": ["#e6e4ea", "#e0dee5"], ":": ["#fde9d0", "#f9e2c6"],
      ring: ["#f9dfe9", "#f5d7e3"],
    },
  },
  gaming: { void: "#1c1733", floorA: "#3d3573", floorB: "#3a326e", accent: "#a78bfa", deco: "star" },
  gym: { floorA: "#d6f2e4", floorB: "#cdeedd", accent: "#ff9466", deco: "leaf" },
};
const DAY_MS = 24 * 60 * 1000; // a plaza day lasts 24 minutes: one real minute is one plaza hour
// Character palettes. The server stores a style as indexes into these, so
// only ever append to them.
const SKIN = ["#ffe0cf", "#f7cdb0", "#e8ae88", "#c98d63", "#9c6545", "#6e432c"];
const HAIR = ["#4a3346", "#7a4b3a", "#c47a4f", "#f2c46d", "#f39ac3", "#9fb7ff", "#b49cf0", "#7dd3b0"];
const OUTFITS = ["#8ab4ff", "#ff9aa2", "#ffcf6e", "#7fdcb4", "#c3a6ff", "#ff9ed2", "#7fd6e8", "#ffb07a"];
const ACCESSORIES = ["ears", "bow", "curl", "none"];
const ACCESSORY_NAMES = ["Cat ears", "Bow", "Curl", "None"];
const SLAB = 10;            // floor block thickness, screen px at zoom 1
const ZPX = 36;             // screen px per tile of height, at zoom 1
const VOX = 0.085;          // character voxel size, in tiles
const TURN_SPEED = 10;      // radians per second
const FIGURE_SCALE = 1.2;   // particles rise relative to character size
const LIGHT = { x: 0.35, y: 0.94 }; // light from the screen's left, so left faces are lighter
const FONT = `"Nunito", "Segoe UI Variable", system-ui, sans-serif`;

// dayCycle says where the plaza is in its day. It's driven by the clock, so
// everyone sees the same time. night and dusk run 0..1.
// The plaza clock runs fast (a shared 24-minute day) unless you've chosen
// to follow your own clock (real time).
function clockMode() { return store.get("habit.clock") === "real" ? "real" : "fast"; }

function dayCycle(now = Date.now()) {
  const local = new Date(now);
  const hour = clockMode() === "real"
    ? local.getHours() + local.getMinutes() / 60 + local.getSeconds() / 3600
    : ((now % DAY_MS) / DAY_MS) * 24;
  const ramp = (a, b) => Math.min(1, Math.max(0, (hour - a) / (b - a)));
  // Night falls 18:30–20:30 and lifts 5:00–7:00.
  const night = hour < 12 ? 1 - ramp(5, 7) : ramp(18.5, 20.5);
  // Warm light around sunrise and sunset.
  const dusk = Math.max(0, 1 - Math.abs(hour - 6) / 1.5, 1 - Math.abs(hour - 19) / 1.7);
  const h = Math.floor(hour), m = Math.floor((hour - h) * 60);
  const time = `${((h + 11) % 12) + 1}:${String(m).padStart(2, "0")} ${h < 12 ? "AM" : "PM"}`;
  const icon = night > 0.6 ? "🌙" : dusk > 0.3 ? (hour < 12 ? "🌅" : "🌇") : "☀️";
  const part = night > 0.6 ? "night" : dusk > 0.3 ? (hour < 12 ? "sunrise" : "sunset") : hour < 12 ? "morning" : hour < 17 ? "afternoon" : "evening";
  return { hour, night, dusk, time, icon, part };
}

// lightsOn is how lit the lamps and windows are: they come on at dusk.
function lightsOn(cycle) {
  return Math.min(1, cycle.night * 1.4 + (cycle.hour > 17 || cycle.hour < 7 ? cycle.dusk * 0.6 : 0));
}

function outdoors() { return !!theme(state.room?.theme).outdoor; }

// shade(hex, f) darkens (f < 1) or lightens (f > 1) a hex colour. Cached,
// because the floor calls it for every tile every frame.
const shadeCache = new Map();
function shade(hex, f) {
  const key = hex + f.toFixed(3);
  let out = shadeCache.get(key);
  if (!out) {
    let h = hex.replace("#", "");
    if (h.length === 3) h = [...h].map((c) => c + c).join("");
    const n = parseInt(h, 16);
    const k = (v) => Math.round(f < 1 ? v * f : v + (255 - v) * (f - 1));
    out = `rgb(${k((n >> 16) & 255)} ${k((n >> 8) & 255)} ${k(n & 255)})`;
    shadeCache.set(key, out);
  }
  return out;
}
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
  afterAuth: null,     // action to finish once signed up, e.g. a post
  boardSort: store.get("habit.boardSort") || "active", // the board's filter
  board: null,         // { room, name, threads, page, more }
  thread: null,        // open thread: { thread, replies }
  lastReply: null,     // reply text, restored if the post fails
  notes: [],           // replies to you, newest first
  particles: [],       // floating sparkles, hearts, coins, petals
  lastFrame: 0,
  hits: [],            // props drawn this frame, for clicks and hover
  birds: [],           // the plaza's pigeons
  shakes: {},          // tree id -> when it was shaken
  bellAt: 0,
  draftStyle: null,    // character being designed in the signup dialog
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
      state.particles = [];
      state.birds = [];
      for (const a of m.avatars) addAvatar(a);
      $("chatTitle").textContent = `Chat in ${plainName(m.room.name)}`;
      renderLog(m.history);
      renderNav();
      hideTooltip();
      if (!state.thread) loadBoard(0); // also fills the room's noticeboard
      break;

    case "enter": addAvatar(m.avatar); break;
    case "leave": state.avatars.delete(m.id); break;

    case "update": {
      const a = state.avatars.get(m.avatar.id);
      if (a) Object.assign(a, { name: m.avatar.name, guest: m.avatar.guest, newbie: m.avatar.newbie, style: m.avatar.style, seat: m.avatar.seat });
      break;
    }

    case "fx": handleFx(m); break;

    case "path": {
      const a = state.avatars.get(m.id);
      if (!a) break;
      a.from = { x: m.from[0], y: m.from[1] };
      a.path = m.path.map(([x, y]) => ({ x, y }));
      // When each step ends: diagonal steps take longer, like on the server.
      const step = m.stepMs || 1;
      a.times = [];
      let t = 0, prev = a.from;
      for (const q of a.path) {
        t += prev.x !== q.x && prev.y !== q.y ? step * Math.SQRT2 : step;
        a.times.push(t);
        prev = q;
      }
      a.stepMs = step;
      a.start = performance.now() - (m.walkedMs || 0);
      break;
    }

    case "chat": {
      appendLog(m.msg);
      const a = state.avatars.get(m.msg.from);
      if (a) {
        a.bubbles.push({ text: m.msg.text, at: performance.now() });
        // People standing nearby turn to look at whoever's talking.
        for (const o of state.avatars.values()) {
          if (o === a || o.path.length) continue;
          const dx = a.from.x - o.from.x, dy = a.from.y - o.from.y;
          if (Math.hypot(dx, dy) <= 4) o.turnTarget = Math.atan2(dy, dx);
        }
      }
      break;
    }

    case "emote": {
      const a = state.avatars.get(m.id);
      if (a) { a.emote = m.emote; a.emoteAt = performance.now(); }
      if (m.emote === "heart") spawnHearts(m.id);
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
      $("notesBtn").hidden = false;
      if (state.pendingText) {
        send({ t: "chat", text: state.pendingText });
        state.pendingText = null;
        $("say").value = "";
      }
      if (state.afterAuth) {
        const f = state.afterAuth;
        state.afterAuth = null;
        f();
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

    case "board":
      if (m.page === 0 || state.board?.room !== m.room || state.board?.sort !== m.sort) {
        state.board = { room: m.room, name: m.name, sort: m.sort, threads: m.threads, page: m.page, more: m.more };
      } else {
        state.board.threads.push(...m.threads);
        Object.assign(state.board, { page: m.page, more: m.more });
      }
      renderBoard();
      break;

    case "thread":
      state.thread = { thread: m.thread, replies: m.replies };
      renderThread(true);
      break;

    case "posted_thread":
      $("threadDialog").close();
      $("threadForm").reset();
      upsertThread(m.thread);
      state.thread = { thread: m.thread, replies: [] };
      if (!$("boardDialog").open) $("boardDialog").showModal();
      renderThread(true);
      break;

    case "votes": applyVotes(m); break;

    case "board_update":
      upsertThread(m.thread);
      if (state.thread?.thread.id === m.thread.id) state.thread.thread = m.thread;
      renderBoard();
      break;

    case "reply":
      if (state.thread?.thread.id === m.thread.id) {
        state.thread.thread = m.thread;
        if (!state.thread.replies.some((r) => r.id === m.reply.id)) state.thread.replies.push(m.reply);
        renderThread(true);
      }
      upsertThread(m.thread);
      state.lastReply = null;
      break;

    case "post_error":
      if ($("threadDialog").open) {
        $("threadError").textContent = m.message;
        $("threadSubmit").disabled = false;
      } else {
        toast(m.message, 5000);
        if (state.lastReply && !$("replyText").value) $("replyText").value = state.lastReply;
        state.lastReply = null;
      }
      break;

    case "notes":
      state.notes = m.notes;
      renderNotesBadge();
      break;

    case "notify":
      state.notes.unshift(m.note);
      renderNotesBadge();
      toast(`New reply in "${m.note.title}"`);
      break;
  }
}

function addAvatar(a) {
  const pos = { x: a.pos[0], y: a.pos[1] };
  state.avatars.set(a.id, {
    ...a, from: pos, path: [], times: [], stepMs: 1, start: 0, gait: 0,
    bubbles: [], emote: null, emoteAt: 0, you: a.id === state.you || a.you,
    facing: Math.PI / 4, turnTarget: Math.PI / 4, // facing the camera
    seed: Math.random(), // staggers blinking and breathing
  });
}

// ---- side panel ----

// Every room is a "line" with its own colour and bullet, like the subway.
const ROOM_LINES = {
  plaza: { color: "#e5484d", bullet: "⛲" },
  gaming: { color: "#8b6bf2", bullet: "🎮" },
  gym: { color: "#f08a3c", bullet: "🏋️" },
};
function roomLine(id) { return ROOM_LINES[id] || { color: "#4f7fd6", bullet: "🚪" }; }
function plainName(name) { return name.replace(/^\p{Extended_Pictographic}️?\s*/u, ""); }

// renderNav updates the station sign: where you are and who's here.
function renderNav() {
  if (!state.room) return;
  const line = roomLine(state.room.id);
  $("stationBullet").textContent = line.bullet;
  $("stationBullet").style.background = line.color;
  $("where").textContent = plainName(state.room.name);
  const n = state.counts[state.room.id] ?? 0;
  $("stationCount").textContent = n === 1 ? "just you" : `${n} here`;
}
$("station").onclick = () => openSubway();

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

// ---- the board: saved posts for each room ----

function el(tag, className, text) {
  const e = document.createElement(tag);
  if (className) e.className = className;
  if (text != null) e.textContent = text;
  return e;
}

function timeAgo(ts) {
  const s = Math.max(0, (Date.now() - new Date(ts)) / 1000);
  if (s < 60) return "just now";
  if (s < 3600) return `${Math.floor(s / 60)}m ago`;
  if (s < 86400) return `${Math.floor(s / 3600)}h ago`;
  if (s < 86400 * 30) return `${Math.floor(s / 86400)}d ago`;
  return new Date(ts).toLocaleDateString();
}

function roomName(id) { return state.rooms.find((r) => r.id === id)?.name ?? id; }

// requireAccount runs action now, or after signing up.
function requireAccount(action) {
  if (state.account) { action(); return; }
  state.afterAuth = action;
  openAuth("signup");
}

// openBoard opens this room's board in the middle of the screen: the list,
// or straight to one thread.
function openBoard(threadId) {
  if (!$("boardDialog").open) $("boardDialog").showModal();
  if (threadId) {
    send({ t: "thread", thread: threadId });
  } else {
    state.thread = null;
    renderBoard();
    loadBoard(0);
  }
}

function closeBoard() {
  if ($("boardDialog").open) $("boardDialog").close();
}

function loadBoard(page) {
  if (state.room) send({ t: "board", room: state.room.id, page, sort: state.boardSort });
}

// upsertThread applies a new or changed thread to the open list. In
// "latest" order activity moves it to the top; in other orders it's
// updated where it is (new posts appear at the top of "new").
function upsertThread(t) {
  const b = state.board;
  if (!b || b.room !== t.room) return;
  const i = b.threads.findIndex((x) => x.id === t.id);
  const merged = { ...t, mine: i >= 0 ? b.threads[i].mine : t.mine };
  if (b.sort === "active" || (b.sort === "new" && i < 0)) {
    b.threads = b.threads.filter((x) => x.id !== t.id);
    b.threads.unshift(merged);
  } else if (i >= 0) {
    b.threads[i] = merged;
  }
}

// voteButtons renders 👍 n 👎 n for a thread; clicking your current vote
// takes it back.
function voteButtons(t) {
  const box = el("div", "votes");
  for (const [value, icon, count, label] of [[1, "👍", t.ups || 0, "Upvote"], [-1, "👎", t.downs || 0, "Downvote"]]) {
    const b = el("button", "vote");
    b.type = "button";
    b.setAttribute("aria-pressed", String(t.mine === value));
    b.setAttribute("aria-label", `${label} (${count})`);
    b.append(el("span", null, icon), el("span", "n", String(count)));
    b.onclick = (e) => {
      e.stopPropagation();
      requireAccount(() => send({ t: "vote", thread: t.id, value: t.mine === value ? 0 : value }));
    };
    box.append(b);
  }
  return box;
}

function renderBoard() {
  $("boardList").hidden = !!state.thread;
  $("threadView").hidden = !state.thread;
  for (const b of document.querySelectorAll(".sorts [data-sort]")) b.setAttribute("aria-selected", String(b.dataset.sort === state.boardSort));
  const list = $("threads");
  list.replaceChildren();
  const b = state.board;
  $("boardTitle").textContent = `📌 ${b?.name ?? state.room?.name ?? ""} board`;
  if (!b) return;
  if (!b.threads.length) list.append(el("li", "empty", "No posts yet. Start the first one?"));
  for (const t of b.threads) {
    const btn = el("button", "thread-open");
    btn.type = "button";
    const replies = t.replies === 1 ? "1 reply" : `${t.replies} replies`;
    btn.append(
      el("span", "title", t.title),
      el("span", "meta", `${t.name} · ${replies} · ${timeAgo(state.boardSort === "active" ? t.bumped : t.ts)}${t.locked ? " · locked" : ""}`),
      el("span", "excerpt", t.text),
    );
    btn.onclick = () => send({ t: "thread", thread: t.id });
    const li = el("li");
    li.append(voteButtons(t), btn);
    list.append(li);
  }
  $("moreThreads").hidden = !b.more;
}

function postHeader(name, op, ts) {
  const meta = el("div", "meta");
  meta.append(el("span", "name", name));
  if (op) meta.append(el("span", "op-tag", "OP"));
  meta.append(document.createTextNode(` · ${timeAgo(ts)}`));
  return meta;
}

function renderThread(scrollToEnd) {
  renderBoard();
  if (!state.thread) return;
  const { thread: t, replies } = state.thread;

  const post = $("threadPost");
  post.replaceChildren(el("h3", null, t.title), postHeader(t.name, true, t.ts), el("p", null, t.text), voteButtons(t));

  const list = $("replies");
  list.replaceChildren();
  for (const r of replies) {
    const li = el("li", "post");
    li.append(postHeader(r.name, r.op, r.ts), el("p", null, r.text));
    list.append(li);
  }
  if (!replies.length) list.append(el("li", "meta", "No replies yet."));

  $("replyForm").hidden = t.locked;
  $("threadLocked").hidden = !t.locked;
  const elsewhere = state.room && t.room !== state.room.id;
  $("goThere").hidden = !elsewhere;
  $("goThere").textContent = `Go to ${roomName(t.room)}`;
  $("goThere").onclick = () => { closeBoard(); send({ t: "teleport", room: t.room }); };

  if (scrollToEnd) {
    const sc = $("threadScroll");
    sc.scrollTop = replies.length ? sc.scrollHeight : 0;
  }
}

// applyVotes updates a thread's totals wherever it's shown.
function applyVotes(m) {
  const update = (t) => {
    if (!t || t.id !== m.thread) return;
    t.ups = m.ups;
    t.downs = m.downs;
    if (m.mine !== undefined) t.mine = m.mine;
  };
  for (const t of state.board?.threads || []) update(t);
  update(state.thread?.thread);
  if ($("boardDialog").open) state.thread ? renderThread(false) : renderBoard();
}

function openThreadDialog() {
  $("threadDialogTitle").textContent = `New post in ${state.room?.name ?? "this room"}`;
  $("threadError").textContent = "";
  $("threadSubmit").disabled = false;
  $("threadDialog").showModal();
}

function sendReply(text) {
  if (!state.thread) return;
  state.lastReply = text;
  send({ t: "post_reply", thread: state.thread.thread.id, text });
  $("replyText").value = "";
}

for (const b of document.querySelectorAll(".sorts [data-sort]")) {
  b.onclick = () => {
    state.boardSort = b.dataset.sort;
    store.set("habit.boardSort", state.boardSort);
    state.thread = null;
    renderBoard();
    loadBoard(0);
  };
}
$("boardClose").onclick = closeBoard;
$("boardDialog").addEventListener("close", () => { state.thread = null; });
$("moreThreads").onclick = () => loadBoard((state.board?.page ?? 0) + 1);
$("backToBoard").onclick = () => { state.thread = null; renderBoard(); loadBoard(0); };
$("newThreadBtn").onclick = () => requireAccount(openThreadDialog);

$("threadCancel").onclick = () => $("threadDialog").close();
$("threadForm").addEventListener("submit", (e) => {
  e.preventDefault();
  const f = e.target;
  $("threadError").textContent = "";
  $("threadSubmit").disabled = true;
  send({ t: "post_thread", title: f.elements.title.value, text: f.elements.text.value });
});
$("replyForm").addEventListener("submit", (e) => {
  e.preventDefault();
  const text = $("replyText").value.trim();
  if (text) requireAccount(() => sendReply(text));
});
$("replyText").addEventListener("keydown", (e) => {
  if (e.key === "Enter" && (e.ctrlKey || e.metaKey)) $("replyForm").requestSubmit();
});

// ---- notifications: replies in threads you posted in ----

function renderNotesBadge() {
  const unread = state.notes.filter((n) => !n.read).length;
  $("notesCount").hidden = unread === 0;
  $("notesCount").textContent = unread > 99 ? "99+" : String(unread);
  $("notesBtn").setAttribute("aria-label", unread ? `Replies to you (${unread} new)` : "Replies to you");
}

function renderNotesList() {
  const list = $("notesList");
  list.replaceChildren();
  if (!state.notes.length) {
    list.append(el("li", "empty", "Nothing yet. When someone replies to a post you made or joined, it shows up here."));
    return;
  }
  for (const n of state.notes) {
    const btn = el("button");
    btn.type = "button";
    btn.append(
      el("span", "title", n.title),
      el("span", "excerpt", n.snippet),
      el("span", "meta", `${roomName(n.room)} · ${timeAgo(n.ts)}`),
    );
    btn.onclick = () => {
      $("notesDialog").close();
      openBoard(n.thread);
    };
    const li = el("li", n.read ? "" : "unread");
    li.append(btn);
    list.append(li);
  }
}

$("notesBtn").onclick = () => {
  renderNotesList(); // drawn before marking read, so new ones stay highlighted
  $("notesDialog").showModal();
  if (state.notes.some((n) => !n.read)) {
    send({ t: "notes_read" });
    for (const n of state.notes) n.read = true;
    renderNotesBadge();
  }
};
$("notesClose").onclick = () => $("notesDialog").close();

// ---- sign up / log in ----

let authMode = "signup";

function openAuth(mode) {
  if (!$("authDialog").open) state.draftStyle = null; // start from your current look
  setAuthMode(mode);
  $("authError").textContent = "";
  $("authSubmit").disabled = false;
  if (!$("authDialog").open) $("authDialog").showModal();
  if (mode === "signup") startCreator();
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

for (const t of document.querySelectorAll(".tabs [data-mode]")) {
  t.onclick = () => {
    setAuthMode(t.dataset.mode);
    if (t.dataset.mode === "signup") startCreator();
  };
}
$("authCancel").onclick = () => { state.pendingText = null; state.afterAuth = null; $("authDialog").close(); };
$("accountBtn").onclick = () => { if (!state.account) openAuth("signup"); };

$("authForm").addEventListener("submit", (e) => {
  e.preventDefault();
  const f = e.target;
  $("authError").textContent = "";
  $("authSubmit").disabled = true;
  const msg = { t: authMode, username: f.username.value, password: f.password.value };
  if (authMode === "signup") {
    msg.birthYear = Number(f.birthYear.value);
    msg.style = state.draftStyle || myStyle();
  }
  send(msg);
});

$("savedCode").onclick = () => { state.recoveryCode = null; $("recoveryDialog").close(); };
$("copyCode").onclick = async () => {
  try { await navigator.clipboard.writeText(state.recoveryCode); toast("Copied"); } catch { toast("Couldn't copy. Please write it down."); }
};

// ---- the world view ----

const canvas = $("world");
// Reassigned briefly while drawing sprites and the character preview.
let ctx = canvas.getContext("2d");

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
  return !!rows && y >= 0 && y < rows.length && x >= 0 && x < rows[y].length && ".,:".includes(rows[y][x]);
}

function doorAt(x, y) {
  return state.room?.doors.find((d) => d.pos[0] === x && d.pos[1] === y);
}

// layout sizes the canvas and places the camera. Small rooms fit the
// screen; big ones (like the plaza) are shown closer up, and the camera
// glides after you.
function layout(now = performance.now()) {
  const dpr = window.devicePixelRatio || 1;
  const w = canvas.clientWidth, h = canvas.clientHeight;
  if (canvas.width !== Math.round(w * dpr) || canvas.height !== Math.round(h * dpr)) {
    canvas.width = Math.round(w * dpr);
    canvas.height = Math.round(h * dpr);
  }
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
  if (!state.room) return { w, h };

  // The world fills the screen, but the chat panel covers the right (or
  // the bottom, on phones) and the station sign the top: frame the room in
  // what's left uncovered.
  const side = $("side");
  let insetR = 0, insetB = 0;
  if (side.offsetParent && !document.body.classList.contains("chat-hidden")) {
    if (window.innerWidth > 760) insetR = side.offsetWidth + 24;
    else insetB = side.offsetHeight + 16;
  }
  if (state.insetR !== insetR) {
    state.insetR = insetR;
    document.documentElement.style.setProperty("--inset-r", `${insetR}px`);
  }
  const top = 64, vw = w - insetR, vh = h - insetB - top;

  const H = state.room.rows.length, W = state.room.rows[0].length;
  // Room footprint in unscaled pixels, plus headroom for avatars and doors.
  const roomW = (W + H) * (TW / 2);
  const roomH = (W + H) * (TH / 2) + 90;
  const fit = Math.min((vw - 24) / roomW, (vh - 24) / roomH);
  const close = Math.min(0.9, Math.max(0.55, Math.min(vw, vh * 1.4) / 1050)); // comfortable size for big rooms
  const v = state.view;
  if (fit >= close) {
    v.z = Math.min(1.6, fit);
    v.ox = vw / 2 - ((W - H) * (TW / 2) * v.z) / 2;
    v.oy = top + (vh - (W + H - 2) * (TH / 2) * v.z) / 2 + 30 * v.z;
    v.follow = null;
    return { w, h };
  }
  // Follow your avatar, keeping a little of the room's edge in view.
  v.z = close;
  const me = state.avatars.get(state.you);
  const p = me ? avatarPos(me, now) : { x: W / 2, y: H / 2 };
  const fx = Math.min(W - 5, Math.max(4, p.x)), fy = Math.min(H - 5, Math.max(4, p.y));
  const tx = vw / 2 - (fx - fy) * (TW / 2) * v.z, ty = top + vh * 0.6 - (fx + fy) * (TH / 2) * v.z;
  const snap = v.follow !== state.room.id || !Number.isFinite(v.ox);
  const k = snap ? 1 : 1 - Math.exp(-Math.min(100, now - (v.t || now)) / 260);
  v.ox += (tx - v.ox) * k;
  v.oy += (ty - v.oy) * k;
  v.t = now;
  v.follow = state.room.id;
  return { w, h };
}

// avatarPos is where an avatar is drawn at a moment. Paths go tile to
// tile, but the drawn avatar follows that point on a critically damped
// spring, so corners become curves, they ease into a walk and out of it,
// and a change of course or a bit of network lag never makes them jump.
const FOLLOW = 20; // spring stiffness (per second): higher hugs the path tighter

function avatarPos(a, now) {
  if (a.posAt === now) return a.pos;
  let target = a.from;
  if (a.path.length) {
    // rAF's timestamp is the frame's start, which can be slightly before
    // the moment the path message arrived, so t can dip below zero.
    const t = Math.max(0, now - a.start);
    if (t >= a.times[a.times.length - 1]) {
      a.from = a.path[a.path.length - 1];
      a.path = [];
      target = a.from;
    } else {
      target = pathPoint(a, t);
    }
  }
  let p = a.pos;
  const dt = a.posAt == null ? Infinity : (now - a.posAt) / 1000;
  if (!p || dt > 0.5 || Math.hypot(target.x - p.x, target.y - p.y) > 3) {
    // First sight, back from a hidden tab, or a teleport: just be there.
    p = { x: target.x, y: target.y };
    a.vel = { x: 0, y: 0 };
  } else if (dt > 0) {
    p = { x: p.x, y: p.y };
    const v = a.vel;
    for (let left = dt; left > 0; left -= 1 / 120) {
      const h = Math.min(left, 1 / 120);
      v.x += (FOLLOW * FOLLOW * (target.x - p.x) - 2 * FOLLOW * v.x) * h;
      v.y += (FOLLOW * FOLLOW * (target.y - p.y) - 2 * FOLLOW * v.y) * h;
      p.x += v.x * h;
      p.y += v.y * h;
    }
    const speed = Math.hypot(v.x, v.y);
    if (!a.path.length && speed < 0.05 && Math.hypot(target.x - p.x, target.y - p.y) < 0.01) {
      p = { x: target.x, y: target.y }; // settled
      a.vel = { x: 0, y: 0 };
    }
    // Face the way they're actually moving.
    if (speed > 0.6) a.turnTarget = Math.atan2(v.y, v.x);
  }
  a.moving = Math.hypot(a.vel.x, a.vel.y) > 0.4;
  a.posAt = now;
  a.pos = p;
  return p;
}

// pathPoint is the exact tile-to-tile position t ms into a path.
function pathPoint(a, t) {
  const ts = a.times;
  let i = 0;
  while (i < ts.length - 1 && ts[i] <= t) i++;
  const t0 = i === 0 ? 0 : ts[i - 1];
  const f = Math.min(1, Math.max(0, (t - t0) / (ts[i] - t0)));
  const p0 = i === 0 ? a.from : a.path[i - 1], p1 = a.path[i];
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

// draw schedules the next frame first, so a bug in one frame can never stop
// the world from rendering.
let lastDrawError = 0;
function draw(now) {
  requestAnimationFrame(draw);
  try {
    drawFrame(now);
  } catch (e) {
    if (now - lastDrawError > 5000) { lastDrawError = now; console.error("draw:", e); }
  }
}

function drawFrame(now) {
  const { w, h } = layout(now);
  // Background: the room's colour with a soft light in the middle.
  const bg = themeColor("void", "--void");
  const g = ctx.createRadialGradient(w / 2, h * 0.45, 0, w / 2, h * 0.45, Math.max(w, h) * 0.7);
  g.addColorStop(0, shade(bg, 1.05));
  g.addColorStop(1, shade(bg, 0.97));
  ctx.fillStyle = g;
  ctx.fillRect(0, 0, w, h);
  if (!state.room) { $("timeChip").hidden = true; return; }
  const z = state.view.z;
  const dt = Math.min(100, now - (state.lastFrame || now));
  state.lastFrame = now;
  state.hits = [];
  state.doorHits = [];
  state.glows = [];
  state.labelQueue = []; // labels are drawn last, so night never dims them

  // Outdoors, the plaza has a sky, a street of buildings and a day/night cycle.
  const cycle = outdoors() ? dayCycle() : null;
  state.lit = cycle ? lightsOn(cycle) : 0;
  const chip = $("timeChip");
  if (cycle) {
    const text = `${cycle.icon} ${cycle.time}${clockMode() === "real" ? " · your time" : ""}`;
    if (chip.textContent !== text) chip.textContent = text;
    chip.hidden = false;
    drawStars(w, h, cycle.night, now);
    // The skyline is drawn once per zoom and light level, then reused.
    const W = state.room.rows[0].length, H = state.room.rows.length, D = BUILDING_DEPTH + 0.6;
    const litStep = Math.round(state.lit * 20) / 20;
    const sky = [];
    for (const x of [-0.5 - D, W + 0.5]) for (const y of [-0.5 - D, H + 0.5]) for (const hh of [0, 13]) sky.push([x, y, hh]);
    worldLayer(`city|${state.room.id}|${litStep}`, extentOf(sky, z), z, () => drawSkyline(z, litStep));
    drawBillboards(z, now, state.lit);
  } else {
    chip.hidden = true;
  }

  // The floor (and, outdoors, the street) is drawn once and reused too.
  const W = state.room.rows[0].length, H = state.room.rows.length;
  const ground = [];
  for (const x of [-4.5, W + STREET_W + 0.5]) for (const y of [-4.5, H + STREET_W + 0.5]) for (const hh of [-1, 0.6]) ground.push([x, y, hh]);
  const floorKey = `floor|${state.room.id}|${themeColor("floorA", "--floor-a")}|${themeColor("floorB", "--floor-b")}`;
  worldLayer(floorKey, extentOf(ground, z), z, () => {
    if (outdoors()) drawStreet(z);
    drawFloor(z);
  });
  if (state.hover && isFloor(state.hover.x, state.hover.y) && !doorAt(state.hover.x, state.hover.y) && !propAt(state.hover.x, state.hover.y)) {
    const { sx, sy } = toScreen(state.hover.x, state.hover.y);
    const accent = theme(state.room.theme).accent;
    diamond(sx, sy, z);
    ctx.globalAlpha = 0.22;
    ctx.fillStyle = accent;
    ctx.fill();
    ctx.globalAlpha = 0.9;
    ctx.strokeStyle = accent;
    ctx.lineWidth = 2;
    ctx.setLineDash([5 * z, 4 * z]);
    ctx.stroke();
    ctx.setLineDash([]);
    ctx.globalAlpha = 1;
  }

  // Doors, props, birds and avatars, drawn back to front; anything off
  // screen is skipped (in a busy plaza that's most of the crowd).
  const visible = (x, y, up = 140) => {
    const { sx, sy } = toScreen(x, y);
    return sx > -120 * z && sx < w + 120 * z && sy > -40 * z && sy < h + up * z;
  };
  const drawables = [];
  for (const d of state.room.doors) drawables.push({ depth: d.pos[0] + d.pos[1] - 0.5, draw: () => drawDoor(d, z, now) });
  for (const p of state.room.props || []) {
    const gm = propGeom(p);
    if (!visible(gm.x, gm.y, 260)) continue;
    // Flat things (manholes, the subway's stairs) go under whoever's on them.
    const lift = p.kind === "board" ? -0.5 : p.kind === "manhole" ? -50 : p.kind === "subway" ? -1.4 : 0;
    drawables.push({ depth: gm.x + gm.y + lift, draw: () => drawProp(p, gm, z, now) });
  }
  updateBirds(now, dt);
  for (const b of state.birds) if (b.mode !== "gone" && visible(b.x, b.y)) drawables.push({ depth: b.x + b.y, draw: () => drawBird(b, z, now) });
  // Whoever is walking or turning is drawn live, at their exact angle and
  // stride, up to a budget per frame (you always); everyone else uses
  // pre-drawn frames, which keeps a packed room fast.
  let liveLeft = LIVE_AVATARS;
  state.spritesLeft = NEW_SPRITES_PER_FRAME;
  for (const a of state.avatars.values()) {
    const p = avatarPos(a, now);
    turn(a, dt);
    // The legs swing with the distance covered, so feet never skate.
    if (a.last) a.gait += Math.hypot(p.x - a.last.x, p.y - a.last.y) * GAIT_PER_TILE;
    a.last = p;
    // The stride grows over the first steps and settles when they stop.
    a.stride = Math.max(0, Math.min(1, (a.stride || 0) + (a.moving ? dt : -dt) / 160));
    if (!a.stride) a.gait = 0;
    if (!a.moving && !a.stride && a.turnTarget != null) {
      // Standing still, settle on one of the pre-drawn angles, so handing
      // over to a pre-drawn frame doesn't show.
      a.turnTarget = Math.round(a.turnTarget / AVATAR_TURN_STEP) * AVATAR_TURN_STEP;
    }
    if (!visible(p.x, p.y, 200)) continue;
    const animating = a.moving || a.stride > 0 || (a.turnTarget != null && Math.abs(angleGap(a.facing, a.turnTarget)) > 1e-3);
    a.live = animating && (a.you || liveLeft-- > 0);
    let depth = p.x + p.y;
    const seat = a.seat && propById(a.seat);
    // Sit in front of a bench that faces the camera, behind one that doesn't.
    if (seat) depth += seat.dir[0] + seat.dir[1] > 0 ? 0.6 : -0.6;
    drawables.push({ depth, draw: () => drawAvatar(a, p, now, z) });
  }
  drawables.sort((p, q) => p.depth - q.depth);
  for (const d of drawables) d.draw();
  if (cycle) drawTraffic(z, now); // the street is in front of everything

  drawParticles(z, dt);
  if (cycle) drawDaylight(w, h, cycle, state.lit, z, now);

  const labels = state.labelQueue;
  state.labelQueue = null;
  for (const l of labels) label(...l);

  // Bubbles go on top of everything.
  for (const a of state.avatars.values()) {
    const p = avatarPos(a, now);
    if (visible(p.x, p.y, 300)) drawBubbles(a, p, now, z);
  }
}

// ---- the city: sky, stars, skyline, street, traffic and daylight ----

function mixColor(a, b, t) {
  const pa = parseInt(a.slice(1), 16), pb = parseInt(b.slice(1), 16);
  const ch = (s) => Math.round(((pa >> s) & 255) + (((pb >> s) & 255) - ((pa >> s) & 255)) * t);
  return `rgb(${ch(16)} ${ch(8)} ${ch(0)})`;
}

function drawStars(w, h, night, now) {
  if (night < 0.05) return;
  for (let i = 0; i < 60; i++) {
    const x = hash2(i, 1) * w, y = hash2(i, 2) * h * 0.5;
    ctx.globalAlpha = night * (0.5 + 0.5 * Math.sin(now / 600 + i * 1.7));
    ctx.fillStyle = i % 5 ? "#ffffff" : "#ffe9a8";
    sparkle(x, y, 1.2 + hash2(i, 3) * 1.8);
  }
  ctx.globalAlpha = 1;
}

// ---- layers: big static drawings (floor, skyline) made once and reused ----
//
// A layer is drawn into its own canvas with the view moved so it fits, then
// placed under the current view each frame, so the camera can move freely.

const layerCache = new Map();

// extentOf returns the screen box of world points [x, y, height] with the
// view origin at 0, 0.
function extentOf(points, z) {
  let x0 = Infinity, y0 = Infinity, x1 = -Infinity, y1 = -Infinity;
  for (const [x, y, h] of points) {
    const sx = (x - y) * (TW / 2) * z, sy = (x + y) * (TH / 2) * z - h * ZPX * z;
    x0 = Math.min(x0, sx); y0 = Math.min(y0, sy); x1 = Math.max(x1, sx); y1 = Math.max(y1, sy);
  }
  return [x0, y0, x1, y1];
}

function worldLayer(key, extent, z, draw) {
  // Big layers (the 500-person plaza) use a lower resolution on very sharp
  // screens, to keep each under ~16 million pixels.
  const area = Math.max(1, (extent[2] - extent[0]) * (extent[3] - extent[1]));
  const dpr = Math.min(window.devicePixelRatio || 1, Math.sqrt(16e6 / area));
  const cacheKey = `${key}|${z.toFixed(3)}|${dpr}`;
  let L = layerCache.get(cacheKey);
  if (!L) {
    if (layerCache.size > 40) layerCache.clear();
    const pad = 4;
    const [x0, y0, x1, y1] = extent;
    const w = Math.ceil(x1 - x0) + pad * 2, h = Math.ceil(y1 - y0) + pad * 2;
    const c = document.createElement("canvas");
    c.width = Math.ceil(w * dpr);
    c.height = Math.ceil(h * dpr);
    const g = c.getContext("2d");
    g.setTransform(dpr, 0, 0, dpr, 0, 0);
    const saved = { ctx, ox: state.view.ox, oy: state.view.oy, glows: state.glows };
    ctx = g;
    state.view.ox = pad - x0;
    state.view.oy = pad - y0;
    state.glows = [];
    try {
      draw();
      L = { canvas: c, ax: pad - x0, ay: pad - y0, w, h, glows: state.glows };
    } finally {
      ctx = saved.ctx;
      state.view.ox = saved.ox;
      state.view.oy = saved.oy;
      state.glows = saved.glows;
    }
    layerCache.set(cacheKey, L);
  }
  const dx = state.view.ox - L.ax, dy = state.view.oy - L.ay;
  ctx.drawImage(L.canvas, dx, dy, L.w, L.h);
  for (const gl of L.glows) state.glows.push({ ...gl, quad: gl.quad.map(([x, y]) => [x + dx, y + dy]) });
}

// ---- the skyline ----
//
// Skyscrapers and walk-ups line the plaza's back edges: glass towers, Art
// Deco towers with setbacks and a spire, red-brick walk-ups with fire
// escapes and rooftop water towers, and brownstones. Laid out from a fixed
// seed, so everyone sees the same skyline.

const BUILDING_DEPTH = 1.6;
const STYLES = {
  glass: { colors: ["#bfe1f2", "#c9e6ef", "#b9d7f0"], h: [7, 10] },
  deco: { colors: ["#f5e6c8", "#efdcc0", "#f3e2d3"], h: [8, 11] },
  brick: { colors: ["#e3a28e", "#d9937e", "#e8b09a"], h: [4, 6] },
  brownstone: { colors: ["#c99a80", "#bf8e74", "#d3a58b"], h: [2.6, 3.4] },
};
const STYLE_CYCLE = ["brick", "glass", "brownstone", "deco", "brick", "glass", "brownstone", "brick", "deco", "glass"];
const AWNINGS = ["#ff8fb8", "#8fc8ff", "#7fd6a8", "#ffb36b", "#b49cf0", "#ff6f6f"];
const SIGNS = ["PIZZA", "BAGELS", "I ♥ HABIT", "DELI", "SHOWS TONIGHT", "HOT DOGS", "24 HRS", "NOODLES"];

// The storefronts for doors in the skyline row.
const DOOR_FRONTS = {
  gaming: { style: "arcade", color: "#3a2f6e", height: 4.4, sign: "ARCADE" },
  gym: { style: "gym", color: "#e3e6ec", height: 5.4, sign: "HABIT GYM", signHue: 2 },
  default: { style: "brick", color: "#e3a28e", height: 4.5, sign: null },
};

function cityBlocks() {
  if (state.room._blocks) return state.room._blocks;
  const rows = state.room.rows, W = rows[0].length, H = rows.length;
  const blocks = [];
  let i = 0, signs = 0;
  for (const side of ["right", "left"]) {
    const len = side === "right" ? W : H;
    let a = -0.5;
    while (a < len - 0.51) {
      const style = STYLE_CYCLE[(i * 3 + (side === "left" ? 5 : 0)) % STYLE_CYCLE.length];
      const s = STYLES[style];
      let width = Math.min(len - 0.5 - a, style === "brownstone" ? 2 : hash2(i * 7 + 3, 11) < 0.5 ? 3 : 4);
      // Give each door's storefront a good 4 tiles centred on the door:
      // the building before it stops short to make room.
      const door = state.room.doors.find((d) => (side === "left" ? d.pos[0] === 0 && d.pos[1] > a : d.pos[1] === 0 && d.pos[0] > a));
      if (door) {
        const dc = side === "left" ? door.pos[1] : door.pos[0];
        if (dc - 2 - a < 1.5) width = Math.min(len - 0.5 - a, dc + 2 - a); // this block is the storefront
        else width = Math.min(width, dc - 2 - a);
      }
      const height = s.h[0] + hash2(i, 5) * (s.h[1] - s.h[0]);
      const sign = (style === "brick" || style === "glass") && hash2(i, 31) < 0.55 && signs < 6 ? SIGNS[signs++ % SIGNS.length] : null;
      blocks.push({
        side, style, a0: a, a1: a + width, height, sign,
        color: s.colors[i % s.colors.length], awning: AWNINGS[(i * 5) % AWNINGS.length],
        seed: i, shop: style !== "glass" && hash2(i, 9) < 0.7, waterTower: style === "brick" && hash2(i, 13) < 0.75,
      });
      a += width;
      i++;
    }
  }
  // The doors are entrances in this row: the building around each becomes
  // that room's storefront (an arcade for Gaming, a gym for the Gym).
  for (const d of state.room.doors) {
    const side = d.pos[0] === 0 ? "left" : "right", c = side === "left" ? d.pos[1] : d.pos[0];
    const b = blocks.find((x) => x.side === side && x.a0 < c && c < x.a1);
    if (!b) continue;
    const front = DOOR_FRONTS[d.target] || DOOR_FRONTS.default;
    Object.assign(b, front, {
      door: { target: d.target, c, label: d.label }, shop: false, waterTower: false,
      signAt: [Math.max(b.a0 + 0.1, c - 1.1), Math.min(b.a1 - 0.1, c + 1.1), 1.3, 2.15],
    });
  }
  state.room._blocks = blocks;
  return blocks;
}

// pw projects a world point (tiles, height in tiles) to the screen.
function pw(x, y, h, z) {
  const { sx, sy } = toScreen(x, y);
  return [sx, sy - h * ZPX * z];
}

// drawBox draws the visible faces (top, +x, +y) of a world-aligned box.
function drawBox(x0, x1, y0, y1, h0, h1, color, z, roof = color) {
  ctx.fillStyle = shade(color, 0.86);
  poly([pw(x1, y0, h0, z), pw(x1, y1, h0, z), pw(x1, y1, h1, z), pw(x1, y0, h1, z)]);
  ctx.fillStyle = shade(color, 0.97);
  poly([pw(x0, y1, h0, z), pw(x1, y1, h0, z), pw(x1, y1, h1, z), pw(x0, y1, h1, z)]);
  ctx.fillStyle = shade(roof, 1.02);
  poly([pw(x0, y0, h1, z), pw(x1, y0, h1, z), pw(x1, y1, h1, z), pw(x0, y1, h1, z)]);
}

// Each building's footprint, plus a helper for quads on its street-facing
// wall: u runs along the street, h up the wall, out away from it.
function blockGeom(b, inset = 0) {
  const D = BUILDING_DEPTH, right = b.side === "right";
  const box = right ? [b.a0 + inset, b.a1 - inset, -0.5 - D + inset, -0.5 - inset] : [-0.5 - D + inset, -0.5 - inset, b.a0 + inset, b.a1 - inset];
  const [, x1, , y1] = box;
  const face = (u0, u1, h0, h1, out = 0, out1 = out) => right
    ? [pw(u0, y1 + out, h0, z_), pw(u1, y1 + out, h0, z_), pw(u1, y1 + out1, h1, z_), pw(u0, y1 + out1, h1, z_)]
    : [pw(x1 + out, u0, h0, z_), pw(x1 + out, u1, h0, z_), pw(x1 + out1, u1, h1, z_), pw(x1 + out1, u0, h1, z_)];
  const line = (u0, h0, u1, h1, out = 0) => right ? [pw(u0, y1 + out, h0, z_), pw(u1, y1 + out, h1, z_)] : [pw(x1 + out, u0, h0, z_), pw(x1 + out, u1, h1, z_)];
  return { box, face, line, wallShade: right ? 0.97 : 0.86 };
}
let z_ = 1; // zoom for blockGeom's helpers while the skyline draws

function drawSkyline(z, lit) {
  z_ = z;
  const D = BUILDING_DEPTH;
  // The tower on the corner where the two rows meet.
  drawBox(-0.5 - D, -0.5, -0.5 - D, -0.5, 0, 9.5, "#e9ddf3", z, "#d9c6f2");
  const order = [...cityBlocks()].sort((a, b) => a.a0 - b.a0 || (a.side === "right" ? -1 : 1));
  for (const b of order) drawBuilding(b, z, lit);
}

function drawBuilding(b, z, lit) {
  const g = blockGeom(b);
  const [x0, x1, y0, y1] = g.box;
  const wall = (c) => shade(c, g.wallShade);
  const pane = (u0, u1, h0, h1, th, day = "#d3e8ff") => {
    const q = g.face(u0, u1, h0, h1);
    const on = lit > th;
    ctx.fillStyle = on ? "#ffd97a" : day;
    poly(q);
    if (on) state.glows.push({ kind: "window", quad: q });
  };
  const grid = (u0, u1, h0, h1, cw, ch, ww, wh, salt) => {
    let row = 0;
    for (let h = h0; h + wh <= h1; h += ch, row++) {
      let col = 0;
      for (let u = u0; u + ww <= u1; u += cw, col++) pane(u, u + ww, h, h + wh, 0.1 + hash2(b.seed * 31 + col, row + salt) * 0.85);
    }
  };

  if (b.style === "arcade") {
    // The arcade: a purple front, game screens glowing in every window, a
    // pixel invader near the roof, and cabinets behind the shop windows.
    drawBox(x0, x1, y0, y1, 0, b.height, b.color, z, hexShade(b.color, 0.8));
    const c = b.door.c;
    const SCREENS = ["#5ef2ff", "#ff6fae", "#ffe14d", "#7dffb0", "#b49cf0"];
    let i = 0;
    for (let h = 2.4; h + 0.45 <= b.height - 1.15; h += 0.72) {
      for (let u = b.a0 + 0.25; u + 0.42 <= b.a1 - 0.15; u += 0.62, i++) {
        const q = g.face(u, u + 0.42, h, h + 0.45);
        const col = SCREENS[(i + b.seed) % SCREENS.length];
        ctx.fillStyle = col;
        poly(q);
        state.glows.push({ kind: "screen", quad: q, color: col });
      }
    }
    const INVADER = ["00100000100", "00010001000", "00111111100", "01101110110", "11111111111", "10111111101", "10100000101", "00011011000"];
    const px = 0.07, iu = c - (11 * px) / 2, ih = b.height - 0.3;
    INVADER.forEach((row, r) => [...row].forEach((on, k) => {
      if (on !== "1") return;
      const q = g.face(iu + k * px, iu + (k + 1) * px, ih - (r + 1) * px, ih - r * px);
      ctx.fillStyle = "#7dffb0";
      poly(q);
      state.glows.push({ kind: "screen", quad: q, color: "#7dffb0" });
    }));
    ctx.fillStyle = wall("#1b1533");
    poly(g.face(b.a0 + 0.12, b.a1 - 0.12, 0, 1.18));
    for (const [u0, u1] of [[b.a0 + 0.25, c - 0.6], [c + 0.6, b.a1 - 0.25]]) {
      for (let u = u0; u + 0.26 <= u1; u += 0.36) { // arcade cabinets
        ctx.fillStyle = wall("#3d3270");
        poly(g.face(u, u + 0.26, 0.05, 0.85));
        const sq = g.face(u + 0.04, u + 0.22, 0.5, 0.72);
        const col = SCREENS[Math.floor(u * 7) % SCREENS.length];
        ctx.fillStyle = col;
        poly(sq);
        state.glows.push({ kind: "screen", quad: sq, color: col });
      }
    }
    ctx.fillStyle = "#120d24";
    poly(g.face(c - 0.45, c + 0.45, 0, 1.08));
  } else if (b.style === "gym") {
    // The gym: concrete with a red stripe, and big windows showing weights.
    drawBox(x0, x1, y0, y1, 0, b.height, b.color, z, hexShade(b.color, 0.85));
    const c = b.door.c;
    grid(b.a0 + 0.3, b.a1 - 0.2, 2.45, b.height - 0.3, 0.6, 0.8, 0.4, 0.5, 3);
    ctx.fillStyle = wall("#ff5a5a");
    poly(g.face(b.a0, b.a1, 1.14, 1.26));
    for (const [u0, u1] of [[b.a0 + 0.15, c - 0.55], [c + 0.55, b.a1 - 0.15]]) {
      if (u1 - u0 < 0.3) continue;
      pane(u0, u1, 0.1, 1.02, hash2(b.seed, Math.round(u0 * 10)) * 0.4, "#d6ebff");
      for (let m = u0 + 0.3; m + 0.25 <= u1; m += 0.6) { // dumbbells on the rack
        ctx.fillStyle = wall("#4a4458");
        poly(g.face(m - 0.2, m + 0.2, 0.43, 0.48));
        poly(g.face(m - 0.24, m - 0.17, 0.33, 0.58));
        poly(g.face(m + 0.17, m + 0.24, 0.33, 0.58));
      }
    }
    ctx.fillStyle = "#2b3140";
    poly(g.face(c - 0.45, c + 0.45, 0, 1.08));
  } else if (b.style === "glass") {
    // A glass tower: a curtain wall of window bands, a setback crown and a mast.
    drawBox(x0, x1, y0, y1, 0, b.height, b.color, z, hexShade(b.color, 0.92));
    grid(b.a0 + 0.15, b.a1 - 0.1, 0.9, b.height - 0.2, 0.5, 0.45, 0.4, 0.3, 1);
    ctx.fillStyle = wall("#9fc7e0"); // lobby
    poly(g.face(b.a0 + 0.2, b.a1 - 0.2, 0, 0.75));
    const c = blockGeom(b, 0.45);
    drawBox(...c.box, b.height, b.height + 1, hexShade(b.color, 1.05), z);
    const mx = (c.box[0] + c.box[1]) / 2, my = (c.box[2] + c.box[3]) / 2;
    ctx.strokeStyle = "#8a8fa8";
    ctx.lineWidth = 2 * z;
    ctx.beginPath();
    ctx.moveTo(...pw(mx, my, b.height + 1, z));
    ctx.lineTo(...pw(mx, my, b.height + 2.4, z));
    ctx.stroke();
    state.glows.push({ kind: "beacon", quad: [pw(mx, my, b.height + 2.4, z)] });
  } else if (b.style === "deco") {
    // An Art Deco tower: three setbacks, pilasters, and a gold spire.
    const tiers = [[0, 0, b.height * 0.62], [0.35, b.height * 0.62, b.height * 0.85], [0.65, b.height * 0.85, b.height]];
    for (const [inset, h0, h1] of tiers) {
      const t = blockGeom(b, inset);
      drawBox(...t.box, h0, h1, b.color, z, hexShade(b.color, 0.9));
      const ua = b.a0 + inset, ub = b.a1 - inset;
      for (let u = ua + 0.2; u + 0.3 <= ub - 0.1; u += 0.45) {
        const q = t.face(u, u + 0.28, h0 + 0.25, h1 - 0.15);
        ctx.fillStyle = shade(hexShade(b.color, 0.82), g.wallShade);
        poly(q);
        for (let h = h0 + 0.35; h + 0.25 <= h1 - 0.2; h += 0.5) {
          const on = lit > 0.1 + hash2(b.seed * 17 + Math.round(u * 10), Math.round(h * 10)) * 0.85;
          const wq = t.face(u + 0.05, u + 0.23, h, h + 0.25);
          ctx.fillStyle = on ? "#ffd97a" : "#d8e6f7";
          poly(wq);
          if (on) state.glows.push({ kind: "window", quad: wq });
        }
      }
    }
    const s = blockGeom(b, 0.85);
    const mx = (s.box[0] + s.box[1]) / 2, my = (s.box[2] + s.box[3]) / 2;
    ctx.fillStyle = "#f2c560";
    poly([pw(mx - 0.18, my, b.height, z), pw(mx, my - 0.18, b.height, z), pw(mx + 0.18, my, b.height, z), pw(mx, my, b.height + 2.6, z)]);
    ctx.fillStyle = "#d9a63e";
    poly([pw(mx, my + 0.18, b.height, z), pw(mx + 0.18, my, b.height, z), pw(mx, my, b.height + 2.6, z)]);
  } else {
    // Brick walk-ups and brownstones.
    const brick = b.style === "brick";
    drawBox(x0, x1, y0, y1, 0, b.height, b.color, z, hexShade(b.color, 0.8));
    ctx.fillStyle = wall(hexShade(b.color, 0.78)); // cornice
    poly(g.face(b.a0, b.a1, b.height - 0.18, b.height, 0, 0.12));
    grid(b.a0 + 0.3, b.a1 - 0.2, 1.25, b.height - 0.3, 0.6, 0.8, 0.32, 0.46, 2);
    if (b.shop) {
      pane(b.a0 + 0.25, b.a1 - 0.25, 0.1, 0.75, hash2(b.seed, 21) * 0.5);
      const strips = Math.round((b.a1 - b.a0) * 4);
      for (let s = 0; s < strips; s++) {
        const u0 = b.a0 + 0.15 + ((b.a1 - b.a0 - 0.3) * s) / strips, u1 = b.a0 + 0.15 + ((b.a1 - b.a0 - 0.3) * (s + 1)) / strips;
        ctx.fillStyle = s % 2 ? "#ffffff" : b.awning;
        poly(g.face(u0, u1, 0.82, 1.0, 0.32, 0));
      }
    } else {
      // A front door up a stoop.
      const mid = (b.a0 + b.a1) / 2;
      ctx.fillStyle = wall("#6d4c3d");
      poly(g.face(mid - 0.2, mid + 0.2, 0.3, 0.95));
      for (let s = 0; s < 3; s++) {
        ctx.fillStyle = shade(hexShade(b.color, 0.85), 1 - s * 0.04);
        poly(g.face(mid - 0.3, mid + 0.3, 0.1 * s, 0.1 * (s + 1), 0.45 - s * 0.13, 0.45 - s * 0.13));
      }
    }
    if (brick && b.height > 3) {
      // A fire escape zig-zagging up the front.
      ctx.strokeStyle = "#4f4358";
      ctx.lineWidth = Math.max(1, 1.2 * z);
      const ua = b.a0 + 0.35, ub = Math.min(b.a1 - 0.35, ua + 1.4);
      for (let h = 1.15, f = 0; h < b.height - 0.6; h += 0.8, f++) {
        ctx.beginPath();
        for (const [a0, b0] of [g.line(ua, h, ub, h, 0.18), g.line(ua, h + 0.3, ub, h + 0.3, 0.18)]) { ctx.moveTo(...a0); ctx.lineTo(...b0); }
        const [l0, l1] = f % 2 ? g.line(ua + 0.1, h, ub - 0.1, h + 0.8, 0.12) : g.line(ub - 0.1, h, ua + 0.1, h + 0.8, 0.12);
        ctx.moveTo(...l0);
        ctx.lineTo(...l1);
        ctx.stroke();
      }
    }
    if (b.waterTower) drawWaterTower(b, z);
  }
}

// drawWaterTower puts a wooden water tank on legs on a brick roof.
function drawWaterTower(b, z) {
  const g = blockGeom(b, 0.5);
  const cx = (g.box[0] + g.box[1]) / 2 + 0.3, cy = (g.box[2] + g.box[3]) / 2;
  const [bx, by] = pw(cx, cy, b.height, z);
  const r = 0.42 * (TW / 2) * z, legs = 0.55 * ZPX * z, tank = 0.9 * ZPX * z;
  ctx.strokeStyle = "#5b4a52";
  ctx.lineWidth = Math.max(1, 1.4 * z);
  ctx.beginPath();
  for (const dx of [-0.8, -0.3, 0.3, 0.8]) { ctx.moveTo(bx + dx * r, by); ctx.lineTo(bx + dx * r * 0.9, by - legs); }
  ctx.stroke();
  const top = by - legs - tank;
  ctx.fillStyle = "#b98a6a";
  ctx.fillRect(bx - r, top, r * 2, tank);
  ctx.fillStyle = "#a57656";
  ctx.fillRect(bx, top, r, tank);
  ctx.fillStyle = "#c99a78";
  oval(bx, top, r, r * 0.45);
  ctx.fillStyle = "#6d5a62";
  poly([[bx - r * 1.05, top], [bx + r * 1.05, top], [bx, top - r * 1.1]]);
  ctx.strokeStyle = "#8a6a52";
  ctx.lineWidth = Math.max(1, z);
  for (const f of [0.3, 0.7]) { ctx.beginPath(); ctx.moveTo(bx - r, top + tank * f); ctx.lineTo(bx + r, top + tank * f); ctx.stroke(); }
}

// drawBillboards draws the neon signs live, so they can flicker and glow.
function drawBillboards(z, now, lit) {
  z_ = z;
  for (const b of cityBlocks()) {
    if (!b.sign) continue;
    const g = blockGeom(b);
    // Storefront signs sit over their door; others high on the wall.
    const [u0, u1, h0, h1] = b.signAt || [b.a0 + 0.3, b.a1 - 0.3, Math.min(b.height - 1.4, 2.6), Math.min(b.height - 1.4, 2.6) + 1.1];
    const q = g.face(u0, u1, h0, h1, 0.06, 0.06);
    const hue = b.signHue ?? (hash2(b.seed, 41) * 360 + now / 40) % 360;
    ctx.fillStyle = "#2b2338";
    poly(q);
    const flick = Math.sin(now / 90 + b.seed) > 0.97 ? 0.4 : 1;
    const color = `hsl(${hue} 95% ${60 + lit * 10}%)`;
    ctx.strokeStyle = color;
    ctx.lineWidth = 2 * z;
    ctx.beginPath();
    q.forEach(([x, y], i) => (i ? ctx.lineTo(x, y) : ctx.moveTo(x, y)));
    ctx.closePath();
    ctx.stroke();
    // The words, drawn onto the wall's plane: a 100×100 box whose x runs
    // along the sign (always left to right on screen) and y down it.
    const [p0, p1, p2, p3] = q;
    const reverse = b.side === "left"; // that wall runs right-to-left on screen
    const ux = (p1[0] - p0[0]) / 100 * (reverse ? -1 : 1), uy = (p1[1] - p0[1]) / 100 * (reverse ? -1 : 1);
    const vx = (p0[0] - p3[0]) / 100, vy = (p0[1] - p3[1]) / 100;
    const [ox, oy] = reverse ? p2 : p3;
    const words = (alpha) => {
      ctx.save();
      const dpr = window.devicePixelRatio || 1;
      ctx.setTransform(dpr * ux, dpr * uy, dpr * vx, dpr * vy, dpr * ox, dpr * oy);
      ctx.globalAlpha = flick * alpha;
      ctx.fillStyle = color;
      ctx.shadowColor = color;
      ctx.shadowBlur = 8;
      ctx.font = `900 46px ${FONT}`;
      ctx.textAlign = "center";
      ctx.textBaseline = "middle";
      ctx.fillText(b.sign, 50, 52, 92);
      ctx.restore();
    };
    words(1);
    state.glows.push({ kind: "neon", quad: q, color, words }); // lit again after dark
  }
}

// ---- the street in front: road, crossing, and traffic ----

const STREET_W = 3.6;   // two lanes of car-sized cars
const CAR_VOX = 0.105;  // vehicle voxel size: a taxi is about as tall as a person and three tiles long
const ASPHALT = "#6b687a";

function drawStreet(z) {
  const rows = state.room.rows, W = rows[0].length, H = rows.length;
  const h = -SLAB / ZPX; // the road sits a curb below the sidewalk
  const y0 = H - 0.5, x0 = W - 0.5;
  // Along the front-left edge, then the front-right, meeting at the corner.
  ctx.fillStyle = ASPHALT;
  poly([pw(-4, y0, h, z), pw(x0, y0, h, z), pw(x0, y0 + STREET_W, h, z), pw(-4, y0 + STREET_W, h, z)]);
  poly([pw(x0, -4, h, z), pw(x0 + STREET_W, -4, h, z), pw(x0 + STREET_W, y0 + STREET_W, h, z), pw(x0, y0 + STREET_W, h, z)]);
  // Dashed yellow centre lines.
  ctx.fillStyle = "#ffd45c";
  for (let x = -4; x < x0 - 0.6; x += 0.9) poly([pw(x, y0 + STREET_W / 2 - 0.03, h, z), pw(x + 0.45, y0 + STREET_W / 2 - 0.03, h, z), pw(x + 0.45, y0 + STREET_W / 2 + 0.03, h, z), pw(x, y0 + STREET_W / 2 + 0.03, h, z)]);
  for (let y = -4; y < y0 - 0.6; y += 0.9) poly([pw(x0 + STREET_W / 2 - 0.03, y, h, z), pw(x0 + STREET_W / 2 + 0.03, y, h, z), pw(x0 + STREET_W / 2 + 0.03, y + 0.45, h, z), pw(x0 + STREET_W / 2 - 0.03, y + 0.45, h, z)]);
  // A zebra crossing where the south path meets the road.
  const path = rows[H - 1].indexOf(":");
  if (path >= 0) {
    ctx.fillStyle = "#f4f2f7";
    for (let s = 0; s < 5; s++) {
      const yy = y0 + 0.15 + s * 0.4;
      poly([pw(path - 0.4, yy, h, z), pw(path + 1.4, yy, h, z), pw(path + 1.4, yy + 0.22, h, z), pw(path - 0.4, yy + 0.22, h, z)]);
    }
  }
}

// Traffic: taxis and a bus going both ways on both roads. Positions come
// from the clock, so everyone sees roughly the same traffic.
const VEHICLES = [
  { kind: "taxi", road: "left", lane: 0, speed: 1.6, phase: 0 },
  { kind: "taxi", road: "left", lane: 1, speed: 1.9, phase: 0.4 },
  { kind: "bus", road: "left", lane: 0, speed: 1.1, phase: 0.6 },
  { kind: "taxi", road: "right", lane: 0, speed: 1.7, phase: 0.2 },
  { kind: "taxi", road: "right", lane: 1, speed: 1.5, phase: 0.7 },
  { kind: "taxi", road: "right", lane: 1, speed: 1.5, phase: 0.2 },
];

function vehicleModel(kind) {
  return cachedModel("vehicle" + kind, () => {
    const vox = [];
    // Built to the characters' scale: a taxi (12×28×12 voxels) is about as
    // tall as a person and three tiles long; the bus is twice as long.
    const wheels = (vs, us) => { for (const v of vs) for (const u of us) box(vox, "wheel" + u + v, u, u, v, v + 2, 0, 2, (a, b, k) => (k === 1 && b === v + 1 ? "#9b93a8" : "#2e2738")); };
    if (kind === "taxi") {
      box(vox, "body", -6, 5, -14, 13, 1, 5, (u, v, k) => (k === 3 && Math.abs(u) > 4 && (v >> 1) % 2 === 0 ? "#3b3346" : "#ffd23f"));
      box(vox, "cabin", -5, 4, -6, 6, 6, 9, (u, v, k) => (u === -5 || u === 4 || v === -6 || v === 6 ? "#bfe3ff" : "#ffd23f"));
      box(vox, "roof", -5, 4, -6, 6, 10, 10, () => "#ffd23f");
      box(vox, "sign", -2, 1, -1, 1, 11, 12, (u, v, k) => (k === 12 ? "#ffe9a8" : "#fff6d5"));
      box(vox, "lights", -5, 4, 13, 13, 3, 4, (u) => (Math.abs(u + 0.5) > 3 ? "#fff2b0" : "#ffd23f"));
      box(vox, "tail", -5, 4, -14, -14, 3, 4, (u) => (Math.abs(u + 0.5) > 3 ? "#ff6f6f" : "#ffd23f"));
      wheels([-11, 8], [-6, 5]);
    } else {
      box(vox, "body", -6, 5, -30, 29, 1, 15, (u, v, k) => (k >= 7 && k <= 12 && v > -29 && v < 28 && (v + 30) % 6 !== 0 ? "#cfe9ff" : k === 5 ? "#ffd23f" : "#5b8def"));
      box(vox, "roof", -6, 5, -30, 29, 16, 16, () => "#e8eef8");
      box(vox, "sign", -4, 3, 30, 30, 13, 14, () => "#ffb35c");
      wheels([-24, 20], [-6, 5]);
    }
    return { voxels: finishModel(vox) };
  });
}

function drawTraffic(z, now) {
  const rows = state.room.rows, W = rows[0].length, H = rows.length;
  const y0 = H - 0.5, x0 = W - 0.5, h = -SLAB / ZPX;
  const len = (W + 8), t = now / 1000;
  const cars = VEHICLES.map((v) => {
    const s = ((t * v.speed / len + v.phase) % 1) * len - 4; // along the road
    const forward = v.lane === 0;
    const across = STREET_W / 2 + (forward ? 0.88 : -0.88);
    return v.road === "left"
      ? { v, x: forward ? s : W + 3 - s, y: y0 + across, angle: forward ? 0 : Math.PI }
      : { v, x: x0 + across, y: forward ? s : H + 3 - s, angle: forward ? Math.PI / 2 : -Math.PI / 2 };
  });
  cars.sort((a, b) => a.x + a.y - (b.x + b.y));
  for (const c of cars) {
    const { sx, sy } = toScreen(c.x, c.y);
    // A soft shadow under the car's footprint.
    const half = (c.v.kind === "bus" ? 30 : 14) * CAR_VOX, wide = 6.5 * CAR_VOX;
    const fx = Math.cos(c.angle), fy = Math.sin(c.angle);
    ctx.fillStyle = "rgb(30 20 40 / 0.2)";
    poly([[-half, -wide], [half, -wide], [half, wide], [-half, wide]].map(([a, b]) => pw(c.x + fx * a - fy * b, c.y + fy * a + fx * b, h, z)));
    drawSprite(sprite("vehicle" + c.v.kind, vehicleModel(c.v.kind), c.angle, CAR_VOX, z), sx, sy - h * ZPX * z);
    if (state.lit > 0.2) state.glows.push({ kind: "headlight", x: sx, y: sy - (h + 0.15) * ZPX * z });
  }
}

// drawDaylight tints the scene for the time of day, then lets the lights
// shine through: lamp pools and heads, lit windows, neon and headlights.
function drawDaylight(w, h, cycle, lit, z, now) {
  if (cycle.night > 0) {
    ctx.globalCompositeOperation = "multiply";
    ctx.fillStyle = mixColor("#ffffff", "#5a5fb0", cycle.night * 0.75);
    ctx.fillRect(0, 0, w, h);
    ctx.globalCompositeOperation = "source-over";
  }
  if (cycle.dusk > 0) {
    ctx.fillStyle = `rgb(255 150 110 / ${0.16 * cycle.dusk})`;
    ctx.fillRect(0, 0, w, h);
  }
  // Lit windows: all filled in one go (blurring each one is too slow).
  if (lit > 0.02) {
    ctx.beginPath();
    for (const g of state.glows) {
      if (g.kind !== "window") continue;
      g.quad.forEach(([x, y], i) => (i ? ctx.lineTo(x, y) : ctx.moveTo(x, y)));
      ctx.closePath();
    }
    ctx.globalAlpha = Math.min(1, lit * 1.2);
    ctx.fillStyle = "#ffdf8a";
    ctx.fill();
    ctx.globalAlpha = 1;
  }
  for (const g of state.glows) {
    if (g.kind === "window") continue;
    if (g.kind === "screen") {
      // Arcade screens shine through any light.
      ctx.globalAlpha = 0.3 + 0.6 * cycle.night;
      ctx.fillStyle = g.color;
      poly(g.quad);
      ctx.globalAlpha = 1;
      continue;
    }
    if (g.kind === "portal") {
      ctx.globalAlpha = 0.25 + 0.55 * cycle.night;
      ctx.fillStyle = g.color;
      poly(g.quad);
      ctx.globalAlpha = 1;
      continue;
    }
    if (g.kind === "neon") {
      // Neon is bright day and night.
      ctx.save();
      ctx.globalAlpha = 0.35 + 0.5 * cycle.night;
      ctx.strokeStyle = g.color;
      ctx.lineWidth = 2.5 * z;
      ctx.shadowColor = g.color;
      ctx.shadowBlur = 14 * z;
      ctx.beginPath();
      g.quad.forEach(([x, y], i) => (i ? ctx.lineTo(x, y) : ctx.moveTo(x, y)));
      ctx.closePath();
      ctx.stroke();
      ctx.restore();
      if (cycle.night > 0.05) g.words?.(cycle.night);
      continue;
    }
    if (lit <= 0.02) continue;
    if (g.kind === "beacon") {
      const [[x, y]] = g.quad;
      ctx.globalAlpha = lit * (0.5 + 0.5 * Math.sin(now / 400));
      ctx.fillStyle = "#ff5a6e";
      oval(x, y, 3 * z, 3 * z);
      ctx.globalAlpha = 1;
    } else if (g.kind === "headlight") {
      const glow = ctx.createRadialGradient(g.x, g.y, 0, g.x, g.y, 30 * z);
      glow.addColorStop(0, `rgb(255 240 180 / ${0.45 * lit})`);
      glow.addColorStop(1, "rgb(255 240 180 / 0)");
      ctx.fillStyle = glow;
      ctx.fillRect(g.x - 30 * z, g.y - 30 * z, 60 * z, 60 * z);
    } else if (g.kind === "lamp") {
      // A warm pool on the ground and a glow round the lamp.
      const pool = ctx.createRadialGradient(g.fx, g.fy, 0, g.fx, g.fy, 70 * z);
      pool.addColorStop(0, `rgb(255 214 120 / ${0.34 * lit})`);
      pool.addColorStop(1, "rgb(255 214 120 / 0)");
      ctx.fillStyle = pool;
      ctx.beginPath();
      ctx.ellipse(g.fx, g.fy, 70 * z, 35 * z, 0, 0, Math.PI * 2);
      ctx.fill();
      const head = ctx.createRadialGradient(g.x, g.y, 0, g.x, g.y, 28 * z);
      head.addColorStop(0, `rgb(255 230 140 / ${0.95 * lit})`);
      head.addColorStop(1, "rgb(255 230 140 / 0)");
      ctx.fillStyle = head;
      ctx.fillRect(g.x - 28 * z, g.y - 28 * z, 56 * z, 56 * z);
    }
  }
}


// ---- drawing helpers ----

function poly(points) {
  ctx.beginPath();
  ctx.moveTo(points[0][0], points[0][1]);
  for (const [px, py] of points.slice(1)) ctx.lineTo(px, py);
  ctx.closePath();
  ctx.fill();
}

function oval(x, y, rx, ry) {
  ctx.beginPath();
  ctx.ellipse(x, y, Math.max(0, rx), Math.max(0, ry), 0, 0, Math.PI * 2);
  ctx.fill();
}

// hexShade is shade() for when the result must stay a hex colour.
function hexShade(hex, f) {
  const n = parseInt(hex.slice(1), 16);
  const k = (v) => Math.round(f < 1 ? v * f : v + (255 - v) * (f - 1));
  return "#" + [(n >> 16) & 255, (n >> 8) & 255, n & 255].map((v) => k(v).toString(16).padStart(2, "0")).join("");
}

// hash2 gives each tile a stable random number, so the same tiles always
// get the same colour shift and decorations.
function hash2(x, y) {
  let h = Math.imul(x, 374761393) + Math.imul(y, 668265263);
  h = Math.imul(h ^ (h >>> 13), 1274126177);
  return ((h ^ (h >>> 16)) >>> 0) / 4294967295;
}

function heartPath(x, y, s) {
  ctx.beginPath();
  ctx.moveTo(x, y + s * 0.35);
  ctx.bezierCurveTo(x, y, x - s * 0.5, y - s * 0.1, x - s * 0.5, y + s * 0.3);
  ctx.bezierCurveTo(x - s * 0.5, y + s * 0.6, x, y + s * 0.8, x, y + s);
  ctx.bezierCurveTo(x, y + s * 0.8, x + s * 0.5, y + s * 0.6, x + s * 0.5, y + s * 0.3);
  ctx.bezierCurveTo(x + s * 0.5, y - s * 0.1, x, y, x, y + s * 0.35);
}

// sparkle draws a four-pointed twinkle.
function sparkle(x, y, r) {
  ctx.beginPath();
  ctx.moveTo(x, y - r);
  ctx.quadraticCurveTo(x, y, x + r, y);
  ctx.quadraticCurveTo(x, y, x, y + r);
  ctx.quadraticCurveTo(x, y, x - r, y);
  ctx.quadraticCurveTo(x, y, x, y - r);
  ctx.fill();
}

// isoEllipse is a flat circle of radius r tiles lying at height h tiles
// around world point (x, y).
function isoEllipse(x, y, h, r, z) {
  const { sx, sy } = toScreen(x, y);
  ctx.beginPath();
  ctx.ellipse(sx, sy - h * ZPX * z, r * Math.SQRT2 * (TW / 2) * z, r * Math.SQRT2 * (TH / 2) * z, 0, 0, Math.PI * 2);
}

// ---- voxel models ----
//
// A model is a list of voxels {part, u, v, k, c, tag, faces} in model space:
// u runs left to right, v back to front (a character's face is at +v), k
// upwards. Models are centred on u = v = 0 and stand on k = 0. faces lists
// which sides have no neighbour in the same part (0: +u, 1: -u, 2: +v, 3:
// -v, 4: top), so hidden faces are never drawn.

function finishModel(vox) {
  const key = (p, u, v, k) => `${p},${u},${v},${k}`;
  const filled = new Set(vox.map((q) => key(q.part, q.u, q.v, q.k)));
  const out = [];
  for (const q of vox) {
    const faces = [];
    if (!filled.has(key(q.part, q.u + 1, q.v, q.k))) faces.push(0);
    if (!filled.has(key(q.part, q.u - 1, q.v, q.k))) faces.push(1);
    if (!filled.has(key(q.part, q.u, q.v + 1, q.k))) faces.push(2);
    if (!filled.has(key(q.part, q.u, q.v - 1, q.k))) faces.push(3);
    if (!filled.has(key(q.part, q.u, q.v, q.k + 1))) faces.push(4);
    if (faces.length) out.push({ ...q, faces });
  }
  return out;
}

function box(vox, part, u0, u1, v0, v1, k0, k1, color) {
  for (let u = u0; u <= u1; u++) for (let v = v0; v <= v1; v++) for (let k = k0; k <= k1; k++) vox.push({ part, u, v, k, c: color(u, v, k) });
}

const modelCache = new Map();
function cachedModel(key, build) {
  if (!modelCache.has(key)) modelCache.set(key, build());
  return modelCache.get(key);
}

// styleOf turns a server style (palette indexes) into colours.
function styleOf(st = {}) {
  const at = (list, i) => list[((i | 0) % list.length + list.length) % list.length];
  return {
    skin: at(SKIN, st.skin), hair: at(HAIR, st.hair), outfit: at(OUTFITS, st.outfit),
    accessory: at(ACCESSORIES, st.accessory),
  };
}

// characterModel builds a chibi voxel character: cube head with eyes,
// blush, hair and an accessory; a small body with arms; little legs.
function characterModel(st) {
  const { skin, hair, outfit, accessory } = styleOf(st);
  return cachedModel(`char|${skin}|${hair}|${outfit}|${accessory}`, () => {
    const pants = hexShade(outfit, 0.62), shoe = "#4a3b5c";
    const vox = [];
    box(vox, "legL", -2, -1, -1, 0, 0, 2, (u, v, k) => (k === 0 ? shoe : pants));
    box(vox, "legR", 0, 1, -1, 0, 0, 2, (u, v, k) => (k === 0 ? shoe : pants));
    box(vox, "body", -3, 2, -2, 1, 3, 7, () => outfit);
    box(vox, "armL", -4, -4, -1, 0, 3, 7, (u, v, k) => (k === 3 ? skin : outfit));
    box(vox, "armR", 3, 3, -1, 0, 3, 7, (u, v, k) => (k === 3 ? skin : outfit));

    // Head: 8×8×7 with the face on the front (v = 3).
    box(vox, "head", -4, 3, -4, 3, 8, 14, (u, v, k) => {
      const front = v === 3, side = u === -4 || u === 3, back = v === -4;
      if (k === 14 || (k === 13 && !front) || (back && k >= 9) || (side && k >= 11 && v <= 1)) return hair;
      if (front && (k === 13 || (k === 12 && (u <= -3 || u >= 2)))) return hair; // fringe
      if (front && (u === -3 || u === -2 || u === 1 || u === 2) && (k === 10 || k === 11)) {
        return k === 11 && (u === -3 || u === 1) ? "#ffffff" : "#2e2135"; // eyes, with a highlight
      }
      if (front && (u === -4 || u === 3) && k === 9) return "#ff9fb5"; // blush
      if (front && (u === -1 || u === 0) && k === 9) return "#c0566e"; // mouth
      return skin;
    });
    for (const q of vox) {
      if (q.part === "head" && q.v === 3 && (q.k === 10 || q.k === 11) && [-3, -2, 1, 2].includes(q.u)) q.tag = "eye";
    }
    if (accessory === "ears") {
      for (const [u0, u1, inner] of [[-4, -3, -3], [2, 3, 2]]) {
        box(vox, "head", u0, u1, -1, 0, 15, 16, (u, v, k) => (v === 0 && k === 15 && u === inner ? "#ffb3cf" : hair));
      }
    } else if (accessory === "bow") {
      for (const v of [-2, -1, 1, 2]) box(vox, "head", 4, 4, v, v, 13, 14, () => "#ff6fae");
      vox.push({ part: "head", u: 4, v: 0, k: 13, c: "#ff3f8e" }, { part: "head", u: 4, v: 0, k: 14, c: "#ff3f8e" });
    } else if (accessory === "curl") {
      vox.push({ part: "head", u: -1, v: 0, k: 15, c: hair }, { part: "head", u: -1, v: 1, k: 16, c: hair }, { part: "head", u: 0, v: 1, k: 17, c: hair });
    }
    return { voxels: finishModel(vox), skin, height: accessory === "ears" || accessory === "curl" ? 17 : 15 };
  });
}

// doorModel is a voxel arch, 8 wide and 12 tall, in the room's colours.
function doorModel(accent) {
  return cachedModel("door" + accent, () => {
    const light = hexShade(accent, 1.4), mid = hexShade(accent, 1.2);
    const vox = [];
    const c = (u, v, k) => ((u + k) % 2 ? light : mid);
    box(vox, "frame", -4, -4, -1, 0, 0, 9, c);
    box(vox, "frame", 3, 3, -1, 0, 0, 9, c);
    box(vox, "frame", -3, 2, -1, 0, 11, 11, c);
    box(vox, "frame", -4, 3, -1, 0, 10, 10, c);
    return { voxels: finishModel(vox) };
  });
}

function decoModel(kind, r) {
  return cachedModel("deco" + kind + Math.floor(r * 20), () => {
    const vox = [];
    if (kind === "flower") {
      const petal = ["#ffc2d9", "#ffe08a", "#c9b8ff", "#ffffff"][Math.floor(r * 20) % 4];
      vox.push({ part: "d", u: 0, v: 0, k: 0, c: "#7fcf8e" });
      for (const [u, v] of [[1, 0], [-1, 0], [0, 1], [0, -1]]) vox.push({ part: "d", u, v, k: 1, c: petal });
      vox.push({ part: "d", u: 0, v: 0, k: 1, c: "#ffcf5c" });
    } else if (kind === "star") {
      vox.push({ part: "d", u: 0, v: 0, k: 0, c: r < 0.1 ? "#fde68a" : "#c4b5fd" });
    } else {
      vox.push({ part: "d", u: 0, v: 0, k: 0, c: "#8fd9b6" }, { part: "d", u: 1, v: 0, k: 0, c: "#6fc79e" }, { part: "d", u: 0, v: 1, k: 1, c: "#8fd9b6" });
    }
    return { voxels: finishModel(vox) };
  });
}

const STONE = ["#f1e8f2", "#e6dbea", "#efe4ec"];

// fountainModel: a round stone basin (2×2 tiles) with a pedestal and a
// little bowl on top. The water is drawn live (see drawFountain).
function fountainModel() {
  return cachedModel("fountain", () => {
    const vox = [];
    const stone = (u, v, k) => STONE[Math.floor(hash2(u * 3 + k, v * 7) * 3)];
    for (let u = -10; u <= 9; u++) {
      for (let v = -10; v <= 9; v++) {
        const d = Math.hypot(u + 0.5, v + 0.5);
        if (d <= 9.6 && d >= 7.9) box(vox, "rim", u, u, v, v, 0, 2, stone);
        if (d <= 3.3 && d >= 2) box(vox, "bowl", u, u, v, v, 6, 6, stone);
      }
    }
    box(vox, "pedestal", -1, 0, -1, 0, 0, 6, stone);
    return { voxels: finishModel(vox) };
  });
}

// benchModel: a wooden bench two tiles long. The sitter faces +v; the
// backrest is behind them.
function benchModel() {
  return cachedModel("bench", () => {
    const vox = [];
    const plank = (u, v, k) => ((v + 4) % 2 ? "#e2ab7d" : "#d49b6c");
    box(vox, "legs", -9, -8, -2, 1, 0, 2, () => "#8a5a3c");
    box(vox, "legs", 7, 8, -2, 1, 0, 2, () => "#8a5a3c");
    box(vox, "seat", -10, 9, -2, 1, 3, 3, plank);
    box(vox, "back", -9, -9, -3, -3, 3, 8, () => "#8a5a3c");
    box(vox, "back", 8, 8, -3, -3, 3, 8, () => "#8a5a3c");
    box(vox, "slats", -10, 9, -3, -3, 6, 6, () => "#e2ab7d");
    box(vox, "slats", -10, 9, -3, -3, 8, 8, () => "#d49b6c");
    return { voxels: finishModel(vox) };
  });
}

// treeModel: a cherry blossom tree in a stone planter, on one tile.
function treeModel() {
  return cachedModel("tree", () => {
    const vox = [];
    box(vox, "planter", -4, 3, -4, 3, 0, 2, (u, v, k) => (u === -4 || u === 3 || v === -4 || v === 3 ? (k === 2 ? "#f6eef3" : "#e9dde6") : "#9b7a62"));
    box(vox, "trunk", -1, 0, -1, 0, 2, 10, (u, v, k) => (k % 3 ? "#9b6b4e" : "#8a5d42"));
    const pinks = ["#ffc6dd", "#ffb3d1", "#ff9fc6", "#ffd6e7"];
    for (let u = -6; u <= 5; u++) {
      for (let v = -6; v <= 5; v++) {
        for (let k = 9; k <= 18; k++) {
          const e = ((u + 0.5) / 5.6) ** 2 + ((v + 0.5) / 5.6) ** 2 + ((k - 13.5) / 4.6) ** 2;
          if (e <= 1) vox.push({ part: "leaves", u, v, k, c: pinks[Math.floor(hash2(u + k * 5, v - k) * 4)] });
        }
      }
    }
    return { voxels: finishModel(vox) };
  });
}

function lampModel(lit) {
  return cachedModel("lamp" + lit, () => {
    const vox = [];
    const iron = "#6b5a7a";
    box(vox, "base", -1, 0, -1, 0, 0, 1, () => iron);
    box(vox, "pole", -1, 0, -1, 0, 2, 14, () => iron);
    box(vox, "glass", -2, 1, -2, 1, 15, 17, () => (lit ? "#ffe27a" : "#fff4cf"));
    box(vox, "cap", -2, 1, -2, 1, 18, 18, () => iron);
    vox.push({ part: "top", u: -1, v: -1, k: 19, c: iron });
    return { voxels: finishModel(vox) };
  });
}

// clockModel: a lamp-post style clock; the face is drawn live on top.
function clockModel() {
  return cachedModel("clock", () => {
    const vox = [];
    const iron = "#6b5a7a";
    box(vox, "base", -2, 1, -2, 1, 0, 1, () => iron);
    box(vox, "pole", -1, 0, -1, 0, 2, 18, () => iron);
    box(vox, "head", -3, 2, -3, 2, 19, 24, () => "#ff8fb8");
    box(vox, "cap", -2, 1, -2, 1, 25, 25, () => iron);
    return { voxels: finishModel(vox) };
  });
}

// bellModel: a little wooden frame with a golden bell that can swing.
function bellModel() {
  return cachedModel("bell", () => {
    const vox = [];
    const wood = "#c98d63";
    box(vox, "frame", -6, -6, -1, 0, 0, 13, () => wood);
    box(vox, "frame", 5, 5, -1, 0, 0, 13, () => wood);
    box(vox, "frame", -6, 5, -1, 0, 14, 14, () => "#a8714c");
    box(vox, "roof", -7, 6, -2, 1, 15, 15, () => "#ff8fb8");
    const gold = (u, v, k) => (k % 2 ? "#ffcf5c" : "#f5b942");
    box(vox, "bell", -1, 0, -1, 0, 12, 13, gold);
    box(vox, "bell", -2, 1, -2, 1, 9, 11, gold);
    box(vox, "bell", -3, 2, -3, 2, 7, 8, gold);
    box(vox, "bell", -1, 0, -1, 0, 6, 6, () => "#a8714c"); // clapper
    return { voxels: finishModel(vox) };
  });
}

// boardModel: a corkboard on two posts. As a freestanding kiosk (in the
// middle of a room) it gets a little striped roof too.
function boardModel(notes, kiosk = false) {
  return cachedModel("board" + notes + kiosk, () => {
    const wood = "#c98d63", woodDark = "#a8714c", cork = "#e9c9a3";
    const vox = [];
    if (kiosk) {
      box(vox, "roof", -7, 6, -2, 2, 13, 13, (u) => ((u + 7) % 4 < 2 ? "#ff8fb8" : "#ffffff"));
      box(vox, "roof", -6, 5, -1, 1, 14, 14, () => "#ff8fb8");
      box(vox, "base", -6, 5, -1, 1, 0, 0, () => "#e9dde6");
    }
    box(vox, "posts", -5, -5, 0, 0, 0, 12, () => woodDark);
    box(vox, "posts", 4, 4, 0, 0, 0, 12, () => woodDark);
    box(vox, "frame", -4, 3, 0, 0, 4, 4, () => wood);
    box(vox, "frame", -4, 3, 0, 0, 12, 12, () => wood);
    box(vox, "cork", -4, 3, 0, 0, 5, 11, () => cork);
    // Paper notes pinned on, one per post (up to six).
    const slots = [[-4, 9], [-1, 10], [2, 9], [-3, 6], [0, 6], [2, 5]];
    const paper = ["#ffffff", "#fff3b0", "#ffd6e8", "#d7ecff", "#e3f7d9", "#efe2ff"];
    for (let i = 0; i < notes; i++) {
      const [u, k] = slots[i];
      box(vox, "note" + i, u, u + 1, 1, 1, k, k + 1, () => paper[i]);
      vox.push({ part: "pin" + i, u: u, v: 2, k: k + 1, c: "#ff5a7a" });
    }
    return { voxels: finishModel(vox) };
  });
}

// birdModel: a little pigeon facing +v; its wings flap when it flies.
function birdModel() {
  return cachedModel("bird", () => {
    const vox = [];
    box(vox, "body", -1, 0, -2, 1, 1, 2, () => "#c4bedb");
    box(vox, "head", -1, 0, 2, 2, 2, 3, () => "#9b92bd");
    vox.push({ part: "beak", u: -1, v: 3, k: 2, c: "#ffb35c" }, { part: "beak", u: 0, v: 3, k: 2, c: "#ffb35c" });
    vox.push({ part: "tail", u: -1, v: -3, k: 2, c: "#8f86b0" }, { part: "tail", u: 0, v: -3, k: 2, c: "#8f86b0" });
    vox.push({ part: "feet", u: -1, v: 0, k: 0, c: "#ffb35c" }, { part: "feet", u: 0, v: 0, k: 0, c: "#ffb35c" });
    box(vox, "wingL", -2, -2, -1, 1, 2, 2, () => "#b0a8cf");
    box(vox, "wingR", 1, 1, -1, 1, 2, 2, () => "#b0a8cf");
    return { voxels: finishModel(vox) };
  });
}

// ---- street furniture ----

const CART_LOOKS = {
  hotdog: { stripes: ["#ff6f6f", "#ffd23f"], emoji: "🌭", name: "Hot dog cart", item: "a hot dog" },
  pretzel: { stripes: ["#5b8def", "#ffffff"], emoji: "🥨", name: "Pretzel cart", item: "a warm pretzel" },
  halal: { stripes: ["#3fbf8a", "#ffffff"], emoji: "🥙", name: "Halal cart", item: "chicken over rice" },
  coffee: { stripes: ["#a0674b", "#f6e7c8"], emoji: "☕", name: "Coffee cart", item: "a coffee" },
};

// cartModel: a street food cart, two tiles long, with a striped umbrella.
// It faces +v (towards its customers).
function cartModel(variant) {
  const look = CART_LOOKS[variant] || CART_LOOKS.hotdog;
  return cachedModel("cart" + variant, () => {
    const vox = [];
    for (const u of [-6, 5]) box(vox, "wheel" + u, u, u + 1, -4, -4, 0, 1, () => "#4a3b5c");
    box(vox, "body", -8, 7, -3, 3, 2, 7, (u, v, k) => (k === 5 ? look.stripes[0] : k === 7 ? "#c9d1da" : "#e4e9ef"));
    box(vox, "window", -6, 5, 3, 3, 4, 4, () => "#fff3c4");
    box(vox, "pole", 0, 0, 0, 0, 8, 15, () => "#5b5068");
    for (let u = -9; u <= 8; u++) {
      for (let v = -6; v <= 5; v++) {
        const edge = Math.max(Math.abs(u + 0.5) / 9, Math.abs(v + 0.5) / 6);
        if (edge <= 1) vox.push({ part: "umbrella", u, v, k: edge > 0.6 ? 16 : 17, c: look.stripes[Math.floor((u + 9) / 3) % 2] });
      }
    }
    return { voxels: finishModel(vox) };
  });
}

// subwayModel: a subway entrance (2×2 tiles) with green railings on three
// sides, open towards the camera, and two green globe lamps.
function subwayModel(lit) {
  return cachedModel("subway" + lit, () => {
    const vox = [];
    const rail = "#3d8b5f";
    const fence = (u0, u1, v0, v1) => box(vox, "rail", u0, u1, v0, v1, 0, 5, (u, v, k) => (k === 5 || (u + v) % 3 === 0 ? rail : null));
    fence(-10, 9, -10, -10);
    fence(-10, -10, -10, 6);
    fence(9, 9, -10, 6);
    for (const u of [-10, 9]) {
      box(vox, "post" + u, u, u, 8, 8, 0, 9, () => rail);
      box(vox, "globe" + u, u - 1, u + 1 > 9 ? 9 : u + 1, 7, 9, 10, 12, () => (lit ? "#b8ffd2" : "#8fe0ad"));
    }
    return { voxels: finishModel(vox.filter((q) => q.c)) };
  });
}

function hydrantModel() {
  return cachedModel("hydrant", () => {
    const vox = [];
    box(vox, "base", -2, 1, -2, 1, 0, 0, () => "#c93a40");
    box(vox, "body", -2, 1, -2, 1, 1, 6, (u, v, k) => (k === 6 ? "#ffd23f" : "#e5484d"));
    box(vox, "cap", -1, 0, -1, 0, 7, 8, () => "#ffd23f");
    box(vox, "nozzle", 2, 2, -1, 0, 3, 4, () => "#c93a40");
    box(vox, "nozzle2", -3, -3, -1, 0, 3, 4, () => "#c93a40");
    box(vox, "front", -1, 0, 2, 2, 3, 4, () => "#c93a40");
    return { voxels: finishModel(vox) };
  });
}

function newsboxModel(color) {
  return cachedModel("newsbox" + color, () => {
    const vox = [];
    for (const u of [-3, 2]) for (const v of [-2, 2]) vox.push({ part: "leg" + u + v, u, v, k: 0, c: "#4a3b5c" }, { part: "leg" + u + v, u, v, k: 1, c: "#4a3b5c" });
    box(vox, "box", -3, 2, -2, 2, 2, 9, (u, v, k) => (v === 2 && k >= 5 && k <= 8 && u > -3 && u < 2 ? "#f4f8ff" : color));
    box(vox, "top", -3, 2, -2, 2, 10, 10, () => hexShade(color, 0.8));
    return { voxels: finishModel(vox) };
  });
}

// basisAt returns a projector from model space (u, v, k, in voxels) to the
// screen, for a model whose origin is at screen point (ox, oy), turned to
// face angle (0 = +x), with voxels of size vs tiles.
function basisAt(ox, oy, angle, vs, z) {
  const F = [Math.cos(angle), Math.sin(angle)], R = [-F[1], F[0]];
  const hw = (TW / 2) * z * vs, hh = (TH / 2) * z * vs, zp = ZPX * z * vs;
  const ux = (R[0] - R[1]) * hw, uy = (R[0] + R[1]) * hh;
  const vx = (F[0] - F[1]) * hw, vy = (F[0] + F[1]) * hh;
  return { F, R, P: (a, b, c) => [ox + a * ux + b * vx, oy + a * uy + b * vy - c * zp] };
}

// basis is basisAt for a model standing at world tile (wx, wy), raised wz
// tiles.
function basis(wx, wy, wz, angle, vs, z) {
  const o = toScreen(wx, wy);
  return basisAt(o.sx, o.sy - wz * ZPX * z, angle, vs, z);
}

const FACE_N = [[1, 0], [-1, 0], [0, 1], [0, -1]];

// drawModel draws a voxel model back to front at a world position. pose
// moves whole parts ({ legL: { du, dv, dk } }); closedEyes swaps the eyes
// for a happy line.
function drawModel(model, wx, wy, wz, angle, vs, z, pose = {}, closedEyes = false) {
  const o = toScreen(wx, wy);
  drawModelAt(model, o.sx, o.sy - wz * ZPX * z, angle, vs, z, pose, closedEyes);
}

function drawModelAt(model, ox, oy, angle, vs, z, pose = {}, closedEyes = false) {
  const { F, R, P } = basisAt(ox, oy, angle, vs, z);
  // Which side faces point at the camera, and how brightly they're lit.
  const sides = FACE_N.map(([a, b]) => {
    const nx = a * R[0] + b * F[0], ny = a * R[1] + b * F[1];
    return { vis: nx + ny > 0.02, f: Math.round((0.8 + 0.13 * (nx * LIGHT.x + ny * LIGHT.y)) * 40) / 40 };
  });
  const du = R[0] + R[1], dv = F[0] + F[1];
  const items = model.voxels.map((q) => {
    const p = pose[q.part];
    const u = q.u + (p?.du || 0), v = q.v + (p?.dv || 0), k = q.k + (p?.dk || 0);
    return { q, u, v, k, d: (u + 0.5) * du + (v + 0.5) * dv + (k + 0.5) };
  });
  items.sort((a, b) => a.d - b.d);

  ctx.lineWidth = 0.6;
  ctx.lineJoin = "round";
  for (const { q, u, v, k } of items) {
    let c = q.c;
    if (q.tag === "eye" && closedEyes) c = q.k === 10 ? "#2e2135" : model.skin;
    for (const fi of q.faces) {
      let quad, f;
      if (fi === 4) {
        quad = [P(u, v, k + 1), P(u + 1, v, k + 1), P(u + 1, v + 1, k + 1), P(u, v + 1, k + 1)];
        f = 1.04;
      } else {
        if (!sides[fi].vis) continue;
        f = sides[fi].f;
        if (fi < 2) {
          const a = fi === 0 ? u + 1 : u;
          quad = [P(a, v, k), P(a, v + 1, k), P(a, v + 1, k + 1), P(a, v, k + 1)];
        } else {
          const b = fi === 2 ? v + 1 : v;
          quad = [P(u, b, k), P(u + 1, b, k), P(u + 1, b, k + 1), P(u, b, k + 1)];
        }
      }
      const col = shade(c, f);
      ctx.fillStyle = col;
      ctx.strokeStyle = col; // hides hairline seams between faces
      ctx.beginPath();
      ctx.moveTo(quad[0][0], quad[0][1]);
      ctx.lineTo(quad[1][0], quad[1][1]);
      ctx.lineTo(quad[2][0], quad[2][1]);
      ctx.lineTo(quad[3][0], quad[3][1]);
      ctx.closePath();
      ctx.fill();
      ctx.stroke();
    }
  }
}

// ---- sprites: props that don't animate are drawn once and reused ----

const spriteCache = new Map();
const CORNERS = [[0, 0, 0], [1, 0, 0], [0, 1, 0], [1, 1, 0], [0, 0, 1], [1, 0, 1], [0, 1, 1], [1, 1, 1]];

function sprite(key, model, angle, vs, z) {
  const dpr = window.devicePixelRatio || 1;
  const cacheKey = `${key}|${angle.toFixed(3)}|${vs}|${z.toFixed(3)}|${dpr}`;
  let s = spriteCache.get(cacheKey);
  if (s) return s;
  if (spriteCache.size > 300) spriteCache.clear();
  s = makeSprite(model, angle, vs, z);
  spriteCache.set(cacheKey, s);
  return s;
}

// makeSprite draws a model (in a pose) into its own little canvas.
function makeSprite(model, angle, vs, z, pose = {}, closedEyes = false) {
  const dpr = window.devicePixelRatio || 1;
  const { P } = basisAt(0, 0, angle, vs, z);
  let x0 = Infinity, y0 = Infinity, x1 = -Infinity, y1 = -Infinity;
  for (const q of model.voxels) {
    const o = pose[q.part] || {};
    for (const [a, b, c] of CORNERS) {
      const [x, y] = P(q.u + (o.du || 0) + a, q.v + (o.dv || 0) + b, q.k + (o.dk || 0) + c);
      x0 = Math.min(x0, x); y0 = Math.min(y0, y); x1 = Math.max(x1, x); y1 = Math.max(y1, y);
    }
  }
  const pad = 2;
  const w = Math.ceil(x1 - x0) + pad * 2, h = Math.ceil(y1 - y0) + pad * 2;
  const c = document.createElement("canvas");
  c.width = Math.ceil(w * dpr);
  c.height = Math.ceil(h * dpr);
  const g = c.getContext("2d");
  g.setTransform(dpr, 0, 0, dpr, 0, 0);
  const saved = ctx;
  ctx = g;
  try {
    drawModelAt(model, pad - x0, pad - y0, angle, vs, z, pose, closedEyes);
  } finally {
    ctx = saved;
  }
  return { canvas: c, ax: pad - x0, ay: pad - y0, w, h };
}

// drawSprite draws s with its origin at (sx, sy) and returns its bounds.
function drawSprite(s, sx, sy) {
  ctx.drawImage(s.canvas, sx - s.ax, sy - s.ay, s.w, s.h);
  return boundsOf(s, sx, sy);
}

function boundsOf(s, sx, sy) {
  return { x0: sx - s.ax, y0: sy - s.ay, x1: sx - s.ax + s.w, y1: sy - s.ay + s.h, s };
}

// spriteHit reports whether the sprite drawn at bounds b has a visible
// pixel at (px, py), so clicks on empty air around a prop go to the floor.
function spriteHit(b, px, py) {
  if (px < b.x0 || px > b.x1 || py < b.y0 || py > b.y1) return false;
  if (!b.s) return true;
  // Look at a small patch round the pointer, so clicks near an edge count.
  const dpr = b.s.canvas.width / b.s.w;
  const g = b.s.canvas.getContext("2d", { willReadFrequently: true });
  const r = Math.max(1, Math.round(2 * dpr));
  const x = Math.floor((px - b.x0) * dpr) - r, y = Math.floor((py - b.y0) * dpr) - r;
  const data = g.getImageData(x, y, r * 2 + 1, r * 2 + 1).data;
  for (let i = 3; i < data.length; i += 4) if (data[i] > 24) return true;
  return false;
}

// ---- floor ----

function tileColors(ch, x, y) {
  const mats = theme(state.room.theme).materials;
  if (mats) {
    const f = state.ringAt;
    if (ch === "." && mats.ring && f) {
      const d = Math.hypot(x - f.x, y - f.y);
      if (d > 2.3 && d < 3.3) return mats.ring;
    }
    return mats[ch] || mats["."];
  }
  return [themeColor("floorA", "--floor-a"), themeColor("floorB", "--floor-b")];
}

// drawFloor draws each tile as a block: a top with a slight colour shift,
// two-tone sides where the floor ends, and voxel decorations (flowers,
// stars or leaves, by community; only on grass where a room has grass).
function drawFloor(z) {
  const rows = state.room.rows;
  const t = theme(state.room.theme);
  const hw = (TW / 2) * z, hh = (TH / 2) * z, sh = SLAB * z, band = sh * 0.35;
  const tiles = [];
  for (let y = 0; y < rows.length; y++) {
    for (let x = 0; x < rows[y].length; x++) {
      if (isFloor(x, y)) tiles.push([x, y, toScreen(x, y), rows[y][x]]);
    }
  }
  const fountain = (state.room.props || []).find((p) => p.kind === "fountain");
  state.ringAt = fountain ? propGeom(fountain) : null;

  for (const [x, y, { sx, sy }, ch] of tiles) {
    const [A] = tileColors(ch, x, y);
    const L = [sx - hw, sy], Bt = [sx, sy + hh], R = [sx + hw, sy];
    const side = (p0, p1, f) => {
      ctx.fillStyle = shade(A, f * 0.95);
      poly([p0, p1, [p1[0], p1[1] + band], [p0[0], p0[1] + band]]);
      ctx.fillStyle = shade(A, f * 0.78);
      poly([[p0[0], p0[1] + band], [p1[0], p1[1] + band], [p1[0], p1[1] + sh], [p0[0], p0[1] + sh]]);
    };
    if (!isFloor(x, y + 1)) side(L, Bt, 0.92);
    if (!isFloor(x + 1, y)) side(Bt, R, 0.8);
  }
  for (const [x, y, { sx, sy }, ch] of tiles) {
    const [A, B] = tileColors(ch, x, y);
    diamond(sx, sy, z);
    ctx.fillStyle = shade((x + y) % 2 ? A : B, 0.975 + hash2(x, y) * 0.05);
    ctx.fill();
    ctx.strokeStyle = shade(A, 0.9);
    ctx.lineWidth = 1;
    ctx.stroke();
  }

  // A soft light across the whole floor.
  const c = toScreen((rows[0].length - 1) / 2, (rows.length - 1) / 2);
  const light = ctx.createRadialGradient(c.sx, c.sy - 20 * z, 0, c.sx, c.sy, (rows.length + rows[0].length) * 18 * z);
  light.addColorStop(0, "rgb(255 255 255 / 0.16)");
  light.addColorStop(1, "rgb(255 255 255 / 0)");
  ctx.fillStyle = light;
  ctx.beginPath();
  for (const [, , { sx, sy }] of tiles) {
    ctx.moveTo(sx, sy - hh);
    ctx.lineTo(sx + hw, sy);
    ctx.lineTo(sx, sy + hh);
    ctx.lineTo(sx - hw, sy);
    ctx.closePath();
  }
  ctx.fill();

  // Decorations on about one tile in four.
  for (const [x, y] of tiles) {
    const r = hash2(x, y);
    if (!t.deco || r > 0.27 || doorAt(x, y) || propAt(x, y)) continue;
    const ox = (hash2(y, x) - 0.5) * 0.5, oy = ((r * 4) % 1 - 0.5) * 0.5;
    drawModel(decoModel(t.deco, r), x + ox, y + oy, 0, r * 6, 0.05, z);
  }
}

// ---- doors ----

// drawStorefrontDoor draws a door that's part of a skyline building: a
// glowing doorway in the storefront, light spilling onto the sidewalk, and
// the head-count over the sign. It returns false if the door isn't in a
// building (indoor rooms use freestanding arches).
function drawStorefrontDoor(d, z) {
  const b = outdoors() && cityBlocks().find((x) => x.door?.target === d.target);
  if (!b) return false;
  z_ = z;
  const g = blockGeom(b), c = b.door.c, accent = theme(d.target).accent;
  const { sx, sy } = toScreen(d.pos[0], d.pos[1]);
  ctx.globalAlpha = 0.28;
  ctx.fillStyle = accent;
  oval(sx, sy, 24 * z, 11 * z);
  ctx.globalAlpha = 1;
  const q = g.face(c - 0.4, c + 0.4, 0.02, 1.0, 0.01, 0.01);
  const top = Math.min(...q.map((p) => p[1])), bottom = Math.max(...q.map((p) => p[1]));
  const grad = ctx.createLinearGradient(0, top, 0, bottom);
  grad.addColorStop(0, hexShade(accent, 0.35));
  grad.addColorStop(1, hexShade(accent, 1.2));
  ctx.fillStyle = grad;
  poly(q);
  state.glows.push({ kind: "portal", quad: q, color: accent });
  state.doorHits.push({ quad: q, pos: d.pos });
  const n = state.counts[d.target] ?? 0;
  const [lx, ly] = b.side === "left" ? pw(-0.45, c, 2.5, z) : pw(c, -0.45, 2.5, z);
  label(`${d.label} · ${n}`, lx, ly, z, accent, "#fff");
  return true;
}

// pointInPoly reports whether (x, y) is inside the polygon.
function pointInPoly(x, y, pts) {
  let inside = false;
  for (let i = 0, j = pts.length - 1; i < pts.length; j = i++) {
    const [xi, yi] = pts[i], [xj, yj] = pts[j];
    if (yi > y !== yj > y && x < ((xj - xi) * (y - yi)) / (yj - yi) + xi) inside = !inside;
  }
  return inside;
}

function doorUnder(px, py) {
  return (state.doorHits || []).find((h) => pointInPoly(px, py, h.quad));
}

function drawDoor(d, z, now) {
  if (drawStorefrontDoor(d, z)) return;
  const [dx, dy] = d.pos;
  const { sx, sy } = toScreen(dx, dy);
  const accent = theme(d.target).accent;
  // Doors sit in the back walls and face into the room.
  const angle = dx === 0 ? 0 : Math.PI / 2;
  const back = 0.42, vs = 0.1;
  const wx = dx - (dx === 0 ? back : 0), wy = dy - (dx === 0 ? 0 : back);

  // A pool of the room's colour spilling onto the floor.
  ctx.globalAlpha = 0.25;
  ctx.fillStyle = accent;
  oval(sx, sy, 22 * z, 10 * z);
  ctx.globalAlpha = 1;

  // The glowing portal, then the arch around it.
  const { P } = basis(wx, wy, 0, angle, vs, z);
  const pts = [P(-3, 0, 0), P(3, 0, 0), P(3, 0, 10), P(-3, 0, 10)];
  const top = Math.min(...pts.map((p) => p[1])), bottom = Math.max(...pts.map((p) => p[1]));
  const pg = ctx.createLinearGradient(0, top, 0, bottom);
  pg.addColorStop(0, hexShade(accent, 0.45));
  pg.addColorStop(1, hexShade(accent, 1.15));
  ctx.save();
  ctx.shadowColor = accent;
  ctx.shadowBlur = 16 * z;
  ctx.fillStyle = pg;
  poly(pts);
  ctx.restore();
  const o = toScreen(wx, wy);
  drawSprite(sprite("door" + accent, doorModel(accent), angle, vs, z), o.sx, o.sy);

  // A little voxel star that bobs on top.
  const bob = Math.sin(now / 400 + dx * 2) * 0.03;
  drawModel(decoModel("star", 0.05), wx, wy, 12.4 * vs + bob, now / 900, 0.09, z);

  const n = state.counts[d.target] ?? 0;
  label(`${d.label} · ${n}`, sx, sy - 15 * vs * ZPX * z - 8 * z, z, accent, "#fff");
}

// ---- props ----

function propAt(x, y) {
  return (state.room?.props || []).find((p) => x >= p.at[0] && x < p.at[0] + p.w && y >= p.at[1] && y < p.at[1] + p.h);
}

function propById(id) {
  return (state.room?.props || []).find((p) => p.id === id);
}

// propGeom is where a prop stands (its centre, in tiles) and which way it
// faces. Boards hang on the back wall.
function propGeom(p) {
  let x = p.at[0] + p.w / 2 - 0.5, y = p.at[1] + p.h / 2 - 0.5, angle = Math.PI / 4;
  if (p.kind === "bench" || p.kind === "cart") {
    angle = Math.atan2(p.dir[1], p.dir[0]);
    x -= p.dir[0] * 0.12;
    y -= p.dir[1] * 0.12;
  } else if (p.kind === "board") {
    if (p.at[1] === 0) { angle = Math.PI / 2; y -= 0.42; } else if (p.at[0] === 0) { angle = 0; x -= 0.42; }
  }
  return { x, y, angle };
}

function addHit(p, b) {
  state.hits.push({ id: p.id, kind: p.kind, ...b });
}

function drawProp(p, gm, z, now) {
  const { sx, sy } = toScreen(gm.x, gm.y);
  switch (p.kind) {
    case "fountain": {
      drawFountain(gm, z, now, "under");
      addHit(p, drawSprite(sprite("fountain", fountainModel(), 0, 0.1, z), sx, sy));
      drawFountain(gm, z, now, "over");
      break;
    }
    case "bench":
      addHit(p, drawSprite(sprite("bench", benchModel(), gm.angle, 0.1, z), sx, sy));
      break;
    case "tree": {
      const s = sprite("tree", treeModel(), gm.angle, 0.1, z);
      const t = (now - (state.shakes[p.id] || -1e9)) / 900;
      if (t < 1) {
        ctx.save();
        ctx.translate(sx, sy);
        ctx.rotate(Math.sin(t * Math.PI * 7) * 0.07 * (1 - t));
        drawSprite(s, 0, 0);
        ctx.restore();
        addHit(p, boundsOf(s, sx, sy));
      } else {
        addHit(p, drawSprite(s, sx, sy));
      }
      break;
    }
    case "lamp": {
      const lit = state.lit > 0.3;
      const s = drawSprite(sprite("lamp" + lit, lampModel(lit), gm.angle, 0.08, z), sx, sy);
      if (state.lit > 0.02) state.glows.push({ kind: "lamp", x: sx, y: s.y0 + 8 * z, fx: sx, fy: sy });
      break;
    }
    case "clock": {
      const s = drawSprite(sprite("clock", clockModel(), gm.angle, 0.08, z), sx, sy);
      addHit(p, s);
      drawClockFace(sx, s.y0 + 11 * z, 10 * z, dayCycle());
      break;
    }
    case "cart": {
      const look = CART_LOOKS[p.variant] || CART_LOOKS.hotdog;
      const s = drawSprite(sprite("cart" + p.variant, cartModel(p.variant), gm.angle, 0.09, z), sx, sy);
      addHit(p, s);
      // The cart's sign: what it sells, bobbing gently.
      ctx.font = `${Math.round(18 * z)}px system-ui, sans-serif`;
      ctx.textAlign = "center";
      ctx.textBaseline = "middle";
      ctx.fillText(look.emoji, sx, s.y0 - 8 * z + Math.sin(now / 500 + p.at[0]) * 2 * z);
      break;
    }
    case "subway": {
      // Stairs going down, then the railings and globes round them.
      const steps = 6;
      for (let i = 0; i < steps; i++) {
        const t0 = i / steps, t1 = (i + 1) / steps;
        const a = p.at[0] + 0.1, b = p.at[0] + p.w - 0.1, y0 = p.at[1] + 0.05 + t0 * (p.h - 0.2), y1 = p.at[1] + 0.05 + t1 * (p.h - 0.2);
        ctx.fillStyle = mixColor("#2a2533", "#8d879a", 1 - t0);
        poly([pw(a, y0, -0.4 * (1 - t0), z), pw(b, y0, -0.4 * (1 - t0), z), pw(b, y1, -0.4 * (1 - t1), z), pw(a, y1, -0.4 * (1 - t1), z)]);
      }
      const lit = state.lit > 0.3;
      const s = drawSprite(sprite("subway" + lit, subwayModel(lit), gm.angle, 0.1, z), sx, sy);
      addHit(p, s);
      if (state.lit > 0.02) for (const dx of [-1, 1]) state.glows.push({ kind: "lamp", x: sx + dx * 30 * z, y: s.y0 + 10 * z, fx: sx + dx * 30 * z, fy: sy });
      label("🚇 Subway", sx, s.y0 - 10 * z, z * 0.85, "#2f7d4f", "#fff");
      break;
    }
    case "hydrant":
      addHit(p, drawSprite(sprite("hydrant", hydrantModel(), gm.angle, 0.06, z), sx, sy));
      break;
    case "newsbox": {
      const color = ["#4f7fd6", "#e5484d", "#f5c242"][Math.floor(hash2(p.at[0], p.at[1]) * 3)];
      addHit(p, drawSprite(sprite("newsbox" + color, newsboxModel(color), gm.angle, 0.07, z), sx, sy));
      break;
    }
    case "manhole": {
      ctx.fillStyle = "#6d6878";
      isoEllipse(gm.x, gm.y, 0.01, 0.32, z);
      ctx.fill();
      ctx.strokeStyle = "#57525f";
      ctx.lineWidth = 1.2 * z;
      for (const r of [0.22, 0.12]) { isoEllipse(gm.x, gm.y, 0.01, r, z); ctx.stroke(); }
      break;
    }
    case "bell": {
      const t = (now - (state.bellAt || -1e9)) / 1600;
      const swing = t < 1 ? Math.sin(t * Math.PI * 6) * 1.4 * (1 - t) : 0;
      drawModel(bellModel(), gm.x, gm.y, 0, gm.angle, 0.09, z, { bell: { du: swing } });
      addHit(p, boundsOf(sprite("bell", bellModel(), gm.angle, 0.09, z), sx, sy)); // at rest, for clicks
      break;
    }
    case "board": {
      const n = boardCount(), notes = Math.min(6, n);
      const kiosk = p.at[0] !== 0 && p.at[1] !== 0; // not against a wall
      const b = drawSprite(sprite(`board${notes}${kiosk}`, boardModel(notes, kiosk), gm.angle, kiosk ? 0.11 : 0.09, z), sx, sy);
      addHit(p, b);
      const more = state.board?.more ? "+" : "";
      label(`📌 Board · ${n}${more}`, sx, b.y0 - 8 * z, z, theme(state.room.theme).accent, "#fff");
      break;
    }
  }
}

// drawFountain draws the water: the basin pool under the stone, and the
// little bowl and jets over it.
function drawFountain(gm, z, now, layer) {
  if (layer === "under") {
    isoEllipse(gm.x, gm.y, 0.16, 0.82, z);
    const { sx, sy } = toScreen(gm.x, gm.y);
    const wg = ctx.createLinearGradient(0, sy - 30 * z, 0, sy + 20 * z);
    wg.addColorStop(0, "#bdeeff");
    wg.addColorStop(1, "#7fd2f5");
    ctx.fillStyle = wg;
    ctx.fill();
    // Ripples spreading from the middle.
    ctx.strokeStyle = "rgb(255 255 255 / 0.55)";
    ctx.lineWidth = 1.2;
    for (let i = 0; i < 3; i++) {
      const t = ((now / 2600) + i / 3) % 1;
      ctx.globalAlpha = 1 - t;
      isoEllipse(gm.x, gm.y, 0.16, 0.25 + t * 0.55, z);
      ctx.stroke();
    }
    ctx.globalAlpha = 1;
    return;
  }
  // The bowl's water and the jets falling into the pool.
  isoEllipse(gm.x, gm.y, 0.7, 0.24, z);
  ctx.fillStyle = "#a9e6fb";
  ctx.fill();
  for (let i = 0; i < 8; i++) {
    const a = (i / 8) * Math.PI * 2 + now / 3000;
    for (let j = 0; j < 4; j++) {
      const t = ((now / 900) + i / 8 + j / 4) % 1;
      const d = 0.2 + t * 0.45;
      const h = 0.72 + 4 * 0.12 * t * (1 - t) - t * 0.56;
      const { sx, sy } = toScreen(gm.x + Math.cos(a) * d, gm.y + Math.sin(a) * d);
      ctx.globalAlpha = 0.85 - t * 0.4;
      ctx.fillStyle = j % 2 ? "#ffffff" : "#c8f0ff";
      oval(sx, sy - h * ZPX * z, 1.6 * z, 1.6 * z);
    }
  }
  ctx.globalAlpha = 1;
}

// drawClockFace draws the plaza clock's face, showing plaza time.
function drawClockFace(x, y, r, cycle) {
  ctx.fillStyle = "#ffffff";
  ctx.strokeStyle = "#ff8fb8";
  ctx.lineWidth = r * 0.22;
  ctx.beginPath();
  ctx.arc(x, y, r, 0, Math.PI * 2);
  ctx.fill();
  ctx.stroke();
  ctx.fillStyle = "#c9b8d8";
  for (let i = 0; i < 12; i += 3) oval(x + Math.sin((i / 12) * Math.PI * 2) * r * 0.72, y - Math.cos((i / 12) * Math.PI * 2) * r * 0.72, r * 0.09, r * 0.09);
  const hand = (turns, len, width) => {
    ctx.lineWidth = width;
    ctx.lineCap = "round";
    ctx.beginPath();
    ctx.moveTo(x, y);
    ctx.lineTo(x + Math.sin(turns * Math.PI * 2) * len, y - Math.cos(turns * Math.PI * 2) * len);
    ctx.stroke();
  };
  ctx.strokeStyle = "#4a3b5c";
  hand((cycle.hour % 12) / 12, r * 0.5, r * 0.16);
  hand(cycle.hour % 1, r * 0.75, r * 0.1);
}

function boardCount() {
  const b = state.board;
  return b && b.room === state.room?.id ? b.threads.length : 0;
}

// ---- effects from using props ----

function handleFx(m) {
  const p = propById(m.prop);
  if (!p) return;
  const gm = propGeom(p);
  const now = performance.now();
  const a = state.avatars.get(m.by);
  if (a) {
    a.turnTarget = Math.atan2(gm.y - a.from.y, gm.x - a.from.x);
    a.actAt = now;
  }
  if (p.kind === "fountain") {
    p.count = m.count;
    const tip = state.tip;
    if (tip?.id === p.id && !$("tooltip").hidden) showTip(PROP_TIPS.fountain(p), tip.px, tip.py, tip.r);
    const from = a ? a.from : { x: gm.x + 1, y: gm.y + 1 };
    state.particles.push({ kind: "coin", x: from.x, y: from.y, x1: gm.x, y1: gm.y, h: 30, vx: 0, vh: 0, life: 0, max: 650, color: "#ffcf3f", size: 3 });
    if (m.by === state.you) toast(`🪙 You made wish #${m.count} ✨`);
  } else if (p.kind === "tree") {
    state.shakes[p.id] = now;
    for (let i = 0; i < 16; i++) {
      state.particles.push({
        kind: "petal", x: gm.x + (Math.random() - 0.5) * 1.1, y: gm.y + (Math.random() - 0.5) * 1.1,
        h: 30 + Math.random() * 18, vx: (Math.random() - 0.5) * 10, vh: -(10 + Math.random() * 10),
        life: -Math.random() * 400, max: 2600, color: ["#ffc6dd", "#ff9fc6", "#ffffff"][i % 3], size: 3 + Math.random() * 2,
      });
    }
  } else if (p.kind === "bell") {
    state.bellAt = now;
    for (let i = 0; i < 3; i++) {
      state.particles.push({ kind: "wave", x: gm.x, y: gm.y, h: 0, vx: 0, vh: 0, life: -i * 350, max: 1400, color: theme(state.room.theme).accent, size: 0 });
    }
    // Everyone in the plaza turns towards the bell and hops.
    for (const o of state.avatars.values()) {
      if (!o.path.length && !o.seat) o.turnTarget = Math.atan2(gm.y - o.from.y, gm.x - o.from.x);
      o.hopAt = now + Math.random() * 250;
    }
  } else if (p.kind === "cart") {
    // Whoever bought something carries it for a while.
    const look = CART_LOOKS[m.variant] || CART_LOOKS.hotdog;
    if (a) a.holding = { emoji: look.emoji, until: now + 30000 };
    if (m.by === state.you) toast(`${look.emoji} You grabbed ${look.item}!`);
  } else if (p.kind === "hydrant") {
    // A big spray of water arcing out and splashing down.
    for (let i = 0; i < 34; i++) {
      const ang = Math.random() * Math.PI * 2;
      state.particles.push({
        kind: "water", x: gm.x, y: gm.y, h: 8, vx: Math.cos(ang) * (20 + Math.random() * 30), vy: Math.sin(ang) * 0.6,
        vh: 30 + Math.random() * 35, life: -Math.random() * 300, max: 1300, color: i % 3 ? "#9fdcff" : "#ffffff", size: 2.5 + Math.random() * 2,
      });
    }
    for (let i = 0; i < 2; i++) state.particles.push({ kind: "ring", x: gm.x, y: gm.y, h: 0.01, vx: 0, vh: 0, life: -i * 300, max: 900, color: "#9fdcff", size: 0 });
  }
}

const HEADLINES = [
  "Local pigeon elected mayor 🐦",
  "Pretzel prices hold steady for 40th year 🥨",
  "Fountain wishes up 300% this week ⛲",
  "Stranger waves; other stranger waves back 👋",
  "Subway runs on time, riders stunned 🚇",
  "Hot dog vendor wins Best Smile award 🌭",
  "Cherry trees refuse to stop blooming 🌸",
  "Area resident finally says hi in chat 💬",
  "Taxi stops for pedestrian; city celebrates 🚕",
  "Scientists confirm the bell is, in fact, very nice 🔔",
];

// ---- avatars ----

// turn eases an avatar's facing towards where it wants to look, the short
// way round.
function turn(a, dt) {
  if (a.turnTarget == null) return;
  const d = angleGap(a.facing, a.turnTarget);
  const step = (TURN_SPEED * dt) / 1000;
  a.facing = angleGap(0, a.facing + (Math.abs(d) <= step ? d : Math.sign(d) * step)); // kept in -π..π
}

// angleGap is the signed short way round from angle a to angle b.
function angleGap(a, b) {
  return ((((b - a) % (2 * Math.PI)) + 3 * Math.PI) % (2 * Math.PI)) - Math.PI;
}

function drawAvatar(a, p, now, z) {
  const { sx, sy } = toScreen(p.x, p.y);
  const model = characterModel(a.style);
  const emoting = a.emote && now - a.emoteAt < EMOTE_MS;
  const walking = a.moving;
  const seat = !walking && a.seat ? propById(a.seat) : null;
  if (seat) a.turnTarget = Math.atan2(seat.dir[1], seat.dir[0]);

  const holding = a.holding && now < a.holding.until ? a.holding : null;
  if (!holding) a.holding = null;

  // Anyone walking or turning is usually drawn live, at their exact angle
  // and stride. Otherwise characters use a few stock poses (standing, a walking
  // frame, sitting, carrying a snack) seen from one of a set of angles;
  // those are drawn once and reused, which is what lets hundreds of people
  // share a room. Emotes and one-off moves are always drawn live.
  let live = !!a.live;
  let angle = live ? a.facing : Math.round(a.facing / AVATAR_TURN_STEP) * AVATAR_TURN_STEP;
  let wz = 0;
  const pose = {};
  let poseKey = "stand";
  if (walking || (live && a.stride > 0)) {
    const cycle = ((a.gait % (Math.PI * 2)) + Math.PI * 2) % (Math.PI * 2);
    const frame = Math.floor((cycle / (Math.PI * 2)) * WALK_FRAMES);
    const ph = live ? Math.sin(cycle) * a.stride : Math.sin(((frame + 0.5) / WALK_FRAMES) * Math.PI * 2);
    pose.legL = { dv: ph * 1.3 };
    pose.legR = { dv: -ph * 1.3 };
    pose.armL = { dv: -ph * 1.1 };
    pose.armR = { dv: ph * 1.1 };
    wz = ph * ph * 0.04; // a soft bob on each step
    poseKey = "walk" + frame;
  }
  if (seat) {
    wz = 0.06;
    pose.legL = { dv: 2, dk: 1.5 };
    pose.legR = { dv: 2, dk: 1.5 };
    poseKey = "seat";
  }
  if (holding) {
    // Carrying a snack: the right arm holds it out in front, steady even
    // while walking, and the snack sits in that hand.
    pose.armR = { dv: 1.6, dk: 1.4 };
    poseKey += "+snack";
  }
  if (emoting) {
    live = true;
    const t = (now - a.emoteAt) / EMOTE_MS;
    switch (a.emote) {
      case "wave": pose.armR = { du: 1.2, dk: 4.5 + Math.sin(now / 90) * 0.5 }; break;
      case "dance": angle += t * Math.PI * 4; wz += Math.abs(Math.sin(now / 120)) * 0.07; break; // two happy spins
      case "laugh": pose.head = { dk: Math.abs(Math.sin(now / 90)) * 0.6 }; break;
      case "heart": pose.armL = { dv: 1.5, dk: 2 }; pose.armR = { dv: 1.5, dk: 2 }; break;
      case "sit": if (!seat) { wz = -2 * VOX; pose.legL = { dv: 2, dk: 2 }; pose.legR = { dv: 2, dk: 2 }; } break;
    }
  }
  if (now - (a.actAt || -1e9) < 500) { pose.armR = { dv: 1.5, dk: 3.5 }; live = true; } // reaching to use something
  const hop = (now - (a.hopAt || -1e9)) / 380;
  if (hop > 0 && hop < 1) wz += Math.sin(hop * Math.PI) * 0.14;
  const blink = (now + a.seed * 4000) % 4200 < 140;
  const happy = emoting && a.emote !== "sit";
  const closed = happy || blink;

  // Soft shadow.
  ctx.fillStyle = "rgb(90 50 90 / 0.15)";
  oval(sx, sy, 13 * z - wz * 40 * z, 5.5 * z);

  const body = () => {
    if (live) {
      drawModel(model, p.x, p.y, wz, angle, VOX, z, pose, closed);
    } else {
      // When lots of new poses are needed at once (a crowd arriving), only
      // a few are drawn each frame; the rest keep their last look a moment.
      const s = avatarSprite(model, a.style, angle, poseKey, pose, closed, z, a.sprite);
      a.sprite = s;
      ctx.drawImage(s.canvas, sx - s.ax, sy - wz * ZPX * z - s.ay, s.w, s.h);
    }
  };
  if (holding) {
    // The snack goes in the right hand, in front of the body or behind it
    // depending on which way they face.
    const { F, R, P } = basis(p.x, p.y, wz, angle, VOX, z);
    const ar = pose.armR || {};
    const hu = 3.5 + (ar.du || 0), hv = 0.9 + (ar.dv || 0), hk = 4.2 + (ar.dk || 0);
    const [hx, hy] = P(hu, hv, hk);
    const inFront = hu * (R[0] + R[1]) + hv * (F[0] + F[1]) >= 0;
    const snack = () => {
      ctx.font = `${Math.round(14 * z)}px system-ui, sans-serif`;
      ctx.textAlign = "center";
      ctx.textBaseline = "middle";
      ctx.fillText(holding.emoji, hx, hy);
    };
    if (inFront) { body(); snack(); } else { snack(); body(); }
  } else {
    body();
  }

  // Name tags show for you and whoever is under the pointer, like Habbo,
  // so a busy room doesn't turn into a wall of labels.
  const top = sy - (model.height + 1) * VOX * ZPX * z - wz * ZPX * z;
  const hovered = state.hover && Math.round(p.x) === state.hover.x && Math.round(p.y) === state.hover.y;
  if (a.you || hovered) {
    const tag = a.you ? `${a.name} (you)` : a.name;
    label(tag, sx, top - 8 * z, z * 0.85, a.you ? theme(state.room.theme).accent : "rgb(74 59 92 / 0.82)", "#fff");
  }
  // Newcomers get a little twinkle; the "new" label only shows on you or
  // whoever you hover.
  if (a.newbie && (a.you || hovered)) {
    label("new ✨", sx, top - 26 * z, z * 0.75, "#3cc28f", "#fff");
  } else if (a.newbie) {
    ctx.fillStyle = "#ffcf3f";
    sparkle(sx + 16 * z, top + 4 * z, (3.6 + Math.sin(now / 250 + a.seed * 6) * 0.9) * z);
  }
  if (emoting) {
    ctx.font = `${Math.round(20 * z)}px system-ui, sans-serif`;
    ctx.textAlign = "center";
    ctx.textBaseline = "middle";
    ctx.fillText(EMOJI[a.emote], sx + 24 * z, top + 10 * z);
  }
}

// ---- avatar sprites: stock poses drawn once, most recently used kept ----

const AVATAR_TURN_STEP = Math.PI / 12; // 24 directions
const WALK_FRAMES = 12;
const GAIT_PER_TILE = 2.6; // radians of stride per tile walked
const LIVE_AVATARS = 6;   // at most this many moving people drawn live a frame
const NEW_SPRITES_PER_FRAME = 6;
const AVATAR_SPRITE_CAP = 2500;
const avatarSprites = new Map();

function avatarSprite(model, style, angle, poseKey, pose, closed, z, fallback) {
  const dpr = window.devicePixelRatio || 1;
  const s = style || {};
  const key = `${s.skin}.${s.hair}.${s.outfit}.${s.accessory}|${Math.round(angle / AVATAR_TURN_STEP)}|${poseKey}|${closed ? 1 : 0}|${z.toFixed(3)}|${dpr}`;
  let sp = avatarSprites.get(key);
  if (sp) {
    avatarSprites.delete(key); // move to the back: most recently used
    avatarSprites.set(key, sp);
    return sp;
  }
  if (fallback && fallback.z === z && state.spritesLeft <= 0) return fallback;
  state.spritesLeft--;
  sp = makeSprite(model, angle, VOX, z, pose, closed);
  sp.z = z;
  avatarSprites.set(key, sp);
  if (avatarSprites.size > AVATAR_SPRITE_CAP) avatarSprites.delete(avatarSprites.keys().next().value);
  return sp;
}

// ---- birds: pigeons that peck about the plaza and fly off if you get close ----

function standableTile() {
  const rows = state.room.rows;
  for (let tries = 0; tries < 60; tries++) {
    const x = Math.floor(Math.random() * rows[0].length), y = Math.floor(Math.random() * rows.length);
    if (rows[y][x] !== "." || propAt(x, y) || doorAt(x, y)) continue;
    if ([...state.avatars.values()].some((a) => Math.hypot(a.from.x - x, a.from.y - y) < 2.5)) continue;
    return { x, y };
  }
  return null;
}

function updateBirds(now, dt) {
  if (state.room.kind !== "plaza") { state.birds = []; return; }
  if (!state.birds.length) {
    for (let i = 0; i < 14; i++) state.birds.push({ mode: "gone", respawnAt: now + i * 700 });
  }
  const s = dt / 1000;
  for (const b of state.birds) {
    if (b.mode === "gone") {
      if (now < b.respawnAt) continue;
      const t = standableTile();
      if (!t) { b.respawnAt = now + 3000; continue; }
      Object.assign(b, { mode: "idle", x: t.x + (Math.random() - 0.5) * 0.6, y: t.y + (Math.random() - 0.5) * 0.6, h: 0, facing: Math.random() * 6.3, nextHop: now + 800 + Math.random() * 2500, peck: Math.random() * 10 });
      continue;
    }
    if (b.mode === "idle") {
      const near = [...state.avatars.values()].find((a) => { const p = avatarPos(a, now); return Math.hypot(p.x - b.x, p.y - b.y) < 1.4; });
      if (near) {
        const p = avatarPos(near, now);
        const away = Math.atan2(b.y - p.y, b.x - p.x) + (Math.random() - 0.5) * 0.8;
        Object.assign(b, { mode: "fly", facing: away, vx: Math.cos(away) * 3, vy: Math.sin(away) * 3, flyT: 0 });
      } else if (now > b.nextHop) {
        const a = b.facing + (Math.random() - 0.5) * 2;
        const nx = b.x + Math.cos(a) * 0.25, ny = b.y + Math.sin(a) * 0.25;
        if (isFloor(Math.round(nx), Math.round(ny)) && !propAt(Math.round(nx), Math.round(ny))) Object.assign(b, { x: nx, y: ny, facing: a, hopT: now });
        b.nextHop = now + 900 + Math.random() * 2500;
      }
    } else if (b.mode === "fly") {
      b.flyT += s;
      b.x += b.vx * s;
      b.y += b.vy * s;
      b.h += 1.6 * s;
      if (b.flyT > 1.8) Object.assign(b, { mode: "gone", respawnAt: now + 6000 + Math.random() * 6000 });
    }
  }
}

function drawBird(b, z, now) {
  const flying = b.mode === "fly";
  const pose = {};
  if (flying) {
    const f = Math.sin(now / 45) * 1.2;
    pose.wingL = { dk: f, du: -0.5 };
    pose.wingR = { dk: f, du: 0.5 };
  } else {
    pose.head = { dk: Math.max(0, Math.sin(now / 300 + b.peck)) > 0.95 ? -1 : 0 }; // the odd peck
  }
  const hopH = b.hopT && now - b.hopT < 200 ? Math.sin(((now - b.hopT) / 200) * Math.PI) * 0.05 : 0;
  const { sx, sy } = toScreen(b.x, b.y);
  if (!flying) {
    ctx.fillStyle = "rgb(90 50 90 / 0.12)";
    oval(sx, sy, 5 * z, 2.2 * z);
  }
  ctx.globalAlpha = flying ? Math.max(0, 1 - Math.max(0, b.flyT - 1.2) / 0.6) : 1;
  drawModel(birdModel(), b.x, b.y, (b.h || 0) + hopH, b.facing, 0.045, z, pose);
  ctx.globalAlpha = 1;
}

// ---- particles: ambient sparkles, hearts, coins, petals, splashes, bell waves ----

function spawnHearts(avatarId) {
  const a = state.avatars.get(avatarId);
  if (!a) return;
  const p = avatarPos(a, performance.now());
  for (let i = 0; i < 7; i++) {
    state.particles.push({
      kind: "heart", x: p.x, y: p.y, h: 40 + Math.random() * 10,
      vx: (Math.random() - 0.5) * 14, vh: 26 + Math.random() * 18, life: 0, max: 1600 + Math.random() * 600,
      color: ["#ff6fae", "#ff9ecb", "#ff4f8b"][i % 3], size: 6 + Math.random() * 4,
    });
  }
}

function drawParticles(z, dt) {
  const rows = state.room.rows;
  // Steam puffs up from the manholes.
  for (const p of state.room.props || []) {
    if (p.kind === "manhole" && Math.random() < dt / 260) {
      state.particles.push({
        kind: "steam", x: p.at[0] + (Math.random() - 0.5) * 0.2, y: p.at[1] + (Math.random() - 0.5) * 0.2, h: 1,
        vx: (Math.random() - 0.3) * 6, vh: 12 + Math.random() * 8, life: 0, max: 2400 + Math.random() * 800, color: "#ffffff", size: 5 + Math.random() * 3,
      });
    }
  }
  // Keep a few sparkles drifting up from the floor.
  const ambient = state.particles.filter((q) => q.kind === "sparkle").length;
  if (ambient < 9 && Math.random() < dt / 300) {
    let x, y;
    do { x = Math.floor(Math.random() * rows[0].length); y = Math.floor(Math.random() * rows.length); } while (!isFloor(x, y));
    state.particles.push({
      kind: "sparkle", x: x + Math.random() - 0.5, y: y + Math.random() - 0.5, h: 4,
      vx: (Math.random() - 0.5) * 4, vh: 9 + Math.random() * 8, life: 0, max: 2600 + Math.random() * 1800,
      color: Math.random() < 0.5 ? "#ffffff" : "#ffe08a", size: 2.5 + Math.random() * 2,
    });
  }
  const s = dt / 1000;
  const spawned = [];
  state.particles = state.particles.filter((q) => {
    q.life += dt;
    if (q.life < q.max) return true;
    if (q.kind === "coin") {
      // Splash! A ring on the water and a burst of sparkles.
      spawned.push({ kind: "ring", x: q.x1, y: q.y1, h: 0.16, vx: 0, vh: 0, life: 0, max: 800, color: "#ffffff", size: 0 });
      for (let i = 0; i < 10; i++) {
        spawned.push({
          kind: "sparkle", x: q.x1 + (Math.random() - 0.5) * 0.5, y: q.y1 + (Math.random() - 0.5) * 0.5, h: 8,
          vx: (Math.random() - 0.5) * 18, vh: 18 + Math.random() * 20, life: 0, max: 900 + Math.random() * 500,
          color: i % 2 ? "#ffe08a" : "#ffffff", size: 2.5 + Math.random() * 2,
        });
      }
    }
    return false;
  });
  state.particles.push(...spawned);

  for (const q of state.particles) {
    if (q.life < 0) continue; // not started yet
    const t = q.life / q.max;
    if (q.kind === "ring" || q.kind === "wave") {
      const r = q.kind === "ring" ? 0.1 + t * 0.5 : 0.3 + t * 3;
      ctx.globalAlpha = (1 - t) * (q.kind === "ring" ? 0.9 : 0.6);
      ctx.strokeStyle = q.color;
      ctx.lineWidth = (q.kind === "ring" ? 1.5 : 3) * z;
      isoEllipse(q.x, q.y, q.h, r, z);
      ctx.stroke();
      continue;
    }
    if (q.kind === "coin") {
      const x = q.x + (q.x1 - q.x) * t, y = q.y + (q.y1 - q.y) * t;
      const { sx, sy } = toScreen(x, y);
      const h = 30 * (1 - t) + Math.sin(t * Math.PI) * 26 + 5 * t;
      ctx.globalAlpha = 1;
      ctx.fillStyle = q.color;
      oval(sx, sy - h * z * FIGURE_SCALE, 3.2 * z * Math.abs(Math.cos(q.life / 60)) + 0.6 * z, 3.2 * z);
      continue;
    }
    q.h += q.vh * s;
    if (q.kind === "petal" && q.h <= 0) { q.h = 0; q.vh = 0; q.vx = 0; }
    if (q.kind === "water") {
      q.vh -= 110 * s; // gravity
      if (q.h <= 0) { q.h = 0; q.vh = 0; }
    }
    if (q.kind === "steam") {
      const { sx, sy } = toScreen(q.x, q.y);
      const r = q.size * z * (1 + t * 2.2);
      ctx.globalAlpha = (1 - t) * 0.32 * Math.min(1, t * 6);
      ctx.fillStyle = q.color;
      oval(sx + q.vx * t * z * 3, sy - q.h * z * FIGURE_SCALE, r, r * 0.8);
      continue;
    }
    const { sx, sy } = toScreen(q.x, q.y);
    ctx.globalAlpha = Math.min(1, t * 4, (1 - t) * 3) * (q.kind === "sparkle" ? 0.85 : 1);
    ctx.fillStyle = q.color;
    const sway = q.kind === "petal" ? Math.sin(q.life / 200) * 6 * z : 0;
    const px = sx + q.vx * t * z * 2 + sway, py = sy - q.h * z * FIGURE_SCALE;
    if (q.kind === "heart") {
      heartPath(px, py, q.size * z);
      ctx.fill();
    } else {
      // Voxel-style: tiny spinning squares (petals spin slower).
      const r = q.size * z * 0.7;
      ctx.save();
      ctx.translate(px, py);
      ctx.rotate(q.life / (q.kind === "petal" ? 500 : 300));
      ctx.fillRect(-r / 2, -r / 2, r, q.kind === "petal" ? r * 0.6 : r);
      ctx.restore();
    }
  }
  ctx.globalAlpha = 1;
}

function label(text, x, y, z, bg, fg) {
  if (state.labelQueue) { state.labelQueue.push([text, x, y, z, bg, fg]); return; }
  ctx.font = `800 ${Math.round(12 * z)}px ${FONT}`;
  const tw = ctx.measureText(text).width;
  const pad = 8 * z, hgt = 21 * z;
  ctx.save();
  ctx.shadowColor = "rgb(80 40 80 / 0.18)";
  ctx.shadowBlur = 6 * z;
  ctx.shadowOffsetY = 2 * z;
  ctx.fillStyle = bg;
  roundRect(x - tw / 2 - pad, y - hgt / 2, tw + pad * 2, hgt, hgt / 2);
  ctx.fill();
  ctx.restore();
  ctx.fillStyle = fg;
  ctx.textAlign = "center";
  ctx.textBaseline = "middle";
  ctx.fillText(text, x, y + 0.5);
}

function roundRect(x, y, w, h, r) {
  ctx.beginPath();
  ctx.roundRect(x, y, w, h, r);
}

function drawBubbles(a, p, now, z) {
  a.bubbles = a.bubbles.filter((b) => now - b.at < BUBBLE_MS);
  if (!a.bubbles.length) return;
  const { sx, sy } = toScreen(p.x, p.y);
  const accent = theme(state.room.theme).accent;
  let y = sy - 21 * VOX * ZPX * z - 10 * z;
  ctx.font = `700 ${Math.round(13 * z)}px ${FONT}`;
  for (let i = a.bubbles.length - 1; i >= 0; i--) {
    const b = a.bubbles[i];
    const age = now - b.at;
    const alpha = age > BUBBLE_MS - 1000 ? (BUBBLE_MS - age) / 1000 : 1;
    const pop = Math.min(1, age / 160); // bubbles pop in
    const text = b.text.length > 60 ? b.text.slice(0, 57) + "…" : b.text;
    const tw = ctx.measureText(text).width;
    const pad = 11 * z, hgt = 28 * z;
    ctx.globalAlpha = alpha;
    ctx.save();
    ctx.translate(sx, y);
    ctx.scale(0.85 + 0.15 * pop, 0.85 + 0.15 * pop);
    ctx.shadowColor = "rgb(90 50 90 / 0.16)";
    ctx.shadowBlur = 10 * z;
    ctx.shadowOffsetY = 3 * z;
    ctx.fillStyle = "#ffffff";
    roundRect(-tw / 2 - pad, -hgt, tw + pad * 2, hgt, hgt / 2);
    ctx.fill();
    if (i === a.bubbles.length - 1) poly([[-5 * z, -1], [5 * z, -1], [0, 6 * z]]); // tail
    ctx.shadowColor = "transparent";
    ctx.strokeStyle = shade(accent, 1.45);
    ctx.lineWidth = 2;
    roundRect(-tw / 2 - pad, -hgt, tw + pad * 2, hgt, hgt / 2);
    ctx.stroke();
    ctx.fillStyle = "#4a3b5c";
    ctx.textAlign = "center";
    ctx.textBaseline = "middle";
    ctx.fillText(text, 0, -hgt / 2);
    ctx.restore();
    ctx.globalAlpha = 1;
    y -= hgt + 6 * z;
  }
}

// ---- pointer input: walking, door peeking, and using props ----

let lastPeek = { key: "", at: 0 };

// propUnder returns the usable prop under the pointer: whatever's drawn
// there (topmost first), or a prop standing on the tile.
function propUnder(px, py, t) {
  const hits = state.hits || [];
  for (let i = hits.length - 1; i >= 0; i--) if (spriteHit(hits[i], px, py)) return hits[i];
  const p = propAt(t.x, t.y);
  return p && p.kind !== "lamp" && p.kind !== "manhole" ? { id: p.id, kind: p.kind } : null;
}

const PROP_TIPS = {
  fountain: (p) => {
    const n = p?.count || 0;
    return ["⛲ Wishing fountain", "Click to toss a coin and make a wish 🪙", n === 1 ? "1 wish made" : `${n} wishes made`];
  },
  bench: () => ["🪑 Bench", "Click to sit down. Walk off to stand up."],
  tree: () => ["🌸 Cherry tree", "Click to give it a shake"],
  bell: () => ["🔔 Hello bell", "Ring it to say hi to everyone here"],
  clock: () => {
    const c = dayCycle();
    return [`🕰️ ${c.time}`, `It's ${c.part} in the plaza.`, clockMode() === "real" ? "Following your clock. Tap the time to speed it up." : "A plaza day lasts 24 minutes. Tap the time to follow your clock."];
  },
  cart: (p) => {
    const look = CART_LOOKS[p?.variant] || CART_LOOKS.hotdog;
    return [`${look.emoji} ${look.name}`, `Click to grab ${look.item}`];
  },
  hydrant: () => ["🚒 Fire hydrant", "Click to open it up. Hot day!"],
  newsbox: () => ["📰 The Habit Daily", "Click to read today's headline"],
  subway: () => ["🚇 Subway", "Click to ride to another room"],
};

function showTip(lines, px, py, r) {
  const tip = $("tooltip");
  tip.replaceChildren();
  lines.forEach((text, i) => {
    const el = document.createElement(i === 0 ? "strong" : "p");
    el.textContent = text;
    tip.append(el);
  });
  tip.style.left = `${Math.min(px + 14, r.width - 270)}px`;
  tip.style.top = `${py + 14}px`;
  tip.hidden = false;
}

canvas.addEventListener("pointermove", (e) => {
  if (!state.room) return;
  const r = canvas.getBoundingClientRect();
  const px = e.clientX - r.left, py = e.clientY - r.top;
  const door = doorUnder(px, py);
  // Hovering a storefront entrance counts as hovering its door tile (peek).
  const t = door ? { x: door.pos[0], y: door.pos[1] } : toTile(px, py);
  state.hover = t;
  const prop = door ? null : propUnder(px, py, t);
  if (prop) {
    canvas.style.cursor = "pointer";
    if (prop.kind === "board") showBoardTip(px, py, r);
    else if (PROP_TIPS[prop.kind]) showTip(PROP_TIPS[prop.kind](propById(prop.id)), px, py, r);
    state.tip = { id: prop.id, kind: prop.kind, px, py, r };
    return;
  }
  state.tip = null;
  canvas.style.cursor = door ? "pointer" : "";
  const d = doorAt(t.x, t.y);
  if (!d) { hideTooltip(); return; }
  const key = `${state.room.id}:${d.target}`;
  const now = performance.now();
  if (key !== lastPeek.key || now - lastPeek.at > 2000) {
    lastPeek = { key, at: now };
    send({ t: "peek", room: d.target });
  }
  const tip = $("tooltip");
  tip.style.left = `${Math.min(px + 14, r.width - 270)}px`;
  tip.style.top = `${py + 14}px`;
});
canvas.addEventListener("pointerleave", () => { state.hover = null; hideTooltip(); });

canvas.addEventListener("click", (e) => {
  if (!state.room) return;
  const r = canvas.getBoundingClientRect();
  const px = e.clientX - r.left, py = e.clientY - r.top;
  const door = doorUnder(px, py);
  if (door) {
    // Clicking a storefront entrance walks you in.
    send({ t: "walk", x: door.pos[0], y: door.pos[1] });
    $("hint").hidden = true;
    return;
  }
  const t = toTile(px, py);
  const prop = propUnder(px, py, t);
  if (prop && prop.kind !== "clock") {
    // The server walks us over (or to a free seat) and uses it on arrival.
    send({ t: "use", prop: prop.id });
    if (prop.kind === "board") openBoard();
    if (prop.kind === "subway") openSubway();
    if (prop.kind === "newsbox") toast(`📰 ${HEADLINES[Math.floor(Math.random() * HEADLINES.length)]}`, 5000);
    $("hint").hidden = true;
    return;
  }
  if (prop) return; // the clock just shows the time on hover
  if (isFloor(t.x, t.y)) {
    send({ t: "walk", x: t.x, y: t.y });
    $("hint").hidden = true;
  }
});

// openSubway shows the subway map: every room, with how many are inside.
function openSubway() {
  const list = $("subwayLines");
  list.replaceChildren();
  state.rooms.forEach((r) => {
    const here = state.room?.id === r.id;
    const b = el("button", "subway-stop");
    b.type = "button";
    b.disabled = here;
    const line = roomLine(r.id);
    const dot = el("span", "line", line.bullet);
    dot.style.background = line.color;
    const n = state.counts[r.id] ?? 0;
    b.append(dot, el("span", "stop", plainName(r.name)), el("span", "meta", here ? "You're here" : `${n} inside`));
    b.onclick = () => {
      $("subwayDialog").close();
      send({ t: "teleport", room: r.id });
      toast(`🚇 Riding to ${r.name}…`);
    };
    const li = el("li");
    li.append(b);
    list.append(li);
  });
  if (!$("subwayDialog").open) $("subwayDialog").showModal();
}
$("subwayClose").onclick = () => $("subwayDialog").close();

// The chat panel can be tucked away to see more of the world.
function setChatHidden(hidden) {
  document.body.classList.toggle("chat-hidden", hidden);
  $("chatShow").hidden = !hidden;
  store.set("habit.chatHidden", hidden ? "1" : "");
}
$("chatHide").onclick = () => setChatHidden(true);
$("chatShow").onclick = () => { setChatHidden(false); $("say").focus(); };
if (store.get("habit.chatHidden")) setChatHidden(true);

// The time chip switches the plaza between a fast shared day and your clock.
$("timeChip").onclick = () => {
  const real = clockMode() !== "real";
  store.set("habit.clock", real ? "real" : "fast");
  layerCache.clear(); // the skyline's lights depend on the hour
  toast(real ? "🕰️ The plaza now follows your clock" : "⏩ Back to plaza time: a day every 24 minutes");
};

// showBoardTip previews the noticeboard: its latest posts.
function showBoardTip(px, py, r) {
  const threads = state.board?.room === state.room.id ? state.board.threads : [];
  const lines = [`📌 ${state.room.name} board`];
  if (!threads.length) lines.push("No posts yet. Click to write the first one!");
  for (const th of threads.slice(0, 3)) lines.push(`${th.title} · ${th.replies} ${th.replies === 1 ? "reply" : "replies"}`);
  if (threads.length) lines.push("Click to read and post.");
  showTip(lines, px, py, r);
}

function showPeek(m) {
  if (!state.hover) return;
  const d = doorAt(state.hover.x, state.hover.y);
  if (!d || d.target !== m.room) return;
  const tip = $("tooltip");
  tip.replaceChildren();
  const title = document.createElement("strong");
  title.textContent = `${m.name} · ${m.count} inside`;
  tip.append(title);
  const recent = m.recent || [];
  if (!recent.length) {
    const p = document.createElement("p");
    p.textContent = "Quiet right now.";
    tip.append(p);
  }
  for (const msg of recent) {
    const p = document.createElement("p");
    p.textContent = `${msg.name}: ${msg.text}`;
    tip.append(p);
  }
  tip.hidden = false;
}

function hideTooltip() { $("tooltip").hidden = true; }

// ---- character creator (on signup) ----
//
// Starts from the look you had as a guest; pick a skin tone, hair, outfit
// and accessory, and watch your character turn in the preview.

const CREATOR_PARTS = [["skin", SKIN, "Skin tone"], ["hair", HAIR, "Hair colour"], ["outfit", OUTFITS, "Outfit colour"]];

function myStyle() {
  const me = state.avatars.get(state.you);
  return { skin: 0, hair: 0, outfit: 0, accessory: 3, ...(me?.style || {}) };
}

function renderCreator() {
  for (const [part, list, name] of CREATOR_PARTS) {
    const row = document.querySelector(`.swatches[data-part="${part}"]`);
    row.replaceChildren();
    list.forEach((color, i) => {
      const b = document.createElement("button");
      b.type = "button";
      b.className = "swatch";
      b.style.background = color;
      b.setAttribute("role", "radio");
      b.setAttribute("aria-checked", String(state.draftStyle[part] === i));
      b.setAttribute("aria-label", `${name} ${i + 1}`);
      b.onclick = () => { state.draftStyle[part] = i; renderCreator(); };
      row.append(b);
    });
  }
  const chips = document.querySelector('.chips[data-part="accessory"]');
  chips.replaceChildren();
  ACCESSORY_NAMES.forEach((name, i) => {
    const b = document.createElement("button");
    b.type = "button";
    b.className = "chip";
    b.textContent = name;
    b.setAttribute("aria-pressed", String(state.draftStyle.accessory === i));
    b.onclick = () => { state.draftStyle.accessory = i; renderCreator(); };
    chips.append(b);
  });
}

let creatorFrame = 0;
function drawCreator(now) {
  creatorFrame = 0;
  if (!$("authDialog").open || authMode !== "signup") return;
  const c = $("creatorPreview");
  const dpr = window.devicePixelRatio || 1;
  const w = c.clientWidth, h = c.clientHeight;
  if (c.width !== Math.round(w * dpr)) { c.width = Math.round(w * dpr); c.height = Math.round(h * dpr); }
  const g = c.getContext("2d");
  g.setTransform(dpr, 0, 0, dpr, 0, 0);
  g.clearRect(0, 0, w, h);
  g.fillStyle = "rgb(90 50 90 / 0.15)";
  g.beginPath();
  g.ellipse(w / 2, h - 16, 26, 10, 0, 0, Math.PI * 2);
  g.fill();
  const saved = ctx;
  ctx = g;
  try {
    drawModelAt(characterModel(state.draftStyle), w / 2, h - 16, now / 1300, VOX, 1.9, { head: { dk: Math.sin(now / 520) * 0.2 } }, now % 4200 < 140);
  } finally {
    ctx = saved;
  }
  creatorFrame = requestAnimationFrame(drawCreator);
}

function startCreator() {
  if (!state.draftStyle) state.draftStyle = myStyle();
  renderCreator();
  if (!creatorFrame) creatorFrame = requestAnimationFrame(drawCreator);
}


connect();
requestAnimationFrame(draw);
