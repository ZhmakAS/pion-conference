package ws

const (
	MessageTypeRoomJoin  string = "ws_room_join"
	MessageTypeRoomLeave string = "ws_room_leave"
)

type Message struct {
	Type    string      `json:"type"`
	Room    string      `json:"room"`
	Payload interface{} `json:"payload"`
}

func NewMessage(typ string, room string, payload interface{}) Message {
	return Message{Type: typ, Room: room, Payload: payload}
}

func NewMessageRoomJoin(room string, clientID string, metadata string) Message {
	return NewMessage(MessageTypeRoomJoin, room, map[string]string{
		"clientID": clientID,
		"metadata": metadata,
	})
}
