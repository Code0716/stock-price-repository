package commands

import (
	"errors"
	"flag"
	"testing"

	"github.com/urfave/cli/v2"
	"go.uber.org/mock/gomock"

	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
	mock_gateway "github.com/Code0716/stock-price-repository/mock/gateway"
)

func TestExportMasterDataCommand_Action(t *testing.T) {
	type fields struct {
		mySQLDumpClient func(ctrl *gomock.Controller) gateway.MySQLDumpClient
		boxClient       func(ctrl *gomock.Controller) gateway.BoxClient
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
				mySQLDumpClient: func(ctrl *gomock.Controller) gateway.MySQLDumpClient {
					mock := mock_gateway.NewMockMySQLDumpClient(ctrl)
					tables := []string{
						gateway.MySQLDumpTableNameStockBrand,
						gateway.MySQLDumpTableNameAppliedStockSplitsHistory,
						gateway.MySQLDumpTableNameNikkeiStockAverageDailyPrice,
						gateway.MySQLDumpTableNameDjiStockAverageDailyStockPrice,
					}
					for _, table := range tables {
						mock.EXPECT().ExportTableAll(gomock.Any(), gomock.Any(), table).Return(nil)
					}
					return mock
				},
				boxClient: func(ctrl *gomock.Controller) gateway.BoxClient {
					mock := mock_gateway.NewMockBoxClient(ctrl)
					mock.EXPECT().UploadFile(gomock.Any(), gomock.Any()).Return(nil).Times(4)
					return mock
				},
			},
			args: args{
				ctx: cli.NewContext(cli.NewApp(), flag.NewFlagSet("test", 0), nil),
			},
			wantErr: false,
		},
		{
			name: "Boxアップロードエラーでもコマンドは成功する",
			fields: fields{
				mySQLDumpClient: func(ctrl *gomock.Controller) gateway.MySQLDumpClient {
					mock := mock_gateway.NewMockMySQLDumpClient(ctrl)
					mock.EXPECT().ExportTableAll(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(4)
					return mock
				},
				boxClient: func(ctrl *gomock.Controller) gateway.BoxClient {
					mock := mock_gateway.NewMockBoxClient(ctrl)
					mock.EXPECT().UploadFile(gomock.Any(), gomock.Any()).Return(
						errors.New("box upload failed"),
					).Times(4)
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

			cmd := NewExportMasterDataCommand(
				tt.fields.mySQLDumpClient(ctrl),
				tt.fields.boxClient(ctrl),
			)
			cliCmd := cmd.Command()
			if err := cliCmd.Action(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("ExportMasterDataCommand.Action() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
