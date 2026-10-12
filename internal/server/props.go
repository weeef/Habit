package server

import (
	"time"

	"habit/internal/store"
	"habit/internal/world"
)

// Things you can do with props: toss a coin in the fountain, shake a tree,
// ring the bell, sit on a bench. Using a prop that's out of reach walks you
// over first; the action happens when you arrive. Effects are broadcast as
// "fx" so everyone in the room sees them.

const (
	useBurst   = 3
	usePerSec  = 0.5
	bellQuiet  = 4 * time.Second // the bell can't be rung again this soon, by anyone
	styleSkins = 6               // palette sizes the client draws from
	styleHairs = 8
	styleFits  = 8
	styleAccs  = 4
)

type propView struct {
	ID      string      `json:"id"`
	Kind    string      `json:"kind"`
	Variant string      `json:"variant,omitempty"`
	At      world.Point `json:"at"`
	W       int         `json:"w"`
	H       int         `json:"h"`
	Dir     world.Point `json:"dir"`
	Count   int         `json:"count,omitempty"` // fountain: wishes made
}

func (r *room) propViews() []propView {
	out := make([]propView, len(r.def.Props))
	for i, p := range r.def.Props {
		out[i] = propView{ID: p.ID, Kind: p.Kind, Variant: p.Variant, At: p.At, W: max(p.W, 1), H: max(p.H, 1), Dir: p.Dir, Count: r.uses[p.ID]}
	}
	return out
}

func randomStyle() store.Style {
	b := randomBytes(4)
	return store.Style{
		Skin: int(b[0]) % styleSkins, Hair: int(b[1]) % styleHairs,
		Outfit: int(b[2]) % styleFits, Accessory: int(b[3]) % styleAccs,
	}
}

func validStyle(st store.Style) bool {
	return st.Skin >= 0 && st.Skin < styleSkins && st.Hair >= 0 && st.Hair < styleHairs &&
		st.Outfit >= 0 && st.Outfit < styleFits && st.Accessory >= 0 && st.Accessory < styleAccs
}

// adjacent reports whether pos is next to (4-way) one of the prop's tiles.
func adjacent(p *world.Prop, pos world.Point) bool {
	for _, t := range p.Tiles() {
		if abs(t.X-pos.X)+abs(t.Y-pos.Y) == 1 {
			return true
		}
	}
	return false
}

// handleUseLocked uses a prop now if it's in reach, or walks there first.
func (sv *Server) handleUseLocked(s *session, id string) {
	r := s.room
	p := r.def.PropByID(id)
	if p == nil || p.Kind == world.PropLamp || p.Flat() {
		return
	}
	if p.Seat() {
		sv.walkToSeatLocked(s, p)
		return
	}
	if adjacent(p, s.pos) && len(s.path) == 0 {
		sv.useLocked(s, p)
		return
	}
	// Walk to the nearest free tile beside it, then use it on arrival.
	var best []world.Point
	for _, t := range p.Tiles() {
		for _, d := range []world.Point{{X: 1}, {X: -1}, {Y: 1}, {Y: -1}} {
			q := world.Point{X: t.X + d.X, Y: t.Y + d.Y}
			if !r.def.Standable(q) {
				continue
			}
			if q == s.pos {
				best = []world.Point{} // already there; just finish the current step
				break
			}
			if path := sv.routeLocked(s, q); path != nil && (best == nil || len(path) < len(best)) {
				best = path
			}
		}
	}
	if best == nil {
		return
	}
	if len(best) == 0 {
		sv.useLocked(s, p)
		return
	}
	sv.standUpLocked(s)
	sv.setPathLocked(s, best)
	s.pendingUse = p.ID
}

// walkToSeatLocked walks to the nearest empty seat on a bench.
func (sv *Server) walkToSeatLocked(s *session, p *world.Prop) {
	taken := map[world.Point]bool{}
	for o := range s.room.sessions {
		if o != s {
			taken[o.pos] = true
		}
	}
	var best []world.Point
	for _, t := range p.Tiles() {
		if taken[t] || t == s.pos {
			continue
		}
		if path := sv.routeLocked(s, t); path != nil && (best == nil || len(path) < len(best)) {
			best = path
		}
	}
	if best != nil {
		sv.standUpLocked(s)
		sv.setPathLocked(s, best)
	}
}

// useLocked does the prop's action and shows it to the room.
func (sv *Server) useLocked(s *session, p *world.Prop) {
	now := sv.opts.Now()
	switch p.Kind {
	case world.PropBoard, world.PropSubway, world.PropNewsbox:
		return // the client opens these itself; walking over was the point
	}
	if !s.useLim.allow(now) {
		return
	}
	fx := map[string]any{"t": "fx", "prop": p.ID, "kind": p.Kind, "by": s.pubID}
	switch p.Kind {
	case world.PropFountain:
		s.room.uses[p.ID]++
		fx["count"] = s.room.uses[p.ID]
		sv.wishedLocked()
		fx["today"], fx["goal"] = sv.today.wishes, wishGoal
	case world.PropBell:
		if now.Sub(s.room.lastBell) < bellQuiet {
			return
		}
		s.room.lastBell = now
	case world.PropTree, world.PropHydrant:
	case world.PropCart:
		fx["variant"] = p.Variant // everyone sees what you grabbed
	default:
		return
	}
	sv.broadcastLocked(s.room, fx, nil)
}

// arrivedLocked runs when an avatar finishes a walk: sit down on a seat, or
// use the prop they walked over to.
func (sv *Server) arrivedLocked(s *session) {
	if seat := s.room.def.SeatAt(s.pos); seat != nil {
		s.seat = seat.ID
		sv.broadcastLocked(s.room, map[string]any{"t": "update", "avatar": sv.viewLocked(s)}, nil)
	}
	if id := s.pendingUse; id != "" {
		s.pendingUse = ""
		if p := s.room.def.PropByID(id); p != nil && adjacent(p, s.pos) {
			sv.useLocked(s, p)
		}
	}
}

func (sv *Server) standUpLocked(s *session) {
	s.pendingUse = ""
	if s.seat != "" {
		s.seat = ""
		sv.broadcastLocked(s.room, map[string]any{"t": "update", "avatar": sv.viewLocked(s)}, nil)
	}
}
