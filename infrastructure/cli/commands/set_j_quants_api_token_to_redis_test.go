package commands

import (
	"flag"
	"testing"

	"github.com/urfave/cli/v2"
	"go.uber.org/mock/gomock"

	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
	mock_gateway "github.com/Code0716/stock-price-repository/mock/gateway"
)

func TestSetJQuantsAPITokenToRedisV1Command_Action(t *testing.T) {
	type fields struct {
		stockAPIClient func(ctrl *gomock.Controller) gateway.StockAPIClient
	}
	type args struct {
		ctx *cli.Context
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
				stockAPIClient: func(ctrl *gomock.Controller) gateway.StockAPIClient {
					mock := mock_gateway.NewMockStockAPIClient(ctrl)
					mock.EXPECT().GetOrSetJQuantsAPIIDTokenToRedis(gomock.Any()).Return("", nil)
					return mock
				},
			},
			args: args{
				ctx: cli.NewContext(cli.NewApp(), flag.NewFlagSet("test", 0), nil),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			c := &SetJQuantsAPITokenToRedisV1Command{
				stockAPIClient: tt.fields.stockAPIClient(ctrl),
			}
			if err := c.Action(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("SetJQuantsAPITokenToRedisV1Command.Action() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
