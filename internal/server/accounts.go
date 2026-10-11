package server

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base32"
	"encoding/hex"
	"errors"
	"regexp"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/argon2"
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
)

var usernameRe = regexp.MustCompile(`^[A-Za-z0-9_]{3,20}$`)

// Names that would let someone pass as staff or as an anonymous ID.
var reservedNames = map[string]bool{
	"anonymous": true, "visitor": true, "admin": true, "mod": true,
	"moderator": true, "staff": true, "habit": true, "system": true,
}

// Account is a username + password account. There is deliberately no email
// or phone field (PLAN.md §5.5).
type Account struct {
	ID           string
	Username     string
	BirthYear    int
	CreatedAt    time.Time
	salt         []byte
	hash         []byte
	recoveryHash []byte
}

// AgeBand returns the account's age band as of now.
func (a *Account) AgeBand(now time.Time) string { return ageBand(a.BirthYear, now) }

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

// Accounts is an in-memory account store. The prototype keeps nothing on
// disk; a restart forgets everyone.
type Accounts struct {
	mu     sync.Mutex
	byName map[string]*Account
	tokens map[string]*Account
}

func NewAccounts() *Accounts {
	return &Accounts{byName: map[string]*Account{}, tokens: map[string]*Account{}}
}

// Signup creates an account and returns it with a session token and a
// one-time recovery code (the only way back in without email).
func (s *Accounts) Signup(username, password string, birthYear int, now time.Time) (acct *Account, token, recovery string, err error) {
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
	acct = &Account{
		ID:           hex.EncodeToString(randomBytes(8)),
		Username:     username,
		BirthYear:    birthYear,
		CreatedAt:    now,
		salt:         salt,
		hash:         hashSecret(password, salt),
		recoveryHash: hashSecret(recovery, salt),
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	key := strings.ToLower(username)
	if _, taken := s.byName[key]; taken {
		return nil, "", "", ErrUsernameTaken
	}
	s.byName[key] = acct
	token = s.newTokenLocked(acct)
	return acct, token, recovery, nil
}

// Login checks a username and password and returns a new session token.
func (s *Accounts) Login(username, password string) (*Account, string, error) {
	s.mu.Lock()
	acct := s.byName[strings.ToLower(username)]
	s.mu.Unlock()
	if acct == nil {
		// Burn the same time as a real check so timing doesn't reveal
		// which usernames exist.
		hashSecret(password, make([]byte, 16))
		return nil, "", ErrBadLogin
	}
	if subtle.ConstantTimeCompare(hashSecret(password, acct.salt), acct.hash) != 1 {
		return nil, "", ErrBadLogin
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return acct, s.newTokenLocked(acct), nil
}

// ByToken resumes a session from a token, or returns nil.
func (s *Accounts) ByToken(token string) *Account {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.tokens[token]
}

func (s *Accounts) newTokenLocked(acct *Account) string {
	t := hex.EncodeToString(randomBytes(24))
	s.tokens[t] = acct
	return t
}

func hashSecret(secret string, salt []byte) []byte {
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
