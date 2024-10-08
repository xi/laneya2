package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
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
	var timer *time.Timer = nil
	lastTime := time.UnixMicro(0)

	defer func() {
		if timer != nil {
			timer.Stop()
		}
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

		if timer != nil {
			timer.Stop()
		}
		frequency := 10 * math.Pow(1.07, float64(player.Speed))
		timeout := time.Duration(float64(time.Second) / frequency)
		timer = time.AfterFunc(time.Until(lastTime.Add(timeout)), func() {
			lastTime = time.Now()
			player.Game.Msg <- PlayerMessage{player, msg}
			timer = nil
		})
	}
}

func (player *Player) writePump() {
	defer player.conn.Close()
	ticker := time.NewTicker(20 * time.Second)

	defer func() {
		ticker.Stop()
		for _ = range player.send {
			// drain
		}
	}()

	for {
		select {
		case data, ok := <-player.send:
			if !ok {
				return
			}
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
		Game:        game,
		send:        make(chan []Message, 5),
		queue:       []Message{},
		conn:        conn,
		alive:       true,
		Id:          game.createId(),
		Pos:         Point{0, 0},
		Health:      100,
		HealthTotal: 100,
		Attack:      5,
		Defense:     0,
		LineOfSight: 5,
		Speed:       0,
		Inventory:   make(map[string]uint),
	}
	conn.SetPongHandler(func(string) error {
		player.alive = true
		return nil
	})
	game.register <- player

	go player.writePump()
	go player.readPump()
}

func serveItems(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Items)
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
	dumpItems := false

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "laneya [-v] [-s] [port]\n")
		fmt.Fprintf(os.Stderr, "laneya [-v] [-s] [--dump-items] [port]\n")
		flag.PrintDefaults()
	}

	flag.BoolVar(&verbose, "v", false, "enable verbose logs")
	flag.BoolVar(&static, "s", false, "serve static files (for development)")
	flag.BoolVar(&dumpItems, "dump-items", false, "dump items.json and exit")
	flag.Parse()

	if dumpItems {
		json.NewEncoder(os.Stdout).Encode(Items)
		return
	}

	addr := "localhost:8000"
	if len(flag.Args()) > 0 {
		addr = fmt.Sprintf("localhost:%s", flag.Args()[0])
	}

	if static {
		http.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, "index.html")
		})
		http.HandleFunc("GET /items.json", serveItems)
		http.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	}

	serve(addr)
}
