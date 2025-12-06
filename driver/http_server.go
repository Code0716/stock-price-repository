package driver

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

type HTTPServer interface {
	GetQueryParam(r *http.Request, key string) string
	GetQueryParamInt(r *http.Request, key string) (int, error)
	GetQueryParamDate(r *http.Request, key string, layout string) (*time.Time, error)
	ParseJSONBody(r *http.Request, v interface{}) error
}

type HTTPServerImpl struct{}

func NewHTTPServer() HTTPServer {
	return &HTTPServerImpl{}
}

func (h *HTTPServerImpl) GetQueryParam(r *http.Request, key string) string {
	return r.URL.Query().Get(key)
}

func (h *HTTPServerImpl) GetQueryParamInt(r *http.Request, key string) (int, error) {
	val := r.URL.Query().Get(key)
	if val == "" {
		return 0, nil
	}
	return strconv.Atoi(val)
}

func (h *HTTPServerImpl) GetQueryParamDate(r *http.Request, key string, layout string) (*time.Time, error) {
	val := r.URL.Query().Get(key)
	if val == "" {
		return nil, nil
	}
	t, err := time.ParseInLocation(layout, val, time.Local)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (h *HTTPServerImpl) ParseJSONBody(r *http.Request, v interface{}) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}
