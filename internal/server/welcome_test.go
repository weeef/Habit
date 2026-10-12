package server

import (
	"context"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"habit/internal/store"
)

// fakeClock is a clock tests can move forward.
type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *fakeClock) Now() time.Time      { c.mu.Lock(); defer c.mu.Unlock(); return c.t }
func (c *fakeClock) Add(d time.Duration) { c.mu.Lock(); defer c.mu.Unlock(); c.t = c.t.Add(d) }
func youPos(c *client) (x, y float64) {
	for _, a := range c.first["avatars"].([]any) {
		a := a.(map[string]any)
		if a["you"] == true {
			p := a["pos"].([]any)
			return p[0].(float64), p[1].(float64)
		}
	}
	c.t.Fatal("no avatar marked you")
	return 0, 0
}

func TestNewcomerIsWelcomed(t *testing.T) {
	clock := &fakeClock{t: time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)}
	sv, err := New(context.Background(), Options{Store: store.NewMemory(), Tick: 2 * time.Millisecond, Step: 2 * time.Millisecond, Now: clock.Now})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go sv.Run(ctx)
	ts := httptest.NewServer(sv.Handler())
	t.Cleanup(ts.Close)
	url := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws"

	regular := dial(t, url)
	signup(regular, "regular", 1990)
	clock.Add(2 * time.Minute) // settled in

	newcomer := dial(t, url)
	// They land right beside the person already here, not at the far spawn.
	rx, ry := youPos(regular)
	nx, ny := youPos(newcomer)
	if max(abs(int(rx-nx)), abs(int(ry-ny))) > crowdRadius {
		t.Fatalf("newcomer should land near the regular at (%v,%v), got (%v,%v)", rx, ry, nx, ny)
	}
	nudge := regular.expect("nudge")
	newID := nudge["id"].(string)

	regular.send(map[string]any{"t": "emote", "emote": "wave"})
	w := newcomer.expect("welcomed")
	if w["to"] != newID {
		t.Fatalf("welcome should name the newcomer %q, got %v", newID, w)
	}
	if n := regular.expect("welcomes")["count"]; n != float64(1) {
		t.Fatalf("want 1 welcome counted, got %v", n)
	}

	// A second wave doesn't count again.
	regular.send(map[string]any{"t": "emote", "emote": "wave"})
	regular.send(map[string]any{"t": "chat", "text": "anyone else?"})
	for {
		m := newcomer.expect("chat")
		if m["msg"].(map[string]any)["text"] == "anyone else?" {
			break
		}
	}
	sv.mu.Lock()
	welcomed, newcomers := sv.welcomed, sv.newcomers
	sv.mu.Unlock()
	if welcomed != 1 || newcomers != 2 { // the regular was new once too
		t.Fatalf("want 1 of 2 welcomed, got %d of %d", welcomed, newcomers)
	}
}
