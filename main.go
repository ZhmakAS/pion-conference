package main

import (
	"fmt"
	"net/http"
	"pion-conference/api"
	"pion-conference/pkg/ws"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	//should be initialized once at the start of the service
	ws.InitRoomsService()

	fmt.Print("Server is running on:3000")
	http.ListenAndServe(":3000", api.GetRouter())
}
