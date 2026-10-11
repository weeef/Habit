# Habit — Product & Technical Plan

> **Pitch:** A social platform where every community is a *place* you can walk into.
> Fast, low-friction posting like an imageboard; ranked, threaded discussion like a
> forum; live voice/text presence like a chat server, all wrapped in a pixel-art
> virtual world where your avatar, your room and your reputation are the game.
> Joining a community should feel like wandering into a room, not
> signing up for a server.

Name: **Habit**. Run a trademark search before launch, since the name is
close to "Habbo". Keep the art and branding clearly original.

**The bet:** Habit wins or loses on one thing: **how easy and approachable
it is to start chatting.** Threads, voice, economy and games all come later.
The MVP (§3) exists to prove that walking into a room and saying hi feels
easier than joining a Discord server.

---

## 0. Positioning & marketing

**Tagline options**
- *"Walk in. Say hi."* (recommended)
- *"Anonymous. Walk in. Say anything."* (for the open-speech angle)
- *"Chat rooms you can actually walk into."*
- *"No servers to join. Just rooms to visit."*

**One-liner:** Habit is a pixel-art world where every chatroom is a real room.
You can see who's inside before you go in, hang out without having to talk, and
join the conversation when you feel like it.

**Who it's for (launch audience)**
1. **People put off by Discord**: they want community, but joining a
   server full of strangers feels like crashing a party. Our core user.
2. **The nostalgic**: grew up on Habbo, Club Penguin, Neopets or old forums
   and miss hanging out *somewhere* instead of scrolling *something*.
