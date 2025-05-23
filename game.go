package main

import (
	"log"
	"sync"
)

type Message map[string]interface{}

type Pile struct {
	Id    int
	Items map[string]uint
}

type Game struct {
	Id         string
	Players    map[*Player]bool
	Monsters   map[*Monster]bool
	Piles      map[Point]*Pile
	Msg        chan PlayerMessage
	MMsg       chan *Monster
	register   chan *Player
	unregister chan *Player
	lastId     int
	Rects      []Rect
	Ladder     Point
	Level      uint
}

var verbose = false
var static = false

var mux = &sync.RWMutex{}
var games = make(map[string]*Game)

func getGame(id string) *Game {
	mux.RLock()
	game, ok := games[id]
	mux.RUnlock()

	if !ok {
		if verbose {
			log.Println("create game", id)
		}
		game = &Game{
			Id:         id,
			Players:    make(map[*Player]bool),
			Monsters:   make(map[*Monster]bool),
			Piles:      make(map[Point]*Pile),
			Msg:        make(chan PlayerMessage),
			MMsg:       make(chan *Monster),
			register:   make(chan *Player),
			unregister: make(chan *Player),
			lastId:     0,
			Level:      1,
		}
		game.generateMap()
		mux.Lock()
		games[id] = game
		mux.Unlock()

		go game.run()
	}

	return game
}

func (game *Game) Enqueue(msg Message) {
	for player, _ := range game.Players {
		player.Enqueue(msg)
	}
}

func (game *Game) Flush() {
	for player, _ := range game.Players {
		player.Flush()
	}
}

func (game *Game) createId() int {
	game.lastId += 1
	return game.lastId
}

func (game *Game) removePlayer(player *Player) {
	if _, ok := game.Players[player]; !ok {
		return
	}

	if verbose {
		log.Println("remove player", game.Id, player.Id)
	}
	delete(game.Players, player)
	close(player.send)

	game.Enqueue(Message{
		"action": "remove",
		"id":     player.Id,
	})
	for item, amount := range player.Inventory {
		game.addToPile(player.Pos, item, amount)
	}
}

func (game *Game) generateMap() {
	for monster := range game.Monsters {
		monster.quit <- true
		delete(game.Monsters, monster)
	}

	for pos := range game.Piles {
		delete(game.Piles, pos)
	}

	prev := Rect{-3, -3, 3, 3}

	game.Rects = []Rect{prev}
	lines := []Rect{}

	for i := 1; i <= 15; i++ {
		rect := randomRect(25)
		if rect.Area() < 150 && rect.Perimeter() < 80 {
			game.Rects = append(game.Rects, rect)

			p1 := prev.Center()
			p2 := rect.Center()

			lines = append(lines, makeRect(p1.X, p1.Y, p2.X, p1.Y))
			lines = append(lines, makeRect(p2.X, p1.Y, p2.X, p2.Y))

			monster := makeMonster(game, rect.RandomPoint())
			game.Monsters[monster] = true

			prev = rect
		}
	}

	game.Ladder = prev.RandomPoint()

	for _, line := range lines {
		game.Rects = append(game.Rects, line)
	}
}

func (game *Game) IsFree(p Point) bool {
	for _, rect := range game.Rects {
		if rect.Contains(p) {
			return true
		}
	}
	return false
}

func (game *Game) MaybeNextLevel() {
	for player := range game.Players {
		if player.Pos != game.Ladder {
			return
		}
	}

	game.Level += 1

	game.generateMap()
	game.Enqueue(Message{
		"action": "setLevel",
		"level":  game.Level,
		"rects":  game.Rects,
		"ladder": game.Ladder,
	})

	for monster := range game.Monsters {
		game.Enqueue(Message{
			"action": "create",
			"type":   "monster",
			"rune":   string(monster.Rune),
			"id":     monster.Id,
			"pos":    monster.Pos,
		})
	}

	for player := range game.Players {
		player.Pos = Point{0, 0}
		game.Enqueue(Message{
			"action": "setPosition",
			"id":     player.Id,
			"pos":    player.Pos,
		})
	}
}

