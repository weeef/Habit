// Package store keeps what has to survive a restart: accounts, login
// sessions and chat history. Memory is the default; package postgres is the
// real database.
package store

import (
	"context"
	"errors"
	"math"
	"time"
)

var (
	ErrNotFound      = errors.New("store: not found")
	ErrUsernameTaken = errors.New("store: username taken")
	ErrNameTaken     = errors.New("store: name taken")
)

// Account is a username + password account. There is deliberately no email
// or phone field (PLAN.md §5.5).
type Account struct {
	ID           string
	Username     string
	BirthYear    int
	CreatedAt    time.Time
	Salt         []byte
	PasswordHash []byte
	RecoveryHash []byte
	Style        Style
	RandomLook   bool // a fresh random look at each login; Style is kept for switching back
	Welcomes     int  // newcomers this account was first to greet (PLAN.md §4.1)

	// Moderation. A zero time means none; Forever means permanent.
	BannedUntil time.Time
	BanReason   string
	MutedUntil  time.Time
}

// Forever is the "until" of a permanent ban or mute.
var Forever = time.Date(9999, 1, 1, 0, 0, 0, 0, time.UTC)

// Banned and Muted report whether a ban or mute is in force at now.
func (a *Account) Banned(now time.Time) bool { return now.Before(a.BannedUntil) }
func (a *Account) Muted(now time.Time) bool  { return now.Before(a.MutedUntil) }

// ModAction is one entry in the moderation log: who did what to whom.
type ModAction struct {
	At       time.Time
	AdminID  string
	Action   string // ban, unban, mute, unmute, kick, clear_chat
	TargetID string // the account acted on, if any
	Room     string // the room, for clear_chat ("" = every room)
	Detail   string // e.g. the reason or how long
}

// Block is a community anyone can start: its own room to walk into and its
// own board (PLAN.md §2.6). Slug is its short name, as in h/gaming.
type Block struct {
	Slug      string
	Name      string
	Bullet    string // an emoji, shown on its line bullet
	Color     string // the bullet's colour, #rrggbb
	About     string
	Template  string // the room layout it was built from
	OwnerID   string // who started it ("" for official blocks)
	Official  bool
	CreatedAt time.Time
	Street    string // the avenue its storefront is on, if it has one
	Lot       int    // which lot on that avenue (1-based); 0 for none
}

// Style is how an account's character looks, chosen at signup and changeable
// later. Each field
// is an index into a palette the client knows.
type Style struct {
	Skin      int `json:"skin"`
	Hair      int `json:"hair"`
	Outfit    int `json:"outfit"`
	Accessory int `json:"accessory"`
}

// Message is one chat line as stored. AccountID records who really sent it so
// moderators can act on an anonymous message; it is never sent to clients.
type Message struct {
	Room      string
	AccountID string
	AvatarID  string
	Name      string
	Text      string
	Newbie    bool
	SentAt    time.Time
}

// Thread is a post on a room's board. Threads are kept forever and listed by
// latest activity (BumpedAt). Name is the author's username if they chose to
// show it, or "" for anonymous; anonymous tags are worked out when shown.
// AccountID is "" once the author's account is gone.
type Thread struct {
	ID        int64
	Room      string
	AccountID string
	Name      string
	Title     string
	Text      string
	CreatedAt time.Time
	BumpedAt  time.Time
	Replies   int
	Ups       int
	Downs     int
}

// ThreadSort is an order for a board.
type ThreadSort string

const (
	SortActive    ThreadSort = "active"    // latest activity: a reply bumps a thread
	SortNew       ThreadSort = "new"       // newest first
	SortHot       ThreadSort = "hot"       // votes and replies, weighted towards recent
	SortTop       ThreadSort = "top"       // most liked (ups minus downs)
	SortDownvoted ThreadSort = "downvoted" // most downvotes
)

// ValidSort reports whether s is a known sort.
func ValidSort(s ThreadSort) bool {
	switch s {
	case SortActive, SortNew, SortHot, SortTop, SortDownvoted:
		return true
	}
	return false
}

// HotScore ranks a thread for SortHot: votes plus replies, divided by its
// age so newer threads rise. Memory and Postgres use the same formula.
func HotScore(t *Thread, now time.Time) float64 {
	hours := max(0, now.Sub(t.CreatedAt).Hours())
	return float64(t.Ups-t.Downs+t.Replies) / math.Pow(hours+2, 1.5)
}

// Reply is an answer in a thread. Name and AccountID work as on Thread.
type Reply struct {
	ID        int64
	ThreadID  int64
	AccountID string
	Name      string
	Text      string
	CreatedAt time.Time
}

// Notification tells AccountID that someone replied in a thread they started
// or replied in. Room, Title and Text (the reply) are filled in when read.
type Notification struct {
	ID        int64
	AccountID string
	ThreadID  int64
	ReplyID   int64
	Room      string
	Title     string
	Text      string
	CreatedAt time.Time
	Read      bool
}

