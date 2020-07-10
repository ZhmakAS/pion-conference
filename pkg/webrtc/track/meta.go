package track

import (
	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v3"
)

type TrackEventType uint8

const (
	EventTypeAdd TrackEventType = iota + 1
	EventTypeRemove
)

type (
	Event struct {
		Track      *webrtc.Track
		Type       TrackEventType
		RTCPStream chan rtcp.Packet
	}

	Info struct {
		PayloadType uint8
		SSRC        uint32
		ID          string
		Label       string
		Kind        webrtc.RTPCodecType
		Mid         string
	}

	Metadata struct {
		Mid      string `json:"mid"`
		UserID   string `json:"userId"`
		StreamID string `json:"streamId"`
		Kind     string `json:"kind"`
	}
)

func NewTrackInfoFromWebRTC(track *webrtc.Track) Info {
	return Info{
		SSRC:        track.SSRC(),
		PayloadType: track.PayloadType(),
		ID:          track.ID(),
		Label:       track.Label(),
		Kind:        track.Kind(),
	}
}

func NewTrackMetadataFromTrack(local Info, userId string) Metadata {
	return Metadata{
		Kind:     local.Kind.String(),
		Mid:      local.Mid,
		StreamID: local.Label,
		UserID:   userId,
	}
}
