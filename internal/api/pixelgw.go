//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=cfg.yaml ../../pixelgw.yaml

package api

import (
	"encoding/json"
	"net/http"

	"github.com/joe714/pixelgw/internal/durable"
	"github.com/joe714/pixelgw/internal/errors"
	"github.com/joe714/pixelgw/internal/hub"
)

var statusCodes = map[error]int{
	errors.ChannelExists:      http.StatusConflict,
	errors.ChannelNotFound:    http.StatusNotFound,
	errors.AppIndexOutOfRange: http.StatusBadRequest,
}

type Server struct {
	hub   *hub.Hub
	store *durable.Store
}

func NewServer(hub *hub.Hub, store *durable.Store) *Server {
	return &Server{hub: hub, store: store}
}

func RenderError(err error) Error {
	return Error{
		Code:    errors.Code(err),
		Message: err.Error(),
	}
}

func StatusCode(err error) int {
	if val, ok := statusCodes[err]; ok {
		return val
	}
	return http.StatusInternalServerError
}

func ServerOptions() StrictHTTPServerOptions {
	return StrictHTTPServerOptions{
		RequestErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			http.Error(w, err.Error(), http.StatusBadRequest)
		},
		ResponseErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(StatusCode(err))
			json.NewEncoder(w).Encode(RenderError(err))
		},
	}
}
