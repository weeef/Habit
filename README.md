# Habit

**Walk in. Say hi.**

Habit is a pixel-art world where every chatroom is a real room. You can see
who's inside before you go in, hang out without having to talk, and join the
conversation when you feel like it. It mixes Discord-style chat,
Reddit/imageboard-style communities and a Habbo-style virtual world, and it's
built to feel far less intimidating than joining a server. No email or phone
number needed, just a username and password.

Like an imageboard, you're **anonymous by default** and can talk about
genuinely anything. The main world is open to everyone 13+, and an
age-checked 18+ "After Dark" area allows anything legal.

- **MVP:** walk-in chat world. See [PLAN.md §3](./PLAN.md#3-the-mvp-walk-in-say-hi-weeks-012)
- **Positioning & marketing:** see [PLAN.md §0](./PLAN.md#0-positioning--marketing)
- **The Penthouse (premium tier + marketplace):** see [PLAN.md §2.3](./PLAN.md#23-the-penthouse-high-tier-membership--verified-marketplace)
- **Full product & technical plan:** [PLAN.md](./PLAN.md)

## Run the prototype

You need Go 1.24 or newer.

```sh
go run ./cmd/habit            # then open http://localhost:8080
go run ./cmd/habit -addr :3000
go test ./...
```

Open the page in two browser windows to see two people in the same world.

**What works now (website, week 0–2 prototype):**
- The Plaza plus two chatrooms (`#general`, `#lounge`) joined by doors. Click
  the floor to walk; walk into a door to enter that room.
- Hover a door to peek: live head-count and the last few messages.
- Room navigator with live counts; click to teleport.
- Look around as a guest with no account. Typing opens a one-step signup
  (username + password + birth year; no email or phone).
- Anonymous by default ("Anonymous #3f9a", different per room and day), with
  an opt-in to show your username.
- Speech bubbles plus a side chat log, quick-start prompts, emotes, a "new"
  badge for first-time visitors, and quiet entry and exit.
- Recovery code shown after your first message.
- Teen (13–17) personal-info filter; under-13s can't sign up.
- List view (plain chat, no world rendering) and a phone-friendly layout.

**Not yet:** everything is in memory (a restart forgets accounts and chat),
recovery codes can't be redeemed yet, and there's no report/block/mute or
moderator tools. Those are next (PLAN.md §3.6, weeks 3–9). The mobile app
comes after the website (PLAN.md Phase 3).
