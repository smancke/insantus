package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_HttpCheck(t *testing.T) {
	for _, test := range []struct {
		name string

		status      int
		contentType string
		body        string
		params      map[string]string

		expectedStatus string
	}{
		{
			name: "up",

			status:      http.StatusOK,
			contentType: "text/plain",

			expectedStatus: StatusUp,
		},
		{
			name: "down",

			status:      http.StatusInternalServerError,
			contentType: "text/plain",

			expectedStatus: StatusDown,
		},
		{
			name: "down with detail",

			status:      http.StatusInternalServerError,
			contentType: "text/plain",
			body:        "down",

			expectedStatus: StatusDown,
		},
		{
			name: "contains",

			status:      http.StatusOK,
			contentType: "text/plain",
			body:        "up and running!",
			params: map[string]string{
				"contains": "running",
			},

			expectedStatus: StatusUp,
		},
		{
			name: "not contains",

			status:      http.StatusOK,
			contentType: "text/plain",
			body:        "down!",
			params: map[string]string{
				"contains": "running",
			},

			expectedStatus: StatusDown,
		},
		{
			name: "spring health format",

			status:      http.StatusOK,
			contentType: "application/vnd.spring-boot.actuator",
			body:        `{"status":"DOWN"}`,
			params: map[string]string{
				"format": FormatSpringHealth,
			},

			expectedStatus: StatusDown,
		},
		{
			name: "content type is not correct",

			status:      http.StatusOK,
			contentType: "text/plain",
			params: map[string]string{
				"format": FormatSpringHealth,
			},

			expectedStatus: StatusDown,
		},
		{
			name: "spring health format response not json",

			status:      http.StatusOK,
			contentType: "application/vnd.spring-boot.actuator",
			body:        `{`,
			params: map[string]string{
				"format": FormatSpringHealth,
			},

			expectedStatus: StatusDown,
		},
		{
			name: "spring health format invalid format",

			status:      http.StatusOK,
			contentType: "application/vnd.spring-boot.actuator",
			body:        `{}`,
			params: map[string]string{
				"format": FormatSpringHealth,
			},

			expectedStatus: StatusDown,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(mockServer(test.status, test.contentType, test.body))
			check, _ := NewHttpCheck(
				"prod",
				"test-check",
				"test check",
				mergeParams(map[string]string{"url": server.URL + "/health"}, test.params),
			)

			results := check.Check()

			require.Equal(t, 1, len(results))
			assert.Equal(t, test.expectedStatus, results[0].Status)
			if test.body != "" && test.expectedStatus != StatusUp {
				assert.Equal(t, test.body, results[0].Detail)
			}
		})
	}
}

func mockServer(status int, contentType, body string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", contentType)
		w.WriteHeader(status)

		w.Write([]byte(body))
	})
	return mux
}

func mergeParams(first, second map[string]string) map[string]string {
	result := first
	for key, value := range second {
		result[key] = value
	}
	return result
}
