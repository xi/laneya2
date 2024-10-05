package main

import (
	"math"
	"time"
)

type Monster struct {
	Game    *Game
	quit    chan bool
	Id      int
	Rune    rune
	Pos     Point
	Health  float64
	Attack  float64
	Defense float64
	Speed   int
}

type MonsterMessage struct {
	Monster *Monster
	Msg     Message
}

func makeMonster(game *Game, pos Point) *Monster {
	monster := &Monster{
		Game:    game,
		quit:    make(chan bool),
		Id:      game.createId(),
		Rune:    'm',
		Pos:     pos,
		Speed:   0,
		Attack:  2 + float64(game.Level),
		Defense: 0 + float64(game.Level),
		Health:  10 + float64(game.Level),
	}
	go monster.run()
	return monster
}

func (monster *Monster) run() {
	frequency := 2 * math.Pow(1.07, float64(monster.Speed))
	timeout := time.Duration(float64(time.Second) / frequency)
	ticker := time.NewTicker(timeout)
	defer ticker.Stop()

	for {
		select {
		case <-monster.quit:
			return
		case <-ticker.C:
			bestDist := 100000
			dir := "left"
			for player := range monster.Game.Players {
				dist := monster.Pos.Dist(player.Pos)
				if dist < bestDist {
					bestDist = dist
					dir = monster.Pos.Dir(player.Pos)
				}
			}

			if bestDist > 10 {
				continue
			}
			if !monster.Game.IsFree(monster.Pos.Move(dir)) {
				dir = RandomDir()
			}

			monster.Game.MMsg <- MonsterMessage{
				monster,
				Message{
					"action": "move",
					"dir":    dir,
				},
			}
		}
	}
}

func (monster *Monster) TakeDamage(attack float64) {
	amount := attack * attack / (attack + monster.Defense)
	if amount > monster.Health {
		monster.quit <- true
		delete(monster.Game.Monsters, monster)
		monster.Game.addToPile(monster.Pos, RandomItem(), 1)
		monster.Game.Enqueue(Message{
			"action": "remove",
			"id":     monster.Id,
		})
	} else {
		monster.Health -= amount
	}
}

func (monster *Monster) Move(dir string) {
	game := monster.Game
	pos := monster.Pos.Move(dir)
	player := game.getPlayerAt(pos)
	if player != nil {
		player.TakeDamage(monster.Attack)
	} else if game.getMonsterAt(pos) == nil && game.IsFree(pos) {
		monster.Pos = pos
		game.Enqueue(Message{
			"action": "setPosition",
			"id":     monster.Id,
			"pos":    monster.Pos,
		})
	}
}
