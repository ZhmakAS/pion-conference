package webrtc

import (
	"errors"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v3"
)

const PLISendInterval = time.Second * 2

var codecTypes = map[webrtc.RTPCodecType]string{
	webrtc.RTPCodecTypeAudio: "audio",
	webrtc.RTPCodecTypeVideo: "video",
}

var PeerConfig = webrtc.Configuration{
	ICEServers: []webrtc.ICEServer{
		{
			URLs: []string{"stun:stun.l.google.com:19302"},
		},
	},
}

type Connector struct {
	*MessageBroker

	mux       sync.Mutex
	closeOnce sync.Once
	clientID  string

	broadCastPeer *webrtc.PeerConnection
	listenPeer    *webrtc.PeerConnection

	signals      chan Payload
	renegotiates chan struct{}
	closes       chan struct{}

	listenTracks map[string][]*webrtc.RTPSender
	localTracks  []*webrtc.Track
}

func NewConnector(clientId string) (*Connector, error) {
	connector := &Connector{
		clientID:     clientId,
		localTracks:  make([]*webrtc.Track, 0),
		signals:      make(chan Payload),
		renegotiates: make(chan struct{}),
		closes:       make(chan struct{}),
		listenTracks: make(map[string][]*webrtc.RTPSender),
	}

	if err := connector.initBroadCastPeer(); err != nil {
		return nil, fmt.Errorf("failed to init broadCast peer connection: %s", err.Error())
	}

	connector.MessageBroker = NewMessageBroker(connector.broadCastPeer, clientId)

	return connector, nil
}

func (c *Connector) Signals() <-chan Payload {
	return c.signals
}

func (c *Connector) Renegotiates() <-chan struct{} {
	return c.renegotiates
}

func (c *Connector) Closes() <-chan struct{} {
	return c.closes
}

func (c *Connector) LocalTracks() []*webrtc.Track {
	return c.localTracks
}

func (c *Connector) ClientID() string {
	return c.clientID
}

func (c *Connector) AddListenTracks(clientId string, sender *webrtc.RTPSender) {
	c.mux.Lock()
	defer c.mux.Unlock()

	senders, ok := c.listenTracks[clientId]
	if !ok {
		senders = make([]*webrtc.RTPSender, 0)
	}
	senders = append(senders, sender)
	c.listenTracks[clientId] = senders
}

func (c *Connector) RemoveListenTracks(clientId string) error {
	c.mux.Lock()
	defer c.mux.Unlock()

	senders, ok := c.listenTracks[clientId]
	if !ok {
		return fmt.Errorf("listen tracks were not found: %s", clientId)
	}

	var err error
	for index := range senders {
		if err = c.listenPeer.RemoveTrack(senders[index]); err != nil {
			return fmt.Errorf("unable to remove listen sender: %s", clientId)
		}
	}

	delete(c.listenTracks, clientId)

	return nil
}

func (c *Connector) HandleBroadcastRemoteOffer(sessionDescription webrtc.SessionDescription) (err error) {
	if c.broadCastPeer.ICEConnectionState() == webrtc.ICEConnectionStateConnected {
		return errors.New("broadCastPeer already in connected state")
	}

	if err = c.broadCastPeer.SetRemoteDescription(sessionDescription); err != nil {
		return fmt.Errorf("error setting remote description: %w", err)
	}

	answer, err := c.broadCastPeer.CreateAnswer(nil)
	if err != nil {
		return fmt.Errorf("error creating answer: %w", err)
	}

	//TODO: should be changed on ICECandidateExchange
	gatherComplete := webrtc.GatheringCompletePromise(c.broadCastPeer)

	if err := c.broadCastPeer.SetLocalDescription(answer); err != nil {
		return fmt.Errorf("error setting local description: %w", err)
	}

	<-gatherComplete

	c.sendSignal(NewSDPPayload(answer, c.clientID))
	c.RenegotiateRequest()

	return nil
}

func (c *Connector) RenegotiateRequest() {
	c.sendSignal(NewRenegotiatePayload(c.clientID))
}

func (c *Connector) HandleListenRemoteOffer(peerConnection *webrtc.PeerConnection, sessionDescription webrtc.SessionDescription) (err error) {
	c.listenPeer = peerConnection
	if err = c.listenPeer.SetRemoteDescription(sessionDescription); err != nil {
		return fmt.Errorf("error setting remote description: %w", err)
	}

	answer, err := c.listenPeer.CreateAnswer(nil)
	if err != nil {
		return fmt.Errorf("error creating answer: %w", err)
	}

	//TODO: should be changed on ICECandidateExchange
	gatherComplete := webrtc.GatheringCompletePromise(c.listenPeer)

	if err := c.listenPeer.SetLocalDescription(answer); err != nil {
		return fmt.Errorf("error setting local description: %w", err)
	}

	<-gatherComplete

	c.signals <- NewSDPPayloadWithRenegotiate(answer, c.clientID)
	return nil
}

