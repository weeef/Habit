// Package server is the realtime gateway: it owns room state, moves avatars,
// carries chat and serves the web client.
package server

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/coder/websocket"

	"habit/internal/store"
	"habit/internal/world"
)

const (
	maxChatRunes   = 280
	chatHistoryLen = 100
	peekRecent     = 3
	sendQueueLen   = 1024 // a busy 500-person room sends a lot at once
	newbieWindow   = 24 * time.Hour
)

var allowedEmotes = map[string]bool{"wave": true, "dance": true, "laugh": true, "heart": true, "sit": true}

// Options configures a Server. Zero values get sensible defaults.
type Options struct {
	Rooms  []*world.Room
	Store  store.Store   // nil keeps everything in memory
	Static fs.FS         // web client files; nil serves only /ws
	Tick   time.Duration // world update interval
	Step   time.Duration // time to walk one tile
	Now    func() time.Time
}

// Server holds the whole world. One mutex guards all room and session state;
// that's plenty for a prototype and keeps door transfers simple.
type Server struct {
	opts     Options
	store    store.Store
	accounts *Accounts
	secret   []byte

	mu         sync.Mutex
	rooms      map[string]*room
	order      []string
	lastCounts string
}

type room struct {
	def      *world.Room
	sessions map[*session]struct{}
	history  []chatMsg
	uses     map[string]int // per prop, e.g. wishes in the fountain (since start)
	lastBell time.Time
}

type session struct {
	send chan []byte
	done chan struct{}
	stop sync.Once

	// Everything below is guarded by Server.mu.
	guestKey   string
	acct       *store.Account
	named      bool
	style      store.Style // the account's character, or a random one for guests
	room       *room
	pubID      string // fresh on every room entry, so avatars can't be followed between rooms
	pos        world.Point
	path       []world.Point
	walked     time.Duration
	limiter    bucket
	threadLim  bucket
	replyLim   bucket
	useLim     bucket
	voteLim    bucket
	sentFirst  bool
	viewing    int64  // board thread open in the client, for live replies
	seat       string // bench prop they're sitting on
	pendingUse string // prop to use when they finish walking
}

func (s *session) close() { s.stop.Do(func() { close(s.done) }) }

// New builds a server and loads each room's recent chat from the store.
func New(ctx context.Context, opts Options) (*Server, error) {
	if opts.Rooms == nil {
		opts.Rooms = world.DefaultRooms()
	}
	if err := world.Validate(opts.Rooms); err != nil {
		return nil, err
	}
	if opts.Store == nil {
		opts.Store = store.NewMemory()
	}
	if opts.Tick == 0 {
		opts.Tick = 50 * time.Millisecond
	}
	if opts.Step == 0 {
		opts.Step = 220 * time.Millisecond
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	secret, err := opts.Store.LoadOrStoreSecret(ctx, randomBytes(32))
	if err != nil {
		return nil, fmt.Errorf("loading secret: %w", err)
	}
	sv := &Server{
		opts:     opts,
		store:    opts.Store,
		accounts: NewAccounts(opts.Store),
		secret:   secret,
		rooms:    map[string]*room{},
	}
	for _, def := range opts.Rooms {
		saved, err := opts.Store.RecentMessages(ctx, def.ID, chatHistoryLen)
		if err != nil {
			return nil, fmt.Errorf("loading %s history: %w", def.ID, err)
		}
		r := &room{def: def, sessions: map[*session]struct{}{}, uses: map[string]int{}}
		for _, m := range saved {
			r.history = append(r.history, chatMsg{From: m.AvatarID, Name: m.Name, Text: m.Text, TS: m.SentAt, Newbie: m.Newbie})
		}
		sv.rooms[def.ID] = r
		sv.order = append(sv.order, def.ID)
	}
	return sv, nil
}

// Handler serves the client at / and the realtime socket at /ws.
func (sv *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", sv.serveWS)
	if sv.opts.Static != nil {
		mux.Handle("/", http.FileServerFS(sv.opts.Static))
	}
	return mux
}

// Run advances avatar movement until ctx is cancelled.
func (sv *Server) Run(ctx context.Context) {
	t := time.NewTicker(sv.opts.Tick)
	defer t.Stop()
	last := sv.opts.Now()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			now := sv.opts.Now()
			sv.tick(now.Sub(last))
			last = now
		}
	}
}

// ---- wire format ----

