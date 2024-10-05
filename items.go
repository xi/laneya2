package main

const (
	CONSUMABLE uint = 1
	WEAPON          = 2
	ARMOR           = 3
)

type Item struct {
	Type        uint
	Health      uint
	HealthTotal uint
	Attack      uint
	Defense     uint
	LineOfSight int
	Speed       int
}

var Items = map[string]Item{
	// consumables
	"Small Potion": Item{
		Type:   CONSUMABLE,
		Health: 10,
	},
	"Potion": Item{
		Type:   CONSUMABLE,
		Health: 25,
	},
	"Great Potion": Item{
		Type:   CONSUMABLE,
		Health: 100,
	},
	"Small Life Elixir": Item{
		Type:        CONSUMABLE,
		HealthTotal: 1,
	},
	"Life Elixir": Item{
		Type:        CONSUMABLE,
		HealthTotal: 5,
	},
	"Great Life Elixir": Item{
		Type:        CONSUMABLE,
		HealthTotal: 20,
	},

	// weapons
	"Butterknive": {
		Type:   WEAPON,
		Attack: 1,
	},
	"Sword": {
		Type:   WEAPON,
		Attack: 3,
	},
	"Battleaxe": Item{
		Type:   WEAPON,
		Attack: 4,
		Speed:  -5,
	},
	"Daggers": Item{
		Type:   WEAPON,
		Attack: 2,
		Speed:  5,
	},
	"Sting": Item{
		Type:        WEAPON,
		Attack:      2,
		LineOfSight: 2,
	},
	"Shield": Item{
		Type:    WEAPON,
		Defense: 3,
	},

	// armor
	"Leather Armor": Item{
		Type:    ARMOR,
		Defense: 2,
		Speed:   -5,
	},
	"Shining Armor": Item{
		Type:        ARMOR,
		Defense:     2,
		LineOfSight: 3,
		Speed:       -5,
	},
	"Heavy Armor": Item{
		Type:    ARMOR,
		Defense: 3,
		Speed:   -10,
	},
	"Spiked Armor": Item{
		Type:    ARMOR,
		Attack:  1,
		Defense: 2,
		Speed:   -10,
	},
	"Cloak": Item{
		Type:        ARMOR,
		Defense:     1,
		LineOfSight: 1,
		Speed:       5,
	},
	"Body Oil": Item{
		Type:   ARMOR,
		Attack: 1,
		Speed:  10,
	},
}
