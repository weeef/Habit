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
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/coder/websocket"

	"habit/internal/world"
)

const (
	maxChatRunes   = 280
	chatHistoryLen = 100
	peekRecent     = 3
	sendQueueLen   = 64
	newbieWindow   = 24 * time.Hour
)

var avatarColors = []string{"#e4572e", "#29335c", "#f3a712", "#a8c686", "#669bbc", "#9b5de5", "#f15bb5", "#00bbf9"}

var allowedEmotes = map[string]bool{"wave": true, "dance": true, "laugh": true, "heart": true, "sit": true}

// Options configures a Server. Zero values get sensible defaults.
type Options struct {
	Rooms  []*world.Room
	Static fs.FS         // web client files; nil serves only /ws
	Tick   time.Duration // world update interval
	Step   time.Duration // time to walk one tile
	Now    func() time.Time
}

// Server holds the whole world. One mutex guards all room and session state;
// that's plenty for a prototype and keeps door transfers simple.
type Server struct {
	opts     Options
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
}

type session struct {
	send chan []byte
	done chan struct{}
	stop sync.Once

	// Everything below is guarded by Server.mu.
	guestKey  string
	acct      *Account
	named     bool
	color     string
	room      *room
	pubID     string // fresh on every room entry, so avatars can't be followed between rooms
	pos       world.Point
	path      []world.Point
	walked    time.Duration
	limiter   bucket
	sentFirst bool
}

func (s *session) close() { s.stop.Do(func() { close(s.done) }) }

// New builds a server. It panics on an invalid room set, which is a
// programming error.
func New(opts Options) *Server {
	if opts.Rooms == nil {
		opts.Rooms = world.DefaultRooms()
	}
	if err := world.Validate(opts.Rooms); err != nil {
		panic(err)
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
	sv := &Server{
		opts:     opts,
		accounts: NewAccounts(),
		secret:   randomBytes(32),
		rooms:    map[string]*room{},
	}
	for _, def := range opts.Rooms {
		sv.rooms[def.ID] = &room{def: def, sessions: map[*session]struct{}{}}
		sv.order = append(sv.order, def.ID)
	}
	return sv
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
	T         string `json:"t"`
	X         int    `json:"x"`
	Y         int    `json:"y"`
	Text      string `json:"text"`
	Room      string `json:"room"`
	Emote     string `json:"emote"`
	On        bool   `json:"on"`
	Token     string `json:"token"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	BirthYear int    `json:"birthYear"`
}

type avatarView struct {
	ID     string      `json:"id"`
	Name   string      `json:"name"`
	Color  string      `json:"color"`
	Pos    world.Point `json:"pos"`
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
		send:     make(chan []byte, sendQueueLen),
		done:     make(chan struct{}),
		guestKey: "g:" + hex.EncodeToString(randomBytes(8)),
	}
	go sv.writeLoop(ctx, conn, s)

	sv.mu.Lock()
	s.color = avatarColors[int(randomBytes(1)[0])%len(avatarColors)]
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
		sv.handle(s, m)
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

func (sv *Server) handle(s *session, m inMsg) {
	// Password hashing is slow on purpose, so do it outside the world lock.
	switch m.T {
	case "signup":
		acct, token, recovery, err := sv.accounts.Signup(strings.TrimSpace(m.Username), m.Password, m.BirthYear, sv.opts.Now())
		sv.finishAuth(s, acct, token, recovery, err)
		return
	case "login":
		acct, token, err := sv.accounts.Login(strings.TrimSpace(m.Username), m.Password)
		sv.finishAuth(s, acct, token, "", err)
		return
	case "resume":
		if acct := sv.accounts.ByToken(m.Token); acct != nil {
			sv.finishAuth(s, acct, m.Token, "", nil)
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
	case "chat":
		sv.chatLocked(s, m.Text)
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
			recent := r.history[max(0, len(r.history)-peekRecent):]
			sv.enqueue(s, map[string]any{"t": "peek", "room": r.def.ID, "name": r.def.Name, "count": len(r.sessions), "recent": recent})
		}
	}
}

func (sv *Server) finishAuth(s *session, acct *Account, token, recovery string, err error) {
	sv.mu.Lock()
	defer sv.mu.Unlock()
	if err != nil {
		sv.enqueue(s, map[string]any{"t": "auth_error", "message": err.Error()})
		return
	}
	s.acct = acct
	sv.enqueue(s, map[string]any{
		"t": "authed", "username": acct.Username, "token": token,
		"recoveryCode": recovery, "ageBand": acct.AgeBand(sv.opts.Now()),
	})
	if s.room != nil {
		// Their anonymous tag changes from "Visitor" to "Anonymous".
		sv.broadcastLocked(s.room, map[string]any{"t": "update", "avatar": sv.viewLocked(s)}, nil)
	}
}

func (sv *Server) walkLocked(s *session, to world.Point) {
	path := world.FindPath(s.room.def, s.pos, to)
	if path == nil {
		return
	}
	s.path = path
	s.walked = 0
	sv.broadcastLocked(s.room, map[string]any{
		"t": "path", "id": s.pubID, "from": s.pos, "path": path,
		"stepMs": sv.opts.Step.Milliseconds(),
	}, nil)
}

func (sv *Server) chatLocked(s *session, text string) {
	if s.acct == nil {
		// Guests can look around, walk and wave; talking needs a name.
		sv.enqueue(s, map[string]any{"t": "need_account"})
		return
	}
	text = strings.TrimSpace(text)
	if text == "" || !utf8.ValidString(text) {
		return
	}
	if utf8.RuneCountInString(text) > maxChatRunes {
		sv.enqueue(s, map[string]any{"t": "notice", "message": fmt.Sprintf("Messages can be up to %d characters.", maxChatRunes)})
		return
	}
	now := sv.opts.Now()
	if s.acct.AgeBand(now) != BandAdult {
		if reason := checkPersonalInfo(text); reason != "" {
			sv.enqueue(s, map[string]any{"t": "notice", "message": reason})
			return
		}
	}
	if !s.limiter.allow(now) {
		sv.enqueue(s, map[string]any{"t": "notice", "message": "Whoa, slow down a little."})
		return
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
	for _, id := range sv.order {
		r := sv.rooms[id]
		for s := range r.sessions {
			if len(s.path) == 0 {
				continue
			}
			s.walked += dt
			for s.walked >= sv.opts.Step && len(s.path) > 0 {
				s.walked -= sv.opts.Step
				prev := s.pos
				s.pos, s.path = s.path[0], s.path[1:]
				if d := r.def.DoorAt(s.pos); d != nil {
					transfers = append(transfers, transfer{s, d, prev})
					s.path = nil
				}
			}
			if len(s.path) == 0 {
				s.walked = 0
			}
		}
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
		},
		"you":     s.pubID,
		"avatars": avatars,
		"history": r.history,
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
	for radius := 0; radius <= 3; radius++ {
		for dy := -radius; dy <= radius; dy++ {
			for dx := -radius; dx <= radius; dx++ {
				q := world.Point{X: p.X + dx, Y: p.Y + dy}
				if max(abs(dx), abs(dy)) == radius && r.def.Floor(q) && r.def.DoorAt(q) == nil && !taken[q] {
					return q
				}
			}
		}
	}
	return p
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
		ID: s.pubID, Name: sv.displayNameLocked(s), Color: s.color, Pos: s.pos,
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