// Store is implemented by Memory and postgres.Store. Usernames are unique
// ignoring case. Sessions are looked up by a hash of the token, so the raw
// token is never stored.
type Store interface {
	CreateAccount(ctx context.Context, a *Account) error
	AccountByUsername(ctx context.Context, username string) (*Account, error)

	CreateSession(ctx context.Context, tokenHash []byte, accountID string, at time.Time) error
	AccountBySession(ctx context.Context, tokenHash []byte) (*Account, error)
	// AddWelcome counts one more newcomer greeted by the account and returns
	// the new total, or ErrNotFound.
	AddWelcome(ctx context.Context, accountID string) (int, error)
	// SetLook saves an account's character look and whether it gets a
	// random one at each login instead. It returns ErrNotFound for an
	// unknown account.
	SetLook(ctx context.Context, accountID string, style Style, random bool) error
	// SetBan bans an account until a time (zero lifts it), or ErrNotFound.
	SetBan(ctx context.Context, accountID string, until time.Time, reason string) error
	// SetMute mutes an account everywhere until a time (zero lifts it), or
	// returns ErrNotFound.
	SetMute(ctx context.Context, accountID string, until time.Time) error
	// LogModAction adds an entry to the moderation log.
	LogModAction(ctx context.Context, a ModAction) error
	// DeleteMessages permanently deletes a room's saved chat ("" = every
	// room) and returns how many messages went.
	DeleteMessages(ctx context.Context, room string) (int64, error)

	// CreateBlock saves a new block, or returns ErrNameTaken if the slug is
	// in use (ignoring case).
	CreateBlock(ctx context.Context, b *Block) error
	// BlockBySlug returns a block, or ErrNotFound.
	BlockBySlug(ctx context.Context, slug string) (*Block, error)
	// SearchBlocks lists blocks whose slug or name contains query (ignoring
	// case; empty matches all): official ones first, then by name.
	SearchBlocks(ctx context.Context, query string, limit int) ([]Block, error)
	// BlocksStartedSince counts the blocks an account started after since.
	BlocksStartedSince(ctx context.Context, ownerID string, since time.Time) (int, error)
	// BlocksByOwner lists an account's blocks, by name.
	BlocksByOwner(ctx context.Context, ownerID string) ([]Block, error)
	// BlocksWithLots lists every block that has a storefront.
	BlocksWithLots(ctx context.Context) ([]Block, error)
	// SetBlockLot moves a block's storefront to a lot ("", 0 to close it).
	// It returns ErrNameTaken if another block has that lot, or ErrNotFound.
	SetBlockLot(ctx context.Context, slug, street string, lot int) error

	// Counters are small named numbers that outlive a restart, like the
	// day's wishes in the fountain. A counter that was never set is 0.
	Counter(ctx context.Context, key string) (int64, error)
	AddCounter(ctx context.Context, key string, delta int64) (int64, error)
	SetCounter(ctx context.Context, key string, value int64) error

	AddMessage(ctx context.Context, m Message) error
	// RecentMessages returns up to limit of a room's latest messages, oldest first.
	RecentMessages(ctx context.Context, room string, limit int) ([]Message, error)

	// CreateThread saves t and sets its ID, BumpedAt and Replies.
	CreateThread(ctx context.Context, t *Thread) error
	// Threads lists a room's board in the given order (ties: newest first).
	Threads(ctx context.Context, room string, sort ThreadSort, now time.Time, limit, offset int) ([]Thread, error)
	// Vote sets an account's vote on a thread: 1 up, -1 down, 0 to take it
	// back. It returns the thread's new totals, or ErrNotFound.
	Vote(ctx context.Context, threadID int64, accountID string, value int) (ups, downs int, err error)
	// VotesBy returns the account's votes on the given threads (missing = 0).
	VotesBy(ctx context.Context, accountID string, threadIDs []int64) (map[int64]int, error)
	Thread(ctx context.Context, id int64) (*Thread, error)
	// DeleteThread removes a thread with its replies, votes and reply
	// notifications, or returns ErrNotFound.
	DeleteThread(ctx context.Context, id int64) error
	// Replies returns a thread's replies, oldest first.
	Replies(ctx context.Context, threadID int64) ([]Reply, error)
	// AddReply saves r, sets its ID, bumps the thread, and notifies everyone
	// else who started or replied in the thread. It returns the new
	// notifications, or ErrNotFound if the thread doesn't exist.
	AddReply(ctx context.Context, r *Reply) ([]Notification, error)

	// Notifications returns an account's latest notifications, newest first.
	Notifications(ctx context.Context, accountID string, limit int) ([]Notification, error)
	MarkNotificationsRead(ctx context.Context, accountID string) error

	// LoadOrStoreSecret returns the server's saved secret, saving fresh first
	// if there isn't one yet. It keeps anonymous tags stable across restarts.
	LoadOrStoreSecret(ctx context.Context, fresh []byte) ([]byte, error)

	Close()
}
