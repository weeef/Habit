package server

import (
	"context"
	"log"

	"habit/internal/store"
)

// Changing your look after signup. An account either keeps a look it
// designed, or gets a fresh random one at each login ("surprise me"); its
// designed look is kept either way, so switching back restores it.
const (
	lookBurst  = 3
	lookPerSec = 0.2
)

func (sv *Server) handleSetLook(ctx context.Context, s *session, style *store.Style, random bool) {
	sv.mu.Lock()
	if s.acct == nil {
		sv.enqueue(s, map[string]any{"t": "need_account"})
		sv.mu.Unlock()
		return
	}
	if style != nil && !validStyle(*style) {
		sv.mu.Unlock()
		return
	}
	if !s.lookLim.allow(sv.opts.Now()) {
		sv.enqueue(s, map[string]any{"t": "notice", "message": "One moment, still getting changed."})
		sv.mu.Unlock()
		return
	}
	saved := s.acct.Style
	if style != nil {
		saved = *style
	}
	acctID := s.acct.ID
	sv.mu.Unlock()

	if err := sv.store.SetLook(ctx, acctID, saved, random); err != nil {
		log.Printf("set look: %v", err)
		sv.mu.Lock()
		sv.enqueue(s, map[string]any{"t": "notice", "message": ErrUnavailable.Error()})
		sv.mu.Unlock()
		return
	}

	sv.mu.Lock()
	defer sv.mu.Unlock()
	if s.acct == nil || s.acct.ID != acctID {
		return // logged out meanwhile
	}
	s.acct.Style, s.acct.RandomLook = saved, random
	if random {
		s.style = randomStyle() // "surprise me" rolls a new look straight away
	} else {
		s.style = saved
	}
	sv.enqueue(s, map[string]any{"t": "look", "style": s.style, "savedStyle": saved, "randomLook": random})
	if s.room != nil {
		sv.broadcastLocked(s.room, map[string]any{"t": "update", "avatar": sv.viewLocked(s)}, nil)
	}
}
