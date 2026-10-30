package server

import (
	"context"
	"errors"
	"hash/fnv"
	"log"
	"sort"
	"time"

	"habit/internal/store"
	"habit/internal/world"
)

// The daily rhythm (PLAN.md §4.1 step 2): a question of the day pinned to
// the Plaza's board, a fountain everyone fills together for fireworks, and
// an evening meetup when the bell rings. Plaza days run on New York time,
// like the city it's drawn from.
const (
	wishGoal      = 100              // wishes in a day for fireworks
	meetupHour    = 20               // 8 pm, New York time
	meetupLength  = 20 * time.Minute // how long the meetup counts as on
	dailyEvery    = 5 * time.Second  // how often the day is checked
	headlineEvery = 10 * time.Minute // how often the newsbox is refreshed
	missedMax     = 30 * 24 * time.Hour
	dailyAuthor   = "Habit"
)

var plazaZone = func() *time.Location {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		log.Printf("New York time zone unavailable, using UTC: %v", err)
		return time.UTC
	}
	return loc
}()

// Questions of the day: pop culture people can argue about happily. Never
// asking for anything personal (teens are here too). Add to the list freely;
// a day keeps the question it was posted with.
var dailyQuestions = []string{
	"Which movie would you erase from your memory just to watch it fresh again?",
	"Best animated movie of all time? 🍿",
	"Marvel or DC? Defend your pick.",
	"Which fictional villain was actually kind of right?",
	"Best Pixar movie, go.",
	"What's a show everyone should binge at least once?",
	"Which video game world would you actually want to live in?",
	"What's your hottest Star Wars take?",
	"What song has been on repeat for you lately? 🎧",
	"Best anime opening of all time?",
	"Which sequel was better than the original?",
	"What meme will never get old?",
	"Minecraft, Fortnite or Roblox: which one wins?",
	"Which Nintendo character would make the best roommate?",
	"Who's the most underrated artist right now?",
	"One album on a desert island. Which one?",
	"What's the best movie soundtrack ever?",
	"Hogwarts, Xavier's School or Camp Half-Blood: where are you enrolling?",
	"If your life were a TV show, what would the theme song be?",
	"Which cartoon from when you were younger still holds up?",
	"Best Pokémon starter of all time? ⚡",
	"Which movie scene still gives you chills?",
	"What game deserves a sequel already?",
	"Which fictional friend group would you join?",
	"Best Spider-Man: Tobey, Andrew or Tom? 🕷️",
	"What internet trend are you glad is over?",
	"Which superpower would be the most annoying to actually have?",
	"What's the most rewatchable movie ever?",
	"Pick your entrance music. What song plays when you walk in?",
	"Best video game boss fight you've ever played?",
	"Which fictional food would you most want to try?",
	"Zelda or Mario: which series is better?",
	"What's the best Studio Ghibli movie?",
	"Which book should get a movie or show next?",
	"Which TV show got cancelled way too soon?",
	"Most iconic movie quote of all time?",
	"What's a game you'll always come back to?",
	"Which fictional pet would you adopt? 🐉",
	"Who wins in a fight: Goku or Superman?",
	"Which movie universe would you survive the longest in?",
	"What's the best cartoon theme song?",
	"Which video game character has the best outfit?",
	"Which reality show would you absolutely win?",
	"What's the most overrated movie? Be brave.",
	"Which song is guaranteed to get everyone dancing? 🪩",
	"Best plot twist you never saw coming? (No spoilers in your first line!)",
	"Which game would make the best movie?",
	"Which YouTuber or streamer do you never skip?",
	"Which Disney song is stuck in your head forever?",
	"Which movie would be better as a video game?",
	"Best duo in pop culture history?",
	"Which era of music would you live in?",
	"Which music video would you jump into?",
	"Which anime would you show someone who's never watched one?",
	"Gotham, Hyrule, Night City or Springfield: where are you moving?",
	"What's the best opening line of any movie or show?",
	"Which character deserved better?",
	"What's the best crossover that never happened?",
	"Which fictional world has the best food?",
	"What's a comfort show you could rewatch forever?",
}

// today is the day's shared state, guarded by Server.mu.
type today struct {
	day       string // plaza date, 2006-01-02
	question  string
	thread    int64 // the question's board thread
	wishes    int
	headlines []string
	meetup    bool // the meetup is on right now
	headAt    time.Time
}

type todayView struct {
	Day       string    `json:"day"`
	Question  string    `json:"question"`
	Thread    int64     `json:"thread"`
	Wishes    int       `json:"wishes"`
	Goal      int       `json:"goal"`
	MeetupAt  time.Time `json:"meetupAt"`
	MeetupEnd time.Time `json:"meetupEnd"`
	Meetup    bool      `json:"meetup"`
	Headlines []string  `json:"headlines"`
}

func plazaDay(t time.Time) string { return t.In(plazaZone).Format(time.DateOnly) }

