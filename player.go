package main

import "github.com/gorilla/websocket"

type Player struct {
	Game        *Game
	Send        chan []Message
	conn        *websocket.Conn
	alive       bool
	Id          int
	Pos         Point
	Speed       float32
	Health      uint
	HealthTotal uint
	Inventory   map[string]uint
}

type PlayerMessage struct {
	Player *Player
	Msg    Message
}

func (player *Player) Move(dir string) {
	game := player.Game
	pos := player.Pos.Move(dir)
	monster := game.getMonsterAt(pos)
	if monster != nil {
		monster.TakeDamage(5)
	} else if game.IsFree(pos) {
		player.Pos = pos
		game.broadcast([]Message{
			Message{
				"action": "setPosition",
				"id":     player.Id,
				"pos":    player.Pos,
			},
		})

		game.MaybeNextLevel()
	}
}

func (player *Player) TakeDamage(amount uint) {
	// TODO: death if amount >= player.Health
	player.Health -= amount
	player.Send <- []Message{
		Message{
			"action":      "setHealth",
			"health":      player.Health,
			"healthTotal": player.HealthTotal,
		},
	}
}

func (player *Player) Heal(amount uint) {
	// TODO: death if amount >= player.Health
	player.Health += amount
	if player.Health > player.HealthTotal {
		player.Health = player.HealthTotal
	}
	player.Send <- []Message{
		Message{
			"action":      "setHealth",
			"health":      player.Health,
			"healthTotal": player.HealthTotal,
		},
	}
}

func (player *Player) AddItem(item string, amount uint) {
	value, ok := player.Inventory[item]
	if ok {
		value += amount
	} else {
		value = amount
	}
	player.Inventory[item] = value

	player.Send <- []Message{
		Message{
			"action": "setInventory",
			"item":   item,
			"amount": value,
		},
	}
}

func (player *Player) RemoveItem(item string, amount uint) {
	value, ok := player.Inventory[item]
	if !ok {
		value = 0
	} else if value <= amount {
		delete(player.Inventory, item)
		value = 0
	} else {
		value -= amount
		player.Inventory[item] = value
	}

	player.Send <- []Message{
		Message{
			"action": "setInventory",
			"item":   item,
			"amount": value,
		},
	}
}

func (player *Player) UseItem(item string) {
	// TODO: send result in a single transaction
	// TODO: check if item is in inventory
	if item == "potion" {
		if player.Health < player.HealthTotal {
			player.RemoveItem(item, 1)
			player.Heal(10)
		}
	}
}
