package main

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

//go:embed index.html
var html []byte

//go:embed style.css
var css []byte

//go:embed main.js
var js []byte

var upgrader = websocket.Upgrader{}
var verbose = false

type Point struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type Rect struct {
	X1 int `json:"x1"`
	Y1 int `json:"y1"`
	X2 int `json:"x2"`
	Y2 int `json:"y2"`
}

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
}

var mux = &sync.RWMutex{}
var games = make(map[string]*Game)

func makeRect(x1 int, y1 int, x2 int, y2 int) Rect {
	if x1 > x2 {
		x1, x2 = x2, x1
	}
	if y1 > y2 {
		y1, y2 = y2, y1
	}
	return Rect{x1, y1, x2, y2}
}

func randomRect(n int) Rect {
	x1 := rand.Intn(2*n) - n
	x2 := rand.Intn(2*n) - n
	y1 := rand.Intn(2*n) - n
	y2 := rand.Intn(2*n) - n
	return makeRect(x1, y1, x2, y2)
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


	for _, line := range lines {
		game.Rects = append(game.Rects, line)
	}
}

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

func (rect *Rect) Contains(x int, y int) bool {
	return x >= rect.X1 && x <= rect.X2 && y >= rect.Y1 && y <= rect.Y2
}

func (rect *Rect) Area() int {
	return (rect.X2 - rect.X1) * (rect.Y2 - rect.Y1)
}

func (rect *Rect) Center() Point {
	return Point{
		(rect.X2 + rect.X1) / 2,
		(rect.Y2 + rect.Y1) / 2,
	}
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

func (game *Game) IsFree(x int, y int) bool {
	for _, rect := range game.Rects {
		if rect.Contains(x, y) {
			return true
		}
	}
	return false
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
				}
			} else {
				log.Println(msg)
			}
		}
	}
}

func (player *Player) readPump() {
	defer func() {
		player.Game.unregister <- player
		player.conn.Close()
	}()

	for {
		msg := Message{}
		err := player.conn.ReadJSON(&msg)
		if err != nil {
			return
		}
		player.Game.Msg <- PlayerMessage{player, msg}
	}
}

func (player *Player) writePump() {
	defer player.conn.Close()
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case data := <-player.Send:
			err := player.conn.WriteJSON(data)
			if err != nil {
				return
			}
		case <-ticker.C:
			if !player.alive {
				return
			}
			player.alive = false
			err := player.conn.WriteMessage(websocket.PingMessage, nil)
			if err != nil {
				return
			}
		}
	}
}

func serveHome(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write(html)
}

func serveCSS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/css")
	w.WriteHeader(http.StatusOK)
	w.Write(css)
}

func serveJS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/javascript")
	w.WriteHeader(http.StatusOK)
	w.Write(js)
}

func serveWs(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	game := getGame(r.PathValue("id"))

	player := &Player{
		Game:  game,
		Send:  make(chan []Message),
		conn:  conn,
		alive: true,
		Id:    game.createId(),
		Pos:   Point{0, 0},
	}
	conn.SetPongHandler(func(string) error {
		player.alive = true
		return nil
	})
	game.register <- player

	go player.writePump()
	go player.readPump()
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "laneya [-v] [port]\n")
		flag.PrintDefaults()
	}

	flag.BoolVar(&verbose, "v", false, "enable verbose logs")
	flag.Parse()

	addr := "localhost:8000"
	if len(flag.Args()) > 0 {
		addr = fmt.Sprintf("localhost:%s", flag.Args()[0])
	}

	http.HandleFunc("GET /{$}", serveHome)
	http.HandleFunc("GET /style.css", serveCSS)
	http.HandleFunc("GET /main.js", serveJS)
	http.HandleFunc("GET /ws/{id}", serveWs)

	ctx, unregisterSignals := signal.NotifyContext(
		context.Background(), os.Interrupt, syscall.SIGTERM,
	)
	ctxFactory := func(l net.Listener) context.Context { return ctx }
	server := &http.Server{Addr: addr, BaseContext: ctxFactory}

	go func() {
		log.Printf("Serving on http://%s", addr)
		err := server.ListenAndServe()
		if err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()
	unregisterSignals()
	log.Println("Shutting down server…")
	server.Shutdown(context.Background())
}
