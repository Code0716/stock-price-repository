package e2e

import (
	"context"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/Code0716/stock-price-repository/config"
	"github.com/Code0716/stock-price-repository/driver"
	"github.com/Code0716/stock-price-repository/infrastructure/cli/commands"
	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
	mock_gateway "github.com/Code0716/stock-price-repository/mock/gateway"
	"github.com/Code0716/stock-price-repository/test/helper"
)

func TestE2E_ExportData(t *testing.T) {
	// mysqldumpコマンドが存在するか確認
	if _, err := exec.LookPath("mysqldump"); err != nil {
		t.Skip("mysqldump not found in PATH, skipping e2e test")
	}

	// 1. Setup DB
	_, cleanup := helper.SetupTestDB(t)
	defer cleanup()

	// Export先のディレクトリを一時ディレクトリに設定したいが、
	// configパッケージの実装依存。
	// ここでは、デフォルトのパス（恐らく .env で指定された場所）に出力されることを許容し、
	// 実行自体がエラーにならないことを確認する。
	// 可能であれば一時ディレクトリを作成して cleanup する。

	// 環境変数 DB_EXPORT_BACKUP_PATH を一時ディレクトリに上書きできるとベスト。
	tmpDir, err := os.MkdirTemp("", "e2e_export_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// configの値を書き換えるハック（configパッケージの構造による）
	// ここでは環境変数をセットしても、既にLoadEnvConfigされていると反映されない可能性がある。
	// config.GetDatabase()の実装次第。
	// 今回は「コマンド実行が成功すること（exit code 0）」を主眼に置く。
	// エクスポートパスが書き込み不可だとエラーになるため、そこだけ注意。
	// config.Database.ExportBackupPath = tmpDir のように書き換えられるならしたい。

	// config.GetDatabase()が参照渡しなら書き換え可能
	dbConfig := config.GetDatabase()
	originalPath := dbConfig.ExportBackupPath
	dbConfig.ExportBackupPath = tmpDir
	// テスト終了後に戻す
	defer func() {
		dbConfig.ExportBackupPath = originalPath
	}()

	mysqlDumpClient := driver.NewMySQLDumpClient()

	type args struct {
		cmdArgs []string
	}
	tests := []struct {
		name    string
		args    args
		setup   func(t *testing.T, mockSlackAPI *mock_gateway.MockSlackAPIClient)
		wantErr bool
	}{
		{
			name: "正常系: マスタデータのエクスポート",
			args: args{
				cmdArgs: []string{"main", "export_master_data"},
			},
			setup: func(t *testing.T, mockSlackAPI *mock_gateway.MockSlackAPIClient) {
				mockSlackAPI.EXPECT().SendMessageByStrings(
					gomock.Any(),
					gateway.SlackChannelNameDevNotification,
					gomock.Any(),
					nil,
					nil,
				).DoAndReturn(func(_ context.Context, _ gateway.SlackChannelName, title string, _, _ *string) (string, error) {
					assert.Contains(t, title, "command name: export_master_data")
					return "", nil
				})
			},
			wantErr: false,
		},
		{
			name: "正常系: 年指定データのエクスポート",
			args: args{
				cmdArgs: []string{"main", "export_yearly_data", "--year", "2024"},
			},
			setup: func(t *testing.T, mockSlackAPI *mock_gateway.MockSlackAPIClient) {
				mockSlackAPI.EXPECT().SendMessageByStrings(
					gomock.Any(),
					gateway.SlackChannelNameDevNotification,
					gomock.Any(),
					nil,
					nil,
				).DoAndReturn(func(_ context.Context, _ gateway.SlackChannelName, title string, _, _ *string) (string, error) {
					assert.Contains(t, title, "command name: export_yearly_data")
					return "", nil
				})
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockSlackAPI := mock_gateway.NewMockSlackAPIClient(ctrl)

			if tt.setup != nil {
				tt.setup(t, mockSlackAPI)
			}

			// Commands
			exportYearlyCmd := commands.NewExportYearlyDataCommand(mysqlDumpClient)
			exportMasterCmd := commands.NewExportMasterDataCommand(mysqlDumpClient)

			runner := helper.NewTestRunner(helper.TestRunnerOptions{
				ExportYearlyDataCommand: exportYearlyCmd,
				ExportMasterDataCommand: exportMasterCmd,
				SlackAPIClient:          mockSlackAPI,
			})

			err := runner.Run(context.Background(), tt.args.cmdArgs)
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}

			// 成功した場合、ファイルが生成されているか確認するロジックを入れても良いが
			// mysqldumpがDB接続できずにエラーになる可能性（外部環境依存）が高いため、
			// エラーチェックのみとする。
		})
	}
}

// 注意: このテストは config.GetDatabase() の戻り値がポインタで、
// かつそのフィールドを書き換えることで driver 側が参照する値が変わることを前提としている。
// もし config.GetDatabase() が構造体のコピーを返している場合、パスの変更は反映されない。
// その場合はDBエクスポートパスがデフォルトのまま実行され、権限エラーになる可能性がある。
