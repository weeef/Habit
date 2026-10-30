// Package postgres is the Postgres-backed store.Store.
package postgres

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"habit/internal/store"
)

//go:embed schema.sql
var schema string

const uniqueViolation = "23505"

type Store struct {
	pool *pgxpool.Pool
}

var _ store.Store = (*Store)(nil)

// Open connects to url (e.g. postgres://user:pass@localhost:5432/habit) and
// creates any missing tables.
func Open(ctx context.Context, url string) (*Store, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("postgres: %w", err)
	}
	if _, err := pool.Exec(ctx, schema); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres: applying schema: %w", err)
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Close() { s.pool.Close() }

const accountCols = `a.id, a.username, a.birth_year, a.created_at, a.salt, a.password_hash, a.recovery_hash,
	a.style_skin, a.style_hair, a.style_outfit, a.style_accessory, a.random_look, a.welcomes,
	a.banned_until, a.ban_reason, a.muted_until`

func scanAccount(row pgx.Row) (*store.Account, error) {
	var a store.Account
	var banned, muted *time.Time
	err := row.Scan(&a.ID, &a.Username, &a.BirthYear, &a.CreatedAt, &a.Salt, &a.PasswordHash, &a.RecoveryHash,
		&a.Style.Skin, &a.Style.Hair, &a.Style.Outfit, &a.Style.Accessory, &a.RandomLook, &a.Welcomes,
		&banned, &a.BanReason, &muted)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, store.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if banned != nil {
		a.BannedUntil = *banned
	}
	if muted != nil {
		a.MutedUntil = *muted
	}
	return &a, nil
}

func (s *Store) CreateAccount(ctx context.Context, a *store.Account) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO accounts (id, username, birth_year, created_at, salt, password_hash, recovery_hash,
		                       style_skin, style_hair, style_outfit, style_accessory)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		a.ID, a.Username, a.BirthYear, a.CreatedAt, a.Salt, a.PasswordHash, a.RecoveryHash,
		a.Style.Skin, a.Style.Hair, a.Style.Outfit, a.Style.Accessory)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
		return store.ErrUsernameTaken
	}
	return err
}

func (s *Store) AccountByUsername(ctx context.Context, username string) (*store.Account, error) {
	return scanAccount(s.pool.QueryRow(ctx,
		`SELECT `+accountCols+` FROM accounts a WHERE lower(a.username) = lower($1)`, username))
}

func (s *Store) AddWelcome(ctx context.Context, accountID string) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx,
		`UPDATE accounts SET welcomes = welcomes + 1 WHERE id = $1 RETURNING welcomes`, accountID).Scan(&n)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, store.ErrNotFound
	}
	return n, err
}

// nullTime stores a zero time as NULL.
func nullTime(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

func (s *Store) SetBan(ctx context.Context, accountID string, until time.Time, reason string) error {
	tag, err := s.pool.Exec(ctx, `UPDATE accounts SET banned_until = $2, ban_reason = $3 WHERE id = $1`,
		accountID, nullTime(until), reason)
	if err == nil && tag.RowsAffected() == 0 {
		return store.ErrNotFound
	}
	return err
}

func (s *Store) SetMute(ctx context.Context, accountID string, until time.Time) error {
	tag, err := s.pool.Exec(ctx, `UPDATE accounts SET muted_until = $2 WHERE id = $1`, accountID, nullTime(until))
	if err == nil && tag.RowsAffected() == 0 {
		return store.ErrNotFound
	}
	return err
}

func (s *Store) LogModAction(ctx context.Context, a store.ModAction) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO mod_actions (at, admin_id, action, target_id, room, detail) VALUES ($1, $2, $3, $4, $5, $6)`,
		a.At, a.AdminID, a.Action, a.TargetID, a.Room, a.Detail)
	return err
}

func (s *Store) DeleteMessages(ctx context.Context, room string) (int64, error) {
	tag, err := s.pool.Exec(ctx, `DELETE FROM messages WHERE $1 = '' OR room = $1`, room)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func (s *Store) DeleteThread(ctx context.Context, id int64) error {
	// Replies, votes and notifications go with it (ON DELETE CASCADE).
	tag, err := s.pool.Exec(ctx, `DELETE FROM threads WHERE id = $1`, id)
	if err == nil && tag.RowsAffected() == 0 {
		return store.ErrNotFound
	}
	return err
}

func (s *Store) SetLook(ctx context.Context, accountID string, style store.Style, random bool) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE accounts SET style_skin = $2, style_hair = $3, style_outfit = $4, style_accessory = $5, random_look = $6
		 WHERE id = $1`,
		accountID, style.Skin, style.Hair, style.Outfit, style.Accessory, random)
	if err == nil && tag.RowsAffected() == 0 {
		return store.ErrNotFound
	}
	return err
}

