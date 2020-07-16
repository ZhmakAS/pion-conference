package webrtc

import (
	"fmt"
	"log"
	"sync"

	"github.com/pion/webrtc/v3"
)

type RoomController struct {
	mux sync.RWMutex

	connectors map[string]*Connector
}

func NewRoomController() *RoomController {
	return &RoomController{
		connectors: make(map[string]*Connector),
	}
}

func (r *RoomController) Add(connector *Connector) {
	r.mux.Lock()
	r.connectors[connector.ClientID()] = connector
	r.mux.Unlock()
}

func (r *RoomController) Delete(connector *Connector) {
	r.mux.Lock()
	r.RemoveClosedTracks(connector)
	delete(r.connectors, connector.ClientID())
	r.mux.Unlock()
}

func (r *RoomController) ProcessSignal(payload map[string]interface{}) error {
	signalPayload, err := NewSignalPayloadFromMap(payload)
	if err != nil {
		return fmt.Errorf("error constructing signal from payload: %s", err)
	}

	switch signal := signalPayload.Signal.(type) {
	//TODO: we will add new signal types e.g. IceCandidate
	case webrtc.SessionDescription:
		return r.handleRemoteSDP(signalPayload, signal)
	}

	return nil
}

func (r *RoomController) handleRemoteSDP(payload *Payload, sessionDescription webrtc.SessionDescription) error {
	if sessionDescription.Type != webrtc.SDPTypeOffer {
		return fmt.Errorf("unsupported webrtc.SDPType %s", sessionDescription.Type)
	}

	connector, ok := r.connectors[payload.ClientId]
	if !ok {
		return fmt.Errorf("unable to find webrtc.Connector by userId %s", payload.ClientId)
	}

	if !payload.Renegotiate {
		return connector.HandleBroadcastRemoteOffer(sessionDescription)
	}

	return r.renegotiateListenPeer(connector, sessionDescription)
}

func (r *RoomController) renegotiateListenPeer(conn *Connector, sessionDescription webrtc.SessionDescription) error {
	r.mux.Lock()
	defer r.mux.Unlock()

	peerConnection, err := webrtc.NewPeerConnection(PeerConfig)
	if err != nil {
		return fmt.Errorf("unable to create new webrtc.PeerConnection:%s", err.Error())
	}

	var tracks []*webrtc.Track
	for clientID, connector := range r.connectors {
		if clientID == conn.ClientID() {
			continue
		}

		tracks = connector.LocalTracks()
		for trackId := range connector.LocalTracks() {

			sender, err := peerConnection.AddTrack(tracks[trackId])
			if err != nil {
				return fmt.Errorf("unable to add listen track to PeerConnection:%s", err.Error())
			}

			conn.AddListenTracks(connector.ClientID(), sender)
		}
	}

	return conn.HandleListenRemoteOffer(peerConnection, sessionDescription)
}

func (r *RoomController) RenegotiateAll(conn *Connector) {
	r.mux.Lock()
	for clientID, connector := range r.connectors {
		if clientID == conn.ClientID() {
			continue
		}
		connector.RenegotiateRequest()
	}

	r.mux.Unlock()
}

func (r *RoomController) BroadCastMessage(message webrtc.DataChannelMessage, conn *Connector) {
	for clientID, connector := range r.connectors {
		if clientID == conn.ClientID() {
			continue
		}

		if err := r.sendMessage(message, connector); err != nil {
			log.Printf("[%s] broadcast error: %s", clientID, err)
		}
	}
}

func (r *RoomController) RemoveClosedTracks(conn *Connector) {
	for clientID, connector := range r.connectors {
		if clientID == conn.ClientID() {
			continue
		}

		if err := connector.RemoveListenTracks(conn.ClientID()); err != nil {
			log.Printf("unable to remove [%s] tracks from active connector ", conn.ClientID())
		}
		connector.RenegotiateRequest()
	}
}

func (r *RoomController) sendMessage(message webrtc.DataChannelMessage, connector *Connector) error {
	if message.IsString {
		return connector.SendText(string(message.Data))
	}

	return connector.Send(message.Data)
}
