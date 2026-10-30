package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"habit/internal/store"
)

// Admins (Options.Admins, set from HABIT_ADMINS) can clear chat, ban and
// globally mute accounts, and kick people. They see real usernames in the
// admin panel; nobody else does. Every action goes in the moderation log.

const maxReasonRunes = 200

// isAdminLocked reports whether s is logged in as an admin.
func (sv *Server) isAdminLocked(s *session) bool {
	return s.acct != nil && sv.admins[strings.ToLower(s.acct.Username)]
}

func (sv *Server) isAdminName(username string) bool { return sv.admins[strings.ToLower(username)] }

// untilText describes when a ban or mute ends.
func untilText(until time.Time) string {
	if !until.Before(store.Forever) {
		return "for good"
	}
	return "until " + until.UTC().Format("Jan 2, 15:04 UTC")
}

// mutedLocked tells s, and returns true, if s can't talk because an admin
// muted their account.
func (sv *Server) mutedLocked(s *session) bool {
	if s.acct == nil || !s.acct.Muted(sv.opts.Now()) {
		return false
	}
	sv.enqueue(s, map[string]any{"t": "notice", "message": "You've been muted " + untilText(s.acct.MutedUntil) + ", so you can't chat or post."})
	return true
}

// sessionsOfLocked lists everyone logged in as the account, in any room.
func (sv *Server) sessionsOfLocked(accountID string) []*session {
	var out []*session
	for _, r := range sv.rooms {
		for o := range r.sessions {
			if o.acct != nil && o.acct.ID == accountID {
				out = append(out, o)
			}
		}
	}
	return out
}

// kickLocked tells s why and disconnects it a moment later, so the message
// arrives first.
func (sv *Server) kickLocked(s *session, message string) {
	sv.enqueue(s, map[string]any{"t": "kicked", "message": message})
	time.AfterFunc(300*time.Millisecond, s.close)
}

func (sv *Server) modReply(s *session, message string) {
	sv.mu.Lock()
	defer sv.mu.Unlock()
	sv.enqueue(s, map[string]any{"t": "mod_done", "message": message})
}

