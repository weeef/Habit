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
}

// Room is the static definition of a room.
//
// Rows draws the floor plan: '.' is walkable floor, anything else is empty
// space. Rows[y][x] is the tile at (x, y).
type Room struct {
	ID       string
	Name     string
	Kind     Kind
	Theme    string // look of the room; the client picks colours from it
	Rows     []string
	Spawn    Point
	Doors    []Door
	Capacity int
}

func (r *Room) W() int { return len(r.Rows[0]) }
func (r *Room) H() int { return len(r.Rows) }

// Floor reports whether p is a walkable floor tile.
func (r *Room) Floor(p Point) bool {
	return p.Y >= 0 && p.Y < r.H() && p.X >= 0 && p.X < len(r.Rows[p.Y]) && r.Rows[p.Y][p.X] == '.'
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
		if len(r.Rows) == 0 {
			return fmt.Errorf("room %q has no rows", r.ID)
		}
		if !r.Floor(r.Spawn) || r.DoorAt(r.Spawn) != nil {
			return fmt.Errorf("room %q: spawn %v is not plain floor", r.ID, r.Spawn)
		}
		for _, d := range r.Doors {
			if !r.Floor(d.At) {
				return fmt.Errorf("room %q: door at %v is not on floor", r.ID, d.At)
			}
			t, ok := byID[d.Target]
			if !ok {
				return fmt.Errorf("room %q: door targets unknown room %q", r.ID, d.Target)
			}
			if !t.Floor(d.TargetAt) || t.DoorAt(d.TargetAt) != nil {
				return fmt.Errorf("room %q: door lands on %v in %q, which is not plain floor", r.ID, d.TargetAt, t.ID)
			}
		}
	}
	return nil
}
