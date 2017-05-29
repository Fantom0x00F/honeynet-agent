package events

const (
	ReturnConfiguration EventType = 101
	ContainerStarted    EventType = 301
	ContainerStopped    EventType = 302
	MotionDetected      EventType = 500
	AgentError          EventType = 900
	Normal              EventType = 5
)

type EventType int

type Event struct {
	Type    EventType `json:"type"`
	Message string `json:"message,omitempty"`
}
