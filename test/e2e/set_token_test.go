package e2e

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/Code0716/stock-price-repository/infrastructure/cli/commands"
	mock_gateway "github.com/Code0716/stock-price-repository/mock/gateway"
	"github.com/Code0716/stock-price-repository/test/helper"
)

func TestE2E_SetJQuantsAPITokenToRedis(t *testing.T) {
	// 1. Setup Redis (Miniredis)
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer mr.Close()

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	type args struct {
		cmdArgs []string
	}
	tests := []struct {
		name    string
		args    args
		setup   func(t *testing.T, mockStockAPI *mock_gateway.MockStockAPIClient, mockSlackAPI *mock_gateway.MockSlackAPIClient)
		wantErr bool
		check   func(t *testing.T)
	}{
		{
			name: "正常系: J-Quants APIトークンがRedisにセットされる",
			args: args{
				cmdArgs: []string{"main", "set_j_quants_api_token_to_redis_v1"},
			},
			setup: func(t *testing.T, mockStockAPI *mock_gateway.MockStockAPIClient, mockSlackAPI *mock_gateway.MockSlackAPIClient) {
				refreshToken := "test-refresh-token"
				idToken := "test-id-token"

				// Setup Expectations
				mockStockAPI.EXPECT().GetOrSetJQuantsAPIIDTokenToRedis(gomock.Any()).DoAndReturn(func(ctx context.Context) (string, error) {
					// Simulate the logic: set tokens to Redis
					err := redisClient.Set(ctx, "j_quants_api_refresh_token", refreshToken, 0).Err()
					if err != nil {
						return "", err
					}
					err = redisClient.Set(ctx, "j_quants_api_id_token", idToken, 0).Err()
					if err != nil {
						return "", err
					}
					return idToken, nil
				})

				// Slack expectations
				mockSlackAPI.EXPECT().SendMessageByStrings(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return("", nil).AnyTimes()
			},
			wantErr: false,
			check: func(t *testing.T) {
				// Verify Redis
				// Check Refresh Token
				val, err := redisClient.Get(context.Background(), "j_quants_api_refresh_token").Result()
				assert.NoError(t, err)
				assert.Equal(t, "test-refresh-token", val)

				// Check ID Token
				val, err = redisClient.Get(context.Background(), "j_quants_api_id_token").Result()
				assert.NoError(t, err)
				assert.Equal(t, "test-id-token", val)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Redis flush
			redisClient.FlushAll(context.Background())

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSlackAPI := mock_gateway.NewMockSlackAPIClient(ctrl)
			mockStockAPI := mock_gateway.NewMockStockAPIClient(ctrl)

			if tt.setup != nil {
				tt.setup(t, mockStockAPI, mockSlackAPI)
			}

			// Commands
			setTokenCmd := commands.NewSetJQuantsAPITokenToRedisV1Command(mockStockAPI)

			runner := helper.NewTestRunner(helper.TestRunnerOptions{
				SetJQuantsAPITokenToRedisV1Command: setTokenCmd,
				SlackAPIClient:                     mockSlackAPI,
			})

			err := runner.Run(context.Background(), tt.args.cmdArgs)
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.check != nil {
				tt.check(t)
			}
		})
	}
}
