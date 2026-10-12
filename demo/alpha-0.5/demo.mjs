// Habit demo, alpha 0.5: a scripted tour of Habit for recording a video.
//
// It builds Habit and runs its own private copy (in memory, on its own
// port, so it never touches your real database), opens Chrome full screen,
// and plays a tour: you walk through the Plaza, make a wish, chat, then
// visit the Gaming arcade and the Gym. A handful of demo characters (plain
// WebSocket clients, no extra windows) walk around, chat and use things
// alongside you, and captions and title cards explain what's going on.
//
// Nothing here changes Habit itself: everything goes through the same
// messages the real client sends.
//
//   node demo.mjs            wait 10 seconds (to start recording), then play
//   node demo.mjs --now      play straight away
//   node demo.mjs --wait 20  wait 20 seconds first
//   node demo.mjs --windowed play in a normal window instead of full screen
//   node demo.mjs --check    play invisibly, saving a screenshot per caption
//   node demo.mjs --hour 21  start at plaza night instead of the afternoon

import { chromium } from "playwright-core";
import { spawn, execFileSync } from "node:child_process";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { mkdirSync, rmSync } from "node:fs";

const HERE = dirname(fileURLToPath(import.meta.url));
const ROOT = resolve(HERE, "../..");
const PORT = Number(process.env.DEMO_PORT || 8090);
const BASE = `http://localhost:${PORT}`;
const PASSWORD = "demo-password-123";

const args = process.argv.slice(2);
const flag = (name) => args.includes(name);
const waitAt = args.indexOf("--wait");
// --check plays the tour in an invisible browser and saves a screenshot at
// each caption into .bin/shots, to check the tour still works.
const checking = flag("--check");
const waitSecs = flag("--now") || checking ? 0 : waitAt >= 0 ? Number(args[waitAt + 1]) || 0 : 10;
// --hour sets the Plaza's time of day when the tour starts (0–24; 14 is a
// sunny afternoon, 19 sunset, 21 night with the skyline lit up).
const hourAt = args.indexOf("--hour");
const startHour = hourAt >= 0 ? Number(args[hourAt + 1]) % 24 || 0 : 14;

const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const later = (ms, fn) => sleep(ms).then(fn).catch((e) => console.warn("  (demo step skipped:", e.message + ")"));

// ---- the private Habit server ----

let server = null;

function startServer() {
  const bin = join(HERE, ".bin", process.platform === "win32" ? "habit-demo.exe" : "habit-demo");
  mkdirSync(dirname(bin), { recursive: true });
  console.log("Building Habit...");
  execFileSync("go", ["build", "-o", bin, "./cmd/habit"], { cwd: ROOT, stdio: "inherit" });
  console.log(`Starting a private demo copy of Habit at ${BASE} (in memory, nothing is saved)...`);
  server = spawn(bin, ["-addr", `:${PORT}`], { cwd: ROOT, env: { ...process.env, DATABASE_URL: "" }, stdio: "ignore" });
  server.on("exit", (code) => { if (code) console.error(`The demo server stopped (exit ${code}). Is port ${PORT} already in use?`); });
}

async function waitForServer() {
  for (let i = 0; i < 100; i++) {
    try {
      if ((await fetch(BASE)).ok) return;
    } catch {}
    await sleep(200);
  }
  throw new Error(`Habit didn't start on ${BASE}`);
}

// ---- demo characters ----

// A Bot is a demo character: a WebSocket client that talks to Habit the
// same way the browser does.
class Bot {
  constructor(name, style) {
    this.name = name;
    this.style = style;
    this.waiters = [];
  }

  async connect() {
    this.ws = new WebSocket(`ws://localhost:${PORT}/ws`);
    this.ws.onmessage = (e) => {
      const m = JSON.parse(e.data);
      this.waiters = this.waiters.filter((w) => !(w.t === m.t && (!w.match || w.match(m)) && (w.resolve(m), true)));
    };
    await new Promise((ok, fail) => { this.ws.onopen = ok; this.ws.onerror = fail; });
    await this.waitFor("room");
    this.send({ t: "signup", username: this.name, password: PASSWORD, birthYear: 1996, style: this.style });
    await this.waitFor("authed");
    this.send({ t: "named", on: true }); // show their names rather than "Anonymous #…"
    return this;
  }

