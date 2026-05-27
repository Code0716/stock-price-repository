package commands

import (
	"flag"
	"testing"

	"github.com/urfave/cli/v2"
	"go.uber.org/mock/gomock"

	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
	mock_gateway "github.com/Code0716/stock-price-repository/mock/gateway"
)

func TestHealthCheckCommand_Action(t *testing.T) {
	type fields struct {
		slackAPIClient func(ctrl *gomock.Controller) gateway.SlackAPIClient
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
				slackAPIClient: func(ctrl *gomock.Controller) gateway.SlackAPIClient {
					mock := mock_gateway.NewMockSlackAPIClient(ctrl)
					mock.EXPECT().SendMessage(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
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

			c := &HealthCheckCommand{
				slackAPIClient: tt.fields.slackAPIClient(ctrl),
			}
			if err := c.Action(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("HealthCheckCommand.Action() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
