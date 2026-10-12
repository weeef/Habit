package world

import (
	"slices"
	"strings"
)

// Blocks are communities anyone can start (PLAN.md §2.6). Each one gets a
// room built from a template, with its board on the back wall and a door
// back out to the Plaza.

// Template is a starting layout for a block's room.
type Template struct {
	ID    string
	Name  string
	About string
	build func() *Room
}

// Templates lists the layouts a new block can start from, in the order
// they're offered.
var Templates = []Template{
	{ID: "lounge", Name: "Lounge", About: "A cozy room with benches to hang out on.", build: lounge},
	{ID: "park", Name: "Park", About: "A leafy park with paths and trees.", build: park},
	{ID: "arcade", Name: "Arcade", About: "A neon room, like Gaming.", build: arcade},
	{ID: "gym", Name: "Gym", About: "A bright round hall, like the Gym.", build: gym},
}

// ValidTemplate reports whether id names a template.
func ValidTemplate(id string) bool {
	return slices.ContainsFunc(Templates, func(t Template) bool { return t.ID == id })
}

// storefronts are where official blocks with a shopfront on the Plaza let
// you back out; everyone else comes up out of the west subway.
var storefronts = map[string]Point{"gaming": {1, 10}, "gym": {10, 1}}

var subwayExit = Point{4, 28}

// BlockRoom builds the room for a block, with its door out to the Plaza.
// name is shown as is (with its bullet emoji); unknown templates fall back
// to the lounge. SetExit points the door at a storefront instead.
func BlockRoom(slug, name, template string) *Room {
	build := lounge
	for _, t := range Templates {
		if t.ID == template {
			build = t.build
		}
	}
	r := build()
	r.ID, r.Name, r.Kind, r.Capacity = slug, name, KindCommunity, 50
	back, ok := storefronts[slug]
	if !ok {
		back = subwayExit
	}
	for i := range r.Doors {
		r.Doors[i].Target, r.Doors[i].TargetAt, r.Doors[i].Label = "plaza", back, "The Plaza"
	}
	return r
}

func filled(w, h int, c byte) []string {
	rows := make([]string, h)
	for y := range rows {
		rows[y] = strings.Repeat(string(c), w)
	}
	return rows
}

func arcade() *Room {
	return &Room{
		Theme: "gaming",
		Rows:  filled(8, 8, '.'),
		Spawn: Point{3, 4},
		Doors: []Door{{At: Point{0, 4}}},
		Props: []Prop{{ID: "board", Kind: PropBoard, At: Point{4, 0}}},
	}
}

func gym() *Room {
	return &Room{
		Theme: "gym",
		Rows: []string{
			"  .....  ",
			" ....... ",
			".........",
			".........",
			".........",
			" ....... ",
			"  .....  ",
		},
		Spawn: Point{4, 3},
		Doors: []Door{{At: Point{3, 0}}},
		Props: []Prop{{ID: "board", Kind: PropBoard, At: Point{6, 0}}},
	}
}

func lounge() *Room {
	return &Room{
		Theme: "lounge",
		Rows:  filled(10, 8, '.'),
		Spawn: Point{4, 4},
		Doors: []Door{{At: Point{0, 3}}},
		Props: []Prop{
			{ID: "board", Kind: PropBoard, At: Point{5, 0}},
			{ID: "bench-a", Kind: PropBench, At: Point{2, 6}, W: 2, H: 1, Dir: Point{0, -1}},
			{ID: "bench-b", Kind: PropBench, At: Point{6, 6}, W: 2, H: 1, Dir: Point{0, -1}},
			{ID: "bench-c", Kind: PropBench, At: Point{9, 3}, W: 1, H: 2, Dir: Point{-1, 0}},
			{ID: "lamp-a", Kind: PropLamp, At: Point{1, 1}},
			{ID: "lamp-b", Kind: PropLamp, At: Point{8, 1}},
			{ID: "tree-a", Kind: PropTree, At: Point{9, 7}},
		},
	}
}

func park() *Room {
	rows := make([]string, 10)
	for y := range rows {
		var b strings.Builder
		for x := range 12 {
			if y == 5 || x == 6 {
				b.WriteByte(',') // paths crossing the park
			} else {
				b.WriteByte('.') // grass
			}
		}
		rows[y] = b.String()
	}
	return &Room{
		Theme: "park",
		Rows:  rows,
		Spawn: Point{5, 5},
		Doors: []Door{{At: Point{0, 5}}},
		Props: []Prop{
			{ID: "board", Kind: PropBoard, At: Point{7, 0}},
			{ID: "tree-a", Kind: PropTree, At: Point{2, 2}},
			{ID: "tree-b", Kind: PropTree, At: Point{9, 2}},
			{ID: "tree-c", Kind: PropTree, At: Point{2, 8}},
			{ID: "tree-d", Kind: PropTree, At: Point{9, 8}},
			{ID: "bench-a", Kind: PropBench, At: Point{4, 3}, W: 1, H: 1, Dir: Point{0, 1}},
			{ID: "bench-b", Kind: PropBench, At: Point{8, 7}, W: 2, H: 1, Dir: Point{0, -1}},
			{ID: "lamp-a", Kind: PropLamp, At: Point{5, 4}},
			{ID: "lamp-b", Kind: PropLamp, At: Point{7, 6}},
		},
	}
}

// SetExit points a block room's door at (room, at): its storefront on an
// avenue, or back to the Plaza.
func SetExit(r *Room, room string, at Point, label string) {
	for i := range r.Doors {
		r.Doors[i].Target, r.Doors[i].TargetAt, r.Doors[i].Label = room, at, label
	}
}

// PlazaExit is where you come out in the Plaza from a block with no
// storefront there: the west subway, or the official blocks' own shops.
func PlazaExit(slug string) Point {
	if p, ok := storefronts[slug]; ok {
		return p
	}
	return subwayExit
}
