package worker

import (
	"../hub"
	"../events"
	"log"
	"encoding/json"
)

type WorkerConfiguration struct {
	ImageName     string `json:"image_name"`
	RunParameters string `json:"run_parameters"`
}

type DockerWorker struct {
	containerId     string
	configuration   WorkerConfiguration
	containerEvents chan ContainerEvent
	Hub             *hub.Hub
}

func NewDockerWorker() *DockerWorker {
	return &DockerWorker{
		containerEvents: make(chan ContainerEvent),
	}
}

func (w *DockerWorker) Start() {
	//todo start
	for {
		select {
		case command := <-w.Hub.Commands:
			w.parseCommand(&command)
		case containerEvent := <-w.containerEvents:
			res, _ := json.Marshal(containerEvent)
			var event = events.Event{Type: events.MotionDetected, Message: string(res)}
			log.Println("Generated event", event)
			w.Hub.Events <- event
		}
	}
}

func (w*DockerWorker) parseCommand(command *events.Command) {
	switch command.Type {
	case events.GetConfiguration:
		res, err := json.Marshal(w.configuration)
		if err != nil {
			log.Println("Can't marshall configuration ", err)
		}
		w.Hub.Events <- events.Event{Type: events.ReturnConfiguration, Message: string(res) }
	case events.SetConfiguration:
		log.Println("Get message", command.Message)
		json.Unmarshal([]byte(command.Message), &w.configuration)
	case events.StartContainer:
		if w.containerId != "" {
			w.Hub.Events <- events.Event{Type: events.AgentError, Message: "Container already started"}
			return
		}
		//todo: start container
	case events.StopContainer:
		if w.containerId == "" {
			w.Hub.Events <- events.Event{Type: events.AgentError, Message: "Container already stopped"}
		}
		//todo: stop container
	}
}
