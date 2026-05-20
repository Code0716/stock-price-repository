package cli

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v2"
	"go.uber.org/mock/gomock"

	"github.com/Code0716/stock-price-repository/infrastructure/cli/commands"
	"github.com/Code0716/stock-price-repository/infrastructure/gateway"
	mock_gateway "github.com/Code0716/stock-price-repository/mock/gateway"
)

func TestRunner_Run(t *testing.T) {
	type fields struct {
		commands       []*commands.Command
		slackAPIClient func(ctrl *gomock.Controller) gateway.SlackAPIClient
	}
	type args struct {
		ctx     context.Context
		cmdArgs []string
	}
	tests := []struct {
		name        string
		fields      fields
		args        args
		wantErr     bool
		errContains string
	}{
		{
			name: "args 不足の場合 not enough arguments エラー",
			fields: fields{
				slackAPIClient: func(ctrl *gomock.Controller) gateway.SlackAPIClient {
					// args チェックは Slack 呼び出し前に return するため、mock 呼び出し期待なし
					return mock_gateway.NewMockSlackAPIClient(ctrl)
				},
			},
			args: args{
				ctx:     context.Background(),
				cmdArgs: []string{"app"},
			},
			wantErr:     true,
			errContains: "not enough arguments",
		},
		{
			name: "正常系: コマンド成功時、time taken を含むメッセージが SendMessageByStrings で通知される",
			fields: fields{
				commands: []*commands.Command{
					{
						Name:   "dummy",
						Action: func(_ *cli.Context) error { return nil },
					},
				},
				slackAPIClient: func(ctrl *gomock.Controller) gateway.SlackAPIClient {
					m := mock_gateway.NewMockSlackAPIClient(ctrl)
					m.EXPECT().
						SendMessageByStrings(
							gomock.Any(),
							gomock.Eq(gateway.SlackChannelNameDevNotification),
							gomock.Any(),
							gomock.Nil(),
							gomock.Nil(),
						).
						DoAndReturn(func(_ context.Context, _ gateway.SlackChannelName, title string, _, _ *string) (string, error) {
							if !strings.Contains(title, "time taken") {
								t.Errorf("expected 'time taken' in title, got: %v", title)
							}
							if !strings.Contains(title, "dummy") {
								t.Errorf("expected 'dummy' (command name) in title, got: %v", title)
							}
							return "", nil
						})
					return m
				},
			},
			args: args{
				ctx:     context.Background(),
				cmdArgs: []string{"app", "dummy"},
			},
			wantErr: false,
		},
		{
			name: "失敗系: コマンド失敗時、time taken を含むメッセージが SendErrMessageNotification で通知される (FIXME 解消の確認)",
			fields: fields{
				commands: []*commands.Command{
					{
						Name:   "failing",
						Action: func(_ *cli.Context) error { return assertErr("command failed") },
					},
				},
				slackAPIClient: func(ctrl *gomock.Controller) gateway.SlackAPIClient {
					m := mock_gateway.NewMockSlackAPIClient(ctrl)
					m.EXPECT().
						SendErrMessageNotification(gomock.Any(), gomock.Any()).
						DoAndReturn(func(_ context.Context, err error) error {
							msg := err.Error()
							if !strings.Contains(msg, "time taken") {
								t.Errorf("expected 'time taken' in err msg, got: %v", msg)
							}
							if !strings.Contains(msg, "failing") {
								t.Errorf("expected 'failing' (command name) in err msg, got: %v", msg)
							}
							return nil
						})
					return m
				},
			},
			args: args{
				ctx:     context.Background(),
				cmdArgs: []string{"app", "failing"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			r := &Runner{
				commands:       tt.fields.commands,
				slackAPIClient: tt.fields.slackAPIClient(ctrl),
			}

			err := r.Run(tt.args.ctx, tt.args.cmdArgs)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// assertErr は失敗系テスト用の固定エラー。
type assertErr string

func (e assertErr) Error() string { return string(e) }
