package commands

import (
	"flag"
	"testing"

	"github.com/urfave/cli/v2"
	"go.uber.org/mock/gomock"

	mock_usecase "github.com/Code0716/stock-price-repository/mock/usecase"
	"github.com/Code0716/stock-price-repository/usecase"
)

func TestCreateNikkeiAndDjiHistoricalDataV1Command_Action(t *testing.T) {
	type fields struct {
		nikkeiInteractor func(ctrl *gomock.Controller) usecase.IndexInteractor
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
				nikkeiInteractor: func(ctrl *gomock.Controller) usecase.IndexInteractor {
					mock := mock_usecase.NewMockIndexInteractor(ctrl)
					mock.EXPECT().CreateNikkeiAndDjiHistoricalData(gomock.Any(), gomock.Any()).Return(nil)
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

			c := &CreateNikkeiAndDjiHistoricalDataV1Command{
				nikkeiInteractor: tt.fields.nikkeiInteractor(ctrl),
			}
			if err := c.Action(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("CreateNikkeiAndDjiHistoricalDataV1Command.Action() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
