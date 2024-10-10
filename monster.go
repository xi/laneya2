package main

import (
	"math"
	"math/rand"
	"time"
)

type MonsterClass struct {
	Rune          rune
	HealthBase    float64
	HealthFactor  float64
	AttackBase    float64
	AttackFactor  float64
	DefenseBase   float64
	DefenseFactor float64
	Speed         int
	Probability   float64
}

type Monster struct {
	Game    *Game
	quit    chan bool
	Id      int
	Rune    rune
	Pos     Point
	Dir     string
	Health  float64
	Attack  float64
	Defense float64
	Speed   int
}

var MonsterClasses = []MonsterClass{
	MonsterClass{
		Rune:         'm',
		HealthBase:   10,
		HealthFactor: 1,
		AttackBase:   2,
		AttackFactor: 1,
		DefenseBase:  2,
		Probability:  5,
	},
	MonsterClass{
		Rune:         'M',
		HealthBase:   20,
		HealthFactor: 2,
		AttackBase:   8,
		AttackFactor: 1,
		DefenseBase:  8,
		Speed:        -2,
		Probability:  1,
	},
	MonsterClass{
		Rune:         's',
		HealthBase:   5,
		HealthFactor: 0.5,
		AttackBase:   2,
		AttackFactor: 1,
		DefenseBase:  2,
		Speed:        10,
		Probability:  2,
	},
	MonsterClass{
		Rune:         'z',
		HealthBase:   12,
		HealthFactor: 1.2,
		AttackBase:   4,
		AttackFactor: 1,
		DefenseBase:  4,
		Speed:        -5,
		Probability:  2,
	},
}

func randomMonsterClass() *MonsterClass {
	total := 0.0
	for _, c := range MonsterClasses {
		total += c.Probability
	}

	x := rand.Float64()
	for _, c := range MonsterClasses {
		p := c.Probability / total
		if x < p {
			return &c
		} else {
			x -= p
		}
	}
	return &MonsterClasses[0]
}

func makeMonster(game *Game, pos Point) *Monster {
	f := float64(game.Level)
	c := randomMonsterClass()

	monster := &Monster{
		Game:    game,
		quit:    make(chan bool),
		Id:      game.createId(),
		Rune:    c.Rune,
		Pos:     pos,
		Dir:     "right",
		Health:  c.HealthBase + c.HealthFactor*f,
		Attack:  c.AttackBase + c.AttackFactor*f,
		Defense: c.DefenseBase + c.DefenseFactor*f,
		Speed:   c.Speed,
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
			monster.Game.MMsg <- monster
		}
	}
}

func (monster *Monster) TakeDamage(attack float64) {
	amount := attack * attack / (attack + monster.Defense)
	if amount >= monster.Health {
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

func (monster *Monster) Move() {
	game := monster.Game

	pos := monster.Pos.Move(monster.Dir)
	player := game.getPlayerAt(pos)

	if player == nil {
		bestDist := 100000
		monster.Dir = "left"
		for player := range game.Players {
			dist := monster.Pos.Dist(player.Pos)
			if dist < bestDist {
				bestDist = dist
				monster.Dir = monster.Pos.Dir(player.Pos)
			}
		}

		if bestDist > 10 {
			return
		}
		if !game.IsFree(monster.Pos.Move(monster.Dir)) {
			monster.Dir = RandomDir()
		}

		pos = monster.Pos.Move(monster.Dir)
		player = game.getPlayerAt(pos)
	}

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
