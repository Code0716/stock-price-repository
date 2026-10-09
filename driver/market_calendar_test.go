package driver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/Code0716/stock-price-repository/config"
	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
	mock_driver "github.com/Code0716/stock-price-repository/mock/driver"
)

func TestStockAPIClient_GetTradingCalendar(t *testing.T) {
	originalJQuants := *config.GetJQuants()
	defer func() {
		*config.GetJQuants() = originalJQuants
	}()

	from := time.Date(2026, 10, 1, 0, 0, 0, 0, time.Local)
	to := time.Date(2026, 10, 12, 0, 0, 0, 0, time.Local)

	tests := []struct {
		name        string
		mockHandler http.HandlerFunc
		want        []*gateway.TradingCalendarDay
		wantErr     bool
	}{
		{
			name: "正常系: HolDiv が文字列でも数値でも読める",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/markets/calendar", r.URL.Path)
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "dummy-key", r.Header.Get("x-api-key"))
				assert.Equal(t, "2026-10-01", r.URL.Query().Get("from"))
				assert.Equal(t, "2026-10-12", r.URL.Query().Get("to"))
				_ = json.NewEncoder(w).Encode(map[string]any{
					"data": []map[string]any{
						{"Date": "2026-10-09", "HolDiv": "1"},
						{"Date": "2026-10-10", "HolDiv": "0"},
						{"Date": "2026-10-12", "HolDiv": 3},
					},
				})
			},
			want: []*gateway.TradingCalendarDay{
				{Date: time.Date(2026, 10, 9, 0, 0, 0, 0, time.Local), HolDiv: 1},
				{Date: time.Date(2026, 10, 10, 0, 0, 0, 0, time.Local), HolDiv: 0},
				{Date: time.Date(2026, 10, 12, 0, 0, 0, 0, time.Local), HolDiv: 3},
			},
		},
		{
			name: "正常系: データが空",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
			},
			want: []*gateway.TradingCalendarDay{},
		},
		{
			name: "異常系: 401",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
			},
			wantErr: true,
		},
		{
			name: "異常系: 500",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantErr: true,
		},
		{
			name: "異常系: JSON が不正",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte("not json"))
			},
			wantErr: true,
		},
		{
			name: "異常系: 日付が不正",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{{"Date": "2026/10/09", "HolDiv": "1"}}})
			},
			wantErr: true,
		},
		{
			name: "異常系: HolDiv が数値でない",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{{"Date": "2026-10-09", "HolDiv": "abc"}}})
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(tt.mockHandler)
			defer ts.Close()

			config.GetJQuants().JQuantsBaseURLV2 = ts.URL
			config.GetJQuants().JQuantsBaseURLV2APIKey = "dummy-key"

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockReq := mock_driver.NewMockHTTPRequest(ctrl)
			mockReq.EXPECT().GetHTTPClient().Return(http.DefaultClient).AnyTimes()

			c := NewStockAPIClient(mockReq, nil)

			got, err := c.GetTradingCalendar(context.Background(), from, to)
			if (err != nil) != tt.wantErr {
				t.Fatalf("GetTradingCalendar() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestEconomicCalendarClient(t *testing.T) {
	original := *config.GetEconomicCalendar()
	defer func() {
		*config.GetEconomicCalendar() = original
	}()

	tests := []struct {
		name    string
		path    string
		status  int
		body    string
		fetch   func(c gateway.EconomicCalendarClient) ([]byte, error)
		wantErr bool
	}{
		{
			name:   "正常系: 日銀",
			path:   "/boj",
			status: http.StatusOK,
			body:   "boj-body",
			fetch: func(c gateway.EconomicCalendarClient) ([]byte, error) {
				return c.FetchBOJSchedulePage(context.Background())
			},
		},
		{
			name:   "正常系: FOMC",
			path:   "/fomc",
			status: http.StatusOK,
			body:   "fomc-body",
			fetch:  func(c gateway.EconomicCalendarClient) ([]byte, error) { return c.FetchFOMCPage(context.Background()) },
		},
		{
			name:   "正常系: BLS ics",
			path:   "/bls.ics",
			status: http.StatusOK,
			body:   "bls-body",
			fetch:  func(c gateway.EconomicCalendarClient) ([]byte, error) { return c.FetchBLSICS(context.Background()) },
		},
		{
			name:    "異常系: 403 (BLS は User-Agent が不適切だと拒否する)",
			path:    "/bls.ics",
			status:  http.StatusForbidden,
			body:    "Access Denied",
			fetch:   func(c gateway.EconomicCalendarClient) ([]byte, error) { return c.FetchBLSICS(context.Background()) },
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotUA, gotPath string
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotUA, gotPath = r.Header.Get("User-Agent"), r.URL.Path
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer ts.Close()

			cfg := config.GetEconomicCalendar()
			cfg.BOJScheduleURL = ts.URL + "/boj"
			cfg.FOMCCalendarURL = ts.URL + "/fomc"
			cfg.BLSICSURL = ts.URL + "/bls.ics"
			cfg.HTTPUserAgent = "test-agent/1.0"

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockReq := mock_driver.NewMockHTTPRequest(ctrl)
			mockReq.EXPECT().GetHTTPClient().Return(http.DefaultClient).AnyTimes()

			got, err := tt.fetch(NewEconomicCalendarClient(mockReq))
			if (err != nil) != tt.wantErr {
				t.Fatalf("fetch error = %v, wantErr %v", err, tt.wantErr)
			}
			assert.Equal(t, tt.path, gotPath)
			assert.Equal(t, "test-agent/1.0", gotUA)
			if !tt.wantErr {
				assert.Equal(t, tt.body, string(got))
			}
		})
	}
}
