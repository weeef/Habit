package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"
	"unicode/utf8"

	"habit/internal/store"
)

// Every room has a board: saved threads with replies, kept forever and
// listed by latest activity. Live chat is the fast layer; the board is what
// stays. Reading is open to guests; posting needs an account.

const (
	maxTitleRunes = 100
	maxPostRunes  = 2000
	boardPageSize = 20
	maxBoardPage  = 500
	maxReplies    = 500                  // a thread locks when full...
	threadLockAge = 180 * 24 * time.Hour // ...or six months after it was started
	notesLimit    = 50
	snippetRunes  = 120

	threadBurst  = 2
	threadPerSec = 1.0 / 300 // a new thread every 5 minutes after the first two
	replyBurst   = 5
	replyPerSec  = 1.0 / 15
	voteBurst    = 10
	votePerSec   = 2
)

type threadView struct {
	ID      int64     `json:"id"`
	Room    string    `json:"room"`
	Title   string    `json:"title"`
	Name    string    `json:"name"`
	Text    string    `json:"text"`
	TS      time.Time `json:"ts"`
	Bumped  time.Time `json:"bumped"`
	Replies int       `json:"replies"`
	Locked  bool      `json:"locked"`
	Ups     int       `json:"ups"`
	Downs   int       `json:"downs"`
	Mine    int       `json:"mine,omitempty"` // the viewer's vote: 1, -1 or 0
}

type replyView struct {
	ID   int64     `json:"id"`
	Name string    `json:"name"`
	Text string    `json:"text"`
	TS   time.Time `json:"ts"`
	OP   bool      `json:"op"` // written by the thread's author
}

type noteView struct {
	ID      int64     `json:"id"`
	Thread  int64     `json:"thread"`
	Room    string    `json:"room"`
	Title   string    `json:"title"`
	Snippet string    `json:"snippet"`
	TS      time.Time `json:"ts"`
	Read    bool      `json:"read"`
}

// postName is how a post's author is shown: their username if they chose to
// show it, otherwise an anonymous tag that's stable within the thread.
func (sv *Server) postName(accountID, shown string, threadID int64) string {
	switch {
	case shown != "":
		return shown
	case accountID == "":
		return "Anonymous" // account deleted
	default:
		return "Anonymous #" + postTag(sv.secret, accountID, threadID)
	}
}

func locked(t *store.Thread, now time.Time) bool {
	return t.Replies >= maxReplies || now.Sub(t.CreatedAt) >= threadLockAge
}

func (sv *Server) threadView(t *store.Thread) threadView {
	return threadView{
		ID: t.ID, Room: t.Room, Title: t.Title, Name: sv.postName(t.AccountID, t.Name, t.ID),
		Text: t.Text, TS: t.CreatedAt, Bumped: t.BumpedAt, Replies: t.Replies, Locked: locked(t, sv.opts.Now()),
		Ups: t.Ups, Downs: t.Downs,
	}
}

// myVotes fills in the viewer's own votes on a list of thread views.
func (sv *Server) myVotes(ctx context.Context, s *session, views []threadView) {
	sv.mu.Lock()
	acct := s.acct
	sv.mu.Unlock()
	if acct == nil || len(views) == 0 {
		return
	}
	ids := make([]int64, len(views))
	for i, v := range views {
		ids[i] = v.ID
	}
	mine, err := sv.store.VotesBy(ctx, acct.ID, ids)
	if err != nil {
		log.Printf("load votes: %v", err)
		return
	}
	for i := range views {
		views[i].Mine = mine[views[i].ID]
	}
}

// handleVote records a vote on a thread and shows the new totals to
// everyone in the thread's room and anyone reading it.
func (sv *Server) handleVote(ctx context.Context, s *session, threadID int64, value int) {
	if value < -1 || value > 1 {
		return
	}
	sv.mu.Lock()
	if s.acct == nil {
		sv.enqueue(s, map[string]any{"t": "need_account"})
		sv.mu.Unlock()
		return
	}
	if !s.voteLim.allow(sv.opts.Now()) {
		sv.mu.Unlock()
		return
	}
	acct := s.acct
	sv.mu.Unlock()

	t, err := sv.store.Thread(ctx, threadID)
	if errors.Is(err, store.ErrNotFound) {
		return
	}
	if err != nil {
		sv.storeFailed(s, "load thread", err)
		return
	}
	ups, downs, err := sv.store.Vote(ctx, threadID, acct.ID, value)
	if err != nil {
		sv.storeFailed(s, "vote", err)
		return
	}
	msg := map[string]any{"t": "votes", "thread": threadID, "ups": ups, "downs": downs}
	sv.mu.Lock()
	defer sv.mu.Unlock()
	sv.enqueue(s, map[string]any{"t": "votes", "thread": threadID, "ups": ups, "downs": downs, "mine": value})
	for _, id := range sv.order {
		room := sv.rooms[id]
		for o := range room.sessions {
			if o != s && (room.def.ID == t.Room || o.viewing == threadID) {
				sv.enqueue(o, msg)
			}
		}
	}
}