func (game *Game) getMonsterAt(pos Point) *Monster {
	for monster := range game.Monsters {
		if monster.Pos == pos {
			return monster
		}
	}
	return nil
}

func (game *Game) getPlayerAt(pos Point) *Player {
	for player := range game.Players {
		if player.Pos == pos {
			return player
		}
	}
	return nil
}

func (game *Game) addToPile(pos Point, item string, amount uint) {
	pile, ok := game.Piles[pos]
	if !ok {
		pile = &Pile{
			Id:    game.createId(),
			Items: make(map[string]uint),
		}
		game.Piles[pos] = pile
	}

	value, ok := pile.Items[item]
	if ok {
		pile.Items[item] = value + amount
	} else {
		pile.Items[item] = amount
		game.Enqueue(Message{
			"action": "create",
			"type":   "pile",
			"id":     pile.Id,
			"rune":   "%",
			"pos":    pos,
		})
	}
}

func (game *Game) run() {
	for {
		select {
		case player := <-game.register:
			if verbose {
				log.Println("create player", game.Id, player.Id)
			}
			player.Enqueue(Message{
				"action": "setId",
				"id":     player.Id,
			})
			player.Enqueue(Message{
				"action":      "setStats",
				"health":      player.Health,
				"healthTotal": player.HealthTotal,
				"attack":      player.Attack,
				"defense":     player.Defense,
				"lineOfSight": player.LineOfSight,
				"speed":       player.Speed,
			})
			player.Enqueue(Message{
				"action": "setLevel",
				"level":  game.Level,
				"rects":  game.Rects,
				"ladder": game.Ladder,
			})
			for monster := range game.Monsters {
				player.Enqueue(Message{
					"action": "create",
					"type":   "monster",
					"rune":   string(monster.Rune),
					"id":     monster.Id,
					"pos":    monster.Pos,
				})
			}
			for pos, pile := range game.Piles {
				player.Enqueue(Message{
					"action": "create",
					"type":   "pile",
					"rune":   "%",
					"id":     pile.Id,
					"pos":    pos,
				})
			}
			for p := range game.Players {
				player.Enqueue(Message{
					"action":      "create",
					"type":        "player",
					"rune":        "@",
					"id":          p.Id,
					"pos":         p.Pos,
					"lineOfSight": p.LineOfSight,
				})
			}

			game.Players[player] = true

			game.Enqueue(Message{
				"action":      "create",
				"type":        "player",
				"rune":        "@",
				"id":          player.Id,
				"pos":         player.Pos,
				"lineOfSight": player.LineOfSight,
			})
		case player := <-game.unregister:
			game.removePlayer(player)
			if len(game.Players) == 0 {
				if verbose {
					log.Println("remove game", game.Id)
				}
				mux.Lock()
				delete(games, game.Id)
				mux.Unlock()
				return
			}
		case pmsg := <-game.Msg:
			if _, ok := game.Players[pmsg.Player]; !ok {
				continue
			}
			if pmsg.Msg["action"] == "move" {
				dir, ok := pmsg.Msg["dir"].(string)
				if ok {
					pmsg.Player.Move(dir)
				}
			} else if pmsg.Msg["action"] == "pickup" {
				pmsg.Player.PickupItems()
			} else if pmsg.Msg["action"] == "drop" {
				item, ok := pmsg.Msg["item"].(string)
				if ok {
					pmsg.Player.DropItem(item)
				}
			} else if pmsg.Msg["action"] == "use" {
				item, ok := pmsg.Msg["item"].(string)
				if ok {
					pmsg.Player.UseItem(item)
				}
			} else if verbose {
				log.Println("unknown action", pmsg.Msg)
			}
		case monster := <-game.MMsg:
			if _, ok := game.Monsters[monster]; !ok {
				continue
			}
			monster.Move()
		}
		game.Flush()
	}
}
