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
	MMsg       chan MonsterMessage
	register   chan *Player
	unregister chan *Player
	lastId     int
	Rects      []Rect
	Ladder     Point
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
			MMsg:       make(chan MonsterMessage),
			register:   make(chan *Player),
			unregister: make(chan *Player),
			lastId:     0,
		}
		game.generateMap()
		mux.Lock()
		games[id] = game
		mux.Unlock()

		go game.run()
	}

	return game
}

func (game *Game) broadcast(msgs []Message) {
	for player, _ := range game.Players {
		player.Send <- msgs
	}
}

func (game *Game) createId() int {
	game.lastId += 1
	return game.lastId
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

	game.generateMap()
	msgs := []Message{
		Message{
			"action": "setLevel",
			"rects":  game.Rects,
			"ladder": game.Ladder,
		},
	}

	for monster := range game.Monsters {
		msgs = append(msgs, Message{
			"action": "create",
			"type":   "monster",
			"rune":   string(monster.Rune),
			"id":     monster.Id,
			"pos":    monster.Pos,
		})
	}

	for player := range game.Players {
		player.Pos = Point{0, 0}
		msgs = append(msgs, Message{
			"action": "setPosition",
			"id":     player.Id,
			"pos":    player.Pos,
		})
	}

	game.broadcast(msgs)
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

func (game *Game) addToPile(pos Point, item string) {
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
		pile.Items[item] = value + 1
	} else {
		pile.Items[item] = 1
		game.broadcast([]Message{
			Message{
				"action": "create",
				"type":   "pile",
				"id":     pile.Id,
				"rune":   "%",
				"pos":    pos,
			},
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
			setup := []Message{
				Message{
					"action": "setId",
					"id":     player.Id,
				},
				Message{
					"action":      "setHealth",
					"health":      player.Health,
					"healthTotal": player.HealthTotal,
				},
				Message{
					"action": "setLevel",
					"rects":  game.Rects,
					"ladder": game.Ladder,
				},
			}
			for monster := range game.Monsters {
				setup = append(setup, Message{
					"action": "create",
					"type":   "monster",
					"rune":   string(monster.Rune),
					"id":     monster.Id,
					"pos":    monster.Pos,
				})
			}
			for pos, pile := range game.Piles {
				setup = append(setup, Message{
					"action": "create",
					"type":   "pile",
					"rune":   "%",
					"id":     pile.Id,
					"pos":    pos,
				})
			}
			for p := range game.Players {
				setup = append(setup, Message{
					"action": "create",
					"type":   "player",
					"rune":   "@",
					"id":     p.Id,
					"pos":    p.Pos,
				})
			}
			player.Send <- setup

			game.Players[player] = true

			game.broadcast([]Message{
				Message{
					"action": "create",
					"type":   "player",
					"rune":   "@",
					"id":     player.Id,
					"pos":    player.Pos,
				},
			})
		case player := <-game.unregister:
			if verbose {
				log.Println("remove player", game.Id, player.Id)
			}
			delete(game.Players, player)
			if len(game.Players) == 0 {
				if verbose {
					log.Println("remove game", game.Id)
				}
				mux.Lock()
				delete(games, game.Id)
				mux.Unlock()
			} else {
				game.broadcast([]Message{
					Message{
						"action": "remove",
						"id":     player.Id,
					},
				})
			}
		case cmsg := <-game.Msg:
			player := cmsg.Player
			msg := cmsg.Msg

			if msg["action"] == "move" {
				dir, ok := msg["dir"].(string)
				if !ok {
					continue
				}
				player.Move(dir)
			} else if msg["action"] == "pickup" {
				pile, ok := game.Piles[player.Pos]
				if ok {
					delete(game.Piles, player.Pos)
					for item, amount := range pile.Items {
						player.AddItem(item, amount)
					}
					game.broadcast([]Message{
						Message{
							"action": "remove",
							"id":     pile.Id,
						},
					})
				}
			} else if msg["action"] == "drop" {
				item, ok := msg["item"].(string)
				if !ok {
					continue
				}
				player.RemoveItem(item, 1)
				game.addToPile(player.Pos, item)
			} else if msg["action"] == "use" {
				item, ok := msg["item"].(string)
				if !ok {
					continue
				}
				player.UseItem(item)
			} else if verbose {
				log.Println("unknown action", msg)
			}
		case mmsg := <-game.MMsg:
			monster := mmsg.Monster
			msg := mmsg.Msg

			if msg["action"] == "move" {
				dir, ok := msg["dir"].(string)
				if !ok {
					continue
				}
				monster.Move(dir)
			}
		}
	}
}
