package main

import (
	"math"

	"github.com/gorilla/websocket"
)

type Player struct {
	Game        *Game
	quit        chan bool
	send        chan []Message
	queue       []Message
	conn        *websocket.Conn
	alive       bool
	Id          int
	Pos         Point
	Health      uint
	HealthTotal uint
	Attack      float64
	Defense     float64
	LineOfSight uint
	Speed       int
	Inventory   map[string]uint
	Weapon      string
	Armor       string
}

type PlayerMessage struct {
	Player *Player
	Msg    Message
}

func (player *Player) Enqueue(msg Message) {
	player.queue = append(player.queue, msg)
}

func (player *Player) Flush() {
	if len(player.queue) > 0 {
		player.send <- player.queue
		player.queue = []Message{}
	}
}

func (player *Player) TakeDamage(attack float64) {
	amount := uint(math.Round(attack * attack / (attack + player.Defense)))
	if amount > player.Health {
		player.quit <- true
	} else {
		player.Health -= amount
		player.AnnounceStats()
	}
}

func (player *Player) AnnounceStats() {
	player.Enqueue(Message{
		"action":      "setStats",
		"health":      player.Health,
		"healthTotal": player.HealthTotal,
		"attack":      player.Attack,
		"defense":     player.Defense,
		"lineOfSight": player.LineOfSight,
		"speed":       player.Speed,
	})
}

func (player *Player) ApplyItem(item Item) {
	player.Health += item.Health + item.HealthTotal
	player.HealthTotal += item.HealthTotal
	player.Attack += item.Attack
	player.Defense += item.Defense
	player.LineOfSight = uint(int(player.LineOfSight) + item.LineOfSight)
	player.Speed += item.Speed
}

func (player *Player) UnapplyItem(item Item) {
	player.Health -= item.Health
	player.HealthTotal -= item.HealthTotal
	player.Attack -= item.Attack
	player.Defense -= item.Defense
	player.LineOfSight = uint(int(player.LineOfSight) - item.LineOfSight)
	player.Speed -= item.Speed
}

func (player *Player) AddItem(item string, amount uint) {
	value, ok := player.Inventory[item]
	if ok {
		value += amount
	} else {
		value = amount
	}
	player.Inventory[item] = value

	player.Enqueue(Message{
		"action": "setInventory",
		"item":   item,
		"amount": value,
	})
}

func (player *Player) RemoveItem(item string) {
	value, ok := player.Inventory[item]
	if !ok {
		value = 0
	} else if value > 1 {
		value -= 1
		player.Inventory[item] = value
	} else {
		value = 0
		delete(player.Inventory, item)
	}

	player.Enqueue(Message{
		"action": "setInventory",
		"item":   item,
		"amount": value,
	})
}

func (player *Player) Move(dir string) {
	game := player.Game
	pos := player.Pos.Move(dir)
	monster := game.getMonsterAt(pos)
	if monster != nil {
		monster.TakeDamage(player.Attack)
	} else if game.IsFree(pos) {
		player.Pos = pos
		game.Enqueue(Message{
			"action": "setPosition",
			"id":     player.Id,
			"pos":    player.Pos,
		})

		game.MaybeNextLevel()
	}
}

func (player *Player) PickupItems() {
	game := player.Game
	pile, ok := game.Piles[player.Pos]
	if ok {
		delete(game.Piles, player.Pos)
		for item, amount := range pile.Items {
			player.AddItem(item, amount)
		}
		game.Enqueue(Message{
			"action": "remove",
			"id":     pile.Id,
		})
	}
}

func (player *Player) DropItem(item string) {
	if _, ok := player.Inventory[item]; ok {
		player.RemoveItem(item)
		player.Game.addToPile(player.Pos, item, 1)
	}
}

func (player *Player) UseItem(name string) {
	if _, ok := player.Inventory[name]; !ok {
		return
	}

	item, ok := Items[name]
	if !ok {
		return
	}

	if item.Health != 0 &&
		item.HealthTotal == 0 &&
		item.Attack == 0 &&
		item.Defense == 0 &&
		item.LineOfSight == 0 &&
		item.Speed == 0 &&
		player.Health == player.HealthTotal {
		return
	}

	switch item.Type {
	case CONSUMABLE:
		player.RemoveItem(name)
		player.ApplyItem(item)
	case WEAPON:
		if old, ok := Items[player.Weapon]; ok {
			player.UnapplyItem(old)
		}
		if name != player.Weapon {
			player.ApplyItem(item)
			player.Weapon = name
		} else {
			player.Weapon = ""
		}
	case ARMOR:
		if old, ok := Items[player.Armor]; ok {
			player.UnapplyItem(old)
		}
		if name != player.Armor {
			player.ApplyItem(item)
			player.Armor = name
		} else {
			player.Armor = ""
		}
	}

	if player.Health > player.HealthTotal {
		player.Health = player.HealthTotal
	}

	player.AnnounceStats()
}
