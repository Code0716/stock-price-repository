package driver

import (
	"context"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Code0716/stock-price-repository/config"
)

// TestHelperProcess is a helper process for testing exec.Command
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	defer os.Exit(0)

	// 引数をチェックしたり、標準出力に書き込んだりする
	// args := os.Args
	// for _, arg := range args {
	// 	fmt.Println(arg)
	// }
}

func TestMySQLDumpClient_ExportTableAll(t *testing.T) {
	// 保存元のexecCommandContextを退避
	oldExecCommandContext := execCommandContext
	// テスト終了後に復元
	defer func() { execCommandContext = oldExecCommandContext }()

	// execCommandContextをモックに置き換え
	execCommandContext = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
		cs := []string{"-test.run=TestHelperProcess", "--", name}
		cs = append(cs, arg...)
		cmd := exec.CommandContext(ctx, os.Args[0], cs...)
		cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
		return cmd
	}

	client := NewMySQLDumpClient()

	// テスト用の一時ディレクトリを作成
	tmpDir, err := os.MkdirTemp("", "export_test")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// config.GetDatabase()のExportBackupPathを一時ディレクトリに変更
	dbConfig := config.GetDatabase()
	originalPath := dbConfig.ExportBackupPath
	dbConfig.ExportBackupPath = tmpDir
	defer func() {
		dbConfig.ExportBackupPath = originalPath
	}()

	err = client.ExportTableAll(context.Background(), "test_file", "test_table")
	assert.NoError(t, err)
}

func TestMySQLDumpClient_ExportTableByYear(t *testing.T) {
	oldExecCommandContext := execCommandContext
	defer func() { execCommandContext = oldExecCommandContext }()

	execCommandContext = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
		cs := []string{"-test.run=TestHelperProcess", "--", name}
		cs = append(cs, arg...)
		cmd := exec.CommandContext(ctx, os.Args[0], cs...)
		cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
		return cmd
	}

	client := NewMySQLDumpClient()

	// テスト用の一時ディレクトリを作成
	tmpDir, err := os.MkdirTemp("", "export_yearly_test")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// config.GetDatabase()のExportBackupPathを一時ディレクトリに変更
	dbConfig := config.GetDatabase()
	originalPath := dbConfig.ExportBackupPath
	dbConfig.ExportBackupPath = tmpDir
	defer func() {
		dbConfig.ExportBackupPath = originalPath
	}()

	err = client.ExportTableByYear(context.Background(), "test_table", 2024)
	assert.NoError(t, err)
}