  waitFor(t, ms = 6000, match = null) {
    return new Promise((resolve) => {
      const w = { t, resolve, match };
      this.waiters.push(w);
      setTimeout(() => resolve(null), ms);
    });
  }

  send(m) { if (this.ws.readyState === WebSocket.OPEN) this.ws.send(JSON.stringify(m)); }
  say(text) { this.send({ t: "chat", text }); }
  walk(x, y) { this.send({ t: "walk", x, y }); }
  use(prop) { this.send({ t: "use", prop }); }
  emote(emote) { this.send({ t: "emote", emote }); }
  async go(room) { this.send({ t: "teleport", room }); await this.waitFor("room"); }
  close() { try { this.ws.close(); } catch {} }

  // The forum: every room has a board of threads. Posts show your name,
  // or an anonymous tag when anon is set; post returns the thread's id.
  async post(title, text, anon = false) {
    if (anon) this.send({ t: "named", on: false });
    this.send({ t: "post_thread", title, text });
    const m = await this.waitFor("posted_thread");
    if (anon) this.send({ t: "named", on: true });
    return m?.thread.id;
  }
  async reply(thread, text, anon = false) {
    if (anon) this.send({ t: "named", on: false });
    this.send({ t: "post_reply", thread, text });
    await sleep(150);
    if (anon) this.send({ t: "named", on: true });
  }
  vote(thread, value) { this.send({ t: "vote", thread, value }); }
  // threadTitled waits for a new thread with this title to go up on the
  // board of the room they're in, and returns its id.
  async threadTitled(title, ms = 30000) {
    const m = await this.waitFor("board_update", ms, (m) => m.thread.title === title);
    return m?.thread.id;
  }
}

// seedForum fills the Plaza's board before the tour starts: a mix of named
// and anonymous threads, replies (with the OP joining in) and votes, so the
// sorts have something to sort.
async function seedForum({ mia, leo, zoe, sam, kai }) {
  const pineapple = await kai.post("Pineapple on pizza: yes or no? 🍍", "Settle this once and for all. Upvote for yes, downvote for no.", true);
  const food = await leo.post("What's the most underrated food?", "Mine's cold pizza for breakfast. Fight me.");
  const squad = await zoe.post("Looking for a squad tonight 🎮", "Anyone up for co-op later? I'll be in the arcade around 8.", true);
  const wins = await sam.post("Post your gym wins 💪", "Hit a 100kg squat today after 6 months of trying. Small wins count!");
  const night = await mia.post("Hot take: the Plaza at night is the best place on the internet", "The skyline lights up, the fountain glows and everyone's out on the benches. Change my mind.", true);

  await leo.reply(night, "agreed, the neon signs at dusk are unreal 🤌", true);
  await zoe.reply(night, "the pigeons are the real stars here tbh 🐦", true);
  await mia.reply(night, "see, people get it ✨", true);
  await kai.reply(night, "ok I'm convinced, see everyone at 9", true);
  await kai.reply(wins, "HUGE. congrats!!");
  await mia.reply(wins, "goals 🙌", true);
  await sam.reply(pineapple, "absolutely not.", true);
  await zoe.reply(pineapple, "yes, and I will die on this hill", true);
  await mia.reply(food, "a really good plain bagel", true);

  for (const b of [leo, zoe, sam, kai]) b.vote(night, 1);
  for (const b of [mia, leo, kai]) b.vote(wins, 1);
  for (const b of [mia, kai]) b.vote(squad, 1);
  for (const b of [zoe, sam]) b.vote(food, 1);
  for (const [b, v] of [[mia, 1], [zoe, 1], [sam, -1], [leo, -1]]) b.vote(pineapple, v);
  await sleep(500);
}

