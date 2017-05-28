package events

type Command struct {
	Type    int
	Message string `json:",omitempty"`
}
