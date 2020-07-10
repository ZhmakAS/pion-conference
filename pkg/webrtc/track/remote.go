package track

import "github.com/pion/webrtc/v3"

type RemoteInfo struct {
	info        Info
	transceiver *webrtc.RTPTransceiver
	receiver    *webrtc.RTPReceiver
	track       *webrtc.Track
}

func NewRemoteInfo(trackInfo Info, transceiver *webrtc.RTPTransceiver, receiver *webrtc.RTPReceiver, track *webrtc.Track) RemoteInfo {
	return RemoteInfo{
		track:       track,
		info:        trackInfo,
		transceiver: transceiver,
		receiver:    receiver,
	}
}

//Getters for RemoteInfo
func (r RemoteInfo) Info() Info {
	return r.info
}

func (r RemoteInfo) Track() *webrtc.Track {
	return r.track
}

func (r RemoteInfo) Transceiver() *webrtc.RTPTransceiver {
	return r.transceiver
}
