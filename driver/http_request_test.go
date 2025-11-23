package driver

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
)

func TestHTTPRequestImpl_Get(t *testing.T) {
	type args struct {
		ctx    context.Context
		values *url.Values
	}
	tests := []struct {
		name    string
		args    args
		handler http.HandlerFunc
		want    []byte
		wantErr bool
	}{
		{
			name: "正常系",
			args: args{
				ctx: context.Background(),
				values: &url.Values{
					"key": []string{"value"},
				},
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}
				if r.URL.Query().Get("key") != "value" {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				w.Write([]byte("response"))
			},
			want:    []byte("response"),
			wantErr: false,
		},
		{
			name: "正常系: values is nil",
			args: args{
				ctx:    context.Background(),
				values: nil,
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}
				w.Write([]byte("response"))
			},
			want:    []byte("response"),
			wantErr: false,
		},
		{
			name: "異常系: サーバーエラー",
			args: args{
				ctx:    context.Background(),
				values: nil,
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			want:    []byte{},
			wantErr: false, // Do returns err only on connection error, not status code
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			r := &HTTPRequestImpl{}
			got, err := r.Get(tt.args.ctx, server.URL, tt.args.values)
			if (err != nil) != tt.wantErr {
				t.Errorf("HTTPRequestImpl.Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// For 500 error, io.ReadAll returns empty body if server sends nothing
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HTTPRequestImpl.Get() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHTTPRequestImpl_Post(t *testing.T) {
	type args struct {
		ctx    context.Context
		values url.Values
	}
	tests := []struct {
		name    string
		args    args
		handler http.HandlerFunc
		want    []byte
		wantErr bool
	}{
		{
			name: "正常系",
			args: args{
				ctx: context.Background(),
				values: url.Values{
					"key": []string{"value"},
				},
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}
				// Check if body contains key=value
				body, _ := io.ReadAll(r.Body)
				if string(body) != "key=value" {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				w.Write([]byte("response"))
			},
			want:    []byte("response"),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			r := &HTTPRequestImpl{}
			got, err := r.Post(tt.args.ctx, server.URL, tt.args.values)
			if (err != nil) != tt.wantErr {
				t.Errorf("HTTPRequestImpl.Post() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HTTPRequestImpl.Post() = %v, want %v", got, tt.want)
			}
		})
	}
}
