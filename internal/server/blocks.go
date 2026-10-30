package server

import (
	"cmp"
	"context"
	"errors"
	"log"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"habit/internal/store"
	"habit/internal/world"
)

// Blocks (PLAN.md §2.6) are communities anyone with an account can start,
// written h/slug. Each is a room to walk into with its own board. Rooms
// for blocks are only kept in memory while someone's around: they load on
// the first visit and unload a little after the last person leaves.
const (
	blockIdle       = 2 * time.Minute // empty this long and a block's room unloads
	blocksPerDay    = 3               // new blocks one account can start a day
	maxBlockName    = 30
	maxBlockAbout   = 200
	directoryLimit  = 50
	directorySearch = 200
)

var slugPattern = regexp.MustCompile(`^[a-z0-9_]{3,21}$`)

// Names nobody can take: the site's own words, and ones that would pass
// for staff.
var reservedSlugs = map[string]bool{
	"plaza": true, "habit": true, "admin": true, "admins": true, "mod": true, "mods": true,
	"moderator": true, "moderators": true, "staff": true, "official": true, "support": true,
	"help": true, "api": true, "ws": true, "h": true, "settings": true, "login": true,
	"signup": true, "about": true, "terms": true, "privacy": true, "safety": true, "news": true,
}

var impersonation = []string{"official", "admin", "habit_", "_habit", "staff", "moderator"}

// The bullets and colours a block can choose from, so every line on the
// subway map stays readable.
var (
	blockBullets = []string{
		"🎮", "🏋️", "🎵", "🎨", "📚", "🎬", "🍕", "☕", "🌱", "🐶", "🐱", "⚽", "🏀", "🎸", "📷", "✈️",
		"💻", "🧶", "🎲", "🧠", "🚲", "🌙", "🔥", "💬", "🎤", "🛹", "🍜", "🌸", "🧪", "🏳️‍🌈",
	}
	blockColors = []string{
		"#e5484d", "#f08a3c", "#e6b400", "#3cc28f", "#2fa3c7", "#4f7fd6", "#8b6bf2", "#d6409f", "#8a6a4f", "#5c6370",
	}
)

// official blocks, created at startup if missing. Their rooms are part of
// the starting world.
var officialBlocks = []store.Block{
	{Slug: "gaming", Name: "Gaming", Bullet: "🎮", Color: "#8b6bf2", About: "Games, streams and finding a squad.", Template: "arcade", Official: true},
	{Slug: "gym", Name: "Gym", Bullet: "🏋️", Color: "#f08a3c", About: "Workouts, PRs and keeping each other going.", Template: "gym", Official: true},
}

type blockView struct {
	Slug     string `json:"slug"`
	Name     string `json:"name"`
	Bullet   string `json:"bullet"`
	Color    string `json:"color"`
	About    string `json:"about"`
	Official bool   `json:"official"`
	Count    int    `json:"count"`
	Yours    bool   `json:"yours,omitempty"`
	Street   string `json:"street,omitempty"` // its storefront's avenue, by name
	Lot      int    `json:"lot,omitempty"`
}

func viewOfBlock(b *store.Block) blockView {
	return blockView{
		Slug: b.Slug, Name: b.Name, Bullet: b.Bullet, Color: b.Color, About: b.About, Official: b.Official,
		Street: world.AvenueName(b.Street), Lot: b.Lot,
	}
}

// seedBlocks makes sure the official blocks exist and ties them to their
// rooms.
func (sv *Server) seedBlocks(ctx context.Context) error {
	for _, b := range officialBlocks {
		b.CreatedAt = sv.opts.Now()
		if err := sv.store.CreateBlock(ctx, &b); err != nil && !errors.Is(err, store.ErrNameTaken) {
			return err
		}
		saved, err := sv.store.BlockBySlug(ctx, b.Slug)
		if err != nil {
			return err
		}
		if r := sv.rooms[b.Slug]; r != nil {
			r.block = saved
		}
	}
	return nil
}

