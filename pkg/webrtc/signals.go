package webrtc

import (
	"fmt"
	"github.com/pion/webrtc/v3"
)

type (
	Renegotiate struct {
		Renegotiate bool `json:"renegotiate"`
	}

	Candidate struct {
		Candidate webrtc.ICECandidateInit `json:"candidate"`
	}

	Payload struct {
		Renegotiate bool        `json:"renegotiate"`
		ClientId    string      `json:"clientId"`
		Signal      interface{} `json:"signal,omitempty"`
	}
)

func NewSDPPayload(sessionDescription webrtc.SessionDescription, clientId string) Payload {
	return Payload{
		Renegotiate: false,
		ClientId:    clientId,
		Signal:      sessionDescription,
	}
}

func NewSDPPayloadWithRenegotiate(sessionDescription webrtc.SessionDescription, clientId string) Payload {
	return Payload{
		Renegotiate: true,
		ClientId:    clientId,
		Signal:      sessionDescription,
	}
}

func NewRenegotiatePayload(clientId string) Payload {
	return Payload{
		ClientId:    clientId,
		Renegotiate: true,
	}
}

func NewSignalPayloadFromMap(rawPayload map[string]interface{}) (*Payload, error) {
	signal, ok := rawPayload["signal"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("no signal property in payload: %#v", rawPayload)
	}

	userID, ok := rawPayload["clientId"].(string)
	if !ok {
		return nil, fmt.Errorf("No userId property in payload: %#v", rawPayload)
	}

	isRenegotiate, ok := rawPayload["renegotiate"].(bool)
	if !ok {
		return nil, fmt.Errorf("no renegotiate property in payload : %#v", rawPayload)
	}

	sdpType, ok := signal["type"]
	if !ok {
		return nil, fmt.Errorf("unexpected signal message: %#v", rawPayload)
	}

	sdpSignal, err := newSDP(sdpType, signal)
	if err != nil {
		return nil, err
	}

	return &Payload{
		Renegotiate: isRenegotiate,
		Signal:      sdpSignal,
		ClientId:    userID,
	}, nil
}

func newSDP(sdpType interface{}, signal map[string]interface{}) (desc webrtc.SessionDescription, err error) {
	sdpTypeString, ok := sdpType.(string)
	if !ok {
		err = fmt.Errorf("expected signal.type to be string: %#v", signal)
		return
	}

	sdp, ok := signal["sdp"]
	if !ok {
		err = fmt.Errorf("expected signal.sdp: %#v", signal)
		return
	}

	sdpString, ok := sdp.(string)
	if !ok {
		err = fmt.Errorf("expected signal.sdp to be string: %#v", signal)
		return
	}
	desc.SDP = sdpString

	switch sdpTypeString {
	case "offer":
		desc.Type = webrtc.SDPTypeOffer
	case "answer":
		desc.Type = webrtc.SDPTypeAnswer
	default:
		err = fmt.Errorf("Unknown sdp type: %s", sdpString)
		return
	}

	return
}
