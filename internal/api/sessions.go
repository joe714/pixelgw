package api

import (
	"context"
)

func (s *Server) GetSessions(ctx context.Context, request GetSessionsRequestObject) (GetSessionsResponseObject, error) {
	resp := GetSessions200JSONResponse{}
	sessions := s.hub.GetSessions()
	for _, s := range sessions {
		resp = append(resp, SessionSummary{
			ID:         &s.SessionID,
			RemoteAddr: &s.RemoteAddr,
			ClientUUID: &s.ClientUUID,
			Channel: &ChannelRef{
				UUID: &s.ChannelUUID,
				Name: &s.ChannelName,
			},
		})
	}

	return resp, nil
}
