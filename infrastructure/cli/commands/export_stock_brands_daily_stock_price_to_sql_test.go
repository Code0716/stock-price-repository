package commands

import (
	"flag"
	"testing"

	"github.com/urfave/cli/v2"
	"go.uber.org/mock/gomock"

	mock_usecase "github.com/Code0716/stock-price-repository/mock/usecase"
	"github.com/Code0716/stock-price-repository/usecase"
)

func TestExportStockBrandsAndDailyPriceToSQLV1Command_Action(t *testing.T) {
	type fields struct {
		exportSQLInteractor func(ctrl *gomock.Controller) usecase.ExportSQLInteractor
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
				exportSQLInteractor: func(ctrl *gomock.Controller) usecase.ExportSQLInteractor {
					mock := mock_usecase.NewMockExportSQLInteractor(ctrl)
					mock.EXPECT().ExportSQLFiles(gomock.Any(), gomock.Any()).Return(nil)
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

			c := &ExportStockBrandsAndDailyPriceToSQLV1Command{
				exportSQLInteractor: tt.fields.exportSQLInteractor(ctrl),
			}
			if err := c.Action(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("ExportStockBrandsAndDailyPriceToSQLV1Command.Action() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
