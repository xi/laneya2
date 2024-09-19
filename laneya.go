package main

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
	"log"
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

type Message struct {
	Action string `json:"action"`
}

type Player struct {
	Game  *Game
	Send  chan []Message
	conn  *websocket.Conn
	alive bool
	Id    int
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
}

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
		mux.Lock()
		games[id] = game
		mux.Unlock()

		go game.run()
	}

	return game
}

func (game *Game) createId() int {
	game.lastId += 1
	return game.lastId
}

func (game *Game) run() {
	for {
		select {
		case player := <-game.register:
			game.Players[player] = true
		case player := <-game.unregister:
			delete(game.Players, player)
			if len(game.Players) == 0 {
				mux.Lock()
				delete(games, game.Id)
				mux.Unlock()
			}
		case cmsg := <-game.Msg:
			player := cmsg.Player
			msg := cmsg.Msg
			log.Println(msg.Action, player)
			// TODO
			player.Send <- []Message{msg}
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
			if player.alive {
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