type inMsg struct {
	T         string       `json:"t"`
	X         int          `json:"x"`
	Y         int          `json:"y"`
	Text      string       `json:"text"`
	Room      string       `json:"room"`
	Emote     string       `json:"emote"`
	On        bool         `json:"on"`
	Token     string       `json:"token"`
	Username  string       `json:"username"`
	Password  string       `json:"password"`
	BirthYear int          `json:"birthYear"`
	Thread    int64        `json:"thread"`
	Title     string       `json:"title"`
	Page      int          `json:"page"`
	Prop      string       `json:"prop"`
	Sort      string       `json:"sort"`
	Value     int          `json:"value"`
	Style     *store.Style `json:"style"`
}

type avatarView struct {
	ID     string      `json:"id"`
	Name   string      `json:"name"`
	Style  store.Style `json:"style"`
	Pos    world.Point `json:"pos"`
	Seat   string      `json:"seat,omitempty"`
	Guest  bool        `json:"guest"`
	Newbie bool        `json:"newbie"`
	You    bool        `json:"you,omitempty"`
}

type chatMsg struct {
	From   string    `json:"from"`
	Name   string    `json:"name"`
	Text   string    `json:"text"`
	TS     time.Time `json:"ts"`
	Newbie bool      `json:"newbie"`
}

type doorView struct {
	Pos    world.Point `json:"pos"`
	Target string      `json:"target"`
	Label  string      `json:"label"`
}

type roomView struct {
	ID    string     `json:"id"`
	Name  string     `json:"name"`
	Kind  world.Kind `json:"kind"`
	Theme string     `json:"theme"`
	Rows  []string   `json:"rows"`
	Doors []doorView `json:"doors"`
	Props []propView `json:"props"`
}

type roomSummary struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Kind  string `json:"kind"`
	Count int    `json:"count"`
}

// ---- connection lifecycle ----

func (sv *Server) serveWS(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	conn.SetReadLimit(4096)
	ctx := r.Context()

	s := &session{
		send:      make(chan []byte, sendQueueLen),
		done:      make(chan struct{}),
		guestKey:  "g:" + hex.EncodeToString(randomBytes(8)),
		limiter:   newBucket(chatBurst, chatPerSec),
		threadLim: newBucket(threadBurst, threadPerSec),
		replyLim:  newBucket(replyBurst, replyPerSec),
		useLim:    newBucket(useBurst, usePerSec),
		voteLim:   newBucket(voteBurst, votePerSec),
		style:     randomStyle(),
	}
	go sv.writeLoop(ctx, conn, s)

	sv.mu.Lock()
	plaza := sv.rooms[sv.order[0]]
	sv.enqueue(s, map[string]any{"t": "welcome", "rooms": sv.summariesLocked()})
	sv.enterLocked(s, plaza, plaza.def.Spawn)
	sv.mu.Unlock()

	defer func() {
		sv.mu.Lock()
		sv.leaveLocked(s)
		sv.mu.Unlock()
		s.close()
		conn.CloseNow()
	}()

	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			return
		}
		var m inMsg
		if json.Unmarshal(data, &m) != nil {
			continue
		}
		select {
		case <-s.done:
			return // too slow to keep up; dropped
		default:
		}
		sv.handle(ctx, s, m)
	}
}

func (sv *Server) writeLoop(ctx context.Context, conn *websocket.Conn, s *session) {
	for {
		select {
		case <-s.done:
			conn.CloseNow()
			return
		case b := <-s.send:
			wctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			err := conn.Write(wctx, websocket.MessageText, b)
			cancel()
			if err != nil {
				s.close()
				conn.CloseNow()
				return
			}
		}
	}
}

// enqueue sends without blocking; a client that can't keep up is dropped.
func (sv *Server) enqueue(s *session, v any) {
	b, err := json.Marshal(v)
	if err != nil {
		log.Printf("marshal: %v", err)
		return
	}
	select {
	case s.send <- b:
	default:
		s.close()
	}
}

func (sv *Server) broadcastLocked(r *room, v any, except *session) {
	for s := range r.sessions {
		if s != except {
			sv.enqueue(s, v)
		}
	}
}

// ---- message handling ----

