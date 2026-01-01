//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=cfg.yaml ../../pixelgw.yaml

package api

import (
	"context"
	"encoding/json"
	ne "errors"
	"log"
	"net/http"
	"strings"

	strictnethttp "github.com/oapi-codegen/runtime/strictmiddleware/nethttp"

	"github.com/joe714/pixelgw/internal/durable"
	"github.com/joe714/pixelgw/internal/errors"
	"github.com/joe714/pixelgw/internal/hub"
)

type contextKey string

var (
	contextKeyAccept = contextKey("Accept")
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
	for key, val := range statusCodes {
		if ne.Is(err, key) {
			return val
		}
	}
	return http.StatusInternalServerError
}

func WantWebp(ctx context.Context) bool {
	if v, ok := ctx.Value(contextKeyAccept).(string); ok {
		r := strings.Contains(v, "image/webp") || strings.Contains(v, "image/*")
		log.Printf("Webp %v (%v)", r, v)
		return r
	}
	return false
}

func AcceptMiddleware(f strictnethttp.StrictHTTPHandlerFunc, operationID string) strictnethttp.StrictHTTPHandlerFunc {
	return strictnethttp.StrictHTTPHandlerFunc(
		func(ctx context.Context,
			w http.ResponseWriter,
			r *http.Request,
			request interface{}) (response interface{}, err error) {
			v := r.Header.Get("Accept")
			if v != "" {
				ctx = context.WithValue(ctx, contextKeyAccept, strings.ToLower(v))
			}
			return f(ctx, w, r, request)
		})
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
