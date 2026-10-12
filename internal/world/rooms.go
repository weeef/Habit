package world

import "strings"

// DefaultRooms is the prototype world: the Plaza plus doors to two
// communities, Gaming and Gym. Doors sit on the back walls (x = 0 or y = 0),
// Habbo-style.
func DefaultRooms() []*Room {
	return append([]*Room{
		plaza(),
		BlockRoom("gaming", "🎮 Gaming", "arcade"),
		BlockRoom("gym", "🏋️ Gym", "gym"),
	}, Avenues()...)
}

// plaza is the city square everyone arrives in, NYC style, big enough for
// a crowd of 500: a paved square round a big wishing fountain, with a lawn
// in each corner of the square, ringed by a sidewalk with food carts,
// hydrants, newspaper boxes, subway entrances and steaming manholes. The
// client lines the back edges with skyscrapers (the Gaming and Gym doors
// are an arcade and a gym in that row) and runs a street with taxis along
// the front.
func plaza() *Room {
	const n = 32
	// lawn reports whether i is inside one of the lawns' spans (4–11 or
	// 20–27), and edge whether it's on a lawn's border, so corners can be
	// rounded off.
	lawn := func(i int) bool { return (i >= 4 && i <= 11) || (i >= 20 && i <= 27) }
	edge := func(i int) bool { return i == 4 || i == 11 || i == 20 || i == 27 }
	rows := make([]string, n)
	for y := range n {
		var b strings.Builder
		for x := range n {
			switch {
			case y == 10 && x <= 2, x == 10 && y <= 2, (x == 15 || x == 16) && y >= 29:
				b.WriteByte(':') // crossings to the Gaming gate, the Gym gate and the street
			case x <= 1 || y <= 1 || x >= 30 || y >= 30:
				b.WriteByte(',') // sidewalk
			case lawn(x) && lawn(y) && !(edge(x) && edge(y)):
				b.WriteByte(';') // grass
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
	oak := func(id string, x, y int) Prop { return Prop{ID: id, Kind: PropTree, Variant: "leafy", At: Point{x, y}} }
	bush := func(id string, x, y int, flowers bool) Prop {
		p := one(id, PropBush, x, y)
		if flowers {
			p.Variant = "flowers"
		}
		return p
	}
	return &Room{
		ID:    "plaza",
		Name:  "The Plaza",
		Kind:  KindPlaza,
		Theme: "plaza",
		Rows:  rows,
		Spawn: Point{15, 30},
		Doors: append([]Door{
			{At: Point{0, 10}, Target: "gaming", TargetAt: Point{3, 4}, Label: "🎮 Gaming"},
			{At: Point{10, 0}, Target: "gym", TargetAt: Point{4, 3}, Label: "🏋️ Gym"},
		}, avenueGates()...),
		Props: []Prop{
			// The middle: the fountain (three tiers, taller than anyone),
			// benches facing it, lamps, and the board.
			{ID: "fountain", Kind: PropFountain, At: Point{14, 14}, W: 4, H: 4},
			bench("bench-n", 15, 11, 2, 1, 0, 1),
			bench("bench-s", 15, 20, 2, 1, 0, -1),
			bench("bench-w", 11, 15, 1, 2, 1, 0),
			bench("bench-e", 20, 15, 1, 2, -1, 0),
			one("lamp-c1", PropLamp, 12, 12), one("lamp-c2", PropLamp, 19, 12),
			one("lamp-c3", PropLamp, 12, 19), one("lamp-c4", PropLamp, 19, 19),
			one("board", PropBoard, 12, 21), // a kiosk off to the fountain's side, so it doesn't hide it
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
			// Greenery on the lawns: a leafy tree and a couple of shrubs each,
			// and flowering shrubs either side of the way in from the street.
			oak("oak-nw", 10, 4), bush("bush-nw1", 11, 8, false), bush("bush-nw2", 8, 11, true),
			oak("oak-ne", 21, 4), bush("bush-ne1", 20, 8, true), bush("bush-ne2", 23, 11, false),
			oak("oak-sw", 10, 27), bush("bush-sw1", 11, 23, true), bush("bush-sw2", 8, 20, false),
			oak("oak-se", 21, 27), bush("bush-se1", 20, 23, false), bush("bush-se2", 23, 20, true),
			bush("bush-gate-w", 13, 28, true), bush("bush-gate-e", 18, 28, true),
			// The sidewalk: food carts, subways, hydrants, newspapers, manholes.
			cart("hotdog", 6, 29, 2, 1, 0, -1),
			cart("halal", 22, 29, 2, 1, 0, -1),
			cart("pretzel", 29, 8, 1, 2, -1, 0),
			cart("coffee", 29, 22, 1, 2, -1, 0),
			{ID: "subway", Kind: PropSubway, At: Point{28, 28}, W: 2, H: 2},
			{ID: "subway-w", Kind: PropSubway, At: Point{2, 27}, W: 2, H: 2},
			one("hydrant-w", PropHydrant, 1, 22), one("hydrant-n", PropHydrant, 22, 1), one("hydrant-e", PropHydrant, 30, 12),
			one("news-w", PropNewsbox, 1, 14), one("news-n", PropNewsbox, 14, 1), one("news-e", PropNewsbox, 30, 18),
			one("manhole-a", PropManhole, 13, 23), one("manhole-b", PropManhole, 18, 9),
			one("manhole-c", PropManhole, 24, 17), one("manhole-d", PropManhole, 8, 13),
			one("bell", PropBell, 3, 3),
			one("clock", PropClock, 28, 3),
		},
		Capacity: 500,
	}
}
