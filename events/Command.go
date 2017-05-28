package events

type Command struct {
	Type    int `json:"type"`
	Message string `json:",omitempty"`
}
