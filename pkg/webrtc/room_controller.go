package webrtc

import (
	"log"
	"sync"

	"pion-conference/pkg/webrtc/track"
)

type RoomController struct {
	mux sync.RWMutex

	connectors     map[string]*Connector
	clientIDBySSRC map[uint32]string
}

func NewRoomPeersManager() *RoomController {
	return &RoomController{
		connectors:     make(map[string]*Connector),
		clientIDBySSRC: make(map[uint32]string),
	}
}

//Main room translating logic
func (r *RoomController) Add(connector *Connector) {
	go r.listenNewTracks(connector)
	go r.listenAll(connector)

	//go func() {
	//	for msg := range transport.MessagesChannel() {
	//		t.broadcast(transport.ClientID(), msg)
	//	}
	//}()

	r.mux.Lock()
	defer r.mux.Unlock()

	r.connectors[connector.ClientID()] = connector
}

// listenIncomingTracks listen for new tracks from new connected peers
func (r *RoomController) listenNewTracks(connector *Connector) {
	for trackEvent := range connector.TrackEventsChannel() {
		switch trackEvent.Type {
		case track.EventTypeAdd:
			r.addTrack(connector.ClientID(), trackEvent)
		}
	}
}

// broadcastRTP broadcast RTP traffic to all connected users
func (r *RoomController) listenAll(connector *Connector) {
	r.mux.Lock()

	var err error
	for otherClientID, otherConnector := range r.connectors {
		if otherClientID != connector.ClientID() {

			for _, localTrack := range otherConnector.LocalTracks() {
				if err = connector.AddTrack(localTrack, otherConnector.RTCPStream()); err != nil {
					log.Printf("Failed to add track to new connected track")
					return
				}
			}
		}
	}

	r.mux.Unlock()
}

//addTrack add track for all connected peers from new connected peer
func (t *RoomController) addTrack(clientID string, event track.Event) {
	t.mux.Lock()
	defer t.mux.Unlock()

	for otherClientID, otherConnector := range t.connectors {
		if otherClientID != clientID {
			if err := otherConnector.AddTrack(event.Track, event.RTCPStream); err != nil {
				log.Printf("[%s] MemoryTracksManager.addTrack Error adding track: %s", otherClientID, err)
				continue
			}
		}
	}
}

//func (t *RoomController) getTransportBySSRC(ssrc uint32) (connector *Connector, ok bool) {
//	t.mux.Lock()
//	defer t.mux.Unlock()
//
//	clientID, ok := t.clientIDBySSRC[ssrc]
//	if !ok {
//		return nil, false
//	}
//
//	connector, ok = t.connectors[clientID]
//	return connector, ok
//}

//func (t *RoomController) GetTracksMetadata(clientID string) (m []track.Metadata, ok bool) {
//	t.mux.RLock()
//	defer t.mux.RUnlock()
//
//	transport, ok := t.connectors[clientID]
//	if !ok {
//		return m, false
//	}
//
//	locals := transport.LocalTracks()
//	m = make([]track.Metadata, 0, len(locals))
//	for _, local := range locals {
//		trackMeta := track.NewTrackMetadataFromTrack(local, t.clientIDBySSRC[local.SSRC])
//
//		log.Printf("[%s] GetTracksMetadata: %d %#v", clientID, local.SSRC, trackMeta)
//		m = append(m, trackMeta)
//	}
//
//	return m, true
//}

//func (t *RoomController) Remove(clientID string) {
//	log.Printf("removePeer: %s", clientID)
//	t.mux.Lock()
//	defer t.mux.Unlock()
//
//	delete(t.connectors, clientID)
//}

////removeTrack remove track from all connected peers from new connected peer
//func (t *RoomController) removeTrack(clientID string, track track.Info) {
//	log.Printf("[%s] removeTrack ssrc: %d from other peers", clientID, track.SSRC)
//
//	t.mux.Lock()
//	defer t.mux.Unlock()
//
//	delete(t.clientIDBySSRC, track.SSRC)
//
//	for otherClientID, otherTransport := range t.connectors {
//		if otherClientID != clientID {
//			err := otherTransport.RemoveTrack(track.SSRC)
//			if err != nil {
//				log.Printf("[%s] removeTrack error removing track: %s", clientID, err)
//			}
//		}
//	}
//}

//func (t *RoomController) broadcast(clientID string, msg webrtc.DataChannelMessage) {
//	t.mux.Lock()
//	defer t.mux.Unlock()
//
//	for otherClientID, otherPeerInRoom := range t.connectors {
//		if otherClientID != clientID {
//			t.log.Printf("[%s] broadcast from %s", otherClientID, clientID)
//			tr := otherPeerInRoom.dataTransceiver
//			var err error
//			if msg.IsString {
//				err = tr.SendText(string(msg.Data))
//			} else {
//				err = tr.Send(msg.Data)
//			}
//			if err != nil {
//				t.log.Printf("[%s] broadcast error: %s", otherClientID, err)
//			}
//		}
//	}
//}