3. **Small niche communities and creators** (pixel artists, retro
   gamers, a streamer's regulars) who want a hangout that feels cosy, not
   like a corporate Slack.

**Why it's marketable**
- **It's visual and clippable.** A busy pixel room full of avatars and speech
  bubbles makes a great 10-second TikTok, Reel or Short. Chat screenshots
  don't. Every room is an ad for itself.
- **The share link drops you straight in.** `habit.gg/r/retro-lounge` opens
  the room in the browser, and you can look around **as a guest** without
  signing up. Sign-up is only asked for when you try to talk (§3).
- **Nothing personal to hand over.** No email, no phone number, no real
  name. Just pick a username and password. That's a selling point in its own
  right: *"No email. No phone. Just a name."*
- **The message is simple.** "Discord is a building full of locked doors;
  Habit is a street where you can see into every window."
- **Built-in FOMO, but kind.** Live head-counts, "friends are here" and
  founding-member furni give people a reason to come back without dark
  patterns.

**Marketing channels (in order)**
1. **Short-form video**: dev-log clips of the world being built, then
   "day in the life of a Habit room" clips. Start posting before the MVP ships.
2. **Seed communities**: invite 5–10 small communities (subreddits, Discord
   servers, streamers' regulars) to get their own Hub at launch, with
   founder badges for their members.
3. **Reserve your name early**: there's no email waitlist (we don't collect
   emails). Instead, pre-launch visitors **claim their username** by creating
   their account early (username + password) and get a "Founding Visitor"
   item that will never be sold again. Launch announcements go out through
   our social accounts and the seed communities.
4. **Creator partnerships**: small streamers hold "come hang in my room"
   nights. The share link means viewers are in the room within seconds.
5. **Nostalgia communities**: Habbo/Club Penguin/old-web forums, posted
   with care, not spam.

**Brand voice:** warm, playful, a little retro, and **unfiltered**. Habit
is the place where you can be anonymous and say what you actually think.
Marketing leads with the *freedom* and the *welcome* ("Anonymous. Walk in. Say
anything."). It never leads with shock content: the 18+ After Dark area
(§2.2) exists, but it isn't the shop window.

---

## 1. What we borrow from each inspiration

| Source | What we keep | What we deliberately avoid |
|---|---|---|
| **Imageboards (4chan)** | **Anonymous by default**, talk about genuinely anything legal, image-first threads, threads that expire ("bump" economy), low barrier to posting, board culture and in-jokes | Zero moderation, no accountability, illegal content, raids/harassment |
| **Reddit** | Communities ("Boards") owned by their own moderators, voting and ranking, threaded comments, wiki/rules per community | Karma farming, repost bots, one global front page that flattens every community into the same feed |
| **Discord** | Real-time chat channels, voice rooms, roles and permissions, presence ("who's here right now") | Siloed servers you can't discover; history that's impossible to search |
| **Habbo** | Isometric pixel-art rooms, customizable avatars, furniture ("furni") and room building, trading, room events and mini-games | Pay-to-win, predatory loot boxes, unmoderated private messages with minors |

**The core twist:** a Board isn't only a feed. It's also a **Hub**, a shared
isometric space. Threads show up as objects in the room (a noticeboard, a
campfire, a TV screen). Posting, voting and showing up earn currency and XP,
which you spend making your avatar and your room your own.

---

## 2. Core concepts (the nouns)

- **Account**: **username + password only.** No email or phone number is ever
  asked for (see §5.5). Your username is only shown when *you* choose.
- **User / Avatar**: one account with two identity modes, switchable at any
  time (see §2.2):
  - **Anonymous (default)**: shows as "Anonymous" with a short ID that is
    stable within one room or thread (like imageboard poster IDs) and changes
    across them. The account still exists server-side for moderation and bans.
    *Public* anonymity, not *system* anonymity.
  - **Named**: your username is shown and your reputation counts.
- **Board**: a community (e.g. `/retro/`, `/cooking/`). Has rules, mods, flair,
  settings (anon-only, named-only or either; all-ages or After Dark 18+;
  image-only?), and a Hub room.
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
- **The Penthouse**: an optional high-price membership tier (~$1,000/year)
  that gives access to a members-only floor of the world and a verified
  marketplace (see §2.3).
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

### 2.2 Anonymity & open speech: the 4chan part

Habit should feel like 4chan in the ways that matter: **nobody knows who you
are, and you can talk about genuinely anything.** It also has to stay open to
teens (§6.1) and stay legal, so it's built in two layers.

**Anonymous by default, everywhere**
- When you speak in a room or post in a thread, you appear as
  **"Anonymous #3f9a"**. The ID stays the same within that room (resetting
  daily) or that thread, so conversations stay followable, but it changes
  across rooms and threads, so nobody can track you around the platform.
- In fully anonymous rooms your avatar wears a **random outfit for the
  visit**, so your look can't give you away either.
- Showing your username is **opt-in**, per room or per message. Boards can
  set rooms to anon-only (imageboard style), named-only (a regular
  hangout) or either.
- No email or phone (§5.5), no public post history on anonymous messages,
  and no "who is this anon" tool for anyone but audited staff.
- Moderators see "Anonymous #3f9a" too. Their "ban this poster" button
  resolves to the account server-side, so mods can enforce rules without
  ever learning who someone is.

**Two layers of speech**

| | **Main world** (13+) | **After Dark** (18+, age-assured) |
|---|---|---|
| Who can enter | Everyone 13+ | Adults who have passed a privacy-preserving age check; teens can never enter |
| Topics | **Any topic is open for discussion**: politics, religion, drugs, sex education, mental health, crime, conspiracy theories, controversial and unpopular opinions. Nothing is off-limits *as a subject*. | Same, plus **anything legal for adults**: explicit sexual talk, graphic descriptions, offensive and dark humour, unfiltered language |
| What's not allowed (beyond the hard lines) | Explicit sexual content, graphic gore, content that *encourages* self-harm or eating disorders (discussing them and seeking support is fine) | Only the hard lines below |
| Filters | Profanity filter optional per room; teen protections always on | Off by default |
| Images (Phase 2) | All-ages images only, scanned | No pornographic images (text only), because of legal and payment-processor exposure; other adult images scanned |

**Hard lines everywhere, even in After Dark** (4chan has these too: its
global rules ban illegal content):
- Anything illegal where we operate. In particular, **any sexual content
  involving minors** is removed and reported to NCMEC.
- Credible threats, incitement to violence, terrorist propaganda.
- **Doxxing**: posting someone's private information.
- Targeted harassment campaigns and organised raids on people or communities.
- Non-consensual intimate imagery; spam, scams and malware.

The aim is **"moderate behaviour, not opinions."** Mods remove what crosses a
hard line or breaks a room's own rules. Nobody gets removed for holding an
unpopular view.

**After Dark age check, without breaking anonymity**
- A third-party age-assurance provider runs a one-off check, usually a selfie
  age estimate or an ID check. The provider doesn't share your identity with
  us and deletes the data. We get back only "**adult: yes**" plus a date.
- Habit stores just that flag on the account. No name, no ID scan, no
  photo. It's optional, and the main world never needs it.
- After Dark is reached through clearly marked doors in the world (a
  "basement" or "back alley"), hidden entirely from teen accounts and from
  mobile app builds by default (app-store rules), and switchable on via the web.

**Why not just one 4chan-style space for everyone?** Because Habit welcomes
13-year-olds, explicit and unfiltered adult content must be behind a real age
check. A self-declared birth year isn't enough. Without that, the UK Online
Safety Act and app-store and payment-processor rules would shut the product
down, and it would put kids at real risk. The two-layer setup keeps both
promises: *anything* can be discussed everywhere, and *anything legal*
goes in After Dark.

### 2.3 The Penthouse: high-tier membership & verified marketplace

**Idea:** like a Bloomberg Terminal, the price is part of the product. A
steep annual fee (around **$999/year**, with a monthly option around
$99/month) buys access to a space where **everyone else has paid the same**.
That filters out trolls, spam and bots, and makes the people inside worth
meeting.

**What members get**
- **The Penthouse floor**: the top floor of a tower in the Plaza. Its
  elevator door shows a live member count, but only members can ride it up.
  Inside are lounges, private meeting rooms and members-only Boards.
- **A high-signal network**: every member is a paying, age- and
  identity-verified adult (verified *by a provider*, not shown to others), so
  rooms are quieter, smarter and almost spam-free. Interest-based lounges
  (investing, collectibles, founders, creators) can grow out of it.
- **The Exchange**: a members-only marketplace for buying and selling real
  goods and services (below).
- **Perks**: Habit+ included, an exclusive Penthouse furni line, early
  access to features, and members-only events (AMAs, live auctions in the
  world).

**Anonymity in the Penthouse: "verified, not exposed"**
- Joining requires an **identity + age check** through a third-party provider
  (e.g. Stripe Identity, Persona or Veriff). This is the price of
  trust: members know everyone inside is a real, verified adult who has paid.
- Habit itself stores only the result (**verified: yes**, date, provider
  reference). Names, ID documents and selfies stay with the provider.
- In lounges, members can still chat as "Anonymous #id" or under their
  username. In **the Exchange**, sellers and buyers use a **stable trading
  handle** with public ratings, because trade needs a reputation that
  sticks.
- Members are **18+ only**. Teen accounts can't see the elevator.

**The Exchange: how buying and selling works**
1. **Seller onboarding**: sellers complete payout onboarding through a
   marketplace payments provider (e.g. **Stripe Connect**), which does the
   legally required identity checks, collects tax info and pays them out.
   Habit never holds anyone's money itself, which keeps us from becoming a
   money transmitter.
2. **Address verification** (needed for shipping and fraud prevention):
   - Shipping addresses are checked with an address-validation API (e.g.
     Smarty, Loqate, or the postal service's own) plus the card's address
     check at payment (AVS).
   - Sellers' addresses come from their payments-provider identity check.
   - Addresses are stored **encrypted, in a separate store** with strict
     staff access, and **shown to the other party only for an active order**.
     Buyers can use parcel lockers or PO boxes for extra privacy.
   - **Platform shipping labels** (via e.g. Shippo or EasyPost) give tracking
     on every order. With some carriers, the seller can ship without ever
     seeing the buyer's full address.
3. **Escrow-style payment**: the buyer pays at checkout. Funds are **held
   by the payments provider** until tracking shows delivery plus a short
   inspection window (e.g. 3 days), then released to the seller.
4. **Disputes**: "not as described" or "not received" claims go to Habit
   staff with chat logs and tracking. Refunds come out of held funds. Repeat
   bad actors lose Exchange access (and their membership, without refund,
   for fraud).
5. **Ratings**: both sides rate each trade. Ratings are attached to the
   trading handle.
6. **Fees**: a **5% fee** per sale (lower than eBay's or StockX's, a perk
   of membership), on top of payment processing.

**What can't be sold** (also required by payment providers): weapons and
parts, drugs and paraphernalia, alcohol and tobacco, prescription medicine,
counterfeits, stolen goods, adult content and services, personal data,
accounts, financial products and securities, gift cards and currency, live
animals, recalled or hazardous items, and **Habit Gems, Pips or items for
real money** (no real-money trading of the game economy). Listings are
checked by automated filters plus review for new sellers and high-value
items.

**Why not launch it early:** an expensive room is only valuable if it
already has interesting people in it. An empty Penthouse is worse than none.
So it arrives in **Phase 4**, once Habit has a large, active user base, and
starts with a **founding cohort**: invite-only, capped at ~250 members, and
priced as a "founding" rate locked in for life. That creates scarcity and
seeds the room with the right people.

**Guardrails so it doesn't wreck the rest of Habit**
- The Penthouse buys **access to a space and a marketplace**, never power:
  no ranking boost, no extra vote weight, no moderation power, no immunity
  from rules elsewhere.
- Members' Penthouse status is **hidden outside the Penthouse unless they
  choose** to show a badge, so the main world doesn't split into rich vs.
  poor.
- The main world stays free and fully featured. The Penthouse is a bonus,
  not a paywall over the core.

---

## 3. The MVP: "Walk in. Say hi." (weeks 0–12)

### 3.1 The one thing the MVP must prove
> **A first-time visitor can go from a shared link to sending their first
> message in under 60 seconds, and it feels less intimidating than joining
> a Discord server.**

If that holds, everything else in this plan can be built on top. If it
doesn't, no amount of threads, economy or mini-games will save it. So the MVP
is **only the walk-in chat world**. The forum and imageboard features move to
Phase 2.

This ordering also lowers launch risk: with **no user image uploads in the
MVP** (avatars and furni are our own art), the biggest legal and safety exposure
(illegal image uploads) doesn't exist until Phase 2, when the scanning pipeline
is built for it.

### 3.2 The MVP user journey (this is the spec)
1. **Click a link** (`habit.gg/r/retro-lounge`) or open the homepage → you're
   **in the Plaza as a guest within ~3 seconds**. No signup, no download.
   You get a random guest avatar.
2. **Look around.** Walk by clicking tiles. See signposts to Hubs, doors with
   live counts ("#general · 9 inside"). Hover a door to read the last few
   messages.
3. **Walk into a room.** You see avatars, speech bubbles and the chat log.
   Guests can read, walk, sit and wave. Nobody is notified that you arrived.
4. **Want to talk?** Click the chat box → a **one-step signup**: pick a
   **username + password**, enter your **birth year** (13+, see §6), and solve a quick
   invisible bot check. **No email, no phone number.** You stay in the same
   room. No reload and no lost place.
5. **Say hi.** Your message appears as a bubble and in the log. Your first
   visit badge tells regulars you're new. **Quick-start prompts** are offered
   ("👋 Hi!", "What's everyone up to?") so the first message takes one tap.
6. **Keep your account safe.** After your first message (not before, so it
   never slows down saying hi), a friendly nudge asks you to save your
   **recovery code**, since there's no email to reset a password with.
7. **Make it yours (lightly).** Pick from ~10 preset avatar looks and colours.
   Favourite the room so it's one click away next time.
8. **Come back.** The navigator shows your favourites with live counts.
   "3 people in #general now."

### 3.3 In scope for the MVP
| Area | MVP includes |
|---|---|
| World | The Plaza + 5–10 seeded Board Hubs, each with 2–4 chatrooms behind doors; one hand-made layout per room type |
| Movement | Click-to-walk, sit on furniture, 5 emotes (wave, dance, sit, laugh, heart) |
| Chat | Text chat as speech bubbles + side log, recent history on entering, emoji, @mentions within the room, quick-start prompts |
| Anonymity | Anonymous by default in every room ("Anonymous #id", per-room IDs resetting daily); opt-in to show your username; Board rooms set to anon-only / named-only / either; mod tools that work on anon IDs |
| Approachability | Guest mode (look but don't talk), door previews and live counts, quiet entry and exit, "first visit" badge, newcomers' corner in each Hub |
| Navigation | Room navigator: search, favourites, live counts, teleport-to-door; shareable room links |
| Accounts | Username + password only (no email/phone), recovery code, birth year (13+, sets teen or adult protections), privacy-friendly bot check, preset avatar picker, change password, delete account |
| Accessibility | List view (plain channel list and chat, no rendering); works on mobile browsers |
| Safety | Report, block and mute; room mods can kick, timeout and ban; word filter + toxicity scoring on chat; rate limits; new-account slow mode; staff admin panel; audit log |
| Teen safety | Teen protections on by default for 13–17 (§6.1): personal-info filter, grooming-risk detection, prioritised reports from teens, all-ages content rules everywhere |
| Platform | **Website first**: web only (desktop + mobile browser), no app install needed. Analytics for the funnel in §3.5. The mobile app follows once the website is proven (Phase 3) |

### 3.4 Explicitly NOT in the MVP
Threads/imageboard posting, image uploads, **After Dark (18+) and age
assurance**, voting, voice,
DMs, friends list, personal rooms, room building, furni inventory, currencies,
quests, mini-games, trading, user-created Boards, native mobile apps.
*(Each is planned below. Saying no to them now is what makes the MVP
shippable in ~12 weeks.)*

### 3.5 MVP success criteria (decide go / no-go on these)
| Metric | Target |
|---|---|
| Link click → in the world (guest) | ≥ 85%, median < 5 s |
| Guest → signs up to chat (first session) | ≥ 25% |
| Signed up → sends first message within 5 min | ≥ 70% |
| Median time from landing to first message | < 60 s for people who chat |
| D7 retention of people who sent a message | ≥ 25% |
| Survey: "easier to join than a Discord server" | ≥ 70% agree |
| Reports per 1k messages / median time-to-action | tracked; < 1 hr action time |

### 3.6 MVP timeline
| Weeks | Milestone |
|---|---|
| 0–2 | Decisions (art style; backend is Go), repo, CI, staging. **Prototype: Plaza + 2 rooms joined by a door, 2 avatars walking and chatting over WebSocket.** Playtest the feel with 10 people. |
| 3–6 | Real gateway (rooms, doors, presence, chat persistence), guest mode, one-step signup, navigator, avatar presets, first art pass |
| 7–9 | Safety tools (report, block, mute, mod kick/ban, filter, admin panel), teen protections (personal-info filter, grooming-risk detection, prioritised teen reports), list view, mobile-browser polish, analytics funnel |
| 10–11 | Closed alpha with 2–3 seed communities; fix whatever slows the time to first message |
| 12 | Public beta launch with 5–10 seeded Hubs + the launch event (§11) |

---

## 3A. After the MVP: feature roadmap

### Phase 2: "the noticeboard" (months 3–6): the forum/imageboard layer
- Boards get **threads**: the Hub noticeboard opens a thread view; Bump mode
  (imageboard) and Ranked mode (Reddit) per Board.
- **Image uploads**, launching *together with* the mandatory safety pipeline:
  CSAM hash matching (PhotoDNA / NCMEC hash lists or a vendor like Thorn
  Safer), NSFW classifier, thumbnailing and CDN.
- Anonymous-by-default posts with per-thread poster IDs; voting; mod queue.
- **After Dark (18+)**: age-assurance provider integration, basement/back-alley
  doors, After Dark Boards and rooms (§2.2).
- Thread objects in the Hub (noticeboard showing the top 5 threads, etc.).
- User-created Boards (by application at first).

### Phase 3: "make it yours" (months 6–9): friends, voice, rooms, economy
- Friends list, "follow friend" teleport, "friends are here" in the navigator.
- DMs (**off by default for minors**, opt-in only, with filters).
- Voice rooms (LiveKit) as walkable 🔊 rooms with proximity voice and
  "stage" tiles for talks.
- Full avatar editor (layered sprites), inventory, furni, **Pips** economy,
  daily quests ("visit 2 new Hubs", "say hi to a newcomer").
- Personal Rooms: place furni on an isometric grid, permissions (open / friends
  / locked / knock / password).
- Channel roles and permissions (locked doors, knock-to-enter, mod-only back
  rooms), searchable chat history, notifications.
- Monetization switches on: cosmetics (Gems) and Habit+ (§8).
- **Mobile app (iOS + Android)**, once the website has proven the "walk in,
  say hi" loop: wrap the web client with Capacitor first (one codebase,
  fast to ship), add push notifications ("3 people in #general now") and
  app-store safety requirements, then move hot paths native if needed.
  After Dark stays hidden in app builds by default (§2.2).

### Phase 4: "the game deepens" (months 9–12+)
- Mini-games inside rooms: trivia, drawing/guessing, simple racing, wired-style
  furni logic (triggers → actions, like Habbo "Wired").
- Trading marketplace for tradeable furni (in-game items for Pips/Gems only;
  escrow, anti-scam and anti-RMT rules).
- **The Penthouse (§2.3)**: founding cohort (~250 invite-only members),
  identity + age verification, the Penthouse floor, then **the Exchange**
  (Stripe Connect payouts, address validation, shipping labels, held funds,
  disputes, ratings) for the founding cohort first, then all members.
- Events: scheduled AMAs, game nights, "Bump Wars" (time-boxed thread contests).
- Seasonal events and a cosmetic-only battle pass.
- Board vs. Board leaderboards (creative contests, not raids).
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
| Realtime | **Go** WebSocket gateway (`coder/websocket`) | Presence, chat, avatar movement; Go handles many concurrent connections cheaply |
| Voice | **LiveKit** (self-hosted or cloud) WebRTC SFU | Proximity voice and voice rooms without building an SFU |
| API | **Go** (`net/http`), REST + an OpenAPI spec for typed clients | One language across API and gateway; decided |
| DB | **PostgreSQL** | Users, boards, posts, economy ledger (needs transactions) |
| Cache / pubsub | **Redis** | Sessions, rate limits, presence, room state fan-out |
| Search | **Meilisearch** or OpenSearch | Thread and Board search |
| Media | Object storage (S3/R2) + CDN + image worker (libvips) | Uploads, thumbnails, sprite sheets |
| Queue | Redis Streams or NATS | Media scanning, notifications, Pips payouts |
| Infra | Docker, a single region to start (Fly.io / Render / AWS ECS), Terraform later | Keep ops small early |
| Observability | OpenTelemetry, Sentry, Grafana | |

**Repo layout (current, grows as phases land)**
```
cmd/habit/          # server binary: world gateway + web client
internal/world/     # room layouts, doors, pathfinding (no runtime state)
internal/server/    # realtime gateway: sessions, movement, chat, accounts, filters
web/static/         # browser client (Canvas 2D for now; PixiJS when the art lands)
# later: internal/store (Postgres), internal/mod (moderation), cmd/worker,
#        infra/ (docker-compose, terraform), mobile/ (Capacitor shell)
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
users(id, username, password_hash, recovery_code_hash, created_at,
      age_band[teen_13_15|teen_16_17|adult], birth_year, adult_verified_at NULL,
      status, trust_score)          -- no email, no phone
sessions(id, user_id, created_at, last_seen_at, ip_hash, ua_hash)
banned_signals(kind[ip_hash|device_hash], value, expires_at)
avatars(user_id, layers_json, display_name)
boards(id, slug, title, mode[bump|ranked], anon_policy[anon_only|named_only|either],
       tier[main|after_dark], created_by)
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
memberships(user_id, tier[penthouse], status, started_at, renews_at, provider_sub_ref)
verifications(user_id, kind[identity_age|seller_kyc], result, provider, provider_ref,
              verified_at)                -- result only; documents stay with provider
trading_handles(user_id, handle, rating_avg, rating_count)
listings(id, seller_id, title, body, price_cents, currency, category, status, images)
orders(id, listing_id, buyer_id, seller_id, amount_cents, fee_cents, status
       [paid|shipped|delivered|released|disputed|refunded], provider_payment_ref,
       label_ref, tracking_no, release_at)
addresses(id, user_id, ciphertext, validated_at)  -- separate encrypted store
disputes(id, order_id, opened_by, reason, status, resolution)
trade_ratings(order_id, rater_id, ratee_id, stars, comment)
channels(id, board_id, kind[text|voice], name, perms_json)
messages(id, channel_id, author_id, is_anon, anon_hash, body, created_at)
```
**Economy rule:** balances are derived from the append-only `ledger` (with
cached totals). Every grant or spend is one DB transaction. This prevents
duplication exploits, the main way virtual economies get broken.

### 5.4 Anonymity implementation
- `anon_hash = HMAC(server_secret_rotated, user_id || thread_id)[:8]`: stable
  within a thread, unlinkable across threads to everyone except the server.
  For chatrooms: `HMAC(secret, user_id || room_id || date)`, so the ID is stable
  within a room for the day and changes daily.
- Anonymous messages are never shown on the poster's profile, and the API
  never returns `author_id` for them to anyone but audited staff tools.
- Mods see "anon #a1b2c3d4", with a "ban the poster" button that resolves
  server-side. Mods never see the real account (only admins/T&S do, and it's
  audited).

### 5.5 Accounts: username + password only
**Principle:** we never ask for an email address or phone number. You're
anonymous to other users, and we keep as little about you as we can.

- **Signup**: username (unique, filtered for slurs and impersonation of
  staff) + password. Minimum 10 characters, no annoying composition rules,
  checked against known-breached password lists (local list or
  k-anonymity lookup). Hashed with **argon2id**.
- **Recovery without email**: at signup we generate a **recovery code**
  (e.g. 4 groups of words). We store only its hash. Showing it is delayed
  until after the first message, then nudged again later ("save this or you
  could lose your account"). Using it lets you set a new password and issues
  a new code.
- **Optional extra security** (Phase 3, never required): passkeys and
  authenticator-app 2FA. Both work without email.
- **Lost both password and recovery code?** The account can't be recovered.
  This is the honest trade-off of no email, and the UI says so plainly.
  Staff never "recover" accounts on request, because that's how accounts get
  stolen.
- **Bot and spam defence without email or phone** (this matters, because free
  accounts are free for spammers too):
  - privacy-friendly bot check at signup (Cloudflare Turnstile, or a
    self-hosted proof-of-work challenge such as ALTCHA)
  - signup rate limits per IP / network
  - new-account slow mode and link and mention limits until an account has
    some history
  - trust earned over time and through good behaviour, not through
    verification
- **Ban enforcement**: bans apply to the account plus short-lived **hashed**
  IP/device signals, so a banned user can't instantly come back. Mods can
  also put a room into "established accounts only" mode during a raid.
- **Data minimisation**: no email, phone or real name stored. IP addresses
  are kept only as salted hashes with short retention (e.g. 30 days) for
  abuse prevention. Chat logs follow a published retention policy. Account
  deletion actually deletes.
- **Be honest in the privacy policy**: "anonymous" means *other users* can't
  identify you and *we* collect almost nothing. It doesn't mean
  untraceable. We still have to respond to valid legal requests with
  whatever little we hold. Don't promise more than that.
- **Payments later (Phase 3)**: buying Gems goes through a payment provider,
  which handles card details and receipts. We store only a payment
  reference against the account, never shown to anyone.
- **The one exception: the Penthouse (§2.3).** Members choose to verify
  identity and age, and Exchange users share shipping addresses. That data
  lives mainly with the verification and payments providers. What we do hold
  (encrypted addresses, verification results) sits in a separate, locked-down
  store. The privacy policy must say this clearly: *free Habit holds almost
  nothing about you; the Penthouse holds more, because trade requires it.*

---

## 6. Moderation & Trust and Safety (non-negotiable, built from day one)

Imageboard-style anonymity plus "talk about anything" plus images plus a
Habbo-style space popular with teens is the **highest-risk combination in
social software**. The answer isn't to give up on anonymity or open speech.
It's to enforce the hard lines (§2.2) very well and to keep teens out of
After Dark. This section is
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
   - New accounts: chat and posting rate limits, no links or images for the
     first X hours, bot check at signup and on anon posts.
   - Trust score built from account age, time active, messages that are not
     reported, mod actions and report accuracy. **Not** from email or phone,
     which we don't collect.
3. **Teen safety**: Habit is open to everyone **13 and up**. See §6.1.
4. **Transparency**: public content policy, mod log per Board, appeals,
   transparency reports (required under the EU DSA at scale).
5. **Anti-raid**: cross-Board brigade detection (vote/visit spikes from an
   outside referrer), Board "lockdown" mode, no organized "raid" features.

### 6.1 Teens on Habit (13+)

Habit is for everyone **13 and older**, like Habbo. Teens are welcome. Combined
with no email or phone, that means **we can never be sure who is an adult and
who is a teen**. So the rule is: **the whole platform is built to be safe for a
13-year-old**, with extra protections layered on for accounts that say they
are under 18.

**Age gate**
- Signup asks for **birth year** in a neutral way (no pre-filled default and no
  "I'm over 18" checkbox to click through).
- Under 13 → signup is refused, kindly, and the browser is blocked from retrying
  straight away with a different year. This keeps us out of COPPA's
  parental-consent rules, which can't be met without contact details anyway.
- We store the **age band** (13–15, 16–17, adult) and birth year only, so bands
  move up automatically. Age can be **raised** only by contacting support with
  an explanation (rarely granted), but **lowered** any time, so someone who
  realises they mis-entered as an adult can fix it instantly.

**Rules for everyone (because anyone might be a teen)**
- **The main world is all-ages content** (any topic, but no explicit sexual
  content or gore; §2.2). Adult content lives only in **After Dark**, which
  needs a real age check, not a typed birth year, and teen accounts can never
  enter. (Phase 2 image uploads are scanned and held to the same rules.)
- **Anonymity doesn't switch off protections.** Other people can't see who
  an anonymous user is, but the server always knows the account's age band.
  So the personal-info filter, grooming-risk detection and teen-report
  priority all work the same in anonymous rooms.
- **Everything happens in public rooms in the MVP**: no DMs, no private rooms.
  Public rooms are the safest place to chat because others, mods and filters
  can all see what's said.
- Chat filters and toxicity scoring on for every room; room mods everywhere.

**Extra protections for teen accounts (13–17), on by default**
- **Personal-info filter**: blocks teens from posting phone numbers, addresses,
  school names, emails and "add me on [other app]" handles, and warns them why.
  This is the classic Habbo protection, and it matters most here.
- **Grooming-risk detection**: classifiers watch for known patterns (asking for
  age, location, photos or secrecy, pushing to move to another app) in messages
  *sent to or near* teen accounts. Hits go to the top of the moderator queue,
  and the sender gets throttled until reviewed.
- **Reports from teens are prioritised**, with a one-tap "this person is making
  me uncomfortable" option that also hides that person from them.
- **Less visibility**: teen profiles show no birth year or age band to others,
  aren't listed in "who's online" searches by name, and can't be followed or
  teleported to by anyone who isn't their friend.
- **Later phases**, when the features arrive:
  - **DMs**: teens can only DM accepted friends. Adults can't send friend
    requests to accounts aged 13–15. Adult↔teen DMs are scanned with the
    grooming-risk models.
  - **Personal rooms**: teens' rooms default to friends-only. Teens can't enter
    adults' locked or password rooms.
  - **Voice**: off by default for 13–15; proximity voice only in public rooms.
  - **Trading and payments**: spending limits for teen accounts, and no trading
    between adults and 13–15s.
- **Age-band rooms (optional, later)**: Boards can create "13–17 only"
  rooms (a "teen lounge"). Self-declared adults can't walk in, though, without
  verification, a determined liar can. So these rooms get extra mod attention,
  not less.

**People and process**
- Moderators on shift during teen peak hours (after school and evenings, by
  launch time zones), with child-safety training.
- A clear escalation path: grooming or exploitation reports go to trained
  staff within minutes, with **NCMEC CyberTipline** reports (US) and law
  enforcement referral when required.
- A **children's risk assessment** before launch (UK OSA requires one), redone
  when new features ship.
- Optional age assurance later (privacy-preserving facial age estimation
  or an ID check through a third party) for anyone who *wants* to unlock
  adult-only features. It's never required to use Habit.

---

## 7. Legal & compliance checklist (talk to a lawyer before launch)
- Terms of Service, Privacy Policy, Content Policy, Community Guidelines.
- **US**: Section 230 protections, DMCA agent registration plus a takedown
  flow, NCMEC CyberTipline reporting obligations, **COPPA** (under-13s are barred
  at signup, see §6.1).
- **EU**: GDPR (data export/delete, lawful basis), **Digital Services Act**
  (notice-and-action, statements of reasons, transparency).
- **UK**: Online Safety Act (illegal content and **children's risk
  assessment**; Habit is "likely to be accessed by children", so the
  children's safety duties apply) and the ICO **Age Appropriate Design Code**
  (high-privacy defaults for under-18s, which §6.1 is designed to meet).
  After Dark's adult content means **highly effective age assurance** is
  required for it (hence a third-party check, not self-declaration). Keep
  pornographic images out entirely.
- **EU**: DSA Article 28 (appropriate measures to protect minors; no
  profiling-based ads to minors).
- **US states**: several states have passed minors' social-media laws
  (age verification / parental consent), many under legal challenge. Get
  advice on which launch states to include.
- **Open speech**: Section 230 (US) protects hosting users' speech. EU/UK
  "illegal content" duties still apply, so we need a notice-and-action flow
  and country-specific legal takedowns (some speech is illegal in Germany or
  France but not the US). Geo-restrict rather than delete worldwide where
  possible.
- **No-email accounts + teens**: confirm with a lawyer that self-declared
  birth year plus the §6.1 protections is enough for launch markets, and
  that the data-minimisation and retention policy fits law-enforcement
  request obligations.
- **The Penthouse & Exchange**:
  - **Marketplace seller laws**: the US **INFORM Consumers Act** (verify
    high-volume sellers and, above set thresholds, show buyers their name
    and contact details), EU **DAC7** (report seller income to tax
    authorities), and similar UK rules. Stripe Connect handles much of the
    verification and reporting.
  - **Sales tax / VAT**: many US states treat marketplaces as "marketplace
    facilitators" that must collect sales tax. EU/UK VAT rules work the same
    way for some sales. Use a tax engine (e.g. Stripe Tax, TaxJar).
  - **Consumer protection**: refund/returns policy, clear listing rules, and
    EU platform-to-business transparency for sellers.
  - **AML / payments**: never hold user funds ourselves; let the payments
    provider hold and release them. Keep the prohibited-items list in line
    with the provider's rules.
  - **High-value subscription**: clear renewal terms, cancellation and
    refund policy (auto-renewal laws in many US states and the EU).
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
4. **The Penthouse** (Phase 4): ~$999/year (or ~$99/month), founding cohort
   first. Even 1,000 members is ~$1M/year in revenue, with almost no extra
   infrastructure cost.
5. **Exchange fees**: 5% of each sale in the Penthouse marketplace.
6. **Creator revenue share** (later): mods/artists design furni and get a
   cut of sales.
7. **No** pay-for-visibility on posts, and **no ads**: advertisers avoid
   anonymous, open-speech platforms (4chan's own history shows this), so the
   business shouldn't depend on them. Cosmetics and subscriptions don't
   care what people talk about.
8. Pick payment providers that accept open-speech UGC platforms, and keep
   After Dark free of pornographic content, so payments aren't at risk.

---

## 9. Metrics that matter
- **North star: weekly chatters**, meaning people who sent at least one
  message that week. It directly measures "approachable enough to talk".
- **Activation funnel**: link → in world → entered a room → signed up →
  first message (targets in §3.5). Time to first message is the number
  everyone on the team should know.
- **Retention**: D1 / D7 / D30 of people who chatted; favourited rooms per user.
- **Health**: reports per 1k posts, time to action on reports, % of content
  actioned automatically vs. by humans, repeat-offender rate.
- **Economy**: Pips inflation (earned vs. sunk), Gem conversion rate,
  ARPPU.
- **Community**: number of Boards with ≥1 active mod and ≥20 weekly posters.
- **Penthouse**: members, renewal rate (the real test of whether the network
  is worth it), weekly active members, Exchange sales volume, dispute rate,
  fraud losses.

---

## 10. Team & budget (rough, lean start)
| Role | MVP (first 12 weeks) |
|---|---|
| Full-stack engineers | 2 |
| Game/realtime engineer (PixiJS, WebSockets) | 1 |
| Pixel artist / designer | 1 (contract OK) |
| Trust & Safety lead / community manager | 1, with child-safety experience (**hire before the closed alpha**, since teens are allowed) |
| Part-time moderators | 2–4 to cover teen peak hours (can start as trained volunteers + paid lead) |
| Founder/PM | 1 |

Infra at MVP scale: roughly $300–$1.5k/month (DB, Redis, CDN, LiveKit,
moderation APIs). Moderation vendors and CSAM scanning are often free or
discounted for small platforms; apply early.

---

## 11. Launch strategy
1. **Pre-launch (from week 0)**: username claiming (early account creation,
   no email) + "Founding Visitor" item; post dev-log clips of the world to short-form video.
2. **Closed alpha (weeks 10–11)**: 2–3 small communities with existing
   culture (retro games, pixel art, a streamer's regulars). Recruit their mods
   first, and watch new visitors' first 60 seconds closely.
3. **Seed the world**: 5–10 Hubs, each with a host community and a few
   **greeters** online at peak hours, so no early visitor walks into an
   empty room.
4. **Launch event (week 12)**: a scheduled opening party in the Plaza with
   founding-member furni (costs nothing, strong reason to show up). Creators
   share their room links on stream.
5. **Open beta**: guest access open to everyone, with all safety systems
   tested under load (run a red-team week of trying to break moderation first).

---

## 12. Top risks & mitigations
| Risk | Mitigation |
|---|---|
| Becomes a haven for illegal or extremist content | Hard lines enforced fast and consistently (§2.2); hash scanning; account-backed anonymity so bans stick; paid T&S; "moderate behaviour, not opinions" applied evenly |
| Open speech scares off seed communities, app stores or payment providers | Main world stays all-ages and welcoming; After Dark is age-gated, web-only by default and off the marketing; no ads in the model; no pornographic images |
| After Dark turns hostile and leaks into the main world | Separate Boards and rooms, no cross-posting into the main world, anti-raid tools, harassment of people is a hard line even in After Dark |
| Child safety incident (teens + anonymity: the top risk) | Whole platform safe for 13-year-olds by default; public rooms only in the MVP; personal-info filter and grooming-risk detection; prioritised teen reports; trained mods on shift at teen peak hours; children's risk assessment before launch (§6.1) |
| Empty world ("ghost town") | Launch small and dense (few Hubs, not many); greeters at peak hours; show *recent* chat on doors so rooms feel alive even when quiet; scheduled events; later, threads on the noticeboard |
| Walking feels slow or gimmicky | Navigator teleport and list view always available; playtest the time to first message weekly; cut anything that adds steps before saying hi |
| Spam and ban evasion (accounts are free, no email or phone) | Bot check, IP/network signup limits, new-account slow mode, hashed IP/device bans, trust earned over time, "established accounts only" room mode |
| Users lose access (no email reset) | Recovery code nudged after first message and again later; optional passkey/2FA later; clear warning at signup |
| Economy exploits / duping | Append-only ledger, server-authoritative everything, rate caps, anomaly alerts |
| Scope creep (it's 4 products in one) | The MVP is walk-in chat only (§3.4 lists what's out); forum next, then voice/economy, mini-games last |
| The Penthouse feels empty or not worth $1k | Launch late (Phase 4) with a hand-picked, capped founding cohort; members-only events; track renewal rate and adjust price/perks before scaling |
| Exchange fraud, scams or prohibited goods | Seller KYC via payments provider, held funds released only after delivery, tracked shipping labels, ratings, dispute process, prohibited-items list with automated + manual listing checks |
| Leak of verification or address data | Keep documents with providers; encrypt addresses in a separate store; strict, audited staff access; show addresses only during active orders |
| Penthouse splits the community into rich vs. poor | Buys access, never power; status hidden outside the Penthouse by default; main world stays free and complete |
| Realtime scaling costs | Room caps and instancing; low tick rate; voice only on demand |

---

## 13. Immediate next steps (this repo)
1. Backend language: **Go** (decided). Age policy is **13+ with
   teen protections** (§6.1). Get a lawyer to review it and do the children's
   risk assessment before the closed alpha.
2. ~~Build the week 0–2 prototype~~ **Done** (Go, see README): the Plaza plus
   two chatrooms joined by doors, server-authoritative walking over WebSocket,
   chat bubbles + side log, door peeking and live counts, guest mode,
   username + password signup, anonymous-by-default names, teen
   personal-info filter, list view.
3. Next for the website (weeks 3–6): Postgres for accounts and chat history,
   recovery-code redemption, report/block/mute, mod tools, real pixel art.
4. Playtest it with 10 people who say Discord intimidates them; time their
   link → first message, and iterate.
5. Put up a landing page where people can claim a username (no email), and
   start posting dev-log clips.
6. Draft the Content Policy now. Pick a CSAM-scanning vendor before Phase 2
   (image uploads).
