package world

// DefaultRooms is the prototype world: the Plaza plus doors to two
// communities, Gaming and Gym. Doors sit on the back walls (x = 0 or y = 0),
// Habbo-style.
func DefaultRooms() []*Room {
	return []*Room{
		{
			ID:    "plaza",
			Name:  "The Plaza",
			Kind:  KindPlaza,
			Theme: "plaza",
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
				{At: Point{0, 3}, Target: "gaming", TargetAt: Point{3, 4}, Label: "🎮 Gaming"},
				{At: Point{6, 0}, Target: "gym", TargetAt: Point{4, 3}, Label: "🏋️ Gym"},
			},
			Capacity: 50,
		},
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
				{At: Point{0, 4}, Target: "plaza", TargetAt: Point{2, 3}, Label: "The Plaza"},
			},
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
				{At: Point{3, 0}, Target: "plaza", TargetAt: Point{6, 2}, Label: "The Plaza"},
			},
			Capacity: 50,
		},
	}
}
