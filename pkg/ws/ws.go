package ws

import (
	"log"
	"pion-conference/pkg/models/api"
	"pion-conference/pkg/models/ws"
	"pion-conference/pkg/webrtc/track"
)

type (
	Subscription struct {
		Controller RoomControllerService
		ClientID   string
		Room       string
		Messages   <-chan ws.Message
	}

	Service struct {
		rooms *RoomsService
	}

	MetadataPayload struct {
		UserID   string           `json:"userId"`
		Metadata []track.Metadata `json:"metadata"`
	}

	wsContext struct {
		msg      chan ws.Message
		client   *ws.Client
		roomCtrl *RoomControllerService
	}
)

func NewService(roomsService *RoomsService) Service {
	return Service{
		rooms: roomsService,
	}
}

func (s *Service) Subscribe(enter api.WsRoomEnter) (*Subscription, error) {
	log.Printf("[%s] New websocket connection - enter: %s", enter.ClientId, enter.RoomId)
	msg := make(chan ws.Message)

	roomCtrl := s.rooms.SetRoomController(enter.RoomId)
	client := ws.NewClientWithID(enter.Conn, enter.ClientId)

	roomCtrl.Add(client)

	ctx := wsContext{
		msg:      msg,
		client:   &client,
		roomCtrl: &roomCtrl,
	}
	go s.listenWS(ctx)

	return &Subscription{
		Controller: roomCtrl,
		ClientID:   enter.ClientId,
		Room:       enter.RoomId,
		Messages:   msg,
	}, nil
}

func (s *Service) listenWS(ctx wsContext) {
	defer s.stopListen(ctx.client, ctx.roomCtrl)

	for message := range ctx.client.Listen() {
		ctx.msg <- message
	}
	close(ctx.msg)
}

func (s *Service) stopListen(client *ws.Client, ctrl *RoomControllerService) {
	log.Printf("[%s] RoomController.Remove from room", client.ID())
	ctrl.Remove(client.ID())

	if err := s.rooms.DeleteRoomController(ctrl.Room()); err != nil {
		log.Printf("RoomsService.DeleteRoomController %s error", ctrl.Room(), err)
	}

	if err := client.Close(); err != nil {
		log.Printf("ws.Client %s close connection  error", client.Close(), err)
	}
}