func (sv *Server) replyView(t *store.Thread, r *store.Reply) replyView {
	return replyView{
		ID: r.ID, Name: sv.postName(r.AccountID, r.Name, t.ID), Text: r.Text, TS: r.CreatedAt,
		OP: r.AccountID != "" && r.AccountID == t.AccountID,
	}
}

func noteViewOf(n store.Notification) noteView {
	snippet := n.Text
	if utf8.RuneCountInString(snippet) > snippetRunes {
		snippet = string([]rune(snippet)[:snippetRunes-1]) + "…"
	}
	return noteView{ID: n.ID, Thread: n.ThreadID, Room: n.Room, Title: n.Title, Snippet: snippet, TS: n.CreatedAt, Read: n.Read}
}

func (sv *Server) postError(s *session, message string) {
	sv.enqueue(s, map[string]any{"t": "post_error", "message": message})
}

// storeFailed logs err and tells the user something went wrong.
func (sv *Server) storeFailed(s *session, op string, err error) {
	log.Printf("%s: %v", op, err)
	sv.postError(s, ErrUnavailable.Error())
}

// ---- reading ----

func (sv *Server) handleBoard(ctx context.Context, s *session, roomID string, page int, sort store.ThreadSort) {
	sv.mu.Lock()
	r, ok := sv.rooms[roomID] // the board of a room you're in, so it's loaded
	sv.mu.Unlock()
	if !ok || page < 0 || page > maxBoardPage {
		return
	}
	if !store.ValidSort(sort) {
		sort = store.SortActive
	}
	threads, err := sv.store.Threads(ctx, roomID, sort, sv.opts.Now(), boardPageSize+1, page*boardPageSize)
	if err != nil {
		sv.storeFailed(s, "list threads", err)
		return
	}
	more := len(threads) > boardPageSize
	views := make([]threadView, 0, boardPageSize)
	for i := range threads[:min(len(threads), boardPageSize)] {
		views = append(views, sv.threadView(&threads[i]))
	}
	sv.myVotes(ctx, s, views)
	sv.mu.Lock()
	s.viewing = 0
	sv.mu.Unlock()
	sv.enqueue(s, map[string]any{"t": "board", "room": roomID, "name": r.def.Name, "page": page, "sort": sort, "threads": views, "more": more})
}

func (sv *Server) handleThread(ctx context.Context, s *session, id int64) {
	t, err := sv.store.Thread(ctx, id)
	if errors.Is(err, store.ErrNotFound) {
		sv.postError(s, "That post isn't here any more.")
		return
	}
	if err != nil {
		sv.storeFailed(s, "load thread", err)
		return
	}
	replies, err := sv.store.Replies(ctx, id)
	if err != nil {
		sv.storeFailed(s, "load replies", err)
		return
	}
	views := make([]replyView, len(replies))
	for i := range replies {
		views[i] = sv.replyView(t, &replies[i])
	}
	tv := []threadView{sv.threadView(t)}
	sv.myVotes(ctx, s, tv)
	sv.mu.Lock()
	s.viewing = id
	sv.mu.Unlock()
	sv.enqueue(s, map[string]any{"t": "thread", "thread": tv[0], "replies": views})
}

// ---- posting ----

// postAllowedLocked runs the checks every post shares: an account, the teen
// filter and a rate limit. It tells the user why if one fails.
func (sv *Server) postAllowedLocked(s *session, limit *bucket, texts ...string) bool {
	if s.acct == nil {
		sv.enqueue(s, map[string]any{"t": "need_account"})
		return false
	}
	if sv.mutedLocked(s) {
		return false
	}
	now := sv.opts.Now()
	if ageBand(s.acct.BirthYear, now) != BandAdult {
		for _, text := range texts {
			if reason := checkPersonalInfo(text); reason != "" {
				sv.postError(s, reason)
				return false
			}
		}
	}
	if !limit.allow(now) {
		sv.postError(s, "You're posting a lot. Give it a few minutes.")
		return false
	}
	return true
}

func validText(text string, maxRunes int) bool {
	return text != "" && utf8.ValidString(text) && utf8.RuneCountInString(text) <= maxRunes
}

