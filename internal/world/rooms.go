package world

// DefaultRooms is the prototype world: the Plaza plus two chatrooms behind
// doors. Doors sit on the back walls (x = 0 or y = 0), Habbo-style.
func DefaultRooms() []*Room {
	return []*Room{
		{
			ID:   "plaza",
			Name: "The Plaza",
			Kind: KindPlaza,
			Rows: []string{
				"..........",
				"..........",
				"..........",
				"..........",
				"..........",
				"..........",
				"..........",
				"..........",
				"..........",
				"..........",
			},
			Spawn: Point{5, 8},
			Doors: []Door{
				{At: Point{0, 3}, Target: "general", TargetAt: Point{1, 4}, Label: "#general"},
				{At: Point{6, 0}, Target: "lounge", TargetAt: Point{3, 1}, Label: "#lounge"},
			},
			Capacity: 50,
		},
		{
			ID:   "general",
			Name: "#general",
			Kind: KindChat,
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
			Spawn: Point{1, 4},
			Doors: []Door{
				{At: Point{0, 4}, Target: "plaza", TargetAt: Point{1, 3}, Label: "The Plaza"},
			},
			Capacity: 50,
		},
		{
			ID:   "lounge",
			Name: "#lounge",
			Kind: KindChat,
			Rows: []string{
				"  .....  ",
				" ....... ",
				".........",
				".........",
				".........",
				" ....... ",
				"  .....  ",
			},
			Spawn: Point{3, 1},
			Doors: []Door{
				{At: Point{3, 0}, Target: "plaza", TargetAt: Point{6, 1}, Label: "The Plaza"},
			},
			Capacity: 50,
		},
	}
}
