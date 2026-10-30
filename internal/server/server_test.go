package server

import (
	"context"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"

	"habit/internal/store"
	"habit/internal/world"
)

type client struct {
	t     *testing.T
	conn  *websocket.Conn
	first map[string]any // the "room" message sent on connect
}

func startServer(t *testing.T) string {
	t.Helper()
	return startServerWith(t, store.NewMemory())
}

func startServerWith(t *testing.T, st store.Store) string {
	t.Helper()
	sv, err := New(context.Background(), Options{Store: st, Tick: 2 * time.Millisecond, Step: 2 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go sv.Run(ctx)
	ts := httptest.NewServer(sv.Handler())
	t.Cleanup(ts.Close)
	return "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws"
}

func dial(t *testing.T, url string) *client {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.CloseNow() })
	c := &client{t: t, conn: conn}
	c.first = c.expect("room")
	return c
}

func (c *client) send(v any) {
	c.t.Helper()
	if err := wsjson.Write(context.Background(), c.conn, v); err != nil {
		c.t.Fatal(err)
	}
}

// expect reads until a message of type typ arrives, skipping others.
func (c *client) expect(typ string) map[string]any {
	c.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for {
		var m map[string]any
		if err := wsjson.Read(ctx, c.conn, &m); err != nil {
			c.t.Fatalf("waiting for %q: %v", typ, err)
		}
		if m["t"] == typ {
			return m
		}
	}
}

func TestGuestMustSignUpToChat(t *testing.T) {
	url := startServer(t)
	a := dial(t, url)
	b := dial(t, url)

	a.send(map[string]any{"t": "chat", "text": "hello?"})
	a.expect("need_account")

	a.send(map[string]any{"t": "signup", "username": "walker", "password": "long enough pw", "birthYear": 1990})
	authed := a.expect("authed")
	if authed["recoveryCode"] == "" || authed["token"] == "" {
		t.Fatalf("signup should return a token and recovery code: %v", authed)
	}

	a.send(map[string]any{"t": "chat", "text": "hi all"})
	msg := b.expect("chat")["msg"].(map[string]any)
	if msg["text"] != "hi all" {
		t.Fatalf("unexpected chat: %v", msg)
	}
	name := msg["name"].(string)
	if !strings.HasPrefix(name, "Anonymous #") {
		t.Fatalf("chat should be anonymous by default, got name %q", name)
	}
	a.expect("first_message")

	a.send(map[string]any{"t": "named", "on": true})
	upd := b.expect("update")["avatar"].(map[string]any)
	if upd["name"] != "walker" {
		t.Fatalf("opting in should show the username, got %v", upd["name"])
	}
}

func TestTeenPersonalInfoFilter(t *testing.T) {
	url := startServer(t)
	a := dial(t, url)
	a.send(map[string]any{"t": "signup", "username": "teen_user", "password": "long enough pw", "birthYear": 2011})
	if band := a.expect("authed")["ageBand"]; band != BandTeen13to15 {
		t.Fatalf("want teen band, got %v", band)
	}
	a.send(map[string]any{"t": "chat", "text": "add me on snap"})
	notice := a.expect("notice")["message"].(string)
	if !strings.Contains(notice, "teen accounts") {
		t.Fatalf("want teen safety notice, got %q", notice)
	}
}

func TestWalkThroughDoor(t *testing.T) {
	url := startServer(t)
	a := dial(t, url)
	watcher := dial(t, url)

	// The Plaza's door to Gaming is at (0, 10).
	a.send(map[string]any{"t": "walk", "x": 0, "y": 10})
	entered := a.expect("room")
	room := entered["room"].(map[string]any)
	if room["id"] != "gaming" {
		t.Fatalf("want to arrive in gaming, got %v", room["id"])
	}
	watcher.expect("leave")

	// Counts update for everyone, so doors show who's inside.
	for {
		counts := watcher.expect("counts")["counts"].(map[string]any)
		if counts["gaming"] == float64(1) && counts["plaza"] == float64(1) {
			break
		}
	}

	// Peeking at a room you aren't in shows its recent chat: an empty list,
	// never null, when nobody has spoken (null crashed the client).
	watcher.send(map[string]any{"t": "peek", "room": "gaming"})
	peek := watcher.expect("peek")
	if peek["count"] != float64(1) {
		t.Fatalf("peek should count 1 person in gaming, got %v", peek["count"])
	}
	if recent, ok := peek["recent"].([]any); !ok || len(recent) != 0 {
		t.Fatalf("want recent = [], got %#v", peek["recent"])
	}
	if history, ok := entered["history"].([]any); !ok || len(history) != 0 {
		t.Fatalf("want history = [] in a quiet room, got %#v", entered["history"])
	}

	// And back again through gaming's door to the Plaza, at (0, 4).
	a.send(map[string]any{"t": "walk", "x": 0, "y": 4})
	if back := a.expect("room")["room"].(map[string]any); back["id"] != "plaza" {
		t.Fatalf("want to arrive back in the plaza, got %v", back["id"])
	}
}