const blockCols = `slug, name, bullet, color, about, template, coalesce(owner_id, ''), official, created_at, street, lot`

func scanBlock(row pgx.Row) (*store.Block, error) {
	var b store.Block
	err := row.Scan(&b.Slug, &b.Name, &b.Bullet, &b.Color, &b.About, &b.Template, &b.OwnerID, &b.Official, &b.CreatedAt, &b.Street, &b.Lot)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, store.ErrNotFound
	}
	return &b, err
}

func (s *Store) CreateBlock(ctx context.Context, b *store.Block) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO blocks (slug, name, bullet, color, about, template, owner_id, official, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, ''), $8, $9)`,
		b.Slug, b.Name, b.Bullet, b.Color, b.About, b.Template, b.OwnerID, b.Official, b.CreatedAt)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return store.ErrNameTaken
	}
	return err
}

func (s *Store) BlockBySlug(ctx context.Context, slug string) (*store.Block, error) {
	return scanBlock(s.pool.QueryRow(ctx, `SELECT `+blockCols+` FROM blocks WHERE lower(slug) = lower($1)`, slug))
}

func (s *Store) SearchBlocks(ctx context.Context, query string, limit int) ([]store.Block, error) {
	return s.queryBlocks(ctx, `SELECT `+blockCols+` FROM blocks
		 WHERE strpos(lower(slug), lower($1)) > 0 OR strpos(lower(name), lower($1)) > 0
		 ORDER BY official DESC, lower(name) LIMIT $2`, query, limit)
}

func (s *Store) BlocksByOwner(ctx context.Context, ownerID string) ([]store.Block, error) {
	return s.queryBlocks(ctx, `SELECT `+blockCols+` FROM blocks WHERE owner_id = $1 ORDER BY lower(name)`, ownerID)
}

func (s *Store) BlocksWithLots(ctx context.Context) ([]store.Block, error) {
	return s.queryBlocks(ctx, `SELECT `+blockCols+` FROM blocks WHERE lot <> 0 ORDER BY lower(name)`)
}

func (s *Store) SetBlockLot(ctx context.Context, slug, street string, lot int) error {
	tag, err := s.pool.Exec(ctx, `UPDATE blocks SET street = $2, lot = $3 WHERE lower(slug) = lower($1)`, slug, street, lot)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return store.ErrNameTaken
	}
	if err == nil && tag.RowsAffected() == 0 {
		return store.ErrNotFound
	}
	return err
}

func (s *Store) queryBlocks(ctx context.Context, sql string, args ...any) ([]store.Block, error) {
	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []store.Block
	for rows.Next() {
		b, err := scanBlock(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *b)
	}
	return out, rows.Err()
}

func (s *Store) BlocksStartedSince(ctx context.Context, ownerID string, since time.Time) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `SELECT count(*) FROM blocks WHERE owner_id = $1 AND created_at > $2`, ownerID, since).Scan(&n)
	return n, err
}

func (s *Store) Counter(ctx context.Context, key string) (int64, error) {
	var n int64
	err := s.pool.QueryRow(ctx, `SELECT value FROM counters WHERE key = $1`, key).Scan(&n)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	return n, err
}

func (s *Store) AddCounter(ctx context.Context, key string, delta int64) (int64, error) {
	var n int64
	err := s.pool.QueryRow(ctx,
		`INSERT INTO counters (key, value) VALUES ($1, $2)
		 ON CONFLICT (key) DO UPDATE SET value = counters.value + EXCLUDED.value RETURNING value`, key, delta).Scan(&n)
	return n, err
}

func (s *Store) SetCounter(ctx context.Context, key string, value int64) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO counters (key, value) VALUES ($1, $2) ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, key, value)
	return err
}

func (s *Store) CreateSession(ctx context.Context, tokenHash []byte, accountID string, at time.Time) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO sessions (token_hash, account_id, created_at) VALUES ($1, $2, $3)`,
		tokenHash, accountID, at)
	return err
}

func (s *Store) AccountBySession(ctx context.Context, tokenHash []byte) (*store.Account, error) {
	return scanAccount(s.pool.QueryRow(ctx,
		`SELECT `+accountCols+` FROM sessions s JOIN accounts a ON a.id = s.account_id WHERE s.token_hash = $1`, tokenHash))
}

