package store

import (
	"cmp"
	"context"
	"slices"
	"strings"
	"sync"
	"time"
)

// memoryMessageCap bounds each room's chat history; only the latest few are
// ever read. Board posts are kept forever.
const memoryMessageCap = 1000

// Memory keeps everything in process memory: a restart forgets it all. It is
// used when no database is configured, and in tests.
type Memory struct {
	mu       sync.Mutex
	byName   map[string]*Account // lower-case username → account
	byID     map[string]*Account
	sessions map[string]string // token hash → account ID
	messages map[string][]Message
	secret   []byte

	threads  map[int64]*Thread
	replies  map[int64][]Reply
	notes    []Notification // oldest first
	lastID   int64          // shared by threads, replies and notifications
	votes    map[voteKey]int
	counters map[string]int64
	blocks   map[string]*Block // lower-case slug → block
}

type voteKey struct {
	thread  int64
	account string
}

func NewMemory() *Memory {
	return &Memory{
		byName:   map[string]*Account{},
		byID:     map[string]*Account{},
		sessions: map[string]string{},
		messages: map[string][]Message{},
		threads:  map[int64]*Thread{},
		replies:  map[int64][]Reply{},
		votes:    map[voteKey]int{},
	}
}

func (m *Memory) CreateAccount(_ context.Context, a *Account) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := strings.ToLower(a.Username)
	if _, taken := m.byName[key]; taken {
		return ErrUsernameTaken
	}
	c := *a
	m.byName[key] = &c
	m.byID[c.ID] = &c
	return nil
}

func (m *Memory) AccountByUsername(_ context.Context, username string) (*Account, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a := m.byName[strings.ToLower(username)]
	if a == nil {
		return nil, ErrNotFound
	}
	c := *a
	return &c, nil
}

func (m *Memory) AddWelcome(_ context.Context, accountID string) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a := m.byID[accountID]
	if a == nil {
		return 0, ErrNotFound
	}
	a.Welcomes++
	return a.Welcomes, nil
}

func (m *Memory) SetLook(_ context.Context, accountID string, style Style, random bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	a := m.byID[accountID]
	if a == nil {
		return ErrNotFound
	}
	a.Style, a.RandomLook = style, random
	return nil
}

func (m *Memory) CreateBlock(_ context.Context, b *Block) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := strings.ToLower(b.Slug)
	if _, taken := m.blocks[key]; taken {
		return ErrNameTaken
	}
	if m.blocks == nil {
		m.blocks = map[string]*Block{}
	}
	c := *b
	m.blocks[key] = &c
	return nil
}

func (m *Memory) BlockBySlug(_ context.Context, slug string) (*Block, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	b := m.blocks[strings.ToLower(slug)]
	if b == nil {
		return nil, ErrNotFound
	}
	c := *b
	return &c, nil
}

func (m *Memory) SearchBlocks(_ context.Context, query string, limit int) ([]Block, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	q := strings.ToLower(query)
	var out []Block
	for _, b := range m.blocks {
		if strings.Contains(strings.ToLower(b.Slug), q) || strings.Contains(strings.ToLower(b.Name), q) {
			out = append(out, *b)
		}
	}
	slices.SortFunc(out, func(a, b Block) int {
		if a.Official != b.Official {
			if a.Official {
				return -1
			}
			return 1
		}
		return cmp.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})
	return out[:min(len(out), limit)], nil
}

func (m *Memory) BlocksByOwner(_ context.Context, ownerID string) ([]Block, error) {
	return m.blocksWhere(func(b *Block) bool { return b.OwnerID == ownerID }), nil
}

func (m *Memory) BlocksWithLots(context.Context) ([]Block, error) {
	return m.blocksWhere(func(b *Block) bool { return b.Lot != 0 }), nil
}

func (m *Memory) blocksWhere(keep func(*Block) bool) []Block {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []Block
	for _, b := range m.blocks {
		if keep(b) {
			out = append(out, *b)
		}
	}
	slices.SortFunc(out, func(a, b Block) int { return cmp.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)) })
	return out
}

func (m *Memory) SetBlockLot(_ context.Context, slug, street string, lot int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	b := m.blocks[strings.ToLower(slug)]
	if b == nil {
		return ErrNotFound
	}
	if lot != 0 {
		for _, o := range m.blocks {
			if o != b && o.Street == street && o.Lot == lot {
				return ErrNameTaken
			}
		}
	}
	b.Street, b.Lot = street, lot
	return nil
}

func (m *Memory) BlocksStartedSince(_ context.Context, ownerID string, since time.Time) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for _, b := range m.blocks {
		if b.OwnerID == ownerID && b.CreatedAt.After(since) {
			n++
		}
	}
	return n, nil
}

func (m *Memory) Counter(_ context.Context, key string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.counters[key], nil
}

func (m *Memory) AddCounter(_ context.Context, key string, delta int64) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.counters == nil {
		m.counters = map[string]int64{}
	}
	m.counters[key] += delta
	return m.counters[key], nil
}

func (m *Memory) SetCounter(_ context.Context, key string, value int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.counters == nil {
		m.counters = map[string]int64{}
	}
	m.counters[key] = value
	return nil
}

