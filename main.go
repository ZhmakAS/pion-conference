package main

import (
	"fmt"
	"net/http"
	"pion-conference/api/handlers"
	"pion-conference/pkg/ws"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

func main() {
	r := chi.NewRouter()

	wsHandlers := handlers.WsHandler{}
	// A good base middleware stack
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

	r.Route("/ws", func(r chi.Router) {
		r.Get("/{room_id}/{user_id}", wsHandlers.CreateRoom)
	})

	//should be initialized once at the start of the service
	ws.InitRoomsService()

	fmt.Print("Server is running on:3000")
	http.ListenAndServe(":3000", r)
}