// Styles index the client's palettes: skin 0–5, hair 0–7, outfit 0–7,
// accessory 0–3 (cat ears, bow, curl, none).
const CAST = {
  mia: { skin: 1, hair: 4, outfit: 5, accessory: 1 },
  leo: { skin: 4, hair: 0, outfit: 0, accessory: 3 },
  zoe: { skin: 2, hair: 6, outfit: 6, accessory: 0 },
  sam: { skin: 3, hair: 2, outfit: 2, accessory: 2 },
  kai: { skin: 5, hair: 7, outfit: 3, accessory: 3 },
};

// ---- the browser: you ----

const OVERLAY_CSS = `
  #quest, #hint, #today { display: none !important; }
  /* The caption and cards are popovers, so they can sit above the board's
     dialogs (which live in the browser's top layer). */
  #demo-caption, #demo-card { margin: 0; border: 0; overflow: visible; }
  #demo-caption {
    position: fixed; z-index: 50; inset: auto; left: calc((100vw - var(--inset-r, 0px)) / 2); bottom: 104px;
    transform: translate(-50%, 12px); opacity: 0; transition: opacity .6s ease, transform .6s ease;
    max-width: min(78vw, 900px); padding: 14px 30px; border-radius: 999px; pointer-events: none;
    background: rgb(34 22 58 / 0.86); color: #fff; text-align: center;
    font: 800 26px/1.3 "Nunito", "Segoe UI", system-ui, sans-serif; box-shadow: 0 10px 30px rgb(30 10 60 / .3);
  }
  #demo-caption.on { opacity: 1; transform: translate(-50%, 0); }
  #demo-caption.top { top: 14px; bottom: auto; left: 50%; max-width: 96vw; white-space: nowrap; padding: 10px 26px; font-size: 22px; }
  #demo-card {
    position: fixed; inset: 0; width: 100vw; height: 100vh; max-width: none; max-height: none; z-index: 60; display: grid; place-content: center; gap: 18px; text-align: center;
    background: radial-gradient(circle at 30% 20%, #ff8fc8, #a77bff 55%, #5b4bd6); color: #fff;
    font-family: "Nunito", "Segoe UI", system-ui, sans-serif; opacity: 0; transition: opacity 1s ease; pointer-events: none;
  }
  #demo-card.on { opacity: 1; }
  #demo-card h1 { margin: 0; font-size: 120px; font-weight: 900; letter-spacing: -2px; text-shadow: 0 8px 30px rgb(60 20 120 / .35); }
  #demo-card .tag { justify-self: center; padding: 6px 18px; border-radius: 999px; background: rgb(255 255 255 / .22); font: 800 22px/1.4 inherit; }
  #demo-card p { margin: 0; font-size: 34px; font-weight: 800; }
  #demo-card small { font-size: 20px; font-weight: 700; opacity: .9; }
`;

async function openBrowser() {
  const browser = await chromium.launch({
    channel: "chrome",
    headless: checking,
    args: [flag("--windowed") || checking ? "--window-size=1600,900" : "--start-fullscreen", "--hide-crash-restore-bubble", "--no-first-run"],
    ignoreDefaultArgs: ["--enable-automation"], // no "controlled by automated software" bar
  });
  const context = await browser.newContext({ viewport: checking ? { width: 1600, height: 900 } : null });
  const page = await context.newPage();
  await page.goto(BASE);
  await page.waitForFunction(() => typeof state !== "undefined" && state.room && state.you);
  await page.addStyleTag({ content: OVERLAY_CSS });
  await page.evaluate((hour) => {
    const caption = document.createElement("div");
    caption.id = "demo-caption";
    const card = document.createElement("div");
    card.id = "demo-card";
    for (const el of [caption, card]) el.popover = "manual";
    document.body.append(caption, card);
    // raise brings the overlays back on top, after a dialog opens.
    window.demoRaise = () => {
      for (const el of [caption, card]) {
        if (el.matches(":popover-open")) el.hidePopover();
        el.showPopover();
      }
    };
    demoRaise();
    // Pin the Plaza's clock (a 24-minute day) to start at the chosen hour,
    // so the tour isn't in the dark if it happens to be plaza night.
    const t0 = Date.now(), realCycle = dayCycle;
    dayCycle = () => realCycle(DAY_MS * (hour / 24) + (Date.now() - t0));
    // Signing up shows a recovery code after your first message; the demo
    // closes it.
    setInterval(() => { if ($("recoveryDialog").open) $("savedCode").click(); }, 200);
  }, startHour);
  return { browser, page };
}

