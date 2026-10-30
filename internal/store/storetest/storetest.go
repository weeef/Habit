// Package storetest is a shared test suite that every store.Store
// implementation must pass.
package storetest

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"habit/internal/store"
)

// Run checks a store implementation. newStore must return an empty store.
func Run(t *testing.T, newStore func(t *testing.T) store.Store) {
	tests := map[string]func(*testing.T, store.Store){
		"Accounts": testAccounts,
		"Look":     testLook,
		"Sessions": testSessions,
		"Messages": testMessages,
		"Board":    testBoard,
		"Notify":   testNotifications,
		"Votes":    testVotesAndSorts,
		"Secret":   testSecret,
		"Counters": testCounters,
		"Blocks":   testBlocks,
	}
	for name, fn := range tests {
		t.Run(name, func(t *testing.T) {
			s := newStore(t)
			t.Cleanup(s.Close)
			fn(t, s)
		})
	}
}

// Whole seconds, so every backend's timestamp precision round-trips exactly.
var now = time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)

func account(id, username string) *store.Account {
	return &store.Account{
		ID: id, Username: username, BirthYear: 1995, CreatedAt: now,
		Salt: []byte("salt"), PasswordHash: []byte("hash"), RecoveryHash: []byte("recovery"),
	}
}

func testAccounts(t *testing.T, s store.Store) {
	ctx := context.Background()
	want := account("a1", "Pixel_Fan")
	want.Style = store.Style{Skin: 3, Hair: 5, Outfit: 2, Accessory: 1}
	if err := s.CreateAccount(ctx, want); err != nil {
		t.Fatal(err)
	}
	got, err := s.AccountByUsername(ctx, "pixel_fan")
	if err != nil {
		t.Fatalf("lookup should ignore case: %v", err)
	}
	if got.ID != want.ID || got.Username != want.Username || got.BirthYear != want.BirthYear ||
		!got.CreatedAt.Equal(want.CreatedAt) || !bytes.Equal(got.Salt, want.Salt) ||
		!bytes.Equal(got.PasswordHash, want.PasswordHash) || !bytes.Equal(got.RecoveryHash, want.RecoveryHash) ||
		got.Style != want.Style {
		t.Fatalf("account didn't round-trip:\nwant %+v\ngot  %+v", want, got)
	}
	if err := s.CreateAccount(ctx, account("a2", "PIXEL_FAN")); !errors.Is(err, store.ErrUsernameTaken) {
		t.Fatalf("want ErrUsernameTaken for a different-case duplicate, got %v", err)
	}
	if _, err := s.AccountByUsername(ctx, "nobody"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}

	for want := 1; want <= 2; want++ {
		if n, err := s.AddWelcome(ctx, "a1"); err != nil || n != want {
			t.Fatalf("AddWelcome: want %d, got %d, %v", want, n, err)
		}
	}
	if got, _ := s.AccountByUsername(ctx, "pixel_fan"); got.Welcomes != 2 {
		t.Fatalf("want 2 welcomes saved, got %d", got.Welcomes)
	}
	if _, err := s.AddWelcome(ctx, "nobody"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("want ErrNotFound for an unknown account, got %v", err)
	}
}

func testLook(t *testing.T, s store.Store) {
	ctx := context.Background()
	if err := s.CreateAccount(ctx, account("a1", "walker")); err != nil {
		t.Fatal(err)
	}
	look := store.Style{Skin: 2, Hair: 3, Outfit: 1, Accessory: 2}
	if err := s.SetLook(ctx, "a1", look, true); err != nil {
		t.Fatal(err)
	}
	got, err := s.AccountByUsername(ctx, "walker")
	if err != nil || got.Style != look || !got.RandomLook {
		t.Fatalf("look didn't save: %+v, %v", got, err)
	}
	if err := s.SetLook(ctx, "a1", look, false); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.AccountByUsername(ctx, "walker"); got.RandomLook {
		t.Fatal("random look should be off again")
	}
	if err := s.SetLook(ctx, "nobody", look, false); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("want ErrNotFound for an unknown account, got %v", err)
	}
}

