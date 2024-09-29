package main

import "time"

type Monster struct {
	Game   *Game
	quit   chan bool
	Id     int
	Rune   rune
	Pos    Point
	Speed  float32
	Health int
}

type MonsterMessage struct {
	Monster *Monster
	Msg     Message
}

func makeMonster(game *Game, pos Point) *Monster {
	monster := &Monster{
		Game:   game,
		quit:   make(chan bool),
		Id:     game.createId(),
		Rune:   'm',
		Pos:    pos,
		Speed:  2,
		Health: 10,
	}
	go monster.run()
	return monster
}

func (monster *Monster) run() {
	timeout := time.Duration(float32(time.Second) / monster.Speed)
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

func (monster *Monster) TakeDamage(amount int) {
	monster.Health -= amount
	if monster.Health <= 0 {
		monster.quit <- true
		delete(monster.Game.Monsters, monster)
		monster.Game.addToPile(monster.Pos, "potion")
		monster.Game.Enqueue(Message{
			"action": "remove",
			"id":     monster.Id,
		})
	}
}

func (monster *Monster) Move(dir string) {
	game := monster.Game
	pos := monster.Pos.Move(dir)
	player := game.getPlayerAt(pos)
	if player != nil {
		player.TakeDamage(2)
	} else if game.getMonsterAt(pos) == nil && game.IsFree(pos) {
		monster.Pos = pos
		game.Enqueue(Message{
			"action": "setPosition",
			"id":     monster.Id,
			"pos":    monster.Pos,
		})
	}
}
