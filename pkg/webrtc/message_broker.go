package webrtc

import (
	"fmt"
	"github.com/pion/webrtc/v3"
)

type MessageBroker struct {
	clientID string

	dataChannel *webrtc.DataChannel

	messageStream  chan webrtc.DataChannelMessage
	peerConnection *webrtc.PeerConnection
}

func NewMessageBroker(peerConnection *webrtc.PeerConnection, clientId string) *MessageBroker {
	broker := &MessageBroker{
		messageStream:  make(chan webrtc.DataChannelMessage),
		peerConnection: peerConnection,
		clientID:       clientId,
	}

	peerConnection.OnDataChannel(broker.onDataChannelHandler)

	return broker
}

func (mb *MessageBroker) onDataChannelHandler(dataChannel *webrtc.DataChannel) {
	dataChannel.OnMessage(mb.onMessageHandler)
	mb.dataChannel = dataChannel
}

func (mb *MessageBroker) MessagesChannel() <-chan webrtc.DataChannelMessage {
	return mb.messageStream
}

func (mb *MessageBroker) onMessageHandler(msg webrtc.DataChannelMessage) {
	mb.messageStream <- msg
}

func (mb *MessageBroker) SendText(message string) error {
	if mb.dataChannel == nil {
		return fmt.Errorf("[%s] No data channel", mb.clientID)
	}

	return mb.dataChannel.SendText(message)
}

func (mb *MessageBroker) Send(message []byte) (err error) {
	if mb.dataChannel == nil {
		return fmt.Errorf("[%s] No data channel", mb.clientID)
	}

	return mb.dataChannel.Send(message)
}

//func (d *DataTransceiver) Close() {
//	d.dataChanOnce.Do(func() {
//		close(d.closeChannel)
//
//		d.mu.Lock()
//		defer d.mu.Unlock()
//
//		d.dataChanClosed = true
//		close(d.messagesChan)
//	})
//}
