package world

import (
	"math"
	"strings"
)

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

// plaza is the park everyone arrives in, big enough for a crowd of 500 and
// laid out like a little Central Park: lawns crossed by winding paths, a
// tree-lined Mall leading up to a Bethesda-style terrace round the wishing
// fountain, and a pond by the way in with a Bow Bridge over it. A sidewalk
// with food carts, hydrants, newspaper boxes, subway entrances, steaming
// manholes and a horse carriage rings it. The client lines the back edges
// with skyscrapers (the Gaming and Gym doors are an arcade and a gym in
// that row) and runs a street with taxis along the front.
//
// Floor: ',' sidewalk, ':' crossings, '.' paths and the terrace, ';' grass,
// '=' the bridge's deck, and '~' the pond (water: not walkable).
func plaza() *Room {
	const n = 32
	type pt struct{ x, y float64 }
	// near reports whether (x, y) is within w of the path through pts.
	near := func(x, y, w float64, pts ...pt) bool {
		for i := 1; i < len(pts); i++ {
			a, b := pts[i-1], pts[i]
			dx, dy := b.x-a.x, b.y-a.y
			t := max(0, min(1, ((x-a.x)*dx+(y-a.y)*dy)/(dx*dx+dy*dy)))
			if math.Hypot(x-a.x-t*dx, y-a.y-t*dy) <= w {
				return true
			}
		}
		return false
	}
	path := func(x, y float64) bool {
		return math.Hypot(x-15.5, y-15.5) <= 5.3 || // the terrace round the fountain
			near(x, y, 1.6, pt{15.5, 29.5}, pt{15.5, 20}) || // the Mall, up from the street
			near(x, y, 0.9, pt{2, 10}, pt{7, 11}, pt{10.8, 12.8}) || // to the Gaming gate
			near(x, y, 0.9, pt{10, 2}, pt{10.5, 6}, pt{12.8, 10.6}) || // to the Gym gate
			near(x, y, 0.9, pt{20, 2}, pt{20.5, 6}, pt{18.2, 10.6}) || // to 2nd Avenue
			near(x, y, 0.9, pt{2, 20}, pt{6, 19.5}, pt{10.8, 18.2}) || // to 1st Avenue
			near(x, y, 0.9, pt{2, 26}, pt{3.5, 24}, pt{4.5, 24}) || // from 3rd Avenue onto the bridge...
			near(x, y, 0.6, pt{10.5, 24}, pt{13.6, 24}) || // ...and off it to the Mall
			near(x, y, 0.9, pt{19.8, 19}, pt{23, 24}, pt{27, 27.5}) || // down to the subway
			near(x, y, 0.9, pt{19.8, 12}, pt{22, 10}, pt{27, 4.5}) // up to the clock
	}
	// The pond: an oval beside the way in from the street, with the bridge
	// across row 24.
	pond := func(x, y float64) bool { return math.Pow((x-7.5)/3.4, 2)+math.Pow((y-24)/2.4, 2) <= 1 }
	rows := make([]string, n)
	for y := range n {
		var b strings.Builder
		for x := range n {
			fx, fy := float64(x), float64(y)
			switch {
			case y == 10 && x <= 2, x == 10 && y <= 2, (x == 15 || x == 16) && y >= 29:
				b.WriteByte(':') // crossings to the Gaming gate, the Gym gate and the street
			case x <= 1 || y <= 1 || x >= 30 || y >= 30:
				b.WriteByte(',') // sidewalk
			case pond(fx, fy) && y == 24:
				b.WriteByte('=') // the bridge
			case pond(fx, fy):
				b.WriteByte('~') // water
			case path(fx, fy):
				b.WriteByte('.') // paths and the terrace
			default:
				b.WriteByte(';') // grass
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
	willow := func(id string, x, y int) Prop {
		return Prop{ID: id, Kind: PropTree, Variant: "willow", At: Point{x, y}}
	}
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
			// benches facing it, and the board. The lawns' lamps light it.
			{ID: "fountain", Kind: PropFountain, At: Point{14, 14}, W: 4, H: 4},
			bench("bench-n", 15, 11, 2, 1, 0, 1),
			bench("bench-s", 15, 20, 2, 1, 0, -1),
			bench("bench-w", 11, 15, 1, 2, 1, 0),
			bench("bench-e", 20, 15, 1, 2, -1, 0),
			one("board", PropBoard, 12, 21), // a kiosk off to the fountain's side, so it doesn't hide it
			// Tree-lined walks round the square, with more benches.
			one("tree-nw", PropTree, 6, 6), one("tree-n", PropTree, 16, 6), one("tree-ne", PropTree, 25, 6),
			one("tree-w", PropTree, 6, 16), one("tree-e", PropTree, 25, 16),
			// Weeping willows round the pond.
			willow("willow-w", 4, 22), willow("willow-n", 10, 21), willow("willow-s", 11, 26),
			one("tree-sw", PropTree, 5, 28), one("tree-s", PropTree, 16, 25), one("tree-se", PropTree, 25, 25),
			bench("bench-nw", 8, 6, 2, 1, 0, 1), bench("bench-ne", 22, 6, 2, 1, 0, 1),
			bench("bench-sw", 6, 27, 2, 1, 0, -1), bench("bench-se", 22, 25, 2, 1, 0, -1), // bench-sw looks out over the pond
			bench("bench-wn", 6, 9, 1, 2, 1, 0), bench("bench-en", 25, 21, 1, 2, -1, 0),
			one("lamp-nw", PropLamp, 9, 9), one("lamp-ne", PropLamp, 22, 9),
			one("lamp-sw", PropLamp, 12, 23), one("lamp-se", PropLamp, 22, 22),
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
			one("manhole-a", PropManhole, 8, 30), one("manhole-b", PropManhole, 30, 5),
			one("manhole-c", PropManhole, 24, 31), one("manhole-d", PropManhole, 1, 17),
			// A horse carriage waiting for a ride round the park.
			{ID: "carriage", Kind: PropCarriage, At: Point{19, 30}, W: 3, H: 1, Dir: Point{-1, 0}},
			one("bell", PropBell, 3, 3),
			one("clock", PropClock, 28, 3),
		},
		Capacity: 500,
	}
}
