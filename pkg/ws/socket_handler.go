package ws

import (
	"fmt"
	"log"
	"pion-conference/pkg/models/ws"
	"pion-conference/pkg/webrtc"
	"sync"

	webrtcc "github.com/pion/webrtc/v3"
)

type SocketHandler struct {
	mux sync.Mutex

	room       string
	clientID   string
	wsRoomCtrl RoomControllerService

	connector    *webrtc.Connector
	subscription *Subscription
}

func NewSocketHandler(subsc *Subscription, room, clientID string, roomCtrl RoomControllerService) SocketHandler {
	return SocketHandler{
		subscription: subsc,
		wsRoomCtrl:   roomCtrl,
		room:         room,
		clientID:     clientID,
	}
}

func (sh *SocketHandler) HandleMessage(message ws.Message) error {
	sh.mux.Lock()
	defer sh.mux.Unlock()

	switch message.Type {
	case "ready":
		return sh.handleReady(message)
	case "signal":
		return sh.handleSignal(message)
	case "ping":
		return nil
	}

	return fmt.Errorf("Unhandled event: %s", message.Type)
}

func (sh *SocketHandler) handleReady(message ws.Message) error {
	payload, ok := message.Payload.(map[string]interface{})
	if !ok {
		return fmt.Errorf("Ready message payload is of wrong type: %T", message.Payload)
	}

	sh.wsRoomCtrl.SetMetadata(sh.clientID, payload["nickname"].(string))

	clients, err := sh.wsRoomCtrl.GetReadyClients()
	if err != nil {
		return fmt.Errorf("Error retreiving ready clients: %w", err)
	}

	err = sh.wsRoomCtrl.Broadcast(
		ws.NewMessage("users", sh.room, map[string]interface{}{
			"initiator": "__SERVER__",
			"peerIds":   []string{"__SERVER__"},
			"nicknames": clients,
		}),
	)
	if err != nil {
		return fmt.Errorf("Error broadcasting users message: %s", err)
	}

	connector, err := webrtc.NewConnector(sh.clientID)
	if err != nil {
		return fmt.Errorf("Error creating new WebRTCTransport: %w", err)
	}
	sh.connector = connector

	roomPeersCtrl := webrtc.NewRoomPeersManager()

	roomPeersCtrl.Add(connector)
	go sh.processLocalSignals(message, connector.SignalChannel())

	return nil
}

func (sh *SocketHandler) handleSignal(message ws.Message) error {
	payload, ok := message.Payload.(map[string]interface{})
	if !ok {
		return fmt.Errorf("[%s] Ignoring signal because it is of unexpected type: %T", sh.clientID, payload)
	}

	if sh.connector == nil {
		return fmt.Errorf("[%s] Ignoring signal '%v' because webRTCTransport is not initialized", sh.clientID, payload)
	}

	return sh.connector.Signal(payload)
}

func (sh *SocketHandler) processLocalSignals(message ws.Message, signals <-chan webrtc.Payload) {
	for signal := range signals {
		if _, ok := signal.Signal.(webrtcc.SessionDescription); ok {
			err := sh.wsRoomCtrl.Emit(sh.clientID, ws.NewMessage("metadata", sh.room, MetadataPayload{
				UserID: "__SERVER__",
			}))
			if err != nil {
				log.Printf("[%s] Error sending metadata: %s", sh.clientID, err)
			}
		}

		err := sh.wsRoomCtrl.Emit(sh.clientID, ws.NewMessage("signal", sh.room, signal))
		if err != nil {
			log.Printf("[%s] Error sending local signal: %s", sh.clientID, err)
			// TODO abort connection
		}
	}

	sh.mux.Lock()
	defer sh.mux.Unlock()
	sh.connector = nil
	log.Printf("[%s] Peer connection closed, emitting hangUp event", sh.clientID)
	sh.wsRoomCtrl.SetMetadata(sh.clientID, "")

	err := sh.wsRoomCtrl.Broadcast(
		ws.NewMessage("hangUp", sh.room, map[string]string{
			"userId": sh.clientID,
		}),
	)
	if err != nil {
		log.Printf("[%s] Error broadcasting hangUp: %s", sh.clientID, err)
	}
}
