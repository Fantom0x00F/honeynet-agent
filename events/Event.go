package events

type Event struct {
	Type    int `json:"type"`
	Message string `json:"message,omitempty"`
}
