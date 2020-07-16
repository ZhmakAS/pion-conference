package ws

import (
	"fmt"
	"log"
	"pion-conference/pkg/models/ws"
	"pion-conference/pkg/webrtc"
	"sync"
)

type SocketHandler struct {
	mux sync.Mutex

	connector *webrtc.Connector
	subsc     *Subscription
}

func NewSocketHandler(subsc *Subscription) SocketHandler {
	return SocketHandler{
		subsc: subsc,
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

	case "hangUp":
		return sh.handleHangUp()

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

	sh.subsc.WsRoomCtrl.SetMetadata(sh.subsc.ClientID, payload["nickname"].(string))

	connector, err := webrtc.NewConnector(sh.subsc.ClientID)
	if err != nil {
		return fmt.Errorf("error creating new webrtc.Connector: %w", err)
	}

	sh.connector = connector

	sh.subsc.WebRtcRoomCtrl.Add(connector)

	joinMessage := ws.NewMessageRoomJoin(sh.subsc.Room, sh.subsc.ClientID, sh.subsc.WsRoomCtrl.Metadata(sh.subsc.ClientID))

	err = sh.subsc.WsRoomCtrl.Emit(sh.subsc.ClientID, joinMessage)
	if err != nil {
		log.Printf("error sending metadata: %s, %s", sh.subsc.ClientID, err)
		return err
	}

	go sh.listenConnectorSignals()

	return nil
}

func (sh *SocketHandler) listenConnectorSignals() {
	for {
		select {

		case signal := <-sh.connector.Signals():
			err := sh.subsc.WsRoomCtrl.Emit(sh.subsc.ClientID, ws.NewMessage("signal", sh.subsc.Room, signal))
			if err != nil {
				log.Printf("[%s] error sending local signal: %s", sh.subsc.ClientID, err)
			}

		case <-sh.connector.Renegotiates():
			sh.subsc.WebRtcRoomCtrl.RenegotiateAll(sh.connector)

		case msg := <-sh.connector.MessagesChannel():
			sh.subsc.WebRtcRoomCtrl.BroadCastMessage(msg, sh.connector)

		case <-sh.connector.Closes():
			sh.subsc.WebRtcRoomCtrl.RemoveClosedTracks(sh.connector)
			return

		}
	}
}

func (sh *SocketHandler) handleSignal(message ws.Message) error {
	payload, ok := message.Payload.(map[string]interface{})
	if !ok {
		return fmt.Errorf("[%s] Ignoring signal because it is of unexpected type: %T", sh.subsc.ClientID, payload)
	}

	if sh.connector == nil {
		return fmt.Errorf("[%s] Ignoring signal '%v' because webRTCTransport is not initialized", sh.subsc.ClientID, payload)
	}

	return sh.subsc.WebRtcRoomCtrl.ProcessSignal(payload)
}

func (sh *SocketHandler) handleHangUp() error {
	if sh.connector == nil {
		return nil
	}

	defer sh.subsc.WebRtcRoomCtrl.Delete(sh.connector)

	closeErr := sh.connector.Close()
	if closeErr != nil {
		return fmt.Errorf("hangUp: Error closing peer connection: %s", closeErr)
	}

	return nil
}
