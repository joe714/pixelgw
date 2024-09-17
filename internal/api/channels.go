//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen --config=cfg.yaml ../../pixelgw.yaml

package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/joe714/pixelgw/internal/durable"
)

func (s *Server) GetChannels(ctx context.Context, request GetChannelsRequestObject) (GetChannelsResponseObject, error) {
	ch, err := s.store.GetAllChannels(ctx)
	if err != nil {
		return GetChannelsdefaultJSONResponse{
				Body:       RenderError(err),
				StatusCode: StatusCode(err),
			},
			nil
	}

	resp := make([]ChannelSummary, 0, len(ch))
	for _, c := range ch {
		resp = append(resp, ChannelSummary{
			UUID:    &c.UUID,
			Name:    c.Name,
			Comment: c.Comment,
		})
	}
	return GetChannels200JSONResponse(resp), nil
}

func (s *Server) CreateChannel(ctx context.Context, request CreateChannelRequestObject) (CreateChannelResponseObject, error) {
	ch, err := s.store.CreateChannel(ctx, request.Body.Name, request.Body.Comment)
	if err != nil {
		return CreateChanneldefaultJSONResponse{
				Body:       RenderError(err),
				StatusCode: StatusCode(err),
			},
			nil
	}
	return CreateChannel201JSONResponse(
			ChannelDetail{
				UUID:    &ch.UUID,
				Name:    ch.Name,
				Comment: ch.Comment,
			}),
		nil
}

func (s *Server) FindChannelByUUID(ctx context.Context, request FindChannelByUUIDRequestObject) (FindChannelByUUIDResponseObject, error) {
	ch, err := s.store.GetChannelByUUID(ctx, request.UUID)
	if err != nil {
		return FindChannelByUUIDdefaultJSONResponse{
				Body:       RenderError(err),
				StatusCode: StatusCode(err),
			},
			nil
	}

	cd := ChannelDetail{
		UUID:    &ch.UUID,
		Name:    ch.Name,
		Comment: ch.Comment,
	}

	apps := make([]AppInstanceDetail, 0, len(ch.Applets))
	for _, i := range ch.Applets {
		a := AppInstanceDetail{
			UUID:  &i.UUID,
			Idx:   &i.Idx,
			AppID: i.AppID,
		}
		if i.Config != nil {
			a.Config = json.RawMessage(*i.Config)
		}
		apps = append(apps, a)
	}
	if len(apps) > 0 {
		cd.Applets = &apps
	}

	subs := make([]DeviceRef, 0, len(ch.Subscribers))
	for _, i := range ch.Subscribers {
		subs = append(subs, DeviceRef{
			UUID: &i.UUID,
			Name: &i.Name,
		})
	}
	if len(subs) > 0 {
		cd.Subscribers = &subs
	}
	return FindChannelByUUID200JSONResponse(cd), nil
}

func (s *Server) CreateChannelApplet(ctx context.Context, request CreateChannelAppletRequestObject) (CreateChannelAppletResponseObject, error) {
	app := durable.ChannelApplet{
		Idx:   -1,
		AppID: request.Body.AppID,
	}

	if request.Body.Idx != nil {
		app.Idx = *request.Body.Idx
	}

	m := s.hub.Catalog.FindManifest(app.AppID)
	if m == nil {
		return CreateChannelApplet400JSONResponse{
				Code:    http.StatusBadRequest,
				Message: "Applet not found",
			},
			nil
	}

	if request.Body.Config != nil {
		cfg := string(request.Body.Config)
		app.Config = &cfg
	}

	// TODO: Verify schema
	err := s.store.CreateChannelApplet(ctx, request.ChannelUUID, &app)
	if err != nil {
		return CreateChannelAppletdefaultJSONResponse{
				Body:       RenderError(err),
				StatusCode: StatusCode(err),
			},
			nil
	}

	s.hub.ReloadApplets(request.ChannelUUID, app.UUID)
	return CreateChannelApplet201JSONResponse{
			AppID:  app.AppID,
			Config: request.Body.Config,
			Idx:    &app.Idx,
			UUID:   &app.UUID,
		},
		nil
}

func (s *Server) DeleteChannelApplet(ctx context.Context, request DeleteChannelAppletRequestObject) (DeleteChannelAppletResponseObject, error) {
	err := s.store.DeleteChannelApplet(ctx, request.ChannelUUID, request.AppletUUID)
	if err != nil {
		return DeleteChannelAppletdefaultJSONResponse{
				Body:       RenderError(err),
				StatusCode: StatusCode(err),
			},
			nil
	}
	s.hub.ReloadApplets(request.ChannelUUID, uuid.Nil)
	return DeleteChannelApplet200Response{}, nil
}

func (s *Server) PatchChannelApplet(ctx context.Context, request PatchChannelAppletRequestObject) (PatchChannelAppletResponseObject, error) {
	idx := request.Body.Idx
	var cfg *string
	if request.Body.Config != nil {
		tmp := string(request.Body.Config)
		cfg = &tmp
	}
	err := s.store.ModifyChannelApplet(ctx, request.ChannelUUID, request.AppletUUID, idx, cfg)
	if err != nil {
		return PatchChannelAppletdefaultJSONResponse{
				Body:       RenderError(err),
				StatusCode: StatusCode(err),
			},
			nil
	}
	s.hub.ReloadApplets(request.ChannelUUID, request.AppletUUID)
	return PatchChannelApplet200Response{}, nil
}
