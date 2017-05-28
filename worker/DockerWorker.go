package worker

import (
	"../hub"
	"../events"
	"log"
	"encoding/json"

	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	//"github.com/docker/engine-api/types/container"

	//"github.com/docker/docker/client"
	//"github.com/docker/docker/api/types"
	//"github.com/docker/docker/api/types/container"
	//"github.com/docker/go-connections/nat"
	"golang.org/x/net/context"
	//"fmt"
	"time"
)

type WorkerConfiguration struct {
	ImageName     string `json:"image_name"`
	RunParameters string `json:"run_parameters"`
}

type DockerWorker struct {
	ctx             *context.Context
	cli             *client.Client
	containerId     string
	configuration   WorkerConfiguration
	containerEvents chan ContainerEvent
	Hub             *hub.Hub
}

func NewDockerWorker(hub *hub.Hub) *DockerWorker {
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	return &DockerWorker{
		ctx:             &ctx,
		cli:             cli,
		Hub:             hub,
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

func (w *DockerWorker) parseCommand(command *events.Command) {
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
		//if w.configuration.ImageName == "" {
		//	w.Hub.Events <- events.Event{Type: events.AgentError, Message: "Empty image name!"}
		//	return
		//}
		w.startContainer()
	case events.StopContainer:
		if w.containerId == "" {
			w.Hub.Events <- events.Event{Type: events.AgentError, Message: "Container already stopped"}
		}
		w.stopContainer()
	}
}

func (w *DockerWorker) startContainer() {
	//images, err := w.cli.ImageList(context.Background(), types.ImageListOptions{})
	images, err := w.cli.ContainerList(*w.ctx, types.ContainerListOptions{All: true})

	if err != nil {
		panic(err)
	}

	image := containsSpecific(&images, w.configuration.ImageName)
	if image == nil {
		w.Hub.Events <- events.Event{Type: events.AgentError, Message: "Image not found"}
		return
	}

	//resp, err := w.cli.ContainerCreate(*w.ctx, &container.Config{
	//	Image: "alpine",
	//	Cmd:   []string{"echo", "hello world"},
	//	ExposedPorts: map[nat.Port]struct{}{"80/tcp": {}},
	//}, nil, nil, "")
	//if err != nil {
	//	panic(err)
	//}
	if err := w.cli.ContainerStart(*w.ctx, image.ID, types.ContainerStartOptions{}); err != nil {
		panic(err)
	}
	log.Println(image.ID)
	w.containerId = image.ID
}

func (w *DockerWorker) stopContainer() {
	images, err := w.cli.ContainerList(*w.ctx, types.ContainerListOptions{All: false})
	if err != nil {
		panic(err)
	}

	image := containsSpecific(&images, w.configuration.ImageName)
	if image == nil || w.containerId != image.ID {
		w.Hub.Events <- events.Event{Type: events.AgentError, Message: "Container was killed"}
		return
	}
	var second = time.Duration(time.Second)
	if err := w.cli.ContainerStop(*w.ctx, w.containerId, &second); err != nil {
		log.Println("Error on stop container", err)
		w.Hub.Events <- events.Event{Type: events.AgentError, Message: "Error on stop container"}
	}
}

func containsSpecific(images *[]types.Container, imageName string) (*types.Container) {
	for _, image := range *images {
		log.Println(image.Image)
		if image.Image == imageName {
			return &image
		}
	}
	return nil
}
