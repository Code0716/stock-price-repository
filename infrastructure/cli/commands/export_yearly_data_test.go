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

func TestExportYearlyDataCommand_Action(t *testing.T) {
	type fields struct {
		mySQLDumpClient func(ctrl *gomock.Controller) gateway.MySQLDumpClient
		boxClient       func(ctrl *gomock.Controller) gateway.BoxClient
	}
	type args struct {
		ctxFunc func() *cli.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "正常系_年指定あり",
			fields: fields{
				mySQLDumpClient: func(ctrl *gomock.Controller) gateway.MySQLDumpClient {
					mock := mock_gateway.NewMockMySQLDumpClient(ctrl)
					year := 2023
					mock.EXPECT().ExportTableByYear(gomock.Any(), gateway.MySQLDumpTableNameStockBrandsDailyPrice, year).Return(nil)
					mock.EXPECT().ExportTableByYear(gomock.Any(), gateway.MySQLDumpTableNameStockBrandsDailyPriceForAnalyze, year).Return(nil)
					return mock
				},
				boxClient: func(ctrl *gomock.Controller) gateway.BoxClient {
					mock := mock_gateway.NewMockBoxClient(ctrl)
					mock.EXPECT().UploadFile(gomock.Any(), gomock.Any()).Return(nil).Times(2)
					return mock
				},
			},
			args: args{
				ctxFunc: func() *cli.Context {
					set := flag.NewFlagSet("test", 0)
					set.Int("year", 0, "year")
					set.Parse([]string{"--year", "2023"})
					return cli.NewContext(cli.NewApp(), set, nil)
				},
			},
			wantErr: false,
		},
		{
			name: "正常系_年未指定_全年自動検出",
			fields: fields{
				mySQLDumpClient: func(ctrl *gomock.Controller) gateway.MySQLDumpClient {
					mock := mock_gateway.NewMockMySQLDumpClient(ctrl)
					mock.EXPECT().GetDistinctYears(gomock.Any(), gateway.MySQLDumpTableNameStockBrandsDailyPrice).
						Return([]int{2023, 2024}, nil)
					for _, year := range []int{2023, 2024} {
						mock.EXPECT().ExportTableByYear(gomock.Any(), gateway.MySQLDumpTableNameStockBrandsDailyPrice, year).Return(nil)
						mock.EXPECT().ExportTableByYear(gomock.Any(), gateway.MySQLDumpTableNameStockBrandsDailyPriceForAnalyze, year).Return(nil)
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
				ctxFunc: func() *cli.Context {
					set := flag.NewFlagSet("test", 0)
					set.Int("year", 0, "year")
					set.Parse([]string{})
					return cli.NewContext(cli.NewApp(), set, nil)
				},
			},
			wantErr: false,
		},
		{
			name: "異常系_GetDistinctYearsエラー",
			fields: fields{
				mySQLDumpClient: func(ctrl *gomock.Controller) gateway.MySQLDumpClient {
					mock := mock_gateway.NewMockMySQLDumpClient(ctrl)
					mock.EXPECT().GetDistinctYears(gomock.Any(), gomock.Any()).
						Return(nil, errors.New("db error"))
					return mock
				},
				boxClient: func(ctrl *gomock.Controller) gateway.BoxClient {
					return mock_gateway.NewMockBoxClient(ctrl)
				},
			},
			args: args{
				ctxFunc: func() *cli.Context {
					set := flag.NewFlagSet("test", 0)
					set.Int("year", 0, "year")
					set.Parse([]string{})
					return cli.NewContext(cli.NewApp(), set, nil)
				},
			},
			wantErr: true,
		},
		{
			name: "正常系_Boxアップロードエラーでもコマンドは成功する",
			fields: fields{
				mySQLDumpClient: func(ctrl *gomock.Controller) gateway.MySQLDumpClient {
					mock := mock_gateway.NewMockMySQLDumpClient(ctrl)
					year := 2023
					mock.EXPECT().ExportTableByYear(gomock.Any(), gomock.Any(), year).Return(nil).Times(2)
					return mock
				},
				boxClient: func(ctrl *gomock.Controller) gateway.BoxClient {
					mock := mock_gateway.NewMockBoxClient(ctrl)
					mock.EXPECT().UploadFile(gomock.Any(), gomock.Any()).Return(
						errors.New("box upload failed"),
					).Times(2)
					return mock
				},
			},
			args: args{
				ctxFunc: func() *cli.Context {
					set := flag.NewFlagSet("test", 0)
					set.Int("year", 0, "year")
					set.Parse([]string{"--year", "2023"})
					return cli.NewContext(cli.NewApp(), set, nil)
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			cmd := NewExportYearlyDataCommand(
				tt.fields.mySQLDumpClient(ctrl),
				tt.fields.boxClient(ctrl),
			)
			cliCmd := cmd.Command()
			if err := cliCmd.Action(tt.args.ctxFunc()); (err != nil) != tt.wantErr {
				t.Errorf("ExportYearlyDataCommand.Action() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