// roomFor returns the room for a slug, loading its block if nobody's been
// there lately. It returns nil if there's no such block.
func (sv *Server) roomFor(ctx context.Context, slug string) (*room, error) {
	slug = strings.ToLower(strings.TrimSpace(slug))
	sv.mu.Lock()
	r := sv.rooms[slug]
	sv.mu.Unlock()
	if r != nil {
		return r, nil
	}
	if !slugPattern.MatchString(slug) {
		return nil, nil
	}
	b, err := sv.store.BlockBySlug(ctx, slug)
	if errors.Is(err, store.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	def := world.BlockRoom(b.Slug, b.Bullet+" "+b.Name, b.Template)
	if world.ValidLot(b.Street, b.Lot) {
		world.SetExit(def, b.Street, world.AvenueExit(b.Lot), "🏙️ "+world.AvenueName(b.Street))
	}
	saved, err := sv.store.RecentMessages(ctx, def.ID, chatHistoryLen)
	if err != nil {
		return nil, err
	}
	saved = sv.visibleChat(saved)

	sv.mu.Lock()
	defer sv.mu.Unlock()
	if r := sv.rooms[slug]; r != nil {
		return r, nil // someone else loaded it meanwhile
	}
	if err := world.ValidateRoom(def, func(id string) *world.Room {
		if r := sv.rooms[id]; r != nil {
			return r.def
		}
		return nil
	}); err != nil {
		return nil, err
	}
	r = newRoom(def, saved)
	r.block = b
	r.emptySince = sv.opts.Now()
	sv.rooms[slug] = r
	sv.order = append(sv.order, slug)
	return r, nil
}

func newRoom(def *world.Room, saved []store.Message) *room {
	r := &room{def: def, sessions: map[*session]struct{}{}, uses: map[string]int{}}
	for _, m := range saved {
		r.history = append(r.history, chatMsg{From: m.AvatarID, Name: m.Name, Text: m.Text, TS: m.SentAt, Newbie: m.Newbie})
	}
	return r
}

// unloadIdleLocked lets go of block rooms nobody has been in for a while.
// Their posts and chat are in the store, ready for the next visit.
func (sv *Server) unloadIdleLocked(now time.Time) {
	kept := sv.order[:0]
	for _, id := range sv.order {
		r := sv.rooms[id]
		if !r.pinned && len(r.sessions) == 0 && now.Sub(r.emptySince) >= blockIdle {
			delete(sv.rooms, id)
			continue
		}
		kept = append(kept, id)
	}
	sv.order = kept
}

// handleBlocks answers a search of the block directory: the busiest first.
func (sv *Server) handleBlocks(ctx context.Context, s *session, query string) {
	query = strings.TrimSpace(query)
	if utf8.RuneCountInString(query) > maxBlockName {
		return
	}
	found, err := sv.store.SearchBlocks(ctx, query, directorySearch)
	if err != nil {
		sv.storeFailed(s, "search blocks", err)
		return
	}
	sv.mu.Lock()
	defer sv.mu.Unlock()
	views := make([]blockView, 0, len(found))
	for i := range found {
		v := viewOfBlock(&found[i])
		if r := sv.rooms[v.Slug]; r != nil {
			v.Count = len(r.sessions)
		}
		v.Yours = s.acct != nil && found[i].OwnerID == s.acct.ID
		views = append(views, v)
	}
	slices.SortStableFunc(views, func(a, b blockView) int { return cmp.Compare(b.Count, a.Count) })
	sv.enqueue(s, map[string]any{"t": "blocks", "q": query, "blocks": views[:min(len(views), directoryLimit)]})
}

type blockForm struct {
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	Bullet   string `json:"bullet"`
	Color    string `json:"color"`
	About    string `json:"about"`
	Template string `json:"template"`
}

func (sv *Server) blockError(s *session, message string) {
	sv.enqueue(s, map[string]any{"t": "block_error", "message": message})
}

// blockFormProblem says what's wrong with a new block, if anything.
func blockFormProblem(f *blockForm) string {
	switch {
	case !slugPattern.MatchString(f.Slug):
		return "The short name needs 3–21 lowercase letters, numbers or underscores."
	case reservedSlugs[f.Slug]:
		return "h/" + f.Slug + " is reserved. Try another short name."
	case slices.ContainsFunc(impersonation, func(w string) bool { return strings.Contains(f.Slug, w) }):
		return "Short names can't look official. Try another."
	case !validText(f.Name, maxBlockName):
		return "Give your block a name (up to 30 characters)."
	case utf8.RuneCountInString(f.About) > maxBlockAbout || !utf8.ValidString(f.About):
		return "The description can be up to 200 characters."
	case !slices.Contains(blockBullets, f.Bullet):
		return "Pick an icon for your block."
	case !slices.Contains(blockColors, f.Color):
		return "Pick a colour for your block."
	case !world.ValidTemplate(f.Template):
		return "Pick a room for your block."
	}
	return ""
}

// handleCreateBlock starts a new block for an account holder and takes
// them into it.
func (sv *Server) handleCreateBlock(ctx context.Context, s *session, f *blockForm) {
	if f == nil {
		return
	}
	f.Name, f.About = strings.TrimSpace(f.Name), strings.TrimSpace(f.About)
	f.Slug = strings.ToLower(strings.TrimSpace(f.Slug))
	if problem := blockFormProblem(f); problem != "" {
		sv.blockError(s, problem)
		return
	}
	sv.mu.Lock()
	acct := s.acct
	if acct == nil {
		sv.enqueue(s, map[string]any{"t": "need_account"})
		sv.mu.Unlock()
		return
	}
	now := sv.opts.Now()
	if ageBand(acct.BirthYear, now) != BandAdult {
		for _, text := range []string{f.Name, f.About} {
			if reason := checkPersonalInfo(text); reason != "" {
				sv.blockError(s, reason)
				sv.mu.Unlock()
				return
			}
		}
	}
	sv.mu.Unlock()

	n, err := sv.store.BlocksStartedSince(ctx, acct.ID, now.Add(-24*time.Hour))
	if err != nil {
		sv.storeFailed(s, "count blocks", err)
		return
	}
	if n >= blocksPerDay {
		sv.blockError(s, "You've started a few blocks today. Give them some love and try again tomorrow.")
		return
	}
	b := &store.Block{
		Slug: f.Slug, Name: f.Name, Bullet: f.Bullet, Color: f.Color, About: f.About,
		Template: f.Template, OwnerID: acct.ID, CreatedAt: now,
	}
	if err := sv.store.CreateBlock(ctx, b); err != nil {
		if errors.Is(err, store.ErrNameTaken) {
			sv.blockError(s, "h/"+f.Slug+" is taken. Try another short name.")
			return
		}
		sv.storeFailed(s, "create block", err)
		return
	}
	log.Printf("block h/%s started", b.Slug)
	v := viewOfBlock(b)
	v.Yours = true
	sv.enqueue(s, map[string]any{"t": "block_created", "block": v})
	sv.goTo(ctx, s, b.Slug)
}

// goTo takes s to the room for slug, loading it if need be.
func (sv *Server) goTo(ctx context.Context, s *session, slug string) {
	r, err := sv.roomFor(ctx, slug)
	if err != nil {
		log.Printf("load h/%s: %v", slug, err)
		sv.mu.Lock()
		sv.enqueue(s, map[string]any{"t": "notice", "message": ErrUnavailable.Error()})
		sv.mu.Unlock()
		return
	}
	sv.mu.Lock()
	defer sv.mu.Unlock()
	if r == nil {
		sv.enqueue(s, map[string]any{"t": "notice", "message": "There's no block called h/" + strings.ToLower(slug) + " yet."})
		return
	}
	if sv.rooms[r.def.ID] != r || s.room == nil || s.room == r {
		return // unloaded meanwhile (it was empty), or nothing to do
	}
	sv.moveLocked(s, r, r.def.Spawn)
}

// ---- storefronts on the avenues ----

// openStorefronts puts every rented lot's door on its avenue at startup.
func (sv *Server) openStorefronts(ctx context.Context) error {
	lots, err := sv.store.BlocksWithLots(ctx)
	if err != nil {
		return err
	}
	for i := range lots {
		b := &lots[i]
		if r := sv.rooms[b.Street]; r != nil && world.ValidLot(b.Street, b.Lot) {
			world.AddLotDoor(r.def, b.Lot, b.Slug, b.Bullet+" "+b.Name, b.Color)
		}
	}
	return nil
}

// handleMyBlocks lists the blocks you started, for picking one to put in a
// shop.
func (sv *Server) handleMyBlocks(ctx context.Context, s *session) {
	sv.mu.Lock()
	acct := s.acct
	sv.mu.Unlock()
	if acct == nil {
		sv.enqueue(s, map[string]any{"t": "need_account"})
		return
	}
	mine, err := sv.store.BlocksByOwner(ctx, acct.ID)
	if err != nil {
		sv.storeFailed(s, "my blocks", err)
		return
	}
	views := make([]blockView, 0, len(mine))
	for i := range mine {
		v := viewOfBlock(&mine[i])
		v.Yours = true
		views = append(views, v)
	}
	sv.enqueue(s, map[string]any{"t": "my_blocks", "blocks": views})
}

// handleClaimLot moves a block's storefront to a lot on an avenue, or
// closes it (street "", lot 0). Only the block's owner can.
func (sv *Server) handleClaimLot(ctx context.Context, s *session, slug, street string, lot int) {
	closing := street == "" && lot == 0
	if !closing && !world.ValidLot(street, lot) {
		return
	}
	sv.mu.Lock()
	acct := s.acct
	sv.mu.Unlock()
	if acct == nil {
		sv.enqueue(s, map[string]any{"t": "need_account"})
		return
	}
	b, err := sv.store.BlockBySlug(ctx, slug)
	if errors.Is(err, store.ErrNotFound) {
		return
	}
	if err != nil {
		sv.storeFailed(s, "find block", err)
		return
	}
	if b.OwnerID == "" || b.OwnerID != acct.ID {
		sv.blockError(s, "Only the person who started a block can choose its storefront.")
		return
	}
	if err := sv.store.SetBlockLot(ctx, b.Slug, street, lot); err != nil {
		if errors.Is(err, store.ErrNameTaken) {
			sv.blockError(s, "Someone just took that shop. Try another one.")
			return
		}
		sv.storeFailed(s, "set lot", err)
		return
	}

	sv.mu.Lock()
	defer sv.mu.Unlock()
	if old := sv.rooms[b.Street]; old != nil && b.Lot != 0 {
		world.RemoveLotDoor(old.def, b.Lot)
		sv.doorsChangedLocked(old)
	}
	if !closing {
		r := sv.rooms[street]
		world.AddLotDoor(r.def, lot, b.Slug, b.Bullet+" "+b.Name, b.Color)
		sv.doorsChangedLocked(r)
	}
	// The block's own door now leads out to its shop (or the Plaza).
	if r := sv.rooms[b.Slug]; r != nil {
		if closing {
			world.SetExit(r.def, "plaza", world.PlazaExit(b.Slug), "The Plaza")
		} else {
			world.SetExit(r.def, street, world.AvenueExit(lot), "🏙️ "+world.AvenueName(street))
		}
		sv.doorsChangedLocked(r)
	}
	b.Street, b.Lot = street, lot
	v := viewOfBlock(b)
	v.Yours = true
	sv.enqueue(s, map[string]any{"t": "lot_claimed", "block": v})
}

// doorsChangedLocked tells everyone in r its doors have changed.
func (sv *Server) doorsChangedLocked(r *room) {
	sv.broadcastLocked(r, map[string]any{"t": "doors", "room": r.def.ID, "doors": doorViews(r)}, nil)
}
