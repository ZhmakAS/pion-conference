package webrtc

import (
	"errors"
	"fmt"
	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v3"
	"io"
	"log"
	"pion-conference/pkg/webrtc/track"
	"sync"
)

type Connector struct {
	mux sync.Mutex
	wg  sync.WaitGroup

	clientID       string
	peerConnection *webrtc.PeerConnection

	signalService *SignalService
	trackEvents   chan track.Event
	rtcpStream    chan rtcp.Packet

	localTracks []*webrtc.Track
}

func NewConnector(clientId string) (*Connector, error) {
	var mediaEngine webrtc.MediaEngine

	mediaEngine.RegisterCodec(webrtc.NewRTPOpusCodec(webrtc.DefaultPayloadTypeOpus, 48000))

	rtcpfb := []webrtc.RTCPFeedback{
		webrtc.RTCPFeedback{
			Type: webrtc.TypeRTCPFBGoogREMB,
		},
		// in Section 6.3.1.
		webrtc.RTCPFeedback{
			Type:      webrtc.TypeRTCPFBNACK,
			Parameter: "pli",
		},
	}

	mediaEngine.RegisterCodec(webrtc.NewRTPVP8CodecExt(webrtc.DefaultPayloadTypeVP8, 90000, rtcpfb, ""))

	api := webrtc.NewAPI(
		webrtc.WithMediaEngine(mediaEngine),
	)

	peerConnectionConfig := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs:           []string{"stun:stun.l.google.com:19302"},
				CredentialType: webrtc.ICECredentialTypePassword,
			},
			{
				URLs:           []string{"stun:global.stun.twilio.com:3478?transport=udp"},
				CredentialType: webrtc.ICECredentialTypePassword,
			},
		},
	}

	peerConnection, err := api.NewPeerConnection(peerConnectionConfig)
	if err != nil {
		return nil, errors.New("unable to create new webrtc PeerConnection")
	}

	signalService, err := NewSignalService(
		peerConnection,
		"__SERVER__",
		clientId,
	)

	connector := &Connector{
		peerConnection: peerConnection,
		clientID:       clientId,
		signalService:  signalService,
		trackEvents:    make(chan track.Event),
		rtcpStream:     make(chan rtcp.Packet),
		localTracks:    make([]*webrtc.Track, 0),
	}

	if err = connector.initPeer(); err != nil {

	}
	return connector, nil
}

func (c *Connector) RTCPStream() chan rtcp.Packet {
	return c.rtcpStream
}

func (c *Connector) LocalTracks() []*webrtc.Track {
	return c.localTracks
}

func (c *Connector) Close() {
	c.wg.Wait()

	//TODO: add closer of signaler/dataStreamer
	close(c.rtcpStream)
	close(c.trackEvents)
}

func (c *Connector) AddTrack(track *webrtc.Track, rtcpStream chan rtcp.Packet) error {
	sender, err := c.peerConnection.AddTrack(track)
	if err != nil {
		log.Printf("Failed to add new track: ClientID :%s", c.clientID)
		return err
	}

	go c.runRTCPListener(sender, rtcpStream)

	return nil
}

func (c *Connector) ClientID() string {
	return c.clientID
}

func (c *Connector) TrackEventsChannel() <-chan track.Event {
	return c.trackEvents
}

func (c *Connector) Signal(payload map[string]interface{}) error {
	return c.signalService.Signal(payload)
}

func (p *Connector) SignalChannel() <-chan Payload {
	return p.signalService.SignalChannel()
}

func (c *Connector) initPeer() error {
	c.peerConnection.OnICEGatheringStateChange(func(state webrtc.ICEGathererState) {
		log.Printf("[%s] ICE gathering state changed: %s", c.clientID, state)
	})

	c.peerConnection.OnTrack(c.handleTrack)

	go c.Close()
	return nil
}
func (c *Connector) handleTrack(remote *webrtc.Track, receiver *webrtc.RTPReceiver) {
	log.Printf("[%s] Remote track: %d", c.clientID, remote.SSRC)

	go c.runPLISender()

	trackID := fmt.Sprintf("client-local-track-%s", c.clientID)

	localTrack, err := c.peerConnection.NewTrack(remote.PayloadType(), remote.SSRC(), trackID, trackID)
	if err != nil {
		log.Println("Failed to obtain localTrack from remote, clientID: %s", c.clientID)
	}

	go c.transmitRTP(remote, localTrack)

	c.sendTrackEvent(track.Event{
		Track: localTrack,
		Type:  track.EventTypeAdd,
	})

	c.addLocalTrack(localTrack)
}