func (m *Memory) CreateSession(_ context.Context, tokenHash []byte, accountID string, _ time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.byID[accountID] == nil {
		return ErrNotFound
	}
	m.sessions[string(tokenHash)] = accountID
	return nil
}

func (m *Memory) AccountBySession(_ context.Context, tokenHash []byte) (*Account, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a := m.byID[m.sessions[string(tokenHash)]]
	if a == nil {
		return nil, ErrNotFound
	}
	c := *a
	return &c, nil
}

func (m *Memory) AddMessage(_ context.Context, msg Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	h := append(m.messages[msg.Room], msg)
	if len(h) > memoryMessageCap {
		h = h[len(h)-memoryMessageCap:]
	}
	m.messages[msg.Room] = h
	return nil
}

func (m *Memory) RecentMessages(_ context.Context, room string, limit int) ([]Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	h := m.messages[room]
	h = h[max(0, len(h)-limit):]
	return append([]Message(nil), h...), nil
}

func (m *Memory) CreateThread(_ context.Context, t *Thread) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastID++
	t.ID, t.BumpedAt, t.Replies, t.Ups, t.Downs = m.lastID, t.CreatedAt, 0, 0, 0
	c := *t
	m.threads[c.ID] = &c
	return nil
}

func (m *Memory) Threads(_ context.Context, room string, sort ThreadSort, now time.Time, limit, offset int) ([]Thread, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []Thread
	for _, t := range m.threads {
		if t.Room == room {
			out = append(out, *t)
		}
	}
	slices.SortFunc(out, func(a, b Thread) int {
		var c int
		switch sort {
		case SortNew:
		case SortHot:
			c = cmp.Compare(HotScore(&b, now), HotScore(&a, now))
		case SortTop:
			c = cmp.Compare(b.Ups-b.Downs, a.Ups-a.Downs)
		case SortDownvoted:
			c = cmp.Compare(b.Downs, a.Downs)
		default:
			c = b.BumpedAt.Compare(a.BumpedAt)
		}
		if c != 0 {
			return c
		}
		return cmp.Compare(b.ID, a.ID)
	})
	out = out[min(offset, len(out)):]
	return out[:min(limit, len(out))], nil
}

func (m *Memory) Vote(_ context.Context, threadID int64, accountID string, value int) (int, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t := m.threads[threadID]
	if t == nil {
		return 0, 0, ErrNotFound
	}
	k := voteKey{threadID, accountID}
	switch m.votes[k] {
	case 1:
		t.Ups--
	case -1:
		t.Downs--
	}
	switch value {
	case 1:
		t.Ups++
	case -1:
		t.Downs++
	}
	if value == 0 {
		delete(m.votes, k)
	} else {
		m.votes[k] = value
	}
	return t.Ups, t.Downs, nil
}

func (m *Memory) VotesBy(_ context.Context, accountID string, threadIDs []int64) (map[int64]int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := map[int64]int{}
	for _, id := range threadIDs {
		if v := m.votes[voteKey{id, accountID}]; v != 0 {
			out[id] = v
		}
	}
	return out, nil
}

func (m *Memory) Thread(_ context.Context, id int64) (*Thread, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t := m.threads[id]
	if t == nil {
		return nil, ErrNotFound
	}
	c := *t
	return &c, nil
}

func (m *Memory) Replies(_ context.Context, threadID int64) ([]Reply, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]Reply(nil), m.replies[threadID]...), nil
}

func (m *Memory) AddReply(_ context.Context, r *Reply) ([]Notification, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t := m.threads[r.ThreadID]
	if t == nil {
		return nil, ErrNotFound
	}
	// Everyone who has posted in the thread so far, apart from the replier.
	var notify []string
	for _, id := range append([]string{t.AccountID}, accountIDs(m.replies[t.ID])...) {
		if id != "" && id != r.AccountID && !slices.Contains(notify, id) {
			notify = append(notify, id)
		}
	}

	m.lastID++
	r.ID = m.lastID
	m.replies[t.ID] = append(m.replies[t.ID], *r)
	t.BumpedAt = r.CreatedAt
	t.Replies++

	var out []Notification
	for _, id := range notify {
		m.lastID++
		n := Notification{
			ID: m.lastID, AccountID: id, ThreadID: t.ID, ReplyID: r.ID,
			Room: t.Room, Title: t.Title, Text: r.Text, CreatedAt: r.CreatedAt,
		}
		m.notes = append(m.notes, n)
		out = append(out, n)
	}
	return out, nil
}

func accountIDs(rs []Reply) []string {
	ids := make([]string, len(rs))
	for i, r := range rs {
		ids[i] = r.AccountID
	}
	return ids
}

func (m *Memory) Notifications(_ context.Context, accountID string, limit int) ([]Notification, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []Notification
	for i := len(m.notes) - 1; i >= 0 && len(out) < limit; i-- {
		if m.notes[i].AccountID == accountID {
			out = append(out, m.notes[i])
		}
	}
	return out, nil
}

func (m *Memory) MarkNotificationsRead(_ context.Context, accountID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.notes {
		if m.notes[i].AccountID == accountID {
			m.notes[i].Read = true
		}
	}
	return nil
}

func (m *Memory) LoadOrStoreSecret(_ context.Context, fresh []byte) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.secret == nil {
		m.secret = append([]byte(nil), fresh...)
	}
	return m.secret, nil
}

func (m *Memory) Close() {}
