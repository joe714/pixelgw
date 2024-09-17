package api

import (
	"context"
	"net/http"

	"github.com/joe714/pixelgw/internal/durable"
)

func (s *Server) GetDevices(ctx context.Context, request GetDevicesRequestObject) (GetDevicesResponseObject, error) {
	devs, err := s.store.GetAllDevices(ctx)
	if err != nil {
		return GetDevicesdefaultJSONResponse{
				Body:       RenderError(err),
				StatusCode: StatusCode(err),
			},
			nil
	}
	resp := make([]DeviceSummary, 0, len(devs))
	for _, d := range devs {
		resp = append(resp, DeviceSummary{
			UUID: &d.UUID,
			Name: &d.Name,
			Channel: &ChannelRef{
				UUID: &d.ChannelUUID,
				Name: d.ChannelName,
			},
		})
	}
	return GetDevices200JSONResponse(resp), nil
}

func (s *Server) GetDeviceByUUID(ctx context.Context, request GetDeviceByUUIDRequestObject) (GetDeviceByUUIDResponseObject, error) {
	d, err := s.store.GetDeviceByUUID(ctx, request.UUID)
	if err != nil {
		return GetDeviceByUUIDdefaultJSONResponse{
				Body:       RenderError(err),
				StatusCode: StatusCode(err),
			},
			nil
	}
	resp := DeviceSummary{
		UUID: &d.UUID,
		Name: &d.Name,
		Channel: &ChannelRef{
			UUID: &d.ChannelUUID,
			Name: d.ChannelName,
		},
	}
	return GetDeviceByUUID200JSONResponse(resp), nil
}

func (s *Server) PatchDevice(ctx context.Context, request PatchDeviceRequestObject) (PatchDeviceResponseObject, error) {
	if request.Body.Name == nil && request.Body.Channel == nil {
		return PatchDevicedefaultJSONResponse{
				Body: Error{
					Code:    http.StatusBadRequest,
					Message: "No attributes provided",
				},
				StatusCode: http.StatusBadRequest,
			},
			nil
	}

	d, err := s.store.GetDeviceByUUID(ctx, request.UUID)
	if err != nil {
		return PatchDevicedefaultJSONResponse{
				Body:       RenderError(err),
				StatusCode: StatusCode(err),
			},
			nil
	}

	if request.Body.Name != nil {
		d.Name = *request.Body.Name
	}

	subscribe := false
	if request.Body.Channel != nil {
		var ch *durable.Channel
		if request.Body.Channel.UUID != nil {
			ch, err = s.store.GetChannelByUUID(ctx, *request.Body.Channel.UUID)
			if err != nil {
				return PatchDevicedefaultJSONResponse{
						Body:       RenderError(err),
						StatusCode: StatusCode(err),
					},
					nil
			}
		}
		if request.Body.Channel.Name != nil {
			if ch == nil {
				ch, err = s.store.GetChannelByName(ctx, *request.Body.Channel.Name)
				if err != nil {
					return PatchDevicedefaultJSONResponse{
							Body:       RenderError(err),
							StatusCode: StatusCode(err),
						},
						nil
				}
			} else if ch.Name != *request.Body.Channel.Name {
				return PatchDevicedefaultJSONResponse{
						Body: Error{
							Code:    http.StatusBadRequest,
							Message: "channel.uuid and channel.name must refer to the same object",
						},
						StatusCode: http.StatusBadRequest,
					},
					nil
			}
		}
		if ch != nil && d.ChannelUUID != ch.UUID {
			subscribe = true
			d.ChannelUUID = ch.UUID
		}
	}
	err = s.store.ModifyDevice(ctx, d)
	if err != nil {
		return PatchDevicedefaultJSONResponse{
				Body:       RenderError(err),
				StatusCode: StatusCode(err),
			},
			nil
	}
	if subscribe {
		s.hub.SubscribeDevice(d.UUID, d.ChannelUUID)
	}
	return PatchDevice200Response{}, nil
}