func (s *Store) AddMessage(ctx context.Context, m store.Message) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO messages (room, account_id, avatar_id, name, body, newbie, sent_at)
		 VALUES ($1, NULLIF($2, ''), $3, $4, $5, $6, $7)`,
		m.Room, m.AccountID, m.AvatarID, m.Name, m.Text, m.Newbie, m.SentAt)
	return err
}

func (s *Store) RecentMessages(ctx context.Context, room string, limit int) ([]store.Message, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT room, coalesce(account_id, ''), avatar_id, name, body, newbie, sent_at
		 FROM messages WHERE room = $1 ORDER BY id DESC LIMIT $2`, room, limit)
	if err != nil {
		return nil, err
	}
	out, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (store.Message, error) {
		var m store.Message
		err := row.Scan(&m.Room, &m.AccountID, &m.AvatarID, &m.Name, &m.Text, &m.Newbie, &m.SentAt)
		return m, err
	})
	slices.Reverse(out)
	return out, err
}

const threadCols = `id, room, coalesce(account_id, ''), name, title, body, created_at, bumped_at, reply_count, ups, downs`

func scanThread(row pgx.Row) (store.Thread, error) {
	var t store.Thread
	err := row.Scan(&t.ID, &t.Room, &t.AccountID, &t.Name, &t.Title, &t.Text, &t.CreatedAt, &t.BumpedAt, &t.Replies, &t.Ups, &t.Downs)
	return t, err
}

// sortOrder is each sort's ORDER BY. hot matches store.HotScore; $4 is now.
var sortOrder = map[store.ThreadSort]string{
	store.SortActive:    `bumped_at DESC, id DESC`,
	store.SortNew:       `id DESC`,
	store.SortHot:       `(ups - downs + reply_count) / power(greatest(extract(epoch FROM $4::timestamptz - created_at) / 3600, 0) + 2, 1.5) DESC, id DESC`,
	store.SortTop:       `ups - downs DESC, id DESC`,
	store.SortDownvoted: `downs DESC, id DESC`,
}

func (s *Store) CreateThread(ctx context.Context, t *store.Thread) error {
	t.BumpedAt, t.Replies, t.Ups, t.Downs = t.CreatedAt, 0, 0, 0
	return s.pool.QueryRow(ctx,
		`INSERT INTO threads (room, account_id, name, title, body, created_at, bumped_at)
		 VALUES ($1, NULLIF($2, ''), $3, $4, $5, $6, $6) RETURNING id`,
		t.Room, t.AccountID, t.Name, t.Title, t.Text, t.CreatedAt).Scan(&t.ID)
}

func (s *Store) Threads(ctx context.Context, room string, sort store.ThreadSort, now time.Time, limit, offset int) ([]store.Thread, error) {
	order, ok := sortOrder[sort]
	if !ok {
		order = sortOrder[store.SortActive]
	}
	rows, err := s.pool.Query(ctx,
		`SELECT `+threadCols+` FROM threads WHERE room = $1 AND $4::timestamptz IS NOT NULL
		 ORDER BY `+order+` LIMIT $2 OFFSET $3`, room, limit, offset, now)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (store.Thread, error) { return scanThread(row) })
}

