package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandler(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "landing page",
			method:     http.MethodGet,
			path:       "/",
			wantStatus: http.StatusOK,
			wantBody:   "delicious cookies. reliable softwares.",
		},
		{
			name:       "notes index",
			method:     http.MethodGet,
			path:       "/notes",
			wantStatus: http.StatusOK,
			wantBody:   "ADS DiscoveryRequest.version_info across stream reconnects",
		},
		{
			name:       "Envoy ADS note",
			method:     http.MethodGet,
			path:       "/notes/envoy/ads-discovery-request-version-info",
			wantStatus: http.StatusOK,
			wantBody:   "successfully accepted",
		},
		{
			name:       "missing page",
			method:     http.MethodGet,
			path:       "/missing",
			wantStatus: http.StatusNotFound,
			wantBody:   "404 page not found",
		},
		{
			name:       "unsupported method",
			method:     http.MethodPost,
			path:       "/",
			wantStatus: http.StatusMethodNotAllowed,
			wantBody:   "Method Not Allowed",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(test.method, test.path, nil)
			response := httptest.NewRecorder()

			newHandler().ServeHTTP(response, request)

			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, test.wantStatus)
			}
			if !strings.Contains(response.Body.String(), test.wantBody) {
				t.Fatalf("body does not contain %q", test.wantBody)
			}
		})
	}
}
