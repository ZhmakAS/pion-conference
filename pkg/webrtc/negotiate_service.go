package webrtc

import (
	"log"
	"sync"

	"github.com/pion/webrtc/v3"
)

type NegotiateService struct {
	remotePeerID         string
	peerConnection       *webrtc.PeerConnection
	onOffer              func(webrtc.SessionDescription, error)
	onRequestNegotiation func()

	negotiationDone   chan struct{}
	mu                sync.Mutex
	queuedNegotiation bool
}

func NewNegotiateService(peerConnection *webrtc.PeerConnection, remotePeerID string, onOffer func(webrtc.SessionDescription, error), onRequestNegotiation func(), ) *NegotiateService {
	n := &NegotiateService{
		peerConnection:       peerConnection,
		remotePeerID:         remotePeerID,
		onOffer:              onOffer,
		onRequestNegotiation: onRequestNegotiation,
	}

	peerConnection.OnSignalingStateChange(n.handleSignalingStateChange)
	return n
}

func (n *NegotiateService) closeDoneChannel() {
	if n.negotiationDone != nil {
		close(n.negotiationDone)
		n.negotiationDone = nil
	}
}

func (n *NegotiateService) Done() <-chan struct{} {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.negotiationDone != nil {
		return n.negotiationDone
	}
	ch := make(chan struct{})
	close(ch)
	return ch
}

func (n *NegotiateService) handleSignalingStateChange(state webrtc.SignalingState) {
	log.Printf("[%s] Signaling state change: %s", n.remotePeerID, state)

	n.mu.Lock()
	defer n.mu.Unlock()

	switch state {
	case webrtc.SignalingStateClosed:
		n.closeDoneChannel()
	case webrtc.SignalingStateStable:
		if n.queuedNegotiation {
			log.Printf("[%s] Executing queued negotiation", n.remotePeerID)
			n.queuedNegotiation = false
			n.negotiate()
		} else {
			n.closeDoneChannel()
		}
	}
}

func (n *NegotiateService) Negotiate() (done <-chan struct{}) {
	log.Printf("[%s] Negotiate", n.remotePeerID)

	n.mu.Lock()
	defer n.mu.Unlock()
	if n.negotiationDone != nil {
		log.Printf("[%s] Negotiate: already negotiating, queueing for later", n.remotePeerID)
		n.queuedNegotiation = true
		return
	}

	log.Printf("[%s] Negotiate: start", n.remotePeerID)
	n.negotiationDone = make(chan struct{})

	n.negotiate()
	return n.negotiationDone
}

func (n *NegotiateService) negotiate() {
	log.Printf("[%s] negotiate: creating offer", n.remotePeerID)
	offer, err := n.peerConnection.CreateOffer(nil)
	n.onOffer(offer, err)
}
