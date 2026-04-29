package driver

import (
	"context"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Code0716/stock-price-repository/config"
)

func TestBoxAPIClient_UploadFile(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T)
		wantErr bool
	}{
		{
			name: "スキップ: BOX_RCLONE_FOLDER_PATH が空",
			setup: func(t *testing.T) {
				t.Setenv("BOX_RCLONE_FOLDER_PATH", "")
				t.Setenv("BOX_RCLONE_REMOTE_NAME", "box")
				config.LoadConfigBOX()
			},
			wantErr: true,
		},
		{
			name: "スキップ: rclone が PATH に存在しない",
			setup: func(t *testing.T) {
				t.Setenv("BOX_RCLONE_FOLDER_PATH", "backup/")
				t.Setenv("BOX_RCLONE_REMOTE_NAME", "box")
				config.LoadConfigBOX()
				t.Setenv("PATH", "")
			},
			wantErr: true,
		},
		{
			name: "異常系: rclone が非0終了",
			setup: func(t *testing.T) {
				t.Setenv("BOX_RCLONE_FOLDER_PATH", "backup/")
				t.Setenv("BOX_RCLONE_REMOTE_NAME", "box")
				config.LoadConfigBOX()
				orig := execCommandContext
				t.Cleanup(func() { execCommandContext = orig })
				execCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
					return exec.CommandContext(ctx, "false")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup(t)
			client := &BoxAPIClient{}
			err := client.UploadFile(context.Background(), "/tmp/test.sql")
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