function you(page) {
  const act = (m) => page.evaluate((m) => send(m), m);
  let shot = 0;
  const snap = (delay) => {
    if (!checking) return;
    const file = join(HERE, ".bin", "shots", `${String(++shot).padStart(2, "0")}.png`);
    later(delay, () => page.screenshot({ path: file }));
  };
  // idle waits until your avatar has finished walking.
  const idle = async (extra = 300) => {
    await sleep(450);
    await page.waitForFunction(() => { const a = state.avatars.get(state.you); return a && !a.path.length; }, null, { timeout: 20000 }).catch(() => {});
    await sleep(extra);
  };
  return {
    act,
    idle,
    async signup(name, style) {
      await act({ t: "signup", username: name, password: PASSWORD, birthYear: 1997, style });
      await page.waitForFunction(() => state.account, null, { timeout: 10000 });
      await act({ t: "named", on: true });
    },
    async walk(x, y) { await act({ t: "walk", x, y }); await idle(); },
    async use(prop, after = 900) { await act({ t: "use", prop }); await idle(after); },
    async say(text) { await act({ t: "chat", text }); await sleep(1200); },
    async emote(emote) { await act({ t: "emote", emote }); await sleep(900); },
    async go(room) {
      await act({ t: "teleport", room });
      await page.waitForFunction((r) => state.room?.id === r, room, { timeout: 10000 });
      await sleep(600);
    },
    // caption shows a line of narration at the bottom, or at the top (top)
    // while the board is open so it doesn't cover it.
    async caption(text, { top = false } = {}) {
      await page.evaluate(([t, top]) => {
        const c = document.getElementById("demo-caption");
        if (!t) { c.classList.remove("on"); return; }
        c.classList.remove("on");
        setTimeout(() => { c.textContent = t; c.classList.toggle("top", top); c.classList.add("on"); }, c.textContent ? 350 : 0);
      }, [text, top]);
      if (text) snap(2500);
    },

    // ---- the forum, driven through the real board UI ----
    async openBoard() {
      await page.evaluate(() => { openBoard(); demoRaise(); });
      await page.locator("#threads .thread-open").first().waitFor({ timeout: 10000 });
      await sleep(1800);
    },
    async sortBy(sort) {
      await page.click(`.sorts [data-sort="${sort}"]`);
      await sleep(1800);
    },
    async upvoteTop() {
      await page.locator("#threads li").first().locator("button.vote").first().click();
      await sleep(1200);
    },
    async openTopThread() {
      await page.locator("#threads .thread-open").first().click();
      await page.locator("#threadView").waitFor({ state: "visible", timeout: 10000 });
      await sleep(2500);
    },
    async reply(text) {
      await page.locator("#replyText").click();
      await page.locator("#replyText").pressSequentially(text, { delay: 55 });
      await sleep(400);
      await page.evaluate(() => $("replyForm").requestSubmit());
      await sleep(1800);
    },
    async backToBoard() {
      await page.click("#backToBoard");
      await sleep(1200);
    },
    // newThread posts anonymously: your name stays hidden on the board.
    async newThread(title, text) {
      await act({ t: "named", on: false });
      await page.click("#newThreadBtn");
      await page.evaluate(() => demoRaise());
      await sleep(600);
      await page.locator('#threadForm [name="title"]').pressSequentially(title, { delay: 45 });
      await sleep(300);
      await page.locator('#threadForm [name="text"]').pressSequentially(text, { delay: 30 });
      await sleep(500);
      await page.click("#threadSubmit");
      await page.waitForFunction((t) => state.thread?.thread.title === t, title, { timeout: 10000 });
      await act({ t: "named", on: true });
      await sleep(1000);
    },
    async closeBoard() {
      await page.click("#boardClose");
      await sleep(800);
    },
    async showNotifications(ms = 3500) {
      await page.click("#notesBtn");
      await page.evaluate(() => demoRaise());
      await sleep(ms);
      await page.click("#notesClose");
      await sleep(600);
    },
    async card(html) {
      await page.evaluate((h) => {
        const c = document.getElementById("demo-card");
        if (h) c.innerHTML = h;
        c.classList.toggle("on", !!h);
      }, html);
      if (html) snap(1500);
    },
  };
}