func TestTeleport(t *testing.T) {
	url := startServer(t)
	a := dial(t, url)
	a.send(map[string]any{"t": "teleport", "room": "gym"})
	if id := a.expect("room")["room"].(map[string]any)["id"]; id != "gym" {
		t.Fatalf("want gym, got %v", id)
	}
}

func TestResumeSession(t *testing.T) {
	url := startServer(t)
	a := dial(t, url)
	a.send(map[string]any{"t": "signup", "username": "comeback", "password": "long enough pw", "birthYear": 1990})
	token := a.expect("authed")["token"].(string)

	b := dial(t, url)
	b.send(map[string]any{"t": "resume", "token": token})
	if got := b.expect("authed")["username"]; got != "comeback" {
		t.Fatalf("want resumed session, got %v", got)
	}
}

func signup(c *client, username string, birthYear int) string {
	c.t.Helper()
	c.send(map[string]any{"t": "signup", "username": username, "password": "long enough pw", "birthYear": birthYear})
	return c.expect("authed")["token"].(string)
}

func TestBoardPostReplyAndNotify(t *testing.T) {
	url := startServer(t)
	op, other, guest := dial(t, url), dial(t, url), dial(t, url)
	opToken := signup(op, "poster", 1990)
	signup(other, "replier", 1990)

	guest.send(map[string]any{"t": "post_thread", "title": "hi", "text": "guest post"})
	guest.expect("need_account")

	op.send(map[string]any{"t": "post_thread", "title": "Best co-op games?", "text": "Looking for something to play with friends."})
	thread := op.expect("posted_thread")["thread"].(map[string]any)
	if !strings.HasPrefix(thread["name"].(string), "Anonymous #") || thread["room"] != "plaza" {
		t.Fatalf("thread should be anonymous and on the plaza board, got %v", thread)
	}
	id := thread["id"]
	// (The day's question may go up on the board too; skip past it.)
	for {
		if upd := other.expect("board_update")["thread"].(map[string]any); upd["id"] == id {
			break
		}
	}

	// Guests can read boards and threads.
	guest.send(map[string]any{"t": "board", "room": "plaza"})
	found := false
	for _, th := range guest.expect("board")["threads"].([]any) {
		found = found || th.(map[string]any)["id"] == id
	}
	if !found {
		t.Fatalf("guests should see the new thread on the board")
	}

	other.send(map[string]any{"t": "thread", "thread": id})
	other.expect("thread")
	other.send(map[string]any{"t": "post_reply", "thread": id, "text": "Deep Rock Galactic!"})
	reply := other.expect("reply")["reply"].(map[string]any)
	if reply["op"] != false || reply["name"] == thread["name"] {
		t.Fatalf("a reply from someone else shouldn't look like OP: %v", reply)
	}

	note := op.expect("notify")["note"].(map[string]any)
	if note["title"] != "Best co-op games?" || note["snippet"] != "Deep Rock Galactic!" || note["read"] != false {
		t.Fatalf("OP should be notified of the reply, got %v", note)
	}

	// OP replying in their own thread is marked OP and keeps their tag.
	op.send(map[string]any{"t": "thread", "thread": id})
	op.expect("thread")
	op.send(map[string]any{"t": "post_reply", "thread": id, "text": "Ooh, thanks"})
	if r := op.expect("reply")["reply"].(map[string]any); r["op"] != true || r["name"] != thread["name"] {
		t.Fatalf("OP's reply should be marked OP with the same tag, got %v", r)
	}
	other.expect("notify")

	// Notifications wait for you when you come back.
	later := dial(t, url)
	later.send(map[string]any{"t": "resume", "token": opToken})
	later.expect("authed")
	notes := later.expect("notes")["notes"].([]any)
	if len(notes) != 1 {
		t.Fatalf("want 1 saved notification, got %v", notes)
	}
	later.send(map[string]any{"t": "notes_read"})
	again := dial(t, url)
	again.send(map[string]any{"t": "resume", "token": opToken})
	if n := again.expect("notes")["notes"].([]any)[0].(map[string]any); n["read"] != true {
		t.Fatalf("notification should be read now, got %v", n)
	}
}

