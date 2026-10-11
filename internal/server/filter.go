package server

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"time"
)

// personalInfoRules is the teen personal-info filter (PLAN.md §6.1). It only
// runs on messages from teen accounts and tells them why a message was held.
var personalInfoRules = []struct {
	re     *regexp.Regexp
	reason string
}{
	{regexp.MustCompile(`\+?\d[\d\s().-]{7,}\d`), "phone numbers"},
	{regexp.MustCompile(`(?i)[a-z0-9._%+-]+@[a-z0-9.-]+\.[a-z]{2,}`), "email addresses"},
	{regexp.MustCompile(`(?i)\b\d+\s+\w+(\s+\w+)?\s+(street|st|avenue|ave|road|rd|lane|ln|drive|dr|court|ct|boulevard|blvd)\b`), "home addresses"},
	{regexp.MustCompile(`(?i)\b(add|dm|message|hmu|find)\s+me\s+(on|at)\b`), "moving the chat to other apps"},
	{regexp.MustCompile(`(?i)\b(snap(chat)?|insta(gram)?|whats\s?app|telegram|kik|discord)\s*[:@]`), "other app handles"},
	{regexp.MustCompile(`(?i)\b(my school is|i go to [\w\s]{1,30}(high|middle|school|academy))\b`), "your school"},
}

// checkPersonalInfo returns a reason if text looks like it shares personal
// details, or "" if it's fine.
func checkPersonalInfo(text string) string {
	for _, r := range personalInfoRules {
		if r.re.MatchString(text) {
			return "For your safety, teen accounts can't share " + r.reason + " on Habit."
		}
	}
	return ""
}

// anonTag is the short public ID shown as "Anonymous #3f9a". It's stable for
// one person in one room for one day, and unlinkable across rooms and days
// for anyone without the server secret (PLAN.md §5.4).
func anonTag(secret []byte, key, roomID string, now time.Time) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(key + "|" + roomID + "|" + now.UTC().Format("2006-01-02")))
	return hex.EncodeToString(mac.Sum(nil))[:4]
}

// postTag is the anonymous tag on a board post. It's stable for one person
// within one thread (so a conversation can be followed) and unlinkable across
// threads.
func postTag(secret []byte, accountID string, threadID int64) string {
	mac := hmac.New(sha256.New, secret)
	fmt.Fprintf(mac, "post|%s|%d", accountID, threadID)
	return hex.EncodeToString(mac.Sum(nil))[:4]
}

// bucket is a token-bucket rate limiter.
type bucket struct {
	burst, perSec float64
	tokens        float64
	last          time.Time
}

const (
	chatBurst  = 5
	chatPerSec = 1.0
)

func newBucket(burst, perSec float64) bucket { return bucket{burst: burst, perSec: perSec} }

func (b *bucket) allow(now time.Time) bool {
	if b.last.IsZero() {
		b.tokens = b.burst
	} else {
		b.tokens = min(b.burst, b.tokens+now.Sub(b.last).Seconds()*b.perSec)
	}
	b.last = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}
