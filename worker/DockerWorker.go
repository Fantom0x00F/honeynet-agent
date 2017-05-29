package worker

import (
	"../hub"
	"../events"
	"log"
	"encoding/json"

	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"golang.org/x/net/context"
	"time"
	"fmt"
	"reflect"
)

type WorkerConfiguration struct {
	ImageName     string `json:"image_name"`
	RunParameters string `json:"run_parameters"`
}

type containerState struct {
	processTree  [][]string
	changedFiles []string
}

type DockerWorker struct {
	ctx             *context.Context
	cli             *client.Client
	containerId     string
	configuration   WorkerConfiguration
	containerEvents chan ContainerEvent
	containerState  *containerState
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
		containerState:  &containerState{},
	}
}

func (w *DockerWorker) Start() {
	go checkForUpdates(w)
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
		if w.configuration.ImageName == "" {
			w.Hub.Events <- events.Event{Type: events.AgentError, Message: "Empty image name!"}
			return
		}
		w.startContainer()
	case events.StopContainer:
		if w.containerId == "" {
			w.Hub.Events <- events.Event{Type: events.AgentError, Message: "Container already stopped"}
		}
		w.stopContainer()
	}
}

func (w *DockerWorker) startContainer() {
	images, err := w.cli.ContainerList(*w.ctx, types.ContainerListOptions{All: true})

	if err != nil {
		w.Hub.Events <- events.Event{Type: events.AgentError, Message: err.Error()}
		return
	}

	image := containsSpecific(&images, w.configuration.ImageName)
	if image == nil {
		w.Hub.Events <- events.Event{Type: events.AgentError, Message: "Image not found"}
		return
	}

	if err := w.cli.ContainerStart(*w.ctx, image.ID, types.ContainerStartOptions{}); err != nil {
		w.Hub.Events <- events.Event{Type: events.AgentError, Message: err.Error()}
		return
	}
	w.containerId = image.ID
	w.Hub.Events <- events.Event{Type: events.Normal, Message: "Container was started"}
}

func (w *DockerWorker) stopContainer() {
	images, err := w.cli.ContainerList(*w.ctx, types.ContainerListOptions{All: false})
	if err != nil {
		w.Hub.Events <- events.Event{Type: events.AgentError, Message: err.Error()}
		return
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
		return
	}
	w.containerId = ""
	w.Hub.Events <- events.Event{Type: events.Normal, Message: "Container was stopped"}
}

func checkForUpdates(w *DockerWorker) {
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-ticker.C:
			if w.containerId != "" {
				if err := checkUpdates(w); err != nil {
					log.Println("Errors due checks", err)
				}
			}
		}
	}
}

func checkUpdates(w *DockerWorker) error {
	images, err := w.cli.ContainerList(*w.ctx, types.ContainerListOptions{All: false})
	if err != nil {
		w.Hub.Events <- events.Event{Type: events.AgentError, Message: err.Error()}
		return err
	}
	image := containsSpecific(&images, w.configuration.ImageName)
	if image == nil || w.containerId != image.ID {
		w.Hub.Events <- events.Event{Type: events.AgentError, Message: "Container was killed"}
		return err
	}
	processes, err := w.cli.ContainerTop(*w.ctx, w.containerId, []string{"-e"})
	if err != nil {
		w.Hub.Events <- events.Event{Type: events.AgentError, Message: err.Error()}
	}
	if len(w.containerState.processTree) != 0 && !reflect.DeepEqual(w.containerState.processTree, processes.Processes) {
		w.Hub.Events <- events.Event{Type: events.MotionDetected, Message: "There is changes in processes!"}
		w.Hub.Events <- events.Event{Type: events.MotionDetected, Message: fmt.Sprint(processes.Processes)}
	}
	w.containerState.processTree = processes.Processes

	changes, err := w.cli.ContainerDiff(*w.ctx, w.containerId)
	if err != nil {
		w.Hub.Events <- events.Event{Type: events.AgentError, Message: err.Error()}
	}
	var changesStr []string
	for _, change := range changes {
		var kindPresentation string
		switch change.Kind {
		case 0:
			kindPresentation = "modified:"
		case 1:
			kindPresentation = "added:"
		case 2:
			kindPresentation = "deleted:"
		default:
			kindPresentation = "unknown:"
		}
		changesStr = append(changesStr, kindPresentation+change.Path)
	}

	if len(w.containerState.changedFiles) != 0 && !reflect.DeepEqual(w.containerState.changedFiles, changesStr) {
		w.Hub.Events <- events.Event{Type: events.MotionDetected, Message: "There is changes in filesystem!"}
		w.Hub.Events <- events.Event{Type: events.MotionDetected, Message: fmt.Sprint(changesStr)}
	}
	w.containerState.changedFiles = changesStr
	return nil
}

func containsSpecific(images *[]types.Container, imageName string) (*types.Container) {
	for _, image := range *images {
		if image.Image == imageName {
			return &image
		}
	}
	return nil
}
