package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{}

func (player *Player) readPump() {
	defer func() {
		player.Game.unregister <- player
		player.conn.Close()
	}()

	for {
		msg := Message{}
		err := player.conn.ReadJSON(&msg)
		if err != nil {
			if verbose {
				log.Println(err)
			}
			return
		}
		player.Game.Msg <- PlayerMessage{player, msg}
	}
}

func (player *Player) writePump() {
	defer player.conn.Close()
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case data := <-player.Send:
			err := player.conn.WriteJSON(data)
			if err != nil {
				if verbose {
					log.Println(err)
				}
				return
			}
		case <-ticker.C:
			if !player.alive {
				return
			}
			player.alive = false
			err := player.conn.WriteMessage(websocket.PingMessage, nil)
			if err != nil {
				if verbose {
					log.Println(err)
				}
				return
			}
		}
	}
}

func serveWs(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		if verbose {
			log.Println(err)
		}
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

func serve(addr string) {
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

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "laneya [-v] [-s] [port]\n")
		flag.PrintDefaults()
	}

	flag.BoolVar(&verbose, "v", false, "enable verbose logs")
	flag.BoolVar(&static, "s", false, "serve static files (for development)")
	flag.Parse()

	addr := "localhost:8000"
	if len(flag.Args()) > 0 {
		addr = fmt.Sprintf("localhost:%s", flag.Args()[0])
	}

	if static {
		http.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, "index.html")
		})
		http.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	}

	serve(addr)
}
