//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=cfg.yaml ../../pixelgw.yaml

package api

import (
	"context"
	"fmt"
	"log"
	"slices"

	"tidbyt.dev/pixlet/runtime"
)

func (s *Server) GetApplets(ctx context.Context, request GetAppletsRequestObject) (GetAppletsResponseObject, error) {
	var resp []App

	keys := make([]string, 0)
	for k, _ := range s.hub.Catalog.Manifests {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, k := range keys {
		m := s.hub.Catalog.FindManifest(k)
		a := App{Id: m.ID, Name: m.Name, Summary: m.Summary, Description: m.Desc, Author: m.Author}
		resp = append(resp, a)
	}
	return GetApplets200JSONResponse(resp), nil
}

func (s *Server) GetAppletByID(ctx context.Context, request GetAppletByIDRequestObject) (GetAppletByIDResponseObject, error) {
	m := s.hub.Catalog.FindManifest(request.Id)
	if m == nil {
		return nil, fmt.Errorf("applet \"%v\" not registered", request.Id)
	}
	resp := App{Id: m.ID, Name: m.Name, Summary: m.Summary, Description: m.Desc, Author: m.Author}

	log.Printf("Load Applet %v", m.ID)
	app, err := runtime.NewAppletFromFS(m.ID, m.Bundle)
	if err != nil {
		return nil, err
	}

	resp.Schema = app.Schema
	return GetAppletByID200JSONResponse(resp), nil
}
