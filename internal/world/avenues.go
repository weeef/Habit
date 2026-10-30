package world

import (
	"fmt"
	"strings"
)

// The town beyond the Plaza: a few avenues lined with shop lots. A block
// can rent one lot for a storefront with a door into it; empty lots show
// as "for lease". Blocks are always reachable by subway too.

// KindStreet is an outdoor avenue in the town.
const KindStreet Kind = "street"

// Avenue lots sit along the back (y = 0) row, four tiles apart.
const (
	avenueW = 28
	avenueH = 8
)

var lotXs = []int{3, 7, 11, 15, 19, 23}

// avenueSpots is where each avenue's gateway sits on the Plaza's skyline.
// Each avenue has its own character: Theme picks its colours and skyline
// in the client, and its props are set out in avenueProps.
var avenueSpots = []struct {
	ID, Name string
	Gate     Point // on the Plaza's back edge
	Theme    string
}{
	{"ave-1", "1st Avenue", Point{0, 20}, "avenue-garden"}, // leafy brownstones
	{"ave-2", "2nd Avenue", Point{20, 0}, "avenue-neon"},   // the theatre district
	{"ave-3", "3rd Avenue", Point{0, 26}, "avenue"},
}

// AvenueExit is where you stand on an avenue after stepping out of a
// block's storefront at lot n.
func AvenueExit(lot int) Point { return Point{lotXs[lot-1], 1} }

// LotDoor is the door tile of lot n (1-based).
func LotDoor(lot int) Point { return Point{lotXs[lot-1], 0} }

// ValidLot reports whether street and lot name a real shop lot.
func ValidLot(street string, lot int) bool {
	if lot < 1 || lot > len(lotXs) {
		return false
	}
	for _, a := range avenueSpots {
		if a.ID == street {
			return true
		}
	}
	return false
}

// avenueGates are the Plaza's doors out to the avenues.
func avenueGates() []Door {
	var out []Door
	for _, a := range avenueSpots {
		out = append(out, Door{At: a.Gate, Target: a.ID, TargetAt: Point{1, 4}, Label: "🏙️ " + a.Name})
	}
	return out
}

// Avenues builds the town's avenues. Storefront doors are added as blocks
// rent lots (see AddLotDoor).
func Avenues() []*Room {
	var out []*Room
	for _, a := range avenueSpots {
		rows := make([]string, avenueH)
		for y := range rows {
			if y <= 1 {
				rows[y] = strings.Repeat(",", avenueW) // the sidewalk along the shops
			} else {
				rows[y] = strings.Repeat(".", avenueW)
			}
		}
		back := Point{a.Gate.X + 1, a.Gate.Y}
		if a.Gate.Y == 0 {
			back = Point{a.Gate.X, a.Gate.Y + 1}
		}
		props := avenueProps(a.Theme)
		out = append(out, &Room{
			ID: a.ID, Name: "🏙️ " + a.Name, Kind: KindStreet, Theme: a.Theme,
			Rows:     rows,
			Spawn:    Point{2, 4},
			Doors:    []Door{{At: Point{0, 4}, Target: "plaza", TargetAt: back, Label: "⛲ The Plaza"}},
			Props:    props,
			Capacity: 200,
			Lots:     lotDoors(),
		})
	}
	return out
}

// avenueProps sets out an avenue's street furniture. Every avenue keeps a
// subway entrance, so it's always a stop on the map.
func avenueProps(theme string) []Prop {
	one := func(id, kind string, x, y int) Prop { return Prop{ID: id, Kind: kind, At: Point{x, y}} }
	variant := func(id, kind, v string, x, y int) Prop { p := one(id, kind, x, y); p.Variant = v; return p }
	bench := func(id string, x, y int) Prop {
		return Prop{ID: id, Kind: PropBench, At: Point{x, y}, W: 2, H: 1, Dir: Point{0, -1}}
	}
	props := []Prop{{ID: "subway", Kind: PropSubway, At: Point{avenueW - 4, 5}, W: 2, H: 2}}
	switch theme {
	case "avenue-garden":
		// A leafy residential street: a row of cherry trees and oaks with
		// flowering planters between them, benches under the trees, and
		// old-fashioned lamps.
		for k, x := range []int{3, 7, 11, 15, 19} {
			if k%2 == 0 {
				props = append(props, one(fmt.Sprintf("tree-%d", k), PropTree, x, 3))
			} else {
				props = append(props, variant(fmt.Sprintf("tree-%d", k), PropTree, "leafy", x, 3))
			}
		}
		for k, x := range []int{5, 9, 13, 17, 21} {
			props = append(props, variant(fmt.Sprintf("planter-%d", k), PropBush, "flowers", x, 3))
		}
		props = append(props,
			bench("bench-a", 4, 6), bench("bench-b", 10, 6), bench("bench-c", 16, 6),
			one("lamp-0", PropLamp, 1, 3), one("lamp-1", PropLamp, 23, 3),
			one("news", PropNewsbox, 1, 6),
			variant("cart-coffee", PropCart, "coffee", 20, 6),
		)
	case "avenue-neon":
		// The theatre district: no trees, lamps all along, food carts for
		// the after-show crowd, newspaper boxes and hydrants.
		for k, x := range []int{2, 6, 10, 14, 18, 22} {
			props = append(props, one(fmt.Sprintf("lamp-%d", k), PropLamp, x, 3))
		}
		props = append(props,
			Prop{ID: "cart-hotdog", Kind: PropCart, Variant: "hotdog", At: Point{4, 5}, W: 2, H: 1, Dir: Point{0, -1}},
			Prop{ID: "cart-pretzel", Kind: PropCart, Variant: "pretzel", At: Point{15, 5}, W: 2, H: 1, Dir: Point{0, -1}},
			bench("bench-a", 9, 6),
			one("news-a", PropNewsbox, 1, 6), one("news-b", PropNewsbox, 12, 2),
			one("hydrant-a", PropHydrant, 8, 2), one("hydrant-b", PropHydrant, 20, 2),
		)
	default:
		props = append(props,
			bench("bench-a", 8, 6), bench("bench-b", 17, 6),
			one("news", PropNewsbox, 1, 6),
			one("hydrant", PropHydrant, 13, 2),
		)
		for k, x := range []int{5, 11, 21} {
			props = append(props, one(fmt.Sprintf("tree-%d", k), PropTree, x, 4))
		}
		for k, x := range []int{2, 14, 26} {
			props = append(props, one(fmt.Sprintf("lamp-%d", k), PropLamp, x, 3))
		}
	}
	return props
}

func lotDoors() []Point {
	out := make([]Point, len(lotXs))
	for i := range lotXs {
		out[i] = LotDoor(i + 1)
	}
	return out
}

// AddLotDoor puts a storefront door at lot n of avenue r, into the block
// slug. Any door already on that lot is replaced.
func AddLotDoor(r *Room, lot int, slug, label, color string) {
	RemoveLotDoor(r, lot)
	r.Doors = append(r.Doors, Door{At: LotDoor(lot), Target: slug, TargetAt: AtSpawn, Label: label, Color: color})
}

// AvenueName is an avenue's name without its emoji, or "" if id isn't one.
func AvenueName(id string) string {
	for _, a := range avenueSpots {
		if a.ID == id {
			return a.Name
		}
	}
	return ""
}

// RemoveLotDoor takes the storefront door off lot n, if there is one.
func RemoveLotDoor(r *Room, lot int) {
	at := LotDoor(lot)
	kept := r.Doors[:0]
	for _, d := range r.Doors {
		if d.At != at {
			kept = append(kept, d)
		}
	}
	r.Doors = kept
}
