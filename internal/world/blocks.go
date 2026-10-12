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

// arcade is a neon games room: a couch and beanbags on a rug facing a big
// screen, arcade cabinets round the edge and a vending machine.
func arcade() *Room {
	rows := make([]string, 10)
	for y := range rows {
		var b strings.Builder
		for x := range 12 {
			if x >= 2 && x <= 8 && y >= 1 && y <= 5 {
				b.WriteByte(',') // the rug in front of the screen
			} else {
				b.WriteByte('.') // carpet
			}
		}
		rows[y] = b.String()
	}
	seat := func(id, variant string, x, y, w int) Prop {
		return Prop{ID: id, Kind: PropBench, Variant: variant, At: Point{x, y}, W: w, H: 1, Dir: Point{0, -1}}
	}
	cabinet := func(id, color string, x, y, dx, dy int) Prop {
		return Prop{ID: id, Kind: PropCabinet, Variant: color, At: Point{x, y}, Dir: Point{dx, dy}}
	}
	return &Room{
		Theme: "gaming",
		Rows:  rows,
		Spawn: Point{3, 4},
		Doors: []Door{{At: Point{0, 4}}},
		Props: []Prop{
			{ID: "board", Kind: PropBoard, At: Point{0, 7}},
			{ID: "screen", Kind: PropScreen, At: Point{4, 0}, W: 3, H: 1},
			seat("couch", "couch", 4, 3, 3),
			seat("beanbag-a", "beanbag", 2, 2, 1),
			seat("beanbag-b", "beanbag", 8, 2, 1),
			cabinet("cabinet-1", "pink", 8, 0, 0, 1),
			cabinet("cabinet-2", "cyan", 9, 0, 0, 1),
			cabinet("cabinet-3", "lime", 10, 0, 0, 1),
			cabinet("cabinet-4", "purple", 11, 3, -1, 0),
			cabinet("cabinet-5", "orange", 11, 5, -1, 0),
			cabinet("cabinet-6", "pink", 11, 7, -1, 0),
			{ID: "vending", Kind: PropVending, Variant: "soda", At: Point{0, 1}, Dir: Point{1, 0}},
		},
	}
}

// gym is a bright hall: treadmills facing a mirror wall, a weights corner
// on rubber flooring, yoga mats, a punching bag and a water cooler.
func gym() *Room {
	rows := make([]string, 10)
	for y := range rows {
		var b strings.Builder
		for x := range 12 {
			switch {
			case x >= 8 && y <= 4:
				b.WriteByte(',') // rubber flooring in the weights corner
			case (x == 4 || x == 6 || x == 8) && (y == 7 || y == 8):
				b.WriteByte(':') // yoga mats
			default:
				b.WriteByte('.') // wooden floor
			}
		}
		rows[y] = b.String()
	}
	treadmill := func(id string, y int) Prop {
		return Prop{ID: id, Kind: PropTreadmill, At: Point{1, y}, W: 2, H: 1, Dir: Point{-1, 0}}
	}
	return &Room{
		Theme: "gym",
		Rows:  rows,
		Spawn: Point{4, 3},
		Doors: []Door{{At: Point{3, 0}}},
		Props: []Prop{
			{ID: "board", Kind: PropBoard, At: Point{0, 8}},
			treadmill("treadmill-1", 2), treadmill("treadmill-2", 4), treadmill("treadmill-3", 6),
			{ID: "weights", Kind: PropWeights, Variant: "dumbbells", At: Point{9, 0}, W: 2, H: 1, Dir: Point{0, 1}},
			{ID: "press", Kind: PropBench, Variant: "press", At: Point{9, 3}, W: 2, H: 1, Dir: Point{0, 1}},
			{ID: "punchbag", Kind: PropPunchbag, At: Point{10, 7}},
			{ID: "cooler", Kind: PropCooler, Variant: "water", At: Point{11, 1}, Dir: Point{-1, 0}},
			{ID: "plant-a", Kind: PropBush, At: Point{0, 9}},
			{ID: "plant-b", Kind: PropBush, Variant: "flowers", At: Point{11, 9}},
		},
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
