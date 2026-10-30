package server

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"habit/internal/store"
)

func startClocked(t *testing.T, st store.Store, clock *fakeClock) (*Server, string) {
	t.Helper()
	sv, err := New(context.Background(), Options{Store: st, Tick: 2 * time.Millisecond, Step: 2 * time.Millisecond, Now: clock.Now})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	sv.dailyCheck(ctx) // the day is ready before anyone connects
	go sv.Run(ctx)
	ts := httptest.NewServer(sv.Handler())
	t.Cleanup(ts.Close)
	return sv, "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws"
}

func TestDailyRhythm(t *testing.T) {
	// A minute before the evening meetup, New York time.
	clock := &fakeClock{t: time.Date(2026, 10, 11, 19, 59, 0, 0, plazaZone)}
	st := store.NewMemory()
	sv, url := startClocked(t, st, clock)
	// Let time pass as the test runs, so people can walk.
	stop := make(chan struct{})
	t.Cleanup(func() { close(stop) })
	go func() {
		for {
			select {
			case <-stop:
				return
			case <-time.After(time.Millisecond):
				clock.Add(time.Millisecond)
			}
		}
	}()

	c := dial(t, url)
	day := c.expect("today")["today"].(map[string]any)
	if day["day"] != "2026-10-11" || day["question"] == "" || day["thread"] == float64(0) || day["meetup"] != false {
		t.Fatalf("unexpected day: %v", day)
	}
	// The question is pinned to the Plaza's board as a thread.
	c.send(map[string]any{"t": "thread", "thread": day["thread"]})
	th := c.expect("thread")["thread"].(map[string]any)
	if !strings.Contains(th["title"].(string), day["question"].(string)) || th["name"] != dailyAuthor {
		t.Fatalf("question thread: %v", th)
	}

	// A wish counts towards today's goal, and is saved.
	c.send(map[string]any{"t": "use", "prop": "fountain"})
	if fx := c.expect("fx"); fx["today"] != float64(1) || fx["goal"] != float64(wishGoal) {
		t.Fatalf("want 1 of %d wishes today, got %v", wishGoal, fx)
	}
	deadline := time.Now().Add(2 * time.Second)
	for n, _ := st.Counter(context.Background(), "wishes:2026-10-11"); n != 1; n, _ = st.Counter(context.Background(), "wishes:2026-10-11") {
		if time.Now().After(deadline) {
			t.Fatal("wish wasn't saved")
		}
		time.Sleep(10 * time.Millisecond)
	}

	// 8 pm: the bell rings and the meetup is on.
	clock.Add(2 * time.Minute)
	sv.dailyCheck(context.Background())
	if fx := c.expect("fx"); fx["kind"] != "bell" {
		t.Fatalf("want the bell, got %v", fx)
	}
	if day := c.expect("today")["today"].(map[string]any); day["meetup"] != true {
		t.Fatalf("meetup should be on: %v", day)
	}

	// Coming back: someone's new post shows (the day's question has its
	// own spot, so it isn't repeated).
	signup(c, "poster", 1990)
	c.send(map[string]any{"t": "post_thread", "title": "Tacos tonight?", "text": "Who's in?"})
	c.expect("posted_thread")
	c.send(map[string]any{"t": "missed", "since": clock.Now().Add(-time.Hour).UnixMilli()})
	if m := c.expect("missed"); m["posts"] != float64(1) || len(m["top"].([]any)) != 1 {
		t.Fatalf("what you missed: %v", m)
	}

	// After a restart the day picks up where it was, with the same thread.
	sv2, _ := startClocked(t, st, clock)
	sv2.mu.Lock()
	again := sv2.todayViewLocked()
	sv2.mu.Unlock()
	if again.Wishes != 1 || again.Thread != int64(day["thread"].(float64)) {
		t.Fatalf("after restart: %+v", again)
	}
}

func TestQuestionOfTheDay(t *testing.T) {
	// The same question all day, and a different one the next day.
	day := "2026-10-30"
	if questionFor(day) != questionFor(day) {
		t.Fatal("a day's question must not change")
	}
	seen := map[string]bool{}
	d, _ := time.Parse(time.DateOnly, day)
	for i := 0; i < 60; i++ {
		today := d.AddDate(0, 0, i).Format(time.DateOnly)
		yesterday := d.AddDate(0, 0, i-1).Format(time.DateOnly)
		if questionFor(today) == questionFor(yesterday) {
			t.Fatalf("%s repeats yesterday's question", today)
		}
		seen[questionFor(today)] = true
	}
	if len(seen) < 20 {
		t.Fatalf("want a good spread of questions over 60 days, got %d different ones", len(seen))
	}
}

func TestStaleQuestionIsReplaced(t *testing.T) {
	// Today's thread was posted with a question that's no longer the pick
	// (the list changed): it's taken down, answers and all, and today's
	// question goes up instead.
	ctx := context.Background()
	st := store.NewMemory()
	clock := &fakeClock{t: time.Date(2026, 10, 30, 12, 0, 0, 0, plazaZone)}
	day := plazaDay(clock.Now())
	old := &store.Thread{Room: "plaza", Name: dailyAuthor, Title: questionTitle("Breakfast for dinner: genius or chaos?"), Text: "?", CreatedAt: clock.Now()}
	if err := st.CreateThread(ctx, old); err != nil {
		t.Fatal(err)
	}
	if _, err := st.AddReply(ctx, &store.Reply{ThreadID: old.ID, Text: "chaos", CreatedAt: clock.Now()}); err != nil {
		t.Fatal(err)
	}
	if err := st.SetCounter(ctx, "daily-thread:"+day, old.ID); err != nil {
		t.Fatal(err)
	}

	sv, _ := startClocked(t, st, clock)
	sv.mu.Lock()
	now := sv.today
	sv.mu.Unlock()
	if now.thread == old.ID || now.question != questionFor(day) {
		t.Fatalf("want a new thread for %q, got thread %d %q", questionFor(day), now.thread, now.question)
	}
	if _, err := st.Thread(ctx, old.ID); err == nil {
		t.Fatal("the old question should be gone")
	}
	th, err := st.Thread(ctx, now.thread)
	if err != nil || th.Title != questionTitle(questionFor(day)) {
		t.Fatalf("want today's question posted, got %+v, %v", th, err)
	}

	// Restarting keeps today's (current) question and its thread.
	sv2, _ := startClocked(t, st, clock)
	sv2.mu.Lock()
	again := sv2.today.thread
	sv2.mu.Unlock()
	if again != now.thread {
		t.Fatalf("a restart shouldn't replace a current question: %d then %d", now.thread, again)
	}
}
