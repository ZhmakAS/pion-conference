package handlers

import (
	"github.com/go-chi/chi"
	"log"
	"net/http"
	"pion-conference/pkg/models/api"
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
	wsService := ws.NewService(ws.GetRoomsService())

	apiRoom := api.WsRoomEnter{
		RoomId:   roomId,
		ClientId: clientId,
		Conn:     conn,
	}

	subcription, err := wsService.Subscribe(apiRoom)
	if err != nil {
		log.Print("ws.Service Subscribe error")
		return
	}

	socketHandler := ws.NewSocketHandler(subcription, apiRoom.RoomId, apiRoom.ClientId, subcription.Controller)

	for message := range subcription.Messages {
		err := socketHandler.HandleMessage(message)
		if err != nil {
			log.Printf("[%s] Error handling websocket message: %s", subcription.ClientID, err)
		}
	}
}
