# Habit demo: alpha 0.5

A scripted, hands-free tour of Habit for recording a demo video (about 90
seconds). It's separate from the app: it doesn't change any of Habit's
code, and it runs its own private copy of Habit (in memory, on port 8090),
so it never touches your real database or your running server.

What it plays:

1. A title card: **Habit · alpha 0.5 · Walk in. Say hi.**
2. **The Plaza:** you walk in, wave, make a wish in the fountain, chat and
   shake a cherry tree, while other characters chat, dance and grab hot dogs.
3. **Gaming:** you play an arcade cabinet for a score and sit on a beanbag;
   others lounge on the couch, play and grab sodas.
4. **The Gym:** you hit the punching bag and grab some water; others lift
   weights and talk about leg day.
5. An end card.

Captions explain each part. The other characters (mia, leo, zoe, sam and
kai) are lightweight scripted clients, not extra windows.

## You need

- **Go** and **Google Chrome** (the same as for running Habit)
- **Node.js** 18 or newer: https://nodejs.org

## Run it

Double-click **`run-demo.bat`** (Windows), or in this folder:

```sh
npm install        # once
node demo.mjs
```

Chrome opens full screen on the title card, and the tour starts after 10
seconds. That's your cue to start recording:

- **Windows:** press **Win + Alt + R** (Xbox Game Bar) to start, and again
  to stop. Videos land in `Videos\Captures`.
- **Any OS:** OBS Studio (https://obsproject.com) gives the smoothest result.
  Record at 1920×1080, 30 or 60 fps.

Options:

| Command | What it does |
| --- | --- |
| `node demo.mjs --now` | start the tour straight away |
| `node demo.mjs --wait 20` | wait 20 seconds before starting |
| `node demo.mjs --windowed` | play in a normal window, not full screen |
| `node demo.mjs --check` | play invisibly and save a screenshot per caption to `.bin/shots` (to check it still works) |

Set `DEMO_PORT` to use a port other than 8090. Press **Ctrl+C** in the
terminal to stop early.

## Changing the tour

Everything is in `tour()` in `demo.mjs`: `me.walk(x, y)`, `me.use("prop-id")`,
`me.say("...")`, `me.go("room")` and `me.caption("...")` for you, and the same
on each character (`mia.say(...)`, etc.), with `later(ms, ...)` to make things
happen while you're busy. Prop IDs and tile positions are in
`internal/world/rooms.go` (the Plaza) and `internal/world/blocks.go` (Gaming
and the Gym).

For LinkedIn: a 16:9 MP4 up to 10 minutes works; around a minute and a half
keeps people watching. Add a line about what you built and the stack (Go,
Postgres, WebSockets, a hand-rolled isometric voxel renderer on canvas).
