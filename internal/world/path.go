package world

import "container/heap"

// FindPath returns the tiles to walk from `from` to `to`, excluding `from`.
// Movement is 4-directional. Door tiles can only be the final step, so an
// avatar never leaves a room by walking across a door on the way somewhere
// else. It returns nil if `to` is unreachable or equal to `from`.
func FindPath(r *Room, from, to Point) []Point {
	if from == to || !r.Floor(to) {
		return nil
	}
	passable := func(p Point) bool {
		return r.Floor(p) && (p == to || r.DoorAt(p) == nil)
	}

	cameFrom := map[Point]Point{}
	cost := map[Point]int{from: 0}
	open := &pq{{p: from, f: manhattan(from, to)}}

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
		for _, d := range [4]Point{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			next := Point{cur.X + d.X, cur.Y + d.Y}
			if !passable(next) {
				continue
			}
			c := cost[cur] + 1
			if old, seen := cost[next]; seen && c >= old {
				continue
			}
			cost[next] = c
			cameFrom[next] = cur
			heap.Push(open, node{p: next, f: c + manhattan(next, to)})
		}
	}
	return nil
}

func manhattan(a, b Point) int { return abs(a.X-b.X) + abs(a.Y-b.Y) }

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