// meetupStart is when the day's meetup begins.
func meetupStart(day string) time.Time {
	d, _ := time.ParseInLocation(time.DateOnly, day, plazaZone)
	return time.Date(d.Year(), d.Month(), d.Day(), meetupHour, 0, 0, 0, plazaZone)
}

// questionFor picks the day's question at random, but the same random pick
// for everyone all day (it's worked out from the date, so a restart doesn't
// change it), and never the same as the day before.
func questionFor(day string) string {
	pick := func(day string) int {
		h := fnv.New32a()
		h.Write([]byte("question|" + day))
		return int(h.Sum32() % uint32(len(dailyQuestions)))
	}
	i := pick(day)
	if d, err := time.Parse(time.DateOnly, day); err == nil && i == pick(d.AddDate(0, 0, -1).Format(time.DateOnly)) {
		i = (i + 1) % len(dailyQuestions)
	}
	return dailyQuestions[i]
}

// questionTitle is how the day's question is titled on the board.
func questionTitle(q string) string { return "💬 Today's question: " + q }

func (sv *Server) todayViewLocked() todayView {
	t := sv.today
	return todayView{
		Day: t.day, Question: t.question, Thread: t.thread, Wishes: t.wishes, Goal: wishGoal,
		MeetupAt: meetupStart(t.day), MeetupEnd: meetupStart(t.day).Add(meetupLength), Meetup: t.meetup,
		Headlines: nonNilStrings(t.headlines),
	}
}

func (sv *Server) broadcastAllLocked(v any) {
	for _, id := range sv.order {
		sv.broadcastLocked(sv.rooms[id], v, nil)
	}
}

