package server

import (
	"context"
	"errors"
	"testing"
	"time"

	"habit/internal/store"
)

var testNow = time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)

func TestSignupAndLogin(t *testing.T) {
	ctx := context.Background()
	s := NewAccounts(store.NewMemory())
	acct, token, recovery, err := s.Signup(ctx, "pixel_fan", "correct horse battery", 1995, store.Style{}, testNow)
	if err != nil {
		t.Fatal(err)
	}
	if token == "" || len(recovery) != 19 {
		t.Fatalf("want token and XXXX-XXXX-XXXX-XXXX recovery code, got %q %q", token, recovery)
	}
	if got, err := s.ByToken(ctx, token); err != nil || got == nil || got.ID != acct.ID {
		t.Fatalf("token should resume the account, got %+v, %v", got, err)
	}
	if got, err := s.ByToken(ctx, "not a token"); got != nil || err != nil {
		t.Fatalf("unknown token should resume nothing, got %+v, %v", got, err)
	}
	if _, _, err := s.Login(ctx, "PIXEL_FAN", "correct horse battery", testNow); err != nil {
		t.Fatalf("login should be case-insensitive on username: %v", err)
	}
	if _, _, err := s.Login(ctx, "pixel_fan", "wrong password!!", testNow); !errors.Is(err, ErrBadLogin) {
		t.Fatalf("want ErrBadLogin, got %v", err)
	}
	if _, _, err := s.Login(ctx, "nobody", "whatever12345", testNow); !errors.Is(err, ErrBadLogin) {
		t.Fatalf("want ErrBadLogin for unknown user, got %v", err)
	}
}

func TestSignupValidation(t *testing.T) {
	ctx := context.Background()
	s := NewAccounts(store.NewMemory())
	cases := []struct {
		name, user, pass string
		year             int
		want             error
	}{
		{"short password", "okname", "short", 1990, ErrPasswordShort},
		{"bad username", "no spaces", "longenough123", 1990, ErrUsernameInvalid},
		{"reserved name", "Anonymous", "longenough123", 1990, ErrUsernameInvalid},
		{"under 13", "kiddo", "longenough123", 2014, ErrTooYoung},
		{"maybe 12", "kiddo", "longenough123", 2013, ErrTooYoung},
		{"future year", "kiddo", "longenough123", 2030, ErrBirthYear},
	}
	for _, c := range cases {
		if _, _, _, err := s.Signup(ctx, c.user, c.pass, c.year, store.Style{}, testNow); !errors.Is(err, c.want) {
			t.Errorf("%s: want %v, got %v", c.name, c.want, err)
		}
	}

	if _, _, _, err := s.Signup(ctx, "taken", "longenough123", 1990, store.Style{}, testNow); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := s.Signup(ctx, "TAKEN", "longenough123", 1990, store.Style{}, testNow); !errors.Is(err, ErrUsernameTaken) {
		t.Fatalf("want ErrUsernameTaken, got %v", err)
	}
}

func TestAgeBand(t *testing.T) {
	cases := map[int]string{
		2012: BandTeen13to15, // 13 or 14
		2010: BandTeen13to15, // 15 or 16: unsure, so the younger band
		2009: BandTeen16to17,
		2008: BandTeen16to17, // 17 or 18
		2007: BandAdult,
	}
	for year, want := range cases {
		if got := ageBand(year, testNow); got != want {
			t.Errorf("born %d: want %s, got %s", year, want, got)
		}
	}
}

func TestPersonalInfoFilter(t *testing.T) {
	blocked := []string{
		"call me 555-123-4567",
		"my email is kid@example.com",
		"i live at 42 Maple Street",
		"add me on snap",
		"snap: coolkid99",
		"my school is Lincoln",
	}
	for _, msg := range blocked {
		if checkPersonalInfo(msg) == "" {
			t.Errorf("should block %q", msg)
		}
	}
	allowed := []string{"hi everyone!", "what's up", "I scored 100 points", "anyone play snake?"}
	for _, msg := range allowed {
		if r := checkPersonalInfo(msg); r != "" {
			t.Errorf("should allow %q, got %q", msg, r)
		}
	}
}

func TestAnonTag(t *testing.T) {
	secret := []byte("s")
	a := anonTag(secret, "u1", "gaming", testNow)
	if a != anonTag(secret, "u1", "gaming", testNow.Add(time.Hour)) {
		t.Error("tag should be stable within a room for the day")
	}
	if a == anonTag(secret, "u1", "gym", testNow) {
		t.Error("tag should change across rooms")
	}
	if a == anonTag(secret, "u1", "gaming", testNow.Add(24*time.Hour)) {
		t.Error("tag should change across days")
	}
}

func TestBucket(t *testing.T) {
	b := newBucket(chatBurst, chatPerSec)
	for i := 0; i < chatBurst; i++ {
		if !b.allow(testNow) {
			t.Fatalf("burst message %d should pass", i)
		}
	}
	if b.allow(testNow) {
		t.Fatal("should be rate limited after the burst")
	}
	if !b.allow(testNow.Add(time.Second)) {
		t.Fatal("should refill after a second")
	}
}
