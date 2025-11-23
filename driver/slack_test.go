package driver

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"go.uber.org/mock/gomock"

	"github.com/Code0716/stock-price-repository/config"
	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
	"github.com/Code0716/stock-price-repository/infrastructure/gateway/resource"
	mock_driver "github.com/Code0716/stock-price-repository/mock/driver"
)

func TestSlackAPIClient_SendMessage(t *testing.T) {
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run() error = %v", err)
	}
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	config.Slack().SlackBotBaseUrl = "https://slack.com/api/chat.postMessage"
	config.Slack().SlackNotificationBotToken = "test-token"

	type fields struct {
		request     func(ctrl *gomock.Controller) HTTPRequest
		redisClient *redis.Client
	}
	type args struct {
		ctx         context.Context
		channelName gateway.SlackChannelName
		message     resource.SlackMessage
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
					mock.EXPECT().GetHttpClient().Return(&http.Client{
						Transport: &MockRoundTripper{
							RoundTripFunc: func(req *http.Request) (*http.Response, error) {
								if req.URL.String() == "https://slack.com/api/chat.postMessage" {
									return &http.Response{
										StatusCode: http.StatusOK,
										Body: io.NopCloser(bytes.NewBufferString(`{
											"ok": true,
											"channel": "C12345678",
											"ts": "1503435956.000247",
											"message": {
												"text": "Hello World",
												"username": "bot",
												"bot_id": "B12345678",
												"type": "message",
												"subtype": "bot_message",
												"ts": "1503435956.000247"
											}
										}`)),
									}, nil
								}
								return &http.Response{
									StatusCode: http.StatusNotFound,
									Body:       io.NopCloser(bytes.NewReader(nil)),
								}, nil
							},
						},
					})
					return mock
				},
				redisClient: redisClient,
			},
			args: args{
				ctx:         context.Background(),
				channelName: gateway.SlackChannelNameDevNotification,
				message:     resource.SlackMessageHealthCheck,
			},
			wantErr: false,
		},
		{
			name: "異常系: Slack API エラー",
			fields: fields{
				request: func(ctrl *gomock.Controller) HTTPRequest {
					mock := mock_driver.NewMockHTTPRequest(ctrl)
					mock.EXPECT().GetHttpClient().Return(&http.Client{
						Transport: &MockRoundTripper{
							RoundTripFunc: func(req *http.Request) (*http.Response, error) {
								return &http.Response{
									StatusCode: http.StatusOK,
									Body: io.NopCloser(bytes.NewBufferString(`{
										"ok": false,
										"error": "invalid_auth"
									}`)),
								}, nil
							},
						},
					})
					return mock
				},
				redisClient: redisClient,
			},
			args: args{
				ctx:         context.Background(),
				channelName: gateway.SlackChannelNameDevNotification,
				message:     resource.SlackMessageHealthCheck,
			},
			wantErr: true,
		},
		{
			name: "異常系: HTTP エラー",
			fields: fields{
				request: func(ctrl *gomock.Controller) HTTPRequest {
					mock := mock_driver.NewMockHTTPRequest(ctrl)
					mock.EXPECT().GetHttpClient().Return(&http.Client{
						Transport: &MockRoundTripper{
							RoundTripFunc: func(req *http.Request) (*http.Response, error) {
								return nil, errors.New("http error")
							},
						},
					})
					return mock
				},
				redisClient: redisClient,
			},
			args: args{
				ctx:         context.Background(),
				channelName: gateway.SlackChannelNameDevNotification,
				message:     resource.SlackMessageHealthCheck,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			c := &SlackAPIClient{
				request:     tt.fields.request(ctrl),
				redisClient: tt.fields.redisClient,
			}
			if err := c.SendMessage(tt.args.ctx, tt.args.channelName, tt.args.message); (err != nil) != tt.wantErr {
				t.Errorf("SlackAPIClient.SendMessage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSlackAPIClient_SendErrMessageNotification(t *testing.T) {
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run() error = %v", err)
	}
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	config.Slack().SlackBotBaseUrl = "https://slack.com/api/chat.postMessage"
	config.Slack().SlackNotificationBotToken = "test-token"

	type fields struct {
		request     func(ctrl *gomock.Controller) HTTPRequest
		redisClient *redis.Client
	}
	type args struct {
		ctx context.Context
		err error
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
					mock.EXPECT().GetHttpClient().Return(&http.Client{
						Transport: &MockRoundTripper{
							RoundTripFunc: func(req *http.Request) (*http.Response, error) {
								return &http.Response{
									StatusCode: http.StatusOK,
									Body: io.NopCloser(bytes.NewBufferString(`{
										"ok": true,
										"ts": "1503435956.000247"
									}`)),
								}, nil
							},
						},
					})
					return mock
				},
				redisClient: redisClient,
			},
			args: args{
				ctx: context.Background(),
				err: errors.New("test error"),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			c := &SlackAPIClient{
				request:     tt.fields.request(ctrl),
				redisClient: tt.fields.redisClient,
			}
			if err := c.SendErrMessageNotification(tt.args.ctx, tt.args.err); (err != nil) != tt.wantErr {
				t.Errorf("SlackAPIClient.SendErrMessageNotification() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSlackAPIClient_SendMessageByStrings(t *testing.T) {
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run() error = %v", err)
	}
	defer s.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: s.Addr(),
	})

	config.Slack().SlackBotBaseUrl = "https://slack.com/api/chat.postMessage"
	config.Slack().SlackNotificationBotToken = "test-token"

	message := "test message"
	ts := "1234567890.123456"

	type fields struct {
		request     func(ctrl *gomock.Controller) HTTPRequest
		redisClient *redis.Client
	}
	type args struct {
		ctx         context.Context
		channelName gateway.SlackChannelName
		title       string
		message     *string
		ts          *string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "正常系: 新規スレッド作成 (ts is nil)",
			fields: fields{
				request: func(ctrl *gomock.Controller) HTTPRequest {
					mock := mock_driver.NewMockHTTPRequest(ctrl)
					// 1回目: タイトル送信 (スレッド親)
					// 2回目: メッセージ送信 (スレッド返信)
					mock.EXPECT().GetHttpClient().Return(&http.Client{
						Transport: &MockRoundTripper{
							RoundTripFunc: func(req *http.Request) (*http.Response, error) {
								return &http.Response{
									StatusCode: http.StatusOK,
									Body: io.NopCloser(bytes.NewBufferString(`{
										"ok": true,
										"ts": "1234567890.123456"
									}`)),
								}, nil
							},
						},
					}).Times(2)
					return mock
				},
				redisClient: redisClient,
			},
			args: args{
				ctx:         context.Background(),
				channelName: gateway.SlackChannelNameDevNotification,
				title:       "Test Title",
				message:     &message,
				ts:          nil,
			},
			want:    "1234567890.123456",
			wantErr: false,
		},
		{
			name: "正常系: 既存スレッドへの返信 (ts is not nil)",
			fields: fields{
				request: func(ctrl *gomock.Controller) HTTPRequest {
					mock := mock_driver.NewMockHTTPRequest(ctrl)
					// 1回目: メッセージ送信 (スレッド返信) のみ
					mock.EXPECT().GetHttpClient().Return(&http.Client{
						Transport: &MockRoundTripper{
							RoundTripFunc: func(req *http.Request) (*http.Response, error) {
								return &http.Response{
									StatusCode: http.StatusOK,
									Body: io.NopCloser(bytes.NewBufferString(`{
										"ok": true,
										"ts": "1234567890.123456"
									}`)),
								}, nil
							},
						},
					}).Times(1)
					return mock
				},
				redisClient: redisClient,
			},
			args: args{
				ctx:         context.Background(),
				channelName: gateway.SlackChannelNameDevNotification,
				title:       "Test Title",
				message:     &message,
				ts:          &ts,
			},
			want:    "1234567890.123456",
			wantErr: false,
		},
		{
			name: "正常系: タイトルのみ (message is nil)",
			fields: fields{
				request: func(ctrl *gomock.Controller) HTTPRequest {
					mock := mock_driver.NewMockHTTPRequest(ctrl)
					// 1回目: タイトル送信 (スレッド親) のみ
					mock.EXPECT().GetHttpClient().Return(&http.Client{
						Transport: &MockRoundTripper{
							RoundTripFunc: func(req *http.Request) (*http.Response, error) {
								return &http.Response{
									StatusCode: http.StatusOK,
									Body: io.NopCloser(bytes.NewBufferString(`{
										"ok": true,
										"ts": "1234567890.123456"
									}`)),
								}, nil
							},
						},
					}).Times(1)
					return mock
				},
				redisClient: redisClient,
			},
			args: args{
				ctx:         context.Background(),
				channelName: gateway.SlackChannelNameDevNotification,
				title:       "Test Title",
				message:     nil,
				ts:          nil,
			},
			want:    "1234567890.123456",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			c := &SlackAPIClient{
				request:     tt.fields.request(ctrl),
				redisClient: tt.fields.redisClient,
			}
			got, err := c.SendMessageByStrings(tt.args.ctx, tt.args.channelName, tt.args.title, tt.args.message, tt.args.ts)
			if (err != nil) != tt.wantErr {
				t.Errorf("SlackAPIClient.SendMessageByStrings() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("SlackAPIClient.SendMessageByStrings() = %v, want %v", got, tt.want)
			}
		})
	}
}
