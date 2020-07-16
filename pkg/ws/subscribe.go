package ws

import (
	"log"

	"pion-conference/pkg/models/api"
	mws "pion-conference/pkg/models/ws"
	"pion-conference/pkg/webrtc"
)

type (
	Subscription struct {
		ClientID string
		Room     string
		Messages <-chan mws.Message

		WsRoomCtrl     *RoomController
		WebRtcRoomCtrl *webrtc.RoomController
	}

	Subscribe struct {
		wsRooms     *RoomsService
		webrtcRooms *webrtc.RoomsService
	}

	subscribeContext struct {
		msg      chan mws.Message
		client   *mws.Client
		roomCtrl *RoomController
	}
)

func NewSubscribe(wsRoomsSvc *RoomsService, webrtcRoomsSvc *webrtc.RoomsService) Subscribe {
	return Subscribe{
		wsRooms:     wsRoomsSvc,
		webrtcRooms: webrtcRoomsSvc,
	}
}

func (s *Subscribe) Subscribe(enter api.WsRoomEnter) (*Subscription, error) {
	log.Printf("[%s] New websocket connection - enter: %s", enter.ClientId, enter.RoomId)
	msg := make(chan mws.Message)

	wsRoomCtrl := s.wsRooms.SetRoomController(enter.RoomId)
	webrtcRoomCtrl := s.webrtcRooms.SetRoomController(enter.RoomId)

	client := mws.NewClientWithID(enter.Conn, enter.ClientId)

	wsRoomCtrl.Add(client)

	ctx := subscribeContext{
		msg:      msg,
		client:   &client,
		roomCtrl: wsRoomCtrl,
	}
	go s.listenWS(ctx)

	return &Subscription{
		WsRoomCtrl:     wsRoomCtrl,
		WebRtcRoomCtrl: webrtcRoomCtrl,
		ClientID:       enter.ClientId,
		Room:           enter.RoomId,
		Messages:       msg,
	}, nil
}

func (s *Subscribe) listenWS(ctx subscribeContext) {
	defer s.stopListen(ctx.client, ctx.roomCtrl)

	for message := range ctx.client.Listen() {
		ctx.msg <- message
	}
	close(ctx.msg)
}

func (s *Subscribe) stopListen(client *mws.Client, ctrl *RoomController) {
	log.Printf("[%s] RoomController.Remove from room", client.ID())
	ctrl.Remove(client.ID())

	if err := s.wsRooms.DeleteRoomController(ctrl.Room()); err != nil {
		log.Printf("RoomsService.DeleteRoomController %s error", ctrl.Room(), err)
	}

	if err := client.Close(); err != nil {
		log.Printf("ws.Client %s close connection  error", client.Close(), err)
	}
}
