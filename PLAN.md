# Habit — Product & Technical Plan

> **Pitch:** A social platform where every community is a *place* you can walk into.
> Fast, low-friction posting like an imageboard; ranked, threaded discussion like a
> forum; live voice/text presence like a chat server, all wrapped in a pixel-art
> virtual world where your avatar, your room and your reputation are the game.
> Joining a community should feel like wandering into a room, not
> signing up for a server.

Name: **Habit**. Run a trademark search before launch, since the name is
close to "Habbo". Keep the art and branding clearly original.

---

## 1. What we borrow from each inspiration

| Source | What we keep | What we deliberately avoid |
|---|---|---|
| **Imageboards (4chan)** | Optional anonymity, image-first threads, threads that expire ("bump" economy), low barrier to posting, board culture and in-jokes | Zero moderation, no accountability, illegal content, raids/harassment |
| **Reddit** | Communities ("Boards") owned by their own moderators, voting and ranking, threaded comments, wiki/rules per community | Karma farming, repost bots, one global front page that flattens every community into the same feed |
| **Discord** | Real-time chat channels, voice rooms, roles and permissions, presence ("who's here right now") | Siloed servers you can't discover; history that's impossible to search |
| **Habbo** | Isometric pixel-art rooms, customizable avatars, furniture ("furni") and room building, trading, room events and mini-games | Pay-to-win, predatory loot boxes, unmoderated private messages with minors |

**The core twist:** a Board isn't only a feed. It's also a **Hub**, a shared
isometric space. Threads show up as objects in the room (a noticeboard, a
campfire, a TV screen). Posting, voting and showing up earn currency and XP,
which you spend making your avatar and your room your own.

---

## 2. Core concepts (the nouns)

- **User / Avatar**: one account with two posting modes:
  - **Named**: avatar visible, your reputation counts.
  - **Anon-in-context**: shows as "Anonymous" with a per-thread ID hash (like
    imageboard poster IDs). The account still exists server-side for moderation
    and bans. *Public* anonymity, not *system* anonymity.
- **Board**: a community (e.g. `/retro/`, `/cooking/`). Has rules, mods, flair,
  settings (anon allowed? NSFW? image-only?), and a Hub room.
- **Hub**: the Board's isometric room. Live avatars, chat bubbles, furni and
  "thread objects" you can click to open.
- **Thread**: an OP (text and/or image) plus replies. Two display modes per Board:
  - **Bump mode** (imageboard): threads sorted by last reply, capped catalog,
    oldest threads fall off and get archived.
  - **Ranked mode** (Reddit): votes plus time decay, nested comments.
- **Channel / Chatroom**: a persistent real-time text/voice channel attached to
  a Board (the Discord layer). Every chatroom is **also a physical room** in
  the world: you walk in through a door to join it and walk out to leave
  (see §2.1).
- **Room**: a user-owned private or public space (your "house"). Decorate it,
  invite people, host events.
- **Furni / Items**: cosmetic objects. Some are tradeable; limited editions
  are earned through events, never only bought.
