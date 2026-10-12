package server

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"habit/internal/store"
)

func newBlock(name, slug string) map[string]any {
	return map[string]any{"t": "create_block", "block": map[string]any{
		"name": name, "slug": slug, "bullet": "🧶", "color": "#d6409f", "about": "Yarn talk.", "template": "lounge",
	}}
}

func TestStartABlock(t *testing.T) {
	clock := &fakeClock{t: time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)}
	st := store.NewMemory()
	sv, url := startClocked(t, st, clock)

	guest := dial(t, url)
	guest.send(newBlock("Knit & Chill", "knitting"))
	guest.expect("need_account")

	maker := dial(t, url)
	signup(maker, "knitter", 1990)
	maker.send(newBlock("Knit & Chill", "Plaza"))
	if msg := maker.expect("block_error")["message"].(string); !strings.Contains(msg, "reserved") {
		t.Fatalf("plaza should be reserved, got %q", msg)
	}
	maker.send(newBlock("Knit & Chill", "knitting"))
	if b := maker.expect("block_created")["block"].(map[string]any); b["slug"] != "knitting" || b["yours"] != true {
		t.Fatalf("unexpected block: %v", b)
	}
	// Starting it takes you in.
	room := maker.expect("room")["room"].(map[string]any)
	if room["id"] != "knitting" || room["name"] != "🧶 Knit & Chill" || room["block"].(map[string]any)["about"] != "Yarn talk." {
		t.Fatalf("should be in the new block, got %v", room)
	}

	// The name is now taken, whatever the case.
	maker.send(newBlock("Other", "KNITTING"))
	if msg := maker.expect("block_error")["message"].(string); !strings.Contains(msg, "taken") {
		t.Fatalf("want taken, got %q", msg)
	}

	// It's in the directory, with the one person inside, and searchable.
	guest.send(map[string]any{"t": "blocks", "q": "chill"})
	list := guest.expect("blocks")["blocks"].([]any)
	if len(list) != 1 || list[0].(map[string]any)["count"] != float64(1) {
		t.Fatalf("directory: %v", list)
	}
	guest.send(map[string]any{"t": "blocks", "q": ""})
	if all := guest.expect("blocks")["blocks"].([]any); len(all) != 3 {
		t.Fatalf("want gaming, gym and knitting, got %v", all)
	}

	// Anyone can ride there.
	guest.send(map[string]any{"t": "teleport", "room": "knitting"})
	if r := guest.expect("room")["room"].(map[string]any); r["id"] != "knitting" {
		t.Fatalf("teleport: %v", r)
	}
	guest.send(map[string]any{"t": "teleport", "room": "nope_nothing"})
	if n := guest.expect("notice")["message"].(string); !strings.Contains(n, "no block") {
		t.Fatalf("unknown block: %q", n)
	}

	// Once everyone has gone, the room unloads after a while...
	guest.send(map[string]any{"t": "teleport", "room": "plaza"})
	guest.expect("room")
	maker.send(map[string]any{"t": "teleport", "room": "plaza"})
	maker.expect("room")
	clock.Add(blockIdle + time.Second)
	sv.dailyCheck(context.Background())
	sv.mu.Lock()
	_, loaded := sv.rooms["knitting"]
	sv.mu.Unlock()
	if loaded {
		t.Fatal("an empty block should unload")
	}
	// ...and a link straight to it loads it again.
	ts := httptest.NewServer(sv.Handler())
	t.Cleanup(ts.Close)
	direct := dial(t, "ws"+strings.TrimPrefix(ts.URL, "http")+"/ws?room=knitting")
	if r := direct.first["room"].(map[string]any); r["id"] != "knitting" {
		t.Fatalf("link should open the block, got %v", r["id"])
	}

	// Three a day is plenty.
	for _, slug := range []string{"crochet", "embroidery"} {
		maker.send(newBlock("More yarn", slug))
		maker.expect("block_created")
	}
	maker.send(newBlock("Too many", "quilting"))
	if msg := maker.expect("block_error")["message"].(string); !strings.Contains(msg, "tomorrow") {
		t.Fatalf("want the daily limit, got %q", msg)
	}
}