func (sv *Server) handle(ctx context.Context, s *session, m inMsg) {
	// Password hashing is slow on purpose and the store may be a network
	// away, so both happen outside the world lock.
	switch m.T {
	case "signup":
		sv.mu.Lock()
		style := s.style // keep the guest's look unless they designed one
		sv.mu.Unlock()
		if m.Style != nil {
			style = *m.Style
		}
		acct, token, recovery, err := sv.accounts.Signup(ctx, strings.TrimSpace(m.Username), m.Password, m.BirthYear, style, sv.opts.Now())
		sv.finishAuth(s, acct, token, recovery, err)
		return
	case "login":
		acct, token, err := sv.accounts.Login(ctx, strings.TrimSpace(m.Username), m.Password, sv.opts.Now())
		sv.finishAuth(s, acct, token, "", err)
		if err == nil {
			sv.sendNotes(ctx, s, acct)
		}
		return
	case "resume":
		acct, err := sv.accounts.ByToken(ctx, m.Token)
		if acct != nil || err != nil {
			sv.finishAuth(s, acct, m.Token, "", err)
		}
		if acct != nil {
			sv.sendNotes(ctx, s, acct)
		}
		return
	case "board":
		sv.handleBoard(ctx, s, m.Room, m.Page, store.ThreadSort(m.Sort))
		return
	case "thread":
		sv.handleThread(ctx, s, m.Thread)
		return
	case "post_thread":
		sv.handlePostThread(ctx, s, m.Title, m.Text)
		return
	case "post_reply":
		sv.handlePostReply(ctx, s, m.Thread, m.Text)
		return
	case "vote":
		sv.handleVote(ctx, s, m.Thread, m.Value)
		return
	case "notes_read":
		sv.handleNotesRead(ctx, s)
		return
	case "chat":
		sv.mu.Lock()
		saved, ok := sv.chatLocked(s, m.Text)
		sv.mu.Unlock()
		if ok {
			if err := sv.store.AddMessage(ctx, saved); err != nil {
				log.Printf("save message: %v", err)
			}
		}
		return
	}

	sv.mu.Lock()
	defer sv.mu.Unlock()
	if s.room == nil {
		return
	}
	switch m.T {
	case "walk":
		sv.walkLocked(s, world.Point{X: m.X, Y: m.Y})
	case "use":
		sv.handleUseLocked(s, m.Prop)
	case "emote":
		if allowedEmotes[m.Emote] {
			sv.broadcastLocked(s.room, map[string]any{"t": "emote", "id": s.pubID, "emote": m.Emote}, nil)
		}
	case "named":
		if s.acct != nil {
			s.named = m.On
			sv.broadcastLocked(s.room, map[string]any{"t": "update", "avatar": sv.viewLocked(s)}, nil)
		}
	case "teleport":
		if r, ok := sv.rooms[m.Room]; ok && r != s.room {
			sv.moveLocked(s, r, r.def.Spawn)
		}
	case "peek":
		if r, ok := sv.rooms[m.Room]; ok {
			recent := nonNil(r.history[max(0, len(r.history)-peekRecent):])
			sv.enqueue(s, map[string]any{"t": "peek", "room": r.def.ID, "name": r.def.Name, "count": len(r.sessions), "recent": recent})
		}
	}
}

func (sv *Server) finishAuth(s *session, acct *store.Account, token, recovery string, err error) {
	sv.mu.Lock()
	defer sv.mu.Unlock()
	if err != nil {
		sv.enqueue(s, map[string]any{"t": "auth_error", "message": err.Error()})
		return
	}
	s.acct = acct
	s.style = acct.Style
	sv.enqueue(s, map[string]any{
		"t": "authed", "username": acct.Username, "token": token,
		"recoveryCode": recovery, "ageBand": ageBand(acct.BirthYear, sv.opts.Now()),
		"style": acct.Style,
	})
	if s.room != nil {
		// Their anonymous tag changes from "Visitor" to "Anonymous", and
		// they take on their account's look.
		sv.broadcastLocked(s.room, map[string]any{"t": "update", "avatar": sv.viewLocked(s)}, nil)
	}
}

func (sv *Server) walkLocked(s *session, to world.Point) {
	path := sv.routeLocked(s, to)
	if path == nil {
		return
	}
	sv.standUpLocked(s)
	sv.setPathLocked(s, path)
}

// routeLocked finds a path for s to `to`. Someone already walking first
// finishes the step they're on, so changing course mid-stride carries on
// smoothly instead of snapping back to the last tile. It returns nil if
// there's no way there.
func (sv *Server) routeLocked(s *session, to world.Point) []world.Point {
	if len(s.path) == 0 {
		return world.FindPath(s.room.def, s.pos, to)
	}
	next := s.path[0]
	if s.room.def.DoorAt(next) != nil {
		return nil // already stepping through a door
	}
	if next == to {
		return []world.Point{next}
	}
	rest := world.FindPath(s.room.def, next, to)
	if rest == nil {
		return nil
	}
	return append([]world.Point{next}, rest...)
}

