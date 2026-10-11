package world

import "testing"

func TestDefaultRoomsValid(t *testing.T) {
	if err := Validate(DefaultRooms()); err != nil {
		t.Fatal(err)
	}
}

func TestValidateCatchesBadDoor(t *testing.T) {
	rooms := []*Room{{
		ID: "a", Rows: []string{"..."}, Spawn: Point{0, 0},
		Doors: []Door{{At: Point{2, 0}, Target: "missing"}},
	}}
	if err := Validate(rooms); err == nil {
		t.Fatal("expected error for door to unknown room")
	}
}

func TestFindPath(t *testing.T) {
	r := &Room{Rows: []string{
		".....",
		".###.",
		".....",
	}}
	path := FindPath(r, Point{0, 1}, Point{4, 1})
	if len(path) != 6 {
		t.Fatalf("want 6 steps around the wall, got %d: %v", len(path), path)
	}
	if path[len(path)-1] != (Point{4, 1}) {
		t.Fatalf("path should end at goal, got %v", path)
	}
	for i, p := range path {
		if !r.Floor(p) {
			t.Fatalf("step %d %v is not floor", i, p)
		}
	}
}

func TestFindPathUnreachable(t *testing.T) {
	r := &Room{Rows: []string{". ."}}
	if p := FindPath(r, Point{0, 0}, Point{2, 0}); p != nil {
		t.Fatalf("want nil, got %v", p)
	}
	if p := FindPath(r, Point{0, 0}, Point{0, 0}); p != nil {
		t.Fatalf("want nil for same tile, got %v", p)
	}
}

func TestFindPathDoorsOnlyAsGoal(t *testing.T) {
	r := &Room{
		Rows:  []string{"...", "...", "..."},
		Doors: []Door{{At: Point{1, 0}}},
	}
	// Walking along the top edge must route around the door tile.
	path := FindPath(r, Point{0, 0}, Point{2, 0})
	for _, p := range path {
		if p == (Point{1, 0}) {
			t.Fatalf("path crossed a door it wasn't heading to: %v", path)
		}
	}
	// But the door itself is a valid destination.
	if p := FindPath(r, Point{1, 2}, Point{1, 0}); len(p) != 2 {
		t.Fatalf("want 2 steps to the door, got %v", p)
	}
}
