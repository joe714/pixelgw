package locations

import (
	"encoding/json"
	"strings"
)

// Location represents a geographic location with timezone info
type Location struct {
	PlaceID     string `json:"place_id"`
	Description string `json:"description"`
	Locality    string `json:"locality"`
	Lat         string `json:"lat"`
	Lng         string `json:"lng"`
	Timezone    string `json:"timezone"`
}

// USCities contains major US cities for location search
// Data sourced from public domain geographic databases
var USCities = []Location{
	// Northeast
	{PlaceID: "new-york-ny", Description: "New York, NY, USA", Locality: "New York", Lat: "40.7128", Lng: "-74.0060", Timezone: "America/New_York"},
	{PlaceID: "brooklyn-ny", Description: "Brooklyn, NY, USA", Locality: "Brooklyn", Lat: "40.6782", Lng: "-73.9442", Timezone: "America/New_York"},
	{PlaceID: "manhattan-ny", Description: "Manhattan, NY, USA", Locality: "Manhattan", Lat: "40.7831", Lng: "-73.9712", Timezone: "America/New_York"},
	{PlaceID: "queens-ny", Description: "Queens, NY, USA", Locality: "Queens", Lat: "40.7282", Lng: "-73.7949", Timezone: "America/New_York"},
	{PlaceID: "buffalo-ny", Description: "Buffalo, NY, USA", Locality: "Buffalo", Lat: "42.8864", Lng: "-78.8784", Timezone: "America/New_York"},
	{PlaceID: "albany-ny", Description: "Albany, NY, USA", Locality: "Albany", Lat: "42.6526", Lng: "-73.7562", Timezone: "America/New_York"},
	{PlaceID: "boston-ma", Description: "Boston, MA, USA", Locality: "Boston", Lat: "42.3601", Lng: "-71.0589", Timezone: "America/New_York"},
	{PlaceID: "cambridge-ma", Description: "Cambridge, MA, USA", Locality: "Cambridge", Lat: "42.3736", Lng: "-71.1097", Timezone: "America/New_York"},
	{PlaceID: "philadelphia-pa", Description: "Philadelphia, PA, USA", Locality: "Philadelphia", Lat: "39.9526", Lng: "-75.1652", Timezone: "America/New_York"},
	{PlaceID: "pittsburgh-pa", Description: "Pittsburgh, PA, USA", Locality: "Pittsburgh", Lat: "40.4406", Lng: "-79.9959", Timezone: "America/New_York"},
	{PlaceID: "harrisburg-pa", Description: "Harrisburg, PA, USA", Locality: "Harrisburg", Lat: "40.2732", Lng: "-76.8867", Timezone: "America/New_York"},
	{PlaceID: "newark-nj", Description: "Newark, NJ, USA", Locality: "Newark", Lat: "40.7357", Lng: "-74.1724", Timezone: "America/New_York"},
	{PlaceID: "trenton-nj", Description: "Trenton, NJ, USA", Locality: "Trenton", Lat: "40.2206", Lng: "-74.7597", Timezone: "America/New_York"},
	{PlaceID: "hartford-ct", Description: "Hartford, CT, USA", Locality: "Hartford", Lat: "41.7658", Lng: "-72.6734", Timezone: "America/New_York"},
	{PlaceID: "providence-ri", Description: "Providence, RI, USA", Locality: "Providence", Lat: "41.8240", Lng: "-71.4128", Timezone: "America/New_York"},
	{PlaceID: "portland-me", Description: "Portland, ME, USA", Locality: "Portland", Lat: "43.6591", Lng: "-70.2568", Timezone: "America/New_York"},
	{PlaceID: "augusta-me", Description: "Augusta, ME, USA", Locality: "Augusta", Lat: "44.3106", Lng: "-69.7795", Timezone: "America/New_York"},
	{PlaceID: "concord-nh", Description: "Concord, NH, USA", Locality: "Concord", Lat: "43.2081", Lng: "-71.5376", Timezone: "America/New_York"},
	{PlaceID: "montpelier-vt", Description: "Montpelier, VT, USA", Locality: "Montpelier", Lat: "44.2601", Lng: "-72.5754", Timezone: "America/New_York"},
	{PlaceID: "burlington-vt", Description: "Burlington, VT, USA", Locality: "Burlington", Lat: "44.4759", Lng: "-73.2121", Timezone: "America/New_York"},

	// Mid-Atlantic / Southeast
	{PlaceID: "washington-dc", Description: "Washington, DC, USA", Locality: "Washington", Lat: "38.9072", Lng: "-77.0369", Timezone: "America/New_York"},
	{PlaceID: "baltimore-md", Description: "Baltimore, MD, USA", Locality: "Baltimore", Lat: "39.2904", Lng: "-76.6122", Timezone: "America/New_York"},
	{PlaceID: "annapolis-md", Description: "Annapolis, MD, USA", Locality: "Annapolis", Lat: "38.9784", Lng: "-76.4922", Timezone: "America/New_York"},
	{PlaceID: "richmond-va", Description: "Richmond, VA, USA", Locality: "Richmond", Lat: "37.5407", Lng: "-77.4360", Timezone: "America/New_York"},
	{PlaceID: "virginia-beach-va", Description: "Virginia Beach, VA, USA", Locality: "Virginia Beach", Lat: "36.8529", Lng: "-75.9780", Timezone: "America/New_York"},
	{PlaceID: "norfolk-va", Description: "Norfolk, VA, USA", Locality: "Norfolk", Lat: "36.8508", Lng: "-76.2859", Timezone: "America/New_York"},
	{PlaceID: "wilmington-de", Description: "Wilmington, DE, USA", Locality: "Wilmington", Lat: "39.7391", Lng: "-75.5398", Timezone: "America/New_York"},
	{PlaceID: "dover-de", Description: "Dover, DE, USA", Locality: "Dover", Lat: "39.1582", Lng: "-75.5244", Timezone: "America/New_York"},
	{PlaceID: "charleston-wv", Description: "Charleston, WV, USA", Locality: "Charleston", Lat: "38.3498", Lng: "-81.6326", Timezone: "America/New_York"},
	{PlaceID: "charlotte-nc", Description: "Charlotte, NC, USA", Locality: "Charlotte", Lat: "35.2271", Lng: "-80.8431", Timezone: "America/New_York"},
	{PlaceID: "raleigh-nc", Description: "Raleigh, NC, USA", Locality: "Raleigh", Lat: "35.7796", Lng: "-78.6382", Timezone: "America/New_York"},
	{PlaceID: "durham-nc", Description: "Durham, NC, USA", Locality: "Durham", Lat: "35.9940", Lng: "-78.8986", Timezone: "America/New_York"},
	{PlaceID: "columbia-sc", Description: "Columbia, SC, USA", Locality: "Columbia", Lat: "34.0007", Lng: "-81.0348", Timezone: "America/New_York"},
	{PlaceID: "charleston-sc", Description: "Charleston, SC, USA", Locality: "Charleston", Lat: "32.7765", Lng: "-79.9311", Timezone: "America/New_York"},
	{PlaceID: "atlanta-ga", Description: "Atlanta, GA, USA", Locality: "Atlanta", Lat: "33.7490", Lng: "-84.3880", Timezone: "America/New_York"},
	{PlaceID: "savannah-ga", Description: "Savannah, GA, USA", Locality: "Savannah", Lat: "32.0809", Lng: "-81.0912", Timezone: "America/New_York"},
	{PlaceID: "jacksonville-fl", Description: "Jacksonville, FL, USA", Locality: "Jacksonville", Lat: "30.3322", Lng: "-81.6557", Timezone: "America/New_York"},
	{PlaceID: "miami-fl", Description: "Miami, FL, USA", Locality: "Miami", Lat: "25.7617", Lng: "-80.1918", Timezone: "America/New_York"},
	{PlaceID: "orlando-fl", Description: "Orlando, FL, USA", Locality: "Orlando", Lat: "28.5383", Lng: "-81.3792", Timezone: "America/New_York"},
	{PlaceID: "tampa-fl", Description: "Tampa, FL, USA", Locality: "Tampa", Lat: "27.9506", Lng: "-82.4572", Timezone: "America/New_York"},
	{PlaceID: "tallahassee-fl", Description: "Tallahassee, FL, USA", Locality: "Tallahassee", Lat: "30.4383", Lng: "-84.2807", Timezone: "America/New_York"},
	{PlaceID: "fort-lauderdale-fl", Description: "Fort Lauderdale, FL, USA", Locality: "Fort Lauderdale", Lat: "26.1224", Lng: "-80.1373", Timezone: "America/New_York"},

	// Midwest
	{PlaceID: "chicago-il", Description: "Chicago, IL, USA", Locality: "Chicago", Lat: "41.8781", Lng: "-87.6298", Timezone: "America/Chicago"},
	{PlaceID: "springfield-il", Description: "Springfield, IL, USA", Locality: "Springfield", Lat: "39.7817", Lng: "-89.6501", Timezone: "America/Chicago"},
	{PlaceID: "detroit-mi", Description: "Detroit, MI, USA", Locality: "Detroit", Lat: "42.3314", Lng: "-83.0458", Timezone: "America/Detroit"},
	{PlaceID: "grand-rapids-mi", Description: "Grand Rapids, MI, USA", Locality: "Grand Rapids", Lat: "42.9634", Lng: "-85.6681", Timezone: "America/Detroit"},
	{PlaceID: "lansing-mi", Description: "Lansing, MI, USA", Locality: "Lansing", Lat: "42.7325", Lng: "-84.5555", Timezone: "America/Detroit"},
	{PlaceID: "ann-arbor-mi", Description: "Ann Arbor, MI, USA", Locality: "Ann Arbor", Lat: "42.2808", Lng: "-83.7430", Timezone: "America/Detroit"},
	{PlaceID: "cleveland-oh", Description: "Cleveland, OH, USA", Locality: "Cleveland", Lat: "41.4993", Lng: "-81.6944", Timezone: "America/New_York"},
	{PlaceID: "columbus-oh", Description: "Columbus, OH, USA", Locality: "Columbus", Lat: "39.9612", Lng: "-82.9988", Timezone: "America/New_York"},
	{PlaceID: "cincinnati-oh", Description: "Cincinnati, OH, USA", Locality: "Cincinnati", Lat: "39.1031", Lng: "-84.5120", Timezone: "America/New_York"},
	{PlaceID: "indianapolis-in", Description: "Indianapolis, IN, USA", Locality: "Indianapolis", Lat: "39.7684", Lng: "-86.1581", Timezone: "America/Indiana/Indianapolis"},
	{PlaceID: "milwaukee-wi", Description: "Milwaukee, WI, USA", Locality: "Milwaukee", Lat: "43.0389", Lng: "-87.9065", Timezone: "America/Chicago"},
	{PlaceID: "madison-wi", Description: "Madison, WI, USA", Locality: "Madison", Lat: "43.0731", Lng: "-89.4012", Timezone: "America/Chicago"},
	{PlaceID: "minneapolis-mn", Description: "Minneapolis, MN, USA", Locality: "Minneapolis", Lat: "44.9778", Lng: "-93.2650", Timezone: "America/Chicago"},
	{PlaceID: "saint-paul-mn", Description: "Saint Paul, MN, USA", Locality: "Saint Paul", Lat: "44.9537", Lng: "-93.0900", Timezone: "America/Chicago"},
	{PlaceID: "des-moines-ia", Description: "Des Moines, IA, USA", Locality: "Des Moines", Lat: "41.5868", Lng: "-93.6250", Timezone: "America/Chicago"},
	{PlaceID: "kansas-city-mo", Description: "Kansas City, MO, USA", Locality: "Kansas City", Lat: "39.0997", Lng: "-94.5786", Timezone: "America/Chicago"},
	{PlaceID: "saint-louis-mo", Description: "Saint Louis, MO, USA", Locality: "Saint Louis", Lat: "38.6270", Lng: "-90.1994", Timezone: "America/Chicago"},
	{PlaceID: "jefferson-city-mo", Description: "Jefferson City, MO, USA", Locality: "Jefferson City", Lat: "38.5767", Lng: "-92.1735", Timezone: "America/Chicago"},
	{PlaceID: "omaha-ne", Description: "Omaha, NE, USA", Locality: "Omaha", Lat: "41.2565", Lng: "-95.9345", Timezone: "America/Chicago"},
	{PlaceID: "lincoln-ne", Description: "Lincoln, NE, USA", Locality: "Lincoln", Lat: "40.8258", Lng: "-96.6852", Timezone: "America/Chicago"},
	{PlaceID: "wichita-ks", Description: "Wichita, KS, USA", Locality: "Wichita", Lat: "37.6872", Lng: "-97.3301", Timezone: "America/Chicago"},
	{PlaceID: "topeka-ks", Description: "Topeka, KS, USA", Locality: "Topeka", Lat: "39.0473", Lng: "-95.6752", Timezone: "America/Chicago"},
	{PlaceID: "fargo-nd", Description: "Fargo, ND, USA", Locality: "Fargo", Lat: "46.8772", Lng: "-96.7898", Timezone: "America/Chicago"},
	{PlaceID: "bismarck-nd", Description: "Bismarck, ND, USA", Locality: "Bismarck", Lat: "46.8083", Lng: "-100.7837", Timezone: "America/Chicago"},
	{PlaceID: "sioux-falls-sd", Description: "Sioux Falls, SD, USA", Locality: "Sioux Falls", Lat: "43.5446", Lng: "-96.7311", Timezone: "America/Chicago"},
	{PlaceID: "pierre-sd", Description: "Pierre, SD, USA", Locality: "Pierre", Lat: "44.3683", Lng: "-100.3510", Timezone: "America/Chicago"},

	// South Central
	{PlaceID: "dallas-tx", Description: "Dallas, TX, USA", Locality: "Dallas", Lat: "32.7767", Lng: "-96.7970", Timezone: "America/Chicago"},
	{PlaceID: "houston-tx", Description: "Houston, TX, USA", Locality: "Houston", Lat: "29.7604", Lng: "-95.3698", Timezone: "America/Chicago"},
	{PlaceID: "austin-tx", Description: "Austin, TX, USA", Locality: "Austin", Lat: "30.2672", Lng: "-97.7431", Timezone: "America/Chicago"},
	{PlaceID: "san-antonio-tx", Description: "San Antonio, TX, USA", Locality: "San Antonio", Lat: "29.4241", Lng: "-98.4936", Timezone: "America/Chicago"},
	{PlaceID: "fort-worth-tx", Description: "Fort Worth, TX, USA", Locality: "Fort Worth", Lat: "32.7555", Lng: "-97.3308", Timezone: "America/Chicago"},
	{PlaceID: "el-paso-tx", Description: "El Paso, TX, USA", Locality: "El Paso", Lat: "31.7619", Lng: "-106.4850", Timezone: "America/Denver"},
	{PlaceID: "new-orleans-la", Description: "New Orleans, LA, USA", Locality: "New Orleans", Lat: "29.9511", Lng: "-90.0715", Timezone: "America/Chicago"},
	{PlaceID: "baton-rouge-la", Description: "Baton Rouge, LA, USA", Locality: "Baton Rouge", Lat: "30.4515", Lng: "-91.1871", Timezone: "America/Chicago"},
	{PlaceID: "memphis-tn", Description: "Memphis, TN, USA", Locality: "Memphis", Lat: "35.1495", Lng: "-90.0490", Timezone: "America/Chicago"},
	{PlaceID: "nashville-tn", Description: "Nashville, TN, USA", Locality: "Nashville", Lat: "36.1627", Lng: "-86.7816", Timezone: "America/Chicago"},
	{PlaceID: "knoxville-tn", Description: "Knoxville, TN, USA", Locality: "Knoxville", Lat: "35.9606", Lng: "-83.9207", Timezone: "America/New_York"},
	{PlaceID: "louisville-ky", Description: "Louisville, KY, USA", Locality: "Louisville", Lat: "38.2527", Lng: "-85.7585", Timezone: "America/Kentucky/Louisville"},
	{PlaceID: "lexington-ky", Description: "Lexington, KY, USA", Locality: "Lexington", Lat: "38.0406", Lng: "-84.5037", Timezone: "America/Kentucky/Louisville"},
	{PlaceID: "frankfort-ky", Description: "Frankfort, KY, USA", Locality: "Frankfort", Lat: "38.2009", Lng: "-84.8733", Timezone: "America/Kentucky/Louisville"},
	{PlaceID: "birmingham-al", Description: "Birmingham, AL, USA", Locality: "Birmingham", Lat: "33.5207", Lng: "-86.8025", Timezone: "America/Chicago"},
	{PlaceID: "montgomery-al", Description: "Montgomery, AL, USA", Locality: "Montgomery", Lat: "32.3792", Lng: "-86.3077", Timezone: "America/Chicago"},
	{PlaceID: "jackson-ms", Description: "Jackson, MS, USA", Locality: "Jackson", Lat: "32.2988", Lng: "-90.1848", Timezone: "America/Chicago"},
	{PlaceID: "little-rock-ar", Description: "Little Rock, AR, USA", Locality: "Little Rock", Lat: "34.7465", Lng: "-92.2896", Timezone: "America/Chicago"},
	{PlaceID: "oklahoma-city-ok", Description: "Oklahoma City, OK, USA", Locality: "Oklahoma City", Lat: "35.4676", Lng: "-97.5164", Timezone: "America/Chicago"},
	{PlaceID: "tulsa-ok", Description: "Tulsa, OK, USA", Locality: "Tulsa", Lat: "36.1540", Lng: "-95.9928", Timezone: "America/Chicago"},

	// Mountain
	{PlaceID: "denver-co", Description: "Denver, CO, USA", Locality: "Denver", Lat: "39.7392", Lng: "-104.9903", Timezone: "America/Denver"},
	{PlaceID: "colorado-springs-co", Description: "Colorado Springs, CO, USA", Locality: "Colorado Springs", Lat: "38.8339", Lng: "-104.8214", Timezone: "America/Denver"},
	{PlaceID: "boulder-co", Description: "Boulder, CO, USA", Locality: "Boulder", Lat: "40.0150", Lng: "-105.2705", Timezone: "America/Denver"},
	{PlaceID: "phoenix-az", Description: "Phoenix, AZ, USA", Locality: "Phoenix", Lat: "33.4484", Lng: "-112.0740", Timezone: "America/Phoenix"},
	{PlaceID: "tucson-az", Description: "Tucson, AZ, USA", Locality: "Tucson", Lat: "32.2226", Lng: "-110.9747", Timezone: "America/Phoenix"},
	{PlaceID: "scottsdale-az", Description: "Scottsdale, AZ, USA", Locality: "Scottsdale", Lat: "33.4942", Lng: "-111.9261", Timezone: "America/Phoenix"},
	{PlaceID: "albuquerque-nm", Description: "Albuquerque, NM, USA", Locality: "Albuquerque", Lat: "35.0844", Lng: "-106.6504", Timezone: "America/Denver"},
	{PlaceID: "santa-fe-nm", Description: "Santa Fe, NM, USA", Locality: "Santa Fe", Lat: "35.6870", Lng: "-105.9378", Timezone: "America/Denver"},
	{PlaceID: "salt-lake-city-ut", Description: "Salt Lake City, UT, USA", Locality: "Salt Lake City", Lat: "40.7608", Lng: "-111.8910", Timezone: "America/Denver"},
	{PlaceID: "las-vegas-nv", Description: "Las Vegas, NV, USA", Locality: "Las Vegas", Lat: "36.1699", Lng: "-115.1398", Timezone: "America/Los_Angeles"},
	{PlaceID: "reno-nv", Description: "Reno, NV, USA", Locality: "Reno", Lat: "39.5296", Lng: "-119.8138", Timezone: "America/Los_Angeles"},
	{PlaceID: "carson-city-nv", Description: "Carson City, NV, USA", Locality: "Carson City", Lat: "39.1638", Lng: "-119.7674", Timezone: "America/Los_Angeles"},
	{PlaceID: "boise-id", Description: "Boise, ID, USA", Locality: "Boise", Lat: "43.6150", Lng: "-116.2023", Timezone: "America/Boise"},
	{PlaceID: "billings-mt", Description: "Billings, MT, USA", Locality: "Billings", Lat: "45.7833", Lng: "-108.5007", Timezone: "America/Denver"},
	{PlaceID: "helena-mt", Description: "Helena, MT, USA", Locality: "Helena", Lat: "46.5891", Lng: "-112.0391", Timezone: "America/Denver"},
	{PlaceID: "cheyenne-wy", Description: "Cheyenne, WY, USA", Locality: "Cheyenne", Lat: "41.1400", Lng: "-104.8202", Timezone: "America/Denver"},

	// Pacific
	{PlaceID: "los-angeles-ca", Description: "Los Angeles, CA, USA", Locality: "Los Angeles", Lat: "34.0522", Lng: "-118.2437", Timezone: "America/Los_Angeles"},
	{PlaceID: "san-francisco-ca", Description: "San Francisco, CA, USA", Locality: "San Francisco", Lat: "37.7749", Lng: "-122.4194", Timezone: "America/Los_Angeles"},
	{PlaceID: "san-diego-ca", Description: "San Diego, CA, USA", Locality: "San Diego", Lat: "32.7157", Lng: "-117.1611", Timezone: "America/Los_Angeles"},
	{PlaceID: "san-jose-ca", Description: "San Jose, CA, USA", Locality: "San Jose", Lat: "37.3382", Lng: "-121.8863", Timezone: "America/Los_Angeles"},
	{PlaceID: "sacramento-ca", Description: "Sacramento, CA, USA", Locality: "Sacramento", Lat: "38.5816", Lng: "-121.4944", Timezone: "America/Los_Angeles"},
	{PlaceID: "oakland-ca", Description: "Oakland, CA, USA", Locality: "Oakland", Lat: "37.8044", Lng: "-122.2712", Timezone: "America/Los_Angeles"},
	{PlaceID: "fresno-ca", Description: "Fresno, CA, USA", Locality: "Fresno", Lat: "36.7378", Lng: "-119.7871", Timezone: "America/Los_Angeles"},
	{PlaceID: "long-beach-ca", Description: "Long Beach, CA, USA", Locality: "Long Beach", Lat: "33.7701", Lng: "-118.1937", Timezone: "America/Los_Angeles"},
	{PlaceID: "seattle-wa", Description: "Seattle, WA, USA", Locality: "Seattle", Lat: "47.6062", Lng: "-122.3321", Timezone: "America/Los_Angeles"},
	{PlaceID: "olympia-wa", Description: "Olympia, WA, USA", Locality: "Olympia", Lat: "47.0379", Lng: "-122.9007", Timezone: "America/Los_Angeles"},
	{PlaceID: "tacoma-wa", Description: "Tacoma, WA, USA", Locality: "Tacoma", Lat: "47.2529", Lng: "-122.4443", Timezone: "America/Los_Angeles"},
	{PlaceID: "spokane-wa", Description: "Spokane, WA, USA", Locality: "Spokane", Lat: "47.6588", Lng: "-117.4260", Timezone: "America/Los_Angeles"},
	{PlaceID: "portland-or", Description: "Portland, OR, USA", Locality: "Portland", Lat: "45.5152", Lng: "-122.6784", Timezone: "America/Los_Angeles"},
	{PlaceID: "salem-or", Description: "Salem, OR, USA", Locality: "Salem", Lat: "44.9429", Lng: "-123.0351", Timezone: "America/Los_Angeles"},
	{PlaceID: "eugene-or", Description: "Eugene, OR, USA", Locality: "Eugene", Lat: "44.0521", Lng: "-123.0868", Timezone: "America/Los_Angeles"},

	// Alaska & Hawaii
	{PlaceID: "anchorage-ak", Description: "Anchorage, AK, USA", Locality: "Anchorage", Lat: "61.2181", Lng: "-149.9003", Timezone: "America/Anchorage"},
	{PlaceID: "juneau-ak", Description: "Juneau, AK, USA", Locality: "Juneau", Lat: "58.3019", Lng: "-134.4197", Timezone: "America/Juneau"},
	{PlaceID: "fairbanks-ak", Description: "Fairbanks, AK, USA", Locality: "Fairbanks", Lat: "64.8378", Lng: "-147.7164", Timezone: "America/Anchorage"},
	{PlaceID: "honolulu-hi", Description: "Honolulu, HI, USA", Locality: "Honolulu", Lat: "21.3069", Lng: "-157.8583", Timezone: "Pacific/Honolulu"},
	{PlaceID: "hilo-hi", Description: "Hilo, HI, USA", Locality: "Hilo", Lat: "19.7074", Lng: "-155.0885", Timezone: "Pacific/Honolulu"},
	{PlaceID: "maui-hi", Description: "Maui, HI, USA", Locality: "Maui", Lat: "20.7984", Lng: "-156.3319", Timezone: "Pacific/Honolulu"},
}

