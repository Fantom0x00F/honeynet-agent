package hub

import (
	"../centralnodeconnection"
	"../events"
	"log"
	"github.com/gorilla/websocket"
	"encoding/json"
)

type Hub struct {
	Conn     *centralnodeconnection.CentralNodeConnection
	Events   chan events.Event
	Commands chan events.Command
}

func NewHub(conn *centralnodeconnection.CentralNodeConnection) *Hub {
	return &Hub{
		Conn:     conn,
		Events:   make(chan events.Event),
		Commands: make(chan events.Command),
	}
}

func (h *Hub) Start() {
	defer h.Conn.Close()
	defer close(h.Events)
	defer close(h.Commands)

	if err := h.Conn.Open(); err != nil {
		log.Fatal("dial:", err)
	}

	go func() {
		for {
			_, message, err := h.Conn.ReadMessage()
			if err != nil {
				log.Println("Receive message error:", err)
				if err := h.Conn.Reconnect(); err != nil {
					log.Fatal("Can't reconnect to socket ", err)
				}
			}
			command := events.Command{}
			if err := json.Unmarshal(message, &command); err != nil {
				log.Println("Failed to unmarshall")
			}
			h.Commands <- command
		}
	}()

	for {
		select {
		case event := <-h.Events:
			if payload, err := json.Marshal(event); err == nil {
				h.Conn.WriteMessage(websocket.TextMessage, payload)
			}
		}
	}
}