func (sv *Server) handleMod(ctx context.Context, s *session, m inMsg) {
	sv.mu.Lock()
	if !sv.isAdminLocked(s) || s.room == nil {
		sv.mu.Unlock()
		return
	}
	admin := s.acct.ID
	here := s.room
	if m.Action == "people" {
		sv.sendPeopleLocked(s)
		sv.mu.Unlock()
		return
	}
	sv.mu.Unlock()

	reason := strings.TrimSpace(m.Reason)
	if r := []rune(reason); len(r) > maxReasonRunes {
		reason = string(r[:maxReasonRunes])
	}
	logAction := func(action, target, room, detail string) {
		err := sv.store.LogModAction(ctx, store.ModAction{At: sv.opts.Now(), AdminID: admin, Action: action, TargetID: target, Room: room, Detail: detail})
		if err != nil {
			log.Printf("mod log: %v", err)
		}
	}

	if m.Action == "clear_chat" {
		room := here.def.ID
		if m.Room == "*" {
			room = "" // every room
		}
		n, err := sv.store.DeleteMessages(ctx, room)
		if err != nil {
			log.Printf("clear chat: %v", err)
			sv.modReply(s, ErrUnavailable.Error())
			return
		}
		sv.mu.Lock()
		for id, r := range sv.rooms {
			if room == "" || id == room {
				r.history = nil
				sv.broadcastLocked(r, map[string]any{"t": "chat_cleared", "room": id}, nil)
			}
		}
		sv.mu.Unlock()
		where := "this room"
		if room == "" {
			where = "every room"
		}
		logAction("clear_chat", "", room, fmt.Sprintf("%d messages", n))
		sv.modReply(s, fmt.Sprintf("Cleared the chat in %s (%d saved messages deleted).", where, n))
		return
	}

	// Everything else acts on someone: a person in the admin's room (by
	// their avatar ID), or any account by username.
	var target *store.Account
	var guest *session
	if m.Target != "" {
		sv.mu.Lock()
		for o := range here.sessions {
			if o.pubID == m.Target {
				if o.acct != nil {
					c := *o.acct
					target = &c
				} else {
					guest = o
				}
			}
		}
		sv.mu.Unlock()
		if target == nil && guest == nil {
			sv.modReply(s, "They've left the room.")
			return
		}
	} else {
		acct, err := sv.store.AccountByUsername(ctx, strings.TrimSpace(m.Username))
		if errors.Is(err, store.ErrNotFound) {
			sv.modReply(s, "There's no account called "+m.Username+".")
			return
		}
		if err != nil {
			log.Printf("mod lookup: %v", err)
			sv.modReply(s, ErrUnavailable.Error())
			return
		}
		target = acct
	}

	if guest != nil {
		// Guests have no account to ban or mute; they can only be sent off.
		if m.Action != "kick" && m.Action != "ban" {
			sv.modReply(s, "That's a visitor without an account. They can't chat, so there's nothing to mute.")
			return
		}
		sv.mu.Lock()
		sv.kickLocked(guest, "You've been removed by a moderator.")
		sv.mu.Unlock()
		logAction("kick", "", here.def.ID, "a visitor")
		sv.modReply(s, "Removed the visitor.")
		return
	}
	if sv.isAdminName(target.Username) {
		sv.modReply(s, "Admins can't act on other admins.")
		return
	}

	now := sv.opts.Now()
	until := store.Forever
	if m.Minutes > 0 {
		until = now.Add(time.Duration(min(m.Minutes, 10*365*24*60)) * time.Minute)
	}
	var err error
	var done string
	switch m.Action {
	case "ban":
		err = sv.store.SetBan(ctx, target.ID, until, reason)
		done = fmt.Sprintf("Banned %s %s.", target.Username, untilText(until))
	case "unban":
		err = sv.store.SetBan(ctx, target.ID, time.Time{}, "")
		done = "Unbanned " + target.Username + "."
	case "mute":
		err = sv.store.SetMute(ctx, target.ID, until)
		done = fmt.Sprintf("Muted %s %s.", target.Username, untilText(until))
	case "unmute":
		err = sv.store.SetMute(ctx, target.ID, time.Time{})
		done = "Unmuted " + target.Username + "."
	case "kick":
		done = "Kicked " + target.Username + "."
	default:
		return
	}
	if err != nil {
		log.Printf("mod %s: %v", m.Action, err)
		sv.modReply(s, ErrUnavailable.Error())
		return
	}

	// Apply it to them right now, wherever they are.
	sv.mu.Lock()
	for _, o := range sv.sessionsOfLocked(target.ID) {
		switch m.Action {
		case "ban":
			o.acct.BannedUntil, o.acct.BanReason = until, reason
			msg := "Your account has been banned " + untilText(until) + "."
			if reason != "" {
				msg += " Reason: " + reason
			}
			sv.kickLocked(o, msg)
		case "unban":
			o.acct.BannedUntil, o.acct.BanReason = time.Time{}, ""
		case "mute":
			o.acct.MutedUntil = until
			sv.enqueue(o, map[string]any{"t": "notice", "message": "A moderator muted you " + untilText(until) + "."})
		case "unmute":
			o.acct.MutedUntil = time.Time{}
			sv.enqueue(o, map[string]any{"t": "notice", "message": "You can chat again."})
		case "kick":
			sv.kickLocked(o, "You've been removed by a moderator.")
		}
	}
	sv.mu.Unlock()

	detail := reason
	if m.Action == "ban" || m.Action == "mute" {
		detail = strings.TrimSpace(untilText(until) + " " + reason)
	}
	logAction(m.Action, target.ID, here.def.ID, detail)
	sv.modReply(s, done)
}

// sendPeopleLocked sends an admin the people in their room, with real
// usernames, so they can act on them.
func (sv *Server) sendPeopleLocked(s *session) {
	now := sv.opts.Now()
	people := []map[string]any{}
	for o := range s.room.sessions {
		if o == s {
			continue
		}
		p := map[string]any{"id": o.pubID, "name": sv.displayNameLocked(o), "guest": o.acct == nil}
		if o.acct != nil {
			p["username"] = o.acct.Username
			p["muted"] = o.acct.Muted(now)
			p["admin"] = sv.isAdminName(o.acct.Username)
		}
		people = append(people, p)
	}
	sv.enqueue(s, map[string]any{"t": "mod_people", "room": s.room.def.Name, "people": people})
}