func (c *Connector) runPLISender() {
	var err error

	for pkt := range c.rtcpStream {
		switch pkt.(type) {
		case *rtcp.PictureLossIndication:
			err = c.peerConnection.WriteRTCP([]rtcp.Packet{pkt})
			if err != nil {
				log.Printf("Failed to resend PLI to peer connection: %s", c.clientID)
				return
			}
		}
	}
}
func (c *Connector) transmitRTP(remote, local *webrtc.Track) {
	var rtpBuf = make([]byte, 1400)

	for {
		n, err := remote.Read(rtpBuf)
		if err != nil {
			log.Printf("Unable to read RTP from remotetrack: clientID: %s", c.clientID)
			return
		}

		if _, err = local.Write(rtpBuf[:n]); err != nil && err != io.ErrClosedPipe {
			log.Printf("Unable to transmit RTP to localtrack: clientID: %s", c.clientID)
			return
		}
	}
}

func (c *Connector) runRTCPListener(sender *webrtc.RTPSender, rtcpStream chan rtcp.Packet) {
	for {
		rtcpPackets, err := sender.ReadRTCP()
		if err != nil {
			log.Printf("Failed to read RCTP packet, clientID: %s", c.clientID)
			return
		}
		for _, rtcpPacket := range rtcpPackets {
			rtcpStream <- rtcpPacket
		}
	}
}

func (c *Connector) sendTrackEvent(event track.Event) {
	c.trackEvents <- event
}

func (c *Connector) addLocalTrack(local *webrtc.Track) {
	c.mux.Lock()
	defer c.mux.Unlock()

	c.localTracks = append(c.localTracks, local)
}

//func (c *Connector) RemoveTrack(ssrc uint32) error {
//	c.mux.Lock()
//	defer c.mux.Unlock()
//
//	pta, ok := c.localTracks[ssrc]
//	if !ok {
//		return fmt.Errorf("Track not found: %d", ssrc)
//	}
//
//	err := c.peerConnection.RemoveTrack(pta.Sender())
//	if err != nil {
//		return err
//	}
//
//	//c.signaller.Negotiate()
//
//	delete(c.localTracks, ssrc)
//	return nil
//}

//
//func (c *Connector) addRemoteTrack(remote track.RemoteInfo) {
//	c.mux.Lock()
//	defer c.mux.Unlock()
//
//	c.remoteTracks[remote.Info().SSRC] = remote
//}
//

//
//func (c *Connector) removeRemoteTrack(ssrc uint32) {
//	c.mux.Lock()
//	defer c.mux.Unlock()
//
//	delete(c.remoteTracks, ssrc)
//}
//
//func (c *Connector) getTransceiverByReceiver(receiver *webrtc.RTPReceiver) (transceiver *webrtc.RTPTransceiver) {
//	for _, tr := range c.peerConnection.GetTransceivers() {
//		if tr.Receiver() == receiver {
//			transceiver = tr
//			break
//		}
//	}
//
//	return
//}
//
//func (c *Connector) getTransceiverBySender(sender *webrtc.RTPSender) (transceiver *webrtc.RTPTransceiver) {
//	for _, tr := range c.peerConnection.GetTransceivers() {
//		if tr.Sender() == sender {
//			transceiver = tr
//			break
//		}
//	}
//
//	return
//}
//
//
//func (c *Connector) transmitRTP(remote track.RemoteInfo) {
//	defer c.stopTransmitting(remote.Info())
//
//	for {
//		pkt, err := remote.Track().ReadRTP()
//		if err != nil {
//			log.Printf("[%s] Remote track has ended: %d: %s", c.clientID, remote.Info().SSRC, err)
//			return
//		}
//
//		log.Printf("[%s] ReadRTP: %s", c.clientID, pkt)
//		c.rtpStream <- pkt
//	}
//}
//

//
//func (c *Connector) stopTransmitting(info track.Info) {
//	c.removeRemoteTrack(info.SSRC)
//	c.sendTrackEvent(track.Event{
//		Info: info,
//		Type: track.EventTypeRemove,
//	})
//	c.wg.Done()
//}

//func (p *WebRTCTransport) Signal(payload map[string]interface{}) error {
//	return p.signaller.Signal(payload)
//}
//
//func (p *WebRTCTransport) SignalChannel() <-chan Payload {
//	return p.signaller.SignalChannel()
//}
//func (p *WebRTCTransport) MessagesChannel() <-chan webrtc.DataChannelMessage {
//	return p.dataTransceiver.MessagesChannel()
//}
