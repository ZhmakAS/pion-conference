package api

import "github.com/gorilla/websocket"

type WsRoomEnter struct {
	ClientId string
	RoomId   string
	Conn     *websocket.Conn
}