func TestBoardTeenFilterAndLimits(t *testing.T) {
	url := startServer(t)
	teen := dial(t, url)
	signup(teen, "young_one", 2011)
	teen.send(map[string]any{"t": "post_thread", "title": "add me on snap", "text": "pls"})
	if msg := teen.expect("post_error")["message"].(string); !strings.Contains(msg, "teen accounts") {
		t.Fatalf("want teen safety error, got %q", msg)
	}

	adult := dial(t, url)
	signup(adult, "grown_up", 1990)
	adult.send(map[string]any{"t": "post_thread", "title": "", "text": "no title"})
	adult.expect("post_error")
	for range threadBurst {
		adult.send(map[string]any{"t": "post_thread", "title": "t", "text": "x"})
		adult.expect("posted_thread")
	}
	adult.send(map[string]any{"t": "post_thread", "title": "t", "text": "x"})
	if msg := adult.expect("post_error")["message"].(string); !strings.Contains(msg, "posting a lot") {
		t.Fatalf("want rate limit, got %q", msg)
	}
}

func TestRestartKeepsAccountsAndChat(t *testing.T) {
	st := store.NewMemory()
	a := dial(t, startServerWith(t, st))
	a.send(map[string]any{"t": "signup", "username": "stayer", "password": "long enough pw", "birthYear": 1990})
	token := a.expect("authed")["token"].(string)
	a.send(map[string]any{"t": "chat", "text": "still here?"})
	before := a.expect("chat")["msg"].(map[string]any)
	// A connection's messages are handled in order, so once this peek is
	// answered the chat has been saved.
	a.send(map[string]any{"t": "peek", "room": "plaza"})
	a.expect("peek")

	// A second server on the same store is a restart.
	url := startServerWith(t, st)
	b := dial(t, url)
	history := b.first["history"].([]any)
	if len(history) != 1 || history[0].(map[string]any)["text"] != "still here?" {
		t.Fatalf("want saved chat on entering, got %v", history)
	}

	b.send(map[string]any{"t": "resume", "token": token})
	if got := b.expect("authed")["username"]; got != "stayer" {
		t.Fatalf("token should survive a restart, got %v", got)
	}
	b.send(map[string]any{"t": "chat", "text": "yes"})
	if after := b.expect("chat")["msg"].(map[string]any); after["name"] != before["name"] {
		t.Fatalf("anonymous tag should survive a restart: %v then %v", before["name"], after["name"])
	}

	c := dial(t, url)
	c.send(map[string]any{"t": "login", "username": "stayer", "password": "long enough pw"})
	c.expect("authed")
}

func TestFountainFromAcrossThePlaza(t *testing.T) {
	url := startServer(t)
	a, watcher := dial(t, url), dial(t, url)
	// a arrives at the spawn on the south path, out of reach of the
	// fountain: using it walks a over first, then tosses the coin.
	a.send(map[string]any{"t": "use", "prop": "fountain"})
	fx := watcher.expect("fx")
	if fx["kind"] != "fountain" || fx["count"] != float64(1) {
		t.Fatalf("want a coin in the fountain, got %v", fx)
	}
	a.send(map[string]any{"t": "use", "prop": "fountain"})
	if fx := watcher.expect("fx"); fx["count"] != float64(2) {
		t.Fatalf("second wish should count 2, got %v", fx["count"])
	}
	// Someone arriving later sees the count on the fountain.
	late := dial(t, url)
	for _, p := range late.first["room"].(map[string]any)["props"].([]any) {
		if p := p.(map[string]any); p["id"] == "fountain" && p["count"] != float64(2) {
			t.Fatalf("want 2 wishes on the fountain, got %v", p["count"])
		}
	}
}