// setPathLocked starts s walking along path and tells the room. A path
// that begins with the step s is already taking keeps its progress.
func (sv *Server) setPathLocked(s *session, path []world.Point) {
	if len(s.path) == 0 || len(path) == 0 || path[0] != s.path[0] {
		s.walked = 0
	}
	s.path = path
	sv.broadcastLocked(s.room, map[string]any{
		"t": "path", "id": s.pubID, "from": s.pos, "path": path,
		"stepMs": sv.opts.Step.Milliseconds(), "walkedMs": s.walked.Milliseconds(),
	}, nil)
}

// stepTime is how long a step from a to b takes; diagonals cover more
// ground, so they take longer.
func (sv *Server) stepTime(a, b world.Point) time.Duration {
	if world.Diagonal(a, b) {
		return time.Duration(float64(sv.opts.Step) * math.Sqrt2)
	}
	return sv.opts.Step
}

// chatLocked posts a message to s's room and returns it for saving, or
// false if it was refused.
func (sv *Server) chatLocked(s *session, text string) (store.Message, bool) {
	if s.room == nil {
		return store.Message{}, false
	}
	if s.acct == nil {
		// Guests can look around, walk and wave; talking needs a name.
		sv.enqueue(s, map[string]any{"t": "need_account"})
		return store.Message{}, false
	}
	text = strings.TrimSpace(text)
	if text == "" || !utf8.ValidString(text) {
		return store.Message{}, false
	}
	if utf8.RuneCountInString(text) > maxChatRunes {
		sv.enqueue(s, map[string]any{"t": "notice", "message": fmt.Sprintf("Messages can be up to %d characters.", maxChatRunes)})
		return store.Message{}, false
	}
	now := sv.opts.Now()
	if ageBand(s.acct.BirthYear, now) != BandAdult {
		if reason := checkPersonalInfo(text); reason != "" {
			sv.enqueue(s, map[string]any{"t": "notice", "message": reason})
			return store.Message{}, false
		}
	}
	if !s.limiter.allow(now) {
		sv.enqueue(s, map[string]any{"t": "notice", "message": "Whoa, slow down a little."})
		return store.Message{}, false
	}

	msg := chatMsg{From: s.pubID, Name: sv.displayNameLocked(s), Text: text, TS: now, Newbie: sv.newbieLocked(s)}
	r := s.room
	r.history = append(r.history, msg)
	if len(r.history) > chatHistoryLen {
		r.history = r.history[len(r.history)-chatHistoryLen:]
	}
	sv.broadcastLocked(r, map[string]any{"t": "chat", "msg": msg}, nil)

	if !s.sentFirst {
		s.sentFirst = true
		sv.enqueue(s, map[string]any{"t": "first_message"})
	}
	return store.Message{
		Room: r.def.ID, AccountID: s.acct.ID, AvatarID: msg.From, Name: msg.Name,
		Text: msg.Text, Newbie: msg.Newbie, SentAt: msg.TS,
	}, true
}

// ---- rooms and movement ----

func (sv *Server) tick(dt time.Duration) {
	sv.mu.Lock()
	defer sv.mu.Unlock()

	type transfer struct {
		s    *session
		door *world.Door
		back world.Point
	}
	var transfers []transfer
	var arrived []*session
	for _, id := range sv.order {
		r := sv.rooms[id]
		for s := range r.sessions {
			if len(s.path) == 0 {
				continue
			}
			s.walked += dt
			door := false
			for len(s.path) > 0 && s.walked >= sv.stepTime(s.pos, s.path[0]) {
				s.walked -= sv.stepTime(s.pos, s.path[0])
				prev := s.pos
				s.pos, s.path = s.path[0], s.path[1:]
				if d := r.def.DoorAt(s.pos); d != nil {
					transfers = append(transfers, transfer{s, d, prev})
					s.path = nil
					door = true
				}
			}
			if len(s.path) == 0 {
				s.walked = 0
				if !door {
					arrived = append(arrived, s)
				}
			}
		}
	}
	for _, s := range arrived {
		sv.arrivedLocked(s)
	}
	for _, t := range transfers {
		target := sv.rooms[t.door.Target]
		if len(target.sessions) >= target.def.Capacity {
			// Full: step back off the door instead of going through.
			t.s.pos = t.back
			sv.broadcastLocked(t.s.room, map[string]any{"t": "path", "id": t.s.pubID, "from": t.back, "path": []world.Point{}, "stepMs": 0}, nil)
			sv.enqueue(t.s, map[string]any{"t": "notice", "message": target.def.Name + " is full right now. Try again in a moment."})
			continue
		}
		sv.moveLocked(t.s, target, t.door.TargetAt)
	}
	sv.broadcastCountsLocked()
}

func (sv *Server) moveLocked(s *session, to *room, at world.Point) {
	sv.leaveLocked(s)
	sv.enterLocked(s, to, at)
	sv.broadcastCountsLocked()
}