func (c *Connector) OnTrackHandler() func(remoteTrack *webrtc.Track, receiver *webrtc.RTPReceiver) {
	return func(remoteTrack *webrtc.Track, receiver *webrtc.RTPReceiver) {
		go c.runPLISender(remoteTrack)

		trackLabel := fmt.Sprintf("pion-%s-%s", codecTypes[remoteTrack.Kind()], c.ClientID())

		localTrack, newTrackErr := c.broadCastPeer.NewTrack(remoteTrack.PayloadType(), remoteTrack.SSRC(), "video", trackLabel)
		if newTrackErr != nil {
			panic(newTrackErr)
		}

		c.localTracks = append(c.localTracks, localTrack)
		if len(c.localTracks) == len(c.broadCastPeer.GetTransceivers()) {
			c.askAllNegotiation()
		}

		go c.transmitRTP(remoteTrack, localTrack)
	}
}

func (c *Connector) ICEConnectionStateChangeHandler(connectionState webrtc.ICEConnectionState) {
	log.Printf("Peer connection state changed: %s %s", c.clientID, connectionState.String())
	if connectionState == webrtc.ICEConnectionStateClosed ||
		connectionState == webrtc.ICEConnectionStateDisconnected ||
		connectionState == webrtc.ICEConnectionStateFailed {

		if err := c.closeBroadCast(); err != nil {
			log.Printf("unable to close peerConnection: %s", err.Error())
		}
	}
}

func (c *Connector) initBroadCastPeer() (err error) {
	c.broadCastPeer, err = webrtc.NewPeerConnection(PeerConfig)
	if err != nil {
		return fmt.Errorf("unable to create new webrtc.PeerConnection:%s", err.Error())
	}

	if _, err = c.broadCastPeer.AddTransceiverFromKind(webrtc.RTPCodecTypeVideo); err != nil {
		return fmt.Errorf("unable to add video codec transceiver1:%s", err.Error())
	}

	c.broadCastPeer.OnICEConnectionStateChange(c.ICEConnectionStateChangeHandler)
	c.broadCastPeer.OnTrack(c.OnTrackHandler())

	return nil
}

func (c *Connector) Close() error {
	if c.listenPeer != nil {
		if err := c.listenPeer.Close(); err != nil {
			return fmt.Errorf("unable to close listen peer: %s", err.Error())
		}
	}

	return c.closeBroadCast()
}

func (c *Connector) runPLISender(remote *webrtc.Track) {
	ticker := time.NewTicker(PLISendInterval)
	for range ticker.C {
		if rtcpSendErr := c.broadCastPeer.WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: remote.SSRC()}}); rtcpSendErr != nil {
			log.Printf("[%s] failed to write PLI to broadcast peer", c.clientID)

			if rtcpSendErr == io.ErrClosedPipe {
				log.Printf("[%s] stop PLI sender,closed pipe: %s", c.clientID, rtcpSendErr.Error())
			}
			return
		}
	}
}

func (c Connector) transmitRTP(remote, local *webrtc.Track) {
	var (
		rtpBuf = make([]byte, 1400)
		err    error
	)
	for {
		i, readErr := remote.Read(rtpBuf)
		if readErr != nil {
			log.Println("failed to read rtp from remote stream", readErr)
			return
		}

		if _, err = local.Write(rtpBuf[:i]); err != nil && err != io.ErrClosedPipe {
			log.Println("failed to write rtp to local stream", err)
			return
		}
	}
}

func (c *Connector) closeBroadCast() error {
	errs := make(chan error)
	c.closeOnce.Do(func() {
		var err error

		c.closes <- struct{}{}

		if c.broadCastPeer != nil {
			if err := c.broadCastPeer.Close(); err != nil {
				err = fmt.Errorf("unable to close broadCastPeer: %s", err.Error())
			}
		}

		close(c.closes)
		close(c.signals)
		close(c.renegotiates)
		errs <- err
	})

	return <-errs
}

func (c *Connector) sendSignal(payload Payload) {
	c.mux.Lock()

	go func() {
		defer c.mux.Unlock()
		select {
		case c.signals <- payload:
			return
		}
	}()
}

func (c *Connector) askAllNegotiation() {
	c.mux.Lock()

	go func() {
		defer c.mux.Unlock()
		select {
		case c.renegotiates <- struct{}{}:
			return
		}
	}()
}
