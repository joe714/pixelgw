package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/joe714/pixelgw/internal/durable"
)

func parseTime(s *string) *time.Time {
	if s == nil {
		return nil
	}
	t, err := time.Parse(time.RFC3339, *s)
	if err != nil {
		return nil
	}
	return &t
}

func parseDeviceInfo(s *string) *map[string]interface{} {
	if s == nil {
		return nil
	}
	var info map[string]interface{}
	if err := json.Unmarshal([]byte(*s), &info); err != nil {
		return nil
	}
	return &info
}

func (s *Server) GetDevices(ctx context.Context, request GetDevicesRequestObject) (GetDevicesResponseObject, error) {
	devs, err := s.store.GetAllDevices(ctx)
	if err != nil {
		return GetDevicesdefaultJSONResponse{
				Body:       RenderError(err),
				StatusCode: StatusCode(err),
			},
			nil
	}

	// Parse fields parameter
	fields := ParseFields(request.Params.Fields)
	includeDeviceInfo := ShouldInclude(fields, "device-info", false)

	// Build a map of currently connected devices from live sessions
	connectedDevices := make(map[string]string) // device UUID -> current IP
	if s.hub != nil {
		for _, sess := range s.hub.GetSessions() {
			connectedDevices[sess.DeviceUUID.String()] = sess.RemoteAddr
		}
	}

	resp := make([]DeviceSummary, 0, len(devs))
	for _, d := range devs {
		ds := DeviceSummary{
			UUID: &d.UUID,
			Name: &d.Name,
			Channel: &ChannelRef{
				UUID: &d.ChannelUUID,
				Name: d.ChannelName,
			},
			LastIP:             d.LastIP,
			LastConnectTime:    parseTime(d.LastConnectTime),
			LastDisconnectTime: parseTime(d.LastDisconnectTime),
		}

		// Include device-info if requested
		if includeDeviceInfo {
			ds.DeviceInfo = parseDeviceInfo(d.DeviceInfo)
			ds.DeviceInfoUpdated = parseTime(d.DeviceInfoUpdated)
		}

		// Check if device is currently connected
		if currentIP, ok := connectedDevices[d.UUID.String()]; ok {
			connected := true
			ds.Connected = &connected
			ds.CurrentIP = &currentIP
		} else {
			connected := false
			ds.Connected = &connected
		}

		resp = append(resp, ds)
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

	// Parse fields parameter
	fields := ParseFields(request.Params.Fields)
	includeDeviceInfo := ShouldInclude(fields, "device-info", false)

	// Check if device is currently connected
	var currentIP *string
	connected := false
	if s.hub != nil {
		for _, sess := range s.hub.GetSessions() {
			if sess.DeviceUUID == request.UUID {
				connected = true
				currentIP = &sess.RemoteAddr
				break
			}
		}
	}

	resp := DeviceSummary{
		UUID: &d.UUID,
		Name: &d.Name,
		Channel: &ChannelRef{
			UUID: &d.ChannelUUID,
			Name: d.ChannelName,
		},
		Connected:          &connected,
		CurrentIP:          currentIP,
		LastIP:             d.LastIP,
		LastConnectTime:    parseTime(d.LastConnectTime),
		LastDisconnectTime: parseTime(d.LastDisconnectTime),
	}

	// Include device-info if requested
	if includeDeviceInfo {
		resp.DeviceInfo = parseDeviceInfo(d.DeviceInfo)
		resp.DeviceInfoUpdated = parseTime(d.DeviceInfoUpdated)
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
	if subscribe && s.hub != nil {
		s.hub.SubscribeDevice(d.UUID, d.ChannelUUID)
	}
	return PatchDevice200Response{}, nil
}

func (s *Server) DeleteDevice(ctx context.Context, request DeleteDeviceRequestObject) (DeleteDeviceResponseObject, error) {
	// Check if device is currently connected
	if s.hub != nil {
		for _, sess := range s.hub.GetSessions() {
			if sess.DeviceUUID == request.UUID {
				return DeleteDevicedefaultJSONResponse{
						Body: Error{
							Code:    http.StatusConflict,
							Message: "Cannot delete device while it is connected",
						},
						StatusCode: http.StatusConflict,
					},
					nil
			}
		}
	}

	err := s.store.DeleteDevice(ctx, request.UUID)
	if err != nil {
		return DeleteDevicedefaultJSONResponse{
				Body:       RenderError(err),
				StatusCode: StatusCode(err),
			},
			nil
	}
	return DeleteDevice200Response{}, nil
}

func (s *Server) CreateDevice(ctx context.Context, request CreateDeviceRequestObject) (CreateDeviceResponseObject, error) {
	// Validate request
	if request.Body == nil {
		return CreateDevice400JSONResponse{
			Code:    http.StatusBadRequest,
			Message: "Request body is required",
		}, nil
	}

	if request.Body.Name == "" {
		return CreateDevice400JSONResponse{
			Code:    http.StatusBadRequest,
			Message: "Device name is required",
		}, nil
	}

	if request.Body.Channel.UUID == nil {
		return CreateDevice400JSONResponse{
			Code:    http.StatusBadRequest,
			Message: "Channel UUID is required",
		}, nil
	}

	// Verify channel exists
	ch, err := s.store.GetChannelByUUID(ctx, *request.Body.Channel.UUID)
	if err != nil {
		return CreateDevicedefaultJSONResponse{
			Body:       RenderError(err),
			StatusCode: StatusCode(err),
		}, nil
	}

	// Create the device
	d, err := s.store.CreateDevice(ctx, request.Body.Name, ch.UUID)
	if err != nil {
		return CreateDevicedefaultJSONResponse{
			Body:       RenderError(err),
			StatusCode: StatusCode(err),
		}, nil
	}

	connected := false
	resp := DeviceSummary{
		UUID: &d.UUID,
		Name: &d.Name,
		Channel: &ChannelRef{
			UUID: &d.ChannelUUID,
			Name: &ch.Name,
		},
		Connected: &connected,
	}

	return CreateDevice201JSONResponse(resp), nil
}