func testSessions(t *testing.T, s store.Store) {
	ctx := context.Background()
	if err := s.CreateAccount(ctx, account("a1", "walker")); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateSession(ctx, []byte("token-hash"), "a1", now); err != nil {
		t.Fatal(err)
	}
	got, err := s.AccountBySession(ctx, []byte("token-hash"))
	if err != nil || got.ID != "a1" {
		t.Fatalf("want account a1, got %+v, %v", got, err)
	}
	if _, err := s.AccountBySession(ctx, []byte("other")); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("want ErrNotFound for an unknown token, got %v", err)
	}
}

func testMessages(t *testing.T, s store.Store) {
	ctx := context.Background()
	if err := s.CreateAccount(ctx, account("a1", "walker")); err != nil {
		t.Fatal(err)
	}
	for i := range 5 {
		m := store.Message{
			Room: "gaming", AccountID: "a1", AvatarID: "av1", Name: "Anonymous #3f9a",
			Text: fmt.Sprintf("msg %d", i), Newbie: i == 0, SentAt: now.Add(time.Duration(i) * time.Second),
		}
		if err := s.AddMessage(ctx, m); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.AddMessage(ctx, store.Message{Room: "gym", AccountID: "a1", AvatarID: "av2", Name: "walker", Text: "elsewhere", SentAt: now}); err != nil {
		t.Fatal(err)
	}

	got, err := s.RecentMessages(ctx, "gaming", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[0].Text != "msg 2" || got[2].Text != "msg 4" {
		t.Fatalf("want the last 3 gaming messages oldest first, got %+v", got)
	}
	m := got[2]
	if m.Room != "gaming" || m.AccountID != "a1" || m.AvatarID != "av1" || m.Name != "Anonymous #3f9a" ||
		m.Newbie || !m.SentAt.Equal(now.Add(4*time.Second)) {
		t.Fatalf("message didn't round-trip: %+v", m)
	}

	if got, _ := s.RecentMessages(ctx, "gym", 10); len(got) != 1 || got[0].Text != "elsewhere" {
		t.Fatalf("rooms should be kept apart, got %+v", got)
	}
	if got, _ := s.RecentMessages(ctx, "empty", 10); len(got) != 0 {
		t.Fatalf("want no messages, got %+v", got)
	}
}

func newThread(t *testing.T, s store.Store, room, author, title string, at time.Time) *store.Thread {
	t.Helper()
	th := &store.Thread{Room: room, AccountID: author, Title: title, Text: title + " body", CreatedAt: at}
	if err := s.CreateThread(context.Background(), th); err != nil {
		t.Fatal(err)
	}
	return th
}

func reply(t *testing.T, s store.Store, thread int64, author, text string, at time.Time) []store.Notification {
	t.Helper()
	r := &store.Reply{ThreadID: thread, AccountID: author, Text: text, CreatedAt: at}
	notes, err := s.AddReply(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	if r.ID == 0 {
		t.Fatal("AddReply should set the reply ID")
	}
	return notes
}

func titles(ts []store.Thread) []string {
	var out []string
	for _, t := range ts {
		out = append(out, t.Title)
	}
	return out
}

func testBoard(t *testing.T, s store.Store) {
	ctx := context.Background()
	for _, id := range []string{"a1", "a2"} {
		if err := s.CreateAccount(ctx, account(id, "user_"+id)); err != nil {
			t.Fatal(err)
		}
	}
	first := newThread(t, s, "gaming", "a1", "first", now)
	newThread(t, s, "gaming", "a2", "second", now.Add(time.Minute))
	newThread(t, s, "gym", "a1", "lifting", now)
	if first.ID == 0 || !first.BumpedAt.Equal(now) || first.Replies != 0 {
		t.Fatalf("CreateThread should set ID, BumpedAt and Replies, got %+v", first)
	}

	got, err := s.Threads(ctx, "gaming", store.SortActive, now, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(titles(got)) != "[second first]" {
		t.Fatalf("want newest first, got %v", titles(got))
	}

	// A reply bumps the older thread back to the top.
	reply(t, s, first.ID, "a2", "bump", now.Add(2*time.Minute))
	got, _ = s.Threads(ctx, "gaming", store.SortActive, now, 10, 0)
	if fmt.Sprint(titles(got)) != "[first second]" || got[0].Replies != 1 || !got[0].BumpedAt.Equal(now.Add(2*time.Minute)) {
		t.Fatalf("reply should bump the thread, got %+v", got)
	}
	if got, _ := s.Threads(ctx, "gaming", store.SortActive, now, 1, 1); fmt.Sprint(titles(got)) != "[second]" {
		t.Fatalf("limit/offset should page, got %v", titles(got))
	}

	th, err := s.Thread(ctx, first.ID)
	if err != nil || th.Title != "first" || th.Text != "first body" || th.AccountID != "a1" || th.Room != "gaming" {
		t.Fatalf("thread didn't round-trip: %+v, %v", th, err)
	}
	if _, err := s.Thread(ctx, 999999); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}

	reply(t, s, first.ID, "a1", "thanks", now.Add(3*time.Minute))
	rs, err := s.Replies(ctx, first.ID)
	if err != nil || len(rs) != 2 || rs[0].Text != "bump" || rs[1].Text != "thanks" || rs[1].AccountID != "a1" {
		t.Fatalf("want both replies oldest first, got %+v, %v", rs, err)
	}

	if _, err := s.AddReply(ctx, &store.Reply{ThreadID: 999999, AccountID: "a1", Text: "x", CreatedAt: now}); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("want ErrNotFound replying to a missing thread, got %v", err)
	}
}

func testNotifications(t *testing.T, s store.Store) {
	ctx := context.Background()
	for _, id := range []string{"op", "b", "c"} {
		if err := s.CreateAccount(ctx, account(id, "user_"+id)); err != nil {
			t.Fatal(err)
		}
	}
	th := newThread(t, s, "gaming", "op", "anyone up?", now)

	// The author replying to their own empty thread notifies nobody.
	if notes := reply(t, s, th.ID, "op", "bump", now.Add(time.Second)); len(notes) != 0 {
		t.Fatalf("want no notifications, got %+v", notes)
	}

	notes := reply(t, s, th.ID, "b", "me!", now.Add(2*time.Second))
	if len(notes) != 1 || notes[0].AccountID != "op" || notes[0].Title != "anyone up?" ||
		notes[0].Room != "gaming" || notes[0].Text != "me!" || notes[0].ID == 0 {
		t.Fatalf("want one notification for op, got %+v", notes)
	}

	// Everyone who has posted hears about a new reply, except its author.
	notes = reply(t, s, th.ID, "c", "me too", now.Add(3*time.Second))
	var who []string
	for _, n := range notes {
		who = append(who, n.AccountID)
	}
	slices.Sort(who)
	if fmt.Sprint(who) != "[b op]" {
		t.Fatalf("want b and op notified, got %v", who)
	}

	got, err := s.Notifications(ctx, "op", 10)
	if err != nil || len(got) != 2 || got[0].Text != "me too" || got[1].Text != "me!" || got[0].Read {
		t.Fatalf("want op's 2 unread notifications newest first, got %+v, %v", got, err)
	}
	if got[0].Title != "anyone up?" || got[0].Room != "gaming" || got[0].ThreadID != th.ID {
		t.Fatalf("notification should carry the thread, got %+v", got[0])
	}
	if got, _ := s.Notifications(ctx, "op", 1); len(got) != 1 {
		t.Fatalf("limit should apply, got %d", len(got))
	}

	if err := s.MarkNotificationsRead(ctx, "op"); err != nil {
		t.Fatal(err)
	}
	got, _ = s.Notifications(ctx, "op", 10)
	if !got[0].Read || !got[1].Read {
		t.Fatalf("want all read, got %+v", got)
	}
	if got, _ := s.Notifications(ctx, "b", 10); len(got) != 1 || got[0].Read {
		t.Fatalf("other accounts should stay unread, got %+v", got)
	}
}

func testVotesAndSorts(t *testing.T, s store.Store) {
	ctx := context.Background()
	for _, id := range []string{"a", "b", "c"} {
		if err := s.CreateAccount(ctx, account(id, "user_"+id)); err != nil {
			t.Fatal(err)
		}
	}
	old := newThread(t, s, "gaming", "a", "old", now.Add(-48*time.Hour))
	liked := newThread(t, s, "gaming", "a", "liked", now.Add(-10*time.Hour))
	hated := newThread(t, s, "gaming", "a", "hated", now.Add(-5*time.Hour))
	fresh := newThread(t, s, "gaming", "a", "fresh", now.Add(-1*time.Hour))

	vote := func(th *store.Thread, who string, v, wantUps, wantDowns int) {
		t.Helper()
		ups, downs, err := s.Vote(ctx, th.ID, who, v)
		if err != nil || ups != wantUps || downs != wantDowns {
			t.Fatalf("vote %d by %s on %s: got %d/%d, %v; want %d/%d", v, who, th.Title, ups, downs, err, wantUps, wantDowns)
		}
	}
	vote(liked, "a", 1, 1, 0)
	vote(liked, "b", 1, 2, 0)
	vote(liked, "c", 1, 3, 0)
	vote(old, "a", 1, 1, 0)
	vote(old, "b", 1, 2, 0)
	vote(hated, "a", -1, 0, 1)
	vote(hated, "b", -1, 0, 2)
	// Changing your mind moves your vote; taking it back removes it.
	vote(fresh, "c", -1, 0, 1)
	vote(fresh, "c", 1, 1, 0)
	vote(fresh, "c", 0, 0, 0)
	vote(fresh, "c", 1, 1, 0)

	mine, err := s.VotesBy(ctx, "b", []int64{old.ID, liked.ID, hated.ID, fresh.ID})
	if err != nil || mine[liked.ID] != 1 || mine[hated.ID] != -1 || mine[fresh.ID] != 0 {
		t.Fatalf("want b's votes, got %v, %v", mine, err)
	}
	if _, _, err := s.Vote(ctx, 999999, "a", 1); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("want ErrNotFound voting on a missing thread, got %v", err)
	}

	for sort, want := range map[store.ThreadSort]string{
		store.SortNew:       "[fresh hated liked old]",
		store.SortTop:       "[liked old fresh hated]",
		store.SortDownvoted: "[hated fresh liked old]",
		store.SortHot:       "[fresh liked old hated]",
	} {
		got, err := s.Threads(ctx, "gaming", sort, now, 10, 0)
		if err != nil {
			t.Fatal(err)
		}
		if fmt.Sprint(titles(got)) != want {
			t.Errorf("%s: want %s, got %v", sort, want, titles(got))
		}
	}
	got, _ := s.Threads(ctx, "gaming", store.SortTop, now, 1, 0)
	if got[0].Ups != 3 || got[0].Downs != 0 {
		t.Fatalf("threads should carry their vote counts, got %+v", got[0])
	}
}

func testCounters(t *testing.T, s store.Store) {
	ctx := context.Background()
	if n, err := s.Counter(ctx, "wishes:2026-10-11"); err != nil || n != 0 {
		t.Fatalf("unset counter: want 0, got %d, %v", n, err)
	}
	for want := int64(1); want <= 3; want++ {
		if n, err := s.AddCounter(ctx, "wishes:2026-10-11", 1); err != nil || n != want {
			t.Fatalf("AddCounter: want %d, got %d, %v", want, n, err)
		}
	}
	if err := s.SetCounter(ctx, "daily", 42); err != nil {
		t.Fatal(err)
	}
	if err := s.SetCounter(ctx, "daily", 43); err != nil {
		t.Fatal(err)
	}
	if n, _ := s.Counter(ctx, "daily"); n != 43 {
		t.Fatalf("SetCounter: want 43, got %d", n)
	}
	if n, _ := s.Counter(ctx, "wishes:2026-10-11"); n != 3 {
		t.Fatalf("want 3, got %d", n)
	}
}

func testBlocks(t *testing.T, s store.Store) {
	ctx := context.Background()
	if err := s.CreateAccount(ctx, account("a1", "founder")); err != nil {
		t.Fatal(err)
	}
	official := &store.Block{Slug: "gaming", Name: "Gaming", Bullet: "🎮", Color: "#8b6bf2", About: "Games", Template: "arcade", Official: true, CreatedAt: now}
	mine := &store.Block{Slug: "Knitting", Name: "Knit & Chill", Bullet: "🧶", Color: "#ff6fae", About: "Yarn talk", Template: "lounge", OwnerID: "a1", CreatedAt: now.Add(time.Hour)}
	for _, b := range []*store.Block{mine, official} {
		if err := s.CreateBlock(ctx, b); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.CreateBlock(ctx, &store.Block{Slug: "knitting", Name: "x", CreatedAt: now}); !errors.Is(err, store.ErrNameTaken) {
		t.Fatalf("want ErrNameTaken for a different-case duplicate, got %v", err)
	}
	got, err := s.BlockBySlug(ctx, "KNITTING")
	if err == nil && got.CreatedAt.Equal(mine.CreatedAt) {
		got.CreatedAt = mine.CreatedAt // same moment; the zone may differ
	}
	if err != nil || *got != *mine {
		t.Fatalf("block didn't round-trip:\nwant %+v\ngot  %+v (%v)", mine, got, err)
	}
	if _, err := s.BlockBySlug(ctx, "nope"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
	all, err := s.SearchBlocks(ctx, "", 10)
	if err != nil || len(all) != 2 || all[0].Slug != "gaming" {
		t.Fatalf("want official first, got %+v (%v)", all, err)
	}
	if found, _ := s.SearchBlocks(ctx, "CHILL", 10); len(found) != 1 || found[0].Slug != "Knitting" {
		t.Fatalf("search by name: %+v", found)
	}
	if n, _ := s.BlocksStartedSince(ctx, "a1", now); n != 1 {
		t.Fatalf("want 1 block started since, got %d", n)
	}
	if n, _ := s.BlocksStartedSince(ctx, "a1", now.Add(2*time.Hour)); n != 0 {
		t.Fatalf("want 0 blocks started later, got %d", n)
	}

	// Storefronts: one block per lot, and they can move or close.
	if mineList, _ := s.BlocksByOwner(ctx, "a1"); len(mineList) != 1 || mineList[0].Slug != "Knitting" {
		t.Fatalf("BlocksByOwner: %+v", mineList)
	}
	if err := s.SetBlockLot(ctx, "knitting", "ave-1", 2); err != nil {
		t.Fatal(err)
	}
	if err := s.SetBlockLot(ctx, "gaming", "ave-1", 2); !errors.Is(err, store.ErrNameTaken) {
		t.Fatalf("want ErrNameTaken for a taken lot, got %v", err)
	}
	if err := s.SetBlockLot(ctx, "gaming", "ave-2", 2); err != nil {
		t.Fatalf("the same lot number on another avenue is fine: %v", err)
	}
	if lots, _ := s.BlocksWithLots(ctx); len(lots) != 2 {
		t.Fatalf("want 2 storefronts, got %+v", lots)
	}
	if err := s.SetBlockLot(ctx, "knitting", "", 0); err != nil {
		t.Fatal(err)
	}
	if err := s.SetBlockLot(ctx, "gaming", "ave-1", 2); err != nil {
		t.Fatalf("a freed lot can be taken: %v", err)
	}
	if b, _ := s.BlockBySlug(ctx, "gaming"); b.Street != "ave-1" || b.Lot != 2 {
		t.Fatalf("lot not saved: %+v", b)
	}
	if err := s.SetBlockLot(ctx, "nope", "ave-1", 3); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func testSecret(t *testing.T, s store.Store) {
	ctx := context.Background()
	first, err := s.LoadOrStoreSecret(ctx, []byte("first"))
	if err != nil || string(first) != "first" {
		t.Fatalf("want the fresh secret stored, got %q, %v", first, err)
	}
	again, err := s.LoadOrStoreSecret(ctx, []byte("second"))
	if err != nil || string(again) != "first" {
		t.Fatalf("want the saved secret kept, got %q, %v", again, err)
	}
}