// enterLocked puts s into r. Arriving is silent: no "X joined" chat line.
func (sv *Server) enterLocked(s *session, r *room, at world.Point) {
	s.room = r
	s.pos = sv.freeTileNearLocked(r, at)
	s.path = nil
	s.walked = 0
	s.seat = ""
	s.pendingUse = ""
	s.pubID = hex.EncodeToString(randomBytes(6))
	r.sessions[s] = struct{}{}

	avatars := make([]avatarView, 0, len(r.sessions))
	for o := range r.sessions {
		v := sv.viewLocked(o)
		v.You = o == s
		avatars = append(avatars, v)
	}
	doors := make([]doorView, len(r.def.Doors))
	for i, d := range r.def.Doors {
		doors[i] = doorView{Pos: d.At, Target: d.Target, Label: d.Label}
	}
	sv.enqueue(s, map[string]any{
		"t": "room",
		"room": roomView{
			ID: r.def.ID, Name: r.def.Name, Kind: r.def.Kind, Theme: r.def.Theme, Rows: r.def.Rows, Doors: doors,
			Props: r.propViews(),
		},
		"you":     s.pubID,
		"avatars": avatars,
		"history": nonNil(r.history),
	})
	sv.broadcastLocked(r, map[string]any{"t": "enter", "avatar": sv.viewLocked(s)}, s)
}

// freeTileNearLocked returns the nearest unoccupied plain floor tile to p, so
// people arriving together don't stack on one spot. It falls back to p.
func (sv *Server) freeTileNearLocked(r *room, p world.Point) world.Point {
	taken := map[world.Point]bool{}
	for o := range r.sessions {
		taken[o.pos] = true
	}
	for radius := 0; radius <= r.def.W()+r.def.H(); radius++ { // widen until there's room
		for dy := -radius; dy <= radius; dy++ {
			for dx := -radius; dx <= radius; dx++ {
				q := world.Point{X: p.X + dx, Y: p.Y + dy}
				if max(abs(dx), abs(dy)) == radius && r.def.Standable(q) && !taken[q] {
					return q
				}
			}
		}
	}
	return p
}

// nonNil makes an empty history encode as [] rather than null, which clients
// would have to special-case.
func nonNil(h []chatMsg) []chatMsg {
	if h == nil {
		return []chatMsg{}
	}
	return h
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// leaveLocked removes s from its room. Leaving is silent too.
func (sv *Server) leaveLocked(s *session) {
	if s.room == nil {
		return
	}
	delete(s.room.sessions, s)
	sv.broadcastLocked(s.room, map[string]any{"t": "leave", "id": s.pubID}, nil)
	s.room = nil
}

func (sv *Server) broadcastCountsLocked() {
	counts := map[string]int{}
	var key strings.Builder
	for _, id := range sv.order {
		n := len(sv.rooms[id].sessions)
		counts[id] = n
		fmt.Fprintf(&key, "%s=%d;", id, n)
	}
	if key.String() == sv.lastCounts {
		return
	}
	sv.lastCounts = key.String()
	msg := map[string]any{"t": "counts", "counts": counts}
	for _, id := range sv.order {
		sv.broadcastLocked(sv.rooms[id], msg, nil)
	}
}

func (sv *Server) summariesLocked() []roomSummary {
	out := make([]roomSummary, 0, len(sv.order))
	for _, id := range sv.order {
		r := sv.rooms[id]
		out = append(out, roomSummary{ID: id, Name: r.def.Name, Kind: string(r.def.Kind), Count: len(r.sessions)})
	}
	return out
}

func (sv *Server) viewLocked(s *session) avatarView {
	return avatarView{
		ID: s.pubID, Name: sv.displayNameLocked(s), Style: s.style, Pos: s.pos, Seat: s.seat,
		Guest: s.acct == nil, Newbie: sv.newbieLocked(s),
	}
}

// displayNameLocked is anonymous by default (PLAN.md §2.2): the username only
// shows when the user has opted in.
func (sv *Server) displayNameLocked(s *session) string {
	if s.acct != nil && s.named {
		return s.acct.Username
	}
	key, prefix := s.guestKey, "Visitor"
	if s.acct != nil {
		key, prefix = "u:"+s.acct.ID, "Anonymous"
	}
	return prefix + " #" + anonTag(sv.secret, key, s.room.def.ID, sv.opts.Now())
}

// newbieLocked drives the "first visit" badge so regulars know to say hi.
func (sv *Server) newbieLocked(s *session) bool {
	return s.acct == nil || sv.opts.Now().Sub(s.acct.CreatedAt) < newbieWindow
}
