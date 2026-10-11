# Habit: MVP brief (handoff for developers and coding agents)

Read this first. The full product plan is in `PLAN.md`; this is the short
version plus the current state of the code.

## 1. What Habit is

A pixel-art, Habbo-style world where **every chatroom is a real room you walk
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
| Backend | **Go** (1.24+). WebSockets via `github.com/coder/websocket`. Password hashing via `golang.org/x/crypto/argon2`. |
| Accounts | **Username + password only. Never ask for email or phone.** Lost password is handled by a one-time **recovery code** shown after the user's first message. |
| Anonymity | **Anonymous by default.** Users show as `Anonymous #3f9a`; the tag is an HMAC of (user, room, day), so it's stable within a room for a day and unlinkable across rooms. Showing your username is opt-in. Avatar IDs sent to clients are fresh on every room entry, so nobody can follow someone between rooms. Mods act on anon IDs; the server resolves the account. |
| Ages | **13+.** Under-13 signups are refused. Teens (13–17) get protections on by default: a personal-info filter (phone numbers, emails, addresses, school, "add me on snap"), and later grooming-risk detection and stricter DMs. Age bands: `teen_13_15`, `teen_16_17`, `adult`. When unsure, round to the younger band. |
| Speech | Any **topic** can be discussed everywhere. The main world has no explicit sexual content or gore. An age-checked 18+ **"After Dark"** area (Phase 2, not MVP) allows anything legal. Hard lines everywhere: illegal content, CSAM, credible threats, doxxing, organised harassment/raids, spam/scams. Principle: "moderate behaviour, not opinions." |
| World structure | The **Plaza** (where you arrive) has doors to **community** rooms. Right now: **Gaming** and **Gym**. Later, each community gets its own chatrooms behind further doors. |
| Monetization | Not in the MVP. Later: cosmetics, Habit+ subscription, and the **Penthouse** premium tier (~$999/yr) with a verified marketplace (Phase 4). No ads, ever. Nothing purchasable affects ranking or moderation. |

## 3. MVP scope (12 weeks)

The MVP must prove one thing: **a first-time visitor can go from a shared link
to their first message in under 60 seconds, and it feels less intimidating
than Discord.**

**In the MVP**
- The Plaza + 5–10 seeded community rooms (currently 2: Gaming, Gym), doors between them.
- Click-to-walk (server-authoritative), emotes (wave, dance, laugh, heart, sit).
- Text chat: speech bubbles over avatars + a side chat log, recent history on entering, quick-start prompts.
- Approachability: guest mode (look, walk, wave; no talking), door peeking (live head-count + last few messages), quiet entry/exit, "new" badge, room navigator with live counts and teleport, shareable room links.
- Accounts: one-step username + password + birth-year signup **without leaving the room**; recovery code; change password; delete account.
- Safety: report, block, mute; room mods can kick, timeout, ban; word filter + toxicity scoring; rate limits; new-account slow mode; teen personal-info filter; staff admin panel; audit log.
- List view (plain chat, no world rendering) and a phone-friendly layout.
- Funnel analytics: link → in world → entered a room → signed up → first message.

**Not in the MVP:** threads/imageboard posting, image uploads, After Dark,
voting, voice, DMs, friends, personal rooms, room building, items/currency,
quests, mini-games, trading, Penthouse/marketplace, native mobile apps.

**Success targets:** ≥25% of guests sign up to chat; ≥70% of signups send a
message within 5 minutes; median time to first message under 60 seconds; ≥25%
of chatters come back after 7 days.

## 4. What's built so far (week 0–2 prototype)

Run with `go run ./cmd/habit`, then open http://localhost:8080. Run the tests with `go test ./...`.

```
cmd/habit/main.go          server binary; flag -addr (default :8080)
internal/world/            pure, no runtime state
  world.go                 Point (JSON [x,y]), Room, Door, Kind, Theme, Validate()
  rooms.go                 DefaultRooms(): plaza, gaming, gym
  path.go                  A* pathfinding; door tiles only allowed as the final step
internal/server/           realtime gateway
  server.go                Server: one mutex guards all state; tick loop moves avatars;
                           door transfers; chat; counts; peek; teleport; enter/leave
  accounts.go              in-memory username+password accounts (argon2id), tokens,
                           recovery codes, age bands, reserved names
  filter.go                teen personal-info filter, anonTag (HMAC), chat rate limiter
  *_test.go                unit tests + end-to-end WebSocket tests
web/embed.go               embeds web/static into the binary
web/static/                index.html, style.css, app.js (Canvas 2D isometric renderer,
                           no build step, no frameworks)
start-windows.bat          git pull, then run (self-updating local copy)
start-mac-linux.command    same for Mac/Linux
```

**WebSocket protocol** (JSON over `/ws`, every message has a `t` field)
- Client → server: `walk {x,y}`, `chat {text}`, `emote {emote}`, `named {on}`, `teleport {room}`, `peek {room}`, `signup {username,password,birthYear}`, `login {username,password}`, `resume {token}`.
- Server → client: `welcome {rooms}`, `room {room,you,avatars,history}`, `enter {avatar}`, `leave {id}`, `update {avatar}`, `path {id,from,path,stepMs}`, `chat {msg}`, `emote {id,emote}`, `counts {counts}`, `peek {room,name,count,recent}`, `need_account`, `authed {username,token,recoveryCode,ageBand}`, `auth_error {message}`, `first_message`, `notice {message}`.

**Key behaviours to preserve**
- Guests can't chat; `chat` from a guest returns `need_account`. The client then opens signup and sends the pending message after `authed`.
- No "X joined/left" chat lines, ever.
- Movement is server-authoritative: the server pathfinds, broadcasts the path, and steps avatars on its tick. Clients only animate.
- Arrivals go to the nearest free tile near the room's spawn or door target.
- Full rooms bounce the avatar back off the door with a notice.
- Rooms have a `theme` (`plaza`, `gaming`, `gym`); the client maps themes to colours in `THEMES` in `app.js`.

**Known gaps (prototype limits)**
- Everything is in memory: a restart forgets accounts and chat.
- Recovery codes are generated and hashed, but can't be redeemed yet.
- No report/block/mute, mod tools, word filter, admin panel or analytics yet.
- Placeholder art (shapes drawn on canvas).

## 5. Next tasks, in order

1. **Persistence:** Postgres for accounts, sessions and chat history. Keep the in-memory store behind an interface for tests.
2. **Recovery-code redemption + change password + delete account.**
3. **Safety tools:** report (message or avatar), block/mute (client- and server-side), room mods (kick/timeout/ban, acting on anon IDs), word filter + toxicity scoring hook, new-account slow mode, audit log, minimal staff admin page.
4. **More communities:** make room definitions data-driven (config/DB), add 3–8 more seeded communities, each with 2–4 chatrooms behind doors off its lobby.
5. **Shareable room links** (`/r/<room>` drops you straight into that room as a guest).
6. **Funnel analytics** for the §3 success targets (privacy-friendly, no third-party trackers).
7. **Real pixel art** for avatars, furniture and doors (likely move rendering to PixiJS).
8. **Closed alpha** with 2–3 small communities; measure time to first message.

## 6. Conventions

- Keep it simple: standard library first, few dependencies. Run `gofmt`, `go vet ./...` and `go test -race ./...` before committing.
- Never add email or phone fields, third-party trackers or ads.
- Anything that shows identity must default to anonymous.
- Any new chat surface must apply the teen filter and rate limits.
- Commit as the repo owner; don't add AI or tool attribution to commits or files.
