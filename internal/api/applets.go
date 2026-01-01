//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=cfg.yaml ../../pixelgw.yaml

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"slices"

	"github.com/joe714/pixelgw/internal/locations"
	"tidbyt.dev/pixlet/encode"
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

func (s *Server) RenderApplet(ctx context.Context, request RenderAppletRequestObject) (RenderAppletResponseObject, error) {
	m := s.hub.Catalog.FindManifest(request.Id)
	if m == nil {
		return RenderApplet404JSONResponse{Code: 404, Message: fmt.Sprintf("applet \"%v\" not found", request.Id)}, nil
	}

	log.Printf("Render Applet %v", m.ID)
	applet, err := runtime.NewAppletFromFS(m.ID, m.Bundle)
	if err != nil {
		return nil, fmt.Errorf("failed to load applet: %w", err)
	}

	// Parse config from query parameter
	config := make(map[string]string)
	if request.Params.Config != nil && *request.Params.Config != "" {
		if err := json.Unmarshal([]byte(*request.Params.Config), &config); err != nil {
			return nil, fmt.Errorf("invalid config JSON: %w", err)
		}
	}

	// Expand location configs from place_id to full location JSON
	if applet.Schema != nil {
		fields := make([]locations.SchemaField, 0, len(applet.Schema.Fields))
		for _, f := range applet.Schema.Fields {
			fields = append(fields, locations.SchemaField{
				ID:   f.ID,
				Type: f.Type,
			})
		}
		config = locations.ExpandLocationConfigs(fields, config)
	}

	// Run the applet with config
	roots, err := applet.RunWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("applet execution failed: %w", err)
	}

	if roots == nil || len(roots) < 1 {
		return nil, fmt.Errorf("applet produced no output")
	}

	// Encode to WebP
	screens := encode.ScreensFromRoots(roots)
	img, err := screens.EncodeWebP(15000)
	if err != nil {
		return nil, fmt.Errorf("encoding failed: %w", err)
	}

	return RenderApplet200ImagewebpResponse{
		Body:          bytes.NewReader(img),
		ContentLength: int64(len(img)),
	}, nil
}
