package locations

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearchLocations(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		limit    int
		wantMin  int // minimum expected results
		wantMax  int // maximum expected results
		contains string // expected place_id in results
	}{
		{
			name:     "search for new york",
			query:    "new york",
			limit:    10,
			wantMin:  1,
			wantMax:  10,
			contains: "new-york-ny",
		},
		{
			name:     "search case insensitive",
			query:    "NEW YORK",
			limit:    10,
			wantMin:  1,
			wantMax:  10,
			contains: "new-york-ny",
		},
		{
			name:     "search partial match",
			query:    "los",
			limit:    10,
			wantMin:  1,
			wantMax:  10,
			contains: "los-angeles-ca",
		},
		{
			name:    "search with limit",
			query:   "a",
			limit:   3,
			wantMin: 3,
			wantMax: 3,
		},
		{
			name:    "empty query returns nil",
			query:   "",
			limit:   10,
			wantMin: 0,
			wantMax: 0,
		},
		{
			name:    "no matches",
			query:   "xyznonexistent",
			limit:   10,
			wantMin: 0,
			wantMax: 0,
		},
		{
			name:    "zero limit uses default",
			query:   "chicago",
			limit:   0,
			wantMin: 1,
			wantMax: 10,
		},
		{
			name:    "negative limit uses default",
			query:   "seattle",
			limit:   -1,
			wantMin: 1,
			wantMax: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := SearchLocations(tt.query, tt.limit)

			assert.GreaterOrEqual(t, len(results), tt.wantMin, "expected at least %d results", tt.wantMin)
			assert.LessOrEqual(t, len(results), tt.wantMax, "expected at most %d results", tt.wantMax)

			if tt.contains != "" {
				found := false
				for _, loc := range results {
					if loc.PlaceID == tt.contains {
						found = true
						break
					}
				}
				assert.True(t, found, "expected results to contain %s", tt.contains)
			}
		})
	}
}

func TestFindByPlaceID(t *testing.T) {
	tests := []struct {
		name    string
		placeID string
		wantNil bool
		wantLoc *Location
	}{
		{
			name:    "find existing location",
			placeID: "new-york-ny",
			wantNil: false,
			wantLoc: &Location{
				PlaceID:     "new-york-ny",
				Description: "New York, NY, USA",
				Locality:    "New York",
				Lat:         "40.7128",
				Lng:         "-74.0060",
				Timezone:    "America/New_York",
			},
		},
		{
			name:    "find another location",
			placeID: "los-angeles-ca",
			wantNil: false,
			wantLoc: &Location{
				PlaceID:     "los-angeles-ca",
				Description: "Los Angeles, CA, USA",
				Locality:    "Los Angeles",
				Lat:         "34.0522",
				Lng:         "-118.2437",
				Timezone:    "America/Los_Angeles",
			},
		},
		{
			name:    "non-existent location",
			placeID: "nonexistent-place",
			wantNil: true,
		},
		{
			name:    "empty place_id",
			placeID: "",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FindByPlaceID(tt.placeID)

			if tt.wantNil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Equal(t, tt.wantLoc.PlaceID, result.PlaceID)
				assert.Equal(t, tt.wantLoc.Description, result.Description)
				assert.Equal(t, tt.wantLoc.Locality, result.Locality)
				assert.Equal(t, tt.wantLoc.Lat, result.Lat)
				assert.Equal(t, tt.wantLoc.Lng, result.Lng)
				assert.Equal(t, tt.wantLoc.Timezone, result.Timezone)
			}
		})
	}
}

func TestLocationToJSON(t *testing.T) {
	loc := Location{
		PlaceID:     "test-place",
		Description: "Test City, TS, USA",
		Locality:    "Test City",
		Lat:         "12.3456",
		Lng:         "-78.9012",
		Timezone:    "America/Test",
	}

	jsonStr, err := loc.ToJSON()
	require.NoError(t, err)

	// Parse it back to verify structure
	var parsed map[string]string
	err = json.Unmarshal([]byte(jsonStr), &parsed)
	require.NoError(t, err)

	assert.Equal(t, "test-place", parsed["place_id"])
	assert.Equal(t, "Test City, TS, USA", parsed["description"])
	assert.Equal(t, "Test City", parsed["locality"])
	assert.Equal(t, "12.3456", parsed["lat"])
	assert.Equal(t, "-78.9012", parsed["lng"])
	assert.Equal(t, "America/Test", parsed["timezone"])
}

