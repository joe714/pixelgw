package api

import (
	"context"
)

func (s *Server) GetSessions(ctx context.Context, request GetSessionsRequestObject) (GetSessionsResponseObject, error) {
	resp := GetSessions200JSONResponse{}
	sessions := s.hub.GetSessions()
	for _, sess := range sessions {
		summary := SessionSummary{
			ID:         &sess.SessionID,
			RemoteAddr: &sess.RemoteAddr,
			Channel: &ChannelRef{
				UUID: &sess.ChannelUUID,
				Name: &sess.ChannelName,
			},
			// TODO: Add device name here too
			Device: &DeviceRef{
				UUID: &sess.DeviceUUID,
			},
			Ephemeral: &sess.Ephemeral,
		}
		if sess.DisplayName != "" {
			summary.DisplayName = &sess.DisplayName
		}
		resp = append(resp, summary)
	}

	return resp, nil
}
