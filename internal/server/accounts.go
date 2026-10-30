package server

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base32"
	"encoding/hex"
	"errors"
	"log"
	"regexp"
	"strings"
	"time"

	"golang.org/x/crypto/argon2"

	"habit/internal/store"
)

// Age bands from PLAN.md §6.1.
const (
	BandTeen13to15 = "teen_13_15"
	BandTeen16to17 = "teen_16_17"
	BandAdult      = "adult"
)

const minPasswordLen = 10

var (
	ErrUsernameTaken   = errors.New("that username is taken")
	ErrUsernameInvalid = errors.New("usernames are 3–20 letters, numbers or underscores")
	ErrPasswordShort   = errors.New("passwords need at least 10 characters")
	ErrTooYoung        = errors.New("sorry, you need to be 13 or older to chat on Habit")
	ErrBirthYear       = errors.New("please enter a real birth year")
	ErrBadLogin        = errors.New("wrong username or password")
	ErrUnavailable     = errors.New("something went wrong on our side, please try again")
	ErrStyle           = errors.New("please pick your character's look again")
)

var usernameRe = regexp.MustCompile(`^[A-Za-z0-9_]{3,20}$`)

// Names that would let someone pass as staff or as an anonymous ID.
var reservedNames = map[string]bool{
	"anonymous": true, "visitor": true, "admin": true, "mod": true,
	"moderator": true, "staff": true, "habit": true, "system": true,
}

func ageBand(birthYear int, now time.Time) string {
	// Birth year only, so the age may be a year high; round towards the
	// younger band to keep protections on when unsure.
	age := now.Year() - birthYear - 1
	switch {
	case age < 16:
		return BandTeen13to15
	case age < 18:
		return BandTeen16to17
	default:
		return BandAdult
	}
}

// Accounts handles signup, login and sessions on top of a store.
type Accounts struct {
	store store.Store
}

func NewAccounts(st store.Store) *Accounts {
	return &Accounts{store: st}
}

// Signup creates an account with the character the user designed and
// returns it with a session token and a one-time recovery code (the only way
// back in without email).
func (s *Accounts) Signup(ctx context.Context, username, password string, birthYear int, style store.Style, now time.Time) (acct *store.Account, token, recovery string, err error) {
	if !validStyle(style) {
		return nil, "", "", ErrStyle
	}
	if !usernameRe.MatchString(username) || reservedNames[strings.ToLower(username)] {
		return nil, "", "", ErrUsernameInvalid
	}
	if len([]rune(password)) < minPasswordLen {
		return nil, "", "", ErrPasswordShort
	}
	if birthYear < now.Year()-120 || birthYear > now.Year() {
		return nil, "", "", ErrBirthYear
	}
	// Someone born 13 years ago this calendar year may still be 12; only let
	// in birth years that guarantee 13+.
	if now.Year()-birthYear-1 < 13 {
		return nil, "", "", ErrTooYoung
	}

	salt := randomBytes(16)
	recovery = recoveryCode()
	acct = &store.Account{
		ID:           hex.EncodeToString(randomBytes(8)),
		Username:     username,
		BirthYear:    birthYear,
		CreatedAt:    now,
		Salt:         salt,
		PasswordHash: hashSecret(password, salt),
		RecoveryHash: hashSecret(recovery, salt),
		Style:        style,
	}
	if err := s.store.CreateAccount(ctx, acct); err != nil {
		if errors.Is(err, store.ErrUsernameTaken) {
			return nil, "", "", ErrUsernameTaken
		}
		return nil, "", "", unavailable("create account", err)
	}
	token, err = s.newSession(ctx, acct, now)
	if err != nil {
		return nil, "", "", err
	}
	return acct, token, recovery, nil
}

// Login checks a username and password and returns a new session token.
func (s *Accounts) Login(ctx context.Context, username, password string, now time.Time) (*store.Account, string, error) {
	acct, err := s.store.AccountByUsername(ctx, username)
	if errors.Is(err, store.ErrNotFound) {
		// Burn the same time as a real check so timing doesn't reveal
		// which usernames exist.
		hashSecret(password, make([]byte, 16))
		return nil, "", ErrBadLogin
	}
	if err != nil {
		return nil, "", unavailable("login", err)
	}
	if subtle.ConstantTimeCompare(hashSecret(password, acct.Salt), acct.PasswordHash) != 1 {
		return nil, "", ErrBadLogin
	}
	token, err := s.newSession(ctx, acct, now)
	if err != nil {
		return nil, "", err
	}
	return acct, token, nil
}

// ByToken resumes a session from a token. It returns nil, nil for an unknown
// token.
func (s *Accounts) ByToken(ctx context.Context, token string) (*store.Account, error) {
	acct, err := s.store.AccountBySession(ctx, hashToken(token))
	if errors.Is(err, store.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, unavailable("resume session", err)
	}
	return acct, nil
}

func (s *Accounts) newSession(ctx context.Context, acct *store.Account, now time.Time) (string, error) {
	t := hex.EncodeToString(randomBytes(24))
	if err := s.store.CreateSession(ctx, hashToken(t), acct.ID, now); err != nil {
		return "", unavailable("create session", err)
	}
	return t, nil
}

// unavailable logs a storage failure and returns the error users see.
func unavailable(op string, err error) error {
	log.Printf("%s: %v", op, err)
	return ErrUnavailable
}

// hashToken is how session tokens are stored, so a leaked database doesn't
// hand out working logins. Tokens are long and random, so no salt is needed.
func hashToken(token string) []byte {
	h := sha256.Sum256([]byte(token))
	return h[:]
}

// hashSlots caps how many password hashes run at once. Each one uses 64 MB
// on purpose, so without a cap a burst of logins can run the server out of
// memory; extra attempts wait their turn instead.
var hashSlots = make(chan struct{}, 2)

func hashSecret(secret string, salt []byte) []byte {
	hashSlots <- struct{}{}
	defer func() { <-hashSlots }()
	return argon2.IDKey([]byte(secret), salt, 1, 64*1024, 4, 32)
}

func recoveryCode() string {
	s := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomBytes(10))
	return s[0:4] + "-" + s[4:8] + "-" + s[8:12] + "-" + s[12:16]
}

func randomBytes(n int) []byte {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return b
}
