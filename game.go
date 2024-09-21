package main

import (
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

type Message map[string]interface{}

type Player struct {
	Game  *Game
	Send  chan []Message
	conn  *websocket.Conn
	alive bool
	Id    int
	Pos   Point
}

type PlayerMessage struct {
	Player *Player
	Msg    Message
}

type Game struct {
	Id         string
	Players    map[*Player]bool
	Msg        chan PlayerMessage
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
		game = &Game{
			Id:         id,
			Players:    make(map[*Player]bool),
			Msg:        make(chan PlayerMessage),
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
	prev := Rect{-5, -5, 5, 5}

	game.Rects = []Rect{prev}
	lines := []Rect{}

	for i := 1; i <= 12; i++ {
		rect := randomRect(50)
		if rect.Area() < 250 {
			game.Rects = append(game.Rects, rect)

			p1 := prev.Center()
			p2 := rect.Center()

			lines = append(lines, makeRect(p1.X, p1.Y, p2.X, p1.Y))
			lines = append(lines, makeRect(p2.X, p1.Y, p2.X, p2.Y))

			prev = rect
		}
	}

	game.Ladder = prev.Center()

	for _, line := range lines {
		game.Rects = append(game.Rects, line)
	}
}

func (game *Game) IsFree(x int, y int) bool {
	for _, rect := range game.Rects {
		if rect.Contains(x, y) {
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

func (game *Game) run() {
	for {
		select {
		case player := <-game.register:
			setup := []Message{
				Message{
					"action": "setId",
					"id":     player.Id,
				},
				Message{
					"action": "setLevel",
					"rects":  game.Rects,
					"ladder": game.Ladder,
				},
			}
			for p := range game.Players {
				setup = append(setup, Message{
					"action": "create",
					"type":   "player",
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
					"id":     player.Id,
					"pos":    player.Pos,
				},
			})
		case player := <-game.unregister:
			delete(game.Players, player)
			if len(game.Players) == 0 {
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
				pos := player.Pos
				if msg["dir"] == "up" {
					pos.Y -= 1
				} else if msg["dir"] == "right" {
					pos.X += 1
				} else if msg["dir"] == "down" {
					pos.Y += 1
				} else if msg["dir"] == "left" {
					pos.X -= 1
				}
				if game.IsFree(pos.X, pos.Y) {
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
			} else {
				log.Println(msg)
			}
		}
	}
}
