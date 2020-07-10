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
		UserID string      `json:"userId"`
		Signal interface{} `json:"signal"`
	}
)

func NewPayloadSDP(userID string, sessionDescription webrtc.SessionDescription) Payload {
	return Payload{
		UserID: userID,
		Signal: sessionDescription,
	}
}

func NewPayloadRenegotiate(userID string) Payload {
	return Payload{
		UserID: userID,
		Signal: Renegotiate{
			Renegotiate: true,
		},
	}
}

func NewPayloadFromMap(payload map[string]interface{}) (p Payload, err error) {
	userID, ok := payload["userId"].(string)
	if !ok {
		err = fmt.Errorf("No userId property in payload: %#v", payload)
		return
	}
	signal, ok := payload["signal"].(map[string]interface{})
	if !ok {
		err = fmt.Errorf("No signal property in payload: %#v", payload)
		return
	}

	var value interface{}

	if candidate, ok := signal["candidate"]; ok {
		value, err = newCandidate(candidate)
	} else if _, ok := signal["renegotiate"]; ok {
		value = newRenegotiate()
	} else if sdpType, ok := signal["type"]; ok {
		value, err = newSDP(sdpType, signal)
	} else {
		err = fmt.Errorf("Unexpected signal message: %#v", payload)
		return
	}

	if err != nil {
		return
	}

	p.UserID = userID
	p.Signal = value
	return
}

func newCandidate(candidate interface{}) (c Candidate, err error) {
	candidateMap, ok := candidate.(map[string]interface{})
	if !ok {
		err = fmt.Errorf("Expected candidate to be a map: %#v", candidate)
		return
	}

	candidateValue, ok := candidateMap["candidate"]
	if !ok {
		err = fmt.Errorf("Expected candidate.candidate %#v", candidate)
	}

	candidateString, ok := candidateValue.(string)
	if !ok {
		err = fmt.Errorf("Expected candidate.candidate to be a string: %#v", candidate)
		return
	}
	sdpMLineIndexValue, ok := candidateMap["sdpMLineIndex"]
	if !ok {
		err = fmt.Errorf("Expected candidate.sdpMLineIndex to exist: %#v", sdpMLineIndexValue)
		return
	}
	sdpMLineIndex, ok := sdpMLineIndexValue.(float64)
	if !ok {
		err = fmt.Errorf("Expected candidate.sdpMLineIndex be float64: %T", sdpMLineIndexValue)
		return
	}

	sdpMid, ok := candidateMap["sdpMid"].(string)
	var sdpMidPtr *string
	if ok {
		sdpMidPtr = &sdpMid
	}

	sdpMLineIndexUint16 := uint16(sdpMLineIndex)
	c.Candidate.Candidate = candidateString
	c.Candidate.SDPMLineIndex = &sdpMLineIndexUint16
	c.Candidate.SDPMid = sdpMidPtr

	return
}

func newRenegotiate() Renegotiate {
	return Renegotiate{
		Renegotiate: true,
	}
}

func newSDP(sdpType interface{}, signal map[string]interface{}) (s webrtc.SessionDescription, err error) {
	sdpTypeString, ok := sdpType.(string)
	if !ok {
		err = fmt.Errorf("Expected signal.type to be string: %#v", signal)
		return
	}

	sdp, ok := signal["sdp"]
	if !ok {
		err = fmt.Errorf("Expected signal.sdp: %#v", signal)
	}

	sdpString, ok := sdp.(string)
	if !ok {
		err = fmt.Errorf("Expected signal.sdp to be string: %#v", signal)
		return
	}
	s.SDP = sdpString

	switch sdpTypeString {
	case "offer":
		s.Type = webrtc.SDPTypeOffer
	case "answer":
		s.Type = webrtc.SDPTypeAnswer
	case "pranswer":
		err = fmt.Errorf("Handling of pranswer signal implemented")
	case "rollback":
		err = fmt.Errorf("Handling of rollback signal not implemented")
	default:
		err = fmt.Errorf("Unknown sdp type: %s", sdpString)
	}

	return
}