// SearchLocations searches for locations matching the query string
func SearchLocations(query string, limit int) []Location {
	if query == "" {
		return nil
	}
	if limit <= 0 {
		limit = 10
	}

	query = strings.ToLower(query)
	var results []Location

	for _, loc := range USCities {
		if len(results) >= limit {
			break
		}
		// Search in description and locality
		if strings.Contains(strings.ToLower(loc.Description), query) ||
			strings.Contains(strings.ToLower(loc.Locality), query) {
			results = append(results, loc)
		}
	}

	return results
}

// FindByPlaceID finds a location by its place_id
func FindByPlaceID(placeID string) *Location {
	for _, loc := range USCities {
		if loc.PlaceID == placeID {
			return &loc
		}
	}
	return nil
}

// ExpandLocationConfigs replaces location place_ids with full location JSON
// The schema tells us which fields are location type, and we expand those
// from place_id to the full JSON object that Pixlet expects
func ExpandLocationConfigs(schemaFields []SchemaField, config map[string]string) map[string]string {
	if len(schemaFields) == 0 || len(config) == 0 {
		return config
	}

	// Create a copy so we don't modify the original
	result := make(map[string]string, len(config))
	for k, v := range config {
		result[k] = v
	}

	// Find location fields in the schema and expand them
	for _, field := range schemaFields {
		if field.Type != "location" {
			continue
		}
		placeID, ok := result[field.ID]
		if !ok || placeID == "" {
			continue
		}

		// Look up the full location data
		loc := FindByPlaceID(placeID)
		if loc == nil {
			continue
		}

		// Convert to JSON as expected by Pixlet
		locJSON, err := loc.ToJSON()
		if err != nil {
			continue
		}
		result[field.ID] = locJSON
	}

	return result
}

// SchemaField represents a field from the applet schema
type SchemaField struct {
	ID   string
	Type string
}

// ToJSON converts a Location to the JSON format expected by Pixlet
func (l *Location) ToJSON() (string, error) {
	// Pixlet expects this specific format
	data := map[string]string{
		"place_id":    l.PlaceID,
		"description": l.Description,
		"locality":    l.Locality,
		"lat":         l.Lat,
		"lng":         l.Lng,
		"timezone":    l.Timezone,
	}
	bytes, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
