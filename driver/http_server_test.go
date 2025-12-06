package driver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHTTPServerImpl_GetQueryParam(t *testing.T) {
	h := NewHTTPServer()

	tests := []struct {
		name     string
		query    string
		key      string
		expected string
	}{
		{
			name:     "exists",
			query:    "?key=value",
			key:      "key",
			expected: "value",
		},
		{
			name:     "not exists",
			query:    "?key=value",
			key:      "other",
			expected: "",
		},
		{
			name:     "empty",
			query:    "",
			key:      "key",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/"+tt.query, nil)
			got := h.GetQueryParam(req, tt.key)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestHTTPServerImpl_GetQueryParamInt(t *testing.T) {
	h := NewHTTPServer()

	tests := []struct {
		name      string
		query     string
		key       string
		expected  int
		expectErr bool
	}{
		{
			name:      "valid int",
			query:     "?key=123",
			key:       "key",
			expected:  123,
			expectErr: false,
		},
		{
			name:      "not exists",
			query:     "?key=123",
			key:       "other",
			expected:  0,
			expectErr: false,
		},
		{
			name:      "invalid int",
			query:     "?key=abc",
			key:       "key",
			expected:  0,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/"+tt.query, nil)
			got, err := h.GetQueryParamInt(req, tt.key)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, got)
			}
		})
	}
}

func TestHTTPServerImpl_GetQueryParamDate(t *testing.T) {
	h := NewHTTPServer()
	layout := "2006-01-02"

	tests := []struct {
		name      string
		query     string
		key       string
		expected  *time.Time
		expectErr bool
	}{
		{
			name:  "valid date",
			query: "?date=2023-01-01",
			key:   "date",
			expected: func() *time.Time {
				t, _ := time.ParseInLocation(layout, "2023-01-01", time.Local)
				return &t
			}(),
			expectErr: false,
		},
		{
			name:      "not exists",
			query:     "?date=2023-01-01",
			key:       "other",
			expected:  nil,
			expectErr: false,
		},
		{
			name:      "invalid date",
			query:     "?date=invalid",
			key:       "date",
			expected:  nil,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/"+tt.query, nil)
			got, err := h.GetQueryParamDate(req, tt.key, layout)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, got)
			}
		})
	}
}

func TestHTTPServerImpl_ParseJSONBody(t *testing.T) {
	h := NewHTTPServer()

	type TestStruct struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	tests := []struct {
		name      string
		body      interface{}
		expected  TestStruct
		expectErr bool
	}{
		{
			name: "valid json",
			body: map[string]interface{}{
				"name": "test",
				"age":  10,
			},
			expected: TestStruct{
				Name: "test",
				Age:  10,
			},
			expectErr: false,
		},
		{
			name:      "invalid json",
			body:      "invalid",
			expected:  TestStruct{},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bodyBytes []byte
			if s, ok := tt.body.(string); ok && s == "invalid" {
				bodyBytes = []byte(s)
			} else {
				bodyBytes, _ = json.Marshal(tt.body)
			}

			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(bodyBytes))
			// ParseJSONBody内でCloseされなくなったため、呼び出し元でCloseする
			defer req.Body.Close()

			var got TestStruct
			err := h.ParseJSONBody(req, &got)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, got)
			}
		})
	}
}
