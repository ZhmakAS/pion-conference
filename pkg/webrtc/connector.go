package webrtc

import (
	"errors"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
)

type Connector struct {
	mu sync.Mutex
	wg sync.WaitGroup

	clientID       string
	peerConnection *webrtc.PeerConnection

	trackEventsCh chan TrackEvent
	rtpStream     chan *rtp.Packet
}

func NewConnector(clientId string) (*Connector, error) {
	peerConnectionConfig := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	peerConnection, err := webrtc.NewPeerConnection(peerConnectionConfig)
	if err != nil {
		return nil, errors.New("unable to create new webrtc PeerConnection")
	}

	connector := &Connector{
		peerConnection: peerConnection,
		clientID:       clientId,
		trackEventsCh:  make(chan TrackEvent),
		rtpStream:      make(chan *rtp.Packet),
	}

	if err = connector.initPeer(); err != nil {

	}
	return connector, nil
}

func (c *Connector) initPeer() error {
	c.peerConnection.OnICEGatheringStateChange(func(state webrtc.ICEGathererState) {
		log.Printf("[%s] ICE gathering state changed: %s", c.clientID, state)
	})

	c.peerConnection.OnTrack()

	go func() {
		c.wg.Wait()
		close(c.rtpStream)
		close(c.trackEventsCh)
	}()

	return nil
}

func (p *Connector) handleTrack(track *webrtc.Track, receiver *webrtc.RTPReceiver) {
	trackInfo := NewTrackInfoFromWebRTC(track)

	rti := remoteTrackInfo{trackInfo, transceiver, receiver, track}

	p.addRemoteTrack(rti)
	p.trackEventsCh <- TrackEvent{
		TrackInfo: trackInfo,
		Type:      TrackEventTypeAdd,
	}

	p.wg.Add(1)
	go func() {
		defer func() {
			p.removeRemoteTrack(trackInfo.SSRC)
			p.trackEventsCh <- TrackEvent{
				TrackInfo: trackInfo,
				Type:      TrackEventTypeRemove,
			}

			p.wg.Done()

			prometheusWebRTCTracksActive.Dec()
			prometheusWebRTCTracksDuration.Observe(time.Now().Sub(start).Seconds())
		}()

		for {
			pkt, err := track.ReadRTP()
			if err != nil {
				p.log.Printf("[%s] Remote track has ended: %d: %s", p.clientID, trackInfo.SSRC, err)
				return
			}
			prometheusRTPPacketsReceived.Inc()
			prometheusRTPPacketsReceivedBytes.Add(float64(pkt.MarshalSize()))
			p.rtpLog.Printf("[%s] ReadRTP: %s", p.clientID, pkt)
			p.rtpCh <- pkt
		}
	}()
}

func (p *Connector) ClientID() string {
	return p.clientID
}

func (p *Connector) WriteRTP(packet *rtp.Packet) (bytes int, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	pta, ok := p.localTracks[packet.SSRC]
	if !ok {
		return 0, fmt.Errorf("Track not found: %d", packet.SSRC)
	}
	if err != nil {
		return 0, err
	}
	err = pta.track.WriteRTP(packet)
	if err == io.ErrClosedPipe {
		// ErrClosedPipe means we don't have any subscribers, this is ok if no peers have connected yet
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	return packet.MarshalSize(), nil
}
