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

	// 2. Setup Config for Test (Override Host to force TCP, and BackupPath to temp dir)
	dbConfig := config.GetDatabase()
	originalHost := dbConfig.Host
	originalPath := dbConfig.ExportBackupPath

	// Force TCP connection by using 127.0.0.1 instead of localhost (which might use socket)
	dbConfig.Host = "127.0.0.1"

	// Create temp dir for export
	tmpDir, err := os.MkdirTemp("", "e2e_export_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	dbConfig.ExportBackupPath = tmpDir

	// Restore config after test
	defer func() {
		dbConfig.Host = originalHost
		dbConfig.ExportBackupPath = originalPath
	}()

	// Use Real Client
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
		})
	}
}