func TestArcadeFurniture(t *testing.T) {
	url := startServer(t)
	a := dial(t, url)
	a.send(map[string]any{"t": "teleport", "room": "gaming"})
	a.expect("room")
	// Playing a cabinet walks you over and shows everyone your score.
	a.send(map[string]any{"t": "use", "prop": "cabinet-2"})
	fx := a.expect("fx")
	if score, _ := fx["score"].(float64); fx["kind"] != "cabinet" || score < 100 || score > 9990 {
		t.Fatalf("want a cabinet score, got %v", fx)
	}
	// The screen is only for watching; the vending machine hands out a soda.
	a.send(map[string]any{"t": "use", "prop": "screen"})
	a.send(map[string]any{"t": "use", "prop": "vending"})
	if fx := a.expect("fx"); fx["kind"] != "vending" || fx["variant"] != "soda" {
		t.Fatalf("want a soda from the vending machine, got %v", fx)
	}
}

func TestSitOnBenchAndStandUp(t *testing.T) {
	url := startServer(t)
	a, watcher := dial(t, url), dial(t, url)
	a.send(map[string]any{"t": "use", "prop": "bench-s"})
	for {
		if av := watcher.expect("update")["avatar"].(map[string]any); av["seat"] == "bench-s" {
			break
		}
	}
	// Walking away stands you up.
	a.send(map[string]any{"t": "walk", "x": 7, "y": 12})
	if av := watcher.expect("update")["avatar"].(map[string]any); av["seat"] != nil {
		t.Fatalf("walking should stand up, got seat %v", av["seat"])
	}
}

func TestBellHasACooldown(t *testing.T) {
	url := startServer(t)
	a, b := dial(t, url), dial(t, url)
	a.send(map[string]any{"t": "use", "prop": "bell"})
	b.expect("fx")
	b.send(map[string]any{"t": "use", "prop": "bell"})
	time.Sleep(300 * time.Millisecond) // b walks over (2ms steps) and tries the bell
	b.send(map[string]any{"t": "use", "prop": "tree-se"})
	if fx := b.expect("fx"); fx["kind"] != "tree" {
		t.Fatalf("the bell shouldn't ring twice in a row, got %v", fx)
	}
}

func TestSignupKeepsCharacterStyle(t *testing.T) {
	url := startServer(t)
	a, b := dial(t, url), dial(t, url)
	style := map[string]any{"skin": 2, "hair": 5, "outfit": 3, "accessory": 1}
	a.send(map[string]any{"t": "signup", "username": "designer", "password": "long enough pw", "birthYear": 1990, "style": style})
	authed := a.expect("authed")
	if got := authed["style"].(map[string]any); got["hair"] != float64(5) || got["accessory"] != float64(1) {
		t.Fatalf("want chosen style back, got %v", got)
	}
	if got := b.expect("update")["avatar"].(map[string]any)["style"].(map[string]any); got["outfit"] != float64(3) {
		t.Fatalf("others should see the new look, got %v", got)
	}

	// Logging in again elsewhere brings the same look.
	c := dial(t, url)
	c.send(map[string]any{"t": "login", "username": "designer", "password": "long enough pw"})
	if got := c.expect("authed")["style"].(map[string]any); got["skin"] != float64(2) {
		t.Fatalf("login should restore the look, got %v", got)
	}

	bad := dial(t, url)
	bad.send(map[string]any{"t": "signup", "username": "badlook", "password": "long enough pw", "birthYear": 1990,
		"style": map[string]any{"skin": 99}})
	bad.expect("auth_error")
}

