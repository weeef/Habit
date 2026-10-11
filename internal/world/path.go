package world

import "container/heap"

// Step costs for the pathfinder: a diagonal step is about 1.4 straight ones.
const (
	straightCost = 10
	diagonalCost = 14
)

// FindPath returns the tiles to walk from `from` to `to`, excluding `from`.
// Movement is 8-directional, but a diagonal step never cuts a corner: both
// tiles beside it must be passable too, so nobody squeezes between props.
// Props block their tiles. Door and seat tiles can only be the final step,
// so an avatar never leaves a room or sits down on the way somewhere else.
// It returns nil if `to` is unreachable or equal to `from`.
func FindPath(r *Room, from, to Point) []Point {
	if from == to || !r.Floor(to) || r.Blocked(to) {
		return nil
	}
	passable := func(p Point) bool {
		if !r.Floor(p) || r.Blocked(p) {
			return false
		}
		return p == to || (r.DoorAt(p) == nil && r.SeatAt(p) == nil)
	}

	cameFrom := map[Point]Point{}
	cost := map[Point]int{from: 0}
	open := &pq{{p: from, f: octile(from, to)}}

	for open.Len() > 0 {
		cur := heap.Pop(open).(node).p
		if cur == to {
			var path []Point
			for p := to; p != from; p = cameFrom[p] {
				path = append(path, p)
			}
			for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
				path[i], path[j] = path[j], path[i]
			}
			return path
		}
		for _, d := range [8]Point{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {1, 1}, {1, -1}, {-1, 1}, {-1, -1}} {
			next := Point{cur.X + d.X, cur.Y + d.Y}
			if !passable(next) {
				continue
			}
			step := straightCost
			if d.X != 0 && d.Y != 0 {
				if !passable(Point{cur.X + d.X, cur.Y}) || !passable(Point{cur.X, cur.Y + d.Y}) {
					continue
				}
				step = diagonalCost
			}
			c := cost[cur] + step
			if old, seen := cost[next]; seen && c >= old {
				continue
			}
			cost[next] = c
			cameFrom[next] = cur
			heap.Push(open, node{p: next, f: c + octile(next, to)})
		}
	}
	return nil
}

// Diagonal reports whether stepping from a to b is a diagonal step.
func Diagonal(a, b Point) bool { return a.X != b.X && a.Y != b.Y }

// octile is the cheapest possible cost from a to b with 8-way movement.
func octile(a, b Point) int {
	dx, dy := abs(a.X-b.X), abs(a.Y-b.Y)
	return straightCost*max(dx, dy) + (diagonalCost-straightCost)*min(dx, dy)
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

type node struct {
	p Point
	f int
}

type pq []node

func (q pq) Len() int           { return len(q) }
func (q pq) Less(i, j int) bool { return q[i].f < q[j].f }
func (q pq) Swap(i, j int)      { q[i], q[j] = q[j], q[i] }
func (q *pq) Push(x any)        { *q = append(*q, x.(node)) }
func (q *pq) Pop() any {
	old := *q
	n := old[len(old)-1]
	*q = old[:len(old)-1]
	return n
}
