package server

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

type client struct {
	t    *testing.T
	conn *websocket.Conn
}

func startServer(t *testing.T) string {
	t.Helper()
	sv := New(Options{Tick: 2 * time.Millisecond, Step: 2 * time.Millisecond})
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
	c.expect("room")
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

	// The Plaza's door to Gaming is at (0, 3).
	a.send(map[string]any{"t": "walk", "x": 0, "y": 3})
	room := a.expect("room")["room"].(map[string]any)
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

	// Peeking at a room you aren't in shows its recent chat.
	watcher.send(map[string]any{"t": "peek", "room": "gaming"})
	peek := watcher.expect("peek")
	if peek["count"] != float64(1) {
		t.Fatalf("peek should count 1 person in gaming, got %v", peek["count"])
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