func TestBoardVotesAndSorts(t *testing.T) {
	url := startServer(t)
	a, b, guest := dial(t, url), dial(t, url), dial(t, url)
	signup(a, "voter_a", 1990)
	signup(b, "voter_b", 1990)

	a.send(map[string]any{"t": "post_thread", "title": "first", "text": "x"})
	first := a.expect("posted_thread")["thread"].(map[string]any)["id"]
	a.send(map[string]any{"t": "post_thread", "title": "second", "text": "y"})
	a.expect("posted_thread")

	guest.send(map[string]any{"t": "vote", "thread": first, "value": 1})
	guest.expect("need_account")

	b.send(map[string]any{"t": "vote", "thread": first, "value": -1})
	if v := b.expect("votes"); v["downs"] != float64(1) || v["mine"] != float64(-1) {
		t.Fatalf("want b's downvote counted, got %v", v)
	}
	// Others in the room see the totals change.
	if v := a.expect("votes"); v["thread"] != first || v["downs"] != float64(1) {
		t.Fatalf("a should see the new totals, got %v", v)
	}

	b.send(map[string]any{"t": "board", "room": "plaza", "sort": "downvoted"})
	board := b.expect("board")
	threads := board["threads"].([]any)
	top := threads[0].(map[string]any)
	if board["sort"] != "downvoted" || top["title"] != "first" || top["mine"] != float64(-1) {
		t.Fatalf("most downvoted first, with b's own vote: %v", board)
	}
	b.send(map[string]any{"t": "board", "room": "plaza", "sort": "new"})
	if top := b.expect("board")["threads"].([]any)[0].(map[string]any); top["title"] != "second" {
		t.Fatalf("newest first, got %v", top["title"])
	}
	b.send(map[string]any{"t": "board", "room": "plaza", "sort": "nonsense"})
	if got := b.expect("board")["sort"]; got != "active" {
		t.Fatalf("unknown sorts fall back to active, got %v", got)
	}
}

func TestFoodCartAndHydrant(t *testing.T) {
	url := startServer(t)
	a, watcher := dial(t, url), dial(t, url)
	a.send(map[string]any{"t": "use", "prop": "cart-hotdog"})
	if fx := watcher.expect("fx"); fx["kind"] != "cart" || fx["variant"] != "hotdog" {
		t.Fatalf("want a hot dog handed out, got %v", fx)
	}
	time.Sleep(50 * time.Millisecond)
	a.send(map[string]any{"t": "use", "prop": "hydrant-w"})
	if fx := watcher.expect("fx"); fx["kind"] != "hydrant" {
		t.Fatalf("want the hydrant to spray, got %v", fx)
	}
	// Manholes are flat: you can walk over them, and there's nothing to use.
	a.send(map[string]any{"t": "walk", "x": 8, "y": 15})
	a.send(map[string]any{"t": "use", "prop": "manhole-a"})
	a.send(map[string]any{"t": "use", "prop": "tree-sw"})
	if fx := watcher.expect("fx"); fx["kind"] != "tree" {
		t.Fatalf("manholes shouldn't do anything, got %v", fx)
	}
}

func TestPlazaHolds500(t *testing.T) {
	sv, err := New(context.Background(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	plaza := sv.rooms["plaza"]
	if plaza.def.Capacity < 500 {
		t.Fatalf("plaza capacity %d, want at least 500", plaza.def.Capacity)
	}
	sv.mu.Lock()
	defer sv.mu.Unlock()
	taken := map[world.Point]bool{}
	for i := range 500 {
		s := &session{send: make(chan []byte, sendQueueLen), done: make(chan struct{}), guestKey: fmt.Sprint("g", i), style: randomStyle()}
		sv.enterLocked(s, plaza, plaza.def.Spawn)
		if taken[s.pos] {
			t.Fatalf("person %d landed on an occupied tile %v", i, s.pos)
		}
		if !plaza.def.Standable(s.pos) {
			t.Fatalf("person %d landed on %v, which isn't open floor", i, s.pos)
		}
		taken[s.pos] = true
	}
}

func TestLoginAttemptsAreRateLimited(t *testing.T) {
	url := startServer(t)
	a := dial(t, url)
	for i := 0; i < authBurst; i++ {
		a.send(map[string]any{"t": "login", "username": "nobody", "password": "wrong password!!"})
		if msg := a.expect("auth_error")["message"]; msg != ErrBadLogin.Error() {
			t.Fatalf("attempt %d: want a normal bad-login error, got %q", i+1, msg)
		}
	}
	a.send(map[string]any{"t": "login", "username": "nobody", "password": "wrong password!!"})
	if msg := a.expect("auth_error")["message"].(string); !strings.Contains(msg, "Too many tries") {
		t.Fatalf("want rate-limit error after %d attempts, got %q", authBurst, msg)
	}
}
