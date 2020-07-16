package handlers

import (
	"github.com/go-chi/chi"
	"log"
	"net/http"
	"pion-conference/pkg/models/api"
	"pion-conference/pkg/webrtc"
	"pion-conference/pkg/ws"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type WsHandler struct{}

func (h WsHandler) CreateRoom(w http.ResponseWriter, r *http.Request) {
	roomId := chi.URLParam(r, "room_id")
	if roomId == "" {
		panic("empty id param")
	}

	clientId := chi.URLParam(r, "user_id")
	if clientId == "" {
		panic("empty id param")
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	wsSubcribe := ws.NewSubscribe(ws.GetRoomsService(), webrtc.GetRoomsService())

	apiRoom := api.WsRoomEnter{
		RoomId:   roomId,
		ClientId: clientId,
		Conn:     conn,
	}

	subscription, err := wsSubcribe.Subscribe(apiRoom)
	if err != nil {
		log.Print("ws.Service Subscribe error")
		return
	}

	socketHandler := ws.NewSocketHandler(subscription)

	for message := range subscription.Messages {
		err := socketHandler.HandleMessage(message)
		if err != nil {
			log.Printf("[%s] Error handling websocket message: %s", subscription.ClientID, err)
		}
	}
}
