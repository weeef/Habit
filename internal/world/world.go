// Package world defines room layouts, the doors that link rooms together,
// and tile pathfinding. It holds no runtime state.
package world

import (
	"encoding/json"
	"fmt"
)

// Point is a tile coordinate. It encodes to JSON as [x, y].
type Point struct{ X, Y int }

func (p Point) MarshalJSON() ([]byte, error) { return json.Marshal([2]int{p.X, p.Y}) }

func (p *Point) UnmarshalJSON(b []byte) error {
	var a [2]int
	if err := json.Unmarshal(b, &a); err != nil {
		return err
	}
	p.X, p.Y = a[0], a[1]
	return nil
}

// Kind says what a room is for.
type Kind string

const (
	KindPlaza     Kind = "plaza"
	KindCommunity Kind = "community" // a community's own space, like Gaming or Gym
	KindChat      Kind = "chat"
)

// Door is a tile that moves an avatar into another room when stepped on.
type Door struct {
	At       Point
	Target   string // target room ID
	TargetAt Point  // where the avatar appears in the target room
	Label    string
	Color    string // a block's storefront colour, #rrggbb
}

// Prop kinds. Benches are seats: you can walk onto them (as the last step)
// and sit. Everything else blocks its tiles; most can be used.
const (
	PropFountain = "fountain" // toss a coin and make a wish
	PropBench    = "bench"    // sit
	PropTree     = "tree"     // shake it
	PropBell     = "bell"     // ring it
	PropLamp     = "lamp"     // decoration
	PropBush     = "bush"     // decoration: a leafy shrub (Variant "flowers" blooms)
	PropBoard    = "board"    // the room's noticeboard
	PropClock    = "clock"    // shows the plaza's time of day
	PropCart     = "cart"     // a food cart (Variant: hotdog, pretzel, halal, coffee); grab a snack
	PropHydrant  = "hydrant"  // sprays water
	PropNewsbox  = "newsbox"  // read a headline
	PropSubway   = "subway"   // a subway entrance: ride to another room
	PropManhole  = "manhole"  // steams; you can walk over it

	// Furniture for community rooms (Gaming's arcade, the Gym).
	PropCabinet   = "cabinet"   // an arcade cabinet (Variant: its colour); play a round
	PropScreen    = "screen"    // a big screen on the back wall; decoration
	PropVending   = "vending"   // a vending machine; grab a soda
	PropTreadmill = "treadmill" // decoration
	PropWeights   = "weights"   // a dumbbell rack; do some reps
	PropPunchbag  = "punchbag"  // throw a punch
	PropCooler    = "cooler"    // a water cooler; grab some water
)

// Decorative reports whether a prop is only for looking at: using it does
// nothing, so nobody walks over to it.
func (p *Prop) Decorative() bool {
	switch p.Kind {
	case PropLamp, PropBush, PropScreen, PropTreadmill:
		return true
	}
	return false
}

// Prop is an object in a room. It covers W×H tiles from At.
type Prop struct {
	ID      string
	Kind    string
	Variant string // e.g. which food a cart sells
	At      Point
	W, H    int
	Dir     Point // benches: the way a sitter faces; carts and furniture: the way they face
}

func (p *Prop) size() (int, int) { return max(p.W, 1), max(p.H, 1) }

// Covers reports whether the prop sits on tile q.
func (p *Prop) Covers(q Point) bool {
	w, h := p.size()
	return q.X >= p.At.X && q.X < p.At.X+w && q.Y >= p.At.Y && q.Y < p.At.Y+h
}

// Tiles lists the tiles the prop covers.
func (p *Prop) Tiles() []Point {
	w, h := p.size()
	var out []Point
	for y := p.At.Y; y < p.At.Y+h; y++ {
		for x := p.At.X; x < p.At.X+w; x++ {
			out = append(out, Point{x, y})
		}
	}
	return out
}

// Seat reports whether the prop is something you sit on.
func (p *Prop) Seat() bool { return p.Kind == PropBench }

// Flat reports whether the prop lies flat on the ground, so you can walk
// straight over it.
func (p *Prop) Flat() bool { return p.Kind == PropManhole }

// Room is the static definition of a room.
//
// Rows draws the floor plan: '.', ',', ':' and ';' are walkable floor in four
// materials (the client picks colours per room, e.g. paving, cobbles, paths
// and lawns in the Plaza); anything else is empty space. Rows[y][x] is the tile at (x, y).
type Room struct {
	ID       string
	Name     string
	Kind     Kind
	Theme    string // look of the room; the client picks colours from it
	Rows     []string
	Spawn    Point
	Doors    []Door
	Props    []Prop
	Capacity int
	Lots     []Point // avenues: shop lots a block can rent for a storefront door
}