// ---- the tour ----

const TITLE = `<h1>Habit</h1><span class="tag">alpha 0.5</span><p>Walk in. Say hi.</p>`;
const END = `<h1>Habit</h1><span class="tag">alpha 0.5</span><p>Walk in. Say hi.</p>
  <small>Live rooms + forums that stay · Anonymous by default · No email or phone</small>
  <small>Built with Go, Postgres and WebSockets</small>`;

async function tour(me, bots) {
  const { mia, leo, zoe, sam, kai } = bots;

  // While the title is up, everyone gets into place round the fountain.
  later(0, () => mia.use("bench-s"));
  later(300, () => leo.walk(19, 22));
  later(600, () => zoe.walk(12, 19));
  later(900, () => sam.use("cart-hotdog"));
  later(1200, () => kai.walk(19, 26));
  await sleep(5500);
  await me.card(null);
  await sleep(1200);

  // The Plaza.
  await me.caption("A cosy virtual world where every chatroom is a real place.");
  later(1800, () => sam.walk(13, 25));
  await me.walk(15, 24);
  later(0, () => mia.emote("wave"));
  later(700, () => leo.say("welcome to the plaza! ✨"));
  await sleep(1800);
  await me.caption("See who's around before you say a word.");
  await me.emote("wave");
  await sleep(1500);

  await me.caption("Make a wish: everyone fills the fountain together.");
  later(0, () => zoe.use("fountain"));
  later(1600, () => leo.use("fountain"));
  await me.use("fountain", 1600);
  later(0, () => kai.use("fountain"));
  await sleep(1200);

  await me.caption("Chat out loud, or just hang out.");
  await me.say("hi everyone! first time here 😊");
  later(400, () => mia.say("omg hi!! welcome 💕"));
  later(1400, () => leo.emote("dance"));
  later(2600, () => sam.say("the hot dogs here are elite 🌭"));
  later(4200, () => kai.say("you have to try the arcade"));
  await sleep(5200);

  // The forum: the Plaza's board.
  await me.caption("Every room has a forum too: threads that stay, with upvotes and downvotes.");
  await me.use("board", 600);
  await me.openBoard();
  await me.caption("Sort it like Reddit: Latest, New, Hot, Top.", { top: true });
  await me.sortBy("hot");
  await me.sortBy("top");
  await me.upvoteTop();
  await me.caption("Anonymous by default, like an imageboard. OP marks the original poster.", { top: true });
  await me.openTopThread();
  await sleep(1500);
  await me.reply("the fountain at night is unreal 🌙");
  await me.backToBoard();

  await me.caption("Start your own thread. No email, no phone, no real name.", { top: true });
  const MY_POST = "First day here. What should I check out?";
  // The others spot the new post and answer it, live.
  later(0, async () => { const id = await mia.threadTitled(MY_POST); await sleep(2200); await mia.reply(id, "welcome!! the arcade is a must 🕹️", true); });
  later(0, async () => { const id = await leo.threadTitled(MY_POST); await sleep(4200); await leo.reply(id, "hit the punching bag in the Gym, trust me 🥊"); });
  later(0, async () => { const id = await zoe.threadTitled(MY_POST); await sleep(6200); await zoe.reply(id, "come find us in Gaming tonight 🎮", true); });
  await me.newThread(MY_POST, "Just walked in from the Plaza. Where's everyone hanging out?");
  await me.caption("Replies arrive live, and you get a 🔔 when someone answers.", { top: true });
  await sleep(7500);
  await me.closeBoard();
  await me.showNotifications();

  await me.caption("Shake a tree, grab a snack, sit on a bench.");
  await me.use("tree-s", 1400);
  // Meanwhile, some of the gang head off to the arcade.
  later(0, () => leo.go("gaming").then(() => leo.use("couch")));
  later(300, () => zoe.go("gaming").then(() => zoe.use("cabinet-4")));
  later(600, () => kai.go("gaming").then(() => kai.use("beanbag-b")));
  await me.say("ok, arcade time 🎮");
  await sleep(1500);

  // Gaming.
  await me.caption("Step into communities, each with a space of its own.");
  await me.go("gaming");
  await sleep(1200);
  await me.caption("Gaming is a neon arcade, with real cabinets to play.");
  later(800, () => zoe.use("cabinet-4"));
  await me.use("cabinet-1", 1800);
  later(0, () => zoe.say("gg!! 🎮"));
  later(1200, () => leo.emote("laugh"));
  await me.say("this arcade is adorable");
  later(0, () => mia.go("gaming").then(() => mia.use("vending")));
  await me.use("beanbag-a", 1200);
  later(0, () => kai.say("the pong screen never stops lol"));
  await sleep(3200);
  later(0, () => mia.say("anyone want a soda? 🥤"));
  await sleep(2200);

  // Meanwhile, the Gym fills up.
  later(0, () => sam.go("gym").then(() => sam.use("punchbag")));
  later(400, () => mia.go("gym").then(() => mia.use("press")));
  later(800, () => kai.go("gym").then(() => kai.use("weights")));

  // The Gym.
  await me.caption("…or the Gym: punching bags, weights and pep talks.");
  await me.go("gym");
  await sleep(1400);
  later(0, () => sam.say("leg day 💪"));
  await me.use("punchbag", 1200);
  later(0, () => sam.use("punchbag"));
  await me.use("punchbag", 1200);
  later(0, () => kai.use("weights"));
  await me.use("cooler", 1000);
  await me.say("ok I live here now 💧");
  later(300, () => mia.emote("heart"));
  await sleep(2500);

  await me.caption("Anonymous by default. No email, no phone number. Just walk in.");
  await sleep(4500);
  await me.caption(null);
  await me.card(END);
  await sleep(7000);
}

