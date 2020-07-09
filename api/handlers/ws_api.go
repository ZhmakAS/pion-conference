package handlers

import (
	"log"
	"net/http"
	"pion-conference/pkg/models/api "

	"pion-conference/pkg/ws"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type WsHandler struct{}

func (h WsHandler) CreateRoom(w http.ResponseWriter, r *http.Request) {
	//TODO: Parse Url Params and validate

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	wsService := ws.NewService(ws.GetRoomsService())

	apiRoom := api.WsRoomEnter{
		Conn: conn,
	}
	subcription, err := wsService.Subscribe(apiRoom)
	if err != nil {
		log.Print("ws.Service Subscribe error")
		return
	}

}
