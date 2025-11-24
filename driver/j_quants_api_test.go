package driver

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"go.uber.org/mock/gomock"

	"github.com/Code0716/stock-price-repository/config"
	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
	mock_driver "github.com/Code0716/stock-price-repository/mock/driver"
)

func TestStockAPIClient_GetStockBrands(t *testing.T) {
	// miniredis setup
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run() error = %v", err)
	}
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	// Set config for tests
	config.GetJQuants().JQuantsBaseURLV1 = "https://api.jquants.com/v1"

	type fields struct {
		request     func(ctrl *gomock.Controller) HTTPRequest
		redisClient *redis.Client
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		setup   func()
		wantErr bool
	}{
		{
			name: "正常系",
			fields: fields{
				request: func(ctrl *gomock.Controller) HTTPRequest {
					mock := mock_driver.NewMockHTTPRequest(ctrl)
					mock.EXPECT().GetHTTPClient().Return(&http.Client{
						Transport: &MockRoundTripper{
							RoundTripFunc: func(req *http.Request) (*http.Response, error) {
								if req.URL.Path == "/v1/listed/info" {
									// Anonymous struct in model, so using map for simplicity or defining inline
									respBody := map[string]any{
										"info": []map[string]any{
											{
												"Code":        "10010", // Suffix 0 for trimming test
												"CompanyName": "Test Company",
												"Date":        "2023-01-01",
												"MarketCode":  "0111",
											},
										},
									}
									b, _ := json.Marshal(respBody)
									return &http.Response{
										StatusCode: http.StatusOK,
										Body:       io.NopCloser(bytes.NewReader(b)),
									}, nil
								}
								return &http.Response{
									StatusCode: http.StatusNotFound,
									Body:       io.NopCloser(bytes.NewReader(nil)),
								}, nil
							},
						},
					}).AnyTimes()
					return mock
				},
				redisClient: redisClient,
			},
			args: args{
				ctx: context.Background(),
			},
			setup: func() {
				s.Set(jQuantsAPIIDTokenRedisKey, "test_token")
			},
			wantErr: false,
		},
		{
			name: "異常系: IDトークン取得エラー",
			fields: fields{
				request: func(ctrl *gomock.Controller) HTTPRequest {
					mock := mock_driver.NewMockHTTPRequest(ctrl)
					// Expect call to get http client for token refresh
					mock.EXPECT().GetHTTPClient().Return(&http.Client{
						Transport: &MockRoundTripper{
							RoundTripFunc: func(_ *http.Request) (*http.Response, error) {
								// Fail token request
								return &http.Response{
									StatusCode: http.StatusInternalServerError,
									Body:       io.NopCloser(bytes.NewReader(nil)),
								}, nil
							},
						},
					}).AnyTimes()
					return mock
				},
				redisClient: redisClient,
			},
			args: args{
				ctx: context.Background(),
			},
			setup: func() {
				s.FlushAll() // トークンなし
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			if tt.setup != nil {
				tt.setup()
			}

			c := &StockAPIClient{
				request:     tt.fields.request(ctrl),
				redisClient: tt.fields.redisClient,
			}
			_, err := c.GetStockBrands(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("StockAPIClient.GetStockBrands() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStockAPIClient_GetAnnounceFinSchedule(t *testing.T) {
	// miniredis setup
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run() error = %v", err)
	}
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	type fields struct {
		request     func(ctrl *gomock.Controller) HTTPRequest
		redisClient *redis.Client
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		setup   func()
		wantErr bool
	}{
		{
			name: "正常系",
			fields: fields{
				request: func(ctrl *gomock.Controller) HTTPRequest {
					mock := mock_driver.NewMockHTTPRequest(ctrl)
					mock.EXPECT().GetHTTPClient().Return(&http.Client{
						Transport: &MockRoundTripper{
							RoundTripFunc: func(req *http.Request) (*http.Response, error) {
								if req.URL.Path == "/v1/fins/announcement" {
									respBody := jQuantsAnnounceFinsScheduleResponse{
										Announcement: []*AnnounceFinSchedule{
											{
												Code:        "10010",
												CompanyName: "Test Company",
												Date:        "2023-01-01",
											},
										},
									}
									b, _ := json.Marshal(respBody)
									return &http.Response{
										StatusCode: http.StatusOK,
										Body:       io.NopCloser(bytes.NewReader(b)),
									}, nil
								}
								return &http.Response{
									StatusCode: http.StatusNotFound,
									Body:       io.NopCloser(bytes.NewReader(nil)),
								}, nil
							},
						},
					}).AnyTimes()
					return mock
				},
				redisClient: redisClient,
			},
			args: args{
				ctx: context.Background(),
			},
			setup: func() {
				s.Set(jQuantsAPIIDTokenRedisKey, "test_token")
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			if tt.setup != nil {
				tt.setup()
			}

			c := &StockAPIClient{
				request:     tt.fields.request(ctrl),
				redisClient: tt.fields.redisClient,
			}
			_, err := c.GetAnnounceFinSchedule(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("StockAPIClient.GetAnnounceFinsSchedule() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStockAPIClient_getDailyPricesBySymbolAndRangeJQ(t *testing.T) {
	// miniredis setup
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run() error = %v", err)
	}
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	type fields struct {
		request     func(ctrl *gomock.Controller) HTTPRequest
		redisClient *redis.Client
	}
	type args struct {
		ctx      context.Context
		symbol   string
		dateFrom time.Time
		dateTo   time.Time
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		setup   func()
		wantErr bool
	}{
		{
			name: "正常系",
			fields: fields{
				request: func(ctrl *gomock.Controller) HTTPRequest {
					mock := mock_driver.NewMockHTTPRequest(ctrl)
					mock.EXPECT().GetHTTPClient().Return(&http.Client{
						Transport: &MockRoundTripper{
							RoundTripFunc: func(req *http.Request) (*http.Response, error) {
								if req.URL.Path == "/v1/prices/daily_quotes" {
									respBody := jQuantsDailyQuotesResponse{
										DailyQuotes: []*jQuantsDailyQuote{
											{
												Code: "10010",
												Date: "2023-01-01",
												Open: decimal.NewFromFloat(100.0),
											},
										},
									}
									b, _ := json.Marshal(respBody)
									return &http.Response{
										StatusCode: http.StatusOK,
										Body:       io.NopCloser(bytes.NewReader(b)),
									}, nil
								}
								return &http.Response{
									StatusCode: http.StatusNotFound,
									Body:       io.NopCloser(bytes.NewReader(nil)),
								}, nil
							},
						},
					}).AnyTimes()
					return mock
				},
				redisClient: redisClient,
			},
			args: args{
				ctx:      context.Background(),
				symbol:   "1001",
				dateFrom: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
				dateTo:   time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
			},
			setup: func() {
				s.Set(jQuantsAPIIDTokenRedisKey, "test_token")
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			if tt.setup != nil {
				tt.setup()
			}

			c := &StockAPIClient{
				request:     tt.fields.request(ctrl),
				redisClient: tt.fields.redisClient,
			}
			_, err := c.getDailyPricesBySymbolAndRangeJQ(tt.args.ctx, tt.args.symbol, tt.args.dateFrom, tt.args.dateTo)
			if (err != nil) != tt.wantErr {
				t.Errorf("StockAPIClient.getDailyPricesBySymbolAndRangeJQ() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStockAPIClient_getFinancialStatementsJQ(t *testing.T) {
	// miniredis setup
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run() error = %v", err)
	}
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	type fields struct {
		request     func(ctrl *gomock.Controller) HTTPRequest
		redisClient *redis.Client
	}
	type args struct {
		ctx    context.Context
		symbol string
		date   *time.Time
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		setup   func()
		wantErr bool
	}{
		{
			name: "正常系",
			fields: fields{
				request: func(ctrl *gomock.Controller) HTTPRequest {
					mock := mock_driver.NewMockHTTPRequest(ctrl)
					mock.EXPECT().GetHTTPClient().Return(&http.Client{
						Transport: &MockRoundTripper{
							RoundTripFunc: func(req *http.Request) (*http.Response, error) {
								if req.URL.Path == "/v1/fins/statements" {
									respBody := jQuantsFinancialStatementsResponse{
										Statements: []jQuantsFinancialStatement{
											{
												LocalCode:     "10010",
												DisclosedDate: "2023-01-01",
											},
										},
									}
									b, _ := json.Marshal(respBody)
									return &http.Response{
										StatusCode: http.StatusOK,
										Body:       io.NopCloser(bytes.NewReader(b)),
									}, nil
								}
								return &http.Response{
									StatusCode: http.StatusNotFound,
									Body:       io.NopCloser(bytes.NewReader(nil)),
								}, nil
							},
						},
					}).AnyTimes()
					return mock
				},
				redisClient: redisClient,
			},
			args: args{
				ctx:    context.Background(),
				symbol: "1001",
				date:   nil,
			},
			setup: func() {
				s.Set(jQuantsAPIIDTokenRedisKey, "test_token")
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			if tt.setup != nil {
				tt.setup()
			}

			c := &StockAPIClient{
				request:     tt.fields.request(ctrl),
				redisClient: tt.fields.redisClient,
			}
			_, err := c.getFinancialStatementsJQ(tt.args.ctx, tt.args.symbol, tt.args.date)
			if (err != nil) != tt.wantErr {
				t.Errorf("StockAPIClient.getFinancialStatementsJQ() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStockAPIClient_getTradingCalendarsInfo(t *testing.T) {
	// miniredis setup
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run() error = %v", err)
	}
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	type fields struct {
		request     func(ctrl *gomock.Controller) HTTPRequest
		redisClient *redis.Client
	}
	type args struct {
		ctx    context.Context
		filter gateway.TradingCalendarsInfoFilter
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		setup   func()
		wantErr bool
	}{
		{
			name: "正常系",
			fields: fields{
				request: func(ctrl *gomock.Controller) HTTPRequest {
					mock := mock_driver.NewMockHTTPRequest(ctrl)
					mock.EXPECT().GetHTTPClient().Return(&http.Client{
						Transport: &MockRoundTripper{
							RoundTripFunc: func(req *http.Request) (*http.Response, error) {
								if req.URL.Path == "/v1/markets/trading_calendar" {
									respBody := TradingCalendarsResponse{
										TradingCalendars: []TradingCalendar{
											{
												Date:            "2023-01-01",
												HolidayDivision: "1",
											},
										},
									}
									b, _ := json.Marshal(respBody)
									return &http.Response{
										StatusCode: http.StatusOK,
										Body:       io.NopCloser(bytes.NewReader(b)),
									}, nil
								}
								return &http.Response{
									StatusCode: http.StatusNotFound,
									Body:       io.NopCloser(bytes.NewReader(nil)),
								}, nil
							},
						},
					}).AnyTimes()
					return mock
				},
				redisClient: redisClient,
			},
			args: args{
				ctx: context.Background(),
				filter: gateway.TradingCalendarsInfoFilter{
					From: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
					To:   time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC),
				},
			},
			setup: func() {
				s.Set(jQuantsAPIIDTokenRedisKey, "test_token")
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			if tt.setup != nil {
				tt.setup()
			}

			c := &StockAPIClient{
				request:     tt.fields.request(ctrl),
				redisClient: tt.fields.redisClient,
			}
			_, err := c.getTradingCalendarsInfo(tt.args.ctx, tt.args.filter)
			if (err != nil) != tt.wantErr {
				t.Errorf("StockAPIClient.getTradingCalendarsInfo() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