// ---- run it ----

async function main() {
  if (checking) {
    rmSync(join(HERE, ".bin", "shots"), { recursive: true, force: true });
    mkdirSync(join(HERE, ".bin", "shots"), { recursive: true });
  }
  startServer();
  await waitForServer();

  console.log("Bringing in the demo characters...");
  const bots = {};
  for (const [name, style] of Object.entries(CAST)) bots[name] = await new Bot(name, style).connect();
  console.log("Filling the Plaza's forum...");
  await seedForum(bots);

  console.log("Opening Chrome...");
  const { browser, page } = await openBrowser();
  const me = you(page);
  await me.card(TITLE);
  await me.signup("alex", { skin: 0, hair: 3, outfit: 4, accessory: 0 });
  await page.bringToFront();

  if (waitSecs > 0) {
    console.log(`\nStart recording now (Win + Alt + R, or OBS). The tour starts in ${waitSecs} seconds...`);
    for (let s = waitSecs; s > 0; s--) { process.stdout.write(`  ${s}... `); await sleep(1000); }
    console.log();
  }
  console.log("Playing the tour (about two minutes)...");
  try {
    await tour(me, bots);
    console.log("Done! Stop your recording now.");
    await sleep(1500);
  } finally {
    for (const b of Object.values(bots)) b.close();
    await browser.close().catch(() => {});
  }
}

function shutdown() {
  if (server && !server.killed) server.kill();
}
process.on("exit", shutdown);
process.on("SIGINT", () => { shutdown(); process.exit(130); });

main().catch((e) => {
  console.error("\nThe demo stopped:", e.message);
  if (/Chromium distribution 'chrome' is not found/i.test(e.message)) console.error("It needs Google Chrome installed: https://www.google.com/chrome/");
  process.exitCode = 1;
}).finally(shutdown);
