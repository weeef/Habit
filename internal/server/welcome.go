package server

import (
	"context"
	"log"
	"slices"
	"time"

	"habit/internal/world"
)

// Welcomes (PLAN.md §4.1): nobody should walk in and leave without someone
// saying hi. A newcomer lands near whoever's already chatting, a couple of
// settled people nearby get a gentle nudge, and whoever greets them first
// (a chat line or a wave close by) gets the credit.
const (
	welcomeWindow  = 5 * time.Minute // how long a newcomer counts as just arrived
	greeterSettled = time.Minute     // been around this long to be asked to greet
	nudgeEvery     = 3 * time.Minute // at most one nudge per person this often
	greetRange     = 6               // tiles: close enough for a hi to count
	maxGreeters    = 2
	crowdRadius    = 4 // tiles: who counts as standing together
)

// arrivalSpotLocked is where someone connecting lands: beside the biggest
// group standing in r, so they walk straight into a conversation, or the
// room's spawn if nobody's there.
func (sv *Server) arrivalSpotLocked(r *room) world.Point {
	best, bestN := r.def.Spawn, 0
	for o := range r.sessions {
		if len(o.path) > 0 {
			continue
		}
		n := 0
		for q := range r.sessions {
			if q != o && len(q.path) == 0 && chebyshev(o.pos, q.pos) <= crowdRadius {
				n++
			}
		}
		if n+1 > bestN {
			best, bestN = o.pos, n+1
		}
	}
	return best
}

// newcomerLocked marks s as just arrived, if they're new, and asks a couple
// of people nearby to say hi.
func (sv *Server) newcomerLocked(s *session) {
	now := sv.opts.Now()
	s.joinedAt = now
	if !sv.newbieLocked(s) {
		return
	}
	s.welcomeUntil = now.Add(welcomeWindow)
	sv.newcomers++
	sv.nudgeForLocked(s)
}

// nudgeForLocked asks a couple of settled people in s's room to say hi to
// s. It runs when a newcomer arrives, and again if they walk into another
// room before anyone has.
func (sv *Server) nudgeForLocked(s *session) {
	now := sv.opts.Now()
	var picks []*session
	for o := range s.room.sessions {
		if o != s && now.Sub(o.joinedAt) >= greeterSettled && now.Sub(o.lastNudge) >= nudgeEvery {
			picks = append(picks, o)
		}
	}
	// People who can chat first, then whoever's closest.
	slices.SortFunc(picks, func(a, b *session) int {
		if (a.acct != nil) != (b.acct != nil) {
			if a.acct != nil {
				return -1
			}
			return 1
		}
		return chebyshev(a.pos, s.pos) - chebyshev(b.pos, s.pos)
	})
	for _, o := range picks[:min(len(picks), maxGreeters)] {
		o.lastNudge = now
		sv.enqueue(o, map[string]any{"t": "nudge", "id": s.pubID})
	}
}

func (s *session) awaitingWelcome(now time.Time) bool {
	return now.Before(s.welcomeUntil)
}

// greetedLocked runs when g says something or waves: any newcomer close by
// who hasn't been welcomed yet now has been, by g.
func (sv *Server) greetedLocked(g *session) {
	now := sv.opts.Now()
	for n := range g.room.sessions {
		if n == g || !n.awaitingWelcome(now) || chebyshev(n.pos, g.pos) > greetRange {
			continue
		}
		n.welcomeUntil = time.Time{}
		sv.welcomed++
		log.Printf("welcome: %d of %d newcomers greeted", sv.welcomed, sv.newcomers)
		sv.broadcastLocked(g.room, map[string]any{"t": "welcomed", "by": g.pubID, "to": n.pubID}, nil)
		if g.acct != nil {
			go sv.countWelcome(g, g.acct.ID)
		}
	}
}

// countWelcome saves one more welcome for the account and tells its owner
// their running total.
func (sv *Server) countWelcome(g *session, accountID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	n, err := sv.store.AddWelcome(ctx, accountID)
	if err != nil {
		log.Printf("save welcome: %v", err)
		return
	}
	sv.mu.Lock()
	defer sv.mu.Unlock()
	if g.acct != nil && g.acct.ID == accountID {
		g.acct.Welcomes = n
		sv.enqueue(g, map[string]any{"t": "welcomes", "count": n})
	}
}

func chebyshev(a, b world.Point) int { return max(abs(a.X-b.X), abs(a.Y-b.Y)) }
