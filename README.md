# Habit

**Walk in. Say hi.**

Habit is a cute, cosy virtual world where every chatroom is a real room. You can see
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
- **Developer handoff brief:** [docs/MVP-BRIEF.md](./docs/MVP-BRIEF.md)
- **Full product & technical plan:** [PLAN.md](./PLAN.md)

## Keep a local copy that updates itself

One-time setup:
1. Install **Git** (https://git-scm.com/downloads) and **Go**
   (https://go.dev/dl/).
2. Open a terminal (on Windows, "Git Bash" or "Command Prompt") in the folder
   where you want Habit, and run:
   ```sh
   git clone -b claude/hybrid-platform-concept-plan-l0im18 https://github.com/weeef/Habit.git habit
   ```
   This creates a `habit` folder linked to the GitHub repo.

From then on, double-click the start script in that folder:
- **Windows:** `start-windows.bat`
- **Mac:** `start-mac-linux.command` (the first time, right-click → Open)
- **Linux:** run `./start-mac-linux.command`

Each time it pulls the latest version from GitHub, then starts Habit and opens
http://localhost:8080. To update the folder without starting Habit, run
`git pull` inside it.

## Run the prototype

You need Go 1.26 or newer.

```sh
go run ./cmd/habit            # then open http://localhost:8080
go run ./cmd/habit -addr :3000
go test ./...
```

Open the page in two browser windows to see two people in the same world.

**Saving data.** Without a database, accounts and chat live in memory and are
lost on restart. To keep them, point Habit at a Postgres database (it creates
its tables itself):

```sh
go run ./cmd/habit -db postgres://user:pass@localhost:5432/habit
# or set DATABASE_URL instead of passing -db
```

To run the database tests too, set `HABIT_TEST_DATABASE_URL` to a throwaway
database (the tests empty it) before `go test ./...`.

**What works now (website, week 0–2 prototype):**
- The Plaza with doors to two communities, **Gaming** and **Gym**, each with
  its own look. Click the floor to walk; walk into a door to enter.
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

- Accounts, logins and chat history saved to Postgres when a database is set.
- Cute voxel characters that turn to face where they walk and look at
  whoever's talking. Design yours when you sign up (skin, hair, outfit,
  accessory).
- **The Plaza** is a little New York: skyscrapers, neon signs, taxis on
  the street, food carts (grab a hot dog), hydrants, newspaper boxes,
  steaming manholes and a subway entrance that rides you to any room. Toss
  a coin in the fountain, sit on a bench, shake a cherry tree, ring the
  hello bell, scatter the pigeons. It has a 24-minute day/night cycle; the
  skyline lights up at dusk.
- Every room has a **noticeboard** on the wall. Click it and the board
  opens in the middle of the screen: text threads and replies that stay
  forever, 👍/👎 votes, and filters (Latest, New, Hot, Top, Most
  downvoted). Anonymous by default (with an "OP" mark), plus a 🔔
  notification when someone replies in a thread you posted in.

**Not yet:** recovery codes can't be redeemed yet, and there's no report/block/mute or
moderator tools. Those are next (PLAN.md §3.6, weeks 3–9). The mobile app
comes after the website (PLAN.md Phase 3).