// runDaily keeps the day's state current until ctx is cancelled.
func (sv *Server) runDaily(ctx context.Context) {
	t := time.NewTicker(dailyEvery)
	defer t.Stop()
	for {
		sv.dailyCheck(ctx)
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// dailyCheck starts a new day when the date changes, refreshes the
// headlines now and then, and starts and ends the meetup.
func (sv *Server) dailyCheck(ctx context.Context) {
	now := sv.opts.Now()
	day := plazaDay(now)
	sv.mu.Lock()
	newDay := sv.today.day != day
	stale := now.Sub(sv.today.headAt) >= headlineEvery
	sv.mu.Unlock()

	if newDay {
		next, err := sv.loadDay(ctx, day, now)
		if err != nil {
			log.Printf("start day %s: %v", day, err)
			return
		}
		sv.mu.Lock()
		sv.today = next
		sv.broadcastAllLocked(map[string]any{"t": "today", "today": sv.todayViewLocked()})
		sv.mu.Unlock()
	} else if stale {
		heads := sv.headlines(ctx, now)
		sv.mu.Lock()
		sv.today.headlines, sv.today.headAt = heads, now
		sv.mu.Unlock()
	}

	sv.mu.Lock()
	defer sv.mu.Unlock()
	sv.unloadIdleLocked(now)
	start := meetupStart(sv.today.day)
	on := !now.Before(start) && now.Before(start.Add(meetupLength))
	if on == sv.today.meetup {
		return
	}
	sv.today.meetup = on
	if on {
		sv.startMeetupLocked()
	}
	sv.broadcastAllLocked(map[string]any{"t": "today", "today": sv.todayViewLocked()})
}

// loadDay reads a day's progress back from the store, pinning its question
// to the Plaza's board the first time.
func (sv *Server) loadDay(ctx context.Context, day string, now time.Time) (today, error) {
	t := today{day: day, question: questionFor(day)}
	wishes, err := sv.store.Counter(ctx, "wishes:"+day)
	if err != nil {
		return t, err
	}
	t.wishes = int(wishes)
	threadKey := "daily-thread:" + day
	t.thread, err = sv.store.Counter(ctx, threadKey)
	if err != nil {
		return t, err
	}
	if t.thread != 0 {
		// The day's thread asks a different question (the list changed):
		// take it down, answers and all, and post today's question instead.
		old, err := sv.store.Thread(ctx, t.thread)
		switch {
		case errors.Is(err, store.ErrNotFound):
			t.thread = 0
		case err != nil:
			return t, err
		case old.Title != questionTitle(t.question):
			if err := sv.store.DeleteThread(ctx, old.ID); err != nil && !errors.Is(err, store.ErrNotFound) {
				return t, err
			}
			log.Printf("replaced the question of the day: %q", old.Title)
			sv.mu.Lock()
			if r := sv.rooms[old.Room]; r != nil {
				sv.broadcastLocked(r, map[string]any{"t": "board_removed", "thread": old.ID}, nil)
			}
			sv.mu.Unlock()
			t.thread = 0
		}
	}
	if t.thread == 0 {
		th := &store.Thread{
			Room: sv.order[0], Name: dailyAuthor, Title: questionTitle(t.question),
			Text:      "Answer below. A new question goes up every day, and they all stay here on the board.",
			CreatedAt: now,
		}
		if err := sv.store.CreateThread(ctx, th); err != nil {
			return t, err
		}
		if err := sv.store.SetCounter(ctx, threadKey, th.ID); err != nil {
			return t, err
		}
		t.thread = th.ID
		sv.mu.Lock()
		if r := sv.rooms[th.Room]; r != nil {
			sv.broadcastLocked(r, map[string]any{"t": "board_update", "thread": sv.threadView(th)}, nil)
		}
		sv.mu.Unlock()
	}
	t.headlines, t.headAt = sv.headlines(ctx, now), now
	return t, nil
}

// headlines are the hottest board posts across all rooms, for the newsbox.
func (sv *Server) headlines(ctx context.Context, now time.Time) []string {
	var all []store.Thread
	for _, id := range sv.order {
		ts, err := sv.store.Threads(ctx, id, store.SortHot, now, 5, 0)
		if err != nil {
			log.Printf("headlines: %v", err)
			return nil
		}
		all = append(all, ts...)
	}
	sort.SliceStable(all, func(i, j int) bool {
		return store.HotScore(&all[i], now) > store.HotScore(&all[j], now)
	})
	var out []string
	for _, t := range all {
		if t.Name == dailyAuthor && t.AccountID == "" {
			continue // the daily question is already front and centre
		}
		out = append(out, t.Title)
		if len(out) == 5 {
			break
		}
	}
	return out
}

// startMeetupLocked rings the Plaza's bell for everyone, with fireworks if
// the fountain was filled today.
func (sv *Server) startMeetupLocked() {
	plaza := sv.rooms[sv.order[0]]
	for i := range plaza.def.Props {
		if p := &plaza.def.Props[i]; p.Kind == world.PropBell {
			plaza.lastBell = sv.opts.Now()
			sv.broadcastLocked(plaza, map[string]any{"t": "fx", "prop": p.ID, "kind": p.Kind, "by": ""}, nil)
			break
		}
	}
	if sv.today.wishes >= wishGoal {
		sv.broadcastLocked(plaza, map[string]any{"t": "fireworks"}, nil)
	}
}

// wishedLocked counts a wish towards today's goal. Filling the fountain
// sets off fireworks over the Plaza and tells every room.
func (sv *Server) wishedLocked() {
	sv.today.wishes++
	go func(key string) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := sv.store.AddCounter(ctx, key, 1); err != nil {
			log.Printf("save wish: %v", err)
		}
	}("wishes:" + sv.today.day)
	if sv.today.wishes == wishGoal {
		sv.broadcastAllLocked(map[string]any{"t": "goal", "wishes": sv.today.wishes, "goal": wishGoal})
		sv.broadcastLocked(sv.rooms[sv.order[0]], map[string]any{"t": "fireworks"}, nil)
	}
}

type missedThread struct {
	ID      int64  `json:"id"`
	Room    string `json:"room"`
	Title   string `json:"title"`
	Replies int    `json:"replies"`
}

// handleMissed tells someone coming back what happened while they were
// away: new posts, busy threads, and how yesterday's fountain did.
func (sv *Server) handleMissed(ctx context.Context, s *session, since time.Time) {
	now := sv.opts.Now()
	if since.Unix() <= 0 || since.After(now) {
		return
	}
	if now.Sub(since) > missedMax {
		since = now.Add(-missedMax)
	}
	var posts int
	var busy []store.Thread
	for _, id := range sv.order {
		ts, err := sv.store.Threads(ctx, id, store.SortActive, now, 50, 0)
		if err != nil {
			sv.storeFailed(s, "missed", err)
			return
		}
		for _, t := range ts {
			if t.BumpedAt.Before(since) {
				break // latest activity first, so the rest are older
			}
			if t.Name == dailyAuthor && t.AccountID == "" {
				continue // the day's question has its own spot
			}
			if t.CreatedAt.After(since) {
				posts++
			}
			busy = append(busy, t)
		}
	}
	sort.SliceStable(busy, func(i, j int) bool { return store.HotScore(&busy[i], now) > store.HotScore(&busy[j], now) })
	top := []missedThread{}
	for _, t := range busy[:min(3, len(busy))] {
		top = append(top, missedThread{ID: t.ID, Room: t.Room, Title: t.Title, Replies: t.Replies})
	}
	yesterday := plazaDay(now.In(plazaZone).AddDate(0, 0, -1))
	wishes, err := sv.store.Counter(ctx, "wishes:"+yesterday)
	if err != nil {
		log.Printf("missed: %v", err)
	}
	sv.mu.Lock()
	defer sv.mu.Unlock()
	sv.enqueue(s, map[string]any{
		"t": "missed", "posts": posts, "active": len(busy), "top": top,
		"yesterdayWishes": wishes, "goal": wishGoal, "awayMinutes": int(now.Sub(since) / time.Minute),
	})
}

func nonNilStrings(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
