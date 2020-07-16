package webrtc

import (
	"fmt"
	"sync"
)

var roomsService *RoomsService

type RoomsService struct {
	controllers map[string]*RoomController
	mux         sync.RWMutex
}

func NewRoomsService() *RoomsService {
	return &RoomsService{
		controllers: make(map[string]*RoomController),
	}
}

func (rs *RoomsService) SetRoomController(roomId string) *RoomController {
	rs.mux.Lock()
	roomCtrl, ok := rs.controllers[roomId]
	if !ok {
		newRoomCrl := NewRoomController()
		rs.controllers[roomId] = newRoomCrl
		roomCtrl = newRoomCrl
	}
	rs.mux.Unlock()
	return roomCtrl
}

func (rs *RoomsService) DeleteRoomController(roomId string) error {
	rs.mux.Lock()
	if _, ok := rs.controllers[roomId]; !ok {
		return fmt.Errorf("RoomController now found: %s", roomId)
	}
	delete(rs.controllers, roomId)

	rs.mux.Unlock()
	return nil
}

func InitRoomsService() {
	if roomsService != nil {
		return
	}
	roomsService = NewRoomsService()
}

func GetRoomsService() *RoomsService {
	return roomsService
}
