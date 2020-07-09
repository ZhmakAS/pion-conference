package webrtc

import "github.com/pion/webrtc/v3"

type TrackEventType uint8

const (
	TrackEventTypeAdd TrackEventType = iota + 1
	TrackEventTypeRemove
)

type (
	TrackEvent struct {
		TrackInfo
		Type TrackEventType
	}

	TrackInfo struct {
		PayloadType uint8
		SSRC        uint32
		ID          string
		Label       string
		Kind        webrtc.RTPCodecType
		Mid         string
	}

	localTrackInfo struct {
		trackInfo   TrackInfo
		transceiver *webrtc.RTPTransceiver
		sender      *webrtc.RTPSender
		track       *webrtc.Track
	}

	remoteTrackInfo struct {
		trackInfo   TrackInfo
		transceiver *webrtc.RTPTransceiver
		receiver    *webrtc.RTPReceiver
		track       *webrtc.Track
	}
)

func NewTrackInfoFromWebRTC(track *webrtc.Track) TrackInfo {
	return TrackInfo{
		SSRC:        track.SSRC(),
		PayloadType: track.PayloadType(),
		ID:          track.ID(),
		Label:       track.Label(),
		Kind:        track.Kind(),
	}
}
