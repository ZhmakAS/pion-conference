package ws

import (
	"fmt"
	"sync"

	"pion-conference/pkg/models/ws"
)

type SocketHandler struct {
	mux sync.Mutex

	subscription Subscription
}

func NewSocketHandler(subsc Subscription) SocketHandler {
	return SocketHandler{
		subscription: subsc,
	}
}

func (sh *SocketHandler) Handle(message ws.Message) error {
	sh.mux.Lock()
	defer sh.mux.Unlock()

	switch message.Type {
	case "ready":
		return sh.handleReady(message)
	}

	return fmt.Errorf("Unhandled event: %s", message.Type)
}

func (sh *SocketHandler) handleReady(message ws.Message) error {
	payload, ok := message.Payload.(map[string]interface{})
	if !ok {
		return fmt.Errorf("Ready message payload is of wrong type: %T", message.Payload)
	}

	return nil
}
