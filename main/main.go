package main

import (
	"flag"
	"os"
	"os/signal"
	"net/url"
	"log"
	"time"

	"github.com/gorilla/websocket"
	"../centralnodeconnection"
)

var addr = flag.String("connection", "localhost:8090", "http service addr")

func main() {
	flag.Parse()
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: "ws", Host: *addr, Path: "/echo"}
	log.Printf("Try to connect to %s", u.String())

	CC := centralnodeconnection.CentralNodeConnection{
		Secret:         "ownsecret",
		ResponseSecret: "ownsecret2",
		Url:            u,
	}

	if err := CC.Open(); err != nil {
		log.Fatal("dial:", err)
	}
	defer CC.Close()

	done := make(chan struct{})

	go func() {
		defer CC.Close()
		defer close(done)
		for {
			_, message, err := CC.ReadMessage()
			if err != nil {
				log.Println("Receive message error:", err)
				if err := CC.Reconnect(); err != nil {
					log.Fatal("Can't reconnect to socket ", err)
				}
			}
			log.Printf("Received: %s", message)
		}
	}()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case t := <- ticker.C:
			err := CC.WriteMessage(websocket.TextMessage, []byte(t.String()))
			if err != nil {
				log.Println(err)
				return
			}
		case <- interrupt:
			log.Println("Interrupt")
			err := CC.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("Write close error: ", err)
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
		}
	}
}