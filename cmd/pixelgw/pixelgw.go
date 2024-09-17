package main

import (
	"log"
	"net/http"

	"github.com/joe714/pixelgw/internal/api"
	"github.com/joe714/pixelgw/internal/durable"
	"github.com/joe714/pixelgw/internal/hub"
	"tidbyt.dev/pixlet/runtime"
)

func main() {
	runtime.InitCache(runtime.NewInMemoryCache())
	fs := http.FileServer(http.Dir("./static"))

	store, err := durable.NewStore()
	if err != nil {
		log.Fatal(err)
	}

	hub := hub.NewHub(store)

	svr := api.NewServer(hub, store)

	root := http.NewServeMux()

	root.Handle("/", fs)
	root.HandleFunc("/ws", hub.GetWsHandler())
	hdlr := api.NewStrictHandlerWithOptions(svr, nil, api.ServerOptions())
	api.HandlerFromMuxWithBaseURL(hdlr, root, "/api")

	s := &http.Server{
		Handler: root,
		Addr:    "0.0.0.0:8080",
	}

	log.Fatal(s.ListenAndServe())
}
