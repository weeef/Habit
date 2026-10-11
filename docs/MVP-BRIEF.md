# Habit: MVP brief (handoff for developers and coding agents)

Read this first. The full product plan is in `PLAN.md`; this is the short
version plus the current state of the code.

## 1. What Habit is

A cute, Habbo-style world where **every chatroom is a real room you walk
into**. It mixes Discord-style live chat, Reddit/imageboard-style communities
and anonymous posting. Tagline: **"Walk in. Say hi."**

The whole bet: **joining a conversation should feel easier and less
intimidating than joining a Discord server.** You can look around without
an account, peek through a door to see who's inside, hang out without
talking, and leave without anyone being told.

## 2. Product decisions already made (don't reopen these)

| Topic | Decision |
|---|---|
| Platform order | **Website first** (desktop + mobile browser). Mobile app later (Phase 3), starting as a Capacitor wrapper of the web client. |
| Backend | **Go** (1.26+). WebSockets via `github.com/coder/websocket`. Postgres 18 via `github.com/jackc/pgx/v5`. Password hashing via `golang.org/x/crypto/argon2`. |
| Accounts | **Username + password only. Never ask for email or phone.** Lost password is handled by a one-time **recovery code** shown after the user's first message. |
| Anonymity | **Anonymous by default.** Users show as `Anonymous #3f9a`; the tag is an HMAC of (user, room, day), so it's stable within a room for a day and unlinkable across rooms. Showing your username is opt-in. Avatar IDs sent to clients are fresh on every room entry, so nobody can follow someone between rooms. Mods act on anon IDs; the server resolves the account. |
| Ages | **13+.** Under-13 signups are refused. Teens (13–17) get protections on by default: a personal-info filter (phone numbers, emails, addresses, school, "add me on snap"), and later grooming-risk detection and stricter DMs. Age bands: `teen_13_15`, `teen_16_17`, `adult`. When unsure, round to the younger band. |
| Speech | Any **topic** can be discussed everywhere. The main world has no explicit sexual content or gore. An age-checked 18+ **"After Dark"** area (Phase 2, not MVP) allows anything legal. Hard lines everywhere: illegal content, CSAM, credible threats, doxxing, organised harassment/raids, spam/scams. Principle: "moderate behaviour, not opinions." |
| World structure | The **Plaza** (where you arrive) has doors to **community** rooms. Right now: **Gaming** and **Gym**. Later, each community gets its own chatrooms behind further doors. |
| The Plaza | A **New York–style square for up to 500 people** (32×32; the camera follows you; arrivals spread out over free tiles). Client-drawn **skyline** on the back edges (glass towers, Art Deco spires, brick walk-ups with fire escapes and water towers, brownstones, animated **neon signs**; never "I ♥ NY", a trademark). **Doors are storefronts in that row**: Gaming is an **arcade** (purple, "ARCADE" marquee, game screens, a pixel invader), the Gym is **HABIT GYM** (concrete, red stripe, dumbbells in the window); click the glowing entrance to walk in. A **street** out front with car-scale taxis and a bus. In the square: a **wishing fountain**, the **noticeboard kiosk** beside it, benches, cherry trees in planters, lamps, a sidewalk clock, a hello bell (4 s cooldown), **food carts** (hot dog, pretzel, halal, coffee: the snack is carried in your hand for 30 s), hydrants, newspaper boxes, two **subway entrances** (ride to any room), steaming manholes (walk over them) and pigeons. **Day/night**: a shared 24-minute day, or (tap the time) your own local clock. Props are server-side (`world.Prop`). Rendering: the skyline and floor are cached layers; characters are cached per look, direction and pose (an LRU of 2,500 sprites) and anything off screen is skipped, so ~255 people draw in ~7 ms. Interactions are small, shared, social moments, never a game with scores. |
| Interface | The world fills the screen; the HUD floats over it in **NYC transit-sign style**: black enamel signs with a thin white stripe, white rounded type, room "line bullets", taxi-yellow buttons. Top left, the **station sign** (where you are, how many are here; opens the subway map). Chat is an enamel panel on the right (hideable), a bottom sheet on phones. Dialogs (sign up, board) are light/dark "paper". |
| Boards | **Every room has a postboard** (like 4chan/Reddit, but text only): threads with replies, **kept forever**. It opens in a **window in the middle of the screen** when you click the room's noticeboard (there's no side-panel board). **Filters**: Latest activity (a reply bumps a thread), New, Hot (votes + replies, weighted to recent), Top, Most downvoted. Accounts can **👍/👎 vote** once per thread (change or take back any time). Threads lock after 6 months or 500 replies but stay readable. Live chat is the fast layer; the board is what stays. Guests can read; posting needs an account. Posts are anonymous by default with a tag stable **within a thread** (HMAC of account + thread) and an "OP" mark. People get **notifications** when someone replies in a thread they started or replied in. **No image uploads** until CSAM scanning (PhotoDNA/NCMEC reporting) and nudity filtering are in place. |
| Art style | **Cute voxels** (PLAN.md §2.4): chibi voxel characters (cube head, eyes, blush, accessory: cat ears, bow or curl) that **turn smoothly** to face where they walk and look at whoever's talking; block floors with voxel flowers/stars/leaves; voxel door arches with glowing portals; a voxel **noticeboard** in every room that opens its board; floating sparkles and hearts; rounded Nunito font (bundled, OFL), pill buttons, light and dark mode. Labels only on you and the hovered avatar. Keep it original. |
| Creator content | Communities will upload their own cosmetics and sprites **after** the safety tools, through the pipeline in PLAN.md §2.5 (small PNGs on Habit's templates, CSAM hash scanning, human review before publishing, DMCA process, Creator Agreement). Adult community owners only at first; no selling at launch. |
| Monetization | Not in the MVP. Later: cosmetics, Habit+ subscription, and the **Penthouse** premium tier (~$999/yr) with a verified marketplace (Phase 4). No ads, ever. Nothing purchasable affects ranking or moderation. |

## 3. MVP scope (12 weeks)

The MVP must prove one thing: **a first-time visitor can go from a shared link
to their first message in under 60 seconds, and it feels less intimidating
than Discord.**

**In the MVP**
- The Plaza + 5–10 seeded community rooms (currently 2: Gaming, Gym), doors between them.
- Click-to-walk (server-authoritative), emotes (wave, dance, laugh, heart, sit).
- Text chat: speech bubbles over avatars + a side chat log, recent history on entering, quick-start prompts.
- Room boards: saved text threads and replies per room, reply notifications (see §2 Boards).
- Approachability: guest mode (look, walk, wave; no talking), door peeking (live head-count + last few messages), quiet entry/exit, "new" badge, room navigator with live counts and teleport, shareable room links.
- Accounts: one-step username + password + birth-year signup **without leaving the room**, with a simple character creator (skin, hair, outfit, accessory; starts from your guest look); recovery code; change password; delete account.
- Safety: report, block, mute; room mods can kick, timeout, ban; word filter + toxicity scoring; rate limits; new-account slow mode; teen personal-info filter; staff admin panel; audit log.
- List view (plain chat, no world rendering) and a phone-friendly layout.
- Funnel analytics: link → in world → entered a room → signed up → first message.

**Not in the MVP:** image uploads, After Dark,
voting, voice, DMs, friends, personal rooms, room building, items/currency,
quests, mini-games, trading, Penthouse/marketplace, native mobile apps.

**Success targets:** ≥25% of guests sign up to chat; ≥70% of signups send a
message within 5 minutes; median time to first message under 60 seconds; ≥25%
of chatters come back after 7 days.

## 4. What's built so far (week 0–2 prototype)

Run with `go run ./cmd/habit`, then open http://localhost:8080. Add
`-db <postgres url>` (or set `DATABASE_URL`) to save data; without it,
everything is in memory. Run the tests with `go test ./...`; set
`HABIT_TEST_DATABASE_URL` to a throwaway database to include the Postgres
tests.

```
cmd/habit/main.go          server binary; flags -addr (default :8080), -db (default $DATABASE_URL)
internal/store/            persistence behind the Store interface
  store.go                 Store, Account, Message, Thread, Reply, Notification;
                           ErrNotFound, ErrUsernameTaken
  memory.go                in-memory Store (default when no -db; used in tests)
  postgres/                Postgres Store (pgx); schema.sql is applied on every start
  storetest/               shared test suite every Store must pass
internal/world/            pure, no runtime state
  world.go                 Point (JSON [x,y]), Room, Door, Kind, Theme, Validate()
  rooms.go                 DefaultRooms(): plaza, gaming, gym
  path.go                  A* pathfinding; door tiles only allowed as the final step
internal/server/           realtime gateway
  server.go                Server: one mutex guards all state; tick loop moves avatars;
                           door transfers; chat; counts; peek; teleport; enter/leave.
                           Loads recent chat per room on start; saves each message
                           after releasing the lock
  accounts.go              username+password accounts (argon2id) on top of the Store,
                           session tokens (stored as SHA-256 hashes), recovery codes,
                           age bands, reserved names
  boards.go                room boards: list/open threads, post threads and replies,
                           thread locking, reply notifications (live + saved)
  props.go                 using props (walk over, then act on arrival), fountain wishes,
                           tree shakes, the bell (with a cooldown), benches (sit/stand),
                           character styles (palette indexes, validated)
  filter.go                teen personal-info filter, anonTag/postTag (HMAC), rate limiters
  *_test.go                unit tests + end-to-end WebSocket tests
web/embed.go               embeds web/static into the binary
web/static/                index.html, style.css, app.js (Canvas 2D isometric renderer,
                           no build step, no frameworks)
start-windows.bat          git pull, then run (self-updating local copy)
start-mac-linux.command    same for Mac/Linux
```

**WebSocket protocol** (JSON over `/ws`, every message has a `t` field)
- Client → server: `walk {x,y}`, `chat {text}`, `emote {emote}`, `named {on}`, `teleport {room}`, `peek {room}`, `signup {username,password,birthYear,style}` (style = `{skin,hair,outfit,accessory}` palette indexes), `login {username,password}`, `resume {token}`, `board {room,page,sort}` (sort: `active`, `new`, `hot`, `top`, `downvoted`), `vote {thread,value}` (1, -1 or 0) → `votes {thread,ups,downs,mine}` to the voter and `votes {thread,ups,downs}` to the room, `thread {thread}`, `post_thread {title,text}` (posts to the room you're in), `post_reply {thread,text}`, `notes_read`, `use {prop}` (walks over first if needed; benches: sit).
- Rooms carry `props` (`{id,kind,at,w,h,dir,count}`); avatars carry `style` and `seat`. Using a prop broadcasts `fx {prop,kind,by,count}` to the room; `authed` includes the account's `style`.
- Server → client: `welcome {rooms}`, `room {room,you,avatars,history}`, `enter {avatar}`, `leave {id}`, `update {avatar}`, `path {id,from,path,stepMs}`, `chat {msg}`, `emote {id,emote}`, `counts {counts}`, `peek {room,name,count,recent}`, `need_account`, `authed {username,token,recoveryCode,ageBand}`, `auth_error {message}`, `first_message`, `notice {message}`, `board {room,name,page,threads,more}`, `thread {thread,replies}`, `posted_thread {thread}`, `board_update {thread}` (new or bumped thread, sent to people in that room), `reply {thread,reply}` (sent to whoever has the thread open), `post_error {message}`, `notes {notes}` (after login/resume), `notify {note}`.

**Key behaviours to preserve**
- Guests can't chat; `chat` from a guest returns `need_account`. The client then opens signup and sends the pending message after `authed`.
- No "X joined/left" chat lines, ever.
- Movement is server-authoritative: the server pathfinds, broadcasts the path, and steps avatars on its tick. Clients only animate.
- Arrivals go to the nearest free tile near the room's spawn or door target.
- Full rooms bounce the avatar back off the door with a notice.
- Rooms have a `theme` (`plaza`, `gaming`, `gym`); the client maps themes to colours in `THEMES` in `app.js`.

**Known gaps (prototype limits)**
- Persistence needs `-db`; without it a restart forgets accounts and chat.
  Saved data: accounts, sessions, chat messages (with the real account ID, for
  moderation; never sent to clients) and the anon-tag secret, so tags stay
  stable across restarts. Sessions don't expire yet and old messages are
  never pruned. Board posts and replies are kept forever by design.
- Boards: no search, no reply-quoting (`>>123`), no public read-only web pages
  per board yet (useful for sharing and search engines).
- Recovery codes are generated and hashed, but can't be redeemed yet.
- No report/block/mute, mod tools, word filter, admin panel or analytics yet.
- Art is a small Canvas 2D voxel renderer in `app.js` (`characterModel`,
  `drawModel`, `drawFloor`, `drawDoor`, `drawProp`, `drawAvatar`, `turn`);
  props that don't animate are drawn once into cached sprites (`sprite`).
- Fountain wish counts are in memory only (reset on restart).
- Known scaling limit: every move and chat line in a room goes to everyone
  in it. Fine for a prototype; a full 500-person plaza will need "area of
  interest" filtering (only send what's near you) and load testing beyond
  the ~255 browser connections tested so far.
- Characters can only be designed at signup; there's no "edit my look" yet.
  Pigeons are client-side decoration, so each viewer sees their own.

## 5. Next tasks, in order

1. ~~**Persistence:** Postgres for accounts, sessions and chat history. Keep the in-memory store behind an interface for tests.~~ Done (`internal/store`).
2. **Recovery-code redemption + change password + delete account.**
3. **Safety tools** (covering chat **and board posts**: mods must be able to remove threads/replies): report (message, post or avatar), block/mute (client- and server-side), room mods (kick/timeout/ban, acting on anon IDs), word filter + toxicity scoring hook, new-account slow mode, audit log, minimal staff admin page.
4. **More communities:** make room definitions data-driven (config/DB), add 3–8 more seeded communities, each with 2–4 chatrooms behind doors off its lobby.
5. **Shareable room links** (`/r/<room>` drops you straight into that room as a guest).
6. **Funnel analytics** for the §3 success targets (privacy-friendly, no third-party trackers).
7. **Real illustrated art** for avatars, furniture and doors as sprite sheets on fixed templates (likely move rendering to PixiJS), so creator uploads (PLAN.md §2.5) fit later.
8. **Closed alpha** with 2–3 small communities; measure time to first message.

## 6. Conventions

- Keep it simple: standard library first, few dependencies. Run `gofmt`, `go vet ./...` and `go test -race ./...` before committing.
- Never add email or phone fields, third-party trackers or ads.
- Anything that shows identity must default to anonymous.
- Any new chat or posting surface must apply the teen filter and rate limits.
- Commit as the repo owner; don't add AI or tool attribution to commits or files.
