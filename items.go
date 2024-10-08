package main

import "math/rand"

const (
	CONSUMABLE uint = 1
	WEAPON          = 2
	ARMOR           = 3
)

type Item struct {
	Type        uint    `json:"type"`
	Value       uint    `json:"value"`
	Health      uint    `json:"health"`
	HealthTotal uint    `json:"healthTotal"`
	Attack      float64 `json:"attack"`
	Defense     float64 `json:"defense"`
	LineOfSight int     `json:"lineOfSight"`
	Speed       int     `json:"speed"`
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
		Value:  50,
		Attack: 2,
	},
	"Sword": {
		Type:   WEAPON,
		Value:  150,
		Attack: 8,
	},
	"Battleaxe": Item{
		Type:   WEAPON,
		Value:  500,
		Attack: 12,
		Speed:  -5,
	},
	"Daggers": Item{
		Type:   WEAPON,
		Value:  300,
		Attack: 4,
		Speed:  5,
	},
	"Sting": Item{
		Type:        WEAPON,
		Value:       400,
		Attack:      6,
		LineOfSight: 2,
	},
	"Shield": Item{
		Type:    WEAPON,
		Value:   300,
		Defense: 10,
	},
	"Masamune": Item{
		Value:  1000,
		Attack: 12,
		Speed:  5,
	},
	"Bastard Sword": Item{
		Type:   WEAPON,
		Value:  1200,
		Attack: 15,
	},
	"Excalibur": Item{
		Type:        WEAPON,
		Value:       1500,
		Attack:      25,
		LineOfSight: 1,
	},

	// armor
	"Leather Armor": Item{
		Type:    ARMOR,
		Value:   100,
		Defense: 4,
		Speed:   -2,
	},
	"Heavy Armor": Item{
		Type:        ARMOR,
		Value:       300,
		Defense:     10,
		Speed:       -10,
		LineOfSight: -1,
	},
	"Cloak": Item{
		Type:        ARMOR,
		Value:       250,
		Defense:     2,
		LineOfSight: 1,
		Speed:       2,
	},
	"Shining Armor": Item{
		Type:        ARMOR,
		Value:       600,
		Defense:     6,
		LineOfSight: 3,
		Speed:       -5,
	},
	"Body Oil": Item{
		Type:  ARMOR,
		Value: 650,
		Speed: 10,
	},
	"Wizard's Robe": Item{
		Type:        ARMOR,
		Value:       700,
		Defense:     4,
		LineOfSight: 1,
		Speed:       10,
	},
	"Forged Armor": Item{
		Type:        ARMOR,
		Value:       1000,
		Defense:     15,
		Speed:       -10,
		LineOfSight: -1,
	},
	"Spiked Armor": Item{
		Type:    ARMOR,
		Value:   1200,
		Attack:  4,
		Defense: 6,
		Speed:   -10,
	},
	"Obsidian Armor": Item{
		Type:    ARMOR,
		Value:   1500,
		Defense: 15,
	},
	"Dragon Scale Armor": Item{
		Type:        ARMOR,
		Value:       2000,
		Defense:     20,
		Speed:       -2,
		HealthTotal: 10,
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