- **Currencies**:
  - **Pips** (soft currency): earned by daily login, quality posts (upvotes
    after a delay, so it can't be spammed), event participation and quests.
  - **Gems** (premium): bought with money. Cosmetics only, **never** ranking,
    visibility or moderation power.
- **Reputation / XP**: per-Board and global levels. They unlock cosmetics and
  small privileges (posting images in strict boards, creating rooms), never
  vote weight.

### 2.1 The walkable world: visiting chatrooms

The defining feature of Habit: **you don't click into a chat, you walk into it.**

**Why: joining should feel like walking into a café, not a job interview.**
Many people find joining a Discord server intimidating. You commit to "join"
before you know what it's like, you land in a wall of channels and a member
list of strangers, your arrival is announced, and the first message feels like
speaking up in a silent room. Walking into a room fixes this because it works
the way people already handle real spaces:

- **No commitment to look around.** Visiting a room isn't "joining" anything.
  There's no member list you're added to and no server icon in your sidebar
  until *you* choose to favourite it.
- **Peek before you enter.** Stand in the doorway and you can see who's
  inside, how busy it is and what they're talking about before stepping in.
- **Lurking is normal and visible-but-quiet.** You can stand at the edge, sit
  on a sofa or watch the noticeboard. Being present without talking is a
  natural, accepted thing to do, not "being a lurker".
- **Body language before words.** Wave, sit next to someone, dance or use an
  emote. You can join a conversation with an action before you have to type.
- **Leave without a trace.** Walking out is as quiet as walking in. No
  "user left the server" message and no awkward goodbye.
- **Small groups, not one giant feed.** In a busy room people cluster
  physically (and with proximity voice, literally), so you talk to the 4
  people near you, not to 300 at once.
- **Friendly onboarding inside the world.** New visitors get a "first visit"
  badge so regulars know to say hi. Boards can appoint **greeters**, and every
  Hub has a "newcomers' corner" near the entrance.

Every design decision about rooms should be checked against this: *does it
make walking in feel less like a commitment and more like a visit?*

```
                 ┌──────────────── The Plaza (spawn) ───────────────┐
                 │  signposts / portals to popular Boards, events   │
                 └──────┬──────────────┬───────────────┬────────────┘
                        │              │               │
                 ┌──────▼─────┐ ┌──────▼─────┐  ┌──────▼─────┐
                 │ /retro/ Hub│ │/cooking/Hub│  │ /art/ Hub  │   ← Board lobbies
                 │ noticeboard│ │            │  │            │
                 └─┬───┬────┬─┘ └────────────┘  └────────────┘
          door ────┘   │    └──── door
       ┌───────▼──┐ ┌──▼───────┐ ┌──────────┐
       │ #general │ │ #trading │ │ 🔊 lounge │   ← chatrooms = rooms
       └──────────┘ └──────────┘ └──────────┘
                         ↑ user rooms ("houses") linked from profiles / room browser
```

- **World graph**: the world is a graph of rooms connected by **doors and
  portals**. The Plaza is where you spawn. Each Board's Hub is its lobby, and
  every chat channel the Board creates becomes a room off that lobby. Users'
  own rooms are reached from their profile, a friend's door, or the room browser.
- **Walking in = joining**: stepping through a door moves your avatar to the
  room, subscribes you to its chat, shows you recent history, and adds you to
  the presence list. Walking out leaves. One room at a time keeps it feeling
  like a place (you can pin a few channels to a sidebar so you don't miss mentions).
- **Seeing before entering**: doors show a live head-count and a small
  preview ("#trading · 14 inside · 🔊 voice active"). Hover a door to read the
  last few messages. Locked or private rooms show a padlock; knock to ask in.
- **Chat in the room**: messages show as speech bubbles over avatars **and**
  in a regular scrollable chat log on the side, so text conversation stays
  usable, readable and searchable (the Discord part) even in a busy room.
- **Voice rooms**: 🔊 rooms use proximity voice. Walk up to a group to hear
  them, walk away to fade out. Rooms can also have a "stage" tile area for
  talks or AMAs, where whoever is standing on it is heard by the whole room.
- **Getting around fast**: walking is the fun default, but there's always a
  shortcut: a **map / room navigator** (search, favourites, "friends are
  here", "trending rooms") to teleport straight to a door. Nobody should
  have to walk across five rooms to reach their usual chat.
- **Follow a friend**: "Go to" on a friend's name teleports you to their room
  (if they allow it), like Habbo's "follow".
- **Crowds**: each room has a cap (e.g. 50 avatars). When it's full, the door
  opens a parallel instance ("#general · room 2"). The **chat log is shared
  across instances**, so the conversation stays whole even when the avatars
  are spread across several copies of the room.
- **Accessibility / low-power mode**: a "list view" toggle shows the same
  room as a plain Discord-style channel list and chat. Same rooms, no rendering,
  for screen readers, slow phones, or people who just want to chat.

---

## 3. Feature set by phase

### Phase 0: Foundations (weeks 0–4)
- Decide name, legal entity, content policy and age policy (**13+ with a
  separate 18+ gated area, or 18+ only**; see §7).
- Design system: pixel-art style guide, avatar base, 20–30 starter furni.
- Repo scaffolding, CI, staging environment, error tracking.

### Phase 1: MVP "the forum works" (weeks 4–12)
Goal: a usable imageboard/forum hybrid. No game world yet.
- Auth (email + OAuth), account settings, 2FA.
- Boards: create (by invite at first), rules, moderator roles.
- Threads and replies with images (upload → scan → thumbnail → CDN).
- Named/anon toggle per post, per-thread poster IDs.
- Bump mode and Ranked mode; catalog view; archive.
- Voting, reporting, mod queue, bans (account, plus device/IP-reputation signals).
- **Mandatory safety pipeline before launch:** hash-matching for known CSAM
  (e.g. PhotoDNA / NCMEC hash lists or a vendor like Thorn Safer), NSFW
  classifier, rate limits, CAPTCHA on anon posting.
- Basic 2D profile with a static avatar picture.

### Phase 2: "the world appears" (weeks 12–24)
- Isometric **Hub** client for each Board: walk, sit, emote, chat bubbles.
- **The Plaza** and the world graph: doors and portals between the Plaza,
  Board Hubs and chatrooms; door previews with live head-counts; room
  navigator / map for teleporting.
- **Walk-in chatrooms**: each Board text channel is a room you enter through
  a door (walk in = join, walk out = leave), with a side chat log next to
  the speech bubbles, and list view as a fallback.
- Thread objects in the Hub (noticeboard showing the top 5 threads, etc.).
- Avatar editor (layered sprites: body, hair, outfit, accessories).
- Inventory, starter furni, **Pips** economy, daily quests ("reply to 3
  threads", "visit 2 new Hubs").
- Personal Rooms: place furni on an isometric grid, permissions (open / friends
  / locked / password).
- Presence: "12 people in /retro/ right now".

### Phase 3: "the Discord layer" (weeks 24–36)
- Channel roles and permissions (who can enter which room, locked doors,
  knock-to-enter, mod-only back rooms), searchable chat history.
- Voice rooms (WebRTC SFU) as walkable 🔊 rooms with proximity voice and
  "stage" tiles for talks.
- "Follow friend" / "Go to" teleport and "friends are here" in the navigator.
- Events: scheduled AMAs, game nights, "Bump Wars" (time-boxed thread contests).
- Friends list, DMs (**off by default for minors**, opt-in only, with filters).
- Notifications (web push, mobile).

### Phase 4: "the game deepens" (months 9–12+)
- Mini-games inside rooms: trivia, drawing/guessing, simple racing, wired-style
  furni logic (triggers → actions, like Habbo "Wired").
- Trading marketplace for tradeable furni (escrow, anti-scam and anti-RMT rules).
- Seasonal events and a cosmetic-only battle pass.
- Board vs. Board leaderboards (creative contests, not raids).
- Mobile apps (React Native / Capacitor wrapper of the web client first).
- Creator tools: Board mods can design custom furni for their community
  (with review).

---

## 4. Gamification design (making it fun without making it toxic)

**Earning loops**
1. *Show up*: daily login streak gives Pips (capped).
2. *Contribute*: a post that receives upvotes from N **distinct, aged**
   accounts after 24h earns Pips. Delayed and diminishing, so farming is hard.
3. *Participate*: events, quests and mini-games give XP and event-exclusive furni.
4. *Create*: popular rooms (unique visitors, likes) earn a room badge.

**Spending loops**
- Avatar clothes, room furni, chat bubble styles, name effects, Hub emotes.

**Guardrails**
- Nothing purchasable affects ranking, visibility, votes or moderation.
- No randomized paid loot boxes. If there are mystery boxes, they're Pips-only
  and show their drop rates.
- Anon posts still earn Pips for the account (encourages good anon posting)
  but don't affect public reputation.
- Daily earning caps; per-Board anti-brigading (votes from accounts with no
  history on that Board are down-weighted).

---

## 5. Technical architecture

### 5.1 Stack recommendation

| Layer | Choice | Why |
|---|---|---|
| Web client | **TypeScript + React (Next.js)** for forum/UI pages | SEO for threads, fast iteration |
| Game client | **PixiJS** (WebGL 2D) inside the React app | Well suited to isometric pixel-art sprites; light; runs in browser |
| Realtime | **WebSockets** via a dedicated gateway (Node + uWebSockets.js, or Elixir/Phoenix) | Presence, chat, avatar movement |
| Voice | **LiveKit** (self-hosted or cloud) WebRTC SFU | Proximity voice and voice rooms without building an SFU |
| API | **Node.js (Fastify) or Go**, REST + a typed client (tRPC/OpenAPI) | Team familiarity; typed contracts |
| DB | **PostgreSQL** | Users, boards, posts, economy ledger (needs transactions) |
| Cache / pubsub | **Redis** | Sessions, rate limits, presence, room state fan-out |
| Search | **Meilisearch** or OpenSearch | Thread and Board search |
| Media | Object storage (S3/R2) + CDN + image worker (sharp/libvips) | Uploads, thumbnails, sprite sheets |
| Queue | Redis Streams / BullMQ, or NATS | Media scanning, notifications, Pips payouts |
| Infra | Docker, a single region to start (Fly.io / Render / AWS ECS), Terraform later | Keep ops small early |
| Observability | OpenTelemetry, Sentry, Grafana | |

**Monorepo layout (proposed)**
```
apps/
  web/          # Next.js: forum pages, account, settings, embeds the game client
  game/         # PixiJS isometric client (package consumed by web)
  api/          # HTTP API
  gateway/      # WebSocket realtime server (rooms, presence, chat)
  workers/      # media pipeline, moderation scans, economy jobs
packages/
  shared/       # types, validation schemas (zod), constants
  ui/           # design-system components
  sprites/      # avatar/furni asset pipeline + manifests
infra/          # docker-compose, terraform, k8s (later)
docs/
```

### 5.2 Realtime room model
- Each Hub/Room is a **room actor** owned by one gateway node (sharded by room
  ID through consistent hashing; Redis tracks room → node).
- The server is authoritative on tile positions: the client sends a "walk to
  (x,y)", the server pathfinds (A* on the tile grid) and broadcasts the path.
- Tick rate is low (~5–10 Hz). This isn't a shooter, and movement is
  tile-based and interpolated on the client.
- Room capacity cap (e.g. 50 avatars). Overflow creates "instances"
  (`/retro/ Hub #2`).
- Chat messages go through the moderation filter before broadcast; they're
  logged with a retention policy.
- **Room transitions**: rooms are linked by `doors(room_id, x, y,
  target_room_id, target_x, target_y)`. When an avatar steps on a door tile,
  the gateway moves it out of one room actor and into the next (possibly on
  another node). The client fades out, loads the new room's layout and sprites
  (prefetched when you approach the door), and fades back in, in under a
  second.
- **Chat vs. avatar state are separate**: avatar positions live in the
  room-instance actor (lost on restart, which is fine). Chat messages belong
  to the *channel* and are stored in Postgres, so all instances of a room,
  and list view, read the same log.
- **Door previews**: presence counts per room are kept in Redis and pushed to
  anyone standing in an adjacent room, so doors update live.

### 5.3 Data model sketch (core tables)
```
users(id, email, created_at, age_band, status, trust_score)
avatars(user_id, layers_json, display_name)
boards(id, slug, title, mode[bump|ranked], anon_policy, nsfw, created_by)
board_roles(board_id, user_id, role[owner|mod|janitor])
threads(id, board_id, author_id, is_anon, anon_hash, title, body, media_id,
        bumped_at, score, locked, archived_at)
posts(id, thread_id, parent_id, author_id, is_anon, anon_hash, body, media_id,
      created_at, deleted_at)
votes(user_id, target_type, target_id, value, created_at)
media(id, uploader_id, sha256, phash, status[pending|ok|blocked], cdn_key)
reports(id, reporter_id, target_type, target_id, reason, status, handled_by)
bans(id, scope[global|board], board_id, user_id, reason, expires_at)
items(id, sku, name, sprite_ref, tradeable, rarity)
inventory(id, user_id, item_id, acquired_via, acquired_at)
rooms(id, owner_id NULL, board_id NULL, channel_id NULL, kind[plaza|hub|chat|voice|user],
      layout_json, access, capacity)
doors(id, room_id, x, y, target_room_id, target_x, target_y, lock[open|knock|role|password])
room_items(room_id, inventory_id, x, y, z, rotation)
ledger(id, user_id, currency, delta, reason, ref_id, created_at)  -- append-only
channels(id, board_id, kind[text|voice], name, perms_json)
messages(id, channel_id, author_id, body, created_at)
```
**Economy rule:** balances are derived from the append-only `ledger` (with
cached totals). Every grant or spend is one DB transaction. This prevents
duplication exploits, the main way virtual economies get broken.

### 5.4 Anonymity implementation
- `anon_hash = HMAC(server_secret_rotated, user_id || thread_id)[:8]`: stable
  within a thread, unlinkable across threads to everyone except the server.
- Mods see "anon #a1b2c3d4", with a "ban the poster" button that resolves
  server-side. Mods never see the real account (only admins/T&S do, and it's
  audited).

---

## 6. Moderation & Trust and Safety (non-negotiable, built from day one)

Imageboard-style anonymity plus images plus a Habbo-style space popular with
teens is the **highest-risk combination in social software**. This section is
what decides whether the product survives.

1. **Layered moderation**
   - *Automated*: CSAM hash matching (blocks the upload and triggers mandatory
     reporting to NCMEC in the US), NSFW/gore classifiers, text toxicity and
     spam scoring, link reputation.
   - *Community*: Board mods and janitors with clear tools (remove, lock,
     ban-from-board, slow mode, anon off).
   - *Platform*: a paid T&S team (or vendor) for escalations, appeals and
     legal requests.
2. **Friction where it matters**
   - New accounts: posting rate limits, no images for the first X hours or
     until verified, CAPTCHA on anon posts.
   - Trust score built from account age, verified email/phone, mod actions
     and report accuracy.
3. **Minor safety (if under-18s are allowed)**
   - Age banding at signup plus age assurance for 18+ areas.
   - NSFW Boards hidden and blocked for minor accounts.
   - DMs and private rooms limited for minors; adults can't DM minor accounts
     they aren't friends with.
   - Chat filter on stricter settings by default; a "safe Hub" mode.
4. **Transparency**: public content policy, mod log per Board, appeals,
   transparency reports (required under the EU DSA at scale).
5. **Anti-raid**: cross-Board brigade detection (vote/visit spikes from an
   outside referrer), Board "lockdown" mode, no organized "raid" features.

---

## 7. Legal & compliance checklist (talk to a lawyer before launch)
- Terms of Service, Privacy Policy, Content Policy, Community Guidelines.
- **US**: Section 230 protections, DMCA agent registration plus a takedown
  flow, NCMEC CyberTipline reporting obligations, **COPPA** (no under-13s
  without verifiable parental consent; easiest to just bar them).
- **EU**: GDPR (data export/delete, lawful basis), **Digital Services Act**
  (notice-and-action, statements of reasons, transparency).
- **UK**: Online Safety Act (illegal content and children's risk assessments,
  age assurance for adult content).
- Virtual currency: no cash-out, so it isn't money transmission. Refund
  policy, regional loot box rules (Belgium/Netherlands), consumer law.
- Payments through Stripe / app stores; app store rules for UGC apps
  (report/block tools, filtering, published contact info).

---

## 8. Monetization
1. **Cosmetics** (Gems): avatar outfits, furni, room themes, name effects.
2. **Supporter subscription** ("Habit+"): monthly Gem stipend, extra room
   slots, bigger uploads, profile flair. No ranking advantages.
3. **Board Boosts**: communities pool Gems to unlock Hub themes, extra
   channels and custom emotes (like Discord server boosts).
4. **Creator revenue share** (later): mods/artists design furni and get a
   cut of sales.
5. **No** pay-for-visibility on posts, and no ads at launch (ads plus
   anonymous UGC means brand-safety problems; revisit later with contextual
   ads only).

---

## 9. Metrics that matter
- **Activation**: % of new users who post or reply within 24h; % who enter a Hub.
- **Retention**: D1 / D7 / D30; weekly active posters.
- **Health**: reports per 1k posts, time to action on reports, % of content
  actioned automatically vs. by humans, repeat-offender rate.
- **Economy**: Pips inflation (earned vs. sunk), Gem conversion rate,
  ARPPU.
- **Community**: number of Boards with ≥1 active mod and ≥20 weekly posters.

---

## 10. Team & budget (rough, lean start)
| Role | MVP (Phase 1–2) |
|---|---|
| Full-stack engineers | 2 |
| Game/realtime engineer (PixiJS, WebSockets) | 1 |
| Pixel artist / designer | 1 (contract OK) |
| Trust & Safety lead / community manager | 1 (**hire before public launch**) |
| Founder/PM | 1 |

Infra at MVP scale: roughly $300–$1.5k/month (DB, Redis, CDN, LiveKit,
moderation APIs). Moderation vendors and CSAM scanning are often free or
discounted for small platforms; apply early.

---

## 11. Launch strategy
1. **Closed alpha**: 5–10 hand-picked Boards around niches with existing
   culture (retro games, pixel art, a specific fandom). Recruit their mods first.
2. **Seed the world**: a launch event in a central "Plaza" Hub with
   exclusive first-season furni (strong FOMO hook, costs nothing).
3. **Invite waves**: each member gets invites; Boards unlock Hub themes as
   they grow.
4. **Open beta**: public signup with all safety systems tested under load
   (run a red-team week of trying to break moderation first).

---

## 12. Top risks & mitigations
| Risk | Mitigation |
|---|---|
| Becomes a haven for illegal or extremist content | Hash scanning, account-backed anonymity, paid T&S, clear policy enforced from day 1 |
| Child safety incident | Age policy decision in Phase 0, DM restrictions, safe defaults, age assurance |
| Empty world ("ghost town") | Launch small and dense; Hubs show the forum activity (threads as objects) so they feel alive even when few avatars are online; scheduled events |
| Economy exploits / duping | Append-only ledger, server-authoritative everything, rate caps, anomaly alerts |
| Scope creep (it's 4 products in one) | Strict phasing: forum first, world second, voice third, mini-games last |
| Realtime scaling costs | Room caps and instancing; low tick rate; voice only on demand |

---

## 13. Immediate next steps (this repo)
1. Pick: **age policy**, **final name**, and **backend language** (Node vs. Go).
2. Scaffold the monorepo (`apps/web`, `apps/api`, `packages/shared`) with
   docker-compose for Postgres and Redis.
3. Build Phase 1 vertical slice: sign up → create Board → post a thread with an
   image (anon or named) → reply → vote → report → mod removes it.
4. Prototype in parallel: a PixiJS **Plaza plus two chatrooms joined by
   doors**. Avatars walk via WebSocket, step through a door into a chatroom,
   and chat with bubbles and a side log. This tests the core "walk in to
   chat" feel and the art direction early.
5. Write the Content Policy draft and pick a CSAM-scanning vendor before any
   public image uploads.
