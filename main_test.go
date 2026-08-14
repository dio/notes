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
			wantBody:   "https://github.com/dio/envoy-one-cluster-many-providers",
		},
		{
			name:       "Envoy ADS note",
			method:     http.MethodGet,
			path:       "/notes/envoy/ads-discovery-request-version-info",
			wantStatus: http.StatusOK,
			wantBody:   "successfully accepted",
		},
		{
			name:       "Envoy provider selection article",
			method:     http.MethodGet,
			path:       "/notes/envoy/one-route-one-cluster-many-providers",
			wantStatus: http.StatusOK,
			wantBody:   "github.com/dio/envoy-one-cluster-many-providers",
		},
		{
			name:       "current Envoy prototypes and issues",
			method:     http.MethodGet,
			path:       "/notes/envoy-prototypes-and-issues",
			wantStatus: http.StatusOK,
			wantBody:   "envoy-callout-credential-injection",
		},
		{
			name:       "Envoy project index",
			method:     http.MethodGet,
			path:       "/notes/envoy/projects/",
			wantStatus: http.StatusOK,
			wantBody:   "Current Envoy prototypes and issue reproductions",
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
