package ws

import (
	"fmt"
	"sync"

	"pion-conference/pkg/models/ws"
)

type RoomControllerService struct {
	room    string
	clients map[string]ws.Client
	mux     sync.RWMutex
}

func NewRoomControllerService(room string) RoomControllerService {
	return RoomControllerService{
		clients: make(map[string]ws.Client),
		room:    room,
	}
}

func (r *RoomControllerService) Add(client ws.Client) {
	r.mux.Lock()
	clientID := client.ID()
	r.clients[clientID] = client
	r.mux.Unlock()
	return
}

func (r *RoomControllerService) Remove(clientID string) {
	r.mux.Lock()
	delete(r.clients, clientID)
	r.mux.Unlock()
	return
}

func (r *RoomControllerService) Size() (value int) {
	r.mux.RLock()
	value = len(r.clients)
	r.mux.RUnlock()
	return
}

func (r *RoomControllerService) Broadcast(msg ws.Message) error {
	r.mux.RLock()
	err := r.broadcast(msg)
	r.mux.RUnlock()
	return err
}

func (r *RoomControllerService) Emit(clientID string, msg ws.Message) error {
	r.mux.RLock()
	err := r.emit(clientID, msg)
	r.mux.RUnlock()
	return err
}

func (r *RoomControllerService) Room() string {
	return r.room
}

func (r *RoomControllerService) broadcast(msg ws.Message) (err error) {
	for clientID := range r.clients {
		if emitErr := r.emit(clientID, msg); emitErr != nil && err == nil {
			err = emitErr
		}
	}
	return
}

func (r *RoomControllerService) emit(clientID string, msg ws.Message) error {
	client, ok := r.clients[clientID]
	if !ok {
		return fmt.Errorf("Client not found, clientID: %s", clientID)
	}
	err := client.Write(msg)
	if err != nil {
		return fmt.Errorf("RoomControllerService.emit error: %w", err)
	}
	return nil
}
