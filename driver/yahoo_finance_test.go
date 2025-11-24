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
	"go.uber.org/mock/gomock"

	"github.com/Code0716/stock-price-repository/config"
	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
	mock_driver "github.com/Code0716/stock-price-repository/mock/driver"
)

func TestStockAPIClient_GetStockPriceChart(t *testing.T) {
	// miniredis setup
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run() error = %v", err)
	}
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	config.GetYahooFinance().BaseURL = "https://query1.finance.yahoo.com"

	type fields struct {
		request     func(ctrl *gomock.Controller) HTTPRequest
		redisClient *redis.Client
	}
	type args struct {
		ctx       context.Context
		symbol    gateway.StockAPISymbol
		interval  gateway.StockAPIInterval
		dateRange gateway.StockAPIValidRange
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
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
								if req.URL.Path == "/v8/finance/chart/1001.T" {
									respBody := map[string]any{
										"chart": map[string]any{
											"result": []map[string]any{
												{
													"meta": map[string]any{
														"symbol":          "1001.T",
														"instrumentType":  "EQUITY",
														"dataGranularity": "1d",
														"range":           "1mo",
													},
													"timestamp": []int64{1672531200}, // 2023-01-01
													"indicators": map[string]any{
														"quote": []map[string]any{
															{
																"open":   []float64{100.0},
																"high":   []float64{110.0},
																"low":    []float64{90.0},
																"close":  []float64{105.0},
																"volume": []int64{1000},
															},
														},
														"adjclose": []map[string]any{
															{
																"adjclose": []float64{105.0},
															},
														},
													},
												},
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
				ctx:       context.Background(),
				symbol:    "1001",
				interval:  gateway.StockAPIInterval1D,
				dateRange: gateway.StockAPIValidRange1MO,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			c := &StockAPIClient{
				request:     tt.fields.request(ctrl),
				redisClient: tt.fields.redisClient,
			}
			_, err := c.GetStockPriceChart(tt.args.ctx, tt.args.symbol, tt.args.interval, tt.args.dateRange)
			if (err != nil) != tt.wantErr {
				t.Errorf("StockAPIClient.GetStockPriceChart() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStockAPIClient_GetIndexPriceChart(t *testing.T) {
	// miniredis setup
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run() error = %v", err)
	}
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	config.GetYahooFinance().BaseURL = "https://query1.finance.yahoo.com"

	type fields struct {
		request     func(ctrl *gomock.Controller) HTTPRequest
		redisClient *redis.Client
	}
	type args struct {
		ctx       context.Context
		symbol    gateway.StockAPISymbol
		interval  gateway.StockAPIInterval
		dateRange gateway.StockAPIValidRange
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
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
								if req.URL.Path == "/v8/finance/chart/^N225" {
									respBody := map[string]any{
										"chart": map[string]any{
											"result": []map[string]any{
												{
													"meta": map[string]any{
														"symbol":          "^N225",
														"instrumentType":  "INDEX",
														"dataGranularity": "1d",
														"range":           "1mo",
													},
													"timestamp": []int64{1672531200},
													"indicators": map[string]any{
														"quote": []map[string]any{
															{
																"open":   []float64{26000.0},
																"high":   []float64{26500.0},
																"low":    []float64{25500.0},
																"close":  []float64{26000.0},
																"volume": []int64{100000},
															},
														},
														"adjclose": []map[string]any{
															{
																"adjclose": []float64{26000.0},
															},
														},
													},
												},
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
				ctx:       context.Background(),
				symbol:    "N225",
				interval:  gateway.StockAPIInterval1D,
				dateRange: gateway.StockAPIValidRange1MO,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			c := &StockAPIClient{
				request:     tt.fields.request(ctrl),
				redisClient: tt.fields.redisClient,
			}
			_, err := c.GetIndexPriceChart(tt.args.ctx, tt.args.symbol, tt.args.interval, tt.args.dateRange)
			if (err != nil) != tt.wantErr {
				t.Errorf("StockAPIClient.GetIndexPriceChart() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStockAPIClient_GetWeeklyIndexPriceChart(t *testing.T) {
	// miniredis setup
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run() error = %v", err)
	}
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	config.GetYahooFinance().BaseURL = "https://query1.finance.yahoo.com"

	type fields struct {
		request     func(ctrl *gomock.Controller) HTTPRequest
		redisClient *redis.Client
	}
	type args struct {
		ctx       context.Context
		symbol    gateway.StockAPISymbol
		dateRange gateway.StockAPIValidRange
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
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
								// Weekly chart request
								if req.URL.Path == "/v8/finance/chart/^N225" && req.URL.Query().Get("interval") == "1wk" {
									// Generate enough data points for 3 months (approx 14 weeks)
									timestamps := make([]int64, 15)
									opens := make([]float64, 15)
									highs := make([]float64, 15)
									lows := make([]float64, 15)
									closes := make([]float64, 15)
									volumes := make([]int64, 15)
									adjCloses := make([]float64, 15)

									baseTime := int64(1672531200) // 2023-01-01
									for i := 0; i < 15; i++ {
										timestamps[i] = baseTime + int64(i*7*24*3600)
										opens[i] = 26000.0
										highs[i] = 26500.0
										lows[i] = 25500.0
										closes[i] = 26000.0
										volumes[i] = 100000
										adjCloses[i] = 26000.0
									}

									respBody := map[string]any{
										"chart": map[string]any{
											"result": []map[string]any{
												{
													"meta": map[string]any{
														"symbol":          "^N225",
														"instrumentType":  "INDEX",
														"dataGranularity": "1wk",
														"range":           "3mo",
													},
													"timestamp": timestamps,
													"indicators": map[string]any{
														"quote": []map[string]any{
															{
																"open":   opens,
																"high":   highs,
																"low":    lows,
																"close":  closes,
																"volume": volumes,
															},
														},
														"adjclose": []map[string]any{
															{
																"adjclose": adjCloses,
															},
														},
													},
												},
											},
										},
									}
									b, _ := json.Marshal(respBody)
									return &http.Response{
										StatusCode: http.StatusOK,
										Body:       io.NopCloser(bytes.NewReader(b)),
									}, nil
								}
								// Daily chart request (for current price)
								if req.URL.Path == "/v8/finance/chart/^N225" && req.URL.Query().Get("interval") == "1d" {
									respBody := map[string]any{
										"chart": map[string]any{
											"result": []map[string]any{
												{
													"meta": map[string]any{
														"symbol":          "^N225",
														"instrumentType":  "INDEX",
														"dataGranularity": "1d",
														"range":           "1d",
													},
													"timestamp": []int64{time.Now().Unix()},
													"indicators": map[string]any{
														"quote": []map[string]any{
															{
																"open":   []float64{26100.0},
																"high":   []float64{26600.0},
																"low":    []float64{25600.0},
																"close":  []float64{26100.0},
																"volume": []int64{100000},
															},
														},
														"adjclose": []map[string]any{
															{
																"adjclose": []float64{26100.0},
															},
														},
													},
												},
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
				ctx:       context.Background(),
				symbol:    "^N225",
				dateRange: gateway.StockAPIValidRange3MO,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			c := &StockAPIClient{
				request:     tt.fields.request(ctrl),
				redisClient: tt.fields.redisClient,
			}
			_, err := c.GetWeeklyIndexPriceChart(tt.args.ctx, tt.args.symbol, tt.args.dateRange)
			if (err != nil) != tt.wantErr {
				t.Errorf("StockAPIClient.GetWeeklyIndexPriceChart() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
