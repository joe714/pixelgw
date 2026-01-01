package api_test

import (
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/google/uuid"
	"github.com/joe714/pixelgw/internal/api"
	"github.com/joe714/pixelgw/internal/durable"
	"github.com/joe714/pixelgw/internal/testutil"
)

// Test device UUIDs for API tests
var (
	testDeviceUUID1 = uuid.MustParse("d0000001-0000-0000-0000-000000000001")
	testDeviceUUID2 = uuid.MustParse("d0000002-0000-0000-0000-000000000002")
	testDeviceUUID3 = uuid.MustParse("d0000003-0000-0000-0000-000000000003")
)

// testServer creates an HTTP test server with the API handlers
func testServer(store *durable.Store) *httptest.Server {
	server := api.NewServer(nil, store) // nil hub for most tests
	strictHandler := api.NewStrictHandlerWithOptions(server, nil, api.ServerOptions())
	handler := api.Handler(strictHandler)
	return httptest.NewServer(handler)
}

// setupTest creates a test store and server, returning both plus a cleanup function
func setupTest() (*durable.Store, *httptest.Server, func()) {
	store, cleanupDB, err := testutil.NewTestStore()
	if err != nil {
		panic(err)
	}

	ts := testServer(store)

	cleanup := func() {
		ts.Close()
		cleanupDB()
	}

	return store, ts, cleanup
}

// helper to make requests and return response
func doRequest(ts *httptest.Server, method, path string, body string) *http.Response {
	var req *http.Request
	var err error

	if body != "" {
		req, err = http.NewRequest(method, ts.URL+path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequest(method, ts.URL+path, nil)
	}
	if err != nil {
		panic(err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	return resp
}
