package webrtc

import "sync"

type RoomController struct {
	mu sync.RWMutex

	transports     map[string]*Connector
	clientIDBySSRC map[uint32]string
}

func NewRoomController() {

}
