package world

import "testing"

func TestDefaultRoomsValid(t *testing.T) {
	if err := Validate(DefaultRooms()); err != nil {
		t.Fatal(err)
	}
}

func TestPropsBlockAndSeatsAreDestinationsOnly(t *testing.T) {
	r := &Room{
		Rows: []string{".....", ".....", ".....", "....."},
		Props: []Prop{
			{ID: "f", Kind: PropFountain, At: Point{2, 0}, W: 1, H: 2},
			{ID: "b", Kind: PropBench, At: Point{1, 2}, Dir: Point{0, -1}},
		},
	}
	// Walking along the top row must go around the fountain and must not
	// sit down on the bench on the way.
	path := FindPath(r, Point{0, 0}, Point{4, 0})
	if path == nil {
		t.Fatal("want a path around the fountain")
	}
	for _, p := range path {
		if r.PropAt(p) != nil {
			t.Fatalf("path crossed prop at %v: %v", p, path)
		}
	}
	if FindPath(r, Point{0, 0}, Point{2, 0}) != nil {
		t.Fatal("the fountain itself is not a destination")
	}
	if p := FindPath(r, Point{0, 2}, Point{1, 2}); len(p) != 1 {
		t.Fatalf("a bench is a valid destination, got %v", p)
	}
}

func TestPlazaLayout(t *testing.T) {
	p := DefaultRooms()[0]
	if p.ID != "plaza" || p.W() != 32 || p.H() != 32 {
		t.Fatalf("unexpected plaza: %s %dx%d", p.ID, p.W(), p.H())
	}
	// Every gate, the spawn and every bench can reach the fountain's side.
	for _, from := range []Point{p.Spawn, {1, 10}, {10, 1}} {
		if FindPath(p, from, Point{13, 15}) == nil {
			t.Errorf("can't walk from %v to the fountain", from)
		}
	}
	for _, d := range p.Doors {
		if FindPath(p, p.Spawn, d.At) == nil {
			t.Errorf("can't reach door to %s", d.Target)
		}
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

func TestFindPathDiagonal(t *testing.T) {
	r := &Room{Rows: []string{".....", ".....", ".....", ".....", "....."}}
	// Open floor: straight across the diagonal, not a zig-zag.
	if p := FindPath(r, Point{0, 0}, Point{4, 4}); len(p) != 4 {
		t.Fatalf("want 4 diagonal steps, got %v", p)
	}
	// No cutting a corner past a blocked tile.
	r = &Room{Rows: []string{"..", " ."}}
	p := FindPath(r, Point{0, 0}, Point{1, 1})
	if len(p) != 2 || p[0] != (Point{1, 0}) {
		t.Fatalf("want to go round the corner, got %v", p)
	}
}

func TestBlockTemplates(t *testing.T) {
	p := plaza()
	lookup := func(id string) *Room {
		if id == "plaza" {
			return p
		}
		return nil
	}
	for _, tp := range Templates {
		r := BlockRoom("knitting", "🧶 Knitting", tp.ID)
		if err := ValidateRoom(r, lookup); err != nil {
			t.Errorf("template %s: %v", tp.ID, err)
		}
		if r.PropByID("board") == nil {
			t.Errorf("template %s has no board", tp.ID)
		}
		if FindPath(r, r.Spawn, r.Doors[0].At) == nil {
			t.Errorf("template %s: can't walk from the spawn to the door", tp.ID)
		}
	}
}
