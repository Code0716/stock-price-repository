package commands

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
	"go.uber.org/mock/gomock"

	"github.com/Code0716/stock-price-repository/config"
	mock_usecase "github.com/Code0716/stock-price-repository/mock/usecase"
)

func TestReadManualEvents(t *testing.T) {
	dir := t.TempDir()
	existing := filepath.Join(dir, "manual.yaml")
	require.NoError(t, os.WriteFile(existing, []byte("- {date: \"2026-12-17\", kind: boj}\n"), 0o600))

	tests := []struct {
		name    string
		path    string
		want    []byte
		wantErr bool
	}{
		{name: "正常系: ファイルを読む", path: existing, want: []byte("- {date: \"2026-12-17\", kind: boj}\n")},
		{name: "正常系: ファイルが無ければ手入力なし(nil)", path: filepath.Join(dir, "none.yaml"), want: nil},
		{name: "正常系: パスが空なら手入力なし(nil)", path: "", want: nil},
		{name: "異常系: ディレクトリは読めない", path: dir, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := readManualEvents(tt.path)
			if (err != nil) != tt.wantErr {
				t.Fatalf("readManualEvents() error = %v, wantErr %v", err, tt.wantErr)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestManualEventsPath(t *testing.T) {
	original := *config.GetEconomicCalendar()
	defer func() { *config.GetEconomicCalendar() = original }()
	config.GetEconomicCalendar().ManualEventsPath = "from/config.yaml"

	assert.Equal(t, "from/flag.yaml", manualEventsPath("from/flag.yaml"), "フラグ優先")
	assert.Equal(t, "from/config.yaml", manualEventsPath(""), "フラグ省略時は設定")
}

func TestSyncMarketCalendarCommand_Action(t *testing.T) {
	dir := t.TempDir()
	manual := filepath.Join(dir, "manual.yaml")
	require.NoError(t, os.WriteFile(manual, []byte("- {date: \"2026-12-17\", kind: boj}\n"), 0o600))

	tests := []struct {
		name    string
		args    []string
		setup   func(m *mock_usecase.MockSyncMarketCalendarInteractor)
		wantErr bool
	}{
		{
			name: "正常系: 手入力YAMLを読んで usecase に渡す",
			args: []string{"sync_market_calendar", "--manual", manual},
			setup: func(m *mock_usecase.MockSyncMarketCalendarInteractor) {
				m.EXPECT().SyncMarketCalendar(gomock.Any(), gomock.Any(), gomock.Eq([]byte("- {date: \"2026-12-17\", kind: boj}\n"))).
					DoAndReturn(func(_ context.Context, now time.Time, _ []byte) error {
						assert.WithinDuration(t, time.Now(), now, time.Minute)
						return nil
					})
			},
		},
		{
			name: "正常系: 手入力ファイルが無ければ nil を渡す",
			args: []string{"sync_market_calendar", "--manual", filepath.Join(dir, "none.yaml")},
			setup: func(m *mock_usecase.MockSyncMarketCalendarInteractor) {
				m.EXPECT().SyncMarketCalendar(gomock.Any(), gomock.Any(), gomock.Nil()).Return(nil)
			},
		},
		{
			name: "異常系: usecase のエラーを返す",
			args: []string{"sync_market_calendar", "--manual", filepath.Join(dir, "none.yaml")},
			setup: func(m *mock_usecase.MockSyncMarketCalendarInteractor) {
				m.EXPECT().SyncMarketCalendar(gomock.Any(), gomock.Any(), gomock.Nil()).Return(errors.New("boom"))
			},
			wantErr: true,
		},
		{
			name:    "異常系: 手入力パスが読めなければ usecase を呼ばずエラー",
			args:    []string{"sync_market_calendar", "--manual", dir},
			setup:   func(m *mock_usecase.MockSyncMarketCalendarInteractor) {},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := mock_usecase.NewMockSyncMarketCalendarInteractor(ctrl)
			tt.setup(m)

			cmd := NewSyncMarketCalendarCommand(m).Command()
			app := cli.NewApp()
			app.Commands = []*cli.Command{cmd.CliCommand()}

			err := app.RunContext(context.Background(), append([]string{"spr"}, tt.args...))
			if (err != nil) != tt.wantErr {
				t.Fatalf("Action() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
