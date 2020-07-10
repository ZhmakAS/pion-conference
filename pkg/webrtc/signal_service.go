package webrtc

import (
	"fmt"
	"log"

	"sync"

	"github.com/pion/webrtc/v3"
)

type SignalService struct {
	mux                 sync.Mutex
	wg                  sync.WaitGroup
	closeOnce           sync.Once
	descriptionSentOnce sync.Once

	initiator    bool
	localPeerID  string
	remotePeerID string

	peerConnection *webrtc.PeerConnection
	negotiateSvc   *NegotiateService

	closed          bool
	signalChannel   chan Payload
	closeChannel    chan struct{}
	descriptionSent chan struct{}
}

func NewSignalService(peerConnection *webrtc.PeerConnection, localPeerID, remotePeerID string) (*SignalService, error) {
	s := &SignalService{
		peerConnection:  peerConnection,
		localPeerID:     localPeerID,
		remotePeerID:    remotePeerID,
		signalChannel:   make(chan Payload),
		closeChannel:    make(chan struct{}),
		descriptionSent: make(chan struct{}),
	}

	negotiator := NewNegotiateService(
		peerConnection,
		s.remotePeerID,
		s.handleLocalOffer,
		s.handleLocalRequestNegotiation,
	)

	s.negotiateSvc = negotiator

	peerConnection.OnICEConnectionStateChange(s.handleICEConnectionStateChange)
	peerConnection.OnICECandidate(s.handleICECandidate)

	return s, s.initialize()
}

func (s *SignalService) Signal(payload map[string]interface{}) error {
	signalPayload, err := NewPayloadFromMap(payload)
	if err != nil {
		return fmt.Errorf("Error constructing signal from payload: %s", err)
	}

	switch signal := signalPayload.Signal.(type) {
	case Candidate:
		log.Printf("[%s] Remote signal.canidate: %v", s.remotePeerID, signal.Candidate.Candidate)
		if signal.Candidate.Candidate != "" {
			return s.peerConnection.AddICECandidate(signal.Candidate)
		}
		return nil
	case Renegotiate:
		log.Printf("[%s] Calling signaller.Negotiate() because remote peer wanted to negotiate", s.remotePeerID)
		s.Negotiate()
		return nil
	case webrtc.SessionDescription:
		log.Printf("[%s] Remote signal.type: %s, signal.sdp: %s", s.remotePeerID, signal.Type, signal.SDP)
		return s.handleRemoteSDP(signal)
	default:
		return fmt.Errorf("[%s] Unexpected signal: %#v ", s.remotePeerID, signal)
	}
}

func (s *SignalService) CloseChannel() <-chan struct{} {
	return s.closeChannel
}

func (s *SignalService) SignalChannel() <-chan Payload {
	return s.signalChannel
}

func (s *SignalService) initialize() error {
	log.Printf("[%s] NewSignaller: Initiator pre-add video transceiver", s.remotePeerID)

	_, err := s.peerConnection.AddTransceiverFromKind(
		webrtc.RTPCodecTypeVideo,
		webrtc.RtpTransceiverInit{
			Direction: webrtc.RTPTransceiverDirectionRecvonly,
		},
	)
	if err != nil {
		log.Printf("ERROR: %s", err)
		return fmt.Errorf("[%s] NewSignaller: Error pre-adding video transceiver: %s", s.remotePeerID, err)
	}

	log.Printf("[%s] NewSignaller: Initiator pre-add audio transceiver", s.remotePeerID)
	_, err = s.peerConnection.AddTransceiverFromKind(
		webrtc.RTPCodecTypeAudio,
		webrtc.RtpTransceiverInit{
			Direction: webrtc.RTPTransceiverDirectionRecvonly,
		},
	)
	if err != nil {
		return fmt.Errorf("[%s] NewSignaller: Error pre-adding audio transceiver: %s", s.remotePeerID, err)
	}

	log.Printf("[%s] NewSignaller: Initiator calling Negotiate()", s.remotePeerID)
	s.negotiateSvc.Negotiate()

	return nil
}

func (s *SignalService) handleICEConnectionStateChange(connectionState webrtc.ICEConnectionState) {
	log.Printf("[%s] Peer connection state changed: %s", s.remotePeerID, connectionState.String())

	if connectionState == webrtc.ICEConnectionStateClosed || connectionState == webrtc.ICEConnectionStateDisconnected ||
		connectionState == webrtc.ICEConnectionStateFailed {
		s.Close()
	}

}

func (s *SignalService) sendSignal(payload Payload) {
	s.mux.Lock()

	go func() {
		defer s.mux.Unlock()

		ch := s.signalChannel
		if s.closed {
			ch = nil
		}

		select {
		case ch <- payload:

		case <-s.closeChannel:
			return
		}
	}()
}

func (s *SignalService) handleLocalRequestNegotiation() {
	log.Printf("[%s] Sending renegotiation request to initiator", s.remotePeerID)
	s.sendSignal(NewPayloadRenegotiate(s.localPeerID))
}

func (s *SignalService) Close() (err error) {
	s.closeOnce.Do(func() {
		close(s.closeChannel)

		s.mux.Lock()
		defer s.mux.Unlock()

		close(s.signalChannel)
		s.closed = true

		err = s.peerConnection.Close()
	})
	s.closeDescriptionSent()
	return
}

func (s *SignalService) handleICECandidate(candidate *webrtc.ICECandidate) {
	// wait until local description is set to prevent sending ice candidates
	log.Printf("[%s] Got ice candidate (waiting)", s.remotePeerID)
	<-s.descriptionSent
	log.Printf("[%s] Got ice candidate (processing...)", s.remotePeerID)

	if candidate == nil {
		return
	}

	payload := Payload{
		UserID: s.localPeerID,
		Signal: Candidate{
			Candidate: candidate.ToJSON(),
		},
	}

	s.sendSignal(payload)
}

func (s *SignalService) handleRemoteSDP(sessionDescription webrtc.SessionDescription) (err error) {
	switch sessionDescription.Type {
	case webrtc.SDPTypeAnswer:
		return s.handleRemoteAnswer(sessionDescription)
	default:
		return fmt.Errorf("[%s] Unexpected sdp type: %s", s.remotePeerID, sessionDescription.Type)
	}
}

func (s *SignalService) handleLocalOffer(offer webrtc.SessionDescription, err error) {
	if err != nil {
		log.Printf("[%s] Error creating local offer: %s", s.remotePeerID, err)
		s.Close()
		return
	}

	err = s.peerConnection.SetLocalDescription(offer)
	if err != nil {
		log.Printf("[%s] Error setting local description from local offer: %s", s.remotePeerID, err)
		s.Close()
		return
	}
	s.sendSignal(NewPayloadSDP(s.localPeerID, offer))

	// allow ice candidates to be sent
	s.closeDescriptionSent()
}

// closeDescriptionSent closes the descriptionSent channel which allows the ICE
// candidates to be processed.
func (s *SignalService) closeDescriptionSent() {
	s.descriptionSentOnce.Do(func() {
		close(s.descriptionSent)
	})
}

// Create an offer and send it to remote peer
func (s *SignalService) Negotiate() <-chan struct{} {
	return s.negotiateSvc.Negotiate()
}

func (s *SignalService) handleRemoteAnswer(sessionDescription webrtc.SessionDescription) (err error) {
	if err = s.peerConnection.SetRemoteDescription(sessionDescription); err != nil {
		return fmt.Errorf("[%s] Error setting remote description: %w", s.remotePeerID, err)
	}
	return nil
}
