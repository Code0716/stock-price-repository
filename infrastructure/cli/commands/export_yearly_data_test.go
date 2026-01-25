package commands

import (
	"flag"
	"testing"
	"time"

	"github.com/urfave/cli/v2"
	"go.uber.org/mock/gomock"

	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
	mock_gateway "github.com/Code0716/stock-price-repository/mock/gateway"
)

func TestExportYearlyDataCommand_Action(t *testing.T) {
	type fields struct {
		mySQLDumpClient func(ctrl *gomock.Controller) gateway.MySQLDumpClient
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
					// StockBrandsDailyPrice
					mock.EXPECT().ExportTableByYear(
						gomock.Any(),
						gateway.MySQLDumpTableNameStockBrandsDailyPrice,
						year,
					).Return(nil)
					// StockBrandsDailyPriceForAnalyze
					mock.EXPECT().ExportTableByYear(
						gomock.Any(),
						gateway.MySQLDumpTableNameStockBrandsDailyPriceForAnalyze,
						year,
					).Return(nil)
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
			name: "正常系_年指定なし(現在年)",
			fields: fields{
				mySQLDumpClient: func(ctrl *gomock.Controller) gateway.MySQLDumpClient {
					mock := mock_gateway.NewMockMySQLDumpClient(ctrl)
					year := time.Now().Year()
					// StockBrandsDailyPrice
					mock.EXPECT().ExportTableByYear(
						gomock.Any(),
						gateway.MySQLDumpTableNameStockBrandsDailyPrice,
						year,
					).Return(nil)
					// StockBrandsDailyPriceForAnalyze
					mock.EXPECT().ExportTableByYear(
						gomock.Any(),
						gateway.MySQLDumpTableNameStockBrandsDailyPriceForAnalyze,
						year,
					).Return(nil)
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			cmd := NewExportYearlyDataCommand(
				tt.fields.mySQLDumpClient(ctrl),
			)
			cliCmd := cmd.Command()
			if err := cliCmd.Action(tt.args.ctxFunc()); (err != nil) != tt.wantErr {
				t.Errorf("ExportYearlyDataCommand.Action() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}