package ws

import (
	"fmt"
	"sync"

	"pion-conference/pkg/models/ws"
)

type RoomController struct {
	room    string
	clients map[string]ws.Client
	mux     sync.RWMutex
}

func NewRoomController(room string) *RoomController {
	return &RoomController{
		clients: make(map[string]ws.Client),
		room:    room,
	}
}

func (r *RoomController) GetReadyClients() (map[string]string, error) {
	filteredClients := map[string]string{}
	clients, err := r.Clients()
	if err != nil {
		return filteredClients, err
	}
	for clientID, nickname := range clients {
		if nickname != "" {
			filteredClients[clientID] = nickname
		}
	}
	return filteredClients, nil
}

func (r *RoomController) Add(client ws.Client) {
	r.mux.Lock()
	clientID := client.ID()
	r.clients[clientID] = client
	r.mux.Unlock()
}

func (r *RoomController) SetMetadata(clientID string, metadata string) (ok bool) {
	r.mux.Lock()
	defer r.mux.Unlock()
	client, ok := r.clients[clientID]
	if ok {
		client.SetMetadata(metadata)
	}
	return
}

func (r *RoomController) Metadata(clientID string) (metadata string) {
	r.mux.Lock()
	defer r.mux.Unlock()
	client, ok := r.clients[clientID]
	if ok {
		metadata = client.Metadata()
	}
	return
}

// Returns clients with metadata
func (r *RoomController) Clients() (clientIDs map[string]string, err error) {
	r.mux.RLock()
	clientIDs = map[string]string{}
	for clientID, client := range r.clients {
		clientIDs[clientID] = client.Metadata()
	}
	r.mux.RUnlock()
	return
}

func (r *RoomController) Remove(clientID string) {
	r.mux.Lock()
	delete(r.clients, clientID)
	r.mux.Unlock()
	return
}

func (r *RoomController) Size() (value int) {
	r.mux.RLock()
	value = len(r.clients)
	r.mux.RUnlock()
	return
}

func (r *RoomController) Broadcast(msg ws.Message) error {
	r.mux.RLock()
	err := r.broadcast(msg)
	r.mux.RUnlock()
	return err
}

func (r *RoomController) Emit(clientID string, msg ws.Message) error {
	r.mux.RLock()
	err := r.emit(clientID, msg)
	r.mux.RUnlock()
	return err
}

func (r *RoomController) Room() string {
	return r.room
}

func (r *RoomController) broadcast(msg ws.Message) (err error) {
	for clientID := range r.clients {
		if emitErr := r.emit(clientID, msg); emitErr != nil && err == nil {
			err = emitErr
		}
	}
	return
}

func (r *RoomController) emit(clientID string, msg ws.Message) error {
	client, ok := r.clients[clientID]
	if !ok {
		return fmt.Errorf("Client not found, clientID: %s", clientID)
	}
	err := client.Write(msg)
	if err != nil {
		return fmt.Errorf("RoomController.emit error: %w", err)
	}
	return nil
}
