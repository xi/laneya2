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

type Client struct {
	Game  *Game
	Send  chan []Message
	conn  *websocket.Conn
	alive bool
}

type ClientMessage struct {
	Client *Client
	Msg    Message
}

type Game struct {
	Id         string
	Clients    map[*Client]bool
	Msg        chan ClientMessage
	register   chan *Client
	unregister chan *Client
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
			Clients:    make(map[*Client]bool),
			Msg:        make(chan ClientMessage),
			register:   make(chan *Client),
			unregister: make(chan *Client),
		}
		mux.Lock()
		games[id] = game
		mux.Unlock()

		go game.run()
	}

	return game
}

func (game *Game) run() {
	for {
		select {
		case client := <-game.register:
			game.Clients[client] = true
		case client := <-game.unregister:
			delete(game.Clients, client)
			if len(game.Clients) == 0 {
				mux.Lock()
				delete(games, game.Id)
				mux.Unlock()
			}
		case cmsg := <-game.Msg:
			client := cmsg.Client
			msg := cmsg.Msg
			log.Println(msg.Action, client)
			// TODO
			client.Send <- []Message{msg}
		}
	}
}

func (client *Client) readPump() {
	defer func() {
		client.Game.unregister <- client
		client.conn.Close()
	}()

	for {
		msg := Message{}
		err := client.conn.ReadJSON(&msg)
		if err != nil {
			return
		}
		client.Game.Msg <- ClientMessage{client, msg}
	}
}

func (client *Client) writePump() {
	defer client.conn.Close()
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case data := <-client.Send:
			err := client.conn.WriteJSON(data)
			if err != nil {
				return
			}
		case <-ticker.C:
			if client.alive {
				return
			}
			client.alive = false
			err := client.conn.WriteMessage(websocket.PingMessage, nil)
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

	client := &Client{
		Game:  game,
		Send:  make(chan []Message),
		conn:  conn,
		alive: true,
	}
	conn.SetPongHandler(func(string) error {
		client.alive = true
		return nil
	})
	game.register <- client

	go client.writePump()
	go client.readPump()
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
