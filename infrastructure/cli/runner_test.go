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
	"github.com/Code0716/stock-price-repository/infrastructure/gateway/resource"
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
			name: "正常系: コマンド成功時、time taken を含む Block Kit メッセージが SendBlockMessage で通知され、@channel を含まない",
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
						SendBlockMessage(
							gomock.Any(),
							gomock.Eq(gateway.SlackChannelNameDevNotification),
							gomock.Any(),
						).
						DoAndReturn(func(_ context.Context, _ gateway.SlackChannelName, msg resource.SlackBlockMessage) error {
							if !strings.Contains(msg.Text, "time taken") {
								t.Errorf("expected 'time taken' in text, got: %v", msg.Text)
							}
							if !strings.Contains(msg.Text, "dummy") {
								t.Errorf("expected 'dummy' (command name) in text, got: %v", msg.Text)
							}
							if strings.Contains(msg.Text, "<!channel>") {
								t.Errorf("成功通知には <!channel> を含めないこと, got: %v", msg.Text)
							}
							return nil
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
			name: "失敗系: コマンド失敗時、time taken と @channel を含む Block Kit メッセージが SendBlockMessage で通知される",
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
						SendBlockMessage(
							gomock.Any(),
							gomock.Eq(gateway.SlackChannelNameDevNotification),
							gomock.Any(),
						).
						DoAndReturn(func(_ context.Context, _ gateway.SlackChannelName, msg resource.SlackBlockMessage) error {
							if !strings.Contains(msg.Text, "time taken") {
								t.Errorf("expected 'time taken' in text, got: %v", msg.Text)
							}
							if !strings.Contains(msg.Text, "failing") {
								t.Errorf("expected 'failing' (command name) in text, got: %v", msg.Text)
							}
							if len(msg.Attachments) == 0 || msg.Attachments[0].Color != "#d93025" {
								t.Errorf("expected red attachment color, got: %+v", msg.Attachments)
							}
							var mentioned bool
							for _, b := range msg.Blocks {
								if b.Text != nil && strings.Contains(b.Text.Text, "<!channel>") {
									mentioned = true
								}
							}
							if !mentioned {
								t.Errorf("失敗通知には <!channel> を含めること")
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
		{
			name: "失敗系: SendBlockMessage 自体が失敗したら SendErrMessageNotification へフォールバックする",
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
						SendBlockMessage(gomock.Any(), gomock.Eq(gateway.SlackChannelNameDevNotification), gomock.Any()).
						Return(assertErr("block message send error"))
					m.EXPECT().
						SendErrMessageNotification(gomock.Any(), gomock.Any()).
						DoAndReturn(func(_ context.Context, err error) error {
							msg := err.Error()
							if !strings.Contains(msg, "failing") {
								t.Errorf("expected 'failing' (command name) in fallback err msg, got: %v", msg)
							}
							if !strings.Contains(msg, "block notify error") {
								t.Errorf("expected block notify error reason in fallback err msg, got: %v", msg)
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
