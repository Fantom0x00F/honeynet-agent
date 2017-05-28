package events

const (
	GetConfiguration CommandType = 101
	SetConfiguration CommandType = 102
	StartContainer   CommandType = 301
	StopContainer    CommandType = 302
)

type CommandType int

type Command struct {
	Type    CommandType `json:"type"`
	Message string `json:"message,omitempty"`
}