package worker

import (
	"log"
	"time"
	"../hub"
	"../events"
)

type Worker interface {
}

type ScheduledWorker struct {
	Hub *hub.Hub
}

func (w *ScheduledWorker) Start() {
	ticker := time.NewTicker(time.Second * 3)
	for {
		select {
		case command := <-w.Hub.Commands:
			log.Println("Received Command", command)
		case t := <-ticker.C:
			var event = events.Event{Type: 1, Message: t.String()}
			log.Println("Generated event", event)
			w.Hub.Events <- event
		}
	}
}
