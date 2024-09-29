package main

import "github.com/gorilla/websocket"

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
	Attack      uint
	Defense     uint
	LineOfSight uint
	Speed       uint
	Inventory   map[string]uint
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

func (player *Player) TakeDamage(amount uint) {
	if amount > player.Health {
		player.quit <- true
	} else {
		player.Health -= amount
		player.Enqueue(Message{
			"action": "setStat",
			"stat":   "health",
			"value":  player.Health,
		})
	}
}

func (player *Player) Heal(amount uint) {
	player.Health += amount
	if player.Health > player.HealthTotal {
		player.Health = player.HealthTotal
	}
	player.Enqueue(Message{
		"action": "setStat",
		"stat":   "health",
		"value":  player.Health,
	})
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

func (player *Player) RemoveItem(item string, amount uint) bool {
	value, ok := player.Inventory[item]
	success := false
	if !ok {
		value = 0
	} else if value <= amount {
		delete(player.Inventory, item)
		value = 0
	} else {
		value -= amount
		player.Inventory[item] = value
		success = true
	}

	player.Enqueue(Message{
		"action": "setInventory",
		"item":   item,
		"amount": value,
	})

	return success
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
	if player.RemoveItem(item, 1) {
		player.Game.addToPile(player.Pos, item, 1)
	}
}

func (player *Player) UseItem(item string) {
	if item == "potion" {
		if player.Health < player.HealthTotal {
			if player.RemoveItem(item, 1) {
				player.Heal(10)
			}
		}
	}
}
