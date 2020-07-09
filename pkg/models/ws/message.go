package ws

type Message struct {
	Type    string      `json:"type"`
	Room    string      `json:"room"`
	Payload interface{} `json:"payload"`
}

func NewMessage(typ string, room string, payload interface{}) Message {
	return Message{Type: typ, Room: room, Payload: payload}
}
