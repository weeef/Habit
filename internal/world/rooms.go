package world

import "strings"

// DefaultRooms is the prototype world: the Plaza plus doors to two
// communities, Gaming and Gym. Doors sit on the back walls (x = 0 or y = 0),
// Habbo-style.
func DefaultRooms() []*Room {
	return []*Room{
		plaza(),
		{
			ID:    "gaming",
			Name:  "🎮 Gaming",
			Kind:  KindCommunity,
			Theme: "gaming",
			Rows: []string{
				"........",
				"........",
				"........",
				"........",
				"........",
				"........",
				"........",
				"........",
			},
			Spawn: Point{3, 4},
			Doors: []Door{
				{At: Point{0, 4}, Target: "plaza", TargetAt: Point{1, 10}, Label: "The Plaza"},
			},
			Props:    []Prop{{ID: "board", Kind: PropBoard, At: Point{4, 0}}},
			Capacity: 50,
		},
		{
			ID:    "gym",
			Name:  "🏋️ Gym",
			Kind:  KindCommunity,
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
			Doors: []Door{
				{At: Point{3, 0}, Target: "plaza", TargetAt: Point{10, 1}, Label: "The Plaza"},
			},
			Props:    []Prop{{ID: "board", Kind: PropBoard, At: Point{6, 0}}},
			Capacity: 50,
		},
	}
}

// plaza is the city square everyone arrives in, NYC style, big enough for
// a crowd of 500: a paved square round a wishing fountain, ringed by a
// sidewalk with food carts, hydrants, newspaper boxes, subway entrances and
// steaming manholes. The client lines the back edges with skyscrapers (the
// Gaming and Gym doors are an arcade and a gym in that row) and runs a
// street with taxis along the front.
func plaza() *Room {
	const n = 32
	rows := make([]string, n)
	for y := range n {
		var b strings.Builder
		for x := range n {
			switch {
			case y == 10 && x <= 2, x == 10 && y <= 2, (x == 15 || x == 16) && y >= 29:
				b.WriteByte(':') // crossings to the Gaming gate, the Gym gate and the street
			case x <= 1 || y <= 1 || x >= 30 || y >= 30:
				b.WriteByte(',') // sidewalk
			default:
				b.WriteByte('.') // the square's paving
			}
		}
		rows[y] = b.String()
	}
	bench := func(id string, x, y, w, h, dx, dy int) Prop {
		return Prop{ID: id, Kind: PropBench, At: Point{x, y}, W: w, H: h, Dir: Point{dx, dy}}
	}
	cart := func(variant string, x, y, w, h, dx, dy int) Prop {
		return Prop{ID: "cart-" + variant, Kind: PropCart, Variant: variant, At: Point{x, y}, W: w, H: h, Dir: Point{dx, dy}}
	}
	one := func(id, kind string, x, y int) Prop { return Prop{ID: id, Kind: kind, At: Point{x, y}} }
	return &Room{
		ID:    "plaza",
		Name:  "The Plaza",
		Kind:  KindPlaza,
		Theme: "plaza",
		Rows:  rows,
		Spawn: Point{15, 30},
		Doors: []Door{
			{At: Point{0, 10}, Target: "gaming", TargetAt: Point{3, 4}, Label: "🎮 Gaming"},
			{At: Point{10, 0}, Target: "gym", TargetAt: Point{4, 3}, Label: "🏋️ Gym"},
		},
		Props: []Prop{
			// The middle: the fountain, benches facing it, lamps, and the board.
			{ID: "fountain", Kind: PropFountain, At: Point{15, 15}, W: 2, H: 2},
			bench("bench-n", 15, 12, 2, 1, 0, 1),
			bench("bench-s", 15, 19, 2, 1, 0, -1),
			bench("bench-w", 12, 15, 1, 2, 1, 0),
			bench("bench-e", 19, 15, 1, 2, -1, 0),
			one("lamp-c1", PropLamp, 12, 12), one("lamp-c2", PropLamp, 19, 12),
			one("lamp-c3", PropLamp, 12, 19), one("lamp-c4", PropLamp, 19, 19),
			one("board", PropBoard, 18, 18), // a kiosk by the fountain
			// Tree-lined walks round the square, with more benches.
			one("tree-nw", PropTree, 6, 6), one("tree-n", PropTree, 16, 6), one("tree-ne", PropTree, 25, 6),
			one("tree-w", PropTree, 6, 16), one("tree-e", PropTree, 25, 16),
			one("tree-sw", PropTree, 6, 25), one("tree-s", PropTree, 16, 25), one("tree-se", PropTree, 25, 25),
			bench("bench-nw", 8, 6, 2, 1, 0, 1), bench("bench-ne", 22, 6, 2, 1, 0, 1),
			bench("bench-sw", 8, 25, 2, 1, 0, -1), bench("bench-se", 22, 25, 2, 1, 0, -1),
			bench("bench-wn", 6, 9, 1, 2, 1, 0), bench("bench-en", 25, 21, 1, 2, -1, 0),
			one("lamp-nw", PropLamp, 9, 9), one("lamp-ne", PropLamp, 22, 9),
			one("lamp-sw", PropLamp, 9, 22), one("lamp-se", PropLamp, 22, 22),
			one("lamp-w", PropLamp, 2, 20), one("lamp-n", PropLamp, 20, 2),
			// The sidewalk: food carts, subways, hydrants, newspapers, manholes.
			cart("hotdog", 6, 29, 2, 1, 0, -1),
			cart("halal", 22, 29, 2, 1, 0, -1),
			cart("pretzel", 29, 8, 1, 2, -1, 0),
			cart("coffee", 29, 22, 1, 2, -1, 0),
			{ID: "subway", Kind: PropSubway, At: Point{28, 28}, W: 2, H: 2},
			{ID: "subway-w", Kind: PropSubway, At: Point{2, 27}, W: 2, H: 2},
			one("hydrant-w", PropHydrant, 1, 22), one("hydrant-n", PropHydrant, 22, 1), one("hydrant-e", PropHydrant, 30, 12),
			one("news-w", PropNewsbox, 1, 14), one("news-n", PropNewsbox, 14, 1), one("news-e", PropNewsbox, 30, 18),
			one("manhole-a", PropManhole, 11, 22), one("manhole-b", PropManhole, 20, 9),
			one("manhole-c", PropManhole, 24, 17), one("manhole-d", PropManhole, 8, 13),
			one("bell", PropBell, 3, 3),
			one("clock", PropClock, 28, 3),
		},
		Capacity: 500,
	}
}
