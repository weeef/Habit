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
var avenueSpots = []struct {
	ID, Name string
	Gate     Point // on the Plaza's back edge
}{
	{"ave-1", "1st Avenue", Point{0, 20}},
	{"ave-2", "2nd Avenue", Point{20, 0}},
	{"ave-3", "3rd Avenue", Point{0, 26}},
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
		one := func(id, kind string, x, y int) Prop { return Prop{ID: id, Kind: kind, At: Point{x, y}} }
		props := []Prop{
			{ID: "subway", Kind: PropSubway, At: Point{avenueW - 4, 5}, W: 2, H: 2},
			{ID: "bench-a", Kind: PropBench, At: Point{8, 6}, W: 2, H: 1, Dir: Point{0, -1}},
			{ID: "bench-b", Kind: PropBench, At: Point{17, 6}, W: 2, H: 1, Dir: Point{0, -1}},
			one("news", PropNewsbox, 1, 6),
			one("hydrant", PropHydrant, 13, 2),
		}
		for k, x := range []int{5, 11, 21} {
			props = append(props, one(fmt.Sprintf("tree-%d", k), PropTree, x, 4))
		}
		for k, x := range []int{2, 14, 26} {
			props = append(props, one(fmt.Sprintf("lamp-%d", k), PropLamp, x, 3))
		}
		out = append(out, &Room{
			ID: a.ID, Name: "🏙️ " + a.Name, Kind: KindStreet, Theme: "avenue",
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
