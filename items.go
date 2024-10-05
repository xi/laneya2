package main

import "math/rand"

const (
	CONSUMABLE uint = 1
	WEAPON          = 2
	ARMOR           = 3
)

type Item struct {
	Type        uint
	Value       uint
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
		Value:  10,
		Health: 10,
	},
	"Potion": Item{
		Type:   CONSUMABLE,
		Value:  50,
		Health: 25,
	},
	"Great Potion": Item{
		Type:   CONSUMABLE,
		Value:  400,
		Health: 100,
	},
	"Small Life Elixir": Item{
		Type:        CONSUMABLE,
		Value:       50,
		HealthTotal: 1,
	},
	"Life Elixir": Item{
		Type:        CONSUMABLE,
		Value:       250,
		HealthTotal: 5,
	},
	"Great Life Elixir": Item{
		Type:        CONSUMABLE,
		Value:       1000,
		HealthTotal: 20,
	},

	// weapons
	"Butterknive": {
		Type:   WEAPON,
		Value:  30,
		Attack: 1,
	},
	"Sword": {
		Type:   WEAPON,
		Value:  100,
		Attack: 3,
	},
	"Battleaxe": Item{
		Type:   WEAPON,
		Value:  500,
		Attack: 4,
		Speed:  -5,
	},
	"Daggers": Item{
		Type:   WEAPON,
		Value:  300,
		Attack: 2,
		Speed:  5,
	},
	"Sting": Item{
		Type:        WEAPON,
		Value:       400,
		Attack:      2,
		LineOfSight: 2,
	},
	"Shield": Item{
		Type:    WEAPON,
		Value:   300,
		Defense: 3,
	},

	// armor
	"Leather Armor": Item{
		Type:    ARMOR,
		Value:   100,
		Defense: 2,
		Speed:   -5,
	},
	"Shining Armor": Item{
		Type:        ARMOR,
		Value:       500,
		Defense:     2,
		LineOfSight: 3,
		Speed:       -5,
	},
	"Heavy Armor": Item{
		Type:    ARMOR,
		Value:   400,
		Defense: 3,
		Speed:   -10,
	},
	"Spiked Armor": Item{
		Type:    ARMOR,
		Value:   1000,
		Attack:  1,
		Defense: 2,
		Speed:   -10,
	},
	"Cloak": Item{
		Type:        ARMOR,
		Value:       400,
		Defense:     1,
		LineOfSight: 1,
		Speed:       5,
	},
	"Body Oil": Item{
		Type:   ARMOR,
		Value:  750,
		Attack: 1,
		Speed:  10,
	},
}

func RandomItem() string {
	total := 0.0
	for _, item := range Items {
		total += 1 / float64(item.Value)
	}

	x := rand.Float64()
	for name, item := range Items {
		p := 1 / float64(item.Value) / total
		if x < p {
			return name
		} else {
			x -= p
		}
	}
	return ""
}
