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

func serveWs(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
func handler(w http.ResponseWriter, r *http.Request) {
	if verbose {
		log.Println(r.Method, r.URL)
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Unsupported Method", http.StatusMethodNotAllowed)
	}

	if r.URL.Path == "/" {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write(html)
	} else if r.URL.Path == "/style.css" {
		w.Header().Set("Content-Type", "text/css")
		w.WriteHeader(http.StatusOK)
		w.Write(css)
	} else if r.URL.Path == "/main.js" {
		w.Header().Set("Content-Type", "text/javascript")
		w.WriteHeader(http.StatusOK)
		w.Write(js)
	} else if r.URL.Path == "/ws/" {
		serveWs(w, r)
	} else {
		http.Error(w, "Not Found", http.StatusNotFound)
	}
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

	http.HandleFunc("/", handler)

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