// AtSpawn as a door's TargetAt means "wherever the room's spawn is", for
// doors into rooms that are built later (blocks).
var AtSpawn = Point{-1, -1}

func (r *Room) W() int { return len(r.Rows[0]) }
func (r *Room) H() int { return len(r.Rows) }

// Floor reports whether p is a floor tile (whether or not a prop stands on it).
func (r *Room) Floor(p Point) bool {
	if p.Y < 0 || p.Y >= r.H() || p.X < 0 || p.X >= len(r.Rows[p.Y]) {
		return false
	}
	switch r.Rows[p.Y][p.X] {
	case '.', ',', ':', ';':
		return true
	}
	return false
}

// PropAt returns the prop on tile p, or nil.
func (r *Room) PropAt(p Point) *Prop {
	for i := range r.Props {
		if r.Props[i].Covers(p) {
			return &r.Props[i]
		}
	}
	return nil
}

// PropByID returns the prop with the given ID, or nil.
func (r *Room) PropByID(id string) *Prop {
	for i := range r.Props {
		if r.Props[i].ID == id {
			return &r.Props[i]
		}
	}
	return nil
}

// Blocked reports whether a prop that can't be walked onto stands on p.
func (r *Room) Blocked(p Point) bool {
	pr := r.PropAt(p)
	return pr != nil && !pr.Seat() && !pr.Flat()
}

// SeatAt returns the seat on tile p, or nil.
func (r *Room) SeatAt(p Point) *Prop {
	if pr := r.PropAt(p); pr != nil && pr.Seat() {
		return pr
	}
	return nil
}

// Standable reports whether an avatar can stand on p: floor with no door,
// and no prop except a flat one.
func (r *Room) Standable(p Point) bool {
	pr := r.PropAt(p)
	return r.Floor(p) && (pr == nil || pr.Flat()) && r.DoorAt(p) == nil
}

// DoorAt returns the door on tile p, or nil.
func (r *Room) DoorAt(p Point) *Door {
	for i := range r.Doors {
		if r.Doors[i].At == p {
			return &r.Doors[i]
		}
	}
	return nil
}

// Validate checks a set of rooms for broken spawns and doors.
func Validate(rooms []*Room) error {
	byID := map[string]*Room{}
	for _, r := range rooms {
		if _, dup := byID[r.ID]; dup {
			return fmt.Errorf("duplicate room %q", r.ID)
		}
		byID[r.ID] = r
	}
	for _, r := range rooms {
		if err := ValidateRoom(r, func(id string) *Room { return byID[id] }); err != nil {
			return err
		}
	}
	return nil
}

// ValidateRoom checks one room, looking up the rooms its doors lead to.
func ValidateRoom(r *Room, lookup func(id string) *Room) error {
	if len(r.Rows) == 0 {
		return fmt.Errorf("room %q has no rows", r.ID)
	}
	if !r.Standable(r.Spawn) {
		return fmt.Errorf("room %q: spawn %v is not plain floor", r.ID, r.Spawn)
	}
	for _, d := range r.Doors {
		if !r.Floor(d.At) || r.PropAt(d.At) != nil {
			return fmt.Errorf("room %q: door at %v is not on clear floor", r.ID, d.At)
		}
		t := lookup(d.Target)
		if t == nil {
			return fmt.Errorf("room %q: door targets unknown room %q", r.ID, d.Target)
		}
		if !t.Standable(d.TargetAt) {
			return fmt.Errorf("room %q: door lands on %v in %q, which is not plain floor", r.ID, d.TargetAt, t.ID)
		}
	}
	ids := map[string]bool{}
	for i := range r.Props {
		p := &r.Props[i]
		if p.ID == "" || ids[p.ID] {
			return fmt.Errorf("room %q: prop IDs must be set and unique (%q)", r.ID, p.ID)
		}
		ids[p.ID] = true
		for _, q := range p.Tiles() {
			if !r.Floor(q) {
				return fmt.Errorf("room %q: prop %q at %v is off the floor", r.ID, p.ID, q)
			}
			if o := r.PropAt(q); o != p {
				return fmt.Errorf("room %q: props %q and %q overlap at %v", r.ID, p.ID, o.ID, q)
			}
		}
		if p.Seat() && p.Dir == (Point{}) {
			return fmt.Errorf("room %q: bench %q needs a direction", r.ID, p.ID)
		}
	}
	return nil
}
