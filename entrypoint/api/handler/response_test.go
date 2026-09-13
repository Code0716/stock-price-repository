package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	pkgerrors "github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestWriteError(t *testing.T) {
	type args struct {
		logMsg string
		err    error
	}
	tests := []struct {
		name           string
		args           args
		wantStatusCode int
		wantBody       string
		wantLogCount   int
		wantLogMsg     string
	}{
		{
			name: "validationError の場合は400を返しログは出力しない",
			args: args{
				logMsg: "should not be logged",
				err:    &validationError{message: "バリデーションエラーです"},
			},
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "バリデーションエラーです\n",
			wantLogCount:   0,
		},
		{
			name: "wrap された validationError の場合も400を返す",
			args: args{
				logMsg: "should not be logged",
				err:    pkgerrors.Wrap(&validationError{message: "wrapされたバリデーションエラーです"}, "wrapped"),
			},
			wantStatusCode: http.StatusBadRequest,
			wantBody:       "wrapされたバリデーションエラーです\n",
			wantLogCount:   0,
		},
		{
			name: "一般エラーの場合はログ出力の上500を返す",
			args: args{
				logMsg: "failed to do something",
				err:    pkgerrors.New("boom"),
			},
			wantStatusCode: http.StatusInternalServerError,
			wantBody:       "内部サーバーエラー\n",
			wantLogCount:   1,
			wantLogMsg:     "failed to do something",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			core, logs := observer.New(zapcore.DebugLevel)
			logger := zap.New(core)

			w := httptest.NewRecorder()
			writeError(w, logger, tt.args.logMsg, tt.args.err)

			assert.Equal(t, tt.wantStatusCode, w.Code)
			assert.Equal(t, tt.wantBody, w.Body.String())
			assert.Equal(t, tt.wantLogCount, logs.Len())
			if tt.wantLogCount > 0 {
				entry := logs.All()[0]
				assert.Equal(t, tt.wantLogMsg, entry.Message)
				assert.Equal(t, zapcore.ErrorLevel, entry.Level)
			}
		})
	}
}
