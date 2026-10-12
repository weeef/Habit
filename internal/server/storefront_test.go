package server

import (
	"strings"
	"testing"
	"time"

	"habit/internal/store"
	"habit/internal/world"
)

func roomDoors(m map[string]any) []map[string]any {
	var out []map[string]any
	for _, d := range m["doors"].([]any) {
		out = append(out, d.(map[string]any))
	}
	return out
}

func TestStorefrontOnAnAvenue(t *testing.T) {
	clock := &fakeClock{t: time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)}
	st := store.NewMemory()
	_, url := startClocked(t, st, clock)
	stop := make(chan struct{})
	t.Cleanup(func() { close(stop) })
	go func() { // let time pass so people can walk
		for {
			select {
			case <-stop:
				return
			case <-time.After(time.Millisecond):
				clock.Add(time.Millisecond)
			}
		}
	}()

	owner := dial(t, url)
	signup(owner, "knitter", 1990)
	owner.send(newBlock("Knit & Chill", "knitting"))
	owner.expect("block_created")
	owner.expect("room")

	// Someone strolling on 1st Avenue sees the shop open.
	walker := dial(t, url)
	walker.send(map[string]any{"t": "teleport", "room": "ave-1"})
	avenue := walker.expect("room")["room"].(map[string]any)
	if len(avenue["lots"].([]any)) == 0 || avenue["kind"] != "street" {
		t.Fatalf("an avenue should have lots: %v", avenue)
	}

	// Only the owner can pick the shop.
	walker.send(map[string]any{"t": "signup", "username": "stranger", "password": "long enough pw", "birthYear": 1990})
	walker.expect("authed")
	walker.send(map[string]any{"t": "claim_lot", "slug": "knitting", "street": "ave-1", "lot": 2})
	if msg := walker.expect("block_error")["message"].(string); !strings.Contains(msg, "Only the person") {
		t.Fatalf("want owner-only, got %q", msg)
	}

	owner.send(map[string]any{"t": "claim_lot", "slug": "knitting", "street": "ave-1", "lot": 2})
	if b := owner.expect("lot_claimed")["block"].(map[string]any); b["street"] != "1st Avenue" || b["lot"] != float64(2) {
		t.Fatalf("lot_claimed: %v", b)
	}
	doors := roomDoors(walker.expect("doors"))
	door := world.LotDoor(2)
	found := false
	for _, d := range doors {
		p := d["pos"].([]any)
		if d["target"] == "knitting" && p[0] == float64(door.X) && p[1] == float64(door.Y) && d["color"] == "#d6409f" {
			found = true
		}
	}
	if !found {
		t.Fatalf("the avenue should get the storefront door: %v", doors)
	}

	// Walking into the shop goes into the block (its room loads first).
	walker.send(map[string]any{"t": "walk", "x": door.X, "y": door.Y})
	inside := walker.expect("room")
	if inside["room"].(map[string]any)["id"] != "knitting" {
		t.Fatalf("want to be in the block, got %v", inside["room"].(map[string]any)["id"])
	}
	// Its door leads back out onto 1st Avenue, in front of the shop.
	exit := roomDoors(inside["room"].(map[string]any))[0]
	if exit["target"] != "ave-1" {
		t.Fatalf("the block's door should lead to its avenue: %v", exit)
	}
	p := exit["pos"].([]any)
	walker.send(map[string]any{"t": "walk", "x": int(p[0].(float64)), "y": int(p[1].(float64))})
	out := walker.expect("room")
	if out["room"].(map[string]any)["id"] != "ave-1" {
		t.Fatalf("want back on the avenue, got %v", out["room"].(map[string]any)["id"])
	}

	// Another block can't take the same shop.
	owner.send(newBlock("Crochet Club", "crochet"))
	owner.expect("block_created")
	owner.send(map[string]any{"t": "claim_lot", "slug": "crochet", "street": "ave-1", "lot": 2})
	if msg := owner.expect("block_error")["message"].(string); !strings.Contains(msg, "took that shop") {
		t.Fatalf("want taken, got %q", msg)
	}

	// After a restart the shop is still there.
	_, url2 := startClocked(t, st, clock)
	again := dial(t, url2)
	again.send(map[string]any{"t": "teleport", "room": "ave-1"})
	still := false
	for _, d := range roomDoors(again.expect("room")["room"].(map[string]any)) {
		still = still || d["target"] == "knitting"
	}
	if !still {
		t.Fatal("storefront should survive a restart")
	}
}
