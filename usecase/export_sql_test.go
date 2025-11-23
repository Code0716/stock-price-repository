package usecase

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
	mock_gateway "github.com/Code0716/stock-price-repository/mock/gateway"
	"github.com/Code0716/stock-price-repository/util"
)

func Test_exportSQLInteractorImpl_ExportSQLFiles(t *testing.T) {
	type fields struct {
		mySQLDumpClient func(ctrl *gomock.Controller) *mock_gateway.MockMySQLDumpClient
	}
	type args struct {
		ctx context.Context
		now time.Time
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
				mySQLDumpClient: func(ctrl *gomock.Controller) *mock_gateway.MockMySQLDumpClient {
					mock := mock_gateway.NewMockMySQLDumpClient(ctrl)
					now := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
					dateStr := util.DatetimeToFileNameDateStr(now)

					// Parallel exports
					mock.EXPECT().ExportTableAll(gomock.Any(), fmt.Sprintf("%s_%s", dateStr, gateway.MySQLDumpTableNameNikkeiStockAverageDailyPrice), gateway.MySQLDumpTableNameNikkeiStockAverageDailyPrice).Return(nil)
					mock.EXPECT().ExportTableAll(gomock.Any(), fmt.Sprintf("%s_%s", dateStr, gateway.MySQLDumpTableNameDjiStockAverageDailyStockPrice), gateway.MySQLDumpTableNameDjiStockAverageDailyStockPrice).Return(nil)
					mock.EXPECT().ExportTableAll(gomock.Any(), fmt.Sprintf("%s_%s", dateStr, gateway.MySQLDumpTableNameStockBrand), gateway.MySQLDumpTableNameStockBrand).Return(nil)
					mock.EXPECT().ExportTableAll(gomock.Any(), fmt.Sprintf("%s_%s", dateStr, gateway.MySQLDumpTableNameSector33AverageDailyPrice), gateway.MySQLDumpTableNameSector33AverageDailyPrice).Return(nil)

					// Sequential exports
					for year := 2019; year <= 2023; year++ {
						mock.EXPECT().ExportDailyStockPriceByYear(gomock.Any(), year).Return(nil)
					}

					return mock
				},
			},
			args: args{
				ctx: context.Background(),
				now: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			wantErr: false,
		},
		{
			name: "異常系: ExportTableAllエラー (一部失敗しても続行するが、ログ出力のみでエラーは返さない仕様に見えるが...)",
			// コードを見ると、errChにエラーを入れて、最後にループで取り出しているが、
			// エラーがあっても return していない。
			// しかし、その後の ExportDailyStockPriceByYear でエラーがあれば return している。
			// 並列処理部分はエラーがあっても無視して進む実装になっているようだ。
			fields: fields{
				mySQLDumpClient: func(ctrl *gomock.Controller) *mock_gateway.MockMySQLDumpClient {
					mock := mock_gateway.NewMockMySQLDumpClient(ctrl)
					now := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
					dateStr := util.DatetimeToFileNameDateStr(now)

					// Parallel exports - one fails
					mock.EXPECT().ExportTableAll(gomock.Any(), fmt.Sprintf("%s_%s", dateStr, gateway.MySQLDumpTableNameNikkeiStockAverageDailyPrice), gateway.MySQLDumpTableNameNikkeiStockAverageDailyPrice).Return(errors.New("export error"))
					mock.EXPECT().ExportTableAll(gomock.Any(), fmt.Sprintf("%s_%s", dateStr, gateway.MySQLDumpTableNameDjiStockAverageDailyStockPrice), gateway.MySQLDumpTableNameDjiStockAverageDailyStockPrice).Return(nil)
					mock.EXPECT().ExportTableAll(gomock.Any(), fmt.Sprintf("%s_%s", dateStr, gateway.MySQLDumpTableNameStockBrand), gateway.MySQLDumpTableNameStockBrand).Return(nil)
					mock.EXPECT().ExportTableAll(gomock.Any(), fmt.Sprintf("%s_%s", dateStr, gateway.MySQLDumpTableNameSector33AverageDailyPrice), gateway.MySQLDumpTableNameSector33AverageDailyPrice).Return(nil)

					// Sequential exports
					for year := 2019; year <= 2023; year++ {
						mock.EXPECT().ExportDailyStockPriceByYear(gomock.Any(), year).Return(nil)
					}

					return mock
				},
			},
			args: args{
				ctx: context.Background(),
				now: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			wantErr: false, // 並列処理のエラーはログ出力のみで、関数としては成功する
		},
		{
			name: "異常系: ExportDailyStockPriceByYearエラー",
			fields: fields{
				mySQLDumpClient: func(ctrl *gomock.Controller) *mock_gateway.MockMySQLDumpClient {
					mock := mock_gateway.NewMockMySQLDumpClient(ctrl)
					now := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
					dateStr := util.DatetimeToFileNameDateStr(now)

					// Parallel exports
					mock.EXPECT().ExportTableAll(gomock.Any(), fmt.Sprintf("%s_%s", dateStr, gateway.MySQLDumpTableNameNikkeiStockAverageDailyPrice), gateway.MySQLDumpTableNameNikkeiStockAverageDailyPrice).Return(nil)
					mock.EXPECT().ExportTableAll(gomock.Any(), fmt.Sprintf("%s_%s", dateStr, gateway.MySQLDumpTableNameDjiStockAverageDailyStockPrice), gateway.MySQLDumpTableNameDjiStockAverageDailyStockPrice).Return(nil)
					mock.EXPECT().ExportTableAll(gomock.Any(), fmt.Sprintf("%s_%s", dateStr, gateway.MySQLDumpTableNameStockBrand), gateway.MySQLDumpTableNameStockBrand).Return(nil)
					mock.EXPECT().ExportTableAll(gomock.Any(), fmt.Sprintf("%s_%s", dateStr, gateway.MySQLDumpTableNameSector33AverageDailyPrice), gateway.MySQLDumpTableNameSector33AverageDailyPrice).Return(nil)

					// Sequential exports - fails at 2019
					mock.EXPECT().ExportDailyStockPriceByYear(gomock.Any(), 2019).Return(errors.New("export error"))

					return mock
				},
			},
			args: args{
				ctx: context.Background(),
				now: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			ei := &exportSQLInteractorImpl{
				mySQLDumpClient: tt.fields.mySQLDumpClient(ctrl),
			}
			if err := ei.ExportSQLFiles(tt.args.ctx, tt.args.now); (err != nil) != tt.wantErr {
				t.Errorf("exportSQLInteractorImpl.ExportSQLFiles() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_exportSQLInteractorImpl_exportTableAll(t *testing.T) {
	type fields struct {
		mySQLDumpClient func(ctrl *gomock.Controller) *mock_gateway.MockMySQLDumpClient
	}
	type args struct {
		ctx       context.Context
		tableName string
		now       time.Time
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
				mySQLDumpClient: func(ctrl *gomock.Controller) *mock_gateway.MockMySQLDumpClient {
					mock := mock_gateway.NewMockMySQLDumpClient(ctrl)
					now := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
					dateStr := util.DatetimeToFileNameDateStr(now)
					tableName := "test_table"
					mock.EXPECT().ExportTableAll(gomock.Any(), fmt.Sprintf("%s_%s", dateStr, tableName), tableName).Return(nil)
					return mock
				},
			},
			args: args{
				ctx:       context.Background(),
				tableName: "test_table",
				now:       time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			wantErr: false,
		},
		{
			name: "異常系",
			fields: fields{
				mySQLDumpClient: func(ctrl *gomock.Controller) *mock_gateway.MockMySQLDumpClient {
					mock := mock_gateway.NewMockMySQLDumpClient(ctrl)
					mock.EXPECT().ExportTableAll(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("export error"))
					return mock
				},
			},
			args: args{
				ctx:       context.Background(),
				tableName: "test_table",
				now:       time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			ei := &exportSQLInteractorImpl{
				mySQLDumpClient: tt.fields.mySQLDumpClient(ctrl),
			}
			if err := ei.exportTableAll(tt.args.ctx, tt.args.tableName, tt.args.now); (err != nil) != tt.wantErr {
				t.Errorf("exportSQLInteractorImpl.exportTableAll() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
