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

type Client struct {
	game *Game
	conn *websocket.Conn
	send chan []byte
}

type Message struct {
	client *Client
	data   []byte
}

type Game struct {
	clients map[*Client]bool
	msg     chan Message
}

var mux = &sync.RWMutex{}
var games = make(map[string]*Game)

func getGame(id string) *Game {
	mux.RLock()
	game, ok := games[id]
	mux.RUnlock()

	if !ok {
		game = &Game{
			msg:     make(chan Message),
			clients: make(map[*Client]bool),
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
		case msg := <-game.msg:
			// TODO
			log.Println(msg.data)
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
		game: game,
		conn: conn,
		send: make(chan []byte, 256),
	}
	game.msg <- Message{client: client, data: []byte("register")}
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