func (sv *Server) handlePostThread(ctx context.Context, s *session, title, text string) {
	title, text = strings.TrimSpace(title), strings.TrimSpace(text)
	if !validText(title, maxTitleRunes) || !validText(text, maxPostRunes) {
		sv.postError(s, fmt.Sprintf("Posts need a title (up to %d characters) and some text (up to %d).", maxTitleRunes, maxPostRunes))
		return
	}

	sv.mu.Lock()
	if s.room == nil || !sv.postAllowedLocked(s, &s.threadLim, title, text) {
		sv.mu.Unlock()
		return
	}
	// New threads go on the board of the room you're standing in.
	t := &store.Thread{Room: s.room.def.ID, AccountID: s.acct.ID, Title: title, Text: text, CreatedAt: sv.opts.Now()}
	if s.named {
		t.Name = s.acct.Username
	}
	sv.mu.Unlock()

	if err := sv.store.CreateThread(ctx, t); err != nil {
		sv.storeFailed(s, "create thread", err)
		return
	}
	v := sv.threadView(t)

	sv.mu.Lock()
	defer sv.mu.Unlock()
	s.viewing = t.ID
	sv.enqueue(s, map[string]any{"t": "posted_thread", "thread": v})
	sv.broadcastLocked(sv.rooms[t.Room], map[string]any{"t": "board_update", "thread": v}, s)
}

func (sv *Server) handlePostReply(ctx context.Context, s *session, threadID int64, text string) {
	text = strings.TrimSpace(text)
	if !validText(text, maxPostRunes) {
		sv.postError(s, fmt.Sprintf("Replies can be up to %d characters.", maxPostRunes))
		return
	}
	t, err := sv.store.Thread(ctx, threadID)
	if errors.Is(err, store.ErrNotFound) {
		sv.postError(s, "That post isn't here any more.")
		return
	}
	if err != nil {
		sv.storeFailed(s, "load thread", err)
		return
	}
	if locked(t, sv.opts.Now()) {
		sv.postError(s, "This thread is locked, so it can't take new replies. Start a new one?")
		return
	}

	sv.mu.Lock()
	if !sv.postAllowedLocked(s, &s.replyLim, text) {
		sv.mu.Unlock()
		return
	}
	r := &store.Reply{ThreadID: t.ID, AccountID: s.acct.ID, Text: text, CreatedAt: sv.opts.Now()}
	if s.named {
		r.Name = s.acct.Username
	}
	sv.mu.Unlock()

	notes, err := sv.store.AddReply(ctx, r)
	if errors.Is(err, store.ErrNotFound) {
		sv.postError(s, "That post isn't here any more.")
		return
	}
	if err != nil {
		sv.storeFailed(s, "add reply", err)
		return
	}
	t.Replies++
	t.BumpedAt = r.CreatedAt
	tv, rv := sv.threadView(t), sv.replyView(t, r)

	sv.mu.Lock()
	defer sv.mu.Unlock()
	if s.viewing != t.ID {
		sv.enqueue(s, map[string]any{"t": "reply", "thread": tv, "reply": rv})
	}
	byAccount := map[string][]noteView{}
	for _, n := range notes {
		byAccount[n.AccountID] = append(byAccount[n.AccountID], noteViewOf(n))
	}
	for _, id := range sv.order {
		room := sv.rooms[id]
		for o := range room.sessions {
			if o.viewing == t.ID {
				sv.enqueue(o, map[string]any{"t": "reply", "thread": tv, "reply": rv})
			}
			if room.def.ID == t.Room {
				sv.enqueue(o, map[string]any{"t": "board_update", "thread": tv})
			}
			if o.acct != nil {
				for _, n := range byAccount[o.acct.ID] {
					sv.enqueue(o, map[string]any{"t": "notify", "note": n})
				}
			}
		}
	}
}

// ---- notifications ----

func (sv *Server) sendNotes(ctx context.Context, s *session, acct *store.Account) {
	notes, err := sv.store.Notifications(ctx, acct.ID, notesLimit)
	if err != nil {
		log.Printf("load notifications: %v", err)
		return
	}
	views := make([]noteView, len(notes))
	for i, n := range notes {
		views[i] = noteViewOf(n)
	}
	sv.enqueue(s, map[string]any{"t": "notes", "notes": views})
}

func (sv *Server) handleNotesRead(ctx context.Context, s *session) {
	sv.mu.Lock()
	acct := s.acct
	sv.mu.Unlock()
	if acct == nil {
		return
	}
	if err := sv.store.MarkNotificationsRead(ctx, acct.ID); err != nil {
		log.Printf("mark notifications read: %v", err)
	}
}