func TestExpandLocationConfigs(t *testing.T) {
	tests := []struct {
		name         string
		schemaFields []SchemaField
		config       map[string]string
		wantExpanded bool // whether the location field should be expanded
	}{
		{
			name: "expand location field",
			schemaFields: []SchemaField{
				{ID: "location", Type: "location"},
				{ID: "other", Type: "text"},
			},
			config: map[string]string{
				"location": "new-york-ny",
				"other":    "some value",
			},
			wantExpanded: true,
		},
		{
			name: "no location fields in schema",
			schemaFields: []SchemaField{
				{ID: "text_field", Type: "text"},
			},
			config: map[string]string{
				"text_field": "value",
			},
			wantExpanded: false,
		},
		{
			name:         "empty schema",
			schemaFields: []SchemaField{},
			config: map[string]string{
				"location": "new-york-ny",
			},
			wantExpanded: false,
		},
		{
			name: "empty config",
			schemaFields: []SchemaField{
				{ID: "location", Type: "location"},
			},
			config:       map[string]string{},
			wantExpanded: false,
		},
		{
			name: "nil config",
			schemaFields: []SchemaField{
				{ID: "location", Type: "location"},
			},
			config:       nil,
			wantExpanded: false,
		},
		{
			name: "location field with invalid place_id",
			schemaFields: []SchemaField{
				{ID: "location", Type: "location"},
			},
			config: map[string]string{
				"location": "nonexistent-place",
			},
			wantExpanded: false,
		},
		{
			name: "location field with empty value",
			schemaFields: []SchemaField{
				{ID: "location", Type: "location"},
			},
			config: map[string]string{
				"location": "",
			},
			wantExpanded: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExpandLocationConfigs(tt.schemaFields, tt.config)

			if tt.wantExpanded {
				// Verify the location was expanded to JSON
				locJSON := result["location"]
				assert.Contains(t, locJSON, "place_id")
				assert.Contains(t, locJSON, "lat")
				assert.Contains(t, locJSON, "lng")
				assert.Contains(t, locJSON, "timezone")

				// Verify original config was not modified
				if tt.config != nil {
					assert.Equal(t, "new-york-ny", tt.config["location"])
				}
			} else {
				// Result should be unchanged or same as input
				if tt.config == nil {
					assert.Nil(t, result)
				}
			}
		})
	}
}

func TestExpandLocationConfigs_PreservesOtherFields(t *testing.T) {
	schemaFields := []SchemaField{
		{ID: "location", Type: "location"},
		{ID: "color", Type: "color"},
		{ID: "name", Type: "text"},
	}
	config := map[string]string{
		"location": "chicago-il",
		"color":    "#ff0000",
		"name":     "My App",
	}

	result := ExpandLocationConfigs(schemaFields, config)

	// Other fields should be preserved
	assert.Equal(t, "#ff0000", result["color"])
	assert.Equal(t, "My App", result["name"])

	// Location should be expanded
	assert.Contains(t, result["location"], "chicago-il")
	assert.Contains(t, result["location"], "America/Chicago")
}

func TestUSCitiesData(t *testing.T) {
	// Verify we have a reasonable number of cities
	assert.Greater(t, len(USCities), 100, "expected at least 100 cities")

	// Verify each city has required fields
	for _, city := range USCities {
		assert.NotEmpty(t, city.PlaceID, "city should have place_id")
		assert.NotEmpty(t, city.Description, "city should have description")
		assert.NotEmpty(t, city.Locality, "city should have locality")
		assert.NotEmpty(t, city.Lat, "city should have lat")
		assert.NotEmpty(t, city.Lng, "city should have lng")
		assert.NotEmpty(t, city.Timezone, "city should have timezone")
	}

	// Verify no duplicate place_ids
	placeIDs := make(map[string]bool)
	for _, city := range USCities {
		assert.False(t, placeIDs[city.PlaceID], "duplicate place_id: %s", city.PlaceID)
		placeIDs[city.PlaceID] = true
	}
}
