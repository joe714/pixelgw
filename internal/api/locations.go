package api

import (
	"context"

	"github.com/joe714/pixelgw/internal/locations"
)

func (s *Server) SearchLocations(ctx context.Context, request SearchLocationsRequestObject) (SearchLocationsResponseObject, error) {
	query := request.Params.Q
	limit := 10
	if request.Params.Limit != nil {
		limit = *request.Params.Limit
	}

	results := locations.SearchLocations(query, limit)

	resp := make([]Location, 0, len(results))
	for _, loc := range results {
		resp = append(resp, Location{
			PlaceID:     loc.PlaceID,
			Description: loc.Description,
			Locality:    loc.Locality,
			Lat:         loc.Lat,
			Lng:         loc.Lng,
			Timezone:    loc.Timezone,
		})
	}

	return SearchLocations200JSONResponse(resp), nil
}

func (s *Server) GetLocationByPlaceID(ctx context.Context, request GetLocationByPlaceIDRequestObject) (GetLocationByPlaceIDResponseObject, error) {
	loc := locations.FindByPlaceID(request.PlaceId)
	if loc == nil {
		return GetLocationByPlaceID404JSONResponse{
			Code:    404,
			Message: "Location not found",
		}, nil
	}

	return GetLocationByPlaceID200JSONResponse{
		PlaceID:     loc.PlaceID,
		Description: loc.Description,
		Locality:    loc.Locality,
		Lat:         loc.Lat,
		Lng:         loc.Lng,
		Timezone:    loc.Timezone,
	}, nil
}
