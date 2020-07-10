package track

import "github.com/pion/webrtc/v3"

type LocalInfo struct {
	info        Info
	transceiver *webrtc.RTPTransceiver
	sender      *webrtc.RTPSender
	track       *webrtc.Track
}

func NewLocalInfo(trackInfo Info, transceiver *webrtc.RTPTransceiver, sender *webrtc.RTPSender, track *webrtc.Track) LocalInfo {
	return LocalInfo{
		track:       track,
		info:        trackInfo,
		transceiver: transceiver,
		sender:      sender,
	}
}

//Getters for LocalInfo
func (r LocalInfo) Info() Info {
	return r.info
}

func (r LocalInfo) Track() *webrtc.Track {
	return r.track
}

func (r LocalInfo) Transceiver() *webrtc.RTPTransceiver {
	return r.transceiver
}

func (r LocalInfo) Sender() *webrtc.RTPSender {
	return r.sender
}
