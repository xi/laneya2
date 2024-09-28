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
	player.Game.broadcast([]Message{
		Message{
			"action":      "setHealth",
			"health":      player.Health,
			"healthTotal": player.HealthTotal,
		},
	})
}