func (s *Store) Vote(ctx context.Context, threadID int64, accountID string, value int) (int, int, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, 0, err
	}
	defer tx.Rollback(ctx)
	// Lock the thread so concurrent votes add up correctly.
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT true FROM threads WHERE id = $1 FOR UPDATE`, threadID).Scan(&exists); errors.Is(err, pgx.ErrNoRows) {
		return 0, 0, store.ErrNotFound
	} else if err != nil {
		return 0, 0, err
	}
	var old int
	err = tx.QueryRow(ctx, `SELECT value FROM thread_votes WHERE thread_id = $1 AND account_id = $2`, threadID, accountID).Scan(&old)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return 0, 0, err
	}
	if value == 0 {
		_, err = tx.Exec(ctx, `DELETE FROM thread_votes WHERE thread_id = $1 AND account_id = $2`, threadID, accountID)
	} else {
		_, err = tx.Exec(ctx,
			`INSERT INTO thread_votes (thread_id, account_id, value) VALUES ($1, $2, $3)
			 ON CONFLICT (thread_id, account_id) DO UPDATE SET value = EXCLUDED.value`, threadID, accountID, value)
	}
	if err != nil {
		return 0, 0, err
	}
	up := func(v int) int {
		if v == 1 {
			return 1
		}
		return 0
	}
	down := func(v int) int {
		if v == -1 {
			return 1
		}
		return 0
	}
	var ups, downs int
	if err := tx.QueryRow(ctx,
		`UPDATE threads SET ups = ups + $2, downs = downs + $3 WHERE id = $1 RETURNING ups, downs`,
		threadID, up(value)-up(old), down(value)-down(old)).Scan(&ups, &downs); err != nil {
		return 0, 0, err
	}
	return ups, downs, tx.Commit(ctx)
}

func (s *Store) VotesBy(ctx context.Context, accountID string, threadIDs []int64) (map[int64]int, error) {
	out := map[int64]int{}
	if len(threadIDs) == 0 {
		return out, nil
	}
	rows, err := s.pool.Query(ctx,
		`SELECT thread_id, value FROM thread_votes WHERE account_id = $1 AND thread_id = ANY($2)`, accountID, threadIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var v int
		if err := rows.Scan(&id, &v); err != nil {
			return nil, err
		}
		out[id] = v
	}
	return out, rows.Err()
}

func (s *Store) Thread(ctx context.Context, id int64) (*store.Thread, error) {
	t, err := scanThread(s.pool.QueryRow(ctx, `SELECT `+threadCols+` FROM threads WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, store.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *Store) Replies(ctx context.Context, threadID int64) ([]store.Reply, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, thread_id, coalesce(account_id, ''), name, body, created_at
		 FROM replies WHERE thread_id = $1 ORDER BY id`, threadID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (store.Reply, error) {
		var r store.Reply
		err := row.Scan(&r.ID, &r.ThreadID, &r.AccountID, &r.Name, &r.Text, &r.CreatedAt)
		return r, err
	})
}

func (s *Store) AddReply(ctx context.Context, r *store.Reply) ([]store.Notification, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// Bumping first locks the thread row, so concurrent replies queue up.
	var room, title string
	err = tx.QueryRow(ctx,
		`UPDATE threads SET bumped_at = $2, reply_count = reply_count + 1 WHERE id = $1 RETURNING room, title`,
		r.ThreadID, r.CreatedAt).Scan(&room, &title)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, store.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := tx.QueryRow(ctx,
		`INSERT INTO replies (thread_id, account_id, name, body, created_at)
		 VALUES ($1, NULLIF($2, ''), $3, $4, $5) RETURNING id`,
		r.ThreadID, r.AccountID, r.Name, r.Text, r.CreatedAt).Scan(&r.ID); err != nil {
		return nil, err
	}

	// Everyone who has posted in the thread, apart from the replier.
	rows, err := tx.Query(ctx,
		`INSERT INTO notifications (account_id, thread_id, reply_id, created_at)
		 SELECT p.account_id, $1, $2, $3 FROM (
		     SELECT account_id FROM threads WHERE id = $1
		     UNION SELECT account_id FROM replies WHERE thread_id = $1
		 ) p
		 WHERE p.account_id IS NOT NULL AND p.account_id <> $4
		 RETURNING id, account_id`,
		r.ThreadID, r.ID, r.CreatedAt, r.AccountID)
	if err != nil {
		return nil, err
	}
	notes, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (store.Notification, error) {
		n := store.Notification{ThreadID: r.ThreadID, ReplyID: r.ID, Room: room, Title: title, Text: r.Text, CreatedAt: r.CreatedAt}
		err := row.Scan(&n.ID, &n.AccountID)
		return n, err
	})
	if err != nil {
		return nil, err
	}
	return notes, tx.Commit(ctx)
}

func (s *Store) Notifications(ctx context.Context, accountID string, limit int) ([]store.Notification, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT n.id, n.account_id, n.thread_id, n.reply_id, t.room, t.title, r.body, n.created_at, n.read
		 FROM notifications n
		 JOIN threads t ON t.id = n.thread_id
		 JOIN replies r ON r.id = n.reply_id
		 WHERE n.account_id = $1 ORDER BY n.id DESC LIMIT $2`, accountID, limit)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (store.Notification, error) {
		var n store.Notification
		err := row.Scan(&n.ID, &n.AccountID, &n.ThreadID, &n.ReplyID, &n.Room, &n.Title, &n.Text, &n.CreatedAt, &n.Read)
		return n, err
	})
}

func (s *Store) MarkNotificationsRead(ctx context.Context, accountID string) error {
	_, err := s.pool.Exec(ctx, `UPDATE notifications SET read = true WHERE account_id = $1 AND NOT read`, accountID)
	return err
}

func (s *Store) LoadOrStoreSecret(ctx context.Context, fresh []byte) ([]byte, error) {
	if _, err := s.pool.Exec(ctx,
		`INSERT INTO settings (key, value) VALUES ('anon_secret', $1) ON CONFLICT (key) DO NOTHING`, fresh); err != nil {
		return nil, err
	}
	var secret []byte
	err := s.pool.QueryRow(ctx, `SELECT value FROM settings WHERE key = 'anon_secret'`).Scan(&secret)
	return secret, err
}
